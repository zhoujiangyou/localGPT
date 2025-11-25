package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Flags
var (
	rootFlag       = flag.String("root", ".", "起始扫描目录")
	extsFlag       = flag.String("ext", ".go,.py", "目标后缀，逗号分隔")
	outDirFlag     = flag.String("out-dir", "results", "结果输出目录")
	levelFlag      = flag.Int("level", 0, "聚合层级 (0表示不聚合，直接扫描root)")
	workersFlag    = flag.Int("workers", runtime.NumCPU(), "最大并发数")
	checkpointFlag = flag.String("checkpoint", "checkpoint.txt", "断点记录文件")
)

func main() {
	flag.Parse()

	// 1. 准备环境
	suffixes := normalizeSuffixes(*extsFlag)
	if len(suffixes) == 0 {
		fmt.Fprintln(os.Stderr, "至少指定一个后缀")
		os.Exit(1)
	}

	absRoot, err := filepath.Abs(*rootFlag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "解析根路径失败: %v\n", err)
		os.Exit(1)
	}

	if err := os.MkdirAll(*outDirFlag, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "创建输出目录失败: %v\n", err)
		os.Exit(1)
	}

	// 2. 加载断点
	completedTasks := loadCheckpoint(*checkpointFlag)
	checkpointMutex := &sync.Mutex{}

	// 3. 发现任务 (Tasks)
	fmt.Println("正在发现任务...")
	tasks, err := findTasks(absRoot, *levelFlag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "发现任务失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("共发现 %d 个任务\n", len(tasks))

	// 4. 启动工作池
	taskCh := make(chan string, len(tasks))
	for _, t := range tasks {
		taskCh <- t
	}
	close(taskCh)

	var wg sync.WaitGroup

	// 策略：如果有聚合层级，则在任务间并发；否则在单任务内并发
	var taskWorkers int
	var scanWorkers int

	if *levelFlag > 0 {
		taskWorkers = *workersFlag
		if taskWorkers < 1 {
			taskWorkers = 1
		}
		scanWorkers = 1 // 任务多时，内部串行以减少开销
	} else {
		taskWorkers = 1
		scanWorkers = *workersFlag // 单任务时，内部并行
	}

	fmt.Printf("并发策略: %d 个任务并行处理, 每个任务内部 %d 并发\n", taskWorkers, scanWorkers)

	for i := 0; i < taskWorkers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for taskRoot := range taskCh {
				// 任务标识符：相对路径
				relPath, _ := filepath.Rel(absRoot, taskRoot)
				if relPath == "." {
					relPath = "root"
				}

				checkpointMutex.Lock()
				if completedTasks[relPath] {
					checkpointMutex.Unlock()
					continue
				}
				checkpointMutex.Unlock()

				err := processTask(taskRoot, relPath, suffixes, *outDirFlag, scanWorkers)
				if err != nil {
					fmt.Fprintf(os.Stderr, "任务失败 [%s]: %v\n", relPath, err)
				} else {
					checkpointMutex.Lock()
					appendCheckpoint(*checkpointFlag, relPath)
					completedTasks[relPath] = true
					checkpointMutex.Unlock()
					fmt.Printf("任务完成: %s\n", relPath)
				}
			}
		}()
	}

	wg.Wait()
	fmt.Println("所有任务处理完毕")
}

// findTasks 查找指定深度的所有目录作为任务根
func findTasks(base string, targetDepth int) ([]string, error) {
	if targetDepth == 0 {
		return []string{base}, nil
	}
	var tasks []string
	
	// 使用WalkDir查找指定深度的目录
	// 注意：这里简单实现，如果是非常深的层级可能效率一般，但对于通常用途足够
	err := filepath.WalkDir(base, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			// 遇到没权限等错误跳过
			return filepath.SkipDir
		}
		
		// 计算当前深度
		rel, _ := filepath.Rel(base, path)
		depth := 0
		if rel != "." {
			depth = len(strings.Split(rel, string(os.PathSeparator)))
		}

		if depth == targetDepth && d.IsDir() {
			tasks = append(tasks, path)
			return filepath.SkipDir // 找到目标层级后，不再深入该分支
		}
		
		if depth > targetDepth {
			return filepath.SkipDir
		}
		return nil
	})

	return tasks, err
}

// processTask 执行单个扫描任务
func processTask(root string, relPath string, suffixes []string, outDir string, workers int) error {
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
		return err
	}
	// 确保在函数退出前关闭临时文件，以便后续操作
	// 使用命名返回值或者显式Close更好，这里为了Defer逻辑简单，稍后Close
	
	bufWriter := bufio.NewWriter(tmpFile)

	// 执行扫描
	count, err := scanToWriter(root, suffixes, bufWriter, workers)
	bufWriter.Flush()
	tmpFile.Close()

	if err != nil {
		os.Remove(tempOutPath)
		return err
	}

	elapsed := time.Since(start)

	// 生成最终文件（带Header）
	finalFile, err := os.Create(finalOutPath)
	if err != nil {
		return err
	}
	defer finalFile.Close()

	// 写入头部信息
	header := fmt.Sprintf("# Total: %d, Duration: %s\n", count, elapsed)
	if _, err := finalFile.WriteString(header); err != nil {
		return err
	}

	// 复制临时文件内容
	tmpFileRead, err := os.Open(tempOutPath)
	if err != nil {
		return err
	}
	_, err = io.Copy(finalFile, tmpFileRead)
	tmpFileRead.Close()
	
	// 删除临时文件
	os.Remove(tempOutPath)

	return err
}

// scanToWriter 扫描目录并将结果写入 Writer
func scanToWriter(root string, suffixes []string, w io.Writer, workerCount int) (int64, error) {
	// 结果通道
	fileCh := make(chan string, 1024)
	var matched int64
	var wgWriter sync.WaitGroup

	// 启动写入协程
	wgWriter.Add(1)
	go func() {
		defer wgWriter.Done()
		for path := range fileCh {
			fmt.Fprintln(w, path)
		}
	}()

	// 并发控制
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
				// 尝试获取令牌进行并发
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

// 辅助函数

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

func appendCheckpoint(path string, taskName string) {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return
	}
	defer f.Close()
	fmt.Fprintln(f, taskName)
}
