package integration_test

import (
	"bytes"
	"net/http"
	"testing"

	"github.com/bidirekt/cli/internal/components"
	"github.com/bidirekt/cli/internal/features/list_participants"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListParticipantsCommand(t *testing.T) {
	const (
		brokerURL = "http://localhost:8080"
		endpoint  = brokerURL + "/api/participants"
	)

	t.Run("prints one name per line in the broker's order, exits 0", func(t *testing.T) {
		httpClient := components.NewHTTPClient(&components.Config{BrokerURL: brokerURL})
		httpmock.ActivateNonDefault(httpClient.StdClient())
		defer httpmock.DeactivateAndReset()

		httpmock.RegisterResponder(http.MethodGet, endpoint,
			httpmock.NewStringResponder(http.StatusOK, `{"message":"participants listed successfully","participants":["orders","billing"]}`))

		command := list_participants.NewListParticipantsCommand(list_participants.NewListParticipantsClient(httpClient))
		var out, errOut bytes.Buffer
		command.SetOut(&out)
		command.SetErr(&errOut)
		command.SetArgs([]string{})

		err := command.Execute()

		require.NoError(t, err)
		assert.Equal(t, 1, httpmock.GetCallCountInfo()["GET "+endpoint])
		assert.Equal(t, "orders\nbilling\n", out.String())
		assert.Empty(t, errOut.String())
	})

	t.Run("empty list prints nothing, exits 0", func(t *testing.T) {
		httpClient := components.NewHTTPClient(&components.Config{BrokerURL: brokerURL})
		httpmock.ActivateNonDefault(httpClient.StdClient())
		defer httpmock.DeactivateAndReset()

		httpmock.RegisterResponder(http.MethodGet, endpoint,
			httpmock.NewStringResponder(http.StatusOK, `{"message":"participants listed successfully","participants":[]}`))

		command := list_participants.NewListParticipantsCommand(list_participants.NewListParticipantsClient(httpClient))
		var out, errOut bytes.Buffer
		command.SetOut(&out)
		command.SetErr(&errOut)
		command.SetArgs([]string{})

		err := command.Execute()

		require.NoError(t, err)
		assert.Empty(t, out.String())
		assert.Empty(t, errOut.String())
	})

	t.Run("non-200 response surfaces the broker message and exits non-zero", func(t *testing.T) {
		httpClient := components.NewHTTPClient(&components.Config{BrokerURL: brokerURL})
		httpmock.ActivateNonDefault(httpClient.StdClient())
		defer httpmock.DeactivateAndReset()

		httpmock.RegisterResponder(http.MethodGet, endpoint,
			httpmock.NewStringResponder(http.StatusInternalServerError, `{"message":"participants listing failed"}`))

		command := list_participants.NewListParticipantsCommand(list_participants.NewListParticipantsClient(httpClient))
		var out, errOut bytes.Buffer
		command.SetOut(&out)
		command.SetErr(&errOut)
		command.SetArgs([]string{})

		err := command.Execute()

		require.EqualError(t, err, "cannot list participants from broker: participants listing failed")
		assert.Empty(t, out.String())
	})

	t.Run("rejects an argument without calling the broker", func(t *testing.T) {
		httpClient := components.NewHTTPClient(&components.Config{BrokerURL: brokerURL})
		httpmock.ActivateNonDefault(httpClient.StdClient())
		defer httpmock.DeactivateAndReset()

		command := list_participants.NewListParticipantsCommand(list_participants.NewListParticipantsClient(httpClient))
		var out, errOut bytes.Buffer
		command.SetOut(&out)
		command.SetErr(&errOut)
		command.SetArgs([]string{"billing"})

		err := command.Execute()

		require.Error(t, err)
		assert.Contains(t, err.Error(), `unknown command "billing"`)
		assert.Zero(t, httpmock.GetTotalCallCount())
	})
}
