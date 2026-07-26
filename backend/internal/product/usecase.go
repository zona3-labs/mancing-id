package product

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	dberrors "github.com/zona3-labs/mancing-id/internal/errors"
	"github.com/zona3-labs/mancing-id/internal/util"
)

type ProductUsecase interface {
	CreateProduct(ctx context.Context, product *Product) error
	GetAllProducts(ctx context.Context) ([]*Product, error)
	GetProductBySlug(ctx context.Context, slug string) (*ProductDetail, error)
	GetProductDetailByID(ctx context.Context, id uuid.UUID) (*ProductDetail, error)
	UpdateProduct(ctx context.Context, product *Product) error
	DeleteProduct(ctx context.Context, id uuid.UUID) error

	// Options
	CreateProductOptionWithValues(ctx context.Context, option *ProductOption, values []*ProductOptionValue) (*ProductOptionWithValues, error)
	GetProductOptions(ctx context.Context, productID uuid.UUID) ([]*ProductOption, error)
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

func (p productUsecase) GetProductBySlug(ctx context.Context, slug string) (*ProductDetail, error) {
	return p.repo.GetProductBySlug(ctx, slug)
}

func (p productUsecase) GetProductDetailByID(ctx context.Context, id uuid.UUID) (*ProductDetail, error) {
	return p.repo.GetProductDetailByID(ctx, id)
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

func (p productUsecase) CreateProductOptionWithValues(ctx context.Context, option *ProductOption, values []*ProductOptionValue) (*ProductOptionWithValues, error) {
	option.ID = uuid.New()
	for _, v := range values {
		v.ID = uuid.New()
	}
	return p.repo.CreateProductOptionWithValues(ctx, option, values)
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

// -------------------------------------------------------
// Images
// -------------------------------------------------------

func (p productUsecase) CreateProductImage(ctx context.Context, image *ProductImage) (*ProductImage, error) {
	image.ID = uuid.New()
	return p.repo.CreateProductImage(ctx, image)
}

func (p productUsecase) GetProductImages(ctx context.Context, productID uuid.UUID) ([]*ProductImage, error) {
	return p.repo.GetProductImages(ctx, productID)
}

func (p productUsecase) GetProductImageByID(ctx context.Context, id uuid.UUID) (*ProductImage, error) {
	return p.repo.GetProductImageByID(ctx, id)
}

func (p productUsecase) SetPrimaryImage(ctx context.Context, productID uuid.UUID, imageID uuid.UUID) error {
	img, err := p.repo.GetProductImageByID(ctx, imageID)
	if err != nil {
		return err
	}
	if img.ProductID != productID {
		return ErrProductImageNotFound
	}
	return p.repo.SetPrimaryImage(ctx, productID, imageID)
}

func (p productUsecase) DeleteProductImage(ctx context.Context, id uuid.UUID) error {
	return p.repo.DeleteProductImage(ctx, id)
}

// -------------------------------------------------------
// Variants
// -------------------------------------------------------

func (p productUsecase) CreateProductVariant(ctx context.Context, variant *ProductVariant, optionValueIDs []uuid.UUID) (*ProductVariantDetail, error) {
	variant.ID = uuid.New()
	return p.repo.CreateProductVariant(ctx, variant, optionValueIDs)
}

func (p productUsecase) GetProductVariantByID(ctx context.Context, id uuid.UUID) (*ProductVariantDetail, error) {
	return p.repo.GetProductVariantByID(ctx, id)
}

func (p productUsecase) GetProductVariants(ctx context.Context, productID uuid.UUID) ([]*ProductVariantDetail, error) {
	return p.repo.GetProductVariants(ctx, productID)
}

func (p productUsecase) UpdateProductVariant(ctx context.Context, variant *ProductVariant) (*ProductVariantDetail, error) {
	return p.repo.UpdateProductVariant(ctx, variant)
}

func (p productUsecase) DeleteProductVariant(ctx context.Context, id uuid.UUID) error {
	return p.repo.DeleteProductVariant(ctx, id)
}
