package settings

import (
	"testing"
)

func TestNormalizeProviderName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"ollama", "ollama"},
		{"OLLAMA", "ollama"},
		{"  ollama  ", "ollama"},
		{"llama-cpp", "llama-cpp"},
		{"llamacpp", "llama-cpp"},
		{"llama_cpp", "llama-cpp"},
		{"llama-server", "llama-cpp"},
		{"unknown", "llama-cpp"},
		{"", "llama-cpp"},
	}

	for _, tt := range tests {
		result := normalizeProviderName(tt.input)
		if result != tt.expected {
			t.Errorf("normalizeProviderName(%q) = %q, expected %q", tt.input, result, tt.expected)
		}
	}
}
