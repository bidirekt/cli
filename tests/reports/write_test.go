package reports_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/bidirekt/cli/internal/reports"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteValidationFailedPrintsTheReportAndSilencesTheError(t *testing.T) {
	validationFailed := &reports.ValidationFailedError{Message: "contract validation failed", Violations: []reports.Violation{
		{Code: "schema.array_without_items", Path: "schemas;Pets", Source: "api.yaml"},
	}}
	var errWriter bytes.Buffer

	err := reports.WriteValidationFailed(&errWriter, validationFailed, nil)

	require.ErrorIs(t, err, reports.ErrSilent)
	assert.Equal(t, `contract validation failed
  - api.yaml: array schema without items at schemas Pets
`, errWriter.String())
}

func TestWriteValidationFailedReturnsAnyOtherErrorAsIs(t *testing.T) {
	participantNotFound := errors.New("cannot validate contract on broker: participant not found")
	var errWriter bytes.Buffer

	err := reports.WriteValidationFailed(&errWriter, participantNotFound, nil)

	assert.Same(t, participantNotFound, err)
	assert.Empty(t, errWriter.String())
}

func TestWriteVerdictPrintsTheDeployableReport(t *testing.T) {
	results := map[string]reports.CanIDeployResult{
		"petstore_api": {Deployable: true},
	}
	var writer bytes.Buffer

	err := reports.WriteVerdict(&writer, "petstore_web 3.0.0", "petstore_web", "production", true, results, []reports.Module{{Report: rawTreeProtocol{}}})

	require.NoError(t, err)
	assert.Equal(t, `petstore_web 3.0.0 can be deployed to production
compared with: petstore_api
`, writer.String())
}

func TestWriteVerdictPrintsTheReportAndSilencesTheError(t *testing.T) {
	results := map[string]reports.CanIDeployResult{
		"petstore_reviews": {
			Deployable: false,
			Trees: map[string]json.RawMessage{
				"topics": json.RawMessage(`{"review_events":[{"reason":"topic_not_found","role":"consumer"}]}`),
			},
		},
	}
	var writer bytes.Buffer

	err := reports.WriteVerdict(&writer, "petstore_web 2.0.0", "petstore_web", "production", false, results, []reports.Module{{Report: rawTreeProtocol{}}})

	require.ErrorIs(t, err, reports.ErrSilent)
	assert.Equal(t, `petstore_web 2.0.0 cannot be deployed to production

petstore_reviews:
  topics of petstore_reviews, checked for petstore_web in production: {"review_events":[{"reason":"topic_not_found","role":"consumer"}]}
`, writer.String())
}

func TestWriteVerdictRefusesATreeNoModuleReportsBeforePrintingAnything(t *testing.T) {
	for _, test := range []struct {
		name       string
		deployable bool
		modules    []reports.Module
	}{
		{name: "a breaking counterpart, without any module", deployable: false},
		{name: "a deployable counterpart, beside the module of another tree", deployable: true, modules: []reports.Module{{Report: rawTreeProtocol{}}}},
	} {
		t.Run(test.name, func(t *testing.T) {
			results := map[string]reports.CanIDeployResult{
				"petstore_api": {
					Deployable: test.deployable,
					Trees: map[string]json.RawMessage{
						"endpoints": json.RawMessage(`{"/pets":{"get":{"200":[{"reason":"provider_resource_not_found","role":"consumer"}]}}}`),
					},
				},
			}
			var writer bytes.Buffer

			err := reports.WriteVerdict(&writer, "petstore_web 2.0.0", "petstore_web", "production", test.deployable, results, test.modules)

			require.EqualError(t, err, "the broker reported protocol endpoints, which this bidirekt doesn't know; update bidirekt")
			assert.NotErrorIs(t, err, reports.ErrSilent)
			assert.Empty(t, writer.String())
		})
	}
}

type rawTreeProtocol struct{}

func (this rawTreeProtocol) ReportKey() string {
	return "topics"
}

func (this rawTreeProtocol) WriteTree(report *strings.Builder, counterpart reports.Counterpart, tree json.RawMessage) error {
	_, err := fmt.Fprintf(report, "  topics of %s, checked for %s in %s: %s\n", counterpart.Name, counterpart.CheckedParticipant, counterpart.Environment, tree)
	return err
}

func (this rawTreeProtocol) ViolationLine(reports.Violation) (headline, explanation string, known bool) {
	return "", "", false
}

func (this rawTreeProtocol) SchemaKeys() map[string][]string {
	return nil
}
