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

Im Projektordner `install.bat` doppelklicken. Eine Administrator-Abfrage kommt dabei nicht. Der Installer kopiert das Programm nach `%LOCALAPPDATA%\SC Cockpit\app` und legt an:

| Was | Wo |
| --- | --- |
| Programm, Doku, mitgelieferte Dashboards | `%LOCALAPPDATA%\SC Cockpit\app` |
| Eigene Dashboards | `%LOCALAPPDATA%\SC Cockpit\user\dashboards` |
| Startmenü | Ordner **SC Cockpit** |
| Desktop | Verknüpfung **SC Cockpit** |

Ein späteres `install.bat` ersetzt nur den Programmordner. Eigene Dashboards und `config.local.json` im Ordner `user` bleiben.

Ohne Installation, zum Ausprobieren oder zum Weiterentwickeln: `start.bat` im geklonten Ordner. Eigene Dashboards liegen dann in `dashboards/local/` und kommen nicht ins Git.

## Starten

**SC Cockpit** auf dem Desktop oder im Startmenü starten. Das schwarze Fenster zeigt eine oder mehrere Zeilen:

```text
Gerät:  http://192.168.x.x:8765/
```

Diese Adresse auf dem anderen Gerät öffnen. `localhost` oder `127.0.0.1` funktioniert nur auf dem Gaming-PC selbst. Das Fenster offen lassen und zu Star Citizen wechseln. Strg+C beendet den Server.

Wenn Windows nach der Firewall fragt: Zugriff nur im privaten Netzwerk erlauben. In einem öffentlichen oder Gastnetz erreichen sich die Geräte oft nicht, auch wenn beide „WLAN“ haben.

Läuft Star Citizen als Administrator, die Verknüpfung **SC Cockpit (Administrator)** verwenden. Ein normales Fenster darf an ein Administrator-Fenster keine Tasten senden.

## Bedienen

Oben steht, ob der PC verbunden ist. Daneben die Dashboard-Liste, darunter die Seiten des gewählten Dashboards.

| Element | Wirkung |
| --- | --- |
| Dashboard-Liste | Wechselt das Panel. Die Wahl bleibt auf dem PC gespeichert. |
| Tab | Seite innerhalb des Dashboards, zum Beispiel Flug oder Energie. |
| Taste | Sendet den hinterlegten Keybind. |
| **Gefahr AUS / AN** | Waffen, Laser und Abwerfen sind gesperrt, bis das auf AN steht. Gilt nur in dieser Browser-Ansicht. |
| **Loslassen** | Alle Tasten, die hängengeblieben sein könnten, werden losgelassen. |
| **Vollbild** | Blendet die Browser-Leiste aus. Auf dem Tablet danach „Zum Startbildschirm hinzufügen“, wenn der Browser das anbietet. |
| **Anpassen** | Editor. In dem Modus werden keine Tasten ins Spiel gesendet. |

Halten heißt: Finger drauf, Taste bleibt unten, Finger weg, Taste geht hoch. Schaltet das Display ab, lässt der Server die Taste nach wenigen Sekunden selbst los.

`Umschalten` auf einem Button merkt sich die Farbe nur auf dem Gerät. Das Spiel sagt dem Panel nicht, ob das Fahrwerk wirklich draußen ist.

## Eigenes Dashboard

Drei Wege, dasselbe Format. Details, erlaubte Tasten und die Default-Belegung stehen in [docs/DASHBOARD.md](docs/DASHBOARD.md).

**Im Browser.** **Anpassen**, Button antippen, Text und Tasten ändern, **Speichern**. **Neu** legt ein leeres Dashboard an, **Kopie** dupliziert das aktuelle. Speichern schreibt immer in deinen Ordner. Ein mitgeliefertes Dashboard wird dabei nicht überschrieben, sondern nur überdeckt. **Dashboard löschen** entfernt diese eigene Fassung.

**Als Datei.** `docs/beispiel.json` in den lokalen Dashboard-Ordner kopieren, Dateiname und `id` ändern, Seite neu laden.

**Mit einer KI.** Der Auftrag muss auf die Referenz zeigen, sonst erfindet das Modell Tasten und Felder. Sinnvoller Auftrag:

```text
Lies docs/DASHBOARD.md und docs/dashboard.schema.json im Repository
https://github.com/MOON-132/sc-cockpit
und erzeuge ein Dashboard für die Prospector.
Nur Funktionen, deren Tasten in der Default-Tabelle stehen oder die ich hier nenne.
Schreibe die Datei dashboards/local/prospector.json.
Keine Makros, keine Selbstzerstörung.
```

Eigene Keybinds aus dem Spiel dazulegen, wenn sie vom Standard abweichen. Export im Spiel: Optionen, Keybindings, Advanced Controls Customization, Control Profiles. Pfad:

`Roberts Space Industries\StarCitizen\LIVE\USER\Client\0\Controls\Mappings\`

Mitgeliefert sind **Standard** (Flug, Energie, Quantum, Kampf) und **Mining**. Die Tasten dort sind Defaults um Alpha 4.10, keine Garantie für dein Profil.

## Deinstallieren

Startmenü, Ordner SC Cockpit, **Deinstallieren**. Oder im Programmordner `uninstall.bat`.

Weg kommen Verknüpfungen und der Programmordner. Eigene Dashboards unter `%LOCALAPPDATA%\SC Cockpit\user` bleiben. Wirklich alles löschen:

```bat
uninstall.bat -Purge
```

## Dateien

```text
server.py                 Server, nur Python-Standardbibliothek
index.html                Oberfläche, wird vom Server ausgeliefert
start.bat                 Start ohne Installation
install.bat               Installation für den aktuellen Benutzer
uninstall.bat             Deinstallation
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

`main` ist die Fassung für den Spiel-PC. Layout-Experimente auf einen eigenen Branch. Ein `git pull` auf dem Spiel-PC und ein Neuladen der Seite übernehmen neue mitgelieferte Dashboards. Die installierte Kopie aktualisiert sich nicht von selbst: nach einem Pull dort erneut `install.bat` ausführen. Der Ordner `user` bleibt dabei.

Nicht committen: `config.local.json`, `dashboards/local/*.json`, `state.json`.

## Grenzen

- Keine Anzeige von Schild, Power, Ziel oder Quantumsstatus.
- Kein Joystick-Ersatz. Fliegen bleibt an Maus, Tastatur oder HOTAS.
- Kein Makro und kein automatischer Quantumsprung.
- Easy Anti-Cheat bewertet simulierte Tasten selbst. Einzelne Keybinds sind dasselbe Verfahren wie bei anderen Touch-Panels. Die Regeln von CIG gelten. Daraus keine Kampfscripte bauen.
- Kein Passwort außer dem optionalen Token, kein HTTPS. Nur im eigenen Netz verwenden.
- Selbstzerstörung und Schleudersitz sind absichtlich nicht in den mitgelieferten Dashboards.

## Stand

`0.2.0`. Mehrere Dashboards, Editor im Browser, Installer und Deinstaller. Das Dashboard-Format ist in `docs/DASHBOARD.md` festgelegt.
