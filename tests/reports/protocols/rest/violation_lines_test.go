package rest_test

import (
	"bytes"
	"testing"

	"github.com/bidirekt/cli/internal/paint"
	"github.com/bidirekt/cli/internal/reports"
	"github.com/bidirekt/cli/internal/reports/protocols/rest"
	"github.com/stretchr/testify/assert"
)

func TestFormatValidationFailedReportRendersTheRestCodes(t *testing.T) {
	modules := []reports.Module{{Report: rest.New()}}

	report := reports.FormatValidationFailedReport(paint.For(&bytes.Buffer{}), "contract validation failed", []reports.Violation{
		{Code: "endpoint.syntax", Path: "provides;rest;/users/{userId}", Source: "api.yaml", Details: map[string]string{"key": "/users/{userId}", "error": "dynamic path segments must use *"}},
		{Code: "status.out_of_range", Path: "provides;rest;/pets;get;responses;999", Source: "api.yaml", Details: map[string]string{"key": "999", "error": "must be between 100 and 599"}},
	}, modules)

	assert.Equal(t, `contract validation failed
  - api.yaml: invalid endpoint "/users/{userId}" in provides, dynamic path segments must use *
  - api.yaml: invalid status code "999" at provides rest /pets get responses, must be between 100 and 599
`, report)
}

func TestFormatViolationLine(t *testing.T) {
	modules := []reports.Module{{Report: rest.New()}}

	t.Run("endpoint.syntax", func(t *testing.T) {
		headline, explanation := reports.FormatViolationLine(reports.Violation{
			Code:    "endpoint.syntax",
			Path:    "provides;rest;/users/{userId}",
			Source:  "api.yaml",
			Details: map[string]string{"key": "/users/{userId}", "error": "dynamic path segments must use *"},
		}, modules)

		assert.Equal(t, `invalid endpoint "/users/{userId}" in provides`, headline)
		assert.Equal(t, `dynamic path segments must use *`, explanation)
	})

	t.Run("endpoint.syntax with a suggestion shows the corrected path", func(t *testing.T) {
		headline, explanation := reports.FormatViolationLine(reports.Violation{
			Code:    "endpoint.syntax",
			Path:    "provides;rest;/adoptions/{petId}",
			Source:  "contracts/generated.yaml",
			Details: map[string]string{"key": "/adoptions/{petId}", "error": "dynamic path segments must use *", "suggestion": "/adoptions/*"},
		}, modules)

		assert.Equal(t, `invalid endpoint "/adoptions/{petId}" in provides`, headline)
		assert.Equal(t, `write "/adoptions/*" (dynamic path segments must use *)`, explanation)
	})

	t.Run("endpoint.syntax without a suggestion keeps the broker error", func(t *testing.T) {
		headline, explanation := reports.FormatViolationLine(reports.Violation{
			Code:    "endpoint.syntax",
			Path:    "provides;rest;/pets/p*",
			Source:  "api.yaml",
			Details: map[string]string{"key": "/pets/p*", "error": "dynamic path segments must use *"},
		}, modules)

		assert.Equal(t, `invalid endpoint "/pets/p*" in provides`, headline)
		assert.Equal(t, `dynamic path segments must use *`, explanation)
	})

	t.Run("endpoint.syntax under consumes names the service", func(t *testing.T) {
		headline, explanation := reports.FormatViolationLine(reports.Violation{
			Code:    "endpoint.syntax",
			Path:    "consumes;petstore;rest;/adoptions/{petId}",
			Source:  "api.yaml",
			Details: map[string]string{"key": "/adoptions/{petId}", "error": "dynamic path segments must use *", "suggestion": "/adoptions/*"},
		}, modules)

		assert.Equal(t, `invalid endpoint "/adoptions/{petId}" in consumes petstore`, headline)
		assert.Equal(t, `write "/adoptions/*" (dynamic path segments must use *)`, explanation)
	})

	t.Run("endpoint.syntax at an unexpected path falls back to the full location", func(t *testing.T) {
		headline, explanation := reports.FormatViolationLine(reports.Violation{
			Code:    "endpoint.syntax",
			Path:    "provides;graphql;/pets/{petId}",
			Source:  "api.yaml",
			Details: map[string]string{"key": "/pets/{petId}", "error": "dynamic path segments must use *"},
		}, modules)

		assert.Equal(t, `invalid endpoint "/pets/{petId}" at provides graphql`, headline)
		assert.Equal(t, `dynamic path segments must use *`, explanation)
	})

	t.Run("status.out_of_range", func(t *testing.T) {
		headline, explanation := reports.FormatViolationLine(reports.Violation{
			Code:    "status.out_of_range",
			Path:    "provides;rest;/pets;get;responses;999",
			Source:  "api.yaml",
			Details: map[string]string{"key": "999", "error": "must be between 100 and 599"},
		}, modules)

		assert.Equal(t, `invalid status code "999" at provides rest /pets get responses`, headline)
		assert.Equal(t, `must be between 100 and 599`, explanation)
	})
}
