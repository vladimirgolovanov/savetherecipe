package publisher

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	minio "github.com/minio/minio-go/v7"
)

type message struct {
	Text           string `json:"text"`
	URL            string `json:"url"`
	ImageURL       string `json:"image_url"`
	TelegramUserID *int64 `json:"telegram_user_id"`
}

type Publisher struct {
	minioClient *minio.Client
	minioBucket string
	rabbitMQURL string
	rabbitQueue string
}

func New(minioClient *minio.Client, minioBucket string, rabbitMQURL string, rabbitQueue string) *Publisher {
	return &Publisher{
		minioClient: minioClient,
		minioBucket: minioBucket,
		rabbitMQURL: rabbitMQURL,
		rabbitQueue: rabbitQueue,
	}
}

func (p *Publisher) Publish(ctx context.Context, sourceURL, text string, imageData []byte, telegramUserID *int64) error {
	if p.rabbitMQURL == "" {
		return nil
	}

	conn, err := amqp.Dial(p.rabbitMQURL)
	if err != nil {
		return fmt.Errorf("rabbitmq dial: %w", err)
	}
	defer conn.Close()

	var imageURL string
	if len(imageData) > 0 && p.minioClient != nil {
		objectName := fmt.Sprintf("%d.jpg", time.Now().UnixNano())
		_, err = p.minioClient.PutObject(ctx, p.minioBucket, objectName, bytes.NewReader(imageData), int64(len(imageData)), minio.PutObjectOptions{
			ContentType: "image/jpeg",
		})
		if err != nil {
			return fmt.Errorf("minio put: %w", err)
		}
		imageURL = fmt.Sprintf("%s/%s/%s", p.minioClient.EndpointURL(), p.minioBucket, objectName)
	}

	body, err := json.Marshal(message{Text: text, URL: sourceURL, ImageURL: imageURL, TelegramUserID: telegramUserID})
	if err != nil {
		return fmt.Errorf("json marshal: %w", err)
	}

	ch, err := conn.Channel()
	if err != nil {
		return fmt.Errorf("amqp channel: %w", err)
	}
	defer ch.Close()

	if _, err = ch.QueueDeclare(p.rabbitQueue, true, false, false, false, nil); err != nil {
		return fmt.Errorf("queue declare: %w", err)
	}

	return ch.PublishWithContext(ctx, "", p.rabbitQueue, false, false, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		Body:         body,
	})
}
