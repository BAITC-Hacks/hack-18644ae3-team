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

	"careerquest/internal/career"
	"careerquest/internal/dataset"
	"careerquest/internal/httpapi"
	"careerquest/internal/recommendation"
)

func main() {
	dataDir := flag.String("data", "case_1/career_quest_dataset", "path to the Career Quest dataset")
	webDir := flag.String("web", "web", "path to frontend assets")
	address := flag.String("addr", ":8081", "HTTP listen address")
	flag.Parse()

	store, err := dataset.Load(*dataDir)
	if err != nil {
		log.Fatalf("load dataset: %v", err)
	}
	careerService := career.New(store)
	recommendationService := recommendation.New(store, careerService)

	server := &http.Server{
		Addr:              *address,
		Handler:           httpapi.NewWithFrontend(store, careerService, recommendationService, *webDir),
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
