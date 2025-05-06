package main

import (
	"aws-ses-sender-go/api"
	"aws-ses-sender-go/cmd/scheduler"
	"aws-ses-sender-go/cmd/sender"
	"aws-ses-sender-go/config"
	"github.com/getsentry/sentry-go"
)

func main() {
	// Sentry
	_ = sentry.Init(sentry.ClientOptions{
		Dsn: config.GetEnv("SENTRY_DSN"),
	})

	// Message Scheduler
	go scheduler.Run()

	// Email Consumer
	go sender.ConsumeSend()
	go sender.ConsumePostSend()

	// HTTP Server
	api.Run()
}
