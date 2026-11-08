package components

import (
	"errors"
	"fmt"
	"sort"
	"strings"
)

const (
	brokerURLEnvOrigin = "from BIDIREKT_BROKER_URL"
	apiKeyPrefix       = "bdk_"
)

var errOnlyOneEnvironmentVariable = errors.New("set both BIDIREKT_BROKER_URL and BIDIREKT_API_KEY, or neither")

type ProfileSources struct {
	EnvBrokerURL string
	EnvAPIKey    string
	FlagProfile  string
	EnvProfile   string
	ReadProfiles func() (map[string]Profile, error)
}

type ResolvedProfile struct {
	BrokerURL string
	APIKey    string
	Origin    string
	Profile   string
	// True only for a profile named by --profile or BIDIREKT_PROFILE that the config file lacks.
	ProfileNotFound bool
	OtherProfiles   []string
}

func ActiveProfileName(flagProfile, envProfile string) string {
	if flagProfile != "" {
		return flagProfile
	}
	if envProfile != "" {
		return envProfile
	}
	return "default"
}

func ResolveProfile(sources ProfileSources) (ResolvedProfile, error) {
	profileName := ActiveProfileName(sources.FlagProfile, sources.EnvProfile)

	var resolved ResolvedProfile
	var apiKeyOrigin string
	switch {
	case sources.EnvBrokerURL != "" && sources.EnvAPIKey != "":
		resolved = ResolvedProfile{BrokerURL: sources.EnvBrokerURL, APIKey: sources.EnvAPIKey, Origin: brokerURLEnvOrigin, Profile: profileName}
		apiKeyOrigin = "from BIDIREKT_API_KEY"
	case sources.EnvBrokerURL != "" || sources.EnvAPIKey != "":
		return ResolvedProfile{}, errOnlyOneEnvironmentVariable
	default:
		profiles, err := sources.ReadProfiles()
		if err != nil {
			return ResolvedProfile{}, err
		}
		profile, found := profiles[profileName]
		resolved = ResolvedProfile{
			BrokerURL:       profile.BrokerURL,
			APIKey:          profile.APIKey,
			Origin:          "profile: " + profileName,
			Profile:         profileName,
			ProfileNotFound: !found && (sources.FlagProfile != "" || sources.EnvProfile != ""),
			OtherProfiles:   otherProfileNames(profiles, profileName),
		}
		apiKeyOrigin = "in profile " + profileName
	}

	if resolved.BrokerURL == "" {
		return resolved, nil
	}
	if err := ValidateBrokerURL(resolved.BrokerURL, resolved.Origin); err != nil {
		return ResolvedProfile{}, err
	}
	if resolved.APIKey != "" && !strings.HasPrefix(resolved.APIKey, apiKeyPrefix) {
		return ResolvedProfile{}, fmt.Errorf("the API key %s doesn't start with %s", apiKeyOrigin, apiKeyPrefix)
	}

	return resolved, nil
}

func otherProfileNames(profiles map[string]Profile, activeProfileName string) []string {
	var names []string
	for name := range profiles {
		if name != activeProfileName {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}
