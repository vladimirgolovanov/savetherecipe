package handler

import (
	"context"
	"savetherecipe/internal/instagram"
	"savetherecipe/internal/models"
	"savetherecipe/internal/processor"
	"savetherecipe/internal/publisher"
	"strings"

	"github.com/getsentry/sentry-go"
)

type Handler struct {
	proc *processor.Processor
	pub  *publisher.Publisher
}

func New(proc *processor.Processor, pub *publisher.Publisher) *Handler {
	return &Handler{proc: proc, pub: pub}
}

func (h *Handler) Handle(ctx context.Context, text string) (*models.Result, error) {
	messageType := classifyMessage(text)
	result, err := h.proc.Process(ctx, text, messageType)
	if err != nil {
		return nil, err
	}

	// todo: telegram user id
	err = h.pub.Publish(ctx, result.SourceURL, result.Caption, result.ImageData, nil)
	if err != nil {
		sentry.CaptureException(err)
	}

	return &models.Result{
		Caption:     result.Caption,
		ImageData:   result.ImageData,
		SourceURL:   result.SourceURL,
		MessageType: result.MessageType,
	}, nil
}

func classifyMessage(text string) models.MessageType {
	text = strings.TrimSpace(text)
	if _, ok := instagram.NormalizeURL(text); ok {
		return models.MessageTypeInstagramURL
	}
	if strings.HasPrefix(text, "http://") || strings.HasPrefix(text, "https://") {
		return models.MessageTypeOtherURL
	}
	return models.MessageTypeText
}
