# Changelog 2025-12-27

## Übersicht

Bugfixes, GitHub Actions Workflow-Reparatur und SettingsModal Modularisierung abgeschlossen.

---

## Bugfixes

### 1. Model-Swap UI Text korrigiert
**Problem:** "Vision-Modell wird geladen" wurde bei JEDEM Modell-Wechsel angezeigt, nicht nur bei Vision-Modellen.

**Lösung:** Text in `ChatWindow.vue` generisch gemacht:
- "Vision-Modell wird geladen" → "Modell wird geladen"
- "Wechsle zum Bildanalyse-Modell..." → "Wechsle zum passenden Modell..."
- "VRAM-Optimierung für GPUs..." → "Modell-Wechsel für optimale Antworten"

**Datei:** `web/src/components/ChatWindow.vue:100`

### 2. Race-Condition bei Model-Swap behoben
**Problem:** Nach Model-Swap kam "llama-server ist nicht aktiv" Fehler, weil `SwitchModel()` zurückkehrte bevor der Server wirklich bereit war.

**Lösung:** Neue `WaitForHealthy()` Funktion die auf Health-Check wartet:
```go
func (s *Server) WaitForHealthy(timeout time.Duration) error {
    deadline := time.Now().Add(timeout)
    for time.Now().Before(deadline) {
        if s.IsHealthy() {
            return nil
        }
        time.Sleep(500 * time.Millisecond)
    }
    return fmt.Errorf("llama-server nicht bereit nach %v", timeout)
}
```

`SwitchModel()` ruft jetzt `WaitForHealthy(30 * time.Second)` auf bevor es zurückkehrt.

**Dateien:**
- `internal/llamaserver/server.go` - WaitForHealthy + SwitchModel
- `internal/llamaserver/server_test.go` - 2 neue Unit-Tests

---

## GitHub Actions Workflow

### Workflow-Vereinfachung (vorher)
Der ursprüngliche Workflow baute Whisper und llama-server Binaries selbst - unnötig, da diese bereits auf `mirror.java-fleet.com` liegen.

**Reduzierung:**
- 338 → 147 Zeilen (-56%)
- 7 → 3 Jobs
- Build-Zeit: ~15 Min → ~2 Min

### Fixes heute

1. **Artifact-Pfad korrigiert**
   - Fehler: `No files were found with the provided path: web/dist/`
   - Ursache: Vite outputDir ist `../cmd/navigator/dist`, nicht `web/dist/`
   - Fix: `path: cmd/navigator/dist/` in upload-artifact

2. **Release Permissions hinzugefügt**
   - Fehler: 403 bei `softprops/action-gh-release`
   - Ursache: Fehlende Berechtigungen für Release-Erstellung
   - Fix: `permissions: contents: write` im release job

**Datei:** `.github/workflows/build-release.yml`

---

## SettingsModal Modularisierung (Phase 3 abgeschlossen)

### Vorher
- 2871 Zeilen in SettingsModal.vue
- 2 Tabs noch inline (web-search, voice)

### Nachher
- 2055 Zeilen (-816 Zeilen, -28%)
- Alle 11 Tabs als separate Komponenten

### Änderungen

**Inline web-search Tab ersetzt (~428 Zeilen):**
```vue
<WebSearchSettingsTab
  :web-search-settings="webSearchSettings"
  :testing-search="testingBraveSearch || testingSearxng"
  @update:web-search-settings="webSearchSettings = $event"
  @test-brave="testBraveSearch"
  @test-searxng="testCustomSearxng"
  @save-settings="saveWebSearchSettings"
/>
```

**Inline voice Tab ersetzt (~418 Zeilen):**
```vue
<VoiceSettingsTab
  :tts-enabled="settings.ttsEnabled"
  :voice-downloading="voiceDownloading"
  :voice-download-component="voiceDownloadComponent"
  :voice-download-status="voiceDownloadStatus"
  :voice-download-progress="voiceDownloadProgress"
  :voice-download-speed="voiceDownloadSpeed"
  :voice-models="voiceModels"
  :voice-assistant-settings="voiceAssistantSettings"
  @toggle-tts="toggleTtsEnabled"
  @download-whisper="downloadWhisper"
  @download-piper="downloadPiper"
  @download-model="downloadVoiceModel"
  @select-whisper-model="selectWhisperModel"
  @select-piper-voice="selectPiperVoice"
  @update:voice-assistant-settings="voiceAssistantSettings = $event"
/>
```

**Datei:** `web/src/components/SettingsModal.vue`

---

## Release v0.8.14

**Veröffentlicht:** 2025-12-27T20:35:17Z

**Assets:**
- `fleet-navigator-linux-amd64` - Linux x64
- `fleet-navigator-windows-amd64.exe` - Windows x64
- `fleet-navigator-darwin-amd64` - macOS Intel
- `fleet-navigator-darwin-arm64` - macOS Apple Silicon

**URL:** https://github.com/Java-Fleet-Systems-Consulting/Fleet-Navigator/releases/tag/v0.8.14

---

## Commits

1. `ec745f1` - fix: Workflow Artifact-Pfad + SettingsModal Modularisierung
2. `aef5d68` - fix: GitHub Release Permissions hinzugefügt

---

## Nächste Schritte (offen)

- [ ] Issue #18: Wake Word Audio funktioniert nicht
- [ ] Issue #19: Tesseract OCR Binary-Pfad/Config
- [ ] FR/ES Übersetzungen vervollständigen
