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
	"github.com/bidirekt/cli/internal/features/publish_contract"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPublishContractFolderCommand(t *testing.T) {
	const (
		brokerURL   = "http://localhost:8080"
		participant = "pets-service"
		version     = "v1"
		endpoint    = brokerURL + "/api/contracts"
	)

	t.Run("publishes a folder with its subfolders and the files passed beside it as one request, each YAML file once, in path order", func(t *testing.T) {
		httpClient := components.NewHTTPClient(&components.Config{BrokerURL: brokerURL})
		httpmock.ActivateNonDefault(httpClient.StdClient())
		defer httpmock.DeactivateAndReset()

		directory := t.TempDir()
		contracts := filepath.Join(directory, "contracts")
		petsContent := "provides:\n  rest:\n    /pets:\n      get:\n        responses:\n          200: Pets\n"
		schemasContent := "schemas:\n  Pets:\n    type: array\n    items:\n      ref: Pet\n"
		storeContent := "provides:\n  rest:\n    /store:\n      get:\n        responses:\n          200: Store\n"

		pets := filepath.Join(contracts, "rest", "pets.yaml")
		schemas := filepath.Join(contracts, "schemas.yml")
		readme := filepath.Join(contracts, "README.md")
		store := filepath.Join(directory, "store.yaml")
		require.NoError(t, os.MkdirAll(filepath.Dir(pets), 0o700))
		require.NoError(t, os.WriteFile(pets, []byte(petsContent), 0o600))
		require.NoError(t, os.WriteFile(schemas, []byte(schemasContent), 0o600))
		require.NoError(t, os.WriteFile(readme, []byte("# pets-service contracts\n"), 0o600))
		require.NoError(t, os.WriteFile(store, []byte(storeContent), 0o600))

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
		command.SetArgs([]string{store, contracts, pets, "--participant", participant, "--version", version})

		err := command.Execute()

		require.NoError(t, err)
		assert.Equal(t, 1, httpmock.GetCallCountInfo()["POST "+endpoint])
		expectedBody, err := json.Marshal(map[string]any{
			"participant": participant,
			"version":     version,
			"contracts": []map[string]string{
				{"source": pets, "content": petsContent},
				{"source": schemas, "content": schemasContent},
				{"source": store, "content": storeContent},
			},
		})
		require.NoError(t, err)
		assert.JSONEq(t, string(expectedBody), string(capturedBody))
		assert.Equal(t, participant+" contract publish successful\n", out.String())
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

		command := publish_contract.NewPublishCommand(
			publish_contract.NewPublishContractClient(httpClient),
		)
		var out, errOut bytes.Buffer
		command.SetOut(&out)
		command.SetErr(&errOut)
		command.SetArgs([]string{docs, "--participant", participant, "--version", version})

		err := command.Execute()

		require.Error(t, err)
		assert.Equal(t, `no contract YAML files in folder: "`+docs+`"`, err.Error())
		assert.Zero(t, httpmock.GetTotalCallCount())
		assert.Empty(t, out.String())
		assert.Empty(t, errOut.String())
	})
}
