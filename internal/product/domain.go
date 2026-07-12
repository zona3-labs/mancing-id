package product

import (
	"time"

	"github.com/google/uuid"
)

type ProductStatus string

const (
	StatusDraft    ProductStatus = "draft"
	StatusActive   ProductStatus = "active"
	StatusArchived ProductStatus = "archived"
)

func (s ProductStatus) IsValid() bool {
	switch s {
	case StatusDraft, StatusActive, StatusArchived:
		return true
	default:
		return false
	}
}

type Product struct {
	ID               uuid.UUID     `json:"id"`
	Name             string        `json:"name"`
	Slug             string        `json:"slug"`
	Description      *string       `json:"description"`
	ShortDescription *string       `json:"short_description"`
	BrandID          *uuid.UUID    `json:"brand_id"`
	Status           ProductStatus `json:"status"`
	IsFeature        bool          `json:"is_feature"`
	CreatedAt        time.Time     `json:"created_at"`
	UpdatedAt        time.Time     `json:"updated_at"`
	DeletedAt        *time.Time    `json:"deleted_at,omitempty"`
}
