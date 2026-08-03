package main

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"
	"github.com/zona3-labs/mancing-id/internal/config"
	"github.com/zona3-labs/mancing-id/internal/upload"
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

func TestCategoryPublicHierarchyAndSlugReservationThroughHTTP(t *testing.T) {
	db := openIntegrationDatabase(t)
	defer db.Close()
	if err := migrations.Reset(db); err != nil {
		t.Fatalf("reset migrations: %v", err)
	}

	api := (&application{db: db, config: &config.Config{HttpServer: &config.HttpserverConfig{}}}).mount()
	blockedParent := createCategoryHTTP(t, api, "Blocked Parent", "")
	blockedChild := createCategoryHTTP(t, api, "Blocked Child", blockedParent.ID)
	blockedActivation := requestJSON(t, api, http.MethodPost, "/api/v1/admin/categories/"+blockedChild.ID+"/activate", map[string]any{"version": blockedChild.Version})
	assertProblem(t, blockedActivation, http.StatusConflict, "category_ancestor_not_active")

	cycleRoot := createCategoryHTTP(t, api, "Cycle Root", "")
	cycleChild := createCategoryHTTP(t, api, "Cycle Child", cycleRoot.ID)
	cycleGrandchild := createCategoryHTTP(t, api, "Cycle Grandchild", cycleChild.ID)
	cycle := requestJSON(t, api, http.MethodPut, "/api/v1/admin/categories/"+cycleRoot.ID, map[string]any{
		"name":      "Cycle Root",
		"parent_id": cycleGrandchild.ID,
		"version":   cycleRoot.Version,
	})
	assertProblem(t, cycle, http.StatusBadRequest, "category_cycle")

	root := createCategoryHTTP(t, api, "Fishing Gear", "")
	child := createCategoryHTTP(t, api, "Fly Fishing", root.ID)
	grandchild := createCategoryHTTP(t, api, "Fly Rods", child.ID)
	activateCategoryHTTP(t, api, root)
	activateCategoryHTTP(t, api, child)
	activateCategoryHTTP(t, api, grandchild)

	collection := requestJSON(t, api, http.MethodGet, "/api/v1/categories", nil)
	if collection.status != http.StatusOK {
		t.Fatalf("public category collection status = %d, body = %s", collection.status, collection.body)
	}
	var collectionBody struct {
		Data []struct {
			ID       string `json:"id"`
			Children []struct {
				ID       string `json:"id"`
				Children []struct {
					ID string `json:"id"`
				} `json:"children"`
			} `json:"children"`
		} `json:"data"`
	}
	decodeJSON(t, collection.body, &collectionBody)
	if len(collectionBody.Data) != 1 || collectionBody.Data[0].ID != root.ID {
		t.Fatalf("public roots = %s", collection.body)
	}
	if len(collectionBody.Data[0].Children) != 1 || collectionBody.Data[0].Children[0].ID != child.ID {
		t.Fatalf("public children = %s", collection.body)
	}
	if len(collectionBody.Data[0].Children[0].Children) != 1 || collectionBody.Data[0].Children[0].Children[0].ID != grandchild.ID {
		t.Fatalf("public grandchildren = %s", collection.body)
	}

	childRead := requestJSON(t, api, http.MethodGet, "/api/v1/categories/"+child.Slug, nil)
	if childRead.status != http.StatusOK {
		t.Fatalf("public non-root category status = %d, body = %s", childRead.status, childRead.body)
	}
	var childReadBody struct {
		Data struct {
			ID       string `json:"id"`
			Children []struct {
				ID string `json:"id"`
			} `json:"children"`
		} `json:"data"`
	}
	decodeJSON(t, childRead.body, &childReadBody)
	if childReadBody.Data.ID != child.ID || len(childReadBody.Data.Children) != 1 || childReadBody.Data.Children[0].ID != grandchild.ID {
		t.Fatalf("public non-root subtree = %s", childRead.body)
	}

	retiredChild := requestJSON(t, api, http.MethodPost, "/api/v1/admin/categories/"+grandchild.ID+"/retire", map[string]any{"version": grandchild.Version + 1})
	if retiredChild.status != http.StatusNoContent {
		t.Fatalf("retire grandchild status = %d, body = %s", retiredChild.status, retiredChild.body)
	}
	retiredChild = requestJSON(t, api, http.MethodPost, "/api/v1/admin/categories/"+child.ID+"/retire", map[string]any{"version": child.Version + 1})
	if retiredChild.status != http.StatusNoContent {
		t.Fatalf("retire child status = %d, body = %s", retiredChild.status, retiredChild.body)
	}

	collection = requestJSON(t, api, http.MethodGet, "/api/v1/categories", nil)
	if bytes.Contains(collection.body, []byte(grandchild.Slug)) || bytes.Contains(collection.body, []byte(child.Slug)) {
		t.Fatalf("retired categories leaked into public collection = %s", collection.body)
	}
	retiredRead := requestJSON(t, api, http.MethodGet, "/api/v1/categories/"+child.Slug, nil)
	if retiredRead.status != http.StatusNotFound {
		t.Fatalf("retired non-root category status = %d, body = %s", retiredRead.status, retiredRead.body)
	}
	retiredRoot := requestJSON(t, api, http.MethodPost, "/api/v1/admin/categories/"+root.ID+"/retire", map[string]any{"version": root.Version + 1})
	if retiredRoot.status != http.StatusNoContent {
		t.Fatalf("retire root status = %d, body = %s", retiredRoot.status, retiredRoot.body)
	}

	reserved := createCategoryWithSlugHTTP(t, api, "Replacement Fishing Gear", "Fishing Gear", "")
	if reserved.Slug == root.Slug {
		t.Fatalf("published slug was reused: %s", reserved.Slug)
	}
	changed := requestJSON(t, api, http.MethodPut, "/api/v1/admin/categories/"+root.ID, map[string]any{
		"name":    "Renamed Fishing Gear",
		"slug":    "renamed-fishing-gear",
		"version": root.Version + 2,
	})
	assertProblem(t, changed, http.StatusConflict, "category_not_editable")

	const uuidSlug = "11111111-1111-1111-1111-111111111111"
	uuidCategory := createCategoryWithSlugHTTP(t, api, "UUID Slug Category", uuidSlug, "")
	activateCategoryHTTP(t, api, uuidCategory)
	uuidRead := requestJSON(t, api, http.MethodGet, "/api/v1/categories/"+uuidSlug, nil)
	if uuidRead.status != http.StatusOK {
		t.Fatalf("UUID-shaped public slug status = %d, body = %s", uuidRead.status, uuidRead.body)
	}
}

type categoryHTTP struct {
	ID      string `json:"id"`
	Slug    string `json:"slug"`
	Version int64  `json:"version"`
}

func createCategoryHTTP(t *testing.T, api http.Handler, name, parentID string) categoryHTTP {
	return createCategoryWithSlugHTTP(t, api, name, "", parentID)
}

func createCategoryWithSlugHTTP(t *testing.T, api http.Handler, name, slug, parentID string) categoryHTTP {
	t.Helper()
	payload := map[string]any{"name": name}
	if slug != "" {
		payload["slug"] = slug
	}
	if parentID != "" {
		payload["parent_id"] = parentID
	}
	created := requestJSON(t, api, http.MethodPost, "/api/v1/admin/categories", payload)
	if created.status != http.StatusCreated {
		t.Fatalf("create category status = %d, body = %s", created.status, created.body)
	}
	var body struct {
		Data categoryHTTP `json:"data"`
	}
	decodeJSON(t, created.body, &body)
	return body.Data
}

func activateCategoryHTTP(t *testing.T, api http.Handler, category categoryHTTP) {
	t.Helper()
	activated := requestJSON(t, api, http.MethodPost, "/api/v1/admin/categories/"+category.ID+"/activate", map[string]any{"version": category.Version})
	if activated.status != http.StatusNoContent {
		t.Fatalf("activate category status = %d, body = %s", activated.status, activated.body)
	}
}

func assertProblem(t *testing.T, result httpResult, status int, code string) {
	t.Helper()
	if result.status != status || result.contentType != "application/problem+json" {
		t.Fatalf("problem response = %#v", result)
	}
	var problem struct {
		Code string `json:"code"`
	}
	decodeJSON(t, result.body, &problem)
	if problem.Code != code {
		t.Fatalf("problem code = %q, want %q; body = %s", problem.Code, code, result.body)
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
	referencedActivated := requestJSON(t, api, http.MethodPost, "/api/v1/admin/brands/"+referencedBody.Data.ID+"/activate", map[string]any{"version": referencedBody.Data.Version})
	if referencedActivated.status != http.StatusNoContent {
		t.Fatalf("activate referenced brand status = %d, body = %s", referencedActivated.status, referencedActivated.body)
	}
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

func TestBrandLogoManagedLifecycleThroughHTTP(t *testing.T) {
	db := openIntegrationDatabase(t)
	defer db.Close()
	if err := migrations.Reset(db); err != nil {
		t.Fatalf("reset migrations: %v", err)
	}

	media := &httpFakeMediaAdapter{}
	api := (&application{
		db:     db,
		media:  media,
		config: &config.Config{HttpServer: &config.HttpserverConfig{}},
	}).mount()
	created := requestJSON(t, api, http.MethodPost, "/api/v1/admin/brands", map[string]any{"name": "Logo Brand"})
	if created.status != http.StatusCreated {
		t.Fatalf("create brand status = %d, body = %s", created.status, created.body)
	}
	var createdBody struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	decodeJSON(t, created.body, &createdBody)

	first := requestMultipart(t, api, "/api/v1/admin/brands/"+createdBody.Data.ID+"/logo", []byte("first"))
	if first.status != http.StatusOK || media.finalized != 1 {
		t.Fatalf("first logo upload = %#v, finalized=%d", first, media.finalized)
	}
	var firstBody struct {
		Data struct {
			LogoPath string `json:"logo_path"`
		} `json:"data"`
	}
	decodeJSON(t, first.body, &firstBody)

	second := requestMultipart(t, api, "/api/v1/admin/brands/"+createdBody.Data.ID+"/logo", []byte("second"))
	if second.status != http.StatusOK || media.finalized != 2 {
		t.Fatalf("replacement logo upload = %#v, finalized=%d", second, media.finalized)
	}
	if len(media.deleted) != 1 || media.deleted[0] != firstBody.Data.LogoPath {
		t.Fatalf("deleted managed logos = %v, want [%q]", media.deleted, firstBody.Data.LogoPath)
	}

	external := requestJSON(t, api, http.MethodPut, "/api/v1/admin/brands/"+createdBody.Data.ID, map[string]any{
		"name":      "Logo Brand",
		"logo_path": "https://external.example/logo.png",
		"version":   1,
	})
	if external.status != http.StatusBadRequest || external.contentType != "application/problem+json" {
		t.Fatalf("external logo input = %#v", external)
	}
	var problem struct {
		Code string `json:"code"`
	}
	decodeJSON(t, external.body, &problem)
	if problem.Code != "managed_logo_required" {
		t.Fatalf("external logo problem = %s", external.body)
	}

	for _, testCase := range []struct {
		errCode string
		err     error
	}{
		{errCode: "invalid_logo_type", err: upload.ErrInvalidFileType},
		{errCode: "logo_too_large", err: upload.ErrFileTooLarge},
		{errCode: "invalid_logo_image", err: upload.ErrImageDecode},
	} {
		media.uploadErr = testCase.err
		invalid := requestMultipart(t, api, "/api/v1/admin/brands/"+createdBody.Data.ID+"/logo", []byte("invalid"))
		if invalid.status != http.StatusBadRequest || invalid.contentType != "application/problem+json" {
			t.Fatalf("%s response = %#v", testCase.errCode, invalid)
		}
		decodeJSON(t, invalid.body, &problem)
		if problem.Code != testCase.errCode {
			t.Fatalf("%s problem = %s", testCase.errCode, invalid.body)
		}
	}
}

type httpFakeMediaAdapter struct {
	next      int
	finalized int
	deleted   []string
	uploadErr error
}

func (f *httpFakeMediaAdapter) UploadTemporary(context.Context, multipart.File, *multipart.FileHeader, uuid.UUID) (upload.TemporaryImage, error) {
	if f.uploadErr != nil {
		return upload.TemporaryImage{}, f.uploadErr
	}
	f.next++
	return upload.TemporaryImage{URL: "https://managed.example/logo-" + strconv.Itoa(f.next)}, nil
}

func (f *httpFakeMediaAdapter) FinalizeTemporary(context.Context, upload.TemporaryImage) error {
	f.finalized++
	return nil
}

func (f *httpFakeMediaAdapter) ScheduleDelete(_ context.Context, logo string) error {
	f.deleted = append(f.deleted, logo)
	return nil
}

func (f *httpFakeMediaAdapter) CleanStaleTemporary(context.Context, time.Time) (int, error) {
	return 0, nil
}

func TestProductUpdateTargetsRequestedProductThroughHTTP(t *testing.T) {
	db := openIntegrationDatabase(t)
	defer db.Close()
	if err := migrations.Reset(db); err != nil {
		t.Fatalf("reset migrations: %v", err)
	}

	api := (&application{db: db, config: &config.Config{HttpServer: &config.HttpserverConfig{}}}).mount()
	brand := requestJSON(t, api, http.MethodPost, "/api/v1/admin/brands", map[string]any{"name": "Spinning Brand"})
	if brand.status != http.StatusCreated {
		t.Fatalf("create product brand status = %d, body = %s", brand.status, brand.body)
	}
	var brandBody struct {
		Data struct {
			ID      string `json:"id"`
			Version int64  `json:"version"`
		} `json:"data"`
	}
	decodeJSON(t, brand.body, &brandBody)
	activatedBrand := requestJSON(t, api, http.MethodPost, "/api/v1/admin/brands/"+brandBody.Data.ID+"/activate", map[string]any{"version": brandBody.Data.Version})
	if activatedBrand.status != http.StatusNoContent {
		t.Fatalf("activate product brand status = %d, body = %s", activatedBrand.status, activatedBrand.body)
	}
	created := requestJSON(t, api, http.MethodPost, "/api/v1/products", map[string]any{
		"name":     "Spinning Reel",
		"status":   "draft",
		"brand_id": brandBody.Data.ID,
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
		"name":     "Tournament Spinning Reel",
		"status":   "draft",
		"brand_id": brandBody.Data.ID,
		"version":  1,
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
		"name":     "Stale Reel",
		"status":   "draft",
		"brand_id": brandBody.Data.ID,
		"version":  1,
	})
	if stale.status != http.StatusConflict {
		t.Fatalf("stale product update = %#v", stale)
	}

	read := requestJSON(t, api, http.MethodGet, "/api/v1/products/"+createdBody.Data.ID, nil)
	if read.status != http.StatusOK || !bytes.Contains(read.body, []byte("Tournament Spinning Reel")) {
		t.Fatalf("updated product read = %#v", read)
	}
}

func TestProductDraftBrandContractThroughHTTP(t *testing.T) {
	db := openIntegrationDatabase(t)
	defer db.Close()
	if err := migrations.Reset(db); err != nil {
		t.Fatalf("reset migrations: %v", err)
	}

	api := (&application{db: db, config: &config.Config{HttpServer: &config.HttpserverConfig{}}}).mount()
	createdBrand := requestJSON(t, api, http.MethodPost, "/api/v1/admin/brands", map[string]any{"name": "Authoring Brand"})
	if createdBrand.status != http.StatusCreated {
		t.Fatalf("create brand status = %d, body = %s", createdBrand.status, createdBrand.body)
	}
	var brandBody struct {
		Data struct {
			ID      string `json:"id"`
			Name    string `json:"name"`
			Slug    string `json:"slug"`
			Version int64  `json:"version"`
		} `json:"data"`
	}
	decodeJSON(t, createdBrand.body, &brandBody)

	draftBrand := requestJSON(t, api, http.MethodPost, "/api/v1/products", map[string]any{
		"name": "Draft Brand Product", "status": "draft", "brand_id": brandBody.Data.ID,
	})
	assertProblem(t, draftBrand, http.StatusConflict, "brand_not_active")

	missingBrand := requestJSON(t, api, http.MethodPost, "/api/v1/products", map[string]any{
		"name": "Missing Brand Product", "status": "draft", "brand_id": "00000000-0000-0000-0000-000000000000",
	})
	assertProblem(t, missingBrand, http.StatusNotFound, "brand_not_found")

	activated := requestJSON(t, api, http.MethodPost, "/api/v1/admin/brands/"+brandBody.Data.ID+"/activate", map[string]any{"version": brandBody.Data.Version})
	if activated.status != http.StatusNoContent {
		t.Fatalf("activate brand status = %d, body = %s", activated.status, activated.body)
	}
	deactivated := requestJSON(t, api, http.MethodPost, "/api/v1/admin/brands/"+brandBody.Data.ID+"/deactivate", map[string]any{"version": brandBody.Data.Version + 1})
	if deactivated.status != http.StatusNoContent {
		t.Fatalf("deactivate brand status = %d, body = %s", deactivated.status, deactivated.body)
	}
	inactiveBrand := requestJSON(t, api, http.MethodPost, "/api/v1/products", map[string]any{
		"name": "Inactive Brand Product", "status": "draft", "brand_id": brandBody.Data.ID,
	})
	assertProblem(t, inactiveBrand, http.StatusConflict, "brand_not_active")
	reactivated := requestJSON(t, api, http.MethodPost, "/api/v1/admin/brands/"+brandBody.Data.ID+"/reactivate", map[string]any{"version": brandBody.Data.Version + 2})
	if reactivated.status != http.StatusNoContent {
		t.Fatalf("reactivate brand status = %d, body = %s", reactivated.status, reactivated.body)
	}

	created := requestJSON(t, api, http.MethodPost, "/api/v1/products", map[string]any{
		"name": "Authoring Reel", "slug": "authoring-reel", "status": "draft", "brand_id": brandBody.Data.ID,
	})
	if created.status != http.StatusCreated {
		t.Fatalf("create product status = %d, body = %s", created.status, created.body)
	}
	var createdBody struct {
		Data struct {
			ID      string `json:"id"`
			Slug    string `json:"slug"`
			Status  string `json:"status"`
			Version int64  `json:"version"`
		} `json:"data"`
	}
	decodeJSON(t, created.body, &createdBody)
	if createdBody.Data.Status != "draft" || createdBody.Data.Slug != "authoring-reel" || createdBody.Data.Version != 1 {
		t.Fatalf("created product = %s", created.body)
	}

	detail := requestJSON(t, api, http.MethodGet, "/api/v1/products/"+createdBody.Data.ID, nil)
	if detail.status != http.StatusOK {
		t.Fatalf("product detail status = %d, body = %s", detail.status, detail.body)
	}
	var detailBody struct {
		Data struct {
			Brand struct {
				ID   string `json:"id"`
				Name string `json:"name"`
				Slug string `json:"slug"`
			} `json:"brand"`
		} `json:"data"`
	}
	decodeJSON(t, detail.body, &detailBody)
	if detailBody.Data.Brand.ID != brandBody.Data.ID || detailBody.Data.Brand.Name != brandBody.Data.Name || detailBody.Data.Brand.Slug != brandBody.Data.Slug {
		t.Fatalf("embedded brand = %s", detail.body)
	}

	updated := requestJSON(t, api, http.MethodPut, "/api/v1/products/"+createdBody.Data.ID, map[string]any{
		"name": "Renamed Authoring Reel", "slug": "renamed-authoring-reel", "status": "draft",
		"brand_id": brandBody.Data.ID, "version": 1,
	})
	if updated.status != http.StatusOK {
		t.Fatalf("update product status = %d, body = %s", updated.status, updated.body)
	}
	var updatedBody struct {
		Data struct {
			ID      string `json:"id"`
			Slug    string `json:"slug"`
			Version int64  `json:"version"`
		} `json:"data"`
	}
	decodeJSON(t, updated.body, &updatedBody)
	if updatedBody.Data.ID != createdBody.Data.ID || updatedBody.Data.Slug != "renamed-authoring-reel" || updatedBody.Data.Version != 2 {
		t.Fatalf("updated product = %s", updated.body)
	}

	stale := requestJSON(t, api, http.MethodPut, "/api/v1/products/"+createdBody.Data.ID, map[string]any{
		"name": "Stale Authoring Reel", "status": "draft", "brand_id": brandBody.Data.ID, "version": 1,
	})
	assertProblem(t, stale, http.StatusConflict, "product_version_conflict")

	missing := requestJSON(t, api, http.MethodPut, "/api/v1/products/00000000-0000-0000-0000-000000000000", map[string]any{
		"name": "Missing", "status": "draft", "brand_id": brandBody.Data.ID, "version": 1,
	})
	assertProblem(t, missing, http.StatusNotFound, "product_not_found")

	deleted := requestJSON(t, api, http.MethodDelete, "/api/v1/products/"+createdBody.Data.ID, map[string]any{"version": 2})
	if deleted.status != http.StatusNoContent {
		t.Fatalf("delete product status = %d, body = %s", deleted.status, deleted.body)
	}
	gone := requestJSON(t, api, http.MethodGet, "/api/v1/products/"+createdBody.Data.ID, nil)
	assertProblem(t, gone, http.StatusNotFound, "product_not_found")
}

func TestProductCreationAndBrandDeletionCannotBypassDraftBrandRule(t *testing.T) {
	db := openIntegrationDatabase(t)
	defer db.Close()
	if err := migrations.Reset(db); err != nil {
		t.Fatalf("reset migrations: %v", err)
	}

	api := (&application{db: db, config: &config.Config{HttpServer: &config.HttpserverConfig{}}}).mount()
	createdBrand := requestJSON(t, api, http.MethodPost, "/api/v1/admin/brands", map[string]any{"name": "Race Brand"})
	if createdBrand.status != http.StatusCreated {
		t.Fatalf("create race brand status = %d, body = %s", createdBrand.status, createdBrand.body)
	}
	var brandBody struct {
		Data struct {
			ID      string `json:"id"`
			Version int64  `json:"version"`
		} `json:"data"`
	}
	decodeJSON(t, createdBrand.body, &brandBody)

	results := make(chan httpResult, 2)
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		results <- requestJSON(t, api, http.MethodPost, "/api/v1/products", map[string]any{
			"name": "Racing Product", "status": "draft", "brand_id": brandBody.Data.ID,
		})
	}()
	go func() {
		defer wg.Done()
		results <- requestJSON(t, api, http.MethodDelete, "/api/v1/admin/brands/"+brandBody.Data.ID, map[string]any{"version": brandBody.Data.Version})
	}()
	wg.Wait()
	close(results)

	var productCreated, brandDeleted int
	for result := range results {
		switch result.status {
		case http.StatusCreated:
			productCreated++
		case http.StatusNoContent:
			brandDeleted++
		case http.StatusConflict, http.StatusNotFound:
		default:
			t.Errorf("unexpected race result = %#v", result)
		}
	}
	if productCreated != 0 || brandDeleted != 1 {
		t.Fatalf("race results: product_created=%d brand_deleted=%d", productCreated, brandDeleted)
	}

	gone := requestJSON(t, api, http.MethodGet, "/api/v1/admin/brands/"+brandBody.Data.ID, nil)
	assertProblem(t, gone, http.StatusNotFound, "brand_not_found")
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

func requestMultipart(t *testing.T, handler http.Handler, path string, content []byte) httpResult {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("logo", "logo.png")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write(content); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodPost, path, &body)
	req.Header.Set("Content-Type", writer.FormDataContentType())
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
