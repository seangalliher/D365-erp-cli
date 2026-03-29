package cmd

import (
	"strings"
	"testing"

	"github.com/seangalliher/d365-erp-cli/internal/config"
)

func TestDoctorCmd_Structure(t *testing.T) {
	t.Parallel()

	cmd := newDoctorCmd()
	if cmd.Use != "doctor" {
		t.Errorf("unexpected Use: %s", cmd.Use)
	}
	if cmd.Short == "" {
		t.Error("Short description should not be empty")
	}
}

func TestCheckConfigDir(t *testing.T) {
	t.Parallel()

	result := checkConfigDir()
	// Should pass on any system with a home directory.
	if result.Status != "pass" {
		t.Errorf("expected pass, got %s: %s", result.Status, result.Message)
	}
}

func TestCheckConfigFiles(t *testing.T) {
	t.Parallel()

	results := checkConfigFiles()
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	// Config file should always parse (defaults if missing).
	if results[0].Status == "fail" {
		t.Errorf("config file check failed: %s", results[0].Message)
	}
}

func TestCheckDNS_ValidHost(t *testing.T) {
	t.Parallel()

	result := checkDNS("https://www.microsoft.com")
	if result.Status != "pass" {
		t.Errorf("expected pass for microsoft.com DNS, got %s: %s", result.Status, result.Message)
	}
}

func TestCheckDNS_InvalidHost(t *testing.T) {
	t.Parallel()

	result := checkDNS("https://this-does-not-exist-d365-test.invalid")
	if result.Status != "fail" {
		t.Errorf("expected fail for invalid host, got %s", result.Status)
	}
}

func TestCheckTokenExpiry_Expired(t *testing.T) {
	t.Parallel()

	sess := &config.Session{
		TokenExpiry: "2020-01-01T00:00:00Z",
	}
	result := checkTokenExpiry(sess)
	if result.Status != "fail" {
		t.Errorf("expected fail for expired token, got %s", result.Status)
	}
	if !strings.Contains(result.Message, "expired") {
		t.Errorf("message should mention expiry: %s", result.Message)
	}
}

func TestCheckTokenExpiry_Valid(t *testing.T) {
	t.Parallel()

	sess := &config.Session{
		TokenExpiry: "2099-01-01T00:00:00Z",
	}
	result := checkTokenExpiry(sess)
	if result.Status != "pass" {
		t.Errorf("expected pass for future token, got %s: %s", result.Status, result.Message)
	}
}

func TestCheckAuthConfig(t *testing.T) {
	t.Parallel()

	result := checkAuthConfig()
	// This test just verifies it runs without panicking.
	if result.Status != "pass" && result.Status != "warn" {
		t.Errorf("expected pass or warn, got %s", result.Status)
	}
}
