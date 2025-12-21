package mate

import (
	"fmt"
	"log"
	"sync"
	"time"
)

// Handler verarbeitet Mate-spezifische Anfragen
type Handler struct {
	mu       sync.RWMutex
	handlers map[string]ActionHandler

	// Callback für Chat-Anfragen
	OnChatRequest func(mateID, sessionID, message string, onChunk func(string)) (string, error)
}

// ActionHandler ist eine Funktion die eine Mate-Aktion verarbeitet
type ActionHandler func(req *MateRequest) (*MateResponse, error)

// NewHandler erstellt einen neuen Mate-Handler
func NewHandler() *Handler {
	h := &Handler{
		handlers: make(map[string]ActionHandler),
	}

	// Standard-Handler registrieren
	h.RegisterHandler("ping", h.handlePing)
	h.RegisterHandler("get_info", h.handleGetInfo)
	h.RegisterHandler("chat", h.handleChat)

	return h
}

// RegisterHandler registriert einen Handler für eine Aktion
func (h *Handler) RegisterHandler(action string, handler ActionHandler) {
	h.mu.Lock()
	h.handlers[action] = handler
	h.mu.Unlock()
}

// HandleRequest verarbeitet eine Mate-Anfrage
func (h *Handler) HandleRequest(req *MateRequest) (*MateResponse, error) {
	h.mu.RLock()
	handler, ok := h.handlers[req.Action]
	h.mu.RUnlock()

	if !ok {
		return &MateResponse{
			RequestID: req.ID,
			Success:   false,
			Error:     fmt.Sprintf("Unknown action: %s", req.Action),
			Timestamp: time.Now(),
		}, nil
	}

	log.Printf("Mate %s: Handling action '%s'", req.MateID, req.Action)
	return handler(req)
}

// --- Standard Handler ---

func (h *Handler) handlePing(req *MateRequest) (*MateResponse, error) {
	return &MateResponse{
		RequestID: req.ID,
		Success:   true,
		Data: map[string]interface{}{
			"pong":      true,
			"timestamp": time.Now().UnixMilli(),
		},
		Timestamp: time.Now(),
	}, nil
}

func (h *Handler) handleGetInfo(req *MateRequest) (*MateResponse, error) {
	mateTypes := GetAllMateTypes()

	return &MateResponse{
		RequestID: req.ID,
		Success:   true,
		Data: map[string]interface{}{
			"navigator_version": "1.0.0",
			"mate_types":        mateTypes,
		},
		Timestamp: time.Now(),
	}, nil
}

func (h *Handler) handleChat(req *MateRequest) (*MateResponse, error) {
	if h.OnChatRequest == nil {
		return &MateResponse{
			RequestID: req.ID,
			Success:   false,
			Error:     "Chat handler not configured",
			Timestamp: time.Now(),
		}, nil
	}

	// Message aus Payload extrahieren
	message, ok := req.Payload["message"].(string)
	if !ok || message == "" {
		return &MateResponse{
			RequestID: req.ID,
			Success:   false,
			Error:     "Missing or empty message",
			Timestamp: time.Now(),
		}, nil
	}

	sessionID, _ := req.Payload["session_id"].(string)
	if sessionID == "" {
		sessionID = req.MateID // Verwende Mate-ID als Session-ID
	}

	// Chat ausführen (synchron, ohne Streaming für diese Variante)
	response, err := h.OnChatRequest(req.MateID, sessionID, message, nil)
	if err != nil {
		return &MateResponse{
			RequestID: req.ID,
			Success:   false,
			Error:     err.Error(),
			Timestamp: time.Now(),
		}, nil
	}

	return &MateResponse{
		RequestID: req.ID,
		Success:   true,
		Data: map[string]interface{}{
			"session_id": sessionID,
			"response":   response,
		},
		Timestamp: time.Now(),
	}, nil
}

// --- Spezifische Aktionen für Mate-Typen ---

// RegisterWriterHandlers registriert Writer-spezifische Handler
func (h *Handler) RegisterWriterHandlers() {
	h.RegisterHandler("document.analyze", func(req *MateRequest) (*MateResponse, error) {
		// Dokument-Analyse anfordern
		content, ok := req.Payload["content"].(string)
		if !ok {
			return &MateResponse{
				RequestID: req.ID,
				Success:   false,
				Error:     "Missing document content",
				Timestamp: time.Now(),
			}, nil
		}

		// Hier würde die KI-Analyse stattfinden
		log.Printf("Document analysis requested by %s: %d chars", req.MateID, len(content))

		return &MateResponse{
			RequestID: req.ID,
			Success:   true,
			Data: map[string]interface{}{
				"status":  "analysis_pending",
				"message": "Document analysis initiated",
			},
			Timestamp: time.Now(),
		}, nil
	})

	h.RegisterHandler("document.suggest", func(req *MateRequest) (*MateResponse, error) {
		// Vorschläge für Dokument generieren
		return &MateResponse{
			RequestID: req.ID,
			Success:   true,
			Data: map[string]interface{}{
				"suggestions": []string{
					"Tipp: Verwenden Sie aktive Verben",
					"Tipp: Vermeiden Sie lange Schachtelsätze",
				},
			},
			Timestamp: time.Now(),
		}, nil
	})
}

// RegisterMailHandlers registriert Mail-spezifische Handler
func (h *Handler) RegisterMailHandlers() {
	h.RegisterHandler("email.analyze", func(req *MateRequest) (*MateResponse, error) {
		// E-Mail-Analyse
		subject, _ := req.Payload["subject"].(string)
		body, _ := req.Payload["body"].(string)

		log.Printf("Email analysis requested: Subject='%s', Body=%d chars", subject, len(body))

		return &MateResponse{
			RequestID: req.ID,
			Success:   true,
			Data: map[string]interface{}{
				"status": "analysis_pending",
			},
			Timestamp: time.Now(),
		}, nil
	})

	h.RegisterHandler("email.draft_reply", func(req *MateRequest) (*MateResponse, error) {
		// Antwort-Entwurf erstellen
		return &MateResponse{
			RequestID: req.ID,
			Success:   true,
			Data: map[string]interface{}{
				"status": "draft_pending",
			},
			Timestamp: time.Now(),
		}, nil
	})

	h.RegisterHandler("appointment.check", func(req *MateRequest) (*MateResponse, error) {
		// Terminanfrage prüfen
		return &MateResponse{
			RequestID: req.ID,
			Success:   true,
			Data: map[string]interface{}{
				"is_appointment_request": false, // Placeholder
			},
			Timestamp: time.Now(),
		}, nil
	})
}

// RegisterWebSearchHandlers registriert Web-Search-spezifische Handler
func (h *Handler) RegisterWebSearchHandlers() {
	h.RegisterHandler("web.search", func(req *MateRequest) (*MateResponse, error) {
		query, ok := req.Payload["query"].(string)
		if !ok {
			return &MateResponse{
				RequestID: req.ID,
				Success:   false,
				Error:     "Missing search query",
				Timestamp: time.Now(),
			}, nil
		}

		log.Printf("Web search requested: '%s'", query)

		return &MateResponse{
			RequestID: req.ID,
			Success:   true,
			Data: map[string]interface{}{
				"status": "search_pending",
				"query":  query,
			},
			Timestamp: time.Now(),
		}, nil
	})

	h.RegisterHandler("web.fetch", func(req *MateRequest) (*MateResponse, error) {
		url, ok := req.Payload["url"].(string)
		if !ok {
			return &MateResponse{
				RequestID: req.ID,
				Success:   false,
				Error:     "Missing URL",
				Timestamp: time.Now(),
			}, nil
		}

		log.Printf("Web fetch requested: %s", url)

		return &MateResponse{
			RequestID: req.ID,
			Success:   true,
			Data: map[string]interface{}{
				"status": "fetch_pending",
				"url":    url,
			},
			Timestamp: time.Now(),
		}, nil
	})
}
