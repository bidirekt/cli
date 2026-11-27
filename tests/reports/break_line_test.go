package reports_test

import (
	"strings"
	"testing"

	"github.com/bidirekt/cli/internal/reports"
	"github.com/stretchr/testify/assert"
)

func TestWriteBreakLineWritesTheTextAfterADashAndEachFixAfterAnArrowBelowIt(t *testing.T) {
	var report strings.Builder

	reports.WriteBreakLine(&report, "    ", reports.BreakLine{
		Text:  "petstore_web reads pet_events, but petstore_api doesn't publish it",
		Fixes: []string{"petstore_web: stop reading it", "petstore_api: publish it"},
	})

	assert.Equal(t, "    - petstore_web reads pet_events, but petstore_api doesn't publish it\n"+
		"      → petstore_web: stop reading it\n"+
		"      → petstore_api: publish it\n",
		report.String())
}

func TestWriteBreakLineWithoutFixesWritesOnlyTheText(t *testing.T) {
	var report strings.Builder

	reports.WriteBreakLine(&report, "  ", reports.BreakLine{Text: "some_future_reason"})

	assert.Equal(t, "  - some_future_reason\n", report.String())
}

func TestFallbackBreakLine(t *testing.T) {
	t.Run("without details is the bare reason", func(t *testing.T) {
		assert.Equal(t, "some_future_reason", reports.FallbackBreakLine("some_future_reason", nil))
	})

	t.Run("with details lists them sorted by key", func(t *testing.T) {
		line := reports.FallbackBreakLine("some_future_reason", map[string]string{
			"property": "$.photoUrl", "consumerName": "petstore_web", "providerName": "petstore_api",
		})

		assert.Equal(t, "some_future_reason (consumerName: petstore_web, property: $.photoUrl, providerName: petstore_api)", line)
	})
}
