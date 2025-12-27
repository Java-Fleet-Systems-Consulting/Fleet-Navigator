package mateshandlers

import (
	"testing"
)

func TestHandlersInitialization(t *testing.T) {
	h := NewHandlers(nil, nil)

	if h == nil {
		t.Error("NewHandlers should not return nil")
	}
}
