package upload_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"mime/multipart"
	"os"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
	"github.com/zona3-labs/mancing-id/internal/config"
	"github.com/zona3-labs/mancing-id/internal/upload"
)

func TestS3UploaderRejectsUnsupportedImageTypeBeforeStorage(t *testing.T) {
	uploader := upload.NewS3Uploader(&s3.Client{}, &config.UploadConfig{MaxSizeMB: 1})
	file, header := testFile(t, []byte("not an image"), "text.txt")

	_, err := uploader.UploadTemporary(context.Background(), file, header, uuid.New())
	if !errors.Is(err, upload.ErrInvalidFileType) {
		t.Fatalf("error = %v, want invalid file type", err)
	}
}

func TestS3UploaderRejectsOversizedImageBeforeStorage(t *testing.T) {
	uploader := upload.NewS3Uploader(&s3.Client{}, &config.UploadConfig{MaxSizeMB: 1})
	file, header := testFile(t, []byte("image data"), "large.png")
	header.Size = 2 * 1024 * 1024

	_, err := uploader.UploadTemporary(context.Background(), file, header, uuid.New())
	if !errors.Is(err, upload.ErrFileTooLarge) {
		t.Fatalf("error = %v, want file too large", err)
	}
}

func TestS3UploaderRejectsUndecodableImageBeforeStorage(t *testing.T) {
	uploader := upload.NewS3Uploader(&s3.Client{}, &config.UploadConfig{MaxSizeMB: 1})
	file, header := testFile(t, []byte("\x89PNG\r\n\x1a\nnot a png"), "broken.png")

	_, err := uploader.UploadTemporary(context.Background(), file, header, uuid.New())
	if !errors.Is(err, upload.ErrImageDecode) {
		t.Fatalf("error = %v, want image decode error", err)
	}
}

func testFile(t *testing.T, content []byte, filename string) (multipart.File, *multipart.FileHeader) {
	t.Helper()
	file, err := os.CreateTemp(t.TempDir(), "upload-*")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.Copy(file, bytes.NewReader(content)); err != nil {
		t.Fatal(err)
	}
	if _, err := file.Seek(0, io.SeekStart); err != nil {
		t.Fatal(err)
	}
	return file, &multipart.FileHeader{Filename: filename, Size: int64(len(content))}
}
