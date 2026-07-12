package category

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
	categoryDb "github.com/zone3-labs/mancing-id/internal/category/db"
	dberrors "github.com/zone3-labs/mancing-id/internal/errors"
)

type categoryPostgresRepository struct {
	queries *categoryDb.Queries
}

func NewCategoryRepository(db *sql.DB) CategoryRepository {
	return &categoryPostgresRepository{queries: categoryDb.New(db)}
}

func (c categoryPostgresRepository) CreateCategory(ctx context.Context, category *Category) error {
	parentID := uuid.NullUUID{}
	if category.ParentID != nil {
		parentID = uuid.NullUUID{UUID: *category.ParentID, Valid: true}
	}
	err := c.queries.CreateCategory(ctx, categoryDb.CreateCategoryParams{
		ID:       category.ID,
		ParentID: parentID,
		Name:     category.Name,
		Slug:     category.Slug,
	})
	if dberrors.IsForeignKeyViolation(err) {
		return ErrParentCategoryNotFound
	}
	return err
}

func (c categoryPostgresRepository) GetAllCategories(ctx context.Context) ([]*Category, error) {
	rows, err := c.queries.GetAllCategories(ctx)
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
			IsActive:  row.IsActive,
			CreatedAt: row.CreatedAt,
			UpdatedAt: row.UpdatedAt,
			DeletedAt: row.DeletedAt,
		})
	}
	return categories, nil
}

func (c categoryPostgresRepository) GetCategoryTree(ctx context.Context, slug string) ([]*Category, error) {
	rows, err := c.queries.GetCategoryTreeBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	categories := make([]*Category, 0, len(rows))
	for _, row := range rows {
		var parentID *uuid.UUID
		if row.ParentID.Valid {
			parentID = &row.ParentID.UUID
		}
		var deletedAt *time.Time
		if row.DeletedAt.Valid {
			deletedAt = &row.DeletedAt.Time
		}
		categories = append(categories, &Category{
			ID:        row.ID,
			ParentID:  parentID,
			Name:      row.Name,
			Slug:      row.Slug,
			IsActive:  row.IsActive,
			CreatedAt: row.CreatedAt,
			UpdatedAt: row.UpdatedAt,
			DeletedAt: deletedAt,
		})
	}
	return categories, nil
}

func (c categoryPostgresRepository) UpdateCategory(ctx context.Context, category *Category) error {
	return c.queries.UpdateCategory(ctx, categoryDb.UpdateCategoryParams{
		ID:       category.ID,
		Name:     category.Name,
		Slug:     category.Slug,
		IsActive: category.IsActive,
	})
}

func (c categoryPostgresRepository) CheckActiveCategoriesByParentId(ctx context.Context, id uuid.UUID) (int64, error) {
	nullID := uuid.NullUUID{UUID: id, Valid: true}
	return c.queries.CheckActiveCategoriesByParentId(ctx, nullID)
}

func (c categoryPostgresRepository) DeleteCategory(ctx context.Context, id uuid.UUID) error {
	result, err := c.queries.DeleteCategory(ctx, id)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrCategoryNotFound
	}
	return nil
}
