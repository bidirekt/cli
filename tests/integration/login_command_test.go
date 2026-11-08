package integration_test

import (
	"bytes"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/bidirekt/cli/internal/components"
	"github.com/bidirekt/cli/internal/features/login"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoginCommand(t *testing.T) {
	const (
		acmeURL       = "https://broker.acme.internal"
		globexURL     = "https://broker.globex.internal"
		acmeKey       = "bdk_acme_new"
		previousKey   = "bdk_acme_previous"
		environments  = `{"message":"environments listed successfully","environments":["production"]}`
		noTerminalMsg = "no terminal to sign in — set BIDIREKT_BROKER_URL and BIDIREKT_API_KEY"
	)

	isolateConfigFile := func(t *testing.T, content string) string {
		t.Helper()

		configFilePath := filepath.Join(t.TempDir(), "bidirekt", "config.json")
		t.Setenv("BIDIREKT_CONFIG_FILE", configFilePath)
		t.Setenv("BIDIREKT_BROKER_URL", "")
		t.Setenv("BIDIREKT_API_KEY", "")
		t.Setenv("BIDIREKT_PROFILE", "")
		if content != "" {
			require.NoError(t, os.MkdirAll(filepath.Dir(configFilePath), 0o700))
			require.NoError(t, os.WriteFile(configFilePath, []byte(content), 0o600))
		}

		return configFilePath
	}

	readConfigFile := func(t *testing.T, configFilePath string) string {
		t.Helper()

		content, err := os.ReadFile(configFilePath)
		require.NoError(t, err)

		return string(content)
	}

	type loginExecution struct {
		stdout        string
		stderr        string
		err           error
		authorization string
		calls         int
	}

	type loginRun struct {
		stdinIsTerminal bool
		stdin           string
		apiKey          string
		args            []string
		endpoint        string
		responder       httpmock.Responder
	}

	executeLogin := func(t *testing.T, run loginRun) loginExecution {
		t.Helper()

		httpClient := components.NewHTTPClient()
		httpmock.ActivateNonDefault(httpClient.StdClient())
		t.Cleanup(httpmock.DeactivateAndReset)

		var authorization string
		if run.responder != nil {
			httpmock.RegisterResponder(http.MethodGet, run.endpoint, func(request *http.Request) (*http.Response, error) {
				authorization = request.Header.Get("Authorization")
				return run.responder(request)
			})
		}

		readSecret := func(any) (string, error) { return run.apiKey, nil }
		command := login.NewLoginCommand(components.NewConfig(), login.NewLoginClient(httpClient), func(any) bool { return run.stdinIsTerminal }, readSecret)
		var out, errOut bytes.Buffer
		command.SetIn(strings.NewReader(run.stdin))
		command.SetOut(&out)
		command.SetErr(&errOut)
		command.SetArgs(run.args)

		err := command.Execute()

		calls := 0
		for _, count := range httpmock.GetCallCountInfo() {
			calls += count
		}
		return loginExecution{stdout: out.String(), stderr: errOut.String(), err: err, authorization: authorization, calls: calls}
	}

	signInAcme := func(stdin string, args ...string) loginRun {
		return loginRun{
			stdinIsTerminal: true,
			stdin:           stdin,
			apiKey:          acmeKey,
			args:            args,
			endpoint:        acmeURL + "/api/environments",
			responder:       httpmock.NewStringResponder(http.StatusOK, environments),
		}
	}

	t.Run("asks the profile, the URL and the key, checks the key and saves both into the chosen profile only", func(t *testing.T) {
		configFilePath := isolateConfigFile(t, `{"profiles":{"globex":{"brokerUrl":"`+globexURL+`","apiKey":"bdk_globex"}}}`)

		execution := executeLogin(t, signInAcme("acme\n"+acmeURL+"\n"))

		require.NoError(t, execution.err)
		assert.Equal(t, 1, execution.calls)
		assert.Equal(t, "Bearer "+acmeKey, execution.authorization)
		assert.Equal(t, "Signed in to "+acmeURL+" as profile acme\n", execution.stdout)
		assert.Equal(t, "Profile [default]: Broker URL: Create an API key at "+acmeURL+"/profile/api-keys/new and paste it here.\nAPI key: \n", execution.stderr)
		assert.JSONEq(t, `{"profiles":{"globex":{"brokerUrl":"`+globexURL+`","apiKey":"bdk_globex"},"acme":{"brokerUrl":"`+acmeURL+`","apiKey":"`+acmeKey+`"}}}`, readConfigFile(t, configFilePath))
	})

	t.Run("the current profile comes from BIDIREKT_PROFILE and Enter keeps it", func(t *testing.T) {
		configFilePath := isolateConfigFile(t, "")
		t.Setenv("BIDIREKT_PROFILE", "acme")

		execution := executeLogin(t, signInAcme("\n"+acmeURL+"\n"))

		require.NoError(t, execution.err)
		assert.True(t, strings.HasPrefix(execution.stderr, "Profile [acme]: "))
		assert.JSONEq(t, `{"profiles":{"acme":{"brokerUrl":"`+acmeURL+`","apiKey":"`+acmeKey+`"}}}`, readConfigFile(t, configFilePath))
	})

	t.Run("--profile skips the profile question and Enter keeps the profile's URL", func(t *testing.T) {
		configFilePath := isolateConfigFile(t, `{"profiles":{"acme":{"brokerUrl":"`+acmeURL+`/"}}}`)

		execution := executeLogin(t, signInAcme("\n", "--profile", "acme"))

		require.NoError(t, execution.err)
		assert.Equal(t, "Broker URL ["+acmeURL+"/]: Create an API key at "+acmeURL+"/profile/api-keys/new and paste it here.\nAPI key: \n", execution.stderr)
		assert.JSONEq(t, `{"profiles":{"acme":{"brokerUrl":"`+acmeURL+`/","apiKey":"`+acmeKey+`"}}}`, readConfigFile(t, configFilePath))
	})

	t.Run("an invalid URL is asked again", func(t *testing.T) {
		isolateConfigFile(t, "")

		execution := executeLogin(t, signInAcme("broker.acme.com\n"+acmeURL+"\n", "--profile", "acme"))

		require.NoError(t, execution.err)
		assert.True(t, strings.HasPrefix(execution.stderr, `Broker URL: invalid broker URL "broker.acme.com" — use http:// or https:// followed by a host`+"\nBroker URL: "))
	})

	t.Run("replacing a different key warns that the previous one is still valid at the previous URL", func(t *testing.T) {
		isolateConfigFile(t, `{"profiles":{"acme":{"brokerUrl":"`+globexURL+`/","apiKey":"`+previousKey+`"}}}`)

		execution := executeLogin(t, signInAcme(acmeURL+"\n", "--profile", "acme"))

		require.NoError(t, execution.err)
		assert.True(t, strings.HasSuffix(execution.stderr, "the previous key is still valid — revoke it at "+globexURL+"/profile\n"))
	})

	t.Run("signing in again with the same key does not warn", func(t *testing.T) {
		isolateConfigFile(t, `{"profiles":{"acme":{"brokerUrl":"`+acmeURL+`","apiKey":"`+acmeKey+`"}}}`)

		execution := executeLogin(t, signInAcme(acmeURL+"\n", "--profile", "acme"))

		require.NoError(t, execution.err)
		assert.NotContains(t, execution.stderr, "previous key")
	})

	t.Run("with both environment variables set it warns they win, and never uses them", func(t *testing.T) {
		configFilePath := isolateConfigFile(t, "")
		t.Setenv("BIDIREKT_BROKER_URL", globexURL)
		t.Setenv("BIDIREKT_API_KEY", "bdk_globex")

		execution := executeLogin(t, signInAcme(acmeURL+"\n", "--profile", "acme"))

		require.NoError(t, execution.err)
		assert.Equal(t, "Bearer "+acmeKey, execution.authorization)
		assert.True(t, strings.HasSuffix(execution.stderr, "BIDIREKT_BROKER_URL and BIDIREKT_API_KEY are set and win over this profile\n"))
		assert.JSONEq(t, `{"profiles":{"acme":{"brokerUrl":"`+acmeURL+`","apiKey":"`+acmeKey+`"}}}`, readConfigFile(t, configFilePath))
	})

	for _, variable := range []string{"BIDIREKT_BROKER_URL", "BIDIREKT_API_KEY"} {
		t.Run("with only "+variable+" set it warns that commands will fail", func(t *testing.T) {
			isolateConfigFile(t, "")
			value := globexURL
			if variable == "BIDIREKT_API_KEY" {
				value = "bdk_globex"
			}
			t.Setenv(variable, value)

			execution := executeLogin(t, signInAcme(acmeURL+"\n", "--profile", "acme"))

			require.NoError(t, execution.err)
			assert.True(t, strings.HasSuffix(execution.stderr, "only one of BIDIREKT_BROKER_URL and BIDIREKT_API_KEY is set — commands will fail until both or neither are\n"))
		})
	}

	t.Run("a key over plain http to a remote host warns and still signs in", func(t *testing.T) {
		isolateConfigFile(t, "")
		run := signInAcme("http://broker.acme.internal\n", "--profile", "acme")
		run.endpoint = "http://broker.acme.internal/api/environments"

		execution := executeLogin(t, run)

		require.NoError(t, execution.err)
		assert.Contains(t, execution.stderr, "\nwarning: the API key travels unencrypted to http://broker.acme.internal\n")
	})

	failures := []struct {
		name   string
		apiKey string
		mock   httpmock.Responder
		err    string
		calls  int
	}{
		{
			name:   "a key without bdk_ ends the login before any request",
			apiKey: "acme_key",
			mock:   httpmock.NewStringResponder(http.StatusOK, environments),
			err:    "an API key starts with bdk_",
		},
		{
			name:   "a key the broker refuses",
			apiKey: acmeKey,
			mock:   httpmock.NewStringResponder(http.StatusUnauthorized, `{"message":"unauthorized"}`),
			err:    "the broker refused this API key",
			calls:  1,
		},
		{
			name:   "a broker answering another status",
			apiKey: acmeKey,
			mock:   httpmock.NewStringResponder(http.StatusForbidden, `{"message":"forbidden"}`),
			err:    "cannot reach the broker at " + acmeURL + ": HTTP 403",
			calls:  1,
		},
		{
			name:   "a URL that is not a broker",
			apiKey: acmeKey,
			mock:   httpmock.NewStringResponder(http.StatusOK, `<html>acme</html>`),
			err:    "cannot reach the broker at " + acmeURL + ": not a Bidirekt broker",
			calls:  1,
		},
		{
			name:   "a broker that is down",
			apiKey: acmeKey,
			mock:   httpmock.NewErrorResponder(errors.New("connection refused")),
			err:    "cannot reach the broker at " + acmeURL + ": Get \"" + acmeURL + "/api/environments\": connection refused",
			calls:  1,
		},
	}
	for _, failure := range failures {
		t.Run(failure.name+" fails and leaves the config file untouched", func(t *testing.T) {
			content := `{"profiles":{"acme":{"brokerUrl":"` + acmeURL + `","apiKey":"` + previousKey + `"}}}`
			configFilePath := isolateConfigFile(t, content)
			run := signInAcme("\n", "--profile", "acme")
			run.apiKey = failure.apiKey
			run.responder = failure.mock

			execution := executeLogin(t, run)

			require.EqualError(t, execution.err, failure.err)
			assert.Equal(t, failure.calls, execution.calls)
			assert.Empty(t, execution.stdout)
			assert.Equal(t, content, readConfigFile(t, configFilePath))
		})
	}

	t.Run("the key never appears on stdout or stderr", func(t *testing.T) {
		isolateConfigFile(t, `{"profiles":{"acme":{"brokerUrl":"`+acmeURL+`","apiKey":"`+previousKey+`"}}}`)
		t.Setenv("BIDIREKT_API_KEY", "bdk_globex")
		run := signInAcme("http://broker.acme.internal\n", "--profile", "acme")
		run.endpoint = "http://broker.acme.internal/api/environments"

		execution := executeLogin(t, run)

		require.NoError(t, execution.err)
		for _, output := range []string{execution.stdout, execution.stderr} {
			assert.NotContains(t, output, acmeKey)
			assert.NotContains(t, output, previousKey)
			assert.NotContains(t, output, "bdk_")
		}
	})

	t.Run("without a terminal it fails before asking and saves nothing", func(t *testing.T) {
		configFilePath := isolateConfigFile(t, "")
		run := signInAcme("acme\n" + acmeURL + "\n")
		run.stdinIsTerminal = false

		execution := executeLogin(t, run)

		require.EqualError(t, execution.err, noTerminalMsg)
		assert.NotContains(t, execution.stderr, "Profile")
		assert.Zero(t, execution.calls)
		assert.NoFileExists(t, configFilePath)
	})

	t.Run("with stdin at /dev/null it fails before asking", func(t *testing.T) {
		configFilePath := isolateConfigFile(t, "")
		nullDevice, err := os.Open(os.DevNull)
		require.NoError(t, err)
		t.Cleanup(func() { assert.NoError(t, nullDevice.Close()) })
		command := login.NewLoginCommand(components.NewConfig(), login.NewLoginClient(components.NewHTTPClient()), components.IsTerminal, components.ReadSecret)
		var out, errOut bytes.Buffer
		command.SetIn(nullDevice)
		command.SetOut(&out)
		command.SetErr(&errOut)
		command.SetArgs([]string{})

		err = command.Execute()

		require.EqualError(t, err, noTerminalMsg)
		assert.NotContains(t, errOut.String(), "Profile")
		assert.NoFileExists(t, configFilePath)
	})

	t.Run("stdin ending without an answer saves nothing", func(t *testing.T) {
		for _, test := range []struct {
			args []string
			err  string
		}{
			{err: "no profile entered"},
			{args: []string{"--profile", "acme"}, err: "no broker URL entered"},
		} {
			configFilePath := isolateConfigFile(t, "")

			execution := executeLogin(t, signInAcme("", test.args...))

			require.EqualError(t, execution.err, test.err)
			assert.Zero(t, execution.calls)
			assert.NoFileExists(t, configFilePath)
		}
	})

	t.Run("an invalid config file fails citing its path", func(t *testing.T) {
		configFilePath := isolateConfigFile(t, `{"profiles":`)

		execution := executeLogin(t, signInAcme(acmeURL+"\n", "--profile", "acme"))

		assert.ErrorContains(t, execution.err, "invalid config file "+configFilePath+":")
		assert.Zero(t, execution.calls)
	})

	t.Run("creates the file with 0600 inside a folder with 0700", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("Windows has no Unix permission bits")
		}
		configFilePath := isolateConfigFile(t, "")

		execution := executeLogin(t, signInAcme(acmeURL+"\n", "--profile", "acme"))

		require.NoError(t, execution.err)
		folderInfo, err := os.Stat(filepath.Dir(configFilePath))
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0o700), folderInfo.Mode().Perm())
		fileInfo, err := os.Stat(configFilePath)
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0o600), fileInfo.Mode().Perm())
	})
}
