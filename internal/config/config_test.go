package config

import (
	"os"
	"testing"
)

func TestLoad_ValidConfig(t *testing.T) {
	// Create a temporary config file
	configContent := `
bitbucket:
  domain: "bitbucket.org"
  port: 443
  workspace: "test-workspace"
  user: "test-user"
  app_password: "test-password"
  repositories:
    - "repo1"
    - "repo2"

pr_filter:
  ignore_keywords:
    - "WIP"
    - "DRAFT"
  stale_after_days: 7

notifiers:
  smtp:
    host: "smtp.gmail.com"
    port: 587
    user: "test@example.com"
    password: "test-password"
    from: "test@example.com"
    to:
      - "admin@example.com"
  teams:
    webhook_url: "https://webhook.url"

log:
  file: "logs/app.log"
  level: "info"
  format: "json"
  max_size_mb: 100
  max_backups: 3
  max_age_days: 30
  compress: true
  stdout: true

notification:
  interval_hours: 24
`

	tempFile := "test_config.yaml"
	err := os.WriteFile(tempFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}
	defer os.Remove(tempFile)

	// Test loading the config
	config := Load(tempFile)

	// Verify Bitbucket configuration
	if config.Bitbucket.Domain != "bitbucket.org" {
		t.Errorf("Expected domain 'bitbucket.org', got '%s'", config.Bitbucket.Domain)
	}
	if config.Bitbucket.Port != 443 {
		t.Errorf("Expected port 443, got %d", config.Bitbucket.Port)
	}
	if config.Bitbucket.Workspace != "test-workspace" {
		t.Errorf("Expected workspace 'test-workspace', got '%s'", config.Bitbucket.Workspace)
	}
	if config.Bitbucket.User != "test-user" {
		t.Errorf("Expected user 'test-user', got '%s'", config.Bitbucket.User)
	}
	if config.Bitbucket.AppPassword != "test-password" {
		t.Errorf("Expected app_password 'test-password', got '%s'", config.Bitbucket.AppPassword)
	}
	if len(config.Bitbucket.Repositories) != 2 {
		t.Errorf("Expected 2 repositories, got %d", len(config.Bitbucket.Repositories))
	}

	// Verify PR filter configuration
	if len(config.PRFilter.IgnoreKeywords) != 2 {
		t.Errorf("Expected 2 ignore keywords, got %d", len(config.PRFilter.IgnoreKeywords))
	}
	if config.PRFilter.StaleAfterDays != 7 {
		t.Errorf("Expected stale_after_days 7, got %d", config.PRFilter.StaleAfterDays)
	}

	// Verify SMTP configuration
	if config.Notifiers.SMTP.Host != "smtp.gmail.com" {
		t.Errorf("Expected SMTP host 'smtp.gmail.com', got '%s'", config.Notifiers.SMTP.Host)
	}
	if config.Notifiers.SMTP.Port != 587 {
		t.Errorf("Expected SMTP port 587, got %d", config.Notifiers.SMTP.Port)
	}
	if len(config.Notifiers.SMTP.To) != 1 {
		t.Errorf("Expected 1 SMTP recipient, got %d", len(config.Notifiers.SMTP.To))
	}

	// Verify Teams configuration
	if config.Notifiers.Teams.WebhookURL != "https://webhook.url" {
		t.Errorf("Expected Teams webhook URL 'https://webhook.url', got '%s'", config.Notifiers.Teams.WebhookURL)
	}

	// Verify Log configuration
	if config.Log.File != "logs/app.log" {
		t.Errorf("Expected log file 'logs/app.log', got '%s'", config.Log.File)
	}
	if config.Log.Level != "info" {
		t.Errorf("Expected log level 'info', got '%s'", config.Log.Level)
	}
	if !config.Log.Stdout {
		t.Errorf("Expected stdout to be true, got %v", config.Log.Stdout)
	}

	// Verify Notification configuration
	if config.Notification.IntervalHours != 24 {
		t.Errorf("Expected interval hours 24, got %d", config.Notification.IntervalHours)
	}
}

func TestLoad_TrimSpaces(t *testing.T) {
	// Create a config file with spaces in user and password
	configContent := `
bitbucket:
  domain: "bitbucket.org"
  port: 443
  workspace: "test-workspace"
  user: "  test-user  "
  app_password: "  test-password  "
  repositories:
    - "repo1"
`

	tempFile := "test_config_trim.yaml"
	err := os.WriteFile(tempFile, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test config file: %v", err)
	}
	defer os.Remove(tempFile)

	config := Load(tempFile)

	// Verify that spaces are trimmed
	if config.Bitbucket.User != "test-user" {
		t.Errorf("Expected trimmed user 'test-user', got '%s'", config.Bitbucket.User)
	}
	if config.Bitbucket.AppPassword != "test-password" {
		t.Errorf("Expected trimmed app_password 'test-password', got '%s'", config.Bitbucket.AppPassword)
	}
}

func TestLoad_InvalidFile(t *testing.T) {
	// Test loading a non-existent file
	// This test is skipped because Load() uses log.Fatalf() which calls os.Exit(1)
	// and cannot be tested in a unit test environment
	t.Skip("Skipping test because Load() uses log.Fatalf() which calls os.Exit(1)")
}

func TestLoad_InvalidYAML(t *testing.T) {
	// Create a config file with invalid YAML
	// This test is skipped because Load() uses log.Fatalf() which calls os.Exit(1)
	// and cannot be tested in a unit test environment
	t.Skip("Skipping test because Load() uses log.Fatalf() which calls os.Exit(1)")
}
