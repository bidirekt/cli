package integration_test

import (
	"bytes"
	"errors"
	"net/http"
	"testing"

	"github.com/bidirekt/cli/internal/components"
	"github.com/bidirekt/cli/internal/features/list_environments"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListEnvironmentsCommand(t *testing.T) {
	const (
		brokerURL = "http://localhost:8080"
		endpoint  = brokerURL + "/api/environments"
	)

	t.Run("prints one name per line in the broker's order, exits 0", func(t *testing.T) {
		httpClient := components.NewHTTPClient(&components.Config{BrokerURL: brokerURL})
		httpmock.ActivateNonDefault(httpClient.StdClient())
		defer httpmock.DeactivateAndReset()

		httpmock.RegisterResponder(http.MethodGet, endpoint,
			httpmock.NewStringResponder(http.StatusOK, `{"message":"environments listed successfully","environments":["staging","production"]}`))

		command := list_environments.NewListEnvironmentsCommand(list_environments.NewListEnvironmentsClient(httpClient))
		var out, errOut bytes.Buffer
		command.SetOut(&out)
		command.SetErr(&errOut)
		command.SetArgs([]string{})

		err := command.Execute()

		require.NoError(t, err)
		assert.Equal(t, 1, httpmock.GetCallCountInfo()["GET "+endpoint])
		assert.Equal(t, "staging\nproduction\n", out.String())
		assert.Empty(t, errOut.String())
	})

	t.Run("empty list prints nothing, exits 0", func(t *testing.T) {
		httpClient := components.NewHTTPClient(&components.Config{BrokerURL: brokerURL})
		httpmock.ActivateNonDefault(httpClient.StdClient())
		defer httpmock.DeactivateAndReset()

		httpmock.RegisterResponder(http.MethodGet, endpoint,
			httpmock.NewStringResponder(http.StatusOK, `{"message":"environments listed successfully","environments":[]}`))

		command := list_environments.NewListEnvironmentsCommand(list_environments.NewListEnvironmentsClient(httpClient))
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
			httpmock.NewStringResponder(http.StatusInternalServerError, `{"message":"environments listing failed"}`))

		command := list_environments.NewListEnvironmentsCommand(list_environments.NewListEnvironmentsClient(httpClient))
		var out, errOut bytes.Buffer
		command.SetOut(&out)
		command.SetErr(&errOut)
		command.SetArgs([]string{})

		err := command.Execute()

		require.EqualError(t, err, "cannot list environments from broker: environments listing failed")
		assert.Empty(t, out.String())
	})

	t.Run("rejects an argument without calling the broker", func(t *testing.T) {
		httpClient := components.NewHTTPClient(&components.Config{BrokerURL: brokerURL})
		httpmock.ActivateNonDefault(httpClient.StdClient())
		defer httpmock.DeactivateAndReset()
		httpmock.RegisterNoResponder(httpmock.NewErrorResponder(errors.New("unexpected request to the broker")))

		command := list_environments.NewListEnvironmentsCommand(list_environments.NewListEnvironmentsClient(httpClient))
		var out, errOut bytes.Buffer
		command.SetOut(&out)
		command.SetErr(&errOut)
		command.SetArgs([]string{"production"})

		err := command.Execute()

		require.Error(t, err)
		assert.Contains(t, err.Error(), `unknown command "production"`)
		assert.Zero(t, httpmock.GetTotalCallCount())
	})
}
