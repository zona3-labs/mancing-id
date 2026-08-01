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
			Version int64 `json:"version"`
		} `json:"data"`
	}
	decodeJSON(t, readAfterUpdate.body, &current)
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
