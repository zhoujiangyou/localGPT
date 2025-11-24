package s3

import (
	"context"
	"fmt"
	"os"
    "sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	aws_config "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/s3/manager"
	"github.com/aws/aws-sdk-go-v2/service/s3"
    "migration_tool/internal/config"
)

type Client struct {
	S3Client *s3.Client
    Uploader *manager.Uploader
    Bucket   string
}

var (
    instance *Client
    once     sync.Once
)

func NewClient(cfg *config.Config) (*Client, error) {
    var err error
    once.Do(func() {
        // Load AWS config with custom endpoint resolver for KS3
        awsCfg, loadErr := aws_config.LoadDefaultConfig(context.TODO(),
            aws_config.WithRegion(cfg.KS3Region),
            aws_config.WithEndpointResolverWithOptions(aws.EndpointResolverWithOptionsFunc(func(service, region string, options ...interface{}) (aws.Endpoint, error) {
                return aws.Endpoint{
                    URL:           "https://" + cfg.KS3Endpoint,
                    SigningRegion: cfg.KS3Region,
                }, nil
            })),
            // Add credentials explicitly if provided in config, otherwise let SDK find them
        )
        
        if loadErr != nil {
            err = loadErr
            return
        }

        s3Client := s3.NewFromConfig(awsCfg)
        
        uploader := manager.NewUploader(s3Client, func(u *manager.Uploader) {
            u.PartSize = 64 * 1024 * 1024 // 64MB part size
            u.Concurrency = 5
        })

        instance = &Client{
            S3Client: s3Client,
            Uploader: uploader,
            Bucket:   cfg.KS3Bucket,
        }
    })
    
    return instance, err
}

func (c *Client) UploadFile(ctx context.Context, sourcePath, destKey string) error {
	file, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	_, err = c.Uploader.Upload(ctx, &s3.PutObjectInput{
		Bucket: aws.String(c.Bucket),
		Key:    aws.String(destKey),
		Body:   file,
	})

	if err != nil {
		return fmt.Errorf("failed to upload object: %w", err)
	}
    
    return nil
}
