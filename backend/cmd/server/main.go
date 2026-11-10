package main

import (
	"log"
	"net/http"
	"os"
	"time"
	"vostrogate/backend/internal/app"
)

func main() {
	a, err := app.New()
	if err != nil {
		log.Fatal(err)
	}
	server := &http.Server{
		Addr:              getenv("HTTP_ADDR", ":8080"),
		Handler:           a.Routes(),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      60 * time.Second,
		IdleTimeout:       120 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}
	log.Printf("Vostrogate API listening on %s", server.Addr)
	log.Fatal(server.ListenAndServe())
}
func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
