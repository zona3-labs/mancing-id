package category

import (
	"context"

	"github.com/google/uuid"
)

type CategoryRepository interface {
	CreateCategory(ctx context.Context, category *Category) error
	GetAllCategories(ctx context.Context) ([]*Category, error)
	GetCategoryTree(ctx context.Context, slug string) ([]*Category, error)
	UpdateCategory(ctx context.Context, Category *Category) error
	CheckActiveCategoriesByParentId(ctx context.Context, id uuid.UUID) (int64, error)
	DeleteCategory(ctx context.Context, id uuid.UUID) error
}
