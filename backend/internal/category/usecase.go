package category

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	dberrors "github.com/zona3-labs/mancing-id/internal/errors"
	"github.com/zona3-labs/mancing-id/internal/util"
)

type CategoryUsecase interface {
	CreateCategory(ctx context.Context, category *Category) error
	GetAllCategories(ctx context.Context) ([]*Category, error)
	GetCategoryBySlug(ctx context.Context, slug string) (*Category, error)
	UpdateCategory(ctx context.Context, category *Category) error
	DeleteCategory(ctx context.Context, id uuid.UUID) error
}

type categoryUsecase struct {
	repo CategoryRepository
}

func NewCategoryUsecase(repo CategoryRepository) CategoryUsecase {
	return &categoryUsecase{repo: repo}
}

func (c categoryUsecase) CreateCategory(ctx context.Context, category *Category) error {
	category.ID = uuid.New()

	base := category.Name
	if category.Slug != "" {
		base = category.Slug
	}
	baseSlug := util.GenerateSlug(base)
	category.Slug = baseSlug

	for attempt := 2; ; attempt++ {
		err := c.repo.CreateCategory(ctx, category)
		if err == nil {
			return nil
		}
		if !dberrors.IsUniqueViolation(err) {
			return err
		}
		category.Slug = fmt.Sprintf("%s-%d", baseSlug, attempt)
	}
}

func (c categoryUsecase) GetAllCategories(ctx context.Context) ([]*Category, error) {
	return c.repo.GetAllCategories(ctx)
}

func (c categoryUsecase) GetCategoryBySlug(ctx context.Context, slug string) (*Category, error) {
	rows, err := c.repo.GetCategoryTree(ctx, slug)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, ErrCategoryNotFound
	}
	return buildTree(rows), nil
}

// buildTree converts a flat list from the recursive CTE into a nested tree.
// The first row is always the root (the anchor of the CTE).
func buildTree(rows []*Category) *Category {
	index := make(map[uuid.UUID]*Category, len(rows))
	for _, row := range rows {
		index[row.ID] = row
	}
	var root *Category
	for _, row := range rows {
		if row.ParentID == nil {
			root = row
		} else if parent, ok := index[*row.ParentID]; ok {
			parent.Children = append(parent.Children, row)
		}
	}
	return root
}

func (c categoryUsecase) UpdateCategory(ctx context.Context, category *Category) error {
	base := category.Name
	if category.Slug != "" {
		base = category.Slug
	}
	baseSlug := util.GenerateSlug(base)
	category.Slug = baseSlug

	for attempt := 2; ; attempt++ {
		err := c.repo.UpdateCategory(ctx, category)
		if err == nil {
			return nil
		}
		if !dberrors.IsUniqueViolation(err) {
			return err
		}
		category.Slug = fmt.Sprintf("%s-%d", baseSlug, attempt)
	}
}

func (c categoryUsecase) DeleteCategory(ctx context.Context, id uuid.UUID) error {
	numChild, err := c.repo.CheckActiveCategoriesByParentId(ctx, id)
	if err != nil {
		return err
	}

	if numChild > 0 {
		return ErrCategoryHasChildren
	}
	return c.repo.DeleteCategory(ctx, id)
}
