package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	t.Parallel()

	cfg := DefaultConfig()

	if cfg.DefaultProfile != "default" {
		t.Errorf("expected default profile name 'default', got %q", cfg.DefaultProfile)
	}
	if _, ok := cfg.Profiles["default"]; !ok {
		t.Error("expected default profile to exist")
	}
	if cfg.Profiles["default"].DaemonIdleTimeout != 1800 {
		t.Errorf("expected daemon idle timeout 1800, got %d", cfg.Profiles["default"].DaemonIdleTimeout)
	}
}

func TestActiveProfile(t *testing.T) {
	t.Parallel()

	cfg := DefaultConfig()
	cfg.Profiles["staging"] = &Profile{
		Name:        "staging",
		Environment: "https://staging.operations.dynamics.com",
		Company:     "STAG",
	}

	// Default profile
	p := cfg.ActiveProfile("")
	if p.Name != "default" {
		t.Errorf("expected default profile, got %q", p.Name)
	}

	// Override profile
	p = cfg.ActiveProfile("staging")
	if p.Name != "staging" {
		t.Errorf("expected staging profile, got %q", p.Name)
	}
	if p.Company != "STAG" {
		t.Errorf("expected company STAG, got %q", p.Company)
	}

	// Non-existent profile returns default
	p = cfg.ActiveProfile("nonexistent")
	if p.Name != "default" {
		t.Errorf("expected fallback to default, got %q", p.Name)
	}
}

func TestSetProfile(t *testing.T) {
	t.Parallel()

	cfg := DefaultConfig()
	cfg.SetProfile(&Profile{
		Name:        "production",
		Environment: "https://prod.operations.dynamics.com",
		Company:     "PROD",
	})

	p, ok := cfg.Profiles["production"]
	if !ok {
		t.Fatal("expected production profile to exist")
	}
	if p.Company != "PROD" {
		t.Errorf("expected company PROD, got %q", p.Company)
	}
}

func TestConfigSaveAndLoad(t *testing.T) {
	t.Parallel()

	// Use a temp directory as home
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, ConfigDir)
	if err := os.MkdirAll(configDir, 0700); err != nil {
		t.Fatal(err)
	}

	cfg := DefaultConfig()
	cfg.Profiles["test"] = &Profile{
		Name:        "test",
		Environment: "https://test.example.com",
		Company:     "TST",
	}

	// Save to temp dir
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(configDir, ConfigFile)
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}

	// Read it back
	readData, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	var loaded Config
	if err := json.Unmarshal(readData, &loaded); err != nil {
		t.Fatal(err)
	}

	if _, ok := loaded.Profiles["test"]; !ok {
		t.Error("expected test profile in loaded config")
	}
	if loaded.Profiles["test"].Company != "TST" {
		t.Errorf("expected company TST, got %q", loaded.Profiles["test"].Company)
	}
}

func TestSessionSaveAndLoad(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, ConfigDir)
	if err := os.MkdirAll(configDir, 0700); err != nil {
		t.Fatal(err)
	}

	sess := &Session{
		Connected:   true,
		Environment: "https://test.example.com",
		Company:     "USMF",
		User:        "admin@test.com",
	}

	data, err := json.MarshalIndent(sess, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(configDir, SessionFile)
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}

	readData, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}

	var loaded Session
	if err := json.Unmarshal(readData, &loaded); err != nil {
		t.Fatal(err)
	}

	if !loaded.Connected {
		t.Error("expected connected=true")
	}
	if loaded.Company != "USMF" {
		t.Errorf("expected company USMF, got %q", loaded.Company)
	}
}

func TestGetEnvOrDefault(t *testing.T) {
	t.Parallel()

	// With no env var set, should return default
	val := GetEnvOrDefault("D365_TEST_NONEXISTENT_12345", "fallback")
	if val != "fallback" {
		t.Errorf("expected 'fallback', got %q", val)
	}
}
