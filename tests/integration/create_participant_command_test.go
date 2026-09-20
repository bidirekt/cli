package integration_test

import (
	"bytes"
	"io"
	"net/http"
	"testing"

	"github.com/bidirekt/cli/internal/components"
	"github.com/bidirekt/cli/internal/features/create_participant"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateParticipantCommand(t *testing.T) {
	const (
		brokerURL = "http://localhost:8080"
		name      = "pets-service"
		endpoint  = brokerURL + "/api/participants"
	)

	t.Run("creates a participant, posts name, prints the message, exits 0", func(t *testing.T) {
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
				return httpmock.NewStringResponse(http.StatusOK, `{"success":true,"message":"participant created"}`), nil
			})

		command := create_participant.NewCreateParticipantCommand(
			create_participant.NewCreateParticipantClient(httpClient),
		)
		var out, errOut bytes.Buffer
		command.SetOut(&out)
		command.SetErr(&errOut)
		command.SetArgs([]string{name})

		err := command.Execute()

		require.NoError(t, err)
		assert.Equal(t, 1, httpmock.GetCallCountInfo()["POST "+endpoint])
		assert.JSONEq(t, `{"participant":"pets-service"}`, string(capturedBody))
		assert.Contains(t, out.String(), name+" participant created")
		assert.Empty(t, errOut.String())
	})

	t.Run("non-200 response surfaces the broker body and exits non-zero", func(t *testing.T) {
		httpClient := components.NewHTTPClient(&components.Config{BrokerURL: brokerURL})
		httpmock.ActivateNonDefault(httpClient.StdClient())
		defer httpmock.DeactivateAndReset()

		httpmock.RegisterResponder(http.MethodPost, endpoint,
			httpmock.NewStringResponder(http.StatusBadRequest, `{"success":false,"message":"participant invalid input"}`))

		command := create_participant.NewCreateParticipantCommand(
			create_participant.NewCreateParticipantClient(httpClient),
		)
		var out, errOut bytes.Buffer
		command.SetOut(&out)
		command.SetErr(&errOut)
		command.SetArgs([]string{name})

		err := command.Execute()

		require.Error(t, err)
		assert.Contains(t, err.Error(), "participant invalid input")
	})
}
