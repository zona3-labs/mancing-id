package product

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	dberrors "github.com/zone3-labs/mancing-id/internal/errors"
	productDb "github.com/zone3-labs/mancing-id/internal/product/db"
)

type productPostgresRepository struct {
	queries *productDb.Queries
}

func NewProductRepository(db *sql.DB) ProductRepository {
	return &productPostgresRepository{queries: productDb.New(db)}
}

func (p productPostgresRepository) CreateProduct(ctx context.Context, product *Product) error {
	return p.queries.CreateProduct(ctx, productDb.CreateProductParams{
		ID:               product.ID,
		Name:             product.Name,
		Slug:             product.Slug,
		Description:      product.Description,
		ShortDescription: product.ShortDescription,
		Status:           productDb.ProductStatus(product.Status),
		BrandID:          product.BrandID,
		IsFeatured:       product.IsFeature,
	})
}

func (p productPostgresRepository) GetAllProducts(ctx context.Context) ([]*Product, error) {
	rows, err := p.queries.GetAllProducts(ctx)
	if err != nil {
		return nil, err
	}
	products := make([]*Product, 0, len(rows))
	for _, row := range rows {
		products = append(products, &Product{
			ID:               row.ID,
			Name:             row.Name,
			Slug:             row.Slug,
			Description:      row.Description,
			ShortDescription: row.ShortDescription,
			BrandID:          row.BrandID,
			Status:           ProductStatus(row.Status),
			IsFeature:        row.IsFeatured,
			CreatedAt:        row.CreatedAt,
			UpdatedAt:        row.UpdatedAt,
			DeletedAt:        row.DeletedAt,
		})
	}
	return products, nil
}

func (p productPostgresRepository) GetProductBySlug(ctx context.Context, slug string) (*Product, error) {
	row, err := p.queries.GetProductBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	return &Product{
		ID:               row.ID,
		Name:             row.Name,
		Slug:             row.Slug,
		Description:      row.Description,
		ShortDescription: row.ShortDescription,
		BrandID:          row.BrandID,
		Status:           ProductStatus(row.Status),
		IsFeature:        row.IsFeatured,
		CreatedAt:        row.CreatedAt,
		UpdatedAt:        row.UpdatedAt,
		DeletedAt:        row.DeletedAt,
	}, nil
}

func (p productPostgresRepository) UpdateProduct(ctx context.Context, product *Product) error {
	return p.queries.UpdateProduct(ctx, productDb.UpdateProductParams{
		ID:               product.ID,
		Name:             product.Name,
		Slug:             product.Slug,
		Description:      product.Description,
		ShortDescription: product.ShortDescription,
		Status:           productDb.ProductStatus(product.Status),
		BrandID:          product.BrandID,
		IsFeatured:       product.IsFeature,
	})
}

func (p productPostgresRepository) DeleteProduct(ctx context.Context, id uuid.UUID) error {
	result, err := p.queries.DeleteProduct(ctx, id)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrProductNotFound
	}
	return nil
}

// -------------------------------------------------------
// Options
// -------------------------------------------------------

func (p productPostgresRepository) CreateProductOption(ctx context.Context, option *ProductOption) (*ProductOption, error) {
	row, err := p.queries.CreateProductOption(ctx, productDb.CreateProductOptionParams{
		ID:        option.ID,
		ProductID: option.ProductID,
		Name:      option.Name,
		Position:  option.Position,
	})
	if dberrors.IsForeignKeyViolation(err) {
		return nil, ErrProductNotFound
	}
	if err != nil {
		return nil, err
	}
	return mapOption(row), nil
}

func (p productPostgresRepository) GetProductOptions(ctx context.Context, productID uuid.UUID) ([]*ProductOption, error) {
	rows, err := p.queries.GetProductOptionsByProductID(ctx, productID)
	if err != nil {
		return nil, err
	}
	options := make([]*ProductOption, 0, len(rows))
	for _, row := range rows {
		options = append(options, mapOption(row))
	}
	return options, nil
}

func (p productPostgresRepository) GetProductOptionByID(ctx context.Context, id uuid.UUID) (*ProductOption, error) {
	row, err := p.queries.GetProductOptionByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return mapOption(row), nil
}

func (p productPostgresRepository) UpdateProductOption(ctx context.Context, option *ProductOption) (*ProductOption, error) {
	row, err := p.queries.UpdateProductOption(ctx, productDb.UpdateProductOptionParams{
		ID:       option.ID,
		Name:     option.Name,
		Position: option.Position,
	})
	if err != nil {
		return nil, err
	}
	return mapOption(row), nil
}

func (p productPostgresRepository) DeleteProductOption(ctx context.Context, id uuid.UUID) error {
	result, err := p.queries.DeleteProductOption(ctx, id)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrOptionNotFound
	}
	return nil
}

// -------------------------------------------------------
// Option Values
// -------------------------------------------------------

func (p productPostgresRepository) CreateProductOptionValue(ctx context.Context, value *ProductOptionValue) (*ProductOptionValue, error) {
	row, err := p.queries.CreateProductOptionValue(ctx, productDb.CreateProductOptionValueParams{
		ID:              value.ID,
		ProductOptionID: value.ProductOptionID,
		Value:           value.Value,
		Position:        value.Position,
	})
	if err != nil {
		return nil, err
	}
	return mapOptionValue(row), nil
}

func (p productPostgresRepository) GetProductOptionValues(ctx context.Context, optionID uuid.UUID) ([]*ProductOptionValue, error) {
	rows, err := p.queries.GetProductOptionValuesByOptionID(ctx, optionID)
	if err != nil {
		return nil, err
	}
	values := make([]*ProductOptionValue, 0, len(rows))
	for _, row := range rows {
		values = append(values, mapOptionValue(row))
	}
	return values, nil
}

func (p productPostgresRepository) DeleteProductOptionValue(ctx context.Context, id uuid.UUID) error {
	result, err := p.queries.DeleteProductOptionValue(ctx, id)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return ErrOptionValueNotFound
	}
	return nil
}

// -------------------------------------------------------
// Mapping helpers
// -------------------------------------------------------

func mapOption(row productDb.ProductOption) *ProductOption {
	return &ProductOption{
		ID:        row.ID,
		ProductID: row.ProductID,
		Name:      row.Name,
		Position:  row.Position,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}
}

func mapOptionValue(row productDb.ProductOptionValue) *ProductOptionValue {
	return &ProductOptionValue{
		ID:              row.ID,
		ProductOptionID: row.ProductOptionID,
		Value:           row.Value,
		Position:        row.Position,
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
	}
}
