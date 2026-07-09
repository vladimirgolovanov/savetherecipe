package scraper

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/getsentry/sentry-go"
)

type Result struct {
	Caption   string
	ImageURLs []string
}

type Scraper struct {
	http *http.Client
}

func New() *Scraper {
	return &Scraper{
		http: &http.Client{Timeout: 10 * time.Second},
	}
}

func (s *Scraper) Fetch(ctx context.Context, url string) (*Result, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; savetherecipe/1.0)")

	resp, err := s.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http get: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		err := fmt.Errorf("http %d for %s", resp.StatusCode, url)
		sentry.CaptureException(err)
		return nil, err // todo: fallback for browser scraper
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	bodyReader := bytes.NewReader(body)
	result := &Result{}
	switch {
	case strings.Contains(string(body), `application/ld+json`):
		result, err = parseJSONLD(bodyReader)
	case strings.Contains(string(body), `og:`):
		result, err = parseOG(bodyReader)
	default:
		result, err = nil, nil // todo: parseHTML
		return result, err
	}

	if result == nil {
		result = &Result{}
	}

	bodyReader.Seek(0, io.SeekStart)
	recipeText, extractErr := extractRecipe(bodyReader, url)
	result.Caption += "\n" + recipeText
	if extractErr != nil {
		sentry.CaptureException(extractErr)
	}
	if err != nil {
		return result, err
	}

	return result, extractErr
}
