package main

import (
	"bufio"
	"fmt"
	"os"
	"strings"
	"sync"
)

// loadCheckpoint 加载断点记录
func loadCheckpoint(path string) map[string]bool {
	m := make(map[string]bool)
	f, err := os.Open(path)
	if err != nil {
		return m
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			m[line] = true
		}
	}
	return m
}

// appendCheckpoint 追加断点记录
func appendCheckpoint(path string, taskName string) {
	// 使用追加模式打开文件，如果文件不存在则创建
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to write checkpoint: %v\n", err)
		return
	}
	defer f.Close()
	
	// 加锁写入防止并发乱序（虽然 append 也是原子的，但 buffer 可能不是）
	// 这里由于可能有多个 worker 调用，最好由调用方加锁，或者这里简单加个锁
	// 但考虑到 file IO，调用方（worker pool）在 main 中已经处理了 checkpoint map 的锁，
	// 文件写入最好也受控。
	// 现在的 main 逻辑里，checkpointMutex 保护了 map 和 file write。
	// 所以这里直接写即可。
	fmt.Fprintln(f, taskName)
}

// CheckpointManager 简单的并发安全封装 (可选，暂保持原函数式风格配合 main 使用)
type CheckpointManager struct {
	mu        sync.Mutex
	path      string
	completed map[string]bool
}
