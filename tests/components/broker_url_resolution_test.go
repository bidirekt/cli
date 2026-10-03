package components_test

import (
	"errors"
	"testing"

	"github.com/bidirekt/cli/internal/components"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveBrokerURLTakesTheFirstSourceThatHasOne(t *testing.T) {
	const (
		flagURL    = "http://flag.internal"
		envURL     = "http://env.internal"
		defaultURL = "http://default.internal"
		stagingURL = "http://staging.internal"
	)
	profiles := map[string]components.Profile{
		"default": {BrokerURL: defaultURL},
		"staging": {BrokerURL: stagingURL},
	}

	tests := []struct {
		name     string
		sources  components.BrokerURLSources
		profiles map[string]components.Profile
		resolved components.ResolvedBrokerURL
	}{
		{
			name:     "--broker-url wins over BIDIREKT_BROKER_URL and the profile",
			sources:  components.BrokerURLSources{FlagBrokerURL: flagURL, EnvBrokerURL: envURL},
			profiles: profiles,
			resolved: components.ResolvedBrokerURL{BrokerURL: flagURL, Origin: "from --broker-url", Profile: "default"},
		},
		{
			name:     "BIDIREKT_BROKER_URL wins over the profile",
			sources:  components.BrokerURLSources{EnvBrokerURL: envURL},
			profiles: profiles,
			resolved: components.ResolvedBrokerURL{BrokerURL: envURL, Origin: "from BIDIREKT_BROKER_URL", Profile: "default"},
		},
		{
			name:     "the default profile without --profile and BIDIREKT_PROFILE",
			profiles: profiles,
			resolved: components.ResolvedBrokerURL{BrokerURL: defaultURL, Origin: "profile: default", Profile: "default"},
		},
		{
			name:     "--profile selects the profile",
			sources:  components.BrokerURLSources{FlagProfile: "staging"},
			profiles: profiles,
			resolved: components.ResolvedBrokerURL{BrokerURL: stagingURL, Origin: "profile: staging", Profile: "staging"},
		},
		{
			name:     "BIDIREKT_PROFILE selects the profile",
			sources:  components.BrokerURLSources{EnvProfile: "staging"},
			profiles: profiles,
			resolved: components.ResolvedBrokerURL{BrokerURL: stagingURL, Origin: "profile: staging", Profile: "staging"},
		},
		{
			name:     "--profile wins over BIDIREKT_PROFILE",
			sources:  components.BrokerURLSources{FlagProfile: "staging", EnvProfile: "default"},
			profiles: profiles,
			resolved: components.ResolvedBrokerURL{BrokerURL: stagingURL, Origin: "profile: staging", Profile: "staging"},
		},
		{
			name:     "--broker-url wins over the selected profile, which stays the active one",
			sources:  components.BrokerURLSources{FlagBrokerURL: flagURL, FlagProfile: "staging"},
			profiles: profiles,
			resolved: components.ResolvedBrokerURL{BrokerURL: flagURL, Origin: "from --broker-url", Profile: "staging"},
		},
		{
			name:     "BIDIREKT_BROKER_URL makes a missing profile irrelevant",
			sources:  components.BrokerURLSources{EnvBrokerURL: envURL, FlagProfile: "missing"},
			profiles: profiles,
			resolved: components.ResolvedBrokerURL{BrokerURL: envURL, Origin: "from BIDIREKT_BROKER_URL", Profile: "missing"},
		},
		{
			name:     "a lower source with an invalid URL is never looked at",
			sources:  components.BrokerURLSources{EnvBrokerURL: envURL},
			profiles: map[string]components.Profile{"default": {BrokerURL: "broker.acme.com"}},
			resolved: components.ResolvedBrokerURL{BrokerURL: envURL, Origin: "from BIDIREKT_BROKER_URL", Profile: "default"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			sources := test.sources
			sources.ReadProfiles = func() (map[string]components.Profile, error) { return test.profiles, nil }

			resolved, err := components.ResolveBrokerURL(sources)

			require.NoError(t, err)
			assert.Equal(t, test.resolved, resolved)
		})
	}
}

func TestResolveBrokerURLReadsTheConfigFileOnlyWhenNoFlagOrEnvironmentHasAURL(t *testing.T) {
	unreadable := func() (map[string]components.Profile, error) {
		return nil, errors.New("invalid config file /home/ci/.config/bidirekt/config.json: unexpected end of JSON input")
	}

	fromFlag, err := components.ResolveBrokerURL(components.BrokerURLSources{FlagBrokerURL: "http://flag.internal", ReadProfiles: unreadable})
	require.NoError(t, err)
	assert.Equal(t, "http://flag.internal", fromFlag.BrokerURL)

	fromEnv, err := components.ResolveBrokerURL(components.BrokerURLSources{EnvBrokerURL: "http://env.internal", ReadProfiles: unreadable})
	require.NoError(t, err)
	assert.Equal(t, "http://env.internal", fromEnv.BrokerURL)

	_, err = components.ResolveBrokerURL(components.BrokerURLSources{ReadProfiles: unreadable})
	assert.EqualError(t, err, "invalid config file /home/ci/.config/bidirekt/config.json: unexpected end of JSON input")
}

func TestResolveBrokerURLReportsAMissingURL(t *testing.T) {
	tests := []struct {
		name     string
		sources  components.BrokerURLSources
		profiles map[string]components.Profile
		resolved components.ResolvedBrokerURL
	}{
		{
			name:     "no source at all",
			profiles: map[string]components.Profile{},
			resolved: components.ResolvedBrokerURL{Origin: "profile: default", Profile: "default"},
		},
		{
			name:     "the default profile missing is not a profile not found",
			profiles: map[string]components.Profile{"staging": {BrokerURL: "http://staging.internal"}},
			resolved: components.ResolvedBrokerURL{Origin: "profile: default", Profile: "default"},
		},
		{
			name:     "a profile named by --profile and missing from the file is not found",
			sources:  components.BrokerURLSources{FlagProfile: "staging"},
			profiles: map[string]components.Profile{},
			resolved: components.ResolvedBrokerURL{Origin: "profile: staging", Profile: "staging", ProfileNotFound: true},
		},
		{
			name:     "a profile named by BIDIREKT_PROFILE and missing from the file is not found",
			sources:  components.BrokerURLSources{EnvProfile: "staging"},
			profiles: map[string]components.Profile{},
			resolved: components.ResolvedBrokerURL{Origin: "profile: staging", Profile: "staging", ProfileNotFound: true},
		},
		{
			name:     "a profile without brokerUrl exists but has no URL",
			sources:  components.BrokerURLSources{FlagProfile: "staging"},
			profiles: map[string]components.Profile{"staging": {}},
			resolved: components.ResolvedBrokerURL{Origin: "profile: staging", Profile: "staging"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			sources := test.sources
			sources.ReadProfiles = func() (map[string]components.Profile, error) { return test.profiles, nil }

			resolved, err := components.ResolveBrokerURL(sources)

			require.NoError(t, err)
			assert.Equal(t, test.resolved, resolved)
		})
	}
}

func TestResolveBrokerURLRejectsAnInvalidURLNamingItsOrigin(t *testing.T) {
	tests := []struct {
		name     string
		sources  components.BrokerURLSources
		profiles map[string]components.Profile
		err      string
	}{
		{
			name:    "from --broker-url",
			sources: components.BrokerURLSources{FlagBrokerURL: "broker.acme.com"},
			err:     `invalid broker URL "broker.acme.com" (from --broker-url) — use http:// or https:// followed by a host`,
		},
		{
			name:    "from BIDIREKT_BROKER_URL",
			sources: components.BrokerURLSources{EnvBrokerURL: "broker.acme.com"},
			err:     `invalid broker URL "broker.acme.com" (from BIDIREKT_BROKER_URL) — use http:// or https:// followed by a host`,
		},
		{
			name:     "from the profile",
			sources:  components.BrokerURLSources{EnvProfile: "staging"},
			profiles: map[string]components.Profile{"staging": {BrokerURL: "broker.acme.com"}},
			err:      `invalid broker URL "broker.acme.com" (profile: staging) — use http:// or https:// followed by a host`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			sources := test.sources
			sources.ReadProfiles = func() (map[string]components.Profile, error) { return test.profiles, nil }

			_, err := components.ResolveBrokerURL(sources)

			assert.EqualError(t, err, test.err)
		})
	}
}

func TestActiveProfileNamePrefersTheFlagThenTheEnvironment(t *testing.T) {
	assert.Equal(t, "staging", components.ActiveProfileName("staging", "other"))
	assert.Equal(t, "other", components.ActiveProfileName("", "other"))
	assert.Equal(t, "default", components.ActiveProfileName("", ""))
}
