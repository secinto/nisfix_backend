package models

import (
	"encoding/json"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// AuditAction represents the type of action in an audit log
// #IMPLEMENTATION_DECISION: Comprehensive action types for compliance and debugging
type AuditAction string

const (
	AuditActionCreate   AuditAction = "CREATE"
	AuditActionUpdate   AuditAction = "UPDATE"
	AuditActionDelete   AuditAction = "DELETE"
	AuditActionLogin    AuditAction = "LOGIN"
	AuditActionLogout   AuditAction = "LOGOUT"
	AuditActionApprove  AuditAction = "APPROVE"
	AuditActionReject   AuditAction = "REJECT"
	AuditActionSubmit   AuditAction = "SUBMIT"
	AuditActionInvite   AuditAction = "INVITE"
	AuditActionAccept   AuditAction = "ACCEPT"
	AuditActionDecline  AuditAction = "DECLINE"
	AuditActionSuspend  AuditAction = "SUSPEND"
	AuditActionActivate AuditAction = "ACTIVATE"
	AuditActionVerify   AuditAction = "VERIFY"
	AuditActionPublish  AuditAction = "PUBLISH"
	AuditActionArchive  AuditAction = "ARCHIVE"
)

// MarshalJSON converts AuditAction to lowercase for JSON serialization
func (a AuditAction) MarshalJSON() ([]byte, error) {
	return json.Marshal(strings.ToLower(string(a)))
}

// UnmarshalJSON converts lowercase JSON to AuditAction
func (a *AuditAction) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	*a = AuditAction(strings.ToUpper(s))
	return nil
}

// IsValid checks if the AuditAction is a valid value
func (a AuditAction) IsValid() bool {
	switch a {
	case AuditActionCreate, AuditActionUpdate, AuditActionDelete, AuditActionLogin,
		AuditActionLogout, AuditActionApprove, AuditActionReject, AuditActionSubmit,
		AuditActionInvite, AuditActionAccept, AuditActionDecline, AuditActionSuspend,
		AuditActionActivate, AuditActionVerify, AuditActionPublish, AuditActionArchive:
		return true
	}
	return false
}

// ResourceType constants for audit logging
const (
	ResourceTypeOrganization  = "organization"
	ResourceTypeUser          = "user"
	ResourceTypeQuestionnaire = "questionnaire"
	ResourceTypeQuestion      = "question"
	ResourceTypeTemplate      = "template"
	ResourceTypeRelationship  = "relationship"
	ResourceTypeRequirement   = "requirement"
	ResourceTypeResponse      = "response"
	ResourceTypeSubmission    = "submission"
	ResourceTypeVerification  = "verification"
	ResourceTypeSecureLink    = "secure_link"
)

// AuditLog represents a comprehensive activity audit trail entry
// #DATA_ASSUMPTION: Audit logs are append-only, never modified or deleted
// #DATA_ASSUMPTION: Changes field stores before/after values for UPDATE actions
// #RETENTION_POLICY: Keep for 2 years minimum for compliance
type AuditLog struct {
	ID primitive.ObjectID `bson:"_id,omitempty" json:"id"`

	// Actor (who performed the action)
	ActorUserID *primitive.ObjectID `bson:"actor_user_id,omitempty" json:"actor_user_id,omitempty"`
	ActorEmail  string              `bson:"actor_email,omitempty" json:"actor_email,omitempty"`
	ActorOrgID  *primitive.ObjectID `bson:"actor_org_id,omitempty" json:"actor_org_id,omitempty"`

	// Action
	Action       AuditAction        `bson:"action" json:"action"`
	ResourceType string             `bson:"resource_type" json:"resource_type"`
	ResourceID   primitive.ObjectID `bson:"resource_id" json:"resource_id"`

	// Context
	Description string                 `bson:"description" json:"description"`
	Changes     map[string]interface{} `bson:"changes,omitempty" json:"changes,omitempty"`

	// Request metadata
	IPAddress string `bson:"ip_address,omitempty" json:"ip_address,omitempty"`
	UserAgent string `bson:"user_agent,omitempty" json:"user_agent,omitempty"`
	RequestID string `bson:"request_id,omitempty" json:"request_id,omitempty"`

	// Timestamp
	CreatedAt time.Time `bson:"created_at" json:"created_at"`
}

// CollectionName returns the MongoDB collection name for audit logs
func (AuditLog) CollectionName() string {
	return "audit_logs"
}

// BeforeCreate sets default values before inserting a new audit log
func (a *AuditLog) BeforeCreate() {
	if a.ID.IsZero() {
		a.ID = primitive.NewObjectID()
	}
	a.CreatedAt = time.Now().UTC()

	if a.Changes == nil {
		a.Changes = map[string]interface{}{}
	}
}

// NewAuditLog creates a new audit log entry
func NewAuditLog(
	action AuditAction,
	resourceType string,
	resourceID primitive.ObjectID,
	description string,
) *AuditLog {
	log := &AuditLog{
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Description:  description,
		Changes:      map[string]interface{}{},
	}
	log.BeforeCreate()
	return log
}

// SetActor sets the actor information
func (a *AuditLog) SetActor(userID *primitive.ObjectID, email string, orgID *primitive.ObjectID) *AuditLog {
	a.ActorUserID = userID
	a.ActorEmail = email
	a.ActorOrgID = orgID
	return a
}

// SetRequestInfo sets the request metadata
func (a *AuditLog) SetRequestInfo(ipAddress, userAgent, requestID string) *AuditLog {
	a.IPAddress = ipAddress
	a.UserAgent = userAgent
	a.RequestID = requestID
	return a
}

// AddChange adds a before/after change record
func (a *AuditLog) AddChange(field string, before, after interface{}) *AuditLog {
	if a.Changes == nil {
		a.Changes = map[string]interface{}{}
	}
	a.Changes[field] = map[string]interface{}{
		"before": before,
		"after":  after,
	}
	return a
}

// AddChanges adds multiple changes at once
func (a *AuditLog) AddChanges(changes map[string]interface{}) *AuditLog {
	if a.Changes == nil {
		a.Changes = map[string]interface{}{}
	}
	for k, v := range changes {
		a.Changes[k] = v
	}
	return a
}

// HasChanges returns true if there are recorded changes
func (a *AuditLog) HasChanges() bool {
	return len(a.Changes) > 0
}

// IsAuthAction returns true if this is an authentication-related action
func (a *AuditLog) IsAuthAction() bool {
	return a.Action == AuditActionLogin || a.Action == AuditActionLogout
}

// IsModificationAction returns true if this is a data modification action
func (a *AuditLog) IsModificationAction() bool {
	return a.Action == AuditActionCreate || a.Action == AuditActionUpdate || a.Action == AuditActionDelete
}

// AuditLogBuilder provides a fluent interface for building audit logs
type AuditLogBuilder struct {
	log *AuditLog
}

// NewAuditLogBuilder creates a new audit log builder
func NewAuditLogBuilder() *AuditLogBuilder {
	return &AuditLogBuilder{
		log: &AuditLog{
			Changes: map[string]interface{}{},
		},
	}
}

// Action sets the action
func (b *AuditLogBuilder) Action(action AuditAction) *AuditLogBuilder {
	b.log.Action = action
	return b
}

// Resource sets the resource type and ID
func (b *AuditLogBuilder) Resource(resourceType string, resourceID primitive.ObjectID) *AuditLogBuilder {
	b.log.ResourceType = resourceType
	b.log.ResourceID = resourceID
	return b
}

// Description sets the description
func (b *AuditLogBuilder) Description(description string) *AuditLogBuilder {
	b.log.Description = description
	return b
}

// Actor sets the actor information
func (b *AuditLogBuilder) Actor(userID *primitive.ObjectID, email string, orgID *primitive.ObjectID) *AuditLogBuilder {
	b.log.ActorUserID = userID
	b.log.ActorEmail = email
	b.log.ActorOrgID = orgID
	return b
}

// RequestInfo sets the request metadata
func (b *AuditLogBuilder) RequestInfo(ipAddress, userAgent, requestID string) *AuditLogBuilder {
	b.log.IPAddress = ipAddress
	b.log.UserAgent = userAgent
	b.log.RequestID = requestID
	return b
}

// Change adds a single change
func (b *AuditLogBuilder) Change(field string, before, after interface{}) *AuditLogBuilder {
	b.log.AddChange(field, before, after)
	return b
}

// Changes adds multiple changes
func (b *AuditLogBuilder) Changes(changes map[string]interface{}) *AuditLogBuilder {
	b.log.AddChanges(changes)
	return b
}

// Build creates the final audit log
func (b *AuditLogBuilder) Build() *AuditLog {
	b.log.BeforeCreate()
	return b.log
}
