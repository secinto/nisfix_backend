package models

import (
	"encoding/json"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestRequirementType_MarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		rt       RequirementType
		expected string
	}{
		{"Questionnaire lowercase", RequirementTypeQuestionnaire, `"questionnaire"`},
		{"CheckFix lowercase", RequirementTypeCheckFix, `"checkfix"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := json.Marshal(tt.rt)
			if err != nil {
				t.Fatalf("MarshalJSON() error = %v", err)
			}
			if string(got) != tt.expected {
				t.Errorf("MarshalJSON() = %v, want %v", string(got), tt.expected)
			}
		})
	}
}

func TestRequirementType_IsValid(t *testing.T) {
	tests := []struct {
		name     string
		rt       RequirementType
		expected bool
	}{
		{"Questionnaire is valid", RequirementTypeQuestionnaire, true},
		{"CheckFix is valid", RequirementTypeCheckFix, true},
		{"Invalid type", RequirementType("INVALID"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.rt.IsValid(); got != tt.expected {
				t.Errorf("IsValid() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestRequirementStatus_IsValid(t *testing.T) {
	tests := []struct {
		name     string
		rs       RequirementStatus
		expected bool
	}{
		{"Pending is valid", RequirementStatusPending, true},
		{"InProgress is valid", RequirementStatusInProgress, true},
		{"Submitted is valid", RequirementStatusSubmitted, true},
		{"UnderReview is valid", RequirementStatusUnderReview, true},
		{"Approved is valid", RequirementStatusApproved, true},
		{"Rejected is valid", RequirementStatusRejected, true},
		{"Expired is valid", RequirementStatusExpired, true},
		{"Invalid status", RequirementStatus("INVALID"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.rs.IsValid(); got != tt.expected {
				t.Errorf("IsValid() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestRequirementStatus_IsTerminal(t *testing.T) {
	tests := []struct {
		name     string
		rs       RequirementStatus
		expected bool
	}{
		{"Approved is terminal", RequirementStatusApproved, true},
		{"Expired is terminal", RequirementStatusExpired, true},
		{"Pending is not terminal", RequirementStatusPending, false},
		{"InProgress is not terminal", RequirementStatusInProgress, false},
		{"Submitted is not terminal", RequirementStatusSubmitted, false},
		{"Rejected is not terminal", RequirementStatusRejected, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.rs.IsTerminal(); got != tt.expected {
				t.Errorf("IsTerminal() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestRequirementStatus_CanTransitionTo(t *testing.T) {
	tests := []struct {
		name     string
		from     RequirementStatus
		to       RequirementStatus
		expected bool
	}{
		// From Pending
		{"Pending -> InProgress", RequirementStatusPending, RequirementStatusInProgress, true},
		{"Pending -> Expired", RequirementStatusPending, RequirementStatusExpired, true},
		{"Pending -> Submitted", RequirementStatusPending, RequirementStatusSubmitted, false},
		{"Pending -> Approved", RequirementStatusPending, RequirementStatusApproved, false},

		// From InProgress
		{"InProgress -> Submitted", RequirementStatusInProgress, RequirementStatusSubmitted, true},
		{"InProgress -> Expired", RequirementStatusInProgress, RequirementStatusExpired, true},
		{"InProgress -> Approved", RequirementStatusInProgress, RequirementStatusApproved, false},

		// From Submitted
		{"Submitted -> Approved", RequirementStatusSubmitted, RequirementStatusApproved, true},
		{"Submitted -> Rejected", RequirementStatusSubmitted, RequirementStatusRejected, true},
		{"Submitted -> UnderReview", RequirementStatusSubmitted, RequirementStatusUnderReview, true},
		{"Submitted -> InProgress", RequirementStatusSubmitted, RequirementStatusInProgress, false},

		// From UnderReview
		{"UnderReview -> Submitted", RequirementStatusUnderReview, RequirementStatusSubmitted, true},
		{"UnderReview -> Approved", RequirementStatusUnderReview, RequirementStatusApproved, false},

		// From Rejected
		{"Rejected -> InProgress", RequirementStatusRejected, RequirementStatusInProgress, true},
		{"Rejected -> Approved", RequirementStatusRejected, RequirementStatusApproved, false},

		// From Terminal states
		{"Approved -> anything", RequirementStatusApproved, RequirementStatusRejected, false},
		{"Expired -> anything", RequirementStatusExpired, RequirementStatusPending, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.from.CanTransitionTo(tt.to); got != tt.expected {
				t.Errorf("CanTransitionTo() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestPriority_IsValid(t *testing.T) {
	tests := []struct {
		name     string
		p        Priority
		expected bool
	}{
		{"Low is valid", PriorityLow, true},
		{"Medium is valid", PriorityMedium, true},
		{"High is valid", PriorityHigh, true},
		{"Invalid priority", Priority("CRITICAL"), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.p.IsValid(); got != tt.expected {
				t.Errorf("IsValid() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestRequirement_BeforeCreate(t *testing.T) {
	userID := primitive.NewObjectID()
	req := &Requirement{
		Title:            "Test Requirement",
		AssignedByUserID: userID,
	}

	req.BeforeCreate()

	if req.ID.IsZero() {
		t.Error("ID should be set")
	}
	if req.CreatedAt.IsZero() {
		t.Error("CreatedAt should be set")
	}
	if req.Status != RequirementStatusPending {
		t.Errorf("Status = %v, want Pending", req.Status)
	}
	if req.Priority != PriorityMedium {
		t.Errorf("Priority = %v, want Medium", req.Priority)
	}
	if len(req.StatusHistory) != 1 {
		t.Errorf("StatusHistory length = %v, want 1", len(req.StatusHistory))
	}
}

func TestRequirement_TransitionStatus(t *testing.T) {
	userID := primitive.NewObjectID()
	req := &Requirement{
		Title:            "Test Requirement",
		AssignedByUserID: userID,
	}
	req.BeforeCreate()

	// Valid transition: Pending -> InProgress
	err := req.TransitionStatus(RequirementStatusInProgress, userID, "Started")
	if err != nil {
		t.Errorf("TransitionStatus() unexpected error = %v", err)
	}
	if req.Status != RequirementStatusInProgress {
		t.Errorf("Status = %v, want InProgress", req.Status)
	}
	if len(req.StatusHistory) != 2 {
		t.Errorf("StatusHistory length = %v, want 2", len(req.StatusHistory))
	}

	// Invalid transition: InProgress -> Approved
	err = req.TransitionStatus(RequirementStatusApproved, userID, "Trying to approve")
	if err != ErrInvalidStatusTransition {
		t.Errorf("TransitionStatus() expected ErrInvalidStatusTransition, got %v", err)
	}
}

func TestRequirement_Start(t *testing.T) {
	userID := primitive.NewObjectID()
	req := &Requirement{
		Title:            "Test Requirement",
		AssignedByUserID: userID,
	}
	req.BeforeCreate()

	err := req.Start(userID)
	if err != nil {
		t.Errorf("Start() unexpected error = %v", err)
	}
	if !req.IsInProgress() {
		t.Error("Requirement should be in progress")
	}
}

func TestRequirement_Submit(t *testing.T) {
	userID := primitive.NewObjectID()
	req := &Requirement{
		Title:            "Test Requirement",
		AssignedByUserID: userID,
	}
	req.BeforeCreate()
	req.Start(userID)

	err := req.Submit(userID)
	if err != nil {
		t.Errorf("Submit() unexpected error = %v", err)
	}
	if !req.IsSubmitted() {
		t.Error("Requirement should be submitted")
	}
}

func TestRequirement_Approve(t *testing.T) {
	userID := primitive.NewObjectID()
	req := &Requirement{
		Title:            "Test Requirement",
		AssignedByUserID: userID,
	}
	req.BeforeCreate()
	req.Start(userID)
	req.Submit(userID)

	err := req.Approve(userID, "Looks good")
	if err != nil {
		t.Errorf("Approve() unexpected error = %v", err)
	}
	if !req.IsApproved() {
		t.Error("Requirement should be approved")
	}
}

func TestRequirement_Reject(t *testing.T) {
	userID := primitive.NewObjectID()
	req := &Requirement{
		Title:            "Test Requirement",
		AssignedByUserID: userID,
	}
	req.BeforeCreate()
	req.Start(userID)
	req.Submit(userID)

	err := req.Reject(userID, "Needs improvement")
	if err != nil {
		t.Errorf("Reject() unexpected error = %v", err)
	}
	if !req.IsRejected() {
		t.Error("Requirement should be rejected")
	}
}

func TestRequirement_RequestRevision(t *testing.T) {
	userID := primitive.NewObjectID()
	req := &Requirement{
		Title:            "Test Requirement",
		AssignedByUserID: userID,
	}
	req.BeforeCreate()
	req.Start(userID)
	req.Submit(userID)

	err := req.RequestRevision(userID, "Please clarify")
	if err != nil {
		t.Errorf("RequestRevision() unexpected error = %v", err)
	}
	if !req.IsUnderReview() {
		t.Error("Requirement should be under review")
	}
}

func TestRequirement_Retry(t *testing.T) {
	userID := primitive.NewObjectID()
	req := &Requirement{
		Title:            "Test Requirement",
		AssignedByUserID: userID,
	}
	req.BeforeCreate()
	req.Start(userID)
	req.Submit(userID)
	req.Reject(userID, "Rejected")

	err := req.Retry(userID)
	if err != nil {
		t.Errorf("Retry() unexpected error = %v", err)
	}
	if !req.IsInProgress() {
		t.Error("Requirement should be in progress after retry")
	}
}

func TestRequirement_IsOverdue(t *testing.T) {
	userID := primitive.NewObjectID()
	past := time.Now().Add(-24 * time.Hour)
	future := time.Now().Add(24 * time.Hour)

	tests := []struct {
		name     string
		dueDate  *time.Time
		status   RequirementStatus
		expected bool
	}{
		{"No due date", nil, RequirementStatusPending, false},
		{"Past due date, non-terminal", &past, RequirementStatusPending, true},
		{"Future due date", &future, RequirementStatusPending, false},
		{"Past due date, terminal status", &past, RequirementStatusApproved, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &Requirement{
				Title:            "Test",
				DueDate:          tt.dueDate,
				Status:           tt.status,
				AssignedByUserID: userID,
			}
			if got := req.IsOverdue(); got != tt.expected {
				t.Errorf("IsOverdue() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestRequirement_DaysUntilDue(t *testing.T) {
	// Use 3 full days + 1 hour buffer to avoid timing edge cases with truncation
	inThreeDays := time.Now().Add((3*24 + 1) * time.Hour)
	userID := primitive.NewObjectID()

	tests := []struct {
		name     string
		dueDate  *time.Time
		expected int
	}{
		{"No due date", nil, -1},
		{"3+ days from now", &inThreeDays, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &Requirement{
				Title:            "Test",
				DueDate:          tt.dueDate,
				AssignedByUserID: userID,
			}
			got := req.DaysUntilDue()
			if got != tt.expected {
				t.Errorf("DaysUntilDue() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestRequirement_NeedsReminder(t *testing.T) {
	userID := primitive.NewObjectID()
	inSevenDays := time.Now().Add(7 * 24 * time.Hour)  // 7 full days
	inOneDay := time.Now().Add(24 * time.Hour)          // 1 full day
	reminderSent := time.Now()

	tests := []struct {
		name           string
		dueDate        *time.Time
		reminderSent   *time.Time
		status         RequirementStatus
		daysBefore     int
		expected       bool
	}{
		{"No due date", nil, nil, RequirementStatusPending, 7, false},
		{"Reminder already sent", &inOneDay, &reminderSent, RequirementStatusPending, 7, false},
		{"Terminal status", &inOneDay, nil, RequirementStatusApproved, 7, false},
		{"Submitted status", &inOneDay, nil, RequirementStatusSubmitted, 7, false},
		{"Due in 7 days, remind 7 days before", &inSevenDays, nil, RequirementStatusPending, 7, true},
		{"Due in 7 days, remind 3 days before", &inSevenDays, nil, RequirementStatusPending, 3, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &Requirement{
				Title:            "Test",
				DueDate:          tt.dueDate,
				ReminderSentAt:   tt.reminderSent,
				Status:           tt.status,
				AssignedByUserID: userID,
			}
			if got := req.NeedsReminder(tt.daysBefore); got != tt.expected {
				t.Errorf("NeedsReminder(%d) = %v, want %v", tt.daysBefore, got, tt.expected)
			}
		})
	}
}

func TestRequirement_LastStatusChange(t *testing.T) {
	userID := primitive.NewObjectID()
	req := &Requirement{
		Title:            "Test",
		AssignedByUserID: userID,
	}
	req.BeforeCreate()
	req.Start(userID)

	last := req.LastStatusChange()
	if last == nil {
		t.Fatal("LastStatusChange() returned nil")
	}
	if last.ToStatus != RequirementStatusInProgress {
		t.Errorf("LastStatusChange().ToStatus = %v, want InProgress", last.ToStatus)
	}
}

func TestRequirement_CollectionName(t *testing.T) {
	req := Requirement{}
	if got := req.CollectionName(); got != "requirements" {
		t.Errorf("CollectionName() = %v, want requirements", got)
	}
}
