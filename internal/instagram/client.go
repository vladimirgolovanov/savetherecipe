package instagram

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	grab_instagram "github.com/vladimirgolovanov/grab-proto/gen/instagram"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

var postRe = regexp.MustCompile(`instagram\.com/(p|reel)/([A-Za-z0-9_-]+)`)

// NormalizeURL validates an Instagram URL and returns a canonical form.
// Returns ("", false) if the URL is not a recognized post or reel link.
func NormalizeURL(raw string) (string, bool) {
	m := postRe.FindStringSubmatch(raw)
	if len(m) < 3 {
		return "", false
	}
	return fmt.Sprintf("https://instagram.com/%s/%s/", m[1], m[2]), true
}

type PostData struct {
	Caption  string
	ImageURL string
}

type Client struct {
	grpc grab_instagram.InstagramClient
	http *http.Client
}

func NewClient(addr string) *Client {
	conn, err := grpc.NewClient(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		panic(fmt.Sprintf("grpc.NewClient: %v", err))
	}
	return &Client{
		grpc: grab_instagram.NewInstagramClient(conn),
		http: &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *Client) Fetch(postURL string) (*PostData, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	stream, err := c.grpc.GetPost(ctx, &grab_instagram.GetPostRequest{
		PostUrls: []string{postURL},
	})
	if err != nil {
		return nil, fmt.Errorf("grpc GetPost: %w", err)
	}

	resp, err := stream.Recv()
	if err != nil {
		if err == io.EOF {
			return nil, fmt.Errorf("service returned empty stream")
		}
		return nil, fmt.Errorf("stream recv: %w", err)
	}

	if resp.GetError() != "" {
		return nil, fmt.Errorf("service error: %s", resp.GetError())
	}

	return &PostData{
		Caption:  resp.GetText(),
		ImageURL: resp.GetImageUrl(),
	}, nil
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
