package chathandlers

import (
	"testing"
)

func TestSetSelectedModel(t *testing.T) {
	h := &Handlers{selectedModel: "qwen2.5:7b"}

	if h.selectedModel != "qwen2.5:7b" {
		t.Errorf("Initial model should be qwen2.5:7b, got %s", h.selectedModel)
	}

	h.SetSelectedModel("llama3.2:3b")

	if h.selectedModel != "llama3.2:3b" {
		t.Errorf("Model should be llama3.2:3b after update, got %s", h.selectedModel)
	}
}

func TestHandlersInitialization(t *testing.T) {
	h := NewHandlers(nil, "test-model")

	if h == nil {
		t.Error("NewHandlers should not return nil")
	}

	if h.selectedModel != "test-model" {
		t.Errorf("Selected model should be test-model, got %s", h.selectedModel)
	}
}
