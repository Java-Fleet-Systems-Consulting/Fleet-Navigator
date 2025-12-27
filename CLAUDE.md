# CLAUDE.md - Fleet Navigator Go

## Git-Konfiguration

**Repository:** `Java-Fleet-Systems-Consulting/Fleet-Navigator`
**SSH-Host:** `github-kofi`
**Committer:** Kofi Mensah <kofi.mensah@Java-fleet.com>

```bash
# Remote-URL Format:
git remote set-url origin git@github-kofi:Java-Fleet-Systems-Consulting/Fleet-Navigator.git
```

---

## Projekt-Übersicht

**Fleet Navigator Go** ist die Go-Portierung des Fleet Navigator - ein KI-gestütztes Experten-System für kleine Büros. Die Anwendung kombiniert ein Go-Backend mit einem Vue.js-Frontend in einer einzigen ausführbaren Datei.

## Architektur

```
Fleet Navigator Go
├── Go Backend (Port 2026)
│   ├── HTTP Server (API + Static Files)
│   ├── WebSocket Server (Mate-Kommunikation)
│   ├── Ollama Integration (Chat/Streaming)
│   └── Security (Ed25519 Pairing)
└── Vue.js Frontend (Embedded)
    ├── Chat View
    ├── Dashboard/Mates View
    └── TailwindCSS Styling
```

## Verzeichnisstruktur

```
Fleet-Navigator-Go/
├── cmd/navigator/
│   ├── main.go              # Hauptanwendung
│   └── dist/                # Embedded Frontend (nach Build)
├── internal/
│   ├── chat/
│   │   ├── ollama.go        # Ollama Chat Service
│   │   ├── adapter.go       # WebSocket-Chat-Bridge
│   │   └── store.go         # Chat-Persistenz (SQLite)
│   ├── config/
│   │   └── config.go        # Konfigurationsverwaltung
│   ├── experte/
│   │   ├── expert.go        # Experten-Datenmodelle
│   │   ├── repository.go    # SQLite Repository
│   │   └── service.go       # Experten-Service
│   ├── llm/                 # LLM Provider Abstraktion
│   │   ├── provider.go      # Provider Interface
│   │   ├── ollama_provider.go # Ollama Implementation
│   │   ├── registry.go      # Model Registry (Katalog)
│   │   └── service.go       # Model Service
│   ├── mate/
│   │   ├── mate.go          # Mate-Typen & Capabilities
│   │   └── handler.go       # Action-Handler
│   ├── models/
│   │   └── selection.go     # Smart Model Selection
│   ├── security/
│   │   ├── pairing.go       # Mate-Pairing Manager
│   │   ├── keypair.go       # Ed25519 Kryptographie
│   │   └── encryption.go    # AES-256 Verschlüsselung
│   ├── tools/               # NEU: KI-Tool-System
│   │   ├── tool.go          # Tool Interface & BaseTool
│   │   ├── registry.go      # Tool Registry
│   │   ├── websearch.go     # DuckDuckGo Web-Suche
│   │   ├── webfetch.go      # URL-Inhalte abrufen
│   │   └── filesearch.go    # Dateisuche (Mate-basiert)
│   ├── vision/              # NEU: Bildanalyse
│   │   └── vision.go        # LLaVA Vision Service
│   ├── websocket/
│   │   ├── server.go        # WebSocket Server
│   │   └── coder_modes.go   # Automatische Sprach-Erkennung für Coder
│   └── updater/
│       └── updater.go       # Auto-Update System
├── web/                     # Vue.js Frontend Source
│   ├── src/
│   │   ├── App.vue
│   │   ├── views/
│   │   │   ├── Chat.vue
│   │   │   └── Dashboard.vue
│   │   └── composables/
│   │       └── useWebSocket.js
│   └── package.json
├── configs/                 # Konfigurationsdateien
├── dist/                    # Kompilierte Binary
├── build.sh                 # Build-Script
├── fleet-navigator.service  # Systemd Service File
├── install-service.sh       # Service Installations-Script
├── go.mod
└── go.sum
```

## Build & Run

### Voraussetzungen

- Go 1.24+
- Node.js 18+ (für Frontend-Build)
- Ollama (lokal installiert)

### Development

```bash
# Backend starten (ohne Frontend)
cd cmd/navigator
DEV=1 go run main.go

# Frontend separat (Hot-Reload)
cd web
npm install
npm run dev
```

### Production Build

```bash
# Vollständiger Build (Backend + Frontend)
./build.sh

# Oder manuell:
cd web && npm install && npm run build && cd ..
cd cmd/navigator
cp -r ../../web/dist ./dist
go build -o ../../dist/fleet-navigator .

# Starten
./dist/fleet-navigator
```

### Flags & Umgebungsvariablen

```bash
# Flags
fleet-navigator -port=2026 -data=/path/to/data

# Umgebungsvariablen
PORT=2026                           # Server-Port (Default: 2026)
OLLAMA_URL=http://localhost:11434  # Ollama API URL
OLLAMA_MODEL=qwen2.5:7b            # Standard-Modell
DEV=1                               # Development-Modus
```

### Systemd Deployment (Production)

Für den Produktivbetrieb steht ein systemd Service bereit:

```bash
# Service installieren
sudo ./install-service.sh

# Service verwalten
sudo systemctl start fleet-navigator    # Starten
sudo systemctl stop fleet-navigator     # Stoppen
sudo systemctl restart fleet-navigator  # Neustarten
sudo systemctl status fleet-navigator   # Status
sudo journalctl -u fleet-navigator -f   # Logs verfolgen
```

**Service-Konfiguration** (`/etc/systemd/system/fleet-navigator.service`):
- Automatischer Neustart bei Fehler
- Sicherheits-Härtung (ProtectSystem, PrivateTmp)
- Ressourcen-Limits (max 2GB RAM, 200% CPU)
- Logging via journald

**Installation kopiert:**
- Binary nach `/opt/fleet-navigator/`
- Frontend-Dist (falls vorhanden)
- Konfiguration aus `configs/`

### Graceful Shutdown

Der Server fährt bei SIGINT (Ctrl+C) oder SIGTERM sauber herunter:

1. Signal wird empfangen
2. Neue Verbindungen werden abgelehnt
3. Laufende Requests haben 10 Sekunden Zeit
4. llama-server wird beendet (falls aktiv)
5. HTTP-Server wird geschlossen

```bash
# Sauberes Beenden (empfohlen)
kill -TERM $(pgrep fleet-navigator)

# Oder via systemd
sudo systemctl stop fleet-navigator
```

### CORS Konfiguration

Cross-Origin Resource Sharing ist integriert für Frontend-Entwicklung:

**Erlaubte Origins (Production):**
- `http://localhost:5173` (Vite Dev Server)
- `http://localhost:2025` (Fleet Navigator)
- `http://localhost:2026` (Fleet Navigator Alt-Port)
- `http://127.0.0.1:*` (localhost-Varianten)

**Development-Modus (`DEV=1`):**
- Alle Origins werden akzeptiert
- Nützlich für lokale Entwicklung

**CORS-Header:**
```
Access-Control-Allow-Origin: <origin>
Access-Control-Allow-Methods: GET, POST, PUT, PATCH, DELETE, OPTIONS
Access-Control-Allow-Headers: Content-Type, Authorization, X-Requested-With
Access-Control-Allow-Credentials: true
Access-Control-Max-Age: 86400  (24h Preflight-Cache)
```

## API Endpoints

### REST API

| Endpoint | Method | Beschreibung |
|----------|--------|--------------|
| `/api/health` | GET | Health-Check mit Status |
| `/api/models` | GET | Verfügbare Ollama-Modelle |
| `/api/models/config` | GET/POST | Smart Selection Konfiguration |
| `/api/mates` | GET | Liste aller Trusted Mates |
| `/api/mates/pending` | GET | Ausstehende Pairing-Anfragen |
| `/api/mates/approve` | POST | Pairing bestätigen |
| `/api/mates/reject` | POST | Pairing ablehnen |
| `/api/mates/remove` | POST | Mate entfernen |
| `/api/config` | GET | Navigator-Konfiguration |

### LLM Model Management API (NEU)

| Endpoint | Method | Beschreibung |
|----------|--------|--------------|
| `/api/llm/status` | GET | LLM-System Status |
| `/api/llm/models` | GET | Alle Modelle (installiert + Registry) |
| `/api/llm/models/installed` | GET | Nur installierte Modelle |
| `/api/llm/models/registry` | GET | Model-Registry (Katalog) |
| `/api/llm/models/featured` | GET | Featured & Trending Modelle |
| `/api/llm/models/pull` | POST | Modell herunterladen (SSE) |
| `/api/llm/models/delete` | POST | Modell löschen |
| `/api/llm/models/details/{name}` | GET | Modell-Details |
| `/api/llm/chat` | POST | Chat mit LLM (SSE Streaming) |

### Chat-Streaming SSE (`/api/chat/send-stream`) - WICHTIG

Server-Sent Events Protokoll für Chat-Streaming:

```javascript
// 1. Start-Event (PFLICHT - Frontend setzt currentChat!)
data: {"chatId":123,"requestId":"req-xxx"}

// 2. Optional: Mode-Switch Event (bei Experten-Wechsel)
data: {"type":"mode_switch","message":"...","newModeId":5}

// 3. Content-Chunks (Streaming)
data: {"content":"Hallo","done":false}
data: {"content":" Welt","done":false}
data: {"content":"!","done":true}

// 4. Done-Event (PFLICHT - Frontend erwartet tokens!)
data: {"tokens":42}

// Bei Fehler:
data: {"error":"Fehler-Nachricht","done":true}
```

**Wichtig:**
- Ohne Start-Event mit `chatId` zeigt das Frontend keine Nachrichten an!
- Wenn `chatId: 0` oder `null` gesendet wird, erstellt der Server automatisch einen neuen Chat
- Das Start-Event enthält dann die neue `chatId` (z.B. `{"chatId": 2, ...}`)
- JavaScript behandelt `0` als falsy - daher ist Auto-Create essentiell für neue Chats

### Frontend-Kompatibilitäts-Endpoints (Java-Migration)

Endpoints für Kompatibilität mit dem Java-Frontend:

| Go Endpoint | Java Equivalent | Funktion |
|------------|-----------------|----------|
| `/api/system/health` | `/api/health` | Health-Check Alias |
| `/api/fleet-mate/mates` | `/api/mates` | Mates-Liste Alias |
| `/api/stats/global` | - | Globale Statistiken |
| `/api/models/custom` | - | Custom-Modelle (Stub) |
| `/api/templates` | - | Prompt-Vorlagen |
| `/api/projects` | - | Projekte (Stub) |

### WebSocket (`/ws`)

**Nachrichtentypen:**

```javascript
// Pairing
{ type: "pairing_request", payload: { mate_name, mate_type, mate_public_key } }
{ type: "pairing_response", payload: { request_id, navigator_public_key, pairing_code } }
{ type: "pairing_approved", payload: { mate_id, mate_name } }
{ type: "pairing_rejected", payload: { request_id } }

// Authentifizierung
{ type: "auth", payload: { mate_id, public_key, signature, nonce } }
{ type: "auth_success", payload: { mate_id, mate_name } }
{ type: "auth_failed", payload: { error } }

// Chat
{ type: "chat", payload: { session_id, message, stream } }
{ type: "chat_stream", payload: { session_id, content, done } }
{ type: "chat_done", payload: { session_id, done: true } }
{ type: "chat_error", payload: { error } }
{ type: "chat_clear", payload: { session_id } }

// System
{ type: "ping" }
{ type: "pong" }
{ type: "error", payload: { error } }
```

## Sicherheitskonzept: Mate-Pairing

Das Pairing funktioniert wie Bluetooth-Geräte:

1. **Mate sendet Pairing-Request** mit Public Key
2. **Navigator zeigt Bestätigungs-Code** (6 Ziffern)
3. **User bestätigt** in der Web-UI
4. **Schlüsselaustausch** wird abgeschlossen
5. **Verschlüsselte Kommunikation** ab jetzt

### Kryptographie

- **Schlüsselpaar:** Ed25519 (Signatur)
- **Key Exchange:** X25519 (ECDH)
- **Verschlüsselung:** AES-256-GCM
- **Pairing-Code:** SHA256(mate_pubkey + nav_pubkey)[:6]

## Smart Model Selection

Automatische Modellauswahl basierend auf Prompt-Inhalt:

| Task-Typ | Modell | Erkennung |
|----------|--------|-----------|
| CODE | qwen2.5-coder:7b | Code-Keywords, Syntax |
| SIMPLE_QA | llama3.2:3b | Kurze Fragen, "was ist" |
| COMPLEX | qwen2.5:7b | Standard |
| VISION | llava:13b | Bildanalyse |

## Daten-Verzeichnis

Standard: `~/.fleet-navigator/`

```
~/.fleet-navigator/
├── navigator_keys.json      # Ed25519 Schlüsselpaar
├── trusted_mates.json       # Vertraute Mates
└── (TODO: experts.db)       # SQLite für Experten
```

## Implementierte Module

### Experten-System (`internal/experte/`)

Vollständiges Experten-System mit:
- **expert.go**: Datenmodelle für Expert und ExpertMode
- **repository.go**: SQLite-Repository mit CRUD-Operationen
- **service.go**: Business-Logic und Caching

**API Endpoints:**
```
GET    /api/experts           # Alle Experten
GET    /api/experts/{id}      # Einzelner Experte
POST   /api/experts           # Experte erstellen
PUT    /api/experts/{id}      # Experte aktualisieren
DELETE /api/experts/{id}      # Experte löschen
GET    /api/experts/{id}/modes    # Modi eines Experten
POST   /api/experts/{id}/modes    # Modus hinzufügen
```

**Standard-Experten:**
- Roland (Rechtsberater) - Rechtliche Beratung
- Maria (Marketing) - Marketing & Kommunikation
- Thomas (IT-Berater) - IT & Digitalisierung

**Anti-Halluzinations-System (NEU 2025-12-25):**
- Konfigurierbarer Anti-Halluzinations-Prompt pro Experte
- Default-Prompt mit Identitäts- und Quellen-Schutz
- API: `GET /api/experts/default-anti-hallucination` für Reset auf Standard
- Feld `antiHallucinationPrompt` in Expert-Struct (leer = Default verwenden)
- WebSearchShowLinks Default auf `false` (RAG-Modus ohne sichtbare Quellen)

**DELEGATE-System (NEU 2025-12-25):**

Automatische Experten-Umschaltung wenn ein Experte delegiert:

```
Format: [[DELEGATE:ExpertName]]

Beispiel:
User: "Ich habe ein Problem mit einem Mietvertrag"
Ewa: "Das klingt nach Mietrecht. Roland kann dir besser helfen."
     [[DELEGATE:Roland]]

→ Tag wird geparst und aus Antwort entfernt
→ SSE "delegation" Event ans Frontend
→ Frontend wechselt automatisch zu Roland
```

**SSE Event Format:**
```json
{
  "type": "delegation",
  "expertId": 2,
  "expertName": "Roland",
  "expertAvatar": "👨‍⚖️",
  "message": "Ich verbinde dich mit Roland..."
}
```

**Experten-Suche:** Findet Experten nach vollem Namen oder Vornamen (case-insensitive)

### Mate-Logik (`internal/mate/`)

- **mate.go**: Mate-Typen und Capabilities
- **handler.go**: Action-Handler für Mate-Requests

**Unterstützte Mate-Typen:**
- `writer` - LibreOffice Writer
- `mail` - Thunderbird
- `outlook` - Microsoft Outlook
- `web-search` - Web-Recherche
- `browser` - Browser-Extension
- `coder` - FleetCoder (mit automatischer Modus-Erkennung)

### Coder-Modus-System (`internal/websocket/coder_modes.go`) - NEU

Automatische Erkennung der Programmiersprache für Coder-Mates:

**Unterstützte Modi:**
| Modus | Icon | Dateiendungen | Sichere Keywords |
|-------|------|---------------|------------------|
| Go | 🐹 | `.go`, `go.mod` | golang, goroutine |
| Java | ☕ | `.java`, `pom.xml` | spring boot, hibernate |
| Python | 🐍 | `.py`, `requirements.txt` | pytest, django, flask |
| JavaScript/TS | 🟨 | `.js`, `.ts`, `package.json` | typescript, react, vue |
| Bash/Shell | 🐚 | `.sh`, `.bash` | bash script, shellcheck |
| Rust | 🦀 | `.rs`, `Cargo.toml` | cargo build, rustc |
| SQL | 🗃️ | `.sql` | mysql, postgresql, sqlite |
| DevOps | 🔧 | `Dockerfile`, `.tf` | kubernetes, terraform |
| PowerShell | 🔷 | `.ps1`, `.psm1` | powershell, cmdlet |
| Batch | 🪟 | `.bat`, `.cmd` | batch script, @echo off |

**Erkennungs-Priorität:**
1. **Dateiendungen** (höchste Sicherheit) - z.B. `main.go` → Go-Modus
2. **Sichere Keywords** - z.B. "golang", "spring boot"
3. **Regex-Patterns** - Kombinationen wie "code in go"
4. **Nachfrage** bei Unsicherheit - z.B. "go" alleine → "Meinst du Go (Golang)?"

**Ambige Keywords (lösen Nachfrage aus):**
- `go` → "Meinst du die Programmiersprache Go (Golang)?"
- `java` → "Meinst du Java oder JavaScript?"
- `script` → "In welcher Sprache soll das Script sein?"
- `c` → "Meinst du C, C++ oder C#?"

**Integration in server.go:**
- Erkennung in `handleChat()` für Mate-Type "coder"
- Modus wird in `trusted_mates.json` persistiert (Feld: `activeMode`)
- Mode-Switch wird als Chat-Stream-Event gesendet

### Config-System (`internal/config/`)

JSON-basierte Konfiguration mit:
- Server-Einstellungen
- Ollama-Konfiguration
- Model Selection
- Security-Settings
- Logging

**Konfigurationsdatei:** `configs/config.json`

### Tool-System (`internal/tools/`) - NEU

KI-gestützte Tools für erweiterte Funktionalität:

**Architektur:**
```
ToolRegistry
├── WebSearchTool (DuckDuckGo)
│   └── Instant Answer API + HTML Fallback
├── WebFetchTool (URL-Inhalte)
│   └── HTML-zu-Text Extraktion
└── FileSearchTool (benötigt Mate)
    └── Dateisuche via verbundenem Mate
```

**API Endpoints:**
```
GET  /api/tools              # Verfügbare Tools
POST /api/tools/execute      # Tool ausführen
POST /api/tools/search       # Web-Suche (DuckDuckGo)
POST /api/tools/fetch        # URL-Inhalt abrufen
```

**FileSearch:**
- Benötigt einen verbundenen Mate für Dateizugriff
- Unterstützt Dateinamen- und Inhaltssuche
- Filtert nach Dateitypen
- Setzt MateConnection Interface voraus

### Vision-System + Tesseract OCR (`internal/vision/`) - AKTUALISIERT 2024-12-24

**Kombinierte Vision + Tesseract Analyse:**
- **Vision (LLaVA)**: Analysiert Logos, Stempel, Unterschriften, Layout
- **Tesseract OCR**: Extrahiert ALLEN Text (unbegrenzt Seiten!)
- **Beide zusammen**: Vollständige Dokumentanalyse für DB-Speicherung

**WICHTIG - Strategie für große Dokumente:**
1. Tesseract extrahiert Text aus ALLEN Seiten (keine Begrenzung)
2. Vision analysiert nur erste Seite (Layout, visuelle Elemente)
3. Kombiniertes Ergebnis wird in Chat-DB gespeichert
4. Bei späteren Anfragen: Text bereits in DB, kein erneutes OCR nötig

**Features:**
- Bildanalyse mit deutschem Prompt
- **Smart-Analyse**: Automatische Erkennung ob Text-Dokument oder Bild
- Dokumentenerkennung (Rechnungen, Briefe, Verträge, etc.)
- **Tesseract OCR**: Unbegrenzte Textextraktion
- **Vision-Validierung**: Vision korrigiert OCR-Fehler (Zahlen, Beträge)
- PDF-zu-Bild Konvertierung (benötigt poppler-utils)
- **PDF-Streaming**: Fortschrittsanzeige für große PDFs (89+ Seiten)

**API Endpoints:**
```
POST /api/vision/analyze     # Bild analysieren (Streaming)
POST /api/vision/document    # Dokument mit Vision+Tesseract analysieren
POST /api/vision/pdf-stream  # PDF mit Fortschrittsanzeige (SSE)
GET  /api/vision/status      # Vision+Tesseract Status
POST /api/vision/ocr         # Reine Text-Extraktion (OCR)
POST /api/vision/smart       # Smart-Analyse (Text vs. Bild automatisch)
POST /api/vision/classify    # Schnelle Klassifizierung (Dokumenttyp)
```

**Tesseract Integration:**
```
~/.fleet-navigator/tesseract/
├── tesseract(.exe)          # Binary
├── tessdata/
│   ├── deu.traineddata      # Deutsch
│   ├── eng.traineddata      # Englisch
│   └── tur.traineddata      # Türkisch
└── [weitere DLLs/libs]
```

**Mirror-Dateien für Tesseract:**
- `tesseract-ocr-windows-x64.zip` (~80 MB)
- `tesseract-ocr-linux-x64.tar.gz` (~70 MB)
- `tesseract-ocr-macos-arm64.tar.gz` (~70 MB)

**Erkannte Dokumenttypen:**
- `invoice` - Rechnung
- `contract` - Vertrag
- `letter` - Brief
- `form` - Formular
- `receipt` - Quittung/Beleg
- `id_card` - Ausweis
- `business_card` - Visitenkarte
- `photo` - Foto (kein Dokument)
- `diagram` - Diagramm/Grafik
- `screenshot` - Screenshot

**Vision-Modelle für llama-server:**
- LLaVA-v1.6-Mistral-7B (empfohlen) + mmproj
- MiniCPM-V-2.6
- Andere GGUF-Modelle mit multimodal Support

**Server-Start mit Vision:**
```bash
# llama-server startet automatisch mit --mmproj wenn:
# 1. Modellname enthält "llava", "vision" oder "minicpm"
# 2. mmproj-Datei im gleichen Verzeichnis gefunden wird
```

### LLM Provider System (`internal/llm/`) - NEU

Abstrahiertes LLM-System wie in der Java-Version:

**Architektur:**
```
ProviderManager
├── OllamaProvider (implementiert)
│   └── HTTP zu Ollama Server
└── LlamaServerProvider (implementiert)
    └── llama-server Prozess (automatisch gestartet)

ModelRegistry
├── Chat-Modelle (qwen2.5, llama3.2, mistral, phi3)
├── Code-Modelle (qwen2.5-coder, deepseek-coder)
├── Vision-Modelle (llava)
└── Compact-Modelle (< 4GB RAM)

ModelService
├── Kombiniert Provider + Registry
├── Automatische Modell-Erkennung
└── Modell-Download via Ollama
```

**Unterstützte Modelle (Registry):**
- Qwen 2.5 (1.5B, 3B, 7B) - Mehrsprachig, exzellentes Deutsch
- Llama 3.2 (1B, 3B) - Meta AI, kompakt
- Qwen 2.5 Coder (3B, 7B) - Code-Generierung
- DeepSeek Coder (1.3B, 6.7B) - State-of-the-Art Code
- LLaVA 1.6 (7B) - Vision/Bildanalyse
- Mistral 7B, Phi-3 Mini - Allrounder

## Noch offen (Verbesserungen)

### Internationalisierung (i18n)
- [x] Frontend DE/EN - Vollständig (1563 Keys)
- [x] Frontend TR - Vollständig (1569 Keys, inkl. türkische Stimmen)
- [ ] Frontend FR - Unvollständig (~41%, ~640/1563 Keys)
- [ ] Frontend ES - Unvollständig (~26%, ~400/1563 Keys)
- [x] Backend-Prompts DE/EN/TR - Experten-System vollständig übersetzt
- [x] Multilinguale Prompts: Verstehen Input in jeder Sprache, Output in Benutzersprache
  - Deutsche Begriffe werden mit Übersetzung zitiert: "Mahnung (payment reminder)"

### Sprachwechsel zur Laufzeit ✅ (Implementiert 2025-12-24)

**Implementiert:**
- [x] API-Endpoint: `GET/POST /api/settings/language` (cmd/navigator/main.go:7700)
- [x] Experten-Prompts dynamisch laden: `GetChatContextWithLocale()` (internal/experte/service.go:425)
- [x] Sprache in Settings-DB persistieren: `SaveLocale()` / `GetLocale()`
- [x] Frontend-Integration: `useLocale.js` ruft Backend bei Sprachwechsel auf
- [x] TTS-Stimmen-Info: Backend gibt verfügbare/installierte Stimmen zurück
  - DE: Thorsten, Kerstin
  - TR: Fahrettin, Fettah
  - EN: Amy, Ryan
- [x] Wake Words: "Ewa" / "Hey Ewa" funktioniert sprachübergreifend (Name)

**Noch offen:**
- [x] TTS-Stimmen: Automatischer Download-Dialog bei Sprachwechsel (SettingsModal.vue)
- [ ] Chat-Context: UI-Hinweis dass neue Chats in neuer Sprache sind

**Ablauf bei Sprachwechsel:**
```
1. User wählt TR im Frontend (SettingsModal)
2. useLocale.setLocale('tr') ruft POST /api/settings/language auf
3. Backend: Speichert in Settings-DB, gibt Stimmen-Info zurück
4. Frontend: Dispatcht 'locale-changed' Event
5. Nächster Chat: GetChatContextWithLocale() verwendet TR-Prompts
```

### Voice-System (internal/voice/)
- [ ] Echte Sound-Dateien laden (Aktivierung, Deaktivierung, Fehler)
- [ ] Chat-Engine Integration (Antworten vom LLM holen)
- [ ] Echte Audio-Ausgabe implementieren (TTS-Abspielen)
- [ ] Sound-Dateien einbetten (embed.go)
- [ ] Voice-System für mehrsprachige Prompts anpassen (DE/EN/TR)

### Tesseract OCR (Dokumentenverarbeitung)
**Wichtig für Experten-System:** Ermöglicht Verarbeitung von mehr als 4 gescannten A4-Seiten!

**Bereits implementiert:**
- [x] Download-Funktion vom Mirror (`internal/setup/handlers.go:DownloadTesseract`)
  - Windows: `tesseract-ocr-windows-x64.zip` (portable)
  - Linux: `tesseract-ocr-linux-x64.tar.gz`
  - macOS: Nicht unterstützt (Hinweis auf `brew install tesseract`)
- [x] Installation nach `~/.fleet-navigator/tesseract/`
- [x] API-Endpoint: `POST /api/vision/ocr` - Reine OCR (Tesseract direkt)
- [x] API-Endpoint: `POST /api/vision/document` - Vision + Tesseract kombiniert
- [x] Tesseract-Binary wird aufgerufen (`vision.TesseractOCRFromBase64()`)
- [x] Vision validiert kritische OCR-Stellen (Beträge, Zahlen)
- [x] Status-Endpoint: `GET /api/vision/status` zeigt Tesseract-Installation

**Noch offen:**
- [ ] Sprachpakete: DEU, ENG, TUR auf **Mirror bereitstellen** (Dateien fehlen!)
- [ ] Setup-Wizard UI: Tesseract-Installation als Option anbieten
- [ ] PDF-OCR: Tesseract in `handleVisionPDFStream` integrieren (aktuell nur Vision)

**Vorteile gegenüber reiner Vision:**
- Keine Token-Limits für Text-Dokumente (unbegrenzte Seitenzahl!)
- Schneller als Vision-Modell
- Weniger VRAM-Verbrauch
- Roland, Franziska & Co. können komplette Verträge/Dokumente analysieren

### Sonstige
- [x] ~~llama.cpp Provider~~ → llama-server implementiert (go-llama.cpp verworfen - zu hoher Wartungsaufwand)
- [ ] Vollständige Signatur-Verifikation bei Auth
- [x] CORS Konfiguration (implementiert in main.go)
- [ ] Rate Limiting
- [ ] Strukturiertes Logging-System
- [x] Graceful Shutdown (implementiert in main.go)
- [x] Systemd Service File (fleet-navigator.service + install-service.sh)

## Unterschiede zur Java-Version

| Feature | Java (Spring Boot) | Go |
|---------|-------------------|-----|
| Binary-Größe | ~50 MB + JRE | ~10 MB |
| Startup | ~3-5 Sekunden | Instant |
| RAM-Verbrauch | ~200-500 MB | ~20-50 MB |
| Dependencies | Maven, viele | Minimal (2) |
| Experten-System | ✅ Vollständig | ✅ Implementiert |
| LLM Provider | ✅ Ollama + llama.cpp | ✅ Ollama + llama-server |
| Model Registry | ✅ Vollständig | ✅ Implementiert |
| Tool-System | ✅ WebSearch, FileSearch | ✅ Implementiert |
| Vision/LLaVA | ✅ Vollständig | ✅ Implementiert |
| Chat-Streaming | ✅ SSE | ✅ SSE mit korrektem Protokoll |

## Unit-Tests (PFLICHT ab 25.12.2024)

### Regel
**Ab sofort werden für ALLE neuen Funktionen Unit-Tests geschrieben!**
Bei Änderungen an bestehenden Funktionen werden ebenfalls Unit-Tests nachgerüstet.

### Go Testing

```bash
# Alle Tests ausführen
go test ./...

# Tests mit Coverage
go test -cover ./...

# Einzelnes Package testen
go test ./internal/llamaserver/

# Verbose Output
go test -v ./internal/setup/
```

### Test-Datei Konvention

```
internal/
├── llamaserver/
│   ├── server.go
│   └── server_test.go      # Tests für server.go
├── setup/
│   ├── handlers.go
│   └── handlers_test.go    # Tests für handlers.go
└── vision/
    ├── vision.go
    └── vision_test.go      # Tests für vision.go
```

### Test-Beispiel

```go
// internal/llamaserver/server_test.go
package llamaserver

import (
    "testing"
)

func TestGetOrExtractLlamaServer(t *testing.T) {
    // Test mit existierendem Verzeichnis
    binPath, libPath, err := GetOrExtractLlamaServer("/tmp/test-data")
    if err != nil {
        t.Errorf("Unexpected error: %v", err)
    }
    // Weitere Assertions...
}

func TestServerStart(t *testing.T) {
    // Test-Setup
    srv := NewServer(DefaultConfig("/tmp/test"))

    // Test ohne Modell sollte Fehler geben
    err := srv.Start("")
    if err == nil {
        t.Error("Expected error when starting without model")
    }
}
```

### Frontend Testing (Vitest)

```bash
cd web
npm run test        # Tests ausführen
npm run test:watch  # Watch-Modus
npm run coverage    # Coverage-Report
```

### Was muss getestet werden?

| Priorität | Komponente | Grund |
|-----------|------------|-------|
| **HOCH** | API-Endpoints | Öffentliche Schnittstelle |
| **HOCH** | Business-Logik | Kernfunktionalität |
| **MITTEL** | Helper-Funktionen | Wiederverwendbarkeit |
| **MITTEL** | Error-Handling | Robustheit |
| **NIEDRIG** | UI-Komponenten | Visuell prüfbar |

---

## Entwicklung

### Neue Module hinzufügen

```go
// internal/mymodule/mymodule.go
package mymodule

type Service struct {
    // ...
}

func NewService() *Service {
    return &Service{}
}
```

### In main.go einbinden

```go
import "fleet-navigator/internal/mymodule"

// In NewApp()
myService := mymodule.NewService()
```

## Tägliche Changelogs

> **Dokumentationssystem:** Änderungen werden täglich in separaten Dateien dokumentiert.
>
> **Format:** `docs/CHANGELOG_YYYY-MM-DD.md`
>
> Dies hält die Hauptdokumentation übersichtlich und ermöglicht detaillierte Nachverfolgung.

### Verfügbare Changelogs

| Datum | Datei | Hauptänderungen |
|-------|-------|-----------------|
| 2025-12-23 | [CHANGELOG_2025-12-23.md](docs/CHANGELOG_2025-12-23.md) | **Mirror-First & MiniCPM-V Default**: Download-Strategie Mirror zuerst, MiniCPM-V-2.6 als Standard-Vision-Modell, Vision-Chaining automatisch aktiviert |
| 2025-12-16 | [CHANGELOG_2025-12-16.md](docs/CHANGELOG_2025-12-16.md) | **Mate Status & Disconnect Fix**: Falscher Online-Status behoben, Disconnect bei Mate-Entfernen |
| 2025-12-15 | [CHANGELOG_2025-12-15.md](docs/CHANGELOG_2025-12-15.md) | **Expert/Modus-Zuordnung pro Message**: Fixe Zuordnung in DB, Security Audit abgeschlossen |
| 2025-12-13 | [CHANGELOG_2025-12-13.md](docs/CHANGELOG_2025-12-13.md) | **Mate Pairing & Encryption Fixes**: messageId Type Mismatch, Pairing-Synchronisation, Thunderbird-Funktionen exponiert |
| 2025-12-12 | [CHANGELOG_2025-12-12.md](docs/CHANGELOG_2025-12-12.md) | **Provider-System festverdrahtet**: Model-Download Provider-abhängig, Provider-Wechsel mit Verbindungsprüfung & Fallback |
| 2025-12-11 | [CHANGELOG_2025-12-11.md](docs/CHANGELOG_2025-12-11.md) | Provider-System Fix, Model Manager Download-Fix, Persistente Settings in DB |
| 2025-12-25 | [CHANGELOG_2025-12-25.md](docs/CHANGELOG_2025-12-25.md) | **Anti-Halluzination konfigurierbar, DELEGATE-Tag für Experten-Umschaltung, Unit-Tests** |
| 2025-12-26 | [CHANGELOG_2025-12-26.md](docs/CHANGELOG_2025-12-26.md) | **SettingsModal.vue Refactoring Phase 2**: 3 weitere Tab-Komponenten integriert, -819 Zeilen (-22%), 9/11 Tabs fertig |
| 2025-12-27 | [CHANGELOG_2025-12-27.md](docs/CHANGELOG_2025-12-27.md) | **Model-Swap Fix, GitHub Actions repariert, SettingsModal Phase 3 fertig**: Race-Condition behoben, Workflow-Permissions, alle 11 Tabs modularisiert |

---

## Migration Status (Stand: 2025-12-27)

### Übersicht nach Modulen

| Modul | Status | Anmerkung |
|-------|--------|-----------|
| Chat/Streaming (SSE) | ✅ 95% | Start-Event, Content, Done-Token, Delegation-Event |
| Experten-System | ✅ 95% | CRUD, Modi, Anti-Halluzination, DELEGATE-Tag |
| Chat History/Persistenz | ✅ 90% | SQLite, Auto-Create bei chatId=0 |
| Mate Pairing/Security | ✅ 90% | Ed25519, AES-256, Pairing-Code |
| Vision/LLaVA | ✅ 85% | Bildanalyse, Streaming |
| Model Registry | ✅ 85% | Katalog, Kategorien, Featured |
| Tools (WebSearch, WebFetch) | ✅ 80% | DuckDuckGo, HTML-Parser |
| FileSearch Tool | ⚠️ 60% | Braucht Mate-Connection |
| **Custom Models** | ❌ 30% | Nur Stub, keine DB-Persistenz |
| System Prompts CRUD | ⚠️ 60% | Endpoint vorhanden, nicht vollständig |
| **Provider-System (Multi-LLM)** | ✅ 80% | llama-server als Default, Ollama optional |
| **Persistente Settings** | ✅ 95% | Sampling, Chaining, Preferences in DB |
| **Model Manager** | ✅ 85% | Provider-basierte Anzeige, Download-Fix |

### Bekannte Probleme & Lösungen

#### 1. Frontend zeigt keine Nachrichten (GELÖST)
**Problem:** Chat-Nachrichten wurden nicht angezeigt.
**Ursache:** Frontend erwartet `chatId` im Start-Event.
**Lösung:** Start-Event Format: `{"chatId":123,"requestId":"req-xxx"}`

#### 2. chatId=0 führt zu Fehler (GELÖST)
**Problem:** Frontend sendet `chatId: 0` für neue Chats, JavaScript behandelt 0 als falsy.
**Lösung:** Auto-Create in `handleChatSendStream()` - bei chatId=0 automatisch neuen Chat erstellen.

#### 3. Expert model: undefined (GELÖST)
**Problem:** Experten zeigten `model: undefined` im Frontend.
**Ursache:** Go JSON-Tag war `base_model`, Frontend erwartet `model`.
**Lösung:** JSON-Tag in `expert.go` geändert: `BaseModel string \`json:"model"\``

#### 4. Expert basePrompt nicht editierbar (GELÖST)
**Problem:** System-Prompt konnte nicht geändert werden.
**Ursache:** JSON-Tag `base_prompt` vs Frontend `basePrompt`.
**Lösung:** JSON-Tag geändert zu `json:"basePrompt"`.

#### 5. /api/custom-models 404 (GELÖST)
**Problem:** Frontend erwartet diesen Endpoint.
**Lösung:** Alias-Route hinzugefügt, gibt `[]` zurück (Stub).

#### 6. PATCH /api/chat/{id}/model 400 (GELÖST)
**Problem:** Frontend will Chat-Modell ändern.
**Lösung:** PATCH-Handler in `handleChatByID` erweitert.

#### 7. Provider immer "ollama" (GELÖST - 2025-12-11)
**Problem:** `/api/llm/providers` gab immer `activeProvider: "ollama"` zurück.
**Ursache:** Hardcoded Wert in `handleLLMProviders()`.
**Lösung:** Provider aus `settingsService.GetActiveProvider()` lesen, Name-Mapping für Frontend.

#### 8. Model Manager Download-Crash (GELÖST - 2025-12-11)
**Problem:** `ReferenceError: downloadStatus is not defined` beim Modell-Download.
**Ursache:** Variable `downloadStatus` nie definiert.
**Lösung:** Variable hinzugefügt, Provider-Check in `downloadOllamaModel()`.

#### 9. Settings nur in localStorage (GELÖST - 2025-12-11)
**Problem:** Wichtige Settings (Sampling, Chaining) gingen bei Browser-Wechsel verloren.
**Lösung:** Neue Backend-Endpoints für persistente Settings in SQLite-DB.

#### 10. Model-Download ignoriert Provider (GELÖST - 2025-12-12)
**Problem:** Model-Download verwendete immer Ollama API, auch wenn llama-cpp aktiv war.
**Ursache:** `handleModelsPull` und `handleLLMPullModel` prüften nicht den aktiven Provider.
**Lösung:** Provider-Prüfung vor jedem Download:
- llama-cpp → GGUF von HuggingFace
- ollama → Ollama API

#### 11. Provider-Wechsel ohne Verbindungsprüfung (GELÖST - 2025-12-12)
**Problem:** Wechsel zu Ollama schlug fehl ohne Fehlermeldung.
**Lösung:** Neuer `/api/llm/providers/switch` Endpoint mit:
- Verbindungsprüfung bei Ollama-Wechsel
- Automatischer Fallback auf llama-cpp bei Fehler
- Detaillierte Fehlermeldung für Frontend

#### 12. Modellverwaltung ignoriert Provider (GELÖST - 2025-12-12)
**Problem:** `/api/llm/models` und `/api/llm/models/installed` fragten immer Ollama ab.
**Ursache:** Keine Provider-Prüfung in diesen Endpoints.
**Lösung:**
- Provider-Prüfung vor jeder Abfrage
- Bei llama-cpp: GGUF-Dateien direkt lesen
- Neue Funktion `FindByFilename` in Registry für Metadaten-Lookup

#### 13. messageId Type Mismatch (GELÖST - 2025-12-13)
**Problem:** `json: cannot unmarshal number into Go struct field EmailClassifyRequest.messageId of type string`
**Ursache:** Thunderbird sendete `messageId` als JavaScript Number, Go erwartete String.
**Lösung (Thunderbird-seitig):** `messageId: String(email.id)` in `fleet-client.js` und `background.js`.

#### 14. Pairing-Synchronisation nach "Vergessen" (GELÖST - 2025-12-13)
**Problem:** Nach "Pairing vergessen" und erneutem Pairing: `cipher: message authentication failed`
**Ursache:** Thunderbird generierte neue MateID + Keys, Navigator hatte alte in `trusted_mates.json`.
**Lösung:**
- `trusted_mates.json` leeren: `echo "[]" > ~/.fleet-navigator/trusted_mates.json`
- Navigator neustarten
- In Thunderbird "Pairing vergessen" und neu verbinden

#### 15. Kategorisierung startete nicht (GELÖST - 2025-12-13)
**Problem:** Button wechselte zu "Abbrechen", aber keine E-Mails wurden verarbeitet.
**Ursache:** `processExistingEmailsManual` war nicht auf `window` exponiert.
**Lösung (Thunderbird-seitig):** In `background.js` hinzugefügt:
```javascript
window.processExistingEmailsManual = processExistingEmailsManual;
window.processSelectedFolders = processSelectedFolders;
```

#### 16. LLM halluziniert Identität und Quellen (GELÖST - 2025-12-25)
**Problem:** Bei "Wer bist du?" erfand das LLM Details wie "Ewa Marek" und zitierte irrelevante Quellen.
**Ursache:** Anti-Halluzinations-Prompt hatte keine Regeln für Identitätsfragen und Quellen-Relevanz.
**Lösung:**
- Verstärkte Anti-Halluzinations-Regeln mit IDENTITÄT und QUELLEN Abschnitten
- Bei Identitätsfragen: NUR Informationen aus System-Prompt verwenden
- Quellen NUR zitieren wenn thematisch relevant
- Anti-Halluzinations-Prompt jetzt pro Experte konfigurierbar

#### 17. DELEGATE-Tag wurde im Output angezeigt (GELÖST - 2025-12-25)
**Problem:** `[[DELEGATE:Roland]]` war im Chat sichtbar statt Experten-Umschaltung.
**Ursache:** DELEGATE-Tag war in Prompts definiert, aber Backend verarbeitete ihn nicht.
**Lösung:**
- Regex-Parser in `handleChatSendStream()` für `[[DELEGATE:ExpertName]]`
- Tag wird aus Antwort entfernt
- SSE "delegation" Event ans Frontend mit expertId, expertName, expertAvatar
- Frontend-Handler in `chatStore.js` wechselt automatisch zum Experten

#### 18. Wake Word Audio funktioniert nicht (OFFEN)
**Problem:** Wake Word Detection mit "Ewa" / "Hey Ewa" funktioniert nicht.
**Status:** Audio-Capture im Browser nicht vollständig implementiert.
**Betroffene Dateien:**
- `internal/voice/service.go`
- `web/src/composables/useVoice.js`
**TODO:**
- [ ] Browser Audio-Capture testen (MediaRecorder API)
- [ ] Whisper STT Integration prüfen
- [ ] Wake Word Pattern Matching debuggen

#### 19. Tesseract OCR Download-API fehlte (GELÖST - 2025-12-26)
**Problem:** API-Endpoint `/api/setup/tesseract/download` existierte nicht, daher konnte Tesseract nicht installiert werden.
**Ursache:** Die `DownloadTesseract()` Funktion existierte, aber kein HTTP-Handler rief sie auf.
**Lösung:**
- Neues Interface `TesseractDownloader` in `internal/setup/handlers.go`
- API-Endpoints: `GET /api/setup/tesseract/status` und `GET /api/setup/tesseract/download` (SSE)
- Frontend: Neuer Tab "Erweiterungen" in SettingsModal mit Tesseract-Installation
- i18n: Übersetzungen für DE, EN, TR hinzugefügt

### Frontend-Kompatibilität: JSON-Mapping

Das Vue-Frontend erwartet camelCase, Go-Structs sollten entsprechend gemappt werden:

```go
// FALSCH (snake_case)
BasePrompt string `json:"base_prompt"`
BaseModel  string `json:"base_model"`

// RICHTIG (camelCase für Frontend)
BasePrompt string `json:"basePrompt"`
BaseModel  string `json:"model"`
```

### Offene Punkte (Go-Version)

**Internationalisierung:**
- [x] TR-Übersetzung vollständig (17 Keys hinzugefügt am 2025-12-24)
- [x] **Sprachwechsel zur Laufzeit** implementiert (Prompts + Stimmen-Info)
- [ ] FR-Übersetzung vervollständigen (~900 fehlende Keys)
- [ ] ES-Übersetzung vervollständigen (~1150 fehlende Keys)
- [x] TTS-Stimmen Download-Dialog bei Sprachwechsel
- [ ] ~~Wake Words: "Ewa" funktioniert sprachübergreifend~~ → Siehe Issue #18

**Voice-System (⚠️ FUNKTIONIERT NICHT - Issue #18):**
- [ ] **Wake Word Detection funktioniert nicht** - Audio-Capture im Browser nicht vollständig
- [ ] Browser MediaRecorder API Integration
- [ ] Whisper STT Backend-Anbindung
- [ ] TTS-Integration (Piper) vervollständigen
- [ ] Sound-Dateien für Feedback einbetten

**Tesseract OCR (✅ GELÖST - Issue #19):**
- [x] Download-Funktion vom Mirror (alle OS)
- [x] API-Endpoints: `/api/setup/tesseract/status` und `/api/setup/tesseract/download`
- [x] Settings UI: Tab "Erweiterungen" mit Tesseract-Installation
- [x] Sprachpakete DEU/ENG/TUR im Mirror vorhanden
- [x] i18n: Übersetzungen für DE, EN, TR

**Sonstige:**
- [ ] Custom Models vollständige DB-Implementation
- [ ] System Prompts CRUD komplett
- [ ] Provider-Switching UI
- [ ] Weitere Frontend-API-Kompatibilitätstests
- [ ] Rate Limiting
- [x] Graceful Shutdown (implementiert)
- [ ] Multi-User / Login-System

### Empfehlung

**Go-Version für v2.0 planen:**
- v1.x bleibt Java (produktiv)
- v2.0 startet mit Go-Backend
- Parallele Entwicklung möglich da gleiches Frontend

---

## Java-Version: Offene Punkte

(Siehe separates CLAUDE.md im JavaFleet-Projekt)

Wichtige Features für Java v1.x:
1. **Login/Logout System** - Datenschutz für Benutzer
2. **Mate-Authentifizierung** - Mates müssen sich beim Navigator authentifizieren
3. **Verschlüsselte Nachrichten** - Ende-zu-Ende zwischen Mates
4. **FileSearch in OS-Mates** - Lokale Dateisuche
5. **WebSearch im Navigator** - DuckDuckGo Integration

---

## Kontakt

**JavaFleet Systems Consulting**
Port 2026 - Das Jahr der Go-Migration!
- das ist der Pfad /home/trainer/NetBeansProjects/ProjekteFMH/Fleet-Navigator/target das ist der richtige pfad aber nur wenn du ein Build gemacht hast du ein build gemach?