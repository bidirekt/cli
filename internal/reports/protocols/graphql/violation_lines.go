package graphql

import (
	"fmt"
	"strings"

	"github.com/bidirekt/cli/internal/reports"
)

func (this Plugin) ViolationLine(violation reports.Violation) (headline, explanation string, known bool) {
	details := violation.Details
	pathAt := reports.At(reports.Location(violation.Path, ""))
	graphAt := reports.At(reports.Location(violation.Path, "graph"))

	switch violation.Code {
	case "schema.scalar_not_allowed":
		return `type "scalar" is only allowed in GraphQL contracts` + reports.At(reports.Location(violation.Path, "type")), "", true
	case "schema.arguments_not_allowed":
		return `"arguments" is only allowed in GraphQL contracts` + reports.At(reports.Location(violation.Path, "arguments")), "", true
	case "schema.arguments_not_object":
		argumentsAt := reports.At(reports.Location(violation.Path, "arguments"))
		switch details["reason"] {
		case "not_object":
			return fmt.Sprintf(`"arguments" must name an object schema%s: %s is %s`, argumentsAt, details["schema"], details["type"]), "", true
		case "nullable":
			return fmt.Sprintf(`"arguments" must name a non-nullable schema%s: %s is nullable`, argumentsAt, details["schema"]), "", true
		}
		return "", "", false
	case "schema.scalar_without_name":
		return "scalar schema without name" + pathAt, "", true
	case "returns.required":
		return `missing "returns"` + pathAt, "", true
	case "graph.without_schema":
		return fmt.Sprintf("graph %q without a GraphQL schema%s", details["graph"], graphAt), "publish its " + schemaFileExtensionsText() + " in the same publish", true
	case "graph.required_by_federation":
		return "the GraphQL schema links Apollo Federation, but no graph is declared" + pathAt, "declare it in provides.graphql.graph", true
	case "graph.name_taken":
		return fmt.Sprintf("graph name %q is already a participant name%s", details["graph"], graphAt), "pick another graph name", true
	case "graph.name_syntax":
		return fmt.Sprintf("invalid graph name %q%s", details["graph"], graphAt), details["error"], true
	case "graph.conflict":
		return fmt.Sprintf("graph %q conflicts with graph %q declared in %s%s", details["graph"], details["declared"], details["declaredIn"], graphAt),
			"a contract belongs to one graph: declare the same name in every file", true
	case "operation.name_syntax":
		return fmt.Sprintf("invalid GraphQL field name %q%s", details["name"], reports.At(reports.Location(violation.Path, details["name"]))),
			`write the field name only; arguments go in "arguments"`, true
	default:
		return "", "", false
	}
}

func schemaFileExtensionsText() string {
	last := len(schemaFileExtensions) - 1

	return strings.Join(schemaFileExtensions[:last], ", ") + " or " + schemaFileExtensions[last]
}
