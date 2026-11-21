# SC Cockpit

SC Cockpit ist ein Touch-Dashboard für Star Citizen. Das Programm läuft auf dem Windows-PC, auf dem das Spiel läuft. Tablet, Handy oder ein zweiter Rechner öffnen nur eine Adresse im Browser. Ein Tipp sendet einen deiner Keybinds an das Spiel.

Es gibt keinen Account, keine Cloud und keine zusätzlichen Python-Pakete. Das Panel liest keine Schiffswerte. Schildbalken, Quantumsprung und Ähnliches bleiben im Spiel.

Ein neues Dashboard ist eine JSON-Datei. Das Format steht verbindlich in [docs/DASHBOARD.md](docs/DASHBOARD.md), das Schema in [docs/dashboard.schema.json](docs/dashboard.schema.json), ein Minimalbeispiel in [docs/beispiel.json](docs/beispiel.json).

## Inhalt

- [Voraussetzungen](#voraussetzungen)
- [Installieren](#installieren)
- [Starten](#starten)
- [Bedienen](#bedienen)
- [Eigenes Dashboard](#eigenes-dashboard)
- [Deinstallieren](#deinstallieren)
- [Dateien](#dateien)
- [Probleme](#probleme)
- [Mitentwickeln](#mitentwickeln)
- [Grenzen](#grenzen)

## Voraussetzungen

- PC mit Windows 10 oder 11
- Python 3, beim Installer **Add python.exe to PATH** ankreuzen, oder den SC-Cockpit-Installer den `winget`-Weg machen lassen
- Spiel und Bedien-Gerät im selben privaten WLAN
- Star Citizen im Vordergrund, sonst gehen die Tasten an das falsche Fenster

Das Bedien-Gerät braucht nur einen aktuellen Browser. Eine App und ein Account sind nicht nötig. Windows auf dem Tablet ist keine Voraussetzung.

## Sofort Live Testen & Installieren (Kein Neu-Installieren nötig!)

Du kannst **SC Cockpit** jederzeit sofort live starten, ohne einen Installations-Assistenten durchklicken zu müssen:

1. **Im geklonten Git-Ordner (nach `git pull`)**:
   - Einfach **`SC-Cockpit-Setup.exe`** oder **`start.bat`** doppelklicken!
   - Da `server.py` und `index.html` direkt daneben liegen, überspringt die `.exe` den Installer komplett und **öffnet sofort das fertige GameGlass Live-Cockpit** direkt aus deinem Ordner heraus.
2. **Wenn SC Cockpit bereits auf dem PC installiert ist**:
   - Auch wenn du eine neue `SC-Cockpit-Setup.exe` herunterlädst und doppelklickst, erkennt sie deine bestehende Installation, aktualisiert im Hintergrund automatisch die neuen Dateien & Schiffe (deine eigenen Schiffe im Ordner `user` bleiben unberührt!) und **startet sofort im Live-Cockpit**.
3. **Erstinstallation auf einem neuen PC**:
   - Nur wenn noch gar nichts installiert ist und die `.exe` einzeln (z. B. aus `Downloads`) gestartet wird, zeigt sie den 1-Klick-Installer (oder lässt sich dort unten links über **PORTABEL TESTEN** ebenfalls ohne Installation direkt starten).
   - Möchtest du den Installer-Bildschirm einmal gezielt erzwingen: `SC-Cockpit-Setup.exe --install`.

## Starten & GameGlass Hangar & Shard Hub (In-App Live Cockpit & 100+ Schiffe)

**SC Cockpit** auf dem Desktop oder im Startmenü starten (oder im Repo direkt `start.bat` bzw. `SC-Cockpit-Setup.exe` doppelklicken). Es öffnet sich der **SC Cockpit // GameGlass Hangar & Shard Hub** (mit **Maximieren-Button `□` / `❐`**, Doppelklick auf die Titelleiste zum Maximieren und frei skalierbaren Fensterrändern) mit vier Bereichen in der linken Seitenleiste:

1. **► LIVE COCKPIT (Dashboards & Schiffe direkt in der App nutzen!)**:
   - **Fenster Maximieren & Dynamisches Grid**: Oben rechts neben Minimieren (`—`) und Schließen (`✕`) sitzt der **Maximieren-Button (`□` / `❐`)**. Im Vollbild-/Maximiert-Modus passt sich das Kachel-Raster automatisch auf bis zu 6 Spalten an.
   - **1-Klick Hangar-Schnellleiste (`★ MEIN HANGAR`)**: Oben unter der Titelleiste erscheinen übersichtlich **nur deine favorisierten Hangar-Schiffe (`★`)** für den 1-Klick-Wechsel im Flug.
   - **Schiffs-Silhouette & Subsystem-Tabs**: Jedes Schiff zeigt seine eigene **Schiffs-Blaupause (Silhouette)** sowie seine spezialisierten Unterpunkte (z. B. *Mining & Laser*, *Salvage & Scraper*, *Fracht & Hangar*, *MedBay & Rettung*, *Scan & Expedition*, *EMP & QED*, *Dropship & Türme*, *Kampf & Waffen*, *Power Management*, *Quantum & Nav*, *Flug & Systeme*).
   - **📌 OVERLAY (Immer im Vordergrund)**: Hält das In-App Live-Cockpit auf Wunsch im Vordergrund über Star Citizen (Borderless Window).
   - **Direktes Senden an Star Citizen**: Ein Klick auf eine Holo-Kachel im Programm fokussiert automatisch das laufende Star-Citizen-Fenster und sendet den Keybind direkt ins Spiel.
2. **★ HANGAR & SCHIFFE (Über 100 Star Citizen Schiffe suchen & favorisieren)**:
   - **Personalisierte Dashboards für jedes Schiff im Spiel**: Über 100 fliegbare Star Citizen Schiffe aller Hersteller (**AEGIS**, **ANVIL**, **RSI**, **DRAKE**, **MISC**, **CRUSADER**, **ORIGIN**, **ARGO**, **CNOU**, **MIRAI**, **GATAC**, **BANU**, **ESPERIA**, **AOPOA**) sind mit ihren echten Rollen, Subsystem-Seiten und einer eigenen **Schiffs-Blaupause (abgebildetes Schiff im Hintergrund)** integriert.
   - **Echtzeit-Suche & Rollen-Filter**: Suche direkt per Tastatur nach Schiffsname (z. B. *Vulture*, *Prospector*, *Carrack*, *Gladius*, *Polaris*, *F8C*, *Cutlass*, *Hercules*), Hersteller oder Rolle (*★ Mein Hangar*, *Alle Schiffe*, *Kampf*, *Mining*, *Salvage*, *Fracht*, *Exploration*).
   - **Favorisieren (`★ HANGAR` / `☆ MERKEN`)**: Markiere deine Schiffe mit einem Klick als Favorit — sie landen sofort in deinem **Hangar**, wo du sie mit **`► ÖFFNEN`** direkt ins Cockpit lädst oder mit **`⚙ ANPASSEN`** im Baukasten individualisierst.
   - **1-Klick Unterpunkt zum aktiven Schiff hinzufügen**: Über die Leiste **`+ MINING`**, **`+ SALVAGE`**, **`+ KAMPF`**, **`+ FRACHT`**, **`+ POWER`** erweiterst du jedes Schiff sofort um zusätzliche Subsystem-Seiten.
3. **TABLET / SERVER**:
   - **Server an- und ausschalten**: Großer Button **SERVER STARTEN** / **SERVER STOPPEN** für externe Tablets & Smartphones im WLAN.
   - Zeigt die WLAN-Adresse (z. B. `http://192.168.1.180:8765/`) und einen QR-Code zum Scannen, den **ALS ADMIN STARTEN**-Button und das Live-Tastenprotokoll.
4. **EINSTELLUNGEN**:
   - Server-Port (`8765`) per `—` / `+` anpassen, Ordner öffnen, Verknüpfungen reparieren und **SC Cockpit direkt in der GUI deinstallieren**.

Wenn Windows beim ersten Serverstart nach der Firewall fragt: Zugriff im privaten Netzwerk erlauben.

## Bedienen: Live Cockpit, Hangar & Schiffs-Suche im Browser / Tablet

Auch auf dem Tablet oder im Browser ist die Oberfläche klar in **3 Modi** oben in der Kopfzeile gegliedert:
1. **`► LIVE COCKPIT`**:
   - Zeigt das aktive Schiff mit seinem **eigenen Schiffs-Hintergrundbild (abgebildete Schiffs-Silhouette & Hersteller-HUD)** hinter den abgeschrägten GameGlass-Kacheln.
   - Oben in der Leiste **`★ MEIN HANGAR`** wechselst du mit 1 Tipp zwischen deinen favorisierten Hangar-Schiffen.
   - Links in der Seitenleiste siehst du die Mini-Vorschau des aktuellen Schiffs sowie alle Unterpunkte (*Mining & Laser*, *Salvage & Scraper*, *Power Management* usw.).
2. **`★ HANGAR & SCHIFFE`**:
   - Öffnet den visuellen **Star Citizen Hangar** mit Suchfeld und Rollen-Filtern (*★ Mein Hangar*, *Alle Schiffe (100+)*, *Kampf & Fighter*, *Mining*, *Salvage*, *Fracht & Hauler*, *Exploration & Scan*, *MedBay & Rescue*, *EMP & QED*).
   - Jede Schiffskarte zeigt das **abgebildete Schiff als technische Blaupause**, Hersteller-Badge, alle enthaltenen Subsystem-Tags sowie die Buttons **`★ IM HANGAR`**, **`► IM COCKPIT ÖFFNEN`** und **`⚙ ANPASSEN`**.
3. **`⚙ BAUKASTEN`**:
   - Fertige Star-Citizen-Presets per 1-Klick einfügen, den Unterpunkt **Power Management** ergänzen, eigene Hintergrundbilder hochladen und mit **SC-Tasten lesen** deine lokale `actionmaps.xml` synchronisieren.

## Deinstallieren

Drei Möglichkeiten:
1. **Direkt in der grafischen Anwendung**: Im **SC Cockpit Control Center** links auf **Einstellungen** klicken und unten auf **Deinstallieren...** klicken.
2. Über Windows **Installierte Apps** oder im Startmenü unter **SC Cockpit → Deinstallieren**.
3. Über `uninstall.bat` (oder `SC-Cockpit-Setup.exe --uninstall`).

Dabei kann per Checkbox gewählt werden, ob nur das Programm und die Verknüpfungen entfernt werden (Standard: eigene Dashboards unter `%LOCALAPPDATA%\SC Cockpit\user` bleiben erhalten) oder ob alles inklusive eigener Dashboards gelöscht werden soll (`-Purge`):

```bat
SC-Cockpit-Setup.exe --uninstall -Purge
```

## Dateien

```text
SC-Cockpit-Setup.exe      Eigenständige Windows-App (Live-Cockpit, Hangar mit 100+ Schiffen & Setup)
installer/                Quellcode (Go/Win32) & Build-Skript (build_installer.py) für die .exe
server.py                 Server, nur Python-Standardbibliothek
sc_ships.py               Star Citizen Schiffs-Datenbank (100+ Schiffe, Rollen-Subsysteme & SVG-Blaupausen)
index.html                GameGlass Web-Oberfläche (Live Cockpit, Hangar & Schiffs-Suche, Baukasten)
start.bat                 Start ohne Installation (inkl. Auto-Git-Pull)
install.bat               Startet SC-Cockpit-Setup.exe (inkl. Auto-Git-Pull)
uninstall.bat             Startet den Deinstaller (Fallback auf uninstall.ps1)
dashboards/*.json         mitgelieferte Basis-Dashboards
dashboards/local/*.json   eigene/angepasste Schiffs-Dashboards & state.json (Favoriten)
docs/DASHBOARD.md         Format, Tasten, API, Regeln für eine KI
docs/dashboard.schema.json
docs/beispiel.json
config.local.json         optional, Port, Token, Halte-Timeout
```

Nach der Installation zeigt das Serverfenster den Ordner der eigenen Dashboards. `config.local.json` liegt dann unter `%LOCALAPPDATA%\SC Cockpit\user\`, nicht neben dem Programm. Vorlage ist `config.local.json.example`.

| Feld | Standard | Bedeutung |
| --- | --- | --- |
| `port` | `8765` | Port der Adresse. |
| `token` | leer | Leer lässt jedes Gerät im WLAN Tasten senden. Ein Geheimnis verlangt ihn einmalig im Browser. |
| `hold_timeout_s` | `4` | Danach wird eine gehaltene Taste losgelassen, wenn das Gerät nichts mehr schickt. |

Den Port ändern, dann Server neu starten und die neue Adresse verwenden. Ein reines Dashboard ist ohne Neustart da.

## Probleme

| Symptom | Was tun |
| --- | --- |
| Das andere Gerät lädt die Seite nicht | Dieselbe private WLAN-Verbindung, nicht Gastnetz. Die Adresse aus dem Fenster nehmen, nicht `localhost`. Firewall-Frage für das private Netzwerk mit Ja beantworten. |
| Seite lädt, Tasten tun nichts | Star Citizen anklicken, damit es vorn ist. Läuft das Spiel als Administrator, SC Cockpit auch so starten. Im Serverfenster muss die Taste geloggt werden, sonst kommt der Klick nicht an. |
| Falsche Taste im Spiel | `keys` an die eigene Belegung anpassen. `y` ist die Taste Y deines Layouts. Modifier heißen `lalt`, `ralt`, `lctrl`, `lshift`, nicht „Alt“. |
| Dashboard fehlt in der Liste | Dateiname `<id>.json`, `id` nur Kleinbuchstaben, Ziffern, Bindestrich. Kein Kommafehler im JSON. Richtiger Ordner, siehe oben. Seite neu laden. |
| „unbekannte Taste“ | Nur Namen aus [docs/DASHBOARD.md](docs/DASHBOARD.md). Höchstens vier pro Button. |
| Taste bleibt hängen | **Loslassen** drücken oder die Seite verlassen. Der Server lässt zusätzlich nach ein paar Sekunden los. |
| Python wird nicht gefunden | Python neu installieren, PATH ankreuzen, neues Fenster öffnen. `py -3 server.py` im Ordner testen. |
| Port belegt | Anderes Programm beenden oder `port` in `config.local.json` ändern. |

Im Serverfenster steht bei einem Nicht-Windows-Rechner **Trockenlauf**. Dann werden Tasten nur angezeigt. Das ist für die Entwicklung, nicht zum Spielen.

## Mitentwickeln

```bat
git clone https://github.com/MOON-132/sc-cockpit.git
cd sc-cockpit
start.bat
```

```bat
py -3 server.py --self-test
```

Der Test prüft die mitgelieferten Dashboards, das Beispiel in `docs/beispiel.json` und die HTTP-Schnittstelle. Er sendet auf einem anderen System als Windows keine echten Tasten.

`main` ist die Fassung für den Spiel-PC. Layout-Experimente auf einen eigenen Branch. Ein `git pull` auf dem Spiel-PC und ein Neuladen der Seite übernehmen neue mitgelieferte Dashboards. Die installierte Kopie aktualisiert sich nicht von selbst: nach einem Pull dort erneut `SC-Cockpit-Setup.exe` (oder `install.bat`) ausführen. Der Ordner `user` bleibt dabei.

Wenn Änderungen an `server.py`, `index.html`, den mitgelieferten Dashboards oder dem Installer-Code unter `installer/` vorgenommen wurden, wird die `SC-Cockpit-Setup.exe` (benötigt Python 3 und Go) neu gebaut mit:

```bat
python installer/build_installer.py
```

Nicht committen: `config.local.json`, `dashboards/local/*.json`, `state.json`.

## Grenzen

- Keine Anzeige von Schild, Power, Ziel oder Quantumsstatus.
- Kein Joystick-Ersatz. Fliegen bleibt an Maus, Tastatur oder HOTAS.
- Kein Makro und kein automatischer Quantumsprung.
- Easy Anti-Cheat bewertet simulierte Tasten selbst. Einzelne Keybinds sind dasselbe Verfahren wie bei anderen Touch-Panels. Die Regeln von CIG gelten. Daraus keine Kampfscripte bauen.
- Kein Passwort außer dem optionalen Token, kein HTTPS. Nur im eigenen Netz verwenden.
- Selbstzerstörung und Schleudersitz sind absichtlich nicht in den mitgelieferten Dashboards.

## Stand

`0.2.0`. Mehrere Dashboards, Editor im Browser, eigenständiger Windows-GUI-Installer & Uninstaller (`SC-Cockpit-Setup.exe`) im Star-Citizen-HUD-Design. Das Dashboard-Format ist in `docs/DASHBOARD.md` festgelegt.

