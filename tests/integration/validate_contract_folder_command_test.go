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
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateContractFolderCommand(t *testing.T) {
	const (
		brokerURL   = "http://localhost:8080"
		participant = "front"
		environment = "production"
		endpoint    = brokerURL + "/api/contracts/validate"
	)

	t.Run("validates a folder with its subfolders and the files passed beside it as one request, each YAML file once, in path order", func(t *testing.T) {
		httpClient := components.NewHTTPClient(&components.Config{BrokerURL: brokerURL})
		httpmock.ActivateNonDefault(httpClient.StdClient())
		defer httpmock.DeactivateAndReset()

		directory := t.TempDir()
		contracts := filepath.Join(directory, "contracts")
		paymentsContent := "consumes:\n  payments:\n    rest:\n      /payments/*:\n        put:\n          request: Payment\n"
		schemasContent := "schemas:\n  Payment:\n    type: object\n    properties:\n      amount:\n        type: integer\n"
		usersContent := "consumes:\n  users:\n    rest:\n      /users/*:\n        get:\n          responses:\n            200: User\n"

		payments := filepath.Join(contracts, "checkout", "payments.yaml")
		schemas := filepath.Join(contracts, "schemas.yml")
		readme := filepath.Join(contracts, "README.md")
		users := filepath.Join(directory, "users.yaml")
		require.NoError(t, os.MkdirAll(filepath.Dir(payments), 0o700))
		require.NoError(t, os.WriteFile(payments, []byte(paymentsContent), 0o600))
		require.NoError(t, os.WriteFile(schemas, []byte(schemasContent), 0o600))
		require.NoError(t, os.WriteFile(readme, []byte("# front contracts\n"), 0o600))
		require.NoError(t, os.WriteFile(users, []byte(usersContent), 0o600))

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
		command.SetArgs([]string{users, contracts, payments, "--participant", participant, "--environment", environment})

		err := command.Execute()

		require.NoError(t, err)
		assert.Equal(t, 1, httpmock.GetCallCountInfo()["POST "+endpoint])
		expectedBody, err := json.Marshal(map[string]any{
			"participant": participant,
			"environment": environment,
			"contracts": []map[string]string{
				{"source": payments, "content": paymentsContent},
				{"source": schemas, "content": schemasContent},
				{"source": users, "content": usersContent},
			},
		})
		require.NoError(t, err)
		assert.JSONEq(t, string(expectedBody), string(capturedBody))
		assert.Equal(t, "front local contract can be deployed to production\nnothing deployed in production to compare with\n", out.String())
		assert.Empty(t, errOut.String())
	})

	t.Run("a folder without YAML files fails naming the folder before any request", func(t *testing.T) {
		httpClient := components.NewHTTPClient(&components.Config{BrokerURL: brokerURL})
		httpmock.ActivateNonDefault(httpClient.StdClient())
		defer httpmock.DeactivateAndReset()
		httpmock.RegisterNoResponder(httpmock.NewErrorResponder(errors.New("unexpected request to the broker")))

		docs := filepath.Join(t.TempDir(), "docs")
		require.NoError(t, os.MkdirAll(docs, 0o700))
		require.NoError(t, os.WriteFile(filepath.Join(docs, "README.md"), []byte("# docs\n"), 0o600))

		command := validate_contract.NewValidateCommand(
			validate_contract.NewValidateContractClient(httpClient),
		)
		var out, errOut bytes.Buffer
		command.SetOut(&out)
		command.SetErr(&errOut)
		command.SetArgs([]string{docs, "--participant", participant, "--environment", environment})

		err := command.Execute()

		require.Error(t, err)
		assert.Equal(t, `no contract YAML files in folder: "`+docs+`"`, err.Error())
		assert.Zero(t, httpmock.GetTotalCallCount())
		assert.Empty(t, out.String())
		assert.Empty(t, errOut.String())
	})
}
