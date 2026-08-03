package upload

import (
	"context"
	"mime/multipart"
	"time"

	"github.com/google/uuid"
)

type TemporaryImage struct {
	ID           string
	URL          string
	TemporaryKey string
	OwnedKey     string
	UploadedAt   time.Time
}

type MediaAdapter interface {
	UploadTemporary(context.Context, multipart.File, *multipart.FileHeader, uuid.UUID) (TemporaryImage, error)
	FinalizeTemporary(context.Context, TemporaryImage) error
	ScheduleDelete(context.Context, string) error
	CleanStaleTemporary(context.Context, time.Time) (int, error)
}
