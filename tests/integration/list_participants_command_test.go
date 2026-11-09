package integration_test

import (
	"bytes"
	"errors"
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
		httpClient := components.NewHTTPClient()
		httpClient.UseBroker(brokerURL, "")
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
		httpClient := components.NewHTTPClient()
		httpClient.UseBroker(brokerURL, "")
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
		httpClient := components.NewHTTPClient()
		httpClient.UseBroker(brokerURL, "")
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
		httpClient := components.NewHTTPClient()
		httpClient.UseBroker(brokerURL, "")
		httpmock.ActivateNonDefault(httpClient.StdClient())
		defer httpmock.DeactivateAndReset()
		httpmock.RegisterNoResponder(httpmock.NewErrorResponder(errors.New("unexpected request to the broker")))

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

	for _, test := range []struct {
		name   string
		status int
		err    error
	}{
		{name: "a 401 returns NotSignedInError", status: http.StatusUnauthorized, err: components.NotSignedInError{BrokerURL: brokerURL}},
		{name: "a 403 returns ForbiddenError", status: http.StatusForbidden, err: components.ForbiddenError{BrokerURL: brokerURL}},
	} {
		t.Run(test.name, func(t *testing.T) {
			httpClient := components.NewHTTPClient()
			httpClient.UseBroker(brokerURL, "bdk_acme")
			httpClient.ExplainAuthFailures(brokerURL)
			httpmock.ActivateNonDefault(httpClient.StdClient())
			defer httpmock.DeactivateAndReset()

			httpmock.RegisterMatcherResponder(http.MethodGet, endpoint, httpmock.HeaderIs("Authorization", "Bearer bdk_acme"),
				httpmock.NewStringResponder(test.status, `{"message":"refused"}`))

			command := list_participants.NewListParticipantsCommand(list_participants.NewListParticipantsClient(httpClient))
			var out, errOut bytes.Buffer
			command.SetOut(&out)
			command.SetErr(&errOut)
			command.SetArgs([]string{})

			err := command.Execute()

			require.ErrorIs(t, err, test.err)
			assert.Equal(t, 1, httpmock.GetTotalCallCount())
			assert.Empty(t, out.String())
		})
	}

	t.Run("a 426 fails with the broker's message as the text to print", func(t *testing.T) {
		const upgradeMessage = "bidirekt 0.0.9 doesn't work with this broker (0.3.0); upgrade bidirekt to 0.1.0 or later"
		httpClient := components.NewHTTPClient()
		httpClient.UseBroker(brokerURL, "bdk_acme")
		httpmock.ActivateNonDefault(httpClient.StdClient())
		defer httpmock.DeactivateAndReset()

		httpmock.RegisterResponder(http.MethodGet, endpoint,
			httpmock.NewStringResponder(http.StatusUpgradeRequired, `{"message":"`+upgradeMessage+`"}`))

		command := list_participants.NewListParticipantsCommand(list_participants.NewListParticipantsClient(httpClient))
		var out, errOut bytes.Buffer
		command.SetOut(&out)
		command.SetErr(&errOut)
		command.SetArgs([]string{})

		err := command.Execute()

		var upgradeRequired components.UpgradeRequiredError
		require.ErrorAs(t, err, &upgradeRequired)
		assert.Equal(t, upgradeMessage, upgradeRequired.UserText())
		assert.Equal(t, 1, httpmock.GetTotalCallCount())
		assert.Empty(t, out.String())
	})
}
