package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ks3sdklib/aws-sdk-go/aws"
	"github.com/ks3sdklib/aws-sdk-go/aws/credentials"
	"github.com/ks3sdklib/aws-sdk-go/service/s3"
)

// Config variables
var (
	inputFile   string
	concurrency int
	accessKey   string
	secretKey   string
	bucket      string
	endpoint    string
	region      string
	prefixOld   string
	prefixNew   string
)

func init() {
	flag.StringVar(&inputFile, "input", "", "Path to the file containing list of files to upload")
	flag.IntVar(&concurrency, "c", 100, "Number of concurrent uploads")
	flag.StringVar(&accessKey, "ak", "AKLT9FZYrFHHQeYoCP9u29cD", "KS3 Access Key")
	flag.StringVar(&secretKey, "sk", "ODZJrWEi0qPpUH59ZLj6Dlrli5fVtwfIQOu50YLn", "KS3 Secret Key")
	flag.StringVar(&bucket, "bucket", "evad-data-storage", "KS3 Bucket Name")
	flag.StringVar(&endpoint, "endpoint", "ks3-cn-tianjin-xm01-internal.ksyuncs.com", "KS3 Endpoint")
	flag.StringVar(&region, "region", "BEIJING", "KS3 Region")
	flag.StringVar(&prefixOld, "prefix-old", "/mnt/pro-output-data-storage/unified_time_line", "Path prefix to replace")
	flag.StringVar(&prefixNew, "prefix-new", "align-frame", "New path prefix")
}

func main() {
	flag.Parse()

	if inputFile == "" {
		log.Fatal("Please provide input file via -input")
	}

	// Configure HTTP Client for high concurrency
	httpClient := &http.Client{
		Transport: &http.Transport{
			MaxIdleConns:        concurrency,
			MaxIdleConnsPerHost: concurrency,
			IdleConnTimeout:     90 * time.Second,
		},
		Timeout: 60 * time.Second,
	}

	// Create KS3 Client
	cre := credentials.NewStaticCredentials(accessKey, secretKey, "")
	cfg := &aws.Config{
		Credentials: cre,
		Region:      region,
		Endpoint:    endpoint,
		HTTPClient:  httpClient,
		LogLevel:    0, 
		DisableSSL:  true, // Default to true for internal endpoint speed
	}

	// If endpoint contains https, DisableSSL should be false
	if strings.HasPrefix(endpoint, "https://") {
		cfg.DisableSSL = false
	}

	svc := s3.New(cfg)

	file, err := os.Open(inputFile)
	if err != nil {
		log.Fatalf("Failed to open input file: %v", err)
	}
	defer file.Close()

	jobs := make(chan string, concurrency*2)
	var wg sync.WaitGroup

	// Counters
	var successCount int64
	var failCount int64

	startTime := time.Now()

	// Progress reporter
	done := make(chan struct{})
	go func() {
		ticker := time.NewTicker(2 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				s := atomic.LoadInt64(&successCount)
				f := atomic.LoadInt64(&failCount)
				elapsed := time.Since(startTime).Seconds()
				rate := 0.0
				if elapsed > 0 {
					rate = float64(s+f) / elapsed
				}
				fmt.Printf("\rProcessed: %d (Success: %d, Failed: %d) | Rate: %.2f files/sec", s+f, s, f, rate)
			case <-done:
				return
			}
		}
	}()

	// Failure Logger
	failLogPath := inputFile + ".failed"
	failLogFile, err := os.Create(failLogPath)
	if err != nil {
		log.Printf("Warning: Could not create failure log file: %v", err)
	}
	
	failChan := make(chan string, 1000)
	var failWg sync.WaitGroup
	failWg.Add(1)
	go func() {
		defer failWg.Done()
		if failLogFile == nil {
			// Just drain channel if file creation failed
			for range failChan {}
			return
		}
		defer failLogFile.Close()
		writer := bufio.NewWriter(failLogFile)
		for path := range failChan {
			writer.WriteString(path + "\n")
		}
		writer.Flush()
	}()

	// Start Workers
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for path := range jobs {
				if path == "" {
					continue
				}
				if err := uploadFile(svc, path); err != nil {
					atomic.AddInt64(&failCount, 1)
					failChan <- path
				} else {
					atomic.AddInt64(&successCount, 1)
				}
			}
		}()
	}

	scanner := bufio.NewScanner(file)
	// Increase buffer for long paths
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" {
			jobs <- line
		}
	}

	if err := scanner.Err(); err != nil {
		log.Printf("Error reading input file: %v", err)
	}

	close(jobs)
	wg.Wait()      // Wait for workers
	close(failChan) // Close fail channel
	failWg.Wait()   // Wait for fail logger
	close(done)    // Stop progress reporter

	// Final report
	fmt.Printf("\n\nDone.\nTotal Time: %v\nSuccess: %d\nFailed: %d\n", time.Since(startTime), successCount, failCount)
	if failCount > 0 {
		fmt.Printf("Failed files listed in: %s\n", failLogPath)
	}
}

func uploadFile(svc *s3.S3, localPath string) error {
	// Generate remote key
	// Replace prefix
	key := strings.Replace(localPath, prefixOld, prefixNew, 1)
	// Remove leading slash if present in key to make it relative
	key = strings.TrimPrefix(key, "/")

	fd, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}
	defer fd.Close()

	_, err = svc.PutObject(&s3.PutObjectInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String(key),
		Body:        fd,
		ACL:         aws.String("public-read"),
		ContentType: aws.String("application/octet-stream"),
	})

	return err
}
