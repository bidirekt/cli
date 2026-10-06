package integration_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/bidirekt/cli/internal/components"
	"github.com/bidirekt/cli/internal/features/validate_contract"
	"github.com/bidirekt/cli/internal/reports"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateContractCommand(t *testing.T) {
	const (
		brokerURL   = "http://localhost:8080"
		participant = "front"
		environment = "production"
		endpoint    = brokerURL + "/api/contracts/validate"
		content     = "consumes:\n  payments:\n    rest:\n      /payments/*:\n        put:\n          request: Payment\n"
	)

	writeContract := func(t *testing.T) string {
		t.Helper()

		file := filepath.Join(t.TempDir(), "contract.yaml")
		require.NoError(t, os.WriteFile(file, []byte(content), 0o600))

		return file
	}

	t.Run("compatible verdict posts participant+environment+contracts, prints can be deployed, exits 0", func(t *testing.T) {
		httpClient := components.NewHTTPClient(&components.Config{BrokerURL: brokerURL})
		httpmock.ActivateNonDefault(httpClient.StdClient())
		defer httpmock.DeactivateAndReset()

		file := writeContract(t)

		var capturedBody []byte
		httpmock.RegisterResponder(http.MethodPost, endpoint,
			func(req *http.Request) (*http.Response, error) {
				body, err := io.ReadAll(req.Body)
				if err != nil {
					return nil, err
				}
				capturedBody = body
				return httpmock.NewStringResponse(http.StatusOK, `{"message":"contract validated successfully","participant":"front","environment":"production","deployable":true,"results":{}}`), nil
			})

		command := validate_contract.NewValidateCommand(
			validate_contract.NewValidateContractClient(httpClient),
		)
		var out, errOut bytes.Buffer
		command.SetOut(&out)
		command.SetErr(&errOut)
		command.SetArgs([]string{file, "--participant", participant, "--environment", environment})

		err := command.Execute()

		require.NoError(t, err)
		assert.Equal(t, 1, httpmock.GetCallCountInfo()["POST "+endpoint])
		expectedBody, err := json.Marshal(map[string]any{
			"participant": participant,
			"environment": environment,
			"contracts":   []map[string]string{{"source": file, "content": content}},
		})
		require.NoError(t, err)
		assert.JSONEq(t, string(expectedBody), string(capturedBody))
		assert.Equal(t, "front local contract can be deployed to production\n", out.String())
		assert.Empty(t, errOut.String())
	})

	t.Run("incompatible verdict prints the break tree on stdout, exits silently non-zero", func(t *testing.T) {
		httpClient := components.NewHTTPClient(&components.Config{BrokerURL: brokerURL})
		httpmock.ActivateNonDefault(httpClient.StdClient())
		defer httpmock.DeactivateAndReset()

		file := writeContract(t)

		responseBody := `{
		  "message": "contract validated successfully",
		  "participant": "front",
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

		command := validate_contract.NewValidateCommand(
			validate_contract.NewValidateContractClient(httpClient),
		)
		var out, errOut bytes.Buffer
		command.SetOut(&out)
		command.SetErr(&errOut)
		command.SetArgs([]string{file, "--participant", participant, "--environment", environment})

		err := command.Execute()

		require.ErrorIs(t, err, reports.ErrSilent)
		assert.Equal(t, `front local contract cannot be deployed to production

payments (v2, deployed):
  PUT /payments/*
    request:
      - front doesn't send "$.currency", but payments requires it → send it
`, out.String())
		assert.Empty(t, errOut.String())
	})

	t.Run("validation failure prints the violation report on stderr, exits silently non-zero", func(t *testing.T) {
		httpClient := components.NewHTTPClient(&components.Config{BrokerURL: brokerURL})
		httpmock.ActivateNonDefault(httpClient.StdClient())
		defer httpmock.DeactivateAndReset()

		file := writeContract(t)

		httpmock.RegisterResponder(http.MethodPost, endpoint,
			httpmock.NewStringResponder(http.StatusBadRequest, `{"message":"contract validation failed","violations":[{"code":"schema.unresolved_name","path":"consumes;payments;rest;/payments/*;put;request","source":"contract.yaml","details":{"schema":"Payment","resource":"consumes payments PUT /payments/* request"}}]}`))

		command := validate_contract.NewValidateCommand(
			validate_contract.NewValidateContractClient(httpClient),
		)
		var out, errOut bytes.Buffer
		command.SetOut(&out)
		command.SetErr(&errOut)
		command.SetArgs([]string{file, "--participant", participant, "--environment", environment})

		err := command.Execute()

		require.ErrorIs(t, err, reports.ErrSilent)
		assert.Equal(t, "contract validation failed\n"+
			"  - contract.yaml: unresolved schema \"Payment\" referenced by consumes payments PUT /payments/* request\n",
			errOut.String())
		assert.Empty(t, out.String())
	})

	t.Run("unknown participant returns the broker message as the error", func(t *testing.T) {
		httpClient := components.NewHTTPClient(&components.Config{BrokerURL: brokerURL})
		httpmock.ActivateNonDefault(httpClient.StdClient())
		defer httpmock.DeactivateAndReset()

		file := writeContract(t)

		httpmock.RegisterResponder(http.MethodPost, endpoint,
			httpmock.NewStringResponder(http.StatusNotFound, `{"message":"participant not found"}`))

		command := validate_contract.NewValidateCommand(
			validate_contract.NewValidateContractClient(httpClient),
		)
		var out, errOut bytes.Buffer
		command.SetOut(&out)
		command.SetErr(&errOut)
		command.SetArgs([]string{file, "--participant", "ghost", "--environment", environment})

		err := command.Execute()

		require.Error(t, err)
		assert.NotErrorIs(t, err, reports.ErrSilent)
		assert.Equal(t, "cannot validate contract on broker: participant not found", err.Error())
		assert.Empty(t, out.String())
		assert.Empty(t, errOut.String())
	})

	t.Run("unknown environment returns the broker message as the error", func(t *testing.T) {
		httpClient := components.NewHTTPClient(&components.Config{BrokerURL: brokerURL})
		httpmock.ActivateNonDefault(httpClient.StdClient())
		defer httpmock.DeactivateAndReset()

		file := writeContract(t)

		httpmock.RegisterResponder(http.MethodPost, endpoint,
			httpmock.NewStringResponder(http.StatusNotFound, `{"message":"environment not found"}`))

		command := validate_contract.NewValidateCommand(
			validate_contract.NewValidateContractClient(httpClient),
		)
		var out, errOut bytes.Buffer
		command.SetOut(&out)
		command.SetErr(&errOut)
		command.SetArgs([]string{file, "--participant", participant, "--environment", "nowhere"})

		err := command.Execute()

		require.Error(t, err)
		assert.NotErrorIs(t, err, reports.ErrSilent)
		assert.Equal(t, "cannot validate contract on broker: environment not found", err.Error())
		assert.Empty(t, out.String())
		assert.Empty(t, errOut.String())
	})

	t.Run("unsupported extension fails before any request", func(t *testing.T) {
		httpClient := components.NewHTTPClient(&components.Config{BrokerURL: brokerURL})
		httpmock.ActivateNonDefault(httpClient.StdClient())
		defer httpmock.DeactivateAndReset()
		httpmock.RegisterNoResponder(httpmock.NewErrorResponder(errors.New("unexpected request to the broker")))

		notes := filepath.Join(t.TempDir(), "notes.txt")
		require.NoError(t, os.WriteFile(notes, []byte("not a contract"), 0o600))

		command := validate_contract.NewValidateCommand(
			validate_contract.NewValidateContractClient(httpClient),
		)
		var out, errOut bytes.Buffer
		command.SetOut(&out)
		command.SetErr(&errOut)
		command.SetArgs([]string{notes, "--participant", participant, "--environment", environment})

		err := command.Execute()

		require.Error(t, err)
		assert.Equal(t, `unsupported contract file extension: "`+notes+`"`, err.Error())
		assert.Zero(t, httpmock.GetTotalCallCount())
	})

	t.Run("missing --environment fails before any request", func(t *testing.T) {
		httpClient := components.NewHTTPClient(&components.Config{BrokerURL: brokerURL})
		httpmock.ActivateNonDefault(httpClient.StdClient())
		defer httpmock.DeactivateAndReset()
		httpmock.RegisterNoResponder(httpmock.NewErrorResponder(errors.New("unexpected request to the broker")))

		file := writeContract(t)

		command := validate_contract.NewValidateCommand(
			validate_contract.NewValidateContractClient(httpClient),
		)
		var out, errOut bytes.Buffer
		command.SetOut(&out)
		command.SetErr(&errOut)
		command.SetArgs([]string{file, "--participant", participant})

		err := command.Execute()

		require.Error(t, err)
		assert.Equal(t, `required flag(s) "environment" not set`, err.Error())
		assert.Zero(t, httpmock.GetTotalCallCount())
	})
}
