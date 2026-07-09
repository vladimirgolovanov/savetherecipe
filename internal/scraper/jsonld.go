package scraper

import (
	"encoding/json"
	"io"
	"strings"

	"golang.org/x/net/html"
)

func parseJSONLD(r io.Reader) (*Result, error) {
	doc, err := html.Parse(r)
	if err != nil {
		return nil, err
	}

	result := &Result{}
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && n.Data == "script" &&
			attrVal(n, "type") == "application/ld+json" && n.FirstChild != nil {
			applyJSONLD(n.FirstChild.Data, result)
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
	}
	walk(doc)
	return result, nil
}

type jsonLDItem struct {
	Type        string          `json:"@type"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Image       json.RawMessage `json:"image"`
}

func applyJSONLD(data string, result *Result) {
	var items []jsonLDItem
	if json.Unmarshal([]byte(data), &items) != nil {
		var item jsonLDItem
		if json.Unmarshal([]byte(data), &item) != nil {
			return
		}
		items = []jsonLDItem{item}
	}

	for _, item := range items {
		if strings.EqualFold(item.Type, "recipe") {
			result.Caption = item.Name
			result.ImageURLs = extractImageURLs(item.Image)
			return
		}
	}
}

func extractImageURLs(raw json.RawMessage) []string {
	if len(raw) == 0 {
		return nil
	}
	var s string
	if json.Unmarshal(raw, &s) == nil {
		return []string{s}
	}
	var arr []json.RawMessage
	if json.Unmarshal(raw, &arr) == nil {
		var urls []string
		for _, item := range arr {
			if json.Unmarshal(item, &s) == nil {
				urls = append(urls, s)
				continue
			}
			var obj struct {
				URL string `json:"url"`
			}
			if json.Unmarshal(item, &obj) == nil && obj.URL != "" {
				urls = append(urls, obj.URL)
			}
		}
		return urls
	}
	var obj struct {
		URL string `json:"url"`
	}
	if json.Unmarshal(raw, &obj) == nil && obj.URL != "" {
		return []string{obj.URL}
	}
	return nil
}
