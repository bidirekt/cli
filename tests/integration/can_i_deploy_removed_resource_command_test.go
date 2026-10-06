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
	              { "reason": "provider_resource_removed_but_still_consumed", "role": "provider" }
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

	require.ErrorIs(t, err, reports.ErrSilent)
	assert.Equal(t, `orders-api v2 cannot be deployed to production

orders-web (v7, deployed):
  GET /users
    response 200:
      - orders-api removed GET /users, but orders-web still calls it → keep it until orders-web stops calling it
`, out.String())
	assert.NotContains(t, errOut.String(), "Error:")
}
