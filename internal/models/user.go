package models

import (
	"encoding/json"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// UserRole represents the role of a user within an organization
// #IMPLEMENTATION_DECISION: UPPERCASE in Go code, lowercase in JSON serialization
type UserRole string

const (
	UserRoleAdmin  UserRole = "ADMIN"
	UserRoleViewer UserRole = "VIEWER"
)

// MarshalJSON converts UserRole to lowercase for JSON serialization
func (ur UserRole) MarshalJSON() ([]byte, error) {
	return json.Marshal(strings.ToLower(string(ur)))
}

// UnmarshalJSON converts lowercase JSON to UserRole
func (ur *UserRole) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	*ur = UserRole(strings.ToUpper(s))
	return nil
}

// IsValid checks if the UserRole is a valid value
func (ur UserRole) IsValid() bool {
	switch ur {
	case UserRoleAdmin, UserRoleViewer:
		return true
	}
	return false
}

// User represents a user account with role-based access within an organization
// #DATA_ASSUMPTION: Email is unique across entire system (not per organization)
// #DATA_ASSUMPTION: Users belong to exactly ONE organization (no multi-org membership)
// #CARDINALITY_ASSUMPTION: Organization 1:N Users - One organization has many users
type User struct {
	ID             primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Email          string             `bson:"email" json:"email"`
	Name           string             `bson:"name,omitempty" json:"name,omitempty"`
	OrganizationID primitive.ObjectID `bson:"organization_id" json:"organization_id"`
	Role           UserRole           `bson:"role" json:"role"`

	// Status
	IsActive    bool       `bson:"is_active" json:"is_active"`
	LastLoginAt *time.Time `bson:"last_login_at,omitempty" json:"last_login_at,omitempty"`

	// Preferences
	Language string `bson:"language" json:"language"`
	Timezone string `bson:"timezone,omitempty" json:"timezone,omitempty"`

	// Audit fields with soft delete support
	CreatedAt time.Time  `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time  `bson:"updated_at" json:"updated_at"`
	DeletedAt *time.Time `bson:"deleted_at,omitempty" json:"deleted_at,omitempty"`
}

// CollectionName returns the MongoDB collection name for users
func (User) CollectionName() string {
	return "users"
}

// IsDeleted returns true if the user has been soft deleted
func (u *User) IsDeleted() bool {
	return u.DeletedAt != nil
}

// BeforeCreate sets default values before inserting a new user
func (u *User) BeforeCreate() {
	now := time.Now().UTC()
	if u.ID.IsZero() {
		u.ID = primitive.NewObjectID()
	}
	u.CreatedAt = now
	u.UpdatedAt = now
	u.IsActive = true

	// Set default language if empty
	if u.Language == "" {
		u.Language = "en"
	}
}

// BeforeUpdate sets the UpdatedAt timestamp
func (u *User) BeforeUpdate() {
	u.UpdatedAt = time.Now().UTC()
}

// SoftDelete marks the user as deleted and inactive
func (u *User) SoftDelete() {
	now := time.Now().UTC()
	u.DeletedAt = &now
	u.UpdatedAt = now
	u.IsActive = false
}

// IsAdmin returns true if the user has admin role
func (u *User) IsAdmin() bool {
	return u.Role == UserRoleAdmin
}

// IsViewer returns true if the user has viewer role
func (u *User) IsViewer() bool {
	return u.Role == UserRoleViewer
}

// UpdateLastLogin updates the last login timestamp
func (u *User) UpdateLastLogin() {
	now := time.Now().UTC()
	u.LastLoginAt = &now
	u.UpdatedAt = now
}

// CanManageOrganization returns true if the user can manage organization settings
func (u *User) CanManageOrganization() bool {
	return u.IsAdmin() && u.IsActive && !u.IsDeleted()
}

// CanInviteSuppliers returns true if the user can invite suppliers
func (u *User) CanInviteSuppliers() bool {
	return u.IsAdmin() && u.IsActive && !u.IsDeleted()
}

// CanCreateRequirements returns true if the user can create requirements
func (u *User) CanCreateRequirements() bool {
	return u.IsAdmin() && u.IsActive && !u.IsDeleted()
}

// CanReviewResponses returns true if the user can review supplier responses
func (u *User) CanReviewResponses() bool {
	return u.IsAdmin() && u.IsActive && !u.IsDeleted()
}
