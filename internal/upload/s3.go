package upload

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg" // register jpeg decoder
	_ "image/png"  // register png decoder
	"mime/multipart"
	"net/http"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/chai2010/webp"
	"github.com/google/uuid"
	"github.com/zone3-labs/mancing-id/internal/config"
)

var (
	ErrInvalidFileType = errors.New("upload: file type not allowed; accepted types are image/jpeg, image/png, image/webp")
	ErrFileTooLarge    = errors.New("upload: file exceeds maximum allowed size")
	ErrImageDecode     = errors.New("upload: failed to decode image")
)

var allowedMIMETypes = map[string]bool{
	"image/jpeg": true,
	"image/png":  true,
	"image/webp": true,
}

// webpQuality is the lossy WebP compression quality (0–100).
// 82 is a good balance between size and visual fidelity.
const webpQuality = 82

// S3Uploader implements FileUploader using an S3-compatible backend (AWS or MinIO).
type S3Uploader struct {
	client       *s3.Client
	bucket       string
	basePrefix   string
	maxSizeBytes int64
	endpoint     string // non-empty → MinIO; used to build public URL
	region       string
}

func NewS3Uploader(client *s3.Client, cfg *config.UploadConfig) FileUploader {
	return &S3Uploader{
		client:       client,
		bucket:       cfg.S3Bucket,
		basePrefix:   cfg.S3BaseKeyPrefix,
		maxSizeBytes: cfg.MaxSizeMB * 1024 * 1024,
		endpoint:     cfg.S3Endpoint,
		region:       cfg.S3Region,
	}
}

// Upload converts the file to WebP, compresses it, stores it in S3/MinIO,
// and returns the public URL.
func (u *S3Uploader) Upload(ctx context.Context, file multipart.File, header *multipart.FileHeader, keyPrefix string) (string, error) {
	// 1. Enforce raw upload size limit.
	if header.Size > u.maxSizeBytes {
		return "", ErrFileTooLarge
	}

	// 2. Detect MIME type from the first 512 bytes.
	buf := make([]byte, 512)
	n, err := file.Read(buf)
	if err != nil {
		return "", fmt.Errorf("upload: failed to read file header: %w", err)
	}
	contentType := strings.TrimSpace(strings.Split(http.DetectContentType(buf[:n]), ";")[0])

	if !allowedMIMETypes[contentType] {
		return "", ErrInvalidFileType
	}

	// 3. Seek back to beginning after MIME sniff.
	if _, err := file.Seek(0, 0); err != nil {
		return "", fmt.Errorf("upload: failed to seek file: %w", err)
	}

	// 4. Decode the source image.
	img, _, err := image.Decode(file)
	if err != nil {
		return "", fmt.Errorf("%w: %s", ErrImageDecode, err.Error())
	}

	// 5. Encode to WebP with lossy compression.
	var webpBuf bytes.Buffer
	if err := webp.Encode(&webpBuf, img, &webp.Options{
		Lossless: false,
		Quality:  webpQuality,
	}); err != nil {
		return "", fmt.Errorf("upload: failed to encode webp: %w", err)
	}

	// 6. Build deterministic object key (always .webp output).
	objectKey := fmt.Sprintf("%s/%s/%s.webp",
		strings.Trim(u.basePrefix, "/"),
		strings.Trim(keyPrefix, "/"),
		uuid.New().String(),
	)

	// 7. Upload the WebP bytes to S3 / MinIO.
	_, err = u.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(u.bucket),
		Key:           aws.String(objectKey),
		Body:          bytes.NewReader(webpBuf.Bytes()),
		ContentType:   aws.String("image/webp"),
		ContentLength: aws.Int64(int64(webpBuf.Len())),
	})
	if err != nil {
		return "", fmt.Errorf("upload: failed to put object: %w", err)
	}

	// 8. Build the public URL.
	var publicURL string
	if u.endpoint != "" {
		// MinIO / custom endpoint: path-style URL.
		publicURL = fmt.Sprintf("%s/%s/%s",
			strings.TrimRight(u.endpoint, "/"),
			u.bucket,
			objectKey,
		)
	} else {
		// AWS S3: virtual-hosted-style URL.
		publicURL = fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s",
			u.bucket,
			u.region,
			objectKey,
		)
	}

	return publicURL, nil
}
