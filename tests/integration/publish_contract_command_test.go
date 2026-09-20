package integration_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/bidirekt/cli/internal/components"
	"github.com/bidirekt/cli/internal/features/publish_contract"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPublishContractCommand(t *testing.T) {
	const (
		brokerURL   = "http://localhost:8080"
		participant = "pets-service"
		version     = "v1"
		endpoint    = brokerURL + "/api/contracts"
	)

	expectedBody := func(t *testing.T, fragments ...[2]string) string {
		t.Helper()

		contracts := make([]map[string]string, 0, len(fragments))
		for _, fragment := range fragments {
			contracts = append(contracts, map[string]string{"source": fragment[0], "content": fragment[1]})
		}

		body, err := json.Marshal(map[string]any{
			"participant": participant,
			"version":     version,
			"contracts":   contracts,
		})
		require.NoError(t, err)

		return string(body)
	}

	t.Run("publishes a single file as an array of one fragment, prints the message, exits 0", func(t *testing.T) {
		httpClient := components.NewHTTPClient(&components.Config{BrokerURL: brokerURL})
		httpmock.ActivateNonDefault(httpClient.StdClient())
		defer httpmock.DeactivateAndReset()

		const content = `{"provides":{"rest":{}}}`
		file := filepath.Join(t.TempDir(), "contract.json")
		require.NoError(t, os.WriteFile(file, []byte(content), 0o600))

		var capturedBody []byte
		httpmock.RegisterResponder(http.MethodPost, endpoint,
			func(req *http.Request) (*http.Response, error) {
				body, err := io.ReadAll(req.Body)
				if err != nil {
					return nil, err
				}
				capturedBody = body
				return httpmock.NewStringResponse(http.StatusOK, `{"message":"contract publish successful"}`), nil
			})

		command := publish_contract.NewPublishCommand(
			publish_contract.NewPublishContractClient(httpClient),
		)
		var out, errOut bytes.Buffer
		command.SetOut(&out)
		command.SetErr(&errOut)
		command.SetArgs([]string{file, "--participant", participant, "--version", version})

		err := command.Execute()

		require.NoError(t, err)
		assert.Equal(t, 1, httpmock.GetCallCountInfo()["POST "+endpoint])
		assert.JSONEq(t, expectedBody(t, [2]string{file, content}), string(capturedBody))
		assert.Contains(t, out.String(), participant+" contract publish successful")
		assert.Empty(t, errOut.String())
	})

	t.Run("publishes many YAML files as one request with a fragment per file, content verbatim", func(t *testing.T) {
		httpClient := components.NewHTTPClient(&components.Config{BrokerURL: brokerURL})
		httpmock.ActivateNonDefault(httpClient.StdClient())
		defer httpmock.DeactivateAndReset()

		directory := t.TempDir()
		petsContent := "# pets\nprovides:\n  rest:\n    /pets:\n      get:\n        responses:\n          200: Pets\n"
		storeContent := "provides:\n  rest:\n    /store:\n      get:\n        responses:\n          200: Store\n"
		schemasContent := "schemas:\n  Pets:\n    type: array\n    items:\n      ref: Pet\n"

		pets := filepath.Join(directory, "pets.yaml")
		store := filepath.Join(directory, "store.yml")
		schemas := filepath.Join(directory, "schemas.yaml")
		require.NoError(t, os.WriteFile(pets, []byte(petsContent), 0o600))
		require.NoError(t, os.WriteFile(store, []byte(storeContent), 0o600))
		require.NoError(t, os.WriteFile(schemas, []byte(schemasContent), 0o600))

		var capturedBody []byte
		httpmock.RegisterResponder(http.MethodPost, endpoint,
			func(req *http.Request) (*http.Response, error) {
				body, err := io.ReadAll(req.Body)
				if err != nil {
					return nil, err
				}
				capturedBody = body
				return httpmock.NewStringResponse(http.StatusOK, `{"message":"contract publish successful"}`), nil
			})

		command := publish_contract.NewPublishCommand(
			publish_contract.NewPublishContractClient(httpClient),
		)
		var out, errOut bytes.Buffer
		command.SetOut(&out)
		command.SetErr(&errOut)
		command.SetArgs([]string{pets, store, schemas, "--participant", participant, "--version", version})

		err := command.Execute()

		require.NoError(t, err)
		assert.Equal(t, 1, httpmock.GetCallCountInfo()["POST "+endpoint])
		assert.JSONEq(t, expectedBody(t,
			[2]string{pets, petsContent},
			[2]string{store, storeContent},
			[2]string{schemas, schemasContent},
		), string(capturedBody))
	})

	t.Run("unsupported extension in the list fails before any request", func(t *testing.T) {
		httpClient := components.NewHTTPClient(&components.Config{BrokerURL: brokerURL})
		httpmock.ActivateNonDefault(httpClient.StdClient())
		defer httpmock.DeactivateAndReset()

		directory := t.TempDir()
		contract := filepath.Join(directory, "contract.yaml")
		notes := filepath.Join(directory, "notes.txt")
		require.NoError(t, os.WriteFile(contract, []byte("provides:\n  rest: {}\n"), 0o600))
		require.NoError(t, os.WriteFile(notes, []byte("not a contract"), 0o600))

		command := publish_contract.NewPublishCommand(
			publish_contract.NewPublishContractClient(httpClient),
		)
		var out, errOut bytes.Buffer
		command.SetOut(&out)
		command.SetErr(&errOut)
		command.SetArgs([]string{contract, notes, "--participant", participant, "--version", version})

		err := command.Execute()

		require.Error(t, err)
		assert.Contains(t, err.Error(), notes)
		assert.Zero(t, httpmock.GetTotalCallCount())
	})

	t.Run("unreadable file in the list fails before any request", func(t *testing.T) {
		httpClient := components.NewHTTPClient(&components.Config{BrokerURL: brokerURL})
		httpmock.ActivateNonDefault(httpClient.StdClient())
		defer httpmock.DeactivateAndReset()

		directory := t.TempDir()
		contract := filepath.Join(directory, "contract.yaml")
		missing := filepath.Join(directory, "missing.yaml")
		require.NoError(t, os.WriteFile(contract, []byte("provides:\n  rest: {}\n"), 0o600))

		command := publish_contract.NewPublishCommand(
			publish_contract.NewPublishContractClient(httpClient),
		)
		var out, errOut bytes.Buffer
		command.SetOut(&out)
		command.SetErr(&errOut)
		command.SetArgs([]string{contract, missing, "--participant", participant, "--version", version})

		err := command.Execute()

		require.Error(t, err)
		assert.Contains(t, err.Error(), missing)
		assert.Zero(t, httpmock.GetTotalCallCount())
	})

	t.Run("validation failure renders each violation with its source, exits silently", func(t *testing.T) {
		httpClient := components.NewHTTPClient(&components.Config{BrokerURL: brokerURL})
		httpmock.ActivateNonDefault(httpClient.StdClient())
		defer httpmock.DeactivateAndReset()

		file := filepath.Join(t.TempDir(), "contract.yaml")
		require.NoError(t, os.WriteFile(file, []byte("provides:\n  rest: {}\n"), 0o600))

		httpmock.RegisterResponder(http.MethodPost, endpoint,
			httpmock.NewStringResponder(http.StatusBadRequest, `{"message":"contract validation failed","violations":[{"code":"schema.unresolved_ref","path":"schemas;Invoice;properties;payment","source":"billing.yaml","details":{"schema":"Payment","property":"Invoice.payment"}},{"code":"schema.unresolved_name","path":"provides;rest;/pets;get;responses;200","source":"pets.yaml","details":{"schema":"Pets","resource":"provides GET /pets 200"}}]}`))

		command := publish_contract.NewPublishCommand(
			publish_contract.NewPublishContractClient(httpClient),
		)
		var out, errOut bytes.Buffer
		command.SetOut(&out)
		command.SetErr(&errOut)
		command.SetArgs([]string{file, "--participant", participant, "--version", version})

		err := command.Execute()

		require.ErrorIs(t, err, publish_contract.ErrSilent)
		assert.Equal(t, "❌ contract validation failed\n"+
			"  - billing.yaml: unresolved ref \"Payment\" in Invoice.payment\n"+
			"  - pets.yaml: unresolved schema \"Pets\" referenced by provides GET /pets 200\n",
			errOut.String())
		assert.Empty(t, out.String())
	})

	t.Run("validation failure with null details renders the violation", func(t *testing.T) {
		httpClient := components.NewHTTPClient(&components.Config{BrokerURL: brokerURL})
		httpmock.ActivateNonDefault(httpClient.StdClient())
		defer httpmock.DeactivateAndReset()

		file := filepath.Join(t.TempDir(), "contract.yaml")
		require.NoError(t, os.WriteFile(file, []byte("provides:\n  rest: {}\n"), 0o600))

		httpmock.RegisterResponder(http.MethodPost, endpoint,
			httpmock.NewStringResponder(http.StatusBadRequest, `{"message":"contract validation failed","violations":[{"code":"schema.array_without_items","path":"schemas;Pets","source":"api.yaml","details":null}]}`))

		command := publish_contract.NewPublishCommand(
			publish_contract.NewPublishContractClient(httpClient),
		)
		var out, errOut bytes.Buffer
		command.SetOut(&out)
		command.SetErr(&errOut)
		command.SetArgs([]string{file, "--participant", participant, "--version", version})

		err := command.Execute()

		require.ErrorIs(t, err, publish_contract.ErrSilent)
		assert.Equal(t, "❌ contract validation failed\n"+
			"  - api.yaml: array schema without items at schemas;Pets\n",
			errOut.String())
		assert.Empty(t, out.String())
	})

	t.Run("unknown violation code falls back to the code with its details", func(t *testing.T) {
		httpClient := components.NewHTTPClient(&components.Config{BrokerURL: brokerURL})
		httpmock.ActivateNonDefault(httpClient.StdClient())
		defer httpmock.DeactivateAndReset()

		file := filepath.Join(t.TempDir(), "contract.yaml")
		require.NoError(t, os.WriteFile(file, []byte("provides:\n  rest: {}\n"), 0o600))

		httpmock.RegisterResponder(http.MethodPost, endpoint,
			httpmock.NewStringResponder(http.StatusBadRequest, `{"message":"contract validation failed","violations":[{"code":"something.new","path":"provides;rest;/pets","source":"api.yaml","details":{"hint":"x"}}]}`))

		command := publish_contract.NewPublishCommand(
			publish_contract.NewPublishContractClient(httpClient),
		)
		var out, errOut bytes.Buffer
		command.SetOut(&out)
		command.SetErr(&errOut)
		command.SetArgs([]string{file, "--participant", participant, "--version", version})

		err := command.Execute()

		require.ErrorIs(t, err, publish_contract.ErrSilent)
		assert.Equal(t, "❌ contract validation failed\n"+
			"  - api.yaml: something.new at provides;rest;/pets (hint: x)\n",
			errOut.String())
		assert.Empty(t, out.String())
	})

	t.Run("failure without violations keeps the single-line message", func(t *testing.T) {
		httpClient := components.NewHTTPClient(&components.Config{BrokerURL: brokerURL})
		httpmock.ActivateNonDefault(httpClient.StdClient())
		defer httpmock.DeactivateAndReset()

		file := filepath.Join(t.TempDir(), "contract.yaml")
		require.NoError(t, os.WriteFile(file, []byte("provides:\n  rest: {}\n"), 0o600))

		httpmock.RegisterResponder(http.MethodPost, endpoint,
			httpmock.NewStringResponder(http.StatusNotFound, `{"message":"participant not found"}`))

		command := publish_contract.NewPublishCommand(
			publish_contract.NewPublishContractClient(httpClient),
		)
		var out, errOut bytes.Buffer
		command.SetOut(&out)
		command.SetErr(&errOut)
		command.SetArgs([]string{file, "--participant", participant, "--version", version})

		err := command.Execute()

		require.Error(t, err)
		assert.NotErrorIs(t, err, publish_contract.ErrSilent)
		assert.Equal(t, "cannot post contract to broker: participant not found", err.Error())
		assert.Empty(t, errOut.String())
	})

	t.Run("missing --participant fails before any request", func(t *testing.T) {
		httpClient := components.NewHTTPClient(&components.Config{BrokerURL: brokerURL})
		httpmock.ActivateNonDefault(httpClient.StdClient())
		defer httpmock.DeactivateAndReset()

		file := filepath.Join(t.TempDir(), "contract.json")
		require.NoError(t, os.WriteFile(file, []byte(`{}`), 0o600))

		command := publish_contract.NewPublishCommand(
			publish_contract.NewPublishContractClient(httpClient),
		)
		var out, errOut bytes.Buffer
		command.SetOut(&out)
		command.SetErr(&errOut)
		command.SetArgs([]string{file, "--version", version})

		err := command.Execute()

		require.Error(t, err)
		assert.Zero(t, httpmock.GetTotalCallCount())
	})
}
