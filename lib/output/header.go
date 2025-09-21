package output

import (
	"fmt"
	"os"
	"strings"
)

const banner = `
 ______     ______     __   __     ______   __  __     ______     __     ______     __   __
/\  ___\   /\  ___\   /\ "-.\ \   /\__  _\ /\ \/\ \   /\  == \   /\ \   /\  __ \   /\ "-.\ \
\ \ \____  \ \  __\   \ \ \-.  \  \/_/\ \/ \ \ \_\ \  \ \  __<   \ \ \  \ \ \/\ \  \ \ \-.  \
 \ \_____\  \ \_____\  \ \_\\"\_\    \ \_\  \ \_____\  \ \_\ \_\  \ \_\  \ \_____\  \ \_\\"\_\
  \/_____/   \/_____/   \/_/ \/_/     \/_/   \/_____/   \/_/ /_/   \/_/   \/_____/   \/_/ \/_/
`

func colorGradient(text string) string {
	lines := strings.Split(text, "\n")
	coloredLines := make([]string, len(lines))

	colors := []string{
		Maroon,    // Deepest dark red at top
		DarkRed,   // Dark red
		Red,       // Your existing red
		Crimson,   // Rich crimson
		LightRed,  // Your existing light red
		BrightRed, // Vivid red at bottom
	}

	for i, line := range lines {
		if strings.TrimSpace(line) == "" {
			coloredLines[i] = line
			continue
		}

		colorIndex := (i * len(colors)) / len(lines)
		color := colors[colorIndex]
		coloredLines[i] = color + line + Reset
	}

	return strings.Join(coloredLines, "\n")
}

func supportsColor() bool {
	_, exists := os.LookupEnv("TERM")
	result := exists && os.Getenv("TERM") != "dumb"

	if _, exists := os.LookupEnv("NO_COLOR"); exists {
		result = false
	}

	return result
}

func PrintBanner(version string) string {
	var text string
	if !supportsColor() {
		text = fmt.Sprintf("%s\nVersion: %s\n", banner, version)
		return text
	}

	coloredBanner := colorGradient(banner)
	text = fmt.Sprintf("%s\nVersion: %s%s%s\n", coloredBanner, LightCyan, version, Reset)

	return text
}
