// Package websocket implementiert den WebSocket-Server für Mate-Kommunikation
package websocket

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"

	"fleet-navigator/internal/security"
	"fleet-navigator/internal/tools"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// In Produktion: Origin prüfen!
		return true
	},
}

// MessageType definiert die Art der WebSocket-Nachricht
type MessageType string

const (
	// Pairing
	MsgPairingRequest  MessageType = "pairing_request"
	MsgPairingResponse MessageType = "pairing_response"
	MsgPairingApproved MessageType = "pairing_approved"
	MsgPairingRejected MessageType = "pairing_rejected"

	// Authentifizierung
	MsgAuth                 MessageType = "auth"
	MsgAuthChallengeRequest MessageType = "auth_challenge_request" // Thunderbird
	MsgAuthChallenge        MessageType = "auth_challenge"         // Navigator → Thunderbird
	MsgAuthSuccess          MessageType = "auth_success"
	MsgAuthFailed           MessageType = "auth_failed"

	// Chat
	MsgChat       MessageType = "chat"
	MsgChatStream MessageType = "chat_stream"
	MsgChatDone   MessageType = "chat_done"
	MsgChatError  MessageType = "chat_error"
	MsgChatClear  MessageType = "chat_clear"

	// Kommunikation
	MsgRequest  MessageType = "request"
	MsgResponse MessageType = "response"
	MsgEvent    MessageType = "event"
	MsgError    MessageType = "error"

	// System
	MsgPing      MessageType = "ping"
	MsgPong      MessageType = "pong"
	MsgHeartbeat MessageType = "heartbeat"     // Thunderbird
	MsgHeartbeatAck MessageType = "heartbeat_ack" // Navigator → Thunderbird

	// Encrypted Messages (Thunderbird)
	MsgEncrypted MessageType = "encrypted"

	// Email Classification (Thunderbird Email-Mate)
	MsgClassifyEmail    MessageType = "classify_email"
	MsgClassifyResponse MessageType = "classify_response"
	MsgGenerateReply    MessageType = "generate_reply"
	MsgReplyGenerated   MessageType = "reply_generated"
	MsgStats            MessageType = "stats"
)

// Message ist das Standard-Nachrichtenformat
type Message struct {
	Type      MessageType     `json:"type"`
	ID        string          `json:"id,omitempty"`
	Payload   json.RawMessage `json:"payload,omitempty"`
	MateID    string          `json:"mateId,omitempty"` // Für Thunderbird encrypted messages
	Encrypted bool            `json:"encrypted,omitempty"`
	Timestamp int64           `json:"timestamp"`
}

// PairingRequestPayload für Pairing-Anfragen
// Unterstützt sowohl camelCase (Thunderbird) als auch snake_case
type PairingRequestPayload struct {
	MateName        string `json:"mateName"`
	MateType        string `json:"mateType"`
	MatePublicKey   string `json:"matePublicKey"`
	MateExchangeKey string `json:"mateExchangeKey,omitempty"` // X25519 für Thunderbird ECDH
}

// PairingResponsePayload für Pairing-Antworten
// Angepasst für Thunderbird-Kompatibilität (camelCase)
type PairingResponsePayload struct {
	Status              string `json:"status"`                        // "pending", "approved", "rejected"
	RequestID           string `json:"requestId,omitempty"`
	NavigatorPublicKey  string `json:"navigatorPublicKey"`
	NavigatorExchangeKey string `json:"navigatorExchangeKey,omitempty"` // X25519 für Thunderbird
	PairingCode         string `json:"pairingCode"`
}

// AuthPayload für Authentifizierung
// Unterstützt beide Formate (snake_case und camelCase)
type AuthPayload struct {
	MateID    string `json:"mateId"`
	PublicKey string `json:"publicKey"`
	Signature string `json:"signature"` // Signierte Nonce
	Nonce     string `json:"nonce"`
}

// ChatPayload für Chat-Nachrichten
type ChatPayload struct {
	SessionID string `json:"session_id,omitempty"`
	Message   string `json:"message"`
	Stream    bool   `json:"stream,omitempty"`
}

// ChatResponsePayload für Chat-Antworten
type ChatResponsePayload struct {
	SessionID string `json:"session_id"`
	Content   string `json:"content"`
	Done      bool   `json:"done,omitempty"`
}

// Client repräsentiert eine WebSocket-Verbindung
type Client struct {
	ID            string
	Conn          *websocket.Conn
	Server        *Server
	Send          chan []byte
	Authenticated bool
	MateID        string
	SecureChannel *security.SecureChannel
	mu            sync.Mutex
}

// ChatHandler definiert das Interface für Chat-Verarbeitung
type ChatHandler interface {
	Chat(sessionID, message string, onChunk func(chunk string)) (string, error)
	ChatWithSystemPrompt(sessionID, message, systemPrompt string, onChunk func(chunk string)) (string, error)
	ClearHistory(sessionID string) error
}

// MateStats speichert Hardware-Stats von einem Mate
type MateStats struct {
	System      map[string]interface{} `json:"system,omitempty"`
	CPU         map[string]interface{} `json:"cpu,omitempty"`
	Memory      map[string]interface{} `json:"memory,omitempty"`
	GPU         []map[string]interface{} `json:"gpu,omitempty"`
	Temperature map[string]interface{} `json:"temperature,omitempty"`
	RemoteIP    string                  `json:"remoteIp,omitempty"`
	UpdatedAt   time.Time               `json:"updatedAt"`
}

// Server ist der WebSocket-Server
type Server struct {
	clients        map[*Client]bool
	clientsByMate  map[string]*Client
	mateStats      map[string]*MateStats // Stats pro Mate
	broadcast      chan []byte
	register       chan *Client
	unregister     chan *Client
	pairingManager *security.PairingManager
	chatHandler    ChatHandler
	mu             sync.RWMutex

	// Callbacks für UI
	OnPairingRequest   func(req *security.PairingRequest)
	OnMateConnected    func(mateID, mateName string)
	OnMateDisconnected func(mateID, mateName string)
}

// NewServer erstellt einen neuen WebSocket-Server
func NewServer(pairingManager *security.PairingManager) *Server {
	s := &Server{
		clients:        make(map[*Client]bool),
		clientsByMate:  make(map[string]*Client),
		mateStats:      make(map[string]*MateStats),
		broadcast:      make(chan []byte, 256),
		register:       make(chan *Client),
		unregister:     make(chan *Client),
		pairingManager: pairingManager,
	}

	// Pairing-Callbacks verknüpfen
	pairingManager.OnPairingRequest = func(req *security.PairingRequest) {
		if s.OnPairingRequest != nil {
			s.OnPairingRequest(req)
		}
	}

	return s
}

// SetChatHandler setzt den Chat-Handler
func (s *Server) SetChatHandler(handler ChatHandler) {
	s.chatHandler = handler
}

// Run startet die Server-Hauptschleife
func (s *Server) Run() {
	for {
		select {
		case client := <-s.register:
			s.mu.Lock()
			s.clients[client] = true
			s.mu.Unlock()
			log.Printf("Client verbunden: %s", client.ID)

		case client := <-s.unregister:
			s.mu.Lock()
			if _, ok := s.clients[client]; ok {
				delete(s.clients, client)
				if client.MateID != "" {
					delete(s.clientsByMate, client.MateID)
					if s.OnMateDisconnected != nil {
						s.OnMateDisconnected(client.MateID, "")
					}
				}
				close(client.Send)
			}
			s.mu.Unlock()
			log.Printf("Client getrennt: %s", client.ID)

		case message := <-s.broadcast:
			s.mu.RLock()
			for client := range s.clients {
				select {
				case client.Send <- message:
				default:
					close(client.Send)
					delete(s.clients, client)
				}
			}
			s.mu.RUnlock()
		}
	}
}

// HandleWebSocket behandelt neue WebSocket-Verbindungen
func (s *Server) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket Upgrade Fehler: %v", err)
		return
	}

	client := &Client{
		ID:     security.GenerateRandomID(8),
		Conn:   conn,
		Server: s,
		Send:   make(chan []byte, 256),
	}

	s.register <- client

	go client.writePump()
	go client.readPump()
}

// readPump liest Nachrichten vom Client
func (c *Client) readPump() {
	defer func() {
		c.Server.unregister <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadLimit(512 * 1024) // 512KB max
	c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, data, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("WebSocket Fehler: %v", err)
			}
			break
		}

		c.handleMessage(data)
	}
}

// writePump sendet Nachrichten an den Client
func (c *Client) writePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleMessage verarbeitet eingehende Nachrichten
func (c *Client) handleMessage(data []byte) {
	var msg Message
	if err := json.Unmarshal(data, &msg); err != nil {
		c.sendError("Ungültiges Nachrichtenformat")
		return
	}

	// Debug: Log all incoming message types
	if msg.Type != MsgHeartbeat && msg.Type != MsgPing {
		log.Printf("📩 WebSocket message received: type=%s, client=%s, authenticated=%v", msg.Type, c.ID, c.Authenticated)
	}

	switch msg.Type {
	case MsgPairingRequest:
		c.handlePairingRequest(msg)

	case MsgAuthChallengeRequest:
		c.handleAuthChallengeRequest(msg)

	case MsgAuth:
		c.handleAuth(msg)

	case MsgHeartbeat:
		c.handleHeartbeat(msg)

	case MsgEncrypted:
		log.Printf("🔐 Encrypted message received from client %s", c.ID)
		c.handleEncryptedMessage(msg)

	case MsgChat:
		c.handleChat(msg)

	case MsgChatClear:
		c.handleChatClear(msg)

	case MsgRequest:
		if !c.Authenticated {
			c.sendError("Nicht authentifiziert")
			return
		}
		c.handleRequest(msg)

	case MsgPing:
		c.sendMessage(MsgPong, nil)

	default:
		log.Printf("Unbekannter Nachrichtentyp: %s", msg.Type)
		// Nicht als Fehler senden - könnte legitime Nachrichten sein die wir noch nicht unterstützen
	}
}

// handlePairingRequest verarbeitet eine Pairing-Anfrage
func (c *Client) handlePairingRequest(msg Message) {
	var payload PairingRequestPayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		c.sendError("Ungültige Pairing-Anfrage")
		return
	}

	log.Printf("Pairing-Anfrage von %s (Typ: %s, ExchangeKey: %v)", payload.MateName, payload.MateType, payload.MateExchangeKey != "")

	// Mit Exchange Key für X25519 ECDH initialisieren
	req, err := c.Server.pairingManager.InitiatePairingWithExchangeKey(
		payload.MateName,
		payload.MateType,
		payload.MatePublicKey,
		payload.MateExchangeKey, // X25519 Exchange Key vom Mate
	)
	if err != nil {
		c.sendError(err.Error())
		return
	}

	// Thunderbird-kompatibles Response-Format mit X25519 Exchange Key
	response := PairingResponsePayload{
		Status:               "pending",
		RequestID:            req.ID,
		NavigatorPublicKey:   c.Server.pairingManager.GetPublicKey(),
		NavigatorExchangeKey: c.Server.pairingManager.GetExchangePublicKey(), // Echter X25519 Exchange Key!
		PairingCode:          req.PairingCode,
	}

	c.sendMessage(MsgPairingResponse, response)
}

// handleAuth verarbeitet Authentifizierung
func (c *Client) handleAuth(msg Message) {
	var payload AuthPayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		c.sendMessage(MsgAuthFailed, map[string]string{"error": "Ungültige Auth-Anfrage"})
		return
	}

	// Prüfen ob Mate vertraut ist
	mate, trusted := c.Server.pairingManager.IsTrusted(payload.PublicKey)
	if !trusted {
		c.sendMessage(MsgAuthFailed, map[string]string{"error": "Mate nicht vertraut"})
		return
	}

	// Signatur prüfen (TODO: Vollständige Implementierung)
	// Hier würde man die Signatur der Nonce mit dem Public Key verifizieren

	c.Authenticated = true
	c.MateID = mate.ID

	// SecureChannel für verschlüsselte Kommunikation initialisieren
	if mate.SharedSecret != "" {
		secureChannel, err := c.Server.pairingManager.GetSecureChannelForMate(mate.ID)
		if err != nil {
			log.Printf("⚠️ SecureChannel konnte nicht erstellt werden: %v", err)
		} else {
			c.SecureChannel = secureChannel
			log.Printf("🔐 SecureChannel für %s initialisiert", mate.Name)
		}
	}

	c.Server.mu.Lock()
	c.Server.clientsByMate[mate.ID] = c
	c.Server.mu.Unlock()

	// Remote-IP speichern
	if c.Conn != nil {
		remoteAddr := c.Conn.RemoteAddr().String()
		// IP ohne Port extrahieren
		if host, _, err := net.SplitHostPort(remoteAddr); err == nil {
			c.Server.SetMateRemoteIP(mate.ID, host)
			log.Printf("📍 Remote-IP für %s: %s", mate.Name, host)
		}
	}

	c.Server.pairingManager.UpdateLastSeen(mate.ID)

	if c.Server.OnMateConnected != nil {
		c.Server.OnMateConnected(mate.ID, mate.Name)
	}

	c.sendMessage(MsgAuthSuccess, map[string]string{
		"mate_id":   mate.ID,
		"mate_name": mate.Name,
	})

	log.Printf("Mate authentifiziert: %s (%s)", mate.Name, mate.ID)
}

// handleRequest verarbeitet normale Anfragen (nach Authentifizierung)
func (c *Client) handleRequest(msg Message) {
	// Hier kommen die eigentlichen Mate-Anfragen hin
	// z.B. Ollama-Abfragen, Experten-Aufrufe, etc.
	log.Printf("Request von Mate %s: %s", c.MateID, string(msg.Payload))

	// Placeholder-Response
	c.sendMessageWithID(MsgResponse, msg.ID, map[string]string{
		"status": "received",
	})
}

// handleChat verarbeitet Chat-Nachrichten
func (c *Client) handleChat(msg Message) {
	if c.Server.chatHandler == nil {
		c.sendMessage(MsgChatError, map[string]string{
			"error": "Chat-Service nicht verfügbar",
		})
		return
	}

	var payload ChatPayload
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		c.sendMessage(MsgChatError, map[string]string{
			"error": "Ungültige Chat-Anfrage",
		})
		return
	}

	// Session-ID generieren falls nicht vorhanden
	sessionID := payload.SessionID
	if sessionID == "" {
		sessionID = c.ID // Verwende Client-ID als Session-ID
	}

	// Mate-spezifischen System-Prompt und Modus holen
	var mateSystemPrompt string
	var currentMode string
	var mateType string
	if c.MateID != "" {
		_, mateSystemPrompt, currentMode, _ = c.Server.pairingManager.GetMateConfig(c.MateID)
		// Mate-Typ ermitteln
		if mate, ok := c.Server.pairingManager.GetTrustedMateByID(c.MateID); ok {
			mateType = mate.Type
		}
	}

	log.Printf("Chat-Anfrage von %s (Typ: %s): %s", c.ID, mateType, payload.Message)

	// Für Coder-Mates: Automatische Modus-Erkennung
	if mateType == "coder" {
		// 1. Prüfe zuerst auf expliziten Modus-Wechsel-Befehl (höchste Priorität)
		if explicitMode, isExplicit := DetectExplicitModeSwitch(payload.Message); isExplicit {
			mode := GetCoderMode(explicitMode)
			if mode != nil {
				log.Printf("🎯 Expliziter Modus-Wechsel: %s %s", mode.Icon, mode.Name)

				// Mode-Switch Event senden
				c.sendMessageWithID(MsgChatStream, msg.ID, ChatResponsePayload{
					SessionID: sessionID,
					Content:   fmt.Sprintf("\n%s **Wechsle zu %s-Modus...**\n\n", mode.Icon, mode.Name),
					Done:      false,
				})

				// Modus speichern
				go func() {
					if err := c.Server.pairingManager.UpdateMateConfig(c.MateID, "", mode.Prompt, mode.ID); err != nil {
						log.Printf("⚠️ Konnte Modus nicht speichern: %v", err)
					}
				}()

				// Setze den System-Prompt für den neuen Modus
				mateSystemPrompt = mode.Prompt

				// Prüfe ob nach dem Modus-Wechsel noch eine Aufgabe kommt
				// z.B. "geh in den bash modus und suche screenshot.png"
				hasFollowUpTask := strings.Contains(strings.ToLower(payload.Message), " und ") ||
					strings.Contains(strings.ToLower(payload.Message), " then ") ||
					strings.Contains(strings.ToLower(payload.Message), " suche ") ||
					strings.Contains(strings.ToLower(payload.Message), " finde ") ||
					strings.Contains(strings.ToLower(payload.Message), " zeig ") ||
					strings.Contains(strings.ToLower(payload.Message), " liste ") ||
					strings.Contains(strings.ToLower(payload.Message), " search ") ||
					strings.Contains(strings.ToLower(payload.Message), " find ")

				if !hasFollowUpTask {
					// Nur Modus-Wechsel ohne Folgeaufgabe → fertig
					c.sendMessageWithID(MsgChatStream, msg.ID, ChatResponsePayload{
						SessionID: sessionID,
						Content:   fmt.Sprintf("Ich bin jetzt im %s-Modus. Wie kann ich dir helfen?\n", mode.Language),
						Done:      false,
					})
					c.sendMessageWithID(MsgChatDone, msg.ID, ChatResponsePayload{
						SessionID: sessionID,
						Done:      true,
					})
					return
				}
				// Mit Folgeaufgabe → weiter zum LLM mit neuem Modus
				log.Printf("📝 Modus gewechselt, führe Folgeaufgabe aus...")
			}
		} else {
			// 2. Automatische Erkennung aus Nachrichteninhalt (nur wenn kein expliziter Wechsel)
			detection := DetectCoderMode(payload.Message, currentMode)

			if detection.Detected {
				if detection.Confident && detection.Mode != nil {
					// Sicherer Match → Modus wechseln
					if currentMode != detection.Mode.ID {
						log.Printf("🔄 Coder-Modus erkannt: %s %s (Match: %s)",
							detection.Mode.Icon, detection.Mode.Name, detection.MatchedOn)

						// Mode-Switch Event senden (wie bei Experten)
						c.sendMessageWithID(MsgChatStream, msg.ID, ChatResponsePayload{
							SessionID: sessionID,
							Content:   fmt.Sprintf("\n%s **Wechsle zu %s-Modus...**\n\n", detection.Mode.Icon, detection.Mode.Name),
							Done:      false,
						})

						// Modus-spezifischen Prompt verwenden
						mateSystemPrompt = detection.Mode.Prompt

						// Modus in Config speichern (für nächste Anfragen)
						go func() {
							if err := c.Server.pairingManager.UpdateMateConfig(c.MateID, "", mateSystemPrompt, detection.Mode.ID); err != nil {
								log.Printf("⚠️ Konnte Modus nicht speichern: %v", err)
							}
						}()
					} else {
						// Gleicher Modus, weiter verwenden
						if mode := GetCoderMode(currentMode); mode != nil {
							mateSystemPrompt = mode.Prompt
						}
					}
				} else if !detection.Confident && detection.AskQuestion != "" {
					// Unsicherer Match → Nachfragen!
					log.Printf("❓ Coder-Modus unsicher, frage nach: %s", detection.AskQuestion)
					c.sendMessageWithID(MsgChatStream, msg.ID, ChatResponsePayload{
						SessionID: sessionID,
						Content:   fmt.Sprintf("🤔 **Kurze Nachfrage:** %s\n\n", detection.AskQuestion),
						Done:      false,
					})
					c.sendMessageWithID(MsgChatDone, msg.ID, ChatResponsePayload{
						SessionID: sessionID,
						Done:      true,
					})
					return // Warte auf Antwort des Users
				}
			} else {
				// Kein neuer Modus erkannt
				if currentMode != "" && currentMode != "general" {
					// Gespeicherter Modus vorhanden → beibehalten
					if mode := GetCoderMode(currentMode); mode != nil {
						mateSystemPrompt = mode.Prompt
						log.Printf("💻 Kein neuer Modus erkannt, behalte %s %s", mode.Icon, mode.Name)
					}
				} else {
					// Kein Modus gespeichert → General Coder verwenden
					generalMode := GetCoderMode("general")
					if generalMode != nil {
						mateSystemPrompt = generalMode.Prompt
						log.Printf("💻 Kein Modus gespeichert, verwende General Coder")
					}
				}
			}
		}
	} else if mateSystemPrompt != "" {
		log.Printf("🎯 Verwende Mate-spezifischen System-Prompt für %s (%d Zeichen)", c.MateID, len(mateSystemPrompt))
	}

	// Chat mit Streaming-Callback
	go func() {
		var err error
		if mateSystemPrompt != "" {
			// Verwende Modus/Mate-spezifischen Prompt
			_, err = c.Server.chatHandler.ChatWithSystemPrompt(sessionID, payload.Message, mateSystemPrompt, func(chunk string) {
				c.sendMessageWithID(MsgChatStream, msg.ID, ChatResponsePayload{
					SessionID: sessionID,
					Content:   chunk,
					Done:      false,
				})
			})
		} else {
			// Verwende Standard-Prompt
			_, err = c.Server.chatHandler.Chat(sessionID, payload.Message, func(chunk string) {
				c.sendMessageWithID(MsgChatStream, msg.ID, ChatResponsePayload{
					SessionID: sessionID,
					Content:   chunk,
					Done:      false,
				})
			})
		}

		if err != nil {
			c.sendMessageWithID(MsgChatError, msg.ID, map[string]string{
				"session_id": sessionID,
				"error":      err.Error(),
			})
			return
		}

		// Fertig-Signal senden
		c.sendMessageWithID(MsgChatDone, msg.ID, ChatResponsePayload{
			SessionID: sessionID,
			Done:      true,
		})
	}()
}

// handleChatClear löscht den Chat-Verlauf
func (c *Client) handleChatClear(msg Message) {
	if c.Server.chatHandler == nil {
		c.sendMessage(MsgChatError, map[string]string{
			"error": "Chat-Service nicht verfügbar",
		})
		return
	}

	var payload struct {
		SessionID string `json:"session_id"`
	}
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		c.sendMessage(MsgChatError, map[string]string{
			"error": "Ungültige Anfrage",
		})
		return
	}

	sessionID := payload.SessionID
	if sessionID == "" {
		sessionID = c.ID
	}

	if err := c.Server.chatHandler.ClearHistory(sessionID); err != nil {
		c.sendMessage(MsgChatError, map[string]string{
			"error": err.Error(),
		})
		return
	}

	c.sendMessageWithID(MsgResponse, msg.ID, map[string]string{
		"status":     "cleared",
		"session_id": sessionID,
	})
}

// sendMessage sendet eine Nachricht an den Client
func (c *Client) sendMessage(msgType MessageType, payload interface{}) {
	c.sendMessageWithID(msgType, "", payload)
}

// sendMessageWithID sendet eine Nachricht mit ID
func (c *Client) sendMessageWithID(msgType MessageType, id string, payload interface{}) {
	var payloadJSON json.RawMessage
	if payload != nil {
		data, _ := json.Marshal(payload)
		payloadJSON = data
	}

	msg := Message{
		Type:      msgType,
		ID:        id,
		Payload:   payloadJSON,
		Timestamp: time.Now().UnixMilli(),
	}

	data, _ := json.Marshal(msg)

	c.mu.Lock()
	defer c.mu.Unlock()

	select {
	case c.Send <- data:
	default:
		log.Printf("Client %s: Send-Buffer voll", c.ID)
	}
}

// sendError sendet eine Fehlermeldung
func (c *Client) sendError(message string) {
	c.sendMessage(MsgError, map[string]string{"error": message})
}

// SendToMate sendet eine Nachricht an einen bestimmten Mate
func (s *Server) SendToMate(mateID string, msgType MessageType, payload interface{}) error {
	s.mu.RLock()
	client, ok := s.clientsByMate[mateID]
	s.mu.RUnlock()

	if !ok {
		return fmt.Errorf("Mate %s nicht verbunden", mateID)
	}

	client.sendMessage(msgType, payload)
	return nil
}

// BroadcastJSON sendet eine JSON-Nachricht an alle verbundenen Clients
// Wird für System-Events wie Wake Word Detection verwendet
func (s *Server) BroadcastJSON(data interface{}) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		log.Printf("BroadcastJSON: JSON Marshal Fehler: %v", err)
		return
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	for client := range s.clients {
		select {
		case client.Send <- jsonData:
		default:
			// Client-Buffer voll, überspringen
			log.Printf("BroadcastJSON: Client-Buffer voll, überspringe")
		}
	}
}

// GetConnectedMates gibt die IDs aller verbundenen Mates zurück
func (s *Server) GetConnectedMates() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	mates := make([]string, 0, len(s.clientsByMate))
	for mateID := range s.clientsByMate {
		mates = append(mates, mateID)
	}
	return mates
}

// GetMateStats gibt die gespeicherten Stats für einen Mate zurück
func (s *Server) GetMateStats(mateID string) *MateStats {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.mateStats[mateID]
}

// SetMateRemoteIP setzt die Remote-IP für einen Mate
func (s *Server) SetMateRemoteIP(mateID, remoteIP string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.mateStats[mateID] == nil {
		s.mateStats[mateID] = &MateStats{}
	}
	s.mateStats[mateID].RemoteIP = remoteIP
	s.mateStats[mateID].UpdatedAt = time.Now()
}

// DisconnectMate trennt einen Mate und benachrichtigt ihn
func (s *Server) DisconnectMate(mateID string) error {
	s.mu.Lock()
	client, ok := s.clientsByMate[mateID]
	s.mu.Unlock()

	if !ok {
		return fmt.Errorf("Mate %s nicht verbunden", mateID)
	}

	// Sende "unpaired" Nachricht an den Mate bevor wir trennen
	client.sendMessage(MsgPairingRejected, map[string]interface{}{
		"reason":  "unpaired",
		"message": "Du wurdest vom Navigator entfernt",
	})

	// Kurz warten damit die Nachricht ankommt
	time.Sleep(100 * time.Millisecond)

	// Verbindung schließen - dies triggert den unregister-Channel
	if client.Conn != nil {
		client.Conn.Close()
	}

	log.Printf("🔌 Mate %s wurde getrennt (unpaired)", mateID)
	return nil
}

// MateInfo enthält Informationen über einen verbundenen Mate
type MateInfo struct {
	MateID       string   `json:"mateId"`
	MateName     string   `json:"mateName"`
	MateType     string   `json:"mateType"`
	Capabilities []string `json:"capabilities"`
}

// GetMateByID gibt Informationen über einen verbundenen Mate zurück
func (s *Server) GetMateByID(mateID string) *MateInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()

	client, exists := s.clientsByMate[mateID]
	if !exists || client == nil {
		return nil
	}

	// Hole Mate-Informationen aus trusted mates
	trustedMates := s.pairingManager.GetTrustedMates()
	for _, tm := range trustedMates {
		if tm.ID == mateID {
			return &MateInfo{
				MateID:       mateID,
				MateName:     tm.Name,
				MateType:     tm.Type,
				Capabilities: []string{}, // TODO: aus Metadata laden
			}
		}
	}

	// Fallback wenn nicht in trusted mates
	return &MateInfo{
		MateID:       mateID,
		MateName:     "Unknown",
		MateType:     "unknown",
		Capabilities: []string{},
	}
}

// ApprovePairing bestätigt ein Pairing und benachrichtigt den Client
func (s *Server) ApprovePairing(requestID string) error {
	mate, err := s.pairingManager.ApprovePairing(requestID)
	if err != nil {
		return err
	}

	// Broadcast an alle nicht-authentifizierten Clients
	// (der anfragende Mate wird es empfangen)
	// Thunderbird-kompatibles Format mit camelCase
	s.mu.Lock() // Write lock da wir Clients authentifizieren
	clientCount := 0
	for client := range s.clients {
		if !client.Authenticated {
			clientCount++
			log.Printf("Sende pairing_approved an Client (ID: %s) und authentifiziere", client.ID)

			// Client als authentifiziert markieren (nach Pairing ist Auth nicht mehr nötig)
			client.MateID = mate.ID
			client.Authenticated = true

			// SecureChannel initialisieren
			secureChannel, err := s.pairingManager.GetSecureChannelForMate(mate.ID)
			if err == nil {
				client.SecureChannel = secureChannel
				log.Printf("🔐 SecureChannel für %s nach Pairing initialisiert", mate.Name)
			} else {
				log.Printf("⚠️ SecureChannel konnte nicht initialisiert werden: %v", err)
			}

			// In clientsByMate registrieren
			s.clientsByMate[mate.ID] = client

			client.sendMessage(MsgPairingApproved, map[string]interface{}{
				"mateId":               mate.ID,
				"mateName":             mate.Name,
				"navigatorExchangeKey": s.pairingManager.GetExchangePublicKey(),
			})
		}
	}
	s.mu.Unlock()
	log.Printf("pairing_approved an %d Clients gesendet und authentifiziert", clientCount)

	return nil
}

// RejectPairing lehnt ein Pairing ab
func (s *Server) RejectPairing(requestID string) error {
	err := s.pairingManager.RejectPairing(requestID)
	if err != nil {
		return err
	}

	// Broadcast Ablehnung
	s.mu.RLock()
	for client := range s.clients {
		if !client.Authenticated {
			client.sendMessage(MsgPairingRejected, map[string]string{
				"request_id": requestID,
			})
		}
	}
	s.mu.RUnlock()

	return nil
}

// handleAuthChallengeRequest verarbeitet eine Auth-Challenge-Anfrage (Thunderbird-Protokoll)
func (c *Client) handleAuthChallengeRequest(msg Message) {
	var payload struct {
		MateID string `json:"mateId"`
	}
	if err := json.Unmarshal(msg.Payload, &payload); err != nil {
		c.sendMessage(MsgAuthFailed, map[string]string{"error": "Ungültige Auth-Challenge-Anfrage"})
		return
	}

	// Generiere Nonce für Challenge
	nonce := security.GenerateRandomID(32)

	// Speichere Nonce für spätere Verifizierung (TODO: Thread-safe storage)
	c.MateID = payload.MateID

	log.Printf("Auth-Challenge-Request von MateID: %s", payload.MateID)

	c.sendMessage(MsgAuthChallenge, map[string]string{
		"nonce": nonce,
	})
}

// handleHeartbeat verarbeitet Heartbeat-Nachrichten (Thunderbird-Protokoll)
func (c *Client) handleHeartbeat(msg Message) {
	// Heartbeat-ACK zurücksenden
	c.sendMessage(MsgHeartbeatAck, map[string]interface{}{
		"timestamp": time.Now().UnixMilli(),
	})

	// LastSeen aktualisieren wenn authentifiziert
	if c.Authenticated && c.MateID != "" {
		c.Server.pairingManager.UpdateLastSeen(c.MateID)
	}
}

// handleEncryptedMessage verarbeitet verschlüsselte Nachrichten (Thunderbird-Protokoll)
func (c *Client) handleEncryptedMessage(msg Message) {
	log.Printf("DEBUG handleEncryptedMessage: start, ClientMateID=%s, MsgMateID=%s, Authenticated=%v", c.MateID, msg.MateID, c.Authenticated)

	if !c.Authenticated {
		log.Printf("DEBUG: Rejected - not authenticated")
		c.sendError("Nicht authentifiziert für verschlüsselte Nachrichten")
		return
	}

	// Thunderbird sendet mateId und payload auf Top-Level, nicht verschachtelt
	// msg.MateID = mateId aus der Nachricht
	// msg.Payload = verschlüsseltes Payload (als JSON-String)

	// Extrahiere das verschlüsselte Payload (kann String oder verschachtelt sein)
	var encryptedPayload string

	// Versuche erst als einfachen String zu parsen
	if err := json.Unmarshal(msg.Payload, &encryptedPayload); err != nil {
		// Fallback: Versuche verschachteltes Format
		var envelope struct {
			MateID  string `json:"mateId"`
			Payload string `json:"payload"`
		}
		if err2 := json.Unmarshal(msg.Payload, &envelope); err2 != nil {
			log.Printf("DEBUG: Unmarshal error (both formats failed): string=%v, nested=%v", err, err2)
			c.sendError("Ungültiges Encrypted-Message Format")
			return
		}
		encryptedPayload = envelope.Payload
	}

	log.Printf("DEBUG: Payload parsed, length=%d", len(encryptedPayload))

	// Prüfen ob SecureChannel vorhanden
	if c.SecureChannel == nil {
		log.Printf("DEBUG: SecureChannel is nil, trying to get one for MateID=%s", c.MateID)
		// Versuche SecureChannel zu initialisieren
		secureChannel, err := c.Server.pairingManager.GetSecureChannelForMate(c.MateID)
		if err != nil {
			log.Printf("ERROR: Keine Verschlüsselung möglich für %s: %v", c.MateID, err)
			c.sendError("Keine Verschlüsselung konfiguriert - bitte erneut pairen")
			return
		}
		c.SecureChannel = secureChannel
		log.Printf("DEBUG: SecureChannel initialized")
	}

	log.Printf("DEBUG: Attempting decryption...")
	// Entschlüsseln
	decryptedJSON, err := c.SecureChannel.DecryptString(encryptedPayload)
	if err != nil {
		log.Printf("ERROR: Entschlüsselung fehlgeschlagen für %s: %v", c.MateID, err)
		c.sendError("Entschlüsselung fehlgeschlagen")
		return
	}

	log.Printf("SUCCESS: Nachricht von %s entschlüsselt (%d bytes)", c.MateID, len(decryptedJSON))
	preview := decryptedJSON
	if len(preview) > 200 {
		preview = preview[:200]
	}
	log.Printf("DEBUG: Decrypted content preview: %s", preview)

	// Innere Nachricht parsen
	var innerMsg struct {
		Type   string          `json:"type"`
		MateID string          `json:"mate_id"`
		Data   json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal([]byte(decryptedJSON), &innerMsg); err != nil {
		log.Printf("❌ Innere Nachricht parsen fehlgeschlagen: %v", err)
		c.sendError("Ungültiges inneres Nachrichtenformat")
		return
	}

	log.Printf("📨 Verarbeite entschlüsselte Nachricht: type=%s", innerMsg.Type)

	// Nachricht entsprechend verarbeiten
	switch innerMsg.Type {
	case "classify_email":
		c.handleClassifyEmail(innerMsg.Data)

	case "generate_reply":
		c.handleGenerateReply(innerMsg.Data)

	case "heartbeat":
		c.sendEncryptedMessage("heartbeat_ack", map[string]interface{}{
			"timestamp": time.Now().UnixMilli(),
		})

	case "stats":
		// Statistiken empfangen und speichern
		c.handleStats(innerMsg.Data)

	case "check_appointment_request":
		c.handleCheckAppointmentRequest(innerMsg.Data)

	case "web_search":
		c.handleWebSearch(innerMsg.Data)

	case "web_fetch":
		c.handleWebFetch(innerMsg.Data)

	default:
		log.Printf("⚠️ Unbekannter verschlüsselter Nachrichtentyp: %s", innerMsg.Type)
	}
}

// handleStats verarbeitet Hardware-Stats von einem Mate
func (c *Client) handleStats(data json.RawMessage) {
	var stats struct {
		System      map[string]interface{}   `json:"system"`
		CPU         map[string]interface{}   `json:"cpu"`
		Memory      map[string]interface{}   `json:"memory"`
		GPU         []map[string]interface{} `json:"gpu"`
		Temperature map[string]interface{}   `json:"temperature"`
	}

	if err := json.Unmarshal(data, &stats); err != nil {
		log.Printf("❌ Stats parsen fehlgeschlagen: %v", err)
		return
	}

	// Stats im Server speichern
	c.Server.mu.Lock()
	if c.Server.mateStats[c.MateID] == nil {
		c.Server.mateStats[c.MateID] = &MateStats{}
	}
	mateStats := c.Server.mateStats[c.MateID]
	mateStats.System = stats.System
	mateStats.CPU = stats.CPU
	mateStats.Memory = stats.Memory
	mateStats.GPU = stats.GPU
	mateStats.Temperature = stats.Temperature
	mateStats.UpdatedAt = time.Now()
	c.Server.mu.Unlock()

	// Hostname für Logging extrahieren
	hostname := "unbekannt"
	if h, ok := stats.System["hostname"].(string); ok {
		hostname = h
	}
	log.Printf("📊 Stats von %s (%s) gespeichert", c.MateID, hostname)
}

// WebSearchRequest für Websuche-Anfragen von Mates
type WebSearchRequest struct {
	RequestID  string `json:"requestId"`
	Query      string `json:"query"`
	MaxResults int    `json:"maxResults,omitempty"`
	Region     string `json:"region,omitempty"`
}

// handleWebSearch verarbeitet Websuche-Anfragen von Mates (z.B. FleetCoder)
func (c *Client) handleWebSearch(data json.RawMessage) {
	var req WebSearchRequest
	if err := json.Unmarshal(data, &req); err != nil {
		log.Printf("❌ WebSearch: Ungültiges Format: %v", err)
		c.sendEncryptedMessage("web_search_error", map[string]interface{}{
			"error": "Ungültiges Anfrage-Format",
		})
		return
	}

	if req.Query == "" {
		c.sendEncryptedMessage("web_search_error", map[string]interface{}{
			"requestId": req.RequestID,
			"error":     "Suchanfrage darf nicht leer sein",
		})
		return
	}

	log.Printf("🔍 WebSearch von %s: %q", c.MateID, req.Query)

	// Defaults setzen
	if req.MaxResults <= 0 {
		req.MaxResults = 5
	}
	if req.Region == "" {
		req.Region = "de-de"
	}

	// WebSearch-Tool erstellen und ausführen
	searchTool := tools.NewWebSearchTool()
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	result, err := searchTool.Execute(ctx, map[string]interface{}{
		"query":      req.Query,
		"maxResults": float64(req.MaxResults),
		"region":     req.Region,
	})

	if err != nil {
		log.Printf("❌ WebSearch Fehler: %v", err)
		c.sendEncryptedMessage("web_search_error", map[string]interface{}{
			"requestId": req.RequestID,
			"error":     err.Error(),
		})
		return
	}

	log.Printf("✅ WebSearch erfolgreich: %d Ergebnisse", len(result.Data.([]interface{})))

	// Ergebnisse zurücksenden
	c.sendEncryptedMessage("web_search_result", map[string]interface{}{
		"requestId": req.RequestID,
		"query":     req.Query,
		"results":   result.Data,
		"success":   result.Success,
	})
}

// WebFetchRequest für URL-Abruf-Anfragen von Mates
type WebFetchRequest struct {
	RequestID string `json:"requestId"`
	URL       string `json:"url"`
	MaxLength int    `json:"maxLength,omitempty"`
}

// handleWebFetch verarbeitet URL-Abruf-Anfragen von Mates
func (c *Client) handleWebFetch(data json.RawMessage) {
	var req WebFetchRequest
	if err := json.Unmarshal(data, &req); err != nil {
		log.Printf("❌ WebFetch: Ungültiges Format: %v", err)
		c.sendEncryptedMessage("web_fetch_error", map[string]interface{}{
			"error": "Ungültiges Anfrage-Format",
		})
		return
	}

	if req.URL == "" {
		c.sendEncryptedMessage("web_fetch_error", map[string]interface{}{
			"requestId": req.RequestID,
			"error":     "URL darf nicht leer sein",
		})
		return
	}

	log.Printf("🌐 WebFetch von %s: %s", c.MateID, req.URL)

	// Defaults setzen
	if req.MaxLength <= 0 {
		req.MaxLength = 50000 // 50KB default
	}

	// WebFetch-Tool erstellen und ausführen
	fetchTool := tools.NewWebFetchTool()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := fetchTool.Execute(ctx, map[string]interface{}{
		"url":       req.URL,
		"maxLength": float64(req.MaxLength),
	})

	if err != nil {
		log.Printf("❌ WebFetch Fehler: %v", err)
		c.sendEncryptedMessage("web_fetch_error", map[string]interface{}{
			"requestId": req.RequestID,
			"error":     err.Error(),
		})
		return
	}

	// Inhalt extrahieren
	content := ""
	if resultMap, ok := result.Data.(map[string]interface{}); ok {
		if c, ok := resultMap["content"].(string); ok {
			content = c
		}
	}

	log.Printf("✅ WebFetch erfolgreich: %d Zeichen", len(content))

	// Ergebnis zurücksenden
	c.sendEncryptedMessage("web_fetch_result", map[string]interface{}{
		"requestId": req.RequestID,
		"url":       req.URL,
		"content":   content,
		"success":   result.Success,
	})
}

// sendEncryptedMessage sendet eine verschlüsselte Nachricht an den Client
func (c *Client) sendEncryptedMessage(msgType string, data interface{}) {
	if c.SecureChannel == nil {
		log.Printf("⚠️ SecureChannel nicht verfügbar für %s", c.MateID)
		return
	}

	innerMsg := map[string]interface{}{
		"type": msgType,
		"data": data,
	}
	innerJSON, _ := json.Marshal(innerMsg)

	encrypted, err := c.SecureChannel.EncryptString(string(innerJSON))
	if err != nil {
		log.Printf("❌ Verschlüsselung fehlgeschlagen: %v", err)
		return
	}

	c.sendMessage(MsgEncrypted, map[string]string{
		"payload": encrypted,
	})
}

// EmailClassifyRequest repräsentiert eine E-Mail-Klassifizierungsanfrage
type EmailClassifyRequest struct {
	MessageID       string            `json:"messageId"`
	AccountEmail    string            `json:"accountEmail"`
	From            string            `json:"from"`
	To              string            `json:"to"`
	Subject         string            `json:"subject"`
	Preview         string            `json:"preview"`
	Date            string            `json:"date"`
	CategoryPrompts map[string]string `json:"categoryPrompts,omitempty"`
	PreferredModel  string            `json:"preferredModel,omitempty"`
}

// handleClassifyEmail verarbeitet eine E-Mail-Klassifizierungsanfrage
func (c *Client) handleClassifyEmail(data json.RawMessage) {
	var req EmailClassifyRequest
	if err := json.Unmarshal(data, &req); err != nil {
		log.Printf("❌ E-Mail-Klassifizierung: Ungültiges Format: %v", err)
		return
	}

	log.Printf("📧 E-Mail-Klassifizierung angefordert: %s von %s", req.Subject, req.From)

	// Kategorien aus den Prompts extrahieren
	categories := make([]string, 0, len(req.CategoryPrompts))
	categoryDescriptions := ""
	for cat, desc := range req.CategoryPrompts {
		categories = append(categories, cat)
		categoryDescriptions += fmt.Sprintf("- %s: %s\n", cat, desc)
	}

	// Falls keine Kategorien, Standardkategorien verwenden
	if len(categories) == 0 {
		categories = []string{"wichtig", "abzuarbeiten", "werbung"}
		categoryDescriptions = `- wichtig: Wichtige E-Mails von bekannten Kontakten, Vorgesetzten, wichtigen Kunden
- abzuarbeiten: E-Mails die eine Antwort oder Aktion erfordern
- werbung: Newsletter, Marketing-E-Mails, Werbung`
	}

	// Prompt für LLM erstellen
	prompt := fmt.Sprintf(`Klassifiziere die folgende E-Mail in GENAU EINE der folgenden Kategorien:
%s

E-Mail-Details:
Von: %s
An: %s
Betreff: %s
Vorschau: %s

Antworte NUR mit dem Kategorienamen (kleingeschrieben, ohne Anführungszeichen oder zusätzlichen Text).
Beispiel: wichtig`, categoryDescriptions, req.From, req.To, req.Subject, req.Preview)

	// Chat-Anfrage an LLM senden
	go func() {
		if c.Server.chatHandler == nil {
			log.Printf("❌ Chat-Handler nicht verfügbar")
			c.sendClassifyResponse(req.MessageID, req.AccountEmail, "abzuarbeiten", 0.5, "Chat-Handler nicht verfügbar")
			return
		}

		// Verwende eine eindeutige Session für Klassifizierung
		sessionID := fmt.Sprintf("classify-%s-%s", c.MateID, req.MessageID)

		var response string
		response, err := c.Server.chatHandler.Chat(sessionID, prompt, nil) // Kein Streaming für Klassifizierung
		if err != nil {
			log.Printf("❌ LLM-Fehler bei Klassifizierung: %v", err)
			c.sendClassifyResponse(req.MessageID, req.AccountEmail, "abzuarbeiten", 0.5, err.Error())
			return
		}

		// Antwort bereinigen und validieren
		category := strings.ToLower(strings.TrimSpace(response))
		category = strings.Trim(category, "\"'.,!?")

		// Prüfen ob gültige Kategorie
		validCategory := false
		for _, cat := range categories {
			if category == strings.ToLower(cat) {
				validCategory = true
				category = strings.ToLower(cat)
				break
			}
		}

		if !validCategory {
			log.Printf("⚠️ Ungültige Kategorie '%s', verwende 'abzuarbeiten'", category)
			category = "abzuarbeiten"
		}

		log.Printf("✅ E-Mail klassifiziert: %s → %s", req.Subject, category)

		// Antwort verschlüsselt senden
		c.sendClassifyResponse(req.MessageID, req.AccountEmail, category, 0.85, "")
	}()
}

// sendClassifyResponse sendet die Klassifizierungsantwort verschlüsselt
func (c *Client) sendClassifyResponse(messageID, accountEmail, category string, confidence float64, reasoning string) {
	response := map[string]interface{}{
		"messageId":    messageID,
		"accountEmail": accountEmail,
		"category":     category,
		"confidence":   confidence,
		"reasoning":    reasoning,
	}

	c.sendEncryptedMessage("classify_response", response)
}

// handleGenerateReply verarbeitet eine Antwort-Generierungsanfrage
func (c *Client) handleGenerateReply(data json.RawMessage) {
	var req struct {
		MessageID      string `json:"messageId"`
		From           string `json:"from"`
		To             string `json:"to"`
		Subject        string `json:"subject"`
		Body           string `json:"body"`
		Date           string `json:"date"`
		PreferredModel string `json:"preferredModel,omitempty"`
	}
	if err := json.Unmarshal(data, &req); err != nil {
		log.Printf("❌ Antwort-Generierung: Ungültiges Format: %v", err)
		return
	}

	log.Printf("📝 Antwort-Generierung angefordert für: %s", req.Subject)

	// Prompt für Antwort-Generierung
	prompt := fmt.Sprintf(`Erstelle eine professionelle, höfliche Antwort auf die folgende E-Mail.
Die Antwort soll auf Deutsch sein, freundlich und sachlich.

Original-E-Mail:
Von: %s
Betreff: %s
Inhalt:
%s

Schreibe nur den Text der Antwort, ohne Anrede-Formeln wie "Sehr geehrte/r" - diese werden automatisch hinzugefügt.`, req.From, req.Subject, req.Body)

	go func() {
		if c.Server.chatHandler == nil {
			log.Printf("❌ Chat-Handler nicht verfügbar")
			return
		}

		sessionID := fmt.Sprintf("reply-%s-%s", c.MateID, req.MessageID)

		response, err := c.Server.chatHandler.Chat(sessionID, prompt, nil)
		if err != nil {
			log.Printf("❌ LLM-Fehler bei Antwort-Generierung: %v", err)
			return
		}

		log.Printf("✅ Antwort generiert für: %s", req.Subject)

		// Antwort verschlüsselt senden
		c.sendEncryptedMessage("reply_generated", map[string]interface{}{
			"messageId":      req.MessageID,
			"suggestedReply": response,
		})
	}()
}

// handleCheckAppointmentRequest prüft ob eine E-Mail eine Terminanfrage ist
func (c *Client) handleCheckAppointmentRequest(data json.RawMessage) {
	var req EmailClassifyRequest
	if err := json.Unmarshal(data, &req); err != nil {
		log.Printf("❌ Terminprüfung: Ungültiges Format: %v", err)
		return
	}

	log.Printf("📅 Terminprüfung angefordert für: %s", req.Subject)

	prompt := fmt.Sprintf(`Prüfe ob die folgende E-Mail eine Terminanfrage enthält.

E-Mail:
Von: %s
Betreff: %s
Vorschau: %s

Antworte NUR mit "JA" wenn die E-Mail eine Terminanfrage, Meeting-Anfrage oder Besprechungsanfrage enthält.
Antworte NUR mit "NEIN" wenn nicht.`, req.From, req.Subject, req.Preview)

	go func() {
		if c.Server.chatHandler == nil {
			return
		}

		sessionID := fmt.Sprintf("appointment-%s-%s", c.MateID, req.MessageID)

		response, err := c.Server.chatHandler.Chat(sessionID, prompt, nil)
		if err != nil {
			log.Printf("❌ LLM-Fehler bei Terminprüfung: %v", err)
			return
		}

		isAppointment := strings.Contains(strings.ToUpper(response), "JA")

		if isAppointment {
			log.Printf("📅 Terminanfrage erkannt in: %s", req.Subject)
			c.sendEncryptedMessage("appointment_request_detected", map[string]interface{}{
				"messageId":      req.MessageID,
				"senderEmail":    req.From,
				"requestDetails": map[string]interface{}{},
			})
		}
	}()
}
