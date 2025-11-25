package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
)

// findTasks 查找指定深度的所有目录作为任务根
func findTasks(base string, targetDepth int) ([]string, error) {
	if targetDepth == 0 {
		return []string{base}, nil
	}
	var tasks []string
	err := filepath.WalkDir(base, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return filepath.SkipDir
		}
		rel, _ := filepath.Rel(base, path)
		depth := 0
		if rel != "." {
			depth = len(strings.Split(rel, string(os.PathSeparator)))
		}
		if depth == targetDepth && d.IsDir() {
			tasks = append(tasks, path)
			return filepath.SkipDir
		}
		if depth > targetDepth {
			return filepath.SkipDir
		}
		return nil
	})
	return tasks, err
}

// scanToWriter 扫描目录并将结果写入 Writer
func scanToWriter(root string, suffixes []string, w io.Writer, workerCount int) (int64, error) {
	fileCh := make(chan string, 1024)
	var matched int64
	var wgWriter sync.WaitGroup

	wgWriter.Add(1)
	go func() {
		defer wgWriter.Done()
		for path := range fileCh {
			fmt.Fprintln(w, path)
		}
	}()

	sem := make(chan struct{}, workerCount)
	var wgScan sync.WaitGroup

	var walk func(string)
	walk = func(dir string) {
		entries, err := os.ReadDir(dir)
		if err != nil {
			return
		}

		for _, entry := range entries {
			if entry.Type()&os.ModeSymlink != 0 {
				continue
			}

			path := filepath.Join(dir, entry.Name())

			if entry.IsDir() {
				select {
				case sem <- struct{}{}:
					wgScan.Add(1)
					go func(p string) {
						defer func() {
							<-sem
							wgScan.Done()
						}()
						walk(p)
					}(path)
				default:
					walk(path)
				}
			} else {
				name := entry.Name()
				for _, ext := range suffixes {
					if len(name) >= len(ext) && strings.EqualFold(name[len(name)-len(ext):], ext) {
						fileCh <- path
						atomic.AddInt64(&matched, 1)
						break
					}
				}
			}
		}
	}

	walk(root)
	wgScan.Wait()
	close(fileCh)
	wgWriter.Wait()

	return matched, nil
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
		out = append(out, item)
	}
	return out
}
