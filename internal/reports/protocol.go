package reports

import (
	"encoding/json"
	"strings"
)

type Protocol interface {
	ReportKey() string
	WriteTree(report *strings.Builder, counterpart Counterpart, tree json.RawMessage) error
	ViolationLine(violation Violation) (headline, explanation string, known bool)
	// Each schema key maps to the types it belongs to, printed as given after "use it with": `"type: object"`.
	SchemaKeys() map[string][]string
}

type Counterpart struct {
	Name               string
	CheckedParticipant string
	Environment        string
}
