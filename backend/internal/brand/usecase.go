package brand

import (
	"context"
	"fmt"
	"mime/multipart"

	"github.com/google/uuid"
	dberrors "github.com/zona3-labs/mancing-id/internal/errors"
	"github.com/zona3-labs/mancing-id/internal/upload"
	"github.com/zona3-labs/mancing-id/internal/util"
)

type BrandUsecase interface {
	CreateBrand(ctx context.Context, brand *Brand) error
	GetBrandByID(ctx context.Context, id uuid.UUID) (*Brand, error)
	GetAllBrands(ctx context.Context) ([]*Brand, error)
	GetPublicBrands(ctx context.Context) ([]*Brand, error)
	GetBrandBySlug(ctx context.Context, slug string) (*Brand, error)
	GetPublicBrandBySlug(ctx context.Context, slug string) (*Brand, error)
	UpdateBrand(ctx context.Context, brand *Brand, expectedVersion int64) error
	ActivateBrand(ctx context.Context, id uuid.UUID, expectedVersion int64) error
	DeactivateBrand(ctx context.Context, id uuid.UUID, expectedVersion int64) error
	ReactivateBrand(ctx context.Context, id uuid.UUID, expectedVersion int64) error
	UploadBrandLogo(ctx context.Context, id uuid.UUID, file multipart.File, header *multipart.FileHeader) (string, error)
	DeleteBrand(ctx context.Context, id uuid.UUID, expectedVersion int64) error
}

type brandUsecase struct {
	repo     BrandRepository
	uploader upload.FileUploader
}

func NewBrandUsecase(repo BrandRepository, uploader upload.FileUploader) BrandUsecase {
	return &brandUsecase{repo: repo, uploader: uploader}
}

func (b brandUsecase) CreateBrand(ctx context.Context, brand *Brand) error {
	brand.ID = uuid.New()
	brand.Status = BrandStatusDraft
	brand.Version = 1

	base := brand.Name
	if brand.Slug != "" {
		base = brand.Slug
	}
	baseSlug := util.GenerateSlug(base)
	if baseSlug == "" {
		return ErrInvalidBrand
	}
	brand.Slug = baseSlug

	for attempt := 1; ; attempt++ {
		if err := b.repo.CreateBrand(ctx, brand); err == nil {
			return nil
		} else if !dberrors.IsUniqueViolation(err) {
			return err
		}
		brand.Slug = fmt.Sprintf("%s-%d", baseSlug, attempt+1)
	}
}

func (b brandUsecase) GetBrandByID(ctx context.Context, id uuid.UUID) (*Brand, error) {
	return b.repo.GetBrandByID(ctx, id)
}

func (b brandUsecase) GetAllBrands(ctx context.Context) ([]*Brand, error) {
	return b.repo.GetAllBrands(ctx)
}

func (b brandUsecase) GetPublicBrands(ctx context.Context) ([]*Brand, error) {
	return b.repo.GetPublicBrands(ctx)
}

func (b brandUsecase) GetBrandBySlug(ctx context.Context, slug string) (*Brand, error) {
	return b.repo.GetBrandBySlug(ctx, slug)
}

func (b brandUsecase) GetPublicBrandBySlug(ctx context.Context, slug string) (*Brand, error) {
	return b.repo.GetPublicBrandBySlug(ctx, slug)
}

func (b brandUsecase) UpdateBrand(ctx context.Context, brand *Brand, expectedVersion int64) error {
	current, err := b.repo.GetBrandByID(ctx, brand.ID)
	if err != nil {
		return err
	}
	if current.Version != expectedVersion {
		return ErrBrandVersionConflict
	}
	if current.Status != BrandStatusDraft {
		return ErrBrandNotEditable
	}

	base := brand.Name
	if brand.Slug != "" {
		base = brand.Slug
	}
	baseSlug := util.GenerateSlug(base)
	if baseSlug == "" {
		return ErrInvalidBrand
	}
	brand.Slug = baseSlug
	for attempt := 1; ; attempt++ {
		if err := b.repo.UpdateBrand(ctx, brand, expectedVersion); err == nil {
			return nil
		} else if !dberrors.IsUniqueViolation(err) {
			return err
		}
		brand.Slug = fmt.Sprintf("%s-%d", baseSlug, attempt+1)
	}
}

func (b brandUsecase) ActivateBrand(ctx context.Context, id uuid.UUID, expectedVersion int64) error {
	return b.transition(ctx, id, expectedVersion, BrandStatusDraft, b.repo.ActivateBrand)
}

func (b brandUsecase) DeactivateBrand(ctx context.Context, id uuid.UUID, expectedVersion int64) error {
	return b.transition(ctx, id, expectedVersion, BrandStatusActive, b.repo.DeactivateBrand)
}

func (b brandUsecase) ReactivateBrand(ctx context.Context, id uuid.UUID, expectedVersion int64) error {
	return b.transition(ctx, id, expectedVersion, BrandStatusInactive, b.repo.ReactivateBrand)
}

func (b brandUsecase) transition(ctx context.Context, id uuid.UUID, expectedVersion int64, status BrandStatus, mutate func(context.Context, uuid.UUID, int64) error) error {
	brand, err := b.repo.GetBrandByID(ctx, id)
	if err != nil {
		return err
	}
	if brand.Version != expectedVersion {
		return ErrBrandVersionConflict
	}
	if brand.Status != status {
		return ErrBrandNotEditable
	}
	return mutate(ctx, id, expectedVersion)
}

func (b brandUsecase) UploadBrandLogo(ctx context.Context, id uuid.UUID, file multipart.File, header *multipart.FileHeader) (string, error) {
	if _, err := b.repo.GetBrandByID(ctx, id); err != nil {
		return "", err
	}
	url, err := b.uploader.Upload(ctx, file, header, id.String())
	if err != nil {
		return "", err
	}
	if err := b.repo.UpdateBrandLogo(ctx, id, url); err != nil {
		return "", err
	}
	return url, nil
}

func (b brandUsecase) DeleteBrand(ctx context.Context, id uuid.UUID, expectedVersion int64) error {
	brand, err := b.repo.GetBrandByID(ctx, id)
	if err != nil {
		return err
	}
	if brand.Version != expectedVersion {
		return ErrBrandVersionConflict
	}
	if brand.Status != BrandStatusDraft {
		return ErrBrandNotEditable
	}
	products, err := b.repo.CountProductsByBrandID(ctx, id)
	if err != nil {
		return err
	}
	if products > 0 {
		return ErrBrandHasProducts
	}
	return b.repo.DeleteBrand(ctx, id, expectedVersion)
}
