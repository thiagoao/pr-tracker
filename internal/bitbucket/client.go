package bitbucket

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"fc-pr-tracker/internal/config"
	"fc-pr-tracker/pkg/models"
)

// Client represents a Bitbucket API client
type Client struct {
	Config  *config.Config
	Client  *http.Client
	BaseURL string // para testes
}

// NewClient creates a new Bitbucket client
func NewClient(cfg *config.Config) *Client {
	return &Client{
		Config: cfg,
		Client: &http.Client{Timeout: 15 * time.Second},
	}
}

// TestConnection checks if the Bitbucket API is reachable and credentials are valid
func (c *Client) TestConnection() error {
	var url string
	if c.BaseURL != "" {
		url = c.BaseURL + "/rest/api/1.0/projects/" + c.Config.Bitbucket.Workspace + "/repos?limit=1"
	} else {
		url = fmt.Sprintf("https://%s:%d/rest/api/1.0/projects/%s/repos?limit=1",
			c.Config.Bitbucket.Domain,
			c.Config.Bitbucket.Port,
			c.Config.Bitbucket.Workspace)
	}

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return fmt.Errorf("error creating test request: %v", err)
	}

	req.Header.Set("Authorization", "Basic "+c.basicAuth())
	slog.Debug("Basic Auth header set for test connection")

	resp, err := c.Client.Do(req)
	if err != nil {
		return fmt.Errorf("error connecting to Bitbucket: %v", err)
	}
	defer resp.Body.Close()

	body, _ := ioutil.ReadAll(resp.Body)
	if resp.StatusCode != 200 {
		return fmt.Errorf("Bitbucket connection test failed: %s (URL: %s, Body: %s)", resp.Status, url, string(body))
	}
	return nil
}

// ListOpenPRs fetches open PRs for a repository
func (c *Client) ListOpenPRs(repo string) ([]models.PullRequest, error) {
	var prs []models.PullRequest
	var baseURL string
	if c.BaseURL != "" {
		baseURL = c.BaseURL + "/rest/api/1.0/projects/" + c.Config.Bitbucket.Workspace + "/repos/" + repo + "/pull-requests?state=OPEN"
	} else {
		baseURL = fmt.Sprintf(
			"https://%s:%d/rest/api/1.0/projects/%s/repos/%s/pull-requests?state=OPEN",
			c.Config.Bitbucket.Domain,
			c.Config.Bitbucket.Port,
			c.Config.Bitbucket.Workspace,
			repo,
		)
	}
	url := baseURL

	headers := map[string]string{
		"Authorization": "Basic " + c.basicAuth(),
		"Content-Type":  "application/json",
	}

	for url != "" {
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, err
		}
		for k, v := range headers {
			req.Header.Set(k, v)
		}

		resp, err := c.Client.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		body, _ := ioutil.ReadAll(resp.Body)
		if resp.StatusCode != 200 {
			return nil, fmt.Errorf("error fetching PRs: %s (URL: %s, Body: %s)", resp.Status, url, string(body))
		}

		var prResp PRListResponse
		if err := json.Unmarshal(body, &prResp); err != nil {
			return nil, err
		}
		prs = append(prs, prResp.Values...)
		// Corrigir paginação: se prResp.Next for relativo, concatenar com BaseURL
		if prResp.Next != "" && strings.HasPrefix(prResp.Next, "/") {
			if c.BaseURL != "" {
				url = c.BaseURL + prResp.Next
			} else {
				url = fmt.Sprintf(
					"https://%s:%d%s",
					c.Config.Bitbucket.Domain,
					c.Config.Bitbucket.Port,
					prResp.Next,
				)
			}
		} else {
			url = prResp.Next
		}
	}
	return prs, nil
}

// GetParticipants fetches PR participants (reviewers)
func (c *Client) GetParticipants(repo string, prID int) ([]models.Participant, error) {
	var url string
	if c.BaseURL != "" {
		url = fmt.Sprintf("%s/rest/api/1.0/projects/%s/repos/%s/pull-requests/%d/participants", c.BaseURL, c.Config.Bitbucket.Workspace, repo, prID)
	} else {
		url = fmt.Sprintf(
			"https://%s:%d/rest/api/1.0/projects/%s/repos/%s/pull-requests/%d/participants",
			c.Config.Bitbucket.Domain,
			c.Config.Bitbucket.Port,
			c.Config.Bitbucket.Workspace,
			repo,
			prID,
		)
	}
	slog.Info("Fetching participants for PR", "pr_id", prID, "repo", repo, "url", url)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Basic "+c.basicAuth())
	slog.Debug("Basic Auth header set for participants fetch")

	resp, err := c.Client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := ioutil.ReadAll(resp.Body)
		slog.Error("Error fetching participants", "status", resp.Status, "url", url, "body", string(body))
		return nil, fmt.Errorf("error fetching participants: %s (URL: %s, Body: %s)", resp.Status, url, string(body))
	}

	body, _ := ioutil.ReadAll(resp.Body)
	var pResp ParticipantsResponse
	if err := json.Unmarshal(body, &pResp); err != nil {
		return nil, err
	}
	return pResp.Values, nil
}

// GetComments fetches all comments/activities for a PR
func (c *Client) GetComments(repo string, prID int) ([]models.Comment, error) {
	var url string
	var comments []models.Comment
	if c.BaseURL != "" {
		url = fmt.Sprintf("%s/rest/api/1.0/projects/%s/repos/%s/pull-requests/%d/activities", c.BaseURL, c.Config.Bitbucket.Workspace, repo, prID)
	} else {
		url = fmt.Sprintf(
			"https://%s:%d/rest/api/1.0/projects/%s/repos/%s/pull-requests/%d/activities",
			c.Config.Bitbucket.Domain,
			c.Config.Bitbucket.Port,
			c.Config.Bitbucket.Workspace,
			repo,
			prID,
		)
	}
	slog.Info("Fetching comments/activities for PR", "pr_id", prID, "repo", repo, "url", url)

	for url != "" {
		req, err := http.NewRequest("GET", url, nil)
		if err != nil {
			return nil, err
		}
		req.Header.Set("Authorization", "Basic "+c.basicAuth())
		slog.Debug("Basic Auth header set for comments/activities fetch")

		resp, err := c.Client.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()

		if resp.StatusCode != 200 {
			body, _ := ioutil.ReadAll(resp.Body)
			slog.Error("Error fetching comments/activities", "status", resp.Status, "url", url, "body", string(body))
			return nil, fmt.Errorf("error fetching comments/activities: %s (URL: %s, Body: %s)", resp.Status, url, string(body))
		}

		body, _ := ioutil.ReadAll(resp.Body)
		var cResp CommentsResponse
		if err := json.Unmarshal(body, &cResp); err != nil {
			return nil, err
		}
		comments = append(comments, cResp.Values...)
		url = cResp.Next
	}
	return comments, nil
}

// Helper methods
func (c *Client) basicAuth() string {
	auth := c.Config.Bitbucket.User + ":" + c.Config.Bitbucket.AppPassword
	return base64.StdEncoding.EncodeToString([]byte(auth))
}

// FilterPRs filters PRs by ignored keywords
func FilterPRs(prs []models.PullRequest, ignoreKeywords []string) []models.PullRequest {
	var filtered []models.PullRequest
	for _, pr := range prs {
		if containsIgnoreKeyword(pr.Title, ignoreKeywords) {
			continue // Ignore PRs with forbidden keywords
		}
		filtered = append(filtered, pr)
	}
	return filtered
}

// containsIgnoreKeyword checks if the title contains any forbidden keyword
func containsIgnoreKeyword(title string, keywords []string) bool {
	titleLower := strings.ToLower(title)
	for _, kw := range keywords {
		if strings.Contains(titleLower, strings.ToLower(kw)) {
			return true
		}
	}
	return false
}

// IsPRApproved checks if PR is approved by all reviewers
func IsPRApproved(participants []models.Participant) bool {
	for _, p := range participants {
		if p.Role == "REVIEWER" && !p.Approved {
			return false
		}
	}
	return true
}

// CountApprovals counts the number of approved reviewers
func CountApprovals(participants []models.Participant) (approved, total int) {
	for _, p := range participants {
		if p.Role == "REVIEWER" {
			total++
			if p.Approved {
				approved++
			}
		}
	}
	return approved, total
}

// GetLastActivity returns the last activity date
func GetLastActivity(pr models.PullRequest, comments []models.Comment) string {
	// Se não houver datas válidas, retorna vazio
	if pr.UpdatedDate == 0 && pr.CreatedDate == 0 {
		return ""
	}
	// Convert millisecond timestamps to time.Time
	lastUpdated := time.UnixMilli(pr.UpdatedDate)
	lastCreated := time.UnixMilli(pr.CreatedDate)

	last := lastUpdated
	if last.IsZero() {
		last = lastCreated
	}

	for _, c := range comments {
		commentTime := time.UnixMilli(c.UpdatedDate)
		if commentTime.After(last) {
			last = commentTime
		}
	}

	if last.IsZero() {
		return ""
	}
	return last.Format(time.RFC3339)
}

// Response types for JSON unmarshaling
type PRListResponse struct {
	Values []models.PullRequest `json:"values"`
	Next   string               `json:"next"`
}

type ParticipantsResponse struct {
	Values []models.Participant `json:"values"`
}

type CommentsResponse struct {
	Values []models.Comment `json:"values"`
	Next   string           `json:"next"`
}
