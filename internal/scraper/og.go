package scraper

import (
	"io"

	"golang.org/x/net/html"
)

func parseOG(r io.Reader) (*Result, error) {
	doc, err := html.Parse(r)
	if err != nil {
		return nil, err
	}

	result := &Result{}
	walkHTML(doc, result)
	return result, nil
}

func walkHTML(n *html.Node, result *Result) {
	if n.Type == html.ElementNode {
		switch n.Data {
		case "meta":
			applyOGMeta(n, result)
		}
	}
	for c := n.FirstChild; c != nil; c = c.NextSibling {
		walkHTML(c, result)
	}
}

func applyOGMeta(n *html.Node, result *Result) {
	property := attrVal(n, "property")
	content := attrVal(n, "content")
	switch property {
	case "og:title":
		result.Caption = content
	case "og:image":
		result.ImageURLs = append(result.ImageURLs, content)
	}
}
