package integration_test

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/bidirekt/cli/internal/components"
	"github.com/bidirekt/cli/internal/features/configure"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigureCommand(t *testing.T) {
	const (
		acmeURL    = "https://broker.acme.internal"
		stagingURL = "http://localhost:8080"
	)

	isolateConfigFile := func(t *testing.T, content string) string {
		t.Helper()

		configFilePath := filepath.Join(t.TempDir(), "bidirekt", "config.json")
		t.Setenv("BIDIREKT_CONFIG_FILE", configFilePath)
		t.Setenv("BIDIREKT_BROKER_URL", "")
		t.Setenv("BIDIREKT_PROFILE", "")
		if content != "" {
			require.NoError(t, os.MkdirAll(filepath.Dir(configFilePath), 0o700))
			require.NoError(t, os.WriteFile(configFilePath, []byte(content), 0o600))
		}

		return configFilePath
	}

	executeConfigure := func(stdinIsTerminal bool, stdin string, args ...string) (string, string, error) {
		command := configure.NewConfigureCommand(components.NewConfig(), func(any) bool { return stdinIsTerminal })
		var out, errOut bytes.Buffer
		command.SetIn(strings.NewReader(stdin))
		command.SetOut(&out)
		command.SetErr(&errOut)
		command.SetArgs(args)

		err := command.Execute()

		return out.String(), errOut.String(), err
	}

	readConfigFile := func(t *testing.T, configFilePath string) string {
		t.Helper()

		content, err := os.ReadFile(configFilePath)
		require.NoError(t, err)

		return string(content)
	}

	t.Run("--broker-url saves into the default profile without asking, outside a terminal", func(t *testing.T) {
		configFilePath := isolateConfigFile(t, "")

		out, errOut, err := executeConfigure(false, "", "--broker-url", acmeURL)

		require.NoError(t, err)
		assert.Empty(t, out)
		assert.Empty(t, errOut)
		assert.JSONEq(t, `{"profiles":{"default":{"brokerUrl":"`+acmeURL+`"}}}`, readConfigFile(t, configFilePath))
	})

	t.Run("--profile staging saves into staging and keeps the other profiles", func(t *testing.T) {
		configFilePath := isolateConfigFile(t, `{"profiles":{"default":{"brokerUrl":"`+acmeURL+`"}}}`)

		_, _, err := executeConfigure(false, "", "--profile", "staging", "--broker-url", stagingURL)

		require.NoError(t, err)
		assert.JSONEq(t, `{"profiles":{"default":{"brokerUrl":"`+acmeURL+`"},"staging":{"brokerUrl":"`+stagingURL+`"}}}`, readConfigFile(t, configFilePath))
	})

	t.Run("BIDIREKT_PROFILE selects the profile, and --profile wins over it", func(t *testing.T) {
		configFilePath := isolateConfigFile(t, "")
		t.Setenv("BIDIREKT_PROFILE", "staging")

		_, _, err := executeConfigure(false, "", "--broker-url", stagingURL)
		require.NoError(t, err)
		_, _, err = executeConfigure(false, "", "--profile", "local", "--broker-url", acmeURL)
		require.NoError(t, err)

		assert.JSONEq(t, `{"profiles":{"staging":{"brokerUrl":"`+stagingURL+`"},"local":{"brokerUrl":"`+acmeURL+`"}}}`, readConfigFile(t, configFilePath))
	})

	t.Run("an invalid --broker-url is rejected naming the flag and nothing is written", func(t *testing.T) {
		configFilePath := isolateConfigFile(t, "")

		_, _, err := executeConfigure(true, stagingURL+"\n", "--broker-url", "broker.acme.com")

		require.EqualError(t, err, `invalid broker URL "broker.acme.com" (from --broker-url) — use http:// or https:// followed by a host`)
		assert.NoFileExists(t, configFilePath)
	})

	t.Run("without --broker-url outside a terminal it fails asking for the flag", func(t *testing.T) {
		configFilePath := isolateConfigFile(t, "")

		_, errOut, err := executeConfigure(false, stagingURL+"\n")

		require.EqualError(t, err, "no terminal to ask for the broker URL — pass --broker-url")
		assert.NotContains(t, errOut, "Broker URL")
		assert.NoFileExists(t, configFilePath)
	})

	t.Run("BIDIREKT_BROKER_URL is ignored", func(t *testing.T) {
		configFilePath := isolateConfigFile(t, "")
		t.Setenv("BIDIREKT_BROKER_URL", acmeURL)

		_, _, err := executeConfigure(false, "")
		require.EqualError(t, err, "no terminal to ask for the broker URL — pass --broker-url")

		_, errOut, err := executeConfigure(true, stagingURL+"\n")
		require.NoError(t, err)
		assert.Equal(t, "Broker URL: ", errOut)
		assert.JSONEq(t, `{"profiles":{"default":{"brokerUrl":"`+stagingURL+`"}}}`, readConfigFile(t, configFilePath))
	})

	t.Run("in a terminal it asks and saves into the active profile", func(t *testing.T) {
		configFilePath := isolateConfigFile(t, `{"profiles":{"default":{"brokerUrl":"`+acmeURL+`"}}}`)

		out, errOut, err := executeConfigure(true, stagingURL+"\n", "--profile", "staging")

		require.NoError(t, err)
		assert.Empty(t, out)
		assert.Equal(t, "Broker URL: ", errOut)
		assert.JSONEq(t, `{"profiles":{"default":{"brokerUrl":"`+acmeURL+`"},"staging":{"brokerUrl":"`+stagingURL+`"}}}`, readConfigFile(t, configFilePath))
	})

	t.Run("in a terminal it shows the current value and Enter keeps it", func(t *testing.T) {
		configFilePath := isolateConfigFile(t, `{"profiles":{"default":{"brokerUrl":"`+acmeURL+`"}}}`)

		_, errOut, err := executeConfigure(true, "\n")

		require.NoError(t, err)
		assert.Equal(t, "Broker URL ["+acmeURL+"]: ", errOut)
		assert.JSONEq(t, `{"profiles":{"default":{"brokerUrl":"`+acmeURL+`"}}}`, readConfigFile(t, configFilePath))
	})

	t.Run("in a terminal a new value overwrites the current one", func(t *testing.T) {
		configFilePath := isolateConfigFile(t, `{"profiles":{"default":{"brokerUrl":"`+acmeURL+`"}}}`)

		_, errOut, err := executeConfigure(true, stagingURL+"\n")

		require.NoError(t, err)
		assert.Equal(t, "Broker URL ["+acmeURL+"]: ", errOut)
		assert.JSONEq(t, `{"profiles":{"default":{"brokerUrl":"`+stagingURL+`"}}}`, readConfigFile(t, configFilePath))
	})

	t.Run("in a terminal stdin ending without an answer saves nothing", func(t *testing.T) {
		configFilePath := isolateConfigFile(t, "")

		_, _, err := executeConfigure(true, "")

		require.EqualError(t, err, "no broker URL entered")
		assert.NoFileExists(t, configFilePath)
	})

	t.Run("an invalid config file fails citing its path", func(t *testing.T) {
		configFilePath := isolateConfigFile(t, `{"profiles":`)

		_, _, err := executeConfigure(false, "", "--broker-url", acmeURL)

		assert.ErrorContains(t, err, "invalid config file "+configFilePath+":")
	})

	t.Run("creates the file with 0600 inside a folder with 0700", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("Windows has no Unix permission bits")
		}
		configFilePath := isolateConfigFile(t, "")

		_, _, err := executeConfigure(false, "", "--broker-url", acmeURL)

		require.NoError(t, err)
		folderInfo, err := os.Stat(filepath.Dir(configFilePath))
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0o700), folderInfo.Mode().Perm())
		fileInfo, err := os.Stat(configFilePath)
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0o600), fileInfo.Mode().Perm())
	})
}
