package product

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	dberrors "github.com/zone3-labs/mancing-id/internal/errors"
	"github.com/zone3-labs/mancing-id/internal/util"
)

type ProductUsecase interface {
	CreateProduct(ctx context.Context, product *Product) error
	GetAllProducts(ctx context.Context) ([]*Product, error)
	GetProductBySlug(ctx context.Context, slug string) (*Product, error)
	UpdateProduct(ctx context.Context, product *Product) error
	DeleteProduct(ctx context.Context, id uuid.UUID) error
}

type productUsecase struct {
	repo ProductRepository
}

func NewProductUsecase(repo ProductRepository) ProductUsecase {
	return &productUsecase{repo: repo}
}

func (p productUsecase) CreateProduct(ctx context.Context, product *Product) error {
	product.ID = uuid.New()
	base := product.Name
	if product.Slug != "" {
		base = product.Slug
	}

	baseSlug := util.GenerateSlug(base)
	product.Slug = baseSlug
	for attempt := 2; ; attempt++ {
		err := p.repo.CreateProduct(ctx, product)
		if err == nil {
			return nil
		}
		if !dberrors.IsUniqueViolation(err) {
			return err
		}
		product.Slug = fmt.Sprintf("%s-%d", baseSlug, attempt)
	}
}

func (p productUsecase) GetAllProducts(ctx context.Context) ([]*Product, error) {
	return p.repo.GetAllProducts(ctx)
}

func (p productUsecase) GetProductBySlug(ctx context.Context, slug string) (*Product, error) {
	product, err := p.repo.GetProductBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	return product, nil
}

func (p productUsecase) UpdateProduct(ctx context.Context, product *Product) error {
	product.ID = uuid.New()
	base := product.Name
	if product.Slug != "" {
		base = product.Slug
	}

	baseSlug := util.GenerateSlug(base)
	product.Slug = baseSlug
	for attempt := 2; ; attempt++ {
		err := p.repo.UpdateProduct(ctx, product)
		if err == nil {
			return nil
		}
		if !dberrors.IsUniqueViolation(err) {
			return err
		}
		product.Slug = fmt.Sprintf("%s-%d", baseSlug, attempt)
	}
}

func (p productUsecase) DeleteProduct(ctx context.Context, id uuid.UUID) error {
	return p.repo.DeleteProduct(ctx, id)
}
