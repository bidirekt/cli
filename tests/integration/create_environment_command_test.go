package integration_test

import (
	"bytes"
	"io"
	"net/http"
	"testing"

	"github.com/bidirekt/cli/internal/components"
	"github.com/bidirekt/cli/internal/features/create_environment"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateEnvironmentCommand(t *testing.T) {
	const (
		brokerURL = "http://localhost:8080"
		name      = "production"
		endpoint  = brokerURL + "/api/environments"
	)

	t.Run("creates an environment, posts name, prints the message, exits 0", func(t *testing.T) {
		httpClient := components.NewHTTPClient(&components.Config{BrokerURL: brokerURL})
		httpmock.ActivateNonDefault(httpClient.StdClient())
		defer httpmock.DeactivateAndReset()

		var capturedBody []byte
		httpmock.RegisterResponder(http.MethodPost, endpoint,
			func(req *http.Request) (*http.Response, error) {
				body, err := io.ReadAll(req.Body)
				if err != nil {
					return nil, err
				}
				capturedBody = body
				return httpmock.NewStringResponse(http.StatusOK, `{"message":"environment created"}`), nil
			})

		command := create_environment.NewCreateEnvironmentCommand(
			create_environment.NewCreateEnvironmentClient(httpClient),
		)
		var out, errOut bytes.Buffer
		command.SetOut(&out)
		command.SetErr(&errOut)
		command.SetArgs([]string{name})

		err := command.Execute()

		require.NoError(t, err)
		assert.Equal(t, 1, httpmock.GetCallCountInfo()["POST "+endpoint])
		assert.JSONEq(t, `{"environment":"production"}`, string(capturedBody))
		assert.Equal(t, name+" environment created\n", out.String())
		assert.Empty(t, errOut.String())
	})

	t.Run("existing environment prints already exists, exits 0", func(t *testing.T) {
		httpClient := components.NewHTTPClient(&components.Config{BrokerURL: brokerURL})
		httpmock.ActivateNonDefault(httpClient.StdClient())
		defer httpmock.DeactivateAndReset()

		httpmock.RegisterResponder(http.MethodPost, endpoint,
			httpmock.NewStringResponder(http.StatusOK, `{"message":"environment already exists"}`))

		command := create_environment.NewCreateEnvironmentCommand(
			create_environment.NewCreateEnvironmentClient(httpClient),
		)
		var out, errOut bytes.Buffer
		command.SetOut(&out)
		command.SetErr(&errOut)
		command.SetArgs([]string{name})

		err := command.Execute()

		require.NoError(t, err)
		assert.Equal(t, name+" environment already exists\n", out.String())
		assert.Empty(t, errOut.String())
	})

	t.Run("non-200 response surfaces the broker body and exits non-zero", func(t *testing.T) {
		httpClient := components.NewHTTPClient(&components.Config{BrokerURL: brokerURL})
		httpmock.ActivateNonDefault(httpClient.StdClient())
		defer httpmock.DeactivateAndReset()

		httpmock.RegisterResponder(http.MethodPost, endpoint,
			httpmock.NewStringResponder(http.StatusBadRequest, `{"message":"environment invalid input"}`))

		command := create_environment.NewCreateEnvironmentCommand(
			create_environment.NewCreateEnvironmentClient(httpClient),
		)
		var out, errOut bytes.Buffer
		command.SetOut(&out)
		command.SetErr(&errOut)
		command.SetArgs([]string{name})

		err := command.Execute()

		require.Error(t, err)
		assert.Equal(t, "cannot post environment to broker: environment invalid input", err.Error())
	})
}
