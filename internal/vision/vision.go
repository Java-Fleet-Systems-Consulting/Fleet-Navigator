// Package vision implements vision AI functionality using LLaVA via Ollama
// with integrated Tesseract OCR for complete text extraction
package vision

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

// Service provides vision AI capabilities using LLaVA
type Service struct {
	ollamaURL   string
	visionModel string
	client      *http.Client
}

// Config holds vision service configuration
type Config struct {
	OllamaURL   string
	VisionModel string
	Timeout     time.Duration
}

// DefaultConfig returns default configuration
func DefaultConfig() Config {
	return Config{
		OllamaURL:   "http://localhost:11434",
		VisionModel: "llava:13b",
		Timeout:     180 * time.Second, // Vision takes longer
	}
}

// VisionMessage represents a message that can contain images
type VisionMessage struct {
	Role    string   `json:"role"`
	Content string   `json:"content"`
	Images  []string `json:"images,omitempty"` // Base64 encoded images
}

// VisionRequest is the request format for Ollama vision models
type VisionRequest struct {
	Model    string          `json:"model"`
	Messages []VisionMessage `json:"messages"`
	Stream   bool            `json:"stream"`
	Options  *VisionOptions  `json:"options,omitempty"`
}

// VisionOptions for vision model
type VisionOptions struct {
	Temperature float64 `json:"temperature,omitempty"`
	NumPredict  int     `json:"num_predict,omitempty"`
}

// VisionResponse is the streaming response from Ollama
type VisionResponse struct {
	Model     string `json:"model"`
	CreatedAt string `json:"created_at"`
	Message   struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	} `json:"message"`
	Done bool `json:"done"`
}

// ImageAnalysis represents the result of image analysis
type ImageAnalysis struct {
	Description string   `json:"description"`
	Text        string   `json:"text,omitempty"`        // OCR extracted text
	Objects     []string `json:"objects,omitempty"`     // Detected objects
	DocumentType string  `json:"documentType,omitempty"` // invoice, contract, letter, etc.
}

// NewService creates a new vision service
func NewService(config Config) *Service {
	if config.Timeout == 0 {
		config.Timeout = 180 * time.Second
	}
	return &Service{
		ollamaURL:   config.OllamaURL,
		visionModel: config.VisionModel,
		client: &http.Client{
			Timeout: config.Timeout,
		},
	}
}

// AnalyzeImage analyzes an image and returns a description
func (s *Service) AnalyzeImage(ctx context.Context, base64Image string, prompt string) (string, error) {
	if prompt == "" {
		prompt = "Beschreibe dieses Bild detailliert auf Deutsch. Wenn Text sichtbar ist, extrahiere ihn."
	}

	messages := []VisionMessage{
		{
			Role:    "user",
			Content: prompt,
			Images:  []string{base64Image},
		},
	}

	return s.chat(ctx, messages, false, nil)
}

// AnalyzeDocument analyzes a document image and extracts text/structure
func (s *Service) AnalyzeDocument(ctx context.Context, base64Image string) (*ImageAnalysis, error) {
	prompt := `Analysiere dieses Dokument auf Deutsch:
1. Was für ein Dokumenttyp ist es? (Rechnung, Vertrag, Brief, Formular, etc.)
2. Extrahiere den wichtigsten Text
3. Fasse den Inhalt zusammen

Antworte strukturiert mit:
DOKUMENTTYP: [typ]
TEXT: [extrahierter text]
ZUSAMMENFASSUNG: [zusammenfassung]`

	result, err := s.AnalyzeImage(ctx, base64Image, prompt)
	if err != nil {
		return nil, err
	}

	// Parse the structured response
	analysis := &ImageAnalysis{
		Description: result,
	}

	// Try to extract document type
	if idx := strings.Index(result, "DOKUMENTTYP:"); idx >= 0 {
		end := strings.Index(result[idx:], "\n")
		if end > 0 {
			analysis.DocumentType = strings.TrimSpace(result[idx+len("DOKUMENTTYP:") : idx+end])
		}
	}

	// Try to extract text
	if idx := strings.Index(result, "TEXT:"); idx >= 0 {
		nextSection := strings.Index(result[idx:], "ZUSAMMENFASSUNG:")
		if nextSection > 0 {
			analysis.Text = strings.TrimSpace(result[idx+len("TEXT:") : idx+nextSection])
		}
	}

	return analysis, nil
}

// StreamAnalyzeImage analyzes an image with streaming response
func (s *Service) StreamAnalyzeImage(ctx context.Context, base64Image string, prompt string, onChunk func(content string, done bool)) error {
	if prompt == "" {
		prompt = "Beschreibe dieses Bild detailliert auf Deutsch."
	}

	messages := []VisionMessage{
		{
			Role:    "user",
			Content: prompt,
			Images:  []string{base64Image},
		},
	}

	_, err := s.chat(ctx, messages, true, onChunk)
	return err
}

// ChatWithImage sends a chat message with an image and streams the response
func (s *Service) ChatWithImage(ctx context.Context, message string, base64Images []string, onChunk func(content string, done bool)) error {
	messages := []VisionMessage{
		{
			Role:    "user",
			Content: message,
			Images:  base64Images,
		},
	}

	_, err := s.chat(ctx, messages, true, onChunk)
	return err
}

// ChatWithImageAndHistory sends a chat message with image and conversation history
func (s *Service) ChatWithImageAndHistory(ctx context.Context, history []VisionMessage, message string, base64Images []string, onChunk func(content string, done bool)) error {
	// Add new message with images
	messages := append(history, VisionMessage{
		Role:    "user",
		Content: message,
		Images:  base64Images,
	})

	_, err := s.chat(ctx, messages, true, onChunk)
	return err
}

// chat is the internal method that handles the actual API call
func (s *Service) chat(ctx context.Context, messages []VisionMessage, stream bool, onChunk func(content string, done bool)) (string, error) {
	reqBody := VisionRequest{
		Model:    s.visionModel,
		Messages: messages,
		Stream:   stream,
		Options: &VisionOptions{
			Temperature: 0.7,
		},
	}

	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("JSON Marshal Fehler: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", s.ollamaURL+"/api/chat", bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", fmt.Errorf("Request Erstellen Fehler: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("Ollama nicht erreichbar: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("Ollama Fehler (Status %d): %s", resp.StatusCode, string(body))
	}

	if stream && onChunk != nil {
		// Streaming mode
		var fullResponse strings.Builder
		scanner := bufio.NewScanner(resp.Body)
		buf := make([]byte, 0, 64*1024)
		scanner.Buffer(buf, 1024*1024)

		for scanner.Scan() {
			line := scanner.Text()
			if line == "" {
				continue
			}

			var visionResp VisionResponse
			if err := json.Unmarshal([]byte(line), &visionResp); err != nil {
				continue
			}

			fullResponse.WriteString(visionResp.Message.Content)
			onChunk(visionResp.Message.Content, visionResp.Done)

			if visionResp.Done {
				break
			}
		}

		if err := scanner.Err(); err != nil {
			return "", fmt.Errorf("Stream lesen Fehler: %w", err)
		}

		return fullResponse.String(), nil
	}

	// Non-streaming mode
	var visionResp VisionResponse
	if err := json.NewDecoder(resp.Body).Decode(&visionResp); err != nil {
		return "", fmt.Errorf("Response Decode Fehler: %w", err)
	}

	return visionResp.Message.Content, nil
}

// IsAvailable checks if the vision model is available
func (s *Service) IsAvailable() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", s.ollamaURL+"/api/tags", nil)
	if err != nil {
		return false
	}

	resp, err := s.client.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false
	}

	var result struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false
	}

	// Check if vision model is in the list
	for _, m := range result.Models {
		if m.Name == s.visionModel || strings.HasPrefix(m.Name, "llava") {
			return true
		}
	}

	return false
}

// GetModel returns the current vision model name
func (s *Service) GetModel() string {
	return s.visionModel
}

// SetModel sets the vision model to use
func (s *Service) SetModel(model string) {
	s.visionModel = model
}

// IsVisionModel checks if a model name is a known vision model
func IsVisionModel(modelName string) bool {
	visionModels := []string{
		"llava",
		"llava:7b",
		"llava:13b",
		"llava:34b",
		"bakllava",
		"llava-llama3",
		"llava-phi3",
		"moondream",
	}

	modelLower := strings.ToLower(modelName)
	for _, vm := range visionModels {
		if strings.HasPrefix(modelLower, vm) {
			return true
		}
	}
	return false
}

// ===== Tesseract OCR Integration =====

// CombinedAnalysisResult enthält Ergebnisse von Vision UND Tesseract
type CombinedAnalysisResult struct {
	*ImageAnalysis // Eingebettete Vision-Analyse

	// Tesseract OCR
	FullOCRText    string `json:"fullOcrText"`              // Vollständiger OCR-Text (alle Seiten)
	TesseractUsed  bool   `json:"tesseractUsed"`            // Wurde Tesseract verwendet?
	PageCount      int    `json:"pageCount"`                // Anzahl der Seiten
	TesseractError string `json:"tesseractError,omitempty"` // Tesseract-Fehler falls aufgetreten
}

// AnalyzeDocumentWithOCR analysiert ein Dokument mit Vision UND Tesseract
// Strategie:
// 1. Tesseract: Schnelle Text-Extraktion (alle Seiten, unbegrenzt)
// 2. Vision: Layout, Logos, Stempel, Unterschriften analysieren
// 3. Vision validiert kritische OCR-Stellen (Beträge, Handschrift)
func (s *Service) AnalyzeDocumentWithOCR(ctx context.Context, base64Image string, dataDir string) (*CombinedAnalysisResult, error) {
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

	// 2. Vision-Analyse für Layout/Struktur
	visionResult, visionErr := s.AnalyzeDocument(ctx, base64Image)
	if visionErr != nil {
		log.Printf("[Vision+OCR] ⚠️ Vision-Fehler: %v", visionErr)
		// Bei Vision-Fehler trotzdem OCR-Text zurückgeben
		if result.TesseractUsed {
			result.ImageAnalysis = &ImageAnalysis{
				Description: "OCR-Text extrahiert (Vision nicht verfügbar)",
				Text:        ocrText,
			}
		} else {
			return nil, fmt.Errorf("weder Vision noch OCR verfügbar: %v", visionErr)
		}
	} else {
		result.ImageAnalysis = visionResult

		// 3. Vision validiert kritische OCR-Stellen (Beträge, Zahlen)
		if result.TesseractUsed && len(ocrText) > 0 {
			validatedText := s.validateOCRWithVision(ctx, base64Image, ocrText)
			if validatedText != "" {
				result.FullOCRText = validatedText
				log.Printf("[Vision+OCR] ✅ OCR durch Vision validiert")
			}
		}

		// OCR-Text in Text einfügen (vollständiger als Vision-OCR)
		if result.TesseractUsed && len(result.FullOCRText) > len(visionResult.Text) {
			result.Text = result.FullOCRText
		}
	}

	return result, nil
}

// validateOCRWithVision nutzt Vision um kritische OCR-Stellen zu validieren
// Prüft: Beträge, Zahlen, handschriftliche Texte
func (s *Service) validateOCRWithVision(ctx context.Context, base64Image string, ocrText string) string {
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

	validated, err := s.AnalyzeImage(validateCtx, base64Image, prompt)
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

// =============================================================================
// DOKUMENT-KLASSIFIZIERUNG
// Erkennt ob ein Bild ein rechtliches/geschäftliches Dokument ist
// =============================================================================

// ImageClassification enthält das Ergebnis der Bildklassifizierung
type ImageClassification struct {
	IsDocument     bool     `json:"isDocument"`     // Ist es ein Dokument?
	DocumentType   string   `json:"documentType"`   // Art des Dokuments (invoice, contract, letter, etc.)
	Confidence     float64  `json:"confidence"`     // Konfidenz 0.0-1.0
	LegalRelevance bool     `json:"legalRelevance"` // Hat rechtliche Relevanz?
	OCRText        string   `json:"ocrText"`        // Extrahierter Text (wenn Dokument)
	Indicators     []string `json:"indicators"`     // Gefundene Indikatoren
}

// ClassifyImage klassifiziert ein Bild als Dokument oder normales Foto
// Verwendet schnelle OCR-Analyse statt langsamem Vision-Modell
func ClassifyImage(base64Image string, dataDir string) (*ImageClassification, error) {
	result := &ImageClassification{
		Confidence: 0.0,
	}

	// Tesseract OCR für schnelle Textextraktion
	ocrText, err := TesseractOCRFromBase64(base64Image, dataDir, "deu+eng")
	if err != nil {
		// Kein Tesseract verfügbar - kann nicht klassifizieren, nehme Foto an
		log.Printf("[ClassifyImage] OCR fehlgeschlagen: %v - nehme Foto an", err)
		return result, nil
	}

	result.OCRText = ocrText
	textLower := strings.ToLower(ocrText)
	textLen := len(ocrText)

	// Wenig Text = wahrscheinlich Foto
	if textLen < 50 {
		result.IsDocument = false
		result.Confidence = 0.9
		return result, nil
	}

	// === RECHTLICHE INDIKATOREN (hohe Relevanz) ===
	legalIndicators := []string{
		"mahnung", "kündigung", "vertrag", "mietvertrag", "arbeitsvertrag",
		"vollmacht", "testament", "urteil", "bescheid", "klage",
		"anwalt", "rechtsanwalt", "gericht", "aktenzeichen", "geschäftszeichen",
		"frist", "zahlungsfrist", "kündigungsfrist", "widerspruch",
		"schadensersatz", "haftung", "gewährleistung", "bürgschaft",
	}

	// === GESCHÄFTLICHE INDIKATOREN ===
	businessIndicators := []string{
		"rechnung", "invoice", "rechnungsnummer", "rechnungsdatum",
		"bestellung", "lieferschein", "angebot", "kostenvoranschlag",
		"steuernummer", "ust-id", "iban", "bic", "kontonummer",
		"zahlungsziel", "skonto", "netto", "brutto", "mwst", "mehrwertsteuer",
		"bankverbindung", "überweisung", "lastschrift",
	}

	// === FORMALE BRIEF-INDIKATOREN ===
	letterIndicators := []string{
		"sehr geehrte", "mit freundlichen grüßen", "hochachtungsvoll",
		"betreff", "datum:", "ihr zeichen", "unser zeichen",
		"absender", "empfänger", "einschreiben",
	}

	// === BEHÖRDLICHE INDIKATOREN ===
	officialIndicators := []string{
		"finanzamt", "standesamt", "einwohnermeldeamt", "arbeitsamt",
		"jobcenter", "sozialamt", "jugendamt", "ordnungsamt",
		"steuerbescheid", "bußgeldbescheid", "mahnbescheid", "vollstreckungsbescheid",
		"amtsgericht", "landgericht", "oberlandesgericht", "bundesgericht",
	}

	// Indikatoren zählen
	foundIndicators := []string{}
	legalCount := 0
	businessCount := 0
	letterCount := 0
	officialCount := 0

	for _, ind := range legalIndicators {
		if strings.Contains(textLower, ind) {
			legalCount++
			foundIndicators = append(foundIndicators, "legal:"+ind)
		}
	}
	for _, ind := range businessIndicators {
		if strings.Contains(textLower, ind) {
			businessCount++
			foundIndicators = append(foundIndicators, "business:"+ind)
		}
	}
	for _, ind := range letterIndicators {
		if strings.Contains(textLower, ind) {
			letterCount++
			foundIndicators = append(foundIndicators, "letter:"+ind)
		}
	}
	for _, ind := range officialIndicators {
		if strings.Contains(textLower, ind) {
			officialCount++
			foundIndicators = append(foundIndicators, "official:"+ind)
		}
	}

	result.Indicators = foundIndicators
	totalIndicators := legalCount + businessCount + letterCount + officialCount

	// === KLASSIFIZIERUNG ===
	if totalIndicators == 0 && textLen < 200 {
		// Wenig Text, keine Indikatoren = Foto
		result.IsDocument = false
		result.Confidence = 0.8
		return result, nil
	}

	// Dokumenttyp bestimmen
	if officialCount >= 2 || (officialCount >= 1 && legalCount >= 1) {
		result.IsDocument = true
		result.DocumentType = "official_document" // Behördliches Dokument
		result.LegalRelevance = true
		result.Confidence = 0.95
	} else if legalCount >= 2 {
		result.IsDocument = true
		result.DocumentType = "legal_document" // Rechtliches Dokument
		result.LegalRelevance = true
		result.Confidence = 0.9
	} else if businessCount >= 3 || (businessCount >= 2 && strings.Contains(textLower, "€")) {
		result.IsDocument = true
		if strings.Contains(textLower, "rechnung") || strings.Contains(textLower, "invoice") {
			result.DocumentType = "invoice"
		} else if strings.Contains(textLower, "angebot") {
			result.DocumentType = "quote"
		} else {
			result.DocumentType = "business_document"
		}
		result.LegalRelevance = businessCount >= 2 && legalCount >= 1
		result.Confidence = 0.85
	} else if letterCount >= 2 {
		result.IsDocument = true
		result.DocumentType = "letter"
		result.LegalRelevance = legalCount >= 1 || officialCount >= 1
		result.Confidence = 0.8
	} else if totalIndicators >= 2 {
		result.IsDocument = true
		result.DocumentType = "document"
		result.LegalRelevance = legalCount >= 1
		result.Confidence = 0.7
	} else if textLen > 500 {
		// Viel Text aber wenige Indikatoren - wahrscheinlich trotzdem Dokument
		result.IsDocument = true
		result.DocumentType = "text_document"
		result.LegalRelevance = false
		result.Confidence = 0.6
	} else {
		// Zu wenig Evidenz
		result.IsDocument = false
		result.Confidence = 0.5
	}

	log.Printf("[ClassifyImage] Ergebnis: isDocument=%v, type=%s, legal=%v, confidence=%.2f, indicators=%d",
		result.IsDocument, result.DocumentType, result.LegalRelevance, result.Confidence, len(foundIndicators))

	return result, nil
}
