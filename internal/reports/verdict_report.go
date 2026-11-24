package reports

import (
	"encoding/json"
	"fmt"
	"maps"
	"slices"
	"sort"
	"strings"

	"github.com/bidirekt/cli/internal/paint"
)

var counterpartHeaderKeys = []string{"deployable", "participantVersion", "members"}

type CanIDeployResult struct {
	Deployable         bool                       `json:"deployable"`
	ParticipantVersion *string                    `json:"participantVersion"`
	Members            map[string]string          `json:"members"`
	Trees              map[string]json.RawMessage `json:"-"`
}

func (this *CanIDeployResult) UnmarshalJSON(data []byte) error {
	type counterpartHeader CanIDeployResult
	var result counterpartHeader
	if err := json.Unmarshal(data, &result); err != nil {
		return err
	}

	var trees map[string]json.RawMessage
	if err := json.Unmarshal(data, &trees); err != nil {
		return err
	}
	maps.DeleteFunc(trees, func(key string, tree json.RawMessage) bool {
		return slices.Contains(counterpartHeaderKeys, key) || isEmptyTree(tree)
	})
	result.Trees = trees

	*this = CanIDeployResult(result)
	return nil
}

func isEmptyTree(tree json.RawMessage) bool {
	var branches map[string]json.RawMessage
	return json.Unmarshal(tree, &branches) == nil && len(branches) == 0
}

type ContractBreak struct {
	Reason  string            `json:"reason"`
	Role    string            `json:"role"`
	Details map[string]string `json:"details"`
}

func FormatDeployableReport(brush paint.Brush, checkedSideLabel, environment string, results map[string]CanIDeployResult) string {
	report := brush.Green(checkedSideLabel+" can be deployed to "+environment) + "\n"
	if len(results) == 0 {
		return report + "nothing deployed in " + environment + " to compare with\n"
	}
	return report + "compared with: " + strings.Join(sortedKeys(results), ", ") + "\n"
}

func FormatSyntaxValidReport(brush paint.Brush) string {
	return brush.Green("contract is valid; add --participant and --environment to compare it with what is deployed") + "\n"
}

func FormatNotDeployableReport(brush paint.Brush, checkedSideLabel, participant, environment string, results map[string]CanIDeployResult, modules []Module) (string, error) {
	var report strings.Builder
	report.WriteString(brush.Red(checkedSideLabel+" cannot be deployed to "+environment) + "\n")

	counterparts := make([]string, 0, len(results))
	for name, result := range results {
		if !result.Deployable {
			counterparts = append(counterparts, name)
		}
	}
	sort.Strings(counterparts)

	for _, name := range counterparts {
		result := results[name]

		report.WriteString("\n" + counterpartHeader(name, result) + ":\n")

		counterpart := Counterpart{Name: name, CheckedParticipant: participant, Environment: environment}
		if err := writeTrees(&report, counterpart, result.Trees, modules); err != nil {
			return "", err
		}
	}

	return report.String(), nil
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func counterpartHeader(name string, result CanIDeployResult) string {
	if len(result.Members) > 0 {
		members := make([]string, 0, len(result.Members))
		for _, member := range sortedKeys(result.Members) {
			members = append(members, member+" "+result.Members[member])
		}
		return fmt.Sprintf("%s (%s, deployed)", name, strings.Join(members, " + "))
	}

	if result.ParticipantVersion != nil {
		return fmt.Sprintf("%s (%s, deployed)", name, *result.ParticipantVersion)
	}

	return name
}

func writeTrees(report *strings.Builder, counterpart Counterpart, trees map[string]json.RawMessage, modules []Module) error {
	for _, reportKey := range sortedKeys(trees) {
		protocol, err := protocolFor(reportKey, modules)
		if err != nil {
			return err
		}

		if err := protocol.WriteTree(report, counterpart, trees[reportKey]); err != nil {
			return err
		}
	}

	return nil
}
