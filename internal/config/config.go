package config

import (
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
)

// Config holds user configuration loaded from yuman.toml.
type Config struct {
	Managers map[string]ManagerConfig `toml:"managers"`
	General  GeneralConfig            `toml:"general"`
}

type ManagerConfig struct {
	Enabled bool   `toml:"enabled"`
	Path    string `toml:"path"` // custom binary path
}

type GeneralConfig struct {
	AURHelper string `toml:"aur_helper"` // "paru" or "yay"
	Timeout   int    `toml:"timeout"`    // seconds per operation
}

// DefaultConfig returns a sensible default configuration.
func DefaultConfig() *Config {
	return &Config{
		Managers: map[string]ManagerConfig{
			"pacman":  {Enabled: true},
			"paru":    {Enabled: true},
			"pip":     {Enabled: true},
			"npm":     {Enabled: true},
			"pnpm":    {Enabled: true},
			"cargo":   {Enabled: true},
			"flatpak": {Enabled: true},
		},
		General: GeneralConfig{
			AURHelper: "paru",
			Timeout:   30,
		},
	}
}

// Load reads the config file from the standard location, falling back to defaults.
func Load() (*Config, error) {
	cfg := DefaultConfig()

	path, err := configPath()
	if err != nil {
		return cfg, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		// File not found is fine — use defaults
		return cfg, nil
	}

	if err := toml.Unmarshal(data, cfg); err != nil {
		return cfg, err
	}

	return cfg, nil
}

// configPath returns the path to the yuman.toml config file.
func configPath() (string, error) {
	// Check XDG_CONFIG_HOME first
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "yuman", "yuman.toml"), nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(home, ".config", "yuman", "yuman.toml"), nil
}

// IsEnabled checks whether a manager is enabled in config.
func (c *Config) IsEnabled(name string) bool {
	if mc, ok := c.Managers[name]; ok {
		return mc.Enabled
	}
	return true // default to enabled if not configured
}

// TimeoutSeconds returns the configured timeout or the default 30s.
func (c *Config) TimeoutSeconds() int {
	if c.General.Timeout > 0 {
		return c.General.Timeout
	}
	return 30
}
