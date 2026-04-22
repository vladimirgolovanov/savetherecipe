package main

import (
	"log"
	"os"
	"time"

	"github.com/getsentry/sentry-go"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"

	"savetherecipe/internal/bot"
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

	token := os.Getenv("TELEGRAM_TOKEN")
	if token == "" {
		log.Fatal("TELEGRAM_TOKEN is required")
	}

	tgBot, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		sentry.CaptureException(err)
		log.Fatalf("create bot: %v", err)
	}

	log.Printf("authorized as @%s", tgBot.Self.UserName)

	b := bot.New(tgBot, os.Getenv("INSTAGRAM_SERVICE_URL"))
	b.Run()
}
