package output

import (
	"fmt"
	"os"
	"strings"
)

const banner = `
 __     __    __     ______   ______     ______     __     __  __     __    __
/\ \   /\ "-./  \   /\  == \ /\  ___\   /\  == \   /\ \   /\ \/\ \   /\ "-./  \
\ \ \  \ \ \-./\ \  \ \  _-/ \ \  __\   \ \  __<   \ \ \  \ \ \_\ \  \ \ \-./\ \
 \ \_\  \ \_\ \ \_\  \ \_\    \ \_____\  \ \_\ \_\  \ \_\  \ \_____\  \ \_\ \ \_\
  \/_/   \/_/  \/_/   \/_/     \/_____/   \/_/ /_/   \/_/   \/_____/   \/_/  \/_/
`

func colorGradient(text string) string {
	lines := strings.Split(text, "\n")
	coloredLines := make([]string, len(lines))

	colors := []string{
		Blue,
		LightBlue,
		Cyan,
		LightCyan,
		Purple,
		LightPurple,
	}

	for i, line := range lines {
		if strings.TrimSpace(line) == "" {
			coloredLines[i] = line
			continue
		}

		color_index := (i * len(colors)) / len(lines)
		color := colors[color_index]
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
