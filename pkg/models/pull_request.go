package models

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"time"
)

// PullRequest represents a Bitbucket pull request
type PullRequest struct {
	ID          int    `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	State       string `json:"state"`
	Open        bool   `json:"open"`
	Closed      bool   `json:"closed"`
	CreatedDate int64  `json:"createdDate"` // Unix timestamp in milliseconds
	UpdatedDate int64  `json:"updatedDate"` // Unix timestamp in milliseconds
	Author      struct {
		User struct {
			DisplayName string `json:"displayName"`
			Username    string `json:"name"`
		} `json:"user"`
		Role     string `json:"role"`
		Approved bool   `json:"approved"`
		Status   string `json:"status"`
	} `json:"author"`
	Participants []Participant `json:"participants"`
	Links        struct {
		Self []struct {
			Href string `json:"href"`
		} `json:"self"`
	} `json:"links"`
}

// Participant represents a PR participant (reviewer, author, etc.)
type Participant struct {
	User struct {
		DisplayName string `json:"displayName"`
		Username    string `json:"name"`
		Email       string `json:"emailAddress"`
		Active      bool   `json:"active"`
		ID          int    `json:"id"`
		Slug        string `json:"slug"`
		Type        string `json:"type"`
		Links       struct {
			Self []struct {
				Href string `json:"href"`
			} `json:"self"`
		} `json:"links"`
	} `json:"user"`
	Role     string `json:"role"`
	Approved bool   `json:"approved"`
	Status   string `json:"status"`
}

// Comment represents a PR comment/activity
type Comment struct {
	ID          int    `json:"id"`
	Content     string `json:"content"`
	CreatedDate int64  `json:"createdDate"` // Unix timestamp in milliseconds
	UpdatedDate int64  `json:"updatedDate"` // Unix timestamp in milliseconds
	User        struct {
		DisplayName string `json:"displayName"`
		Username    string `json:"name"`
		Email       string `json:"emailAddress"`
	} `json:"user"`
}

// FileNotificationStateStore handles notification state persistence
type FileNotificationStateStore struct {
	Path string
}

// GetLastNotificationTime retrieves the last notification time from file
func (s *FileNotificationStateStore) GetLastNotificationTime() (time.Time, error) {
	data, err := ioutil.ReadFile(s.Path)
	if err != nil {
		return time.Time{}, nil // Return zero time if file doesn't exist
	}

	var timestamp time.Time
	err = json.Unmarshal(data, &timestamp)
	if err != nil {
		return time.Time{}, fmt.Errorf("error parsing timestamp: %v", err)
	}

	return timestamp, nil
}

// SetLastNotificationTime saves the current time as last notification time
func (s *FileNotificationStateStore) SetLastNotificationTime(t time.Time) error {
	data, err := json.Marshal(t)
	if err != nil {
		return fmt.Errorf("error marshaling timestamp: %v", err)
	}

	err = ioutil.WriteFile(s.Path, data, 0644)
	if err != nil {
		return fmt.Errorf("error writing timestamp file: %v", err)
	}

	return nil
}
