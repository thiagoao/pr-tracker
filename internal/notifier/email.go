package notifier

import (
	"crypto/tls"
	"fmt"
	"log/slog"
	"net/smtp"
	"strings"
	"text/template"

	"fc-pr-tracker/internal/bitbucket"
	"fc-pr-tracker/internal/config"
	"fc-pr-tracker/pkg/models"
)

// EmailNotifier implements email notifications
type EmailNotifier struct {
	config *config.Config
}

// NewEmailNotifier creates a new email notifier
func NewEmailNotifier(cfg *config.Config) *EmailNotifier {
	return &EmailNotifier{config: cfg}
}

// Notify sends email notifications for stale PRs
func (e *EmailNotifier) Notify(allPRs []models.PullRequest, repoPRs map[string][]models.PullRequest,
	prParticipants map[int][]models.Participant, staleAfterDays int) error {

	if len(allPRs) == 0 {
		return nil
	}

	subject := fmt.Sprintf("Stale Pull Requests Alert - %d PRs need attention", len(allPRs))
	body, err := e.generateEmailBody(allPRs, repoPRs, prParticipants, staleAfterDays)
	if err != nil {
		return fmt.Errorf("error generating email body: %v", err)
	}

	return e.sendEmail(subject, body)
}

// generateEmailBody creates the email content
func (e *EmailNotifier) generateEmailBody(allPRs []models.PullRequest, repoPRs map[string][]models.PullRequest,
	prParticipants map[int][]models.Participant, staleAfterDays int) (string, error) {

	tmpl := `
Stale Pull Requests Alert

The following {{.TotalPRs}} pull requests have been inactive for {{.StaleDays}} days or more:

{{range $repo, $prs := .RepoPRs}}
Repository: {{$repo}}
{{range $prs}}
- PR #{{.ID}}: {{.Title}}
  Author: {{.Author.User.DisplayName}} ({{.Author.User.Username}})
  Link: {{(index .Links.Self 0).Href}}
  Created: {{.CreatedDate}}
  Updated: {{.UpdatedDate}}
  Approvals: {{index $.ApprovalCounts .ID "approved"}}/{{index $.ApprovalCounts .ID "total"}} reviewers
{{end}}
{{end}}

Total stale PRs: {{.TotalPRs}}

This is an automated notification from the PR Tracker service.
`

	t := template.Must(template.New("email").Parse(tmpl))

	// Calculate approval counts for each PR
	approvalCounts := make(map[int]map[string]int)
	for prID, participants := range prParticipants {
		approved, total := bitbucket.CountApprovals(participants)
		approvalCounts[prID] = map[string]int{
			"approved": approved,
			"total":    total,
		}
	}

	data := struct {
		TotalPRs       int
		StaleDays      int
		RepoPRs        map[string][]models.PullRequest
		ApprovalCounts map[int]map[string]int
	}{
		TotalPRs:       len(allPRs),
		StaleDays:      staleAfterDays,
		RepoPRs:        repoPRs,
		ApprovalCounts: approvalCounts,
	}

	var body strings.Builder
	err := t.Execute(&body, data)
	if err != nil {
		return "", err
	}

	return body.String(), nil
}

// sendEmail sends the email using SMTP
func (e *EmailNotifier) sendEmail(subject, body string) error {
	to := strings.Join(e.config.Notifiers.SMTP.To, ",")
	msg := fmt.Sprintf("To: %s\r\nFrom: %s\r\nSubject: %s\r\n\r\n%s",
		to, e.config.Notifiers.SMTP.From, subject, body)

	addr := fmt.Sprintf("%s:%d", e.config.Notifiers.SMTP.Host, e.config.Notifiers.SMTP.Port)

	var err error

	// Determine authentication and connection method based on configuration
	if e.config.Notifiers.SMTP.User != "" && e.config.Notifiers.SMTP.Password != "" {
		// Use authentication
		auth := smtp.PlainAuth("", e.config.Notifiers.SMTP.User, e.config.Notifiers.SMTP.Password,
			e.config.Notifiers.SMTP.Host)

		if e.config.Notifiers.SMTP.Port == 587 {
			// Use STARTTLS for port 587
			err = smtp.SendMail(addr, auth, e.config.Notifiers.SMTP.From, e.config.Notifiers.SMTP.To, []byte(msg))
		} else if e.config.Notifiers.SMTP.Port == 465 {
			// Use TLS for port 465
			err = e.sendWithTLS(addr, auth, e.config.Notifiers.SMTP.From, e.config.Notifiers.SMTP.To, []byte(msg))
		} else {
			// For other ports (like 1025 for local testing), try without TLS first
			err = smtp.SendMail(addr, auth, e.config.Notifiers.SMTP.From, e.config.Notifiers.SMTP.To, []byte(msg))
		}
	} else {
		// No authentication (for local testing servers)
		if e.config.Notifiers.SMTP.Port == 587 {
			err = smtp.SendMail(addr, nil, e.config.Notifiers.SMTP.From, e.config.Notifiers.SMTP.To, []byte(msg))
		} else {
			// For local testing servers (like MailHog on port 1025)
			err = smtp.SendMail(addr, nil, e.config.Notifiers.SMTP.From, e.config.Notifiers.SMTP.To, []byte(msg))
		}
	}

	if err != nil {
		slog.Error("Failed to send email", "error", err)
		return fmt.Errorf("failed to send email: %v", err)
	}

	slog.Info("Email notification sent successfully", "recipients", e.config.Notifiers.SMTP.To)
	return nil
}

// sendWithTLS sends email with TLS encryption
func (e *EmailNotifier) sendWithTLS(addr string, auth smtp.Auth, from string, to []string, msg []byte) error {
	// Create TLS config
	tlsConfig := &tls.Config{
		ServerName: e.config.Notifiers.SMTP.Host,
	}

	// Connect to SMTP server
	conn, err := tls.Dial("tcp", addr, tlsConfig)
	if err != nil {
		return err
	}
	defer conn.Close()

	// Create SMTP client
	client, err := smtp.NewClient(conn, e.config.Notifiers.SMTP.Host)
	if err != nil {
		return err
	}
	defer client.Close()

	// Authenticate
	if err = client.Auth(auth); err != nil {
		return err
	}

	// Set sender
	if err = client.Mail(from); err != nil {
		return err
	}

	// Set recipients
	for _, recipient := range to {
		if err = client.Rcpt(recipient); err != nil {
			return err
		}
	}

	// Send message
	writer, err := client.Data()
	if err != nil {
		return err
	}
	defer writer.Close()

	_, err = writer.Write(msg)
	return err
}
