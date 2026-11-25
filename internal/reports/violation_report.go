package reports

import (
	"cmp"
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/bidirekt/cli/internal/paint"
)

type Violation struct {
	Code    string            `json:"code"`
	Path    string            `json:"path"`
	Source  string            `json:"source"`
	Details map[string]string `json:"details"`
	// Details that hold a list; the broker sends them in "details" too, beside the text ones.
	ListDetails map[string][]string `json:"-"`
}

func (this *Violation) UnmarshalJSON(data []byte) error {
	type violationFields Violation
	var wire struct {
		violationFields
		Details map[string]json.RawMessage `json:"details"`
	}
	if err := json.Unmarshal(data, &wire); err != nil {
		return err
	}

	*this = Violation(wire.violationFields)
	if wire.Details != nil {
		this.Details = map[string]string{}
	}
	for key, value := range wire.Details {
		var text string
		if json.Unmarshal(value, &text) == nil {
			this.Details[key] = text
			continue
		}

		var list []string
		if err := json.Unmarshal(value, &list); err != nil {
			return fmt.Errorf("violation detail %q is neither text nor a list of text: %s", key, value)
		}
		if this.ListDetails == nil {
			this.ListDetails = map[string][]string{}
		}
		this.ListDetails[key] = list
	}

	return nil
}

func FormatValidationFailedReport(brush paint.Brush, message string, violations []Violation, modules []Module) string {
	var report strings.Builder
	report.WriteString(brush.Red(message) + "\n")

	for _, violation := range violations {
		headline, explanation := FormatViolationLine(violation, modules)

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

func FormatViolationLine(violation Violation, modules []Module) (headline, explanation string) {
	details := violation.Details
	keyAt := At(Location(violation.Path, details["key"]))
	typeAt := At(Location(violation.Path, "type"))
	pathAt := At(Location(violation.Path, ""))

	switch violation.Code {
	case "key.unknown":
		return fmt.Sprintf("unknown key %q%s", details["key"], keyAt), ""
	case "value.invalid_kind":
		return fmt.Sprintf("unexpected %s%s, expected %s", details["got"], pathAt, details["expected"]), ""
	case "service.name_syntax":
		return fmt.Sprintf("invalid service name %q%s", details["key"], keyAt), details["error"]
	case "property.name_syntax":
		return fmt.Sprintf("invalid property name %q%s", details["key"], keyAt), details["error"]
	case "schema.name_syntax":
		return fmt.Sprintf("invalid schema name %q%s", details["key"], keyAt), details["error"]
	case "schema.description_syntax":
		return fmt.Sprintf("invalid description %q%s", details["value"], At(Location(violation.Path, "description"))), details["error"]
	case "schema.invalid_type":
		if details["value"] == "" {
			return `missing "type"` + typeAt, "expected one of: " + details["allowed"]
		}
		return fmt.Sprintf(`invalid value %q for "type"%s`, details["value"], typeAt), "expected one of: " + details["allowed"]
	case "schema.array_without_items":
		return "array schema without items" + pathAt, ""
	case "schema.map_without_values":
		return "map schema without values" + pathAt, ""
	case "schema.object_without_properties":
		return "object schema without properties" + pathAt, `declare the keys you read, or use "type: map" with "values" if the keys vary`
	case "schema.key_not_allowed":
		if details["type"] == "ref" {
			return fmt.Sprintf(`%q is not allowed beside "ref"%s`, details["key"], pathAt), ""
		}
		if rightTypes := typesOwning(details["key"], modules); len(rightTypes) > 0 {
			return fmt.Sprintf(`%q is not allowed on %s schema%s`, details["key"], withArticle(details["type"]), pathAt),
				"use it with " + strings.Join(rightTypes, " or ")
		}
		return fallbackViolationLine(violation), ""
	case "schema.optional_not_allowed":
		switch details["position"] {
		case "items":
			return `"optional" is not allowed on array items` + pathAt,
				`mark the property that holds the array optional, or use "nullable: true" for an item that may be null`
		case "values":
			return `"optional" is not allowed on map values` + pathAt,
				`mark the property that holds the map optional, or use "nullable: true" for a value that may be null`
		case "schema":
			return `"optional" is not allowed on a schema` + pathAt, `mark optional the property that uses it, beside its "ref"`
		}
		return fallbackViolationLine(violation), ""
	case "resource.duplicate":
		if details["declaredIn"] == violation.Source {
			return fmt.Sprintf("duplicate resource %q, declared twice", details["resource"]), ""
		}
		return fmt.Sprintf("duplicate resource %q, also declared in %s", details["resource"], details["declaredIn"]), ""
	case "resource.type_conflict":
		if details["values"] != "" {
			return fmt.Sprintf("conflicting values for property %q of %s: [%s] here, [%s] in %s",
				details["property"], details["resource"], details["values"], details["declaredValues"], details["declaredIn"]), ""
		}
		if details["arguments"] != "" || details["declaredArguments"] != "" {
			return fmt.Sprintf("conflicting arguments for property %q of %s: %s here, %s in %s",
				details["property"], details["resource"], cmp.Or(details["arguments"], "none"), cmp.Or(details["declaredArguments"], "none"), details["declaredIn"]), ""
		}
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
	case "schema.ref_cycle":
		cycle := violation.ListDetails["schemas"]
		if len(cycle) == 0 {
			return fallbackViolationLine(violation), ""
		}
		return fmt.Sprintf(`schema %q only refers to itself through ref (%s): give one of them a type`, cycle[0], strings.Join(slices.Concat(cycle, cycle[:1]), " → ")), ""
	case "members.required":
		if details["type"] == "union" {
			return `"type: union" needs "members"` + pathAt + `: map each key to a ref`, ""
		}
		return `"type: enum" needs "members"` + pathAt + `: list its values`, ""
	case "members.empty":
		membersAt := At(Location(violation.Path, "members"))
		if details["type"] == "union" {
			return `"members" is empty` + membersAt + `: list at least one member`, ""
		}
		return `"members" is empty` + membersAt + `: list at least one value`, ""
	case "members.duplicate":
		return fmt.Sprintf(`value %q is repeated in "members"%s`, details["value"], membersOwnerAt(violation.Path, lastSegment(violation.Path))), ""
	case "members.not_text":
		return `each value in "members" must be non-empty text` + membersOwnerAt(violation.Path, lastSegment(violation.Path)), ""
	case "member.ref_required":
		return fmt.Sprintf(`member %q needs a "ref" to a schema%s`, details["member"], membersOwnerAt(violation.Path, details["member"])), ""
	case "member.name_syntax":
		return fmt.Sprintf(`member key %q %s%s`, details["key"], details["error"], membersOwnerAt(violation.Path, details["key"])), ""
	default:
		for _, module := range modules {
			if headline, explanation, known := module.Report.ViolationLine(violation); known {
				return headline, explanation
			}
		}
		return fallbackViolationLine(violation), ""
	}
}

var typesOwningKey = map[string][]string{
	"properties": {`"type: object"`},
	"items":      {`"type: array"`},
	"values":     {`"type: map"`},
	"members":    {`"type: enum"`, `"type: union"`},
}

func typesOwning(key string, modules []Module) []string {
	types := typesOwningKey[key]
	for _, module := range modules {
		types = slices.Concat(types, module.Report.SchemaKeys()[key])
	}
	return types
}

func membersOwnerAt(path, entry string) string {
	return At(Location(strings.TrimSuffix(path, ";members;"+entry), ""))
}

func lastSegment(path string) string {
	return path[strings.LastIndex(path, ";")+1:]
}

func withArticle(schemaType string) string {
	// "union" starts with a vowel letter but not a vowel sound.
	if strings.IndexAny(schemaType, "aeio") == 0 {
		return "an " + schemaType
	}

	return "a " + schemaType
}

func fallbackViolationLine(violation Violation) string {
	line := violation.Code + At(Location(violation.Path, ""))

	values := make(map[string]string, len(violation.Details)+len(violation.ListDetails))
	maps.Copy(values, violation.Details)
	for key, list := range violation.ListDetails {
		values[key] = "[" + strings.Join(list, ", ") + "]"
	}

	if len(values) == 0 {
		return line
	}

	pairs := make([]string, 0, len(values))
	for _, key := range slices.Sorted(maps.Keys(values)) {
		pairs = append(pairs, key+": "+values[key])
	}

	return fmt.Sprintf("%s (%s)", line, strings.Join(pairs, ", "))
}

func Location(path, quoted string) string {
	if path == quoted {
		return ""
	}

	return strings.ReplaceAll(strings.TrimSuffix(path, ";"+quoted), ";", " ")
}

func At(location string) string {
	if location == "" {
		return ""
	}

	return " at " + location
}
