package reports

import (
	"cmp"
	"fmt"
	"maps"
	"slices"
	"sort"
	"strconv"
	"strings"

	"github.com/bidirekt/cli/internal/paint"
)

// Endpoints nest as endpoint -> method -> interaction -> breaks, where interaction is
// "request" or a response status code ("200").
type CanIDeployResult struct {
	Deployable         bool                                             `json:"deployable"`
	ParticipantVersion *string                                          `json:"participantVersion"`
	Endpoints          map[string]map[string]map[string][]ContractBreak `json:"endpoints"`
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

func FormatNotDeployableReport(brush paint.Brush, checkedSideLabel, participant, environment string, results map[string]CanIDeployResult) string {
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

		report.WriteString("\n" + name)
		if result.ParticipantVersion != nil {
			fmt.Fprintf(&report, " (%s, deployed)", *result.ParticipantVersion)
		}
		report.WriteString(":\n")

		var blocks []fieldBlock
		endpoints := sortedKeys(result.Endpoints)
		for _, endpoint := range endpoints {
			methods := sortedKeys(result.Endpoints[endpoint])
			for _, method := range methods {
				checked := checkedInteraction{
					participant: participant,
					counterpart: name,
					environment: environment,
					method:      strings.ToUpper(method),
					endpoint:    endpoint,
				}

				removed, interactions := splitRemovedBreak(result.Endpoints[endpoint][method])
				methodBlocks, interactions := splitFieldBlocks(checked, interactions)
				blocks = append(blocks, methodBlocks...)
				if removed == nil && len(interactions) == 0 {
					continue
				}

				fmt.Fprintf(&report, "  %s %s\n", strings.ToUpper(method), endpoint)

				if removed != nil {
					writeBreakLine(&report, "    ", formatBreakLine(checked, *removed))
				}

				for _, interaction := range sortedInteractions(interactions) {
					checked.isRequest = interaction == "request"

					if checked.isRequest {
						checked.status = ""
						report.WriteString("    request:\n")
					} else {
						checked.status = interaction
						fmt.Fprintf(&report, "    response %s:\n", interaction)
					}

					for _, contractBreak := range interactions[interaction] {
						writeBreakLine(&report, "      ", formatBreakLine(checked, contractBreak))
					}
				}
			}
		}

		for _, block := range mergeFieldBlocks(blocks) {
			writeFieldBlock(&report, name, block)
		}
	}

	return report.String()
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

// splitRemovedBreak takes the removed-endpoint break out of a method's interactions: the broker sends it once per
// consumed interaction, but it is about the whole endpoint. Interactions left without breaks are dropped.
func splitRemovedBreak(interactions map[string][]ContractBreak) (*ContractBreak, map[string][]ContractBreak) {
	var removed *ContractBreak
	remaining := make(map[string][]ContractBreak, len(interactions))
	for _, interaction := range sortedInteractions(interactions) {
		var kept []ContractBreak
		for _, contractBreak := range interactions[interaction] {
			if contractBreak.Reason != "provider_resource_removed_but_still_consumed" {
				kept = append(kept, contractBreak)
			} else if removed == nil {
				removed = &contractBreak
			}
		}
		if len(kept) > 0 {
			remaining[interaction] = kept
		}
	}
	return removed, remaining
}

// splitFieldBlocks takes the property breaks a template names out of a method's interactions, each as a block of one
// place. Interactions left without breaks are dropped.
func splitFieldBlocks(checked checkedInteraction, interactions map[string][]ContractBreak) ([]fieldBlock, map[string][]ContractBreak) {
	var blocks []fieldBlock
	remaining := make(map[string][]ContractBreak, len(interactions))
	for interaction, contractBreaks := range interactions {
		for _, contractBreak := range contractBreaks {
			if block, named := toFieldBlock(checked, interaction, contractBreak); named {
				blocks = append(blocks, block)
			} else {
				remaining[interaction] = append(remaining[interaction], contractBreak)
			}
		}
	}
	return blocks, remaining
}

// sortedInteractions orders a method's interaction keys as "request" first, then the
// response status codes sorted.
func sortedInteractions(interactions map[string][]ContractBreak) []string {
	return slices.SortedFunc(maps.Keys(interactions), compareInteractions)
}

func compareInteractions(a, b string) int {
	switch {
	case a == b:
		return 0
	case a == "request":
		return -1
	case b == "request":
		return 1
	default:
		return strings.Compare(a, b)
	}
}

// One block has one header and one set of fixes, so a break's direction and role are part of its block's key too.
type fieldBlockKey struct {
	field                                    string
	reason                                   string
	consumerType, providerType, propertyType string
	direction, role                          string
}

type fieldBlock struct {
	fieldBlockKey
	propertyBreakLine
	places []breakPlace
}

type breakPlace struct {
	method, endpoint, interaction, property string
	counterpartField                        string
}

func toFieldBlock(checked checkedInteraction, interaction string, contractBreak ContractBreak) (fieldBlock, bool) {
	checked.isRequest = interaction == "request"
	line, named := formatPropertyBreakLine(checked, contractBreak)
	if !named {
		return fieldBlock{}, false
	}

	details := contractBreak.Details
	checkedField, otherField := details["consumerField"], details["providerField"]
	if contractBreak.Role == "provider" {
		checkedField, otherField = otherField, checkedField
	}
	direction := "response"
	if checked.isRequest {
		direction = "request"
	}

	return fieldBlock{
		fieldBlockKey: fieldBlockKey{
			field:        cmp.Or(checkedField, otherField, propertyPathPhrase(details["property"])),
			reason:       contractBreak.Reason,
			consumerType: details["consumerPropertyType"],
			providerType: details["providerPropertyType"],
			propertyType: details["propertyType"],
			direction:    direction,
			role:         contractBreak.Role,
		},
		propertyBreakLine: line,
		places: []breakPlace{{
			method:           checked.method,
			endpoint:         checked.endpoint,
			interaction:      interaction,
			property:         details["property"],
			counterpartField: otherField,
		}},
	}, true
}

func mergeFieldBlocks(blocks []fieldBlock) []fieldBlock {
	slices.SortFunc(blocks, func(a, b fieldBlock) int {
		return cmp.Or(
			compareFieldBlockKeys(a.fieldBlockKey, b.fieldBlockKey),
			comparePlaces(a.places[0], b.places[0]),
			strings.Compare(a.places[0].counterpartField, b.places[0].counterpartField),
		)
	})
	// A place repeated in a block prints once, with the first of its counterpart fields.
	blocks = slices.CompactFunc(blocks, func(a, b fieldBlock) bool {
		return a.fieldBlockKey == b.fieldBlockKey && comparePlaces(a.places[0], b.places[0]) == 0
	})

	var merged []fieldBlock
	for _, block := range blocks {
		if len(merged) > 0 && merged[len(merged)-1].fieldBlockKey == block.fieldBlockKey {
			last := &merged[len(merged)-1]
			last.places = append(last.places, block.places...)
		} else {
			merged = append(merged, block)
		}
	}
	return merged
}

func compareFieldBlockKeys(a, b fieldBlockKey) int {
	return cmp.Or(
		strings.Compare(a.field, b.field),
		strings.Compare(a.reason, b.reason),
		strings.Compare(a.consumerType, b.consumerType),
		strings.Compare(a.providerType, b.providerType),
		strings.Compare(a.propertyType, b.propertyType),
		strings.Compare(a.direction, b.direction),
		strings.Compare(a.role, b.role),
	)
}

func comparePlaces(a, b breakPlace) int {
	return cmp.Or(
		strings.Compare(a.endpoint, b.endpoint),
		strings.Compare(a.method, b.method),
		compareInteractions(a.interaction, b.interaction),
		strings.Compare(a.property, b.property),
	)
}

func writeFieldBlock(report *strings.Builder, counterpart string, block fieldBlock) {
	fmt.Fprintf(report, "  %s — %s", block.field, block.text)
	if len(block.places) > 1 {
		fmt.Fprintf(report, " (%d places)", len(block.places))
	}
	report.WriteString("\n")

	otherFix := block.otherFix
	somePlaceNamesTheOtherSidesField := slices.ContainsFunc(block.places, func(place breakPlace) bool { return place.counterpartField != "" })
	if somePlaceNamesTheOtherSidesField && !block.otherFixNamesParent {
		otherFix += " there"
	}
	fmt.Fprintf(report, "    → %s\n    → %s\n", block.checkedFix, otherFix)

	for _, place := range block.places {
		interaction := "response " + place.interaction
		if place.interaction == "request" {
			interaction = "request"
		}
		fmt.Fprintf(report, "      · %s %s %s at %s", place.method, place.endpoint, interaction, propertyPathPhrase(place.property))
		if place.counterpartField != "" {
			fmt.Fprintf(report, " (%s: %s)", counterpart, place.counterpartField)
		}
		report.WriteString("\n")
	}
}

type checkedInteraction struct {
	participant string
	counterpart string
	environment string
	method      string
	endpoint    string
	isRequest   bool
	status      string
}

type breakLine struct {
	text  string
	fixes []string
}

func writeBreakLine(report *strings.Builder, indent string, line breakLine) {
	fmt.Fprintf(report, "%s- %s\n", indent, line.text)
	for _, fix := range line.fixes {
		fmt.Fprintf(report, "%s  → %s\n", indent, fix)
	}
}

func formatBreakLine(interaction checkedInteraction, contractBreak ContractBreak) breakLine {
	me, other := interaction.participant, interaction.counterpart
	resource := interaction.method + " " + interaction.endpoint
	reason, details := contractBreak.Reason, contractBreak.Details
	asConsumer, asProvider := contractBreak.Role == "consumer", contractBreak.Role == "provider"
	onRequest, onResponse := interaction.isRequest, !interaction.isRequest

	switch {
	case reason == "provider_resource_not_found" && asConsumer:
		return breakLine{
			text:  fmt.Sprintf("%s calls %s, but %s doesn't provide it", me, resource, other),
			fixes: []string{me + ": stop calling it", other + ": publish it"},
		}
	case reason == "provider_resource_interaction_not_found" && asConsumer && onRequest:
		return breakLine{
			text:  fmt.Sprintf("%s sends a request body, but %s declares none", me, other),
			fixes: []string{me + ": stop sending it", other + ": accept a request body"},
		}
	case reason == "provider_resource_interaction_not_found" && asConsumer && onResponse && details["providerStatuses"] == "":
		return breakLine{
			text:  fmt.Sprintf("%s reads response %s, but %s declares no response", me, interaction.status, other),
			fixes: []string{me + ": stop reading it", other + ": answer " + interaction.status},
		}
	case reason == "provider_resource_interaction_not_found" && asConsumer && onResponse:
		providerStatuses := orList(details["providerStatuses"])
		return breakLine{
			text:  fmt.Sprintf("%s reads response %s, but %s answers only %s", me, interaction.status, other, providerStatuses),
			fixes: []string{me + ": read response " + providerStatuses, other + ": answer " + interaction.status},
		}
	case reason == "provider_resource_not_deployed_in_environment" && asConsumer:
		deployedIn := ""
		if environments, ok := details["deployedEnvironments"]; ok {
			deployedIn = fmt.Sprintf(" (deployed in: %s)", environments)
		}
		return breakLine{
			text:  fmt.Sprintf("%s calls %s, but %s is not deployed in %s%s", me, resource, other, interaction.environment, deployedIn),
			fixes: []string{fmt.Sprintf("%s: deploy it to %s first", other, interaction.environment)},
		}
	case reason == "provider_resource_removed_but_still_consumed" && asProvider:
		return breakLine{
			text:  fmt.Sprintf("%s removed %s, but %s still calls it", me, resource, other),
			fixes: []string{fmt.Sprintf("%s: keep it until %s stops calling it", me, other), other + ": stop calling it"},
		}
	default:
		return breakLine{text: fallbackBreakLine(reason, details)}
	}
}

type propertyBreakLine struct {
	text                 string
	checkedFix, otherFix string
	otherFixNamesParent  bool
}

func formatPropertyBreakLine(interaction checkedInteraction, contractBreak ContractBreak) (propertyBreakLine, bool) {
	reason, details := contractBreak.Reason, contractBreak.Details
	consumerType, providerType := details["consumerPropertyType"], details["providerPropertyType"]
	asConsumer, asProvider := contractBreak.Role == "consumer", contractBreak.Role == "provider"
	consumer, provider := interaction.participant, interaction.counterpart
	if asProvider {
		consumer, provider = provider, consumer
	}
	property := propertyBreak{
		consumer:          consumer,
		provider:          provider,
		consumerField:     details["consumerField"],
		providerField:     details["providerField"],
		path:              details["property"],
		consumerIsChecked: asConsumer,
		onRequest:         interaction.isRequest,
	}
	onRequest, onResponse := interaction.isRequest, !interaction.isRequest

	switch {
	case !asConsumer && !asProvider || property.path == "":
		return propertyBreakLine{}, false
	case reason == "property_missing_in_provider" && onResponse:
		return property.toPropertyBreakLine(propertyBreakPhrases{
			consumerChecked:        fmt.Sprintf("%s reads it, but %s doesn't provide it", consumer, provider),
			providerChecked:        fmt.Sprintf("%s doesn't provide it, but %s reads it", provider, consumer),
			consumerFix:            "stop reading it, or handle its absence in the code and mark it optional",
			providerFix:            "provide it",
			readerFixMarksOptional: true,
		}), true
	case reason == "property_optional_in_provider_required_in_consumer" && onResponse:
		return property.toPropertyBreakLine(propertyBreakPhrases{
			consumerChecked:        fmt.Sprintf("%s requires it, but %s only sometimes provides it", consumer, provider),
			providerChecked:        fmt.Sprintf("%s provides it only sometimes, but %s requires it", provider, consumer),
			consumerFix:            "handle its absence in the code, then mark it optional",
			providerFix:            "always provide it",
			readerFixMarksOptional: true,
		}), true
	case reason == "property_type_mismatch" && onResponse:
		return property.toPropertyBreakLine(propertyBreakPhrases{
			consumerChecked: fmt.Sprintf("%s reads it as %s, but %s provides %s", consumer, consumerType, provider, providerType),
			providerChecked: fmt.Sprintf("%s provides it as %s, but %s reads %s", provider, providerType, consumer, consumerType),
			consumerFix:     "read it as " + providerType,
			providerFix:     "provide it as " + consumerType,
		}), true
	case reason == "property_nullable_in_provider_not_nullable_in_consumer" && onResponse:
		return property.toPropertyBreakLine(propertyBreakPhrases{
			consumerChecked: fmt.Sprintf("%s doesn't accept null, but %s may provide null", consumer, provider),
			providerChecked: fmt.Sprintf("%s may provide null, but %s doesn't accept it", provider, consumer),
			consumerFix:     "handle null in the code, then mark it nullable",
			providerFix:     "stop providing null",
		}), true
	case reason == "property_missing_in_consumer" && onRequest:
		return property.toPropertyBreakLine(propertyBreakPhrases{
			consumerChecked:        fmt.Sprintf("%s doesn't send it, but %s requires it", consumer, provider),
			providerChecked:        fmt.Sprintf("%s requires it, but %s doesn't send it", provider, consumer),
			consumerFix:            "send it",
			providerFix:            "stop requiring it, then mark it optional",
			readerFixMarksOptional: true,
		}), true
	case reason == "property_optional_in_consumer_required_in_provider" && onRequest:
		return property.toPropertyBreakLine(propertyBreakPhrases{
			consumerChecked:        fmt.Sprintf("%s sends it only sometimes, but %s requires it", consumer, provider),
			providerChecked:        fmt.Sprintf("%s requires it, but %s sends it only sometimes", provider, consumer),
			consumerFix:            "always send it",
			providerFix:            "handle its absence in the code, then mark it optional",
			readerFixMarksOptional: true,
		}), true
	case reason == "property_type_mismatch" && onRequest:
		return property.toPropertyBreakLine(propertyBreakPhrases{
			consumerChecked: fmt.Sprintf("%s sends it as %s, but %s expects %s", consumer, consumerType, provider, providerType),
			providerChecked: fmt.Sprintf("%s expects it as %s, but %s sends %s", provider, providerType, consumer, consumerType),
			consumerFix:     "send " + providerType,
			providerFix:     "accept " + consumerType,
		}), true
	case reason == "property_nullable_in_consumer_not_nullable_in_provider" && onRequest:
		return property.toPropertyBreakLine(propertyBreakPhrases{
			consumerChecked: fmt.Sprintf("%s may send null, but %s doesn't accept it", consumer, provider),
			providerChecked: fmt.Sprintf("%s doesn't accept null, but %s may send it", provider, consumer),
			consumerFix:     "stop sending null",
			providerFix:     "handle null in the code, then mark it nullable",
		}), true
	default:
		return propertyBreakLine{}, false
	}
}

type propertyBreak struct {
	consumer, provider           string
	consumerField, providerField string
	path                         string
	consumerIsChecked            bool
	onRequest                    bool
}

type propertyBreakPhrases struct {
	consumerChecked, providerChecked string
	consumerFix, providerFix         string
	readerFixMarksOptional           bool
}

func (this propertyBreak) toPropertyBreakLine(phrases propertyBreakPhrases) propertyBreakLine {
	consumerFix, consumerFixNamesParent := phrases.consumerFix, false
	providerFix, providerFixNamesParent := phrases.providerFix, false

	// An array's items and a map's values aren't fields to mark optional: their reader reads the parent field the way
	// the other side produces it. That fix names the parent, so it doesn't point at the places' fields with "there".
	isItemsOrValues := strings.HasSuffix(this.path, "[]") || strings.HasSuffix(this.path, "{}")
	if isItemsOrValues && phrases.readerFixMarksOptional {
		if this.onRequest {
			providerFix, providerFixNamesParent = fmt.Sprintf("read %s the way %s sends it", this.parentPhrase(this.providerField), this.consumer), true
		} else {
			consumerFix, consumerFixNamesParent = fmt.Sprintf("read %s the way %s provides it", this.parentPhrase(this.consumerField), this.provider), true
		}
	}

	consumerLine, providerLine := this.consumer+": "+consumerFix, this.provider+": "+providerFix
	if this.consumerIsChecked {
		return propertyBreakLine{text: phrases.consumerChecked, checkedFix: consumerLine, otherFix: providerLine, otherFixNamesParent: providerFixNamesParent}
	}
	return propertyBreakLine{text: phrases.providerChecked, checkedFix: providerLine, otherFix: consumerLine, otherFixNamesParent: consumerFixNamesParent}
}

func (this propertyBreak) parentPhrase(readerField string) string {
	itemsOrValues := this.path[len(this.path)-len("[]"):]
	if readerField == "" {
		return propertyPathPhrase(strings.TrimSuffix(this.path, itemsOrValues))
	}
	return strings.TrimSuffix(readerField, itemsOrValues)
}

func orList(commaList string) string {
	last := strings.LastIndex(commaList, ", ")
	if last < 0 {
		return commaList
	}
	return commaList[:last] + " or " + commaList[last+len(", "):]
}

func propertyPathPhrase(path string) string {
	if path == "$" {
		return "the body"
	}
	return strconv.Quote(path)
}

// fallbackBreakLine renders a break verbatim with its details when no template matches its
// reason, role and interaction, so the CLI never swallows a break.
func fallbackBreakLine(reason string, details map[string]string) string {
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
