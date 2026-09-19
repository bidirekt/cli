package publish_contract

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatValidationFailedReportRendersEveryCode(t *testing.T) {
	const allowedTypes = "object, array, string, integer, float, boolean"

	report := formatValidationFailedReport("contract validation failed", []Violation{
		{Code: "key.unknown", Path: "provides;rest;/pets;patch", Source: "api.yaml", Details: map[string]string{"key": "patch"}},
		{Code: "value.invalid_kind", Path: "provides;rest;/pets;get", Source: "api.yaml", Details: map[string]string{"expected": "mapping", "got": "string"}},
		{Code: "endpoint.syntax", Path: "provides;rest;/users/{userId}", Source: "api.json", Details: map[string]string{"key": "/users/{userId}", "error": "dynamic path segments must use *"}},
		{Code: "service.name_syntax", Path: "consumes;Payments-API", Source: "api.json", Details: map[string]string{"key": "Payments-API", "error": "must be snake_case"}},
		{Code: "status.out_of_range", Path: "provides;rest;/pets;get;responses;999", Source: "api.yaml", Details: map[string]string{"key": "999", "error": "must be between 100 and 599"}},
		{Code: "schema.invalid_type", Path: "schemas;Pet;properties;id;type", Source: "api.yaml", Details: map[string]string{"value": "strng", "allowed": allowedTypes}},
		{Code: "schema.array_without_items", Path: "schemas;Pets", Source: "api.yaml"},
		{Code: "resource.duplicate", Path: "provides;rest;/pets;get;responses;200", Source: "store.yaml", Details: map[string]string{"resource": "provides GET /pets 200", "declaredIn": "pets.yaml"}},
		{Code: "resource.type_conflict", Path: "consumes;payments;rest;/invoices;get;responses;200", Source: "b.yaml", Details: map[string]string{"resource": "consumes payments GET /invoices 200", "property": "$.id", "type": "integer", "declaredIn": "a.yaml", "declaredType": "string"}},
		{Code: "schema.duplicate", Path: "schemas;Pet", Source: "schemas.yaml", Details: map[string]string{"schema": "Pet", "declaredIn": "billing.yaml"}},
		{Code: "schema.unresolved_name", Path: "provides;rest;/pets;get;responses;200", Source: "pets.yaml", Details: map[string]string{"schema": "Pets", "resource": "provides GET /pets 200"}},
		{Code: "schema.unresolved_ref", Path: "schemas;Invoice;properties;payment", Source: "billing.yaml", Details: map[string]string{"schema": "Payment", "property": "Invoice.payment"}},
		{Code: "schema.too_deep", Path: "schemas;Owner", Source: "schemas.yaml", Details: map[string]string{"schema": "Owner", "maxDepth": "10"}},
		{Code: "something.new", Path: "provides;rest;/pets", Source: "api.yaml", Details: map[string]string{"hint": "x"}},
	})

	assert.Equal(t, `❌ contract validation failed
  - api.yaml: unknown key "patch" at provides;rest;/pets;patch
  - api.yaml: unexpected string at provides;rest;/pets;get, expected mapping
  - api.json: invalid endpoint "/users/{userId}" at provides;rest;/users/{userId}
      dynamic path segments must use *
  - api.json: invalid service name "Payments-API" at consumes;Payments-API
      must be snake_case
  - api.yaml: invalid status code "999" at provides;rest;/pets;get;responses;999
      must be between 100 and 599
  - api.yaml: invalid value "strng" for "type" at schemas;Pet;properties;id;type
      expected one of: object, array, string, integer, float, boolean
  - api.yaml: array schema without items at schemas;Pets
  - store.yaml: duplicate resource "provides GET /pets 200", also declared in pets.yaml
  - b.yaml: conflicting type for property "$.id" of consumes payments GET /invoices 200: integer here, string in a.yaml
  - schemas.yaml: duplicate schema "Pet", also declared in billing.yaml
  - pets.yaml: unresolved schema "Pets" referenced by provides GET /pets 200
  - billing.yaml: unresolved ref "Payment" in Invoice.payment
  - schemas.yaml: schema "Owner" is deeper than 10 levels
  - api.yaml: something.new at provides;rest;/pets (hint: x)
`, report)
}

func TestFormatValidationFailedReportOmitsTheSourceWhenEmpty(t *testing.T) {
	report := formatValidationFailedReport("contract validation failed", []Violation{
		{Code: "schema.array_without_items", Path: "schemas;Pets"},
	})

	assert.Equal(t, `❌ contract validation failed
  - array schema without items at schemas;Pets
`, report)
}

func TestFormatViolationLine(t *testing.T) {
	const allowedTypes = "object, array, string, integer, float, boolean"

	t.Run("key.unknown", func(t *testing.T) {
		headline, explanation := formatViolationLine(Violation{
			Code:    "key.unknown",
			Path:    "provides;rest;/pets;patch",
			Source:  "api.yaml",
			Details: map[string]string{"key": "patch"},
		})

		assert.Equal(t, `unknown key "patch" at provides;rest;/pets;patch`, headline)
		assert.Empty(t, explanation)
	})

	t.Run("value.invalid_kind", func(t *testing.T) {
		headline, explanation := formatViolationLine(Violation{
			Code:    "value.invalid_kind",
			Path:    "provides;rest;/pets;get",
			Source:  "api.yaml",
			Details: map[string]string{"expected": "mapping", "got": "string"},
		})

		assert.Equal(t, `unexpected string at provides;rest;/pets;get, expected mapping`, headline)
		assert.Empty(t, explanation)
	})

	t.Run("endpoint.syntax", func(t *testing.T) {
		headline, explanation := formatViolationLine(Violation{
			Code:    "endpoint.syntax",
			Path:    "provides;rest;/users/{userId}",
			Source:  "api.json",
			Details: map[string]string{"key": "/users/{userId}", "error": "dynamic path segments must use *"},
		})

		assert.Equal(t, `invalid endpoint "/users/{userId}" at provides;rest;/users/{userId}`, headline)
		assert.Equal(t, `dynamic path segments must use *`, explanation)
	})

	t.Run("service.name_syntax", func(t *testing.T) {
		headline, explanation := formatViolationLine(Violation{
			Code:    "service.name_syntax",
			Path:    "consumes;Payments-API",
			Source:  "api.json",
			Details: map[string]string{"key": "Payments-API", "error": "must be snake_case"},
		})

		assert.Equal(t, `invalid service name "Payments-API" at consumes;Payments-API`, headline)
		assert.Equal(t, `must be snake_case`, explanation)
	})

	t.Run("status.out_of_range", func(t *testing.T) {
		headline, explanation := formatViolationLine(Violation{
			Code:    "status.out_of_range",
			Path:    "provides;rest;/pets;get;responses;999",
			Source:  "api.yaml",
			Details: map[string]string{"key": "999", "error": "must be between 100 and 599"},
		})

		assert.Equal(t, `invalid status code "999" at provides;rest;/pets;get;responses;999`, headline)
		assert.Equal(t, `must be between 100 and 599`, explanation)
	})

	t.Run("schema.invalid_type", func(t *testing.T) {
		headline, explanation := formatViolationLine(Violation{
			Code:    "schema.invalid_type",
			Path:    "schemas;Pet;properties;id;type",
			Source:  "api.yaml",
			Details: map[string]string{"value": "strng", "allowed": allowedTypes},
		})

		assert.Equal(t, `invalid value "strng" for "type" at schemas;Pet;properties;id;type`, headline)
		assert.Equal(t, `expected one of: object, array, string, integer, float, boolean`, explanation)
	})

	t.Run("schema.invalid_type without a value reports the type as missing", func(t *testing.T) {
		headline, explanation := formatViolationLine(Violation{
			Code:    "schema.invalid_type",
			Path:    "schemas;Pet;properties;tags;items",
			Source:  "api.yaml",
			Details: map[string]string{"value": "", "allowed": allowedTypes},
		})

		assert.Equal(t, `missing "type" at schemas;Pet;properties;tags;items`, headline)
		assert.Equal(t, `expected one of: object, array, string, integer, float, boolean`, explanation)
	})

	t.Run("schema.array_without_items", func(t *testing.T) {
		headline, explanation := formatViolationLine(Violation{
			Code:   "schema.array_without_items",
			Path:   "schemas;Pets",
			Source: "api.yaml",
		})

		assert.Equal(t, `array schema without items at schemas;Pets`, headline)
		assert.Empty(t, explanation)
	})

	t.Run("resource.duplicate", func(t *testing.T) {
		headline, explanation := formatViolationLine(Violation{
			Code:    "resource.duplicate",
			Path:    "provides;rest;/pets;get;responses;200",
			Source:  "store.yaml",
			Details: map[string]string{"resource": "provides GET /pets 200", "declaredIn": "pets.yaml"},
		})

		assert.Equal(t, `duplicate resource "provides GET /pets 200", also declared in pets.yaml`, headline)
		assert.Empty(t, explanation)
	})

	t.Run("resource.duplicate in the same file", func(t *testing.T) {
		headline, explanation := formatViolationLine(Violation{
			Code:    "resource.duplicate",
			Path:    "provides;rest;/pets;get;responses;200",
			Source:  "pets.yaml",
			Details: map[string]string{"resource": "provides GET /pets 200", "declaredIn": "pets.yaml"},
		})

		assert.Equal(t, `duplicate resource "provides GET /pets 200", declared twice`, headline)
		assert.Empty(t, explanation)
	})

	t.Run("resource.type_conflict", func(t *testing.T) {
		headline, explanation := formatViolationLine(Violation{
			Code:   "resource.type_conflict",
			Path:   "consumes;payments;rest;/invoices;get;responses;200",
			Source: "b.yaml",
			Details: map[string]string{
				"resource":     "consumes payments GET /invoices 200",
				"property":     "$.id",
				"type":         "integer",
				"declaredIn":   "a.yaml",
				"declaredType": "string",
			},
		})

		assert.Equal(t, `conflicting type for property "$.id" of consumes payments GET /invoices 200: integer here, string in a.yaml`, headline)
		assert.Empty(t, explanation)
	})

	t.Run("schema.duplicate", func(t *testing.T) {
		headline, explanation := formatViolationLine(Violation{
			Code:    "schema.duplicate",
			Path:    "schemas;Pet",
			Source:  "schemas.yaml",
			Details: map[string]string{"schema": "Pet", "declaredIn": "billing.yaml"},
		})

		assert.Equal(t, `duplicate schema "Pet", also declared in billing.yaml`, headline)
		assert.Empty(t, explanation)
	})

	t.Run("schema.duplicate in the same file", func(t *testing.T) {
		headline, explanation := formatViolationLine(Violation{
			Code:    "schema.duplicate",
			Path:    "schemas;Pet",
			Source:  "schemas.yaml",
			Details: map[string]string{"schema": "Pet", "declaredIn": "schemas.yaml"},
		})

		assert.Equal(t, `duplicate schema "Pet", declared twice`, headline)
		assert.Empty(t, explanation)
	})

	t.Run("schema.unresolved_name", func(t *testing.T) {
		headline, explanation := formatViolationLine(Violation{
			Code:    "schema.unresolved_name",
			Path:    "provides;rest;/pets;get;responses;200",
			Source:  "pets.yaml",
			Details: map[string]string{"schema": "Pets", "resource": "provides GET /pets 200"},
		})

		assert.Equal(t, `unresolved schema "Pets" referenced by provides GET /pets 200`, headline)
		assert.Empty(t, explanation)
	})

	t.Run("schema.unresolved_ref", func(t *testing.T) {
		headline, explanation := formatViolationLine(Violation{
			Code:    "schema.unresolved_ref",
			Path:    "schemas;Invoice;properties;payment",
			Source:  "billing.yaml",
			Details: map[string]string{"schema": "Payment", "property": "Invoice.payment"},
		})

		assert.Equal(t, `unresolved ref "Payment" in Invoice.payment`, headline)
		assert.Empty(t, explanation)
	})

	t.Run("schema.too_deep", func(t *testing.T) {
		headline, explanation := formatViolationLine(Violation{
			Code:    "schema.too_deep",
			Path:    "schemas;Owner",
			Source:  "schemas.yaml",
			Details: map[string]string{"schema": "Owner", "maxDepth": "10"},
		})

		assert.Equal(t, `schema "Owner" is deeper than 10 levels`, headline)
		assert.Empty(t, explanation)
	})

	t.Run("unknown code falls back to the code with its sorted details", func(t *testing.T) {
		headline, explanation := formatViolationLine(Violation{
			Code:    "something.new",
			Path:    "provides;rest;/pets",
			Source:  "api.yaml",
			Details: map[string]string{"hint": "x", "area": "y"},
		})

		assert.Equal(t, `something.new at provides;rest;/pets (area: y, hint: x)`, headline)
		assert.Empty(t, explanation)
	})

	t.Run("unknown code without details", func(t *testing.T) {
		headline, explanation := formatViolationLine(Violation{
			Code:   "something.new",
			Path:   "provides;rest;/pets",
			Source: "api.yaml",
		})

		assert.Equal(t, `something.new at provides;rest;/pets`, headline)
		assert.Empty(t, explanation)
	})

	t.Run("unknown code without path", func(t *testing.T) {
		headline, explanation := formatViolationLine(Violation{
			Code:    "something.new",
			Source:  "api.yaml",
			Details: map[string]string{"hint": "x"},
		})

		assert.Equal(t, `something.new (hint: x)`, headline)
		assert.Empty(t, explanation)
	})

	t.Run("unknown code without path or details", func(t *testing.T) {
		headline, explanation := formatViolationLine(Violation{
			Code:   "something.new",
			Source: "api.yaml",
		})

		assert.Equal(t, `something.new`, headline)
		assert.Empty(t, explanation)
	})
}
