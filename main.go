package main

import (
	"aws-ses-sender-go/api"
	"aws-ses-sender-go/cmd"
	"aws-ses-sender-go/config"
	"aws-ses-sender-go/model"
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/getsentry/sentry-go"
)

func main() {
	if err := sentry.Init(sentry.ClientOptions{
		Dsn:              config.GetEnv("SENTRY_DSN"),
		Environment:      config.GetEnv("ENV", "dev"),
		TracesSampleRate: 1.0,
	}); err != nil {
		log.Printf("Sentry initialization failed: %v", err)
	}
	defer sentry.Flush(2 * time.Second)

	db := config.GetDB()
	if err := model.AutoMigrate(db); err != nil {
		log.Fatalf("Failed to run database migrations: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		log.Printf("Received signal: %v, initiating graceful shutdown...", sig)
		cancel()
	}()

	defer func() {
		if err := config.CloseDB(); err != nil {
			log.Printf("Error closing database connection: %v", err)
		}
	}()

	go cmd.RunScheduler(ctx)
	go cmd.RunSender(ctx)

	api.Run(ctx)
	log.Println("Application shutdown complete")
}
