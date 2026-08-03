package upload

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"log"
	"mime/multipart"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/chai2010/webp"
	"github.com/google/uuid"
	"github.com/zona3-labs/mancing-id/internal/config"
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

const webpQuality = 82

type S3Uploader struct {
	client       *s3.Client
	bucket       string
	basePrefix   string
	maxSizeBytes int64
	endpoint     string
	region       string
}

func NewS3Uploader(client *s3.Client, cfg *config.UploadConfig) *S3Uploader {
	return &S3Uploader{
		client:       client,
		bucket:       cfg.S3Bucket,
		basePrefix:   cfg.S3BaseKeyPrefix,
		maxSizeBytes: cfg.MaxSizeMB * 1024 * 1024,
		endpoint:     cfg.S3Endpoint,
		region:       cfg.S3Region,
	}
}

func (u *S3Uploader) UploadTemporary(ctx context.Context, file multipart.File, header *multipart.FileHeader, ownerID uuid.UUID) (TemporaryImage, error) {
	if header == nil {
		return TemporaryImage{}, ErrInvalidFileType
	}
	if header.Size > u.maxSizeBytes {
		return TemporaryImage{}, ErrFileTooLarge
	}

	buf := make([]byte, 512)
	n, err := file.Read(buf)
	if err != nil {
		return TemporaryImage{}, fmt.Errorf("upload: failed to read file header: %w", err)
	}
	contentType := strings.TrimSpace(strings.Split(http.DetectContentType(buf[:n]), ";")[0])
	if !allowedMIMETypes[contentType] {
		return TemporaryImage{}, ErrInvalidFileType
	}
	if _, err := file.Seek(0, 0); err != nil {
		return TemporaryImage{}, fmt.Errorf("upload: failed to seek file: %w", err)
	}

	img, _, err := image.Decode(file)
	if err != nil {
		return TemporaryImage{}, fmt.Errorf("%w: %s", ErrImageDecode, err.Error())
	}
	var webpBuf bytes.Buffer
	if err := webp.Encode(&webpBuf, img, &webp.Options{Lossless: false, Quality: webpQuality}); err != nil {
		return TemporaryImage{}, fmt.Errorf("upload: failed to encode webp: %w", err)
	}

	id := uuid.New()
	basePrefix := strings.Trim(u.basePrefix, "/")
	temporaryKey := fmt.Sprintf("%s/temporary/%s/%s.webp", basePrefix, ownerID, id)
	ownedKey := fmt.Sprintf("%s/brands/%s/%s.webp", basePrefix, ownerID, id)
	_, err = u.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        aws.String(u.bucket),
		Key:           aws.String(temporaryKey),
		Body:          bytes.NewReader(webpBuf.Bytes()),
		ContentType:   aws.String("image/webp"),
		ContentLength: aws.Int64(int64(webpBuf.Len())),
	})
	if err != nil {
		return TemporaryImage{}, fmt.Errorf("upload: failed to put object: %w", err)
	}

	return TemporaryImage{
		ID:           id.String(),
		URL:          u.publicURL(ownedKey),
		TemporaryKey: temporaryKey,
		OwnedKey:     ownedKey,
		UploadedAt:   time.Now().UTC(),
	}, nil
}

func (u *S3Uploader) FinalizeTemporary(ctx context.Context, image TemporaryImage) error {
	_, err := u.client.CopyObject(ctx, &s3.CopyObjectInput{
		Bucket:      aws.String(u.bucket),
		CopySource:  aws.String(url.PathEscape(u.bucket + "/" + image.TemporaryKey)),
		Key:         aws.String(image.OwnedKey),
		ContentType: aws.String("image/webp"),
	})
	if err != nil {
		return fmt.Errorf("upload: failed to finalize object: %w", err)
	}
	if _, err := u.client.DeleteObject(ctx, &s3.DeleteObjectInput{Bucket: aws.String(u.bucket), Key: aws.String(image.TemporaryKey)}); err != nil {
		return fmt.Errorf("upload: failed to remove temporary object: %w", err)
	}
	return nil
}

func (u *S3Uploader) ScheduleDelete(ctx context.Context, objectURL string) error {
	key, managed := u.managedObjectKey(objectURL)
	if !managed {
		return nil
	}
	if _, err := u.client.DeleteObject(ctx, &s3.DeleteObjectInput{Bucket: aws.String(u.bucket), Key: aws.String(key)}); err != nil {
		return fmt.Errorf("upload: failed to schedule object cleanup: %w", err)
	}
	return nil
}

func (u *S3Uploader) CleanStaleTemporary(ctx context.Context, before time.Time) (int, error) {
	prefix := strings.Trim(u.basePrefix, "/") + "/temporary/"
	paginator := s3.NewListObjectsV2Paginator(u.client, &s3.ListObjectsV2Input{Bucket: aws.String(u.bucket), Prefix: aws.String(prefix)})
	cleaned := 0
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return cleaned, fmt.Errorf("upload: failed to list temporary objects: %w", err)
		}
		for _, object := range page.Contents {
			if object.LastModified == nil || !object.LastModified.Before(before) {
				continue
			}
			if _, err := u.client.DeleteObject(ctx, &s3.DeleteObjectInput{Bucket: aws.String(u.bucket), Key: object.Key}); err != nil {
				return cleaned, fmt.Errorf("upload: failed to clean temporary object: %w", err)
			}
			cleaned++
		}
	}
	return cleaned, nil
}

func (u *S3Uploader) StartTemporaryCleanup(ctx context.Context, interval, maxAge time.Duration) {
	if interval <= 0 || maxAge < 0 {
		return
	}
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				if _, err := u.CleanStaleTemporary(ctx, time.Now().UTC().Add(-maxAge)); err != nil {
					log.Printf("temporary media cleanup failed: %v", err)
				}
			case <-ctx.Done():
				return
			}
		}
	}()
}

func (u *S3Uploader) managedObjectKey(objectURL string) (string, bool) {
	parsed, err := url.Parse(objectURL)
	if err != nil {
		return "", false
	}
	basePrefix := strings.Trim(u.basePrefix, "/")
	var key string
	if u.endpoint != "" {
		endpoint, err := url.Parse(u.endpoint)
		if err != nil || parsed.Scheme != endpoint.Scheme || parsed.Host != endpoint.Host {
			return "", false
		}
		prefix := "/" + strings.Trim(u.bucket, "/") + "/"
		if !strings.HasPrefix(parsed.Path, prefix) {
			return "", false
		}
		key = strings.TrimPrefix(parsed.Path, prefix)
	} else {
		expectedHost := fmt.Sprintf("%s.s3.%s.amazonaws.com", u.bucket, u.region)
		if parsed.Host != expectedHost {
			return "", false
		}
		key = strings.TrimPrefix(parsed.Path, "/")
	}
	if !strings.HasPrefix(key, basePrefix+"/brands/") {
		return "", false
	}
	return key, true
}

func (u *S3Uploader) publicURL(objectKey string) string {
	if u.endpoint != "" {
		return fmt.Sprintf("%s/%s/%s", strings.TrimRight(u.endpoint, "/"), u.bucket, objectKey)
	}
	return fmt.Sprintf("https://%s.s3.%s.amazonaws.com/%s", u.bucket, u.region, objectKey)
}
