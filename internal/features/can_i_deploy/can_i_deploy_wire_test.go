package can_i_deploy

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCanIDeployResponseBodyUnmarshalsResults(t *testing.T) {
	payload := `{
	  "message": "contract checked successfully",
	  "participant": "payments-web",
	  "version": "abc123",
	  "environment": "production",
	  "deployable": false,
	  "results": {
	    "payments-api": {
	      "deployable": false,
	      "participantVersion": "3.1.0",
	      "endpoints": {
	        "/payments/*": {
	          "get": {
	            "request": [
	              { "reason": "property_missing_in_provider", "details": { "property": "currency" } }
	            ],
	            "200": [
	              {
	                "reason": "property_type_mismatch",
	                "details": {
	                  "property": "amount",
	                  "consumerPropertyType": "string",
	                  "providerPropertyType": "number"
	                }
	              }
	            ]
	          }
	        }
	      }
	    },
	    "users": {
	      "deployable": true,
	      "participantVersion": null,
	      "endpoints": {}
	    }
	  }
	}`

	var body CanIDeployResponseBody
	require.NoError(t, json.Unmarshal([]byte(payload), &body))

	assert.Equal(t, "contract checked successfully", body.Message)
	assert.False(t, body.Deployable)
	assert.Equal(t, "production", body.Environment)
	require.Len(t, body.Results, 2)

	payments := body.Results["payments-api"]
	assert.False(t, payments.Deployable)
	require.NotNil(t, payments.ParticipantVersion)
	assert.Equal(t, "3.1.0", *payments.ParticipantVersion)

	interactions := payments.Endpoints["/payments/*"]["get"]
	require.Len(t, interactions, 2)

	requestBreaks := interactions["request"]
	require.Len(t, requestBreaks, 1)
	assert.Equal(t, "property_missing_in_provider", requestBreaks[0].Reason)
	assert.Equal(t, map[string]string{"property": "currency"}, requestBreaks[0].Details)

	responseBreaks := interactions["200"]
	require.Len(t, responseBreaks, 1)
	assert.Equal(t, "property_type_mismatch", responseBreaks[0].Reason)
	assert.Equal(t, map[string]string{
		"property":             "amount",
		"consumerPropertyType": "string",
		"providerPropertyType": "number",
	}, responseBreaks[0].Details)

	users := body.Results["users"]
	assert.True(t, users.Deployable)
	assert.Nil(t, users.ParticipantVersion)
	require.NotNil(t, users.Endpoints)
	assert.Empty(t, users.Endpoints)
}
