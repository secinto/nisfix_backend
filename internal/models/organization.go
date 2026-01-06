// Package models defines all MongoDB document models for the NisFix B2B Supplier Security Portal
// #SCHEMA_IMPLEMENTATION: Using MongoDB with BSON ObjectID primary keys
package models

import (
	"encoding/json"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// OrganizationType represents the type of organization (Company or Supplier)
// #IMPLEMENTATION_DECISION: UPPERCASE in Go code, lowercase in JSON serialization per unified blueprint
type OrganizationType string

const (
	OrganizationTypeCompany  OrganizationType = "COMPANY"
	OrganizationTypeSupplier OrganizationType = "SUPPLIER"
)

// MarshalJSON converts OrganizationType to lowercase for JSON serialization
func (ot OrganizationType) MarshalJSON() ([]byte, error) {
	return json.Marshal(strings.ToLower(string(ot)))
}

// UnmarshalJSON converts lowercase JSON to OrganizationType
func (ot *OrganizationType) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	*ot = OrganizationType(strings.ToUpper(s))
	return nil
}

// IsValid checks if the OrganizationType is a valid value
func (ot OrganizationType) IsValid() bool {
	switch ot {
	case OrganizationTypeCompany, OrganizationTypeSupplier:
		return true
	}
	return false
}

// Address represents a physical address
// #NORMALIZATION_DECISION: Embedded as 1:1 relationship, rarely queried independently
type Address struct {
	Street     string `bson:"street,omitempty" json:"street,omitempty"`
	City       string `bson:"city,omitempty" json:"city,omitempty"`
	PostalCode string `bson:"postal_code,omitempty" json:"postal_code,omitempty"`
	Country    string `bson:"country,omitempty" json:"country,omitempty"`
}

// OrganizationSettings contains organization-specific configuration
// #IMPLEMENTATION_DECISION: Merged settings from both system and data architecture plans
type OrganizationSettings struct {
	// From System Architecture
	DefaultDueDays     int      `bson:"default_due_days" json:"default_due_days"`
	RequireCheckFix    bool     `bson:"require_checkfix" json:"require_checkfix"`
	MinCheckFixGrade   string   `bson:"min_checkfix_grade" json:"min_checkfix_grade"`
	NotificationEmails []string `bson:"notification_emails" json:"notification_emails"`

	// From Data Architecture
	DefaultLanguage      string `bson:"default_language" json:"default_language"`
	NotificationsEnabled bool   `bson:"notifications_enabled" json:"notifications_enabled"`
	ReminderDaysBefore   int    `bson:"reminder_days_before" json:"reminder_days_before"`
}

// DefaultOrganizationSettings returns default settings for a new organization
func DefaultOrganizationSettings() OrganizationSettings {
	return OrganizationSettings{
		DefaultDueDays:       30,
		RequireCheckFix:      false,
		MinCheckFixGrade:     "C",
		NotificationEmails:   []string{},
		DefaultLanguage:      "en",
		NotificationsEnabled: true,
		ReminderDaysBefore:   7,
	}
}

// Organization represents both Company and Supplier entities
// #DATA_ASSUMPTION: Slug generated from name, must be URL-safe lowercase alphanumeric with hyphens
// #DATA_ASSUMPTION: Domain field populated by supplier when linking CheckFix, used for verification
type Organization struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Type        OrganizationType   `bson:"type" json:"type"`
	Name        string             `bson:"name" json:"name"`
	Slug        string             `bson:"slug" json:"slug"`
	Domain      string             `bson:"domain,omitempty" json:"domain,omitempty"`
	Description string             `bson:"description,omitempty" json:"description,omitempty"`

	// Contact Information
	ContactEmail string   `bson:"contact_email" json:"contact_email"`
	ContactPhone string   `bson:"contact_phone,omitempty" json:"contact_phone,omitempty"`
	Address      *Address `bson:"address,omitempty" json:"address,omitempty"`

	// CheckFix Integration (Suppliers only)
	CheckFixAccountID string     `bson:"checkfix_account_id,omitempty" json:"checkfix_account_id,omitempty"`
	CheckFixLinkedAt  *time.Time `bson:"checkfix_linked_at,omitempty" json:"checkfix_linked_at,omitempty"`

	// Settings
	Settings OrganizationSettings `bson:"settings" json:"settings"`

	// Audit fields with soft delete support
	CreatedAt time.Time  `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time  `bson:"updated_at" json:"updated_at"`
	DeletedAt *time.Time `bson:"deleted_at,omitempty" json:"deleted_at,omitempty"`
}

// CollectionName returns the MongoDB collection name for organizations
func (Organization) CollectionName() string {
	return "organizations"
}

// IsDeleted returns true if the organization has been soft deleted
func (o *Organization) IsDeleted() bool {
	return o.DeletedAt != nil
}

// BeforeCreate sets default values before inserting a new organization
func (o *Organization) BeforeCreate() {
	now := time.Now().UTC()
	if o.ID.IsZero() {
		o.ID = primitive.NewObjectID()
	}
	o.CreatedAt = now
	o.UpdatedAt = now

	// Set default settings if empty
	if o.Settings.DefaultLanguage == "" {
		o.Settings = DefaultOrganizationSettings()
	}
}

// BeforeUpdate sets the UpdatedAt timestamp
func (o *Organization) BeforeUpdate() {
	o.UpdatedAt = time.Now().UTC()
}

// SoftDelete marks the organization as deleted
func (o *Organization) SoftDelete() {
	now := time.Now().UTC()
	o.DeletedAt = &now
	o.UpdatedAt = now
}

// IsCompany returns true if the organization is a company
func (o *Organization) IsCompany() bool {
	return o.Type == OrganizationTypeCompany
}

// IsSupplier returns true if the organization is a supplier
func (o *Organization) IsSupplier() bool {
	return o.Type == OrganizationTypeSupplier
}

// HasCheckFixLinked returns true if the supplier has linked CheckFix
func (o *Organization) HasCheckFixLinked() bool {
	return o.CheckFixAccountID != "" && o.CheckFixLinkedAt != nil
}
