package brand

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	brandDB "github.com/zona3-labs/mancing-id/internal/brand/db"
)

type brandPostgresRepository struct {
	db      *sql.DB
	queries *brandDB.Queries
}

func NewBrandRepository(db *sql.DB) BrandRepository {
	return &brandPostgresRepository{db: db, queries: brandDB.New(db)}
}

func (r brandPostgresRepository) CreateBrand(ctx context.Context, brand *Brand) error {
	row, err := r.queries.CreateBrand(ctx, brandDB.CreateBrandParams{
		ID:       brand.ID,
		Name:     brand.Name,
		Slug:     brand.Slug,
		LogoPath: brand.LogoPath,
	})
	if err != nil {
		return err
	}
	*brand = fromDBBrand(row)
	return nil
}

func (r brandPostgresRepository) GetBrandByID(ctx context.Context, id uuid.UUID) (*Brand, error) {
	row, err := r.queries.GetBrandByID(ctx, id)
	if err == sql.ErrNoRows {
		return nil, ErrBrandNotFound
	}
	if err != nil {
		return nil, err
	}
	brand := fromDBBrand(row)
	return &brand, nil
}

func (r brandPostgresRepository) GetAllBrands(ctx context.Context) ([]*Brand, error) {
	rows, err := r.queries.GetAllBrands(ctx)
	return mapDBBrands(rows, err)
}

func (r brandPostgresRepository) GetPublicBrands(ctx context.Context) ([]*Brand, error) {
	rows, err := r.queries.GetPublicBrands(ctx)
	return mapDBBrands(rows, err)
}

func (r brandPostgresRepository) GetBrandBySlug(ctx context.Context, slug string) (*Brand, error) {
	row, err := r.queries.GetBrandBySlug(ctx, slug)
	if err == sql.ErrNoRows {
		return nil, ErrBrandNotFound
	}
	if err != nil {
		return nil, err
	}
	brand := fromDBBrand(row)
	return &brand, nil
}

func (r brandPostgresRepository) GetPublicBrandBySlug(ctx context.Context, slug string) (*Brand, error) {
	row, err := r.queries.GetPublicBrandBySlug(ctx, slug)
	if err == sql.ErrNoRows {
		return nil, ErrBrandNotFound
	}
	if err != nil {
		return nil, err
	}
	brand := fromDBBrand(row)
	return &brand, nil
}

func (r brandPostgresRepository) UpdateBrand(ctx context.Context, brand *Brand, expectedVersion int64) error {
	row, err := r.queries.UpdateBrand(ctx, brandDB.UpdateBrandParams{
		ID:       brand.ID,
		Name:     brand.Name,
		Slug:     brand.Slug,
		LogoPath: brand.LogoPath,
		Version:  expectedVersion,
	})
	if err == sql.ErrNoRows {
		return r.classifyMutationFailure(ctx, brand.ID, expectedVersion, BrandStatusDraft)
	}
	if err != nil {
		return err
	}
	*brand = fromDBBrand(row)
	return nil
}

func (r brandPostgresRepository) ActivateBrand(ctx context.Context, id uuid.UUID, expectedVersion int64) error {
	result, err := r.queries.ActivateBrand(ctx, brandDB.ActivateBrandParams{ID: id, Version: expectedVersion})
	return r.classifyTransitionResult(ctx, id, expectedVersion, BrandStatusDraft, result, err)
}

func (r brandPostgresRepository) DeactivateBrand(ctx context.Context, id uuid.UUID, expectedVersion int64) error {
	result, err := r.queries.DeactivateBrand(ctx, brandDB.DeactivateBrandParams{ID: id, Version: expectedVersion})
	return r.classifyTransitionResult(ctx, id, expectedVersion, BrandStatusActive, result, err)
}

func (r brandPostgresRepository) ReactivateBrand(ctx context.Context, id uuid.UUID, expectedVersion int64) error {
	result, err := r.queries.ReactivateBrand(ctx, brandDB.ReactivateBrandParams{ID: id, Version: expectedVersion})
	return r.classifyTransitionResult(ctx, id, expectedVersion, BrandStatusInactive, result, err)
}

func (r brandPostgresRepository) CountProductsByBrandID(ctx context.Context, id uuid.UUID) (int64, error) {
	return r.queries.CountProductsByBrandID(ctx, uuid.NullUUID{UUID: id, Valid: true})
}

func (r brandPostgresRepository) UpdateBrandLogo(ctx context.Context, id uuid.UUID, logoPath string) error {
	result, err := r.queries.UpdateBrandLogo(ctx, brandDB.UpdateBrandLogoParams{ID: id, LogoPath: &logoPath})
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

func (r brandPostgresRepository) DeleteBrand(ctx context.Context, id uuid.UUID, expectedVersion int64) error {
	result, err := r.queries.DeleteBrand(ctx, brandDB.DeleteBrandParams{ID: id, Version: expectedVersion})
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		brand, lookupErr := r.GetBrandByID(ctx, id)
		if lookupErr != nil {
			return lookupErr
		}
		if brand.Version != expectedVersion {
			return ErrBrandVersionConflict
		}
		if brand.Status != BrandStatusDraft {
			return ErrBrandNotEditable
		}
		products, countErr := r.CountProductsByBrandID(ctx, id)
		if countErr != nil {
			return countErr
		}
		if products > 0 {
			return ErrBrandHasProducts
		}
		return ErrBrandVersionConflict
	}
	return nil
}

func (r brandPostgresRepository) classifyTransitionResult(ctx context.Context, id uuid.UUID, expectedVersion int64, expectedStatus BrandStatus, result sql.Result, err error) error {
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 1 {
		return nil
	}
	return r.classifyMutationFailure(ctx, id, expectedVersion, expectedStatus)
}

func (r brandPostgresRepository) classifyMutationFailure(ctx context.Context, id uuid.UUID, expectedVersion int64, expectedStatus BrandStatus) error {
	brand, err := r.GetBrandByID(ctx, id)
	if err != nil {
		return err
	}
	if brand.Version != expectedVersion {
		return ErrBrandVersionConflict
	}
	if brand.Status != expectedStatus {
		return ErrBrandNotEditable
	}
	return ErrBrandVersionConflict
}

func mapDBBrands(rows []brandDB.Brand, err error) ([]*Brand, error) {
	if err != nil {
		return nil, err
	}
	brands := make([]*Brand, 0, len(rows))
	for _, row := range rows {
		brand := fromDBBrand(row)
		brands = append(brands, &brand)
	}
	return brands, nil
}

func fromDBBrand(row brandDB.Brand) Brand {
	return Brand{
		ID:        row.ID,
		Name:      row.Name,
		Slug:      row.Slug,
		LogoPath:  row.LogoPath,
		Status:    BrandStatus(row.Status),
		Version:   row.Version,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
		DeletedAt: row.DeletedAt,
	}
}
