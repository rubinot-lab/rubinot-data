package api

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestNormalizeCreatureName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"Ancient Scarab", "ancient_scarab"},
		{"ANCIENT_SCARAB", "ancient_scarab"},
		{"ferumbras", "ferumbras"},
		{"Demon (Goblin)", "demon_goblin"},
		{"Ferumbras' Ascendant", "ferumbras_ascendant"},
		{"  spaced  ", "spaced"},
		{"Dragon Lord", "dragon_lord"},
	}

	for _, tc := range tests {
		t.Run(tc.input, func(t *testing.T) {
			got := normalizeCreatureName(tc.input)
			if got != tc.want {
				t.Fatalf("normalizeCreatureName(%q) = %q, want %q", tc.input, got, tc.want)
			}
		})
	}
}

func TestCreatureAssetHandler(t *testing.T) {
	tmpDir := t.TempDir()
	creaturesDir := filepath.Join(tmpDir, "creatures")
	os.MkdirAll(creaturesDir, 0755)

	minimalGIF := []byte("GIF89a\x01\x00\x01\x00\x80\x00\x00\xff\xff\xff\x00\x00\x00!\xf9\x04\x00\x00\x00\x00\x00,\x00\x00\x00\x00\x01\x00\x01\x00\x00\x02\x02D\x01\x00;")
	os.WriteFile(filepath.Join(creaturesDir, "ancient_scarab.gif"), minimalGIF, 0644)
	os.WriteFile(filepath.Join(creaturesDir, "ferumbras.gif"), minimalGIF, 0644)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/v1/assets/creatures/:name", handleCreatureAsset(tmpDir))

	tests := []struct {
		path       string
		wantStatus int
		wantType   string
	}{
		{"/v1/assets/creatures/ancient_scarab", http.StatusOK, "image/gif"},
		{"/v1/assets/creatures/Ancient%20Scarab", http.StatusOK, "image/gif"},
		{"/v1/assets/creatures/ANCIENT_SCARAB", http.StatusOK, "image/gif"},
		{"/v1/assets/creatures/ferumbras", http.StatusOK, "image/gif"},
		{"/v1/assets/creatures/Ferumbras", http.StatusOK, "image/gif"},
		{"/v1/assets/creatures/nonexistent", http.StatusNotFound, ""},
	}

	for _, tc := range tests {
		t.Run(tc.path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			if rec.Code != tc.wantStatus {
				t.Fatalf("path %q: expected status %d, got %d: %s", tc.path, tc.wantStatus, rec.Code, rec.Body.String())
			}
			if tc.wantType != "" {
				ct := rec.Header().Get("Content-Type")
				if ct != tc.wantType {
					t.Fatalf("path %q: expected content-type %q, got %q", tc.path, tc.wantType, ct)
				}
			}
			if tc.wantStatus == http.StatusOK {
				cacheControl := rec.Header().Get("Cache-Control")
				if cacheControl == "" {
					t.Fatalf("path %q: expected Cache-Control header", tc.path)
				}
			}
		})
	}
}

func TestItemAssetHandler(t *testing.T) {
	tmpDir := t.TempDir()
	itemsDir := filepath.Join(tmpDir, "items")
	os.MkdirAll(itemsDir, 0755)

	minimalGIF := []byte("GIF89a\x01\x00\x01\x00\x80\x00\x00\xff\xff\xff\x00\x00\x00!\xf9\x04\x00\x00\x00\x00\x00,\x00\x00\x00\x00\x01\x00\x01\x00\x00\x02\x02D\x01\x00;")
	os.WriteFile(filepath.Join(itemsDir, "12645.gif"), minimalGIF, 0644)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/v1/assets/items/:itemId", handleItemAsset(tmpDir, ""))

	tests := []struct {
		name       string
		path       string
		wantStatus int
	}{
		{"cached item", "/v1/assets/items/12645", http.StatusOK},
		{"invalid non-numeric ID", "/v1/assets/items/abc", http.StatusBadRequest},
		{"not cached no upstream", "/v1/assets/items/99999", http.StatusNotFound},
		{"negative ID", "/v1/assets/items/-1", http.StatusBadRequest},
		{"zero ID", "/v1/assets/items/0", http.StatusBadRequest},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tc.path, nil)
			rec := httptest.NewRecorder()
			router.ServeHTTP(rec, req)

			if rec.Code != tc.wantStatus {
				t.Fatalf("expected status %d, got %d: %s", tc.wantStatus, rec.Code, rec.Body.String())
			}
			if tc.wantStatus == http.StatusOK {
				ct := rec.Header().Get("Content-Type")
				if ct != "image/gif" {
					t.Fatalf("expected image/gif, got %q", ct)
				}
				cc := rec.Header().Get("Cache-Control")
				if !strings.Contains(cc, "immutable") {
					t.Fatalf("expected immutable cache, got %q", cc)
				}
			}
		})
	}
}

func TestItemAssetHandlerUpstreamProxy(t *testing.T) {
	minimalGIF := []byte("GIF89a\x01\x00\x01\x00\x80\x00\x00\xff\xff\xff\x00\x00\x00!\xf9\x04\x00\x00\x00\x00\x00,\x00\x00\x00\x00\x01\x00\x01\x00\x00\x02\x02D\x01\x00;")

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/objects/55555.gif" {
			w.Header().Set("Content-Type", "image/gif")
			w.Write(minimalGIF)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}))
	defer upstream.Close()

	tmpDir := t.TempDir()
	os.MkdirAll(filepath.Join(tmpDir, "items"), 0755)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/v1/assets/items/:itemId", handleItemAsset(tmpDir, upstream.URL))

	req := httptest.NewRequest(http.MethodGet, "/v1/assets/items/55555", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	cached, err := os.ReadFile(filepath.Join(tmpDir, "items", "55555.gif"))
	if err != nil {
		t.Fatalf("expected cached file, got error: %v", err)
	}
	if len(cached) != len(minimalGIF) {
		t.Fatalf("cached file size mismatch: %d vs %d", len(cached), len(minimalGIF))
	}

	rec2 := httptest.NewRecorder()
	router.ServeHTTP(rec2, httptest.NewRequest(http.MethodGet, "/v1/assets/items/55555", nil))
	if rec2.Code != http.StatusOK {
		t.Fatalf("expected 200 from cache, got %d", rec2.Code)
	}

	rec3 := httptest.NewRecorder()
	router.ServeHTTP(rec3, httptest.NewRequest(http.MethodGet, "/v1/assets/items/99999", nil))
	if rec3.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec3.Code)
	}
}
