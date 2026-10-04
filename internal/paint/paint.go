package paint

import (
	"io"
	"os"

	"github.com/bidirekt/cli/internal/components"
)

const (
	green = "\x1b[32m"
	red   = "\x1b[31m"
	reset = "\x1b[0m"
)

type Brush struct {
	enabled bool
}

func For(writer io.Writer) Brush {
	return Brush{enabled: colorEnabledFor(writer)}
}

func (this Brush) Green(text string) string {
	return this.wrap(green, text)
}

func (this Brush) Red(text string) string {
	return this.wrap(red, text)
}

func (this Brush) wrap(color, text string) string {
	if !this.enabled {
		return text
	}
	return color + text + reset
}

func colorEnabledFor(writer io.Writer) bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	if force := os.Getenv("CLICOLOR_FORCE"); force != "" && force != "0" {
		return true
	}
	return components.IsTerminal(writer)
}
