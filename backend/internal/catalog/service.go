package catalog

import (
	"context"
	"fmt"
	"mime/multipart"
	"time"

	"github.com/google/uuid"
	"github.com/zona3-labs/mancing-id/internal/brand"
	dberrors "github.com/zona3-labs/mancing-id/internal/errors"
	"github.com/zona3-labs/mancing-id/internal/product"
	"github.com/zona3-labs/mancing-id/internal/transaction"
	"github.com/zona3-labs/mancing-id/internal/upload"
	"github.com/zona3-labs/mancing-id/internal/util"
)

type BrandRepository interface {
	GetBrandByID(context.Context, uuid.UUID) (*brand.Brand, error)
	AssociateLogo(context.Context, transaction.DBTX, uuid.UUID, string) (*string, error)
}

type BrandLogoService struct {
	brands       BrandRepository
	media        upload.MediaAdapter
	transactions transaction.Manager
}

type ProductReader interface {
	GetAllProducts(context.Context) ([]*product.Product, error)
	GetProductBySlug(context.Context, string) (*product.ProductDetail, error)
	GetProductDetailByID(context.Context, uuid.UUID) (*product.ProductDetail, error)
}

type ProductService struct {
	brands       brand.ProductBrandRepository
	products     ProductReader
	drafts       product.DraftProductRepository
	transactions transaction.Manager
}

type BrandDeletionService struct {
	brands       brand.CatalogBrandRepository
	products     product.ProductReferenceRepository
	transactions transaction.Manager
}

func NewBrandDeletionService(brands brand.CatalogBrandRepository, products product.ProductReferenceRepository, transactions transaction.Manager) *BrandDeletionService {
	return &BrandDeletionService{brands: brands, products: products, transactions: transactions}
}

func (s *BrandDeletionService) DeleteBrand(ctx context.Context, id uuid.UUID, expectedVersion int64) error {
	return s.transactions.WithinTransaction(ctx, func(txContext context.Context, tx transaction.DBTX) error {
		current, err := s.brands.GetBrandByIDForUpdate(txContext, tx, id)
		if err != nil {
			return err
		}
		if current.Version != expectedVersion {
			return brand.ErrBrandVersionConflict
		}
		if current.Status != brand.BrandStatusDraft {
			return brand.ErrBrandNotEditable
		}
		count, err := s.products.CountProductsByBrandIDInTransaction(txContext, tx, id)
		if err != nil {
			return err
		}
		if count > 0 {
			return brand.ErrBrandHasProducts
		}
		return s.brands.DeleteBrandInTransaction(txContext, tx, id, expectedVersion)
	})
}

func NewProductService(brands brand.ProductBrandRepository, products ProductReader, drafts product.DraftProductRepository, transactions transaction.Manager) *ProductService {
	return &ProductService{brands: brands, products: products, drafts: drafts, transactions: transactions}
}

func (s *ProductService) CreateProduct(ctx context.Context, draft *product.Product) error {
	if draft.Status != product.StatusDraft {
		return product.ErrProductDraftRequired
	}
	if draft.BrandID == nil {
		return product.ErrProductBrandRequired
	}
	draft.ID = uuid.New()
	draft.Status = product.StatusDraft
	draft.Version = 1
	baseSlug := draft.Name
	if draft.Slug != "" {
		baseSlug = draft.Slug
	}
	baseSlug = util.GenerateSlug(baseSlug)
	if baseSlug == "" {
		return product.ErrInvalidProduct
	}

	for attempt := 1; ; attempt++ {
		draft.Slug = productSlug(baseSlug, attempt)
		err := s.transactions.WithinTransaction(ctx, func(txContext context.Context, tx transaction.DBTX) error {
			return s.validateActiveBrandAndCreate(txContext, tx, draft)
		})
		if err == nil {
			return nil
		}
		if !dberrors.IsUniqueViolation(err) {
			return err
		}
	}
}

func (s *ProductService) GetAllProducts(ctx context.Context) ([]*product.Product, error) {
	return s.products.GetAllProducts(ctx)
}

func (s *ProductService) GetProductBySlug(ctx context.Context, slug string) (*product.ProductDetail, error) {
	detail, err := s.products.GetProductBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	return s.withBrand(ctx, detail)
}

func (s *ProductService) GetProductDetailByID(ctx context.Context, id uuid.UUID) (*product.ProductDetail, error) {
	detail, err := s.products.GetProductDetailByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return s.withBrand(ctx, detail)
}

func (s *ProductService) UpdateProduct(ctx context.Context, draft *product.Product) error {
	for attempt := 1; ; attempt++ {
		err := s.transactions.WithinTransaction(ctx, func(txContext context.Context, tx transaction.DBTX) error {
			current, err := s.drafts.GetProductForUpdate(txContext, tx, draft.ID)
			if err != nil {
				return err
			}
			if current.Version != draft.Version {
				return product.ErrProductVersionConflict
			}
			if current.Status != product.StatusDraft {
				return product.ErrProductNotEditable
			}
			if draft.BrandID == nil {
				return product.ErrProductBrandRequired
			}
			if draft.Status != product.StatusDraft {
				return product.ErrProductDraftRequired
			}
			baseSlug := draft.Name
			if draft.Slug != "" {
				baseSlug = draft.Slug
			}
			baseSlug = util.GenerateSlug(baseSlug)
			if baseSlug == "" {
				return product.ErrInvalidProduct
			}
			draft.Slug = productSlug(baseSlug, attempt)
			if err := s.validateActiveBrand(txContext, tx, *draft.BrandID); err != nil {
				return err
			}
			return s.drafts.UpdateProductInTransaction(txContext, tx, draft)
		})
		if err == nil {
			return nil
		}
		if !dberrors.IsUniqueViolation(err) {
			return err
		}
	}
}

func (s *ProductService) DeleteProduct(ctx context.Context, id uuid.UUID, expectedVersion int64) error {
	return s.transactions.WithinTransaction(ctx, func(txContext context.Context, tx transaction.DBTX) error {
		current, err := s.drafts.GetProductForUpdate(txContext, tx, id)
		if err != nil {
			return err
		}
		if current.Version != expectedVersion {
			return product.ErrProductVersionConflict
		}
		if current.Status != product.StatusDraft {
			return product.ErrProductNotEditable
		}
		return s.drafts.DeleteDraftProduct(txContext, tx, id, expectedVersion)
	})
}

func (s *ProductService) validateActiveBrandAndCreate(ctx context.Context, tx transaction.DBTX, draft *product.Product) error {
	if err := s.validateActiveBrand(ctx, tx, *draft.BrandID); err != nil {
		return err
	}
	return s.drafts.CreateProductInTransaction(ctx, tx, draft)
}

func (s *ProductService) validateActiveBrand(ctx context.Context, tx transaction.DBTX, id uuid.UUID) error {
	assigned, err := s.brands.GetBrandByIDForUpdate(ctx, tx, id)
	if err != nil {
		return err
	}
	if assigned.Status != brand.BrandStatusActive {
		return product.ErrProductBrandNotActive
	}
	return nil
}

func (s *ProductService) withBrand(ctx context.Context, detail *product.ProductDetail) (*product.ProductDetail, error) {
	if detail.BrandID == nil {
		return detail, nil
	}
	assigned, err := s.brands.GetBrandByID(ctx, *detail.BrandID)
	if err != nil {
		return nil, err
	}
	detail.Brand = &product.BrandSummary{ID: assigned.ID, Name: assigned.Name, Slug: assigned.Slug, LogoPath: assigned.LogoPath}
	return detail, nil
}

func productSlug(base string, attempt int) string {
	if attempt == 1 {
		return base
	}
	return fmt.Sprintf("%s-%d", base, attempt)
}

func NewBrandLogoService(brands BrandRepository, media upload.MediaAdapter, transactions transaction.Manager) *BrandLogoService {
	return &BrandLogoService{brands: brands, media: media, transactions: transactions}
}

func (s *BrandLogoService) UploadBrandLogo(ctx context.Context, id uuid.UUID, file multipart.File, header *multipart.FileHeader) (string, error) {
	if _, err := s.brands.GetBrandByID(ctx, id); err != nil {
		return "", err
	}

	temporary, err := s.media.UploadTemporary(ctx, file, header, id)
	if err != nil {
		return "", err
	}

	var previousLogo *string
	if err := s.transactions.WithinTransaction(ctx, func(txContext context.Context, tx transaction.DBTX) error {
		previousLogo, err = s.brands.AssociateLogo(txContext, tx, id, temporary.URL)
		return err
	}); err != nil {
		return "", err
	}

	if err := s.media.FinalizeTemporary(ctx, temporary); err != nil {
		return "", err
	}
	current, err := s.brands.GetBrandByID(ctx, id)
	if err != nil {
		return "", err
	}
	if current.LogoPath == nil || *current.LogoPath != temporary.URL {
		if err := s.media.ScheduleDelete(ctx, temporary.URL); err != nil {
			return "", err
		}
		if current.LogoPath == nil {
			return "", nil
		}
		return *current.LogoPath, nil
	}
	if previousLogo != nil && *previousLogo != "" {
		if err := s.media.ScheduleDelete(ctx, *previousLogo); err != nil {
			return "", err
		}
	}
	return temporary.URL, nil
}

func (s *BrandLogoService) CleanStaleTemporaryImages(ctx context.Context, before time.Time) (int, error) {
	return s.media.CleanStaleTemporary(ctx, before)
}
