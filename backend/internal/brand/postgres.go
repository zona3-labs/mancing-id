package brand

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	brandDB "github.com/zona3-labs/mancing-id/internal/brand/db"
	"github.com/zona3-labs/mancing-id/internal/transaction"
)

type brandPostgresRepository struct {
	db      *sql.DB
	queries *brandDB.Queries
}

func NewBrandRepository(db *sql.DB) BrandRepository {
	return &brandPostgresRepository{db: db, queries: brandDB.New(db)}
}

func NewCatalogBrandRepository(db *sql.DB) CatalogBrandRepository {
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

func (r brandPostgresRepository) GetBrandByIDForUpdate(ctx context.Context, tx transaction.DBTX, id uuid.UUID) (*Brand, error) {
	var row brandDB.Brand
	err := tx.QueryRowContext(ctx, `
		SELECT id, name, slug, logo_path, status, version, created_at, updated_at, deleted_at
		FROM brands
		WHERE id = $1 AND deleted_at IS NULL
		FOR UPDATE`, id).Scan(
		&row.ID, &row.Name, &row.Slug, &row.LogoPath, &row.Status, &row.Version,
		&row.CreatedAt, &row.UpdatedAt, &row.DeletedAt,
	)
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

func (r brandPostgresRepository) AssociateLogo(ctx context.Context, tx transaction.DBTX, id uuid.UUID, logoPath string) (*string, error) {
	var previous sql.NullString
	if err := tx.QueryRowContext(ctx, `
		SELECT logo_path
		FROM brands
		WHERE id = $1 AND deleted_at IS NULL
		FOR UPDATE`, id).Scan(&previous); err == sql.ErrNoRows {
		return nil, ErrBrandNotFound
	} else if err != nil {
		return nil, err
	}

	result, err := tx.ExecContext(ctx, `
		UPDATE brands
		SET logo_path = $2, updated_at = NOW()
		WHERE id = $1 AND deleted_at IS NULL`, id, logoPath)
	if err != nil {
		return nil, err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return nil, err
	}
	if affected != 1 {
		return nil, ErrBrandNotFound
	}
	if !previous.Valid {
		return nil, nil
	}
	return &previous.String, nil
}

func (r brandPostgresRepository) DeleteBrandInTransaction(ctx context.Context, tx transaction.DBTX, id uuid.UUID, expectedVersion int64) error {
	result, err := tx.ExecContext(ctx, `
		UPDATE brands
		SET deleted_at = NOW(), version = version + 1, updated_at = NOW()
		WHERE id = $1 AND version = $2 AND status = 'draft' AND deleted_at IS NULL`, id, expectedVersion)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
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
