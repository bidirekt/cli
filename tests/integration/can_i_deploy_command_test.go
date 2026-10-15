package integration_test

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"testing"

	"github.com/bidirekt/cli/internal/components"
	"github.com/bidirekt/cli/internal/features/can_i_deploy"
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
		assert.Equal(t, "front v1 can be deployed to production\nnothing deployed in production to compare with\n", out.String())
		assert.Empty(t, errOut.String())
	})

	t.Run("deployable verdict names the counterparts it was compared with, in alphabetical order", func(t *testing.T) {
		httpClient := components.NewHTTPClient(&components.Config{BrokerURL: brokerURL})
		httpmock.ActivateNonDefault(httpClient.StdClient())
		defer httpmock.DeactivateAndReset()

		responseBody := `{
		  "message": "contract checked successfully",
		  "participant": "front",
		  "version": "v1",
		  "environment": "production",
		  "deployable": true,
		  "results": {
		    "site": { "deployable": true, "participantVersion": "v4", "endpoints": {} },
		    "admin": { "deployable": true, "participantVersion": "v2", "endpoints": {} }
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

		require.NoError(t, err)
		assert.Equal(t, "front v1 can be deployed to production\ncompared with: admin, site\n", out.String())
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
		          "put": {
		            "request": [
		              {
		                "reason": "property_missing_in_consumer",
		                "role": "consumer",
		                "details": { "property": "$.currency", "propertyType": "string", "consumerName": "front", "providerName": "payments" }
		              }
		            ],
		            "200": [
		              { "reason": "some_future_reason", "role": "consumer", "details": { "property": "$.amount" } }
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
		assert.Equal(t, `front v1 cannot be deployed to production

payments (v2, deployed):
  PUT /payments/*
    request:
      - front doesn't send "$.currency", but payments requires it
        → front: send it
        → payments: stop requiring it, then mark it optional
    response 200:
      - some_future_reason (property: $.amount)
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
		              { "reason": "provider_resource_not_found", "role": "consumer" }
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
		assert.Contains(t, out.String(), "  GET /payments/*\n    response 200:\n      - front calls GET /payments/*, but payments doesn't provide it\n        → front: stop calling it\n        → payments: publish it\n")
	})

	t.Run("a status or request body the provider lacks on a method it has is told apart from a missing method", func(t *testing.T) {
		httpClient := components.NewHTTPClient(&components.Config{BrokerURL: brokerURL})
		httpmock.ActivateNonDefault(httpClient.StdClient())
		defer httpmock.DeactivateAndReset()

		responseBody := `{
		  "message": "contract checked successfully",
		  "participant": "site",
		  "version": "v1",
		  "environment": "production",
		  "deployable": false,
		  "results": {
		    "api": {
		      "deployable": false,
		      "participantVersion": null,
		      "endpoints": {
		        "/api/newsletter/unsubscribe": {
		          "post": { "200": [{ "reason": "provider_resource_interaction_not_found", "role": "consumer", "details": { "providerStatuses": "204" } }] },
		          "delete": { "204": [{ "reason": "provider_resource_not_found", "role": "consumer" }] }
		        },
		        "/api/users/*/restore": {
		          "post": { "request": [{ "reason": "provider_resource_interaction_not_found", "role": "consumer", "details": { "providerStatuses": "204, 404" } }] }
		        },
		        "/api/files": {
		          "post": { "201": [{ "reason": "provider_resource_interaction_not_found", "role": "consumer" }] }
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
		command.SetArgs([]string{"site", "--version", version, "--environment", environment})

		err := command.Execute()

		require.Error(t, err)
		assert.Equal(t, `site v1 cannot be deployed to production

api:
  POST /api/files
    response 201:
      - site reads response 201, but api declares no response
        → site: stop reading it
        → api: answer 201
  DELETE /api/newsletter/unsubscribe
    response 204:
      - site calls DELETE /api/newsletter/unsubscribe, but api doesn't provide it
        → site: stop calling it
        → api: publish it
  POST /api/newsletter/unsubscribe
    response 200:
      - site reads response 200, but api answers only 204
        → site: read response 204
        → api: answer 200
  POST /api/users/*/restore
    request:
      - site sends a request body, but api declares none
        → site: stop sending it
        → api: accept a request body
`, out.String())
		assert.Empty(t, errOut.String())
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
		httpmock.RegisterNoResponder(httpmock.NewErrorResponder(errors.New("unexpected request to the broker")))

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
