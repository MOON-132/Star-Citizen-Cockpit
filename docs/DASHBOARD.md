# Dashboard-Format

Diese Datei ist die Referenz für neue Dashboards. Sie gilt für Menschen und für eine KI. Wenn eine KI ein Dashboard baut, hält sie sich an diesen Text und an [dashboard.schema.json](dashboard.schema.json). Ein gültiges Minimalbeispiel liegt in [beispiel.json](beispiel.json).

Ein Dashboard sendet nur Tasten an das gerade fokussierte Fenster. Es liest nichts aus Star Citizen. Ein Button ist eine Aktion, keine Abfolge mit Wartezeiten.

## Wohin die Datei kommt

Der Dateiname ist immer `<id>.json`. Die `id` im JSON ist derselbe Name.

| Situation | Ordner |
| --- | --- |
| Aus dem geklonten Repo mit `start.bat` | `dashboards/local/<id>.json` |
| Nach Installation (`SC-Cockpit-Setup.exe` / `install.bat`) | `%LOCALAPPDATA%\SC Cockpit\user\dashboards\<id>.json` |
| Mitliefern, für alle, im Git | `dashboards/<id>.json` |

`dashboards/local/` wird nicht committed. Eine lokale Datei mit derselben `id` wie eine mitgelieferte ersetzt diese nur für dich. Die Datei im Git bleibt. Löschen der lokalen Datei stellt das Original wieder her.

Der Server liest die Dateien bei jedem Abruf neu. Nach dem Speichern die Browserseite neu laden. Der Server muss dafür nicht neu starten.

Nicht benutzen: die Ids `local` und `state`, und keine Datei `state.json`.

## Aufbau

```json
{
  "id": "prospector",
  "title": "Prospector",
  "pages": [
    {
      "id": "laser",
      "label": "Laser",
      "groups": [
        {
          "title": "Abbau",
          "buttons": [
            {
              "id": "mining-laser",
              "label": "Laser",
              "hint": "Linksklick halten",
              "keys": ["mouse1"],
              "mode": "hold",
              "color": "red",
              "wide": true,
              "danger": true
            }
          ]
        }
      ]
    }
  ]
}
```

Beim Speichern über den Editor oder `PUT /api/dashboard/<id>` normalisiert der Server die Datei: unbekannte Felder fallen weg, Texte werden gekürzt, Ids werden zu ASCII-Slugs, unbekannte Farben werden `slate`. Eine von Hand in den Ordner gelegte Datei wird beim Laden nicht umgeschrieben. Trotzdem dieses Schema einhalten, sonst wirkt ein Button anders als gedacht oder der Server meldet beim Speichern einen Fehler.

### Dashboard

| Feld | Pflicht | Grenze | Bedeutung |
| --- | --- | --- | --- |
| `id` | ja | `^[a-z0-9][a-z0-9-]{0,40}$` | Kleinbuchstaben, Ziffern, Bindestrich. Gleich dem Dateinamen ohne `.json`. |
| `title` | ja | 40 Zeichen | Name in der Auswahlliste. Umlaute sind hier erlaubt. |
| `pages` | ja | 1 bis 12 | Seiten, oben als Tabs. |

### Seite

| Feld | Pflicht | Grenze | Bedeutung |
| --- | --- | --- | --- |
| `id` | ja | 40, Slug | Stabiler Name, nur intern. |
| `label` | ja | 24 Zeichen | Text auf dem Tab. |
| `groups` | ja | mindestens eine Gruppe mit Buttons | Blöcke untereinander. |

Eine Seite ohne Button wird beim Speichern abgelehnt.

### Gruppe

| Feld | Pflicht | Grenze | Bedeutung |
| --- | --- | --- | --- |
| `title` | ja | 40 Zeichen | Überschrift über den Buttons. |
| `buttons` | ja | mindestens einer | Die Tasten. |

### Button

| Feld | Pflicht | Grenze | Bedeutung |
| --- | --- | --- | --- |
| `id` | ja | 40, eindeutig im Dashboard | Intern, fürs Log. Nicht zwei Buttons dieselbe `id` geben. |
| `label` | ja | 40 Zeichen | Großer Text auf der Taste. |
| `hint` | nein | 60 Zeichen | Kleine Zeile, üblicherweise die Taste so wie im Spiel, z. B. `R-Alt + R`. |
| `keys` | ja | 1 bis 4 Einträge | Was gesendet wird. Modifier zuerst, die eigentliche Taste zuletzt. |
| `mode` | nein | `tap`, `toggle`, `hold` | Fehlt er, gilt `tap`. |
| `color` | nein | siehe unten | Fehlt sie, gilt `slate`. |
| `wide` | nein | `true` oder `false` | Doppelte Breite. |
| `danger` | nein | `true` oder `false` | Die Oberfläche sendet erst nach **Gefahr AN**. Der Server erzwingt das nicht. |

`keys: ["ralt", "r"]` heißt: Right Alt drücken, `R` tippen, beide loslassen. Nicht `["r", "ralt"]`.

### mode

| Wert | Verhalten |
| --- | --- |
| `tap` | Kurzer Druck. Für Schalter, die das Spiel selbst umlegt: Fahrwerk, Lichter, Power-Stufe. |
| `toggle` | Wie `tap`, plus eine Markierung nur auf dem Gerät. Das Spiel wird nicht ausgelesen. Nach einem Neuladen ist die Markierung weg, auch wenn das Schiff noch im anderen Zustand ist. |
| `hold` | Taste bleibt unten, solange der Finger darauf liegt. Für Boost, Master Mode, Radar-Ping, Waffen, Laser. Lässt das Gerät los oder schläft es ein, lässt der Server nach `hold_timeout_s` los (Standard 4 Sekunden). |

Es gibt keinen Modus für Makros, Wartezeiten oder „erst B, dann Maus“. Master Mode ist `hold` auf `b`. Der Sprung selbst bleibt am Spieler.

### color

Nur diese Werte, sonst wird beim Speichern `slate` daraus:

| Wert | Nutzen |
| --- | --- |
| `amber` | Hauptaktion, Antrieb, Quantum |
| `cyan` | Schilde, Fahrwerk, Abwehr |
| `red` | Feuer, Boost, Gefahr |
| `blue` | Flugmodus, Ziele |
| `green` | Systeme einschalten |
| `slate` | Selten, ohne Dringlichkeit |

### danger

`true` für alles, was schießt, einen Laser zündet oder Fracht abwirft. Nicht für Fahrwerk, Lichter oder Power. Selbstzerstörung und Schleudersitz gar nicht anlegen.

## Tastennamen

Modifier und Sondertasten immer mit diesen Namen, nicht als Zeichen:

`lshift` `rshift` `lctrl` `rctrl` `lalt` `ralt` `lwin` `rwin`

Buchstaben `a` bis `z`, Ziffern `0` bis `9`, Funktionstasten `f1` bis `f12`.

Weitere Namen: `space` `tab` `enter` `escape` `backspace` `capslock` `pause` `insert` `delete` `home` `end` `pageup` `pagedown` `left` `up` `right` `down`

Nummernblock: `num0` bis `num9`, `numadd`, `numsub`, `nummul`, `numdiv`, `numdot`, `numenter`

Maus: `mouse1` links, `mouse2` rechts, `mouse3` mitte, `wheelup`, `wheeldown`

Erlaubt, aber lieber die Namen von oben: `ctrl` meint `lctrl`, `shift` meint `lshift`, `alt` meint `lalt`, `altgr` meint `ralt`, `esc`, `return`, `del`, `pgup`, `lmb`, `rmb`, `mmb`.

Ein einzelnes Zeichen wie `ü` ist auf Windows erlaubt und wird über das aktive Tastaturlayout in einen Scancode übersetzt. Auf einer deutschen Tastatur ist `y` die Taste Y, nicht die physische Position einer US-Tastatur. Bindestrich, Komma und Ähnliches nur als dieses eine Zeichen, wenn es keine benannte Taste gibt.

Höchstens vier Einträge. Unbekannte Namen lehnt der Server ab.

## Standardbelegung, nur als Ausgangspunkt

Das sind die häufig genannten Tastatur-Defaults um Alpha 4.10, nicht das Profil des Spielers. Steht im Spiel eine andere Taste, diese nehmen und in `hint` so hinschreiben, wie sie im Keybind-Menü steht. Schildseiten (vorne, hinten, links, rechts) haben hier keine Defaults, weil die Belegung oft leer oder geändert ist. Die nicht erfinden.

| Aktion | keys | mode | danger |
| --- | --- | --- | --- |
| Flight Ready | `ralt`, `r` | tap | nein |
| Fahrwerk | `n` | toggle | nein |
| VTOL | `k` | toggle | nein |
| Schiffslichter | `l` | toggle | nein |
| Nachtsicht | `ralt`, `l` | toggle | nein |
| Boost | `lshift` | hold | nein |
| Decoupled | `c` | toggle | nein |
| Cruise | `lalt`, `c` | toggle | nein |
| Landeanfrage | `lalt`, `n` | tap | nein |
| Türen und Ports | `ralt`, `k` | tap | nein |
| Mehr Power Waffen | `f5` | tap | nein |
| Mehr Power Antrieb | `f6` | tap | nein |
| Mehr Power Schilde | `f7` | tap | nein |
| Power zurücksetzen | `f8` | tap | nein |
| Gesamt-Power | `u` | toggle | nein |
| Triebwerke | `i` | toggle | nein |
| Waffen-Power | `p` | toggle | nein |
| Schild-Power | `o` | toggle | nein |
| Master Mode | `b` | hold | nein |
| mobiGlas | `f1` | tap | nein |
| Radar-Ping | `tab` | hold | nein |
| Ziel wechseln | `t` | tap | nein |
| Gimbal-Modus | `g` | tap | nein |
| Flares | `h` | tap | nein |
| Noise | `j` | tap | nein |
| Raketenmodus | `mouse3` | tap | nein |
| Munition nachladen | `lalt`, `1` | tap | nein |
| Waffengruppe 1 | `mouse1` | hold | ja |
| Waffengruppe 2 | `mouse2` | hold | ja |
| Mining-Modus | `m` | toggle | nein |
| Mining-Laser | `mouse1` | hold | ja |
| Mining-Modul 1 bis 3 | `lalt` plus `1`, `2` oder `3` | tap | nein |
| Fracht abwerfen | `lalt`, `j` | tap | ja |

Die eigene Belegung exportiert Star Citizen über Optionen, Keybindings, Advanced Controls Customization, Control Profiles. Die Datei liegt unter:

`Roberts Space Industries\StarCitizen\LIVE\USER\Client\0\Controls\Mappings\`

`actionmaps.xml` im Profil enthält oft nur das, was vom Standard abweicht.

## Anlegen ohne Editor

1. [beispiel.json](beispiel.json) in den lokalen Ordner kopieren.
2. Datei und `id` umbenennen, z. B. `prospector.json` und `"id": "prospector"`.
3. Seiten und Buttons anpassen. JSON braucht doppelte Anführungszeichen und keine Kommentare, kein Komma nach dem letzten Eintrag.
4. Seite im Browser neu laden. Das neue Dashboard steht in der Liste.

Prüfen, ohne den Server dauerhaft zu starten:

```bat
py -3 server.py --self-test
```

Das prüft die mitgelieferten Dashboards und dieses Beispiel. Eine eigene Datei prüft der Server beim Speichern. Startet der Server schon, geht das auch so:

```bat
curl -X PUT http://127.0.0.1:8765/api/dashboard/prospector -H "Content-Type: application/json" --data-binary @prospector.json
```

Kommt `{"error": ...}` zurück, ist das der Grund. Ein Token aus `config.local.json` gehört zusätzlich in den Header `X-Token`.

## HTTP-API

Basis ist die Adresse aus dem Serverfenster, zum Beispiel `http://192.168.1.20:8765`. Alles ist JSON, UTF-8. Mit gesetztem Token schickt jeder Aufruf außer der HTML-Seite `X-Token`.

| Aufruf | Wirkung |
| --- | --- |
| `GET /api/health` | `{ok, version, dry_run, active, dashboards, held}` |
| `GET /api/dashboards` | Liste `{id, title, source}` und das aktive Id. `source` ist `builtin` oder `local`. |
| `GET /api/dashboard/<id>` | Ein Dashboard. `_source` sagt, woher es kommt. |
| `GET /api/layout` | Dasselbe für das gerade aktive Dashboard. |
| `PUT /api/dashboard/<id>` | Speichert nur in den lokalen Ordner. Body ist das Dashboard. `activate` fehlt oder ist `true`: dieses Dashboard wird aktiv. |
| `POST /api/dashboard/<id>` | Wie PUT. |
| `DELETE /api/dashboard/<id>` | Löscht nur die lokale Datei, nie eine mitgelieferte. |
| `POST /api/active` | Body `{"id": "mining"}`. |
| `POST /api/press` | Body `{"id": "gear", "keys": ["n"], "mode": "tap"}`. `mode` ist `tap`, `hold`, `down` oder `up`. `hold` ohne `down`/`up` hält `hold_ms` Millisekunden, 1 bis 10000, sonst etwa 400. Die Webseite benutzt für Halten selbst `down` und `up`. |
| `POST /api/release` | Alle noch gedrückten Tasten loslassen. |

Fehler antworten mit einem HTTP-Code ungleich 200 und `{"error": "Grund"}`.

## Anweisung für eine KI

Lies diese Datei, bevor du ein Dashboard erzeugst. Halte dich daran, auch wenn der Nutzer nur „bau mir ein Panel für die Prospector“ schreibt.

1. Eine Datei, ein Schiff oder eine Tätigkeit. Nicht alle Schiffe in ein Dashboard pressen, wenn der Nutzer mehrere nennt: dann mehrere Dateien.
2. Ausgabe ist nur das JSON, gültig nach [dashboard.schema.json](dashboard.schema.json), oder du schreibst genau `dashboards/local/<id>.json` bzw. die User-Datei der Installation. `id`, Dateiname und keine zweite Datei.
3. Keine Felder außerhalb des Schemas. Keine Kommentare, kein Markdown um das JSON, wenn der Nutzer die Datei direkt speichern will.
4. Ids auf Englisch und als Slug (`mining-laser`). Beschriftungen in der Sprache des Nutzers. `hint` zeigt die Taste so, wie ein Mensch sie liest (`L-Alt + 1`), `keys` die Maschinennamen (`["lalt", "1"]`).
5. Hat der Nutzer keine eigenen Keybinds genannt, nur die Tabelle oben verwenden und im `hint` und kurz im Begleittext sagen, dass es Defaults um Alpha 4.10 sind und er sie prüfen muss. Fehlende Binds nicht raten. Schildseiten weglassen, bis der Nutzer die Tasten nennt.
6. Waffenfeuer, Mining-Laser und Abwerfen: `mode` `hold` oder `tap` wie in der Tabelle, `danger` `true`, `color` `red`. Flight Ready, Power und Fahrwerk nicht als Gefahr markieren.
7. Keine Selbstzerstörung, kein Schleudersitz, kein Makro, keine Mausbewegung, keine Folge aus mehreren Schritten auf einem Button.
8. Lieber wenige große Buttons als jede Taste des Spiels. Eine Seite pro Aufgabe: Flug, Energie, Quantum, Kampf, Mining. Gruppen bündeln, was gleichzeitig gebraucht wird.
9. `wide: true` für die eine Hauptaktion der Seite. Maximal 12 Seiten.
10. Nach dem Schreiben die Datei gegen das Schema halten: jede Seite hat Buttons, jede `keys`-Liste hat 1 bis 4 bekannte Namen, Modifier stehen vorn.

## Konfiguration neben den Dashboards

`config.local.json` liegt im Repo-Ordner oder, nach der Installation, in `%LOCALAPPDATA%\SC Cockpit\user\`. Vorlage: `config.local.json.example`.

| Feld | Bedeutung |
| --- | --- |
| `port` | TCP-Port, Standard `8765`. |
| `token` | Leer heißt: jedes Gerät im WLAN darf Tasten senden. Ein langes Geheimnis verlangt den Header `X-Token`. Das Tablet fragt einmal und speichert ihn im Browser. |
| `hold_timeout_s` | Nach so vielen Sekunden ohne neues „noch gedrückt“ vom Browser wird die Taste losgelassen. Standard `4`. |

`state.json` daneben merkt sich nur die zuletzt gewählte Dashboard-Id. Nicht von Hand pflegen und nicht committen.
