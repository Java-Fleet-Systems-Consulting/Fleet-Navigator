// Package llamaserver - Vision/Multimodal Support für llama-server
// Unterstützt Bildanalyse und Dokumentenerkennung via OpenAI-kompatibler API
package llamaserver

import (
	"bufio"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

// VisionService bietet Bildanalyse-Funktionen über llama-server
type VisionService struct {
	server      *Server
	httpClient  *http.Client
	visionModel string // Aktuell geladenes Vision-Modell
}

// NewVisionService erstellt einen neuen Vision-Service
func NewVisionService(server *Server) *VisionService {
	return &VisionService{
		server: server,
		httpClient: &http.Client{
			Timeout: 5 * time.Minute, // Vision braucht länger
		},
	}
}

// ContentPart repräsentiert einen Teil des multimodalen Contents (OpenAI-Format)
type ContentPart struct {
	Type     string    `json:"type"`               // "text" oder "image_url"
	Text     string    `json:"text,omitempty"`     // Für type="text"
	ImageURL *ImageURL `json:"image_url,omitempty"` // Für type="image_url"
}

// ImageURL enthält die Bild-URL (Base64 Data-URL)
type ImageURL struct {
	URL    string `json:"url"`              // "data:image/jpeg;base64,..." oder HTTP-URL
	Detail string `json:"detail,omitempty"` // "low", "high", "auto"
}

// VisionChatMessage ist eine Chat-Nachricht mit multimodalem Content
type VisionChatMessage struct {
	Role    string        `json:"role"`
	Content []ContentPart `json:"content"`
}

// VisionChatRequest ist das Request-Format für Vision-Chat
type VisionChatRequest struct {
	Messages    []VisionChatMessage `json:"messages"`
	Stream      bool                `json:"stream"`
	Temperature float64             `json:"temperature,omitempty"`
	MaxTokens   int                 `json:"max_tokens,omitempty"`
}

// ImageAnalysisResult enthält das Ergebnis der Bildanalyse
type ImageAnalysisResult struct {
	Description   string       `json:"description"`            // Vollständige Beschreibung
	DocumentType  DocumentType `json:"documentType,omitempty"` // Erkannter Dokumenttyp
	ExtractedText string       `json:"extractedText,omitempty"` // OCR-Text bei Dokumenten
	IsDocument    bool         `json:"isDocument"`             // True wenn es ein Textdokument ist
	Confidence    float64      `json:"confidence"`             // Konfidenz der Erkennung (0-1)
	Language      string       `json:"language,omitempty"`     // Erkannte Sprache

	// Kontext-Informationen
	Sender       *SenderInfo  `json:"sender,omitempty"`       // Absender-Informationen
	Context      DocContext   `json:"context,omitempty"`      // Kontext (behördlich, schulisch, etc.)
	Urgency      Urgency      `json:"urgency,omitempty"`      // Dringlichkeit
	ActionNeeded bool         `json:"actionNeeded,omitempty"` // Handlungsbedarf erkannt?
	Summary      string       `json:"summary,omitempty"`      // Kurze Zusammenfassung

	// Visuelle Analyse (NEU)
	GraphicElements string `json:"graphicElements,omitempty"` // Beschreibung aller grafischen Elemente
	Layout          string `json:"layout,omitempty"`          // Layout-Beschreibung
}

// SenderInfo enthält Informationen über den Absender
type SenderInfo struct {
	Name         string `json:"name,omitempty"`         // Name der Organisation/Person
	Type         string `json:"type,omitempty"`         // Typ: "behörde", "schule", "firma", "privat", "verein"
	LogoDetected bool   `json:"logoDetected,omitempty"` // Logo erkannt?
	Address      string `json:"address,omitempty"`      // Adresse falls erkannt
	Contact      string `json:"contact,omitempty"`      // Kontaktdaten falls erkannt
}

// DocContext definiert den Kontext eines Dokuments
type DocContext string

const (
	ContextUnknown    DocContext = "unknown"
	ContextOfficial   DocContext = "official"   // Behördlich/Amtlich
	ContextEducation  DocContext = "education"  // Schule/Universität
	ContextBusiness   DocContext = "business"   // Geschäftlich/Firma
	ContextMedical    DocContext = "medical"    // Medizinisch/Arzt
	ContextLegal      DocContext = "legal"      // Rechtlich/Anwalt
	ContextFinancial  DocContext = "financial"  // Finanziell/Bank
	ContextPrivate    DocContext = "private"    // Privat
	ContextAdvertising DocContext = "advertising" // Werbung
)

// Urgency definiert die Dringlichkeit
type Urgency string

const (
	UrgencyUnknown  Urgency = "unknown"
	UrgencyLow      Urgency = "low"      // Kann warten
	UrgencyNormal   Urgency = "normal"   // Normal bearbeiten
	UrgencyHigh     Urgency = "high"     // Zeitnah bearbeiten
	UrgencyCritical Urgency = "critical" // Sofort handeln (Fristen, Mahnungen)
)

// DocumentType definiert den Typ eines erkannten Dokuments
type DocumentType string

const (
	DocTypeUnknown      DocumentType = "unknown"
	DocTypeInvoice      DocumentType = "invoice"       // Rechnung
	DocTypeReminder     DocumentType = "reminder"      // Mahnung (!)
	DocTypeContract     DocumentType = "contract"      // Vertrag
	DocTypeLetter       DocumentType = "letter"        // Brief
	DocTypeForm         DocumentType = "form"          // Formular
	DocTypeReceipt      DocumentType = "receipt"       // Quittung/Beleg
	DocTypeNotice       DocumentType = "notice"        // Bescheid (Behörde)
	DocTypeOffer        DocumentType = "offer"         // Angebot
	DocTypeCancellation DocumentType = "cancellation"  // Kündigung
	DocTypeIDCard       DocumentType = "id_card"       // Ausweis
	DocTypeBusinessCard DocumentType = "business_card" // Visitenkarte
	DocTypePhoto        DocumentType = "photo"         // Foto (kein Dokument)
	DocTypeDiagram      DocumentType = "diagram"       // Diagramm/Grafik
	DocTypeScreenshot   DocumentType = "screenshot"    // Screenshot
	DocTypeMedical      DocumentType = "medical"       // Arztbrief/Befund
)

// IsAvailable prüft ob Vision verfügbar ist (llama-server läuft mit Vision-Modell)
func (vs *VisionService) IsAvailable() bool {
	if vs.server == nil {
		return false
	}
	return vs.server.IsHealthy()
}

// AnalyzeImage analysiert ein Bild und gibt eine Beschreibung zurück
func (vs *VisionService) AnalyzeImage(ctx context.Context, base64Image string, prompt string) (string, error) {
	if prompt == "" {
		prompt = "Beschreibe dieses Bild detailliert auf Deutsch."
	}

	return vs.chatWithImage(ctx, prompt, base64Image, false, nil)
}

// StreamAnalyzeImage analysiert ein Bild mit Streaming-Response
func (vs *VisionService) StreamAnalyzeImage(ctx context.Context, base64Image string, prompt string, onChunk func(content string, done bool)) error {
	if prompt == "" {
		prompt = "Beschreibe dieses Bild detailliert auf Deutsch."
	}

	_, err := vs.chatWithImage(ctx, prompt, base64Image, true, onChunk)
	return err
}

// AnalyzeDocument analysiert ein Dokument-Bild und extrahiert strukturierte Informationen
// inkl. Kontext-Erkennung (Logo, Absender, Dringlichkeit)
func (vs *VisionService) AnalyzeDocument(ctx context.Context, base64Image string) (*ImageAnalysisResult, error) {
	// Vision-Modell extrahiert Text direkt aus dem Bild
	ocrText := "" // Text wird aus der Vision-Antwort extrahiert

	// Erweiterter Prompt mit Kontext-Erkennung UND grafischen Elementen
	prompt := `Analysiere dieses Dokument VISUELL und INHALTLICH auf Deutsch.

=== WICHTIGE REGELN ===

1. SEI EHRLICH: Wenn du etwas nicht erkennen kannst, schreibe "NICHT ERKENNBAR" oder "UNKLAR".
   Es ist völlig in Ordnung, wenn du unsicher bist - erfinde NICHTS!

2. KEINE PHANTASIE: Beschreibe NUR was du tatsächlich siehst. Keine Vermutungen, keine Erfindungen.
   - Falsch: "Das Logo zeigt vermutlich ein Haus" (wenn du es nicht klar siehst)
   - Richtig: "Logo vorhanden, aber Details nicht klar erkennbar"

3. MARKIERE UNSICHERHEIT: Bei unsicheren Erkennungen schreibe "(unsicher)" dahinter.
   Beispiel: "ABSENDER_NAME: Müller GmbH (unsicher)"

4. QUALITÄT ZÄHLT: Lieber weniger Information die stimmt, als viel Information die falsch ist.

WICHTIG: Beschreibe ALLE sichtbaren Elemente - nicht nur den Text!

=== VISUELLE ANALYSE ===

1. GRAFISCHE_ELEMENTE: Beschreibe alle visuellen Elemente die du siehst:
   - Logos (Position, Farben, Form)
   - Stempel oder Siegel
   - Unterschriften (handschriftlich?)
   - Tabellen oder Listen
   - Grafiken, Diagramme, Bilder
   - Farbige Bereiche, Hervorhebungen
   - QR-Codes, Barcodes
   - Wasserzeichen

2. LAYOUT: Beschreibe das Layout des Dokuments:
   - Kopfzeile/Briefkopf vorhanden?
   - Spalten-Layout?
   - Fußzeile mit Kontaktdaten?
   - Seitenränder, Abstände

3. LOGO_ERKANNT: Ist ein Logo oder Briefkopf sichtbar? (JA/NEIN)
   Falls JA: Beschreibe das Logo kurz (Farben, Form, Text im Logo)

=== INHALTLICHE ANALYSE ===

4. DOKUMENTTYP: Was für ein Dokument ist das?
   (Brief, Rechnung, Mahnung, Vertrag, Formular, Bescheid, Mitteilung, Quittung, Angebot, Kündigung, oder Sonstiges)

5. ABSENDER_NAME: Wer ist der Absender? (Name der Organisation/Firma/Behörde/Schule, oder "Unbekannt")

6. ABSENDER_TYP: Was für ein Absender ist das?
   (behörde, schule, universität, firma, bank, versicherung, arzt, anwalt, verein, privat, werbung, oder unbekannt)

7. KONTEXT: In welchem Kontext ist dieses Dokument einzuordnen?
   (official=behördlich, education=Bildung, business=geschäftlich, medical=medizinisch, legal=rechtlich, financial=finanziell, private=privat, advertising=Werbung)

8. DRINGLICHKEIT: Wie dringend erscheint dieses Dokument?
   (critical=Fristen/Mahnungen/Sofort handeln, high=zeitnah bearbeiten, normal=normal bearbeiten, low=kann warten)

9. HANDLUNGSBEDARF: Erfordert das Dokument eine Reaktion/Handlung? (JA/NEIN)

10. ZUSAMMENFASSUNG: Fasse den Inhalt in 1-2 Sätzen zusammen.

11. SPRACHE: In welcher Sprache ist das Dokument? (deutsch/englisch/andere)

Antworte EXAKT in diesem Format:
GRAFISCHE_ELEMENTE: [detaillierte Beschreibung aller visuellen Elemente]
LAYOUT: [Beschreibung des Layouts]
LOGO_ERKANNT: [JA/NEIN - falls JA, beschreibe es]
DOKUMENTTYP: [typ]
ABSENDER_NAME: [name]
ABSENDER_TYP: [typ]
KONTEXT: [kontext]
DRINGLICHKEIT: [stufe]
HANDLUNGSBEDARF: [JA/NEIN]
ZUSAMMENFASSUNG: [text]
SPRACHE: [sprache]`

	response, err := vs.chatWithImage(ctx, prompt, base64Image, false, nil)
	if err != nil {
		return nil, fmt.Errorf("Bildanalyse fehlgeschlagen: %w", err)
	}

	// Response parsen mit erweiterter Kontext-Erkennung
	result := vs.parseDocumentResponseWithContext(response, ocrText)
	return result, nil
}

// parseDocumentResponseWithContext parst die erweiterte Dokument-Antwort mit Kontext
func (vs *VisionService) parseDocumentResponseWithContext(response, ocrText string) *ImageAnalysisResult {
	result := &ImageAnalysisResult{
		Description:   response,
		DocumentType:  DocTypeUnknown,
		ExtractedText: ocrText,
		IsDocument:    true,
		Confidence:    0.5,
		Context:       ContextUnknown,
		Urgency:       UrgencyUnknown,
	}

	lines := strings.Split(response, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		valueLower := strings.ToLower(value)

		switch key {
		// Visuelle Analyse (NEU)
		case "GRAFISCHE_ELEMENTE":
			result.GraphicElements = value

		case "LAYOUT":
			result.Layout = value

		case "DOKUMENTTYP":
			result.DocumentType = vs.extractDocumentType(value)

		case "LOGO_ERKANNT":
			if result.Sender == nil {
				result.Sender = &SenderInfo{}
			}
			// Logo kann "JA" oder "JA - blaues Logo mit..." sein
			result.Sender.LogoDetected = strings.HasPrefix(valueLower, "ja")

		case "ABSENDER_NAME":
			if result.Sender == nil {
				result.Sender = &SenderInfo{}
			}
			if valueLower != "unbekannt" && valueLower != "unknown" {
				result.Sender.Name = value
			}

		case "ABSENDER_TYP":
			if result.Sender == nil {
				result.Sender = &SenderInfo{}
			}
			result.Sender.Type = valueLower

		case "KONTEXT":
			result.Context = vs.parseContext(valueLower)

		case "DRINGLICHKEIT":
			result.Urgency = vs.parseUrgency(valueLower)

		case "HANDLUNGSBEDARF":
			result.ActionNeeded = valueLower == "ja" || valueLower == "yes"

		case "ZUSAMMENFASSUNG":
			result.Summary = value

		case "SPRACHE":
			result.Language = value
		}
	}

	// Konfidenz basierend auf erkannten Informationen
	if result.GraphicElements != "" && result.Sender != nil && result.Sender.Name != "" {
		result.Confidence = 0.90 // Visuelle + Inhaltliche Erkennung
	} else if result.Sender != nil && result.Sender.Name != "" {
		result.Confidence = 0.85
	} else if result.DocumentType != DocTypeUnknown {
		result.Confidence = 0.75
	}

	return result
}

// parseContext konvertiert String zu DocContext
func (vs *VisionService) parseContext(s string) DocContext {
	switch {
	case strings.Contains(s, "official") || strings.Contains(s, "behörd") || strings.Contains(s, "amt"):
		return ContextOfficial
	case strings.Contains(s, "education") || strings.Contains(s, "schul") || strings.Contains(s, "uni"):
		return ContextEducation
	case strings.Contains(s, "business") || strings.Contains(s, "geschäft") || strings.Contains(s, "firma"):
		return ContextBusiness
	case strings.Contains(s, "medical") || strings.Contains(s, "medizin") || strings.Contains(s, "arzt"):
		return ContextMedical
	case strings.Contains(s, "legal") || strings.Contains(s, "recht") || strings.Contains(s, "anwalt"):
		return ContextLegal
	case strings.Contains(s, "financial") || strings.Contains(s, "finanz") || strings.Contains(s, "bank"):
		return ContextFinancial
	case strings.Contains(s, "private") || strings.Contains(s, "privat"):
		return ContextPrivate
	case strings.Contains(s, "advertising") || strings.Contains(s, "werbung"):
		return ContextAdvertising
	default:
		return ContextUnknown
	}
}

// parseUrgency konvertiert String zu Urgency
func (vs *VisionService) parseUrgency(s string) Urgency {
	switch {
	case strings.Contains(s, "critical") || strings.Contains(s, "sofort") || strings.Contains(s, "frist"):
		return UrgencyCritical
	case strings.Contains(s, "high") || strings.Contains(s, "hoch") || strings.Contains(s, "zeitnah"):
		return UrgencyHigh
	case strings.Contains(s, "normal"):
		return UrgencyNormal
	case strings.Contains(s, "low") || strings.Contains(s, "niedrig") || strings.Contains(s, "warten"):
		return UrgencyLow
	default:
		return UrgencyUnknown
	}
}

// ClassifyImage klassifiziert ein Bild schnell (Dokument vs. Foto/Grafik)
func (vs *VisionService) ClassifyImage(ctx context.Context, base64Image string) (*ImageAnalysisResult, error) {
	prompt := `Klassifiziere dieses Bild kurz auf Deutsch:
1. Ist es ein TEXTDOKUMENT (Brief, Rechnung, Formular, etc.) oder ein BILD (Foto, Grafik, Screenshot)?
2. Wenn Textdokument: Welcher Typ? (Rechnung/Brief/Vertrag/Formular/Quittung/Sonstiges)

Antworte NUR mit:
TYP: [TEXTDOKUMENT oder BILD]
DOKUMENTTYP: [typ oder KEINER]`

	response, err := vs.chatWithImage(ctx, prompt, base64Image, false, nil)
	if err != nil {
		return nil, err
	}

	result := &ImageAnalysisResult{
		Description: response,
	}

	// Schnelle Klassifizierung parsen
	responseLower := strings.ToLower(response)
	result.IsDocument = strings.Contains(responseLower, "textdokument")

	// Dokumenttyp extrahieren
	result.DocumentType = vs.extractDocumentType(response)

	return result, nil
}

// OCR extrahiert Text aus einem Bild mit dem Vision-Modell
func (vs *VisionService) OCR(ctx context.Context, base64Image string) (string, error) {
	return vs.visionOCR(ctx, base64Image)
}

// OCRWithOptions - Kompatibilitäts-Wrapper (useVisionEnhancement wird ignoriert, immer Vision)
func (vs *VisionService) OCRWithOptions(ctx context.Context, base64Image string, useVisionEnhancement bool) (string, error) {
	return vs.visionOCR(ctx, base64Image)
}

// visionOCR verwendet das Vision-Modell für OCR
func (vs *VisionService) visionOCR(ctx context.Context, base64Image string) (string, error) {
	if !vs.IsAvailable() {
		return "", fmt.Errorf("Vision-Modell ist nicht verfügbar")
	}

	prompt := `Extrahiere den gesamten sichtbaren Text aus diesem Bild.
Gib NUR den extrahierten Text zurück, keine Erklärungen.
Behalte die Formatierung so gut wie möglich bei (Zeilenumbrüche, Absätze).
Falls kein Text sichtbar ist, antworte mit: [KEIN TEXT ERKANNT]`

	return vs.chatWithImage(ctx, prompt, base64Image, false, nil)
}

// chatWithImage führt einen Chat mit Bild durch
func (vs *VisionService) chatWithImage(ctx context.Context, prompt, base64Image string, stream bool, onChunk func(content string, done bool)) (string, error) {
	if !vs.IsAvailable() {
		return "", fmt.Errorf("llama-server ist nicht verfügbar")
	}

	// Bild-Format erkennen und Data-URL erstellen
	imageURL := vs.createImageDataURL(base64Image)

	// Multimodales Message-Format (OpenAI-kompatibel)
	message := VisionChatMessage{
		Role: "user",
		Content: []ContentPart{
			{Type: "text", Text: prompt},
			{Type: "image_url", ImageURL: &ImageURL{URL: imageURL, Detail: "high"}},
		},
	}

	requestBody := VisionChatRequest{
		Messages:    []VisionChatMessage{message},
		Stream:      stream,
		Temperature: 0.3, // Niedrigere Temperatur für präzisere Analyse
		MaxTokens:   4096,
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return "", fmt.Errorf("JSON-Fehler: %w", err)
	}

	url := fmt.Sprintf("http://localhost:%d/v1/chat/completions", vs.server.config.Port)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("Request-Fehler: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := vs.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("llama-server nicht erreichbar: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("llama-server Fehler %d: %s", resp.StatusCode, string(body))
	}

	if stream && onChunk != nil {
		return vs.readStreamResponse(resp.Body, onChunk)
	}

	return vs.readResponse(resp.Body)
}

// readResponse liest eine nicht-streaming Response
func (vs *VisionService) readResponse(body io.Reader) (string, error) {
	var response struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.NewDecoder(body).Decode(&response); err != nil {
		return "", fmt.Errorf("Response-Decode-Fehler: %w", err)
	}

	if len(response.Choices) == 0 {
		return "", fmt.Errorf("keine Antwort vom Modell")
	}

	return response.Choices[0].Message.Content, nil
}

// readStreamResponse liest eine Streaming-Response
func (vs *VisionService) readStreamResponse(body io.Reader, onChunk func(content string, done bool)) (string, error) {
	reader := bufio.NewReader(body)
	var fullResponse strings.Builder

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			return fullResponse.String(), fmt.Errorf("Stream-Lesefehler: %w", err)
		}

		line = strings.TrimSpace(line)
		if line == "" || !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			onChunk("", true)
			break
		}

		var chunk struct {
			Choices []struct {
				Delta struct {
					Content string `json:"content"`
				} `json:"delta"`
				FinishReason *string `json:"finish_reason"`
			} `json:"choices"`
		}

		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}

		if len(chunk.Choices) > 0 {
			content := chunk.Choices[0].Delta.Content
			done := chunk.Choices[0].FinishReason != nil

			fullResponse.WriteString(content)
			if content != "" || done {
				onChunk(content, done)
			}
		}
	}

	return fullResponse.String(), nil
}

// createImageDataURL erstellt eine Data-URL aus Base64-Daten
func (vs *VisionService) createImageDataURL(base64Data string) string {
	// Wenn bereits eine Data-URL, direkt zurückgeben
	if strings.HasPrefix(base64Data, "data:") {
		return base64Data
	}

	// MIME-Type aus Magic Bytes erkennen
	mimeType := vs.detectImageMimeType(base64Data)

	return fmt.Sprintf("data:%s;base64,%s", mimeType, base64Data)
}

// detectImageMimeType erkennt den MIME-Type aus Base64-Daten
func (vs *VisionService) detectImageMimeType(base64Data string) string {
	// Erste Bytes dekodieren für Magic-Byte-Erkennung
	decoded, err := base64.StdEncoding.DecodeString(base64Data[:min(100, len(base64Data))])
	if err != nil || len(decoded) < 4 {
		return "image/jpeg" // Fallback
	}

	// Magic Bytes prüfen
	switch {
	case decoded[0] == 0xFF && decoded[1] == 0xD8:
		return "image/jpeg"
	case decoded[0] == 0x89 && decoded[1] == 0x50 && decoded[2] == 0x4E && decoded[3] == 0x47:
		return "image/png"
	case decoded[0] == 0x47 && decoded[1] == 0x49 && decoded[2] == 0x46:
		return "image/gif"
	case decoded[0] == 0x52 && decoded[1] == 0x49 && decoded[2] == 0x46 && decoded[3] == 0x46:
		return "image/webp"
	default:
		return "image/jpeg"
	}
}

// parseDocumentResponse parst die strukturierte Dokument-Antwort
func (vs *VisionService) parseDocumentResponse(response string) *ImageAnalysisResult {
	result := &ImageAnalysisResult{
		Description:  response,
		DocumentType: DocTypeUnknown,
		Confidence:   0.5,
	}

	lines := strings.Split(response, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "DOKUMENTTYP:") {
			typeStr := strings.TrimSpace(strings.TrimPrefix(line, "DOKUMENTTYP:"))
			result.DocumentType = vs.extractDocumentType(typeStr)
		}

		if strings.HasPrefix(line, "IST_TEXTDOKUMENT:") {
			value := strings.ToUpper(strings.TrimSpace(strings.TrimPrefix(line, "IST_TEXTDOKUMENT:")))
			result.IsDocument = value == "JA" || value == "YES" || value == "TRUE"
		}

		if strings.HasPrefix(line, "TEXT_INHALT:") {
			result.ExtractedText = strings.TrimSpace(strings.TrimPrefix(line, "TEXT_INHALT:"))
		}

		if strings.HasPrefix(line, "SPRACHE:") {
			result.Language = strings.TrimSpace(strings.TrimPrefix(line, "SPRACHE:"))
		}
	}

	// Konfidenz basierend auf Dokumenttyp
	if result.DocumentType != DocTypeUnknown {
		result.Confidence = 0.8
	}

	return result
}

// extractDocumentType extrahiert den Dokumenttyp aus einem String
func (vs *VisionService) extractDocumentType(typeStr string) DocumentType {
	typeLower := strings.ToLower(typeStr)

	// WICHTIG: Reihenfolge beachten! Spezifischere Typen zuerst.
	switch {
	// Dringende Dokumente zuerst
	case strings.Contains(typeLower, "mahnung") || strings.Contains(typeLower, "zahlungserinnerung") || strings.Contains(typeLower, "reminder"):
		return DocTypeReminder
	case strings.Contains(typeLower, "kündigung") || strings.Contains(typeLower, "cancellation"):
		return DocTypeCancellation
	case strings.Contains(typeLower, "bescheid") || strings.Contains(typeLower, "notice"):
		return DocTypeNotice

	// Geschäftsdokumente
	case strings.Contains(typeLower, "rechnung") || strings.Contains(typeLower, "invoice"):
		return DocTypeInvoice
	case strings.Contains(typeLower, "angebot") || strings.Contains(typeLower, "offer") || strings.Contains(typeLower, "kostenvoranschlag"):
		return DocTypeOffer
	case strings.Contains(typeLower, "vertrag") || strings.Contains(typeLower, "contract"):
		return DocTypeContract
	case strings.Contains(typeLower, "quittung") || strings.Contains(typeLower, "beleg") || strings.Contains(typeLower, "receipt"):
		return DocTypeReceipt

	// Medizinische Dokumente
	case strings.Contains(typeLower, "arztbrief") || strings.Contains(typeLower, "befund") || strings.Contains(typeLower, "rezept") || strings.Contains(typeLower, "medical"):
		return DocTypeMedical

	// Allgemeine Dokumente
	case strings.Contains(typeLower, "brief") || strings.Contains(typeLower, "letter") || strings.Contains(typeLower, "mitteilung"):
		return DocTypeLetter
	case strings.Contains(typeLower, "formular") || strings.Contains(typeLower, "form") || strings.Contains(typeLower, "antrag"):
		return DocTypeForm

	// Ausweise/Karten
	case strings.Contains(typeLower, "ausweis") || strings.Contains(typeLower, "personalausweis") || strings.Contains(typeLower, "id"):
		return DocTypeIDCard
	case strings.Contains(typeLower, "visitenkarte") || strings.Contains(typeLower, "business card"):
		return DocTypeBusinessCard

	// Nicht-Dokumente
	case strings.Contains(typeLower, "foto") || strings.Contains(typeLower, "photo") || strings.Contains(typeLower, "bild"):
		return DocTypePhoto
	case strings.Contains(typeLower, "diagramm") || strings.Contains(typeLower, "grafik") || strings.Contains(typeLower, "diagram") || strings.Contains(typeLower, "chart"):
		return DocTypeDiagram
	case strings.Contains(typeLower, "screenshot"):
		return DocTypeScreenshot

	default:
		return DocTypeUnknown
	}
}

// ===== PDF-Unterstützung =====

// ConvertPDFToImages konvertiert ein PDF in Bilder (eine Seite pro Bild)
// Benötigt pdftoppm (poppler-utils) auf dem System
func ConvertPDFToImages(pdfPath string, outputDir string, dpi int) ([]string, error) {
	if dpi == 0 {
		dpi = 200 // Standard-DPI für gute OCR-Qualität
	}

	// Verzeichnis erstellen
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return nil, fmt.Errorf("Output-Verzeichnis erstellen: %w", err)
	}

	// pdftoppm ausführen
	outputPrefix := filepath.Join(outputDir, "page")
	cmd := exec.Command("pdftoppm", "-png", "-r", fmt.Sprintf("%d", dpi), pdfPath, outputPrefix)

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("PDF-Konvertierung fehlgeschlagen (ist poppler-utils installiert?): %w", err)
	}

	// Generierte Bilder finden
	pattern := filepath.Join(outputDir, "page-*.png")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("Bilder suchen: %w", err)
	}

	if len(matches) == 0 {
		return nil, fmt.Errorf("keine Bilder generiert")
	}

	log.Printf("PDF konvertiert: %d Seiten", len(matches))
	return matches, nil
}

// LoadImageAsBase64 lädt ein Bild und gibt es als Base64 zurück
func LoadImageAsBase64(imagePath string) (string, error) {
	data, err := os.ReadFile(imagePath)
	if err != nil {
		return "", fmt.Errorf("Bild lesen: %w", err)
	}
	return base64.StdEncoding.EncodeToString(data), nil
}

// AnalyzePDF analysiert ein PDF-Dokument (alle Seiten)
func (vs *VisionService) AnalyzePDF(ctx context.Context, pdfPath string) ([]*ImageAnalysisResult, error) {
	// Temporäres Verzeichnis für Bilder
	tempDir, err := os.MkdirTemp("", "fleet-pdf-*")
	if err != nil {
		return nil, fmt.Errorf("Temp-Verzeichnis: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// PDF zu Bildern konvertieren
	images, err := ConvertPDFToImages(pdfPath, tempDir, 200)
	if err != nil {
		return nil, err
	}

	// Jede Seite analysieren
	results := make([]*ImageAnalysisResult, 0, len(images))
	for i, imagePath := range images {
		log.Printf("Analysiere Seite %d/%d...", i+1, len(images))

		base64Data, err := LoadImageAsBase64(imagePath)
		if err != nil {
			log.Printf("Warnung: Seite %d konnte nicht geladen werden: %v", i+1, err)
			continue
		}

		result, err := vs.AnalyzeDocument(ctx, base64Data)
		if err != nil {
			log.Printf("Warnung: Seite %d konnte nicht analysiert werden: %v", i+1, err)
			continue
		}

		results = append(results, result)
	}

	return results, nil
}

// SmartAnalyze analysiert ein Bild intelligent und entscheidet automatisch den besten Ansatz
func (vs *VisionService) SmartAnalyze(ctx context.Context, base64Image string) (*ImageAnalysisResult, error) {
	// Schritt 1: Schnelle Klassifizierung
	classification, err := vs.ClassifyImage(ctx, base64Image)
	if err != nil {
		return nil, fmt.Errorf("Klassifizierung fehlgeschlagen: %w", err)
	}

	// Schritt 2: Je nach Typ unterschiedlich verarbeiten
	if classification.IsDocument {
		// Textdokument: Detaillierte Analyse mit OCR
		result, err := vs.AnalyzeDocument(ctx, base64Image)
		if err != nil {
			return classification, nil // Fallback auf Klassifizierung
		}
		return result, nil
	}

	// Kein Textdokument: Einfache Bildbeschreibung
	description, err := vs.AnalyzeImage(ctx, base64Image, "Beschreibe dieses Bild detailliert auf Deutsch. Was ist zu sehen?")
	if err != nil {
		return classification, nil
	}

	return &ImageAnalysisResult{
		Description:  description,
		DocumentType: classification.DocumentType,
		IsDocument:   false,
		Confidence:   0.7,
	}, nil
}

// min gibt das Minimum zweier ints zurück
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// ===== Tesseract OCR Integration =====

// TesseractOCR führt OCR mit Tesseract aus (unbegrenzte Textmenge)
// dataDir ist das Fleet-Navigator Datenverzeichnis (~/.fleet-navigator)
// languages sind die Sprachen für OCR (z.B. "deu+eng+tur")
func TesseractOCR(imagePath string, dataDir string, languages string) (string, error) {
	if languages == "" {
		languages = "deu+eng" // Standard: Deutsch + Englisch
	}

	// Tesseract Binary finden
	tesseractBinary := findTesseractBinary(dataDir)
	if tesseractBinary == "" {
		return "", fmt.Errorf("Tesseract nicht installiert. Bitte über Setup installieren.")
	}

	// Tessdata-Verzeichnis (Sprachdateien)
	tessdataDir := filepath.Join(filepath.Dir(tesseractBinary), "tessdata")
	if _, err := os.Stat(tessdataDir); os.IsNotExist(err) {
		// Fallback: tessdata neben dem Binary
		tessdataDir = filepath.Join(filepath.Dir(tesseractBinary), "..", "tessdata")
	}

	// Tesseract ausführen: tesseract input.png stdout -l deu+eng
	args := []string{imagePath, "stdout", "-l", languages}

	// Tessdata-Pfad setzen wenn vorhanden
	if _, err := os.Stat(tessdataDir); err == nil {
		args = append([]string{"--tessdata-dir", tessdataDir}, args...)
	}

	cmd := exec.Command(tesseractBinary, args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	log.Printf("[Tesseract] Führe OCR aus: %s %v", tesseractBinary, args)

	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("Tesseract-Fehler: %w (stderr: %s)", err, stderr.String())
	}

	text := strings.TrimSpace(stdout.String())
	log.Printf("[Tesseract] ✅ OCR erfolgreich: %d Zeichen extrahiert", len(text))

	return text, nil
}

// TesseractOCRFromBase64 führt OCR auf Base64-Bilddaten aus
func TesseractOCRFromBase64(base64Image string, dataDir string, languages string) (string, error) {
	// Base64 dekodieren
	imageData, err := base64.StdEncoding.DecodeString(base64Image)
	if err != nil {
		// Versuche Data-URL Prefix zu entfernen
		if strings.HasPrefix(base64Image, "data:") {
			parts := strings.SplitN(base64Image, ",", 2)
			if len(parts) == 2 {
				imageData, err = base64.StdEncoding.DecodeString(parts[1])
				if err != nil {
					return "", fmt.Errorf("Base64-Dekodierung fehlgeschlagen: %w", err)
				}
			}
		} else {
			return "", fmt.Errorf("Base64-Dekodierung fehlgeschlagen: %w", err)
		}
	}

	// Temporäre Datei erstellen
	tmpFile, err := os.CreateTemp("", "fleet-ocr-*.png")
	if err != nil {
		return "", fmt.Errorf("Temp-Datei erstellen: %w", err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.Write(imageData); err != nil {
		tmpFile.Close()
		return "", fmt.Errorf("Temp-Datei schreiben: %w", err)
	}
	tmpFile.Close()

	return TesseractOCR(tmpFile.Name(), dataDir, languages)
}

// TesseractOCRMultiPage führt OCR auf mehreren Bildern aus (PDF-Seiten)
// Gibt den kombinierten Text aller Seiten zurück
func TesseractOCRMultiPage(imagePaths []string, dataDir string, languages string) (string, error) {
	var allText strings.Builder

	for i, imagePath := range imagePaths {
		text, err := TesseractOCR(imagePath, dataDir, languages)
		if err != nil {
			log.Printf("[Tesseract] ⚠️ Seite %d fehlgeschlagen: %v", i+1, err)
			continue
		}

		if allText.Len() > 0 {
			allText.WriteString("\n\n--- Seite ")
			allText.WriteString(fmt.Sprintf("%d", i+1))
			allText.WriteString(" ---\n\n")
		}
		allText.WriteString(text)
	}

	return allText.String(), nil
}

// findTesseractBinary sucht die Tesseract-Binary
func findTesseractBinary(dataDir string) string {
	var binaryName string
	if runtime.GOOS == "windows" {
		binaryName = "tesseract.exe"
	} else {
		binaryName = "tesseract"
	}

	// 1. Prüfe Fleet-Navigator Verzeichnis
	fleetPath := filepath.Join(dataDir, "tesseract", binaryName)
	if _, err := os.Stat(fleetPath); err == nil {
		return fleetPath
	}

	// 2. Suche in Unterordnern (Windows-Build hat oft einen Unterordner)
	tesseractDir := filepath.Join(dataDir, "tesseract")
	var foundPath string
	filepath.Walk(tesseractDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.Name() == binaryName && !info.IsDir() {
			foundPath = path
			return filepath.SkipAll
		}
		return nil
	})
	if foundPath != "" {
		return foundPath
	}

	// 3. Prüfe System-PATH (Linux/macOS)
	if path, err := exec.LookPath(binaryName); err == nil {
		return path
	}

	return ""
}

// IsTesseractInstalled prüft ob Tesseract installiert ist
func IsTesseractInstalled(dataDir string) bool {
	return findTesseractBinary(dataDir) != ""
}

// GetTesseractLanguages gibt die verfügbaren Sprachen zurück
func GetTesseractLanguages(dataDir string) []string {
	tesseractBinary := findTesseractBinary(dataDir)
	if tesseractBinary == "" {
		return nil
	}

	cmd := exec.Command(tesseractBinary, "--list-langs")
	output, err := cmd.Output()
	if err != nil {
		return nil
	}

	lines := strings.Split(string(output), "\n")
	var languages []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		// Erste Zeile ist "List of available languages"
		if line != "" && !strings.HasPrefix(line, "List") {
			languages = append(languages, line)
		}
	}

	return languages
}

// ===== Kombinierte Vision + Tesseract Analyse =====

// CombinedAnalysisResult enthält Ergebnisse von Vision UND Tesseract
type CombinedAnalysisResult struct {
	*ImageAnalysisResult           // Eingebettete Vision-Analyse

	// Tesseract OCR
	FullOCRText      string `json:"fullOcrText"`      // Vollständiger OCR-Text (alle Seiten)
	TesseractUsed    bool   `json:"tesseractUsed"`    // Wurde Tesseract verwendet?
	PageCount        int    `json:"pageCount"`        // Anzahl der Seiten
	TesseractError   string `json:"tesseractError,omitempty"` // Tesseract-Fehler falls aufgetreten
}

// AnalyzeDocumentWithOCR analysiert ein Dokument mit Vision UND Tesseract
// Strategie:
// 1. Tesseract: Schnelle Text-Extraktion (alle Seiten, unbegrenzt)
// 2. Vision: Layout, Logos, Stempel, Unterschriften analysieren
// 3. Vision validiert kritische OCR-Stellen (Beträge, Handschrift)
func (vs *VisionService) AnalyzeDocumentWithOCR(ctx context.Context, base64Image string, dataDir string) (*CombinedAnalysisResult, error) {
	result := &CombinedAnalysisResult{
		PageCount: 1,
	}

	// 1. Tesseract OCR für vollständigen Text (IMMER zuerst!)
	ocrText, ocrErr := TesseractOCRFromBase64(base64Image, dataDir, "deu+eng+tur")
	if ocrErr != nil {
		log.Printf("[Vision+OCR] ⚠️ Tesseract-Fehler: %v", ocrErr)
		result.TesseractError = ocrErr.Error()
		result.TesseractUsed = false
	} else {
		result.FullOCRText = ocrText
		result.TesseractUsed = true
		log.Printf("[Vision+OCR] ✅ Tesseract: %d Zeichen extrahiert", len(ocrText))
	}

	// 2. Vision-Analyse für Layout/Struktur UND OCR-Validierung
	visionResult, visionErr := vs.AnalyzeDocument(ctx, base64Image)
	if visionErr != nil {
		log.Printf("[Vision+OCR] ⚠️ Vision-Fehler: %v", visionErr)
		// Bei Vision-Fehler trotzdem OCR-Text zurückgeben
		if result.TesseractUsed {
			result.ImageAnalysisResult = &ImageAnalysisResult{
				Description:   "OCR-Text extrahiert (Vision nicht verfügbar)",
				ExtractedText: ocrText,
				IsDocument:    true,
			}
		} else {
			return nil, fmt.Errorf("weder Vision noch OCR verfügbar: %v", visionErr)
		}
	} else {
		result.ImageAnalysisResult = visionResult

		// 3. Vision validiert kritische OCR-Stellen (Beträge, Zahlen)
		if result.TesseractUsed && len(ocrText) > 0 {
			validatedText := vs.validateOCRWithVision(ctx, base64Image, ocrText)
			if validatedText != "" {
				result.FullOCRText = validatedText
				log.Printf("[Vision+OCR] ✅ OCR durch Vision validiert")
			}
		}

		// OCR-Text in ExtractedText einfügen (vollständiger als Vision-OCR)
		if result.TesseractUsed && len(result.FullOCRText) > len(visionResult.ExtractedText) {
			result.ExtractedText = result.FullOCRText
		}
	}

	return result, nil
}

// validateOCRWithVision nutzt Vision um kritische OCR-Stellen zu validieren
// Prüft: Beträge, Zahlen, handschriftliche Texte
func (vs *VisionService) validateOCRWithVision(ctx context.Context, base64Image string, ocrText string) string {
	// Prüfe ob kritische Stellen vorhanden sind (Zahlen, Beträge)
	hasNumbers := strings.ContainsAny(ocrText, "0123456789")
	hasEuro := strings.Contains(ocrText, "€") || strings.Contains(strings.ToLower(ocrText), "eur")

	// Nur validieren wenn kritische Inhalte vorhanden
	if !hasNumbers && !hasEuro {
		return ocrText // Keine Validierung nötig
	}

	// Vision-Prompt für OCR-Validierung
	prompt := fmt.Sprintf(`Überprüfe den folgenden OCR-Text auf Fehler, besonders bei:
- Zahlen und Beträgen (€, EUR)
- Handschriftlichen Texten
- Datumsangaben
- Namen und Adressen

OCR-TEXT:
%s

Korrigiere offensichtliche OCR-Fehler (z.B. 0/O Verwechslung, l/1 Verwechslung).
Gib NUR den korrigierten Text zurück, keine Erklärungen.
Falls alles korrekt ist, gib den Text unverändert zurück.`, ocrText)

	// Kurzes Timeout für Validierung (nicht blockieren wenn langsam)
	validateCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	validated, err := vs.chatWithImage(validateCtx, prompt, base64Image, false, nil)
	if err != nil {
		log.Printf("[Vision+OCR] ⚠️ Validierung fehlgeschlagen: %v", err)
		return ocrText // Original zurückgeben bei Fehler
	}

	// Prüfe ob Validierung sinnvoll ist (nicht zu kurz, nicht leer)
	validated = strings.TrimSpace(validated)
	if len(validated) < len(ocrText)/2 {
		log.Printf("[Vision+OCR] ⚠️ Validierung zu kurz, ignoriert")
		return ocrText
	}

	return validated
}

// AnalyzePDFWithOCR analysiert ein PDF mit Vision UND Tesseract
// Vision: Erste Seite für Layout/Struktur
// Tesseract: ALLE Seiten für vollständigen Text
func (vs *VisionService) AnalyzePDFWithOCR(ctx context.Context, pdfPath string, dataDir string) (*CombinedAnalysisResult, error) {
	// Temporäres Verzeichnis für Bilder
	tempDir, err := os.MkdirTemp("", "fleet-pdf-ocr-*")
	if err != nil {
		return nil, fmt.Errorf("Temp-Verzeichnis: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// PDF zu Bildern konvertieren
	images, err := ConvertPDFToImages(pdfPath, tempDir, 200)
	if err != nil {
		return nil, fmt.Errorf("PDF-Konvertierung: %w", err)
	}

	result := &CombinedAnalysisResult{
		PageCount: len(images),
	}

	log.Printf("[Vision+OCR] PDF mit %d Seiten erkannt", len(images))

	// 1. Tesseract OCR für ALLE Seiten (WICHTIG: Vollständiger Text!)
	allOCRText, ocrErr := TesseractOCRMultiPage(images, dataDir, "deu+eng+tur")
	if ocrErr != nil {
		log.Printf("[Vision+OCR] ⚠️ Tesseract-Fehler: %v", ocrErr)
		result.TesseractError = ocrErr.Error()
		result.TesseractUsed = false
	} else {
		result.FullOCRText = allOCRText
		result.TesseractUsed = true
		log.Printf("[Vision+OCR] ✅ Tesseract: %d Zeichen aus %d Seiten extrahiert", len(allOCRText), len(images))
	}

	// 2. Vision-Analyse NUR für erste Seite (Layout, Logos, Struktur)
	if len(images) > 0 {
		firstPageBase64, err := LoadImageAsBase64(images[0])
		if err != nil {
			log.Printf("[Vision+OCR] ⚠️ Erste Seite laden: %v", err)
		} else {
			visionResult, visionErr := vs.AnalyzeDocument(ctx, firstPageBase64)
			if visionErr != nil {
				log.Printf("[Vision+OCR] ⚠️ Vision-Fehler: %v", visionErr)
			} else {
				result.ImageAnalysisResult = visionResult
				log.Printf("[Vision+OCR] ✅ Vision: Layout/Struktur analysiert")
			}
		}
	}

	// Fallback wenn Vision fehlgeschlagen
	if result.ImageAnalysisResult == nil {
		result.ImageAnalysisResult = &ImageAnalysisResult{
			Description:   fmt.Sprintf("PDF mit %d Seiten (OCR-Text extrahiert)", len(images)),
			ExtractedText: allOCRText,
			IsDocument:    true,
		}
	} else {
		// Vollständigen OCR-Text einfügen
		result.ExtractedText = allOCRText
	}

	return result, nil
}
