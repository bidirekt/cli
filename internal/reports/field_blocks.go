package reports

import (
	"cmp"
	"fmt"
	"slices"
	"strconv"
	"strings"
)

// One block has one header and one set of fixes, so a break's direction and role are part of its block's key too.
type fieldBlockKey struct {
	field                                    string
	reason                                   string
	consumerType, providerType, propertyType string
	member                                   string
	direction, role                          string
	propertyBreakLine
}

type fieldBlock struct {
	fieldBlockKey
	places []breakPlace
}

type breakPlace struct {
	method, endpoint, interaction, property string
	counterpartField                        string
}

func toFieldBlock(checked checkedInteraction, interaction string, contractBreak ContractBreak) (fieldBlock, bool) {
	checked.isRequest = interaction == "request"
	property, line, named := formatPropertyBreakLine(checked, contractBreak)
	if !named {
		return fieldBlock{}, false
	}

	details := contractBreak.Details
	checkedField, otherField := property.checkedAndOtherFields()
	direction := "response"
	if checked.isRequest {
		direction = "request"
	}

	return fieldBlock{
		fieldBlockKey: fieldBlockKey{
			field:             cmp.Or(checkedField, otherField, propertyPathPhrase(property.path)),
			reason:            contractBreak.Reason,
			consumerType:      details["consumerPropertyType"],
			providerType:      details["providerPropertyType"],
			propertyType:      details["propertyType"],
			member:            details["member"],
			direction:         direction,
			role:              contractBreak.Role,
			propertyBreakLine: line,
		},
		places: []breakPlace{{
			method:           checked.method,
			endpoint:         checked.endpoint,
			interaction:      interaction,
			property:         property.path,
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
		strings.Compare(a.member, b.member),
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
	if somePlaceNamesTheOtherSidesField {
		otherFix += " there"
	}
	fmt.Fprintf(report, "    → %s\n    → %s\n", block.checkedFix, otherFix)

	for _, place := range block.places {
		fmt.Fprintf(report, "      · %s %s %s at %s", place.method, place.endpoint, interactionPhrase(place.interaction), propertyPathPhrase(place.property))
		if place.counterpartField != "" {
			fmt.Fprintf(report, " (%s: %s)", counterpart, place.counterpartField)
		}
		report.WriteString("\n")
	}
}

type propertyBreakLine struct {
	text                 string
	checkedFix, otherFix string
}

func formatPropertyBreakLine(interaction checkedInteraction, contractBreak ContractBreak) (propertyBreak, propertyBreakLine, bool) {
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
	}
	member := strconv.Quote(details["member"])
	onRequest, onResponse := interaction.isRequest, !interaction.isRequest

	switch {
	case !asConsumer && !asProvider || property.path == "":
		return propertyBreak{}, propertyBreakLine{}, false
	case reason == "property_missing_in_provider" && onResponse:
		return property, property.toPropertyBreakLine(propertyBreakPhrases{
			consumerChecked: fmt.Sprintf("%s reads it, but %s doesn't provide it", consumer, provider),
			providerChecked: fmt.Sprintf("%s doesn't provide it, but %s reads it", provider, consumer),
			consumerFix:     "stop reading it, or handle its absence in the code and mark it optional",
			providerFix:     "provide it",
		}), true
	case reason == "property_optional_in_provider_required_in_consumer" && onResponse:
		return property, property.toPropertyBreakLine(propertyBreakPhrases{
			consumerChecked: fmt.Sprintf("%s requires it, but %s only sometimes provides it", consumer, provider),
			providerChecked: fmt.Sprintf("%s provides it only sometimes, but %s requires it", provider, consumer),
			consumerFix:     "handle its absence in the code, then mark it optional",
			providerFix:     "always provide it",
		}), true
	case reason == "property_type_mismatch" && onResponse:
		return property, property.toPropertyBreakLine(propertyBreakPhrases{
			consumerChecked: fmt.Sprintf("%s reads it as %s, but %s provides %s", consumer, consumerType, provider, providerType),
			providerChecked: fmt.Sprintf("%s provides it as %s, but %s reads %s", provider, providerType, consumer, consumerType),
			consumerFix:     "read it as " + providerType,
			providerFix:     "provide it as " + consumerType,
		}), true
	case reason == "property_nullable_in_provider_not_nullable_in_consumer" && onResponse:
		return property, property.toPropertyBreakLine(propertyBreakPhrases{
			consumerChecked: fmt.Sprintf("%s doesn't accept null, but %s may provide null", consumer, provider),
			providerChecked: fmt.Sprintf("%s may provide null, but %s doesn't accept it", provider, consumer),
			consumerFix:     "handle null in the code, then mark it nullable",
			providerFix:     "stop providing null",
		}), true
	case reason == "property_member_missing_in_provider" && onResponse:
		return property, property.toPropertyBreakLine(propertyBreakPhrases{
			consumerChecked: fmt.Sprintf("%s reads %s, but %s doesn't provide it", consumer, member, provider),
			providerChecked: fmt.Sprintf("%s doesn't provide %s, but %s reads it", provider, member, consumer),
			consumerFix:     "stop reading " + member,
			providerFix:     "provide " + member,
		}), true
	case reason == "property_missing_in_consumer" && onRequest:
		return property, property.toPropertyBreakLine(propertyBreakPhrases{
			consumerChecked: fmt.Sprintf("%s doesn't send it, but %s requires it", consumer, provider),
			providerChecked: fmt.Sprintf("%s requires it, but %s doesn't send it", provider, consumer),
			consumerFix:     "send it",
			providerFix:     "stop requiring it, then mark it optional",
		}), true
	case reason == "property_optional_in_consumer_required_in_provider" && onRequest:
		return property, property.toPropertyBreakLine(propertyBreakPhrases{
			consumerChecked: fmt.Sprintf("%s sends it only sometimes, but %s requires it", consumer, provider),
			providerChecked: fmt.Sprintf("%s requires it, but %s sends it only sometimes", provider, consumer),
			consumerFix:     "always send it",
			providerFix:     "handle its absence in the code, then mark it optional",
		}), true
	case reason == "property_type_mismatch" && onRequest:
		return property, property.toPropertyBreakLine(propertyBreakPhrases{
			consumerChecked: fmt.Sprintf("%s sends it as %s, but %s expects %s", consumer, consumerType, provider, providerType),
			providerChecked: fmt.Sprintf("%s expects it as %s, but %s sends %s", provider, providerType, consumer, consumerType),
			consumerFix:     "send " + providerType,
			providerFix:     "accept " + consumerType,
		}), true
	case reason == "property_nullable_in_consumer_not_nullable_in_provider" && onRequest:
		return property, property.toPropertyBreakLine(propertyBreakPhrases{
			consumerChecked: fmt.Sprintf("%s may send null, but %s doesn't accept it", consumer, provider),
			providerChecked: fmt.Sprintf("%s doesn't accept null, but %s may send it", provider, consumer),
			consumerFix:     "stop sending null",
			providerFix:     "handle null in the code, then mark it nullable",
		}), true
	case reason == "property_member_missing_in_provider" && onRequest:
		return property, property.toPropertyBreakLine(propertyBreakPhrases{
			consumerChecked: fmt.Sprintf("%s sends %s, but %s doesn't accept it", consumer, member, provider),
			providerChecked: fmt.Sprintf("%s doesn't accept %s, but %s sends it", provider, member, consumer),
			consumerFix:     "stop sending " + member,
			providerFix:     "accept " + member,
		}), true
	default:
		return propertyBreak{}, propertyBreakLine{}, false
	}
}

type propertyBreak struct {
	consumer, provider           string
	consumerField, providerField string
	path                         string
	consumerIsChecked            bool
}

type propertyBreakPhrases struct {
	consumerChecked, providerChecked string
	consumerFix, providerFix         string
}

func (this propertyBreak) checkedAndOtherFields() (checkedField, otherField string) {
	if this.consumerIsChecked {
		return this.consumerField, this.providerField
	}
	return this.providerField, this.consumerField
}

func (this propertyBreak) toPropertyBreakLine(phrases propertyBreakPhrases) propertyBreakLine {
	consumerLine, providerLine := this.consumer+": "+phrases.consumerFix, this.provider+": "+phrases.providerFix
	if this.consumerIsChecked {
		return propertyBreakLine{text: phrases.consumerChecked, checkedFix: consumerLine, otherFix: providerLine}
	}
	return propertyBreakLine{text: phrases.providerChecked, checkedFix: providerLine, otherFix: consumerLine}
}
