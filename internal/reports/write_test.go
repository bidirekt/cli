package reports

import (
	"bytes"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWriteValidationFailedPrintsTheReportAndSilencesTheError(t *testing.T) {
	validationFailed := &ValidationFailedError{Message: "contract validation failed", Violations: []Violation{
		{Code: "schema.array_without_items", Path: "schemas;Pets", Source: "api.yaml"},
	}}
	var errWriter bytes.Buffer

	err := WriteValidationFailed(&errWriter, validationFailed)

	require.ErrorIs(t, err, ErrSilent)
	assert.Equal(t, `contract validation failed
  - api.yaml: array schema without items at schemas Pets
`, errWriter.String())
}

func TestWriteValidationFailedReturnsAnyOtherErrorAsIs(t *testing.T) {
	participantNotFound := errors.New("cannot validate contract on broker: participant not found")
	var errWriter bytes.Buffer

	err := WriteValidationFailed(&errWriter, participantNotFound)

	assert.Same(t, participantNotFound, err)
	assert.Empty(t, errWriter.String())
}

func TestWriteVerdictPrintsTheDeployableLine(t *testing.T) {
	var writer bytes.Buffer

	err := WriteVerdict(&writer, "petstore_web 3.0.0", "petstore_web", "production", true, map[string]CanIDeployResult{})

	require.NoError(t, err)
	assert.Equal(t, "petstore_web 3.0.0 can be deployed to production\n", writer.String())
}

func TestWriteVerdictPrintsTheReportAndSilencesTheError(t *testing.T) {
	results := map[string]CanIDeployResult{
		"petstore_reviews": {
			Deployable: false,
			Endpoints: map[string]map[string]map[string][]ContractBreak{
				"/reviews/*": {
					"get": {
						"200": {
							{Reason: "provider_resource_not_found", Role: "consumer"},
						},
					},
				},
			},
		},
	}
	var writer bytes.Buffer

	err := WriteVerdict(&writer, "petstore_web 2.0.0", "petstore_web", "production", false, results)

	require.ErrorIs(t, err, ErrSilent)
	assert.Equal(t, `petstore_web 2.0.0 cannot be deployed to production

petstore_reviews:
  GET /reviews/*
    response 200:
      - petstore_web calls GET /reviews/*, but petstore_reviews doesn't provide it → stop calling it, or wait until petstore_reviews publishes it
`, writer.String())
}
