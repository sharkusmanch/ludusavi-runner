package config

import (
	"os"
	"path/filepath"
	"runtime"
)

const (
	// AppName is the application name used for config directories.
	AppName = "ludusavi-runner"
	// ConfigFileName is the default config file name.
	ConfigFileName = "config.toml"
	// EnvPrefix is the prefix for environment variables.
	EnvPrefix = "LUDUSAVI_RUNNER"
)

// DefaultConfigDir returns the default configuration directory for the current OS.
func DefaultConfigDir() (string, error) {
	switch runtime.GOOS {
	case "windows":
		// %APPDATA%\ludusavi-runner
		appData := os.Getenv("APPDATA")
		if appData == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return "", err
			}
			appData = filepath.Join(home, "AppData", "Roaming")
		}
		return filepath.Join(appData, AppName), nil

	case "darwin":
		// ~/Library/Application Support/ludusavi-runner
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, "Library", "Application Support", AppName), nil

	default:
		// Linux and other Unix-like systems
		// $XDG_CONFIG_HOME/ludusavi-runner or ~/.config/ludusavi-runner
		if xdgConfig := os.Getenv("XDG_CONFIG_HOME"); xdgConfig != "" {
			return filepath.Join(xdgConfig, AppName), nil
		}
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, ".config", AppName), nil
	}
}

// DefaultConfigPath returns the full path to the default config file.
func DefaultConfigPath() (string, error) {
	dir, err := DefaultConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, ConfigFileName), nil
}

// DefaultLogDir returns the default log directory for the current OS.
func DefaultLogDir() (string, error) {
	switch runtime.GOOS {
	case "windows":
		// %LOCALAPPDATA%\ludusavi-runner\logs
		localAppData := os.Getenv("LOCALAPPDATA")
		if localAppData == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return "", err
			}
			localAppData = filepath.Join(home, "AppData", "Local")
		}
		return filepath.Join(localAppData, AppName, "logs"), nil

	case "darwin":
		// ~/Library/Logs/ludusavi-runner
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, "Library", "Logs", AppName), nil

	default:
		// Linux: $XDG_STATE_HOME/ludusavi-runner or ~/.local/state/ludusavi-runner
		if xdgState := os.Getenv("XDG_STATE_HOME"); xdgState != "" {
			return filepath.Join(xdgState, AppName), nil
		}
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		return filepath.Join(home, ".local", "state", AppName), nil
	}
}
