package internal

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/bidirekt/cli/internal/components"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const noBrokerConfigured = `no broker configured — run "bidirekt login" or set BIDIREKT_BROKER_URL and BIDIREKT_API_KEY`

type brokerStub struct {
	url            string
	requests       *atomic.Int32
	authorizations func() []string
}

func startBrokerStub(t *testing.T) brokerStub {
	t.Helper()

	requests := &atomic.Int32{}
	var lock sync.Mutex
	var authorizations []string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		requests.Add(1)
		lock.Lock()
		authorizations = append(authorizations, request.Header.Get("Authorization"))
		lock.Unlock()
		_, _ = writer.Write([]byte(`{"message":"participant created"}`))
	}))
	t.Cleanup(server.Close)

	return brokerStub{
		url:      server.URL,
		requests: requests,
		authorizations: func() []string {
			lock.Lock()
			defer lock.Unlock()
			return slices.Clone(authorizations)
		},
	}
}

func isolateEnvironment(t *testing.T) string {
	t.Helper()

	configFilePath := filepath.Join(t.TempDir(), "bidirekt", "config.json")
	t.Setenv("BIDIREKT_CONFIG_FILE", configFilePath)
	t.Setenv("BIDIREKT_BROKER_URL", "")
	t.Setenv("BIDIREKT_API_KEY", "")
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

func executeRoot(t *testing.T, dependencies *components.Components, args ...string) rootExecution {
	t.Helper()

	rootCommand := newRootCommand(dependencies)
	var stdout, stderr bytes.Buffer
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

			execution := executeRoot(t, components.New(), arg)

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
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			isolateEnvironment(t)

			execution := executeRoot(t, components.New(), args...)

			require.NoError(t, execution.err)
			assert.Empty(t, execution.stderr)
		})
	}
}

func TestBrokerCommandWithoutABrokerFailsWithoutCallingIt(t *testing.T) {
	configFilePath := isolateEnvironment(t)
	broker := startBrokerStub(t)
	dependencies := components.New()
	dependencies.HTTPClient.UseBroker(broker.url, "")

	execution := executeRoot(t, dependencies, "create-participant", "pets")

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
		{args: []string{"list-participants", "--broker-url", "x"}, err: "unknown flag: --broker-url"},
	}

	for _, test := range tests {
		t.Run(strings.Join(test.args, " "), func(t *testing.T) {
			configFilePath := isolateEnvironment(t)
			broker := startBrokerStub(t)

			execution := executeRoot(t, components.New(), test.args...)

			require.EqualError(t, execution.err, test.err)
			assert.Empty(t, execution.stderr)
			assert.Zero(t, broker.requests.Load())
			assert.NoFileExists(t, configFilePath)
		})
	}
}

func TestBrokerComesFromTheEnvironmentPairOrTheActiveProfileAndIsAnnouncedOnStderr(t *testing.T) {
	envBroker := startBrokerStub(t)
	defaultBroker := startBrokerStub(t)
	globexBroker := startBrokerStub(t)
	configFileContent := `{"profiles":{"default":{"brokerUrl":"` + defaultBroker.url + `","apiKey":"bdk_default"},"globex":{"brokerUrl":"` + globexBroker.url + `","apiKey":"bdk_globex"}}}`

	tests := []struct {
		name          string
		envURL        string
		envKey        string
		envProfile    string
		args          []string
		broker        brokerStub
		brokerLine    string
		authorization string
	}{
		{
			name:          "BIDIREKT_BROKER_URL and BIDIREKT_API_KEY win over the profile",
			envURL:        envBroker.url,
			envKey:        "bdk_env",
			args:          []string{"--profile", "globex"},
			broker:        envBroker,
			brokerLine:    "Broker: " + envBroker.url + " (from BIDIREKT_BROKER_URL)\n",
			authorization: "Bearer bdk_env",
		},
		{
			name:          "the default profile is used without --profile and BIDIREKT_PROFILE",
			broker:        defaultBroker,
			brokerLine:    "Broker: " + defaultBroker.url + " (profile: default)\n",
			authorization: "Bearer bdk_default",
		},
		{
			name:          "--profile selects the profile",
			args:          []string{"--profile", "globex"},
			broker:        globexBroker,
			brokerLine:    "Broker: " + globexBroker.url + " (profile: globex)\n",
			authorization: "Bearer bdk_globex",
		},
		{
			name:          "BIDIREKT_PROFILE selects the profile",
			envProfile:    "globex",
			broker:        globexBroker,
			brokerLine:    "Broker: " + globexBroker.url + " (profile: globex)\n",
			authorization: "Bearer bdk_globex",
		},
		{
			name:          "--profile wins over BIDIREKT_PROFILE",
			envProfile:    "default",
			args:          []string{"--profile", "globex"},
			broker:        globexBroker,
			brokerLine:    "Broker: " + globexBroker.url + " (profile: globex)\n",
			authorization: "Bearer bdk_globex",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			configFilePath := isolateEnvironment(t)
			writeConfigFile(t, configFilePath, configFileContent)
			t.Setenv("BIDIREKT_BROKER_URL", test.envURL)
			t.Setenv("BIDIREKT_API_KEY", test.envKey)
			t.Setenv("BIDIREKT_PROFILE", test.envProfile)
			requestsBefore := test.broker.requests.Load()

			execution := executeRoot(t, components.New(), append([]string{"create-participant", "pets"}, test.args...)...)

			require.NoError(t, execution.err)
			assert.Equal(t, "pets participant created\n", execution.stdout)
			assert.Equal(t, test.brokerLine, execution.stderr)
			assert.Equal(t, requestsBefore+1, test.broker.requests.Load())
			authorizations := test.broker.authorizations()
			assert.Equal(t, test.authorization, authorizations[len(authorizations)-1])
		})
	}
}

func TestProfileWithoutAKeySendsNoAuthorization(t *testing.T) {
	configFilePath := isolateEnvironment(t)
	broker := startBrokerStub(t)
	writeConfigFile(t, configFilePath, `{"profiles":{"default":{"brokerUrl":"`+broker.url+`"}}}`)

	execution := executeRoot(t, components.New(), "create-participant", "pets")

	require.NoError(t, execution.err)
	assert.Equal(t, []string{""}, broker.authorizations())
}

func TestEnvironmentPairIgnoresTheConfigFile(t *testing.T) {
	tests := []struct {
		name              string
		configFileContent string
		args              []string
	}{
		{name: "a corrupt config file", configFileContent: `{"profiles":`},
		{name: "a config file without the named profile", configFileContent: `{"profiles":{}}`, args: []string{"--profile", "acme"}},
		{name: "no config file"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			configFilePath := isolateEnvironment(t)
			if test.configFileContent != "" {
				writeConfigFile(t, configFilePath, test.configFileContent)
			}
			broker := startBrokerStub(t)
			t.Setenv("BIDIREKT_BROKER_URL", broker.url)
			t.Setenv("BIDIREKT_API_KEY", "bdk_env")

			execution := executeRoot(t, components.New(), append([]string{"create-participant", "pets"}, test.args...)...)

			require.NoError(t, execution.err)
			assert.Equal(t, []string{"Bearer bdk_env"}, broker.authorizations())
		})
	}
}

func TestConfigFileIsReadWithoutTheEnvironmentPair(t *testing.T) {
	configFilePath := isolateEnvironment(t)
	writeConfigFile(t, configFilePath, `{"profiles":`)
	broker := startBrokerStub(t)

	execution := executeRoot(t, components.New(), "create-participant", "pets")

	assert.ErrorContains(t, execution.err, "invalid config file "+configFilePath+":")
	assert.Zero(t, broker.requests.Load())
}

func TestOnlyOneOfTheEnvironmentPairFailsWithoutCallingTheBroker(t *testing.T) {
	tests := []struct {
		name   string
		envURL bool
		envKey bool
	}{
		{name: "only BIDIREKT_BROKER_URL", envURL: true},
		{name: "only BIDIREKT_API_KEY", envKey: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			configFilePath := isolateEnvironment(t)
			broker := startBrokerStub(t)
			writeConfigFile(t, configFilePath, `{"profiles":{"default":{"brokerUrl":"`+broker.url+`","apiKey":"bdk_default"}}}`)
			if test.envURL {
				t.Setenv("BIDIREKT_BROKER_URL", broker.url)
			}
			if test.envKey {
				t.Setenv("BIDIREKT_API_KEY", "bdk_env")
			}

			execution := executeRoot(t, components.New(), "create-participant", "pets")

			require.EqualError(t, execution.err, "set both BIDIREKT_BROKER_URL and BIDIREKT_API_KEY, or neither")
			assert.Empty(t, execution.stderr)
			assert.Zero(t, broker.requests.Load())
		})
	}
}

func TestKeyWithoutThePrefixNeverLeavesTheMachine(t *testing.T) {
	tests := []struct {
		name    string
		profile string
		envKey  string
		err     string
	}{
		{name: "from BIDIREKT_API_KEY", envKey: "acme_key", err: "the API key from BIDIREKT_API_KEY doesn't start with bdk_"},
		{name: "in the profile", profile: "acme_key", err: "the API key in profile default doesn't start with bdk_"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			configFilePath := isolateEnvironment(t)
			broker := startBrokerStub(t)
			writeConfigFile(t, configFilePath, `{"profiles":{"default":{"brokerUrl":"`+broker.url+`","apiKey":"`+test.profile+`"}}}`)
			if test.envKey != "" {
				t.Setenv("BIDIREKT_BROKER_URL", broker.url)
				t.Setenv("BIDIREKT_API_KEY", test.envKey)
			}

			execution := executeRoot(t, components.New(), "create-participant", "pets")

			require.EqualError(t, execution.err, test.err)
			assert.Empty(t, execution.stderr)
			assert.Zero(t, broker.requests.Load())
		})
	}
}

func TestBrokerLineHidesThePasswordAndTheKey(t *testing.T) {
	isolateEnvironment(t)
	broker := startBrokerStub(t)
	t.Setenv("BIDIREKT_BROKER_URL", strings.Replace(broker.url, "http://", "http://ci:segredo@", 1))
	t.Setenv("BIDIREKT_API_KEY", "bdk_segredo")

	execution := executeRoot(t, components.New(), "create-participant", "pets")

	require.NoError(t, execution.err)
	assert.Equal(t, "Broker: "+strings.Replace(broker.url, "http://", "http://ci:xxxxx@", 1)+" (from BIDIREKT_BROKER_URL)\n", execution.stderr)
	assert.NotContains(t, execution.stderr+execution.stdout, "segredo")
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
			name:   "from BIDIREKT_BROKER_URL",
			envURL: "ftp://x",
			err:    `invalid broker URL "ftp://x" (from BIDIREKT_BROKER_URL) — use http:// or https:// followed by a host`,
		},
		{
			name:              "from the profile",
			configFileContent: `{"profiles":{"globex":{"brokerUrl":"http://"}}}`,
			args:              []string{"--profile", "globex"},
			err:               `invalid broker URL "http://" (profile: globex) — use http:// or https:// followed by a host`,
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
			if test.envURL != "" {
				t.Setenv("BIDIREKT_BROKER_URL", test.envURL)
				t.Setenv("BIDIREKT_API_KEY", "bdk_env")
			}
			broker := startBrokerStub(t)
			dependencies := components.New()
			dependencies.HTTPClient.UseBroker(broker.url, "")

			execution := executeRoot(t, dependencies, append([]string{"create-participant", "pets"}, test.args...)...)

			require.EqualError(t, execution.err, test.err)
			assert.Empty(t, execution.stderr)
			assert.Zero(t, broker.requests.Load())
		})
	}
}

func TestProfileNamedByFlagOrEnvironmentMustExist(t *testing.T) {
	tests := []struct {
		name       string
		envProfile string
		args       []string
	}{
		{name: "named by --profile", args: []string{"--profile", "acme"}},
		{name: "named by BIDIREKT_PROFILE", envProfile: "acme"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			configFilePath := isolateEnvironment(t)
			writeConfigFile(t, configFilePath, `{"profiles":{"default":{"brokerUrl":"http://localhost:8080"}}}`)
			t.Setenv("BIDIREKT_PROFILE", test.envProfile)
			broker := startBrokerStub(t)
			dependencies := components.New()
			dependencies.HTTPClient.UseBroker(broker.url, "")

			execution := executeRoot(t, dependencies, append([]string{"create-participant", "pets"}, test.args...)...)

			require.EqualError(t, execution.err, `profile "acme" not found in `+configFilePath)
			assert.Empty(t, execution.stderr)
			assert.Zero(t, broker.requests.Load())
		})
	}
}

func TestMissingBrokerURLFailsWithoutWritingTheConfigFile(t *testing.T) {
	configFilePath := isolateEnvironment(t)
	const configFileContent = `{"profiles":{"acme":{"brokerUrl":"http://broker.acme.example"}}}`
	writeConfigFile(t, configFilePath, configFileContent)
	broker := startBrokerStub(t)
	dependencies := components.New()
	dependencies.HTTPClient.UseBroker(broker.url, "")

	execution := executeRoot(t, dependencies, "create-participant", "pets")

	require.EqualError(t, execution.err, noBrokerConfigured+"\nthe config file has the profile acme — pass --profile acme or set BIDIREKT_PROFILE=acme")
	assert.Empty(t, execution.stderr)
	assert.Zero(t, broker.requests.Load())
	saved, err := os.ReadFile(configFilePath)
	require.NoError(t, err)
	assert.JSONEq(t, configFileContent, string(saved))
}

func TestMissingBrokerURLNamesTheProfilesOfTheConfigFile(t *testing.T) {
	tests := []struct {
		name              string
		configFileContent string
		err               string
	}{
		{
			name: "no config file",
			err:  noBrokerConfigured,
		},
		{
			name:              "a config file without profiles",
			configFileContent: `{"profiles":{}}`,
			err:               noBrokerConfigured,
		},
		{
			name:              "only the local profile",
			configFileContent: `{"profiles":{"local":{"brokerUrl":"http://localhost:8080"}}}`,
			err:               noBrokerConfigured + "\nthe config file has the profile local — pass --profile local or set BIDIREKT_PROFILE=local",
		},
		{
			name:              "several profiles, in alphabetical order",
			configFileContent: `{"profiles":{"local":{"brokerUrl":"http://localhost:8080"},"acme":{"brokerUrl":"https://broker.acme.example"}}}`,
			err:               noBrokerConfigured + "\nthe config file has the profiles acme, local — pass --profile <name> or set BIDIREKT_PROFILE=<name>",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			configFilePath := isolateEnvironment(t)
			if test.configFileContent != "" {
				writeConfigFile(t, configFilePath, test.configFileContent)
			}
			broker := startBrokerStub(t)
			dependencies := components.New()
			dependencies.HTTPClient.UseBroker(broker.url, "")

			execution := executeRoot(t, dependencies, "list-participants")

			require.EqualError(t, execution.err, test.err)
			assert.Empty(t, execution.stdout)
			assert.Empty(t, execution.stderr)
			assert.Zero(t, broker.requests.Load())
		})
	}
}

func TestDotEnvInTheWorkingDirectoryIsIgnored(t *testing.T) {
	isolateEnvironment(t)
	broker := startBrokerStub(t)
	workingDirectory := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(workingDirectory, ".env"), []byte("BIDIREKT_BROKER_URL="+broker.url+"\n"), 0o600))
	t.Chdir(workingDirectory)

	execution := executeRoot(t, components.New(), "create-participant", "pets")

	require.EqualError(t, execution.err, noBrokerConfigured)
	assert.Zero(t, broker.requests.Load())
}

func TestAuthFailuresPrintOnlyTheirHint(t *testing.T) {
	statuses := []struct {
		status int
		hint   func(brokerURL string) string
	}{
		{status: http.StatusUnauthorized, hint: func(brokerURL string) string {
			return "not signed in to " + brokerURL + ` — run "bidirekt login" or set BIDIREKT_BROKER_URL and BIDIREKT_API_KEY`
		}},
		{status: http.StatusForbidden, hint: func(brokerURL string) string {
			return "the API key can't do this — check its permission at " + brokerURL + "/profile"
		}},
	}
	commands := [][]string{
		{"create-participant", "pets"},
		{"list-participants"},
		{"can-i-deploy", "pets", "--version", "v1", "--environment", "production"},
		{"record-deployment", "pets", "--version", "v1", "--environment", "production"},
	}

	for _, status := range statuses {
		for _, args := range commands {
			t.Run(http.StatusText(status.status)+" "+strings.Join(args, " "), func(t *testing.T) {
				isolateEnvironment(t)
				server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
					writer.WriteHeader(status.status)
					_, _ = writer.Write([]byte(`{"message":"refused"}`))
				}))
				t.Cleanup(server.Close)
				t.Setenv("BIDIREKT_BROKER_URL", server.URL)
				t.Setenv("BIDIREKT_API_KEY", "bdk_acme")

				execution := executeRoot(t, components.New(), args...)
				var printed bytes.Buffer
				printError(&printed, execution.err)

				require.Error(t, execution.err)
				assert.Equal(t, status.hint(server.URL)+"\n", printed.String())
				assert.Empty(t, execution.stdout)
				assert.Equal(t, "Broker: "+server.URL+" (from BIDIREKT_BROKER_URL)\n", execution.stderr)
			})
		}
	}
}

type roundTripFunc func(request *http.Request) (*http.Response, error)

func (this roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return this(request)
}

func TestKeyOverPlainHTTPToARemoteHostWarnsAndTheRequestProceeds(t *testing.T) {
	isolateEnvironment(t)
	const brokerURL = "http://broker.acme.internal:8080"
	t.Setenv("BIDIREKT_BROKER_URL", brokerURL)
	t.Setenv("BIDIREKT_API_KEY", "bdk_acme")
	dependencies := components.New()
	var requested []string
	dependencies.HTTPClient.StdClient().Transport = roundTripFunc(func(request *http.Request) (*http.Response, error) {
		requested = append(requested, request.URL.String())
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(`{"message":"participant created"}`)),
			Request:    request,
		}, nil
	})

	execution := executeRoot(t, dependencies, "create-participant", "pets")

	require.NoError(t, execution.err)
	assert.Equal(t, "Broker: "+brokerURL+" (from BIDIREKT_BROKER_URL)\nwarning: the API key travels unencrypted to "+brokerURL+"\n", execution.stderr)
	assert.Equal(t, []string{brokerURL + "/api/participants"}, requested)
	assert.NotContains(t, execution.stderr, "bdk_acme")
}

func TestBrokerRequestsCarryTheVersionOfAReleaseBuildOnly(t *testing.T) {
	tests := []struct {
		name        string
		version     string
		cliVersions []string
	}{
		{name: "a release build sends its version", version: "0.2.0", cliVersions: []string{"0.2.0"}},
		{name: "a dev build sends none", version: "dev"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			isolateEnvironment(t)
			t.Setenv("BIDIREKT_BROKER_URL", "https://broker.acme.internal")
			t.Setenv("BIDIREKT_API_KEY", "bdk_acme")
			builtVersion := version
			version = test.version
			t.Cleanup(func() { version = builtVersion })
			dependencies := components.New()
			var sent [][]string
			dependencies.HTTPClient.StdClient().Transport = roundTripFunc(func(request *http.Request) (*http.Response, error) {
				sent = append(sent, request.Header.Values("Bidirekt-Cli-Version"))
				return &http.Response{
					StatusCode: http.StatusOK,
					Header:     http.Header{"Content-Type": []string{"application/json"}},
					Body:       io.NopCloser(strings.NewReader(`{"message":"participant created"}`)),
					Request:    request,
				}, nil
			})

			execution := executeRoot(t, dependencies, "create-participant", "pets")

			require.NoError(t, execution.err)
			assert.Equal(t, [][]string{test.cliVersions}, sent)
		})
	}
}

func TestUpgradeRequiredPrintsOnlyTheBrokerMessage(t *testing.T) {
	const upgradeMessage = "bidirekt 0.0.9 doesn't work with this broker (0.3.0); upgrade bidirekt to 0.1.0 or later"
	commands := [][]string{
		{"create-participant", "pets"},
		{"list-participants"},
		{"can-i-deploy", "pets", "--version", "v1", "--environment", "production"},
		{"record-deployment", "pets", "--version", "v1", "--environment", "production"},
	}

	for _, args := range commands {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			isolateEnvironment(t)
			t.Setenv("BIDIREKT_BROKER_URL", "https://broker.acme.internal")
			t.Setenv("BIDIREKT_API_KEY", "bdk_acme")
			dependencies := components.New()
			dependencies.HTTPClient.StdClient().Transport = roundTripFunc(func(request *http.Request) (*http.Response, error) {
				return &http.Response{
					StatusCode: http.StatusUpgradeRequired,
					Header:     http.Header{"Content-Type": []string{"application/json"}},
					Body:       io.NopCloser(strings.NewReader(`{"message":"` + upgradeMessage + `"}`)),
					Request:    request,
				}, nil
			})

			execution := executeRoot(t, dependencies, args...)
			var printed bytes.Buffer
			printError(&printed, execution.err)

			require.Error(t, execution.err)
			assert.Equal(t, upgradeMessage+"\n", printed.String())
			assert.Empty(t, execution.stdout)
		})
	}
}

func TestLogoutIsListedAndNeverCallsTheBroker(t *testing.T) {
	configFilePath := isolateEnvironment(t)
	broker := startBrokerStub(t)
	writeConfigFile(t, configFilePath, `{"profiles":{"acme":{"brokerUrl":"`+broker.url+`","apiKey":"bdk_acme"}}}`)
	dependencies := components.New()
	dependencies.HTTPClient.UseBroker(broker.url, "")

	help := executeRoot(t, dependencies, "--help")
	logout := executeRoot(t, dependencies, "logout", "--profile", "acme")

	require.NoError(t, help.err)
	assert.Contains(t, help.stdout, "logout")
	require.NoError(t, logout.err)
	assert.Equal(t, "Signed out of "+broker.url+" (profile acme). The key is still valid — revoke it at "+broker.url+"/profile\n", logout.stdout)
	assert.Empty(t, logout.stderr)
	assert.Zero(t, broker.requests.Load())
}
