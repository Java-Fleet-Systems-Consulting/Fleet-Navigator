# Changelog 2025-12-25

## Übersicht

Heute wurden wichtige Verbesserungen am Experten-System implementiert:
- **Anti-Halluzinations-Prompt** jetzt pro Experte konfigurierbar
- **DELEGATE-Tag** für automatische Experten-Umschaltung
- **Unit-Tests** für neue Funktionen

---

## 1. Anti-Halluzinations-System verbessert

### Problem
Das LLM halluzinierte bei Identitätsfragen ("Wer bist du?"):
- Erfand Details wie "Ewa Marek, polnische Wurzeln"
- Zitierte irrelevante Quellen (z.B. Unternehmens-Links bei persönlichen Fragen)

### Lösung

**Verstärkte Anti-Halluzinations-Regeln** in `internal/experte/expert.go`:

```go
const DefaultAntiHallucinationPrompt = `
## KRITISCH - KEINE HALLUZINATIONEN!
- Erfinde NIEMALS Informationen, Fakten, Namen oder Quellen
- Wenn du etwas nicht weißt, sage es ehrlich

## IDENTITÄT - WER DU BIST
- Bei Fragen wie "Wer bist du?" antworte NUR mit Informationen aus diesem System-Prompt
- Erfinde KEINE zusätzlichen Details über dich selbst
- Zitiere bei Identitätsfragen KEINE externen Quellen

## QUELLEN - NUR WENN RELEVANT
- Zitiere Quellen NUR wenn sie DIREKT zur gestellten Frage passen
- Bei persönlichen Fragen (Wer bist du? Wie geht es dir?) → KEINE Quellen nötig
- NIEMALS zufällige oder thematisch unpassende Quellen einfügen
`
```

### Konfigurierbar pro Experte

Neues Feld `antiHallucinationPrompt` in Expert-Struct:
- Leer = Default-Prompt wird verwendet
- Gesetzt = Custom-Prompt für diesen Experten

**API Endpoint:**
```
GET /api/experts/default-anti-hallucination
→ Gibt den Standard-Anti-Halluzinations-Prompt zurück (für Reset-Button)
```

**DB-Migration:**
```sql
ALTER TABLE experts ADD COLUMN anti_hallucination_prompt TEXT DEFAULT ''
```

### WebSearchShowLinks Default

Geändert von `true` auf `false`:
- RAG-Modus: Quellen werden intern genutzt, aber nicht angezeigt
- Verhindert irrelevante Quellen-Zitate

---

## 2. DELEGATE-Tag für Experten-Umschaltung

### Problem
Der `[[DELEGATE:ExpertName]]` Tag war in den Experten-Prompts definiert, wurde aber nie verarbeitet:
```
User: "Ich habe ein Problem mit einem Mietvertrag"
Ewa: "Das klingt nach Mietrecht. Roland kann dir besser helfen."
     [[DELEGATE:Roland]]  ← War sichtbar im Chat!
```

### Lösung

**Backend-Implementierung** in `cmd/navigator/main.go`:

```go
// DELEGATE-Tag prüfen und verarbeiten
delegatePattern := regexp.MustCompile(`\[\[DELEGATE:([^\]]+)\]\]`)
if matches := delegatePattern.FindStringSubmatch(fullResponse); len(matches) > 1 {
    delegateExpertName := strings.TrimSpace(matches[1])

    // Experten nach Name suchen (case-insensitive, Vorname oder voller Name)
    allExperts, _ := app.expertenService.GetAllExperts(true)
    for _, exp := range allExperts {
        if strings.EqualFold(exp.Name, delegateExpertName) ||
           strings.HasPrefix(strings.ToLower(exp.Name), strings.ToLower(delegateExpertName)) {
            delegatedToExpert = &exp
            break
        }
    }

    // Tag aus Antwort entfernen
    fullResponse = strings.TrimSpace(delegatePattern.ReplaceAllString(fullResponse, ""))
}

// SSE Event an Frontend senden
if delegatedToExpert != nil {
    delegateData := map[string]interface{}{
        "type":         "delegation",
        "expertId":     delegatedToExpert.ID,
        "expertName":   delegatedToExpert.Name,
        "expertAvatar": delegatedToExpert.Avatar,
        "message":      fmt.Sprintf("Ich verbinde dich mit %s...", delegatedToExpert.Name),
    }
    // ... senden via SSE
}
```

**Frontend-Handler** (bereits vorhanden in `chatStore.js`):
```javascript
} else if (parsed.type === 'delegation') {
    // System-Nachricht anzeigen
    const delegationMsg = {
        role: 'SYSTEM',
        content: `🔄 ${parsed.message}`,
        isDelegationMessage: true,
        expertAvatar: parsed.expertAvatar
    }
    messages.value.push(delegationMsg)

    // Experten wechseln
    setTimeout(() => {
        selectedExpertId.value = parsed.expertId
    }, 500)
}
```

---

## 3. Frontend UI für Anti-Halluzination

**CreateExpertModal.vue:**
- Neuer ausklappbarer Bereich "Anti-Halluzinations-Regeln"
- Textarea für Custom-Prompt
- Reset-Button lädt Default-Prompt via API

**api.js:**
```javascript
async getDefaultAntiHallucinationPrompt() {
    const response = await api.get('/experts/default-anti-hallucination')
    return response.data.prompt
}
```

**i18n (DE/EN/TR):**
```json
"antiHallucination": {
    "title": "Anti-Halluzinations-Regeln",
    "description": "Regeln die verhindern, dass das LLM Fakten erfindet",
    "placeholder": "Leer = Standard-Regeln verwenden",
    "resetToDefault": "Standard wiederherstellen"
}
```

---

## 4. Unit-Tests

### Backend Tests

**internal/experte/expert_test.go:**
- `TestCustomAntiHallucinationPrompt` - Custom-Prompt wird verwendet
- `TestDefaultAntiHallucinationPromptUsedWhenEmpty` - Default bei leerem Feld
- `TestDefaultExpertsWebSearchShowLinksIsFalse` - RAG-Modus Default
- `TestDefaultExpertsHaveAutoWebSearch` - Auto-WebSearch aktiviert
- `TestGetFullPromptRAGMode` - RAG-Modus in GetFullPrompt

### Frontend Tests

**web/src/__tests__/:**
- InfoDialog.spec.ts
- HelpView.spec.ts
- App.spec.ts (Setup-Wizard)

---

## Commits

```
4227986 fix: Verstärkte Anti-Halluzinations-Regeln für Experten
c2a11ac feat: Konfigurierbarer Anti-Halluzinations-Prompt pro Experte
6c233d8 feat: Frontend UI für konfigurierbaren Anti-Halluzinations-Prompt
f3981b6 feat: DELEGATE-Tag für automatische Experten-Umschaltung
```

---

## Offene Frontend-Arbeiten (für morgen mit Kat)

1. **DELEGATE-System testen** - Vollständiger Flow im Browser prüfen
2. **System-Nachricht Styling** - Delegation-Nachricht schöner gestalten
3. **Anti-Halluzination UI** - CreateExpertModal.vue finalisieren
4. **Experten-Avatar Animation** - Beim Wechsel animieren

---

## Dateien geändert

### Backend
- `cmd/navigator/main.go` - DELEGATE-Tag Parser, API-Endpoint
- `internal/experte/expert.go` - AntiHallucinationPrompt, DefaultAntiHallucinationPrompt
- `internal/experte/repository.go` - DB-Migration, CRUD für neues Feld
- `internal/experte/expert_test.go` - Unit-Tests

### Frontend
- `web/src/components/CreateExpertModal.vue` - Anti-Halluzination UI
- `web/src/services/api.js` - getDefaultAntiHallucinationPrompt()
- `web/src/i18n/locales/de.json` - Übersetzungen
- `web/src/i18n/locales/en.json` - Übersetzungen
- `web/src/i18n/locales/tr.json` - Übersetzungen
