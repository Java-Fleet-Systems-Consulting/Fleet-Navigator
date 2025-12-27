// Package chathandlers enthält die HTTP-Handler für Chat-CRUD-Endpoints
// HINWEIS: Der komplexe handleChatSendStream bleibt in main.go wegen vieler Abhängigkeiten
package chathandlers

import (
	"log"
	"net/http"
	"strconv"
	"strings"

	"fleet-navigator/internal/api/common"
	"fleet-navigator/internal/chat"
)

// Handlers enthält die HTTP-Handler für Chat-Endpoints
type Handlers struct {
	store         *chat.Store
	selectedModel string
}

// NewHandlers erstellt neue Chat-Handler
func NewHandlers(store *chat.Store, selectedModel string) *Handlers {
	return &Handlers{
		store:         store,
		selectedModel: selectedModel,
	}
}

// SetSelectedModel aktualisiert das ausgewählte Modell
func (h *Handlers) SetSelectedModel(model string) {
	h.selectedModel = model
}

// RegisterRoutes registriert die Chat-API-Routen
// HINWEIS: /api/chat/send-stream wird NICHT hier registriert (bleibt in main.go)
func (h *Handlers) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/api/chat/new", h.handleNew)
	mux.HandleFunc("/api/chat/all", h.handleAll)
	mux.HandleFunc("/api/chat/history/", h.handleHistory)
	// /api/chat/{id} wird weiterhin in main.go behandelt wegen Komplexität
}

// handleNew - POST /api/chat/new
func (h *Handlers) handleNew(w http.ResponseWriter, r *http.Request) {
	if !common.RequirePOST(w, r) {
		return
	}

	var req struct {
		Title string `json:"title"`
		Model string `json:"model"`
	}

	if err := common.DecodeJSON(r, &req); err != nil {
		// Wenn kein Body, Default-Werte verwenden
		req.Title = "Neuer Chat"
		req.Model = h.selectedModel
	}

	if req.Title == "" {
		req.Title = "Neuer Chat"
	}
	if req.Model == "" {
		req.Model = h.selectedModel
	}

	chatObj, err := h.store.CreateChat(req.Title, req.Model)
	if err != nil {
		common.WriteInternalError(w, err, "Chat konnte nicht erstellt werden")
		return
	}

	common.WriteJSON(w, http.StatusCreated, chatObj)
}

// handleAll - GET /api/chat/all
func (h *Handlers) handleAll(w http.ResponseWriter, r *http.Request) {
	if !common.RequireGET(w, r) {
		return
	}

	chats, err := h.store.GetAllChats()
	if err != nil {
		common.WriteInternalError(w, err, "Chats konnten nicht geladen werden")
		return
	}

	common.WriteJSON(w, http.StatusOK, chats)
}

// handleHistory - GET /api/chat/history/{id}
func (h *Handlers) handleHistory(w http.ResponseWriter, r *http.Request) {
	if !common.RequireGET(w, r) {
		return
	}

	idStr := strings.TrimPrefix(r.URL.Path, "/api/chat/history/")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		common.WriteBadRequest(w, "Invalid chat ID")
		return
	}

	chatObj, err := h.store.GetChat(id)
	if err != nil {
		common.WriteInternalError(w, err, "Chat konnte nicht geladen werden")
		return
	}
	if chatObj == nil {
		common.WriteNotFound(w, "Chat not found")
		return
	}

	common.WriteJSON(w, http.StatusOK, chatObj)
}

// HandleByID behandelt /api/chat/{id} mit allen Sub-Endpoints
// Kann von main.go aufgerufen werden
func (h *Handlers) HandleByID(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/api/chat/")

	// Überspringen wenn es ein reservierter Endpoint ist
	if path == "" || path == "new" || path == "all" || path == "send-stream" ||
		(len(path) >= 7 && path[:7] == "history") {
		return
	}

	parts := strings.Split(path, "/")
	if len(parts) == 0 {
		common.WriteBadRequest(w, "Invalid path")
		return
	}

	id, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		common.WriteBadRequest(w, "Invalid chat ID")
		return
	}

	var subEndpoint string
	var subID int64
	if len(parts) > 1 {
		subEndpoint = parts[1]
		if len(parts) > 2 {
			subID, _ = strconv.ParseInt(parts[2], 10, 64)
		}
	}

	switch subEndpoint {
	case "model":
		h.handleModel(w, r, id)
	case "rename":
		h.handleRename(w, r, id)
	case "expert":
		h.handleExpert(w, r, id)
	case "messages":
		h.handleMessages(w, r, id, subID)
	case "fork":
		h.handleFork(w, r, id)
	case "export":
		h.handleExport(w, r, id)
	default:
		h.handleBaseCRUD(w, r, id)
	}
}

func (h *Handlers) handleModel(w http.ResponseWriter, r *http.Request, id int64) {
	if r.Method != http.MethodPatch {
		common.WriteMethodNotAllowed(w)
		return
	}

	var req struct {
		Model string `json:"model"`
	}
	if err := common.DecodeJSON(r, &req); err != nil {
		common.WriteBadRequest(w, "Invalid JSON")
		return
	}
	if req.Model == "" {
		common.WriteBadRequest(w, "Model is required")
		return
	}

	if err := h.store.UpdateChat(id, nil, &req.Model); err != nil {
		common.WriteInternalError(w, err, "Model konnte nicht aktualisiert werden")
		return
	}

	common.WriteJSON(w, http.StatusOK, map[string]interface{}{"status": "updated", "model": req.Model})
}

func (h *Handlers) handleRename(w http.ResponseWriter, r *http.Request, id int64) {
	if r.Method != http.MethodPatch {
		common.WriteMethodNotAllowed(w)
		return
	}

	var req struct {
		NewTitle string `json:"newTitle"`
	}
	if err := common.DecodeJSON(r, &req); err != nil {
		common.WriteBadRequest(w, "Invalid JSON")
		return
	}
	if req.NewTitle == "" {
		common.WriteBadRequest(w, "newTitle is required")
		return
	}

	log.Printf("Renaming chat %d to: %s", id, req.NewTitle)
	chatObj, err := h.store.RenameChat(id, req.NewTitle)
	if err != nil {
		common.WriteInternalError(w, err, "Chat konnte nicht umbenannt werden")
		return
	}
	if chatObj == nil {
		common.WriteNotFound(w, "Chat not found")
		return
	}

	common.WriteJSON(w, http.StatusOK, chatObj)
}

func (h *Handlers) handleExpert(w http.ResponseWriter, r *http.Request, id int64) {
	if r.Method != http.MethodPatch {
		common.WriteMethodNotAllowed(w)
		return
	}

	var req struct {
		ExpertID *int64 `json:"expertId"`
	}
	if err := common.DecodeJSON(r, &req); err != nil {
		common.WriteBadRequest(w, "Invalid JSON")
		return
	}

	log.Printf("Updating chat %d expert to: %v", id, req.ExpertID)
	if err := h.store.UpdateChatExpert(id, req.ExpertID); err != nil {
		common.WriteInternalError(w, err, "Expert konnte nicht aktualisiert werden")
		return
	}

	common.WriteJSON(w, http.StatusOK, map[string]interface{}{"status": "updated", "expertId": req.ExpertID})
}

func (h *Handlers) handleMessages(w http.ResponseWriter, r *http.Request, chatID, messageID int64) {
	if r.Method != http.MethodDelete || messageID == 0 {
		common.WriteError(w, http.StatusMethodNotAllowed, "Method not allowed or missing message ID")
		return
	}

	log.Printf("Deleting message %d from chat %d", messageID, chatID)
	if err := h.store.DeleteMessage(chatID, messageID); err != nil {
		common.WriteInternalError(w, err, "Nachricht konnte nicht gelöscht werden")
		return
	}

	common.WriteJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *Handlers) handleFork(w http.ResponseWriter, r *http.Request, id int64) {
	if r.Method != http.MethodPost {
		common.WriteMethodNotAllowed(w)
		return
	}

	var req struct {
		NewTitle string `json:"newTitle"`
	}
	_ = common.DecodeJSON(r, &req) // Optional body

	original, err := h.store.GetChat(id)
	if err != nil {
		common.WriteInternalError(w, err, "Chat konnte nicht geladen werden")
		return
	}
	if original == nil {
		common.WriteNotFound(w, "Chat not found")
		return
	}

	forkTitle := req.NewTitle
	if forkTitle == "" {
		forkTitle = original.Title + " (Fork)"
	}

	log.Printf("Forking chat %d to: %s", id, forkTitle)
	forkedChat, err := h.store.ForkChat(id, forkTitle)
	if err != nil {
		common.WriteInternalError(w, err, "Chat konnte nicht geforkt werden")
		return
	}

	common.WriteJSON(w, http.StatusOK, forkedChat)
}

func (h *Handlers) handleExport(w http.ResponseWriter, r *http.Request, id int64) {
	if r.Method != http.MethodGet {
		common.WriteMethodNotAllowed(w)
		return
	}

	log.Printf("Exporting chat %d", id)
	export, err := h.store.ExportChat(id)
	if err != nil {
		common.WriteInternalError(w, err, "Chat konnte nicht exportiert werden")
		return
	}

	common.WriteJSON(w, http.StatusOK, export)
}

func (h *Handlers) handleBaseCRUD(w http.ResponseWriter, r *http.Request, id int64) {
	switch r.Method {
	case http.MethodGet:
		chatObj, err := h.store.GetChat(id)
		if err != nil {
			common.WriteInternalError(w, err, "Chat konnte nicht geladen werden")
			return
		}
		if chatObj == nil {
			common.WriteNotFound(w, "Chat not found")
			return
		}
		common.WriteJSON(w, http.StatusOK, chatObj)

	case http.MethodDelete:
		if err := h.store.DeleteChat(id); err != nil {
			common.WriteInternalError(w, err, "Chat konnte nicht gelöscht werden")
			return
		}
		common.WriteJSON(w, http.StatusOK, map[string]string{"status": "deleted"})

	case http.MethodPut, http.MethodPatch:
		var req struct {
			Title *string `json:"title"`
			Model *string `json:"model"`
		}
		if err := common.DecodeJSON(r, &req); err != nil {
			common.WriteBadRequest(w, "Invalid JSON")
			return
		}
		if err := h.store.UpdateChat(id, req.Title, req.Model); err != nil {
			common.WriteInternalError(w, err, "Chat konnte nicht aktualisiert werden")
			return
		}
		common.WriteJSON(w, http.StatusOK, map[string]string{"status": "updated"})

	default:
		common.WriteMethodNotAllowed(w)
	}
}
