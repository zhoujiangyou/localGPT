package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/ks3sdklib/aws-sdk-go/service/s3"
)

// processTask 执行单个扫描任务，并可选上传结果
func processTask(root string, relPath string, suffixes []string, keywords []string, outDir string, workers int, ks3Svc *s3.S3, bucket string, remotePrefix string) (int64, error) {
	start := time.Now()

	// 构造输出文件名：将路径分隔符替换为横杠
	safeName := strings.ReplaceAll(relPath, string(os.PathSeparator), "-")
	if safeName == "." || safeName == "" {
		safeName = "root"
	}
	safeName += ".result.txt"
	finalOutPath := filepath.Join(outDir, safeName)
	tempOutPath := finalOutPath + ".tmp"

	// 创建临时文件存放路径列表
	tmpFile, err := os.Create(tempOutPath)
	if err != nil {
		return 0, err
	}

	bufWriter := bufio.NewWriter(tmpFile)

	// 执行扫描
	count, err := scanToWriter(root, suffixes, keywords, bufWriter, workers)
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
	
	// 写入头部信息
	header := fmt.Sprintf("# Total: %d, Duration: %s, Root: %s\n", count, elapsed, root)
	if _, err := finalFile.WriteString(header); err != nil {
		finalFile.Close()
		return 0, err
	}

	// 复制临时文件内容
	tmpFileRead, err := os.Open(tempOutPath)
	if err != nil {
		finalFile.Close()
		return 0, err
	}
	_, err = io.Copy(finalFile, tmpFileRead)
	tmpFileRead.Close()
	finalFile.Close() // 写入完成，关闭

	// 删除临时文件
	os.Remove(tempOutPath)

	// 上传到 KS3
	if ks3Svc != nil {
		remoteKey := filepath.Join(remotePrefix, safeName)
		// 这里的 remoteKey 可能会包含反斜杠（Windows），KS3 key 一般用正斜杠
		remoteKey = filepath.ToSlash(remoteKey)
		
		fmt.Printf("[%s] 正在上传到 KS3: %s ...\n", relPath, remoteKey)
		if err := uploadFile(ks3Svc, bucket, finalOutPath, remoteKey); err != nil {
			return count, fmt.Errorf("scan success but upload failed: %v", err)
		}
	}

	return count, nil
}
