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
	db      *sql.DB
	queries *categoryDB.Queries
}

const categoryHierarchyLockKey int64 = 4242

func NewCategoryRepository(db *sql.DB) CategoryRepository {
	return &categoryPostgresRepository{db: db, queries: categoryDB.New(db)}
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

func (r categoryPostgresRepository) GetActiveCategories(ctx context.Context) ([]*Category, error) {
	rows, err := r.queries.GetActiveCategories(ctx)
	if err != nil {
		return nil, err
	}
	categories := make([]*Category, 0, len(rows))
	for _, row := range rows {
		categories = append(categories, fromTreeRow(row.ID, row.ParentID, row.Name, row.Slug, row.Status, row.Version, row.CreatedAt, row.UpdatedAt, row.DeletedAt))
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
		categories = append(categories, fromTreeRow(row.ID, row.ParentID, row.Name, row.Slug, row.Status, row.Version, row.CreatedAt, row.UpdatedAt, row.DeletedAt))
	}
	return categories, nil
}

func (r categoryPostgresRepository) GetActiveCategoryTree(ctx context.Context, slug string) ([]*Category, error) {
	rows, err := r.queries.GetActiveCategoryTreeBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	categories := make([]*Category, 0, len(rows))
	for _, row := range rows {
		categories = append(categories, fromTreeRow(row.ID, row.ParentID, row.Name, row.Slug, row.Status, row.Version, row.CreatedAt, row.UpdatedAt, row.DeletedAt))
	}
	return categories, nil
}

func (r categoryPostgresRepository) UpdateCategory(ctx context.Context, category *Category, expectedVersion int64) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, "SELECT pg_advisory_xact_lock($1)", categoryHierarchyLockKey); err != nil {
		return err
	}
	queries := r.queries.WithTx(tx)
	parentID := uuid.NullUUID{}
	if category.ParentID != nil {
		parentID = uuid.NullUUID{UUID: *category.ParentID, Valid: true}
	}

	row, err := queries.UpdateCategory(ctx, categoryDB.UpdateCategoryParams{
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
	return tx.Commit()
}

func (r categoryPostgresRepository) ActivateCategory(ctx context.Context, id uuid.UUID, expectedVersion int64) error {
	result, err := r.runHierarchyMutation(ctx, func(queries *categoryDB.Queries) (sql.Result, error) {
		return queries.ActivateCategory(ctx, categoryDB.ActivateCategoryParams{ID: id, Version: expectedVersion})
	})
	if err != nil {
		return err
	}
	return r.classifyTransitionResult(ctx, id, expectedVersion, result)
}

func (r categoryPostgresRepository) RetireCategory(ctx context.Context, id uuid.UUID, expectedVersion int64) error {
	result, err := r.runHierarchyMutation(ctx, func(queries *categoryDB.Queries) (sql.Result, error) {
		return queries.RetireCategory(ctx, categoryDB.RetireCategoryParams{ID: id, Version: expectedVersion})
	})
	if err != nil {
		return err
	}
	return r.classifyTransitionResult(ctx, id, expectedVersion, result)
}

func (r categoryPostgresRepository) CheckCategoriesByParentID(ctx context.Context, id uuid.UUID) (int64, error) {
	return r.queries.CheckCategoriesByParentID(ctx, uuid.NullUUID{UUID: id, Valid: true})
}

func (r categoryPostgresRepository) CheckNonRetiredCategoriesByParentID(ctx context.Context, id uuid.UUID) (int64, error) {
	return r.queries.CheckNonRetiredCategoriesByParentID(ctx, uuid.NullUUID{UUID: id, Valid: true})
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
		if category.Status == CategoryStatusActive {
			children, childErr := r.CheckNonRetiredCategoriesByParentID(ctx, id)
			if childErr != nil {
				return childErr
			}
			if children > 0 {
				return ErrCategoryHasChildren
			}
		}
		return ErrCategoryNotEditable
	}
	return ErrCategoryVersionConflict
}

func (r categoryPostgresRepository) classifyTransitionResult(ctx context.Context, id uuid.UUID, expectedVersion int64, result sql.Result) error {
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 1 {
		return nil
	}
	return r.classifyWriteFailure(ctx, id, expectedVersion)
}

func (r categoryPostgresRepository) runHierarchyMutation(ctx context.Context, mutation func(*categoryDB.Queries) (sql.Result, error)) (sql.Result, error) {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, "SELECT pg_advisory_xact_lock($1)", categoryHierarchyLockKey); err != nil {
		return nil, err
	}
	result, err := mutation(r.queries.WithTx(tx))
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return result, nil
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

func fromTreeRow(id uuid.UUID, parentID uuid.NullUUID, name, slug string, status categoryDB.CategoryStatus, version int64, createdAt, updatedAt time.Time, deletedAt sql.NullTime) *Category {
	var parent *uuid.UUID
	if parentID.Valid {
		parent = &parentID.UUID
	}
	return &Category{
		ID:        id,
		ParentID:  parent,
		Name:      name,
		Slug:      slug,
		Status:    CategoryStatus(status),
		Version:   version,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
		DeletedAt: nullTimePointer(deletedAt),
	}
}

func nullTimePointer(value sql.NullTime) *time.Time {
	if !value.Valid {
		return nil
	}
	return &value.Time
}
