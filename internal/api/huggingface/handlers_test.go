package huggingface

import (
	"testing"
)

func TestExtractParamSize(t *testing.T) {
	tests := []struct {
		modelName string
		tags      []string
		expected  string
	}{
		{"TheBloke/Llama-2-7B-GGUF", []string{}, "7B"},
		{"TheBloke/Llama-2-13B-GGUF", []string{}, "13B"},
		{"TheBloke/Llama-2-70B-GGUF", []string{}, "70B"},
		{"qwen2.5-1.5b-instruct-gguf", []string{}, "1.5B"},
		{"SomeModel-GGUF", []string{"7B", "text-generation"}, "7B"},
		{"UnknownModel", []string{}, ""},
	}

	for _, tt := range tests {
		result := extractParamSize(tt.modelName, tt.tags)
		if result != tt.expected {
			t.Errorf("extractParamSize(%q, %v) = %q, expected %q", tt.modelName, tt.tags, result, tt.expected)
		}
	}
}

func TestEstimateModelSize(t *testing.T) {
	tests := []struct {
		modelName  string
		paramSize  string
		expectSize bool
		expectGB   float64
	}{
		{"model-q4_k_m", "7B", true, 3.5},
		{"model-q8_0", "7B", true, 7.0},
		{"model-f16", "7B", true, 14.0},
		{"model", "0", false, 0},
		{"model", "", false, 0},
	}

	for _, tt := range tests {
		sizeBytes, sizeHuman := estimateModelSize(tt.modelName, tt.paramSize)
		if tt.expectSize {
			if sizeBytes == 0 {
				t.Errorf("estimateModelSize(%q, %q) returned 0 bytes", tt.modelName, tt.paramSize)
			}
			if sizeHuman == "" {
				t.Errorf("estimateModelSize(%q, %q) returned empty size string", tt.modelName, tt.paramSize)
			}
		} else {
			if sizeBytes != 0 {
				t.Errorf("estimateModelSize(%q, %q) = %d, expected 0", tt.modelName, tt.paramSize, sizeBytes)
			}
		}
	}
}

func TestDetectCategory(t *testing.T) {
	tests := []struct {
		tags     []string
		expected string
	}{
		{[]string{"text-generation", "code"}, "code"},
		{[]string{"coder", "python"}, "code"},
		{[]string{"vision", "image"}, "vision"},
		{[]string{"llava", "multimodal"}, "vision"},
		{[]string{"text-generation"}, "chat"},
		{[]string{}, "chat"},
	}

	for _, tt := range tests {
		result := detectCategory(tt.tags)
		if result != tt.expected {
			t.Errorf("detectCategory(%v) = %q, expected %q", tt.tags, result, tt.expected)
		}
	}
}
