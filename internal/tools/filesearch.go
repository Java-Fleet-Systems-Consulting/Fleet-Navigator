package tools

import (
	"context"
	"fmt"
)

// FileSearchTool searches for files - requires a Mate to function
type FileSearchTool struct {
	BaseTool
	// MateProvider is a function that returns a Mate connection for file operations
	// This will be set by the mate package when a Mate is connected
	MateProvider func(mateID string) (MateConnection, error)
}

// MateConnection represents a connection to a Mate that can perform file operations
type MateConnection interface {
	// SearchFiles searches for files matching the query
	SearchFiles(ctx context.Context, query string, options FileSearchOptions) ([]FileSearchResult, error)
	// ReadFile reads the content of a file
	ReadFile(ctx context.Context, path string) ([]byte, error)
	// ListDirectory lists files in a directory
	ListDirectory(ctx context.Context, path string) ([]FileInfo, error)
	// GetMateID returns the Mate's ID
	GetMateID() string
	// GetMateType returns the Mate's type (e.g., "windows", "linux", "macos")
	GetMateType() string
}

// FileSearchOptions configures file search behavior
type FileSearchOptions struct {
	// SearchIn specifies directories to search in
	SearchIn []string `json:"searchIn,omitempty"`
	// FileTypes filters by file extension (e.g., ".pdf", ".docx")
	FileTypes []string `json:"fileTypes,omitempty"`
	// SearchContent if true, searches within file contents (slower)
	SearchContent bool `json:"searchContent"`
	// MaxResults limits the number of results
	MaxResults int `json:"maxResults"`
	// IncludeHidden if true, includes hidden files
	IncludeHidden bool `json:"includeHidden"`
	// MaxDepth limits directory recursion depth
	MaxDepth int `json:"maxDepth"`
}

// FileInfo represents basic file information
type FileInfo struct {
	Name     string `json:"name"`
	Path     string `json:"path"`
	IsDir    bool   `json:"isDir"`
	Size     int64  `json:"size"`
	Modified string `json:"modified"`
}

// NewFileSearchTool creates a new file search tool
func NewFileSearchTool() *FileSearchTool {
	return &FileSearchTool{
		BaseTool: BaseTool{
			name:        "file_search",
			toolType:    ToolTypeFileSearch,
			description: "Sucht nach Dateien auf dem lokalen System (benötigt einen verbundenen Mate)",
			schema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"query": map[string]interface{}{
						"type":        "string",
						"description": "Suchbegriff für Dateinamen oder -inhalt",
					},
					"searchIn": map[string]interface{}{
						"type":        "array",
						"items":       map[string]interface{}{"type": "string"},
						"description": "Verzeichnisse in denen gesucht werden soll",
					},
					"fileTypes": map[string]interface{}{
						"type":        "array",
						"items":       map[string]interface{}{"type": "string"},
						"description": "Dateitypen filtern (z.B. .pdf, .docx)",
					},
					"searchContent": map[string]interface{}{
						"type":        "boolean",
						"description": "In Dateiinhalten suchen (langsamer)",
						"default":     false,
					},
					"maxResults": map[string]interface{}{
						"type":        "integer",
						"description": "Maximale Anzahl Ergebnisse",
						"default":     10,
					},
					"mateId": map[string]interface{}{
						"type":        "string",
						"description": "ID des zu verwendenden Mates",
					},
				},
				"required": []string{"query"},
			},
		},
	}
}

func (t *FileSearchTool) RequiresMate() bool {
	return true // FileSearch needs a Mate to access the user's filesystem
}

func (t *FileSearchTool) Execute(ctx context.Context, params map[string]interface{}) (*ToolResult, error) {
	// Check if MateProvider is configured
	if t.MateProvider == nil {
		return &ToolResult{
			Success: false,
			Error:   "Kein Mate verbunden. FileSearch benötigt einen Fleet-Mate auf dem lokalen System.",
		}, nil
	}

	// Extract parameters
	query, ok := params["query"].(string)
	if !ok || query == "" {
		return nil, NewToolError(t.name, "query parameter is required", nil)
	}

	mateID, _ := params["mateId"].(string)
	if mateID == "" {
		return &ToolResult{
			Success: false,
			Error:   "Keine Mate-ID angegeben. Bitte wähle einen verbundenen Mate aus.",
		}, nil
	}

	// Get Mate connection
	mate, err := t.MateProvider(mateID)
	if err != nil {
		return &ToolResult{
			Success: false,
			Error:   fmt.Sprintf("Mate '%s' nicht erreichbar: %v", mateID, err),
		}, nil
	}

	// Build search options
	options := FileSearchOptions{
		MaxResults: 10,
	}

	if searchIn, ok := params["searchIn"].([]interface{}); ok {
		for _, dir := range searchIn {
			if dirStr, ok := dir.(string); ok {
				options.SearchIn = append(options.SearchIn, dirStr)
			}
		}
	}

	if fileTypes, ok := params["fileTypes"].([]interface{}); ok {
		for _, ft := range fileTypes {
			if ftStr, ok := ft.(string); ok {
				options.FileTypes = append(options.FileTypes, ftStr)
			}
		}
	}

	if searchContent, ok := params["searchContent"].(bool); ok {
		options.SearchContent = searchContent
	}

	if maxResults, ok := params["maxResults"].(float64); ok {
		options.MaxResults = int(maxResults)
	}

	// Execute search via Mate
	results, err := mate.SearchFiles(ctx, query, options)
	if err != nil {
		return &ToolResult{
			Success: false,
			Error:   fmt.Sprintf("Suche fehlgeschlagen: %v", err),
			Source:  fmt.Sprintf("mate:%s", mate.GetMateID()),
		}, nil
	}

	return &ToolResult{
		Success: true,
		Data:    results,
		Source:  fmt.Sprintf("mate:%s (%s)", mate.GetMateID(), mate.GetMateType()),
	}, nil
}

// SetMateProvider sets the function to get Mate connections
func (t *FileSearchTool) SetMateProvider(provider func(string) (MateConnection, error)) {
	t.MateProvider = provider
}
