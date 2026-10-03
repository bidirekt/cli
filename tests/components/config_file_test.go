package components_test

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/bidirekt/cli/internal/components"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigFileFromEnvironment(t *testing.T) {
	home := t.TempDir()
	userConfigFile := filepath.Join(home, ".config", "bidirekt", "config.json")
	if runtime.GOOS == "windows" {
		userConfigFile = filepath.Join(home, "bidirekt", "config.json")
	}

	tests := []struct {
		name          string
		configFile    string
		xdgConfigHome string
		expected      string
	}{
		{
			name:          "BIDIREKT_CONFIG_FILE wins over XDG_CONFIG_HOME",
			configFile:    filepath.Join(home, "custom.json"),
			xdgConfigHome: filepath.Join(home, "xdg"),
			expected:      filepath.Join(home, "custom.json"),
		},
		{
			name:          "XDG_CONFIG_HOME holds a bidirekt folder",
			xdgConfigHome: filepath.Join(home, "xdg"),
			expected:      filepath.Join(home, "xdg", "bidirekt", "config.json"),
		},
		{
			name:     "without either, the user config folder",
			expected: userConfigFile,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv("BIDIREKT_CONFIG_FILE", test.configFile)
			t.Setenv("XDG_CONFIG_HOME", test.xdgConfigHome)
			t.Setenv("HOME", home)
			t.Setenv("AppData", home)

			configFile, err := components.ConfigFileFromEnvironment()

			require.NoError(t, err)
			assert.Equal(t, test.expected, configFile.Path)
		})
	}
}

func TestConfigFileFromEnvironmentFailsWithoutAUserFolder(t *testing.T) {
	t.Setenv("BIDIREKT_CONFIG_FILE", "")
	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", "")
	t.Setenv("AppData", "")

	_, err := components.ConfigFileFromEnvironment()

	assert.ErrorContains(t, err, "locate config file")
}

func TestConfigFileReadProfiles(t *testing.T) {
	t.Run("a missing file holds no profiles", func(t *testing.T) {
		profiles, err := components.ConfigFile{Path: filepath.Join(t.TempDir(), "config.json")}.ReadProfiles()

		require.NoError(t, err)
		assert.Empty(t, profiles)
	})

	t.Run("a file without profiles holds no profiles", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "config.json")
		require.NoError(t, os.WriteFile(path, []byte(`{}`), 0o600))

		profiles, err := components.ConfigFile{Path: path}.ReadProfiles()

		require.NoError(t, err)
		assert.Empty(t, profiles)
	})

	t.Run("reads every profile", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "config.json")
		content := `{"profiles":{"default":{"brokerUrl":"https://broker.acme.internal"},"staging":{"brokerUrl":"http://localhost:8080"}}}`
		require.NoError(t, os.WriteFile(path, []byte(content), 0o600))

		profiles, err := components.ConfigFile{Path: path}.ReadProfiles()

		require.NoError(t, err)
		assert.Equal(t, map[string]components.Profile{
			"default": {BrokerURL: "https://broker.acme.internal"},
			"staging": {BrokerURL: "http://localhost:8080"},
		}, profiles)
	})

	for _, test := range []struct {
		name    string
		content string
	}{
		{name: "invalid JSON fails citing the path", content: `{"profiles":`},
		{name: "JSON of another shape fails citing the path", content: `{"profiles":["default"]}`},
	} {
		t.Run(test.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.json")
			require.NoError(t, os.WriteFile(path, []byte(test.content), 0o600))

			_, err := components.ConfigFile{Path: path}.ReadProfiles()

			assert.ErrorContains(t, err, "invalid config file "+path+":")
		})
	}
}

func TestConfigFileWriteProfile(t *testing.T) {
	t.Run("creates the folder with 0700 and the file with 0600", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("Windows has no Unix permission bits")
		}
		folder := filepath.Join(t.TempDir(), "config", "bidirekt")
		path := filepath.Join(folder, "config.json")

		err := components.ConfigFile{Path: path}.WriteProfile("default", components.Profile{BrokerURL: "https://broker.acme.internal"})

		require.NoError(t, err)
		folderInfo, err := os.Stat(folder)
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0o700), folderInfo.Mode().Perm())
		fileInfo, err := os.Stat(path)
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0o600), fileInfo.Mode().Perm())
	})

	t.Run("writes the profiles format", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "config.json")

		err := components.ConfigFile{Path: path}.WriteProfile("default", components.Profile{BrokerURL: "https://broker.acme.internal"})

		require.NoError(t, err)
		written, err := os.ReadFile(path)
		require.NoError(t, err)
		assert.JSONEq(t, `{"profiles":{"default":{"brokerUrl":"https://broker.acme.internal"}}}`, string(written))
	})

	t.Run("keeps the other profiles and replaces the written one", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "config.json")
		content := `{"profiles":{"default":{"brokerUrl":"https://broker.acme.internal"},"staging":{"brokerUrl":"http://old.internal"}}}`
		require.NoError(t, os.WriteFile(path, []byte(content), 0o600))

		err := components.ConfigFile{Path: path}.WriteProfile("staging", components.Profile{BrokerURL: "http://localhost:8080"})

		require.NoError(t, err)
		written, err := os.ReadFile(path)
		require.NoError(t, err)
		assert.JSONEq(t, `{"profiles":{"default":{"brokerUrl":"https://broker.acme.internal"},"staging":{"brokerUrl":"http://localhost:8080"}}}`, string(written))
	})

	t.Run("leaves a file it cannot read untouched", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "config.json")
		require.NoError(t, os.WriteFile(path, []byte(`{"profiles":`), 0o600))

		err := components.ConfigFile{Path: path}.WriteProfile("default", components.Profile{BrokerURL: "https://broker.acme.internal"})

		assert.ErrorContains(t, err, "invalid config file "+path+":")
		written, readErr := os.ReadFile(path)
		require.NoError(t, readErr)
		assert.Equal(t, `{"profiles":`, string(written))
	})
}
