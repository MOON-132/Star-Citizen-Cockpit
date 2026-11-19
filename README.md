# SC Cockpit

Touch-Dashboard für Star Citizen. Es läuft auf dem Gaming-PC und wird über eine Adresse im Browser bedient: Surface, Handy, Laptop, egal welches System.

Ein Tipp sendet einen Keybind an das fokussierte Spiel. Kein Account, keine Cloud, keine extra Python-Pakete. Schildbalken kann das Panel nicht lesen, dafür gibt es keine offizielle Schnittstelle.

## Installieren

Im Projektordner `install.bat` doppelklicken.

Das richtet für den aktuellen Windows-Benutzer ein:

- Programm unter `%LOCALAPPDATA%\SC Cockpit\app`
- eigene Dashboards unter `%LOCALAPPDATA%\SC Cockpit\user\dashboards`
- Verknüpfung auf dem Desktop und im Startmenü unter **SC Cockpit**

Python 3 muss installiert sein. Fehlt es, bietet der Installer `winget` an. Beim Python-Installer **Add python.exe to PATH** ankreuzen.

## Starten

**SC Cockpit** im Startmenü oder auf dem Desktop starten. Das Fenster zeigt eine Adresse wie `http://192.168.x.x:8765/`. Diese Adresse auf dem anderen Gerät öffnen. Das Fenster offen lassen und zu Star Citizen wechseln. Das Spiel muss im Vordergrund sein.

Wenn die Firewall fragt: nur privates Netzwerk erlauben.

Läuft Star Citizen als Administrator, die Verknüpfung **SC Cockpit (Administrator)** starten. Sonst kommen die Tasten nicht an.

## Deinstallieren

Startmenü → SC Cockpit → **Deinstallieren**.

Programm und Verknüpfungen werden entfernt. Eigene Dashboards bleiben liegen. Mit `uninstall.bat -Purge` werden auch die Dashboards gelöscht.

## Dashboards

Jedes Dashboard ist eine JSON-Datei.

| Ort | Bedeutung |
| --- | --- |
| `dashboards/*.json` | mitgeliefert, wird bei einem Update ersetzt |
| `dashboards/local/*.json` | eigene, wenn du aus dem Repo startest |
| `%LOCALAPPDATA%\SC Cockpit\user\dashboards` | eigene, nach der Installation |

Eine lokale Datei mit derselben Id legt sich über die mitgelieferte. Löschen im Editor stellt das Original wieder her.

Mitgeliefert sind **Standard** (Flug, Energie, Quantum, Kampf) und **Mining**. Die Tasten sind die Standardbelegung um Alpha 4.10. Eigene Belegung im Spiel gewinnt: im Browser **Anpassen**, Button antippen, Tasten eintragen, **Speichern**.

Neues Dashboard: **Anpassen → Neu**. Oder eine Datei `dashboards/local/prospector.json` anlegen und die Seite neu laden. Felder eines Buttons:

| Feld | Bedeutung |
| --- | --- |
| `label` | Text auf dem Button |
| `hint` | kleine Zeile |
| `keys` | z. B. `["n"]` oder `["ralt", "r"]`, höchstens 4 |
| `mode` | `tap`, `toggle` (nur die Anzeige) oder `hold` |
| `color` | `amber`, `cyan`, `red`, `blue`, `green`, `slate` |
| `wide` | doppelte Breite |
| `danger` | erst nach **Gefahr AN** |

Tastennamen: `a`–`z`, `0`–`9`, `f1`–`f12`, `lshift`, `rshift`, `lctrl`, `rctrl`, `lalt`, `ralt`, `space`, `tab`, `enter`, Pfeile, `mouse1`, `mouse2`, `mouse3`. Auf Windows kommt der Scancode aus dem aktiven Layout, damit Y auf einer deutschen Tastatur Y bleibt.

Im Browser gebaute Dashboards kann eine KI genauso als JSON-Datei liefern. Die Datei in den lokalen Ordner legen, Seite neu laden.

## Entwickeln

```bat
git clone https://github.com/MOON-132/sc-cockpit.git
cd sc-cockpit
start.bat
```

`start.bat` ist der portable Start ohne Installation. Eigene Dashboards liegen dann in `dashboards/local/` und werden nicht committed.

```bat
py -3 server.py --self-test
```

## Grenzen

- Nur Tasten und Mausklicks, keine Spielwerte.
- Ein Button ist eine Aktion, kein Makro. Master Mode ist ein Halten von `B`, kein automatischer Quantum-Sprung.
- Einzelne Keybinds sind dasselbe Verfahren wie bei anderen Touch-Panels. Die Regeln von CIG gelten. Keine Kampfscripte.
- Nur fürs eigene WLAN. Optional ein Token in `config.local.json` (`port`, `token`, `hold_timeout_s`). Im installierten Betrieb liegt die Datei im Ordner `user`.
- Selbstzerstörung und Schleudersitz sind absichtlich nicht dabei.

## Stand

`0.2.0` — mehrere Dashboards, Editor im Browser, Installer und Deinstaller fürs Startmenü.
