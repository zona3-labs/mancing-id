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
	GetCategoryByID(ctx context.Context, id uuid.UUID) (*Category, error)
	GetAllCategories(ctx context.Context) ([]*Category, error)
	GetActiveCategories(ctx context.Context) ([]*Category, error)
	GetCategoryBySlug(ctx context.Context, slug string) (*Category, error)
	GetActiveCategoryBySlug(ctx context.Context, slug string) (*Category, error)
	UpdateCategory(ctx context.Context, category *Category, expectedVersion int64) error
	ActivateCategory(ctx context.Context, id uuid.UUID, expectedVersion int64) error
	RetireCategory(ctx context.Context, id uuid.UUID, expectedVersion int64) error
	DeleteCategory(ctx context.Context, id uuid.UUID, expectedVersion int64) error
}

type categoryUsecase struct {
	repo CategoryRepository
}

func NewCategoryUsecase(repo CategoryRepository) CategoryUsecase {
	return &categoryUsecase{repo: repo}
}

func (c categoryUsecase) CreateCategory(ctx context.Context, category *Category) error {
	category.ID = uuid.New()
	category.Status = CategoryStatusDraft
	category.Version = 1

	base := category.Name
	if category.Slug != "" {
		base = category.Slug
	}
	baseSlug := util.GenerateSlug(base)
	if baseSlug == "" {
		return ErrInvalidCategory
	}
	category.Slug = baseSlug

	for attempt := 1; ; attempt++ {
		err := c.repo.CreateCategory(ctx, category)
		if err == nil {
			return nil
		}
		if !dberrors.IsUniqueViolation(err) {
			return err
		}
		category.Slug = fmt.Sprintf("%s-%d", baseSlug, attempt+1)
	}
}

func (c categoryUsecase) GetCategoryByID(ctx context.Context, id uuid.UUID) (*Category, error) {
	return c.repo.GetCategoryByID(ctx, id)
}

func (c categoryUsecase) GetAllCategories(ctx context.Context) ([]*Category, error) {
	return c.repo.GetAllCategories(ctx)
}

func (c categoryUsecase) GetActiveCategories(ctx context.Context) ([]*Category, error) {
	rows, err := c.repo.GetActiveCategories(ctx)
	if err != nil {
		return nil, err
	}
	return buildForest(rows), nil
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

func (c categoryUsecase) GetActiveCategoryBySlug(ctx context.Context, slug string) (*Category, error) {
	rows, err := c.repo.GetActiveCategoryTree(ctx, slug)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, ErrCategoryNotFound
	}
	return buildTree(rows), nil
}

func (c categoryUsecase) UpdateCategory(ctx context.Context, category *Category, expectedVersion int64) error {
	if err := c.validateParent(ctx, category.ID, category.ParentID); err != nil {
		return err
	}

	base := category.Name
	if category.Slug != "" {
		base = category.Slug
	}
	baseSlug := util.GenerateSlug(base)
	if baseSlug == "" {
		return ErrInvalidCategory
	}
	category.Slug = baseSlug

	for attempt := 1; ; attempt++ {
		err := c.repo.UpdateCategory(ctx, category, expectedVersion)
		if err == nil {
			return nil
		}
		if !dberrors.IsUniqueViolation(err) {
			return err
		}
		category.Slug = fmt.Sprintf("%s-%d", baseSlug, attempt+1)
	}
}

func (c categoryUsecase) ActivateCategory(ctx context.Context, id uuid.UUID, expectedVersion int64) error {
	category, err := c.repo.GetCategoryByID(ctx, id)
	if err != nil {
		return err
	}
	if category.Version != expectedVersion {
		return ErrCategoryVersionConflict
	}
	if category.Status != CategoryStatusDraft {
		return ErrCategoryNotEditable
	}
	ancestors, err := c.categoryAncestors(ctx, category.ID, category.ParentID)
	if err != nil {
		if err == ErrCategoryNotFound {
			return ErrParentCategoryNotFound
		}
		return err
	}
	for _, parent := range ancestors {
		if parent.Status != CategoryStatusActive {
			return ErrCategoryAncestorNotActive
		}
	}
	return c.repo.ActivateCategory(ctx, id, expectedVersion)
}

func (c categoryUsecase) RetireCategory(ctx context.Context, id uuid.UUID, expectedVersion int64) error {
	category, err := c.repo.GetCategoryByID(ctx, id)
	if err != nil {
		return err
	}
	if category.Version != expectedVersion {
		return ErrCategoryVersionConflict
	}
	if category.Status != CategoryStatusActive {
		return ErrCategoryNotEditable
	}
	return c.repo.RetireCategory(ctx, id, expectedVersion)
}

func (c categoryUsecase) DeleteCategory(ctx context.Context, id uuid.UUID, expectedVersion int64) error {
	category, err := c.repo.GetCategoryByID(ctx, id)
	if err != nil {
		return err
	}
	if category.Version != expectedVersion {
		return ErrCategoryVersionConflict
	}
	if category.Status != CategoryStatusDraft {
		return ErrCategoryNotEditable
	}

	numChildren, err := c.repo.CheckCategoriesByParentID(ctx, id)
	if err != nil {
		return err
	}
	if numChildren > 0 {
		return ErrCategoryHasChildren
	}
	return c.repo.DeleteCategory(ctx, id, expectedVersion)
}

func (c categoryUsecase) validateParent(ctx context.Context, categoryID uuid.UUID, parentID *uuid.UUID) error {
	if parentID == nil {
		return nil
	}
	_, err := c.categoryAncestors(ctx, categoryID, parentID)
	if err == ErrCategoryNotFound {
		return ErrParentCategoryNotFound
	}
	return err
}

func (c categoryUsecase) categoryAncestors(ctx context.Context, categoryID uuid.UUID, parentID *uuid.UUID) ([]*Category, error) {
	if parentID == nil {
		return nil, nil
	}

	ancestors := make([]*Category, 0)
	visited := map[uuid.UUID]struct{}{categoryID: {}}
	currentID := *parentID
	for {
		if _, seen := visited[currentID]; seen {
			return nil, ErrCategoryCycle
		}
		visited[currentID] = struct{}{}

		parent, err := c.repo.GetCategoryByID(ctx, currentID)
		if err != nil {
			return nil, err
		}
		ancestors = append(ancestors, parent)
		if parent.ParentID == nil {
			return ancestors, nil
		}
		currentID = *parent.ParentID
	}
}

// buildTree converts a flat list from the recursive CTE into a nested tree.
func buildTree(rows []*Category) *Category {
	roots := buildForest(rows)
	if len(roots) == 0 {
		return nil
	}
	return roots[0]
}

func buildForest(rows []*Category) []*Category {
	if len(rows) == 0 {
		return []*Category{}
	}

	index := make(map[uuid.UUID]*Category, len(rows))
	for _, row := range rows {
		index[row.ID] = row
	}
	roots := make([]*Category, 0)
	for _, row := range rows {
		if row.ParentID == nil {
			roots = append(roots, row)
			continue
		}
		if parent, ok := index[*row.ParentID]; ok {
			parent.Children = append(parent.Children, row)
		} else {
			roots = append(roots, row)
		}
	}
	return roots
}
