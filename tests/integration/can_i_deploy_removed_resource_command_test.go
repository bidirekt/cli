package integration_test

import (
	"bytes"
	"net/http"
	"testing"

	"github.com/bidirekt/cli/internal/components"
	"github.com/bidirekt/cli/internal/features/can_i_deploy"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCanIDeployCommandRemovedResourceStillConsumed(t *testing.T) {
	const (
		brokerURL   = "http://localhost:8080"
		participant = "orders-api"
		version     = "v2"
		environment = "production"
		endpoint    = brokerURL + "/api/can-i-deploy"
	)

	httpClient := components.NewHTTPClient(&components.Config{BrokerURL: brokerURL})
	httpmock.ActivateNonDefault(httpClient.StdClient())
	defer httpmock.DeactivateAndReset()

	responseBody := `{
	  "message": "contract checked successfully",
	  "participant": "orders-api",
	  "version": "v2",
	  "environment": "production",
	  "deployable": false,
	  "results": {
	    "orders-web": {
	      "deployable": false,
	      "participantVersion": "v7",
	      "endpoints": {
	        "/users": {
	          "get": {
	            "200": [
	              { "reason": "provider_resource_removed_but_still_consumed" }
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

	require.ErrorIs(t, err, can_i_deploy.ErrSilent)
	assert.Equal(t, `❌ orders-api cannot be deployed to production

orders-web (v7):
  GET /users
    response 200:
      - resource removed but still consumed
`, out.String())
	assert.NotContains(t, errOut.String(), "Error:")
}
