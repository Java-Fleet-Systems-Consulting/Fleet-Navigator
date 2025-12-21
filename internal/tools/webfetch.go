package tools

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"golang.org/x/net/html"
)

// WebFetchTool fetches and extracts content from URLs
type WebFetchTool struct {
	BaseTool
	client    *http.Client
	userAgent string
}

// WebFetchResult contains the fetched content
type WebFetchResult struct {
	URL         string `json:"url"`
	Title       string `json:"title"`
	Content     string `json:"content"`
	ContentType string `json:"contentType"`
	StatusCode  int    `json:"statusCode"`
	// MetaDescription from meta tags
	MetaDescription string `json:"metaDescription,omitempty"`
	// Links found on the page
	Links []string `json:"links,omitempty"`
}

// NewWebFetchTool creates a new web fetch tool
func NewWebFetchTool() *WebFetchTool {
	return &WebFetchTool{
		BaseTool: BaseTool{
			name:        "web_fetch",
			toolType:    ToolTypeWebSearch, // Grouped with web tools
			description: "Lädt den Inhalt einer URL und extrahiert den Text - schneller für AI als Suche",
			schema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"url": map[string]interface{}{
						"type":        "string",
						"description": "Die URL der Webseite",
					},
					"extractLinks": map[string]interface{}{
						"type":        "boolean",
						"description": "Links auf der Seite extrahieren",
						"default":     false,
					},
					"maxLength": map[string]interface{}{
						"type":        "integer",
						"description": "Maximale Textlänge (default: 10000)",
						"default":     10000,
					},
				},
				"required": []string{"url"},
			},
		},
		client: &http.Client{
			Timeout: 15 * time.Second,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 5 {
					return fmt.Errorf("too many redirects")
				}
				return nil
			},
		},
		userAgent: "Mozilla/5.0 (compatible; FleetNavigator/1.0; +https://javafleet.de/navigator)",
	}
}

func (t *WebFetchTool) RequiresMate() bool {
	return false
}

func (t *WebFetchTool) Execute(ctx context.Context, params map[string]interface{}) (*ToolResult, error) {
	urlStr, ok := params["url"].(string)
	if !ok || urlStr == "" {
		return nil, NewToolError(t.name, "url parameter is required", nil)
	}

	// Ensure URL has scheme
	if !strings.HasPrefix(urlStr, "http://") && !strings.HasPrefix(urlStr, "https://") {
		urlStr = "https://" + urlStr
	}

	extractLinks := false
	if el, ok := params["extractLinks"].(bool); ok {
		extractLinks = el
	}

	maxLength := 10000
	if ml, ok := params["maxLength"].(float64); ok {
		maxLength = int(ml)
	}

	// Fetch the URL
	result, err := t.fetchURL(ctx, urlStr, extractLinks, maxLength)
	if err != nil {
		return &ToolResult{
			Success: false,
			Error:   err.Error(),
			Source:  "web_fetch",
		}, nil
	}

	return &ToolResult{
		Success: true,
		Data:    result,
		Source:  "web_fetch",
	}, nil
}

func (t *WebFetchTool) fetchURL(ctx context.Context, urlStr string, extractLinks bool, maxLength int) (*WebFetchResult, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", urlStr, nil)
	if err != nil {
		return nil, NewToolError(t.name, "failed to create request", err)
	}

	req.Header.Set("User-Agent", t.userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "de-DE,de;q=0.9,en;q=0.8")

	resp, err := t.client.Do(req)
	if err != nil {
		return nil, NewToolError(t.name, "request failed", err)
	}
	defer resp.Body.Close()

	result := &WebFetchResult{
		URL:         urlStr,
		StatusCode:  resp.StatusCode,
		ContentType: resp.Header.Get("Content-Type"),
	}

	if resp.StatusCode >= 400 {
		return result, NewToolError(t.name, fmt.Sprintf("HTTP %d", resp.StatusCode), nil)
	}

	// Read body with limit
	limitedReader := io.LimitReader(resp.Body, int64(maxLength*10)) // Read more, truncate later
	body, err := io.ReadAll(limitedReader)
	if err != nil {
		return nil, NewToolError(t.name, "failed to read response", err)
	}

	// Check if it's HTML
	contentType := strings.ToLower(resp.Header.Get("Content-Type"))
	if strings.Contains(contentType, "text/html") || strings.Contains(contentType, "application/xhtml") {
		// Parse HTML and extract text
		t.parseHTML(string(body), result, extractLinks)
	} else if strings.Contains(contentType, "text/") || strings.Contains(contentType, "json") {
		// Plain text or JSON - use as-is
		result.Content = string(body)
	} else {
		result.Content = fmt.Sprintf("[Binary content: %s]", contentType)
	}

	// Truncate if too long
	if len(result.Content) > maxLength {
		result.Content = result.Content[:maxLength] + "\n... [truncated]"
	}

	return result, nil
}

func (t *WebFetchTool) parseHTML(htmlContent string, result *WebFetchResult, extractLinks bool) {
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		// Fall back to regex-based extraction
		result.Content = t.extractTextRegex(htmlContent)
		return
	}

	var sb strings.Builder
	var links []string

	var extractText func(*html.Node)
	extractText = func(n *html.Node) {
		// Skip script, style, nav, footer, header
		if n.Type == html.ElementNode {
			switch n.Data {
			case "script", "style", "nav", "footer", "header", "aside", "noscript":
				return
			case "title":
				// Extract title
				if n.FirstChild != nil && n.FirstChild.Type == html.TextNode {
					result.Title = strings.TrimSpace(n.FirstChild.Data)
				}
				return
			case "meta":
				// Extract meta description
				var name, content string
				for _, attr := range n.Attr {
					if attr.Key == "name" {
						name = strings.ToLower(attr.Val)
					}
					if attr.Key == "content" {
						content = attr.Val
					}
				}
				if name == "description" && content != "" {
					result.MetaDescription = content
				}
				return
			case "a":
				// Extract links if requested
				if extractLinks {
					for _, attr := range n.Attr {
						if attr.Key == "href" && strings.HasPrefix(attr.Val, "http") {
							links = append(links, attr.Val)
						}
					}
				}
			case "br", "p", "div", "li", "h1", "h2", "h3", "h4", "h5", "h6", "tr":
				sb.WriteString("\n")
			}
		}

		if n.Type == html.TextNode {
			text := strings.TrimSpace(n.Data)
			if text != "" {
				sb.WriteString(text)
				sb.WriteString(" ")
			}
		}

		for c := n.FirstChild; c != nil; c = c.NextSibling {
			extractText(c)
		}
	}

	extractText(doc)

	// Clean up the text
	result.Content = t.cleanText(sb.String())

	if extractLinks && len(links) > 0 {
		// Deduplicate links
		seen := make(map[string]bool)
		var uniqueLinks []string
		for _, link := range links {
			if !seen[link] {
				seen[link] = true
				uniqueLinks = append(uniqueLinks, link)
			}
		}
		result.Links = uniqueLinks
	}
}

func (t *WebFetchTool) extractTextRegex(htmlContent string) string {
	// Remove script and style blocks
	scriptRe := regexp.MustCompile(`(?is)<script[^>]*>.*?</script>`)
	styleRe := regexp.MustCompile(`(?is)<style[^>]*>.*?</style>`)
	htmlContent = scriptRe.ReplaceAllString(htmlContent, "")
	htmlContent = styleRe.ReplaceAllString(htmlContent, "")

	// Extract title
	titleRe := regexp.MustCompile(`(?is)<title[^>]*>(.*?)</title>`)
	if match := titleRe.FindStringSubmatch(htmlContent); len(match) > 1 {
		htmlContent = "# " + strings.TrimSpace(match[1]) + "\n\n" + htmlContent
	}

	// Remove HTML tags
	tagRe := regexp.MustCompile(`<[^>]+>`)
	text := tagRe.ReplaceAllString(htmlContent, " ")

	return t.cleanText(text)
}

func (t *WebFetchTool) cleanText(text string) string {
	// Normalize whitespace
	spaceRe := regexp.MustCompile(`[ \t]+`)
	text = spaceRe.ReplaceAllString(text, " ")

	// Normalize newlines
	newlineRe := regexp.MustCompile(`\n\s*\n+`)
	text = newlineRe.ReplaceAllString(text, "\n\n")

	// Decode common HTML entities
	text = strings.ReplaceAll(text, "&nbsp;", " ")
	text = strings.ReplaceAll(text, "&amp;", "&")
	text = strings.ReplaceAll(text, "&lt;", "<")
	text = strings.ReplaceAll(text, "&gt;", ">")
	text = strings.ReplaceAll(text, "&quot;", "\"")
	text = strings.ReplaceAll(text, "&#39;", "'")

	return strings.TrimSpace(text)
}
