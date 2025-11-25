package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/ks3sdklib/aws-sdk-go/service/s3"
)

// processTaskWithCounter 执行单个扫描任务，并实时更新全局计数器
func processTaskWithCounter(root string, relPath string, suffixes []string, keywords []string, outDir string, workers int, ks3Svc *s3.S3, bucket string, remotePrefix string, globalCounter *int64) (int64, error) {
	start := time.Now()

	safeName := strings.ReplaceAll(relPath, string(os.PathSeparator), "-")
	if safeName == "." || safeName == "" {
		safeName = "root"
	}
	safeName += ".result.txt"
	finalOutPath := filepath.Join(outDir, safeName)
	tempOutPath := finalOutPath + ".tmp"

	tmpFile, err := os.Create(tempOutPath)
	if err != nil {
		return 0, err
	}

	bufWriter := bufio.NewWriter(tmpFile)

	// 使用支持实时回调的扫描函数
	count, err := scanToWriterWithCallback(root, suffixes, keywords, bufWriter, workers, func(n int64) {
		// 每次发现匹配文件时，原子增加全局计数器
		atomic.AddInt64(globalCounter, n)
	})
	
	bufWriter.Flush()
	tmpFile.Close()

	if err != nil {
		os.Remove(tempOutPath)
		return 0, err
	}

	elapsed := time.Since(start)

	// 生成最终文件（带Header）
	finalFile, err := os.Create(finalOutPath)
	if err != nil {
		return 0, err
	}
	
	header := fmt.Sprintf("# Total: %d, Duration: %s, Root: %s\n", count, elapsed, root)
	if _, err := finalFile.WriteString(header); err != nil {
		finalFile.Close()
		return 0, err
	}

	tmpFileRead, err := os.Open(tempOutPath)
	if err != nil {
		finalFile.Close()
		return 0, err
	}
	_, err = io.Copy(finalFile, tmpFileRead)
	tmpFileRead.Close()
	finalFile.Close()

	os.Remove(tempOutPath)

	// 上传到 KS3
	if ks3Svc != nil {
		remoteKey := filepath.Join(remotePrefix, safeName)
		remoteKey = filepath.ToSlash(remoteKey)
		
		// 为了避免打断进度条，这里就不打印上传日志了，或者可以另行处理日志
		// fmt.Printf("[%s] 正在上传到 KS3: %s ...\n", relPath, remoteKey)
		if err := uploadFile(ks3Svc, bucket, finalOutPath, remoteKey); err != nil {
			return count, fmt.Errorf("scan success but upload failed: %v", err)
		}
	}

	return count, nil
}
