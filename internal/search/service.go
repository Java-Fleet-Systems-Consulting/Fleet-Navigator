package search

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"

	"golang.org/x/net/html"
)

// TimeFilter defines time ranges for search
type TimeFilter string

const (
	TimeFilterDay   TimeFilter = "day"
	TimeFilterWeek  TimeFilter = "week"
	TimeFilterMonth TimeFilter = "month"
	TimeFilterYear  TimeFilter = "year"
)

// SearchResult represents a single search result
type SearchResult struct {
	Title       string `json:"title"`
	URL         string `json:"url"`
	Snippet     string `json:"snippet"`
	Content     string `json:"content,omitempty"` // Full content if fetched
	Source      string `json:"source"`            // brave, searxng, duckduckgo
	Relevance   int    `json:"relevance,omitempty"`
}

// SearchOptions configures search behavior
type SearchOptions struct {
	MaxResults       int        `json:"maxResults"`
	Domains          []string   `json:"domains,omitempty"`          // Preferred domains
	ExcludeDomains   []string   `json:"excludeDomains,omitempty"`   // Excluded domains
	TimeFilter       TimeFilter `json:"timeFilter,omitempty"`
	FetchFullContent bool       `json:"fetchFullContent"`
	ReRank           bool       `json:"reRank"`
	MaxContentLength int        `json:"maxContentLength"`
	ExpertContext    string     `json:"expertContext,omitempty"` // z.B. "Rechtsberater"
	Region           string     `json:"region"`                  // de-de, en-us etc.
}

// DefaultSearchOptions returns sensible defaults
func DefaultSearchOptions() SearchOptions {
	return SearchOptions{
		MaxResults:       7,
		FetchFullContent: false,
		ReRank:           true,
		MaxContentLength: 2000,
		Region:           "de-de",
	}
}

// Settings holds WebSearch configuration
type Settings struct {
	BraveAPIKey         string   `json:"braveApiKey,omitempty"`
	SearXNGInstances    []string `json:"searxngInstances"`
	CustomSearXNG       string   `json:"customSearxng,omitempty"`
	EnableQueryOptimize bool     `json:"enableQueryOptimize"`
	EnableContentFetch  bool     `json:"enableContentFetch"`
	EnableReRanking     bool     `json:"enableReRanking"`
	EnableMultiQuery    bool     `json:"enableMultiQuery"`
	OptimizationModel   string   `json:"optimizationModel,omitempty"`
	MonthlySearchCount  int      `json:"monthlySearchCount"`
	CurrentMonth        string   `json:"currentMonth"`
}

// DefaultSettings returns default configuration
func DefaultSettings() Settings {
	return Settings{
		// Eigene SearXNG-Instanz (Java Fleet) als primäre Quelle
		CustomSearXNG: "https://search.java-fleet.com",
		// Fallback-Instanzen falls eigene nicht erreichbar
		SearXNGInstances: []string{
			"https://searx.be",
			"https://search.sapti.me",
			"https://searx.tiekoetter.com",
		},
		EnableQueryOptimize: true,
		EnableContentFetch:  true,
		EnableReRanking:     true,
		OptimizationModel:   "llama3.2:3b",
	}
}

// cacheEntry stores cached results
type cacheEntry struct {
	results   []SearchResult
	timestamp time.Time
}

type contentCacheEntry struct {
	content   string
	timestamp time.Time
}

// Service handles web searches
type Service struct {
	client       *http.Client
	settings     Settings
	settingsMu   sync.RWMutex

	// Caches
	searchCache  map[string]cacheEntry
	contentCache map[string]contentCacheEntry
	cacheMu      sync.RWMutex

	// Cache TTLs
	searchCacheTTL  time.Duration
	contentCacheTTL time.Duration

	userAgent string
}

// NewService creates a new search service
func NewService() *Service {
	return &Service{
		client: &http.Client{
			Timeout: 15 * time.Second,
		},
		settings:        DefaultSettings(),
		searchCache:     make(map[string]cacheEntry),
		contentCache:    make(map[string]contentCacheEntry),
		searchCacheTTL:  15 * time.Minute,
		contentCacheTTL: 30 * time.Minute,
		userAgent:       "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
	}
}

// GetSettings returns current settings
func (s *Service) GetSettings() Settings {
	s.settingsMu.RLock()
	defer s.settingsMu.RUnlock()
	return s.settings
}

// UpdateSettings updates settings
func (s *Service) UpdateSettings(settings Settings) {
	s.settingsMu.Lock()
	defer s.settingsMu.Unlock()
	s.settings = settings
}

// Search performs a web search with options
func (s *Service) Search(ctx context.Context, query string, opts SearchOptions) ([]SearchResult, error) {
	if query == "" {
		return nil, fmt.Errorf("empty query")
	}

	// Check cache
	cacheKey := s.buildCacheKey(query, opts)
	if cached := s.getFromCache(cacheKey); cached != nil {
		return cached, nil
	}

	var results []SearchResult
	var err error

	s.settingsMu.RLock()
	hasBraveKey := s.settings.BraveAPIKey != ""
	s.settingsMu.RUnlock()

	// Try Brave API first if available
	if hasBraveKey {
		results, err = s.searchBrave(ctx, query, opts)
		if err == nil && len(results) > 0 {
			results = s.postProcess(results, query, opts)
			s.putInCache(cacheKey, results)
			return results, nil
		}
	}

	// Fallback to SearXNG
	results, err = s.searchSearXNG(ctx, query, opts)
	if err == nil && len(results) > 0 {
		results = s.postProcess(results, query, opts)
		s.putInCache(cacheKey, results)
		return results, nil
	}

	// Final fallback to DuckDuckGo
	results, err = s.searchDuckDuckGo(ctx, query, opts)
	if err != nil {
		return nil, fmt.Errorf("all search backends failed: %w", err)
	}

	results = s.postProcess(results, query, opts)
	s.putInCache(cacheKey, results)
	return results, nil
}

// searchBrave uses Brave Search API
func (s *Service) searchBrave(ctx context.Context, query string, opts SearchOptions) ([]SearchResult, error) {
	s.settingsMu.RLock()
	apiKey := s.settings.BraveAPIKey
	s.settingsMu.RUnlock()

	if apiKey == "" {
		return nil, fmt.Errorf("no Brave API key configured")
	}

	// Build Brave API URL
	apiURL := fmt.Sprintf(
		"https://api.search.brave.com/res/v1/web/search?q=%s&count=%d",
		url.QueryEscape(query),
		opts.MaxResults,
	)

	// Add time filter
	if opts.TimeFilter != "" {
		freshnessMap := map[TimeFilter]string{
			TimeFilterDay:   "pd",
			TimeFilterWeek:  "pw",
			TimeFilterMonth: "pm",
			TimeFilterYear:  "py",
		}
		if freshness, ok := freshnessMap[opts.TimeFilter]; ok {
			apiURL += "&freshness=" + freshness
		}
	}

	// Add region
	if opts.Region != "" {
		apiURL += "&country=" + strings.Split(opts.Region, "-")[0]
	}

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("X-Subscription-Token", apiKey)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Brave API returned %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Parse Brave response
	var braveResp struct {
		Web struct {
			Results []struct {
				Title       string `json:"title"`
				URL         string `json:"url"`
				Description string `json:"description"`
			} `json:"results"`
		} `json:"web"`
	}

	if err := json.Unmarshal(body, &braveResp); err != nil {
		return nil, err
	}

	var results []SearchResult
	for _, r := range braveResp.Web.Results {
		results = append(results, SearchResult{
			Title:   r.Title,
			URL:     r.URL,
			Snippet: r.Description,
			Source:  "brave",
		})
	}

	// Increment monthly counter
	s.incrementSearchCount()

	return results, nil
}

// searchSearXNG uses SearXNG instances
func (s *Service) searchSearXNG(ctx context.Context, query string, opts SearchOptions) ([]SearchResult, error) {
	s.settingsMu.RLock()
	instances := s.settings.SearXNGInstances
	customInstance := s.settings.CustomSearXNG
	s.settingsMu.RUnlock()

	// Try custom instance first
	if customInstance != "" {
		instances = append([]string{customInstance}, instances...)
	}

	var lastErr error
	for _, instance := range instances {
		results, err := s.searchSearXNGInstance(ctx, instance, query, opts)
		if err == nil && len(results) > 0 {
			return results, nil
		}
		lastErr = err
	}

	if lastErr != nil {
		return nil, fmt.Errorf("all SearXNG instances failed: %w", lastErr)
	}
	return nil, fmt.Errorf("no results from SearXNG")
}

func (s *Service) searchSearXNGInstance(ctx context.Context, instance, query string, opts SearchOptions) ([]SearchResult, error) {
	// Build SearXNG URL
	apiURL := fmt.Sprintf(
		"%s/search?q=%s&format=json&language=%s",
		strings.TrimSuffix(instance, "/"),
		url.QueryEscape(query),
		opts.Region,
	)

	// Add time filter
	if opts.TimeFilter != "" {
		apiURL += "&time_range=" + string(opts.TimeFilter)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", s.userAgent)
	req.Header.Set("Accept", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("SearXNG returned %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Parse SearXNG response
	var searxResp struct {
		Results []struct {
			Title   string `json:"title"`
			URL     string `json:"url"`
			Content string `json:"content"`
		} `json:"results"`
	}

	if err := json.Unmarshal(body, &searxResp); err != nil {
		return nil, err
	}

	var results []SearchResult
	for i, r := range searxResp.Results {
		if i >= opts.MaxResults {
			break
		}
		results = append(results, SearchResult{
			Title:   r.Title,
			URL:     r.URL,
			Snippet: r.Content,
			Source:  "searxng",
		})
	}

	return results, nil
}

// searchDuckDuckGo as fallback
func (s *Service) searchDuckDuckGo(ctx context.Context, query string, opts SearchOptions) ([]SearchResult, error) {
	// Try instant answer API first
	apiURL := fmt.Sprintf(
		"https://api.duckduckgo.com/?q=%s&format=json&no_html=1&skip_disambig=1&kl=%s",
		url.QueryEscape(query),
		url.QueryEscape(opts.Region),
	)

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", s.userAgent)

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var ddgResp struct {
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

	if err := json.Unmarshal(body, &ddgResp); err != nil {
		return nil, err
	}

	var results []SearchResult

	// Add abstract
	if ddgResp.Abstract != "" {
		results = append(results, SearchResult{
			Title:   ddgResp.AbstractSource,
			URL:     ddgResp.AbstractURL,
			Snippet: ddgResp.Abstract,
			Source:  "duckduckgo",
		})
	}

	// Add direct results
	for _, r := range ddgResp.Results {
		if len(results) >= opts.MaxResults {
			break
		}
		results = append(results, SearchResult{
			Title:   extractTitle(r.Text),
			URL:     r.FirstURL,
			Snippet: r.Text,
			Source:  "duckduckgo",
		})
	}

	// Add related topics
	for _, t := range ddgResp.RelatedTopics {
		if len(results) >= opts.MaxResults {
			break
		}
		if t.FirstURL != "" {
			results = append(results, SearchResult{
				Title:   extractTitle(t.Text),
				URL:     t.FirstURL,
				Snippet: t.Text,
				Source:  "duckduckgo",
			})
		}
	}

	// Fallback to HTML scraping if no results
	if len(results) == 0 {
		return s.searchDuckDuckGoHTML(ctx, query, opts)
	}

	return results, nil
}

// searchDuckDuckGoHTML scrapes DuckDuckGo HTML version
func (s *Service) searchDuckDuckGoHTML(ctx context.Context, query string, opts SearchOptions) ([]SearchResult, error) {
	// Use html.duckduckgo.com - more reliable than lite version
	searchURL := fmt.Sprintf(
		"https://html.duckduckgo.com/html/?q=%s&kl=%s",
		url.QueryEscape(query),
		url.QueryEscape(opts.Region),
	)

	req, err := http.NewRequestWithContext(ctx, "GET", searchURL, nil)
	if err != nil {
		return nil, err
	}
	// Set headers to mimic a real browser
	req.Header.Set("User-Agent", s.userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8")
	req.Header.Set("Accept-Language", "de-DE,de;q=0.9,en;q=0.8")
	req.Header.Set("DNT", "1")

	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return s.parseDDGHTML(string(body), opts.MaxResults), nil
}

// parseDDGHTML parses the html.duckduckgo.com results page
func (s *Service) parseDDGHTML(htmlContent string, maxResults int) []SearchResult {
	var results []SearchResult

	// Parse using regex patterns for result__a (title/link) and result__snippet
	// Pattern for result links: class="result__a" href="URL">Title</a>
	linkPattern := regexp.MustCompile(`class="result__a"[^>]*href="([^"]+)"[^>]*>([^<]+)</a>`)
	// Pattern for snippets: class="result__snippet">...content...</a>
	snippetPattern := regexp.MustCompile(`class="result__snippet"[^>]*>([^<]+)`)

	linkMatches := linkPattern.FindAllStringSubmatch(htmlContent, -1)
	snippetMatches := snippetPattern.FindAllStringSubmatch(htmlContent, -1)

	for i, match := range linkMatches {
		if len(results) >= maxResults {
			break
		}
		if len(match) < 3 {
			continue
		}

		resultURL := match[1]
		resultTitle := strings.TrimSpace(match[2])

		// Skip DuckDuckGo internal links
		if strings.Contains(resultURL, "duckduckgo.com") {
			continue
		}

		// Decode redirect URL if present
		if strings.HasPrefix(resultURL, "//duckduckgo.com/l/?uddg=") {
			if decoded, err := url.QueryUnescape(strings.TrimPrefix(resultURL, "//duckduckgo.com/l/?uddg=")); err == nil {
				// Extract actual URL (before &rut=)
				if idx := strings.Index(decoded, "&"); idx != -1 {
					resultURL = decoded[:idx]
				} else {
					resultURL = decoded
				}
			}
		}

		snippet := ""
		if i < len(snippetMatches) && len(snippetMatches[i]) > 1 {
			snippet = strings.TrimSpace(snippetMatches[i][1])
		}

		results = append(results, SearchResult{
			Title:   stripHTMLTags(resultTitle),
			URL:     resultURL,
			Snippet: stripHTMLTags(snippet),
			Source:  "duckduckgo",
		})
	}

	return results
}

func (s *Service) parseDDGLiteHTML(htmlContent string, maxResults int) []SearchResult {
	var results []SearchResult
	lines := strings.Split(htmlContent, "\n")
	var currentTitle, currentURL, currentDesc string

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.Contains(line, `class="result-link"`) || strings.Contains(line, `rel="nofollow"`) {
			if hrefStart := strings.Index(line, `href="`); hrefStart != -1 {
				hrefStart += 6
				if hrefEnd := strings.Index(line[hrefStart:], `"`); hrefEnd != -1 {
					currentURL = line[hrefStart : hrefStart+hrefEnd]
				}
			}
			if titleStart := strings.Index(line, ">"); titleStart != -1 {
				if titleEnd := strings.Index(line[titleStart:], "</a>"); titleEnd != -1 {
					currentTitle = strings.TrimSpace(line[titleStart+1 : titleStart+titleEnd])
					currentTitle = stripHTMLTags(currentTitle)
				}
			}
		}

		if strings.Contains(line, `class="result-snippet"`) {
			currentDesc = stripHTMLTags(line)
		}

		if currentURL != "" && currentTitle != "" && !strings.Contains(currentURL, "duckduckgo.com") {
			results = append(results, SearchResult{
				Title:   currentTitle,
				URL:     currentURL,
				Snippet: currentDesc,
				Source:  "duckduckgo",
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

// postProcess applies filtering, re-ranking and content fetching
func (s *Service) postProcess(results []SearchResult, query string, opts SearchOptions) []SearchResult {
	// Filter by domains
	if len(opts.Domains) > 0 || len(opts.ExcludeDomains) > 0 {
		results = s.filterByDomains(results, opts.Domains, opts.ExcludeDomains)
	}

	// Re-rank by relevance
	if opts.ReRank {
		results = s.reRankResults(results, query)
	}

	// Fetch full content if requested
	if opts.FetchFullContent {
		results = s.enrichWithContent(results, opts.MaxContentLength)
	}

	// Limit results
	if len(results) > opts.MaxResults {
		results = results[:opts.MaxResults]
	}

	return results
}

// filterByDomains filters results by domain preferences
func (s *Service) filterByDomains(results []SearchResult, preferred, excluded []string) []SearchResult {
	var filtered []SearchResult

	for _, r := range results {
		domain := extractDomain(r.URL)

		// Check exclusion
		isExcluded := false
		for _, ex := range excluded {
			if strings.Contains(domain, ex) {
				isExcluded = true
				break
			}
		}
		if isExcluded {
			continue
		}

		filtered = append(filtered, r)
	}

	// Sort preferred domains first
	if len(preferred) > 0 {
		var preferredResults, otherResults []SearchResult
		for _, r := range filtered {
			domain := extractDomain(r.URL)
			isPreferred := false
			for _, pref := range preferred {
				if strings.Contains(domain, pref) {
					isPreferred = true
					break
				}
			}
			if isPreferred {
				preferredResults = append(preferredResults, r)
			} else {
				otherResults = append(otherResults, r)
			}
		}
		filtered = append(preferredResults, otherResults...)
	}

	return filtered
}

// reRankResults scores results by keyword relevance
func (s *Service) reRankResults(results []SearchResult, query string) []SearchResult {
	keywords := strings.Fields(strings.ToLower(query))

	for i := range results {
		score := 0
		titleLower := strings.ToLower(results[i].Title)
		snippetLower := strings.ToLower(results[i].Snippet)
		urlLower := strings.ToLower(results[i].URL)

		for _, kw := range keywords {
			if strings.Contains(titleLower, kw) {
				score += 3 // Title matches are most important
			}
			if strings.Contains(snippetLower, kw) {
				score += 1
			}
			if strings.Contains(urlLower, kw) {
				score += 2
			}
		}
		results[i].Relevance = score
	}

	// Sort by relevance (simple bubble sort, results are usually small)
	for i := 0; i < len(results)-1; i++ {
		for j := i + 1; j < len(results); j++ {
			if results[j].Relevance > results[i].Relevance {
				results[i], results[j] = results[j], results[i]
			}
		}
	}

	return results
}

// enrichWithContent fetches full page content
func (s *Service) enrichWithContent(results []SearchResult, maxLength int) []SearchResult {
	for i := range results {
		content, err := s.FetchPageContent(results[i].URL, maxLength)
		if err == nil && content != "" {
			results[i].Content = content
		}
	}
	return results
}

// FetchPageContent fetches and cleans page content
func (s *Service) FetchPageContent(urlStr string, maxLength int) (string, error) {
	// Check cache first
	s.cacheMu.RLock()
	if entry, ok := s.contentCache[urlStr]; ok {
		if time.Since(entry.timestamp) < s.contentCacheTTL {
			s.cacheMu.RUnlock()
			return entry.content, nil
		}
	}
	s.cacheMu.RUnlock()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", urlStr, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("User-Agent", s.userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml")
	req.Header.Set("Accept-Language", "de-DE,de;q=0.9,en;q=0.8")

	resp, err := s.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	// Read limited body
	body, err := io.ReadAll(io.LimitReader(resp.Body, int64(maxLength*10)))
	if err != nil {
		return "", err
	}

	// Parse and extract text
	content := s.extractTextFromHTML(string(body))

	// Truncate if needed
	if len(content) > maxLength {
		content = content[:maxLength] + "..."
	}

	// Cache the result
	s.cacheMu.Lock()
	s.contentCache[urlStr] = contentCacheEntry{
		content:   content,
		timestamp: time.Now(),
	}
	s.cacheMu.Unlock()

	return content, nil
}

// extractTextFromHTML parses HTML and extracts clean text
func (s *Service) extractTextFromHTML(htmlContent string) string {
	doc, err := html.Parse(strings.NewReader(htmlContent))
	if err != nil {
		return s.extractTextRegex(htmlContent)
	}

	var sb strings.Builder
	var extractText func(*html.Node)
	extractText = func(n *html.Node) {
		// Skip unwanted elements
		if n.Type == html.ElementNode {
			switch n.Data {
			case "script", "style", "nav", "footer", "header", "aside", "noscript", "iframe", "form":
				return
			case "br", "p", "div", "li", "h1", "h2", "h3", "h4", "h5", "h6", "tr", "td":
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
	return s.cleanText(sb.String())
}

func (s *Service) extractTextRegex(htmlContent string) string {
	// Remove script and style blocks
	scriptRe := regexp.MustCompile(`(?is)<script[^>]*>.*?</script>`)
	styleRe := regexp.MustCompile(`(?is)<style[^>]*>.*?</style>`)
	htmlContent = scriptRe.ReplaceAllString(htmlContent, "")
	htmlContent = styleRe.ReplaceAllString(htmlContent, "")

	// Remove HTML tags
	tagRe := regexp.MustCompile(`<[^>]+>`)
	text := tagRe.ReplaceAllString(htmlContent, " ")

	return s.cleanText(text)
}

func (s *Service) cleanText(text string) string {
	// Normalize whitespace
	spaceRe := regexp.MustCompile(`[ \t]+`)
	text = spaceRe.ReplaceAllString(text, " ")

	// Normalize newlines
	newlineRe := regexp.MustCompile(`\n\s*\n+`)
	text = newlineRe.ReplaceAllString(text, "\n\n")

	// Decode HTML entities
	text = strings.ReplaceAll(text, "&nbsp;", " ")
	text = strings.ReplaceAll(text, "&amp;", "&")
	text = strings.ReplaceAll(text, "&lt;", "<")
	text = strings.ReplaceAll(text, "&gt;", ">")
	text = strings.ReplaceAll(text, "&quot;", "\"")
	text = strings.ReplaceAll(text, "&#39;", "'")

	return strings.TrimSpace(text)
}

// FormatForContext formats results for LLM context
func (s *Service) FormatForContext(results []SearchResult, includeURLs bool) string {
	if len(results) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("Suchergebnisse:\n\n")

	for i, r := range results {
		sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, r.Title))
		if includeURLs {
			sb.WriteString(fmt.Sprintf("   URL: %s\n", r.URL))
		}
		if r.Content != "" {
			sb.WriteString(fmt.Sprintf("   %s\n", truncateText(r.Content, 500)))
		} else if r.Snippet != "" {
			sb.WriteString(fmt.Sprintf("   %s\n", r.Snippet))
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// FormatSourcesFooter creates a sources footer for responses
func (s *Service) FormatSourcesFooter(results []SearchResult) string {
	if len(results) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("\n\n---\n**Quellen:**\n")

	for i, r := range results {
		sb.WriteString(fmt.Sprintf("- [%d] [%s](%s)\n", i+1, r.Title, r.URL))
	}

	return sb.String()
}

// Cache helpers
func (s *Service) buildCacheKey(query string, opts SearchOptions) string {
	return fmt.Sprintf("%s|%d|%s", strings.ToLower(query), opts.MaxResults, opts.TimeFilter)
}

func (s *Service) getFromCache(key string) []SearchResult {
	s.cacheMu.RLock()
	defer s.cacheMu.RUnlock()

	if entry, ok := s.searchCache[key]; ok {
		if time.Since(entry.timestamp) < s.searchCacheTTL {
			return entry.results
		}
	}
	return nil
}

func (s *Service) putInCache(key string, results []SearchResult) {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()

	// Limit cache size
	if len(s.searchCache) > 100 {
		// Remove oldest entries
		for k := range s.searchCache {
			delete(s.searchCache, k)
			if len(s.searchCache) <= 50 {
				break
			}
		}
	}

	s.searchCache[key] = cacheEntry{
		results:   results,
		timestamp: time.Now(),
	}
}

func (s *Service) incrementSearchCount() {
	s.settingsMu.Lock()
	defer s.settingsMu.Unlock()

	currentMonth := time.Now().Format("2006-01")
	if s.settings.CurrentMonth != currentMonth {
		s.settings.CurrentMonth = currentMonth
		s.settings.MonthlySearchCount = 0
	}
	s.settings.MonthlySearchCount++
}

// Helper functions
func extractTitle(text string) string {
	if idx := strings.Index(text, " - "); idx > 0 && idx < 100 {
		return text[:idx]
	}
	if len(text) > 100 {
		return text[:100] + "..."
	}
	return text
}

func extractDomain(urlStr string) string {
	if u, err := url.Parse(urlStr); err == nil {
		return strings.ToLower(u.Host)
	}
	return ""
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

func truncateText(text string, maxLen int) string {
	if len(text) <= maxLen {
		return text
	}
	return text[:maxLen] + "..."
}
