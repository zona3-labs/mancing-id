package catalog_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/zona3-labs/mancing-id/internal/brand"
	"github.com/zona3-labs/mancing-id/internal/catalog"
	"github.com/zona3-labs/mancing-id/internal/product"
	"github.com/zona3-labs/mancing-id/internal/transaction"
)

func TestProductServiceCreatesDraftOnlyWithActiveBrandAndEmbedsBrandSummary(t *testing.T) {
	brandID := uuid.New()
	products := &fakeProductReader{details: map[uuid.UUID]*product.ProductDetail{}}
	drafts := &fakeDraftProductRepository{}
	service := catalog.NewProductService(
		&fakeProductBrandRepository{brands: map[uuid.UUID]*brand.Brand{
			brandID: {ID: brandID, Name: "Acme Fishing", Slug: "acme-fishing", Status: brand.BrandStatusActive},
		}},
		products,
		drafts,
		&fakeTransactionManager{},
	)

	draft := &product.Product{Name: "Spinning Reel", BrandID: &brandID, Status: product.StatusDraft}
	if err := service.CreateProduct(context.Background(), draft); err != nil {
		t.Fatalf("create product: %v", err)
	}
	if draft.ID == uuid.Nil || draft.Status != product.StatusDraft || draft.Version != 1 {
		t.Fatalf("created draft = %+v", draft)
	}

	products.details[draft.ID] = &product.ProductDetail{Product: *draft}
	detail, err := service.GetProductDetailByID(context.Background(), draft.ID)
	if err != nil {
		t.Fatalf("get product detail: %v", err)
	}
	if detail.Brand == nil || detail.Brand.ID != brandID || detail.Brand.Name != "Acme Fishing" {
		t.Fatalf("brand summary = %+v", detail.Brand)
	}
}

func TestProductServiceRejectsNonActiveBrandAndStaleOrMissingWrites(t *testing.T) {
	brandID := uuid.New()
	drafts := &fakeDraftProductRepository{}
	service := catalog.NewProductService(
		&fakeProductBrandRepository{brands: map[uuid.UUID]*brand.Brand{
			brandID: {ID: brandID, Status: brand.BrandStatusDraft},
		}},
		&fakeProductReader{details: map[uuid.UUID]*product.ProductDetail{}},
		drafts,
		&fakeTransactionManager{},
	)

	draft := &product.Product{Name: "Draft Brand Product", BrandID: &brandID, Status: product.StatusDraft}
	if err := service.CreateProduct(context.Background(), draft); !errors.Is(err, product.ErrProductBrandNotActive) {
		t.Fatalf("create with draft brand error = %v", err)
	}

	missingID := uuid.New()
	if err := service.UpdateProduct(context.Background(), &product.Product{
		ID: missingID, Name: "Missing", BrandID: &brandID, Status: product.StatusDraft, Version: 1,
	}); !errors.Is(err, product.ErrProductNotFound) {
		t.Fatalf("missing update error = %v", err)
	}
}

func TestProductServiceUpdatesRequestedDraftAndDeletesWithVersion(t *testing.T) {
	brandID := uuid.New()
	productID := uuid.New()
	current := &product.Product{ID: productID, Name: "Old Reel", Slug: "old-reel", BrandID: &brandID, Status: product.StatusDraft, Version: 1}
	drafts := &fakeDraftProductRepository{products: map[uuid.UUID]*product.Product{productID: current}}
	service := catalog.NewProductService(
		&fakeProductBrandRepository{brands: map[uuid.UUID]*brand.Brand{
			brandID: {ID: brandID, Status: brand.BrandStatusActive},
		}},
		&fakeProductReader{details: map[uuid.UUID]*product.ProductDetail{}},
		drafts,
		&fakeTransactionManager{},
	)

	updated := &product.Product{ID: productID, Name: "New Reel", Slug: "new-reel", BrandID: &brandID, Status: product.StatusDraft, Version: 1}
	if err := service.UpdateProduct(context.Background(), updated); err != nil {
		t.Fatalf("update product: %v", err)
	}
	if updated.ID != productID || updated.Version != 2 || updated.Slug != "new-reel" {
		t.Fatalf("updated product = %+v", updated)
	}
	if err := service.DeleteProduct(context.Background(), productID, 2); err != nil {
		t.Fatalf("delete product: %v", err)
	}
	if _, ok := drafts.products[productID]; ok {
		t.Fatal("deleted product remains")
	}
}

type fakeProductBrandRepository struct {
	brands map[uuid.UUID]*brand.Brand
}

func (f *fakeProductBrandRepository) GetBrandByID(_ context.Context, id uuid.UUID) (*brand.Brand, error) {
	value, ok := f.brands[id]
	if !ok {
		return nil, brand.ErrBrandNotFound
	}
	copy := *value
	return &copy, nil
}

func (f *fakeProductBrandRepository) GetBrandByIDForUpdate(ctx context.Context, _ transaction.DBTX, id uuid.UUID) (*brand.Brand, error) {
	return f.GetBrandByID(ctx, id)
}

type fakeProductReader struct {
	details map[uuid.UUID]*product.ProductDetail
}

func (f *fakeProductReader) GetAllProducts(context.Context) ([]*product.Product, error) {
	return nil, nil
}

func (f *fakeProductReader) GetProductBySlug(context.Context, string) (*product.ProductDetail, error) {
	return nil, product.ErrProductNotFound
}

func (f *fakeProductReader) GetProductDetailByID(_ context.Context, id uuid.UUID) (*product.ProductDetail, error) {
	detail, ok := f.details[id]
	if !ok {
		return nil, product.ErrProductNotFound
	}
	copy := *detail
	return &copy, nil
}

type fakeDraftProductRepository struct {
	products map[uuid.UUID]*product.Product
}

func (f *fakeDraftProductRepository) CreateProductInTransaction(_ context.Context, _ transaction.DBTX, draft *product.Product) error {
	if f.products == nil {
		f.products = make(map[uuid.UUID]*product.Product)
	}
	copy := *draft
	f.products[draft.ID] = &copy
	return nil
}

func (f *fakeDraftProductRepository) GetProductForUpdate(_ context.Context, _ transaction.DBTX, id uuid.UUID) (*product.Product, error) {
	draft, ok := f.products[id]
	if !ok {
		return nil, product.ErrProductNotFound
	}
	copy := *draft
	return &copy, nil
}

func (f *fakeDraftProductRepository) UpdateProductInTransaction(_ context.Context, _ transaction.DBTX, draft *product.Product) error {
	draft.Version++
	copy := *draft
	f.products[draft.ID] = &copy
	return nil
}

func (f *fakeDraftProductRepository) DeleteDraftProduct(_ context.Context, _ transaction.DBTX, id uuid.UUID, _ int64) error {
	delete(f.products, id)
	return nil
}
