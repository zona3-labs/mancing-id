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

	// Options
	CreateProductOption(ctx context.Context, option *ProductOption) (*ProductOption, error)
	GetProductOptions(ctx context.Context, productID uuid.UUID) ([]*ProductOption, error)
	UpdateProductOption(ctx context.Context, option *ProductOption) (*ProductOption, error)
	DeleteProductOption(ctx context.Context, id uuid.UUID) error

	// Option Values
	CreateProductOptionValue(ctx context.Context, value *ProductOptionValue) (*ProductOptionValue, error)
	GetProductOptionValues(ctx context.Context, optionID uuid.UUID) ([]*ProductOptionValue, error)
	DeleteProductOptionValue(ctx context.Context, id uuid.UUID) error
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

// -------------------------------------------------------
// Options
// -------------------------------------------------------

func (p productUsecase) CreateProductOption(ctx context.Context, option *ProductOption) (*ProductOption, error) {
	option.ID = uuid.New()
	return p.repo.CreateProductOption(ctx, option)
}

func (p productUsecase) GetProductOptions(ctx context.Context, productID uuid.UUID) ([]*ProductOption, error) {
	return p.repo.GetProductOptions(ctx, productID)
}

func (p productUsecase) UpdateProductOption(ctx context.Context, option *ProductOption) (*ProductOption, error) {
	return p.repo.UpdateProductOption(ctx, option)
}

func (p productUsecase) DeleteProductOption(ctx context.Context, id uuid.UUID) error {
	return p.repo.DeleteProductOption(ctx, id)
}

// -------------------------------------------------------
// Option Values
// -------------------------------------------------------

func (p productUsecase) CreateProductOptionValue(ctx context.Context, value *ProductOptionValue) (*ProductOptionValue, error) {
	value.ID = uuid.New()
	return p.repo.CreateProductOptionValue(ctx, value)
}

func (p productUsecase) GetProductOptionValues(ctx context.Context, optionID uuid.UUID) ([]*ProductOptionValue, error) {
	return p.repo.GetProductOptionValues(ctx, optionID)
}

func (p productUsecase) DeleteProductOptionValue(ctx context.Context, id uuid.UUID) error {
	return p.repo.DeleteProductOptionValue(ctx, id)
}
