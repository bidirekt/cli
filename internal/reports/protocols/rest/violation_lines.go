package rest

import (
	"fmt"
	"strings"

	"github.com/bidirekt/cli/internal/reports"
)

func (this Plugin) ViolationLine(violation reports.Violation) (headline, explanation string, known bool) {
	details := violation.Details

	switch violation.Code {
	case "endpoint.syntax":
		headline = fmt.Sprintf("invalid endpoint %q%s", details["key"], endpointOwner(violation.Path, details["key"]))
		if suggestion := details["suggestion"]; suggestion != "" {
			return headline, fmt.Sprintf("write %q (%s)", suggestion, details["error"]), true
		}
		return headline, details["error"], true
	case "status.out_of_range":
		return fmt.Sprintf("invalid status code %q%s", details["key"], reports.At(reports.Location(violation.Path, details["key"]))), details["error"], true
	default:
		return "", "", false
	}
}

func endpointOwner(path, key string) string {
	owner, isRest := strings.CutSuffix(path, ";rest;"+key)
	service, isConsumes := strings.CutPrefix(owner, "consumes;")

	switch {
	case isRest && owner == "provides":
		return " in provides"
	case isRest && isConsumes && service != "" && !strings.Contains(service, ";"):
		return " in consumes " + service
	default:
		return reports.At(reports.Location(path, key))
	}
}
