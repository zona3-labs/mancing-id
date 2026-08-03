package brand

import (
	"context"

	"github.com/google/uuid"
	"github.com/zona3-labs/mancing-id/internal/transaction"
)

type BrandRepository interface {
	CreateBrand(ctx context.Context, brand *Brand) error
	GetBrandByID(ctx context.Context, id uuid.UUID) (*Brand, error)
	GetAllBrands(ctx context.Context) ([]*Brand, error)
	GetPublicBrands(ctx context.Context) ([]*Brand, error)
	GetBrandBySlug(ctx context.Context, slug string) (*Brand, error)
	GetPublicBrandBySlug(ctx context.Context, slug string) (*Brand, error)
	UpdateBrand(ctx context.Context, brand *Brand, expectedVersion int64) error
	ActivateBrand(ctx context.Context, id uuid.UUID, expectedVersion int64) error
	DeactivateBrand(ctx context.Context, id uuid.UUID, expectedVersion int64) error
	ReactivateBrand(ctx context.Context, id uuid.UUID, expectedVersion int64) error
	AssociateLogo(ctx context.Context, tx transaction.DBTX, id uuid.UUID, logoPath string) (*string, error)
}

type ProductBrandRepository interface {
	GetBrandByID(context.Context, uuid.UUID) (*Brand, error)
	GetBrandByIDForUpdate(context.Context, transaction.DBTX, uuid.UUID) (*Brand, error)
}

type CatalogBrandRepository interface {
	GetBrandByID(context.Context, uuid.UUID) (*Brand, error)
	GetBrandByIDForUpdate(context.Context, transaction.DBTX, uuid.UUID) (*Brand, error)
	DeleteBrandInTransaction(context.Context, transaction.DBTX, uuid.UUID, int64) error
}
