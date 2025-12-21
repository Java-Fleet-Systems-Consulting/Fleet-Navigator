package tools

import (
	"context"
	"fmt"
	"sync"
)

// Registry manages all available tools
type Registry struct {
	mu    sync.RWMutex
	tools map[string]Tool
}

// NewRegistry creates a new tool registry with default tools
func NewRegistry() *Registry {
	r := &Registry{
		tools: make(map[string]Tool),
	}

	// Register default tools
	r.Register(NewWebSearchTool())
	r.Register(NewWebFetchTool())
	r.Register(NewFileSearchTool())

	return r
}

// Register adds a tool to the registry
func (r *Registry) Register(tool Tool) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.tools[tool.Name()]; exists {
		return fmt.Errorf("tool '%s' already registered", tool.Name())
	}

	r.tools[tool.Name()] = tool
	return nil
}

// Get returns a tool by name
func (r *Registry) Get(name string) (Tool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tool, ok := r.tools[name]
	return tool, ok
}

// GetByType returns all tools of a specific type
func (r *Registry) GetByType(toolType ToolType) []Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []Tool
	for _, tool := range r.tools {
		if tool.Type() == toolType {
			result = append(result, tool)
		}
	}
	return result
}

// List returns all registered tools
func (r *Registry) List() []Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]Tool, 0, len(r.tools))
	for _, tool := range r.tools {
		result = append(result, tool)
	}
	return result
}

// ListAvailable returns tools that are currently usable
// (e.g., excludes tools that require Mate if no Mate is connected)
func (r *Registry) ListAvailable() []Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []Tool
	for _, tool := range r.tools {
		// For now, include all tools
		// Later we can filter based on Mate availability
		result = append(result, tool)
	}
	return result
}

// Execute runs a tool with the given parameters
func (r *Registry) Execute(ctx context.Context, toolName string, params map[string]interface{}) (*ToolResult, error) {
	tool, ok := r.Get(toolName)
	if !ok {
		return nil, fmt.Errorf("tool '%s' not found", toolName)
	}

	return tool.Execute(ctx, params)
}

// ExecuteRequest runs a tool request
func (r *Registry) ExecuteRequest(ctx context.Context, req *ToolRequest) (*ToolResult, error) {
	tool, ok := r.Get(string(req.Type))
	if !ok {
		// Try by type name
		tools := r.GetByType(req.Type)
		if len(tools) == 0 {
			return nil, fmt.Errorf("no tool found for type '%s'", req.Type)
		}
		tool = tools[0]
	}

	// If MateID is specified and tool requires mate, add it to params
	if req.MateID != "" {
		if req.Parameters == nil {
			req.Parameters = make(map[string]interface{})
		}
		req.Parameters["mateId"] = req.MateID
	}

	return tool.Execute(ctx, req.Parameters)
}

// GetToolDefinitions returns tool definitions for LLM function calling
func (r *Registry) GetToolDefinitions() []ToolDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var defs []ToolDefinition
	for _, tool := range r.tools {
		defs = append(defs, ToolDefinition{
			Name:         tool.Name(),
			Description:  tool.Description(),
			Parameters:   tool.ParameterSchema(),
			RequiresMate: tool.RequiresMate(),
		})
	}
	return defs
}

// ToolDefinition is used for LLM function calling setup
type ToolDefinition struct {
	Name         string                 `json:"name"`
	Description  string                 `json:"description"`
	Parameters   map[string]interface{} `json:"parameters"`
	RequiresMate bool                   `json:"requiresMate"`
}

// SetFileSearchMateProvider configures the Mate provider for file search
func (r *Registry) SetFileSearchMateProvider(provider func(string) (MateConnection, error)) {
	r.mu.Lock()
	defer r.mu.Unlock()

	if tool, ok := r.tools["file_search"]; ok {
		if fst, ok := tool.(*FileSearchTool); ok {
			fst.SetMateProvider(provider)
		}
	}
}

// ToolInfo provides serializable tool information
type ToolInfo struct {
	Name         string   `json:"name"`
	Type         ToolType `json:"type"`
	Description  string   `json:"description"`
	RequiresMate bool     `json:"requiresMate"`
	Available    bool     `json:"available"`
}

// GetToolInfo returns information about all tools
func (r *Registry) GetToolInfo() []ToolInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var infos []ToolInfo
	for _, tool := range r.tools {
		available := true
		if tool.RequiresMate() {
			// Check if FileSearch has a provider
			if fst, ok := tool.(*FileSearchTool); ok {
				available = fst.MateProvider != nil
			}
		}

		infos = append(infos, ToolInfo{
			Name:         tool.Name(),
			Type:         tool.Type(),
			Description:  tool.Description(),
			RequiresMate: tool.RequiresMate(),
			Available:    available,
		})
	}
	return infos
}
