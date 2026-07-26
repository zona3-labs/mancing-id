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
	GetAllBrand(ctx context.Context) ([]*Brand, error)
	GetBrandBySlug(ctx context.Context, slug string) (*Brand, error)
	UpdateBrand(ctx context.Context, brand *Brand) error
	UploadBrandLogo(ctx context.Context, id uuid.UUID, file multipart.File, header *multipart.FileHeader) (string, error)
	DeleteBrand(ctx context.Context, id uuid.UUID) error
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
	base := brand.Name
	if brand.Slug != "" {
		base = brand.Slug
	}

	baseSlug := util.GenerateSlug(base)
	brand.Slug = baseSlug
	for attempt := 2; ; attempt++ {
		err := b.repo.CreateBrand(ctx, brand)
		if err == nil {
			return nil
		}
		if !dberrors.IsUniqueViolation(err) {
			return err
		}
		brand.Slug = fmt.Sprintf("%s-%d", baseSlug, attempt)
	}
}

func (b brandUsecase) GetAllBrand(ctx context.Context) ([]*Brand, error) {
	return b.repo.GetAllBrands(ctx)
}

func (b brandUsecase) GetBrandBySlug(ctx context.Context, slug string) (*Brand, error) {
	brand, err := b.repo.GetBrandBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}

	return brand, nil
}

func (b brandUsecase) UpdateBrand(ctx context.Context, brand *Brand) error {
	base := brand.Name
	if brand.Slug != "" {
		base = brand.Slug
	}

	baseSlug := util.GenerateSlug(base)
	brand.Slug = baseSlug
	for attempt := 2; ; attempt++ {
		err := b.repo.UpdateBrand(ctx, brand)
		if err == nil {
			return nil
		}
		if !dberrors.IsUniqueViolation(err) {
			return err
		}
		brand.Slug = fmt.Sprintf("%s-%d", baseSlug, attempt)
	}
}

func (b brandUsecase) UploadBrandLogo(ctx context.Context, id uuid.UUID, file multipart.File, header *multipart.FileHeader) (string, error) {
	url, err := b.uploader.Upload(ctx, file, header, id.String())
	if err != nil {
		return "", err
	}

	if err := b.repo.UpdateBrandLogo(ctx, id, url); err != nil {
		return "", err
	}

	return url, nil
}

func (b brandUsecase) DeleteBrand(ctx context.Context, id uuid.UUID) error {
	return b.repo.DeleteBrand(ctx, id)
}
