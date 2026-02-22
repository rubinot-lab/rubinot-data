package api

import (
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/giovannirco/rubinot-data/internal/scraper"
)

type status struct {
	HTTPCode int    `json:"http_code"`
	Message  string `json:"message,omitempty"`
}

type information struct {
	Timestamp string   `json:"timestamp"`
	Status    status   `json:"status"`
	Sources   []string `json:"sources,omitempty"`
}

type worldResponse struct {
	Information information         `json:"information"`
	World       scraper.WorldResult `json:"world"`
}

func NewRouter() *gin.Engine {
	r := gin.Default()

	r.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "rubinot-data api up"})
	})
	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "pong"})
	})
	r.GET("/healthz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	r.GET("/readyz", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	r.GET("/versions", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"service": "rubinot-data", "version": getEnv("APP_VERSION", "dev")})
	})

	v1 := r.Group("/v1")
	{
		v1.GET("/world/:name", getWorld)
	}

	return r
}

func getWorld(c *gin.Context) {
	name := c.Param("name")
	baseURL := getEnv("RUBINOT_BASE_URL", "https://www.rubinot.com.br")

	result, sourceURL, err := scraper.FetchWorld(baseURL, name)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{
			"information": information{
				Timestamp: time.Now().UTC().Format(time.RFC3339),
				Status: status{HTTPCode: http.StatusBadGateway, Message: err.Error()},
				Sources: []string{sourceURL},
			},
		})
		return
	}

	c.JSON(http.StatusOK, worldResponse{
		Information: information{
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Status:    status{HTTPCode: http.StatusOK, Message: "ok"},
			Sources:   []string{sourceURL},
		},
		World: result,
	})
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
