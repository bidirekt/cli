package reports

import (
	"bytes"
	"testing"

	"github.com/bidirekt/cli/internal/paint"
	"github.com/stretchr/testify/assert"
)

func TestFormatValidationFailedReportRendersEveryCode(t *testing.T) {
	const allowedTypes = "object, array, map, string, integer, float, number, boolean"

	report := FormatValidationFailedReport(paint.For(&bytes.Buffer{}), "contract validation failed", []Violation{
		{Code: "key.unknown", Path: "provides;rest;/pets;patch", Source: "api.yaml", Details: map[string]string{"key": "patch"}},
		{Code: "value.invalid_kind", Path: "provides;rest;/pets;get", Source: "api.yaml", Details: map[string]string{"expected": "mapping", "got": "string"}},
		{Code: "endpoint.syntax", Path: "provides;rest;/users/{userId}", Source: "api.yaml", Details: map[string]string{"key": "/users/{userId}", "error": "dynamic path segments must use *"}},
		{Code: "service.name_syntax", Path: "consumes;Payments-API", Source: "api.yaml", Details: map[string]string{"key": "Payments-API", "error": "must be snake_case"}},
		{Code: "status.out_of_range", Path: "provides;rest;/pets;get;responses;999", Source: "api.yaml", Details: map[string]string{"key": "999", "error": "must be between 100 and 599"}},
		{Code: "schema.invalid_type", Path: "schemas;Pet;properties;id;type", Source: "api.yaml", Details: map[string]string{"value": "strng", "allowed": allowedTypes}},
		{Code: "schema.array_without_items", Path: "schemas;Pets", Source: "api.yaml"},
		{Code: "schema.map_without_values", Path: "schemas;Inventory", Source: "api.yaml"},
		{Code: "schema.object_without_properties", Path: "schemas;Owner;properties;tags", Source: "api.yaml"},
		{Code: "schema.key_not_allowed", Path: "schemas;Owner;properties;pets", Source: "api.yaml", Details: map[string]string{"key": "values", "type": "object"}},
		{Code: "schema.key_not_allowed", Path: "schemas;Owner;properties;pet", Source: "api.yaml", Details: map[string]string{"key": "type", "type": "ref"}},
		{Code: "schema.optional_not_allowed", Path: "schemas;Pet;properties;tags;items", Source: "api.yaml", Details: map[string]string{"position": "items"}},
		{Code: "resource.duplicate", Path: "provides;rest;/pets;get;responses;200", Source: "store.yaml", Details: map[string]string{"resource": "provides GET /pets 200", "declaredIn": "pets.yaml"}},
		{Code: "resource.type_conflict", Path: "consumes;payments;rest;/invoices;get;responses;200", Source: "b.yaml", Details: map[string]string{"resource": "consumes payments GET /invoices 200", "property": "$.id", "type": "integer", "declaredIn": "a.yaml", "declaredType": "string"}},
		{Code: "schema.duplicate", Path: "schemas;Pet", Source: "schemas.yaml", Details: map[string]string{"schema": "Pet", "declaredIn": "billing.yaml"}},
		{Code: "schema.unresolved_name", Path: "provides;rest;/pets;get;responses;200", Source: "pets.yaml", Details: map[string]string{"schema": "Pets", "resource": "provides GET /pets 200"}},
		{Code: "schema.unresolved_ref", Path: "schemas;Invoice;properties;payment", Source: "billing.yaml", Details: map[string]string{"schema": "Payment", "property": "Invoice.payment"}},
		{Code: "schema.too_deep", Path: "schemas;Owner", Source: "schemas.yaml", Details: map[string]string{"schema": "Owner", "maxDepth": "10"}},
		{Code: "something.new", Path: "provides;rest;/pets", Source: "api.yaml", Details: map[string]string{"hint": "x"}},
	})

	assert.Equal(t, `contract validation failed
  - api.yaml: unknown key "patch" at provides rest /pets
  - api.yaml: unexpected string at provides rest /pets get, expected mapping
  - api.yaml: invalid endpoint "/users/{userId}" at provides rest, dynamic path segments must use *
  - api.yaml: invalid service name "Payments-API" at consumes, must be snake_case
  - api.yaml: invalid status code "999" at provides rest /pets get responses, must be between 100 and 599
  - api.yaml: invalid value "strng" for "type" at schemas Pet properties id, expected one of: object, array, map, string, integer, float, number, boolean
  - api.yaml: array schema without items at schemas Pets
  - api.yaml: map schema without values at schemas Inventory
  - api.yaml: object schema without properties at schemas Owner properties tags, declare the keys you read, or use "type: map" with "values" if the keys vary
  - api.yaml: "values" is not allowed on an object schema at schemas Owner properties pets, use it with "type: map"
  - api.yaml: "type" is not allowed beside "ref" at schemas Owner properties pet
  - api.yaml: "optional" is not allowed on array items at schemas Pet properties tags items, mark the property that holds the array optional, or use "nullable: true" for an item that may be null
  - store.yaml: duplicate resource "provides GET /pets 200", also declared in pets.yaml
  - b.yaml: conflicting type for property "$.id" of consumes payments GET /invoices 200: integer here, string in a.yaml
  - schemas.yaml: duplicate schema "Pet", also declared in billing.yaml
  - pets.yaml: unresolved schema "Pets" referenced by provides GET /pets 200
  - billing.yaml: unresolved ref "Payment" in Invoice.payment
  - schemas.yaml: schema "Owner" is deeper than 10 levels
  - api.yaml: something.new at provides rest /pets (hint: x)
`, report)
}

func TestFormatValidationFailedReportOmitsTheSourceWhenEmpty(t *testing.T) {
	report := FormatValidationFailedReport(paint.For(&bytes.Buffer{}), "contract validation failed", []Violation{
		{Code: "schema.array_without_items", Path: "schemas;Pets"},
	})

	assert.Equal(t, `contract validation failed
  - array schema without items at schemas Pets
`, report)
}

func TestFormatValidationFailedReportPaintsTheHeadlineRed(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	t.Setenv("CLICOLOR_FORCE", "1")

	report := FormatValidationFailedReport(paint.For(&bytes.Buffer{}), "contract validation failed", []Violation{
		{Code: "schema.array_without_items", Path: "schemas;Pets"},
	})

	assert.Equal(t, "\x1b[31mcontract validation failed\x1b[0m\n"+
		"  - array schema without items at schemas Pets\n",
		report)
}

func TestFormatViolationLine(t *testing.T) {
	const allowedTypes = "object, array, map, string, integer, float, number, boolean"

	t.Run("key.unknown", func(t *testing.T) {
		headline, explanation := FormatViolationLine(Violation{
			Code:    "key.unknown",
			Path:    "provides;rest;/pets;patch",
			Source:  "api.yaml",
			Details: map[string]string{"key": "patch"},
		})

		assert.Equal(t, `unknown key "patch" at provides rest /pets`, headline)
		assert.Empty(t, explanation)
	})

	t.Run("key.unknown at the root has no location", func(t *testing.T) {
		headline, explanation := FormatViolationLine(Violation{
			Code:    "key.unknown",
			Path:    "version",
			Source:  "api.yaml",
			Details: map[string]string{"key": "version"},
		})

		assert.Equal(t, `unknown key "version"`, headline)
		assert.Empty(t, explanation)
	})

	t.Run("value.invalid_kind", func(t *testing.T) {
		headline, explanation := FormatViolationLine(Violation{
			Code:    "value.invalid_kind",
			Path:    "provides;rest;/pets;get",
			Source:  "api.yaml",
			Details: map[string]string{"expected": "mapping", "got": "string"},
		})

		assert.Equal(t, `unexpected string at provides rest /pets get, expected mapping`, headline)
		assert.Empty(t, explanation)
	})

	t.Run("endpoint.syntax", func(t *testing.T) {
		headline, explanation := FormatViolationLine(Violation{
			Code:    "endpoint.syntax",
			Path:    "provides;rest;/users/{userId}",
			Source:  "api.yaml",
			Details: map[string]string{"key": "/users/{userId}", "error": "dynamic path segments must use *"},
		})

		assert.Equal(t, `invalid endpoint "/users/{userId}" at provides rest`, headline)
		assert.Equal(t, `dynamic path segments must use *`, explanation)
	})

	t.Run("service.name_syntax", func(t *testing.T) {
		headline, explanation := FormatViolationLine(Violation{
			Code:    "service.name_syntax",
			Path:    "consumes;Payments-API",
			Source:  "api.yaml",
			Details: map[string]string{"key": "Payments-API", "error": "must be snake_case"},
		})

		assert.Equal(t, `invalid service name "Payments-API" at consumes`, headline)
		assert.Equal(t, `must be snake_case`, explanation)
	})

	t.Run("status.out_of_range", func(t *testing.T) {
		headline, explanation := FormatViolationLine(Violation{
			Code:    "status.out_of_range",
			Path:    "provides;rest;/pets;get;responses;999",
			Source:  "api.yaml",
			Details: map[string]string{"key": "999", "error": "must be between 100 and 599"},
		})

		assert.Equal(t, `invalid status code "999" at provides rest /pets get responses`, headline)
		assert.Equal(t, `must be between 100 and 599`, explanation)
	})

	t.Run("schema.invalid_type", func(t *testing.T) {
		headline, explanation := FormatViolationLine(Violation{
			Code:    "schema.invalid_type",
			Path:    "schemas;Pet;properties;id;type",
			Source:  "api.yaml",
			Details: map[string]string{"value": "strng", "allowed": allowedTypes},
		})

		assert.Equal(t, `invalid value "strng" for "type" at schemas Pet properties id`, headline)
		assert.Equal(t, `expected one of: object, array, map, string, integer, float, number, boolean`, explanation)
	})

	t.Run("schema.invalid_type without a value reports the type as missing", func(t *testing.T) {
		headline, explanation := FormatViolationLine(Violation{
			Code:    "schema.invalid_type",
			Path:    "schemas;Pet;properties;tags;items",
			Source:  "api.yaml",
			Details: map[string]string{"value": "", "allowed": allowedTypes},
		})

		assert.Equal(t, `missing "type" at schemas Pet properties tags items`, headline)
		assert.Equal(t, `expected one of: object, array, map, string, integer, float, number, boolean`, explanation)
	})

	t.Run("schema.array_without_items", func(t *testing.T) {
		headline, explanation := FormatViolationLine(Violation{
			Code:   "schema.array_without_items",
			Path:   "schemas;Pets",
			Source: "api.yaml",
		})

		assert.Equal(t, `array schema without items at schemas Pets`, headline)
		assert.Empty(t, explanation)
	})

	t.Run("schema.map_without_values", func(t *testing.T) {
		headline, explanation := FormatViolationLine(Violation{
			Code:   "schema.map_without_values",
			Path:   "schemas;Inventory",
			Source: "api.yaml",
		})

		assert.Equal(t, `map schema without values at schemas Inventory`, headline)
		assert.Empty(t, explanation)
	})

	t.Run("schema.object_without_properties suggests a map", func(t *testing.T) {
		headline, explanation := FormatViolationLine(Violation{
			Code:   "schema.object_without_properties",
			Path:   "schemas;Owner;properties;tags",
			Source: "api.yaml",
		})

		assert.Equal(t, `object schema without properties at schemas Owner properties tags`, headline)
		assert.Equal(t, `declare the keys you read, or use "type: map" with "values" if the keys vary`, explanation)
	})

	t.Run("schema.key_not_allowed values on an object", func(t *testing.T) {
		headline, explanation := FormatViolationLine(Violation{
			Code:    "schema.key_not_allowed",
			Path:    "schemas;Owner;properties;pets",
			Source:  "api.yaml",
			Details: map[string]string{"key": "values", "type": "object"},
		})

		assert.Equal(t, `"values" is not allowed on an object schema at schemas Owner properties pets`, headline)
		assert.Equal(t, `use it with "type: map"`, explanation)
	})

	t.Run("schema.key_not_allowed properties on a map", func(t *testing.T) {
		headline, explanation := FormatViolationLine(Violation{
			Code:    "schema.key_not_allowed",
			Path:    "schemas;Inventory",
			Source:  "api.yaml",
			Details: map[string]string{"key": "properties", "type": "map"},
		})

		assert.Equal(t, `"properties" is not allowed on a map schema at schemas Inventory`, headline)
		assert.Equal(t, `use it with "type: object"`, explanation)
	})

	t.Run("schema.key_not_allowed items on a string", func(t *testing.T) {
		headline, explanation := FormatViolationLine(Violation{
			Code:    "schema.key_not_allowed",
			Path:    "schemas;Pet;properties;name",
			Source:  "api.yaml",
			Details: map[string]string{"key": "items", "type": "string"},
		})

		assert.Equal(t, `"items" is not allowed on a string schema at schemas Pet properties name`, headline)
		assert.Equal(t, `use it with "type: array"`, explanation)
	})

	t.Run("schema.key_not_allowed values on an integer", func(t *testing.T) {
		headline, explanation := FormatViolationLine(Violation{
			Code:    "schema.key_not_allowed",
			Path:    "schemas;Pet;properties;age",
			Source:  "api.yaml",
			Details: map[string]string{"key": "values", "type": "integer"},
		})

		assert.Equal(t, `"values" is not allowed on an integer schema at schemas Pet properties age`, headline)
		assert.Equal(t, `use it with "type: map"`, explanation)
	})

	t.Run("schema.key_not_allowed beside ref", func(t *testing.T) {
		headline, explanation := FormatViolationLine(Violation{
			Code:    "schema.key_not_allowed",
			Path:    "schemas;Owner;properties;pet",
			Source:  "api.yaml",
			Details: map[string]string{"key": "type", "type": "ref"},
		})

		assert.Equal(t, `"type" is not allowed beside "ref" at schemas Owner properties pet`, headline)
		assert.Empty(t, explanation)
	})

	t.Run("schema.key_not_allowed beside ref with a key another type owns", func(t *testing.T) {
		headline, explanation := FormatViolationLine(Violation{
			Code:    "schema.key_not_allowed",
			Path:    "schemas;Owner;properties;pet",
			Source:  "api.yaml",
			Details: map[string]string{"key": "values", "type": "ref"},
		})

		assert.Equal(t, `"values" is not allowed beside "ref" at schemas Owner properties pet`, headline)
		assert.Empty(t, explanation)
	})

	t.Run("schema.optional_not_allowed on array items", func(t *testing.T) {
		headline, explanation := FormatViolationLine(Violation{
			Code:    "schema.optional_not_allowed",
			Path:    "schemas;Pet;properties;tags;items",
			Source:  "api.yaml",
			Details: map[string]string{"position": "items"},
		})

		assert.Equal(t, `"optional" is not allowed on array items at schemas Pet properties tags items`, headline)
		assert.Equal(t, `mark the property that holds the array optional, or use "nullable: true" for an item that may be null`, explanation)
	})

	t.Run("schema.optional_not_allowed on map values", func(t *testing.T) {
		headline, explanation := FormatViolationLine(Violation{
			Code:    "schema.optional_not_allowed",
			Path:    "schemas;Pet;properties;scores;values",
			Source:  "api.yaml",
			Details: map[string]string{"position": "values"},
		})

		assert.Equal(t, `"optional" is not allowed on map values at schemas Pet properties scores values`, headline)
		assert.Equal(t, `mark the property that holds the map optional, or use "nullable: true" for a value that may be null`, explanation)
	})

	t.Run("schema.optional_not_allowed on a named schema", func(t *testing.T) {
		headline, explanation := FormatViolationLine(Violation{
			Code:    "schema.optional_not_allowed",
			Path:    "schemas;Pet",
			Source:  "api.yaml",
			Details: map[string]string{"position": "schema"},
		})

		assert.Equal(t, `"optional" is not allowed on a schema at schemas Pet`, headline)
		assert.Equal(t, `mark optional the property that uses it, beside its "ref"`, explanation)
	})

	t.Run("schema.optional_not_allowed at an unknown position falls back to the code with its details", func(t *testing.T) {
		headline, explanation := FormatViolationLine(Violation{
			Code:    "schema.optional_not_allowed",
			Path:    "schemas;Pet;properties;owner",
			Source:  "api.yaml",
			Details: map[string]string{"position": "property"},
		})

		assert.Equal(t, `schema.optional_not_allowed at schemas Pet properties owner (position: property)`, headline)
		assert.Empty(t, explanation)
	})

	t.Run("resource.duplicate", func(t *testing.T) {
		headline, explanation := FormatViolationLine(Violation{
			Code:    "resource.duplicate",
			Path:    "provides;rest;/pets;get;responses;200",
			Source:  "store.yaml",
			Details: map[string]string{"resource": "provides GET /pets 200", "declaredIn": "pets.yaml"},
		})

		assert.Equal(t, `duplicate resource "provides GET /pets 200", also declared in pets.yaml`, headline)
		assert.Empty(t, explanation)
	})

	t.Run("resource.duplicate in the same file", func(t *testing.T) {
		headline, explanation := FormatViolationLine(Violation{
			Code:    "resource.duplicate",
			Path:    "provides;rest;/pets;get;responses;200",
			Source:  "pets.yaml",
			Details: map[string]string{"resource": "provides GET /pets 200", "declaredIn": "pets.yaml"},
		})

		assert.Equal(t, `duplicate resource "provides GET /pets 200", declared twice`, headline)
		assert.Empty(t, explanation)
	})

	t.Run("resource.type_conflict", func(t *testing.T) {
		headline, explanation := FormatViolationLine(Violation{
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
		headline, explanation := FormatViolationLine(Violation{
			Code:    "schema.duplicate",
			Path:    "schemas;Pet",
			Source:  "schemas.yaml",
			Details: map[string]string{"schema": "Pet", "declaredIn": "billing.yaml"},
		})

		assert.Equal(t, `duplicate schema "Pet", also declared in billing.yaml`, headline)
		assert.Empty(t, explanation)
	})

	t.Run("schema.duplicate in the same file", func(t *testing.T) {
		headline, explanation := FormatViolationLine(Violation{
			Code:    "schema.duplicate",
			Path:    "schemas;Pet",
			Source:  "schemas.yaml",
			Details: map[string]string{"schema": "Pet", "declaredIn": "schemas.yaml"},
		})

		assert.Equal(t, `duplicate schema "Pet", declared twice`, headline)
		assert.Empty(t, explanation)
	})

	t.Run("schema.unresolved_name", func(t *testing.T) {
		headline, explanation := FormatViolationLine(Violation{
			Code:    "schema.unresolved_name",
			Path:    "provides;rest;/pets;get;responses;200",
			Source:  "pets.yaml",
			Details: map[string]string{"schema": "Pets", "resource": "provides GET /pets 200"},
		})

		assert.Equal(t, `unresolved schema "Pets" referenced by provides GET /pets 200`, headline)
		assert.Empty(t, explanation)
	})

	t.Run("schema.unresolved_ref", func(t *testing.T) {
		headline, explanation := FormatViolationLine(Violation{
			Code:    "schema.unresolved_ref",
			Path:    "schemas;Invoice;properties;payment",
			Source:  "billing.yaml",
			Details: map[string]string{"schema": "Payment", "property": "Invoice.payment"},
		})

		assert.Equal(t, `unresolved ref "Payment" in Invoice.payment`, headline)
		assert.Empty(t, explanation)
	})

	t.Run("schema.too_deep", func(t *testing.T) {
		headline, explanation := FormatViolationLine(Violation{
			Code:    "schema.too_deep",
			Path:    "schemas;Owner",
			Source:  "schemas.yaml",
			Details: map[string]string{"schema": "Owner", "maxDepth": "10"},
		})

		assert.Equal(t, `schema "Owner" is deeper than 10 levels`, headline)
		assert.Empty(t, explanation)
	})

	t.Run("unknown code falls back to the code with its sorted details", func(t *testing.T) {
		headline, explanation := FormatViolationLine(Violation{
			Code:    "something.new",
			Path:    "provides;rest;/pets",
			Source:  "api.yaml",
			Details: map[string]string{"hint": "x", "area": "y"},
		})

		assert.Equal(t, `something.new at provides rest /pets (area: y, hint: x)`, headline)
		assert.Empty(t, explanation)
	})

	t.Run("unknown code without details", func(t *testing.T) {
		headline, explanation := FormatViolationLine(Violation{
			Code:   "something.new",
			Path:   "provides;rest;/pets",
			Source: "api.yaml",
		})

		assert.Equal(t, `something.new at provides rest /pets`, headline)
		assert.Empty(t, explanation)
	})

	t.Run("unknown code without path", func(t *testing.T) {
		headline, explanation := FormatViolationLine(Violation{
			Code:    "something.new",
			Source:  "api.yaml",
			Details: map[string]string{"hint": "x"},
		})

		assert.Equal(t, `something.new (hint: x)`, headline)
		assert.Empty(t, explanation)
	})

	t.Run("unknown code without path or details", func(t *testing.T) {
		headline, explanation := FormatViolationLine(Violation{
			Code:   "something.new",
			Source: "api.yaml",
		})

		assert.Equal(t, `something.new`, headline)
		assert.Empty(t, explanation)
	})
}
