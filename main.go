package main

import (
	"context"
	"log"
	"os"
	"savetherecipe/internal/handler"
	"savetherecipe/internal/scraper"

	"github.com/getsentry/sentry-go"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"

	"savetherecipe/internal/api"
	"savetherecipe/internal/bot"
	"savetherecipe/internal/instagram"
	"savetherecipe/internal/processor"
	"savetherecipe/internal/publisher"
)

func main() {
	_ = godotenv.Load()
	log.Printf("Loaded env.\n")

	defer initSentry()()
	log.Printf("Loaded sentry.\n")

	ctx := context.Background()

	igClient := instagram.NewClient(os.Getenv("INSTAGRAM_GRPC_ADDR"))
	log.Printf("Instagram client initialized.\n")

	scraperClient := scraper.New()
	log.Printf("Scraper initialized.\n")

	proc := processor.New(igClient, scraperClient)
	log.Printf("Processor initialized.\n")

	minioClient, minioBucket, err := initMinio(ctx)
	if err != nil {
		log.Printf("minio: %v", err)
	}
	log.Printf("Minio client initialized.\n")

	rabbit, err := initRabbitMQ()
	if err != nil {
		log.Printf("rabbit mq: %v", err)
	}
	log.Printf("Publisher initialized.\n")

	var rabbitURL, rabbitQueue string
	if rabbit != nil {
		rabbitURL = rabbit.RabbitMQURL
		rabbitQueue = rabbit.Queue
	}
	pub := publisher.New(minioClient, minioBucket, rabbitURL, rabbitQueue)

	h := handler.New(proc, pub)

	if token := os.Getenv("TELEGRAM_TOKEN"); token != "" {
		telegramBotApi, err := tgbotapi.NewBotAPI(token)
		if err != nil {
			sentry.CaptureException(err)
			log.Fatalf("create bot: %v", err)
		}
		log.Printf("authorized as @%s", telegramBotApi.Self.UserName)

		go bot.New(telegramBotApi, h).Run()

	} else {
		log.Println("TELEGRAM_TOKEN not set, running HTTP-only mode")
	}

	api.New(h).ListenAndServe()
}
