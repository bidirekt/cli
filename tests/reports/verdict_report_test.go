package reports_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/bidirekt/cli/internal/paint"
	"github.com/bidirekt/cli/internal/reports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFormatDeployableReportNamesTheCounterpartsComparedWithInAlphabeticalOrder(t *testing.T) {
	apiVersion := "4.0.0"
	results := map[string]reports.CanIDeployResult{
		"petstore_reviews": {Deployable: true},
		"petstore_api":     {Deployable: true, ParticipantVersion: &apiVersion},
		"petstore_admin":   {Deployable: true},
	}

	report := reports.FormatDeployableReport(paint.For(&bytes.Buffer{}), "petstore_web 3.0.0", "production", results)

	assert.Equal(t, `petstore_web 3.0.0 can be deployed to production
compared with: petstore_admin, petstore_api, petstore_reviews
`, report)
}

func TestFormatDeployableReportSaysWhenNothingIsDeployedToCompareWith(t *testing.T) {
	report := reports.FormatDeployableReport(paint.For(&bytes.Buffer{}), "petstore_web 3.0.0", "production", map[string]reports.CanIDeployResult{})

	assert.Equal(t, `petstore_web 3.0.0 can be deployed to production
nothing deployed in production to compare with
`, report)
}

func TestFormatDeployableReportPaintsOnlyTheVerdictGreen(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	t.Setenv("CLICOLOR_FORCE", "1")
	results := map[string]reports.CanIDeployResult{
		"petstore_api": {Deployable: true},
	}

	report := reports.FormatDeployableReport(paint.For(&bytes.Buffer{}), "petstore_web 3.0.0", "production", results)

	assert.Equal(t, "\x1b[32mpetstore_web 3.0.0 can be deployed to production\x1b[0m\ncompared with: petstore_api\n", report)
}

func TestFormatSyntaxValidReportSaysTheContractIsValidAndHowToCompareIt(t *testing.T) {
	report := reports.FormatSyntaxValidReport(paint.For(&bytes.Buffer{}))

	assert.Equal(t, "contract is valid; add --participant and --environment to compare it with what is deployed\n", report)
}

func TestFormatSyntaxValidReportPaintsTheLineGreen(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	t.Setenv("CLICOLOR_FORCE", "1")

	report := reports.FormatSyntaxValidReport(paint.For(&bytes.Buffer{}))

	assert.Equal(t, "\x1b[32mcontract is valid; add --participant and --environment to compare it with what is deployed\x1b[0m\n", report)
}

func TestFormatNotDeployableReportHeadsEachBreakingCounterpartAndHandsItsTreesToTheirModule(t *testing.T) {
	apiVersion := "4.0.0"
	paymentsVersion := "1.4.0"
	results := map[string]reports.CanIDeployResult{
		"petstore_api": {
			Deployable:         false,
			ParticipantVersion: &apiVersion,
			Trees: map[string]json.RawMessage{
				"topics": json.RawMessage(`{"pet_events":[{"reason":"topic_not_found","role":"consumer"}]}`),
			},
		},
		"petstore_reviews": {
			Deployable: false,
			Trees: map[string]json.RawMessage{
				"topics": json.RawMessage(`{"review_events":[{"reason":"topic_not_found","role":"consumer"}]}`),
			},
		},
		"petstore_payments": {Deployable: true, ParticipantVersion: &paymentsVersion},
	}

	report, err := reports.FormatNotDeployableReport(paint.For(&bytes.Buffer{}), "petstore_web 2.0.0", "petstore_web", "production", results, []reports.Module{
		{Report: echoProtocol{reportKey: "topics"}},
	})

	require.NoError(t, err)
	assert.Equal(t, `petstore_web 2.0.0 cannot be deployed to production

petstore_api (4.0.0, deployed):
  topics of petstore_api, checked for petstore_web in production: {"pet_events":[{"reason":"topic_not_found","role":"consumer"}]}

petstore_reviews:
  topics of petstore_reviews, checked for petstore_web in production: {"review_events":[{"reason":"topic_not_found","role":"consumer"}]}
`, report)
}

func TestFormatNotDeployableReportNamesTheMembersOfACounterpartInPlaceOfItsVersion(t *testing.T) {
	results := map[string]reports.CanIDeployResult{
		"petstore_graph": {
			Deployable: false,
			Members:    map[string]string{"petstore_reviews": "2.0.0", "petstore_api": "1.4.0", "petstore_orders": "1.0.0"},
			Trees: map[string]json.RawMessage{
				"topics": json.RawMessage(`{"pet_events":[{"reason":"topic_not_found","role":"consumer"}]}`),
			},
		},
	}

	report, err := reports.FormatNotDeployableReport(paint.For(&bytes.Buffer{}), "petstore_web 2.0.0", "petstore_web", "production", results, []reports.Module{
		{Report: echoProtocol{reportKey: "topics"}},
	})

	require.NoError(t, err)
	assert.Equal(t, `petstore_web 2.0.0 cannot be deployed to production

petstore_graph (petstore_api 1.4.0 + petstore_orders 1.0.0 + petstore_reviews 2.0.0, deployed):
  topics of petstore_graph, checked for petstore_web in production: {"pet_events":[{"reason":"topic_not_found","role":"consumer"}]}
`, report)
}

func TestFormatNotDeployableReportMarksTheMemberThatIsTheCheckedParticipantAsTheCandidate(t *testing.T) {
	results := map[string]reports.CanIDeployResult{
		"petstore_graph": {
			Deployable: false,
			Members:    map[string]string{"petstore_reviews": "v1", "petstore_api": "v2"},
			Trees: map[string]json.RawMessage{
				"topics": json.RawMessage(`{"pet_events":[{"reason":"topic_not_found","role":"consumer"}]}`),
			},
		},
	}

	report, err := reports.FormatNotDeployableReport(paint.For(&bytes.Buffer{}), "petstore_api v2", "petstore_api", "production", results, []reports.Module{
		{Report: echoProtocol{reportKey: "topics"}},
	})

	require.NoError(t, err)
	assert.Equal(t, `petstore_api v2 cannot be deployed to production

petstore_graph (petstore_api v2 candidate + petstore_reviews v1, deployed):
  topics of petstore_graph, checked for petstore_api in production: {"pet_events":[{"reason":"topic_not_found","role":"consumer"}]}
`, report)
}

func TestFormatNotDeployableReportMarksACandidateWithoutAVersionOnce(t *testing.T) {
	results := map[string]reports.CanIDeployResult{
		"petstore_graph": {
			Deployable: false,
			Members:    map[string]string{"petstore_reviews": "v1", "petstore_api": "candidate"},
			Trees: map[string]json.RawMessage{
				"topics": json.RawMessage(`{"pet_events":[{"reason":"topic_not_found","role":"consumer"}]}`),
			},
		},
	}

	report, err := reports.FormatNotDeployableReport(paint.For(&bytes.Buffer{}), "petstore_api local contract", "petstore_api", "production", results, []reports.Module{
		{Report: echoProtocol{reportKey: "topics"}},
	})

	require.NoError(t, err)
	assert.Equal(t, `petstore_api local contract cannot be deployed to production

petstore_graph (petstore_api candidate + petstore_reviews v1, deployed):
  topics of petstore_graph, checked for petstore_api in production: {"pet_events":[{"reason":"topic_not_found","role":"consumer"}]}
`, report)
}

func TestFormatNotDeployableReportWritesTheTreesOfACounterpartInReportKeyOrder(t *testing.T) {
	results := map[string]reports.CanIDeployResult{
		"petstore_api": {
			Deployable: false,
			Trees: map[string]json.RawMessage{
				"topics": json.RawMessage(`{"pet_events":[{"reason":"topic_not_found","role":"consumer"}]}`),
				"queues": json.RawMessage(`{"adoptions":[{"reason":"queue_not_found","role":"consumer"}]}`),
			},
		},
	}

	report, err := reports.FormatNotDeployableReport(paint.For(&bytes.Buffer{}), "petstore_web 2.0.0", "petstore_web", "production", results, []reports.Module{
		{Report: echoProtocol{reportKey: "topics"}},
		{Report: echoProtocol{reportKey: "queues"}},
	})

	require.NoError(t, err)
	assert.Equal(t, `petstore_web 2.0.0 cannot be deployed to production

petstore_api:
  queues of petstore_api, checked for petstore_web in production: {"adoptions":[{"reason":"queue_not_found","role":"consumer"}]}
  topics of petstore_api, checked for petstore_web in production: {"pet_events":[{"reason":"topic_not_found","role":"consumer"}]}
`, report)
}

func TestFormatNotDeployableReportRefusesATreeNoModuleReports(t *testing.T) {
	results := map[string]reports.CanIDeployResult{
		"petstore_api": {
			Deployable: false,
			Trees: map[string]json.RawMessage{
				"endpoints": json.RawMessage(`{"/pets":{"get":{"200":[{"reason":"provider_resource_not_found","role":"consumer"}]}}}`),
			},
		},
	}

	report, err := reports.FormatNotDeployableReport(paint.For(&bytes.Buffer{}), "petstore_web 2.0.0", "petstore_web", "production", results, []reports.Module{
		{Report: echoProtocol{reportKey: "topics"}},
	})

	require.EqualError(t, err, "the broker reported protocol endpoints, which this bidirekt doesn't know; update bidirekt")
	assert.Empty(t, report)
}

func TestFormatNotDeployableReportReturnsTheErrorOfATreeItsModuleCannotWrite(t *testing.T) {
	failure := errors.New("cannot parse the topics of petstore_api: unexpected end of JSON input")
	results := map[string]reports.CanIDeployResult{
		"petstore_api": {
			Deployable: false,
			Trees: map[string]json.RawMessage{
				"topics": json.RawMessage(`{"pet_events":[`),
			},
		},
	}

	report, err := reports.FormatNotDeployableReport(paint.For(&bytes.Buffer{}), "petstore_web 2.0.0", "petstore_web", "production", results, []reports.Module{
		{Report: echoProtocol{reportKey: "topics", failure: failure}},
	})

	require.ErrorIs(t, err, failure)
	assert.Empty(t, report)
}

func TestFormatNotDeployableReportPaintsTheHeadlineRed(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	t.Setenv("CLICOLOR_FORCE", "1")
	results := map[string]reports.CanIDeployResult{
		"petstore_reviews": {
			Deployable: false,
			Trees: map[string]json.RawMessage{
				"topics": json.RawMessage(`{"review_events":[{"reason":"topic_not_found","role":"consumer"}]}`),
			},
		},
	}

	report, err := reports.FormatNotDeployableReport(paint.For(&bytes.Buffer{}), "petstore_web 2.0.0", "petstore_web", "production", results, []reports.Module{
		{Report: echoProtocol{reportKey: "topics"}},
	})

	require.NoError(t, err)
	assert.Equal(t, "\x1b[31mpetstore_web 2.0.0 cannot be deployed to production\x1b[0m\n"+
		"\n"+
		"petstore_reviews:\n"+
		`  topics of petstore_reviews, checked for petstore_web in production: {"review_events":[{"reason":"topic_not_found","role":"consumer"}]}`+"\n",
		report)
}

func TestCanIDeployResultReadsTheHeaderKeysAndKeepsEveryOtherKeyAsATree(t *testing.T) {
	var result reports.CanIDeployResult

	err := json.Unmarshal([]byte(`{
	  "deployable": false,
	  "participantVersion": "4.0.0",
	  "members": {"petstore_api": "1.4.0", "petstore_reviews": "2.0.0"},
	  "endpoints": {"/pets": {"get": {"200": [{"reason": "provider_resource_not_found", "role": "consumer"}]}}},
	  "topics": {"pet_events": [{"reason": "topic_not_found", "role": "consumer"}]}
	}`), &result)

	require.NoError(t, err)
	assert.False(t, result.Deployable)
	require.NotNil(t, result.ParticipantVersion)
	assert.Equal(t, "4.0.0", *result.ParticipantVersion)
	assert.Equal(t, map[string]string{"petstore_api": "1.4.0", "petstore_reviews": "2.0.0"}, result.Members)
	require.Len(t, result.Trees, 2)
	assert.JSONEq(t, `{"/pets": {"get": {"200": [{"reason": "provider_resource_not_found", "role": "consumer"}]}}}`, string(result.Trees["endpoints"]))
	assert.JSONEq(t, `{"pet_events": [{"reason": "topic_not_found", "role": "consumer"}]}`, string(result.Trees["topics"]))
}

func TestCanIDeployResultCountsAnEmptyTreeAsAbsent(t *testing.T) {
	var result reports.CanIDeployResult

	err := json.Unmarshal([]byte(`{"deployable": true, "participantVersion": null, "endpoints": {}, "topics": { }, "queues": null}`), &result)

	require.NoError(t, err)
	assert.True(t, result.Deployable)
	assert.Nil(t, result.ParticipantVersion)
	assert.Nil(t, result.Members)
	assert.Empty(t, result.Trees)
}

type echoProtocol struct {
	reportKey string
	failure   error
}

func (this echoProtocol) ReportKey() string {
	return this.reportKey
}

func (this echoProtocol) WriteTree(report *strings.Builder, counterpart reports.Counterpart, tree json.RawMessage) error {
	if this.failure != nil {
		return this.failure
	}

	_, err := fmt.Fprintf(report, "  %s of %s, checked for %s in %s: %s\n", this.reportKey, counterpart.Name, counterpart.CheckedParticipant, counterpart.Environment, tree)
	return err
}

func (this echoProtocol) ViolationLine(reports.Violation) (headline, explanation string, known bool) {
	return "", "", false
}

func (this echoProtocol) SchemaKeys() map[string][]string {
	return nil
}
