// Package modeltemplates verwaltet Chat-Templates für verschiedene LLM-Familien
package modeltemplates

import "time"

// ModelTemplate definiert das Chat-Template für eine Modell-Familie
type ModelTemplate struct {
	ID                  int64     `json:"id"`
	Name                string    `json:"name"`                // z.B. "Qwen", "Gemma", "Llama 3"
	Pattern             string    `json:"pattern"`             // Regex-Pattern für Modellname, z.B. "(?i)qwen"
	TemplateFormat      string    `json:"templateFormat"`      // z.B. "chatml", "gemma", "llama3"
	SupportsSystemRole  bool      `json:"supportsSystemRole"`  // Unterstützt native System-Rolle
	SystemEmbedStrategy string    `json:"systemEmbedStrategy"` // "native", "embed_in_user", "prepend_user"
	SystemPrefix        string    `json:"systemPrefix"`        // Prefix für eingebetteten System-Prompt
	SystemSuffix        string    `json:"systemSuffix"`        // Suffix für eingebetteten System-Prompt
	Description         string    `json:"description"`         // Beschreibung des Templates
	Priority            int       `json:"priority"`            // Höhere Priorität = wird zuerst geprüft
	IsActive            bool      `json:"isActive"`            // Template aktiv?
	CreatedAt           time.Time `json:"createdAt"`
	UpdatedAt           time.Time `json:"updatedAt"`
}

// SystemEmbedStrategy Konstanten
const (
	// StrategyNative - System-Prompt als separate Rolle senden (Standard für Qwen, Llama)
	StrategyNative = "native"
	// StrategyEmbedInUser - System-Prompt in erste User-Nachricht einbetten (für Gemma)
	StrategyEmbedInUser = "embed_in_user"
	// StrategyPrependUser - System-Prompt als erste User-Nachricht voranstellen
	StrategyPrependUser = "prepend_user"
)

// TemplateFormat Konstanten
const (
	FormatChatML  = "chatml"  // Qwen, Phi, Yi
	FormatGemma   = "gemma"   // Google Gemma
	FormatLlama2  = "llama2"  // Meta Llama 2
	FormatLlama3  = "llama3"  // Meta Llama 3
	FormatMistral = "mistral" // Mistral AI
	FormatAlpaca  = "alpaca"  // Alpaca-Format
	FormatVicuna  = "vicuna"  // Vicuna-Format
	FormatZephyr  = "zephyr"  // Zephyr-Format
)

// DefaultTemplates gibt die Standard-Templates zurück
func DefaultTemplates() []ModelTemplate {
	return []ModelTemplate{
		{
			Name:                "Qwen (ChatML)",
			Pattern:             "(?i)qwen",
			TemplateFormat:      FormatChatML,
			SupportsSystemRole:  true,
			SystemEmbedStrategy: StrategyNative,
			Description:         "Alibaba Qwen Modelle - volle System-Prompt Unterstützung",
			Priority:            100,
			IsActive:            true,
		},
		{
			Name:                "Gemma (Google)",
			Pattern:             "(?i)gemma",
			TemplateFormat:      FormatGemma,
			SupportsSystemRole:  false,
			SystemEmbedStrategy: StrategyEmbedInUser,
			SystemPrefix:        "[SYSTEM-ANWEISUNGEN - BEFOLGE DIESE STRIKT]\n",
			SystemSuffix:        "\n[ENDE DER SYSTEM-ANWEISUNGEN]\n\nBenutzer-Nachricht: ",
			Description:         "Google Gemma - System-Prompt wird in User-Nachricht eingebettet",
			Priority:            100,
			IsActive:            true,
		},
		{
			Name:                "Llama 3",
			Pattern:             "(?i)(llama-?3|llama.?3)",
			TemplateFormat:      FormatLlama3,
			SupportsSystemRole:  true,
			SystemEmbedStrategy: StrategyNative,
			Description:         "Meta Llama 3 - volle System-Prompt Unterstützung",
			Priority:            100,
			IsActive:            true,
		},
		{
			Name:                "Llama 2",
			Pattern:             "(?i)(llama-?2|llama.?2)",
			TemplateFormat:      FormatLlama2,
			SupportsSystemRole:  true,
			SystemEmbedStrategy: StrategyNative,
			Description:         "Meta Llama 2 - volle System-Prompt Unterstützung",
			Priority:            90,
			IsActive:            true,
		},
		{
			Name:                "Mistral",
			Pattern:             "(?i)(mistral|mixtral)",
			TemplateFormat:      FormatMistral,
			SupportsSystemRole:  false,
			SystemEmbedStrategy: StrategyEmbedInUser,
			SystemPrefix:        "[Kontext]\n",
			SystemSuffix:        "\n[/Kontext]\n\n",
			Description:         "Mistral AI - System-Prompt eingeschränkt, wird eingebettet",
			Priority:            100,
			IsActive:            true,
		},
		{
			Name:                "Phi (Microsoft)",
			Pattern:             "(?i)phi",
			TemplateFormat:      FormatChatML,
			SupportsSystemRole:  true,
			SystemEmbedStrategy: StrategyNative,
			Description:         "Microsoft Phi - ChatML Format mit System-Support",
			Priority:            100,
			IsActive:            true,
		},
		{
			Name:                "Yi",
			Pattern:             "(?i)yi-",
			TemplateFormat:      FormatChatML,
			SupportsSystemRole:  true,
			SystemEmbedStrategy: StrategyNative,
			Description:         "01.AI Yi - ChatML Format mit System-Support",
			Priority:            100,
			IsActive:            true,
		},
		{
			Name:                "Falcon",
			Pattern:             "(?i)falcon",
			TemplateFormat:      FormatChatML,
			SupportsSystemRole:  true,
			SystemEmbedStrategy: StrategyNative,
			Description:         "TII Falcon - System-Prompt unterstützt",
			Priority:            100,
			IsActive:            true,
		},
		{
			Name:                "Default (Fallback)",
			Pattern:             ".*",
			TemplateFormat:      FormatChatML,
			SupportsSystemRole:  true,
			SystemEmbedStrategy: StrategyNative,
			Description:         "Standard-Fallback für unbekannte Modelle",
			Priority:            0, // Niedrigste Priorität
			IsActive:            true,
		},
	}
}
