package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"time"

	"fc-pr-tracker/internal/bitbucket"
	"fc-pr-tracker/internal/config"
	"fc-pr-tracker/internal/logger"
	"fc-pr-tracker/internal/notifier"
	"fc-pr-tracker/pkg/models"
)

func main() {
	// Load configuration
	cfg := config.Load("config.yaml")

	// Initialize logger
	logger.Init(cfg)

	slog.Info("PR monitoring service started",
		"log_file", cfg.Log.File,
		"log_level", cfg.Log.Level)

	// Test Bitbucket connection
	bitbucketClient := bitbucket.NewClient(cfg)
	err := bitbucketClient.TestConnection()
	if err != nil {
		slog.Error("Bitbucket connection test failed", "error", err)
		os.Exit(1)
	}

	slog.Info("Bitbucket connection test succeeded")
	slog.Info("Loaded configuration",
		"workspace", cfg.Bitbucket.Workspace,
		"user", cfg.Bitbucket.User,
		"repositories", cfg.Bitbucket.Repositories,
		"stale_after_days", cfg.PRFilter.StaleAfterDays,
		"email_recipients", cfg.Notifiers.SMTP.To,
		"notification_interval_hours", cfg.Notification.IntervalHours,
	)

	// Setup context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle graceful shutdown
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, os.Interrupt)
	go func() {
		<-sigs
		slog.Info("Shutting down gracefully...")
		cancel()
	}()

	// Run the service
	err = run(ctx, cfg)
	if err != nil {
		slog.Error("Application error", "error", err)
		os.Exit(1)
	}

	slog.Info("Shutdown complete.")
}

// run contains the main monitoring logic
func run(ctx context.Context, cfg *config.Config) error {
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

	// Initialize Bitbucket client
	bitbucketClient := bitbucket.NewClient(cfg)

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
