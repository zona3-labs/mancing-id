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
	Version          int64         `json:"version"`
	IsFeature        bool          `json:"is_feature"`
	CreatedAt        time.Time     `json:"created_at"`
	UpdatedAt        time.Time     `json:"updated_at"`
	DeletedAt        *time.Time    `json:"deleted_at,omitempty"`
}

type ProductOption struct {
	ID        uuid.UUID  `json:"id"`
	ProductID uuid.UUID  `json:"product_id"`
	Name      string     `json:"name"`
	Position  int16      `json:"position"`
	IsActive  bool       `json:"is_active"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`
}

type ProductOptionValue struct {
	ID              uuid.UUID  `json:"id"`
	ProductOptionID uuid.UUID  `json:"product_option_id"`
	Value           string     `json:"value"`
	Position        int16      `json:"position"`
	IsActive        bool       `json:"is_active"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	DeletedAt       *time.Time `json:"deleted_at,omitempty"`
}

type ProductOptionWithValues struct {
	ProductOption
	Values []*ProductOptionValue `json:"values"`
}

type ProductImage struct {
	ID        uuid.UUID `json:"id"`
	ProductID uuid.UUID `json:"product_id"`
	Url       string    `json:"url"`
	AltText   *string   `json:"alt_text,omitempty"`
	Position  int16     `json:"position"`
	IsPrimary bool      `json:"is_primary"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type ProductDetail struct {
	Product
	Brand    *BrandSummary              `json:"brand,omitempty"`
	Options  []*ProductOptionWithValues `json:"options"`
	Images   []*ProductImage            `json:"images"`
	Variants []*ProductVariantDetail    `json:"variants"`
}

type BrandSummary struct {
	ID       uuid.UUID `json:"id"`
	Name     string    `json:"name"`
	Slug     string    `json:"slug"`
	LogoPath *string   `json:"logo_path,omitempty"`
}

type VariantStatus string

const (
	VariantStatusDraft    VariantStatus = "draft"
	VariantStatusActive   VariantStatus = "active"
	VariantStatusArchived VariantStatus = "archived"
)

func (s VariantStatus) IsValid() bool {
	switch s {
	case VariantStatusDraft, VariantStatusActive, VariantStatusArchived:
		return true
	default:
		return false
	}
}

type ProductVariant struct {
	ID        uuid.UUID     `json:"id"`
	ProductID uuid.UUID     `json:"product_id"`
	ImageID   *uuid.UUID    `json:"image_id,omitempty"`
	Sku       string        `json:"sku"`
	Price     string        `json:"price"`
	Stock     int32         `json:"stock"`
	Weight    *string       `json:"weight,omitempty"`
	Status    VariantStatus `json:"status"`
	CreatedAt time.Time     `json:"created_at"`
	UpdatedAt time.Time     `json:"updated_at"`
	DeletedAt *time.Time    `json:"deleted_at,omitempty"`
}

type ProductVariantDetail struct {
	ProductVariant
	OptionValues []*ProductOptionValue `json:"option_values"`
}
