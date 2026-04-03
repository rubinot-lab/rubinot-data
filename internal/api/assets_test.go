package api

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
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
