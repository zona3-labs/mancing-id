package upload

import (
	"context"
	"mime/multipart"
)

// FileUploader is the interface that wraps the Upload method.
// Implementations include S3Uploader (for AWS S3 and MinIO).
type FileUploader interface {
	// Upload stores the given file and returns its publicly accessible URL.
	// keyPrefix is used to namespace the object key (e.g. the brand's UUID).
	Upload(ctx context.Context, file multipart.File, header *multipart.FileHeader, keyPrefix string) (string, error)
}
