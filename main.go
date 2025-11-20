package main

import (
	"bufio"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

func main() {
	root := flag.String("root", ".", "起始扫描目录")
	exts := flag.String("ext", ".go,.py", "目标后缀，逗号分隔")
	out := flag.String("out", "matched_files.txt", "输出文件路径")
	workers := flag.Int("workers", runtime.NumCPU(), "最大并发目录数")
	flag.Parse()

	suffixes := normalizeSuffixes(*exts)
	if len(suffixes) == 0 {
		fmt.Fprintln(os.Stderr, "至少指定一个后缀")
		os.Exit(1)
	}

	start := time.Now()
	count, err := scanAndWrite(*root, suffixes, *out, *workers)
	elapsed := time.Since(start)
	if err != nil {
		fmt.Fprintln(os.Stderr, "扫描失败:", err)
		os.Exit(1)
	}
	fmt.Printf("扫描完成，匹配文件 %d 个，耗时 %s\n", count, elapsed)
}

func normalizeSuffixes(extList string) []string {
	items := strings.Split(extList, ",")
	out := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if !strings.HasPrefix(item, ".") {
			item = "." + item
		}
		out = append(out, strings.ToLower(item))
	}
	return out
}

func scanAndWrite(root string, suffixes []string, outFile string, concurrency int) (int64, error) {
	info, err := os.Stat(root)
	if err != nil {
		return 0, err
	}
	if !info.IsDir() {
		return 0, errors.New("root 必须是目录")
	}

	if err := os.MkdirAll(filepath.Dir(outFile), 0o755); err != nil {
		return 0, err
	}
	f, err := os.Create(outFile)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	writer := bufio.NewWriter(f)
	defer writer.Flush()

	fileCh := make(chan string, concurrency*8)
	var writerWG sync.WaitGroup
	writerWG.Add(1)
	go func() {
		defer writerWG.Done()
		for path := range fileCh {
			fmt.Fprintln(writer, path)
		}
	}()

	var matched int64
	sem := make(chan struct{}, max(1, concurrency))

	var walkWG sync.WaitGroup
	var walkDir func(string)
	walkDir = func(dir string) {
		walkWG.Add(1)
		go func(path string) {
			defer walkWG.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			entries, err := os.ReadDir(path)
			if err != nil {
				fmt.Fprintf(os.Stderr, "跳过 %s: %v\n", path, err)
				return
			}

			for _, entry := range entries {
				if entry.Type()&os.ModeSymlink != 0 {
					continue
				}
				child := filepath.Join(path, entry.Name())
				if entry.IsDir() {
					walkDir(child)
					continue
				}
				name := strings.ToLower(entry.Name())
				for _, ext := range suffixes {
					if strings.HasSuffix(name, ext) {
						fileCh <- child
						atomic.AddInt64(&matched, 1)
						break
					}
				}
			}
		}(dir)
	}

	walkDir(root)
	walkWG.Wait()
	close(fileCh)
	writerWG.Wait()
	return matched, nil
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
