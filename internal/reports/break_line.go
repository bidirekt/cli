package reports

import (
	"fmt"
	"sort"
	"strings"
)

type BreakLine struct {
	Text  string
	Fixes []string
}

func WriteBreakLine(report *strings.Builder, indent string, line BreakLine) {
	fmt.Fprintf(report, "%s- %s\n", indent, line.Text)
	for _, fix := range line.Fixes {
		fmt.Fprintf(report, "%s  → %s\n", indent, fix)
	}
}

// FallbackBreakLine renders a break verbatim with its details when no template of its protocol
// matches it, so the CLI never swallows a break.
func FallbackBreakLine(reason string, details map[string]string) string {
	if len(details) == 0 {
		return reason
	}

	keys := make([]string, 0, len(details))
	for key := range details {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	pairs := make([]string, 0, len(keys))
	for _, key := range keys {
		pairs = append(pairs, key+": "+details[key])
	}

	return fmt.Sprintf("%s (%s)", reason, strings.Join(pairs, ", "))
}
