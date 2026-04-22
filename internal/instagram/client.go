package instagram

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

var shortcodeRe = regexp.MustCompile(`instagram\.com/(?:p|reel|tv)/([A-Za-z0-9_-]+)`)

// ExtractShortcode pulls the shortcode from any Instagram post/reel URL.
func ExtractShortcode(url string) (string, bool) {
	m := shortcodeRe.FindStringSubmatch(url)
	if len(m) < 2 {
		return "", false
	}
	return m[1], true
}

type PostData struct {
	Caption  string `json:"caption"`
	ImageURL string `json:"image_url"`
}

type Client struct {
	serviceURL string
	http       *http.Client
}

func NewClient(serviceURL string) *Client {
	return &Client{
		serviceURL: serviceURL,
		http:       &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *Client) Fetch(shortcode string) (*PostData, error) {
	url := fmt.Sprintf("%s?shortcode=%s", strings.TrimRight(c.serviceURL, "/"), shortcode)
	resp, err := c.http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("service returned %d", resp.StatusCode)
	}

	var data PostData
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("decode: %w", err)
	}
	return &data, nil
}

func (c *Client) DownloadImage(imageURL string) ([]byte, error) {
	resp, err := c.http.Get(imageURL)
	if err != nil {
		return nil, fmt.Errorf("download: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("image download returned %d", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read image: %w", err)
	}
	return data, nil
}

var hashtagRe = regexp.MustCompile(`#\S+`)

// CleanCaption removes hashtags and trims whitespace.
func CleanCaption(text string) string {
	text = hashtagRe.ReplaceAllString(text, "")
	return strings.TrimSpace(text)
}
