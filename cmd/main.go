package main

import (
	"log"
	"os"

	"github.com/zone3-labs/mancing-id/internal/config"
	"github.com/zone3-labs/mancing-id/internal/infrastructure"
	"github.com/zone3-labs/mancing-id/internal/upload"
)

// @title           Mancing ID API
// @version         1.0
// @description     This is the API server for Mancing ID.
// @BasePath        /api/v1
func main() {
	cfg := config.InitConfig()

	db, err := infrastructure.NewPostgresDB(cfg)
	if err != nil {
		log.Fatalf("failed to connect to postgres: %v", err)
		os.Exit(1)
	}
	defer db.Close()

	s3Client, err := infrastructure.NewS3Client(cfg)
	if err != nil {
		log.Fatalf("failed to initialise S3 client: %v", err)
		os.Exit(1)
	}

	api := application{
		config:   cfg,
		db:       db,
		uploader: upload.NewS3Uploader(s3Client, cfg.Upload),
	}

	if err := api.run(api.mount()); err != nil {
		log.Fatalf("server failed to run: %v", err)
		os.Exit(1)
	}
}
