// Package llm - Model Registry
// Zentrale Registry aller verfuegbaren GGUF-Modelle
package llm

import (
	"sort"
	"strings"
)

// ModelCategory definiert die Modell-Kategorie
type ModelCategory string

const (
	CategoryChat    ModelCategory = "chat"
	CategoryCode    ModelCategory = "code"
	CategoryVision  ModelCategory = "vision"
	CategoryCompact ModelCategory = "compact"
)

// ModelRegistryEntry enthaelt alle Informationen zu einem Modell
type ModelRegistryEntry struct {
	ID              string        `json:"id"`
	DisplayName     string        `json:"display_name"`
	Provider        string        `json:"provider"`
	Architecture    string        `json:"architecture"`
	Version         string        `json:"version"`
	ParameterSize   string        `json:"parameter_size"`
	Quantization    string        `json:"quantization"`
	HuggingFaceRepo string        `json:"huggingface_repo,omitempty"`
	Filename        string        `json:"filename"`
	SizeBytes       int64         `json:"size_bytes"`
	SizeHuman       string        `json:"size_human"`
	Description     string        `json:"description"`
	Languages       []string      `json:"languages"`
	UseCases        []string      `json:"use_cases"`
	License         string        `json:"license"`
	Rating          float32       `json:"rating"`
	Downloads       int           `json:"downloads"`
	MinRamGB        int           `json:"min_ram_gb"`
	RecommendedRamGB int          `json:"recommended_ram_gb"`
	GPUAccelSupported bool        `json:"gpu_accel_supported"`
	Featured        bool          `json:"featured"`
	Trending        bool          `json:"trending"`
	Category        ModelCategory `json:"category"`
	ReleaseDate     string        `json:"release_date,omitempty"`
	TrainedUntil    string        `json:"trained_until,omitempty"`
	ContextWindow   string        `json:"context_window,omitempty"`
	PrimaryTasks    string        `json:"primary_tasks,omitempty"`
	Strengths       string        `json:"strengths,omitempty"`
	Limitations     string        `json:"limitations,omitempty"`
	IsVisionModel   bool          `json:"is_vision_model,omitempty"`
	MmprojFilename  string        `json:"mmproj_filename,omitempty"`
	// Ollama-spezifisch
	OllamaName      string        `json:"ollama_name,omitempty"` // z.B. "qwen2.5:7b"
	// Context-Größe
	ContextSize     int           `json:"context_size,omitempty"` // Max. Context in Tokens (z.B. 32768, 131072)
}

// ModelRegistry verwaltet den Modell-Katalog
type ModelRegistry struct {
	models []ModelRegistryEntry
}

// NewModelRegistry erstellt eine neue Model Registry mit Standard-Modellen
func NewModelRegistry() *ModelRegistry {
	registry := &ModelRegistry{
		models: make([]ModelRegistryEntry, 0),
	}
	registry.initializeDefaultModels()
	return registry
}

// initializeDefaultModels initialisiert die Standard-Modelle
func (r *ModelRegistry) initializeDefaultModels() {
	// ===== DEUTSCHE & MEHRSPRACHIGE CHAT-MODELLE =====

	r.models = append(r.models, ModelRegistryEntry{
		ID:              "qwen2.5-3b-instruct",
		DisplayName:     "Qwen 2.5 (3B) - Instruct",
		Provider:        "Alibaba Cloud",
		Architecture:    "qwen2",
		Version:         "2.5",
		ParameterSize:   "3B",
		Quantization:    "Q4_K_M",
		HuggingFaceRepo: "Qwen/Qwen2.5-3B-Instruct-GGUF",
		Filename:        "qwen2.5-3b-instruct-q4_k_m.gguf",
		OllamaName:      "qwen2.5:3b",
		SizeBytes:       1_967_004_960,
		SizeHuman:       "1.97 GB",
		Description:     "Exzellentes mehrsprachiges Modell mit hervorragendem Deutsch. Sehr gute Balance zwischen Geschwindigkeit und Qualitaet.",
		Languages:       []string{"Deutsch", "Englisch", "Franzoesisch", "Spanisch", "und 25+ weitere"},
		UseCases:        []string{"Briefe schreiben", "E-Mails", "Chat", "Uebersetzungen", "Zusammenfassungen"},
		License:         "Apache 2.0",
		Rating:          4.9,
		Downloads:       120000,
		MinRamGB:        4,
		RecommendedRamGB: 8,
		GPUAccelSupported: true,
		Featured:        true,
		Trending:        true,
		Category:        CategoryChat,
		ReleaseDate:     "2024-09",
		TrainedUntil:    "2024-06",
		ContextWindow:   "32K tokens",
		ContextSize:     32768, // 32K
		PrimaryTasks:    "Chat, Briefe schreiben, E-Mails, Uebersetzungen",
		Strengths:       "Exzellentes Deutsch, Mehrsprachig (29 Sprachen), Hohe Qualitaet bei geringer Groesse",
		Limitations:     "Bei sehr langen Dokumenten (>32K) kann Kontext verloren gehen",
	})

	r.models = append(r.models, ModelRegistryEntry{
		ID:              "llama-3.2-1b-instruct",
		DisplayName:     "Llama 3.2 (1B) - Instruct",
		Provider:        "Meta AI",
		Architecture:    "llama",
		Version:         "3.2",
		ParameterSize:   "1B",
		Quantization:    "Q4_K_M",
		HuggingFaceRepo: "bartowski/Llama-3.2-1B-Instruct-GGUF",
		Filename:        "Llama-3.2-1B-Instruct-Q4_K_M.gguf",
		OllamaName:      "llama3.2:1b",
		SizeBytes:       711_000_000,
		SizeHuman:       "711 MB",
		Description:     "Extrem kompaktes Modell von Meta AI. Ideal fuer ressourcenbeschraenkte Systeme.",
		Languages:       []string{"Deutsch", "Englisch", "und weitere"},
		UseCases:        []string{"Schnelle Antworten", "Einfache Aufgaben", "Chat", "Testen"},
		License:         "Llama 3.2 Community License",
		Rating:          4.3,
		Downloads:       89000,
		MinRamGB:        2,
		RecommendedRamGB: 4,
		GPUAccelSupported: true,
		Featured:        true,
		Trending:        true,
		Category:        CategoryCompact,
		ReleaseDate:     "2024-09",
		TrainedUntil:    "2024-06",
		ContextWindow:   "128K tokens",
		ContextSize:     131072, // 128K
		PrimaryTasks:    "Quick Answers, Simple Chat, Testing",
		Strengths:       "Extrem klein, Sehr schnell, Laeuft auf schwacher Hardware, 128K Context!",
		Limitations:     "Begrenzte Faehigkeiten bei komplexen Aufgaben",
	})

	r.models = append(r.models, ModelRegistryEntry{
		ID:              "llama-3.2-3b-instruct",
		DisplayName:     "Llama 3.2 (3B) - Instruct",
		Provider:        "Meta AI",
		Architecture:    "llama",
		Version:         "3.2",
		ParameterSize:   "3B",
		Quantization:    "Q4_K_M",
		HuggingFaceRepo: "bartowski/Llama-3.2-3B-Instruct-GGUF",
		Filename:        "Llama-3.2-3B-Instruct-Q4_K_M.gguf",
		OllamaName:      "llama3.2:3b",
		SizeBytes:       2_018_066_080,
		SizeHuman:       "2.02 GB",
		Description:     "Schnelles Allzweck-Modell von Meta AI. Gutes Deutsch, sehr effizient.",
		Languages:       []string{"Deutsch", "Englisch", "und weitere"},
		UseCases:        []string{"Chat", "Briefe", "Q&A", "Allgemeine Aufgaben"},
		License:         "Llama 3.2 Community License",
		Rating:          4.7,
		Downloads:       125000,
		MinRamGB:        4,
		RecommendedRamGB: 8,
		GPUAccelSupported: true,
		Featured:        true,
		Trending:        true,
		Category:        CategoryChat,
		ContextWindow:   "128K tokens",
		ContextSize:     131072, // 128K
	})

	r.models = append(r.models, ModelRegistryEntry{
		ID:              "qwen2.5-7b-instruct",
		DisplayName:     "Qwen 2.5 (7B) - Instruct",
		Provider:        "Alibaba Cloud",
		Architecture:    "qwen2",
		Version:         "2.5",
		ParameterSize:   "7B",
		Quantization:    "Q4_K_M",
		HuggingFaceRepo: "bartowski/Qwen2.5-7B-Instruct-GGUF",
		Filename:        "Qwen2.5-7B-Instruct-Q4_K_M.gguf",
		OllamaName:      "qwen2.5:7b",
		SizeBytes:       4_736_032_032,
		SizeHuman:       "4.73 GB",
		Description:     "Premium-Modell mit exzellenter Qualitaet auf Deutsch. Besonders stark bei komplexen Aufgaben.",
		Languages:       []string{"Deutsch", "Englisch", "Chinesisch", "und 25+ weitere"},
		UseCases:        []string{"Komplexe Texte", "Code", "Analyse", "Mehrsprachig", "Mathematik"},
		License:         "Apache 2.0",
		Rating:          4.9,
		Downloads:       89000,
		MinRamGB:        8,
		RecommendedRamGB: 16,
		GPUAccelSupported: true,
		Featured:        true,
		Trending:        true,
		Category:        CategoryChat,
		ContextWindow:   "128K tokens",
		ContextSize:     131072, // 128K
	})

	r.models = append(r.models, ModelRegistryEntry{
		ID:              "llama-3.1-8b-instruct",
		DisplayName:     "Llama 3.1 (8B) - Instruct",
		Provider:        "Meta AI",
		Architecture:    "llama",
		Version:         "3.1",
		ParameterSize:   "8B",
		Quantization:    "Q4_K_M",
		HuggingFaceRepo: "bartowski/Meta-Llama-3.1-8B-Instruct-GGUF",
		Filename:        "Meta-Llama-3.1-8B-Instruct-Q4_K_M.gguf",
		OllamaName:      "llama3.1:8b",
		SizeBytes:       4_920_000_000,
		SizeHuman:       "4.92 GB",
		Description:     "Metas neuestes 8B Modell mit 128K Context. Exzellent fuer Deutsch, schnell und vielseitig.",
		Languages:       []string{"Deutsch", "Englisch", "Franzoesisch", "Spanisch", "und weitere"},
		UseCases:        []string{"Chat", "Briefe", "Analyse", "Zusammenfassungen", "Code", "Lange Dokumente"},
		License:         "Llama 3.1 Community License",
		Rating:          4.9,
		Downloads:       185000,
		MinRamGB:        8,
		RecommendedRamGB: 12,
		GPUAccelSupported: true,
		Featured:        true,
		Trending:        true,
		Category:        CategoryChat,
		ReleaseDate:     "2024-07",
		TrainedUntil:    "2024-03",
		ContextWindow:   "128K tokens",
		ContextSize:     131072, // 128K
		PrimaryTasks:    "Chat, Analyse, Briefe, Code, lange Dokumente",
		Strengths:       "128K Context, Multilingual, Schnell, State-of-the-Art 8B Modell",
		Limitations:     "Llama Community License (kommerzielle Nutzung eingeschraenkt)",
	})

	// ===== CODE-GENERIERUNG =====

	r.models = append(r.models, ModelRegistryEntry{
		ID:              "qwen2.5-coder-3b-instruct",
		DisplayName:     "Qwen 2.5 Coder (3B) - Instruct",
		Provider:        "Alibaba Cloud",
		Architecture:    "qwen2",
		Version:         "2.5",
		ParameterSize:   "3B",
		Quantization:    "Q4_K_M",
		HuggingFaceRepo: "Qwen/Qwen2.5-Coder-3B-Instruct-GGUF",
		Filename:        "qwen2.5-coder-3b-instruct-q4_k_m.gguf",
		OllamaName:      "qwen2.5-coder:3b",
		SizeBytes:       1_967_004_960,
		SizeHuman:       "1.97 GB",
		Description:     "Code-fokussiertes LLM, trainiert auf 5.5T Tokens Code-Daten.",
		Languages:       []string{"Python", "Java", "C++", "JavaScript", "Go", "Rust", "SQL"},
		UseCases:        []string{"Code Generation", "Code Completion", "Code Explanation", "Debugging"},
		License:         "Apache 2.0",
		Rating:          4.8,
		Downloads:       54000,
		MinRamGB:        4,
		RecommendedRamGB: 8,
		GPUAccelSupported: true,
		Featured:        true,
		Category:        CategoryCode,
		ReleaseDate:     "2024-11",
		ContextWindow:   "128K tokens",
		ContextSize:     131072, // 128K
		PrimaryTasks:    "Code Generation, Code Completion, Code Explanation, Debugging",
		Strengths:       "Exzellente Code-Qualitaet, Multi-Language Support, 128k Context",
		Limitations:     "Nicht fuer natuerlichsprachige Konversation optimiert",
	})

	r.models = append(r.models, ModelRegistryEntry{
		ID:              "qwen2.5-coder-7b-instruct",
		DisplayName:     "Qwen 2.5 Coder (7B) - Instruct",
		Provider:        "Alibaba Cloud",
		Architecture:    "qwen2",
		Version:         "2.5",
		ParameterSize:   "7B",
		Quantization:    "Q4_K_M",
		HuggingFaceRepo: "bartowski/Qwen2.5-Coder-7B-Instruct-GGUF",
		Filename:        "Qwen2.5-Coder-7B-Instruct-Q4_K_M.gguf",
		OllamaName:      "qwen2.5-coder:7b",
		SizeBytes:       4_736_032_032,
		SizeHuman:       "4.73 GB",
		Description:     "Premium Code-Modell mit hoechster Qualitaet.",
		Languages:       []string{"Python", "Java", "C++", "JavaScript", "Go", "Rust", "SQL"},
		UseCases:        []string{"Komplexer Code", "Architektur", "Code Review", "Dokumentation"},
		License:         "Apache 2.0",
		Rating:          4.9,
		Downloads:       43000,
		MinRamGB:        8,
		RecommendedRamGB: 16,
		GPUAccelSupported: true,
		Featured:        true,
		Category:        CategoryCode,
		ReleaseDate:     "2024-11",
		ContextWindow:   "128K tokens",
		ContextSize:     131072, // 128K
	})

	r.models = append(r.models, ModelRegistryEntry{
		ID:              "deepseek-coder-6.7b-instruct",
		DisplayName:     "DeepSeek Coder (6.7B) - Instruct",
		Provider:        "DeepSeek AI",
		Architecture:    "deepseek",
		Version:         "1.0",
		ParameterSize:   "6.7B",
		Quantization:    "Q4_K_M",
		HuggingFaceRepo: "TheBloke/deepseek-coder-6.7B-instruct-GGUF",
		Filename:        "deepseek-coder-6.7b-instruct.Q4_K_M.gguf",
		OllamaName:      "deepseek-coder:6.7b",
		SizeBytes:       4_150_000_000,
		SizeHuman:       "4.15 GB",
		Description:     "TOP CODE-MODELL: State-of-the-Art Ergebnisse in Code-Benchmarks.",
		Languages:       []string{"Python", "Java", "C++", "JavaScript", "Go", "TypeScript", "C#"},
		UseCases:        []string{"Code Generation", "Code Completion", "Bug Fixing", "Code Explanation"},
		License:         "DeepSeek License",
		Rating:          4.9,
		Downloads:       156000,
		MinRamGB:        8,
		RecommendedRamGB: 12,
		GPUAccelSupported: true,
		Featured:        true,
		Trending:        true,
		Category:        CategoryCode,
		ReleaseDate:     "2024-01",
		ContextWindow:   "16K tokens",
		ContextSize:     16384, // 16K
		Strengths:       "State-of-the-Art Code Quality, 87% HumanEval, Multi-Language",
		Limitations:     "16K Context (kleiner als Qwen)",
	})

	// ===== VISION MODELLE =====

	r.models = append(r.models, ModelRegistryEntry{
		ID:              "llava-1.6-mistral-7b",
		DisplayName:     "LLaVA 1.6 Mistral (7B)",
		Provider:        "Haotian Liu (UW Madison)",
		Architecture:    "llava",
		Version:         "1.6",
		ParameterSize:   "7B",
		Quantization:    "Q4_K_M",
		HuggingFaceRepo: "cjpais/llava-1.6-mistral-7b-gguf",
		Filename:        "llava-v1.6-mistral-7b.Q4_K_M.gguf",
		OllamaName:      "llava:7b",
		SizeBytes:       4_370_000_000,
		SizeHuman:       "4.37 GB",
		Description:     "VISION-MODELL: Kombiniert Bildverstaendnis mit Sprachmodell.",
		Languages:       []string{"Englisch", "Deutsch (eingeschraenkt)", "Bilder"},
		UseCases:        []string{"Bildanalyse", "Bildbeschreibung", "OCR", "Visuelles Q&A"},
		License:         "Apache 2.0",
		Rating:          4.7,
		Downloads:       67000,
		MinRamGB:        8,
		RecommendedRamGB: 12,
		GPUAccelSupported: true,
		Featured:        true,
		Category:        CategoryVision,
		ReleaseDate:     "2024-01",
		ContextWindow:   "4K tokens",
		ContextSize:     4096, // 4K
		Strengths:       "Versteht Bilder, OCR, Multimodal, Gute Beschreibungen",
		Limitations:     "Benoetigt MMPROJ-Datei fuer Bildverarbeitung",
		IsVisionModel:   true,
		MmprojFilename:  "mmproj-model-f16.gguf",
	})

	r.models = append(r.models, ModelRegistryEntry{
		ID:              "llava-1.6-vicuna-13b",
		DisplayName:     "LLaVA 1.6 Vicuna (13B)",
		Provider:        "Haotian Liu (UW Madison)",
		Architecture:    "llava",
		Version:         "1.6",
		ParameterSize:   "13B",
		Quantization:    "Q4_K_M",
		HuggingFaceRepo: "cjpais/llava-v1.6-vicuna-13b-gguf",
		Filename:        "llava-v1.6-vicuna-13b.Q4_K_M.gguf",
		OllamaName:      "llava:13b",
		SizeBytes:       7_870_000_000,
		SizeHuman:       "7.87 GB",
		Description:     "VISION-MODELL: Groessere Version mit besserer Bildanalyse und detaillierteren Beschreibungen.",
		Languages:       []string{"Englisch", "Deutsch (eingeschraenkt)", "Bilder"},
		UseCases:        []string{"Komplexe Bildanalyse", "Dokumentenanalyse", "OCR", "Visuelles Reasoning"},
		License:         "Apache 2.0",
		Rating:          4.8,
		Downloads:       45000,
		MinRamGB:        12,
		RecommendedRamGB: 16,
		GPUAccelSupported: true,
		Featured:        false,
		Category:        CategoryVision,
		ReleaseDate:     "2024-01",
		ContextWindow:   "4K tokens",
		ContextSize:     4096, // 4K
		Strengths:       "Bessere Detailerkennung, Dokumentenanalyse, Komplexe Szenen",
		Limitations:     "Benoetigt mehr VRAM, Langsamer als 7B Version",
		IsVisionModel:   true,
		MmprojFilename:  "mmproj-model-f16.gguf",
	})

	r.models = append(r.models, ModelRegistryEntry{
		ID:              "moondream2",
		DisplayName:     "Moondream 2 (1.6B)",
		Provider:        "vikhyatk",
		Architecture:    "moondream",
		Version:         "2.0",
		ParameterSize:   "1.6B",
		Quantization:    "fp16",
		HuggingFaceRepo: "vikhyatk/moondream2",
		Filename:        "moondream2-text-model-f16.gguf",
		OllamaName:      "moondream:latest",
		SizeBytes:       3_300_000_000,
		SizeHuman:       "3.3 GB",
		Description:     "VISION-MODELL: Kleines aber effizientes Vision-Modell. Ideal fuer schwache Hardware.",
		Languages:       []string{"Englisch", "Bilder"},
		UseCases:        []string{"Schnelle Bildanalyse", "Objekterkennung", "Einfache OCR"},
		License:         "Apache 2.0",
		Rating:          4.4,
		Downloads:       89000,
		MinRamGB:        4,
		RecommendedRamGB: 6,
		GPUAccelSupported: true,
		Featured:        true,
		Category:        CategoryVision,
		ReleaseDate:     "2024-03",
		ContextWindow:   "2K tokens",
		ContextSize:     2048, // 2K
		Strengths:       "Sehr schnell, Wenig RAM, Gute Basisanalyse",
		Limitations:     "Weniger detailliert als LLaVA, Nur Englisch",
		IsVisionModel:   true,
		MmprojFilename:  "moondream2-mmproj-f16.gguf",
	})

	r.models = append(r.models, ModelRegistryEntry{
		ID:              "llava-phi3-mini",
		DisplayName:     "LLaVA Phi-3 Mini (3.8B)",
		Provider:        "xtuner",
		Architecture:    "llava-phi3",
		Version:         "1.0",
		ParameterSize:   "3.8B",
		Quantization:    "Q4_K_M",
		HuggingFaceRepo: "xtuner/llava-phi-3-mini-gguf",
		Filename:        "llava-phi-3-mini-int4.gguf",
		OllamaName:      "llava-phi3:latest",
		SizeBytes:       2_400_000_000,
		SizeHuman:       "2.4 GB",
		Description:     "VISION-MODELL: Kompakte LLaVA-Version basierend auf Phi-3. Guter Kompromiss aus Groesse und Qualitaet.",
		Languages:       []string{"Englisch", "Deutsch (eingeschraenkt)", "Bilder"},
		UseCases:        []string{"Bildanalyse", "Schnelle Antworten", "Mobile Anwendungen"},
		License:         "MIT",
		Rating:          4.5,
		Downloads:       34000,
		MinRamGB:        6,
		RecommendedRamGB: 8,
		GPUAccelSupported: true,
		Featured:        false,
		Category:        CategoryVision,
		ReleaseDate:     "2024-05",
		ContextWindow:   "4K tokens",
		ContextSize:     4096, // 4K
		Strengths:       "Schnell, Kompakt, Gute Balance aus Qualitaet und Groesse",
		Limitations:     "Nicht so detailliert wie groessere LLaVA Versionen",
		IsVisionModel:   true,
		MmprojFilename:  "llava-phi-3-mini-mmproj-f16.gguf",
	})

	r.models = append(r.models, ModelRegistryEntry{
		ID:              "bakllava-1",
		DisplayName:     "BakLLaVA (7B)",
		Provider:        "SkunkworksAI",
		Architecture:    "bakllava",
		Version:         "1.0",
		ParameterSize:   "7B",
		Quantization:    "Q4_K_M",
		HuggingFaceRepo: "mys/ggml_bakllava-1",
		Filename:        "ggml-model-q4_k.gguf",
		OllamaName:      "bakllava:latest",
		SizeBytes:       4_080_000_000,
		SizeHuman:       "4.08 GB",
		Description:     "VISION-MODELL: Alternative zu LLaVA mit Mistral-Basis. Oft bessere Reasoning-Faehigkeiten.",
		Languages:       []string{"Englisch", "Bilder"},
		UseCases:        []string{"Bildanalyse", "Visuelles Reasoning", "Komplexe Fragen zu Bildern"},
		License:         "Apache 2.0",
		Rating:          4.6,
		Downloads:       28000,
		MinRamGB:        8,
		RecommendedRamGB: 12,
		GPUAccelSupported: true,
		Featured:        false,
		Category:        CategoryVision,
		ReleaseDate:     "2023-12",
		ContextWindow:   "4K tokens",
		ContextSize:     4096, // 4K
		Strengths:       "Gutes Reasoning, Mistral-Basis, Detaillierte Antworten",
		Limitations:     "Nur Englisch, Aeltere Version",
		IsVisionModel:   true,
		MmprojFilename:  "mmproj-model-f16.gguf",
	})

	// ===== KOMPAKTE MODELLE =====

	r.models = append(r.models, ModelRegistryEntry{
		ID:              "qwen2.5-1.5b-instruct",
		DisplayName:     "Qwen 2.5 (1.5B) - Instruct",
		Provider:        "Alibaba Cloud",
		Architecture:    "qwen2",
		Version:         "2.5",
		ParameterSize:   "1.5B",
		Quantization:    "Q4_K_M",
		HuggingFaceRepo: "Qwen/Qwen2.5-1.5B-Instruct-GGUF",
		Filename:        "qwen2.5-1.5b-instruct-q4_k_m.gguf",
		OllamaName:      "qwen2.5:1.5b",
		SizeBytes:       1_049_000_000,
		SizeHuman:       "1.05 GB",
		Description:     "Kompaktes mehrsprachiges Modell. Besseres Deutsch als Llama 1B.",
		Languages:       []string{"Deutsch", "Englisch", "und weitere"},
		UseCases:        []string{"Chat", "Briefe", "Q&A", "Fuer schwache PCs"},
		License:         "Apache 2.0",
		Rating:          4.5,
		Downloads:       78000,
		MinRamGB:        3,
		RecommendedRamGB: 4,
		GPUAccelSupported: true,
		Featured:        false,
		Category:        CategoryCompact,
		ContextWindow:   "32K tokens",
		ContextSize:     32768, // 32K
	})

	r.models = append(r.models, ModelRegistryEntry{
		ID:              "mistral-7b-instruct",
		DisplayName:     "Mistral 7B v0.3 - Instruct",
		Provider:        "Mistral AI",
		Architecture:    "mistral",
		Version:         "0.3",
		ParameterSize:   "7B",
		Quantization:    "Q4_K_M",
		HuggingFaceRepo: "MaziyarPanahi/Mistral-7B-Instruct-v0.3-GGUF",
		Filename:        "Mistral-7B-Instruct-v0.3.Q4_K_M.gguf",
		OllamaName:      "mistral:7b",
		SizeBytes:       4_368_438_688,
		SizeHuman:       "4.37 GB",
		Description:     "Balanced Allrounder-Modell. Gutes Deutsch, sehr vielseitig.",
		Languages:       []string{"Deutsch", "Englisch", "Franzoesisch", "und weitere"},
		UseCases:        []string{"Chat", "Analyse", "Zusammenfassungen", "Vielseitig"},
		License:         "Apache 2.0",
		Rating:          4.7,
		Downloads:       98000,
		MinRamGB:        8,
		RecommendedRamGB: 12,
		GPUAccelSupported: true,
		Featured:        false,
		Category:        CategoryChat,
		ContextWindow:   "32K tokens",
		ContextSize:     32768, // 32K
	})

	r.models = append(r.models, ModelRegistryEntry{
		ID:              "phi-3-mini-instruct",
		DisplayName:     "Phi-3 Mini (3.8B) - Instruct",
		Provider:        "Microsoft",
		Architecture:    "phi3",
		Version:         "3",
		ParameterSize:   "3.8B",
		Quantization:    "Q4_K_M",
		HuggingFaceRepo: "microsoft/Phi-3-mini-4k-instruct-gguf",
		Filename:        "Phi-3-mini-4k-instruct-q4.gguf",
		OllamaName:      "phi3:mini",
		SizeBytes:       2_356_934_272,
		SizeHuman:       "2.36 GB",
		Description:     "Kompaktes High-Performance Modell von Microsoft. Gutes Deutsch trotz kleiner Groesse.",
		Languages:       []string{"Deutsch", "Englisch", "und weitere"},
		UseCases:        []string{"Chat", "Q&A", "Zusammenfassungen", "Schnelle Antworten"},
		License:         "MIT",
		Rating:          4.6,
		Downloads:       67000,
		MinRamGB:        4,
		RecommendedRamGB: 8,
		GPUAccelSupported: true,
		Featured:        true,
		Category:        CategoryChat,
		ContextWindow:   "4K tokens",
		ContextSize:     4096, // 4K (Phi-3-mini-4k-instruct)
	})
}

// GetAllModels gibt alle Modelle zurueck
func (r *ModelRegistry) GetAllModels() []ModelRegistryEntry {
	return r.models
}

// GetFeaturedModels gibt Featured-Modelle zurueck
func (r *ModelRegistry) GetFeaturedModels() []ModelRegistryEntry {
	result := make([]ModelRegistryEntry, 0)
	for _, m := range r.models {
		if m.Featured {
			result = append(result, m)
		}
	}
	// Nach Rating sortieren
	sort.Slice(result, func(i, j int) bool {
		return result[i].Rating > result[j].Rating
	})
	return result
}

// GetTrendingModels gibt Trending-Modelle zurueck
func (r *ModelRegistry) GetTrendingModels() []ModelRegistryEntry {
	result := make([]ModelRegistryEntry, 0)
	for _, m := range r.models {
		if m.Trending {
			result = append(result, m)
		}
	}
	// Nach Downloads sortieren
	sort.Slice(result, func(i, j int) bool {
		return result[i].Downloads > result[j].Downloads
	})
	if len(result) > 5 {
		result = result[:5]
	}
	return result
}

// GetByCategory gibt Modelle einer Kategorie zurueck
func (r *ModelRegistry) GetByCategory(category ModelCategory) []ModelRegistryEntry {
	result := make([]ModelRegistryEntry, 0)
	for _, m := range r.models {
		if m.Category == category {
			result = append(result, m)
		}
	}
	return result
}

// GetByMaxRAM gibt Modelle bis zu einer bestimmten RAM-Groesse zurueck
func (r *ModelRegistry) GetByMaxRAM(maxRAMGB int) []ModelRegistryEntry {
	result := make([]ModelRegistryEntry, 0)
	for _, m := range r.models {
		if m.MinRamGB <= maxRAMGB {
			result = append(result, m)
		}
	}
	// Nach Rating sortieren
	sort.Slice(result, func(i, j int) bool {
		return result[i].Rating > result[j].Rating
	})
	return result
}

// FindByID findet ein Modell nach ID
func (r *ModelRegistry) FindByID(id string) *ModelRegistryEntry {
	for _, m := range r.models {
		if m.ID == id {
			return &m
		}
	}
	return nil
}

// FindByOllamaName findet ein Modell nach Ollama-Name
func (r *ModelRegistry) FindByOllamaName(name string) *ModelRegistryEntry {
	// Normalisieren (z.B. "qwen2.5:7b" oder "qwen2.5")
	normalizedName := strings.ToLower(name)

	for _, m := range r.models {
		if strings.ToLower(m.OllamaName) == normalizedName {
			return &m
		}
		// Auch ohne Tag pruefen
		if strings.HasPrefix(strings.ToLower(m.OllamaName), normalizedName+":") ||
		   strings.HasPrefix(normalizedName, strings.ToLower(m.OllamaName)+":") {
			return &m
		}
	}
	return nil
}

// FindByFilename findet ein Modell nach GGUF-Dateiname
func (r *ModelRegistry) FindByFilename(filename string) *ModelRegistryEntry {
	normalizedFilename := strings.ToLower(filename)

	for _, m := range r.models {
		if strings.ToLower(m.Filename) == normalizedFilename {
			return &m
		}
		// Auch teilweise Übereinstimmung prüfen (z.B. "Qwen2.5-7B-Instruct-Q5_K_M.gguf" vs "qwen2.5-7b-instruct-q4_k_m.gguf")
		// Extrahiere Basis-Name ohne Quantisierung
		baseName := strings.Split(normalizedFilename, "-q")[0]
		if baseName != "" && strings.Contains(strings.ToLower(m.Filename), baseName) {
			return &m
		}
	}
	return nil
}

// Search sucht Modelle nach Suchbegriff
func (r *ModelRegistry) Search(query string) []ModelRegistryEntry {
	query = strings.ToLower(query)
	result := make([]ModelRegistryEntry, 0)

	for _, m := range r.models {
		if strings.Contains(strings.ToLower(m.DisplayName), query) ||
		   strings.Contains(strings.ToLower(m.Description), query) ||
		   containsIgnoreCase(m.Languages, query) ||
		   containsIgnoreCase(m.UseCases, query) {
			result = append(result, m)
		}
	}
	return result
}

// containsIgnoreCase prueft ob ein String in einem Slice enthalten ist (case-insensitive)
func containsIgnoreCase(slice []string, s string) bool {
	for _, item := range slice {
		if strings.Contains(strings.ToLower(item), s) {
			return true
		}
	}
	return false
}

// AddModel fuegt ein Modell hinzu
func (r *ModelRegistry) AddModel(model ModelRegistryEntry) {
	r.models = append(r.models, model)
}

// GetVisionModels gibt alle Vision-Modelle zurueck
func (r *ModelRegistry) GetVisionModels() []ModelRegistryEntry {
	result := make([]ModelRegistryEntry, 0)
	for _, m := range r.models {
		if m.IsVisionModel || m.Category == CategoryVision {
			result = append(result, m)
		}
	}
	return result
}

// GetCodeModels gibt alle Code-Modelle zurueck
func (r *ModelRegistry) GetCodeModels() []ModelRegistryEntry {
	return r.GetByCategory(CategoryCode)
}

// GetCompactModels gibt alle kompakten Modelle zurueck (< 4GB RAM)
func (r *ModelRegistry) GetCompactModels() []ModelRegistryEntry {
	return r.GetByMaxRAM(4)
}

// GetModelContextSize gibt die maximale Context-Größe für ein Modell zurück
// Sucht sowohl nach Ollama-Name als auch nach Dateiname
// Gibt 0 zurück wenn nicht gefunden (dann wird Default verwendet)
func (r *ModelRegistry) GetModelContextSize(modelName string) int {
	// Erst nach Ollama-Name suchen
	if entry := r.FindByOllamaName(modelName); entry != nil && entry.ContextSize > 0 {
		return entry.ContextSize
	}

	// Dann nach Dateiname suchen
	if entry := r.FindByFilename(modelName); entry != nil && entry.ContextSize > 0 {
		return entry.ContextSize
	}

	// Nach ID suchen
	if entry := r.FindByID(modelName); entry != nil && entry.ContextSize > 0 {
		return entry.ContextSize
	}

	return 0 // Nicht gefunden → Default (64K) verwenden
}

// DefaultContextSize ist die Standard-Context-Größe wenn kein Modell-spezifischer Wert existiert
const DefaultContextSize = 16384 // 16K (65K braucht zu viel VRAM für die meisten GPUs)

// GetEffectiveContextSize gibt die effektive Context-Größe für ein Modell zurück
// Nutzt den Modell-spezifischen Wert oder den Default (64K)
// Der Experten-Context kann diese weiter einschränken
func (r *ModelRegistry) GetEffectiveContextSize(modelName string, expertContextOverride int) int {
	modelMax := r.GetModelContextSize(modelName)

	// Wenn Modell einen Wert hat und er kleiner als Default ist, nutzen
	if modelMax > 0 && modelMax < DefaultContextSize {
		// Experten-Override nur wenn kleiner als Modell-Max
		if expertContextOverride > 0 && expertContextOverride < modelMax {
			return expertContextOverride
		}
		return modelMax
	}

	// Default: 64K, außer Experte will weniger
	if expertContextOverride > 0 && expertContextOverride < DefaultContextSize {
		return expertContextOverride
	}

	// Wenn Modell größeren Context hat, trotzdem Default nutzen (VRAM-Optimierung)
	// User kann über Slider erhöhen wenn gewünscht
	return DefaultContextSize
}
