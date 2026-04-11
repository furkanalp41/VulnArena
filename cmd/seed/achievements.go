package main

type achievementSeed struct {
	slug        string
	name        string
	description string
	iconSVG     string
	category    string
	xpReward    int
}

func buildAchievements() []achievementSeed {
	return []achievementSeed{
		{
			slug:        "first-blood-spiller",
			name:        "First Blood Spiller",
			description: "Draw first blood — be the absolute first to pwn any challenge.",
			category:    "special",
			xpReward:    50,
			iconSVG: `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100" fill="none">
  <circle cx="50" cy="50" r="46" stroke="#ff4444" stroke-width="3" fill="#1a0a0a"/>
  <circle cx="50" cy="50" r="38" stroke="#ff6644" stroke-width="1" stroke-dasharray="4 3" fill="none"/>
  <path d="M50 20 C50 20 62 38 62 50 C62 62 56 68 50 72 C44 68 38 62 38 50 C38 38 50 20 50 20Z" fill="#ff4444" opacity="0.9"/>
  <path d="M50 28 C50 28 58 40 58 50 C58 58 54 63 50 66 C46 63 42 58 42 50 C42 40 50 28 50 28Z" fill="#ff6644" opacity="0.7"/>
  <path d="M50 36 C50 36 54 43 54 50 C54 54 52 57 50 59 C48 57 46 54 46 50 C46 43 50 36 50 36Z" fill="#ffaa44" opacity="0.8"/>
  <text x="50" y="88" text-anchor="middle" font-family="monospace" font-size="7" font-weight="bold" fill="#ff4444">1ST BLOOD</text>
</svg>`,
		},
		{
			slug:        "sqli-master",
			name:        "SQLi Master",
			description: "Master of injection — solve 3 or more injection-category challenges.",
			category:    "mastery",
			xpReward:    75,
			iconSVG: `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100" fill="none">
  <circle cx="50" cy="50" r="46" stroke="#00e5ff" stroke-width="3" fill="#0a1a1a"/>
  <circle cx="50" cy="50" r="38" stroke="#00e5ff" stroke-width="1" stroke-dasharray="4 3" fill="none"/>
  <rect x="25" y="28" width="50" height="36" rx="3" stroke="#00e5ff" stroke-width="2" fill="#0d2d2d"/>
  <text x="50" y="42" text-anchor="middle" font-family="monospace" font-size="8" fill="#00e5ff">SELECT *</text>
  <text x="50" y="52" text-anchor="middle" font-family="monospace" font-size="8" fill="#00ffaa">FROM pwned</text>
  <text x="50" y="60" text-anchor="middle" font-family="monospace" font-size="7" fill="#ff4444">' OR 1=1--</text>
  <line x1="25" y1="68" x2="75" y2="68" stroke="#00e5ff" stroke-width="1"/>
  <text x="50" y="88" text-anchor="middle" font-family="monospace" font-size="7" font-weight="bold" fill="#00e5ff">SQLi MASTER</text>
</svg>`,
		},
		{
			slug:        "night-owl",
			name:        "Night Owl",
			description: "Hack the night — solve a challenge between midnight and 5 AM.",
			category:    "dedication",
			xpReward:    25,
			iconSVG: `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100" fill="none">
  <circle cx="50" cy="50" r="46" stroke="#aa55ff" stroke-width="3" fill="#0d0a1a"/>
  <circle cx="50" cy="50" r="38" stroke="#aa55ff" stroke-width="1" stroke-dasharray="4 3" fill="none"/>
  <circle cx="58" cy="38" r="16" fill="#1a1030"/>
  <circle cx="65" cy="32" r="14" fill="#0d0a1a"/>
  <circle cx="35" cy="55" r="2" fill="#ffcc00" opacity="0.8"/>
  <circle cx="60" cy="62" r="1.5" fill="#ffcc00" opacity="0.6"/>
  <circle cx="42" cy="35" r="1" fill="#ffcc00" opacity="0.5"/>
  <circle cx="70" cy="50" r="1.5" fill="#ffcc00" opacity="0.4"/>
  <circle cx="30" cy="45" r="1" fill="#ffcc00" opacity="0.7"/>
  <text x="50" y="80" text-anchor="middle" font-family="monospace" font-size="8" fill="#aa55ff">00:00-05:00</text>
  <text x="50" y="88" text-anchor="middle" font-family="monospace" font-size="7" font-weight="bold" fill="#aa55ff">NIGHT OWL</text>
</svg>`,
		},
		{
			slug:        "persistence",
			name:        "Persistence",
			description: "Relentless operator — maintain a 7-day consecutive solving streak.",
			category:    "dedication",
			xpReward:    50,
			iconSVG: `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100" fill="none">
  <circle cx="50" cy="50" r="46" stroke="#00ff88" stroke-width="3" fill="#0a1a0d"/>
  <circle cx="50" cy="50" r="38" stroke="#00ff88" stroke-width="1" stroke-dasharray="4 3" fill="none"/>
  <g transform="translate(26, 32)">
    <rect x="0"  y="16" width="6" height="8"  rx="1" fill="#00ff88" opacity="0.3"/>
    <rect x="8"  y="12" width="6" height="12" rx="1" fill="#00ff88" opacity="0.4"/>
    <rect x="16" y="8"  width="6" height="16" rx="1" fill="#00ff88" opacity="0.5"/>
    <rect x="24" y="6"  width="6" height="18" rx="1" fill="#00ff88" opacity="0.6"/>
    <rect x="32" y="4"  width="6" height="20" rx="1" fill="#00ff88" opacity="0.7"/>
    <rect x="40" y="2"  width="6" height="22" rx="1" fill="#00ff88" opacity="0.85"/>
    <rect x="48" y="0"  width="6" height="24" rx="1" fill="#00ff88" opacity="1.0"/>
  </g>
  <text x="50" y="80" text-anchor="middle" font-family="monospace" font-size="8" fill="#00ff88">7 DAYS</text>
  <text x="50" y="88" text-anchor="middle" font-family="monospace" font-size="7" font-weight="bold" fill="#00ff88">PERSISTENCE</text>
</svg>`,
		},
		{
			slug:        "pentester",
			name:        "Pentester",
			description: "Seasoned operator — solve 10 or more challenges in the Arena.",
			category:    "combat",
			xpReward:    100,
			iconSVG: `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 100 100" fill="none">
  <circle cx="50" cy="50" r="46" stroke="#ffcc00" stroke-width="3" fill="#1a1500"/>
  <circle cx="50" cy="50" r="38" stroke="#ffcc00" stroke-width="1" stroke-dasharray="4 3" fill="none"/>
  <path d="M50 22 L56 38 L73 38 L59 48 L64 64 L50 54 L36 64 L41 48 L27 38 L44 38 Z" fill="none" stroke="#ffcc00" stroke-width="2"/>
  <path d="M50 28 L54 39 L66 39 L57 46 L60 57 L50 50 L40 57 L43 46 L34 39 L46 39 Z" fill="#ffcc00" opacity="0.3"/>
  <text x="50" y="52" text-anchor="middle" font-family="monospace" font-size="10" font-weight="bold" fill="#ffcc00">10</text>
  <text x="50" y="80" text-anchor="middle" font-family="monospace" font-size="7" fill="#ffcc00">CHALLENGES</text>
  <text x="50" y="88" text-anchor="middle" font-family="monospace" font-size="7" font-weight="bold" fill="#ffcc00">PENTESTER</text>
</svg>`,
		},
	}
}
