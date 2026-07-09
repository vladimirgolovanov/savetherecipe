package main

import (
	"context"
	"fmt"
	"os"
	"strconv"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

func initMinio(ctx context.Context) (client *minio.Client, minioBucket string, err error) {
	if os.Getenv("MINIO_ENDPOINT") == "" ||
		os.Getenv("MINIO_ACCESS_KEY") == "" ||
		os.Getenv("MINIO_SECRET_KEY") == "" ||
		os.Getenv("MINIO_BUCKET") == "" {
		return nil, "", nil
	}

	useSSL, _ := strconv.ParseBool(os.Getenv("MINIO_USE_SSL"))

	mc, err := minio.New(os.Getenv("MINIO_ENDPOINT"), &minio.Options{
		Creds:  credentials.NewStaticV4(os.Getenv("MINIO_ACCESS_KEY"), os.Getenv("MINIO_SECRET_KEY"), ""),
		Secure: useSSL,
	})
	if err != nil {
		return nil, "", fmt.Errorf("minio client: %w", err)
	}

	exists, err := mc.BucketExists(ctx, os.Getenv("MINIO_BUCKET"))
	if err != nil {
		return nil, "", fmt.Errorf("minio bucket exists: %w", err)
	}
	if !exists {
		if err = mc.MakeBucket(ctx, os.Getenv("MINIO_BUCKET"), minio.MakeBucketOptions{}); err != nil {
			return nil, "", fmt.Errorf("minio make bucket: %w", err)
		}
	}

	return mc, os.Getenv("MINIO_BUCKET"), nil
}

type RabbitMQ struct {
	RabbitMQURL string
	Queue       string
}

func initRabbitMQ() (*RabbitMQ, error) {
	if os.Getenv("RABBITMQ_URL") == "" ||
		os.Getenv("RABBITMQ_QUEUE") == "" {
		return nil, nil
	}

	rabbit := RabbitMQ{
		RabbitMQURL: os.Getenv("RABBITMQ_URL"),
		Queue:       os.Getenv("RABBITMQ_QUEUE"),
	}

	return &rabbit, nil
}
