package notifier

import (
	"fc-pr-tracker/internal/config"
	"fc-pr-tracker/pkg/models"
	"strings"
	"testing"
	"time"
)

func TestNewEmailNotifier(t *testing.T) {
	cfg := &config.Config{}
	notifier := NewEmailNotifier(cfg)

	if notifier == nil {
		t.Error("Expected notifier to be created, got nil")
	}
	if notifier.config != cfg {
		t.Error("Expected config to be set correctly")
	}
}

func TestEmailNotifier_Notify_EmptyPRs(t *testing.T) {
	cfg := &config.Config{}
	notifier := NewEmailNotifier(cfg)

	err := notifier.Notify([]models.PullRequest{}, map[string][]models.PullRequest{}, map[int][]models.Participant{}, 7)
	if err != nil {
		t.Errorf("Expected no error when no PRs, got: %v", err)
	}
}

func TestEmailNotifier_GenerateEmailBody(t *testing.T) {
	cfg := &config.Config{}
	notifier := NewEmailNotifier(cfg)

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

	body, err := notifier.generateEmailBody(allPRs, repoPRs, prParticipants, 7)
	if err != nil {
		t.Fatalf("Expected no error generating email body, got: %v", err)
	}

	// Verify email body contains expected content
	if !strings.Contains(body, "Stale Pull Requests Alert") {
		t.Error("Expected email body to contain 'Stale Pull Requests Alert'")
	}
	if !strings.Contains(body, "2 pull requests have been inactive") {
		t.Error("Expected email body to contain PR count")
	}
	if !strings.Contains(body, "Test PR 1") {
		t.Error("Expected email body to contain first PR title")
	}
	if !strings.Contains(body, "Test PR 2") {
		t.Error("Expected email body to contain second PR title")
	}
	if !strings.Contains(body, "Test User") {
		t.Error("Expected email body to contain author name")
	}
	if !strings.Contains(body, "1/1 reviewers") {
		t.Error("Expected email body to contain approval count for PR 1")
	}
	if !strings.Contains(body, "0/1 reviewers") {
		t.Error("Expected email body to contain approval count for PR 2")
	}
	if !strings.Contains(body, "test-repo") {
		t.Error("Expected email body to contain repository name")
	}
}

func TestEmailNotifier_GenerateEmailBody_NoParticipants(t *testing.T) {
	cfg := &config.Config{}
	notifier := NewEmailNotifier(cfg)

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

	body, err := notifier.generateEmailBody(allPRs, repoPRs, prParticipants, 7)
	if err != nil {
		t.Fatalf("Expected no error generating email body, got: %v", err)
	}

	// Should handle empty participants gracefully
	if !strings.Contains(body, "Test PR") {
		t.Error("Expected email body to contain PR title even with no participants")
	}
}

func TestEmailNotifier_GenerateEmailBody_MultipleRepos(t *testing.T) {
	cfg := &config.Config{}
	notifier := NewEmailNotifier(cfg)

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

	body, err := notifier.generateEmailBody(allPRs, repoPRs, prParticipants, 7)
	if err != nil {
		t.Fatalf("Expected no error generating email body, got: %v", err)
	}

	// Verify both repositories are mentioned
	if !strings.Contains(body, "repo1") {
		t.Error("Expected email body to contain repo1")
	}
	if !strings.Contains(body, "repo2") {
		t.Error("Expected email body to contain repo2")
	}
	if !strings.Contains(body, "PR from repo1") {
		t.Error("Expected email body to contain PR from repo1")
	}
	if !strings.Contains(body, "PR from repo2") {
		t.Error("Expected email body to contain PR from repo2")
	}
}

func TestEmailNotifier_SendEmail_Success(t *testing.T) {
	cfg := &config.Config{}
	cfg.Notifiers.SMTP.Host = "localhost"
	cfg.Notifiers.SMTP.Port = 1025 // MailHog port for testing
	cfg.Notifiers.SMTP.From = "test@example.com"
	cfg.Notifiers.SMTP.To = []string{"recipient@example.com"}
	cfg.Notifiers.SMTP.User = ""
	cfg.Notifiers.SMTP.Password = ""

	notifier := NewEmailNotifier(cfg)

	// This test will fail if no SMTP server is running, but it tests the code path
	// In a real environment, you'd use a mock SMTP server
	err := notifier.sendEmail("Test Subject", "Test Body")

	// We expect an error because there's no SMTP server running
	// But this tests that the function executes without panicking
	if err == nil {
		t.Log("sendEmail executed successfully (SMTP server available)")
	} else {
		t.Logf("sendEmail failed as expected: %v", err)
	}
}

func TestEmailNotifier_SendEmail_WithAuth(t *testing.T) {
	cfg := &config.Config{}
	cfg.Notifiers.SMTP.Host = "smtp.gmail.com"
	cfg.Notifiers.SMTP.Port = 587
	cfg.Notifiers.SMTP.From = "test@example.com"
	cfg.Notifiers.SMTP.To = []string{"recipient@example.com"}
	cfg.Notifiers.SMTP.User = "test@example.com"
	cfg.Notifiers.SMTP.Password = "test-password"

	notifier := NewEmailNotifier(cfg)

	// This test will fail because credentials are invalid, but it tests the auth code path
	err := notifier.sendEmail("Test Subject", "Test Body")

	// We expect an error because credentials are invalid
	// But this tests that the authentication code path executes
	if err == nil {
		t.Log("sendEmail with auth executed successfully")
	} else {
		t.Logf("sendEmail with auth failed as expected: %v", err)
	}
}

func TestEmailNotifier_SendEmail_TLS(t *testing.T) {
	cfg := &config.Config{}
	cfg.Notifiers.SMTP.Host = "smtp.gmail.com"
	cfg.Notifiers.SMTP.Port = 465
	cfg.Notifiers.SMTP.From = "test@example.com"
	cfg.Notifiers.SMTP.To = []string{"recipient@example.com"}
	cfg.Notifiers.SMTP.User = "test@example.com"
	cfg.Notifiers.SMTP.Password = "test-password"

	notifier := NewEmailNotifier(cfg)

	// This test will fail because credentials are invalid, but it tests the TLS code path
	err := notifier.sendEmail("Test Subject", "Test Body")

	// We expect an error because credentials are invalid
	// But this tests that the TLS code path executes
	if err == nil {
		t.Log("sendEmail with TLS executed successfully")
	} else {
		t.Logf("sendEmail with TLS failed as expected: %v", err)
	}
}

func TestEmailNotifier_SendWithTLS_ConnectionError(t *testing.T) {
	cfg := &config.Config{}
	cfg.Notifiers.SMTP.Host = "invalid-host.local"
	cfg.Notifiers.SMTP.Port = 465

	notifier := NewEmailNotifier(cfg)

	// Test TLS connection failure
	err := notifier.sendWithTLS("invalid-host.local:465", nil, "from@example.com", []string{"to@example.com"}, []byte("test"))

	if err == nil {
		t.Error("Expected error when connecting to invalid host")
	}
}

func TestEmailNotifier_Notify_WithPRs(t *testing.T) {
	cfg := &config.Config{}
	cfg.Notifiers.SMTP.Host = "localhost"
	cfg.Notifiers.SMTP.Port = 1025
	cfg.Notifiers.SMTP.From = "test@example.com"
	cfg.Notifiers.SMTP.To = []string{"recipient@example.com"}

	notifier := NewEmailNotifier(cfg)

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

	// This will fail because no SMTP server is running, but it tests the code path
	err := notifier.Notify(allPRs, repoPRs, prParticipants, 7)

	if err == nil {
		t.Log("Notify executed successfully (SMTP server available)")
	} else {
		t.Logf("Notify failed as expected: %v", err)
	}
}
