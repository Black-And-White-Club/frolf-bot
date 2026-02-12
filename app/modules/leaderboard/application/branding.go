package leaderboardservice

import "image/color"

// ChartPalette defines the color palette for chart generation.
type ChartPalette struct {
	Background  color.RGBA // Background color
	GridLines   color.RGBA // Grid line color
	PrimaryLine color.RGBA // Primary data line color
	TextColor   color.RGBA // Text/label color
	AccentLine  color.RGBA // Accent/secondary line color
}

// ObsidianForestPalette is the default chart palette matching the PWA Obsidian Forest theme.
var ObsidianForestPalette = ChartPalette{
	Background:  color.RGBA{10, 22, 40, 255},    // #0a1628 → Deep Navy
	GridLines:   color.RGBA{26, 42, 63, 255},    // #1a2a3f → Muted blue-gray
	PrimaryLine: color.RGBA{197, 160, 78, 255},  // #c5a04e → Gold Accent
	TextColor:   color.RGBA{226, 232, 240, 255}, // #e2e8f0 → Off-white
	AccentLine:  color.RGBA{139, 92, 246, 255},  // #8b5cf6 → Amethyst Aura
}
