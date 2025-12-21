// Package tools defines the tool system for Fleet Navigator
// Tools are capabilities that can be executed by experts during conversations
package tools

import (
	"context"
	"fmt"
)

// ToolType represents the type of tool
type ToolType string

const (
	ToolTypeWebSearch  ToolType = "web_search"
	ToolTypeFileSearch ToolType = "file_search"
	ToolTypeCalculator ToolType = "calculator"
	ToolTypeDateTime   ToolType = "datetime"
)

// ToolResult represents the result of a tool execution
type ToolResult struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	// Source indicates where the result came from (e.g., "duckduckgo", "mate:windows-pc")
	Source string `json:"source,omitempty"`
}

// ToolRequest represents a request to execute a tool
type ToolRequest struct {
	Type       ToolType               `json:"type"`
	Parameters map[string]interface{} `json:"parameters"`
	// MateID is optional - if set, the request should be routed to a specific Mate
	MateID string `json:"mateId,omitempty"`
}

// Tool is the interface that all tools must implement
type Tool interface {
	// Name returns the tool's unique name
	Name() string
	// Type returns the tool type
	Type() ToolType
	// Description returns a human-readable description
	Description() string
	// Execute runs the tool with the given parameters
	Execute(ctx context.Context, params map[string]interface{}) (*ToolResult, error)
	// RequiresMate returns true if this tool needs a Mate to function
	RequiresMate() bool
	// ParameterSchema returns JSON schema for the parameters
	ParameterSchema() map[string]interface{}
}

// BaseTool provides common functionality for tools
type BaseTool struct {
	name        string
	toolType    ToolType
	description string
	schema      map[string]interface{}
}

func (t *BaseTool) Name() string                      { return t.name }
func (t *BaseTool) Type() ToolType                    { return t.toolType }
func (t *BaseTool) Description() string               { return t.description }
func (t *BaseTool) ParameterSchema() map[string]interface{} { return t.schema }

// SearchResult represents a single search result
type SearchResult struct {
	Title       string `json:"title"`
	URL         string `json:"url"`
	Description string `json:"description"`
	Source      string `json:"source,omitempty"`
}

// FileSearchResult represents a file search result
type FileSearchResult struct {
	FileName  string `json:"fileName"`
	FilePath  string `json:"filePath"`
	FileType  string `json:"fileType"`
	Size      int64  `json:"size"`
	Modified  string `json:"modified"`
	Snippet   string `json:"snippet,omitempty"`
	MatchType string `json:"matchType"` // "name", "content", "metadata"
}

// ToolError represents a tool-specific error
type ToolError struct {
	Tool    string
	Message string
	Cause   error
}

func (e *ToolError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("tool '%s': %s: %v", e.Tool, e.Message, e.Cause)
	}
	return fmt.Sprintf("tool '%s': %s", e.Tool, e.Message)
}

// NewToolError creates a new tool error
func NewToolError(tool, message string, cause error) *ToolError {
	return &ToolError{Tool: tool, Message: message, Cause: cause}
}
