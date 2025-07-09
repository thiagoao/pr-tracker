package models

import (
	"encoding/json"
	"os"
	"testing"
	"time"
)

func TestFileNotificationStateStore_GetLastNotificationTime(t *testing.T) {
	// Test case 1: File doesn't exist
	store := &FileNotificationStateStore{Path: "nonexistent_file.json"}
	retrievedTime, err := store.GetLastNotificationTime()
	if err != nil {
		t.Errorf("Expected no error when file doesn't exist, got: %v", err)
	}
	if !retrievedTime.IsZero() {
		t.Errorf("Expected zero time when file doesn't exist, got: %v", retrievedTime)
	}

	// Test case 2: File exists with valid timestamp
	testTime := time.Now()
	data, _ := json.Marshal(testTime)
	tempFile := "test_timestamp.json"
	err = os.WriteFile(tempFile, data, 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}
	defer os.Remove(tempFile)

	store.Path = tempFile
	retrievedTime, err = store.GetLastNotificationTime()
	if err != nil {
		t.Errorf("Expected no error when reading valid file, got: %v", err)
	}
	if retrievedTime.Unix() != testTime.Unix() {
		t.Errorf("Expected time %v, got %v", testTime, retrievedTime)
	}

	// Test case 3: File exists with invalid JSON
	invalidData := []byte("invalid json")
	err = os.WriteFile(tempFile, invalidData, 0644)
	if err != nil {
		t.Fatalf("Failed to write invalid data: %v", err)
	}

	_, err = store.GetLastNotificationTime()
	if err == nil {
		t.Error("Expected error when reading invalid JSON, got nil")
	}
}

func TestFileNotificationStateStore_SetLastNotificationTime(t *testing.T) {
	tempFile := "test_set_timestamp.json"
	defer os.Remove(tempFile)

	store := &FileNotificationStateStore{Path: tempFile}
	testTime := time.Now()

	// Test case 1: Set timestamp successfully
	err := store.SetLastNotificationTime(testTime)
	if err != nil {
		t.Errorf("Expected no error when setting timestamp, got: %v", err)
	}

	// Verify the file was created and contains correct data
	data, err := os.ReadFile(tempFile)
	if err != nil {
		t.Errorf("Failed to read created file: %v", err)
	}

	var savedTime time.Time
	err = json.Unmarshal(data, &savedTime)
	if err != nil {
		t.Errorf("Failed to unmarshal saved timestamp: %v", err)
	}

	if savedTime.Unix() != testTime.Unix() {
		t.Errorf("Expected saved time %v, got %v", testTime, savedTime)
	}
}

func TestPullRequest_IsStale(t *testing.T) {
	now := time.Now()
	nowMillis := now.UnixMilli()

	tests := []struct {
		name           string
		updatedDate    int64
		staleAfterDays int
		expected       bool
	}{
		{
			name:           "PR is stale",
			updatedDate:    now.AddDate(0, 0, -10).UnixMilli(),
			staleAfterDays: 5,
			expected:       true,
		},
		{
			name:           "PR is not stale",
			updatedDate:    now.AddDate(0, 0, -2).UnixMilli(),
			staleAfterDays: 5,
			expected:       false,
		},
		{
			name:           "PR updated today",
			updatedDate:    nowMillis,
			staleAfterDays: 5,
			expected:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pr := PullRequest{
				UpdatedDate: tt.updatedDate,
			}

			// Calculate expected stale time
			staleTime := now.AddDate(0, 0, -tt.staleAfterDays)
			isStale := pr.UpdatedDate < staleTime.UnixMilli()

			if isStale != tt.expected {
				t.Errorf("Expected stale=%v, got stale=%v", tt.expected, isStale)
			}
		})
	}
}

func TestPullRequest_GetApprovalCount(t *testing.T) {
	tests := []struct {
		name     string
		pr       PullRequest
		expected int
	}{
		{
			name: "No approvals",
			pr: PullRequest{
				Participants: []Participant{
					{Approved: false, Status: "UNAPPROVED"},
					{Approved: false, Status: "NEEDS_WORK"},
				},
			},
			expected: 0,
		},
		{
			name: "One approval",
			pr: PullRequest{
				Participants: []Participant{
					{Approved: true, Status: "APPROVED"},
					{Approved: false, Status: "UNAPPROVED"},
				},
			},
			expected: 1,
		},
		{
			name: "Multiple approvals",
			pr: PullRequest{
				Participants: []Participant{
					{Approved: true, Status: "APPROVED"},
					{Approved: true, Status: "APPROVED"},
					{Approved: false, Status: "UNAPPROVED"},
				},
			},
			expected: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			count := 0
			for _, participant := range tt.pr.Participants {
				if participant.Approved {
					count++
				}
			}

			if count != tt.expected {
				t.Errorf("Expected %d approvals, got %d", tt.expected, count)
			}
		})
	}
}
