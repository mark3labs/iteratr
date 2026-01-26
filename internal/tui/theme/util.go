package theme

import (
	"fmt"
	"image/color"
	"strings"

	lipgloss "charm.land/lipgloss/v2"
)

// ApplyGradient applies a color gradient from colorA to colorB across the string.
// Uses ~8 color stops for performance rather than per-character coloring.
func ApplyGradient(text, colorA, colorB string) string {
	runes := []rune(text)
	if len(runes) == 0 {
		return text
	}

	// Use ~8 color stops for performance
	const maxStops = 8
	segmentSize := len(runes) / maxStops
	if segmentSize < 1 {
		segmentSize = 1
	}

	var result strings.Builder
	for i := 0; i < len(runes); i += segmentSize {
		end := i + segmentSize
		if end > len(runes) {
			end = len(runes)
		}

		pos := float64(i) / float64(len(runes))
		color := InterpolateColor(colorA, colorB, pos)
		style := lipgloss.NewStyle().Foreground(lipgloss.Color(color))
		result.WriteString(style.Render(string(runes[i:end])))
	}

	return result.String()
}

// InterpolateColor blends between two hex colors based on position (0.0 to 1.0)
func InterpolateColor(colorA, colorB string, pos float64) string {
	// Parse hex colors (format: #RRGGBB)
	r1, g1, b1 := ParseHexColor(colorA)
	r2, g2, b2 := ParseHexColor(colorB)

	// Interpolate each channel
	r := uint8(float64(r1)*(1-pos) + float64(r2)*pos)
	g := uint8(float64(g1)*(1-pos) + float64(g2)*pos)
	b := uint8(float64(b1)*(1-pos) + float64(b2)*pos)

	// Return as hex color string
	return FormatHexColor(r, g, b)
}

// ParseHexColor extracts RGB values from hex color string
func ParseHexColor(hex string) (uint8, uint8, uint8) {
	// Remove # prefix if present
	if len(hex) > 0 && hex[0] == '#' {
		hex = hex[1:]
	}

	// Parse RGB values
	var r, g, b uint8
	if len(hex) == 6 {
		_, _ = fmt.Sscanf(hex, "%02x%02x%02x", &r, &g, &b)
	}

	return r, g, b
}

// HexToColor converts a hex color string to a color.Color
func HexToColor(hex string) color.Color {
	r, g, b := ParseHexColor(hex)
	return color.RGBA{R: r, G: g, B: b, A: 255}
}

// FormatHexColor converts RGB values to hex color string
func FormatHexColor(r, g, b uint8) string {
	return fmt.Sprintf("#%02x%02x%02x", r, g, b)
}
