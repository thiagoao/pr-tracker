package config

import (
	"io/ioutil"
	"log"
	"log/slog"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config represents the application configuration
type Config struct {
	Bitbucket struct {
		Domain       string   `yaml:"domain"`
		Port         int      `yaml:"port"`
		Workspace    string   `yaml:"workspace"`
		User         string   `yaml:"user"`
		AppPassword  string   `yaml:"app_password"`
		Repositories []string `yaml:"repositories"`
	} `yaml:"bitbucket"`
	PRFilter struct {
		IgnoreKeywords []string `yaml:"ignore_keywords"`
		StaleAfterDays int      `yaml:"stale_after_days"`
	} `yaml:"pr_filter"`
	Notifiers struct {
		SMTP struct {
			Host     string   `yaml:"host"`
			Port     int      `yaml:"port"`
			User     string   `yaml:"user"`
			Password string   `yaml:"password"`
			From     string   `yaml:"from"`
			To       []string `yaml:"to"`
		} `yaml:"smtp"`
		Teams struct {
			WebhookURL string `yaml:"webhook_url"`
		} `yaml:"teams"`
	} `yaml:"notifiers"`
	Log struct {
		File       string `yaml:"file"`
		Level      string `yaml:"level"`
		Format     string `yaml:"format"`
		MaxSizeMB  int    `yaml:"max_size_mb"`
		MaxBackups int    `yaml:"max_backups"`
		MaxAgeDays int    `yaml:"max_age_days"`
		Compress   bool   `yaml:"compress"`
		Stdout     bool   `yaml:"stdout"`
	} `yaml:"log"`
	Notification struct {
		IntervalHours int `yaml:"interval_hours"`
	} `yaml:"notification"`
}

// Load reads and parses the configuration file
func Load(path string) *Config {
	var config Config
	data, err := ioutil.ReadFile(path)
	if err != nil {
		log.Fatalf("Error reading configuration file: %v", err)
	}

	err = yaml.Unmarshal(data, &config)
	if err != nil {
		log.Fatalf("Error parsing YAML: %v", err)
	}

	// Trim spaces from Bitbucket user and app_password if not empty
	originalUser := config.Bitbucket.User
	originalPass := config.Bitbucket.AppPassword
	config.Bitbucket.User = strings.TrimSpace(config.Bitbucket.User)
	config.Bitbucket.AppPassword = strings.TrimSpace(config.Bitbucket.AppPassword)
	if config.Bitbucket.User != originalUser {
		slog.Debug("Trimmed spaces from Bitbucket user in config.")
	}
	if config.Bitbucket.AppPassword != originalPass {
		slog.Debug("Trimmed spaces from Bitbucket app_password in config.")
	}
	return &config
}
