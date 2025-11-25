package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ks3sdklib/aws-sdk-go/aws"
	"github.com/ks3sdklib/aws-sdk-go/aws/credentials"
	"github.com/ks3sdklib/aws-sdk-go/service/s3"
)

// Flags
var (
	rootFlag       = flag.String("root", ".", "起始扫描目录")
	extsFlag       = flag.String("ext", ".go,.py", "目标后缀，逗号分隔")
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

				err := processTask(taskRoot, relPath, suffixes, *outDirFlag, scanWorkers, ks3Svc)
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

// processTask 执行单个扫描任务，并可选上传结果
func processTask(root string, relPath string, suffixes []string, outDir string, workers int, ks3Svc *s3.S3) error {
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
	// 这里不能直接 defer close，因为如果需要上传，我们可能需要重新打开或确保已落盘
	
	// 写入头部信息
	header := fmt.Sprintf("# Total: %d, Duration: %s, Root: %s\n", count, elapsed, root)
	if _, err := finalFile.WriteString(header); err != nil {
		finalFile.Close()
		return err
	}

	// 复制临时文件内容
	tmpFileRead, err := os.Open(tempOutPath)
	if err != nil {
		finalFile.Close()
		return err
	}
	_, err = io.Copy(finalFile, tmpFileRead)
	tmpFileRead.Close()
	finalFile.Close() // 写入完成，关闭

	// 删除临时文件
	os.Remove(tempOutPath)

	// 上传到 KS3
	if ks3Svc != nil {
		remoteKey := filepath.Join(*remotePrefixFlag, safeName)
		// 这里的 remoteKey 可能会包含反斜杠（Windows），KS3 key 一般用正斜杠
		remoteKey = filepath.ToSlash(remoteKey)
		
		fmt.Printf("[%s] 正在上传到 KS3: %s ...\n", relPath, remoteKey)
		if err := uploadFile(ks3Svc, finalOutPath, remoteKey); err != nil {
			return fmt.Errorf("scan success but upload failed: %v", err)
		}
	}

	return nil
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

// initKS3Client 初始化 KS3 客户端
func initKS3Client(ak, sk, endpoint, region string, maxConns int) *s3.S3 {
	// Configure HTTP Client for high concurrency
	httpClient := &http.Client{
		Transport: &http.Transport{
			MaxIdleConns:        maxConns,
			MaxIdleConnsPerHost: maxConns,
			IdleConnTimeout:     90 * time.Second,
		},
		Timeout: 60 * time.Second,
	}

	cre := credentials.NewStaticCredentials(ak, sk, "")
	cfg := &aws.Config{
		Credentials: cre,
		Region:      region,
		Endpoint:    endpoint,
		HTTPClient:  httpClient,
		LogLevel:    0,
		DisableSSL:  true, // Default to true for internal endpoint speed
	}

	if strings.HasPrefix(endpoint, "https://") {
		cfg.DisableSSL = false
	}

	return s3.New(cfg)
}

// uploadFile 上传单个文件到 KS3
func uploadFile(svc *s3.S3, localPath, remoteKey string) error {
	fd, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}
	defer fd.Close()

	_, err = svc.PutObject(&s3.PutObjectInput{
		Bucket:      bucketFlag,
		Key:         aws.String(remoteKey),
		Body:        fd,
		ACL:         aws.String("public-read"),
		ContentType: aws.String("text/plain"),
	})

	return err
}

// 辅助函数

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
