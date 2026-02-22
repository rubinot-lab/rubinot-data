package api

import (
	"os"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	defaultAPIRelease = "v0.2.0"
	defaultAPICommit  = "unknown"
	apiVersion        = 1
)

type apiVersionInfo struct {
	Version int    `json:"version"`
	Release string `json:"release"`
	Commit  string `json:"commit"`
}

type status struct {
	HTTPCode int    `json:"http_code"`
	Error    int    `json:"error,omitempty"`
	Message  string `json:"message,omitempty"`
}

type information struct {
	API       apiVersionInfo `json:"api"`
	Timestamp string         `json:"timestamp"`
	Status    status         `json:"status"`
	Sources   []string       `json:"sources"`
}

func successEnvelope(payloadKey string, payload any, sources []string) gin.H {
	return gin.H{
		"information": buildInformation(200, 0, "ok", sources),
		payloadKey:    payload,
	}
}

func errorEnvelope(httpCode, errorCode int, message string, sources []string) gin.H {
	return gin.H{
		"information": buildInformation(httpCode, errorCode, message, sources),
	}
}

func buildInformation(httpCode, errorCode int, message string, sources []string) information {
	return information{
		API: apiVersionInfo{
			Version: apiVersion,
			Release: getEnv("APP_VERSION", defaultAPIRelease),
			Commit:  getEnv("APP_COMMIT", defaultAPICommit),
		},
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Status: status{
			HTTPCode: httpCode,
			Error:    errorCode,
			Message:  message,
		},
		Sources: normalizeSources(sources),
	}
}

func normalizeSources(sources []string) []string {
	if len(sources) == 0 {
		return []string{}
	}
	return sources
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
