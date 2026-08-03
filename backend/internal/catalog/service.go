package catalog

import (
	"context"
	"mime/multipart"
	"time"

	"github.com/google/uuid"
	"github.com/zona3-labs/mancing-id/internal/brand"
	"github.com/zona3-labs/mancing-id/internal/transaction"
	"github.com/zona3-labs/mancing-id/internal/upload"
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
