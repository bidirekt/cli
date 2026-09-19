package integration_test

import (
	"bytes"
	"io"
	"net/http"
	"testing"

	"github.com/contracttesting/cli/internal/components"
	"github.com/contracttesting/cli/internal/features/can_i_deploy"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCanIDeployCommand(t *testing.T) {
	const (
		brokerURL   = "http://localhost:8080"
		participant = "front"
		version     = "v1"
		environment = "production"
		endpoint    = brokerURL + "/api/can-i-deploy"
	)

	t.Run("deployable verdict posts participant+version+environment, prints deployable, exits 0", func(t *testing.T) {
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
				return httpmock.NewStringResponse(http.StatusOK, `{"message":"contract checked successfully","deployable":true,"environment":"production","results":{}}`), nil
			})

		command := can_i_deploy.NewCanIDeployCommand(
			can_i_deploy.NewCanIDeployClient(httpClient),
		)
		var out, errOut bytes.Buffer
		command.SetOut(&out)
		command.SetErr(&errOut)
		command.SetArgs([]string{participant, "--version", version, "--environment", environment})

		err := command.Execute()

		require.NoError(t, err)
		assert.Equal(t, 1, httpmock.GetCallCountInfo()["POST "+endpoint])
		assert.JSONEq(t, `{"participant":"front","version":"v1","environment":"production"}`, string(capturedBody))
		assert.Contains(t, out.String(), "front can be deployed to production")
		assert.Empty(t, errOut.String())
	})

	t.Run("not-deployable verdict prints the human report, exits non-zero, no Error: line", func(t *testing.T) {
		httpClient := components.NewHTTPClient(&components.Config{BrokerURL: brokerURL})
		httpmock.ActivateNonDefault(httpClient.StdClient())
		defer httpmock.DeactivateAndReset()

		responseBody := `{
		  "message": "contract checked successfully",
		  "participant": "front",
		  "version": "v1",
		  "environment": "production",
		  "deployable": false,
		  "results": {
		    "payments": {
		      "deployable": false,
		      "participantVersion": "v2",
		      "endpoints": {
		        "/payments/*": {
		          "get": {
		            "request": [
		              { "reason": "property_missing_in_provider", "details": { "property": "currency" } }
		            ],
		            "200": [
		              { "reason": "some_future_reason", "details": { "property": "amount" } }
		            ]
		          }
		        }
		      }
		    },
		    "users": {
		      "deployable": true,
		      "participantVersion": "v3",
		      "endpoints": {}
		    }
		  }
		}`
		httpmock.RegisterResponder(http.MethodPost, endpoint,
			httpmock.NewStringResponder(http.StatusOK, responseBody))

		command := can_i_deploy.NewCanIDeployCommand(
			can_i_deploy.NewCanIDeployClient(httpClient),
		)
		var out, errOut bytes.Buffer
		command.SetOut(&out)
		command.SetErr(&errOut)
		command.SetArgs([]string{participant, "--version", version, "--environment", environment})

		err := command.Execute()

		require.Error(t, err)
		assert.Equal(t, `❌ front cannot be deployed to production

payments (v2):
  GET /payments/*
    request:
      - property "currency" is missing in provider
    response 200:
      - some_future_reason (property: amount)
`, out.String())
		assert.NotContains(t, errOut.String(), "Error:")
	})

	t.Run("counterpart version omitted from the block header when null", func(t *testing.T) {
		httpClient := components.NewHTTPClient(&components.Config{BrokerURL: brokerURL})
		httpmock.ActivateNonDefault(httpClient.StdClient())
		defer httpmock.DeactivateAndReset()

		responseBody := `{
		  "message": "contract checked successfully",
		  "participant": "front",
		  "version": "v1",
		  "environment": "production",
		  "deployable": false,
		  "results": {
		    "payments": {
		      "deployable": false,
		      "participantVersion": null,
		      "endpoints": {
		        "/payments/*": {
		          "get": {
		            "200": [
		              { "reason": "provider_resource_not_found" }
		            ]
		          }
		        }
		      }
		    }
		  }
		}`
		httpmock.RegisterResponder(http.MethodPost, endpoint,
			httpmock.NewStringResponder(http.StatusOK, responseBody))

		command := can_i_deploy.NewCanIDeployCommand(
			can_i_deploy.NewCanIDeployClient(httpClient),
		)
		var out, errOut bytes.Buffer
		command.SetOut(&out)
		command.SetErr(&errOut)
		command.SetArgs([]string{participant, "--version", version, "--environment", environment})

		err := command.Execute()

		require.Error(t, err)
		assert.Contains(t, out.String(), "\npayments:\n")
		assert.NotContains(t, out.String(), "payments (")
		assert.Contains(t, out.String(), "  GET /payments/*\n    response 200:\n      - no matching resource in provider")
	})

	t.Run("non-2xx response renders the broker message to stderr and exits non-zero", func(t *testing.T) {
		httpClient := components.NewHTTPClient(&components.Config{BrokerURL: brokerURL})
		httpmock.ActivateNonDefault(httpClient.StdClient())
		defer httpmock.DeactivateAndReset()

		httpmock.RegisterResponder(http.MethodPost, endpoint,
			httpmock.NewStringResponder(http.StatusNotFound, `{"message":"participant not found"}`))

		command := can_i_deploy.NewCanIDeployCommand(
			can_i_deploy.NewCanIDeployClient(httpClient),
		)
		var out, errOut bytes.Buffer
		command.SetOut(&out)
		command.SetErr(&errOut)
		command.SetArgs([]string{"ghost", "--version", version, "--environment", environment})

		err := command.Execute()

		require.Error(t, err)
		assert.Contains(t, errOut.String(), "participant not found")
		assert.NotContains(t, errOut.String(), "{")
		assert.NotContains(t, errOut.String(), "\"success\"")
	})

	t.Run("missing --version fails before any request", func(t *testing.T) {
		httpClient := components.NewHTTPClient(&components.Config{BrokerURL: brokerURL})
		httpmock.ActivateNonDefault(httpClient.StdClient())
		defer httpmock.DeactivateAndReset()

		command := can_i_deploy.NewCanIDeployCommand(
			can_i_deploy.NewCanIDeployClient(httpClient),
		)
		var out, errOut bytes.Buffer
		command.SetOut(&out)
		command.SetErr(&errOut)
		command.SetArgs([]string{participant, "--environment", environment})

		err := command.Execute()

		require.Error(t, err)
		assert.Zero(t, httpmock.GetTotalCallCount())
	})
}
