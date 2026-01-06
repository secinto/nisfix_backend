package models

import (
	"encoding/json"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// RequirementType represents the type of requirement
// #IMPLEMENTATION_DECISION: QUESTIONNAIRE for questionnaire-based, CHECKFIX for CheckFix grade
type RequirementType string

const (
	RequirementTypeQuestionnaire RequirementType = "QUESTIONNAIRE"
	RequirementTypeCheckFix      RequirementType = "CHECKFIX"
)

// MarshalJSON converts RequirementType to lowercase for JSON serialization
func (rt RequirementType) MarshalJSON() ([]byte, error) {
	return json.Marshal(strings.ToLower(string(rt)))
}

// UnmarshalJSON converts lowercase JSON to RequirementType
func (rt *RequirementType) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	*rt = RequirementType(strings.ToUpper(s))
	return nil
}

// IsValid checks if the RequirementType is a valid value
func (rt RequirementType) IsValid() bool {
	switch rt {
	case RequirementTypeQuestionnaire, RequirementTypeCheckFix:
		return true
	}
	return false
}

// RequirementStatus represents the status of a requirement
// #IMPLEMENTATION_DECISION: Includes UNDER_REVIEW for revision workflow
type RequirementStatus string

const (
	RequirementStatusPending     RequirementStatus = "PENDING"
	RequirementStatusInProgress  RequirementStatus = "IN_PROGRESS"
	RequirementStatusSubmitted   RequirementStatus = "SUBMITTED"
	RequirementStatusUnderReview RequirementStatus = "UNDER_REVIEW"
	RequirementStatusApproved    RequirementStatus = "APPROVED"
	RequirementStatusRejected    RequirementStatus = "REJECTED"
	RequirementStatusExpired     RequirementStatus = "EXPIRED"
)

// MarshalJSON converts RequirementStatus to lowercase with underscores for JSON serialization
func (rs RequirementStatus) MarshalJSON() ([]byte, error) {
	return json.Marshal(strings.ToLower(string(rs)))
}

// UnmarshalJSON converts lowercase JSON to RequirementStatus
func (rs *RequirementStatus) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	*rs = RequirementStatus(strings.ToUpper(s))
	return nil
}

// IsValid checks if the RequirementStatus is a valid value
func (rs RequirementStatus) IsValid() bool {
	switch rs {
	case RequirementStatusPending, RequirementStatusInProgress, RequirementStatusSubmitted,
		RequirementStatusUnderReview, RequirementStatusApproved, RequirementStatusRejected,
		RequirementStatusExpired:
		return true
	}
	return false
}

// IsTerminal returns true if this status is a terminal state
func (rs RequirementStatus) IsTerminal() bool {
	return rs == RequirementStatusApproved || rs == RequirementStatusExpired
}

// CanTransitionTo checks if a transition to the target status is allowed
// #BUSINESS_RULE: RequirementStatus transitions:
// PENDING -> IN_PROGRESS (start) | EXPIRED (timeout)
// IN_PROGRESS -> SUBMITTED (submit) | EXPIRED (timeout)
// SUBMITTED -> APPROVED (company) | REJECTED (company) | UNDER_REVIEW (revision)
// UNDER_REVIEW -> SUBMITTED (resubmit)
// APPROVED -> (terminal state)
// REJECTED -> IN_PROGRESS (retry allowed)
// EXPIRED -> (terminal state)
func (rs RequirementStatus) CanTransitionTo(target RequirementStatus) bool {
	switch rs {
	case RequirementStatusPending:
		return target == RequirementStatusInProgress || target == RequirementStatusExpired
	case RequirementStatusInProgress:
		return target == RequirementStatusSubmitted || target == RequirementStatusExpired
	case RequirementStatusSubmitted:
		return target == RequirementStatusApproved || target == RequirementStatusRejected || target == RequirementStatusUnderReview
	case RequirementStatusUnderReview:
		return target == RequirementStatusSubmitted
	case RequirementStatusRejected:
		return target == RequirementStatusInProgress
	case RequirementStatusApproved, RequirementStatusExpired:
		return false // Terminal states
	}
	return false
}

// Priority represents the priority level of a requirement
type Priority string

const (
	PriorityLow    Priority = "LOW"
	PriorityMedium Priority = "MEDIUM"
	PriorityHigh   Priority = "HIGH"
)

// MarshalJSON converts Priority to lowercase for JSON serialization
func (p Priority) MarshalJSON() ([]byte, error) {
	return json.Marshal(strings.ToLower(string(p)))
}

// UnmarshalJSON converts lowercase JSON to Priority
func (p *Priority) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	*p = Priority(strings.ToUpper(s))
	return nil
}

// IsValid checks if the Priority is a valid value
func (p Priority) IsValid() bool {
	switch p {
	case PriorityLow, PriorityMedium, PriorityHigh:
		return true
	}
	return false
}

// RequirementStatusChange represents a change in requirement status for audit tracking
type RequirementStatusChange struct {
	FromStatus RequirementStatus  `bson:"from_status" json:"from_status"`
	ToStatus   RequirementStatus  `bson:"to_status" json:"to_status"`
	ChangedBy  primitive.ObjectID `bson:"changed_by" json:"changed_by"`
	Reason     string             `bson:"reason,omitempty" json:"reason,omitempty"`
	ChangedAt  time.Time          `bson:"changed_at" json:"changed_at"`
}

// Requirement represents a specific requirement that a Company assigns to a Supplier
// #DATA_ASSUMPTION: SupplierID denormalized from relationship for efficient querying
// #DATA_ASSUMPTION: CompanyID denormalized from relationship for efficient querying
// #CARDINALITY_ASSUMPTION: Relationship 1:N Requirements - One relationship has many requirements
type Requirement struct {
	ID             primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	RelationshipID primitive.ObjectID `bson:"relationship_id" json:"relationship_id"`
	CompanyID      primitive.ObjectID `bson:"company_id" json:"company_id"`
	SupplierID     primitive.ObjectID `bson:"supplier_id" json:"supplier_id"`

	// Requirement details
	Type        RequirementType `bson:"type" json:"type"`
	Title       string          `bson:"title" json:"title"`
	Description string          `bson:"description,omitempty" json:"description,omitempty"`
	Priority    Priority        `bson:"priority" json:"priority"`

	// For Questionnaire requirements
	QuestionnaireID *primitive.ObjectID `bson:"questionnaire_id,omitempty" json:"questionnaire_id,omitempty"`
	PassingScore    *int                `bson:"passing_score,omitempty" json:"passing_score,omitempty"`

	// For CheckFix requirements
	MinimumGrade     *string `bson:"minimum_grade,omitempty" json:"minimum_grade,omitempty"`
	MaxReportAgeDays *int    `bson:"max_report_age_days,omitempty" json:"max_report_age_days,omitempty"`

	// Timing
	DueDate        *time.Time `bson:"due_date,omitempty" json:"due_date,omitempty"`
	ReminderSentAt *time.Time `bson:"reminder_sent_at,omitempty" json:"reminder_sent_at,omitempty"`

	// Status tracking
	Status        RequirementStatus         `bson:"status" json:"status"`
	StatusHistory []RequirementStatusChange `bson:"status_history" json:"status_history"`

	// Assignment
	AssignedByUserID primitive.ObjectID `bson:"assigned_by_user_id" json:"assigned_by_user_id"`
	AssignedAt       time.Time          `bson:"assigned_at" json:"assigned_at"`

	// Audit fields
	CreatedAt time.Time `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time `bson:"updated_at" json:"updated_at"`
}

// CollectionName returns the MongoDB collection name for requirements
func (Requirement) CollectionName() string {
	return "requirements"
}

// BeforeCreate sets default values before inserting a new requirement
func (r *Requirement) BeforeCreate() {
	now := time.Now().UTC()
	if r.ID.IsZero() {
		r.ID = primitive.NewObjectID()
	}
	r.CreatedAt = now
	r.UpdatedAt = now
	r.AssignedAt = now
	r.Status = RequirementStatusPending

	// Initialize history with creation
	r.StatusHistory = []RequirementStatusChange{
		{
			FromStatus: "",
			ToStatus:   RequirementStatusPending,
			ChangedBy:  r.AssignedByUserID,
			Reason:     "Requirement assigned",
			ChangedAt:  now,
		},
	}

	// Set default priority if empty
	if r.Priority == "" {
		r.Priority = PriorityMedium
	}
}

// BeforeUpdate sets the UpdatedAt timestamp
func (r *Requirement) BeforeUpdate() {
	r.UpdatedAt = time.Now().UTC()
}

// TransitionStatus changes the requirement status with audit tracking
func (r *Requirement) TransitionStatus(newStatus RequirementStatus, changedBy primitive.ObjectID, reason string) error {
	if !r.Status.CanTransitionTo(newStatus) {
		return ErrInvalidStatusTransition
	}

	now := time.Now().UTC()
	change := RequirementStatusChange{
		FromStatus: r.Status,
		ToStatus:   newStatus,
		ChangedBy:  changedBy,
		Reason:     reason,
		ChangedAt:  now,
	}

	r.StatusHistory = append(r.StatusHistory, change)
	r.Status = newStatus
	r.UpdatedAt = now

	return nil
}

// Start marks the requirement as in progress
func (r *Requirement) Start(changedBy primitive.ObjectID) error {
	return r.TransitionStatus(RequirementStatusInProgress, changedBy, "Response started")
}

// Submit marks the requirement as submitted
func (r *Requirement) Submit(changedBy primitive.ObjectID) error {
	return r.TransitionStatus(RequirementStatusSubmitted, changedBy, "Response submitted")
}

// Approve marks the requirement as approved
func (r *Requirement) Approve(changedBy primitive.ObjectID, reason string) error {
	return r.TransitionStatus(RequirementStatusApproved, changedBy, reason)
}

// Reject marks the requirement as rejected
func (r *Requirement) Reject(changedBy primitive.ObjectID, reason string) error {
	return r.TransitionStatus(RequirementStatusRejected, changedBy, reason)
}

// RequestRevision marks the requirement as under review
func (r *Requirement) RequestRevision(changedBy primitive.ObjectID, reason string) error {
	return r.TransitionStatus(RequirementStatusUnderReview, changedBy, reason)
}

// Expire marks the requirement as expired
func (r *Requirement) Expire() error {
	if r.Status != RequirementStatusPending && r.Status != RequirementStatusInProgress {
		return ErrInvalidStatusTransition
	}
	r.Status = RequirementStatusExpired
	r.UpdatedAt = time.Now().UTC()
	return nil
}

// Retry allows retrying a rejected requirement
func (r *Requirement) Retry(changedBy primitive.ObjectID) error {
	return r.TransitionStatus(RequirementStatusInProgress, changedBy, "Retrying after rejection")
}

// Resubmit resubmits a requirement after revision
func (r *Requirement) Resubmit(changedBy primitive.ObjectID) error {
	return r.TransitionStatus(RequirementStatusSubmitted, changedBy, "Resubmitted after revision")
}

// IsPending returns true if the requirement is pending
func (r *Requirement) IsPending() bool {
	return r.Status == RequirementStatusPending
}

// IsInProgress returns true if the requirement is in progress
func (r *Requirement) IsInProgress() bool {
	return r.Status == RequirementStatusInProgress
}

// IsSubmitted returns true if the requirement has been submitted
func (r *Requirement) IsSubmitted() bool {
	return r.Status == RequirementStatusSubmitted
}

// IsApproved returns true if the requirement has been approved
func (r *Requirement) IsApproved() bool {
	return r.Status == RequirementStatusApproved
}

// IsRejected returns true if the requirement has been rejected
func (r *Requirement) IsRejected() bool {
	return r.Status == RequirementStatusRejected
}

// IsExpired returns true if the requirement has expired
func (r *Requirement) IsExpired() bool {
	return r.Status == RequirementStatusExpired
}

// IsUnderReview returns true if the requirement is under review for revision
func (r *Requirement) IsUnderReview() bool {
	return r.Status == RequirementStatusUnderReview
}

// IsQuestionnaireRequirement returns true if this is a questionnaire requirement
func (r *Requirement) IsQuestionnaireRequirement() bool {
	return r.Type == RequirementTypeQuestionnaire
}

// IsCheckFixRequirement returns true if this is a CheckFix requirement
func (r *Requirement) IsCheckFixRequirement() bool {
	return r.Type == RequirementTypeCheckFix
}

// IsOverdue returns true if the requirement is past its due date
func (r *Requirement) IsOverdue() bool {
	if r.DueDate == nil {
		return false
	}
	return time.Now().UTC().After(*r.DueDate) && !r.Status.IsTerminal()
}

// DaysUntilDue returns the number of days until the due date
func (r *Requirement) DaysUntilDue() int {
	if r.DueDate == nil {
		return -1
	}
	duration := time.Until(*r.DueDate)
	return int(duration.Hours() / 24)
}

// NeedsReminder returns true if a reminder should be sent
func (r *Requirement) NeedsReminder(reminderDaysBefore int) bool {
	if r.DueDate == nil || r.ReminderSentAt != nil {
		return false
	}
	if r.Status.IsTerminal() || r.Status == RequirementStatusSubmitted {
		return false
	}
	return r.DaysUntilDue() <= reminderDaysBefore
}

// MarkReminderSent marks that a reminder has been sent
func (r *Requirement) MarkReminderSent() {
	now := time.Now().UTC()
	r.ReminderSentAt = &now
	r.UpdatedAt = now
}

// CanStartResponse returns true if a response can be started
func (r *Requirement) CanStartResponse() bool {
	return r.IsPending()
}

// CanBeSubmitted returns true if the requirement can be submitted
func (r *Requirement) CanBeSubmitted() bool {
	return r.IsInProgress()
}

// CanBeReviewed returns true if the requirement can be reviewed
func (r *Requirement) CanBeReviewed() bool {
	return r.IsSubmitted()
}

// LastStatusChange returns the most recent status change
func (r *Requirement) LastStatusChange() *RequirementStatusChange {
	if len(r.StatusHistory) == 0 {
		return nil
	}
	return &r.StatusHistory[len(r.StatusHistory)-1]
}
