package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ks3sdklib/aws-sdk-go/service/s3"
)

// Flags
var (
	rootFlag       = flag.String("root", ".", "起始扫描目录")
	extsFlag       = flag.String("ext", ".go,.py", "目标后缀，逗号分隔 (若为空则匹配所有后缀)")
	keywordsFlag   = flag.String("keywords", "", "文件名关键词，逗号分隔 (若为空则匹配所有文件)")
	outDirFlag     = flag.String("out-dir", "results", "结果输出目录")
	levelFlag      = flag.Int("level", 0, "聚合层级 (0表示不聚合，直接扫描root)")
	workersFlag    = flag.Int("workers", runtime.NumCPU(), "最大并发数")
	checkpointFlag = flag.String("checkpoint", "checkpoint.txt", "断点记录文件")

	// KS3 Flags
	uploadFlag       = flag.Bool("upload", false, "是否上传结果文件到 KS3")
	akFlag           = flag.String("ak", "", "KS3 Access Key")
	skFlag           = flag.String("sk", "", "KS3 Secret Key")
	bucketFlag       = flag.String("bucket", "", "KS3 Bucket Name")
	endpointFlag     = flag.String("endpoint", "ks3-cn-tianjin-xm01-internal.ksyuncs.com", "KS3 Endpoint")
	regionFlag       = flag.String("region", "BEIJING", "KS3 Region")
	remotePrefixFlag = flag.String("remote-prefix", "scan-results", "KS3 存储路径前缀")
)

func main() {
	flag.Parse()
	startTime := time.Now()

	// 1. 准备环境
	suffixes := normalizeSuffixes(*extsFlag)
	keywords := normalizeKeywords(*keywordsFlag)

	if len(suffixes) == 0 && len(keywords) == 0 {
		fmt.Fprintln(os.Stderr, "必须指定至少一个过滤条件: -ext 或 -keywords")
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

	// 2. 初始化 KS3 客户端 (如果启用了上传)
	var ks3Svc *s3.S3
	if *uploadFlag {
		if *akFlag == "" || *skFlag == "" || *bucketFlag == "" {
			fmt.Fprintln(os.Stderr, "启用上传必须指定 -ak, -sk, -bucket")
			os.Exit(1)
		}
		ks3Svc = initKS3Client(*akFlag, *skFlag, *endpointFlag, *regionFlag, *workersFlag)
		fmt.Println("KS3 客户端初始化成功")
	}

	// 3. 加载断点
	completedTasks := loadCheckpoint(*checkpointFlag)
	checkpointMutex := &sync.Mutex{}

	// 4. 发现任务 (Tasks)
	fmt.Println("正在发现任务...")
	tasks, err := findTasks(absRoot, *levelFlag)
	if err != nil {
		fmt.Fprintf(os.Stderr, "发现任务失败: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("共发现 %d 个任务\n", len(tasks))

	// 5. 启动工作池
	taskCh := make(chan string, len(tasks))
	for _, t := range tasks {
		taskCh <- t
	}
	close(taskCh)

	var wg sync.WaitGroup
	var globalMatchedCount int64

	// 启动进度报告协程
	doneCh := make(chan struct{})
	go func() {
		ticker := time.NewTicker(2 * time.Second) // 每2秒输出一次
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				count := atomic.LoadInt64(&globalMatchedCount)
				duration := time.Since(startTime).Round(time.Second)
				fmt.Printf("\r[运行中] 已扫描命中: %d 文件 | 耗时: %s", count, duration)
			case <-doneCh:
				return
			}
		}
	}()

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

				// 调用 processor 处理任务
				// 注意：这里传入 &globalMatchedCount 用于实时更新计数
				_, err := processTaskWithCounter(taskRoot, relPath, suffixes, keywords, *outDirFlag, scanWorkers, ks3Svc, *bucketFlag, *remotePrefixFlag, &globalMatchedCount)
				if err != nil {
					// 使用 \n 换行避免覆盖进度条
					fmt.Fprintf(os.Stderr, "\n任务失败 [%s]: %v\n", relPath, err)
				} else {
					checkpointMutex.Lock()
					appendCheckpoint(*checkpointFlag, relPath)
					completedTasks[relPath] = true
					checkpointMutex.Unlock()
					// 完成时如果不想刷屏，可以注释掉下面这行，或者保留
					// fmt.Printf("\n任务完成: %s (Matched: %d)\n", relPath, count)
				}
			}
		}()
	}

	wg.Wait()
	close(doneCh)
	
	totalDuration := time.Since(startTime)
	fmt.Println("\n--------------------------------------------------") // 先换行，避免覆盖最后一行的进度
	fmt.Printf("所有任务处理完毕\n")
	fmt.Printf("全局统计 - 扫描命中文件总量: %d\n", atomic.LoadInt64(&globalMatchedCount))
	fmt.Printf("全局统计 - 总耗时: %s\n", totalDuration)
	fmt.Println("--------------------------------------------------")
}
