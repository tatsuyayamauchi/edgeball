package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func newTestMux(t *testing.T) *http.ServeMux {
	t.Helper()
	latency, err := parseLatency("50:20ms")
	if err != nil {
		t.Fatal(err)
	}
	h := newItemsHandler(60, 100, latency)
	h.sleep = func(time.Duration) {} // テストでは実際に待たない
	mux := http.NewServeMux()
	mux.Handle("GET /api/items/{id}", h)
	return mux
}

func TestItemsHandler(t *testing.T) {
	tests := []struct {
		name             string
		path             string
		wantCode         int
		wantCacheControl string
	}{
		{name: "default max-age", path: "/api/items/1", wantCode: 200, wantCacheControl: "public, max-age=60"},
		{name: "max_age override", path: "/api/items/1?max_age=5", wantCode: 200, wantCacheControl: "public, max-age=5"},
		{name: "max_age=0 is no-store", path: "/api/items/1?max_age=0", wantCode: 200, wantCacheControl: "no-store"},
		{name: "invalid max_age", path: "/api/items/1?max_age=-1", wantCode: 400},
		{name: "last item", path: "/api/items/100", wantCode: 200, wantCacheControl: "public, max-age=60"},
		{name: "out of range", path: "/api/items/101", wantCode: 404},
		{name: "zero", path: "/api/items/0", wantCode: 404},
		{name: "not a number", path: "/api/items/abc", wantCode: 404},
	}
	mux := newTestMux(t)
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			mux.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tt.path, nil))

			if rec.Code != tt.wantCode {
				t.Fatalf("code = %d, want %d", rec.Code, tt.wantCode)
			}
			if tt.wantCode != http.StatusOK {
				return
			}
			if got := rec.Header().Get("Cache-Control"); got != tt.wantCacheControl {
				t.Errorf("Cache-Control = %q, want %q", got, tt.wantCacheControl)
			}
			if got := rec.Header().Get("Content-Type"); got != "application/json" {
				t.Errorf("Content-Type = %q", got)
			}
			if rec.Header().Get("X-Origin-Delay") == "" {
				t.Error("X-Origin-Delay is missing")
			}
		})
	}
}

func TestItemsHandlerBody(t *testing.T) {
	rec := httptest.NewRecorder()
	newTestMux(t).ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/api/items/42", nil))

	var got item
	if err := json.NewDecoder(rec.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if got.ID != 42 || got.Name != "item-42" {
		t.Errorf("got %+v", got)
	}
	if time.Since(got.GeneratedAt) > time.Minute {
		t.Errorf("generated_at = %v, want close to now", got.GeneratedAt)
	}
}
