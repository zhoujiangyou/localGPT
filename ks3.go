package main

import (
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/ks3sdklib/aws-sdk-go/aws"
	"github.com/ks3sdklib/aws-sdk-go/aws/credentials"
	"github.com/ks3sdklib/aws-sdk-go/service/s3"
)

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
func uploadFile(svc *s3.S3, bucket, localPath, remoteKey string) error {
	fd, err := os.Open(localPath)
	if err != nil {
		return fmt.Errorf("open file: %w", err)
	}
	defer fd.Close()

	_, err = svc.PutObject(&s3.PutObjectInput{
		Bucket:      aws.String(bucket),
		Key:         aws.String(remoteKey),
		Body:        fd,
		ACL:         aws.String("public-read"),
		ContentType: aws.String("text/plain"),
	})

	return err
}
