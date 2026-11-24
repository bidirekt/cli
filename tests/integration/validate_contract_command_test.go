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
	restreports "github.com/bidirekt/cli/internal/reports/protocols/rest"
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

	restModules := []reports.Module{{Report: restreports.New()}}

	writeContract := func(t *testing.T) string {
		t.Helper()

		file := filepath.Join(t.TempDir(), "contract.yaml")
		require.NoError(t, os.WriteFile(file, []byte(content), 0o600))

		return file
	}

	runValidate := func(t *testing.T, modules []reports.Module, responder httpmock.Responder, args ...string) (string, string, error) {
		t.Helper()

		httpClient := components.NewHTTPClient()
		httpClient.UseBroker(brokerURL, "")
		httpmock.ActivateNonDefault(httpClient.StdClient())
		t.Cleanup(httpmock.DeactivateAndReset)
		httpmock.RegisterResponder(http.MethodPost, endpoint, responder)

		command := validate_contract.NewValidateCommand(
			validate_contract.NewValidateContractClient(httpClient),
			modules,
		)
		var out, errOut bytes.Buffer
		command.SetOut(&out)
		command.SetErr(&errOut)
		command.SetArgs(args)

		err := command.Execute()

		return out.String(), errOut.String(), err
	}

	unexpectedRequest := httpmock.NewErrorResponder(errors.New("unexpected request to the broker"))

	t.Run("compatible verdict posts participant+environment+contracts, prints can be deployed, exits 0", func(t *testing.T) {
		file := writeContract(t)

		var capturedBody []byte
		responder := func(req *http.Request) (*http.Response, error) {
			body, err := io.ReadAll(req.Body)
			if err != nil {
				return nil, err
			}
			capturedBody = body
			return httpmock.NewStringResponse(http.StatusOK, `{"message":"contract validated successfully","participant":"front","environment":"production","deployable":true,"results":{}}`), nil
		}

		out, errOut, err := runValidate(t, restModules, responder, file, "--participant", participant, "--environment", environment)

		require.NoError(t, err)
		assert.Equal(t, 1, httpmock.GetCallCountInfo()["POST "+endpoint])
		expectedBody, err := json.Marshal(map[string]any{
			"participant": participant,
			"environment": environment,
			"contracts":   []map[string]string{{"source": file, "content": content}},
		})
		require.NoError(t, err)
		assert.JSONEq(t, string(expectedBody), string(capturedBody))
		assert.Equal(t, "front local contract can be deployed to production\nnothing deployed in production to compare with\n", out)
		assert.Empty(t, errOut)
	})

	t.Run("compatible verdict names the counterparts it was compared with, in alphabetical order", func(t *testing.T) {
		file := writeContract(t)

		responseBody := `{
		  "message": "contract validated successfully",
		  "participant": "front",
		  "environment": "production",
		  "deployable": true,
		  "results": {
		    "site": { "deployable": true, "participantVersion": "v4", "endpoints": {} },
		    "admin": { "deployable": true, "participantVersion": "v2", "endpoints": {} }
		  }
		}`
		responder := httpmock.NewStringResponder(http.StatusOK, responseBody)

		out, errOut, err := runValidate(t, restModules, responder, file, "--participant", participant, "--environment", environment)

		require.NoError(t, err)
		assert.Equal(t, "front local contract can be deployed to production\ncompared with: admin, site\n", out)
		assert.Empty(t, errOut)
	})

	t.Run("incompatible verdict prints the break tree on stdout, exits silently non-zero", func(t *testing.T) {
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
		                "details": { "providerField": "UpdatePaymentRequest.currency", "property": "$.currency", "propertyType": "string", "consumerName": "front", "providerName": "payments" }
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
		responder := httpmock.NewStringResponder(http.StatusOK, responseBody)

		out, errOut, err := runValidate(t, restModules, responder, file, "--participant", participant, "--environment", environment)

		require.ErrorIs(t, err, reports.ErrSilent)
		assert.Equal(t, `front local contract cannot be deployed to production

payments (v2, deployed):
  UpdatePaymentRequest.currency — front doesn't send it, but payments requires it
    → front: send it
    → payments: stop requiring it, then mark it optional there
      · PUT /payments/* request at "$.currency" (payments: UpdatePaymentRequest.currency)
`, out)
		assert.Empty(t, errOut)
	})

	t.Run("compatible verdict reads the same when the counterparts carry no empty endpoints tree", func(t *testing.T) {
		file := writeContract(t)

		responseBody := `{
		  "message": "contract validated successfully",
		  "participant": "front",
		  "environment": "production",
		  "deployable": true,
		  "results": {
		    "site": { "deployable": true, "participantVersion": "v4" },
		    "admin": { "deployable": true, "participantVersion": "v2" }
		  }
		}`
		responder := httpmock.NewStringResponder(http.StatusOK, responseBody)

		out, errOut, err := runValidate(t, restModules, responder, file, "--participant", participant, "--environment", environment)

		require.NoError(t, err)
		assert.Equal(t, "front local contract can be deployed to production\ncompared with: admin, site\n", out)
		assert.Empty(t, errOut)
	})

	t.Run("incompatible verdict reads the same when a compatible counterpart carries no empty endpoints tree", func(t *testing.T) {
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
		                "details": { "providerField": "UpdatePaymentRequest.currency", "property": "$.currency", "propertyType": "string", "consumerName": "front", "providerName": "payments" }
		              }
		            ]
		          }
		        }
		      }
		    },
		    "users": {
		      "deployable": true,
		      "participantVersion": "v3"
		    }
		  }
		}`
		responder := httpmock.NewStringResponder(http.StatusOK, responseBody)

		out, errOut, err := runValidate(t, restModules, responder, file, "--participant", participant, "--environment", environment)

		require.ErrorIs(t, err, reports.ErrSilent)
		assert.Equal(t, `front local contract cannot be deployed to production

payments (v2, deployed):
  UpdatePaymentRequest.currency — front doesn't send it, but payments requires it
    → front: send it
    → payments: stop requiring it, then mark it optional there
      · PUT /payments/* request at "$.currency" (payments: UpdatePaymentRequest.currency)
`, out)
		assert.Empty(t, errOut)
	})

	for _, test := range []struct {
		name     string
		modules  []reports.Module
		protocol string
		tree     string
	}{
		{
			name:     "an endpoints tree without any module",
			protocol: "endpoints",
			tree:     `{ "/payments/*": { "put": { "request": [{ "reason": "provider_resource_not_found", "role": "consumer" }] } } }`,
		},
		{
			name:     "an operations tree beside the REST module",
			modules:  restModules,
			protocol: "operations",
			tree:     `{ "query": { "payment": { "returns": [{ "reason": "provider_resource_not_found", "role": "consumer" }] } } }`,
		},
	} {
		t.Run(test.name+" fails naming the protocol, printing nothing", func(t *testing.T) {
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
			      "` + test.protocol + `": ` + test.tree + `
			    }
			  }
			}`
			responder := httpmock.NewStringResponder(http.StatusOK, responseBody)

			out, errOut, err := runValidate(t, test.modules, responder, file, "--participant", participant, "--environment", environment)

			require.EqualError(t, err, "the broker reported protocol "+test.protocol+", which this bidirekt doesn't know; update bidirekt")
			assert.NotErrorIs(t, err, reports.ErrSilent)
			assert.Empty(t, out)
			assert.Empty(t, errOut)
		})
	}

	t.Run("validation failure prints the violation report on stderr, exits silently non-zero", func(t *testing.T) {
		file := writeContract(t)

		responder := httpmock.NewStringResponder(http.StatusBadRequest, `{"message":"contract validation failed","violations":[{"code":"schema.unresolved_name","path":"consumes;payments;rest;/payments/*;put;request","source":"contract.yaml","details":{"schema":"Payment","resource":"consumes payments PUT /payments/* request"}}]}`)

		out, errOut, err := runValidate(t, restModules, responder, file, "--participant", participant, "--environment", environment)

		require.ErrorIs(t, err, reports.ErrSilent)
		assert.Equal(t, "contract validation failed\n"+
			"  - contract.yaml: unresolved schema \"Payment\" referenced by consumes payments PUT /payments/* request\n",
			errOut)
		assert.Empty(t, out)
	})

	t.Run("optional on array items prints where it is and how to fix it", func(t *testing.T) {
		file := writeContract(t)

		responder := httpmock.NewStringResponder(http.StatusBadRequest, `{"message":"contract validation failed","violations":[{"code":"schema.optional_not_allowed","path":"schemas;Pet;properties;tags;items","source":"contract.yaml","details":{"position":"items"}}]}`)

		out, errOut, err := runValidate(t, restModules, responder, file, "--participant", participant, "--environment", environment)

		require.ErrorIs(t, err, reports.ErrSilent)
		assert.Equal(t, "contract validation failed\n"+
			"  - contract.yaml: \"optional\" is not allowed on array items at schemas Pet properties tags items, mark the property that holds the array optional, or use \"nullable: true\" for an item that may be null\n",
			errOut)
		assert.Empty(t, out)
	})

	t.Run("property name that confuses the path prints where it is and why", func(t *testing.T) {
		file := writeContract(t)

		responder := httpmock.NewStringResponder(http.StatusBadRequest, `{"message":"contract validation failed","violations":[{"code":"property.name_syntax","path":"schemas;Pet;properties;user.name","source":"contract.yaml","details":{"key":"user.name","error":"must not contain \".\", \"[\", \"]\", \"{\", \"}\" or \";\""}}]}`)

		out, errOut, err := runValidate(t, restModules, responder, file, "--participant", participant, "--environment", environment)

		require.ErrorIs(t, err, reports.ErrSilent)
		assert.Equal(t, "contract validation failed\n"+
			"  - contract.yaml: invalid property name \"user.name\" at schemas Pet properties, must not contain \".\", \"[\", \"]\", \"{\", \"}\" or \";\"\n",
			errOut)
		assert.Empty(t, out)
	})

	t.Run("unknown participant suggests creating it with create-participant", func(t *testing.T) {
		file := writeContract(t)

		responder := httpmock.NewStringResponder(http.StatusNotFound, `{"message":"participant not found"}`)

		out, errOut, err := runValidate(t, restModules, responder, file, "--participant", "ghost", "--environment", environment)

		require.Error(t, err)
		assert.NotErrorIs(t, err, reports.ErrSilent)
		assert.Equal(t, `participant "ghost" not found → create it with: bidirekt create-participant ghost`, err.Error())
		assert.Empty(t, out)
		assert.Empty(t, errOut)
	})

	t.Run("unknown environment returns the broker message as the error", func(t *testing.T) {
		file := writeContract(t)

		responder := httpmock.NewStringResponder(http.StatusNotFound, `{"message":"environment not found"}`)

		out, errOut, err := runValidate(t, restModules, responder, file, "--participant", participant, "--environment", "nowhere")

		require.Error(t, err)
		assert.NotErrorIs(t, err, reports.ErrSilent)
		assert.Equal(t, "cannot validate contract on broker: environment not found", err.Error())
		assert.Empty(t, out)
		assert.Empty(t, errOut)
	})

	t.Run("unsupported extension fails before any request", func(t *testing.T) {
		notes := filepath.Join(t.TempDir(), "notes.txt")
		require.NoError(t, os.WriteFile(notes, []byte("not a contract"), 0o600))

		_, _, err := runValidate(t, restModules, unexpectedRequest, notes, "--participant", participant, "--environment", environment)

		require.Error(t, err)
		assert.Equal(t, `unsupported contract file extension: "`+notes+`"`, err.Error())
		assert.Zero(t, httpmock.GetTotalCallCount())
	})

	t.Run("only --participant fails before any request", func(t *testing.T) {
		file := writeContract(t)

		_, _, err := runValidate(t, restModules, unexpectedRequest, file, "--participant", participant)

		require.Error(t, err)
		assert.Equal(t, "if any flags in the group [participant environment] are set they must all be set; missing [environment]", err.Error())
		assert.Zero(t, httpmock.GetTotalCallCount())
	})

	t.Run("no flags posts only the contracts, prints that the contract is valid, exits 0", func(t *testing.T) {
		file := writeContract(t)

		var capturedBody []byte
		responder := func(req *http.Request) (*http.Response, error) {
			body, err := io.ReadAll(req.Body)
			if err != nil {
				return nil, err
			}
			capturedBody = body
			return httpmock.NewStringResponse(http.StatusOK, `{"message":"contract validated successfully"}`), nil
		}

		out, errOut, err := runValidate(t, restModules, responder, file)

		require.NoError(t, err)
		assert.Equal(t, 1, httpmock.GetCallCountInfo()["POST "+endpoint])
		expectedBody, err := json.Marshal(map[string]any{
			"contracts": []map[string]string{{"source": file, "content": content}},
		})
		require.NoError(t, err)
		assert.JSONEq(t, string(expectedBody), string(capturedBody))
		assert.Equal(t, "contract is valid; add --participant and --environment to compare it with what is deployed\n", out)
		assert.Empty(t, errOut)
	})

	t.Run("no flags and a validation failure prints the violation report on stderr, exits silently non-zero", func(t *testing.T) {
		file := writeContract(t)

		responder := httpmock.NewStringResponder(http.StatusBadRequest, `{"message":"contract validation failed","violations":[{"code":"schema.unresolved_name","path":"consumes;payments;rest;/payments/*;put;request","source":"contract.yaml","details":{"schema":"Payment","resource":"consumes payments PUT /payments/* request"}}]}`)

		out, errOut, err := runValidate(t, restModules, responder, file)

		require.ErrorIs(t, err, reports.ErrSilent)
		assert.Equal(t, "contract validation failed\n"+
			"  - contract.yaml: unresolved schema \"Payment\" referenced by consumes payments PUT /payments/* request\n",
			errOut)
		assert.Empty(t, out)
	})
}
