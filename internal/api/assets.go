package api

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

func normalizeCreatureName(name string) string {
	normalized := strings.TrimSpace(name)
	normalized = strings.ToLower(normalized)
	normalized = strings.NewReplacer(
		" ", "_",
		"(", "",
		")", "",
		"'", "",
	).Replace(normalized)
	return normalized
}

func toTitleCase(normalized string) string {
	parts := strings.Split(normalized, "_")
	for i, p := range parts {
		if len(p) > 0 {
			parts[i] = strings.ToUpper(p[:1]) + p[1:]
		}
	}
	return strings.Join(parts, "_")
}

var creatureProxyClient = &http.Client{Timeout: 10 * time.Second}

func handleCreatureAsset(assetsBaseDir string) gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Param("name")
		normalized := normalizeCreatureName(name)

		localPath := filepath.Join(assetsBaseDir, "creatures", normalized+".gif")
		if data, err := os.ReadFile(localPath); err == nil {
			c.Header("Cache-Control", "public, max-age=86400")
			c.Data(http.StatusOK, "image/gif", data)
			return
		}

		titleCased := toTitleCase(normalized)
		upstreamURL := fmt.Sprintf("https://tibia.fandom.com/wiki/Special:Filepath/%s.gif", titleCased)
		resp, fetchErr := creatureProxyClient.Get(upstreamURL)
		if fetchErr != nil || resp.StatusCode != http.StatusOK {
			if resp != nil {
				resp.Body.Close()
			}
			c.String(http.StatusNotFound, "creature not found")
			return
		}
		defer resp.Body.Close()

		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			c.String(http.StatusInternalServerError, "failed to read upstream")
			return
		}

		os.MkdirAll(filepath.Dir(localPath), 0755)
		os.WriteFile(localPath, body, 0644)

		c.Header("Cache-Control", "public, max-age=86400")
		c.Data(http.StatusOK, "image/gif", body)
	}
}

func handleStaticAsset(assetsBaseDir, subDir, contentType, extension string) gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Param("name")
		if name == "" {
			name = c.Param("type")
		}
		if name == "" {
			c.String(http.StatusBadRequest, "name required")
			return
		}

		name = filepath.Base(name)

		localPath := filepath.Join(assetsBaseDir, subDir, name+extension)
		data, err := os.ReadFile(localPath)
		if err != nil {
			c.String(http.StatusNotFound, "not found")
			return
		}

		c.Header("Cache-Control", "public, max-age=86400")
		c.Data(http.StatusOK, contentType, data)
	}
}

var itemProxyClient = &http.Client{Timeout: 10 * time.Second}

func handleItemAsset(assetsBaseDir string, upstreamStaticURL string) gin.HandlerFunc {
	return func(c *gin.Context) {
		itemIDStr := c.Param("itemId")
		itemID, err := strconv.Atoi(itemIDStr)
		if err != nil || itemID <= 0 {
			c.String(http.StatusBadRequest, "invalid item ID")
			return
		}

		filename := fmt.Sprintf("%d.gif", itemID)
		localPath := filepath.Join(assetsBaseDir, "items", filename)

		if data, readErr := os.ReadFile(localPath); readErr == nil {
			c.Header("Cache-Control", "public, max-age=31536000, immutable")
			c.Data(http.StatusOK, "image/gif", data)
			return
		}

		if upstreamStaticURL == "" {
			c.String(http.StatusNotFound, "item not found")
			return
		}

		upstreamURL := fmt.Sprintf("%s/objects/%d.gif", strings.TrimRight(upstreamStaticURL, "/"), itemID)
		resp, fetchErr := itemProxyClient.Get(upstreamURL)
		if fetchErr != nil || resp.StatusCode != http.StatusOK {
			if resp != nil {
				resp.Body.Close()
			}
			c.String(http.StatusNotFound, "item not found")
			return
		}
		defer resp.Body.Close()

		body, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			c.String(http.StatusInternalServerError, "failed to read upstream")
			return
		}

		os.MkdirAll(filepath.Dir(localPath), 0755)
		os.WriteFile(localPath, body, 0644)

		c.Header("Cache-Control", "public, max-age=31536000, immutable")
		c.Data(http.StatusOK, "image/gif", body)
	}
}
