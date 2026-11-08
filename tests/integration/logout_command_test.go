package integration_test

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/bidirekt/cli/internal/components"
	"github.com/bidirekt/cli/internal/features/logout"
	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogoutCommand(t *testing.T) {
	const (
		acmeURL    = "https://broker.acme.internal"
		globexURL  = "http://localhost:8080/"
		acmeKey    = "bdk_acme_secret"
		globexKey  = "bdk_globex_secret"
		bothSigned = `{"profiles":{` +
			`"default":{"brokerUrl":"` + acmeURL + `","apiKey":"` + acmeKey + `"},` +
			`"globex":{"brokerUrl":"` + globexURL + `","apiKey":"` + globexKey + `"}}}`
		envVarsStillSet = "BIDIREKT_BROKER_URL and BIDIREKT_API_KEY are still set and keep signing commands in\n"
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

	executeLogout := func(t *testing.T, args ...string) (string, string, error) {
		t.Helper()

		httpmock.Activate(t)
		command := logout.NewLogoutCommand(components.NewConfig())
		var out, errOut bytes.Buffer
		command.SetOut(&out)
		command.SetErr(&errOut)
		command.SetArgs(args)

		err := command.Execute()

		assert.Zero(t, httpmock.GetTotalCallCount())
		return out.String(), errOut.String(), err
	}

	readConfigFile := func(t *testing.T, configFilePath string) string {
		t.Helper()

		content, err := os.ReadFile(configFilePath)
		require.NoError(t, err)

		return string(content)
	}

	t.Run("removes the key of the default profile, keeps its URL and the other profiles", func(t *testing.T) {
		configFilePath := isolateConfigFile(t, bothSigned)

		out, errOut, err := executeLogout(t)

		require.NoError(t, err)
		assert.Equal(t, "Signed out of "+acmeURL+" (profile default). The key is still valid — revoke it at "+acmeURL+"/profile\n", out)
		assert.Empty(t, errOut)
		assert.JSONEq(t, `{"profiles":{`+
			`"default":{"brokerUrl":"`+acmeURL+`"},`+
			`"globex":{"brokerUrl":"`+globexURL+`","apiKey":"`+globexKey+`"}}}`, readConfigFile(t, configFilePath))
	})

	t.Run("--profile selects the profile and the revoke URL drops the trailing slash", func(t *testing.T) {
		configFilePath := isolateConfigFile(t, bothSigned)

		out, _, err := executeLogout(t, "--profile", "globex")

		require.NoError(t, err)
		assert.Equal(t, "Signed out of "+globexURL+" (profile globex). The key is still valid — revoke it at http://localhost:8080/profile\n", out)
		assert.JSONEq(t, `{"profiles":{`+
			`"default":{"brokerUrl":"`+acmeURL+`","apiKey":"`+acmeKey+`"},`+
			`"globex":{"brokerUrl":"`+globexURL+`"}}}`, readConfigFile(t, configFilePath))
	})

	t.Run("BIDIREKT_PROFILE selects the profile, and --profile wins over it", func(t *testing.T) {
		configFilePath := isolateConfigFile(t, bothSigned)
		t.Setenv("BIDIREKT_PROFILE", "globex")

		out, _, err := executeLogout(t)
		require.NoError(t, err)
		assert.Contains(t, out, "(profile globex)")

		out, _, err = executeLogout(t, "--profile", "default")
		require.NoError(t, err)
		assert.Contains(t, out, "(profile default)")

		assert.JSONEq(t, `{"profiles":{"default":{"brokerUrl":"`+acmeURL+`"},"globex":{"brokerUrl":"`+globexURL+`"}}}`, readConfigFile(t, configFilePath))
	})

	t.Run("prints the broker URL redacted", func(t *testing.T) {
		isolateConfigFile(t, `{"profiles":{"default":{"brokerUrl":"https://bob:hunter2@broker.acme.internal/","apiKey":"`+acmeKey+`"}}}`)

		out, _, err := executeLogout(t)

		require.NoError(t, err)
		assert.Equal(t, "Signed out of https://bob:xxxxx@broker.acme.internal/ (profile default). The key is still valid — revoke it at https://bob:xxxxx@broker.acme.internal/profile\n", out)
		assert.NotContains(t, out, "hunter2")
		assert.NotContains(t, out, acmeKey)
	})

	t.Run("with both environment variables set it signs the profile out and warns they still sign in", func(t *testing.T) {
		configFilePath := isolateConfigFile(t, bothSigned)
		t.Setenv("BIDIREKT_BROKER_URL", acmeURL)
		t.Setenv("BIDIREKT_API_KEY", acmeKey)

		out, errOut, err := executeLogout(t)

		require.NoError(t, err)
		assert.Contains(t, out, "Signed out of "+acmeURL+" (profile default)")
		assert.Equal(t, envVarsStillSet, errOut)
		assert.NotContains(t, readConfigFile(t, configFilePath), acmeKey)
	})

	t.Run("with only one environment variable set it doesn't warn", func(t *testing.T) {
		isolateConfigFile(t, bothSigned)
		t.Setenv("BIDIREKT_API_KEY", acmeKey)

		_, errOut, err := executeLogout(t)

		require.NoError(t, err)
		assert.Empty(t, errOut)
	})

	t.Run("a profile without a key says so, exits 0 and leaves the file untouched", func(t *testing.T) {
		content := `{"profiles":{"default":{"brokerUrl":"` + acmeURL + `"}}}`
		configFilePath := isolateConfigFile(t, content)

		out, errOut, err := executeLogout(t)

		require.NoError(t, err)
		assert.Equal(t, "not signed in on profile default\n", out)
		assert.Empty(t, errOut)
		assert.Equal(t, content, readConfigFile(t, configFilePath))
	})

	t.Run("a missing profile is not signed in and no file is created", func(t *testing.T) {
		configFilePath := isolateConfigFile(t, "")

		out, _, err := executeLogout(t, "--profile", "globex")

		require.NoError(t, err)
		assert.Equal(t, "not signed in on profile globex\n", out)
		assert.NoFileExists(t, configFilePath)
	})

	t.Run("an invalid config file fails citing its path", func(t *testing.T) {
		configFilePath := isolateConfigFile(t, `{"profiles":`)

		_, _, err := executeLogout(t)

		assert.ErrorContains(t, err, "invalid config file "+configFilePath+":")
	})

	t.Run("keeps the file at 0600", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("Windows has no Unix permission bits")
		}
		configFilePath := isolateConfigFile(t, bothSigned)

		_, _, err := executeLogout(t)

		require.NoError(t, err)
		fileInfo, err := os.Stat(configFilePath)
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0o600), fileInfo.Mode().Perm())
	})
}
