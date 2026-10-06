package reports

import (
	"fmt"
	"sort"
	"strings"

	"github.com/bidirekt/cli/internal/paint"
)

type Violation struct {
	Code    string            `json:"code"`
	Path    string            `json:"path"`
	Source  string            `json:"source"`
	Details map[string]string `json:"details"`
}

func FormatValidationFailedReport(brush paint.Brush, message string, violations []Violation) string {
	var report strings.Builder
	report.WriteString(brush.Red(message) + "\n")

	for _, violation := range violations {
		headline, explanation := FormatViolationLine(violation)

		report.WriteString("  - ")
		if violation.Source != "" {
			report.WriteString(violation.Source + ": ")
		}
		report.WriteString(headline)
		if explanation != "" {
			report.WriteString(", " + explanation)
		}
		report.WriteString("\n")
	}

	return report.String()
}

func FormatViolationLine(violation Violation) (headline, explanation string) {
	details := violation.Details
	keyAt := at(location(violation.Path, details["key"]))
	typeAt := at(location(violation.Path, "type"))
	pathAt := at(location(violation.Path, ""))

	switch violation.Code {
	case "key.unknown":
		return fmt.Sprintf("unknown key %q%s", details["key"], keyAt), ""
	case "value.invalid_kind":
		return fmt.Sprintf("unexpected %s%s, expected %s", details["got"], pathAt, details["expected"]), ""
	case "endpoint.syntax":
		return fmt.Sprintf("invalid endpoint %q%s", details["key"], keyAt), details["error"]
	case "service.name_syntax":
		return fmt.Sprintf("invalid service name %q%s", details["key"], keyAt), details["error"]
	case "status.out_of_range":
		return fmt.Sprintf("invalid status code %q%s", details["key"], keyAt), details["error"]
	case "schema.invalid_type":
		if details["value"] == "" {
			return `missing "type"` + typeAt, "expected one of: " + details["allowed"]
		}
		return fmt.Sprintf(`invalid value %q for "type"%s`, details["value"], typeAt), "expected one of: " + details["allowed"]
	case "schema.array_without_items":
		return "array schema without items" + pathAt, ""
	case "resource.duplicate":
		if details["declaredIn"] == violation.Source {
			return fmt.Sprintf("duplicate resource %q, declared twice", details["resource"]), ""
		}
		return fmt.Sprintf("duplicate resource %q, also declared in %s", details["resource"], details["declaredIn"]), ""
	case "resource.type_conflict":
		return fmt.Sprintf("conflicting type for property %q of %s: %s here, %s in %s",
			details["property"], details["resource"], details["type"], details["declaredType"], details["declaredIn"]), ""
	case "schema.duplicate":
		if details["declaredIn"] == violation.Source {
			return fmt.Sprintf("duplicate schema %q, declared twice", details["schema"]), ""
		}
		return fmt.Sprintf("duplicate schema %q, also declared in %s", details["schema"], details["declaredIn"]), ""
	case "schema.unresolved_name":
		return fmt.Sprintf("unresolved schema %q referenced by %s", details["schema"], details["resource"]), ""
	case "schema.unresolved_ref":
		return fmt.Sprintf("unresolved ref %q in %s", details["schema"], details["property"]), ""
	case "schema.too_deep":
		return fmt.Sprintf("schema %q is deeper than %s levels", details["schema"], details["maxDepth"]), ""
	default:
		return fallbackViolationLine(violation), ""
	}
}

func fallbackViolationLine(violation Violation) string {
	line := violation.Code + at(location(violation.Path, ""))

	if len(violation.Details) == 0 {
		return line
	}

	keys := make([]string, 0, len(violation.Details))
	for key := range violation.Details {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	pairs := make([]string, 0, len(keys))
	for _, key := range keys {
		pairs = append(pairs, key+": "+violation.Details[key])
	}

	return fmt.Sprintf("%s (%s)", line, strings.Join(pairs, ", "))
}

func location(path, quoted string) string {
	if path == "" {
		return ""
	}

	segments := strings.Split(path, ";")
	if segments[len(segments)-1] == quoted {
		segments = segments[:len(segments)-1]
	}

	return strings.Join(segments, " ")
}

func at(location string) string {
	if location == "" {
		return ""
	}

	return " at " + location
}
