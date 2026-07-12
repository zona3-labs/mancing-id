package product

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
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
