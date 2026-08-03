package brand

import (
	"time"

	"github.com/google/uuid"
)

type Brand struct {
	ID        uuid.UUID   `json:"id"`
	Name      string      `json:"name"`
	Slug      string      `json:"slug"`
	LogoPath  *string     `json:"logo_path"`
	Status    BrandStatus `json:"status"`
	Version   int64       `json:"version"`
	CreatedAt time.Time   `json:"created_at"`
	UpdatedAt time.Time   `json:"updated_at"`
	DeletedAt *time.Time  `json:"deleted_at,omitempty"`
}

type BrandStatus string

const (
	BrandStatusDraft    BrandStatus = "draft"
	BrandStatusActive   BrandStatus = "active"
	BrandStatusInactive BrandStatus = "inactive"
)
