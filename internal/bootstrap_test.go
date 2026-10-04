package internal

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/bidirekt/cli/internal/components"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const noBrokerConfigured = `no broker configured — pass --broker-url, set BIDIREKT_BROKER_URL, or run "bidirekt configure"`

type brokerStub struct {
	url      string
	requests *atomic.Int32
}

func startBrokerStub(t *testing.T) brokerStub {
	t.Helper()

	requests := &atomic.Int32{}
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		requests.Add(1)
		_, _ = writer.Write([]byte(`{"message":"participant created"}`))
	}))
	t.Cleanup(server.Close)

	return brokerStub{url: server.URL, requests: requests}
}

func isolateEnvironment(t *testing.T) string {
	t.Helper()

	configFilePath := filepath.Join(t.TempDir(), "bidirekt", "config.json")
	t.Setenv("BIDIREKT_CONFIG_FILE", configFilePath)
	t.Setenv("BIDIREKT_BROKER_URL", "")
	t.Setenv("BIDIREKT_PROFILE", "")

	return configFilePath
}

func writeConfigFile(t *testing.T, configFilePath, content string) {
	t.Helper()

	require.NoError(t, os.MkdirAll(filepath.Dir(configFilePath), 0o700))
	require.NoError(t, os.WriteFile(configFilePath, []byte(content), 0o600))
}

type rootExecution struct {
	stdout string
	stderr string
	err    error
}

func executeRoot(t *testing.T, dependencies *components.Components, stdinIsTerminal bool, stdin string, args ...string) rootExecution {
	t.Helper()

	rootCommand := newRootCommand(dependencies, func(any) bool { return stdinIsTerminal })
	var stdout, stderr bytes.Buffer
	rootCommand.SetIn(strings.NewReader(stdin))
	rootCommand.SetOut(&stdout)
	rootCommand.SetErr(&stderr)
	rootCommand.SetArgs(args)

	err := rootCommand.Execute()

	return rootExecution{stdout: stdout.String(), stderr: stderr.String(), err: err}
}

func TestVersionIsPrintedByTheFlagAndTheCommand(t *testing.T) {
	for _, arg := range []string{"--version", "version"} {
		t.Run(arg, func(t *testing.T) {
			isolateEnvironment(t)

			execution := executeRoot(t, components.New(), false, "", arg)

			require.NoError(t, execution.err)
			assert.Equal(t, "bidirekt version dev\n", execution.stdout)
			assert.Empty(t, execution.stderr)
		})
	}
}

func TestCommandsThatDoNotTalkToTheBrokerNeverResolveIt(t *testing.T) {
	for _, args := range [][]string{
		{"version"},
		{"--version"},
		{"--help"},
		{"help"},
		{"help", "create-participant"},
		{"completion", "bash"},
		{"create-participant", "--help"},
		{"configure", "--broker-url", "http://localhost:8080"},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			isolateEnvironment(t)

			execution := executeRoot(t, components.New(), true, "http://localhost:8080\n", args...)

			require.NoError(t, execution.err)
			assert.Empty(t, execution.stderr)
		})
	}
}

func TestBrokerCommandOutsideATerminalWithoutABrokerFailsWithoutCallingIt(t *testing.T) {
	configFilePath := isolateEnvironment(t)
	broker := startBrokerStub(t)
	dependencies := components.New()
	dependencies.HTTPClient.SetBaseURL(broker.url)

	execution := executeRoot(t, dependencies, false, "", "create-participant", "pets")

	require.EqualError(t, execution.err, noBrokerConfigured)
	assert.Empty(t, execution.stdout)
	assert.Empty(t, execution.stderr)
	assert.Zero(t, broker.requests.Load())
	assert.NoFileExists(t, configFilePath)
}

func TestUsageErrorsComeBeforeAnyBrokerResolution(t *testing.T) {
	tests := []struct {
		args []string
		err  string
	}{
		{args: []string{"record-deployment", "pets", "--environment", "production"}, err: `required flag(s) "version" not set`},
		{args: []string{"create-participant"}, err: "accepts 1 arg(s), received 0"},
		{args: []string{"publish", "--participant", "pets", "--version", "v1"}, err: "requires at least 1 arg(s), only received 0"},
		{args: []string{"can-i-deploy", "pets", "--unknown"}, err: "unknown flag: --unknown"},
	}

	for _, test := range tests {
		t.Run(strings.Join(test.args, " "), func(t *testing.T) {
			configFilePath := isolateEnvironment(t)
			broker := startBrokerStub(t)

			execution := executeRoot(t, components.New(), true, broker.url+"\n", test.args...)

			require.EqualError(t, execution.err, test.err)
			assert.Empty(t, execution.stderr)
			assert.Zero(t, broker.requests.Load())
			assert.NoFileExists(t, configFilePath)
		})
	}
}

func TestBrokerURLComesFromTheFirstSourceThatHasOneAndIsAnnouncedOnStderr(t *testing.T) {
	flagBroker := startBrokerStub(t)
	envBroker := startBrokerStub(t)
	defaultBroker := startBrokerStub(t)
	stagingBroker := startBrokerStub(t)
	configFileContent := `{"profiles":{"default":{"brokerUrl":"` + defaultBroker.url + `"},"staging":{"brokerUrl":"` + stagingBroker.url + `"}}}`

	tests := []struct {
		name       string
		envURL     string
		envProfile string
		args       []string
		broker     brokerStub
		brokerLine string
	}{
		{
			name:       "--broker-url wins over BIDIREKT_BROKER_URL and the profile",
			envURL:     envBroker.url,
			args:       []string{"--broker-url", flagBroker.url},
			broker:     flagBroker,
			brokerLine: "Broker: " + flagBroker.url + " (from --broker-url)\n",
		},
		{
			name:       "BIDIREKT_BROKER_URL wins over the profile",
			envURL:     envBroker.url,
			broker:     envBroker,
			brokerLine: "Broker: " + envBroker.url + " (from BIDIREKT_BROKER_URL)\n",
		},
		{
			name:       "the default profile is used without --profile and BIDIREKT_PROFILE",
			broker:     defaultBroker,
			brokerLine: "Broker: " + defaultBroker.url + " (profile: default)\n",
		},
		{
			name:       "--profile selects the profile",
			args:       []string{"--profile", "staging"},
			broker:     stagingBroker,
			brokerLine: "Broker: " + stagingBroker.url + " (profile: staging)\n",
		},
		{
			name:       "BIDIREKT_PROFILE selects the profile",
			envProfile: "staging",
			broker:     stagingBroker,
			brokerLine: "Broker: " + stagingBroker.url + " (profile: staging)\n",
		},
		{
			name:       "--profile wins over BIDIREKT_PROFILE",
			envProfile: "default",
			args:       []string{"--profile", "staging"},
			broker:     stagingBroker,
			brokerLine: "Broker: " + stagingBroker.url + " (profile: staging)\n",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			configFilePath := isolateEnvironment(t)
			writeConfigFile(t, configFilePath, configFileContent)
			t.Setenv("BIDIREKT_BROKER_URL", test.envURL)
			t.Setenv("BIDIREKT_PROFILE", test.envProfile)
			requestsBefore := test.broker.requests.Load()

			execution := executeRoot(t, components.New(), false, "", append([]string{"create-participant", "pets"}, test.args...)...)

			require.NoError(t, execution.err)
			assert.Equal(t, "pets participant created\n", execution.stdout)
			assert.Equal(t, test.brokerLine, execution.stderr)
			assert.Equal(t, requestsBefore+1, test.broker.requests.Load())
		})
	}
}

func TestConfigFileIsReadOnlyWhenNoFlagOrEnvironmentHasAURL(t *testing.T) {
	configFilePath := isolateEnvironment(t)
	writeConfigFile(t, configFilePath, `{"profiles":`)
	broker := startBrokerStub(t)

	fromFlag := executeRoot(t, components.New(), false, "", "create-participant", "pets", "--broker-url", broker.url)
	require.NoError(t, fromFlag.err)

	t.Setenv("BIDIREKT_BROKER_URL", broker.url)
	fromEnv := executeRoot(t, components.New(), false, "", "create-participant", "pets")
	require.NoError(t, fromEnv.err)

	t.Setenv("BIDIREKT_BROKER_URL", "")
	fromProfile := executeRoot(t, components.New(), false, "", "create-participant", "pets")
	assert.ErrorContains(t, fromProfile.err, "invalid config file "+configFilePath+":")
	assert.Equal(t, int32(2), broker.requests.Load())
}

func TestBrokerLineHidesThePassword(t *testing.T) {
	isolateEnvironment(t)
	broker := startBrokerStub(t)
	brokerURL := strings.Replace(broker.url, "http://", "http://ci:segredo@", 1)

	execution := executeRoot(t, components.New(), false, "", "create-participant", "pets", "--broker-url", brokerURL)

	require.NoError(t, execution.err)
	assert.Equal(t, "Broker: "+strings.Replace(broker.url, "http://", "http://ci:xxxxx@", 1)+" (from --broker-url)\n", execution.stderr)
	assert.NotContains(t, execution.stderr, "segredo")
	assert.Equal(t, int32(1), broker.requests.Load())
}

func TestInvalidBrokerURLFailsNamingItsOrigin(t *testing.T) {
	tests := []struct {
		name              string
		envURL            string
		configFileContent string
		args              []string
		err               string
	}{
		{
			name: "from --broker-url",
			args: []string{"--broker-url", "broker.acme.com"},
			err:  `invalid broker URL "broker.acme.com" (from --broker-url) — use http:// or https:// followed by a host`,
		},
		{
			name:   "from BIDIREKT_BROKER_URL",
			envURL: "ftp://x",
			err:    `invalid broker URL "ftp://x" (from BIDIREKT_BROKER_URL) — use http:// or https:// followed by a host`,
		},
		{
			name:              "from the profile",
			configFileContent: `{"profiles":{"staging":{"brokerUrl":"http://"}}}`,
			args:              []string{"--profile", "staging"},
			err:               `invalid broker URL "http://" (profile: staging) — use http:// or https:// followed by a host`,
		},
		{
			name:   "with the password hidden",
			envURL: "https://ci:segredo@",
			err:    `invalid broker URL "https://ci:xxxxx@" (from BIDIREKT_BROKER_URL) — use http:// or https:// followed by a host`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			configFilePath := isolateEnvironment(t)
			if test.configFileContent != "" {
				writeConfigFile(t, configFilePath, test.configFileContent)
			}
			t.Setenv("BIDIREKT_BROKER_URL", test.envURL)
			broker := startBrokerStub(t)
			dependencies := components.New()
			dependencies.HTTPClient.SetBaseURL(broker.url)

			execution := executeRoot(t, dependencies, true, broker.url+"\n", append([]string{"create-participant", "pets"}, test.args...)...)

			require.EqualError(t, execution.err, test.err)
			assert.Empty(t, execution.stderr)
			assert.Zero(t, broker.requests.Load())
		})
	}
}

func TestProfileNamedOutsideATerminalMustExist(t *testing.T) {
	tests := []struct {
		name       string
		envProfile string
		args       []string
	}{
		{name: "named by --profile", args: []string{"--profile", "staging"}},
		{name: "named by BIDIREKT_PROFILE", envProfile: "staging"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			configFilePath := isolateEnvironment(t)
			writeConfigFile(t, configFilePath, `{"profiles":{"default":{"brokerUrl":"http://localhost:8080"}}}`)
			t.Setenv("BIDIREKT_PROFILE", test.envProfile)
			broker := startBrokerStub(t)
			dependencies := components.New()
			dependencies.HTTPClient.SetBaseURL(broker.url)

			execution := executeRoot(t, dependencies, false, "", append([]string{"create-participant", "pets"}, test.args...)...)

			require.EqualError(t, execution.err, `profile "staging" not found in `+configFilePath)
			assert.Empty(t, execution.stderr)
			assert.Zero(t, broker.requests.Load())
		})
	}
}

func TestMissingBrokerURLIsAskedInATerminalAndSavedToTheActiveProfile(t *testing.T) {
	tests := []struct {
		name       string
		envProfile string
		args       []string
		profile    string
	}{
		{name: "the default profile", profile: "default"},
		{name: "a profile named by --profile is created", args: []string{"--profile", "staging"}, profile: "staging"},
		{name: "a profile named by BIDIREKT_PROFILE is created", envProfile: "staging", profile: "staging"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			configFilePath := isolateEnvironment(t)
			writeConfigFile(t, configFilePath, `{"profiles":{"other":{"brokerUrl":"http://other.internal"}}}`)
			t.Setenv("BIDIREKT_PROFILE", test.envProfile)
			broker := startBrokerStub(t)
			args := append([]string{"create-participant", "pets"}, test.args...)

			asked := executeRoot(t, components.New(), true, broker.url+"\n", args...)

			require.NoError(t, asked.err)
			assert.Equal(t, "Broker URL: Broker: "+broker.url+" (profile: "+test.profile+")\n", asked.stderr)
			assert.Equal(t, "pets participant created\n", asked.stdout)
			assert.Equal(t, int32(1), broker.requests.Load())
			saved, err := os.ReadFile(configFilePath)
			require.NoError(t, err)
			assert.JSONEq(t, `{"profiles":{"other":{"brokerUrl":"http://other.internal"},"`+test.profile+`":{"brokerUrl":"`+broker.url+`"}}}`, string(saved))

			reused := executeRoot(t, components.New(), true, "", args...)

			require.NoError(t, reused.err)
			assert.Equal(t, "Broker: "+broker.url+" (profile: "+test.profile+")\n", reused.stderr)
			assert.Equal(t, int32(2), broker.requests.Load())
		})
	}
}

func TestPromptEndingWithoutAnswerFailsWithoutSaving(t *testing.T) {
	configFilePath := isolateEnvironment(t)
	broker := startBrokerStub(t)
	dependencies := components.New()
	dependencies.HTTPClient.SetBaseURL(broker.url)

	execution := executeRoot(t, dependencies, true, "", "create-participant", "pets")

	require.EqualError(t, execution.err, "no broker URL entered")
	assert.Equal(t, "Broker URL: ", execution.stderr)
	assert.Zero(t, broker.requests.Load())
	assert.NoFileExists(t, configFilePath)
}

func TestDotEnvInTheWorkingDirectoryIsIgnored(t *testing.T) {
	isolateEnvironment(t)
	broker := startBrokerStub(t)
	workingDirectory := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(workingDirectory, ".env"), []byte("BIDIREKT_BROKER_URL="+broker.url+"\n"), 0o600))
	t.Chdir(workingDirectory)

	execution := executeRoot(t, components.New(), false, "", "create-participant", "pets")

	require.EqualError(t, execution.err, noBrokerConfigured)
	assert.Zero(t, broker.requests.Load())
}
