package voicehandlers

import (
	"testing"
)

func TestHandlersInitialization(t *testing.T) {
	h := NewHandlers(nil)

	if h == nil {
		t.Error("NewHandlers should not return nil")
	}

	if h.settings != nil {
		t.Error("Settings should be nil initially")
	}
}
