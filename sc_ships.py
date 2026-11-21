#!/usr/bin/env python3
"""
Star Citizen Complete Ship Database & Personalized Dashboard Generator.
Provides 95+ iconic Star Citizen flyable ships across all manufacturers (AEGIS, ANVIL, RSI, DRAKE, MISC,
CRUSADER, ORIGIN, ARGO, CNOU, MIRAI,GATAC, ESPERIA, BANU, TUMBRIL, GREYCAT, KRUGER, AOPOA).
Each ship gets:
  - Tailored subsystem pages based on its actual roles (Flight, Combat, Mining, Salvage, Cargo, Medical,
    Exploration/Scan, E-War/QED, Dropship/Turrets, Power & Shields).
  - A custom high-tech SVG ship blueprint + manufacturer HUD artwork (`data:image/svg+xml;utf8,...`)
    depicting the ship's distinct silhouette, manufacturer emblem, hull classification, and technical specs
    so every ship is immediately recognizable on the dashboard and in the Hangar!
"""

from __future__ import annotations
import urllib.parse

# Ship Hull Silhouette SVG Paths (viewBox 0 0 800 450, centered around 400, 225)
HULL_PATHS = {
    "fighter_delta": (
        # Sleek Aegis/Sabre/Gladius/Eclipse delta wing interceptor
        "M 400 45 L 430 115 L 520 210 L 665 265 L 665 295 L 470 290 L 445 355 L 485 385 L 485 400 L 420 385 L 400 405 "
        "L 380 385 L 315 400 L 315 385 L 355 355 L 330 290 L 135 295 L 135 265 L 280 210 L 370 115 Z",
        "M 400 75 L 415 155 L 400 260 L 385 155 Z M 280 235 L 520 235 M 355 355 L 445 355"
    ),
    "fighter_anvil": (
        # Anvil Hornet / Hurricane / Hawk / F8C forward-swept / twin-boom heavy fighter
        "M 400 55 L 425 110 L 440 175 L 645 150 L 675 195 L 545 265 L 455 275 L 450 365 L 505 390 L 430 395 L 400 375 "
        "L 370 395 L 295 390 L 350 365 L 345 275 L 255 265 L 125 195 L 155 150 L 360 175 L 375 110 Z",
        "M 400 90 L 418 175 L 400 250 L 382 175 Z M 345 205 L 455 205 M 210 190 L 340 230 M 590 190 L 460 230"
    ),
    "twin_industrial": (
        # Drake Vulture / Caterpillar / Cutlass twin-prong front industrial hull
        "M 335 55 L 370 55 L 375 165 L 425 165 L 430 55 L 465 55 L 475 185 L 575 215 L 585 285 L 495 295 L 485 385 "
        "L 315 385 L 305 295 L 215 285 L 225 215 L 325 185 Z",
        "M 335 110 L 372 110 M 428 110 L 465 110 M 350 195 L 450 195 L 455 350 L 345 350 Z"
    ),
    "mining_bubble": (
        # MISC Prospector / Argo Mole / Orion industrial mining vessel
        "M 400 60 C 445 60 460 105 460 145 L 575 180 L 595 235 L 540 275 L 515 375 L 445 395 L 355 395 L 285 375 "
        "L 260 275 L 205 235 L 225 180 L 340 145 C 340 105 355 60 400 60 Z",
        "M 365 95 C 365 72 435 72 435 95 L 435 145 L 365 145 Z M 280 215 A 45 45 0 1 0 280 216 M 520 215 A 45 45 0 1 0 520 216"
    ),
    "crusader_wing": (
        # Crusader C2 / M2 / A2 / Mercury Star Runner / Spirit wide-body swept wing
        "M 400 50 L 445 110 L 545 175 L 715 235 L 690 285 L 545 275 L 495 365 L 435 395 L 365 395 L 305 365 "
        "L 255 275 L 110 285 L 85 235 L 255 175 L 355 110 Z",
        "M 400 80 L 445 190 L 430 355 L 370 355 L 355 190 Z M 245 215 L 555 215"
    ),
    "constellation_nacelle": (
        # RSI Constellation / Aurora / Perseus / Polaris quad-nacelle / angular wedge
        "M 400 40 L 445 95 L 460 175 L 575 155 L 590 295 L 485 290 L 495 390 L 450 405 L 435 365 L 365 365 L 350 405 "
        "L 305 390 L 315 290 L 210 295 L 225 155 L 340 175 L 355 95 Z",
        "M 400 65 L 430 145 L 430 345 L 370 345 L 370 145 Z M 240 175 L 240 285 M 560 175 L 560 285"
    ),
    "carrack_expedition": (
        # Anvil Carrack / 890 Jump / 600i / Odyssey long expedition capital/sub-capital
        "M 400 35 L 465 95 L 495 190 L 595 230 L 575 295 L 495 305 L 475 395 L 400 410 L 325 395 L 305 305 "
        "L 225 295 L 205 230 L 305 190 L 335 95 Z",
        "M 400 65 L 445 130 L 445 360 L 355 360 L 355 130 Z M 310 220 L 490 220 M 335 290 L 465 290"
    ),
    "hauler_hull": (
        # MISC Freelancer / Hull-A / Hull-C / Starfarer elongated cargo spine
        "M 400 40 C 440 40 455 85 455 140 L 455 210 L 585 220 L 585 270 L 460 275 L 475 385 L 425 405 L 375 405 "
        "L 325 385 L 340 275 L 215 270 L 215 220 L 345 210 L 345 140 C 345 85 360 40 400 40 Z",
        "M 370 90 L 430 90 L 430 365 L 370 365 Z M 255 245 L 545 245"
    ),
    "alien_organic": (
        # Banu Defender / Khartu-al / San'tok.yāi / Syulen / Glaive exotic geometry
        "M 400 95 L 445 60 L 535 45 L 615 135 L 560 255 L 645 345 L 525 395 L 435 305 L 400 345 L 365 305 "
        "L 275 395 L 155 345 L 240 255 L 185 135 L 265 45 L 355 60 Z",
        "M 400 120 A 55 55 0 1 0 400 230 A 55 55 0 1 0 400 120 M 265 95 L 360 185 M 535 95 L 440 185"
    ),
}

MFG_COLORS = {
    "AEGIS": ("#00e5ff", "#071a26"),
    "ANVIL": ("#3d8bff", "#08172b"),
    "RSI": ("#00d4ff", "#071924"),
    "DRAKE": ("#ffab00", "#1f1507"),
    "MISC": ("#ff9100", "#1c1308"),
    "CRUSADER": ("#4fc3f7", "#081c29"),
    "ORIGIN": ("#80e5ff", "#091d26"),
    "ARGO": ("#ff6d00", "#211106"),
    "CNOU": ("#00e676", "#071f14"),
    "MIRAI": ("#ff3d57", "#240a10"),
    "GATAC": ("#00e5ff", "#081e24"),
    "ESPERIA": ("#ff3d57", "#210911"),
    "BANU": ("#ffd740", "#1f1907"),
    "AOPOA": ("#00e676", "#071f14"),
    "KRUGER": ("#8da2b8", "#101822"),
    "TUMBRIL": ("#ffab00", "#1c1508"),
    "GREYCAT": ("#ffab00", "#1c1508"),
}


def make_ship_svg_data_uri(ship_name: str, mfg: str, role: str, hull_type: str) -> str:
    """Generates a custom GameGlass-style technical blueprint background with the ship's silhouette."""
    stroke_col, bg_col = MFG_COLORS.get(mfg.upper(), ("#00e5ff", "#071824"))
    outer_path, inner_path = HULL_PATHS.get(hull_type, HULL_PATHS["fighter_delta"])
    safe_name = ship_name.upper().replace("&", "&amp;").replace("<", "&lt;").replace(">", "&gt;")
    safe_mfg = mfg.upper().replace("&", "&amp;")
    safe_role = role.upper().replace("&", "&amp;")

    svg = f"""<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 800 450" width="100%" height="100%">
  <defs>
    <radialGradient id="glow" cx="50%" cy="50%" r="55%">
      <stop offset="0%" stop-color="{stroke_col}" stop-opacity="0.24"/>
      <stop offset="60%" stop-color="{bg_col}" stop-opacity="0.75"/>
      <stop offset="100%" stop-color="#05080d" stop-opacity="0.98"/>
    </radialGradient>
    <pattern id="grid" width="40" height="40" patternUnits="userSpaceOnUse">
      <path d="M 40 0 L 0 0 0 40" fill="none" stroke="{stroke_col}" stroke-width="0.6" stroke-opacity="0.12"/>
    </pattern>
  </defs>
  <rect width="800" height="450" fill="url(#glow)"/>
  <rect width="800" height="450" fill="url(#grid)"/>
  <!-- HUD Telemetry Rings -->
  <circle cx="400" cy="225" r="185" fill="none" stroke="{stroke_col}" stroke-width="1" stroke-opacity="0.22" stroke-dasharray="12 6"/>
  <circle cx="400" cy="225" r="145" fill="none" stroke="{stroke_col}" stroke-width="0.8" stroke-opacity="0.16"/>
  <line x1="160" y1="225" x2="640" y2="225" stroke="{stroke_col}" stroke-width="0.7" stroke-opacity="0.18"/>
  <line x1="400" y1="20" x2="400" y2="430" stroke="{stroke_col}" stroke-width="0.7" stroke-opacity="0.18"/>
  <!-- Ship Blueprint Silhouette -->
  <path d="{outer_path}" fill="{stroke_col}" fill-opacity="0.11" stroke="{stroke_col}" stroke-width="2.4" stroke-linejoin="round"/>
  <path d="{inner_path}" fill="none" stroke="{stroke_col}" stroke-width="1.3" stroke-opacity="0.55"/>
  <!-- HUD Corner Technical Readouts -->
  <path d="M 25 45 L 25 25 L 85 25" fill="none" stroke="{stroke_col}" stroke-width="2" stroke-opacity="0.6"/>
  <path d="M 775 45 L 775 25 L 715 25" fill="none" stroke="{stroke_col}" stroke-width="2" stroke-opacity="0.6"/>
  <path d="M 25 405 L 25 425 L 85 425" fill="none" stroke="{stroke_col}" stroke-width="2" stroke-opacity="0.6"/>
  <path d="M 775 405 L 775 425 L 715 425" fill="none" stroke="{stroke_col}" stroke-width="2" stroke-opacity="0.6"/>
  <text x="38" y="48" fill="{stroke_col}" font-family="Bahnschrift,Segoe UI,sans-serif" font-size="15" font-weight="bold" letter-spacing="3" opacity="0.85">{safe_mfg} // TACTICAL SHARD</text>
  <text x="38" y="70" fill="#eef5fc" font-family="Bahnschrift,Segoe UI,sans-serif" font-size="20" font-weight="bold" letter-spacing="2" opacity="0.9">{safe_name}</text>
  <text x="38" y="412" fill="{stroke_col}" font-family="Consolas,monospace" font-size="12" letter-spacing="2" opacity="0.75">CLASS: {safe_role}  //  SYS: ONLINE</text>
</svg>"""
    return "data:image/svg+xml;utf8," + urllib.parse.quote(svg)


# Complete Star Citizen Ship Catalog: (id, ship_name, manufacturer, role_label, hull_type, subsystem_tags)
# Subsystem tags: "flug", "kampf", "mining", "salvage", "cargo", "medical", "exploration", "ewar", "dropship", "energie", "quantum"
SC_SHIPS_DB = [
    # AEGIS DYNAMICS
    ("aegis-avenger-titan", "Aegis Avenger Titan", "AEGIS", "Light Freight & Interceptor", "fighter_delta", ["flug", "kampf", "cargo", "energie", "quantum"]),
    ("aegis-avenger-stalker", "Aegis Avenger Stalker", "AEGIS", "Bounty Hunter & Interdiction", "fighter_delta", ["flug", "kampf", "ewar", "energie", "quantum"]),
    ("aegis-avenger-warlock", "Aegis Avenger Warlock", "AEGIS", "EMP Electronic Warfare", "fighter_delta", ["flug", "ewar", "kampf", "energie", "quantum"]),
    ("aegis-gladius", "Aegis Gladius", "AEGIS", "Light Fighter", "fighter_delta", ["flug", "kampf", "energie", "quantum"]),
    ("aegis-sabre", "Aegis Sabre", "AEGIS", "Stealth Space Superiority", "fighter_delta", ["flug", "kampf", "energie", "quantum"]),
    ("aegis-sabre-comet", "Aegis Sabre Comet", "AEGIS", "Stealth Fighter", "fighter_delta", ["flug", "kampf", "energie", "quantum"]),
    ("aegis-sabre-raven", "Aegis Sabre Raven", "AEGIS", "EMP Stealth Interceptor", "fighter_delta", ["flug", "ewar", "kampf", "energie", "quantum"]),
    ("aegis-sabre-firebird", "Aegis Sabre Firebird", "AEGIS", "Missile Stealth Strike", "fighter_delta", ["flug", "kampf", "energie", "quantum"]),
    ("aegis-sabre-peregrine", "Aegis Sabre Peregrine", "AEGIS", "High-Speed Racing", "fighter_delta", ["flug", "quantum", "energie"]),
    ("aegis-vanguard-warden", "Aegis Vanguard Warden", "AEGIS", "Long-Range Heavy Fighter", "fighter_anvil", ["flug", "kampf", "energie", "quantum"]),
    ("aegis-vanguard-sentinel", "Aegis Vanguard Sentinel", "AEGIS", "EMP & E-War Heavy Fighter", "fighter_anvil", ["flug", "ewar", "kampf", "energie", "quantum"]),
    ("aegis-vanguard-harbinger", "Aegis Vanguard Harbinger", "AEGIS", "Heavy Fighter Bomber", "fighter_anvil", ["flug", "kampf", "energie", "quantum"]),
    ("aegis-vanguard-hoplite", "Aegis Vanguard Hoplite", "AEGIS", "Combat Dropship", "fighter_anvil", ["flug", "dropship", "kampf", "energie", "quantum"]),
    ("aegis-eclipse", "Aegis Eclipse", "AEGIS", "Stealth Torpedo Bomber", "fighter_delta", ["flug", "kampf", "energie", "quantum"]),
    ("aegis-retaliator", "Aegis Retaliator", "AEGIS", "Modular Heavy Bomber & Cargo", "carrack_expedition", ["flug", "kampf", "cargo", "energie", "quantum"]),
    ("aegis-redeemer", "Aegis Redeemer", "AEGIS", "Heavy Gunship", "fighter_anvil", ["flug", "kampf", "dropship", "energie", "quantum"]),
    ("aegis-reclaimer", "Aegis Reclaimer", "AEGIS", "Capital Heavy Salvage", "twin_industrial", ["salvage", "flug", "cargo", "energie", "quantum"]),
    ("aegis-hammerhead", "Aegis Hammerhead", "AEGIS", "Anti-Fighter Corvette", "carrack_expedition", ["flug", "kampf", "dropship", "energie", "quantum"]),
    ("aegis-idris", "Aegis Idris-P", "AEGIS", "Capital Frigate", "carrack_expedition", ["flug", "kampf", "cargo", "energie", "quantum"]),

    # ANVIL AEROSPACE
    ("anvil-arrow", "Anvil Arrow", "ANVIL", "Agile Light Fighter", "fighter_delta", ["flug", "kampf", "energie", "quantum"]),
    ("anvil-gladiator", "Anvil Gladiator", "ANVIL", "Carrier Torpedo Bomber", "fighter_anvil", ["flug", "kampf", "energie", "quantum"]),
    ("anvil-hawk", "Anvil Hawk", "ANVIL", "Bounty Hunter & EMP", "fighter_anvil", ["flug", "ewar", "kampf", "energie", "quantum"]),
    ("anvil-hornet-f7c", "Anvil F7C Hornet Mk II", "ANVIL", "Medium Space Superiority", "fighter_anvil", ["flug", "kampf", "energie", "quantum"]),
    ("anvil-super-hornet", "Anvil F7C-M Super Hornet", "ANVIL", "Two-Seat Medium Fighter", "fighter_anvil", ["flug", "kampf", "energie", "quantum"]),
    ("anvil-hornet-ghost", "Anvil F7C-S Hornet Ghost", "ANVIL", "Stealth Medium Fighter", "fighter_anvil", ["flug", "kampf", "energie", "quantum"]),
    ("anvil-hornet-tracker", "Anvil F7C-R Hornet Tracker", "ANVIL", "Deep-Scan Fighter", "fighter_anvil", ["flug", "exploration", "kampf", "energie", "quantum"]),
    ("anvil-f8c-lightning", "Anvil F8C Lightning", "ANVIL", "Heavy Space Superiority", "fighter_anvil", ["flug", "kampf", "energie", "quantum"]),
    ("anvil-hurricane", "Anvil Hurricane", "ANVIL", "Heavy Turret Glass Cannon", "fighter_anvil", ["flug", "kampf", "energie", "quantum"]),
    ("anvil-terrapin", "Anvil Terrapin", "ANVIL", "Armored Recon & Pathfinder", "mining_bubble", ["flug", "exploration", "energie", "quantum"]),
    ("anvil-terrapin-medic", "Anvil Terrapin Medic", "ANVIL", "Armored Medical Evac", "mining_bubble", ["medical", "flug", "energie", "quantum"]),
    ("anvil-valkyrie", "Anvil Valkyrie", "ANVIL", "Heavy Troop Dropship", "fighter_anvil", ["flug", "dropship", "cargo", "kampf", "energie"]),
    ("anvil-carrack", "Anvil Carrack", "ANVIL", "Multi-Crew Deep Exploration", "carrack_expedition", ["exploration", "flug", "medical", "cargo", "energie", "quantum"]),
    ("anvil-c8x-pisces", "Anvil C8X Pisces Expedition", "ANVIL", "Snub Pathfinder & Shuttle", "fighter_delta", ["flug", "exploration", "cargo", "energie", "quantum"]),
    ("anvil-c8r-pisces", "Anvil C8R Pisces Rescue", "ANVIL", "Medical Rescue Shuttle", "fighter_delta", ["medical", "flug", "energie", "quantum"]),

    # ROBERTS SPACE INDUSTRIES (RSI)
    ("rsi-aurora-mr", "RSI Aurora MR", "RSI", "Starter Light Interdiction", "constellation_nacelle", ["flug", "kampf", "cargo", "energie", "quantum"]),
    ("rsi-aurora-ln", "RSI Aurora LN", "RSI", "Militia Light Fighter", "constellation_nacelle", ["flug", "kampf", "energie", "quantum"]),
    ("rsi-mantis", "RSI Mantis", "RSI", "Quantum Enforcement (QED)", "constellation_nacelle", ["ewar", "flug", "kampf", "energie", "quantum"]),
    ("rsi-scorpius", "RSI Scorpius", "RSI", "X-Wing Heavy Interceptor", "fighter_anvil", ["flug", "kampf", "energie", "quantum"]),
    ("rsi-scorpius-antares", "RSI Scorpius Antares", "RSI", "EMP & Quantum Jammer", "fighter_anvil", ["ewar", "flug", "kampf", "energie", "quantum"]),
    ("rsi-zeus-mk2-cl", "RSI Zeus Mk II CL", "RSI", "Medium Cargo Clipper", "fighter_delta", ["cargo", "flug", "energie", "quantum"]),
    ("rsi-zeus-mk2-es", "RSI Zeus Mk II ES", "RSI", "Medium Expedition & Recon", "fighter_delta", ["exploration", "flug", "cargo", "energie", "quantum"]),
    ("rsi-constellation-andromeda", "RSI Constellation Andromeda", "RSI", "Multi-Crew Gunship & Freighter", "constellation_nacelle", ["flug", "kampf", "cargo", "energie", "quantum"]),
    ("rsi-constellation-taurus", "RSI Constellation Taurus", "RSI", "Heavy Medium Freighter", "constellation_nacelle", ["cargo", "flug", "kampf", "energie", "quantum"]),
    ("rsi-constellation-aquila", "RSI Constellation Aquila", "RSI", "Deep Space Exploration", "constellation_nacelle", ["exploration", "flug", "cargo", "energie", "quantum"]),
    ("rsi-constellation-phoenix", "RSI Constellation Phoenix", "RSI", "Luxury Command & Touring", "constellation_nacelle", ["flug", "kampf", "cargo", "energie", "quantum"]),
    ("rsi-polaris", "RSI Polaris", "RSI", "Capital Torpedo Corvette", "carrack_expedition", ["kampf", "flug", "cargo", "medical", "energie", "quantum"]),

    # DRAKE INTERPLANETARY
    ("drake-cutter", "Drake Cutter", "DRAKE", "Starter Compact Hauler", "twin_industrial", ["flug", "cargo", "energie", "quantum"]),
    ("drake-cutter-rambler", "Drake Cutter Rambler", "DRAKE", "Long-Range Expedition Starter", "twin_industrial", ["exploration", "flug", "energie", "quantum"]),
    ("drake-cutter-scout", "Drake Cutter Scout", "DRAKE", "Recon & Radar Starter", "twin_industrial", ["exploration", "flug", "energie", "quantum"]),
    ("drake-buccaneer", "Drake Buccaneer", "DRAKE", "Interceptor Fighter", "fighter_anvil", ["flug", "kampf", "energie", "quantum"]),
    ("drake-herald", "Drake Herald", "DRAKE", "Data Runner & E-War", "fighter_delta", ["ewar", "flug", "quantum", "energie"]),
    ("drake-vulture", "Drake Vulture", "DRAKE", "Solo Light Salvage", "twin_industrial", ["salvage", "flug", "cargo", "energie", "quantum"]),
    ("drake-cutlass-black", "Drake Cutlass Black", "DRAKE", "Multi-Role Medium Fighter & Cargo", "twin_industrial", ["flug", "kampf", "cargo", "dropship", "energie", "quantum"]),
    ("drake-cutlass-red", "Drake Cutlass Red", "DRAKE", "Search & Rescue Medical", "twin_industrial", ["medical", "flug", "exploration", "energie", "quantum"]),
    ("drake-cutlass-blue", "Drake Cutlass Blue", "DRAKE", "Quantum Dampener & Patrol", "twin_industrial", ["ewar", "kampf", "flug", "energie", "quantum"]),
    ("drake-cutlass-steel", "Drake Cutlass Steel", "DRAKE", "Heavy Troop Dropship", "twin_industrial", ["dropship", "kampf", "flug", "energie", "quantum"]),
    ("drake-corsair", "Drake Corsair", "DRAKE", "Heavy Armed Explorer", "crusader_wing", ["flug", "kampf", "exploration", "cargo", "energie", "quantum"]),
    ("drake-caterpillar", "Drake Caterpillar", "DRAKE", "Modular Heavy Freighter", "twin_industrial", ["cargo", "flug", "kampf", "energie", "quantum"]),

    # MISC (MUSASHI INDUSTRIAL & STARFLIGHT CONCERN)
    ("misc-prospector", "MISC Prospector", "MISC", "Solo Laser Mining", "mining_bubble", ["mining", "exploration", "flug", "energie", "quantum"]),
    ("misc-reliant-kore", "MISC Reliant Kore", "MISC", "Vertical-Wing Light Hauler", "crusader_wing", ["cargo", "flug", "kampf", "energie", "quantum"]),
    ("misc-reliant-tana", "MISC Reliant Tana", "MISC", "Light Skirmisher", "crusader_wing", ["flug", "kampf", "energie", "quantum"]),
    ("misc-reliant-sen", "MISC Reliant Sen", "MISC", "Mobile Science & Survey", "crusader_wing", ["exploration", "flug", "energie", "quantum"]),
    ("misc-freelancer", "MISC Freelancer", "MISC", "Mercantile Medium Freighter", "hauler_hull", ["cargo", "flug", "kampf", "energie", "quantum"]),
    ("misc-freelancer-max", "MISC Freelancer MAX", "MISC", "Wide-Body Heavy Cargo", "hauler_hull", ["cargo", "flug", "energie", "quantum"]),
    ("misc-freelancer-dur", "MISC Freelancer DUR", "MISC", "Long-Range Explorer", "hauler_hull", ["exploration", "flug", "cargo", "energie", "quantum"]),
    ("misc-freelancer-mis", "MISC Freelancer MIS", "MISC", "Militarized Missile Gunship", "hauler_hull", ["kampf", "flug", "cargo", "energie", "quantum"]),
    ("misc-hull-a", "MISC Hull A", "MISC", "External Spindle Freighter", "hauler_hull", ["cargo", "flug", "energie", "quantum"]),
    ("misc-hull-c", "MISC Hull C", "MISC", "Heavy Spindle Bulk Hauler", "hauler_hull", ["cargo", "flug", "energie", "quantum"]),
    ("misc-starfarer", "MISC Starfarer", "MISC", "Heavy Fuel Tanker & Refuel", "hauler_hull", ["cargo", "flug", "energie", "quantum"]),
    ("misc-starlancer-max", "MISC Starlancer MAX", "MISC", "Deep-Space Heavy Transport", "hauler_hull", ["cargo", "flug", "kampf", "energie", "quantum"]),

    # CRUSADER INDUSTRIES
    ("crusader-ares-ion", "Crusader Ares Ion", "CRUSADER", "S7 Laser Anti-Capital Fighter", "crusader_wing", ["kampf", "flug", "energie", "quantum"]),
    ("crusader-ares-inferno", "Crusader Ares Inferno", "CRUSADER", "S7 Ballistic Gatling Fighter", "crusader_wing", ["kampf", "flug", "energie", "quantum"]),
    ("crusader-intrepid", "Crusader Intrepid", "CRUSADER", "Light Starter Courier", "crusader_wing", ["flug", "cargo", "energie", "quantum"]),
    ("crusader-spirit-c1", "Crusader C1 Spirit", "CRUSADER", "Medium Cargo & Tractor Beam", "crusader_wing", ["cargo", "flug", "energie", "quantum"]),
    ("crusader-spirit-a1", "Crusader A1 Spirit", "CRUSADER", "Medium Tactical Bomber", "crusader_wing", ["kampf", "flug", "energie", "quantum"]),
    ("crusader-mercury-star-runner", "Crusader Mercury Star Runner", "CRUSADER", "Data Runner & Smuggler", "crusader_wing", ["cargo", "ewar", "flug", "energie", "quantum"]),
    ("crusader-c2-hercules", "Crusader C2 Hercules", "CRUSADER", "Heavy Vehicle & Cargo Starlifter", "crusader_wing", ["cargo", "flug", "energie", "quantum"]),
    ("crusader-m2-hercules", "Crusader M2 Hercules", "CRUSADER", "Military Armored Starlifter", "crusader_wing", ["cargo", "dropship", "kampf", "flug", "energie", "quantum"]),
    ("crusader-a2-hercules", "Crusader A2 Hercules", "CRUSADER", "MOAB Heavy Gunship & Bomber", "crusader_wing", ["kampf", "cargo", "flug", "energie", "quantum"]),

    # ORIGIN JUMPWORKS
    ("origin-100i", "Origin 100i", "ORIGIN", "Luxury Touring Starter", "fighter_delta", ["flug", "cargo", "energie", "quantum"]),
    ("origin-125a", "Origin 125a", "ORIGIN", "Luxury Light Fighter", "fighter_delta", ["flug", "kampf", "energie", "quantum"]),
    ("origin-135c", "Origin 135c", "ORIGIN", "Luxury Light Cargo", "fighter_delta", ["cargo", "flug", "energie", "quantum"]),
    ("origin-300i", "Origin 300i", "ORIGIN", "Luxury Touring Interceptor", "fighter_delta", ["flug", "kampf", "energie", "quantum"]),
    ("origin-315p", "Origin 315p", "ORIGIN", "Custom Pathfinder & Tractor", "fighter_delta", ["exploration", "flug", "cargo", "energie", "quantum"]),
    ("origin-325a", "Origin 325a", "ORIGIN", "Luxury Interdiction Fighter", "fighter_delta", ["kampf", "flug", "energie", "quantum"]),
    ("origin-350r", "Origin 350r", "ORIGIN", "Twin-Engine Pure Racer", "fighter_delta", ["flug", "quantum", "energie"]),
    ("origin-400i", "Origin 400i", "ORIGIN", "High-Performance Pathfinder", "carrack_expedition", ["exploration", "flug", "cargo", "energie", "quantum"]),
    ("origin-600i-explorer", "Origin 600i Explorer", "ORIGIN", "Luxury Expedition Vessel", "carrack_expedition", ["exploration", "flug", "cargo", "energie", "quantum"]),
    ("origin-890-jump", "Origin 890 Jump", "ORIGIN", "Capital Luxury Touring Yacht", "carrack_expedition", ["flug", "medical", "cargo", "energie", "quantum"]),

    # ARGO ASTRONAUTICS
    ("argo-mole", "Argo MOLE", "ARGO", "Multi-Crew Triple-Laser Mining", "mining_bubble", ["mining", "exploration", "flug", "energie", "quantum"]),
    ("argo-srv", "Argo SRV", "ARGO", "Heavy Tractor & Recovery Tug", "mining_bubble", ["cargo", "flug", "energie", "quantum"]),
    ("argo-raft", "Argo RAFT", "ARGO", "Container Freight Hauler", "twin_industrial", ["cargo", "flug", "energie", "quantum"]),
    ("argo-mpuv-tractor", "Argo MPUV Tractor", "ARGO", "Industrial Utility Pod", "mining_bubble", ["cargo", "flug", "energie"]),

    # CONSOLIDATED OUTLAND (CNOU), MIRAI, ALIEN & SPECIALTY
    ("cnou-mustang-alpha", "C.O. Mustang Alpha", "CNOU", "Sleek Light Cargo Starter", "fighter_anvil", ["flug", "cargo", "kampf", "energie", "quantum"]),
    ("cnou-nomad", "C.O. Nomad", "CNOU", "Open-Bed Pickup Freighter", "fighter_delta", ["cargo", "flug", "energie", "quantum"]),
    ("mirai-fury", "Mirai Fury", "MIRAI", "Snub Swarm Fighter", "alien_organic", ["kampf", "flug", "energie"]),
    ("mirai-razor", "Mirai Razor", "MIRAI", "Formula Racing Interceptor", "fighter_delta", ["flug", "quantum", "energie"]),
    ("mirai-guardian", "Mirai Guardian", "MIRAI", "Heavy Long-Range Fighter", "alien_organic", ["kampf", "flug", "energie", "quantum"]),
    ("gatac-syulen", "Gatac Syulen", "GATAC", "Xi'an Vertical Cargo Starter", "alien_organic", ["flug", "cargo", "kampf", "energie", "quantum"]),
    ("banu-defender", "Banu Defender", "BANU", "Tachyon Twin-Arm Escort Fighter", "alien_organic", ["kampf", "flug", "energie", "quantum"]),
    ("esperia-prowler", "Esperia Prowler", "ESPERIA", "Tevarin Stealth Dropship", "alien_organic", ["dropship", "kampf", "flug", "energie", "quantum"]),
    ("esperia-talon", "Esperia Talon", "ESPERIA", "Tevarin Light Strike Fighter", "alien_organic", ["kampf", "flug", "energie", "quantum"]),
    ("aopoa-santokyai", "Aopoa San'tok.yāi", "AOPOA", "Xi'an Medium Fighter", "alien_organic", ["kampf", "flug", "energie", "quantum"]),
]


def _build_page_for_tag(tag: str, ship_name: str, mfg: str) -> dict:
    if tag == "mining":
        return {
            "id": "mining",
            "label": "Mining & Laser",
            "groups": [
                {
                    "title": f"{ship_name} // Mining-Laser & Extraktion",
                    "buttons": [
                        {"id": "mining-mode", "sc": "v_toggle_mining_mode", "label": "Mining-Modus", "hint": "M", "keys": ["m"], "mode": "toggle", "color": "amber", "wide": True},
                        {"id": "mining-laser", "label": "Laser / Extraktion", "hint": "Maus Links halten", "keys": ["mouse1"], "mode": "hold", "color": "red", "danger": True, "wide": True},
                        {"id": "laser-up", "label": "Laser-Leistung +", "hint": "Mausrad Hoch", "keys": ["wheelup"], "mode": "tap", "color": "amber"},
                        {"id": "laser-down", "label": "Laser-Leistung —", "hint": "Mausrad Runter", "keys": ["wheeldown"], "mode": "tap", "color": "cyan"},
                        {"id": "mine-1", "sc": "v_mining_use_consumable1", "label": "Modul 1 (Surge)", "hint": "L-Alt + 1", "keys": ["lalt", "1"], "mode": "tap", "color": "cyan"},
                        {"id": "mine-2", "sc": "v_mining_use_consumable2", "label": "Modul 2 (Optimum)", "hint": "L-Alt + 2", "keys": ["lalt", "2"], "mode": "tap", "color": "cyan"},
                        {"id": "mine-3", "sc": "v_mining_use_consumable3", "label": "Modul 3 (Brandt)", "hint": "L-Alt + 3", "keys": ["lalt", "3"], "mode": "tap", "color": "cyan"},
                        {"id": "jettison", "sc": "v_jettison_volatile_cargo", "label": "Instabiles Erz abwerfen", "hint": "L-Alt + J", "keys": ["lalt", "j"], "mode": "tap", "color": "red", "danger": True},
                    ],
                }
            ],
        }
    if tag == "salvage":
        return {
            "id": "salvage",
            "label": "Salvage & Scraper",
            "groups": [
                {
                    "title": f"{ship_name} // Hüllen-Bergung & Fraktur",
                    "buttons": [
                        {"id": "salvage-mode", "sc": "v_toggle_salvage_mode", "label": "Salvage-Modus", "hint": "M", "keys": ["m"], "mode": "toggle", "color": "amber", "wide": True},
                        {"id": "salvage-beam", "label": "Scraper / Strahl", "hint": "Maus Links halten", "keys": ["mouse1"], "mode": "hold", "color": "red", "danger": True, "wide": True},
                        {"id": "salvage-gimbal", "sc": "v_salvage_toggle_gimbal_mode", "label": "Salvage Gimbal", "hint": "G", "keys": ["g"], "mode": "toggle", "color": "cyan"},
                        {"id": "salvage-all", "sc": "v_salvage_focus_all_heads", "label": "Fokus alle Köpfe", "hint": "L-Alt + S", "keys": ["lalt", "s"], "mode": "tap", "color": "amber"},
                        {"id": "salvage-left", "sc": "v_salvage_focus_left", "label": "Fokus Kopf links", "hint": "L-Alt + A", "keys": ["lalt", "a"], "mode": "tap", "color": "cyan"},
                        {"id": "salvage-right", "sc": "v_salvage_focus_right", "label": "Fokus Kopf rechts", "hint": "L-Alt + D", "keys": ["lalt", "d"], "mode": "tap", "color": "cyan"},
                        {"id": "salvage-fracture", "sc": "v_salvage_toggle_fire_fracture", "label": "Fraktur-Feld (Brechen)", "hint": "R-Alt + W", "keys": ["ralt", "w"], "mode": "tap", "color": "red", "wide": True},
                        {"id": "salvage-cm", "sc": "v_salvage_toggle_fire_disintegrate", "label": "Desintegration (CM)", "hint": "R-Alt + S", "keys": ["ralt", "s"], "mode": "tap", "color": "amber", "wide": True},
                    ],
                }
            ],
        }
    if tag == "cargo":
        return {
            "id": "cargo",
            "label": "Fracht & Hangar",
            "groups": [
                {
                    "title": f"{ship_name} // Rampen & ATC",
                    "buttons": [
                        {"id": "doors-all", "sc": "v_toggle_all_portlocks", "label": "Port Lock", "hint": "R-Alt + K", "keys": ["ralt", "k"], "mode": "toggle", "color": "amber", "wide": True},
                        {"id": "atc-req", "sc": "v_atc_request", "label": "Landeerlaubnis (ATC)", "hint": "L-Alt + N", "keys": ["lalt", "n"], "mode": "tap", "color": "green", "wide": True},
                        {"id": "cargo-atc", "sc": "v_atc_loading_area_request", "label": "Ladebereich anfragen", "hint": "R-Alt + N", "keys": ["ralt", "n"], "mode": "tap", "color": "cyan", "wide": True},
                        {"id": "tractor-mode", "label": "Traktorstrahl-Modus", "hint": "B / M", "keys": ["m"], "mode": "toggle", "color": "cyan"},
                    ],
                }
            ],
        }
    if tag == "medical":
        return {
            "id": "medical",
            "label": "MedBay & Rettung",
            "groups": [
                {
                    "title": f"{ship_name} // Search & Rescue (MedBay)",
                    "buttons": [
                        {"id": "med-doors", "sc": "v_toggle_all_portlocks", "label": "Port Lock", "hint": "R-Alt + K", "keys": ["ralt", "k"], "mode": "toggle", "color": "red", "wide": True},
                        {"id": "med-lights", "sc": "v_lights", "label": "Einsatz-Suchscheinwerfer", "hint": "L", "keys": ["l"], "mode": "toggle", "color": "amber", "wide": True},
                        {"id": "med-ping", "sc": "v_invoke_ping", "label": "Biometrie / Radar Ping", "hint": "Tab halten", "keys": ["tab"], "mode": "hold", "color": "cyan"},
                        {"id": "med-atc", "sc": "v_atc_request", "label": "Klinik-Landeerlaubnis", "hint": "L-Alt + N", "keys": ["lalt", "n"], "mode": "tap", "color": "green"},
                    ],
                }
            ],
        }
    if tag == "exploration":
        return {
            "id": "exploration",
            "label": "Scan & Expedition",
            "groups": [
                {
                    "title": f"{ship_name} // Tiefenraum-Sensorik & Kartografie",
                    "buttons": [
                        {"id": "exp-ping", "sc": "v_invoke_ping", "label": "Tiefenraum Radar-Ping", "hint": "Tab halten", "keys": ["tab"], "mode": "hold", "color": "cyan", "wide": True},
                        {"id": "exp-scan", "sc": "v_toggle_scan_mode", "label": "Analyse / Scan-Modus", "hint": "V", "keys": ["v"], "mode": "toggle", "color": "cyan", "wide": True},
                        {"id": "exp-map", "sc": "v_starmap", "label": "Skyline Sternkarte", "hint": "F2", "keys": ["f2"], "mode": "tap", "color": "blue"},
                        {"id": "exp-night", "label": "Nachtsicht / Restlicht", "hint": "R-Alt + L", "keys": ["ralt", "l"], "mode": "toggle", "color": "green"},
                    ],
                }
            ],
        }
    if tag == "ewar":
        return {
            "id": "ewar",
            "label": "EMP & QED / E-War",
            "groups": [
                {
                    "title": f"{ship_name} // Elektronische Kampfführung & Interdiction",
                    "buttons": [
                        {"id": "ewar-emp", "label": "EMP / QED Feld aktivieren", "hint": "Rechtsklick halten", "keys": ["mouse2"], "mode": "hold", "color": "cyan", "danger": True, "wide": True},
                        {"id": "ewar-snare", "label": "Quantum Snare / Dampener", "hint": "Linksklick halten", "keys": ["mouse1"], "mode": "hold", "color": "amber", "danger": True, "wide": True},
                        {"id": "ewar-noise", "sc": "v_weapon_countermeasure_noise_launch", "label": "Tarn-Störfeld (Noise)", "hint": "J", "keys": ["j"], "mode": "tap", "color": "cyan"},
                        {"id": "ewar-flare", "sc": "v_weapon_countermeasure_decoy_launch", "label": "Täuschkörper (Flare)", "hint": "H", "keys": ["h"], "mode": "tap", "color": "amber"},
                    ],
                }
            ],
        }
    if tag == "dropship":
        return {
            "id": "dropship",
            "label": "Dropship & Türme",
            "groups": [
                {
                    "title": f"{ship_name} // Truppenabwurf & Gefechtszone",
                    "buttons": [
                        {"id": "drop-ramps", "sc": "v_toggle_all_portlocks", "label": "Port Lock", "hint": "R-Alt + K", "keys": ["ralt", "k"], "mode": "toggle", "color": "amber", "wide": True},
                        {"id": "drop-vtol", "sc": "v_vtol_toggle", "label": "VTOL Landeschub", "hint": "K", "keys": ["k"], "mode": "toggle", "color": "cyan", "wide": True},
                        {"id": "drop-flare", "sc": "v_weapon_countermeasure_decoy_launch", "label": "Abwurf-Flares", "hint": "H", "keys": ["h"], "mode": "tap", "color": "red"},
                        {"id": "drop-noise", "sc": "v_weapon_countermeasure_noise_launch", "label": "Rauch / Noise-Schirm", "hint": "J", "keys": ["j"], "mode": "tap", "color": "cyan"},
                    ],
                }
            ],
        }
    if tag == "kampf":
        return {
            "id": "kampf",
            "label": "Kampf & Waffen",
            "groups": [
                {
                    "title": f"{ship_name} // Gefechtskontrolle",
                    "buttons": [
                        {"id": "target-hostile", "sc": "v_target_cycle_in_view_fwd", "label": "Ziel wechseln", "hint": "T", "keys": ["t"], "mode": "tap", "color": "red"},
                        {"id": "gimbal", "sc": "v_weapon_gimbals_state_toggle", "label": "Gimbal Modus", "hint": "G", "keys": ["g"], "mode": "tap", "color": "blue"},
                        {"id": "missiles", "sc": "v_operator_mode_cycle_forward", "label": "Nächster Operator-Modus", "hint": "Mittelklick", "keys": ["mouse3"], "mode": "tap", "color": "amber"},
                        {"id": "flares", "sc": "v_weapon_countermeasure_decoy_launch", "label": "Täuschkörper (Flare)", "hint": "H", "keys": ["h"], "mode": "tap", "color": "amber"},
                        {"id": "noise", "sc": "v_weapon_countermeasure_noise_launch", "label": "Störfeld (Noise)", "hint": "J", "keys": ["j"], "mode": "tap", "color": "cyan"},
                    ],
                }
            ],
        }
    if tag == "energie":
        return {
            "id": "energie",
            "label": "Energie & Schilde",
            "groups": [
                {
                    "title": f"{ship_name} // Live Kondensator & Schild-Matrix",
                    "buttons": [
                        {"id": "live-pwr", "kind": "power", "label": "Energie-Kondensator", "hint": "F5 / F6 / F7 / F8", "keys": ["f8"], "mode": "tap", "color": "amber", "wide": True},
                        {"id": "live-shd", "kind": "shield", "label": "Schild-Matrix", "hint": "Sektoren", "keys": ["num5"], "mode": "tap", "color": "cyan", "wide": True},
                        {"id": "power-all", "sc": "v_power_toggle", "label": "Hauptstrom", "hint": "U", "keys": ["u"], "mode": "toggle", "color": "green"},
                        {"id": "power-thrusters", "sc": "v_power_toggle_thrusters", "label": "Triebwerke", "hint": "I", "keys": ["i"], "mode": "toggle", "color": "amber"},
                        {"id": "power-wep", "sc": "v_power_toggle_weapons", "label": "Waffen-Power", "hint": "P", "keys": ["p"], "mode": "toggle", "color": "red"},
                        {"id": "power-shd", "sc": "v_power_toggle_shields", "label": "Schild-Power", "hint": "O", "keys": ["o"], "mode": "toggle", "color": "cyan"},
                    ],
                }
            ],
        }
    if tag == "quantum":
        return {
            "id": "quantum",
            "label": "Quantum & Nav",
            "groups": [
                {
                    "title": f"{ship_name} // Quantum Drive & Navigation",
                    "buttons": [
                        {"id": "master-mode", "sc": "v_toggle_quantum_mode", "label": "Master Mode (SCM/NAV)", "hint": "B halten", "keys": ["b"], "mode": "hold", "color": "amber", "wide": True},
                        {"id": "starmap", "sc": "v_starmap", "label": "Skyline Starmap", "hint": "F2", "keys": ["f2"], "mode": "tap", "color": "blue"},
                        {"id": "mobiglas", "sc": "mobiglas", "label": "mobiGlas", "hint": "F1", "keys": ["f1"], "mode": "tap", "color": "blue"},
                        {"id": "ping", "sc": "v_invoke_ping", "label": "Radar Ping", "hint": "Tab halten", "keys": ["tab"], "mode": "hold", "color": "cyan", "wide": True},
                    ],
                }
            ],
        }
    # Default: "flug"
    return {
        "id": "flug",
        "label": "Flug & Systeme",
        "groups": [
            {
                "title": f"{mfg} // Flugsteuerung & Landung",
                "buttons": [
                    {"id": "flight-ready", "sc": "v_flightready", "label": "Flight Ready", "hint": "R-Alt + R", "keys": ["ralt", "r"], "mode": "tap", "color": "green", "wide": True},
                    {"id": "gear", "sc": "v_toggle_landing_system", "label": "Fahrwerk", "hint": "N", "keys": ["n"], "mode": "toggle", "color": "cyan"},
                    {"id": "vtol", "sc": "v_vtol_toggle", "label": "VTOL Düsen", "hint": "K", "keys": ["k"], "mode": "toggle", "color": "cyan"},
                    {"id": "decoupled", "sc": "v_ifcs_vector_decoupling_toggle", "label": "Decoupled", "hint": "C", "keys": ["c"], "mode": "toggle", "color": "blue"},
                    {"id": "cruise", "sc": "v_ifcs_toggle_cruise_control", "label": "Tempomat (Cruise)", "hint": "L-Alt + C", "keys": ["lalt", "c"], "mode": "toggle", "color": "blue"},
                    {"id": "lights", "sc": "v_lights", "label": "Scheinwerfer", "hint": "L", "keys": ["l"], "mode": "toggle", "color": "amber"},
                    {"id": "doors", "sc": "v_toggle_all_portlocks", "label": "Port Lock", "hint": "R-Alt + K", "keys": ["ralt", "k"], "mode": "toggle", "color": "slate"},
                    {"id": "atc", "sc": "v_atc_request", "label": "Landeerlaubnis (ATC)", "hint": "L-Alt + N", "keys": ["lalt", "n"], "mode": "tap", "color": "amber", "wide": True},
                ],
            }
        ],
    }


def generate_ship_dashboard(entry: tuple) -> dict:
    ship_id, ship_name, mfg, role, hull_type, tags = entry
    pages = [_build_page_for_tag(t, ship_name, mfg) for t in tags]
    bg_uri = make_ship_svg_data_uri(ship_name, mfg, role, hull_type)
    return {
        "id": ship_id,
        "title": ship_name,
        "ship": ship_name,
        "manufacturer": mfg,
        "role": role,
        "hull_type": hull_type,
        "subsystems": list(tags),
        "background": bg_uri,
        "bg_opacity": 0.48,
        "pages": pages,
    }


def all_generated_ship_dashboards() -> list[dict]:
    return [generate_ship_dashboard(e) for e in SC_SHIPS_DB]
