package main

import (
	"context"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"careerquest/internal/auth"
	"careerquest/internal/career"
	"careerquest/internal/dataset"
	"careerquest/internal/httpapi"
	"careerquest/internal/postgres"
	"careerquest/internal/recommendation"
)

func main() {
	dataDir := flag.String("data", "case_1/career_quest_dataset", "path to the Career Quest dataset")
	webDir := flag.String("web", "web", "path to frontend assets")
	address := flag.String("addr", ":8081", "HTTP listen address")
	databaseURL := flag.String("database-url", databaseURLFromEnv(), "PostgreSQL connection URL")
	seed := flag.Bool("seed", true, "idempotently seed the supplied dataset")
	flag.Parse()

	startupContext, startupCancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer startupCancel()
	store, err := postgres.Open(startupContext, *databaseURL)
	if err != nil {
		log.Fatalf("open database: %v (start PostgreSQL with 'docker compose up -d db')", err)
	}
	defer store.Close()
	if err := store.Migrate(startupContext); err != nil {
		log.Fatalf("run migrations: %v", err)
	}
	if *seed {
		seedData, err := dataset.Load(*dataDir)
		if err != nil {
			log.Fatalf("load seed dataset: %v", err)
		}
		if err := store.Seed(startupContext, seedData); err != nil {
			log.Fatalf("seed database: %v", err)
		}
	}
	careerService := career.New(store)
	recommendationService := recommendation.New(store, careerService)
	authService := auth.NewService(store)

	server := &http.Server{
		Addr:              *address,
		Handler:           httpapi.NewWithAuth(store, careerService, recommendationService, authService, *webDir),
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Printf("Career Quest listening on %s (dataset snapshot %s)", *address, store.Meta().AsOfDate)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("serve: %v", err)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(ctx); err != nil {
		log.Printf("shutdown: %v", err)
	}
}

func databaseURLFromEnv() string {
	if value := os.Getenv("DATABASE_URL"); value != "" {
		return value
	}
	return "postgres://careerquest:careerquest@localhost:55432/careerquest?sslmode=disable"
}
