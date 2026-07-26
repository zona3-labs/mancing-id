package product

import (
	"context"

	"github.com/google/uuid"
)

type ProductRepository interface {
	// Product
	CreateProduct(ctx context.Context, Product *Product) error
	GetAllProducts(ctx context.Context) ([]*Product, error)
	GetProductBySlug(ctx context.Context, slug string) (*ProductDetail, error)
	GetProductDetailByID(ctx context.Context, id uuid.UUID) (*ProductDetail, error)
	UpdateProduct(ctx context.Context, Product *Product) error
	DeleteProduct(ctx context.Context, id uuid.UUID) error

	// Options
	CreateProductOptionWithValues(ctx context.Context, option *ProductOption, values []*ProductOptionValue) (*ProductOptionWithValues, error)
	GetProductOptions(ctx context.Context, productID uuid.UUID) ([]*ProductOption, error)
	GetProductOptionByID(ctx context.Context, id uuid.UUID) (*ProductOption, error)
	UpdateProductOption(ctx context.Context, option *ProductOption) (*ProductOption, error)
	DeleteProductOption(ctx context.Context, id uuid.UUID) error

	// Option Values
	CreateProductOptionValue(ctx context.Context, value *ProductOptionValue) (*ProductOptionValue, error)
	GetProductOptionValues(ctx context.Context, optionID uuid.UUID) ([]*ProductOptionValue, error)
	DeleteProductOptionValue(ctx context.Context, id uuid.UUID) error

	// Images
	CreateProductImage(ctx context.Context, image *ProductImage) (*ProductImage, error)
	GetProductImages(ctx context.Context, productID uuid.UUID) ([]*ProductImage, error)
	GetProductImageByID(ctx context.Context, id uuid.UUID) (*ProductImage, error)
	SetPrimaryImage(ctx context.Context, productID uuid.UUID, imageID uuid.UUID) error
	DeleteProductImage(ctx context.Context, id uuid.UUID) error

	// Variants
	CreateProductVariant(ctx context.Context, variant *ProductVariant, optionValueIDs []uuid.UUID) (*ProductVariantDetail, error)
	GetProductVariantByID(ctx context.Context, id uuid.UUID) (*ProductVariantDetail, error)
	GetProductVariants(ctx context.Context, productID uuid.UUID) ([]*ProductVariantDetail, error)
	UpdateProductVariant(ctx context.Context, variant *ProductVariant) (*ProductVariantDetail, error)
	DeleteProductVariant(ctx context.Context, id uuid.UUID) error
}
