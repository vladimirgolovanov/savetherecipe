package scraper

import (
	"io"
	"net/url"
	"regexp"
	"strings"

	"github.com/PuerkitoBio/goquery"
	"github.com/go-shiori/go-readability"
)

func extractRecipe(body io.Reader, u string) (string, error) {
	pageURL, _ := url.Parse(u)

	doc, err := goquery.NewDocumentFromReader(body)
	if err != nil {
		return "", err
	}

	if text := extractBySelector(doc, "[itemtype*='Recipe']"); len(text) > 300 {
		return text, nil
	}

	selectors := []string{
		".recipe", ".recipe-card", ".recipe-content",
		".recipe-body", ".recipe-instructions",
		"#recipe", "#recipe-card",
		".wprm-recipe", ".tasty-recipe", // популярные WP-плагины
		".mv-recipe",
	}
	for _, sel := range selectors {
		if text := extractBySelector(doc, sel); len(text) > 300 {
			return text, nil
		}
	}

	htmlNode := doc.Selection.Nodes[0]
	article, err := readability.FromDocument(htmlNode, pageURL)
	if err == nil && len(article.TextContent) > 300 {
		contentDoc, err := goquery.NewDocumentFromReader(strings.NewReader(article.Content))
		if err != nil {
			return "", err
		}
		content := cleanText(contentDoc.Find("body").Text())
		return content, nil
	}

	return extractWithNoiseCleaning(doc), nil
}

func extractBySelector(doc *goquery.Document, selector string) string {
	var parts []string
	re := regexp.MustCompile(`<[^>]+>`)

	doc.Find(selector).Each(func(_ int, s *goquery.Selection) {
		html, err := s.Html()
		if err != nil {
			return
		}
		html = strings.NewReplacer(
			"<p", "\n<p",
			"<li", "\n<li",
			"<br", "\n<br",
			"<div", "\n<div",
			"<h1", "\n<h1",
			"<h2", "\n<h2",
			"<h3", "\n<h3",
		).Replace(html)
		text := strings.TrimSpace(re.ReplaceAllString(html, ""))
		if text != "" {
			parts = append(parts, text)
		}
	})
	return cleanText(strings.Join(parts, "\n"))
}

func extractWithNoiseCleaning(doc *goquery.Document) string {
	noise := []string{
		"header", "footer", "nav", "aside",
		"[class*='sidebar']", "[class*='widget']",
		"[class*='related']", "[class*='comment']",
		"[class*='social']", "[class*='share']",
		"script", "style", "noscript",
	}
	for _, sel := range noise {
		doc.Find(sel).Remove()
	}

	for _, sel := range []string{"article", "main"} {
		if text := extractBySelector(doc, sel); len(text) > 300 {
			return text
		}
	}

	return strings.TrimSpace(doc.Find("body").Text())
}

func cleanText(text string) string {
	re := regexp.MustCompile(`\n{3,}`)
	text = re.ReplaceAllString(text, "\n\n")
	re = regexp.MustCompile(`[ \t]+`)
	text = re.ReplaceAllString(text, " ")
	return strings.TrimSpace(text)
}
