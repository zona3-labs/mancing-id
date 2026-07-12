package infrastructure

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/zone3-labs/mancing-id/internal/config"
)

// NewS3Client initialises an AWS S3 client from application config.
// When cfg.Upload.S3Endpoint is non-empty (e.g. MinIO), path-style URLs are
// used automatically so no manual switching is needed.
func NewS3Client(cfg *config.Config) (*s3.Client, error) {
	uc := cfg.Upload

	awsCfg := aws.Config{
		Region: uc.S3Region,
		Credentials: credentials.NewStaticCredentialsProvider(
			uc.S3AccessKey,
			uc.S3SecretKey,
			"",
		),
	}

	opts := []func(*s3.Options){}

	if uc.S3Endpoint != "" {
		opts = append(opts, func(o *s3.Options) {
			o.BaseEndpoint = aws.String(uc.S3Endpoint)
			o.UsePathStyle = true
		})
	}

	client := s3.NewFromConfig(awsCfg, opts...)

	// Smoke-test: list buckets to verify credentials & connectivity.
	if _, err := client.ListBuckets(context.Background(), &s3.ListBucketsInput{}); err != nil {
		return nil, fmt.Errorf("s3: failed to connect: %w", err)
	}

	return client, nil
}
