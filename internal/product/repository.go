package product

import (
	"context"

	"github.com/google/uuid"
)

type ProductRepository interface {
	// Product
	CreateProduct(ctx context.Context, Product *Product) error
	GetAllProducts(ctx context.Context) ([]*Product, error)
	GetProductBySlug(ctx context.Context, slug string) (*Product, error)
	UpdateProduct(ctx context.Context, Product *Product) error
	DeleteProduct(ctx context.Context, id uuid.UUID) error

	// Options
	CreateProductOption(ctx context.Context, option *ProductOption) (*ProductOption, error)
	GetProductOptions(ctx context.Context, productID uuid.UUID) ([]*ProductOption, error)
	GetProductOptionByID(ctx context.Context, id uuid.UUID) (*ProductOption, error)
	UpdateProductOption(ctx context.Context, option *ProductOption) (*ProductOption, error)
	DeleteProductOption(ctx context.Context, id uuid.UUID) error

	// Option Values
	CreateProductOptionValue(ctx context.Context, value *ProductOptionValue) (*ProductOptionValue, error)
	GetProductOptionValues(ctx context.Context, optionID uuid.UUID) ([]*ProductOptionValue, error)
	DeleteProductOptionValue(ctx context.Context, id uuid.UUID) error
}
