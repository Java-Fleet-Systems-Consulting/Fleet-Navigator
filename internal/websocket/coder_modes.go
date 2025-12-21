// Package websocket - Automatische Modus-Erkennung für Coder-Mates
package websocket

import (
	"regexp"
	"strings"
)

// CoderMode repräsentiert einen Programmier-Modus
type CoderMode struct {
	ID          string
	Name        string
	Icon        string
	Language    string
	Extensions  []string         // Dateiendungen (sicher)
	Keywords    []string         // Sichere Keywords
	Patterns    []*regexp.Regexp // Regex-Patterns für Kombinationen
	Prompt      string           // System-Prompt für diesen Modus
}

// ModeDetectionResult ist das Ergebnis der Modus-Erkennung
type ModeDetectionResult struct {
	Detected    bool       // Wurde ein Modus erkannt?
	Mode        *CoderMode // Erkannter Modus (nil wenn unsicher)
	Confident   bool       // Sind wir sicher?
	AskQuestion string     // Nachfrage falls unsicher
	MatchedOn   string     // Was hat gematcht? (für Logging)
}

// coderModes definiert alle verfügbaren Programmier-Modi
var coderModes = map[string]*CoderMode{
	"general": {
		ID:         "general",
		Name:       "General Coder",
		Icon:       "💻",
		Language:   "Multi-Language",
		Extensions: []string{},
		Keywords:   []string{},
		Patterns:   []*regexp.Regexp{},
		Prompt: `Du bist FleetCoder, ein Coding-Assistent mit Tool-Zugriff.

VERFÜGBARE TOOLS:
- write_file: Dateien ERSTELLEN oder ändern → {"path":"X","content":"..."}
- read_file: Dateien LESEN → {"path":"X"}
- list_files: Verzeichnis AUFLISTEN → {"path":".","pattern":"*.X"}
- shell: Befehle AUSFÜHREN → {"command":"X"}
- search: Dateien DURCHSUCHEN → {"pattern":"X","path":"."}

FORMAT für Tool-Aufrufe:
<tool>{"name":"TOOLNAME","args":{...}}</tool>

WICHTIG - Tool-Auswahl:
• "erstelle/schreibe/erzeuge X" → write_file (Inhalt generieren!)
• "zeige/lies/öffne X" → read_file
• "finde/suche/liste X" → list_files oder search
• "führe aus/starte" → shell

Antworte NUR mit dem passenden Tool-Aufruf, OHNE Erklärungen.`,
	},
	"go": {
		ID:       "go",
		Name:     "Go Developer",
		Icon:     "🐹",
		Language: "Go",
		Extensions: []string{".go", "go.mod", "go.sum"},
		Keywords:   []string{"golang", "goroutine", "goroutines", "go-funktion", "go-code", "go-programm"},
		Patterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)\b(code|programm|funktion|script)\s+(in\s+)?go\b`),
			regexp.MustCompile(`(?i)\bgo\s+(code|programm|funktion|datei)\b`),
			regexp.MustCompile(`(?i)\bgo\s+lang(uage)?\b`),
		},
		Prompt: `Du bist ein erfahrener Go-Entwickler. Du folgst den Go Best Practices:
- Idiomatisches Go (effective Go)
- Klare Fehlerbehandlung mit error returns
- Goroutines und Channels für Concurrency
- Interfaces für Abstraktion
- go fmt, go vet, golint Standards
- Kurze, prägnante Variablennamen in kleinem Scope
- Testing mit go test
Bevorzuge die Standardbibliothek. Vermeide über-komplizierte Abstraktionen.`,
	},
	"java": {
		ID:       "java",
		Name:     "Java Developer",
		Icon:     "☕",
		Language: "Java",
		Extensions: []string{".java", "pom.xml", "build.gradle", ".gradle"},
		Keywords:   []string{"spring boot", "spring framework", "hibernate", "jpa", "maven", "gradle", "junit"},
		Patterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)\b(code|programm|klasse|class)\s+(in\s+)?java\b`),
			regexp.MustCompile(`(?i)\bjava\s+(code|programm|klasse|class|datei)\b`),
		},
		Prompt: `Du bist ein erfahrener Java-Entwickler. Du folgst den Java Best Practices:
- Clean Code Prinzipien
- SOLID Design Patterns
- Spring Boot / Spring Framework
- JPA/Hibernate für Persistenz
- Maven/Gradle Build-Systeme
- JUnit 5 für Testing
Bevorzuge moderne Java Features (Records, Pattern Matching, var).`,
	},
	"python": {
		ID:       "python",
		Name:     "Python Developer",
		Icon:     "🐍",
		Language: "Python",
		Extensions: []string{".py", ".pyw", "requirements.txt", "pyproject.toml", "setup.py", "Pipfile"},
		Keywords:   []string{"python3", "python2", "pip install", "pytest", "django", "flask", "fastapi", "pandas", "numpy"},
		Patterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)\b(code|script|programm)\s+(in\s+)?python\b`),
			regexp.MustCompile(`(?i)\bpython\s+(code|script|programm|datei)\b`),
		},
		Prompt: `Du bist ein erfahrener Python-Entwickler. Du folgst den Python Best Practices:
- PEP 8 Style Guide
- Type Hints (typing module)
- Virtual Environments (venv, poetry)
- pytest für Testing
- Docstrings (Google/NumPy Style)
- List/Dict Comprehensions wo sinnvoll
Bevorzuge die Standardbibliothek. Nutze f-strings.`,
	},
	"javascript": {
		ID:       "javascript",
		Name:     "JavaScript/TypeScript",
		Icon:     "🟨",
		Language: "JavaScript/TypeScript",
		Extensions: []string{".js", ".mjs", ".cjs", ".ts", ".tsx", ".jsx", "package.json", "tsconfig.json"},
		Keywords:   []string{"typescript", "nodejs", "node.js", "npm install", "yarn add", "react", "vue", "angular", "express", "nextjs", "nuxt"},
		Patterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)\b(code|script|programm)\s+(in\s+)?(javascript|typescript|js|ts)\b`),
			regexp.MustCompile(`(?i)\b(javascript|typescript|js|ts)\s+(code|script|programm|datei)\b`),
		},
		Prompt: `Du bist ein erfahrener JavaScript/TypeScript-Entwickler. Du folgst den Best Practices:
- TypeScript für Type Safety
- ESLint/Prettier für Code-Qualität
- Modern ES6+ Syntax (async/await, destructuring)
- React/Vue/Angular Patterns
- Node.js für Backend
- Jest/Vitest für Testing
Bevorzuge TypeScript. Vermeide any-Types.`,
	},
	"bash": {
		ID:       "bash",
		Name:     "Shell Scripting",
		Icon:     "🐚",
		Language: "Bash/Shell",
		Extensions: []string{".sh", ".bash", ".zsh", "Makefile", ".bashrc", ".zshrc"},
		Keywords:   []string{"bash script", "shell script", "shellcheck", "shebang", "#!/bin/bash", "#!/bin/sh"},
		Patterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)\b(bash|shell|sh)\s+(script|code|datei)\b`),
			regexp.MustCompile(`(?i)\b(script|code)\s+(in\s+)?(bash|shell|sh)\b`),
		},
		Prompt: `Du bist Shell/Bash-Experte mit Tool-Zugriff. Nutze Tools statt zu erklären.

FORMAT: <tool>{"name": "X", "args": {...}}</tool>

TOOLS: shell, list_files, read_file, write_file, search

BEISPIELE:
"suche sh" → <tool>{"name": "list_files", "args": {"path": ".", "pattern": "*.sh", "recursive": true}}</tool>
"ls -la" → <tool>{"name": "shell", "args": {"command": "ls -la"}}</tool>

Bei Skripten: set -euo pipefail, proper quoting.`,
	},
	"rust": {
		ID:       "rust",
		Name:     "Rust Developer",
		Icon:     "🦀",
		Language: "Rust",
		Extensions: []string{".rs", "Cargo.toml", "Cargo.lock"},
		Keywords:   []string{"cargo build", "cargo run", "rustc", "crate", "lifetime", "borrow checker"},
		Patterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)\b(code|programm|funktion)\s+(in\s+)?rust\b`),
			regexp.MustCompile(`(?i)\brust\s+(code|programm|funktion|datei)\b`),
		},
		Prompt: `Du bist ein erfahrener Rust-Entwickler. Du folgst den Rust Best Practices:
- Ownership und Borrowing korrekt nutzen
- Result/Option für Error Handling
- Traits für Polymorphismus
- Cargo für Build und Dependencies
- clippy für Linting
- rustfmt für Formatierung
Vermeide unsafe wo möglich.`,
	},
	"sql": {
		ID:       "sql",
		Name:     "SQL Developer",
		Icon:     "🗃️",
		Language: "SQL",
		Extensions: []string{".sql"},
		Keywords:   []string{"select from", "insert into", "create table", "alter table", "mysql", "postgresql", "sqlite", "mariadb"},
		Patterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)\bsql\s+(query|abfrage|statement|code)\b`),
			regexp.MustCompile(`(?i)\b(query|abfrage)\s+(in\s+)?sql\b`),
			regexp.MustCompile(`(?i)\bdatenbank(en)?\s+(abfrage|query)\b`),
		},
		Prompt: `Du bist ein erfahrener SQL/Datenbank-Entwickler. Du folgst den Best Practices:
- Normalisierte Datenbankdesigns (3NF)
- Indexierung für Performance
- Prepared Statements gegen SQL Injection
- Transaktionen für Datenkonsistenz
- JOINs statt Subqueries wo sinnvoll
Berücksichtige MySQL/PostgreSQL Unterschiede.`,
	},
	"devops": {
		ID:       "devops",
		Name:     "DevOps Engineer",
		Icon:     "🔧",
		Language: "DevOps/Infrastructure",
		Extensions: []string{"Dockerfile", "docker-compose.yml", "docker-compose.yaml", ".tf", ".tfvars", "Jenkinsfile", ".gitlab-ci.yml", ".github/workflows"},
		Keywords:   []string{"dockerfile", "docker-compose", "kubernetes", "k8s", "terraform", "ansible", "ci/cd", "pipeline", "helm"},
		Patterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)\b(docker|kubernetes|k8s|terraform|ansible)\b`),
			regexp.MustCompile(`(?i)\b(ci|cd|pipeline|deployment)\b`),
		},
		Prompt: `Du bist ein erfahrener DevOps Engineer. Du folgst den Best Practices:
- Infrastructure as Code (Terraform, Ansible)
- Container-Orchestrierung (Docker, Kubernetes)
- CI/CD Pipelines (GitHub Actions, GitLab CI)
- Security Best Practices (Secrets Management)
- GitOps Workflows
- 12-Factor App Prinzipien
Automatisiere alles was möglich ist.`,
	},
	"powershell": {
		ID:       "powershell",
		Name:     "PowerShell",
		Icon:     "🔷",
		Language: "PowerShell",
		Extensions: []string{".ps1", ".psm1", ".psd1"},
		Keywords:   []string{"powershell", "pwsh", "cmdlet", "get-command", "invoke-command", "write-host"},
		Patterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)\bpowershell\s+(script|code|datei)\b`),
			regexp.MustCompile(`(?i)\b(script|code)\s+(in\s+)?powershell\b`),
		},
		Prompt: `Du bist ein erfahrener PowerShell-Entwickler. Du folgst den Best Practices:
- Approved Verbs (Get-, Set-, New-, Remove-, etc.)
- Cmdlet-Naming Convention (Verb-Noun)
- Pipeline-orientiertes Design
- Error Handling mit try/catch
- Comment-Based Help für Dokumentation
- Module-Struktur für Wiederverwendbarkeit
- PSScriptAnalyzer für Linting
Nutze moderne PowerShell 7+ Features wo möglich.`,
	},
	"batch": {
		ID:       "batch",
		Name:     "Windows Batch",
		Icon:     "🪟",
		Language: "Batch/CMD",
		Extensions: []string{".bat", ".cmd"},
		Keywords:   []string{"batch script", "batch datei", "cmd script", "@echo off", "setlocal"},
		Patterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)\b(batch|cmd|bat)\s+(script|datei|file)\b`),
			regexp.MustCompile(`(?i)\b(script|datei)\s+(in\s+)?(batch|cmd)\b`),
			regexp.MustCompile(`(?i)\bwindows\s+(script|batch)\b`),
		},
		Prompt: `Du bist ein erfahrener Windows Batch-Entwickler. Du folgst den Best Practices:
- @echo off am Anfang
- setlocal enabledelayedexpansion bei Bedarf
- Proper Error Handling mit errorlevel
- Variablen in %% für Batch, % für CMD
- Kommentare mit REM oder ::
- Saubere Ausgabe mit echo.
- Exit Codes korrekt setzen
Beachte die Unterschiede zwischen CMD und Batch.`,
	},
}

// ambiguousKeywords sind Keywords die alleine nicht eindeutig sind
var ambiguousKeywords = map[string]string{
	"go":     "Meinst du die Programmiersprache Go (Golang)?",
	"script": "In welcher Sprache soll das Script sein? (Bash, Python, ...)",
	"code":   "In welcher Programmiersprache soll der Code sein?",
	"java":   "Meinst du Java oder JavaScript?",
	"c":      "Meinst du C, C++ oder C#?",
}

// modeSwitchPatterns erkennt explizite Modus-Wechsel-Befehle
// Format: Regex → Mode-ID
var modeSwitchPatterns = map[*regexp.Regexp]string{
	// Flexible Formen: "geh in den X modus", "wechsle zu X", "wechsel zu X", etc.
	// Muster: (verb) ... (sprache) ... (modus)?
	regexp.MustCompile(`(?i)(wechsel|wechsle|switch|geh|mach)\s+.*\b(go|golang)\b`):       "go",
	regexp.MustCompile(`(?i)(wechsel|wechsle|switch|geh|mach)\s+.*\b(java)\b`):            "java",
	regexp.MustCompile(`(?i)(wechsel|wechsle|switch|geh|mach)\s+.*\b(python)\b`):          "python",
	regexp.MustCompile(`(?i)(wechsel|wechsle|switch|geh|mach)\s+.*\b(javascript|typescript|js|ts)\b`): "javascript",
	regexp.MustCompile(`(?i)(wechsel|wechsle|switch|geh|mach)\s+.*\b(bash|shell)\b`):      "bash",
	regexp.MustCompile(`(?i)(wechsel|wechsle|switch|geh|mach)\s+.*\b(rust)\b`):            "rust",
	regexp.MustCompile(`(?i)(wechsel|wechsle|switch|geh|mach)\s+.*\b(sql)\b`):             "sql",
	regexp.MustCompile(`(?i)(wechsel|wechsle|switch|geh|mach)\s+.*\b(devops|docker|k8s)\b`): "devops",
	regexp.MustCompile(`(?i)(wechsel|wechsle|switch|geh|mach)\s+.*\b(powershell|ps1)\b`):  "powershell",
	regexp.MustCompile(`(?i)(wechsel|wechsle|switch|geh|mach)\s+.*\b(batch|cmd|bat)\b`):   "batch",
	regexp.MustCompile(`(?i)(wechsel|wechsle|switch|geh|mach)\s+.*\b(general|allgemein)\b`): "general",

	// Kurzform: "Go-Modus", "Python mode", "bash modus", etc.
	regexp.MustCompile(`(?i)^(go|golang)[\-\s]*(modus|mode)$`):       "go",
	regexp.MustCompile(`(?i)^(java)[\-\s]*(modus|mode)$`):            "java",
	regexp.MustCompile(`(?i)^(python)[\-\s]*(modus|mode)$`):          "python",
	regexp.MustCompile(`(?i)^(javascript|typescript|js|ts)[\-\s]*(modus|mode)$`): "javascript",
	regexp.MustCompile(`(?i)^(bash|shell)[\-\s]*(modus|mode)$`):      "bash",
	regexp.MustCompile(`(?i)^(rust)[\-\s]*(modus|mode)$`):            "rust",
	regexp.MustCompile(`(?i)^(sql)[\-\s]*(modus|mode)$`):             "sql",
	regexp.MustCompile(`(?i)^(devops)[\-\s]*(modus|mode)$`):          "devops",
	regexp.MustCompile(`(?i)^(powershell|ps1)[\-\s]*(modus|mode)$`):  "powershell",
	regexp.MustCompile(`(?i)^(batch|cmd|bat)[\-\s]*(modus|mode)$`):   "batch",
	regexp.MustCompile(`(?i)^(general|allgemein)[\-\s]*(modus|mode)$`): "general",

	// Verwende-Form: "verwende Go", "use Python", "nutze bash"
	regexp.MustCompile(`(?i)^(verwende|use|nutze)\s+(go|golang)$`):   "go",
	regexp.MustCompile(`(?i)^(verwende|use|nutze)\s+(java)$`):        "java",
	regexp.MustCompile(`(?i)^(verwende|use|nutze)\s+(python)$`):      "python",
	regexp.MustCompile(`(?i)^(verwende|use|nutze)\s+(javascript|typescript|js|ts)$`): "javascript",
	regexp.MustCompile(`(?i)^(verwende|use|nutze)\s+(bash|shell)$`):  "bash",
	regexp.MustCompile(`(?i)^(verwende|use|nutze)\s+(rust)$`):        "rust",
	regexp.MustCompile(`(?i)^(verwende|use|nutze)\s+(sql)$`):         "sql",
	regexp.MustCompile(`(?i)^(verwende|use|nutze)\s+(devops)$`):      "devops",
	regexp.MustCompile(`(?i)^(verwende|use|nutze)\s+(powershell)$`):  "powershell",
	regexp.MustCompile(`(?i)^(verwende|use|nutze)\s+(batch|cmd)$`):   "batch",

	// "in den X modus" ohne verb am anfang
	regexp.MustCompile(`(?i)^in\s+(den\s+)?(go|golang)\s*(modus|mode)`):       "go",
	regexp.MustCompile(`(?i)^in\s+(den\s+)?(java)\s*(modus|mode)`):            "java",
	regexp.MustCompile(`(?i)^in\s+(den\s+)?(python)\s*(modus|mode)`):          "python",
	regexp.MustCompile(`(?i)^in\s+(den\s+)?(javascript|typescript|js|ts)\s*(modus|mode)`): "javascript",
	regexp.MustCompile(`(?i)^in\s+(den\s+)?(bash|shell)\s*(modus|mode)`):      "bash",
	regexp.MustCompile(`(?i)^in\s+(den\s+)?(rust)\s*(modus|mode)`):            "rust",
	regexp.MustCompile(`(?i)^in\s+(den\s+)?(sql)\s*(modus|mode)`):             "sql",
	regexp.MustCompile(`(?i)^in\s+(den\s+)?(devops)\s*(modus|mode)`):          "devops",
	regexp.MustCompile(`(?i)^in\s+(den\s+)?(powershell)\s*(modus|mode)`):      "powershell",
	regexp.MustCompile(`(?i)^in\s+(den\s+)?(batch|cmd)\s*(modus|mode)`):       "batch",
}

// DetectExplicitModeSwitch prüft ob ein expliziter Modus-Wechsel angefordert wird
func DetectExplicitModeSwitch(message string) (string, bool) {
	message = strings.TrimSpace(message)
	for pattern, modeID := range modeSwitchPatterns {
		if pattern.MatchString(message) {
			return modeID, true
		}
	}
	return "", false
}

// DetectCoderMode erkennt den passenden Modus aus einer Nachricht
func DetectCoderMode(message string, currentMode string) *ModeDetectionResult {
	messageLower := strings.ToLower(message)

	result := &ModeDetectionResult{
		Detected:  false,
		Confident: false,
	}

	// 1. Prüfe auf Dateiendungen (höchste Priorität, immer sicher)
	for _, mode := range coderModes {
		for _, ext := range mode.Extensions {
			// Suche nach Dateinamen mit dieser Endung
			extPattern := regexp.MustCompile(`(?i)[\w\-./]+` + regexp.QuoteMeta(ext) + `\b`)
			if extPattern.MatchString(message) {
				result.Detected = true
				result.Mode = mode
				result.Confident = true
				result.MatchedOn = "Dateiendung: " + ext
				return result
			}
		}
	}

	// 2. Prüfe auf sichere Keywords
	for _, mode := range coderModes {
		for _, keyword := range mode.Keywords {
			if strings.Contains(messageLower, strings.ToLower(keyword)) {
				result.Detected = true
				result.Mode = mode
				result.Confident = true
				result.MatchedOn = "Keyword: " + keyword
				return result
			}
		}
	}

	// 3. Prüfe auf Regex-Patterns (Kombinationen)
	for _, mode := range coderModes {
		for _, pattern := range mode.Patterns {
			if pattern.MatchString(message) {
				result.Detected = true
				result.Mode = mode
				result.Confident = true
				result.MatchedOn = "Pattern: " + pattern.String()
				return result
			}
		}
	}

	// 4. Prüfe auf ambige Keywords (nachfragen!)
	for keyword, question := range ambiguousKeywords {
		// Nur als ganzes Wort matchen
		wordPattern := regexp.MustCompile(`(?i)\b` + regexp.QuoteMeta(keyword) + `\b`)
		if wordPattern.MatchString(message) {
			// Prüfe ob es nicht schon durch sichere Matches abgedeckt wurde
			result.Detected = true
			result.Confident = false
			result.AskQuestion = question
			result.MatchedOn = "Ambig: " + keyword
			return result
		}
	}

	return result
}

// GetCoderMode gibt einen Modus nach ID zurück
func GetCoderMode(modeID string) *CoderMode {
	return coderModes[modeID]
}

// GetAllCoderModes gibt alle Modi zurück
func GetAllCoderModes() map[string]*CoderMode {
	return coderModes
}

// BuildModePrompt kombiniert den Modus-Prompt mit der Arbeitsanweisung
func BuildModePrompt(mode *CoderMode, userMessage string) string {
	if mode == nil {
		return userMessage
	}

	return mode.Prompt + "\n\nArbeitsanweisung: " + userMessage
}
