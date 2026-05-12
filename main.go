package main

import (
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/getsentry/sentry-go"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/joho/godotenv"

	"savetherecipe/internal/api"
	"savetherecipe/internal/bot"
	"savetherecipe/internal/instagram"
	"savetherecipe/internal/publisher"
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

	var pub *publisher.Publisher
	if minioEndpoint := os.Getenv("MINIO_ENDPOINT"); minioEndpoint != "" &&
		os.Getenv("MINIO_ACCESS_KEY") != "" &&
		os.Getenv("MINIO_SECRET_KEY") != "" &&
		os.Getenv("MINIO_BUCKET") != "" &&
		os.Getenv("RABBITMQ_URL") != "" &&
		os.Getenv("RABBITMQ_QUEUE") != "" {

		useSSL, _ := strconv.ParseBool(os.Getenv("MINIO_USE_SSL"))
		var err error
		pub, err = publisher.New(publisher.Config{
			MinioEndpoint:  minioEndpoint,
			MinioAccessKey: os.Getenv("MINIO_ACCESS_KEY"),
			MinioSecretKey: os.Getenv("MINIO_SECRET_KEY"),
			MinioBucket:    os.Getenv("MINIO_BUCKET"),
			MinioUseSSL:    useSSL,
			RabbitMQURL:    os.Getenv("RABBITMQ_URL"),
			RabbitMQQueue:  os.Getenv("RABBITMQ_QUEUE"),
		})
		if err != nil {
			log.Printf("publisher init: %v", err)
		} else {
			defer pub.Close()
			log.Println("publisher ready (MinIO + RabbitMQ)")
		}
	}

	if token := os.Getenv("TELEGRAM_TOKEN"); token != "" {
		tgBot, err := tgbotapi.NewBotAPI(token)
		if err != nil {
			sentry.CaptureException(err)
			log.Fatalf("create bot: %v", err)
		}
		log.Printf("authorized as @%s", tgBot.Self.UserName)
		b := bot.New(tgBot, os.Getenv("INSTAGRAM_GRPC_ADDR"), pub)
		go b.Run()
	} else {
		log.Println("TELEGRAM_TOKEN not set, running HTTP-only mode")
	}

	log.Printf("HTTP server listening on %s", httpAddr)
	if err := http.ListenAndServe(httpAddr, mux); err != nil {
		log.Fatalf("http server: %v", err)
	}
}
