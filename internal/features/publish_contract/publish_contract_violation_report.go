package publish_contract

import (
	"fmt"
	"sort"
	"strings"
)

func formatValidationFailedReport(message string, violations []Violation) string {
	var report strings.Builder
	fmt.Fprintf(&report, "❌ %s\n", message)

	for _, violation := range violations {
		headline, explanation := formatViolationLine(violation)

		report.WriteString("  - ")
		if violation.Source != "" {
			report.WriteString(violation.Source + ": ")
		}
		report.WriteString(headline + "\n")

		if explanation != "" {
			report.WriteString("      " + explanation + "\n")
		}
	}

	return report.String()
}

func formatViolationLine(violation Violation) (headline, explanation string) {
	details := violation.Details
	path := violation.Path

	switch violation.Code {
	case "key.unknown":
		return fmt.Sprintf("unknown key %q at %s", details["key"], path), ""
	case "value.invalid_kind":
		return fmt.Sprintf("unexpected %s at %s, expected %s", details["got"], path, details["expected"]), ""
	case "endpoint.syntax":
		return fmt.Sprintf("invalid endpoint %q at %s", details["key"], path), details["error"]
	case "service.name_syntax":
		return fmt.Sprintf("invalid service name %q at %s", details["key"], path), details["error"]
	case "status.out_of_range":
		return fmt.Sprintf("invalid status code %q at %s", details["key"], path), details["error"]
	case "schema.invalid_type":
		if details["value"] == "" {
			return fmt.Sprintf(`missing "type" at %s`, path), "expected one of: " + details["allowed"]
		}
		return fmt.Sprintf(`invalid value %q for "type" at %s`, details["value"], path), "expected one of: " + details["allowed"]
	case "schema.array_without_items":
		return "array schema without items at " + path, ""
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
	line := violation.Code
	if violation.Path != "" {
		line += " at " + violation.Path
	}

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
