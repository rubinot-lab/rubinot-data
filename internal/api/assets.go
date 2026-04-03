package api

import (
	"net/http"
	"os"
	"path/filepath"
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

func handleCreatureAsset(assetsBaseDir string) gin.HandlerFunc {
	return func(c *gin.Context) {
		name := c.Param("name")
		normalized := normalizeCreatureName(name)

		localPath := filepath.Join(assetsBaseDir, "creatures", normalized+".gif")
		data, err := os.ReadFile(localPath)
		if err != nil {
			c.String(http.StatusNotFound, "creature not found")
			return
		}

		c.Header("Cache-Control", "public, max-age=86400")
		c.Header("Last-Modified", time.Now().UTC().Format(http.TimeFormat))
		c.Data(http.StatusOK, "image/gif", data)
	}
}
