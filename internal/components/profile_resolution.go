package components

import "sort"

const (
	BrokerURLFlagOrigin = "from --broker-url"
	brokerURLEnvOrigin  = "from BIDIREKT_BROKER_URL"
)

type ProfileSources struct {
	FlagBrokerURL string
	EnvBrokerURL  string
	FlagProfile   string
	EnvProfile    string
	ReadProfiles  func() (map[string]Profile, error)
}

type ResolvedProfile struct {
	BrokerURL string
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
	switch {
	case sources.FlagBrokerURL != "":
		resolved = ResolvedProfile{BrokerURL: sources.FlagBrokerURL, Origin: BrokerURLFlagOrigin, Profile: profileName}
	case sources.EnvBrokerURL != "":
		resolved = ResolvedProfile{BrokerURL: sources.EnvBrokerURL, Origin: brokerURLEnvOrigin, Profile: profileName}
	default:
		profiles, err := sources.ReadProfiles()
		if err != nil {
			return ResolvedProfile{}, err
		}
		profile, found := profiles[profileName]
		resolved = ResolvedProfile{
			BrokerURL:       profile.BrokerURL,
			Origin:          "profile: " + profileName,
			Profile:         profileName,
			ProfileNotFound: !found && (sources.FlagProfile != "" || sources.EnvProfile != ""),
			OtherProfiles:   otherProfileNames(profiles, profileName),
		}
	}

	if resolved.BrokerURL == "" {
		return resolved, nil
	}
	if err := ValidateBrokerURL(resolved.BrokerURL, resolved.Origin); err != nil {
		return ResolvedProfile{}, err
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
