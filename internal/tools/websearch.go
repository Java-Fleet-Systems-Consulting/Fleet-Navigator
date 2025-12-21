package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// WebSearchTool performs web searches using DuckDuckGo
type WebSearchTool struct {
	BaseTool
	client    *http.Client
	userAgent string
}

// NewWebSearchTool creates a new web search tool
func NewWebSearchTool() *WebSearchTool {
	return &WebSearchTool{
		BaseTool: BaseTool{
			name:        "web_search",
			toolType:    ToolTypeWebSearch,
			description: "Sucht im Internet nach Informationen mit DuckDuckGo",
			schema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{
						"type":        "string",
						"description": "Die Suchanfrage",
					},
					"maxResults": map[string]interface{}{
						"type":        "integer",
						"description": "Maximale Anzahl der Ergebnisse (default: 5)",
						"default":     5,
					},
					"region": map[string]interface{}{
						"type":        "string",
						"description": "Region für die Suche (z.B. de-de, en-us)",
						"default":     "de-de",
					},
				},
				"required": []string{"query"},
			},
		},
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		userAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
	}
}

func (t *WebSearchTool) RequiresMate() bool {
	return false // WebSearch runs directly on the Navigator
}

func (t *WebSearchTool) Execute(ctx context.Context, params map[string]interface{}) (*ToolResult, error) {
	// Extract parameters
	query, ok := params["query"].(string)
	if !ok || query == "" {
		return nil, NewToolError(t.name, "query parameter is required", nil)
	}

	maxResults := 5
	if mr, ok := params["maxResults"].(float64); ok {
		maxResults = int(mr)
	}

	region := "de-de"
	if r, ok := params["region"].(string); ok {
		region = r
	}

	// Perform the search
	results, err := t.searchDuckDuckGo(ctx, query, maxResults, region)
	if err != nil {
		return &ToolResult{
			Success: false,
			Error:   err.Error(),
			Source:  "duckduckgo",
		}, nil
	}

	return &ToolResult{
		Success: true,
		Data:    results,
		Source:  "duckduckgo",
	}, nil
}

// searchDuckDuckGo performs a search using DuckDuckGo's instant answer API
func (t *WebSearchTool) searchDuckDuckGo(ctx context.Context, query string, maxResults int, region string) ([]SearchResult, error) {
	// Use DuckDuckGo's instant answer API (JSON)
	apiURL := fmt.Sprintf(
		"https://api.duckduckgo.com/?q=%s&format=json&no_html=1&skip_disambig=1&kl=%s",
		url.QueryEscape(query),
		url.QueryEscape(region),
	)

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, NewToolError(t.name, "failed to create request", err)
	}
	req.Header.Set("User-Agent", t.userAgent)

	resp, err := t.client.Do(req)
	if err != nil {
		return nil, NewToolError(t.name, "search request failed", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, NewToolError(t.name, "failed to read response", err)
	}

	// Parse DuckDuckGo response
	var ddgResponse struct {
		Abstract       string `json:"Abstract"`
		AbstractSource string `json:"AbstractSource"`
		AbstractURL    string `json:"AbstractURL"`
		RelatedTopics  []struct {
			Text     string `json:"Text"`
			FirstURL string `json:"FirstURL"`
		} `json:"RelatedTopics"`
		Results []struct {
			Text     string `json:"Text"`
			FirstURL string `json:"FirstURL"`
		} `json:"Results"`
	}

	if err := json.Unmarshal(body, &ddgResponse); err != nil {
		return nil, NewToolError(t.name, "failed to parse response", err)
	}

	var results []SearchResult

	// Add abstract if available
	if ddgResponse.Abstract != "" {
		results = append(results, SearchResult{
			Title:       ddgResponse.AbstractSource,
			URL:         ddgResponse.AbstractURL,
			Description: ddgResponse.Abstract,
			Source:      "duckduckgo",
		})
	}

	// Add direct results
	for _, r := range ddgResponse.Results {
		if len(results) >= maxResults {
			break
		}
		results = append(results, SearchResult{
			Title:       extractTitle(r.Text),
			URL:         r.FirstURL,
			Description: r.Text,
			Source:      "duckduckgo",
		})
	}

	// Add related topics
	for _, topic := range ddgResponse.RelatedTopics {
		if len(results) >= maxResults {
			break
		}
		if topic.FirstURL != "" {
			results = append(results, SearchResult{
				Title:       extractTitle(topic.Text),
				URL:         topic.FirstURL,
				Description: topic.Text,
				Source:      "duckduckgo",
			})
		}
	}

	// If no results from API, try HTML scraping fallback
	if len(results) == 0 {
		return t.searchDuckDuckGoHTML(ctx, query, maxResults, region)
	}

	return results, nil
}

// searchDuckDuckGoHTML is a fallback that scrapes DuckDuckGo's HTML results
func (t *WebSearchTool) searchDuckDuckGoHTML(ctx context.Context, query string, maxResults int, region string) ([]SearchResult, error) {
	// Use DuckDuckGo's lite version for easier parsing
	searchURL := fmt.Sprintf(
		"https://lite.duckduckgo.com/lite/?q=%s&kl=%s",
		url.QueryEscape(query),
		url.QueryEscape(region),
	)

	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return nil, NewToolError(t.name, "failed to create request", err)
	}
	req.Header.Set("User-Agent", t.userAgent)

	resp, err := t.client.Do(req)
	if err != nil {
		return nil, NewToolError(t.name, "search request failed", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, NewToolError(t.name, "failed to read response", err)
	}

	// Simple parsing of lite.duckduckgo.com results
	return t.parseLiteResults(string(body), maxResults), nil
}

// parseLiteResults extracts search results from DuckDuckGo Lite HTML
func (t *WebSearchTool) parseLiteResults(html string, maxResults int) []SearchResult {
	var results []SearchResult

	// Split by result blocks - DuckDuckGo Lite uses simple table structure
	// Look for links that are actual results (not navigation)
	lines := strings.Split(html, "\n")
	var currentTitle, currentURL, currentDesc string

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Look for result links
		if strings.Contains(line, `class="result-link"`) || strings.Contains(line, `rel="nofollow"`) {
			// Extract URL and title
			if hrefStart := strings.Index(line, `href="`); hrefStart != -1 {
				hrefStart += 6
				if hrefEnd := strings.Index(line[hrefStart:], `"`); hrefEnd != -1 {
					currentURL = line[hrefStart : hrefStart+hrefEnd]
				}
			}
			// Extract title (text between > and </a>)
			if titleStart := strings.Index(line, ">"); titleStart != -1 {
				if titleEnd := strings.Index(line[titleStart:], "</a>"); titleEnd != -1 {
					currentTitle = strings.TrimSpace(line[titleStart+1 : titleStart+titleEnd])
					currentTitle = stripHTMLTags(currentTitle)
				}
			}
		}

		// Look for snippet/description
		if strings.Contains(line, `class="result-snippet"`) || strings.Contains(line, `<td class="result-snippet"`) {
			currentDesc = stripHTMLTags(line)
		}

		// When we have both URL and title, create a result
		if currentURL != "" && currentTitle != "" && !strings.Contains(currentURL, "duckduckgo.com") {
			results = append(results, SearchResult{
				Title:       currentTitle,
				URL:         currentURL,
				Description: currentDesc,
				Source:      "duckduckgo",
			})
			currentTitle = ""
			currentURL = ""
			currentDesc = ""

			if len(results) >= maxResults {
				break
			}
		}
	}

	return results
}

// Helper functions

func extractTitle(text string) string {
	// Extract first part before " - " or first 100 chars
	if idx := strings.Index(text, " - "); idx > 0 && idx < 100 {
		return text[:idx]
	}
	if len(text) > 100 {
		return text[:100] + "..."
	}
	return text
}

func stripHTMLTags(s string) string {
	var result strings.Builder
	inTag := false
	for _, r := range s {
		if r == '<' {
			inTag = true
		} else if r == '>' {
			inTag = false
		} else if !inTag {
			result.WriteRune(r)
		}
	}
	return strings.TrimSpace(result.String())
}
