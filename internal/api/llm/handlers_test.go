package llmhandlers

import (
	"testing"
)

func TestNormalizeProviderName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"ollama", "ollama"},
		{"llama-cpp", "llama-server"},
		{"llamacpp", "llama-server"},
		{"llama_cpp", "llama-server"},
		{"llama-server", "llama-server"},
		{"java-llama-cpp", "llama-server"},
		{"unknown", "llama-server"},
		{"", "llama-server"},
	}

	for _, tt := range tests {
		result := normalizeProviderName(tt.input)
		if result != tt.expected {
			t.Errorf("normalizeProviderName(%q) = %q, expected %q", tt.input, result, tt.expected)
		}
	}
}

func TestIsLlamaCppProvider(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"llama-cpp", true},
		{"java-llama-cpp", true},
		{"llama-server", true},
		{"ollama", false},
		{"openai", false},
		{"", false},
	}

	for _, tt := range tests {
		result := isLlamaCppProvider(tt.input)
		if result != tt.expected {
			t.Errorf("isLlamaCppProvider(%q) = %v, expected %v", tt.input, result, tt.expected)
		}
	}
}

func TestGetProviderDisplayName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"ollama", "Ollama"},
		{"llama-server", "llama-server (lokal)"},
		{"llama-cpp", "llama-server (lokal)"},
		{"java-llama-cpp", "llama-server (lokal)"},
		{"unknown", "unknown"},
	}

	for _, tt := range tests {
		result := getProviderDisplayName(tt.input)
		if result != tt.expected {
			t.Errorf("getProviderDisplayName(%q) = %q, expected %q", tt.input, result, tt.expected)
		}
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		input    int64
		expected string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{1073741824, "1.0 GB"},
		{5368709120, "5.0 GB"},
	}

	for _, tt := range tests {
		result := formatBytes(tt.input)
		if result != tt.expected {
			t.Errorf("formatBytes(%d) = %q, expected %q", tt.input, result, tt.expected)
		}
	}
}
