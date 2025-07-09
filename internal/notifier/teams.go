package notifier

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	"fc-pr-tracker/internal/bitbucket"
	"fc-pr-tracker/internal/config"
	"fc-pr-tracker/pkg/models"
)

// TeamsNotifier implements Microsoft Teams notifications
type TeamsNotifier struct {
	webhookURL string
}

// NewTeamsNotifier creates a new Teams notifier
func NewTeamsNotifier(cfg *config.Config) *TeamsNotifier {
	return &TeamsNotifier{webhookURL: cfg.Notifiers.Teams.WebhookURL}
}

// Notify sends Teams notifications for stale PRs
func (t *TeamsNotifier) Notify(allPRs []models.PullRequest, repoPRs map[string][]models.PullRequest,
	prParticipants map[int][]models.Participant, staleAfterDays int) error {

	if len(allPRs) == 0 {
		return nil
	}

	payload, err := t.generateTeamsPayload(allPRs, repoPRs, prParticipants, staleAfterDays)
	if err != nil {
		return fmt.Errorf("error generating Teams payload: %v", err)
	}

	return t.sendTeamsNotification(payload)
}

// generateTeamsPayload creates the Teams message payload
func (t *TeamsNotifier) generateTeamsPayload(allPRs []models.PullRequest, repoPRs map[string][]models.PullRequest,
	prParticipants map[int][]models.Participant, staleAfterDays int) ([]byte, error) {

	var sections []map[string]interface{}

	for repo, prs := range repoPRs {
		var facts []map[string]interface{}
		for _, pr := range prs {
			// Calculate approval count for this PR
			participants := prParticipants[pr.ID]
			approved, total := bitbucket.CountApprovals(participants)

			facts = append(facts, map[string]interface{}{
				"name": fmt.Sprintf("PR #%d", pr.ID),
				"value": fmt.Sprintf("[%s](%s) by %s (%d/%d approvals)",
					pr.Title, pr.Links.Self[0].Href, pr.Author.User.DisplayName, approved, total),
			})
		}

		sections = append(sections, map[string]interface{}{
			"activityTitle": fmt.Sprintf("Repository: %s", repo),
			"facts":         facts,
		})
	}

	payload := map[string]interface{}{
		"@type":      "MessageCard",
		"@context":   "http://schema.org/extensions",
		"themeColor": "FF0000",
		"summary":    fmt.Sprintf("Stale Pull Requests Alert - %d PRs need attention", len(allPRs)),
		"sections": append([]map[string]interface{}{
			{
				"activityTitle":    "🚨 Stale Pull Requests Alert",
				"activitySubtitle": fmt.Sprintf("%d pull requests have been inactive for %d days or more", len(allPRs), staleAfterDays),
				"text":             "The following pull requests need attention:",
			},
		}, append(sections, map[string]interface{}{
			"activityTitle": "📊 Summary",
			"facts": []map[string]interface{}{
				{
					"name":  "Total Stale PRs",
					"value": fmt.Sprintf("%d", len(allPRs)),
				},
				{
					"name":  "Stale Threshold",
					"value": fmt.Sprintf("%d days", staleAfterDays),
				},
			},
		})...),
	}

	return json.Marshal(payload)
}

// sendTeamsNotification sends the notification to Microsoft Teams
func (t *TeamsNotifier) sendTeamsNotification(payload []byte) error {
	resp, err := http.Post(t.webhookURL, "application/json", bytes.NewBuffer(payload))
	if err != nil {
		slog.Error("Failed to send Teams notification", "error", err)
		return fmt.Errorf("failed to send Teams notification: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		slog.Error("Teams notification failed", "status", resp.StatusCode)
		return fmt.Errorf("Teams notification failed with status: %d", resp.StatusCode)
	}

	slog.Info("Teams notification sent successfully")
	return nil
}
