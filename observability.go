package main

import (
	"log"
	"os"
	"time"

	"github.com/getsentry/sentry-go"
)

func initSentry() func() {
	dsn := os.Getenv("SENTRY_DSN")
	if dsn == "" {
		return func() {}
	}

	if err := sentry.Init(sentry.ClientOptions{
		Dsn:              dsn,
		TracesSampleRate: 1.0,
	}); err != nil {
		log.Printf("sentry init: %v", err)
	}

	return func() { sentry.Flush(2 * time.Second) }
}
