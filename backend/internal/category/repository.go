package category

import (
	"context"

	"github.com/google/uuid"
)

type CategoryRepository interface {
	CreateCategory(ctx context.Context, category *Category) error
	GetCategoryByID(ctx context.Context, id uuid.UUID) (*Category, error)
	GetAllCategories(ctx context.Context) ([]*Category, error)
	GetCategoryTree(ctx context.Context, slug string) ([]*Category, error)
	UpdateCategory(ctx context.Context, category *Category, expectedVersion int64) error
	CheckCategoriesByParentID(ctx context.Context, id uuid.UUID) (int64, error)
	DeleteCategory(ctx context.Context, id uuid.UUID, expectedVersion int64) error
}
