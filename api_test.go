package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"
)

// newTestServer returns a handler backed by a fresh temp-file database.
func newTestServer(t *testing.T) (http.Handler, *App) {
	t.Helper()
	db, err := openDB(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	app := &App{db: db, baseURL: "http://127.0.0.1:8080"}
	return app.routes(), app
}

func do(t *testing.T, h http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	req.Host = "localhost:8080" // the app only serves local hosts
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func decodeJob(t *testing.T, rec *httptest.ResponseRecorder) Job {
	t.Helper()
	var j Job
	if err := json.Unmarshal(rec.Body.Bytes(), &j); err != nil {
		t.Fatalf("bad job JSON %q: %v", rec.Body.String(), err)
	}
	return j
}

func TestCreateJob(t *testing.T) {
	h, _ := newTestServer(t)

	tests := []struct {
		name string
		body string
		want int
	}{
		{"valid", `{"company":"Acme","title":"Dev"}`, http.StatusCreated},
		{"missing title", `{"company":"Acme"}`, http.StatusBadRequest},
		{"missing company", `{"title":"Dev"}`, http.StatusBadRequest},
		{"invalid json", `{nope`, http.StatusBadRequest},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if rec := do(t, h, "POST", "/api/jobs", tt.body); rec.Code != tt.want {
				t.Fatalf("status = %d, want %d (%s)", rec.Code, tt.want, rec.Body)
			}
		})
	}

	j := decodeJob(t, do(t, h, "POST", "/api/jobs", `{"company":"Acme","title":"Dev"}`))
	if j.ID == 0 || j.Status != "wishlist" || j.CreatedAt == "" {
		t.Fatalf("expected id and default status wishlist, got %+v", j)
	}
}

func TestListJobs(t *testing.T) {
	h, app := newTestServer(t)

	rec := do(t, h, "GET", "/api/jobs", "")
	if rec.Code != http.StatusOK || rec.Body.String() != "[]\n" {
		t.Fatalf("empty list should be [], got %d %q", rec.Code, rec.Body)
	}

	// Insert with explicit timestamps so the ordering check is deterministic.
	for i, c := range []string{"Old", "New"} {
		_, err := app.db.Exec(`INSERT INTO jobs (company, title, created_at) VALUES (?, 'T', ?)`,
			c, []string{"2026-01-01 00:00:00", "2026-02-01 00:00:00"}[i])
		if err != nil {
			t.Fatal(err)
		}
	}
	var jobs []Job
	if err := json.Unmarshal(do(t, h, "GET", "/api/jobs", "").Body.Bytes(), &jobs); err != nil {
		t.Fatal(err)
	}
	if len(jobs) != 2 || jobs[0].Company != "New" {
		t.Fatalf("expected newest first, got %+v", jobs)
	}
}

func TestGetJob(t *testing.T) {
	h, _ := newTestServer(t)
	created := decodeJob(t, do(t, h, "POST", "/api/jobs", `{"company":"Acme","title":"Dev"}`))

	if rec := do(t, h, "GET", "/api/jobs/1", ""); rec.Code != http.StatusOK || decodeJob(t, rec).ID != created.ID {
		t.Fatalf("get existing failed: %d %s", rec.Code, rec.Body)
	}
	if rec := do(t, h, "GET", "/api/jobs/999", ""); rec.Code != http.StatusNotFound {
		t.Fatalf("missing job = %d, want 404", rec.Code)
	}
	if rec := do(t, h, "GET", "/api/jobs/abc", ""); rec.Code != http.StatusBadRequest {
		t.Fatalf("bad id = %d, want 400", rec.Code)
	}
}

func TestUpdateJob(t *testing.T) {
	h, _ := newTestServer(t)
	created := decodeJob(t, do(t, h, "POST", "/api/jobs", `{"company":"Acme","title":"Dev","notes":"keep?"}`))

	// PUT is a full replace: omitted fields are cleared, empty status falls back to wishlist.
	rec := do(t, h, "PUT", "/api/jobs/1", `{"company":"Beta","title":"Lead","status":"applied"}`)
	if rec.Code != http.StatusOK {
		t.Fatalf("update = %d %s", rec.Code, rec.Body)
	}
	got := decodeJob(t, rec)
	if got.ID != created.ID || got.Company != "Beta" || got.Status != "applied" || got.Notes != "" || got.CreatedAt == "" {
		t.Fatalf("unexpected result %+v", got)
	}

	if got := decodeJob(t, do(t, h, "PUT", "/api/jobs/1", `{"company":"Beta","title":"Lead"}`)); got.Status != "wishlist" {
		t.Fatalf("status default = %q, want wishlist", got.Status)
	}
	if rec := do(t, h, "PUT", "/api/jobs/1", `{"company":"","title":"x"}`); rec.Code != http.StatusBadRequest {
		t.Fatalf("empty company = %d, want 400", rec.Code)
	}
	if rec := do(t, h, "PUT", "/api/jobs/999", `{"company":"A","title":"B"}`); rec.Code != http.StatusNotFound {
		t.Fatalf("missing job = %d, want 404", rec.Code)
	}
}

func TestDeleteJob(t *testing.T) {
	h, _ := newTestServer(t)
	do(t, h, "POST", "/api/jobs", `{"company":"Acme","title":"Dev"}`)

	if rec := do(t, h, "DELETE", "/api/jobs/1", ""); rec.Code != http.StatusNoContent {
		t.Fatalf("delete = %d, want 204", rec.Code)
	}
	if rec := do(t, h, "GET", "/api/jobs/1", ""); rec.Code != http.StatusNotFound {
		t.Fatalf("after delete get = %d, want 404", rec.Code)
	}
	if rec := do(t, h, "DELETE", "/api/jobs/1", ""); rec.Code != http.StatusNotFound {
		t.Fatalf("second delete = %d, want 404", rec.Code)
	}
}

func TestLocalOnly(t *testing.T) {
	h, _ := newTestServer(t)
	tests := []struct {
		name, host, origin string
		want               int
	}{
		{"localhost", "localhost:8080", "", http.StatusOK},
		{"loopback ip", "127.0.0.1:8080", "", http.StatusOK},
		{"local origin", "localhost:5173", "http://localhost:5173", http.StatusOK},
		{"foreign host (DNS rebinding)", "evil.example.com", "", http.StatusForbidden},
		{"foreign origin", "localhost:8080", "https://evil.example.com", http.StatusForbidden},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/api/jobs", nil)
			req.Host = tt.host
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, req)
			if rec.Code != tt.want {
				t.Fatalf("status = %d, want %d", rec.Code, tt.want)
			}
		})
	}
}

func TestJobDescription(t *testing.T) {
	h, _ := newTestServer(t)
	created := decodeJob(t, do(t, h, "POST", "/api/jobs", `{"company":"Acme","title":"Dev","description":"Build Go services"}`))
	if created.Description != "Build Go services" {
		t.Fatalf("description = %q", created.Description)
	}
	got := decodeJob(t, do(t, h, "PUT", "/api/jobs/1", `{"company":"Acme","title":"Dev","description":"Updated"}`))
	if got.Description != "Updated" {
		t.Fatalf("description after update = %q", got.Description)
	}
}

// A database created before the description column existed must keep its rows after upgrade.
func TestMigrateAddsDescription(t *testing.T) {
	path := filepath.Join(t.TempDir(), "old.db")
	old, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	_, err = old.Exec(`CREATE TABLE jobs (
	  id INTEGER PRIMARY KEY, company TEXT NOT NULL, title TEXT NOT NULL,
	  url TEXT NOT NULL DEFAULT '', status TEXT NOT NULL DEFAULT 'wishlist',
	  applied_on TEXT NOT NULL DEFAULT '', notes TEXT NOT NULL DEFAULT '',
	  created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP);
	  INSERT INTO jobs (company, title) VALUES ('Acme', 'Dev');`)
	if err != nil {
		t.Fatal(err)
	}
	old.Close()

	for i := 0; i < 2; i++ { // opening twice also proves the migration is idempotent
		db, err := openDB(path)
		if err != nil {
			t.Fatalf("open #%d: %v", i, err)
		}
		jobs, err := listJobs(db)
		if err != nil || len(jobs) != 1 || jobs[0].Company != "Acme" || jobs[0].Description != "" {
			t.Fatalf("open #%d: jobs=%+v err=%v", i, jobs, err)
		}
		db.Close()
	}
}
