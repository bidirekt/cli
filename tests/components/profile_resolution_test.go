package components_test

import (
	"errors"
	"testing"

	"github.com/bidirekt/cli/internal/components"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveProfileTakesTheFirstSourceThatHasOne(t *testing.T) {
	const (
		flagURL    = "http://flag.internal"
		envURL     = "http://env.internal"
		defaultURL = "http://default.internal"
		globexURL  = "http://globex.internal"
	)
	profiles := map[string]components.Profile{
		"default": {BrokerURL: defaultURL},
		"globex":  {BrokerURL: globexURL},
	}

	tests := []struct {
		name     string
		sources  components.ProfileSources
		profiles map[string]components.Profile
		resolved components.ResolvedProfile
	}{
		{
			name:     "--broker-url wins over BIDIREKT_BROKER_URL and the profile",
			sources:  components.ProfileSources{FlagBrokerURL: flagURL, EnvBrokerURL: envURL},
			profiles: profiles,
			resolved: components.ResolvedProfile{BrokerURL: flagURL, Origin: "from --broker-url", Profile: "default"},
		},
		{
			name:     "BIDIREKT_BROKER_URL wins over the profile",
			sources:  components.ProfileSources{EnvBrokerURL: envURL},
			profiles: profiles,
			resolved: components.ResolvedProfile{BrokerURL: envURL, Origin: "from BIDIREKT_BROKER_URL", Profile: "default"},
		},
		{
			name:     "the default profile without --profile and BIDIREKT_PROFILE",
			profiles: profiles,
			resolved: components.ResolvedProfile{BrokerURL: defaultURL, Origin: "profile: default", Profile: "default", OtherProfiles: []string{"globex"}},
		},
		{
			name:     "--profile selects the profile",
			sources:  components.ProfileSources{FlagProfile: "globex"},
			profiles: profiles,
			resolved: components.ResolvedProfile{BrokerURL: globexURL, Origin: "profile: globex", Profile: "globex", OtherProfiles: []string{"default"}},
		},
		{
			name:     "BIDIREKT_PROFILE selects the profile",
			sources:  components.ProfileSources{EnvProfile: "globex"},
			profiles: profiles,
			resolved: components.ResolvedProfile{BrokerURL: globexURL, Origin: "profile: globex", Profile: "globex", OtherProfiles: []string{"default"}},
		},
		{
			name:     "--profile wins over BIDIREKT_PROFILE",
			sources:  components.ProfileSources{FlagProfile: "globex", EnvProfile: "default"},
			profiles: profiles,
			resolved: components.ResolvedProfile{BrokerURL: globexURL, Origin: "profile: globex", Profile: "globex", OtherProfiles: []string{"default"}},
		},
		{
			name:     "--broker-url wins over the selected profile, which stays the active one",
			sources:  components.ProfileSources{FlagBrokerURL: flagURL, FlagProfile: "globex"},
			profiles: profiles,
			resolved: components.ResolvedProfile{BrokerURL: flagURL, Origin: "from --broker-url", Profile: "globex"},
		},
		{
			name:     "BIDIREKT_BROKER_URL makes a missing profile irrelevant",
			sources:  components.ProfileSources{EnvBrokerURL: envURL, FlagProfile: "missing"},
			profiles: profiles,
			resolved: components.ResolvedProfile{BrokerURL: envURL, Origin: "from BIDIREKT_BROKER_URL", Profile: "missing"},
		},
		{
			name:     "a lower source with an invalid URL is never looked at",
			sources:  components.ProfileSources{EnvBrokerURL: envURL},
			profiles: map[string]components.Profile{"default": {BrokerURL: "broker.acme.com"}},
			resolved: components.ResolvedProfile{BrokerURL: envURL, Origin: "from BIDIREKT_BROKER_URL", Profile: "default"},
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

func TestResolveProfileReadsTheConfigFileOnlyWhenNoFlagOrEnvironmentHasAURL(t *testing.T) {
	unreadable := func() (map[string]components.Profile, error) {
		return nil, errors.New("invalid config file /home/ci/.config/bidirekt/config.json: unexpected end of JSON input")
	}

	fromFlag, err := components.ResolveProfile(components.ProfileSources{FlagBrokerURL: "http://flag.internal", ReadProfiles: unreadable})
	require.NoError(t, err)
	assert.Equal(t, "http://flag.internal", fromFlag.BrokerURL)

	fromEnv, err := components.ResolveProfile(components.ProfileSources{EnvBrokerURL: "http://env.internal", ReadProfiles: unreadable})
	require.NoError(t, err)
	assert.Equal(t, "http://env.internal", fromEnv.BrokerURL)

	_, err = components.ResolveProfile(components.ProfileSources{ReadProfiles: unreadable})
	assert.EqualError(t, err, "invalid config file /home/ci/.config/bidirekt/config.json: unexpected end of JSON input")
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
			name:    "from --broker-url",
			sources: components.ProfileSources{FlagBrokerURL: "broker.acme.com"},
			err:     `invalid broker URL "broker.acme.com" (from --broker-url) — use http:// or https:// followed by a host`,
		},
		{
			name:    "from BIDIREKT_BROKER_URL",
			sources: components.ProfileSources{EnvBrokerURL: "broker.acme.com"},
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
