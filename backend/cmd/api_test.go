package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"testing"

	_ "github.com/lib/pq"
	"github.com/zona3-labs/mancing-id/internal/config"
	"github.com/zona3-labs/mancing-id/migrations"
)

func TestCategoryDraftLifecycleThroughHTTP(t *testing.T) {
	db := openIntegrationDatabase(t)
	defer db.Close()

	if err := migrations.Reset(db); err != nil {
		t.Fatalf("reset migrations: %v", err)
	}

	api := (&application{
		db: db,
		config: &config.Config{
			HttpServer: &config.HttpserverConfig{},
		},
	}).mount()

	parent := requestJSON(t, api, http.MethodPost, "/api/v1/admin/categories", map[string]any{
		"name": "Fishing Gear",
	})
	if parent.status != http.StatusCreated {
		t.Fatalf("create parent status = %d, body = %s", parent.status, parent.body)
	}
	var parentBody struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	decodeJSON(t, parent.body, &parentBody)

	created := requestJSON(t, api, http.MethodPost, "/api/v1/admin/categories", map[string]any{
		"name":      "Fly Fishing",
		"parent_id": parentBody.Data.ID,
	})
	if created.status != http.StatusCreated {
		t.Fatalf("create status = %d, body = %s", created.status, created.body)
	}

	var createdBody struct {
		Data struct {
			ID      string `json:"id"`
			Name    string `json:"name"`
			Slug    string `json:"slug"`
			Status  string `json:"status"`
			Version int64  `json:"version"`
		} `json:"data"`
	}
	decodeJSON(t, created.body, &createdBody)
	if createdBody.Data.Name != "Fly Fishing" || createdBody.Data.Slug != "fly-fishing" || createdBody.Data.Status != "draft" || createdBody.Data.Version != 1 {
		t.Fatalf("created category = %s", created.body)
	}

	categoryID := createdBody.Data.ID
	childBySlug := requestJSON(t, api, http.MethodGet, "/api/v1/admin/categories/"+createdBody.Data.Slug, nil)
	if childBySlug.status != http.StatusOK {
		t.Fatalf("child slug read status = %d, body = %s", childBySlug.status, childBySlug.body)
	}
	var childBySlugBody struct {
		Data struct {
			ID       string `json:"id"`
			Name     string `json:"name"`
			Children []any  `json:"children"`
		} `json:"data"`
	}
	decodeJSON(t, childBySlug.body, &childBySlugBody)
	if childBySlugBody.Data.ID != categoryID || childBySlugBody.Data.Name != "Fly Fishing" {
		t.Fatalf("child slug category = %s", childBySlug.body)
	}

	cycle := requestJSON(t, api, http.MethodPut, "/api/v1/admin/categories/"+categoryID, map[string]any{
		"name":      "Fly Fishing",
		"parent_id": categoryID,
		"version":   1,
	})
	if cycle.status != http.StatusBadRequest || cycle.contentType != "application/problem+json" {
		t.Fatalf("self-parent category = %#v", cycle)
	}
	var cycleProblem struct {
		Code string `json:"code"`
	}
	decodeJSON(t, cycle.body, &cycleProblem)
	if cycleProblem.Code != "category_cycle" {
		t.Fatalf("self-parent problem = %s", cycle.body)
	}

	read := requestJSON(t, api, http.MethodGet, "/api/v1/admin/categories/"+categoryID, nil)
	if read.status != http.StatusOK {
		t.Fatalf("read status = %d, body = %s", read.status, read.body)
	}

	updated := requestJSON(t, api, http.MethodPut, "/api/v1/admin/categories/"+categoryID, map[string]any{
		"name":      "Fly Reels",
		"parent_id": parentBody.Data.ID,
		"version":   1,
	})
	if updated.status != http.StatusOK {
		t.Fatalf("update status = %d, body = %s", updated.status, updated.body)
	}

	var updatedBody struct {
		Data struct {
			Version int64  `json:"version"`
			Slug    string `json:"slug"`
		} `json:"data"`
	}
	decodeJSON(t, updated.body, &updatedBody)
	if updatedBody.Data.Version != 2 || updatedBody.Data.Slug != "fly-reels" {
		t.Fatalf("updated category = %s", updated.body)
	}

	var wg sync.WaitGroup
	results := make(chan httpResult, 2)
	for _, name := range []string{"Fly Rods", "Fly Lines"} {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			results <- requestJSON(t, api, http.MethodPut, "/api/v1/admin/categories/"+categoryID, map[string]any{
				"name":    name,
				"version": 2,
			})
		}(name)
	}
	wg.Wait()
	close(results)

	var success, conflict int
	for result := range results {
		switch result.status {
		case http.StatusOK:
			success++
		case http.StatusConflict:
			conflict++
			if result.contentType != "application/problem+json" {
				t.Errorf("conflict content type = %q", result.contentType)
			}
			var problem struct {
				Code string `json:"code"`
			}
			decodeJSON(t, result.body, &problem)
			if problem.Code != "category_version_conflict" {
				t.Errorf("conflict problem = %s", result.body)
			}
		}
	}
	if success != 1 || conflict != 1 {
		t.Fatalf("concurrent updates: success=%d conflict=%d", success, conflict)
	}

	missing := requestJSON(t, api, http.MethodPut, "/api/v1/admin/categories/00000000-0000-0000-0000-000000000000", map[string]any{
		"name":    "Missing",
		"version": 1,
	})
	if missing.status != http.StatusNotFound || missing.contentType != "application/problem+json" {
		t.Fatalf("missing update = %#v", missing)
	}

	var missingProblem struct {
		Code string `json:"code"`
	}
	decodeJSON(t, missing.body, &missingProblem)
	if missingProblem.Code != "category_not_found" {
		t.Fatalf("missing problem = %s", missing.body)
	}

	missingDelete := requestJSON(t, api, http.MethodDelete, "/api/v1/admin/categories/00000000-0000-0000-0000-000000000000", map[string]any{
		"version": 1,
	})
	if missingDelete.status != http.StatusNotFound {
		t.Fatalf("missing delete status = %d, body = %s", missingDelete.status, missingDelete.body)
	}

	readAfterUpdate := requestJSON(t, api, http.MethodGet, "/api/v1/admin/categories/"+categoryID, nil)
	var current struct {
		Data struct {
			ParentID string `json:"parent_id"`
			Version  int64  `json:"version"`
		} `json:"data"`
	}
	decodeJSON(t, readAfterUpdate.body, &current)
	if current.Data.ParentID != parentBody.Data.ID {
		t.Fatalf("parent cleared by update = %s", readAfterUpdate.body)
	}
	deleted := requestJSON(t, api, http.MethodDelete, "/api/v1/admin/categories/"+categoryID, map[string]any{
		"version": current.Data.Version,
	})
	if deleted.status != http.StatusNoContent {
		t.Fatalf("delete status = %d, body = %s", deleted.status, deleted.body)
	}

	gone := requestJSON(t, api, http.MethodGet, "/api/v1/admin/categories/"+categoryID, nil)
	if gone.status != http.StatusNotFound {
		t.Fatalf("deleted category status = %d, body = %s", gone.status, gone.body)
	}
}

func TestCategoryPublicLifecycleThroughHTTP(t *testing.T) {
	db := openIntegrationDatabase(t)
	defer db.Close()
	if err := migrations.Reset(db); err != nil {
		t.Fatalf("reset migrations: %v", err)
	}

	api := (&application{db: db, config: &config.Config{HttpServer: &config.HttpserverConfig{}}}).mount()
	created := requestJSON(t, api, http.MethodPost, "/api/v1/admin/categories", map[string]any{"name": "Fishing Gear"})
	if created.status != http.StatusCreated {
		t.Fatalf("create status = %d, body = %s", created.status, created.body)
	}
	var createdBody struct {
		Data struct {
			ID      string `json:"id"`
			Slug    string `json:"slug"`
			Version int64  `json:"version"`
		} `json:"data"`
	}
	decodeJSON(t, created.body, &createdBody)

	draftRead := requestJSON(t, api, http.MethodGet, "/api/v1/categories/"+createdBody.Data.Slug, nil)
	if draftRead.status != http.StatusNotFound {
		t.Fatalf("draft public read status = %d, body = %s", draftRead.status, draftRead.body)
	}

	activated := requestJSON(t, api, http.MethodPost, "/api/v1/admin/categories/"+createdBody.Data.ID+"/activate", map[string]any{"version": createdBody.Data.Version})
	if activated.status != http.StatusNoContent {
		t.Fatalf("activate status = %d, body = %s", activated.status, activated.body)
	}

	publicRead := requestJSON(t, api, http.MethodGet, "/api/v1/categories/"+createdBody.Data.Slug, nil)
	if publicRead.status != http.StatusOK {
		t.Fatalf("active public read status = %d, body = %s", publicRead.status, publicRead.body)
	}

	var activeBody struct {
		Data struct {
			Status   string `json:"status"`
			Children []any  `json:"children"`
		} `json:"data"`
	}
	decodeJSON(t, publicRead.body, &activeBody)
	if activeBody.Data.Status != "active" {
		t.Fatalf("active public category = %s", publicRead.body)
	}

	child := requestJSON(t, api, http.MethodPost, "/api/v1/admin/categories", map[string]any{
		"name":      "Fly Fishing",
		"parent_id": createdBody.Data.ID,
	})
	if child.status != http.StatusCreated {
		t.Fatalf("create child status = %d, body = %s", child.status, child.body)
	}
	var childBody struct {
		Data struct {
			ID      string `json:"id"`
			Version int64  `json:"version"`
		} `json:"data"`
	}
	decodeJSON(t, child.body, &childBody)
	childActivated := requestJSON(t, api, http.MethodPost, "/api/v1/admin/categories/"+childBody.Data.ID+"/activate", map[string]any{"version": childBody.Data.Version})
	if childActivated.status != http.StatusNoContent {
		t.Fatalf("activate child status = %d, body = %s", childActivated.status, childActivated.body)
	}

	publicRead = requestJSON(t, api, http.MethodGet, "/api/v1/categories/"+createdBody.Data.Slug, nil)
	decodeJSON(t, publicRead.body, &activeBody)
	if len(activeBody.Data.Children) != 1 {
		t.Fatalf("active category children = %s", publicRead.body)
	}

	retireParent := requestJSON(t, api, http.MethodPost, "/api/v1/admin/categories/"+createdBody.Data.ID+"/retire", map[string]any{"version": createdBody.Data.Version + 1})
	if retireParent.status != http.StatusConflict {
		t.Fatalf("retire parent with child status = %d, body = %s", retireParent.status, retireParent.body)
	}
	var retireProblem struct {
		Code string `json:"code"`
	}
	decodeJSON(t, retireParent.body, &retireProblem)
	if retireProblem.Code != "category_has_children" {
		t.Fatalf("retire parent problem = %s", retireParent.body)
	}

	retireChild := requestJSON(t, api, http.MethodPost, "/api/v1/admin/categories/"+childBody.Data.ID+"/retire", map[string]any{"version": childBody.Data.Version + 1})
	if retireChild.status != http.StatusNoContent {
		t.Fatalf("retire child status = %d, body = %s", retireChild.status, retireChild.body)
	}
	retired := requestJSON(t, api, http.MethodPost, "/api/v1/admin/categories/"+createdBody.Data.ID+"/retire", map[string]any{"version": createdBody.Data.Version + 1})
	if retired.status != http.StatusNoContent {
		t.Fatalf("retire status = %d, body = %s", retired.status, retired.body)
	}
	retiredRead := requestJSON(t, api, http.MethodGet, "/api/v1/categories/"+createdBody.Data.Slug, nil)
	if retiredRead.status != http.StatusNotFound {
		t.Fatalf("retired public read status = %d, body = %s", retiredRead.status, retiredRead.body)
	}
}

func TestBrandLifecycleThroughHTTP(t *testing.T) {
	db := openIntegrationDatabase(t)
	defer db.Close()
	if err := migrations.Reset(db); err != nil {
		t.Fatalf("reset migrations: %v", err)
	}

	api := (&application{db: db, config: &config.Config{HttpServer: &config.HttpserverConfig{}}}).mount()
	created := requestJSON(t, api, http.MethodPost, "/api/v1/admin/brands", map[string]any{
		"name": "Acme Fishing",
		"slug": "acme",
	})
	if created.status != http.StatusCreated {
		t.Fatalf("create brand status = %d, body = %s", created.status, created.body)
	}
	var createdBody struct {
		Data struct {
			ID      string `json:"id"`
			Name    string `json:"name"`
			Slug    string `json:"slug"`
			Status  string `json:"status"`
			Version int64  `json:"version"`
		} `json:"data"`
	}
	decodeJSON(t, created.body, &createdBody)
	if createdBody.Data.Name != "Acme Fishing" || createdBody.Data.Slug != "acme" || createdBody.Data.Status != "draft" || createdBody.Data.Version != 1 {
		t.Fatalf("created brand = %s", created.body)
	}

	draftPublicRead := requestJSON(t, api, http.MethodGet, "/api/v1/brands/acme", nil)
	if draftPublicRead.status != http.StatusNotFound || draftPublicRead.contentType != "application/problem+json" {
		t.Fatalf("draft public brand = %#v", draftPublicRead)
	}
	draftPublicCollection := requestJSON(t, api, http.MethodGet, "/api/v1/brands", nil)
	if draftPublicCollection.status != http.StatusOK || bytes.Contains(draftPublicCollection.body, []byte(`"slug":"acme"`)) {
		t.Fatalf("draft public brand collection = %#v", draftPublicCollection)
	}

	updated := requestJSON(t, api, http.MethodPut, "/api/v1/admin/brands/"+createdBody.Data.ID, map[string]any{
		"name":    "Acme Tackle",
		"version": 1,
	})
	if updated.status != http.StatusOK {
		t.Fatalf("update brand status = %d, body = %s", updated.status, updated.body)
	}
	var updatedBody struct {
		Data struct {
			Name    string `json:"name"`
			Slug    string `json:"slug"`
			Status  string `json:"status"`
			Version int64  `json:"version"`
		} `json:"data"`
	}
	decodeJSON(t, updated.body, &updatedBody)
	if updatedBody.Data.Name != "Acme Tackle" || updatedBody.Data.Slug != "acme-tackle" || updatedBody.Data.Status != "draft" || updatedBody.Data.Version != 2 {
		t.Fatalf("updated brand = %s", updated.body)
	}

	var wg sync.WaitGroup
	results := make(chan httpResult, 2)
	for _, name := range []string{"Acme Rods", "Acme Lines"} {
		wg.Add(1)
		go func(name string) {
			defer wg.Done()
			results <- requestJSON(t, api, http.MethodPut, "/api/v1/admin/brands/"+createdBody.Data.ID, map[string]any{
				"name":    name,
				"version": 2,
			})
		}(name)
	}
	wg.Wait()
	close(results)
	var success, conflict int
	for result := range results {
		switch result.status {
		case http.StatusOK:
			success++
		case http.StatusConflict:
			conflict++
			if result.contentType != "application/problem+json" {
				t.Errorf("brand conflict content type = %q", result.contentType)
			}
		default:
			t.Errorf("unexpected concurrent brand update = %#v", result)
		}
	}
	if success != 1 || conflict != 1 {
		t.Fatalf("concurrent brand updates: success=%d conflict=%d", success, conflict)
	}
	currentRead := requestJSON(t, api, http.MethodGet, "/api/v1/admin/brands/"+createdBody.Data.ID, nil)
	if currentRead.status != http.StatusOK {
		t.Fatalf("current brand status = %d, body = %s", currentRead.status, currentRead.body)
	}
	var currentBody struct {
		Data struct {
			Slug string `json:"slug"`
		} `json:"data"`
	}
	decodeJSON(t, currentRead.body, &currentBody)

	activated := requestJSON(t, api, http.MethodPost, "/api/v1/admin/brands/"+createdBody.Data.ID+"/activate", map[string]any{"version": 3})
	if activated.status != http.StatusNoContent {
		t.Fatalf("activate brand status = %d, body = %s", activated.status, activated.body)
	}

	publicRead := requestJSON(t, api, http.MethodGet, "/api/v1/brands/"+currentBody.Data.Slug, nil)
	if publicRead.status != http.StatusOK {
		t.Fatalf("active public brand status = %d, body = %s", publicRead.status, publicRead.body)
	}
	var publicBody struct {
		Data struct {
			Status  string `json:"status"`
			Version int64  `json:"version"`
		} `json:"data"`
	}
	decodeJSON(t, publicRead.body, &publicBody)
	if publicBody.Data.Status != "active" || publicBody.Data.Version != 4 {
		t.Fatalf("active public brand = %s", publicRead.body)
	}
	reserved := requestJSON(t, api, http.MethodPost, "/api/v1/admin/brands", map[string]any{
		"name": "Replacement Brand",
		"slug": currentBody.Data.Slug,
	})
	if reserved.status != http.StatusCreated {
		t.Fatalf("reserved slug brand status = %d, body = %s", reserved.status, reserved.body)
	}
	var reservedBody struct {
		Data struct {
			ID   string `json:"id"`
			Slug string `json:"slug"`
		} `json:"data"`
	}
	decodeJSON(t, reserved.body, &reservedBody)
	if reservedBody.Data.Slug == currentBody.Data.Slug {
		t.Fatalf("published brand slug was reused: %s", reserved.body)
	}
	reservedDelete := requestJSON(t, api, http.MethodDelete, "/api/v1/admin/brands/"+reservedBody.Data.ID, map[string]any{"version": 1})
	if reservedDelete.status != http.StatusNoContent {
		t.Fatalf("delete reserved slug brand status = %d, body = %s", reservedDelete.status, reservedDelete.body)
	}

	staleUpdate := requestJSON(t, api, http.MethodPut, "/api/v1/admin/brands/"+createdBody.Data.ID, map[string]any{
		"name":    "Stale Acme",
		"version": 2,
	})
	if staleUpdate.status != http.StatusConflict {
		t.Fatalf("stale brand update = %#v", staleUpdate)
	}

	deactivated := requestJSON(t, api, http.MethodPost, "/api/v1/admin/brands/"+createdBody.Data.ID+"/deactivate", map[string]any{"version": 4})
	if deactivated.status != http.StatusNoContent {
		t.Fatalf("deactivate brand status = %d, body = %s", deactivated.status, deactivated.body)
	}
	inactivePublicRead := requestJSON(t, api, http.MethodGet, "/api/v1/brands/"+currentBody.Data.Slug, nil)
	if inactivePublicRead.status != http.StatusOK {
		t.Fatalf("inactive public brand status = %d, body = %s", inactivePublicRead.status, inactivePublicRead.body)
	}
	decodeJSON(t, inactivePublicRead.body, &publicBody)
	if publicBody.Data.Status != "inactive" || publicBody.Data.Version != 5 {
		t.Fatalf("inactive public brand = %s", inactivePublicRead.body)
	}

	updatedInactive := requestJSON(t, api, http.MethodPut, "/api/v1/admin/brands/"+createdBody.Data.ID, map[string]any{
		"name":    "Renamed Acme",
		"version": 5,
	})
	if updatedInactive.status != http.StatusConflict {
		t.Fatalf("inactive brand update = %#v", updatedInactive)
	}

	reactivated := requestJSON(t, api, http.MethodPost, "/api/v1/admin/brands/"+createdBody.Data.ID+"/reactivate", map[string]any{"version": 5})
	if reactivated.status != http.StatusNoContent {
		t.Fatalf("reactivate brand status = %d, body = %s", reactivated.status, reactivated.body)
	}

	changedPublishedSlug := requestJSON(t, api, http.MethodPut, "/api/v1/admin/brands/"+createdBody.Data.ID, map[string]any{
		"name":    "Renamed Acme",
		"slug":    "renamed-acme",
		"version": 6,
	})
	if changedPublishedSlug.status != http.StatusConflict {
		t.Fatalf("published brand update = %#v", changedPublishedSlug)
	}

	referencedDraft := requestJSON(t, api, http.MethodPost, "/api/v1/admin/brands", map[string]any{"name": "Referenced Brand"})
	if referencedDraft.status != http.StatusCreated {
		t.Fatalf("create referenced brand status = %d, body = %s", referencedDraft.status, referencedDraft.body)
	}
	var referencedBody struct {
		Data struct {
			ID      string `json:"id"`
			Version int64  `json:"version"`
		} `json:"data"`
	}
	decodeJSON(t, referencedDraft.body, &referencedBody)
	product := requestJSON(t, api, http.MethodPost, "/api/v1/products", map[string]any{
		"name":     "Referenced Product",
		"status":   "draft",
		"brand_id": referencedBody.Data.ID,
	})
	if product.status != http.StatusCreated {
		t.Fatalf("create referenced product status = %d, body = %s", product.status, product.body)
	}
	referencedDelete := requestJSON(t, api, http.MethodDelete, "/api/v1/admin/brands/"+referencedBody.Data.ID, map[string]any{"version": referencedBody.Data.Version})
	if referencedDelete.status != http.StatusConflict {
		t.Fatalf("delete referenced brand = %#v", referencedDelete)
	}

	deletableDraft := requestJSON(t, api, http.MethodPost, "/api/v1/admin/brands", map[string]any{"name": "Disposable Brand"})
	if deletableDraft.status != http.StatusCreated {
		t.Fatalf("create deletable brand status = %d, body = %s", deletableDraft.status, deletableDraft.body)
	}
	var deletableBody struct {
		Data struct {
			ID      string `json:"id"`
			Slug    string `json:"slug"`
			Version int64  `json:"version"`
		} `json:"data"`
	}
	decodeJSON(t, deletableDraft.body, &deletableBody)
	deleted := requestJSON(t, api, http.MethodDelete, "/api/v1/admin/brands/"+deletableBody.Data.ID, map[string]any{"version": deletableBody.Data.Version})
	if deleted.status != http.StatusNoContent {
		t.Fatalf("delete draft brand status = %d, body = %s", deleted.status, deleted.body)
	}

	missing := requestJSON(t, api, http.MethodGet, "/api/v1/admin/brands/00000000-0000-0000-0000-000000000000", nil)
	if missing.status != http.StatusNotFound || missing.contentType != "application/problem+json" {
		t.Fatalf("missing brand = %#v", missing)
	}
}

func TestProductUpdateTargetsRequestedProductThroughHTTP(t *testing.T) {
	db := openIntegrationDatabase(t)
	defer db.Close()
	if err := migrations.Reset(db); err != nil {
		t.Fatalf("reset migrations: %v", err)
	}

	api := (&application{db: db, config: &config.Config{HttpServer: &config.HttpserverConfig{}}}).mount()
	created := requestJSON(t, api, http.MethodPost, "/api/v1/products", map[string]any{
		"name":   "Spinning Reel",
		"status": "draft",
	})
	if created.status != http.StatusCreated {
		t.Fatalf("create product status = %d, body = %s", created.status, created.body)
	}
	var createdBody struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	decodeJSON(t, created.body, &createdBody)

	updated := requestJSON(t, api, http.MethodPut, "/api/v1/products/"+createdBody.Data.ID, map[string]any{
		"name":    "Tournament Spinning Reel",
		"status":  "draft",
		"version": 1,
	})
	if updated.status != http.StatusOK {
		t.Fatalf("update product status = %d, body = %s", updated.status, updated.body)
	}
	var updatedBody struct {
		Data struct {
			ID      string `json:"id"`
			Name    string `json:"name"`
			Version int64  `json:"version"`
		} `json:"data"`
	}
	decodeJSON(t, updated.body, &updatedBody)
	if updatedBody.Data.ID != createdBody.Data.ID || updatedBody.Data.Name != "Tournament Spinning Reel" || updatedBody.Data.Version != 2 {
		t.Fatalf("updated product = %s", updated.body)
	}
	stale := requestJSON(t, api, http.MethodPut, "/api/v1/products/"+createdBody.Data.ID, map[string]any{
		"name":    "Stale Reel",
		"status":  "draft",
		"version": 1,
	})
	if stale.status != http.StatusConflict {
		t.Fatalf("stale product update = %#v", stale)
	}

	read := requestJSON(t, api, http.MethodGet, "/api/v1/products/"+createdBody.Data.ID, nil)
	if read.status != http.StatusOK || !bytes.Contains(read.body, []byte("Tournament Spinning Reel")) {
		t.Fatalf("updated product read = %#v", read)
	}
}

type httpResult struct {
	status      int
	body        []byte
	contentType string
}

func requestJSON(t *testing.T, handler http.Handler, method, path string, payload map[string]any) httpResult {
	t.Helper()
	var body io.Reader
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			t.Fatal(err)
		}
		body = bytes.NewReader(encoded)
	}

	req := httptest.NewRequest(method, path, body)
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, req)
	return httpResult{
		status:      recorder.Code,
		body:        recorder.Body.Bytes(),
		contentType: recorder.Header().Get("Content-Type"),
	}
}

func decodeJSON(t *testing.T, body []byte, target any) {
	t.Helper()
	if err := json.Unmarshal(body, target); err != nil {
		t.Fatalf("decode %s: %v", body, err)
	}
}

func openIntegrationDatabase(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_DSN")
	if dsn == "" {
		dsn = "host=localhost port=5435 user=postgres password=postgres dbname=mancing_id sslmode=disable"
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	if err := db.Ping(); err != nil {
		db.Close()
		t.Fatalf("ping database: %v", err)
	}
	return db
}
