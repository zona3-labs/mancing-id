package brand

import (
	"context"

	"github.com/google/uuid"
)

type BrandRepository interface {
	CreateBrand(ctx context.Context, brand *Brand) error
	GetAllBrands(ctx context.Context) ([]*Brand, error)
	GetBrandBySlug(ctx context.Context, slug string) (*Brand, error)
	UpdateBrand(ctx context.Context, Brand *Brand) error
	UpdateBrandLogo(ctx context.Context, id uuid.UUID, logoPath string) error
	DeleteBrand(ctx context.Context, id uuid.UUID) error
}
