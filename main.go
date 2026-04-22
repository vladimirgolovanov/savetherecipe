package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/getsentry/sentry-go"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"

	"savetherecipe/internal/api"
	"savetherecipe/internal/bot"
	"savetherecipe/internal/instagram"
)

func main() {
	_ = godotenv.Load()

	if dsn := os.Getenv("SENTRY_DSN"); dsn != "" {
		if err := sentry.Init(sentry.ClientOptions{
			Dsn:              dsn,
			TracesSampleRate: 1.0,
		}); err != nil {
			log.Printf("sentry init: %v", err)
		}
		defer sentry.Flush(2 * time.Second)
	}

	igClient := instagram.NewClient(os.Getenv("INSTAGRAM_GRPC_ADDR"))
	mux := http.NewServeMux()
	api.NewHandler(igClient).Register(mux)

	httpPort := os.Getenv("HTTP_PORT")
	if httpPort == "" {
		httpPort = "8080"
	}
	httpAddr := ":" + httpPort

	if token := os.Getenv("TELEGRAM_TOKEN"); token != "" {
		tgBot, err := tgbotapi.NewBotAPI(token)
		if err != nil {
			sentry.CaptureException(err)
			log.Fatalf("create bot: %v", err)
		}
		log.Printf("authorized as @%s", tgBot.Self.UserName)
		b := bot.New(tgBot, os.Getenv("INSTAGRAM_GRPC_ADDR"))
		go b.Run()
	} else {
		log.Println("TELEGRAM_TOKEN not set, running HTTP-only mode")
	}

	log.Printf("HTTP server listening on %s", httpAddr)
	if err := http.ListenAndServe(httpAddr, mux); err != nil {
		log.Fatalf("http server: %v", err)
	}
}
