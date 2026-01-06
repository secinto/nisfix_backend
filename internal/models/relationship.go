package models

import (
	"encoding/json"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// RelationshipStatus represents the status of a company-supplier relationship
// #IMPLEMENTATION_DECISION: State machine with defined transitions
type RelationshipStatus string

const (
	RelationshipStatusPending    RelationshipStatus = "PENDING"
	RelationshipStatusActive     RelationshipStatus = "ACTIVE"
	RelationshipStatusRejected   RelationshipStatus = "REJECTED"
	RelationshipStatusSuspended  RelationshipStatus = "SUSPENDED"
	RelationshipStatusTerminated RelationshipStatus = "TERMINATED"
)

// MarshalJSON converts RelationshipStatus to lowercase for JSON serialization
func (rs RelationshipStatus) MarshalJSON() ([]byte, error) {
	return json.Marshal(strings.ToLower(string(rs)))
}

// UnmarshalJSON converts lowercase JSON to RelationshipStatus
func (rs *RelationshipStatus) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	*rs = RelationshipStatus(strings.ToUpper(s))
	return nil
}

// IsValid checks if the RelationshipStatus is a valid value
func (rs RelationshipStatus) IsValid() bool {
	switch rs {
	case RelationshipStatusPending, RelationshipStatusActive, RelationshipStatusRejected,
		RelationshipStatusSuspended, RelationshipStatusTerminated:
		return true
	}
	return false
}

// IsTerminal returns true if this status is a terminal state
func (rs RelationshipStatus) IsTerminal() bool {
	return rs == RelationshipStatusRejected || rs == RelationshipStatusTerminated
}

// CanTransitionTo checks if a transition to the target status is allowed
// #BUSINESS_RULE: RelationshipStatus transitions:
// PENDING -> ACTIVE (accept) | REJECTED (decline)
// ACTIVE -> SUSPENDED (company action) | TERMINATED (either party)
// SUSPENDED -> ACTIVE (reactivate) | TERMINATED (finalize)
// REJECTED -> (terminal state)
// TERMINATED -> (terminal state)
func (rs RelationshipStatus) CanTransitionTo(target RelationshipStatus) bool {
	switch rs {
	case RelationshipStatusPending:
		return target == RelationshipStatusActive || target == RelationshipStatusRejected
	case RelationshipStatusActive:
		return target == RelationshipStatusSuspended || target == RelationshipStatusTerminated
	case RelationshipStatusSuspended:
		return target == RelationshipStatusActive || target == RelationshipStatusTerminated
	case RelationshipStatusRejected, RelationshipStatusTerminated:
		return false // Terminal states
	}
	return false
}

// SupplierClassification represents the risk classification of a supplier
// #IMPLEMENTATION_DECISION: 3-tier classification per unified blueprint (CRITICAL, IMPORTANT, STANDARD)
type SupplierClassification string

const (
	SupplierClassificationCritical  SupplierClassification = "CRITICAL"
	SupplierClassificationImportant SupplierClassification = "IMPORTANT"
	SupplierClassificationStandard  SupplierClassification = "STANDARD"
)

// MarshalJSON converts SupplierClassification to lowercase for JSON serialization
func (sc SupplierClassification) MarshalJSON() ([]byte, error) {
	return json.Marshal(strings.ToLower(string(sc)))
}

// UnmarshalJSON converts lowercase JSON to SupplierClassification
func (sc *SupplierClassification) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	*sc = SupplierClassification(strings.ToUpper(s))
	return nil
}

// IsValid checks if the SupplierClassification is a valid value
func (sc SupplierClassification) IsValid() bool {
	switch sc {
	case SupplierClassificationCritical, SupplierClassificationImportant, SupplierClassificationStandard:
		return true
	}
	return false
}

// Priority returns the numeric priority (higher = more critical)
func (sc SupplierClassification) Priority() int {
	switch sc {
	case SupplierClassificationCritical:
		return 3
	case SupplierClassificationImportant:
		return 2
	case SupplierClassificationStandard:
		return 1
	}
	return 0
}

// StatusChange represents a change in relationship status for audit tracking
// #NORMALIZATION_DECISION: Embedded for audit trail without separate collection
type StatusChange struct {
	FromStatus RelationshipStatus `bson:"from_status" json:"from_status"`
	ToStatus   RelationshipStatus `bson:"to_status" json:"to_status"`
	ChangedBy  primitive.ObjectID `bson:"changed_by" json:"changed_by"`
	Reason     string             `bson:"reason,omitempty" json:"reason,omitempty"`
	ChangedAt  time.Time          `bson:"changed_at" json:"changed_at"`
}

// CompanySupplierRelationship tracks the business relationship between a Company and a Supplier
// #DATA_ASSUMPTION: SupplierID is null until invitation accepted (email-based invite)
// #DATA_ASSUMPTION: One relationship per Company-Supplier pair (enforced by unique index)
// #RELATIONSHIP_PATTERN: Many-to-Many bridge table pattern between Company and Supplier orgs
type CompanySupplierRelationship struct {
	ID         primitive.ObjectID  `bson:"_id,omitempty" json:"id"`
	CompanyID  primitive.ObjectID  `bson:"company_id" json:"company_id"`
	SupplierID *primitive.ObjectID `bson:"supplier_id,omitempty" json:"supplier_id,omitempty"`

	// Invitation details
	InvitedEmail    string             `bson:"invited_email" json:"invited_email"`
	InvitedByUserID primitive.ObjectID `bson:"invited_by_user_id" json:"invited_by_user_id"`
	InvitedAt       time.Time          `bson:"invited_at" json:"invited_at"`

	// Status tracking
	Status        RelationshipStatus `bson:"status" json:"status"`
	StatusHistory []StatusChange     `bson:"status_history" json:"status_history"`

	// Classification
	Classification SupplierClassification `bson:"classification" json:"classification"`
	Notes          string                 `bson:"notes,omitempty" json:"notes,omitempty"`

	// Service details
	ServicesProvided []string `bson:"services_provided,omitempty" json:"services_provided,omitempty"`
	ContractRef      string   `bson:"contract_ref,omitempty" json:"contract_ref,omitempty"`

	// Response tracking (denormalized)
	AcceptedAt *time.Time `bson:"accepted_at,omitempty" json:"accepted_at,omitempty"`
	RejectedAt *time.Time `bson:"rejected_at,omitempty" json:"rejected_at,omitempty"`

	// Audit fields
	CreatedAt time.Time `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time `bson:"updated_at" json:"updated_at"`
}

// CollectionName returns the MongoDB collection name for relationships
func (CompanySupplierRelationship) CollectionName() string {
	return "company_supplier_relationships"
}

// BeforeCreate sets default values before inserting a new relationship
func (r *CompanySupplierRelationship) BeforeCreate() {
	now := time.Now().UTC()
	if r.ID.IsZero() {
		r.ID = primitive.NewObjectID()
	}
	r.CreatedAt = now
	r.UpdatedAt = now
	r.InvitedAt = now
	r.Status = RelationshipStatusPending

	// Initialize history with creation
	r.StatusHistory = []StatusChange{
		{
			FromStatus: "",
			ToStatus:   RelationshipStatusPending,
			ChangedBy:  r.InvitedByUserID,
			Reason:     "Invitation sent",
			ChangedAt:  now,
		},
	}

	// Set default classification if empty
	if r.Classification == "" {
		r.Classification = SupplierClassificationStandard
	}

	if r.ServicesProvided == nil {
		r.ServicesProvided = []string{}
	}
}

// BeforeUpdate sets the UpdatedAt timestamp
func (r *CompanySupplierRelationship) BeforeUpdate() {
	r.UpdatedAt = time.Now().UTC()
}

// TransitionStatus changes the relationship status with audit tracking
func (r *CompanySupplierRelationship) TransitionStatus(newStatus RelationshipStatus, changedBy primitive.ObjectID, reason string) error {
	if !r.Status.CanTransitionTo(newStatus) {
		return ErrInvalidStatusTransition
	}

	now := time.Now().UTC()
	change := StatusChange{
		FromStatus: r.Status,
		ToStatus:   newStatus,
		ChangedBy:  changedBy,
		Reason:     reason,
		ChangedAt:  now,
	}

	r.StatusHistory = append(r.StatusHistory, change)
	r.Status = newStatus
	r.UpdatedAt = now

	// Update denormalized timestamps
	switch newStatus {
	case RelationshipStatusActive:
		r.AcceptedAt = &now
	case RelationshipStatusRejected:
		r.RejectedAt = &now
	case RelationshipStatusPending, RelationshipStatusSuspended, RelationshipStatusTerminated:
		// No additional timestamp updates needed for these statuses
	}

	return nil
}

// Accept accepts the invitation and activates the relationship
func (r *CompanySupplierRelationship) Accept(supplierID primitive.ObjectID, changedBy primitive.ObjectID) error {
	r.SupplierID = &supplierID
	return r.TransitionStatus(RelationshipStatusActive, changedBy, "Invitation accepted")
}

// Decline declines the invitation
func (r *CompanySupplierRelationship) Decline(changedBy primitive.ObjectID, reason string) error {
	return r.TransitionStatus(RelationshipStatusRejected, changedBy, reason)
}

// Suspend suspends the relationship
func (r *CompanySupplierRelationship) Suspend(changedBy primitive.ObjectID, reason string) error {
	return r.TransitionStatus(RelationshipStatusSuspended, changedBy, reason)
}

// Reactivate reactivates a suspended relationship
func (r *CompanySupplierRelationship) Reactivate(changedBy primitive.ObjectID, reason string) error {
	return r.TransitionStatus(RelationshipStatusActive, changedBy, reason)
}

// Terminate terminates the relationship
func (r *CompanySupplierRelationship) Terminate(changedBy primitive.ObjectID, reason string) error {
	return r.TransitionStatus(RelationshipStatusTerminated, changedBy, reason)
}

// IsPending returns true if the relationship is pending
func (r *CompanySupplierRelationship) IsPending() bool {
	return r.Status == RelationshipStatusPending
}

// IsActive returns true if the relationship is active
func (r *CompanySupplierRelationship) IsActive() bool {
	return r.Status == RelationshipStatusActive
}

// IsSuspended returns true if the relationship is suspended
func (r *CompanySupplierRelationship) IsSuspended() bool {
	return r.Status == RelationshipStatusSuspended
}

// IsTerminated returns true if the relationship is terminated
func (r *CompanySupplierRelationship) IsTerminated() bool {
	return r.Status == RelationshipStatusTerminated
}

// IsRejected returns true if the relationship was rejected
func (r *CompanySupplierRelationship) IsRejected() bool {
	return r.Status == RelationshipStatusRejected
}

// HasSupplier returns true if a supplier has been assigned
func (r *CompanySupplierRelationship) HasSupplier() bool {
	return r.SupplierID != nil
}

// CanReceiveRequirements returns true if requirements can be created for this relationship
// #BUSINESS_RULE: Requirements cannot be assigned to suppliers with TERMINATED relationships
func (r *CompanySupplierRelationship) CanReceiveRequirements() bool {
	return r.IsActive() && r.HasSupplier()
}

// UpdateClassification updates the supplier classification
func (r *CompanySupplierRelationship) UpdateClassification(classification SupplierClassification) {
	r.Classification = classification
	r.UpdatedAt = time.Now().UTC()
}

// IsCriticalSupplier returns true if this is a critical supplier
func (r *CompanySupplierRelationship) IsCriticalSupplier() bool {
	return r.Classification == SupplierClassificationCritical
}

// LastStatusChange returns the most recent status change
func (r *CompanySupplierRelationship) LastStatusChange() *StatusChange {
	if len(r.StatusHistory) == 0 {
		return nil
	}
	return &r.StatusHistory[len(r.StatusHistory)-1]
}
