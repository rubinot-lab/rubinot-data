package main

import (
	"context"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/giovannirco/rubinot-data/internal/api"
	"github.com/giovannirco/rubinot-data/internal/observability"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

func main() {
	mode := os.Getenv("GIN_MODE")
	if mode == "" {
		mode = gin.ReleaseMode
	}
	gin.SetMode(mode)

	shutdownTracer, err := observability.InitTracer(context.Background())
	if err != nil {
		log.Printf("otel init failed: %v", err)
	} else {
		defer func() {
			_ = shutdownTracer(context.Background())
		}()
	}

	r, err := api.NewRouter()
	if err != nil {
		log.Fatalf("router init failed: %v", err)
	}
	r.Use(otelgin.Middleware("rubinot-data"))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("starting rubinot-data on :%s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatal(err)
	}
}
