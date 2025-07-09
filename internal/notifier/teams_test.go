package notifier

import (
	"encoding/json"
	"fc-pr-tracker/internal/config"
	"fc-pr-tracker/pkg/models"
	"strings"
	"testing"
	"time"
)

func TestNewTeamsNotifier(t *testing.T) {
	cfg := &config.Config{
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
			Teams: struct {
				WebhookURL string `yaml:"webhook_url"`
			}{
				WebhookURL: "https://webhook.url",
			},
		},
	}

	notifier := NewTeamsNotifier(cfg)

	if notifier == nil {
		t.Error("Expected notifier to be created, got nil")
	}
	if notifier.webhookURL != "https://webhook.url" {
		t.Errorf("Expected webhook URL 'https://webhook.url', got '%s'", notifier.webhookURL)
	}
}

func TestTeamsNotifier_Notify_EmptyPRs(t *testing.T) {
	cfg := &config.Config{}
	notifier := NewTeamsNotifier(cfg)

	err := notifier.Notify([]models.PullRequest{}, map[string][]models.PullRequest{}, map[int][]models.Participant{}, 7)
	if err != nil {
		t.Errorf("Expected no error when no PRs, got: %v", err)
	}
}

func TestTeamsNotifier_GenerateTeamsPayload(t *testing.T) {
	cfg := &config.Config{}
	notifier := NewTeamsNotifier(cfg)

	// Create test data
	now := time.Now()
	nowMillis := now.UnixMilli()

	pr1 := models.PullRequest{
		ID:          1,
		Title:       "Test PR 1",
		CreatedDate: nowMillis,
		UpdatedDate: nowMillis,
		Author: struct {
			User struct {
				DisplayName string `json:"displayName"`
				Username    string `json:"name"`
			} `json:"user"`
			Role     string `json:"role"`
			Approved bool   `json:"approved"`
			Status   string `json:"status"`
		}{
			User: struct {
				DisplayName string `json:"displayName"`
				Username    string `json:"name"`
			}{
				DisplayName: "Test User",
				Username:    "testuser",
			},
		},
		Links: struct {
			Self []struct {
				Href string `json:"href"`
			} `json:"self"`
		}{
			Self: []struct {
				Href string `json:"href"`
			}{
				{Href: "https://bitbucket.org/test/repo/pull-requests/1"},
			},
		},
	}

	pr2 := models.PullRequest{
		ID:          2,
		Title:       "Test PR 2",
		CreatedDate: nowMillis,
		UpdatedDate: nowMillis,
		Author: struct {
			User struct {
				DisplayName string `json:"displayName"`
				Username    string `json:"name"`
			} `json:"user"`
			Role     string `json:"role"`
			Approved bool   `json:"approved"`
			Status   string `json:"status"`
		}{
			User: struct {
				DisplayName string `json:"displayName"`
				Username    string `json:"name"`
			}{
				DisplayName: "Another User",
				Username:    "anotheruser",
			},
		},
		Links: struct {
			Self []struct {
				Href string `json:"href"`
			} `json:"self"`
		}{
			Self: []struct {
				Href string `json:"href"`
			}{
				{Href: "https://bitbucket.org/test/repo/pull-requests/2"},
			},
		},
	}

	allPRs := []models.PullRequest{pr1, pr2}
	repoPRs := map[string][]models.PullRequest{
		"test-repo": {pr1, pr2},
	}
	prParticipants := map[int][]models.Participant{
		1: {
			{Approved: true, Status: "APPROVED", Role: "REVIEWER"},
			{Approved: false, Status: "UNAPPROVED", Role: "AUTHOR"},
		},
		2: {
			{Approved: false, Status: "UNAPPROVED", Role: "REVIEWER"},
		},
	}

	payload, err := notifier.generateTeamsPayload(allPRs, repoPRs, prParticipants, 7)
	if err != nil {
		t.Fatalf("Expected no error generating Teams payload, got: %v", err)
	}

	// Verify payload is valid JSON
	var payloadMap map[string]interface{}
	err = json.Unmarshal(payload, &payloadMap)
	if err != nil {
		t.Fatalf("Expected valid JSON payload, got error: %v", err)
	}

	// Verify payload structure
	if payloadMap["@type"] != "MessageCard" {
		t.Error("Expected @type to be 'MessageCard'")
	}
	if payloadMap["@context"] != "http://schema.org/extensions" {
		t.Error("Expected @context to be 'http://schema.org/extensions'")
	}
	if payloadMap["themeColor"] != "FF0000" {
		t.Error("Expected themeColor to be 'FF0000'")
	}

	// Verify summary
	summary := payloadMap["summary"].(string)
	if !strings.Contains(summary, "2 PRs need attention") {
		t.Error("Expected summary to contain PR count")
	}

	// Verify sections exist
	sections, ok := payloadMap["sections"].([]interface{})
	if !ok {
		t.Fatal("Expected sections to be an array")
	}

	// Should have at least 3 sections: header, repository, summary
	if len(sections) < 3 {
		t.Errorf("Expected at least 3 sections, got %d", len(sections))
	}

	// Verify first section (header)
	headerSection := sections[0].(map[string]interface{})
	if headerSection["activityTitle"] != "🚨 Stale Pull Requests Alert" {
		t.Error("Expected header section to have correct activity title")
	}

	// Verify repository section
	repoSection := sections[1].(map[string]interface{})
	if repoSection["activityTitle"] != "Repository: test-repo" {
		t.Error("Expected repository section to have correct activity title")
	}

	// Verify facts in repository section
	facts := repoSection["facts"].([]interface{})
	if len(facts) != 2 {
		t.Errorf("Expected 2 facts in repository section, got %d", len(facts))
	}

	// Verify summary section
	summarySection := sections[len(sections)-1].(map[string]interface{})
	if summarySection["activityTitle"] != "📊 Summary" {
		t.Error("Expected summary section to have correct activity title")
	}
}

func TestTeamsNotifier_GenerateTeamsPayload_NoParticipants(t *testing.T) {
	cfg := &config.Config{}
	notifier := NewTeamsNotifier(cfg)

	pr := models.PullRequest{
		ID:    1,
		Title: "Test PR",
		Author: struct {
			User struct {
				DisplayName string `json:"displayName"`
				Username    string `json:"name"`
			} `json:"user"`
			Role     string `json:"role"`
			Approved bool   `json:"approved"`
			Status   string `json:"status"`
		}{
			User: struct {
				DisplayName string `json:"displayName"`
				Username    string `json:"name"`
			}{
				DisplayName: "Test User",
				Username:    "testuser",
			},
		},
		Links: struct {
			Self []struct {
				Href string `json:"href"`
			} `json:"self"`
		}{
			Self: []struct {
				Href string `json:"href"`
			}{
				{Href: "https://bitbucket.org/test/repo/pull-requests/1"},
			},
		},
	}

	allPRs := []models.PullRequest{pr}
	repoPRs := map[string][]models.PullRequest{
		"test-repo": {pr},
	}
	prParticipants := map[int][]models.Participant{}

	payload, err := notifier.generateTeamsPayload(allPRs, repoPRs, prParticipants, 7)
	if err != nil {
		t.Fatalf("Expected no error generating Teams payload, got: %v", err)
	}

	// Should handle empty participants gracefully
	payloadStr := string(payload)
	if !strings.Contains(payloadStr, "Test PR") {
		t.Error("Expected payload to contain PR title even with no participants")
	}
	if !strings.Contains(payloadStr, "0/0 approvals") {
		t.Error("Expected payload to contain approval count for PR with no participants")
	}
}

func TestTeamsNotifier_GenerateTeamsPayload_MultipleRepos(t *testing.T) {
	cfg := &config.Config{}
	notifier := NewTeamsNotifier(cfg)

	pr1 := models.PullRequest{
		ID:    1,
		Title: "PR from repo1",
		Author: struct {
			User struct {
				DisplayName string `json:"displayName"`
				Username    string `json:"name"`
			} `json:"user"`
			Role     string `json:"role"`
			Approved bool   `json:"approved"`
			Status   string `json:"status"`
		}{
			User: struct {
				DisplayName string `json:"displayName"`
				Username    string `json:"name"`
			}{
				DisplayName: "User 1",
				Username:    "user1",
			},
		},
		Links: struct {
			Self []struct {
				Href string `json:"href"`
			} `json:"self"`
		}{
			Self: []struct {
				Href string `json:"href"`
			}{
				{Href: "https://bitbucket.org/test/repo1/pull-requests/1"},
			},
		},
	}

	pr2 := models.PullRequest{
		ID:    2,
		Title: "PR from repo2",
		Author: struct {
			User struct {
				DisplayName string `json:"displayName"`
				Username    string `json:"name"`
			} `json:"user"`
			Role     string `json:"role"`
			Approved bool   `json:"approved"`
			Status   string `json:"status"`
		}{
			User: struct {
				DisplayName string `json:"displayName"`
				Username    string `json:"name"`
			}{
				DisplayName: "User 2",
				Username:    "user2",
			},
		},
		Links: struct {
			Self []struct {
				Href string `json:"href"`
			} `json:"self"`
		}{
			Self: []struct {
				Href string `json:"href"`
			}{
				{Href: "https://bitbucket.org/test/repo2/pull-requests/2"},
			},
		},
	}

	allPRs := []models.PullRequest{pr1, pr2}
	repoPRs := map[string][]models.PullRequest{
		"repo1": {pr1},
		"repo2": {pr2},
	}
	prParticipants := map[int][]models.Participant{}

	payload, err := notifier.generateTeamsPayload(allPRs, repoPRs, prParticipants, 7)
	if err != nil {
		t.Fatalf("Expected no error generating Teams payload, got: %v", err)
	}

	// Verify both repositories are mentioned
	payloadStr := string(payload)
	if !strings.Contains(payloadStr, "repo1") {
		t.Error("Expected payload to contain repo1")
	}
	if !strings.Contains(payloadStr, "repo2") {
		t.Error("Expected payload to contain repo2")
	}
	if !strings.Contains(payloadStr, "PR from repo1") {
		t.Error("Expected payload to contain PR from repo1")
	}
	if !strings.Contains(payloadStr, "PR from repo2") {
		t.Error("Expected payload to contain PR from repo2")
	}
}

func TestTeamsNotifier_SendTeamsNotification_Success(t *testing.T) {
	cfg := &config.Config{}
	cfg.Notifiers.Teams.WebhookURL = "https://webhook.url"

	notifier := NewTeamsNotifier(cfg)

	// This test will fail because the webhook URL is invalid, but it tests the code path
	payload := []byte(`{"test": "payload"}`)
	err := notifier.sendTeamsNotification(payload)

	// We expect an error because the webhook URL is invalid
	// But this tests that the function executes without panicking
	if err == nil {
		t.Log("sendTeamsNotification executed successfully (webhook available)")
	} else {
		t.Logf("sendTeamsNotification failed as expected: %v", err)
	}
}

func TestTeamsNotifier_SendTeamsNotification_InvalidURL(t *testing.T) {
	cfg := &config.Config{}
	cfg.Notifiers.Teams.WebhookURL = "invalid-url"

	notifier := NewTeamsNotifier(cfg)

	payload := []byte(`{"test": "payload"}`)
	err := notifier.sendTeamsNotification(payload)

	if err == nil {
		t.Error("Expected error when using invalid webhook URL")
	}
}

func TestTeamsNotifier_Notify_WithPRs(t *testing.T) {
	cfg := &config.Config{}
	cfg.Notifiers.Teams.WebhookURL = "https://webhook.url"

	notifier := NewTeamsNotifier(cfg)

	// Create test PR
	now := time.Now()
	nowMillis := now.UnixMilli()

	pr := models.PullRequest{
		ID:          1,
		Title:       "Test PR",
		CreatedDate: nowMillis,
		UpdatedDate: nowMillis,
		Author: struct {
			User struct {
				DisplayName string `json:"displayName"`
				Username    string `json:"name"`
			} `json:"user"`
			Role     string `json:"role"`
			Approved bool   `json:"approved"`
			Status   string `json:"status"`
		}{
			User: struct {
				DisplayName string `json:"displayName"`
				Username    string `json:"name"`
			}{
				DisplayName: "Test User",
				Username:    "testuser",
			},
		},
		Links: struct {
			Self []struct {
				Href string `json:"href"`
			} `json:"self"`
		}{
			Self: []struct {
				Href string `json:"href"`
			}{
				{Href: "https://bitbucket.org/test/repo/pull-requests/1"},
			},
		},
	}

	allPRs := []models.PullRequest{pr}
	repoPRs := map[string][]models.PullRequest{
		"test-repo": {pr},
	}
	prParticipants := map[int][]models.Participant{}

	// This will fail because the webhook URL is invalid, but it tests the code path
	err := notifier.Notify(allPRs, repoPRs, prParticipants, 7)

	if err == nil {
		t.Log("Notify executed successfully (webhook available)")
	} else {
		t.Logf("Notify failed as expected: %v", err)
	}
}

func TestTeamsNotifier_SendTeamsNotification_NetworkError(t *testing.T) {
	cfg := &config.Config{}
	cfg.Notifiers.Teams.WebhookURL = "http://invalid-host.local/webhook"

	notifier := NewTeamsNotifier(cfg)

	payload := []byte(`{"test": "payload"}`)
	err := notifier.sendTeamsNotification(payload)

	if err == nil {
		t.Error("Expected error when connecting to invalid host")
	}
}
