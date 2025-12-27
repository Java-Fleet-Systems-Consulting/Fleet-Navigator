package visionhandlers

import (
	"testing"
)

func TestLoadImageAsBase64(t *testing.T) {
	// Test mit nicht existierender Datei
	_, err := loadImageAsBase64("/non/existent/path.png")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}

func TestHandlersInitialization(t *testing.T) {
	h := NewHandlers(nil, "/tmp/test")

	if h == nil {
		t.Error("NewHandlers should not return nil")
	}

	if h.dataDir != "/tmp/test" {
		t.Errorf("DataDir should be /tmp/test, got %s", h.dataDir)
	}
}
