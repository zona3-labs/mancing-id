package category

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	categoryDB "github.com/zona3-labs/mancing-id/internal/category/db"
	dberrors "github.com/zona3-labs/mancing-id/internal/errors"
)

type categoryPostgresRepository struct {
	queries *categoryDB.Queries
}

func NewCategoryRepository(db *sql.DB) CategoryRepository {
	return &categoryPostgresRepository{queries: categoryDB.New(db)}
}

func (r categoryPostgresRepository) CreateCategory(ctx context.Context, category *Category) error {
	parentID := uuid.NullUUID{}
	if category.ParentID != nil {
		parentID = uuid.NullUUID{UUID: *category.ParentID, Valid: true}
	}

	row, err := r.queries.CreateCategory(ctx, categoryDB.CreateCategoryParams{
		ID:       category.ID,
		ParentID: parentID,
		Name:     category.Name,
		Slug:     category.Slug,
	})
	if dberrors.IsForeignKeyViolation(err) {
		return ErrParentCategoryNotFound
	}
	if err != nil {
		return err
	}
	*category = fromDBCategory(row)
	return nil
}

func (r categoryPostgresRepository) GetCategoryByID(ctx context.Context, id uuid.UUID) (*Category, error) {
	row, err := r.queries.GetCategoryByID(ctx, id)
	if err == sql.ErrNoRows {
		return nil, ErrCategoryNotFound
	}
	if err != nil {
		return nil, err
	}
	category := fromDBCategory(row)
	return &category, nil
}

func (r categoryPostgresRepository) GetAllCategories(ctx context.Context) ([]*Category, error) {
	rows, err := r.queries.GetAllCategories(ctx)
	if err != nil {
		return nil, err
	}
	categories := make([]*Category, 0, len(rows))
	for _, row := range rows {
		category := fromDBCategory(row)
		categories = append(categories, &category)
	}
	return categories, nil
}

func (r categoryPostgresRepository) GetCategoryTree(ctx context.Context, slug string) ([]*Category, error) {
	rows, err := r.queries.GetCategoryTreeBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	categories := make([]*Category, 0, len(rows))
	for _, row := range rows {
		var parentID *uuid.UUID
		if row.ParentID.Valid {
			parentID = &row.ParentID.UUID
		}
		categories = append(categories, &Category{
			ID:        row.ID,
			ParentID:  parentID,
			Name:      row.Name,
			Slug:      row.Slug,
			Status:    CategoryStatus(row.Status),
			Version:   row.Version,
			CreatedAt: row.CreatedAt,
			UpdatedAt: row.UpdatedAt,
			DeletedAt: nullTimePointer(row.DeletedAt),
		})
	}
	return categories, nil
}

func (r categoryPostgresRepository) UpdateCategory(ctx context.Context, category *Category, expectedVersion int64) error {
	parentID := uuid.NullUUID{}
	if category.ParentID != nil {
		parentID = uuid.NullUUID{UUID: *category.ParentID, Valid: true}
	}

	row, err := r.queries.UpdateCategory(ctx, categoryDB.UpdateCategoryParams{
		ID:       category.ID,
		ParentID: parentID,
		Name:     category.Name,
		Slug:     category.Slug,
		Version:  expectedVersion,
	})
	if err == sql.ErrNoRows {
		return r.classifyWriteFailure(ctx, category.ID, expectedVersion)
	}
	if dberrors.IsForeignKeyViolation(err) {
		return ErrParentCategoryNotFound
	}
	if err != nil {
		return err
	}
	*category = fromDBCategory(row)
	return nil
}

func (r categoryPostgresRepository) CheckCategoriesByParentID(ctx context.Context, id uuid.UUID) (int64, error) {
	return r.queries.CheckCategoriesByParentID(ctx, uuid.NullUUID{UUID: id, Valid: true})
}

func (r categoryPostgresRepository) DeleteCategory(ctx context.Context, id uuid.UUID, expectedVersion int64) error {
	result, err := r.queries.DeleteCategory(ctx, categoryDB.DeleteCategoryParams{ID: id, Version: expectedVersion})
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return r.classifyWriteFailure(ctx, id, expectedVersion)
	}
	return nil
}

func (r categoryPostgresRepository) classifyWriteFailure(ctx context.Context, id uuid.UUID, expectedVersion int64) error {
	category, err := r.GetCategoryByID(ctx, id)
	if err != nil {
		return err
	}
	if category.Version != expectedVersion {
		return ErrCategoryVersionConflict
	}
	if category.Status != CategoryStatusDraft {
		return ErrCategoryNotEditable
	}
	return ErrCategoryVersionConflict
}

func fromDBCategory(row categoryDB.Category) Category {
	var parentID *uuid.UUID
	if row.ParentID.Valid {
		parentID = &row.ParentID.UUID
	}
	return Category{
		ID:        row.ID,
		ParentID:  parentID,
		Name:      row.Name,
		Slug:      row.Slug,
		Status:    CategoryStatus(row.Status),
		Version:   row.Version,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
		DeletedAt: row.DeletedAt,
	}
}

func nullTimePointer(value sql.NullTime) *time.Time {
	if !value.Valid {
		return nil
	}
	return &value.Time
}
