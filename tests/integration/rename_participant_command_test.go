package integration_test

import (
	"bytes"
	"io"
	"net/http"
	"testing"

	"github.com/bidirekt/cli/internal/components"
	"github.com/bidirekt/cli/internal/features/rename_participant"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenameParticipantCommand(t *testing.T) {
	const (
		brokerURL = "http://localhost:8080"
		oldName   = "pets-service"
		newName   = "orders-service"
		endpoint  = brokerURL + "/api/participants/rename"
	)

	t.Run("renames a participant, posts name+newName, prints the message, exits 0", func(t *testing.T) {
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
				return httpmock.NewStringResponse(http.StatusOK, `{"success":true,"message":"participant renamed"}`), nil
			})

		command := rename_participant.NewRenameParticipantCommand(
			rename_participant.NewRenameParticipantClient(httpClient),
		)
		var out, errOut bytes.Buffer
		command.SetOut(&out)
		command.SetErr(&errOut)
		command.SetArgs([]string{oldName, newName})

		err := command.Execute()

		require.NoError(t, err)
		assert.Equal(t, 1, httpmock.GetCallCountInfo()["POST "+endpoint])
		assert.JSONEq(t, `{"oldName":"pets-service","newName":"orders-service"}`, string(capturedBody))
		assert.Contains(t, out.String(), oldName+" participant renamed to "+newName)
		assert.Empty(t, errOut.String())
	})

	t.Run("only one argument fails before any request", func(t *testing.T) {
		httpClient := components.NewHTTPClient(&components.Config{BrokerURL: brokerURL})
		httpmock.ActivateNonDefault(httpClient.StdClient())
		defer httpmock.DeactivateAndReset()

		command := rename_participant.NewRenameParticipantCommand(
			rename_participant.NewRenameParticipantClient(httpClient),
		)
		var out, errOut bytes.Buffer
		command.SetOut(&out)
		command.SetErr(&errOut)
		command.SetArgs([]string{oldName})

		err := command.Execute()

		require.Error(t, err)
		assert.Zero(t, httpmock.GetTotalCallCount())
	})

	t.Run("non-200 response surfaces the broker body and exits non-zero", func(t *testing.T) {
		httpClient := components.NewHTTPClient(&components.Config{BrokerURL: brokerURL})
		httpmock.ActivateNonDefault(httpClient.StdClient())
		defer httpmock.DeactivateAndReset()

		httpmock.RegisterResponder(http.MethodPost, endpoint,
			httpmock.NewStringResponder(http.StatusBadRequest, `{"success":false,"message":"participant already exists"}`))

		command := rename_participant.NewRenameParticipantCommand(
			rename_participant.NewRenameParticipantClient(httpClient),
		)
		var out, errOut bytes.Buffer
		command.SetOut(&out)
		command.SetErr(&errOut)
		command.SetArgs([]string{oldName, newName})

		err := command.Execute()

		require.Error(t, err)
		assert.Contains(t, err.Error(), "participant already exists")
	})
}
