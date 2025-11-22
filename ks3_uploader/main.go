package main

import (
	"bufio"
	"flag"
	"fmt"
	"io"
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

	if strings.HasPrefix(endpoint, "https://") {
		cfg.DisableSSL = false
	}

	svc := s3.New(cfg)

	// Open Input File (optimized for large files)
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

	// --- Logging System Setup ---

	// 1. Success Logger
	successLogPath := inputFile + ".success"
	successLogFile, err := os.Create(successLogPath)
	if err != nil {
		log.Printf("Warning: Could not create success log file: %v", err)
	}
	successChan := make(chan string, 10000)
	var logWg sync.WaitGroup
	
	logWg.Add(1)
	go func() {
		defer logWg.Done()
		if successLogFile == nil {
			for range successChan {} // Drain
			return
		}
		defer successLogFile.Close()
		writer := bufio.NewWriterSize(successLogFile, 64*1024) // 64KB buffer
		for line := range successChan {
			writer.WriteString(line + "\n")
		}
		writer.Flush()
	}()

	// 2. Failure Logger
	failLogPath := inputFile + ".failed"
	failLogFile, err := os.Create(failLogPath)
	if err != nil {
		log.Printf("Warning: Could not create failure log file: %v", err)
	}
	failChan := make(chan string, 10000)
	
	logWg.Add(1)
	go func() {
		defer logWg.Done()
		if failLogFile == nil {
			for range failChan {} // Drain
			return
		}
		defer failLogFile.Close()
		writer := bufio.NewWriterSize(failLogFile, 64*1024)
		for line := range failChan {
			writer.WriteString(line + "\n")
		}
		writer.Flush()
	}()

	// Progress Reporter
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

	// Start Workers
	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for path := range jobs {
				if path == "" {
					continue
				}
				
				// Upload
				remoteKey, err := uploadFile(svc, path)
				
				if err != nil {
					atomic.AddInt64(&failCount, 1)
					failChan <- path
				} else {
					atomic.AddInt64(&successCount, 1)
					// Log format: LocalPath <tab> RemoteKey
					successChan <- fmt.Sprintf("%s\t%s", path, remoteKey) 
				}
			}
		}()
	}

	// Optimized File Reading for Large Input Files (e.g., 12GB)
	// Instead of Scanner (which has token limits), we use bufio.Reader
	reader := bufio.NewReaderSize(file, 1024*1024) // 1MB buffer for reading
	for {
		line, err := reader.ReadString('\n')
		if line != "" {
			// Trim whitespace
			line = strings.TrimSpace(line)
			if line != "" {
				jobs <- line
			}
		}
		if err != nil {
			if err != io.EOF {
				log.Printf("Error reading input file: %v", err)
			}
			break
		}
	}

	close(jobs)
	wg.Wait()         // Wait for workers to finish
	
	close(successChan) // Close log channels
	close(failChan)
	logWg.Wait()      // Wait for loggers to flush
	
	close(done)       // Stop progress reporter

	// Final report
	fmt.Printf("\n\nDone.\nTotal Time: %v\nSuccess: %d\nFailed: %d\n", time.Since(startTime), successCount, failCount)
	fmt.Printf("Success log: %s\n", successLogPath)
	if failCount > 0 {
		fmt.Printf("Failed log: %s\n", failLogPath)
	}
}

func uploadFile(svc *s3.S3, localPath string) (string, error) {
	// Generate remote key
	key := strings.Replace(localPath, prefixOld, prefixNew, 1)
	key = strings.TrimPrefix(key, "/")

	fd, err := os.Open(localPath)
	if err != nil {
		return "", fmt.Errorf("open file: %w", err)
	}
	defer fd.Close()

	_, err = svc.PutObject(&s3.PutObjectInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String(key),
		Body:        fd,
		ACL:         aws.String("public-read"),
		ContentType: aws.String("application/octet-stream"),
	})

	if err != nil {
		return "", err
	}
	
	return key, nil
}
