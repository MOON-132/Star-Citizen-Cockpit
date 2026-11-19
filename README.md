# SC Cockpit

Touch-Dashboard für Star Citizen auf einem Windows-Tablet (z. B. Surface x86).

Auf dem Gaming-PC läuft ein kleiner Python-Server. Das Tablet öffnet die Seite im Browser, im selben WLAN. Ein Tipp sendet deine Keybinds per Scancode an das fokussierte Spiel. Es gibt keinen Account, keine Cloud und keine extra Python-Pakete.

Das hier ist ein Befehls-Panel, kein zweiter Joystick und keine Anzeige von Schildwerten. Star Citizen hat dafür keine offizielle Schnittstelle.

## Schnellstart

1. Python 3 auf dem Gaming-PC installieren und „Add python.exe to PATH“ ankreuzen. [python.org](https://www.python.org/downloads/windows/)
2. Dieses Repository klonen:

```bat
git clone https://github.com/MOON-132/sc-cockpit.git
cd sc-cockpit
```

3. `start.bat` doppelklicken. Das Fenster zeigt eine Adresse wie `http://192.168.x.x:8765/`.
4. Die Adresse im Browser des Surface öffnen (Edge). Optional: Vollbild, oder „Zum Start hinzufügen“.
5. Wenn die Windows-Firewall fragt: nur privates Netzwerk erlauben.
6. Star Citizen muss im Vordergrund sein. Das Dashboard tippt Tasten, es wechselt nicht selbst ins Spiel.

Läuft Star Citizen als Administrator, `start.bat` ebenfalls per Rechtsklick **Als Administrator ausführen**. Sonst kommen die Tasten nicht an.

## Tasten an deine Belegung anpassen

Die Tasten in `layout.json` sind die **Standardbelegung um Alpha 4.10** (Flight Ready ist R-Alt+R, Fahrwerk N, Power F5–F8, Master Mode B halten). Wenn du umbelegt hast, stimmen sie nicht.

Persönliche Belegung, die nicht ins Repository soll:

1. `layout.local.json.example` nach `layout.local.json` kopieren, oder `layout.json` dorthin kopieren.
2. Tasten ändern. Der Server liest `layout.local.json`, sobald die Datei existiert, sonst `layout.json`.
3. Auf dem Tablet die Seite neu laden. Der Server muss nicht neu starten.

`layout.local.json` steht in `.gitignore`.

Deine echten Binds stehen in Star Citizen unter  
`...\Roberts Space Industries\StarCitizen\LIVE\USER\Client\0\Controls\Mappings\`  
(Export über Optionen → Keybindings → Advanced → Control Profiles). Die Datei `actionmaps.xml` im Profil enthält oft nur Abweichungen vom Standard.

### Felder eines Buttons

| Feld | Bedeutung |
| --- | --- |
| `id` | Eindeutig, wird im Log gezeigt |
| `label` | Text auf dem Button |
| `hint` | Kleine Zeile, z. B. die Taste |
| `keys` | Liste, z. B. `["n"]` oder `["ralt", "r"]`. Maximal 4 |
| `mode` | `tap` (kurz), `toggle` (tap + Anzeige am Tablet), `hold` (gedrückt lassen, solange der Finger liegt) |
| `color` | `amber`, `cyan`, `red`, `blue`, `green`, `slate` |
| `wide` | `true` = doppelte Breite |
| `danger` | `true` = geht erst nach **Gefahr AN** (Waffen, Laser, Abwurf) |
| `hold_ms` | optional, nur für `mode: "hold"` als zeitgesteuertes Halten ohne Finger-Tracking |

Tastennamen: `a`–`z`, `0`–`9`, `f1`–`f12`, `lshift`, `rshift`, `lctrl`, `rctrl`, `lalt`, `ralt`, `space`, `tab`, `enter`, `escape`, Pfeile, `mouse1`, `mouse2`, `mouse3`. Auf Windows wird der Scancode aus dem aktiven Layout erzeugt. Ein `y` ist damit die Taste Y, auch auf einem deutschen Keyboard.

Eine Seite im Layout ist eine Gruppe von Panels. Seiten löschen oder neue anlegen und die Tablet-Seite neu laden.

## Auf mehreren Rechnern weiterentwickeln

`main` ist die Version, die auf dem Gaming-PC läuft.

```bat
git pull
git add -A
git commit -m "Mining-Seite für die Prospector"
git push
```

Größere Layout-Experimente auf einen Branch:

```bat
git checkout -b layout-prospector
```

Der Gaming-PC braucht nur den geklonten Ordner und Python. Ein `git pull` plus Neuladen der Seite übernimmt ein neues Layout. Läuft der Server schon, muss er für reine Layout-Änderungen nicht neu starten.

`config.local.json` (Vorlage: `config.local.json.example`) ist ebenfalls lokal:

```json
{ "port": 8765, "token": "ein-langes-geheimnis", "hold_timeout_s": 4 }
```

Leeres `token` heißt: jeder im WLAN kann Tasten senden. In einem geteilten Netz ein Token setzen. Das Tablet fragt einmal danach und speichert es im Browser.

## Grenzen

- Es werden nur Tasten und Mausklicks gesendet. Schildbalken, Quantum-Kalibrierung und ähnliche Anzeigen kann das Panel nicht lesen.
- Ein Button ist eine Aktion, kein Makro mit Wartezeiten und Mausbewegung. Master Mode ist ein Halten von `B`, kein automatischer Quantum-Sprung.
- Easy Anti-Cheat bewertet simulierte Tasten selbst. Einzelne Keybinds sind dasselbe Verfahren wie bei GameGlass. Lies die Regeln von CIG. Keine Kampfscripte daraus bauen.
- Nicht als Passwort-Tresor oder aus dem Internet heraus betreiben. Kein HTTPS, gedacht für das Heimnetz.
- Selbstzerstörung und Schleudersitz sind absichtlich nicht im Layout.

## Prüfen ohne Spiel

```bat
py -3 server.py --self-test
```

Auf einem Nicht-Windows-Rechner startet der Server im Trockenlauf und schreibt die Tasten nur ins Fenster.

## Stand

`0.1.0` — Server, Touch-Dashboard, Standard-Layout (Flug, Energie, Quantum, Kampf, Mining-Beispiel), lokale Layout-Datei, optionales Token.
