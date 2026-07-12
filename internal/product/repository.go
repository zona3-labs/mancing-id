package product

import (
	"context"

	"github.com/google/uuid"
)

type ProductRepository interface {
	CreateProduct(ctx context.Context, Product *Product) error
	GetAllProducts(ctx context.Context) ([]*Product, error)
	GetProductBySlug(ctx context.Context, slug string) (*Product, error)
	UpdateProduct(ctx context.Context, Product *Product) error
	DeleteProduct(ctx context.Context, id uuid.UUID) error
}
