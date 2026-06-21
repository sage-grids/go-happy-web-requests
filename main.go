package main

import (
	"log"
	"net/http"
	"os"

	"github.com/sage-grids/go-happy-web-requests/internal/handler"
	"github.com/sage-grids/go-happy-web-requests/internal/middleware"
)

func main() {
	if os.Getenv("API_TOKEN") == "" && os.Getenv("AUTH_DISABLED") != "true" {
		log.Fatal("refusing to start: API_TOKEN is not set (set AUTH_DISABLED=true to run without authentication)")
	}

	http.HandleFunc("/health", handler.Health)
	http.HandleFunc("/api/v1/fetch", middleware.Auth(handler.Fetch))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Server starting on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
