package integration_test

import (
	"bytes"
	"net/http"
	"testing"

	"github.com/bidirekt/cli/internal/components"
	"github.com/bidirekt/cli/internal/features/can_i_deploy"
	"github.com/bidirekt/cli/internal/reports"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCanIDeployCommandRemovedInteractionStillConsumed(t *testing.T) {
	const (
		brokerURL   = "http://localhost:8080"
		participant = "petstore_api"
		version     = "2.0.0"
		environment = "production"
		endpoint    = brokerURL + "/api/can-i-deploy"
	)

	runCanIDeploy := func(t *testing.T, endpoints string) (string, string, error) {
		t.Helper()

		httpClient := components.NewHTTPClient(&components.Config{BrokerURL: brokerURL})
		httpmock.ActivateNonDefault(httpClient.StdClient())
		t.Cleanup(httpmock.DeactivateAndReset)

		responseBody := `{
		  "message": "contract checked successfully",
		  "participant": "petstore_api",
		  "version": "2.0.0",
		  "environment": "production",
		  "deployable": false,
		  "results": {
		    "petstore_web": {
		      "deployable": false,
		      "participantVersion": "1.0.0",
		      "endpoints": ` + endpoints + `
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

		return out.String(), errOut.String(), err
	}

	t.Run("a response status the provider no longer answers names the statuses it answers now", func(t *testing.T) {
		out, errOut, err := runCanIDeploy(t, `{
		  "/pets/*": {
		    "delete": {
		      "200": [
		        { "reason": "provider_resource_interaction_removed_but_still_consumed", "role": "provider", "details": { "providerStatuses": "204, 404" } }
		      ]
		    }
		  }
		}`)

		require.ErrorIs(t, err, reports.ErrSilent)
		assert.Equal(t, `petstore_api 2.0.0 cannot be deployed to production

petstore_web (1.0.0, deployed):
  DELETE /pets/*
    response 200:
      - petstore_api now answers DELETE /pets/* only with 204 or 404, but petstore_web still reads response 200
        → petstore_api: keep answering 200 until petstore_web stops reading it
        → petstore_web: read response 204 or 404
`, out)
		assert.NotContains(t, errOut, "Error:")
	})

	t.Run("a response status on a method left without responses says the provider no longer answers it", func(t *testing.T) {
		out, errOut, err := runCanIDeploy(t, `{
		  "/pets/*": {
		    "delete": {
		      "200": [
		        { "reason": "provider_resource_interaction_removed_but_still_consumed", "role": "provider" }
		      ]
		    }
		  }
		}`)

		require.ErrorIs(t, err, reports.ErrSilent)
		assert.Equal(t, `petstore_api 2.0.0 cannot be deployed to production

petstore_web (1.0.0, deployed):
  DELETE /pets/*
    response 200:
      - petstore_api no longer answers DELETE /pets/*, but petstore_web still reads response 200
        → petstore_api: keep answering 200 until petstore_web stops reading it
        → petstore_web: stop reading it
`, out)
		assert.NotContains(t, errOut, "Error:")
	})

	t.Run("a dropped request body says the consumer still sends it", func(t *testing.T) {
		out, errOut, err := runCanIDeploy(t, `{
		  "/pets": {
		    "post": {
		      "request": [
		        { "reason": "provider_resource_interaction_removed_but_still_consumed", "role": "provider" }
		      ]
		    }
		  }
		}`)

		require.ErrorIs(t, err, reports.ErrSilent)
		assert.Equal(t, `petstore_api 2.0.0 cannot be deployed to production

petstore_web (1.0.0, deployed):
  POST /pets
    request:
      - petstore_api dropped the request body of POST /pets, but petstore_web still sends it
        → petstore_api: keep accepting it until petstore_web stops sending it
        → petstore_web: stop sending it
`, out)
		assert.NotContains(t, errOut, "Error:")
	})
}
