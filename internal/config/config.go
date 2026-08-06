// Package config loads user settings from ~/.sandbox.
//
// Only settings that change behaviour today are defined. The file is optional
// and never written by the CLI: absent file means defaults.
package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

const (
	dirName  = ".sandbox"
	fileName = "config.json"

	// DefaultImage is the sandbox image name, built on demand when absent.
	DefaultImage = "sandbox-cli:latest"
	// DefaultLogLevel keeps normal runs quiet.
	DefaultLogLevel = "info"
)

// Config is the user-tunable subset of sandbox behaviour.
type Config struct {
	// Image overrides the sandbox image name.
	Image string `json:"image"`
	// LogLevel is one of debug, info, warn, error.
	LogLevel string `json:"log_level"`
}

// Default returns the configuration used when nothing is set.
func Default() Config {
	return Config{Image: DefaultImage, LogLevel: DefaultLogLevel}
}

// Dir returns ~/.sandbox.
func Dir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("locate home directory: %w", err)
	}
	return filepath.Join(home, dirName), nil
}

// Path returns the config file location.
func Path() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, fileName), nil
}

// LoadDockerfile returns the contents of ~/.sandbox/Dockerfile, or "" if there is
// none. It is how a user adds a language runtime the default image omits, without
// patching the embedded one.
func LoadDockerfile() (string, error) {
	dir, err := Dir()
	if err != nil {
		return "", err
	}
	path := filepath.Join(dir, "Dockerfile")
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("read %s: %w", path, err)
	}
	return string(data), nil
}

// Load reads the config file. A missing file is not an error. Fields absent from
// the file keep their default, since the defaults are decoded into first.
func Load() (Config, error) {
	cfg := Default()

	path, err := Path()
	if err != nil {
		return cfg, err
	}
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return cfg, nil
	}
	if err != nil {
		return cfg, fmt.Errorf("read %s: %w", path, err)
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Default(), fmt.Errorf("parse %s: %w", path, err)
	}
	return cfg, nil
}
