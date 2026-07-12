package brand

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	brandDb "github.com/zone3-labs/mancing-id/internal/brand/db"
)

type brandPostgresRepository struct {
	queries *brandDb.Queries
}

func NewBrandRepository(db *sql.DB) BrandRepository {
	return &brandPostgresRepository{queries: brandDb.New(db)}
}

func (b brandPostgresRepository) CreateBrand(ctx context.Context, brand *Brand) error {
	return b.queries.CreateBrand(ctx, brandDb.CreateBrandParams{
		ID:       brand.ID,
		Name:     brand.Name,
		Slug:     brand.Slug,
		LogoPath: brand.LogoPath,
	})
}

func (b brandPostgresRepository) GetAllBrands(ctx context.Context) ([]*Brand, error) {
	rows, err := b.queries.GetAllBrands(ctx)
	if err != nil {
		return nil, err
	}
	brands := make([]*Brand, 0, len(rows))
	for _, row := range rows {
		brands = append(brands, &Brand{
			ID:        row.ID,
			Name:      row.Name,
			Slug:      row.Slug,
			LogoPath:  row.LogoPath,
			IsActive:  row.IsActive,
			CreatedAt: row.CreatedAt,
			UpdatedAt: row.UpdatedAt,
			DeletedAt: row.DeletedAt,
		})
	}
	return brands, nil
}

func (b brandPostgresRepository) GetBrandBySlug(ctx context.Context, slug string) (*Brand, error) {
	row, err := b.queries.GetBrandBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	return &Brand{
		ID:        row.ID,
		Name:      row.Name,
		Slug:      row.Slug,
		LogoPath:  row.LogoPath,
		IsActive:  row.IsActive,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
		DeletedAt: row.DeletedAt,
	}, nil
}

func (b brandPostgresRepository) UpdateBrand(ctx context.Context, brand *Brand) error {
	return b.queries.UpdateBrand(ctx, brandDb.UpdateBrandParams{
		ID:       brand.ID,
		Name:     brand.Name,
		Slug:     brand.Slug,
		LogoPath: brand.LogoPath,
		IsActive: brand.IsActive,
	})
}

func (b brandPostgresRepository) UpdateBrandLogo(ctx context.Context, id uuid.UUID, logoPath string) error {
	result, err := b.queries.UpdateBrandLogo(ctx, brandDb.UpdateBrandLogoParams{
		ID:       id,
		LogoPath: &logoPath,
	})
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrBrandNotFound
	}
	return nil
}

func (b brandPostgresRepository) DeleteBrand(ctx context.Context, id uuid.UUID) error {
	result, err := b.queries.DeleteBrand(ctx, id)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrBrandNotFound
	}
	return nil
}
