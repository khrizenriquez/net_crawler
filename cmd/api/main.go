package main

import (
	"log"
	"net"
	"net/http"
	"os"
	"time"

	"github.com/duku/net-lab/internal/api"
	"github.com/duku/net-lab/internal/store"
)

func main() {
	host, port := env("DUKU_API_HOST", "127.0.0.1"), env("DUKU_API_PORT", "8080")
	if host != "127.0.0.1" && host != "localhost" && !(host == "0.0.0.0" && os.Getenv("DUKU_ALLOW_CONTAINER_BIND") == "true") {
		log.Fatalf("refusing to bind API to non-loopback host %q", host)
	}
	if _, err := api.ParsePort(port); err != nil {
		log.Fatal(err)
	}
	memory := store.NewMemory()
	var repository store.Repository = memory
	if databaseURL := os.Getenv("DATABASE_URL"); databaseURL != "" {
		postgres, err := store.NewPostgres(databaseURL)
		if err != nil {
			log.Fatal(err)
		}
		repository = postgres
	}
	if env("DUKU_DEMO_SEED", "true") == "true" && len(repository.Radios()) == 0 {
		repository.SeedDemo()
	}
	address := net.JoinHostPort(host, port)
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		repository.TickSchedules(time.Now())
		for now := range ticker.C {
			repository.TickSchedules(now)
		}
	}()
	log.Printf("duku-net-lab API listening on http://%s", address)
	log.Fatal(http.ListenAndServe(address, api.New(repository, os.Getenv("DUKU_HOST_TOKEN")).Handler()))
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
