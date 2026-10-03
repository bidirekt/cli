package components

const (
	BrokerURLFlagOrigin = "from --broker-url"
	brokerURLEnvOrigin  = "from BIDIREKT_BROKER_URL"
)

type BrokerURLSources struct {
	FlagBrokerURL string
	EnvBrokerURL  string
	FlagProfile   string
	EnvProfile    string
	ReadProfiles  func() (map[string]Profile, error)
}

type ResolvedBrokerURL struct {
	BrokerURL string
	Origin    string
	Profile   string
	// True only for a profile named by --profile or BIDIREKT_PROFILE that the config file lacks.
	ProfileNotFound bool
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

func ResolveBrokerURL(sources BrokerURLSources) (ResolvedBrokerURL, error) {
	profileName := ActiveProfileName(sources.FlagProfile, sources.EnvProfile)

	var resolved ResolvedBrokerURL
	switch {
	case sources.FlagBrokerURL != "":
		resolved = ResolvedBrokerURL{BrokerURL: sources.FlagBrokerURL, Origin: BrokerURLFlagOrigin, Profile: profileName}
	case sources.EnvBrokerURL != "":
		resolved = ResolvedBrokerURL{BrokerURL: sources.EnvBrokerURL, Origin: brokerURLEnvOrigin, Profile: profileName}
	default:
		profiles, err := sources.ReadProfiles()
		if err != nil {
			return ResolvedBrokerURL{}, err
		}
		profile, found := profiles[profileName]
		resolved = ResolvedBrokerURL{
			BrokerURL:       profile.BrokerURL,
			Origin:          "profile: " + profileName,
			Profile:         profileName,
			ProfileNotFound: !found && (sources.FlagProfile != "" || sources.EnvProfile != ""),
		}
	}

	if resolved.BrokerURL == "" {
		return resolved, nil
	}
	if err := ValidateBrokerURL(resolved.BrokerURL, resolved.Origin); err != nil {
		return ResolvedBrokerURL{}, err
	}

	return resolved, nil
}
