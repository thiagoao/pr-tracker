package bitbucket

import (
	"encoding/json"
	"fc-pr-tracker/internal/config"
	"fc-pr-tracker/pkg/models"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	cfg := &config.Config{}
	client := NewClient(cfg)

	if client == nil {
		t.Error("Expected client to be created, got nil")
	}
	if client.Config != cfg {
		t.Error("Expected config to be set correctly")
	}
	if client.Client == nil {
		t.Error("Expected HTTP client to be created")
	}
}

func TestClient_basicAuth(t *testing.T) {
	cfg := &config.Config{
		Bitbucket: struct {
			Domain       string   `yaml:"domain"`
			Port         int      `yaml:"port"`
			Workspace    string   `yaml:"workspace"`
			User         string   `yaml:"user"`
			AppPassword  string   `yaml:"app_password"`
			Repositories []string `yaml:"repositories"`
		}{
			User:        "testuser",
			AppPassword: "testpass",
		},
	}

	client := &Client{Config: cfg}
	auth := client.basicAuth()

	expected := "dGVzdHVzZXI6dGVzdHBhc3M=" // base64 of "testuser:testpass"
	if auth != expected {
		t.Errorf("Expected auth '%s', got '%s'", expected, auth)
	}
}

func TestFilterPRs(t *testing.T) {
	prs := []models.PullRequest{
		{Title: "Normal PR"},
		{Title: "WIP: Work in progress"},
		{Title: "DRAFT: Another draft"},
		{Title: "Another normal PR"},
		{Title: "wip: lowercase"},
		{Title: "draft: lowercase draft"},
	}

	ignoreKeywords := []string{"WIP", "DRAFT"}

	filtered := FilterPRs(prs, ignoreKeywords)

	expectedCount := 2 // Only "Normal PR" and "Another normal PR" should remain
	if len(filtered) != expectedCount {
		t.Errorf("Expected %d PRs after filtering, got %d", expectedCount, len(filtered))
	}

	// Check that filtered PRs don't contain ignored keywords
	for _, pr := range filtered {
		if containsIgnoreKeyword(pr.Title, ignoreKeywords) {
			t.Errorf("Filtered PR should not contain ignored keywords: %s", pr.Title)
		}
	}
}

func TestFilterPRs_EmptyKeywords(t *testing.T) {
	prs := []models.PullRequest{
		{Title: "Normal PR"},
		{Title: "WIP: Work in progress"},
		{Title: "DRAFT: Another draft"},
	}

	ignoreKeywords := []string{}

	filtered := FilterPRs(prs, ignoreKeywords)

	// Should return all PRs when no keywords to ignore
	if len(filtered) != len(prs) {
		t.Errorf("Expected %d PRs when no keywords to ignore, got %d", len(prs), len(filtered))
	}
}

func TestFilterPRs_NoPRs(t *testing.T) {
	prs := []models.PullRequest{}
	ignoreKeywords := []string{"WIP", "DRAFT"}

	filtered := FilterPRs(prs, ignoreKeywords)

	if len(filtered) != 0 {
		t.Errorf("Expected 0 PRs when input is empty, got %d", len(filtered))
	}
}

func TestContainsIgnoreKeyword(t *testing.T) {
	tests := []struct {
		name     string
		title    string
		keywords []string
		expected bool
	}{
		{
			name:     "Title contains WIP",
			title:    "WIP: Work in progress",
			keywords: []string{"WIP", "DRAFT"},
			expected: true,
		},
		{
			name:     "Title contains DRAFT",
			title:    "DRAFT: Another draft",
			keywords: []string{"WIP", "DRAFT"},
			expected: true,
		},
		{
			name:     "Title contains lowercase wip",
			title:    "wip: lowercase",
			keywords: []string{"WIP", "DRAFT"},
			expected: true,
		},
		{
			name:     "Title does not contain keywords",
			title:    "Normal PR",
			keywords: []string{"WIP", "DRAFT"},
			expected: false,
		},
		{
			name:     "Empty keywords",
			title:    "WIP: Work in progress",
			keywords: []string{},
			expected: false,
		},
		{
			name:     "Empty title",
			title:    "",
			keywords: []string{"WIP", "DRAFT"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := containsIgnoreKeyword(tt.title, tt.keywords)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v for title '%s' with keywords %v", tt.expected, result, tt.title, tt.keywords)
			}
		})
	}
}

func TestIsPRApproved(t *testing.T) {
	tests := []struct {
		name         string
		participants []models.Participant
		expected     bool
	}{
		{
			name: "PR is approved",
			participants: []models.Participant{
				{Approved: true, Status: "APPROVED", Role: "REVIEWER"},
				{Approved: false, Status: "UNAPPROVED", Role: "AUTHOR"},
			},
			expected: true,
		},
		{
			name: "PR is not approved",
			participants: []models.Participant{
				{Approved: false, Status: "UNAPPROVED", Role: "REVIEWER"},
				{Approved: false, Status: "NEEDS_WORK", Role: "REVIEWER"},
			},
			expected: false,
		},
		{
			name:         "No participants",
			participants: []models.Participant{},
			expected:     true,
		},
		{
			name: "Multiple approvals",
			participants: []models.Participant{
				{Approved: true, Status: "APPROVED", Role: "REVIEWER"},
				{Approved: true, Status: "APPROVED", Role: "REVIEWER"},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsPRApproved(tt.participants)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestCountApprovals(t *testing.T) {
	tests := []struct {
		name             string
		participants     []models.Participant
		expectedApproved int
		expectedTotal    int
	}{
		{
			name: "No approvals",
			participants: []models.Participant{
				{Approved: false, Status: "UNAPPROVED", Role: "REVIEWER"},
				{Approved: false, Status: "NEEDS_WORK", Role: "REVIEWER"},
			},
			expectedApproved: 0,
			expectedTotal:    2,
		},
		{
			name: "One approval",
			participants: []models.Participant{
				{Approved: true, Status: "APPROVED", Role: "REVIEWER"},
				{Approved: false, Status: "UNAPPROVED", Role: "AUTHOR"},
			},
			expectedApproved: 1,
			expectedTotal:    1,
		},
		{
			name: "Multiple approvals",
			participants: []models.Participant{
				{Approved: true, Status: "APPROVED", Role: "REVIEWER"},
				{Approved: true, Status: "APPROVED", Role: "REVIEWER"},
				{Approved: false, Status: "UNAPPROVED", Role: "AUTHOR"},
			},
			expectedApproved: 2,
			expectedTotal:    2,
		},
		{
			name:             "No participants",
			participants:     []models.Participant{},
			expectedApproved: 0,
			expectedTotal:    0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			approved, total := CountApprovals(tt.participants)
			if approved != tt.expectedApproved {
				t.Errorf("Expected %d approved, got %d", tt.expectedApproved, approved)
			}
			if total != tt.expectedTotal {
				t.Errorf("Expected %d total, got %d", tt.expectedTotal, total)
			}
		})
	}
}

func TestGetLastActivity(t *testing.T) {
	now := time.Now()
	nowMillis := now.UnixMilli()

	pr := models.PullRequest{
		UpdatedDate: nowMillis,
	}

	comments := []models.Comment{
		{
			CreatedDate: now.Add(-2 * time.Hour).UnixMilli(),
			Content:     "Old comment",
		},
		{
			CreatedDate: now.Add(-1 * time.Hour).UnixMilli(),
			Content:     "Recent comment",
		},
	}

	// Test with comments
	lastActivity := GetLastActivity(pr, comments)
	if lastActivity == "" {
		t.Error("Expected last activity to be found, got empty string")
	}

	// Test without comments
	lastActivity = GetLastActivity(pr, []models.Comment{})
	if lastActivity == "" {
		t.Error("Expected last activity to be found from PR update date, got empty string")
	}

	// Test with PR that has no update date
	prNoUpdate := models.PullRequest{
		UpdatedDate: 0,
		CreatedDate: 0,
	}
	lastActivity = GetLastActivity(prNoUpdate, []models.Comment{})
	if lastActivity != "" {
		t.Errorf("Expected empty activity for PR with no update date, got '%s'", lastActivity)
	}
}

func newTestClient(handler http.HandlerFunc, cfg *config.Config) *Client {
	ts := httptest.NewServer(handler)
	c := &Client{Config: cfg, Client: ts.Client(), BaseURL: ts.URL}
	return c
}

func TestClient_TestConnection_Success(t *testing.T) {
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
	client := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte(`{"values":[]}`))
	}, cfg)
	if err := client.TestConnection(); err != nil {
		t.Errorf("Expected success, got error: %v", err)
	}
}

func TestClient_TestConnection_FailStatus(t *testing.T) {
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
	client := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(401)
		w.Write([]byte(`{"errors":[{"message":"Unauthorized"}]}`))
	}, cfg)
	err := client.TestConnection()
	if err == nil {
		t.Error("Expected error for unauthorized, got nil")
	}
}

func TestClient_TestConnection_BadRequest(t *testing.T) {
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
	client := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(400)
		w.Write([]byte(`{"errors":[{"message":"Bad Request"}]}`))
	}, cfg)
	err := client.TestConnection()
	if err == nil {
		t.Error("Expected error for bad request, got nil")
	}
}

func TestClient_ListOpenPRs_Success(t *testing.T) {
	cfg := &config.Config{
		Bitbucket: struct {
			Domain       string   `yaml:"domain"`
			Port         int      `yaml:"port"`
			Workspace    string   `yaml:"workspace"`
			User         string   `yaml:"user"`
			AppPassword  string   `yaml:"app_password"`
			Repositories []string `yaml:"repositories"`
		}{},
	}
	pr := models.PullRequest{ID: 1, Title: "Test PR"}
	resp := map[string]interface{}{"values": []models.PullRequest{pr}}
	body, _ := json.Marshal(resp)
	client := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write(body)
	}, cfg)
	prs, err := client.ListOpenPRs("repo1")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if len(prs) != 1 || prs[0].ID != 1 {
		t.Errorf("Expected 1 PR with ID 1, got %+v", prs)
	}
}

func TestClient_ListOpenPRs_Pagination(t *testing.T) {
	cfg := &config.Config{
		Bitbucket: struct {
			Domain       string   `yaml:"domain"`
			Port         int      `yaml:"port"`
			Workspace    string   `yaml:"workspace"`
			User         string   `yaml:"user"`
			AppPassword  string   `yaml:"app_password"`
			Repositories []string `yaml:"repositories"`
		}{},
	}
	pr1 := models.PullRequest{ID: 1, Title: "PR1"}
	pr2 := models.PullRequest{ID: 2, Title: "PR2"}
	page1 := map[string]interface{}{"values": []models.PullRequest{pr1}, "next": "/page2"}
	page2 := map[string]interface{}{"values": []models.PullRequest{pr2}}
	body1, _ := json.Marshal(page1)
	body2, _ := json.Marshal(page2)
	calls := 0
	client := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			w.WriteHeader(200)
			w.Write(body1)
		} else {
			w.WriteHeader(200)
			w.Write(body2)
		}
	}, cfg)
	prs, err := client.ListOpenPRs("repo1")
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if len(prs) != 2 {
		t.Errorf("Expected 2 PRs, got %+v", prs)
	}
}

func TestClient_ListOpenPRs_HTTPError(t *testing.T) {
	cfg := &config.Config{
		Bitbucket: struct {
			Domain       string   `yaml:"domain"`
			Port         int      `yaml:"port"`
			Workspace    string   `yaml:"workspace"`
			User         string   `yaml:"user"`
			AppPassword  string   `yaml:"app_password"`
			Repositories []string `yaml:"repositories"`
		}{},
	}
	client := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		w.Write([]byte(`{"error":"server error"}`))
	}, cfg)
	_, err := client.ListOpenPRs("repo1")
	if err == nil {
		t.Error("Expected error for HTTP 500, got nil")
	}
}

func TestClient_ListOpenPRs_BadJSON(t *testing.T) {
	cfg := &config.Config{
		Bitbucket: struct {
			Domain       string   `yaml:"domain"`
			Port         int      `yaml:"port"`
			Workspace    string   `yaml:"workspace"`
			User         string   `yaml:"user"`
			AppPassword  string   `yaml:"app_password"`
			Repositories []string `yaml:"repositories"`
		}{},
	}
	client := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("not json"))
	}, cfg)
	_, err := client.ListOpenPRs("repo1")
	if err == nil {
		t.Error("Expected error for bad JSON, got nil")
	}
}

func TestClient_GetParticipants_Success(t *testing.T) {
	cfg := &config.Config{
		Bitbucket: struct {
			Domain       string   `yaml:"domain"`
			Port         int      `yaml:"port"`
			Workspace    string   `yaml:"workspace"`
			User         string   `yaml:"user"`
			AppPassword  string   `yaml:"app_password"`
			Repositories []string `yaml:"repositories"`
		}{},
	}
	p := models.Participant{Role: "REVIEWER", Approved: true}
	resp := map[string]interface{}{"values": []models.Participant{p}}
	body, _ := json.Marshal(resp)
	client := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write(body)
	}, cfg)
	ps, err := client.GetParticipants("repo1", 1)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if len(ps) != 1 || ps[0].Role != "REVIEWER" {
		t.Errorf("Expected 1 participant with role REVIEWER, got %+v", ps)
	}
}

func TestClient_GetParticipants_HTTPError(t *testing.T) {
	cfg := &config.Config{
		Bitbucket: struct {
			Domain       string   `yaml:"domain"`
			Port         int      `yaml:"port"`
			Workspace    string   `yaml:"workspace"`
			User         string   `yaml:"user"`
			AppPassword  string   `yaml:"app_password"`
			Repositories []string `yaml:"repositories"`
		}{},
	}
	client := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		w.Write([]byte(`{"error":"not found"}`))
	}, cfg)
	_, err := client.GetParticipants("repo1", 1)
	if err == nil {
		t.Error("Expected error for HTTP 404, got nil")
	}
}

func TestClient_GetParticipants_BadJSON(t *testing.T) {
	cfg := &config.Config{
		Bitbucket: struct {
			Domain       string   `yaml:"domain"`
			Port         int      `yaml:"port"`
			Workspace    string   `yaml:"workspace"`
			User         string   `yaml:"user"`
			AppPassword  string   `yaml:"app_password"`
			Repositories []string `yaml:"repositories"`
		}{},
	}
	client := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("not json"))
	}, cfg)
	_, err := client.GetParticipants("repo1", 1)
	if err == nil {
		t.Error("Expected error for bad JSON, got nil")
	}
}

func TestClient_GetComments_Success(t *testing.T) {
	cfg := &config.Config{
		Bitbucket: struct {
			Domain       string   `yaml:"domain"`
			Port         int      `yaml:"port"`
			Workspace    string   `yaml:"workspace"`
			User         string   `yaml:"user"`
			AppPassword  string   `yaml:"app_password"`
			Repositories []string `yaml:"repositories"`
		}{},
	}
	c := models.Comment{ID: 1, Content: "Test comment"}
	resp := map[string]interface{}{"values": []models.Comment{c}}
	body, _ := json.Marshal(resp)
	client := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write(body)
	}, cfg)
	cs, err := client.GetComments("repo1", 1)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}
	if len(cs) != 1 || cs[0].ID != 1 {
		t.Errorf("Expected 1 comment with ID 1, got %+v", cs)
	}
}

func TestClient_GetComments_HTTPError(t *testing.T) {
	cfg := &config.Config{
		Bitbucket: struct {
			Domain       string   `yaml:"domain"`
			Port         int      `yaml:"port"`
			Workspace    string   `yaml:"workspace"`
			User         string   `yaml:"user"`
			AppPassword  string   `yaml:"app_password"`
			Repositories []string `yaml:"repositories"`
		}{},
	}
	client := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(500)
		w.Write([]byte(`{"error":"server error"}`))
	}, cfg)
	_, err := client.GetComments("repo1", 1)
	if err == nil {
		t.Error("Expected error for HTTP 500, got nil")
	}
}

func TestClient_GetComments_BadJSON(t *testing.T) {
	cfg := &config.Config{
		Bitbucket: struct {
			Domain       string   `yaml:"domain"`
			Port         int      `yaml:"port"`
			Workspace    string   `yaml:"workspace"`
			User         string   `yaml:"user"`
			AppPassword  string   `yaml:"app_password"`
			Repositories []string `yaml:"repositories"`
		}{},
	}
	client := newTestClient(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		w.Write([]byte("not json"))
	}, cfg)
	_, err := client.GetComments("repo1", 1)
	if err == nil {
		t.Error("Expected error for bad JSON, got nil")
	}
}
