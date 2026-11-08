package components_test

import (
	"errors"
	"testing"

	"github.com/bidirekt/cli/internal/components"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveProfileTakesTheEnvironmentPairOrTheActiveProfile(t *testing.T) {
	const (
		envURL     = "http://env.internal"
		envKey     = "bdk_env"
		defaultURL = "http://default.internal"
		defaultKey = "bdk_default"
		globexURL  = "http://globex.internal"
		globexKey  = "bdk_globex"
	)
	profiles := map[string]components.Profile{
		"default": {BrokerURL: defaultURL, APIKey: defaultKey},
		"globex":  {BrokerURL: globexURL, APIKey: globexKey},
	}

	tests := []struct {
		name     string
		sources  components.ProfileSources
		profiles map[string]components.Profile
		resolved components.ResolvedProfile
	}{
		{
			name:     "BIDIREKT_BROKER_URL and BIDIREKT_API_KEY win over the profile",
			sources:  components.ProfileSources{EnvBrokerURL: envURL, EnvAPIKey: envKey},
			profiles: profiles,
			resolved: components.ResolvedProfile{BrokerURL: envURL, APIKey: envKey, Origin: "from BIDIREKT_BROKER_URL", Profile: "default"},
		},
		{
			name:     "the default profile without --profile and BIDIREKT_PROFILE",
			profiles: profiles,
			resolved: components.ResolvedProfile{BrokerURL: defaultURL, APIKey: defaultKey, Origin: "profile: default", Profile: "default", OtherProfiles: []string{"globex"}},
		},
		{
			name:     "--profile selects the profile",
			sources:  components.ProfileSources{FlagProfile: "globex"},
			profiles: profiles,
			resolved: components.ResolvedProfile{BrokerURL: globexURL, APIKey: globexKey, Origin: "profile: globex", Profile: "globex", OtherProfiles: []string{"default"}},
		},
		{
			name:     "BIDIREKT_PROFILE selects the profile",
			sources:  components.ProfileSources{EnvProfile: "globex"},
			profiles: profiles,
			resolved: components.ResolvedProfile{BrokerURL: globexURL, APIKey: globexKey, Origin: "profile: globex", Profile: "globex", OtherProfiles: []string{"default"}},
		},
		{
			name:     "--profile wins over BIDIREKT_PROFILE",
			sources:  components.ProfileSources{FlagProfile: "globex", EnvProfile: "default"},
			profiles: profiles,
			resolved: components.ResolvedProfile{BrokerURL: globexURL, APIKey: globexKey, Origin: "profile: globex", Profile: "globex", OtherProfiles: []string{"default"}},
		},
		{
			name:     "a profile without apiKey resolves without a key",
			sources:  components.ProfileSources{FlagProfile: "globex"},
			profiles: map[string]components.Profile{"globex": {BrokerURL: globexURL}},
			resolved: components.ResolvedProfile{BrokerURL: globexURL, Origin: "profile: globex", Profile: "globex"},
		},
		{
			name:     "the environment pair makes a missing profile irrelevant",
			sources:  components.ProfileSources{EnvBrokerURL: envURL, EnvAPIKey: envKey, FlagProfile: "missing"},
			profiles: profiles,
			resolved: components.ResolvedProfile{BrokerURL: envURL, APIKey: envKey, Origin: "from BIDIREKT_BROKER_URL", Profile: "missing"},
		},
		{
			name:     "a profile with an invalid URL or key is never looked at",
			sources:  components.ProfileSources{EnvBrokerURL: envURL, EnvAPIKey: envKey},
			profiles: map[string]components.Profile{"default": {BrokerURL: "broker.acme.com", APIKey: "acme"}},
			resolved: components.ResolvedProfile{BrokerURL: envURL, APIKey: envKey, Origin: "from BIDIREKT_BROKER_URL", Profile: "default"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			sources := test.sources
			sources.ReadProfiles = func() (map[string]components.Profile, error) { return test.profiles, nil }

			resolved, err := components.ResolveProfile(sources)

			require.NoError(t, err)
			assert.Equal(t, test.resolved, resolved)
		})
	}
}

func TestResolveProfileReadsTheConfigFileOnlyWithoutTheEnvironmentPair(t *testing.T) {
	readCount := 0
	unreadable := func() (map[string]components.Profile, error) {
		readCount++
		return nil, errors.New("invalid config file /home/ci/.config/bidirekt/config.json: unexpected end of JSON input")
	}

	fromEnv, err := components.ResolveProfile(components.ProfileSources{EnvBrokerURL: "http://env.internal", EnvAPIKey: "bdk_env", ReadProfiles: unreadable})
	require.NoError(t, err)
	assert.Equal(t, "http://env.internal", fromEnv.BrokerURL)
	assert.Equal(t, "bdk_env", fromEnv.APIKey)
	assert.Zero(t, readCount)

	_, err = components.ResolveProfile(components.ProfileSources{ReadProfiles: unreadable})
	assert.EqualError(t, err, "invalid config file /home/ci/.config/bidirekt/config.json: unexpected end of JSON input")
	assert.Equal(t, 1, readCount)
}

func TestResolveProfileRejectsOnlyOneOfTheEnvironmentPair(t *testing.T) {
	for _, sources := range []components.ProfileSources{
		{EnvBrokerURL: "http://env.internal"},
		{EnvAPIKey: "bdk_env"},
	} {
		t.Run(sources.EnvBrokerURL+sources.EnvAPIKey, func(t *testing.T) {
			sources.ReadProfiles = func() (map[string]components.Profile, error) {
				return map[string]components.Profile{"default": {BrokerURL: "http://default.internal", APIKey: "bdk_default"}}, nil
			}

			_, err := components.ResolveProfile(sources)

			assert.EqualError(t, err, "set both BIDIREKT_BROKER_URL and BIDIREKT_API_KEY, or neither")
		})
	}
}

func TestResolveProfileRejectsAKeyWithoutThePrefixNamingItsOrigin(t *testing.T) {
	tests := []struct {
		name     string
		sources  components.ProfileSources
		profiles map[string]components.Profile
		err      string
	}{
		{
			name:    "from BIDIREKT_API_KEY",
			sources: components.ProfileSources{EnvBrokerURL: "http://env.internal", EnvAPIKey: "acme_key"},
			err:     "the API key from BIDIREKT_API_KEY doesn't start with bdk_",
		},
		{
			name:     "in the profile",
			sources:  components.ProfileSources{FlagProfile: "acme"},
			profiles: map[string]components.Profile{"acme": {BrokerURL: "https://broker.acme.internal", APIKey: "BDK_acme"}},
			err:      "the API key in profile acme doesn't start with bdk_",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			sources := test.sources
			sources.ReadProfiles = func() (map[string]components.Profile, error) { return test.profiles, nil }

			_, err := components.ResolveProfile(sources)

			assert.EqualError(t, err, test.err)
		})
	}
}

func TestResolveProfileReportsAMissingURL(t *testing.T) {
	tests := []struct {
		name     string
		sources  components.ProfileSources
		profiles map[string]components.Profile
		resolved components.ResolvedProfile
	}{
		{
			name:     "no source at all",
			profiles: map[string]components.Profile{},
			resolved: components.ResolvedProfile{Origin: "profile: default", Profile: "default"},
		},
		{
			name:     "the default profile missing is not a profile not found",
			profiles: map[string]components.Profile{"globex": {BrokerURL: "http://globex.internal"}},
			resolved: components.ResolvedProfile{Origin: "profile: default", Profile: "default", OtherProfiles: []string{"globex"}},
		},
		{
			name:     "a profile named by --profile and missing from the file is not found",
			sources:  components.ProfileSources{FlagProfile: "globex"},
			profiles: map[string]components.Profile{},
			resolved: components.ResolvedProfile{Origin: "profile: globex", Profile: "globex", ProfileNotFound: true},
		},
		{
			name:     "a profile named by BIDIREKT_PROFILE and missing from the file is not found",
			sources:  components.ProfileSources{EnvProfile: "globex"},
			profiles: map[string]components.Profile{},
			resolved: components.ResolvedProfile{Origin: "profile: globex", Profile: "globex", ProfileNotFound: true},
		},
		{
			name:     "a profile without brokerUrl exists but has no URL",
			sources:  components.ProfileSources{FlagProfile: "globex"},
			profiles: map[string]components.Profile{"globex": {}},
			resolved: components.ResolvedProfile{Origin: "profile: globex", Profile: "globex"},
		},
		{
			name: "the other profiles of the file come in alphabetical order",
			profiles: map[string]components.Profile{
				"local":  {BrokerURL: "http://localhost:8080"},
				"globex": {BrokerURL: "https://broker.globex.example"},
				"acme":   {BrokerURL: "https://broker.acme.example"},
			},
			resolved: components.ResolvedProfile{Origin: "profile: default", Profile: "default", OtherProfiles: []string{"acme", "globex", "local"}},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			sources := test.sources
			sources.ReadProfiles = func() (map[string]components.Profile, error) { return test.profiles, nil }

			resolved, err := components.ResolveProfile(sources)

			require.NoError(t, err)
			assert.Equal(t, test.resolved, resolved)
		})
	}
}

func TestResolveProfileRejectsAnInvalidURLNamingItsOrigin(t *testing.T) {
	tests := []struct {
		name     string
		sources  components.ProfileSources
		profiles map[string]components.Profile
		err      string
	}{
		{
			name:    "from BIDIREKT_BROKER_URL",
			sources: components.ProfileSources{EnvBrokerURL: "broker.acme.com", EnvAPIKey: "bdk_env"},
			err:     `invalid broker URL "broker.acme.com" (from BIDIREKT_BROKER_URL) — use http:// or https:// followed by a host`,
		},
		{
			name:     "from the profile",
			sources:  components.ProfileSources{EnvProfile: "globex"},
			profiles: map[string]components.Profile{"globex": {BrokerURL: "broker.acme.com"}},
			err:      `invalid broker URL "broker.acme.com" (profile: globex) — use http:// or https:// followed by a host`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			sources := test.sources
			sources.ReadProfiles = func() (map[string]components.Profile, error) { return test.profiles, nil }

			_, err := components.ResolveProfile(sources)

			assert.EqualError(t, err, test.err)
		})
	}
}

func TestActiveProfileNamePrefersTheFlagThenTheEnvironment(t *testing.T) {
	assert.Equal(t, "globex", components.ActiveProfileName("globex", "other"))
	assert.Equal(t, "other", components.ActiveProfileName("", "other"))
	assert.Equal(t, "default", components.ActiveProfileName("", ""))
}
