package processor

import (
	"context"
	"fmt"
	"savetherecipe/internal/models"
	"savetherecipe/internal/scraper"
	"strings"

	"savetherecipe/internal/instagram"
)

type Processor struct {
	ig      *instagram.Client
	scraper *scraper.Scraper
}

func New(ig *instagram.Client, scraper *scraper.Scraper) *Processor {
	return &Processor{ig: ig, scraper: scraper}
}

func (p *Processor) Process(ctx context.Context, text string, messageType models.MessageType) (*models.Result, error) {
	text = strings.TrimSpace(text)

	switch messageType {
	case models.MessageTypeInstagramURL:
		return p.processInstagram(ctx, text)
	case models.MessageTypeOtherURL:
		return p.processURL(ctx, text)
	default:
		return &models.Result{MessageType: messageType, Caption: text}, nil
	}
}

func (p *Processor) processInstagram(ctx context.Context, text string) (*models.Result, error) {
	canonicalURL, _ := instagram.NormalizeURL(text)

	post, err := p.ig.Fetch(canonicalURL)
	if err != nil {
		return nil, fmt.Errorf("get post data: %w", err)
	}

	caption := instagram.CleanCaption(post.Caption)

	return &models.Result{
		MessageType: models.MessageTypeInstagramURL,
		Caption:     caption,
		ImageData:   post.ImageData,
		SourceURL:   text,
	}, nil
}

func (p *Processor) processURL(ctx context.Context, text string) (*models.Result, error) {
	result, err := p.scraper.Fetch(ctx, text)
	if err != nil {
		return nil, err
	}
	if result == nil {
		return &models.Result{MessageType: models.MessageTypeOtherURL}, nil
	}

	return &models.Result{
		MessageType: models.MessageTypeOtherURL,
		Caption:     result.Caption,
		SourceURL:   text,
	}, nil
}
