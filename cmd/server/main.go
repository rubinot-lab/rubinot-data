package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/giovannirco/rubinot-data/internal/api"
)

func main() {
	mode := os.Getenv("GIN_MODE")
	if mode == "" {
		mode = gin.ReleaseMode
	}
	gin.SetMode(mode)

	r := api.NewRouter()
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("starting rubinot-data on :%s", port)
	if err := r.Run(":" + port); err != nil {
		log.Fatal(err)
	}
}
