// Package config manages CLI configuration including profiles, environment
// variables, and config files. Configuration is layered:
//  1. Defaults (hardcoded)
//  2. Config file (~/.d365cli/config.json)
//  3. Profile-specific overrides
//  4. Environment variables (D365_*)
//  5. Command-line flags (highest priority)
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

const (
	// ConfigDir is the directory name for CLI configuration.
	ConfigDir = ".d365cli"
	// ConfigFile is the config file name.
	ConfigFile = "config.json"
	// SessionFile stores active session state.
	SessionFile = "session.json"
)

// Config represents the full CLI configuration.
type Config struct {
	mu sync.RWMutex

	// DefaultProfile is the active profile name.
	DefaultProfile string `json:"default_profile"`
	// Profiles contains named configuration profiles.
	Profiles map[string]*Profile `json:"profiles"`
	// Telemetry enables anonymous usage telemetry.
	Telemetry bool `json:"telemetry"`
	// AutoUpdate enables automatic update checks.
	AutoUpdate bool `json:"auto_update"`
}

// Profile represents a named configuration profile.
type Profile struct {
	Name         string `json:"name"`
	Environment  string `json:"environment"`
	Company      string `json:"company"`
	OutputFormat string `json:"output_format,omitempty"`
	AuthMethod   string `json:"auth_method,omitempty"`
	ClientID     string `json:"client_id,omitempty"`
	TenantID     string `json:"tenant_id,omitempty"`

	// Daemon settings
	DaemonIdleTimeout int `json:"daemon_idle_timeout,omitempty"` // seconds

	// Retry settings
	MaxRetries int `json:"max_retries,omitempty"`
	RetryDelay int `json:"retry_delay_ms,omitempty"` // milliseconds
}

// Session represents the active session state persisted across CLI invocations.
type Session struct {
	Connected   bool   `json:"connected"`
	Environment string `json:"environment"`
	Company     string `json:"company"`
	User        string `json:"user"`
	TokenExpiry string `json:"token_expiry"`
	DaemonPID   int    `json:"daemon_pid,omitempty"`
	DaemonAddr  string `json:"daemon_addr,omitempty"`
	ProfileName string `json:"profile_name"`
}

// DefaultConfig returns a config with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		DefaultProfile: "default",
		Profiles: map[string]*Profile{
			"default": {
				Name:              "default",
				OutputFormat:      "json",
				DaemonIdleTimeout: 1800, // 30 minutes
				MaxRetries:        3,
				RetryDelay:        1000,
			},
		},
		Telemetry:  false,
		AutoUpdate: true,
	}
}

// ConfigDirPath returns the path to the CLI config directory.
func ConfigDirPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}
	return filepath.Join(home, ConfigDir), nil
}

// EnsureConfigDir creates the config directory if it doesn't exist.
func EnsureConfigDir() (string, error) {
	dir, err := ConfigDirPath()
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", fmt.Errorf("cannot create config directory: %w", err)
	}
	return dir, nil
}

// Load reads the config from disk, or returns defaults if no config exists.
func Load() (*Config, error) {
	dir, err := ConfigDirPath()
	if err != nil {
		return DefaultConfig(), nil
	}

	path := filepath.Join(dir, ConfigFile)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultConfig(), nil
		}
		return nil, fmt.Errorf("cannot read config: %w", err)
	}

	cfg := DefaultConfig()
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("cannot parse config: %w", err)
	}
	return cfg, nil
}

// Save writes the config to disk.
func (c *Config) Save() error {
	c.mu.RLock()
	defer c.mu.RUnlock()

	dir, err := EnsureConfigDir()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("cannot marshal config: %w", err)
	}

	path := filepath.Join(dir, ConfigFile)
	return os.WriteFile(path, data, 0600)
}

// ActiveProfile returns the current active profile.
func (c *Config) ActiveProfile(override string) *Profile {
	c.mu.RLock()
	defer c.mu.RUnlock()

	name := c.DefaultProfile
	if override != "" {
		name = override
	}

	if p, ok := c.Profiles[name]; ok {
		return p
	}
	// Return default if profile not found
	if p, ok := c.Profiles["default"]; ok {
		return p
	}
	return &Profile{Name: "default"}
}

// SetProfile adds or updates a profile.
func (c *Config) SetProfile(p *Profile) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.Profiles == nil {
		c.Profiles = make(map[string]*Profile)
	}
	c.Profiles[p.Name] = p
}

// LoadSession reads the active session from disk.
func LoadSession() (*Session, error) {
	dir, err := ConfigDirPath()
	if err != nil {
		return &Session{}, nil
	}

	path := filepath.Join(dir, SessionFile)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &Session{}, nil
		}
		return nil, fmt.Errorf("cannot read session: %w", err)
	}

	var sess Session
	if err := json.Unmarshal(data, &sess); err != nil {
		return nil, fmt.Errorf("cannot parse session: %w", err)
	}
	return &sess, nil
}

// SaveSession writes the session to disk.
func SaveSession(sess *Session) error {
	dir, err := EnsureConfigDir()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(sess, "", "  ")
	if err != nil {
		return fmt.Errorf("cannot marshal session: %w", err)
	}

	path := filepath.Join(dir, SessionFile)
	return os.WriteFile(path, data, 0600)
}

// ClearSession removes the session file.
func ClearSession() error {
	dir, err := ConfigDirPath()
	if err != nil {
		return err
	}
	path := filepath.Join(dir, SessionFile)
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// GetEnvOrDefault reads an environment variable with a fallback.
func GetEnvOrDefault(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// Environment variable names.
const (
	EnvURL          = "D365_URL"
	EnvCompany      = "D365_COMPANY"
	EnvClientID     = "D365_CLIENT_ID"
	EnvClientSecret = "D365_CLIENT_SECRET"
	EnvTenantID     = "D365_TENANT_ID"
	EnvAuthMethod   = "D365_AUTH_METHOD"
	EnvProfile      = "D365_PROFILE"
	EnvOutput       = "D365_OUTPUT"
	EnvCI           = "D365_CI"
)
