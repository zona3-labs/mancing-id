package config

import (
	"log"

	"github.com/spf13/viper"
)

type UploadConfig struct {
	MaxSizeMB       int64  `mapstructure:"UPLOAD_MAX_SIZE_MB"`
	S3Region        string `mapstructure:"UPLOAD_S3_REGION"`
	S3Bucket        string `mapstructure:"UPLOAD_S3_BUCKET"`
	S3AccessKey     string `mapstructure:"UPLOAD_S3_ACCESS_KEY"`
	S3SecretKey     string `mapstructure:"UPLOAD_S3_SECRET_KEY"`
	S3BaseKeyPrefix string `mapstructure:"UPLOAD_S3_BASE_KEY_PREFIX"`
	// Optional: set to a custom URL (e.g. http://localhost:9000) for MinIO.
	// When set, path-style URLs are used automatically.
	S3Endpoint string `mapstructure:"UPLOAD_S3_ENDPOINT"`
}

func initUploadConfig() *UploadConfig {
	cfg := &UploadConfig{}
	if err := viper.Unmarshal(cfg); err != nil {
		log.Fatalf("error mapping upload config: %v", err)
	}
	return cfg
}
