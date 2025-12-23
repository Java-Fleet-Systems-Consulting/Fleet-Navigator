<template>
  <div class="help-page">
    <header class="help-header">
      <div class="help-logo" @click="goBack">
        <span class="logo-icon">🚢</span>
        <span class="logo-text">Fleet Navigator</span>
      </div>
      <button class="close-btn" @click="closeHelp">✕</button>
    </header>

    <main class="help-content">
      <!-- Topic Navigation -->
      <nav class="topic-nav">
        <button
          v-for="topic in topics"
          :key="topic.id"
          class="topic-btn"
          :class="{ active: currentTopic === topic.id }"
          @click="currentTopic = topic.id"
        >
          <span class="topic-icon">{{ topic.icon }}</span>
          <span class="topic-name">{{ topic.name }}</span>
        </button>
      </nav>

      <!-- Topic Content -->
      <article class="topic-content">
        <!-- Vision Model -->
        <div v-if="currentTopic === 'vision'" class="topic-article">
          <h1>👁️ Was ist ein Vision-Modell?</h1>

          <div class="intro-box">
            <p>Ein <strong>Vision-Modell</strong> ist eine KI, die <em>Bilder verstehen</em> kann - nicht nur Text.</p>
          </div>

          <h2>Was kann ein Vision-Modell?</h2>
          <div class="feature-grid">
            <div class="feature-card">
              <span class="icon">📄</span>
              <h3>Dokumente lesen</h3>
              <p>Rechnungen, Briefe, Verträge analysieren und den Inhalt extrahieren</p>
            </div>
            <div class="feature-card">
              <span class="icon">📷</span>
              <h3>Bilder beschreiben</h3>
              <p>Fotos analysieren und beschreiben was darauf zu sehen ist</p>
            </div>
            <div class="feature-card">
              <span class="icon">✉️</span>
              <h3>Text erkennen (OCR)</h3>
              <p>Gedruckten und handgeschriebenen Text aus Bildern extrahieren</p>
            </div>
            <div class="feature-card">
              <span class="icon">🏷️</span>
              <h3>Kategorisieren</h3>
              <p>Dokumente automatisch nach Typ einordnen</p>
            </div>
          </div>

          <h2>Vergleich mit ChatGPT Vision</h2>
          <div class="comparison-box">
            <div class="compare-item cloud">
              <h4>☁️ ChatGPT Vision (GPT-4V)</h4>
              <ul>
                <li>Läuft in der Cloud bei OpenAI</li>
                <li>Bilder werden hochgeladen</li>
                <li>Kosten pro Bild/Anfrage</li>
                <li>Internet erforderlich</li>
              </ul>
            </div>
            <div class="compare-item local">
              <h4>💻 Fleet Navigator Vision (MiniCPM-V)</h4>
              <ul>
                <li>Läuft lokal auf deinem PC</li>
                <li>Bilder verlassen nie den Rechner</li>
                <li>Keine laufenden Kosten</li>
                <li>Funktioniert offline</li>
              </ul>
            </div>
          </div>

          <div class="info-callout">
            <span class="callout-icon">💡</span>
            <p><strong>MiniCPM-V 2.6</strong> erreicht auf Benchmarks GPT-4V-Niveau und ist besonders gut bei Dokumentenerkennung und OCR.</p>
          </div>
        </div>

        <!-- Instruct Model -->
        <div v-if="currentTopic === 'instruct'" class="topic-article">
          <h1>🎯 Was ist ein Instruct-Modell?</h1>

          <div class="intro-box">
            <p>Ein <strong>Instruct-Modell</strong> wurde speziell darauf trainiert, <em>Anweisungen zu befolgen</em> und hilfreiche Antworten zu geben.</p>
          </div>

          <h2>Base vs. Instruct</h2>
          <div class="comparison-table">
            <div class="table-header">
              <span>Eigenschaft</span>
              <span>Base-Modell</span>
              <span>Instruct-Modell</span>
            </div>
            <div class="table-row">
              <span>Training</span>
              <span>Texte aus dem Internet</span>
              <span>+ Anweisungen & Feedback</span>
            </div>
            <div class="table-row">
              <span>Verhalten</span>
              <span>Vervollständigt Text</span>
              <span>Beantwortet Fragen</span>
            </div>
            <div class="table-row">
              <span>Beispiel-Input</span>
              <span>"Die Hauptstadt von Frankreich"</span>
              <span>"Was ist die Hauptstadt von Frankreich?"</span>
            </div>
            <div class="table-row">
              <span>Output</span>
              <span>"ist Paris. Die Stadt..."</span>
              <span>"Die Hauptstadt von Frankreich ist Paris."</span>
            </div>
          </div>

          <div class="info-callout success">
            <span class="callout-icon">✅</span>
            <p>Fleet Navigator verwendet ausschließlich <strong>Instruct-Modelle</strong> wie Llama 3.2 Instruct, die speziell für Konversationen optimiert sind.</p>
          </div>

          <h2>Warum "Instruct" besser ist für Assistenten</h2>
          <ul class="benefit-list">
            <li><strong>Folgt Anweisungen:</strong> Versteht was du willst</li>
            <li><strong>Strukturierte Antworten:</strong> Formatiert Antworten sinnvoll</li>
            <li><strong>Konversationsfähig:</strong> Merkt sich den Kontext</li>
            <li><strong>Sicherheit:</strong> Vermeidet schädliche Inhalte</li>
          </ul>
        </div>

        <!-- Parameter (8B, 3B, etc.) -->
        <div v-if="currentTopic === 'parameters'" class="topic-article">
          <h1>🔢 Was bedeuten 1B, 3B, 8B?</h1>

          <div class="intro-box">
            <p><strong>B = Billion = Milliarde</strong><br>
            Die Zahl gibt an, wie viele <em>Parameter</em> (lernbare Werte) das Modell hat.</p>
          </div>

          <h2>Größenvergleich</h2>
          <div class="size-comparison">
            <div class="size-card small">
              <div class="size-header">
                <span class="size-icon">🚀</span>
                <span class="size-label">Klein</span>
              </div>
              <div class="size-value">1B - 3B</div>
              <div class="size-example">Llama 3.2 1B, 3B</div>
              <ul>
                <li>~1-2 GB Speicher</li>
                <li>Schnell, auch auf CPU</li>
                <li>Einfache Aufgaben</li>
                <li>Gut für ältere Hardware</li>
              </ul>
            </div>
            <div class="size-card medium">
              <div class="size-header">
                <span class="size-icon">⭐</span>
                <span class="size-label">Standard</span>
              </div>
              <div class="size-value">7B - 8B</div>
              <div class="size-example">Llama 3.1 8B, Mistral 7B</div>
              <ul>
                <li>~4-5 GB Speicher</li>
                <li>Gute Balance</li>
                <li>Die meisten Aufgaben</li>
                <li>Empfohlen für GPU</li>
              </ul>
            </div>
            <div class="size-card large">
              <div class="size-header">
                <span class="size-icon">🏆</span>
                <span class="size-label">Groß</span>
              </div>
              <div class="size-value">13B - 70B</div>
              <div class="size-example">Llama 3.1 70B</div>
              <ul>
                <li>~8-40+ GB Speicher</li>
                <li>Beste Qualität</li>
                <li>Komplexe Aufgaben</li>
                <li>Braucht starke GPU</li>
              </ul>
            </div>
          </div>

          <h2>Mehr Parameter = Besser?</h2>
          <div class="info-callout">
            <span class="callout-icon">⚖️</span>
            <div>
              <p><strong>Nicht immer!</strong> Es kommt auf die Aufgabe an:</p>
              <ul>
                <li><strong>Einfache Fragen:</strong> 3B reicht völlig</li>
                <li><strong>Coding, Analyse:</strong> 8B empfohlen</li>
                <li><strong>Wissenschaft, Kreativ:</strong> 13B+ sinnvoll</li>
              </ul>
            </div>
          </div>

          <h2>Quantisierung (Q4, Q8)</h2>
          <p class="explanation">
            <strong>Q4_K_M, Q8_0</strong> etc. beschreiben die <em>Kompression</em> des Modells:
          </p>
          <ul class="benefit-list">
            <li><strong>Q8:</strong> Höchste Qualität, größere Datei</li>
            <li><strong>Q4:</strong> 50% kleiner, minimal schlechtere Qualität</li>
            <li><strong>K_M:</strong> Optimierte Quantisierung für beste Balance</li>
          </ul>
        </div>

        <!-- Context Size -->
        <div v-if="currentTopic === 'context'" class="topic-article">
          <h1>📚 Was ist Context-Größe?</h1>

          <div class="intro-box">
            <p>Die <strong>Context-Größe</strong> (gemessen in <em>Tokens</em>) bestimmt, wie viel Text die KI gleichzeitig "im Kopf" behalten kann.</p>
          </div>

          <h2>Token-Erklärung</h2>
          <div class="token-example">
            <div class="token-box">
              <span class="token">"Hello"</span>
              <span class="token">" world"</span>
              <span class="token">"!"</span>
              <span class="equals">=</span>
              <span class="count">3 Tokens</span>
            </div>
            <p>Ein Token ist etwa 3/4 eines Wortes. 1000 Tokens ≈ 750 Wörter ≈ 1,5 Seiten</p>
          </div>

          <h2>Context-Größen im Vergleich</h2>
          <div class="context-comparison">
            <div class="context-bar" style="--width: 10%">
              <span class="label">2K</span>
              <span class="desc">~3 Seiten</span>
            </div>
            <div class="context-bar" style="--width: 25%">
              <span class="label">8K</span>
              <span class="desc">~12 Seiten</span>
            </div>
            <div class="context-bar" style="--width: 50%">
              <span class="label">32K</span>
              <span class="desc">~50 Seiten</span>
            </div>
            <div class="context-bar highlight" style="--width: 80%">
              <span class="label">128K</span>
              <span class="desc">~200 Seiten (Fleet Navigator)</span>
            </div>
            <div class="context-bar" style="--width: 100%">
              <span class="label">1M+</span>
              <span class="desc">Ganzes Buch</span>
            </div>
          </div>

          <div class="info-callout success">
            <span class="callout-icon">🚀</span>
            <p>Fleet Navigator unterstützt <strong>128K Context</strong> - das entspricht etwa einem ganzen Roman oder 200 Seiten Dokumente!</p>
          </div>

          <h2>Wofür braucht man großen Context?</h2>
          <div class="feature-grid">
            <div class="feature-card">
              <span class="icon">📖</span>
              <h3>Lange Dokumente</h3>
              <p>Ganze Verträge oder Berichte analysieren</p>
            </div>
            <div class="feature-card">
              <span class="icon">💬</span>
              <h3>Lange Gespräche</h3>
              <p>KI erinnert sich an frühere Nachrichten</p>
            </div>
            <div class="feature-card">
              <span class="icon">🔍</span>
              <h3>Code-Analyse</h3>
              <p>Ganze Codebasen verstehen</p>
            </div>
            <div class="feature-card">
              <span class="icon">📊</span>
              <h3>Zusammenfassungen</h3>
              <p>Lange Texte in Kurzform bringen</p>
            </div>
          </div>
        </div>

        <!-- Local vs Cloud -->
        <div v-if="currentTopic === 'local'" class="topic-article">
          <h1>💻 Lokal vs. ☁️ Cloud (ChatGPT)</h1>

          <div class="intro-box">
            <p>Fleet Navigator läuft <strong>komplett lokal</strong> auf deinem Computer - ganz anders als ChatGPT, das in der Cloud läuft.</p>
          </div>

          <h2>Der große Unterschied</h2>
          <div class="comparison-box large">
            <div class="compare-item cloud">
              <h4>☁️ Cloud-KI (ChatGPT, Claude, Gemini)</h4>
              <div class="flow-diagram">
                <span class="step">Du</span>
                <span class="arrow">→</span>
                <span class="step highlight-bad">Internet</span>
                <span class="arrow">→</span>
                <span class="step">Server (USA)</span>
                <span class="arrow">→</span>
                <span class="step highlight-bad">Internet</span>
                <span class="arrow">→</span>
                <span class="step">Du</span>
              </div>
              <ul>
                <li>❌ Daten verlassen deinen PC</li>
                <li>❌ Abhängig von Internet</li>
                <li>❌ Monatliche Kosten ($20+)</li>
                <li>❌ Anbieter kann Nutzung einschränken</li>
                <li>✅ Sehr leistungsfähig</li>
              </ul>
            </div>
            <div class="compare-item local">
              <h4>💻 Lokale KI (Fleet Navigator)</h4>
              <div class="flow-diagram">
                <span class="step">Du</span>
                <span class="arrow">→</span>
                <span class="step highlight-good">Dein PC</span>
                <span class="arrow">→</span>
                <span class="step">Du</span>
              </div>
              <ul>
                <li>✅ Daten bleiben auf deinem PC</li>
                <li>✅ Funktioniert offline</li>
                <li>✅ Einmalige Einrichtung, keine Kosten</li>
                <li>✅ Volle Kontrolle</li>
                <li>⚡ Gut für die meisten Aufgaben</li>
              </ul>
            </div>
          </div>

          <h2>Wann welche Lösung?</h2>
          <div class="use-cases">
            <div class="use-case local">
              <h4>💻 Fleet Navigator ideal für:</h4>
              <ul>
                <li>Vertrauliche Dokumente (Verträge, Finanzen)</li>
                <li>Firmen-interne Daten</li>
                <li>Offline-Arbeit (Zug, Flugzeug)</li>
                <li>Datenschutz-kritische Bereiche</li>
                <li>Wiederkehrende Aufgaben (unbegrenzt)</li>
              </ul>
            </div>
            <div class="use-case cloud">
              <h4>☁️ Cloud-KI besser für:</h4>
              <ul>
                <li>Maximale Leistung bei komplexen Aufgaben</li>
                <li>Aktuelle Informationen (Internet-Suche)</li>
                <li>Schwache Hardware ohne GPU</li>
                <li>Gelegentliche Nutzung</li>
              </ul>
            </div>
          </div>
        </div>

        <!-- Experts -->
        <div v-if="currentTopic === 'experts'" class="topic-article">
          <h1>👨‍💼 Fokussierte Experten</h1>

          <div class="intro-box">
            <p><strong>Experten</strong> sind spezialisierte KI-Persönlichkeiten, die auf bestimmte Aufgabenbereiche optimiert sind.</p>
          </div>

          <h2>Unterschied zu ChatGPT</h2>
          <div class="comparison-box">
            <div class="compare-item cloud">
              <h4>ChatGPT / Claude</h4>
              <p>Ein Allrounder für alles:</p>
              <ul>
                <li>Generalist, weiß "ein bisschen von allem"</li>
                <li>Keine spezifische Rolle</li>
                <li>Jede Frage startet bei Null</li>
              </ul>
            </div>
            <div class="compare-item local">
              <h4>Fleet Navigator Experten</h4>
              <p>Spezialisierte Fachleute:</p>
              <ul>
                <li>Jeder Experte hat tiefes Fachwissen</li>
                <li>Definierte Rolle und Kommunikationsstil</li>
                <li>Fokussiert auf bestimmte Aufgaben</li>
              </ul>
            </div>
          </div>

          <h2>Beispiel-Experten</h2>
          <div class="expert-grid">
            <div class="expert-card">
              <span class="expert-icon">⚖️</span>
              <h4>Rechtsberater</h4>
              <p>Analysiert Verträge, erklärt juristische Begriffe, weist auf Risiken hin</p>
            </div>
            <div class="expert-card">
              <span class="expert-icon">💰</span>
              <h4>Finanzexperte</h4>
              <p>Prüft Rechnungen, erklärt Steuerkonzepte, analysiert Finanzdokumente</p>
            </div>
            <div class="expert-card">
              <span class="expert-icon">💻</span>
              <h4>IT-Spezialist</h4>
              <p>Hilft bei technischen Fragen, erklärt Software, löst Probleme</p>
            </div>
            <div class="expert-card">
              <span class="expert-icon">📝</span>
              <h4>Texter</h4>
              <p>Schreibt und verbessert Texte, E-Mails, Berichte</p>
            </div>
          </div>

          <div class="info-callout">
            <span class="callout-icon">💡</span>
            <p>Du kannst <strong>eigene Experten erstellen</strong> - definiere Rolle, Fachwissen und Kommunikationsstil nach deinen Bedürfnissen!</p>
          </div>
        </div>

        <!-- Voice -->
        <div v-if="currentTopic === 'voice'" class="topic-article">
          <h1>🎤 Voice: Whisper & Piper</h1>

          <div class="intro-box">
            <p>Fleet Navigator unterstützt <strong>Spracheingabe</strong> (STT) und <strong>Sprachausgabe</strong> (TTS) - komplett lokal!</p>
          </div>

          <h2>Die zwei Komponenten</h2>
          <div class="voice-comparison">
            <div class="voice-card">
              <div class="voice-header whisper">
                <span class="icon">🎙️</span>
                <h3>Whisper (STT)</h3>
              </div>
              <p class="voice-subtitle">Speech-to-Text von OpenAI</p>
              <ul>
                <li><strong>Was:</strong> Wandelt Sprache in Text</li>
                <li><strong>Wie:</strong> Du sprichst, KI tippt</li>
                <li><strong>Qualität:</strong> Sehr hohe Genauigkeit</li>
                <li><strong>Sprachen:</strong> 50+ inkl. Deutsch</li>
              </ul>
              <div class="voice-example">
                <span class="input">🎤 "Schreibe mir eine E-Mail an..."</span>
                <span class="arrow">→</span>
                <span class="output">📝 Text erscheint</span>
              </div>
            </div>
            <div class="voice-card">
              <div class="voice-header piper">
                <span class="icon">🔊</span>
                <h3>Piper (TTS)</h3>
              </div>
              <p class="voice-subtitle">Text-to-Speech, lokal</p>
              <ul>
                <li><strong>Was:</strong> Wandelt Text in Sprache</li>
                <li><strong>Wie:</strong> KI antwortet, du hörst</li>
                <li><strong>Qualität:</strong> Natürlich klingende Stimmen</li>
                <li><strong>Stimmen:</strong> Männlich & weiblich</li>
              </ul>
              <div class="voice-example">
                <span class="input">📝 KI-Antwort</span>
                <span class="arrow">→</span>
                <span class="output">🔊 Wird vorgelesen</span>
              </div>
            </div>
          </div>

          <h2>Warum lokal statt Cloud?</h2>
          <div class="benefit-list">
            <li><strong>Privatsphäre:</strong> Deine Sprache wird nicht hochgeladen</li>
            <li><strong>Offline:</strong> Funktioniert ohne Internet</li>
            <li><strong>Schnell:</strong> Keine Latenz durch Server</li>
            <li><strong>Kostenlos:</strong> Keine API-Kosten pro Minute</li>
          </div>
        </div>

        <!-- Chaining -->
        <div v-if="currentTopic === 'chaining'" class="topic-article">
          <h1>🔗 Vision Chaining</h1>

          <div class="intro-box">
            <p><strong>Vision Chaining</strong> kombiniert zwei Modelle: Ein Vision-Modell "sieht" das Bild, ein Analyse-Modell verarbeitet die Erkennung weiter.</p>
          </div>

          <h2>Wie funktioniert es?</h2>
          <div class="chain-diagram">
            <div class="chain-step">
              <span class="step-icon">📷</span>
              <span class="step-label">Bild/Dokument</span>
            </div>
            <span class="chain-arrow">→</span>
            <div class="chain-step">
              <span class="step-icon">👁️</span>
              <span class="step-label">Vision-Modell</span>
              <span class="step-desc">Erkennt & beschreibt</span>
            </div>
            <span class="chain-arrow">→</span>
            <div class="chain-step">
              <span class="step-icon">🧠</span>
              <span class="step-label">Analyse-Modell</span>
              <span class="step-desc">Verarbeitet & antwortet</span>
            </div>
            <span class="chain-arrow">→</span>
            <div class="chain-step">
              <span class="step-icon">💬</span>
              <span class="step-label">Antwort</span>
            </div>
          </div>

          <h2>Beispiel: Rechnung analysieren</h2>
          <div class="example-flow">
            <div class="flow-step">
              <span class="num">1</span>
              <p>Du lädst eine Rechnung hoch</p>
            </div>
            <div class="flow-step">
              <span class="num">2</span>
              <p><strong>Vision-Modell:</strong> "Das ist eine Rechnung von ACME GmbH, Datum 15.12.2024, Betrag 1.234,56 EUR für Beratungsleistungen..."</p>
            </div>
            <div class="flow-step">
              <span class="num">3</span>
              <p><strong>Analyse-Modell:</strong> Verarbeitet die Beschreibung und gibt dir eine strukturierte Zusammenfassung oder beantwortet deine Fragen</p>
            </div>
          </div>

          <div class="info-callout success">
            <span class="callout-icon">⚡</span>
            <p><strong>Vorteil:</strong> Das Analyse-Modell kann ein kleineres, schnelleres Modell sein - die schwere Arbeit (Bilderkennung) macht das spezialisierte Vision-Modell.</p>
          </div>
        </div>

        <!-- RAG & Websuche -->
        <div v-if="currentTopic === 'rag'" class="topic-article">
          <h1>🔍 RAG & Websuche</h1>

          <div class="intro-box">
            <p><strong>RAG</strong> (Retrieval Augmented Generation) erweitert die KI mit aktuellem Wissen aus dem Web - ohne die Daten dauerhaft zu speichern.</p>
          </div>

          <h2>Was ist RAG?</h2>
          <div class="chain-diagram">
            <div class="chain-step">
              <span class="step-icon">❓</span>
              <span class="step-label">Deine Frage</span>
            </div>
            <span class="chain-arrow">→</span>
            <div class="chain-step">
              <span class="step-icon">🔍</span>
              <span class="step-label">Web-Suche</span>
              <span class="step-desc">Aktuelle Infos holen</span>
            </div>
            <span class="chain-arrow">→</span>
            <div class="chain-step">
              <span class="step-icon">🧠</span>
              <span class="step-label">KI + Kontext</span>
              <span class="step-desc">Antwort generieren</span>
            </div>
            <span class="chain-arrow">→</span>
            <div class="chain-step">
              <span class="step-icon">💬</span>
              <span class="step-label">Antwort</span>
            </div>
          </div>

          <p class="explanation">
            Die KI sucht zuerst im Web nach relevanten Informationen und nutzt diese als <em>Kontext</em> für die Antwort.
            So kann sie auch über aktuelle Ereignisse sprechen, die nach ihrem Training passiert sind.
          </p>

          <h2>Die Quellen-Einstellung</h2>
          <div class="comparison-box">
            <div class="compare-item local">
              <h4>✅ Quellen-Links anzeigen</h4>
              <p class="setting-desc">Die Standardeinstellung</p>
              <ul>
                <li>Antwort enthält <code>[1]</code>, <code>[2]</code> Referenzen</li>
                <li>Am Ende werden Quellen-Links angezeigt</li>
                <li>Du kannst nachprüfen woher die Info stammt</li>
                <li>Transparenz und Nachvollziehbarkeit</li>
              </ul>
              <div class="example-box">
                <strong>Beispiel-Antwort:</strong>
                <p>"Laut aktuellen Berichten [1] ist der DAX heute um 2% gestiegen..."</p>
                <p class="sources">Quellen: [1] finanzen.net/dax...</p>
              </div>
            </div>
            <div class="compare-item cloud">
              <h4>🔇 Nur RAG (ohne Links)</h4>
              <p class="setting-desc">Für sauberen Text</p>
              <ul>
                <li>Web-Inhalte werden genutzt</li>
                <li>Aber KEINE Links in der Antwort</li>
                <li>Keine <code>[1]</code>, <code>[2]</code> Referenzen</li>
                <li>Flüssiger, natürlicher Text</li>
              </ul>
              <div class="example-box">
                <strong>Beispiel-Antwort:</strong>
                <p>"Der DAX ist heute um 2% gestiegen, angetrieben durch positive Quartalszahlen..."</p>
                <p class="no-sources">(Keine Quellenangaben)</p>
              </div>
            </div>
          </div>

          <h2>Wann welche Einstellung?</h2>
          <div class="use-cases">
            <div class="use-case local">
              <h4>✅ Mit Quellen-Links</h4>
              <ul>
                <li>Recherche & Faktenfindung</li>
                <li>Wenn du Quellen überprüfen willst</li>
                <li>Für Berichte mit Quellenangaben</li>
                <li>Bei kritischen Informationen</li>
              </ul>
            </div>
            <div class="use-case cloud">
              <h4>🔇 Ohne Quellen-Links</h4>
              <ul>
                <li>Texte für Präsentationen</li>
                <li>Natürlich klingende Antworten</li>
                <li>Wenn du die Info nur brauchst, nicht die Quelle</li>
                <li>Für saubere Exports</li>
              </ul>
            </div>
          </div>

          <div class="info-callout">
            <span class="callout-icon">⚙️</span>
            <div>
              <p><strong>Einstellung pro Experte:</strong></p>
              <p>Du findest die Option "Quellen-Links anzeigen" in den Experten-Einstellungen unter <em>RAG & Websuche</em>. Jeder Experte kann eigene Einstellungen haben!</p>
            </div>
          </div>

          <h2>Wichtig: Keine erfundenen Quellen!</h2>
          <div class="info-callout success">
            <span class="callout-icon">🛡️</span>
            <div>
              <p>Fleet Navigator verhindert, dass die KI Quellen <strong>erfindet</strong>.</p>
              <ul>
                <li>Quellen werden nur angezeigt wenn tatsächlich gesucht wurde</li>
                <li>Bei einfachen Fragen ("Wer bist du?") → Keine Quellen</li>
                <li>Keine halluzinierten URLs aus dem Training</li>
              </ul>
            </div>
          </div>
        </div>
      </article>
    </main>
  </div>
</template>

<script setup>
import { ref, onMounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'

const route = useRoute()
const router = useRouter()

const topics = [
  { id: 'vision', name: 'Vision-Modelle', icon: '👁️' },
  { id: 'instruct', name: 'Instruct-Modelle', icon: '🎯' },
  { id: 'parameters', name: 'Was bedeutet 8B?', icon: '🔢' },
  { id: 'context', name: 'Context-Größe', icon: '📚' },
  { id: 'local', name: 'Lokal vs. Cloud', icon: '💻' },
  { id: 'experts', name: 'Fokussierte Experten', icon: '👨‍💼' },
  { id: 'voice', name: 'Voice (Whisper/Piper)', icon: '🎤' },
  { id: 'chaining', name: 'Vision Chaining', icon: '🔗' },
  { id: 'rag', name: 'RAG & Websuche', icon: '🔍' }
]

const currentTopic = ref('vision')

onMounted(() => {
  // Check for topic in query params
  if (route.query.topic && topics.find(t => t.id === route.query.topic)) {
    currentTopic.value = route.query.topic
  }
})

watch(() => route.query.topic, (newTopic) => {
  if (newTopic && topics.find(t => t.id === newTopic)) {
    currentTopic.value = newTopic
  }
})

function goBack() {
  if (window.history.length > 1) {
    router.back()
  } else {
    window.close()
  }
}

function closeHelp() {
  window.close()
}
</script>

<style scoped>
.help-page {
  min-height: 100vh;
  background: linear-gradient(135deg, #1a1a2e 0%, #16213e 100%);
  color: #e2e8f0;
}

.help-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 20px 30px;
  background: rgba(0, 0, 0, 0.3);
  border-bottom: 1px solid #333;
}

.help-logo {
  display: flex;
  align-items: center;
  gap: 10px;
  cursor: pointer;
}

.logo-icon {
  font-size: 28px;
}

.logo-text {
  font-size: 20px;
  font-weight: 600;
  color: #fff;
}

.close-btn {
  background: #333;
  border: none;
  color: #fff;
  width: 36px;
  height: 36px;
  border-radius: 50%;
  font-size: 18px;
  cursor: pointer;
  transition: background 0.2s;
}

.close-btn:hover {
  background: #444;
}

.help-content {
  display: flex;
  max-width: 1400px;
  margin: 0 auto;
  padding: 30px;
  gap: 30px;
}

/* Topic Navigation */
.topic-nav {
  width: 260px;
  flex-shrink: 0;
  display: flex;
  flex-direction: column;
  gap: 8px;
  position: sticky;
  top: 30px;
  height: fit-content;
}

.topic-btn {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 14px 16px;
  background: #2d2d44;
  border: 2px solid transparent;
  border-radius: 12px;
  cursor: pointer;
  transition: all 0.2s;
  text-align: left;
}

.topic-btn:hover {
  border-color: #4f46e5;
  background: #3d3d5c;
}

.topic-btn.active {
  border-color: #4f46e5;
  background: #4f46e5;
}

.topic-icon {
  font-size: 22px;
}

.topic-name {
  color: #fff;
  font-size: 14px;
  font-weight: 500;
}

/* Topic Content */
.topic-content {
  flex: 1;
  max-width: 900px;
}

.topic-article {
  animation: fadeIn 0.3s ease;
}

@keyframes fadeIn {
  from { opacity: 0; transform: translateY(10px); }
  to { opacity: 1; transform: translateY(0); }
}

.topic-article h1 {
  font-size: 32px;
  color: #fff;
  margin-bottom: 20px;
}

.topic-article h2 {
  font-size: 22px;
  color: #fff;
  margin: 30px 0 15px;
}

.intro-box {
  background: linear-gradient(135deg, #4f46e5 0%, #7c3aed 100%);
  border-radius: 16px;
  padding: 24px;
  margin-bottom: 30px;
}

.intro-box p {
  font-size: 18px;
  line-height: 1.6;
  margin: 0;
}

.intro-box strong {
  color: #fff;
}

.intro-box em {
  color: #c4b5fd;
  font-style: normal;
  font-weight: 500;
}

/* Feature Grid */
.feature-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 16px;
  margin: 20px 0;
}

.feature-card {
  background: #2d2d44;
  border-radius: 12px;
  padding: 20px;
}

.feature-card .icon {
  font-size: 32px;
  display: block;
  margin-bottom: 10px;
}

.feature-card h3 {
  color: #fff;
  font-size: 16px;
  margin-bottom: 8px;
}

.feature-card p {
  color: #94a3b8;
  font-size: 14px;
  margin: 0;
  line-height: 1.5;
}

/* Comparison Box */
.comparison-box {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 20px;
  margin: 20px 0;
}

.comparison-box.large {
  gap: 30px;
}

.compare-item {
  background: #2d2d44;
  border-radius: 12px;
  padding: 20px;
  border-left: 4px solid #666;
}

.compare-item.cloud {
  border-left-color: #f59e0b;
}

.compare-item.local {
  border-left-color: #10b981;
}

.compare-item h4 {
  color: #fff;
  font-size: 16px;
  margin-bottom: 12px;
}

.compare-item ul {
  margin: 0;
  padding-left: 20px;
}

.compare-item li {
  color: #94a3b8;
  margin-bottom: 6px;
  font-size: 14px;
}

/* Flow Diagram */
.flow-diagram {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
  margin: 15px 0;
  padding: 15px;
  background: rgba(0, 0, 0, 0.2);
  border-radius: 8px;
}

.flow-diagram .step {
  background: #4f46e5;
  color: #fff;
  padding: 8px 14px;
  border-radius: 6px;
  font-size: 13px;
}

.flow-diagram .arrow {
  color: #666;
  font-size: 18px;
}

.flow-diagram .highlight-bad {
  background: #dc2626;
}

.flow-diagram .highlight-good {
  background: #10b981;
}

/* Info Callout */
.info-callout {
  background: #1e3a5f;
  border-left: 4px solid #3b82f6;
  border-radius: 8px;
  padding: 20px;
  display: flex;
  gap: 15px;
  margin: 20px 0;
}

.info-callout.success {
  background: #064e3b;
  border-left-color: #10b981;
}

.callout-icon {
  font-size: 28px;
  flex-shrink: 0;
}

.info-callout p {
  margin: 0;
  line-height: 1.6;
}

.info-callout ul {
  margin: 10px 0 0;
  padding-left: 20px;
}

/* Comparison Table */
.comparison-table {
  background: #2d2d44;
  border-radius: 12px;
  overflow: hidden;
  margin: 20px 0;
}

.table-header {
  display: grid;
  grid-template-columns: 150px 1fr 1fr;
  background: #1e1e2e;
  padding: 14px 20px;
  font-weight: 600;
  color: #94a3b8;
  font-size: 14px;
}

.table-row {
  display: grid;
  grid-template-columns: 150px 1fr 1fr;
  padding: 14px 20px;
  border-bottom: 1px solid #3d3d5c;
  font-size: 14px;
}

.table-row:last-child {
  border-bottom: none;
}

.table-row span:first-child {
  color: #94a3b8;
}

/* Benefit List */
.benefit-list {
  margin: 15px 0;
  padding-left: 0;
  list-style: none;
}

.benefit-list li {
  padding: 10px 0 10px 30px;
  position: relative;
  border-bottom: 1px solid #3d3d5c;
}

.benefit-list li:before {
  content: "✓";
  position: absolute;
  left: 0;
  color: #10b981;
  font-weight: bold;
}

/* Size Cards */
.size-comparison {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 20px;
  margin: 20px 0;
}

.size-card {
  background: #2d2d44;
  border-radius: 12px;
  padding: 20px;
  border-top: 4px solid #666;
}

.size-card.small { border-top-color: #10b981; }
.size-card.medium { border-top-color: #3b82f6; }
.size-card.large { border-top-color: #f59e0b; }

.size-header {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 10px;
}

.size-icon { font-size: 24px; }
.size-label { color: #94a3b8; font-size: 14px; }
.size-value { font-size: 28px; font-weight: 700; color: #fff; }
.size-example { color: #94a3b8; font-size: 13px; margin-bottom: 15px; }

.size-card ul {
  margin: 0;
  padding-left: 20px;
}

.size-card li {
  font-size: 13px;
  color: #94a3b8;
  margin-bottom: 6px;
}

/* Token Example */
.token-example {
  background: #2d2d44;
  border-radius: 12px;
  padding: 20px;
  margin: 20px 0;
}

.token-box {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
  margin-bottom: 12px;
}

.token {
  background: #4f46e5;
  color: #fff;
  padding: 8px 12px;
  border-radius: 6px;
  font-family: monospace;
}

.equals { color: #666; font-size: 20px; }
.count { color: #10b981; font-weight: 600; }

.token-example > p {
  color: #94a3b8;
  margin: 0;
  font-size: 14px;
}

/* Context Bars */
.context-comparison {
  display: flex;
  flex-direction: column;
  gap: 12px;
  margin: 20px 0;
}

.context-bar {
  display: flex;
  align-items: center;
  gap: 15px;
  background: #2d2d44;
  padding: 12px 16px;
  border-radius: 8px;
  width: calc(var(--width, 50%));
  min-width: 200px;
}

.context-bar.highlight {
  background: #4f46e5;
}

.context-bar .label {
  font-weight: 700;
  color: #fff;
  min-width: 50px;
}

.context-bar .desc {
  color: #94a3b8;
  font-size: 14px;
}

.context-bar.highlight .desc {
  color: #c4b5fd;
}

/* Expert Grid */
.expert-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 16px;
  margin: 20px 0;
}

.expert-card {
  background: #2d2d44;
  border-radius: 12px;
  padding: 20px;
  text-align: center;
}

.expert-icon {
  font-size: 40px;
  display: block;
  margin-bottom: 12px;
}

.expert-card h4 {
  color: #fff;
  margin-bottom: 8px;
}

.expert-card p {
  color: #94a3b8;
  font-size: 14px;
  margin: 0;
  line-height: 1.5;
}

/* Voice Cards */
.voice-comparison {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 20px;
  margin: 20px 0;
}

.voice-card {
  background: #2d2d44;
  border-radius: 12px;
  padding: 24px;
}

.voice-header {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 8px;
}

.voice-header .icon { font-size: 28px; }
.voice-header h3 { color: #fff; margin: 0; }

.voice-header.whisper { color: #f59e0b; }
.voice-header.piper { color: #10b981; }

.voice-subtitle {
  color: #94a3b8;
  font-size: 13px;
  margin-bottom: 15px;
}

.voice-card ul {
  margin: 0 0 15px;
  padding-left: 20px;
}

.voice-card li {
  font-size: 14px;
  color: #94a3b8;
  margin-bottom: 6px;
}

.voice-example {
  display: flex;
  align-items: center;
  gap: 10px;
  background: rgba(0, 0, 0, 0.2);
  padding: 12px;
  border-radius: 8px;
  font-size: 13px;
}

.voice-example .arrow { color: #666; }

/* Chain Diagram */
.chain-diagram {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 15px;
  flex-wrap: wrap;
  background: #2d2d44;
  padding: 30px;
  border-radius: 12px;
  margin: 20px 0;
}

.chain-step {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  background: #1e1e2e;
  padding: 20px;
  border-radius: 12px;
  min-width: 120px;
}

.chain-step .step-icon { font-size: 32px; }
.chain-step .step-label { color: #fff; font-weight: 600; }
.chain-step .step-desc { color: #94a3b8; font-size: 12px; }

.chain-arrow {
  font-size: 24px;
  color: #4f46e5;
}

/* Example Flow */
.example-flow {
  background: #2d2d44;
  border-radius: 12px;
  padding: 24px;
  margin: 20px 0;
}

.flow-step {
  display: flex;
  gap: 15px;
  padding: 15px 0;
  border-bottom: 1px solid #3d3d5c;
}

.flow-step:last-child { border-bottom: none; }

.flow-step .num {
  width: 28px;
  height: 28px;
  background: #4f46e5;
  color: #fff;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: 600;
  flex-shrink: 0;
}

.flow-step p {
  margin: 0;
  color: #94a3b8;
  line-height: 1.6;
}

.flow-step strong { color: #fff; }

/* Use Cases */
.use-cases {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 20px;
  margin: 20px 0;
}

.use-case {
  background: #2d2d44;
  border-radius: 12px;
  padding: 20px;
  border-top: 4px solid #666;
}

.use-case.local { border-top-color: #10b981; }
.use-case.cloud { border-top-color: #f59e0b; }

.use-case h4 {
  color: #fff;
  margin-bottom: 12px;
}

.use-case ul {
  margin: 0;
  padding-left: 20px;
}

.use-case li {
  color: #94a3b8;
  font-size: 14px;
  margin-bottom: 6px;
}

.explanation {
  color: #94a3b8;
  line-height: 1.6;
}

/* RAG-specific styles */
.setting-desc {
  color: #94a3b8;
  font-size: 13px;
  margin-bottom: 12px;
}

.example-box {
  background: rgba(0, 0, 0, 0.2);
  border-radius: 8px;
  padding: 15px;
  margin-top: 15px;
}

.example-box strong {
  color: #fff;
  font-size: 13px;
  display: block;
  margin-bottom: 8px;
}

.example-box p {
  margin: 0;
  font-size: 14px;
  color: #c4b5fd;
  font-style: italic;
}

.example-box .sources {
  margin-top: 10px;
  color: #10b981;
  font-size: 12px;
  font-style: normal;
}

.example-box .no-sources {
  margin-top: 10px;
  color: #94a3b8;
  font-size: 12px;
  font-style: normal;
}

.compare-item code {
  background: rgba(79, 70, 229, 0.3);
  color: #c4b5fd;
  padding: 2px 6px;
  border-radius: 4px;
  font-size: 13px;
}

/* Responsive */
@media (max-width: 900px) {
  .help-content {
    flex-direction: column;
  }

  .topic-nav {
    width: 100%;
    flex-direction: row;
    flex-wrap: wrap;
    position: static;
  }

  .topic-btn {
    flex: 1;
    min-width: 140px;
    justify-content: center;
  }

  .feature-grid,
  .comparison-box,
  .size-comparison,
  .voice-comparison,
  .expert-grid,
  .use-cases {
    grid-template-columns: 1fr;
  }

  .table-header,
  .table-row {
    grid-template-columns: 1fr;
    gap: 8px;
  }
}
</style>
