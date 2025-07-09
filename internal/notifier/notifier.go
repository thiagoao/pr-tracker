package notifier

import (
	"fc-pr-tracker/pkg/models"
)

// Notifier interface defines the contract for notification services
type Notifier interface {
	Notify(allPRs []models.PullRequest, repoPRs map[string][]models.PullRequest,
		prParticipants map[int][]models.Participant, staleAfterDays int) error
}
