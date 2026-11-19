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

## Installieren

`SC-Cockpit-Setup.exe` (oder `install.bat`) doppelklicken. Es öffnet sich der grafische Installer im Star-Citizen-MFD-Design. Eine Administrator-Abfrage kommt dabei nicht. Der Installer ist komplett eigenständig (alle Programmdateien und Dashboards sind in der `.exe` eingebettet), prüft auf Wunsch Python 3 bzw. bietet die automatische Installation über `winget` an, kopiert das Programm nach `%LOCALAPPDATA%\SC Cockpit\app` und legt an:

| Was | Wo |
| --- | --- |
| Programm, Doku, mitgelieferte Dashboards, Uninstaller | `%LOCALAPPDATA%\SC Cockpit\app` |
| Eigene Dashboards | `%LOCALAPPDATA%\SC Cockpit\user\dashboards` |
| Startmenü | Ordner **SC Cockpit** |
| Desktop | Verknüpfung **SC Cockpit** (mit SC-Cockpit-Icon) |
| Windows „Installierte Apps“ | Eintrag **SC Cockpit** zur einfachen Deinstallation |

Ein späterer Start von `SC-Cockpit-Setup.exe` erkennt die bestehende Installation automatisch und bietet ein Update (oder die Deinstallation) an. Eigene Dashboards und `config.local.json` im Ordner `user` bleiben beim Update erhalten.

Ohne Installation, zum Ausprobieren oder zum Weiterentwickeln: `start.bat` im geklonten Ordner. Eigene Dashboards liegen dann in `dashboards/local/` und kommen nicht ins Git.

## Starten & GameGlass Shard Hub (In-App Live Cockpit)

**SC Cockpit** auf dem Desktop oder im Startmenü starten. Es öffnet sich der **SC Cockpit // GameGlass Shard Hub** im holografischen GameGlass-MFD-Design (abgeschrägte Holo-Shards, Electric Cyan & Quantum Amber) mit vier Bereichen in der linken Seitenleiste:

1. **LIVE COCKPIT (Dashboards direkt in der App nutzen!)**:
   - **Sofort bedienbares In-App Cockpit-Deck**: Du kannst alle deine Dashboards und Shards (**Standard**, **Mining**, eigene Schiffe) direkt innerhalb des Windows-Programms bedienen — ideal für einen Zweitmonitor, Touchscreen-Monitor oder Windows-Tablet!
   - **Seiten & Shards umschalten**: Oben zwischen den Dashboards (`◄` / `►`) und darunter zwischen den Seiten-Reitern (*Flug*, *Energie & Schilde*, *Quantum*, *Kampf*) wechseln.
   - **Direktes Senden an Star Citizen**: Ein Klick auf eine Holo-Kachel im Programm fokussiert automatisch das laufende Star-Citizen-Fenster und sendet den Keybind direkt ins Spiel.
   - **Waffen SAFE / ARMED**: Gesicherte Tasten (Waffen, Laser, Abwerfen) werden über den Schalter **WAFFEN SAFE / ARMED** oben rechts scharfgeschaltet.
2. **SHARDS & DECKS**:
   - Übersicht aller mitgelieferten und eigenen Shards mit Seiten- und Tastenanzahl.
   - Per Klick: **IM COCKPIT NUTZEN** (lädt das Dashboard sofort in das In-App Live-Cockpit), **BAUKASTEN** (öffnet den visuellen Shard-Editor), **KOPIE**, **LÖSCHEN**, **+ NEUER SHARD** sowie **SC-TASTEN LESEN** (liest deine `actionmaps.xml` aus Star Citizen).
3. **TABLET / SERVER**:
   - **Server an- und ausschalten**: Großer Button **SERVER STARTEN** / **SERVER STOPPEN** für externe Tablets & Smartphones im WLAN.
   - Zeigt die echte WLAN-Adresse (z. B. `http://192.168.1.180:8765/`), den **ALS ADMIN STARTEN**-Button und das Live-Tastenprotokoll.
4. **EINSTELLUNGEN**:
   - Server-Port (`8765`) per `—` / `+` anpassen, Ordner öffnen, Verknüpfungen reparieren und **SC Cockpit direkt in der GUI deinstallieren**.

Wenn Windows beim ersten Serverstart nach der Firewall fragt: Zugriff im privaten Netzwerk erlauben.

## Bedienen & Live-Telemetrie

Oben steht der Verbindungsstatus zum PC sowie die **Live-Telemetrie-Leiste**:
- **Energie-Dreieck (Live)**: Zeigt die aktuelle prozentuale Verteilung auf **Waffen (F5)**, **Antrieb (F6)** und **Schilde (F7)**.
- **Schild-Sektoren (Live)**: Zeigt die Verteilung auf **Vorn**, **Hinten**, **Links** und **Rechts**.
- **Schiffssysteme & Star Citizen Status**: Erkennt automatisch über die lokale Star Citizen `Game.log`, welches Schiff aktiv ist, und zeigt den Status von Hauptstrom (`PWR`), Fahrwerk (`GEAR`), `VTOL`, `LICHT`, `DECOUPLED` sowie den `Master Mode` (`SCM` / `NAV`). Über `POST /api/telemetry` können zudem externe Tools Live-Werte ans Dashboard senden.

| Element | Wirkung |
| --- | --- |
| Dashboard-Liste | Wechselt das Panel. Die Wahl bleibt auf dem PC und lokal im Browser gespeichert (auch bei einem Seiten-Reload sofort verfügbar). |
| Tab | Seite innerhalb des Dashboards, zum Beispiel Flug, Energie & Schilde oder Kampf. |
| Taste / Widget | Sendet den hinterlegten Keybind (Standard-Impuls 80 ms wie in `tablet-deck`, oder `hold` solange der Finger gedrückt bleibt). |
| **Gefahr AUS / AN** | Gesicherte Tasten (mit gelb-rotem Warnstreifen wie Waffen, Laser, Selbstzerstörung, Abwerfen) sind gesperrt, bis das auf AN steht. |
| **Loslassen** | Alle Tasten, die hängengeblieben sein könnten, werden sofort losgelassen. |
| **Vollbild** | Blendet die Browser-Leiste aus. |
| **Baukasten / Anpassen** | Öffnet den modularen Baukasten-Editor (inspiriert von Home Assistant & Tablet Deck). |

## Eigenes Dashboard (Modularer Baukasten & Hintergrundbilder)

Drei Wege, dasselbe Format. Details, erlaubte Tasten und die Default-Belegung stehen in [docs/DASHBOARD.md](docs/DASHBOARD.md).

1. **Modularer Baukasten im Browser (`Baukasten / Anpassen`)**:
   - **1-Klick Star Citizen Presets**: Aus über 35 fertigen Star-Citizen-Bausteinen wählen (z. B. *Flight Ready = R-Alt + R*, *Fahrwerk = N*, *Landeerlaubnis = L-Alt + N*, *Master Mode = B*, *Täuschkörper*, *Mining-Module*) oder danach suchen — ein Klick fügt den fertigen Baustein sofort in die aktuelle Seite ein.
   - **Interaktive Live-Widgets**: Füge per Klick komplette Spezial-Kacheln wie den **Schild-Kompass (Interaktiv)** oder das **Energie-Dreieck (Live-Widget)** wie in Home Assistant hinzu.
   - **SC-Tasten aus Spiel lesen**: Ein Klick auf **SC-Tasten aus Spiel lesen** durchsucht automatisch deine lokale Star-Citizen-Installation (`LIVE/user/client/0/Profiles/default/actionmaps.xml`) und übernimmt deine echten Keybinds ins Dashboard.
   - **Eigene Hintergrundbilder & Cockpit-Themes**: Wähle zwischen fertigen Star-Citizen-Themes (*Aegis Tactical Grid*, *Crusader Horizon*, *Drake Industrial*, *Quantum Starmap*), füge eine Bild-URL ein oder lade ein **eigenes Hintergrundbild** direkt vom Gerät hoch — inklusive einstellbarer Deckkraft (`5%` bis `95%`).
2. **Als Datei**: `docs/beispiel.json` in den lokalen Dashboard-Ordner kopieren, Dateiname und `id` ändern, Seite neu laden.
3. **Mit einer KI**: Siehe Beispiel-Prompt in [docs/DASHBOARD.md](docs/DASHBOARD.md).

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

