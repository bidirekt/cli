package components

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
)

type Profile struct {
	BrokerURL string `json:"brokerUrl"`
}

type ConfigFile struct {
	Path string
}

type configFileContent struct {
	Profiles map[string]Profile `json:"profiles"`
}

func ConfigFileFromEnvironment() (ConfigFile, error) {
	if path := os.Getenv("BIDIREKT_CONFIG_FILE"); path != "" {
		return ConfigFile{Path: path}, nil
	}

	if configHome := os.Getenv("XDG_CONFIG_HOME"); configHome != "" {
		return ConfigFile{Path: filepath.Join(configHome, "bidirekt", "config.json")}, nil
	}

	if runtime.GOOS == "windows" {
		appData, err := os.UserConfigDir()
		if err != nil {
			return ConfigFile{}, fmt.Errorf("locate config file: %w", err)
		}
		return ConfigFile{Path: filepath.Join(appData, "bidirekt", "config.json")}, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return ConfigFile{}, fmt.Errorf("locate config file: %w", err)
	}
	return ConfigFile{Path: filepath.Join(home, ".config", "bidirekt", "config.json")}, nil
}

func (this ConfigFile) ReadProfiles() (map[string]Profile, error) {
	data, err := os.ReadFile(this.Path)
	if errors.Is(err, fs.ErrNotExist) {
		return map[string]Profile{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read config file: %w", err)
	}

	var content configFileContent
	if err := json.Unmarshal(data, &content); err != nil {
		return nil, fmt.Errorf("invalid config file %s: %w", this.Path, err)
	}
	if content.Profiles == nil {
		return map[string]Profile{}, nil
	}

	return content.Profiles, nil
}

func (this ConfigFile) WriteProfile(name string, profile Profile) error {
	profiles, err := this.ReadProfiles()
	if err != nil {
		return err
	}
	profiles[name] = profile

	data, err := json.MarshalIndent(configFileContent{Profiles: profiles}, "", "  ")
	if err != nil {
		return fmt.Errorf("serialize config file: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(this.Path), 0o700); err != nil {
		return fmt.Errorf("create config folder: %w", err)
	}
	if err := os.WriteFile(this.Path, append(data, '\n'), 0o600); err != nil {
		return fmt.Errorf("write config file: %w", err)
	}

	return nil
}
