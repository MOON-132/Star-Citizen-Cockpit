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

## Starten & GameGlass Shard Hub (In-App Live Cockpit & Flotten-Manager)

**SC Cockpit** auf dem Desktop oder im Startmenü starten (oder im Repo direkt `start.bat` doppelklicken). Es öffnet sich der **SC Cockpit // GameGlass Shard Hub** im holografischen GameGlass-MFD-Design (abgeschrägte Holo-Shards, Electric Cyan & Quantum Amber) mit vier Bereichen in der linken Seitenleiste:

1. **LIVE COCKPIT (Dashboards & Schiffe direkt in der App nutzen!)**:
   - **1-Klick Schiffsflotten-Leiste (Quick Ship Switcher)**: Oben unter der Titelleiste siehst du alle deine Schiffe nebeneinander (z. B. **Aegis Gladius**, **MISC Prospector**, **Drake Vulture**, **Crusader C2** sowie deine eigenen Schiffe) und wechselst mit **einem Klick** das aktive Schiff.
   - **Unterpunkte / Subsystem-Tabs je Schiff**: Darunter liegen die spezialisierten Unterpunkte des gewählten Schiffs (z. B. *Mining & Laser*, *Salvage & Scraper*, *Fracht & Hangar*, *Flug & Quantum*, *Energie & Schilde*, *Kampf*).
   - **📌 OVERLAY (Immer im Vordergrund)**: Hält das In-App Live-Cockpit auf Wunsch im Vordergrund über Star Citizen (im Borderless-Window-Modus).
   - **Direktes Senden an Star Citizen**: Ein Klick auf eine Holo-Kachel im Programm fokussiert automatisch das laufende Star-Citizen-Fenster und sendet den Keybind direkt ins Spiel.
   - **Waffen SAFE / ARMED**: Gesicherte Tasten (Waffen, Laser, Abwerfen) werden über den Schalter **WAFFEN SAFE / ARMED** oben rechts scharfgeschaltet.
2. **SHARDS & DECKS (Flotte, Schiffe & Unterpunkte verwalten)**:
   - Übersicht deiner gesamten Schiffsflotte inklusive Hersteller-Badge (`[AEGIS]`, `[MISC]`, `[DRAKE]`, `[CRUSADER]`), Rolle und aller vorhandenen Unterpunkte.
   - **1-Klick Unterpunkt zum aktiven Schiff hinzufügen**: Über die Leiste **UNTERPUNKT ZUM AKTIVEN SCHIFF (+)** kannst du jedem Schiff sofort per Klick auf **`+ MINING`**, **`+ SALVAGE`**, **`+ KAMPF`**, **`+ FRACHT`** oder **`+ ENERGIE`** einen komplett bestückten neuen Unterpunkt (Seite) hinzufügen!
   - Per Klick: **SCHIFF WÄHLEN**, **BAUKASTEN**, **KOPIE**, **LÖSCHEN**, **+ NEUES SCHIFF** sowie **SC-TASTEN LESEN** (liest deine `actionmaps.xml` aus Star Citizen).
3. **TABLET / SERVER**:
   - **Server an- und ausschalten**: Großer Button **SERVER STARTEN** / **SERVER STOPPEN** für externe Tablets & Smartphones im WLAN.
   - Zeigt die echte WLAN-Adresse (z. B. `http://192.168.1.180:8765/`), den **ALS ADMIN STARTEN**-Button und das Live-Tastenprotokoll.
4. **EINSTELLUNGEN**:
   - Server-Port (`8765`) per `—` / `+` anpassen, Ordner öffnen, Verknüpfungen reparieren und **SC Cockpit direkt in der GUI deinstallieren**.

Wenn Windows beim ersten Serverstart nach der Firewall fragt: Zugriff im privaten Netzwerk erlauben.

## Bedienen, Flotten-Wechsel & Live-Telemetrie

Sowohl im Browser (Tablet/Smartphone) als auch in der Windows-App hast du sofortigen Zugriff auf deine gesamte Flotte:
- **1-Tap Schiffsflotten-Leiste**: Oben im Web-Interface sitzen die **Schiffs-Chips** mit Hersteller-Badge, Schiffsname und Rolle. Ein Tipp wechselt sofort auf das jeweilige Schiff.
- **Mitgelieferte Schiffe & Spezial-Shards**:
  1. **Aegis Gladius // Allround & Kampf** (*Flug*, *Energie & Schilde*, *Quantum*, *Kampf*)
  2. **MISC Prospector // Mining** (*Mining & Laser*, *Scan & Flug*, *Energie & Schilde*)
  3. **Drake Vulture // Salvage** (*Salvage & Scraper*, *Flug & Landung*, *Energie & Schilde*)
  4. **Crusader C2 // Cargo & Hauling** (*Fracht & Hangar*, *Flug & Quantum*, *Schilde & Abwehr*)
- **Live-Telemetrie-Leiste**: Zeigt **Energie-Kondensator** (`Waffen / Antrieb / Schilde`), **Schild-Sektoren** (`Bug / Heck / Backbord / Steuerbord`), `Master Mode` (`SCM` / `NAV`) sowie Status-Pills für `PWR`, `GEAR`, `VTOL`, `LIGHT` und `DECOUPLED`.

## Neues Schiff & Unterpunkte anlegen (Modularer Baukasten)

1. **`+ Neues Schiff` (Schiffs- & Subsystem-Assistent)**:
   - Klicke oben rechts auf **`+ Neues Schiff`**:
   - Gib **Schiffsname** (z. B. *RSI Constellation* oder *Argo Mole*), **Hersteller**, **Rolle** und **Cockpit-Atmosphäre** ein.
   - Hake einfach per Checkbox an, welche **Unterpunkte (Subsystem-Seiten)** das Schiff erhalten soll:
     - ☑ **Flug & Start/Landung**
     - ☑ **Energie & Schilde (Live)**
     - ☑ **Quantum & Navigation**
     - ☐ **Kampf & Abwehr**
     - ☐ **Mining & Erzabbau**
     - ☐ **Salvage & Bergung**
     - ☐ **Fracht, Rampen & Hangar**
   - Ein Klick auf **Schiff in Flotte anlegen ►** baut sofort das komplette Schiff mit allen gewählten Unterpunkten.
2. **Bestehendem Schiff neue Unterpunkte hinzufügen (`+ Unterpunkt`)**:
   - Links unter der Seiten-Navigation auf **`+ Unterpunkt`** klicken (oder im Baukasten das Dropdown *„+ Fertigen Unterpunkt (Seite) zum Schiff hinzufügen...“* bzw. in der Windows-App **`+ MINING` / `+ SALVAGE` / `+ KAMPF` / `+ FRACHT`** wählen).
3. **Einzelne Kacheln & Hintergrundbilder anpassen (`Baukasten`)**:
   - Über 35 fertige Star-Citizen-Presets und interaktive Live-Widgets per 1-Klick einfügen, eigene Hintergrundbilder hochladen und mit **SC-Tasten lesen** deine persönliche `actionmaps.xml` synchronisieren.

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
SC-Cockpit-Setup.exe      Eigenständiger Windows-GUI-Installer & Uninstaller (Star-Citizen-HUD)
installer/                Quellcode (Go/Win32) & Build-Skript (build_installer.py) für die .exe
server.py                 Server, nur Python-Standardbibliothek
index.html                Oberfläche, wird vom Server ausgeliefert
start.bat                 Start ohne Installation
install.bat               Startet SC-Cockpit-Setup.exe (Fallback auf install.ps1)
uninstall.bat             Startet den Deinstaller (Fallback auf uninstall.ps1)
dashboards/*.json         mitgelieferte Dashboards
dashboards/local/*.json   eigene Dashboards im Repo-Betrieb
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

