// Package middleware enthält HTTP-Middleware für Security und Rate Limiting.
//
// Dieses Paket implementiert verschiedene Sicherheitsmaßnahmen:
//   - CORS (Cross-Origin Resource Sharing) mit Whitelist
//   - Rate Limiting mit Token Bucket Algorithmus
//   - Security Headers (X-Frame-Options, X-Content-Type-Options, etc.)
//   - Request Size Limiting
//
// Erstellt: 2025-12-15 (Security Audit)
package middleware

import (
	"log"
	"net/http"
	"strings"
	"sync"
	"time"
)

// =============================================================================
// KONFIGURATION
// =============================================================================

// SecurityConfig enthält alle Einstellungen für die Security-Middleware.
// Diese Struktur wird beim Start der Anwendung konfiguriert.
type SecurityConfig struct {
	// AllowedOrigins: Liste der erlaubten CORS Origins (z.B. "http://localhost:2025")
	// Nur diese Origins dürfen Cross-Origin-Requests an die API senden.
	AllowedOrigins []string

	// MaxRequestSize: Maximale Größe des Request-Body in Bytes.
	// Schützt vor DoS-Angriffen durch übermäßig große Requests.
	MaxRequestSize int64

	// RateLimitPerSec: Anzahl der erlaubten Requests pro Sekunde pro IP.
	// Nach Überschreitung werden Requests mit HTTP 429 abgelehnt.
	RateLimitPerSec int

	// RateLimitBurst: Maximale Anzahl gleichzeitiger Requests (Burst).
	// Erlaubt kurzzeitige Lastspitzen über dem normalen Limit.
	RateLimitBurst int

	// EnableRateLimit: Schalter zum Aktivieren/Deaktivieren des Rate Limitings.
	// Kann für Entwicklung/Tests deaktiviert werden.
	EnableRateLimit bool
}

// DefaultSecurityConfig gibt eine sichere Standard-Konfiguration zurück.
// Diese Defaults sind für lokale Desktop-Nutzung optimiert.
func DefaultSecurityConfig() SecurityConfig {
	return SecurityConfig{
		// Erlaubte Origins: Nur localhost für Desktop-App
		AllowedOrigins: []string{
			"http://localhost:2025",   // Fleet Navigator Frontend
			"http://127.0.0.1:2025",   // Alternative localhost
			"http://localhost:5173",   // Vite Dev Server (Entwicklung)
			"http://127.0.0.1:5173",   // Alternative Vite
		},
		MaxRequestSize:  10 * 1024 * 1024, // 10 MB - ausreichend für große Chat-Nachrichten
		RateLimitPerSec: 100,              // 100 Requests/Sekunde - großzügig für lokale Nutzung
		RateLimitBurst:  200,              // 200 Burst - für schnelle UI-Interaktionen
		EnableRateLimit: true,             // Standardmäßig aktiviert
	}
}

// =============================================================================
// SECURITY MIDDLEWARE (Hauptkomponente)
// =============================================================================

// SecurityMiddleware ist die zentrale Sicherheitskomponente.
// Sie kombiniert alle Sicherheitsmaßnahmen in einer einzigen Middleware.
type SecurityMiddleware struct {
	config      SecurityConfig // Aktuelle Konfiguration
	rateLimiter *RateLimiter   // Rate Limiter Instanz
}

// NewSecurityMiddleware erstellt eine neue Security-Middleware mit der gegebenen Konfiguration.
// Der Rate Limiter wird automatisch initialisiert und gestartet.
func NewSecurityMiddleware(config SecurityConfig) *SecurityMiddleware {
	return &SecurityMiddleware{
		config:      config,
		rateLimiter: NewRateLimiter(config.RateLimitPerSec, config.RateLimitBurst),
	}
}

// Wrap umhüllt einen HTTP-Handler mit allen Sicherheitsmaßnahmen.
// Die Reihenfolge der Prüfungen ist wichtig für die Sicherheit:
//  1. Rate Limiting (frühzeitig ablehnen bei Überlastung)
//  2. Request Size (vor dem Lesen des Body)
//  3. Security Headers (für alle Responses)
//  4. CORS (für Cross-Origin-Requests)
//  5. Preflight-Handling (OPTIONS-Requests)
func (sm *SecurityMiddleware) Wrap(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		// ---------------------------------------------------------------------
		// SCHRITT 1: Rate Limiting
		// Prüft ob der Client zu viele Requests sendet.
		// Bei Überschreitung wird sofort HTTP 429 zurückgegeben.
		// ---------------------------------------------------------------------
		if sm.config.EnableRateLimit {
			clientIP := getClientIP(r)
			if !sm.rateLimiter.Allow(clientIP) {
				http.Error(w, "Too Many Requests", http.StatusTooManyRequests)
				log.Printf("SECURITY: Rate Limit überschritten für IP: %s", clientIP)
				return
			}
		}

		// ---------------------------------------------------------------------
		// SCHRITT 2: Request Size Limit
		// Verhindert DoS durch übermäßig große Request-Bodies.
		// Content-Length wird VOR dem Lesen geprüft.
		// ---------------------------------------------------------------------
		if r.ContentLength > sm.config.MaxRequestSize {
			http.Error(w, "Request Entity Too Large", http.StatusRequestEntityTooLarge)
			return
		}
		// MaxBytesReader begrenzt auch Requests ohne Content-Length Header
		r.Body = http.MaxBytesReader(w, r.Body, sm.config.MaxRequestSize)

		// ---------------------------------------------------------------------
		// SCHRITT 3: Security Headers
		// Setzt alle empfohlenen Security-Header für die Response.
		// ---------------------------------------------------------------------
		sm.setSecurityHeaders(w)

		// ---------------------------------------------------------------------
		// SCHRITT 4: CORS Headers
		// Behandelt Cross-Origin-Requests und setzt entsprechende Header.
		// ---------------------------------------------------------------------
		sm.handleCORS(w, r)

		// ---------------------------------------------------------------------
		// SCHRITT 5: Preflight Requests (OPTIONS)
		// Browser senden OPTIONS-Requests vor Cross-Origin-Requests.
		// Diese werden hier mit 204 No Content beantwortet.
		// ---------------------------------------------------------------------
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		// Alle Prüfungen bestanden - Request an den eigentlichen Handler weiterleiten
		next.ServeHTTP(w, r)
	})
}

// =============================================================================
// SECURITY HEADERS
// =============================================================================

// setSecurityHeaders setzt alle empfohlenen HTTP-Security-Header.
// Diese Header schützen vor verschiedenen Angriffen wie XSS, Clickjacking, etc.
func (sm *SecurityMiddleware) setSecurityHeaders(w http.ResponseWriter) {

	// X-Content-Type-Options: nosniff
	// Verhindert MIME-Type-Sniffing durch den Browser.
	// Browser interpretieren Dateien nur gemäß Content-Type Header.
	w.Header().Set("X-Content-Type-Options", "nosniff")

	// X-Frame-Options: DENY
	// Verhindert Clickjacking-Angriffe.
	// Die Seite kann nicht in iframes eingebettet werden.
	w.Header().Set("X-Frame-Options", "DENY")

	// X-XSS-Protection: 1; mode=block
	// Aktiviert den XSS-Filter in älteren Browsern.
	// Bei erkanntem XSS wird die Seite blockiert statt bereinigt.
	w.Header().Set("X-XSS-Protection", "1; mode=block")

	// Referrer-Policy: strict-origin-when-cross-origin
	// Kontrolliert welche Referrer-Informationen gesendet werden.
	// Bei Cross-Origin nur der Origin, bei Same-Origin die volle URL.
	w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")

	// Permissions-Policy (ehemals Feature-Policy)
	// Kontrolliert Browser-Features.
	// Fleet Navigator benötigt Mikrofon für Voice-Input (STT).
	w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(self), camera=()")
}

// =============================================================================
// CORS (Cross-Origin Resource Sharing)
// =============================================================================

// handleCORS verarbeitet CORS-Requests sicher.
// Nur Origins aus der Whitelist erhalten Zugriff.
func (sm *SecurityMiddleware) handleCORS(w http.ResponseWriter, r *http.Request) {
	origin := r.Header.Get("Origin")

	// Origin gegen Whitelist prüfen
	// Nur explizit erlaubte Origins erhalten Access-Control-Allow-Origin
	if origin != "" && sm.isAllowedOrigin(origin) {
		// Access-Control-Allow-Origin: Erlaubt Cross-Origin-Requests von diesem Origin
		w.Header().Set("Access-Control-Allow-Origin", origin)

		// Access-Control-Allow-Credentials: Erlaubt Cookies/Auth bei Cross-Origin
		w.Header().Set("Access-Control-Allow-Credentials", "true")
	}
	// WICHTIG: Bei unbekanntem Origin wird KEIN Allow-Origin gesetzt!
	// Der Browser blockiert dann den Cross-Origin-Request.

	// Erlaubte HTTP-Methoden für CORS
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")

	// Erlaubte HTTP-Header für CORS
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Request-ID")

	// Max-Age: Browser darf CORS-Preflight-Ergebnis 24h cachen
	// Reduziert die Anzahl der OPTIONS-Requests
	w.Header().Set("Access-Control-Max-Age", "86400")
}

// isAllowedOrigin prüft ob ein Origin in der Whitelist ist.
// Exakter String-Vergleich - keine Wildcards oder Patterns.
func (sm *SecurityMiddleware) isAllowedOrigin(origin string) bool {
	for _, allowed := range sm.config.AllowedOrigins {
		if origin == allowed {
			return true
		}
	}
	return false
}

// AddAllowedOrigin fügt zur Laufzeit einen neuen erlaubten Origin hinzu.
// Nützlich für dynamische Konfiguration oder Tests.
func (sm *SecurityMiddleware) AddAllowedOrigin(origin string) {
	sm.config.AllowedOrigins = append(sm.config.AllowedOrigins, origin)
}

// =============================================================================
// CLIENT-IP ERKENNUNG
// =============================================================================

// getClientIP extrahiert die echte Client-IP aus dem Request.
// Berücksichtigt Proxy-Header in dieser Reihenfolge:
//  1. X-Forwarded-For (Standard für Proxies)
//  2. X-Real-IP (Nginx-Standard)
//  3. RemoteAddr (direkte Verbindung)
func getClientIP(r *http.Request) string {
	// X-Forwarded-For: Kann mehrere IPs enthalten (Client, Proxy1, Proxy2, ...)
	// Die erste IP ist der ursprüngliche Client
	xff := r.Header.Get("X-Forwarded-For")
	if xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}

	// X-Real-IP: Einzelne IP vom Proxy gesetzt
	xri := r.Header.Get("X-Real-IP")
	if xri != "" {
		return xri
	}

	// RemoteAddr: Direkte Verbindung (Format: "IP:Port")
	// Port muss entfernt werden
	ip := r.RemoteAddr
	if idx := strings.LastIndex(ip, ":"); idx != -1 {
		ip = ip[:idx]
	}
	return ip
}

// =============================================================================
// RATE LIMITER (Token Bucket Algorithmus)
// =============================================================================

// RateLimiter implementiert Rate Limiting mit dem Token Bucket Algorithmus.
//
// Funktionsweise:
//   - Jede IP hat einen "Bucket" mit Tokens
//   - Tokens werden mit konstanter Rate aufgefüllt (rate pro Sekunde)
//   - Jeder Request verbraucht 1 Token
//   - Bei 0 Tokens wird der Request abgelehnt
//   - Burst erlaubt kurzzeitige Überschreitung
//
// Thread-Safety: Alle Operationen sind durch Mutex geschützt.
type RateLimiter struct {
	mu      sync.Mutex               // Mutex für Thread-Safety
	buckets map[string]*tokenBucket  // Bucket pro Client-IP
	rate    int                      // Tokens pro Sekunde (Auffüllrate)
	burst   int                      // Maximale Tokens im Bucket
	cleanup time.Duration            // Intervall für Bucket-Cleanup
}

// tokenBucket speichert den Zustand für eine einzelne IP.
type tokenBucket struct {
	tokens     float64   // Aktuelle Anzahl Tokens (kann Dezimalstellen haben)
	lastUpdate time.Time // Zeitpunkt der letzten Aktualisierung
}

// NewRateLimiter erstellt einen neuen Rate Limiter.
// Startet automatisch eine Hintergrund-Goroutine für Cleanup.
//
// Parameter:
//   - ratePerSec: Tokens die pro Sekunde aufgefüllt werden
//   - burst: Maximale Anzahl Tokens (erlaubt kurze Lastspitzen)
func NewRateLimiter(ratePerSec, burst int) *RateLimiter {
	rl := &RateLimiter{
		buckets: make(map[string]*tokenBucket),
		rate:    ratePerSec,
		burst:   burst,
		cleanup: 10 * time.Minute, // Inaktive Buckets nach 10 Min entfernen
	}

	// Cleanup-Goroutine im Hintergrund starten
	// Entfernt regelmäßig inaktive Buckets um Speicher freizugeben
	go rl.cleanupLoop()

	return rl
}

// Allow prüft ob ein Request von der gegebenen IP erlaubt ist.
// Gibt true zurück wenn erlaubt, false wenn Rate Limit überschritten.
//
// Algorithmus:
//  1. Bucket für IP suchen oder neu erstellen
//  2. Tokens basierend auf vergangener Zeit auffüllen
//  3. 1 Token verbrauchen wenn verfügbar
func (rl *RateLimiter) Allow(clientIP string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()

	// Bucket für diese IP suchen oder neu erstellen
	bucket, exists := rl.buckets[clientIP]
	if !exists {
		// Neue IP: Bucket mit vollem Burst-Kontingent erstellen
		bucket = &tokenBucket{
			tokens:     float64(rl.burst),
			lastUpdate: now,
		}
		rl.buckets[clientIP] = bucket
	}

	// Tokens auffüllen basierend auf vergangener Zeit
	// Beispiel: Bei rate=100 und 0.5s vergangen -> 50 Tokens hinzufügen
	elapsed := now.Sub(bucket.lastUpdate).Seconds()
	bucket.tokens += elapsed * float64(rl.rate)

	// Nicht mehr als Burst-Maximum erlauben
	if bucket.tokens > float64(rl.burst) {
		bucket.tokens = float64(rl.burst)
	}
	bucket.lastUpdate = now

	// Prüfen ob genug Tokens für diesen Request vorhanden
	if bucket.tokens >= 1 {
		bucket.tokens-- // 1 Token verbrauchen
		return true     // Request erlaubt
	}

	return false // Rate Limit überschritten
}

// cleanupLoop läuft im Hintergrund und entfernt inaktive Buckets.
// Verhindert Speicherlecks bei vielen verschiedenen Client-IPs.
func (rl *RateLimiter) cleanupLoop() {
	ticker := time.NewTicker(rl.cleanup)
	for range ticker.C {
		rl.mu.Lock()
		// Threshold: Buckets die länger als cleanup-Intervall inaktiv sind
		threshold := time.Now().Add(-rl.cleanup)
		for ip, bucket := range rl.buckets {
			if bucket.lastUpdate.Before(threshold) {
				delete(rl.buckets, ip) // Inaktiven Bucket entfernen
			}
		}
		rl.mu.Unlock()
	}
}

// =============================================================================
// HILFSFUNKTIONEN FÜR SICHERES LOGGING
// =============================================================================

// SanitizeForLog entfernt sensible Daten aus Strings für sicheres Logging.
// Verhindert dass Passwörter, API-Keys etc. in Logs landen.
//
// Parameter:
//   - input: Der zu bereinigende String
//   - maxLen: Maximale Länge (längere Strings werden gekürzt)
func SanitizeForLog(input string, maxLen int) string {
	if len(input) > maxLen {
		return "[TRUNCATED: " + string(rune(len(input))) + " chars]"
	}
	// Bei sensiblem Inhalt: Komplett maskieren
	// TODO: Hier könnten Pattern-Matches für Passwörter, Keys etc. ergänzt werden
	return "[CONTENT HIDDEN]"
}

// SafeErrorMessage gibt eine sichere Fehlermeldung für Clients zurück.
// Der echte Fehler wird intern geloggt, der Client sieht nur die generische Nachricht.
//
// Verhindert Information Leakage durch detaillierte Fehlermeldungen.
func SafeErrorMessage(err error, publicMsg string) string {
	if err != nil {
		// Echten Fehler intern loggen (für Debugging)
		log.Printf("Interner Fehler: %v", err)
	}
	// Generische Nachricht an Client zurückgeben
	return publicMsg
}
