package catalog_test

import (
	"context"
	"database/sql"
	"errors"
	"mime/multipart"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/zona3-labs/mancing-id/internal/brand"
	"github.com/zona3-labs/mancing-id/internal/catalog"
	"github.com/zona3-labs/mancing-id/internal/transaction"
	"github.com/zona3-labs/mancing-id/internal/upload"
)

func TestBrandLogoServiceAssociatesBeforeFinalizingAndCleansReplacement(t *testing.T) {
	brandID := uuid.New()
	events := make([]string, 0, 6)
	previous := "https://assets.example/catalog/brand/old.webp"
	brands := &fakeBrandRepository{
		brand:  &brand.Brand{ID: brandID, Name: "Acme", LogoPath: &previous},
		events: &events,
	}
	media := &fakeMediaAdapter{
		temporary: upload.TemporaryImage{ID: "temporary-1", URL: "https://assets.example/catalog/brand/new.webp"},
		events:    &events,
	}
	transactions := &fakeTransactionManager{events: &events}
	service := catalog.NewBrandLogoService(brands, media, transactions)

	logo, err := service.UploadBrandLogo(context.Background(), brandID, nil, nil)
	if err != nil {
		t.Fatalf("upload brand logo: %v", err)
	}
	if logo != media.temporary.URL {
		t.Fatalf("logo = %q, want %q", logo, media.temporary.URL)
	}
	assertEvents(t, events, []string{
		"brand lookup",
		"temporary upload",
		"transaction begin",
		"brand association",
		"transaction commit",
		"temporary finalize",
		"brand lookup",
		"schedule old logo cleanup",
	})
}

func TestBrandLogoServiceDoesNotFinalizeWhenAssociationFails(t *testing.T) {
	brandID := uuid.New()
	events := make([]string, 0, 5)
	associationErr := errors.New("association failed")
	brands := &fakeBrandRepository{
		brand:        &brand.Brand{ID: brandID, Name: "Acme"},
		associateErr: associationErr,
		events:       &events,
	}
	media := &fakeMediaAdapter{
		temporary: upload.TemporaryImage{ID: "temporary-1", URL: "https://assets.example/catalog/brand/new.webp"},
		events:    &events,
	}
	service := catalog.NewBrandLogoService(brands, media, &fakeTransactionManager{events: &events})

	_, err := service.UploadBrandLogo(context.Background(), brandID, nil, nil)
	if !errors.Is(err, associationErr) {
		t.Fatalf("error = %v, want %v", err, associationErr)
	}
	if media.finalized != 0 {
		t.Fatalf("finalized temporary images = %d, want 0", media.finalized)
	}
	assertEvents(t, events, []string{
		"brand lookup",
		"temporary upload",
		"transaction begin",
		"brand association",
		"transaction rollback",
	})
}

func TestBrandLogoServiceCleansStaleTemporaryImagesThroughMediaAdapter(t *testing.T) {
	media := &fakeMediaAdapter{cleaned: 4}
	service := catalog.NewBrandLogoService(
		&fakeBrandRepository{},
		media,
		&fakeTransactionManager{},
	)

	cleaned, err := service.CleanStaleTemporaryImages(context.Background(), time.Unix(100, 0))
	if err != nil {
		t.Fatalf("clean stale images: %v", err)
	}
	if cleaned != 4 {
		t.Fatalf("cleaned = %d, want 4", cleaned)
	}
}

type fakeBrandRepository struct {
	brand        *brand.Brand
	associateErr error
	events       *[]string
}

func (f *fakeBrandRepository) GetBrandByID(context.Context, uuid.UUID) (*brand.Brand, error) {
	if f.events != nil {
		*f.events = append(*f.events, "brand lookup")
	}
	return f.brand, nil
}

func (f *fakeBrandRepository) AssociateLogo(_ context.Context, _ transaction.DBTX, _ uuid.UUID, _ string) (*string, error) {
	if f.events != nil {
		*f.events = append(*f.events, "brand association")
	}
	if f.associateErr != nil {
		return nil, f.associateErr
	}
	previousLogo := f.brand.LogoPath
	logo := "https://assets.example/catalog/brand/new.webp"
	f.brand.LogoPath = &logo
	if previousLogo == nil {
		return nil, nil
	}
	return previousLogo, nil
}

type fakeMediaAdapter struct {
	temporary upload.TemporaryImage
	events    *[]string
	finalized int
	cleaned   int
}

func (f *fakeMediaAdapter) UploadTemporary(context.Context, multipart.File, *multipart.FileHeader, uuid.UUID) (upload.TemporaryImage, error) {
	if f.events != nil {
		*f.events = append(*f.events, "temporary upload")
	}
	return f.temporary, nil
}

func (f *fakeMediaAdapter) FinalizeTemporary(context.Context, upload.TemporaryImage) error {
	f.finalized++
	if f.events != nil {
		*f.events = append(*f.events, "temporary finalize")
	}
	return nil
}

func (f *fakeMediaAdapter) ScheduleDelete(context.Context, string) error {
	if f.events != nil {
		*f.events = append(*f.events, "schedule old logo cleanup")
	}
	return nil
}

func (f *fakeMediaAdapter) CleanStaleTemporary(context.Context, time.Time) (int, error) {
	return f.cleaned, nil
}

type fakeTransactionManager struct {
	events *[]string
}

func (f *fakeTransactionManager) WithinTransaction(ctx context.Context, fn func(context.Context, transaction.DBTX) error) error {
	if f.events != nil {
		*f.events = append(*f.events, "transaction begin")
	}
	if err := fn(ctx, fakeDBTX{}); err != nil {
		if f.events != nil {
			*f.events = append(*f.events, "transaction rollback")
		}
		return err
	}
	if f.events != nil {
		*f.events = append(*f.events, "transaction commit")
	}
	return nil
}

type fakeDBTX struct{}

func (fakeDBTX) ExecContext(context.Context, string, ...any) (sql.Result, error) {
	return nil, nil
}

func (fakeDBTX) QueryContext(context.Context, string, ...any) (*sql.Rows, error) {
	return nil, nil
}

func (fakeDBTX) QueryRowContext(context.Context, string, ...any) *sql.Row {
	return nil
}

func assertEvents(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("events = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("events = %v, want %v", got, want)
		}
	}
}
