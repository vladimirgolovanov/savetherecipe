package publisher

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"

	minio "github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type Config struct {
	MinioEndpoint  string
	MinioAccessKey string
	MinioSecretKey string
	MinioBucket    string
	MinioUseSSL    bool
	RabbitMQURL    string
	RabbitMQQueue  string
}

type Publisher struct {
	minioClient *minio.Client
	amqpConn    *amqp.Connection
	bucket      string
	queue       string
}

type message struct {
	Text           string `json:"text"`
	URL            string `json:"url"`
	ImageURL       string `json:"image_url"`
	TelegramUserID *int64 `json:"telegram_user_id"`
}

func New(cfg Config) (*Publisher, error) {
	mc, err := minio.New(cfg.MinioEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.MinioAccessKey, cfg.MinioSecretKey, ""),
		Secure: cfg.MinioUseSSL,
	})
	if err != nil {
		return nil, fmt.Errorf("minio client: %w", err)
	}

	ctx := context.Background()
	exists, err := mc.BucketExists(ctx, cfg.MinioBucket)
	if err != nil {
		return nil, fmt.Errorf("minio bucket exists: %w", err)
	}
	if !exists {
		if err = mc.MakeBucket(ctx, cfg.MinioBucket, minio.MakeBucketOptions{}); err != nil {
			return nil, fmt.Errorf("minio make bucket: %w", err)
		}
	}

	conn, err := amqp.Dial(cfg.RabbitMQURL)
	if err != nil {
		return nil, fmt.Errorf("rabbitmq dial: %w", err)
	}

	return &Publisher{
		minioClient: mc,
		amqpConn:    conn,
		bucket:      cfg.MinioBucket,
		queue:       cfg.RabbitMQQueue,
	}, nil
}

func (p *Publisher) Publish(ctx context.Context, sourceURL, text string, imageData []byte, telegramUserID *int64) error {
	objectName := fmt.Sprintf("%d.jpg", time.Now().UnixNano())

	_, err := p.minioClient.PutObject(ctx, p.bucket, objectName, bytes.NewReader(imageData), int64(len(imageData)), minio.PutObjectOptions{
		ContentType: "image/jpeg",
	})
	if err != nil {
		return fmt.Errorf("minio put: %w", err)
	}

	imageURL := fmt.Sprintf("%s/%s/%s", p.minioClient.EndpointURL(), p.bucket, objectName)

	body, err := json.Marshal(message{Text: text, URL: sourceURL, ImageURL: imageURL, TelegramUserID: telegramUserID})
	if err != nil {
		return fmt.Errorf("json marshal: %w", err)
	}

	ch, err := p.amqpConn.Channel()
	if err != nil {
		return fmt.Errorf("amqp channel: %w", err)
	}
	defer ch.Close()

	if _, err = ch.QueueDeclare(p.queue, true, false, false, false, nil); err != nil {
		return fmt.Errorf("queue declare: %w", err)
	}

	return ch.PublishWithContext(ctx, "", p.queue, false, false, amqp.Publishing{
		ContentType:  "application/json",
		DeliveryMode: amqp.Persistent,
		Body:         body,
	})
}

func (p *Publisher) Close() {
	_ = p.amqpConn.Close()
}
