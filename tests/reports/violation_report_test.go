package reports_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"testing"

	"github.com/bidirekt/cli/internal/paint"
	"github.com/bidirekt/cli/internal/reports"
	"github.com/stretchr/testify/assert"
)

func TestFormatValidationFailedReportRendersEveryCode(t *testing.T) {
	const allowedTypes = "object, array, map, string, integer, float, number, boolean"

	report := reports.FormatValidationFailedReport(paint.For(&bytes.Buffer{}), "contract validation failed", []reports.Violation{
		{Code: "key.unknown", Path: "provides;rest;/pets;patch", Source: "api.yaml", Details: map[string]string{"key": "patch"}},
		{Code: "value.invalid_kind", Path: "provides;rest;/pets;get", Source: "api.yaml", Details: map[string]string{"expected": "mapping", "got": "string"}},
		{Code: "service.name_syntax", Path: "consumes;Payments-API", Source: "api.yaml", Details: map[string]string{"key": "Payments-API", "error": "must be snake_case"}},
		{Code: "property.name_syntax", Path: "schemas;Pet;properties;user.name", Source: "api.yaml", Details: map[string]string{"key": "user.name", "error": `must not contain ".", "[", "]", "{", "}" or ";"`}},
		{Code: "schema.name_syntax", Path: "schemas;Pet.v2", Source: "api.yaml", Details: map[string]string{"key": "Pet.v2", "error": `must not contain ".", "[", "]", "{", "}" or ";"`}},
		{Code: "schema.description_syntax", Path: "schemas;Pet;properties;weight;description", Source: "api.yaml", Details: map[string]string{"value": "In kilograms.", "error": "must be ASCII letters and digits, with single spaces between words"}},
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
		{Code: "members.required", Path: "schemas;Pet;properties;status", Source: "api.yaml", Details: map[string]string{"type": "enum"}},
		{Code: "members.empty", Path: "schemas;Order;properties;payment;members", Source: "api.yaml", Details: map[string]string{"type": "union"}},
		{Code: "members.duplicate", Path: "schemas;Pet;properties;status;members;2", Source: "api.yaml", Details: map[string]string{"value": "sold"}},
		{Code: "members.not_text", Path: "schemas;Pet;properties;status;members;1", Source: "api.yaml"},
		{Code: "member.ref_required", Path: "schemas;Order;properties;payment;members;card", Source: "api.yaml", Details: map[string]string{"member": "card"}},
		{Code: "member.name_syntax", Path: "schemas;Order;properties;payment;members;card(x)", Source: "api.yaml", Details: map[string]string{"key": "card(x)", "error": `must not contain ".", "[", "]", "{", "}", ";", "(" or ")"`}},
		{Code: "something.new", Path: "provides;rest;/pets", Source: "api.yaml", Details: map[string]string{"hint": "x"}},
	}, nil)

	assert.Equal(t, `contract validation failed
  - api.yaml: unknown key "patch" at provides rest /pets
  - api.yaml: unexpected string at provides rest /pets get, expected mapping
  - api.yaml: invalid service name "Payments-API" at consumes, must be snake_case
  - api.yaml: invalid property name "user.name" at schemas Pet properties, must not contain ".", "[", "]", "{", "}" or ";"
  - api.yaml: invalid schema name "Pet.v2" at schemas, must not contain ".", "[", "]", "{", "}" or ";"
  - api.yaml: invalid description "In kilograms." at schemas Pet properties weight, must be ASCII letters and digits, with single spaces between words
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
  - api.yaml: "type: enum" needs "members" at schemas Pet properties status: list its values
  - api.yaml: "members" is empty at schemas Order properties payment: list at least one member
  - api.yaml: value "sold" is repeated in "members" at schemas Pet properties status
  - api.yaml: each value in "members" must be non-empty text at schemas Pet properties status
  - api.yaml: member "card" needs a "ref" to a schema at schemas Order properties payment
  - api.yaml: member key "card(x)" must not contain ".", "[", "]", "{", "}", ";", "(" or ")" at schemas Order properties payment
  - api.yaml: something.new at provides rest /pets (hint: x)
`, report)
}

func TestFormatValidationFailedReportOmitsTheSourceWhenEmpty(t *testing.T) {
	report := reports.FormatValidationFailedReport(paint.For(&bytes.Buffer{}), "contract validation failed", []reports.Violation{
		{Code: "schema.array_without_items", Path: "schemas;Pets"},
	}, nil)

	assert.Equal(t, `contract validation failed
  - array schema without items at schemas Pets
`, report)
}

func TestFormatValidationFailedReportPaintsTheHeadlineRed(t *testing.T) {
	t.Setenv("NO_COLOR", "")
	t.Setenv("CLICOLOR_FORCE", "1")

	report := reports.FormatValidationFailedReport(paint.For(&bytes.Buffer{}), "contract validation failed", []reports.Violation{
		{Code: "schema.array_without_items", Path: "schemas;Pets"},
	}, nil)

	assert.Equal(t, "\x1b[31mcontract validation failed\x1b[0m\n"+
		"  - array schema without items at schemas Pets\n",
		report)
}

func TestFormatViolationLine(t *testing.T) {
	const allowedTypes = "object, array, map, string, integer, float, number, boolean"
	const pathSafeNameError = `must not contain ".", "[", "]", "{", "}" or ";"`

	t.Run("key.unknown", func(t *testing.T) {
		headline, explanation := reports.FormatViolationLine(reports.Violation{
			Code:    "key.unknown",
			Path:    "provides;rest;/pets;patch",
			Source:  "api.yaml",
			Details: map[string]string{"key": "patch"},
		}, nil)

		assert.Equal(t, `unknown key "patch" at provides rest /pets`, headline)
		assert.Empty(t, explanation)
	})

	t.Run("key.unknown at the root has no location", func(t *testing.T) {
		headline, explanation := reports.FormatViolationLine(reports.Violation{
			Code:    "key.unknown",
			Path:    "version",
			Source:  "api.yaml",
			Details: map[string]string{"key": "version"},
		}, nil)

		assert.Equal(t, `unknown key "version"`, headline)
		assert.Empty(t, explanation)
	})

	t.Run("value.invalid_kind", func(t *testing.T) {
		headline, explanation := reports.FormatViolationLine(reports.Violation{
			Code:    "value.invalid_kind",
			Path:    "provides;rest;/pets;get",
			Source:  "api.yaml",
			Details: map[string]string{"expected": "mapping", "got": "string"},
		}, nil)

		assert.Equal(t, `unexpected string at provides rest /pets get, expected mapping`, headline)
		assert.Empty(t, explanation)
	})

	t.Run("service.name_syntax", func(t *testing.T) {
		headline, explanation := reports.FormatViolationLine(reports.Violation{
			Code:    "service.name_syntax",
			Path:    "consumes;Payments-API",
			Source:  "api.yaml",
			Details: map[string]string{"key": "Payments-API", "error": "must be snake_case"},
		}, nil)

		assert.Equal(t, `invalid service name "Payments-API" at consumes`, headline)
		assert.Equal(t, `must be snake_case`, explanation)
	})

	t.Run("service.name_syntax with a semicolon in the key leaves the whole key out of the location", func(t *testing.T) {
		headline, explanation := reports.FormatViolationLine(reports.Violation{
			Code:    "service.name_syntax",
			Path:    "consumes;Bad;Svc",
			Source:  "api.yaml",
			Details: map[string]string{"key": "Bad;Svc", "error": "must be snake_case"},
		}, nil)

		assert.Equal(t, `invalid service name "Bad;Svc" at consumes`, headline)
		assert.Equal(t, `must be snake_case`, explanation)
	})

	t.Run("property.name_syntax", func(t *testing.T) {
		headline, explanation := reports.FormatViolationLine(reports.Violation{
			Code:    "property.name_syntax",
			Path:    "schemas;Pet;properties;user.name",
			Source:  "api.yaml",
			Details: map[string]string{"key": "user.name", "error": pathSafeNameError},
		}, nil)

		assert.Equal(t, `invalid property name "user.name" at schemas Pet properties`, headline)
		assert.Equal(t, pathSafeNameError, explanation)
	})

	t.Run("property.name_syntax with an empty name", func(t *testing.T) {
		headline, explanation := reports.FormatViolationLine(reports.Violation{
			Code:    "property.name_syntax",
			Path:    "schemas;Pet;properties;",
			Source:  "api.yaml",
			Details: map[string]string{"key": "", "error": "must not be empty"},
		}, nil)

		assert.Equal(t, `invalid property name "" at schemas Pet properties`, headline)
		assert.Equal(t, `must not be empty`, explanation)
	})

	t.Run("property.name_syntax with a semicolon in the name leaves the whole name out of the location", func(t *testing.T) {
		headline, explanation := reports.FormatViolationLine(reports.Violation{
			Code:    "property.name_syntax",
			Path:    "schemas;Pet;properties;x;y",
			Source:  "api.yaml",
			Details: map[string]string{"key": "x;y", "error": pathSafeNameError},
		}, nil)

		assert.Equal(t, `invalid property name "x;y" at schemas Pet properties`, headline)
		assert.Equal(t, pathSafeNameError, explanation)
	})

	t.Run("property.name_syntax with a control character escapes it", func(t *testing.T) {
		headline, explanation := reports.FormatViolationLine(reports.Violation{
			Code:    "property.name_syntax",
			Path:    "schemas;Pet;properties;a\x00b",
			Source:  "api.yaml",
			Details: map[string]string{"key": "a\x00b", "error": "must not contain control characters"},
		}, nil)

		assert.Equal(t, `invalid property name "a\x00b" at schemas Pet properties`, headline)
		assert.Equal(t, `must not contain control characters`, explanation)
	})

	t.Run("schema.name_syntax", func(t *testing.T) {
		headline, explanation := reports.FormatViolationLine(reports.Violation{
			Code:    "schema.name_syntax",
			Path:    "schemas;Pet.v2",
			Source:  "api.yaml",
			Details: map[string]string{"key": "Pet.v2", "error": pathSafeNameError},
		}, nil)

		assert.Equal(t, `invalid schema name "Pet.v2" at schemas`, headline)
		assert.Equal(t, pathSafeNameError, explanation)
	})

	t.Run("schema.name_syntax with an empty name", func(t *testing.T) {
		headline, explanation := reports.FormatViolationLine(reports.Violation{
			Code:    "schema.name_syntax",
			Path:    "schemas;",
			Source:  "api.yaml",
			Details: map[string]string{"key": "", "error": "must not be empty"},
		}, nil)

		assert.Equal(t, `invalid schema name "" at schemas`, headline)
		assert.Equal(t, `must not be empty`, explanation)
	})

	t.Run("schema.name_syntax with a control character escapes it", func(t *testing.T) {
		headline, explanation := reports.FormatViolationLine(reports.Violation{
			Code:    "schema.name_syntax",
			Path:    "schemas;Pet\x00",
			Source:  "api.yaml",
			Details: map[string]string{"key": "Pet\x00", "error": "must not contain control characters"},
		}, nil)

		assert.Equal(t, `invalid schema name "Pet\x00" at schemas`, headline)
		assert.Equal(t, `must not contain control characters`, explanation)
	})

	t.Run("schema.description_syntax", func(t *testing.T) {
		headline, explanation := reports.FormatViolationLine(reports.Violation{
			Code:    "schema.description_syntax",
			Path:    "schemas;Pet;properties;weight;description",
			Source:  "api.yaml",
			Details: map[string]string{"value": "In kilograms.", "error": "must be ASCII letters and digits, with single spaces between words"},
		}, nil)

		assert.Equal(t, `invalid description "In kilograms." at schemas Pet properties weight`, headline)
		assert.Equal(t, `must be ASCII letters and digits, with single spaces between words`, explanation)
	})

	t.Run("schema.description_syntax with an empty description", func(t *testing.T) {
		headline, explanation := reports.FormatViolationLine(reports.Violation{
			Code:    "schema.description_syntax",
			Path:    "schemas;Pet;description",
			Source:  "api.yaml",
			Details: map[string]string{"value": "", "error": "must not be empty"},
		}, nil)

		assert.Equal(t, `invalid description "" at schemas Pet`, headline)
		assert.Equal(t, `must not be empty`, explanation)
	})

	t.Run("schema.invalid_type", func(t *testing.T) {
		headline, explanation := reports.FormatViolationLine(reports.Violation{
			Code:    "schema.invalid_type",
			Path:    "schemas;Pet;properties;id;type",
			Source:  "api.yaml",
			Details: map[string]string{"value": "strng", "allowed": allowedTypes},
		}, nil)

		assert.Equal(t, `invalid value "strng" for "type" at schemas Pet properties id`, headline)
		assert.Equal(t, `expected one of: object, array, map, string, integer, float, number, boolean`, explanation)
	})

	t.Run("schema.invalid_type without a value reports the type as missing", func(t *testing.T) {
		headline, explanation := reports.FormatViolationLine(reports.Violation{
			Code:    "schema.invalid_type",
			Path:    "schemas;Pet;properties;tags;items",
			Source:  "api.yaml",
			Details: map[string]string{"value": "", "allowed": allowedTypes},
		}, nil)

		assert.Equal(t, `missing "type" at schemas Pet properties tags items`, headline)
		assert.Equal(t, `expected one of: object, array, map, string, integer, float, number, boolean`, explanation)
	})

	t.Run("schema.array_without_items", func(t *testing.T) {
		headline, explanation := reports.FormatViolationLine(reports.Violation{
			Code:   "schema.array_without_items",
			Path:   "schemas;Pets",
			Source: "api.yaml",
		}, nil)

		assert.Equal(t, `array schema without items at schemas Pets`, headline)
		assert.Empty(t, explanation)
	})

	t.Run("schema.map_without_values", func(t *testing.T) {
		headline, explanation := reports.FormatViolationLine(reports.Violation{
			Code:   "schema.map_without_values",
			Path:   "schemas;Inventory",
			Source: "api.yaml",
		}, nil)

		assert.Equal(t, `map schema without values at schemas Inventory`, headline)
		assert.Empty(t, explanation)
	})

	t.Run("schema.object_without_properties suggests a map", func(t *testing.T) {
		headline, explanation := reports.FormatViolationLine(reports.Violation{
			Code:   "schema.object_without_properties",
			Path:   "schemas;Owner;properties;tags",
			Source: "api.yaml",
		}, nil)

		assert.Equal(t, `object schema without properties at schemas Owner properties tags`, headline)
		assert.Equal(t, `declare the keys you read, or use "type: map" with "values" if the keys vary`, explanation)
	})

	t.Run("schema.key_not_allowed values on an object", func(t *testing.T) {
		headline, explanation := reports.FormatViolationLine(reports.Violation{
			Code:    "schema.key_not_allowed",
			Path:    "schemas;Owner;properties;pets",
			Source:  "api.yaml",
			Details: map[string]string{"key": "values", "type": "object"},
		}, nil)

		assert.Equal(t, `"values" is not allowed on an object schema at schemas Owner properties pets`, headline)
		assert.Equal(t, `use it with "type: map"`, explanation)
	})

	t.Run("schema.key_not_allowed properties on a map", func(t *testing.T) {
		headline, explanation := reports.FormatViolationLine(reports.Violation{
			Code:    "schema.key_not_allowed",
			Path:    "schemas;Inventory",
			Source:  "api.yaml",
			Details: map[string]string{"key": "properties", "type": "map"},
		}, nil)

		assert.Equal(t, `"properties" is not allowed on a map schema at schemas Inventory`, headline)
		assert.Equal(t, `use it with "type: object"`, explanation)
	})

	t.Run("schema.key_not_allowed items on a string", func(t *testing.T) {
		headline, explanation := reports.FormatViolationLine(reports.Violation{
			Code:    "schema.key_not_allowed",
			Path:    "schemas;Pet;properties;name",
			Source:  "api.yaml",
			Details: map[string]string{"key": "items", "type": "string"},
		}, nil)

		assert.Equal(t, `"items" is not allowed on a string schema at schemas Pet properties name`, headline)
		assert.Equal(t, `use it with "type: array"`, explanation)
	})

	t.Run("schema.key_not_allowed values on an integer", func(t *testing.T) {
		headline, explanation := reports.FormatViolationLine(reports.Violation{
			Code:    "schema.key_not_allowed",
			Path:    "schemas;Pet;properties;age",
			Source:  "api.yaml",
			Details: map[string]string{"key": "values", "type": "integer"},
		}, nil)

		assert.Equal(t, `"values" is not allowed on an integer schema at schemas Pet properties age`, headline)
		assert.Equal(t, `use it with "type: map"`, explanation)
	})

	t.Run("schema.key_not_allowed members on a string", func(t *testing.T) {
		headline, explanation := reports.FormatViolationLine(reports.Violation{
			Code:    "schema.key_not_allowed",
			Path:    "schemas;Pet;properties;name",
			Source:  "api.yaml",
			Details: map[string]string{"key": "members", "type": "string"},
		}, nil)

		assert.Equal(t, `"members" is not allowed on a string schema at schemas Pet properties name`, headline)
		assert.Equal(t, `use it with "type: enum" or "type: union"`, explanation)
	})

	t.Run("schema.key_not_allowed on a union", func(t *testing.T) {
		headline, explanation := reports.FormatViolationLine(reports.Violation{
			Code:    "schema.key_not_allowed",
			Path:    "schemas;Order;properties;payment",
			Source:  "api.yaml",
			Details: map[string]string{"key": "properties", "type": "union"},
		}, nil)

		assert.Equal(t, `"properties" is not allowed on a union schema at schemas Order properties payment`, headline)
		assert.Equal(t, `use it with "type: object"`, explanation)
	})

	t.Run("schema.key_not_allowed beside ref", func(t *testing.T) {
		headline, explanation := reports.FormatViolationLine(reports.Violation{
			Code:    "schema.key_not_allowed",
			Path:    "schemas;Owner;properties;pet",
			Source:  "api.yaml",
			Details: map[string]string{"key": "type", "type": "ref"},
		}, nil)

		assert.Equal(t, `"type" is not allowed beside "ref" at schemas Owner properties pet`, headline)
		assert.Empty(t, explanation)
	})

	t.Run("schema.key_not_allowed beside ref with a key another type owns", func(t *testing.T) {
		headline, explanation := reports.FormatViolationLine(reports.Violation{
			Code:    "schema.key_not_allowed",
			Path:    "schemas;Owner;properties;pet",
			Source:  "api.yaml",
			Details: map[string]string{"key": "values", "type": "ref"},
		}, nil)

		assert.Equal(t, `"values" is not allowed beside "ref" at schemas Owner properties pet`, headline)
		assert.Empty(t, explanation)
	})

	t.Run("schema.optional_not_allowed on array items", func(t *testing.T) {
		headline, explanation := reports.FormatViolationLine(reports.Violation{
			Code:    "schema.optional_not_allowed",
			Path:    "schemas;Pet;properties;tags;items",
			Source:  "api.yaml",
			Details: map[string]string{"position": "items"},
		}, nil)

		assert.Equal(t, `"optional" is not allowed on array items at schemas Pet properties tags items`, headline)
		assert.Equal(t, `mark the property that holds the array optional, or use "nullable: true" for an item that may be null`, explanation)
	})

	t.Run("schema.optional_not_allowed on map values", func(t *testing.T) {
		headline, explanation := reports.FormatViolationLine(reports.Violation{
			Code:    "schema.optional_not_allowed",
			Path:    "schemas;Pet;properties;scores;values",
			Source:  "api.yaml",
			Details: map[string]string{"position": "values"},
		}, nil)

		assert.Equal(t, `"optional" is not allowed on map values at schemas Pet properties scores values`, headline)
		assert.Equal(t, `mark the property that holds the map optional, or use "nullable: true" for a value that may be null`, explanation)
	})

	t.Run("schema.optional_not_allowed on a named schema", func(t *testing.T) {
		headline, explanation := reports.FormatViolationLine(reports.Violation{
			Code:    "schema.optional_not_allowed",
			Path:    "schemas;Pet",
			Source:  "api.yaml",
			Details: map[string]string{"position": "schema"},
		}, nil)

		assert.Equal(t, `"optional" is not allowed on a schema at schemas Pet`, headline)
		assert.Equal(t, `mark optional the property that uses it, beside its "ref"`, explanation)
	})

	t.Run("schema.optional_not_allowed at an unknown position falls back to the code with its details", func(t *testing.T) {
		headline, explanation := reports.FormatViolationLine(reports.Violation{
			Code:    "schema.optional_not_allowed",
			Path:    "schemas;Pet;properties;owner",
			Source:  "api.yaml",
			Details: map[string]string{"position": "property"},
		}, nil)

		assert.Equal(t, `schema.optional_not_allowed at schemas Pet properties owner (position: property)`, headline)
		assert.Empty(t, explanation)
	})

	t.Run("resource.duplicate", func(t *testing.T) {
		headline, explanation := reports.FormatViolationLine(reports.Violation{
			Code:    "resource.duplicate",
			Path:    "provides;rest;/pets;get;responses;200",
			Source:  "store.yaml",
			Details: map[string]string{"resource": "provides GET /pets 200", "declaredIn": "pets.yaml"},
		}, nil)

		assert.Equal(t, `duplicate resource "provides GET /pets 200", also declared in pets.yaml`, headline)
		assert.Empty(t, explanation)
	})

	t.Run("resource.duplicate in the same file", func(t *testing.T) {
		headline, explanation := reports.FormatViolationLine(reports.Violation{
			Code:    "resource.duplicate",
			Path:    "provides;rest;/pets;get;responses;200",
			Source:  "pets.yaml",
			Details: map[string]string{"resource": "provides GET /pets 200", "declaredIn": "pets.yaml"},
		}, nil)

		assert.Equal(t, `duplicate resource "provides GET /pets 200", declared twice`, headline)
		assert.Empty(t, explanation)
	})

	t.Run("resource.type_conflict", func(t *testing.T) {
		headline, explanation := reports.FormatViolationLine(reports.Violation{
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
		}, nil)

		assert.Equal(t, `conflicting type for property "$.id" of consumes payments GET /invoices 200: integer here, string in a.yaml`, headline)
		assert.Empty(t, explanation)
	})

	t.Run("resource.type_conflict between enum values", func(t *testing.T) {
		headline, explanation := reports.FormatViolationLine(reports.Violation{
			Code:   "resource.type_conflict",
			Path:   "consumes;petstore_api;rest;/pets;get;responses;200",
			Source: "pets.yaml",
			Details: map[string]string{
				"resource":       "consumes petstore_api GET /pets 200",
				"property":       "status",
				"type":           "enum",
				"values":         "a, b",
				"declaredIn":     "other.yaml",
				"declaredType":   "enum",
				"declaredValues": "a, c",
			},
		}, nil)

		assert.Equal(t, `conflicting values for property "status" of consumes petstore_api GET /pets 200: [a, b] here, [a, c] in other.yaml`, headline)
		assert.Empty(t, explanation)
	})

	t.Run("resource.type_conflict between arguments", func(t *testing.T) {
		headline, explanation := reports.FormatViolationLine(reports.Violation{
			Code:   "resource.type_conflict",
			Path:   "consumes;petstore_api;graphql;query;pet;returns",
			Source: "pets.yaml",
			Details: map[string]string{
				"resource":          "consumes petstore_api query pet",
				"property":          "$.reviews",
				"type":              "array",
				"arguments":         "ReviewsArgs",
				"declaredIn":        "other.yaml",
				"declaredType":      "array",
				"declaredArguments": "FirstReviewsArgs",
			},
		}, nil)

		assert.Equal(t, `conflicting arguments for property "$.reviews" of consumes petstore_api query pet: ReviewsArgs here, FirstReviewsArgs in other.yaml`, headline)
		assert.Empty(t, explanation)
	})

	t.Run("resource.type_conflict between arguments and none", func(t *testing.T) {
		headline, explanation := reports.FormatViolationLine(reports.Violation{
			Code:   "resource.type_conflict",
			Path:   "consumes;petstore_api;graphql;query;pet;returns",
			Source: "pets.yaml",
			Details: map[string]string{
				"resource":          "consumes petstore_api query pet",
				"property":          "$.reviews",
				"type":              "array",
				"arguments":         "ReviewsArgs",
				"declaredIn":        "other.yaml",
				"declaredType":      "array",
				"declaredArguments": "",
			},
		}, nil)

		assert.Equal(t, `conflicting arguments for property "$.reviews" of consumes petstore_api query pet: ReviewsArgs here, none in other.yaml`, headline)
		assert.Empty(t, explanation)
	})

	t.Run("members.required on an enum", func(t *testing.T) {
		headline, explanation := reports.FormatViolationLine(reports.Violation{
			Code:    "members.required",
			Path:    "schemas;Pet;properties;status",
			Source:  "api.yaml",
			Details: map[string]string{"type": "enum"},
		}, nil)

		assert.Equal(t, `"type: enum" needs "members" at schemas Pet properties status: list its values`, headline)
		assert.Empty(t, explanation)
	})

	t.Run("members.required on a union", func(t *testing.T) {
		headline, explanation := reports.FormatViolationLine(reports.Violation{
			Code:    "members.required",
			Path:    "schemas;Order;properties;payment",
			Source:  "api.yaml",
			Details: map[string]string{"type": "union"},
		}, nil)

		assert.Equal(t, `"type: union" needs "members" at schemas Order properties payment: map each key to a ref`, headline)
		assert.Empty(t, explanation)
	})

	t.Run("members.empty on an enum", func(t *testing.T) {
		headline, explanation := reports.FormatViolationLine(reports.Violation{
			Code:    "members.empty",
			Path:    "schemas;Pet;properties;status;members",
			Source:  "api.yaml",
			Details: map[string]string{"type": "enum"},
		}, nil)

		assert.Equal(t, `"members" is empty at schemas Pet properties status: list at least one value`, headline)
		assert.Empty(t, explanation)
	})

	t.Run("members.empty on a union", func(t *testing.T) {
		headline, explanation := reports.FormatViolationLine(reports.Violation{
			Code:    "members.empty",
			Path:    "schemas;Order;properties;payment;members",
			Source:  "api.yaml",
			Details: map[string]string{"type": "union"},
		}, nil)

		assert.Equal(t, `"members" is empty at schemas Order properties payment: list at least one member`, headline)
		assert.Empty(t, explanation)
	})

	t.Run("members.duplicate", func(t *testing.T) {
		headline, explanation := reports.FormatViolationLine(reports.Violation{
			Code:    "members.duplicate",
			Path:    "schemas;Pet;properties;status;members;2",
			Source:  "api.yaml",
			Details: map[string]string{"value": "sold"},
		}, nil)

		assert.Equal(t, `value "sold" is repeated in "members" at schemas Pet properties status`, headline)
		assert.Empty(t, explanation)
	})

	t.Run("members.not_text", func(t *testing.T) {
		headline, explanation := reports.FormatViolationLine(reports.Violation{
			Code:   "members.not_text",
			Path:   "schemas;Pet;properties;status;members;1",
			Source: "api.yaml",
		}, nil)

		assert.Equal(t, `each value in "members" must be non-empty text at schemas Pet properties status`, headline)
		assert.Empty(t, explanation)
	})

	t.Run("member.ref_required", func(t *testing.T) {
		headline, explanation := reports.FormatViolationLine(reports.Violation{
			Code:    "member.ref_required",
			Path:    "schemas;Order;properties;payment;members;card",
			Source:  "api.yaml",
			Details: map[string]string{"member": "card"},
		}, nil)

		assert.Equal(t, `member "card" needs a "ref" to a schema at schemas Order properties payment`, headline)
		assert.Empty(t, explanation)
	})

	t.Run("member.name_syntax", func(t *testing.T) {
		headline, explanation := reports.FormatViolationLine(reports.Violation{
			Code:    "member.name_syntax",
			Path:    "schemas;Order;properties;payment;members;card(x)",
			Source:  "api.yaml",
			Details: map[string]string{"key": "card(x)", "error": `must not contain ".", "[", "]", "{", "}", ";", "(" or ")"`},
		}, nil)

		assert.Equal(t, `member key "card(x)" must not contain ".", "[", "]", "{", "}", ";", "(" or ")" at schemas Order properties payment`, headline)
		assert.Empty(t, explanation)
	})

	t.Run("member.name_syntax with a semicolon in the key keeps the location at the schema", func(t *testing.T) {
		headline, explanation := reports.FormatViolationLine(reports.Violation{
			Code:    "member.name_syntax",
			Path:    "schemas;Order;properties;payment;members;a;members;b",
			Source:  "api.yaml",
			Details: map[string]string{"key": "a;members;b", "error": `must not contain ".", "[", "]", "{", "}", ";", "(" or ")"`},
		}, nil)

		assert.Equal(t, `member key "a;members;b" must not contain ".", "[", "]", "{", "}", ";", "(" or ")" at schemas Order properties payment`, headline)
		assert.Empty(t, explanation)
	})

	t.Run("schema.duplicate", func(t *testing.T) {
		headline, explanation := reports.FormatViolationLine(reports.Violation{
			Code:    "schema.duplicate",
			Path:    "schemas;Pet",
			Source:  "schemas.yaml",
			Details: map[string]string{"schema": "Pet", "declaredIn": "billing.yaml"},
		}, nil)

		assert.Equal(t, `duplicate schema "Pet", also declared in billing.yaml`, headline)
		assert.Empty(t, explanation)
	})

	t.Run("schema.duplicate in the same file", func(t *testing.T) {
		headline, explanation := reports.FormatViolationLine(reports.Violation{
			Code:    "schema.duplicate",
			Path:    "schemas;Pet",
			Source:  "schemas.yaml",
			Details: map[string]string{"schema": "Pet", "declaredIn": "schemas.yaml"},
		}, nil)

		assert.Equal(t, `duplicate schema "Pet", declared twice`, headline)
		assert.Empty(t, explanation)
	})

	t.Run("schema.unresolved_name", func(t *testing.T) {
		headline, explanation := reports.FormatViolationLine(reports.Violation{
			Code:    "schema.unresolved_name",
			Path:    "provides;rest;/pets;get;responses;200",
			Source:  "pets.yaml",
			Details: map[string]string{"schema": "Pets", "resource": "provides GET /pets 200"},
		}, nil)

		assert.Equal(t, `unresolved schema "Pets" referenced by provides GET /pets 200`, headline)
		assert.Empty(t, explanation)
	})

	t.Run("schema.unresolved_ref", func(t *testing.T) {
		headline, explanation := reports.FormatViolationLine(reports.Violation{
			Code:    "schema.unresolved_ref",
			Path:    "schemas;Invoice;properties;payment",
			Source:  "billing.yaml",
			Details: map[string]string{"schema": "Payment", "property": "Invoice.payment"},
		}, nil)

		assert.Equal(t, `unresolved ref "Payment" in Invoice.payment`, headline)
		assert.Empty(t, explanation)
	})

	t.Run("unknown code falls back to the code with its sorted details", func(t *testing.T) {
		headline, explanation := reports.FormatViolationLine(reports.Violation{
			Code:    "something.new",
			Path:    "provides;rest;/pets",
			Source:  "api.yaml",
			Details: map[string]string{"hint": "x", "area": "y"},
		}, nil)

		assert.Equal(t, `something.new at provides rest /pets (area: y, hint: x)`, headline)
		assert.Empty(t, explanation)
	})

	t.Run("unknown code without details", func(t *testing.T) {
		headline, explanation := reports.FormatViolationLine(reports.Violation{
			Code:   "something.new",
			Path:   "provides;rest;/pets",
			Source: "api.yaml",
		}, nil)

		assert.Equal(t, `something.new at provides rest /pets`, headline)
		assert.Empty(t, explanation)
	})

	t.Run("unknown code without path", func(t *testing.T) {
		headline, explanation := reports.FormatViolationLine(reports.Violation{
			Code:    "something.new",
			Source:  "api.yaml",
			Details: map[string]string{"hint": "x"},
		}, nil)

		assert.Equal(t, `something.new (hint: x)`, headline)
		assert.Empty(t, explanation)
	})

	t.Run("unknown code without path or details", func(t *testing.T) {
		headline, explanation := reports.FormatViolationLine(reports.Violation{
			Code:   "something.new",
			Source: "api.yaml",
		}, nil)

		assert.Equal(t, `something.new`, headline)
		assert.Empty(t, explanation)
	})
}

func TestFormatValidationFailedReportWritesTheLineAModuleGivesItsCode(t *testing.T) {
	report := reports.FormatValidationFailedReport(paint.For(&bytes.Buffer{}), "contract validation failed", []reports.Violation{
		{Code: "topic.name_syntax", Path: "provides;topics;Pet Events", Source: "api.yaml", Details: map[string]string{"key": "Pet Events", "error": "must be snake_case"}},
		{Code: "schema.array_without_items", Path: "schemas;Pets", Source: "api.yaml"},
	}, []reports.Module{{Report: topicsProtocol{}}})

	assert.Equal(t, `contract validation failed
  - api.yaml: invalid topic name "Pet Events" at provides topics, must be snake_case
  - api.yaml: array schema without items at schemas Pets
`, report)
}

func TestFormatViolationLineAsksTheModules(t *testing.T) {
	modules := []reports.Module{{Report: topicsProtocol{}}}

	t.Run("a code a module knows gets the module's line", func(t *testing.T) {
		headline, explanation := reports.FormatViolationLine(reports.Violation{
			Code:    "topic.name_syntax",
			Path:    "provides;topics;Pet Events",
			Source:  "api.yaml",
			Details: map[string]string{"key": "Pet Events", "error": "must be snake_case"},
		}, modules)

		assert.Equal(t, `invalid topic name "Pet Events" at provides topics`, headline)
		assert.Equal(t, `must be snake_case`, explanation)
	})

	t.Run("a code no module knows falls back to the code with its sorted details", func(t *testing.T) {
		headline, explanation := reports.FormatViolationLine(reports.Violation{
			Code:    "something.new",
			Path:    "provides;topics;pet_events",
			Source:  "api.yaml",
			Details: map[string]string{"hint": "x"},
		}, modules)

		assert.Equal(t, `something.new at provides topics pet_events (hint: x)`, headline)
		assert.Empty(t, explanation)
	})

	t.Run("schema.key_not_allowed with a key a module declares names the module's type", func(t *testing.T) {
		headline, explanation := reports.FormatViolationLine(reports.Violation{
			Code:    "schema.key_not_allowed",
			Path:    "schemas;PetEvent",
			Source:  "api.yaml",
			Details: map[string]string{"key": "partitions", "type": "object"},
		}, modules)

		assert.Equal(t, `"partitions" is not allowed on an object schema at schemas PetEvent`, headline)
		assert.Equal(t, `use it with "type: topic"`, explanation)
	})

	t.Run("schema.key_not_allowed with a core key a module also declares names the core types first", func(t *testing.T) {
		headline, explanation := reports.FormatViolationLine(reports.Violation{
			Code:    "schema.key_not_allowed",
			Path:    "schemas;Pet;properties;name",
			Source:  "api.yaml",
			Details: map[string]string{"key": "members", "type": "string"},
		}, modules)

		assert.Equal(t, `"members" is not allowed on a string schema at schemas Pet properties name`, headline)
		assert.Equal(t, `use it with "type: enum" or "type: union" or "type: topic"`, explanation)
	})

	t.Run("schema.key_not_allowed with a key the module declares is unknown without the module", func(t *testing.T) {
		headline, explanation := reports.FormatViolationLine(reports.Violation{
			Code:    "schema.key_not_allowed",
			Path:    "schemas;PetEvent",
			Source:  "api.yaml",
			Details: map[string]string{"key": "partitions", "type": "object"},
		}, nil)

		assert.Equal(t, `schema.key_not_allowed at schemas PetEvent (key: partitions, type: object)`, headline)
		assert.Empty(t, explanation)
	})
}

type topicsProtocol struct{}

func (this topicsProtocol) ReportKey() string {
	return "topics"
}

func (this topicsProtocol) WriteTree(*strings.Builder, reports.Counterpart, json.RawMessage) error {
	return nil
}

func (this topicsProtocol) ViolationLine(violation reports.Violation) (headline, explanation string, known bool) {
	if violation.Code != "topic.name_syntax" {
		return "", "", false
	}

	return fmt.Sprintf("invalid topic name %q%s", violation.Details["key"], reports.At(reports.Location(violation.Path, violation.Details["key"]))), violation.Details["error"], true
}

func (this topicsProtocol) SchemaKeys() map[string][]string {
	return map[string][]string{
		"partitions": {`"type: topic"`},
		"members":    {`"type: topic"`},
	}
}
