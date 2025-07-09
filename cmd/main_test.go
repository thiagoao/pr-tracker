package main

import (
	"context"
	"fc-pr-tracker/internal/bitbucket"
	"fc-pr-tracker/internal/config"
	"fc-pr-tracker/internal/notifier"
	"fc-pr-tracker/pkg/models"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

// runWithMock allows injecting a mock Bitbucket client for testing
func runWithMock(ctx context.Context, cfg *config.Config, mockClient *bitbucket.Client) error {
	// Initialize notifiers
	notifiers := []notifier.Notifier{
		notifier.NewEmailNotifier(cfg),
	}

	if cfg.Notifiers.Teams.WebhookURL != "" {
		notifiers = append(notifiers, notifier.NewTeamsNotifier(cfg))
	}

	// Initialize state store
	stateStore := &models.FileNotificationStateStore{Path: "tmp/last_notification.txt"}
	checkFreq := time.Duration(cfg.Notification.IntervalHours) * time.Hour

	// Use the mock client instead of creating a new one
	bitbucketClient := mockClient

	for {
		select {
		case <-ctx.Done():
			return nil
		default:
			lastNotified, err := stateStore.GetLastNotificationTime()
			if err != nil {
				return err
			}

			interval := time.Duration(cfg.Notification.IntervalHours) * time.Hour
			shouldNotify := lastNotified.IsZero() || time.Since(lastNotified) >= interval

			if !shouldNotify {
				slog.Info("No notification sent (interval not reached)", "last_notified", lastNotified)
				slog.Info("Sleeping until next check...", "hours", cfg.Notification.IntervalHours)
				select {
				case <-ctx.Done():
					return nil
				case <-time.After(checkFreq):
					// continue loop
				}
				continue
			}

			var allPRsToNotify []models.PullRequest
			repoPRsToNotify := make(map[string][]models.PullRequest)
			prParticipants := make(map[int][]models.Participant)

			for _, repo := range cfg.Bitbucket.Repositories {
				slog.Info("Fetching open PRs for repository", "repo", repo)
				prs, err := bitbucketClient.ListOpenPRs(repo)
				if err != nil {
					slog.Error("Error fetching PRs for repository", "repo", repo, "error", err)
					continue
				}
				slog.Info("Total open PRs", "repo", repo, "total", len(prs))

				filtered := bitbucket.FilterPRs(prs, cfg.PRFilter.IgnoreKeywords)
				slog.Info("PRs after keyword filter", "repo", repo, "filtered_total", len(filtered))

				for _, pr := range filtered {
					participants, err := bitbucketClient.GetParticipants(repo, pr.ID)
					if err != nil {
						slog.Error("Error fetching PR participants", "repo", repo, "pr_id", pr.ID, "error", err)
						continue
					}
					prParticipants[pr.ID] = participants

					if bitbucket.IsPRApproved(participants) {
						continue
					}

					comments, err := bitbucketClient.GetComments(repo, pr.ID)
					if err != nil {
						slog.Error("Error fetching PR comments", "repo", repo, "pr_id", pr.ID, "error", err)
						continue
					}

					lastActivity := bitbucket.GetLastActivity(pr, comments)
					if lastActivity == "" {
						slog.Warn("No last activity date found for PR", "repo", repo, "pr_id", pr.ID, "title", pr.Title)
						continue
					}

					lastTime, err := time.Parse(time.RFC3339, lastActivity)
					if err != nil {
						slog.Warn("Error parsing PR last activity date", "repo", repo, "pr_id", pr.ID, "title", pr.Title, "date", lastActivity, "error", err)
						continue
					}

					daysWithoutActivity := int(time.Since(lastTime).Hours() / 24)
					if daysWithoutActivity >= cfg.PRFilter.StaleAfterDays {
						allPRsToNotify = append(allPRsToNotify, pr)
						repoPRsToNotify[repo] = append(repoPRsToNotify[repo], pr)
					}
				}
			}

			if len(allPRsToNotify) > 0 {
				slog.Info("Sending summary notification email", "prs_to_notify", len(allPRsToNotify))
				for _, notifier := range notifiers {
					err := notifier.Notify(allPRsToNotify, repoPRsToNotify, prParticipants, cfg.PRFilter.StaleAfterDays)
					if err != nil {
						slog.Error("Error notifying", "error", err)
					}
				}
				err = stateStore.SetLastNotificationTime(time.Now())
				if err != nil {
					slog.Error("Error updating last notification time", "error", err)
				}
			} else {
				slog.Info("No PRs to notify in this cycle.")
			}

			slog.Info("Sleeping until next check...", "hours", cfg.Notification.IntervalHours)
			select {
			case <-ctx.Done():
				return nil
			case <-time.After(checkFreq):
				// continue loop
			}
		}
	}
}

// createMockBitbucketServer creates a mock Bitbucket server for testing
func createMockBitbucketServer() (*httptest.Server, *bitbucket.Client) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Mock successful responses
		w.WriteHeader(200)
		w.Write([]byte(`{"values":[]}`))
	}))

	cfg := &config.Config{
		Bitbucket: struct {
			Domain       string   `yaml:"domain"`
			Port         int      `yaml:"port"`
			Workspace    string   `yaml:"workspace"`
			User         string   `yaml:"user"`
			AppPassword  string   `yaml:"app_password"`
			Repositories []string `yaml:"repositories"`
		}{
			Workspace: "test-workspace",
		},
	}

	client := &bitbucket.Client{
		Config:  cfg,
		Client:  server.Client(),
		BaseURL: server.URL,
	}

	return server, client
}

func TestRun_EmptyConfig(t *testing.T) {
	// Create a minimal config for testing
	cfg := &config.Config{
		Bitbucket: struct {
			Domain       string   `yaml:"domain"`
			Port         int      `yaml:"port"`
			Workspace    string   `yaml:"workspace"`
			User         string   `yaml:"user"`
			AppPassword  string   `yaml:"app_password"`
			Repositories []string `yaml:"repositories"`
		}{
			Domain:       "bitbucket.org",
			Port:         443,
			Workspace:    "test-workspace",
			User:         "test-user",
			AppPassword:  "test-password",
			Repositories: []string{},
		},
		PRFilter: struct {
			IgnoreKeywords []string `yaml:"ignore_keywords"`
			StaleAfterDays int      `yaml:"stale_after_days"`
		}{
			IgnoreKeywords: []string{"WIP", "DRAFT"},
			StaleAfterDays: 7,
		},
		Notifiers: struct {
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
		}{
			SMTP: struct {
				Host     string   `yaml:"host"`
				Port     int      `yaml:"port"`
				User     string   `yaml:"user"`
				Password string   `yaml:"password"`
				From     string   `yaml:"from"`
				To       []string `yaml:"to"`
			}{
				Host:     "smtp.gmail.com",
				Port:     587,
				User:     "test@example.com",
				Password: "test-password",
				From:     "test@example.com",
				To:       []string{"admin@example.com"},
			},
		},
		Notification: struct {
			IntervalHours int `yaml:"interval_hours"`
		}{
			IntervalHours: 24,
		},
	}

	// Create mock Bitbucket server
	server, mockClient := createMockBitbucketServer()
	defer server.Close()

	// Create context with short timeout for testing
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Run the service with mock client
	err := runWithMock(ctx, cfg, mockClient)
	if err != nil {
		t.Errorf("Expected no error when running with empty config, got: %v", err)
	}
}

func TestRun_WithRepositories(t *testing.T) {
	// Create a config with repositories
	cfg := &config.Config{
		Bitbucket: struct {
			Domain       string   `yaml:"domain"`
			Port         int      `yaml:"port"`
			Workspace    string   `yaml:"workspace"`
			User         string   `yaml:"user"`
			AppPassword  string   `yaml:"app_password"`
			Repositories []string `yaml:"repositories"`
		}{
			Domain:       "bitbucket.org",
			Port:         443,
			Workspace:    "test-workspace",
			User:         "test-user",
			AppPassword:  "test-password",
			Repositories: []string{"test-repo"},
		},
		PRFilter: struct {
			IgnoreKeywords []string `yaml:"ignore_keywords"`
			StaleAfterDays int      `yaml:"stale_after_days"`
		}{
			IgnoreKeywords: []string{"WIP", "DRAFT"},
			StaleAfterDays: 7,
		},
		Notifiers: struct {
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
		}{
			SMTP: struct {
				Host     string   `yaml:"host"`
				Port     int      `yaml:"port"`
				User     string   `yaml:"user"`
				Password string   `yaml:"password"`
				From     string   `yaml:"from"`
				To       []string `yaml:"to"`
			}{
				Host:     "smtp.gmail.com",
				Port:     587,
				User:     "test@example.com",
				Password: "test-password",
				From:     "test@example.com",
				To:       []string{"admin@example.com"},
			},
		},
		Notification: struct {
			IntervalHours int `yaml:"interval_hours"`
		}{
			IntervalHours: 24,
		},
	}

	// Create mock Bitbucket server
	server, mockClient := createMockBitbucketServer()
	defer server.Close()

	// Create context with short timeout for testing
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Run the service with mock client
	err := runWithMock(ctx, cfg, mockClient)
	if err != nil {
		t.Errorf("Expected no error when running with repositories, got: %v", err)
	}
}

func TestRun_WithTeamsNotifier(t *testing.T) {
	// Create a config with Teams notifier
	cfg := &config.Config{
		Bitbucket: struct {
			Domain       string   `yaml:"domain"`
			Port         int      `yaml:"port"`
			Workspace    string   `yaml:"workspace"`
			User         string   `yaml:"user"`
			AppPassword  string   `yaml:"app_password"`
			Repositories []string `yaml:"repositories"`
		}{
			Domain:       "bitbucket.org",
			Port:         443,
			Workspace:    "test-workspace",
			User:         "test-user",
			AppPassword:  "test-password",
			Repositories: []string{},
		},
		PRFilter: struct {
			IgnoreKeywords []string `yaml:"ignore_keywords"`
			StaleAfterDays int      `yaml:"stale_after_days"`
		}{
			IgnoreKeywords: []string{"WIP", "DRAFT"},
			StaleAfterDays: 7,
		},
		Notifiers: struct {
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
		}{
			SMTP: struct {
				Host     string   `yaml:"host"`
				Port     int      `yaml:"port"`
				User     string   `yaml:"user"`
				Password string   `yaml:"password"`
				From     string   `yaml:"from"`
				To       []string `yaml:"to"`
			}{
				Host:     "smtp.gmail.com",
				Port:     587,
				User:     "test@example.com",
				Password: "test-password",
				From:     "test@example.com",
				To:       []string{"admin@example.com"},
			},
			Teams: struct {
				WebhookURL string `yaml:"webhook_url"`
			}{
				WebhookURL: "https://webhook.url",
			},
		},
		Notification: struct {
			IntervalHours int `yaml:"interval_hours"`
		}{
			IntervalHours: 24,
		},
	}

	// Create mock Bitbucket server
	server, mockClient := createMockBitbucketServer()
	defer server.Close()

	// Create context with short timeout for testing
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Run the service with mock client
	err := runWithMock(ctx, cfg, mockClient)
	if err != nil {
		t.Errorf("Expected no error when running with Teams notifier, got: %v", err)
	}
}

func TestRun_ContextCancellation(t *testing.T) {
	// Create a minimal config
	cfg := &config.Config{
		Bitbucket: struct {
			Domain       string   `yaml:"domain"`
			Port         int      `yaml:"port"`
			Workspace    string   `yaml:"workspace"`
			User         string   `yaml:"user"`
			AppPassword  string   `yaml:"app_password"`
			Repositories []string `yaml:"repositories"`
		}{
			Domain:       "bitbucket.org",
			Port:         443,
			Workspace:    "test-workspace",
			User:         "test-user",
			AppPassword:  "test-password",
			Repositories: []string{},
		},
		PRFilter: struct {
			IgnoreKeywords []string `yaml:"ignore_keywords"`
			StaleAfterDays int      `yaml:"stale_after_days"`
		}{
			IgnoreKeywords: []string{"WIP", "DRAFT"},
			StaleAfterDays: 7,
		},
		Notifiers: struct {
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
		}{
			SMTP: struct {
				Host     string   `yaml:"host"`
				Port     int      `yaml:"port"`
				User     string   `yaml:"user"`
				Password string   `yaml:"password"`
				From     string   `yaml:"from"`
				To       []string `yaml:"to"`
			}{
				Host:     "smtp.gmail.com",
				Port:     587,
				User:     "test@example.com",
				Password: "test-password",
				From:     "test@example.com",
				To:       []string{"admin@example.com"},
			},
		},
		Notification: struct {
			IntervalHours int `yaml:"interval_hours"`
		}{
			IntervalHours: 24,
		},
	}

	// Create mock Bitbucket server
	server, mockClient := createMockBitbucketServer()
	defer server.Close()

	// Create context and cancel it immediately
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Run the service - should return immediately due to cancelled context
	err := runWithMock(ctx, cfg, mockClient)
	if err != nil {
		t.Errorf("Expected no error when context is cancelled, got: %v", err)
	}
}

func TestFileNotificationStateStore_Integration(t *testing.T) {
	// Test the state store functionality
	stateStore := &models.FileNotificationStateStore{Path: "test_last_notification.txt"}
	defer func() {
		// Clean up test file
		os.Remove("test_last_notification.txt")
	}()

	// Test initial state (should be zero time)
	lastTime, err := stateStore.GetLastNotificationTime()
	if err != nil {
		t.Errorf("Expected no error getting initial notification time, got: %v", err)
	}
	if !lastTime.IsZero() {
		t.Error("Expected initial notification time to be zero")
	}

	// Test setting notification time
	now := time.Now()
	err = stateStore.SetLastNotificationTime(now)
	if err != nil {
		t.Errorf("Expected no error setting notification time, got: %v", err)
	}

	// Test getting the set time
	retrievedTime, err := stateStore.GetLastNotificationTime()
	if err != nil {
		t.Errorf("Expected no error getting set notification time, got: %v", err)
	}
	if retrievedTime.Unix() != now.Unix() {
		t.Errorf("Expected retrieved time to match set time, got %v vs %v", retrievedTime, now)
	}
}
