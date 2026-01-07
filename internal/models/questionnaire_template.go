package models

import (
	"encoding/json"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// TemplateCategory represents the category of a questionnaire template
// #IMPLEMENTATION_DECISION: System templates for ISO27001, GDPR, NIS2; CUSTOM for company-created
type TemplateCategory string

const (
	TemplateCategoryISO27001 TemplateCategory = "ISO27001"
	TemplateCategoryGDPR     TemplateCategory = "GDPR"
	TemplateCategoryNIS2     TemplateCategory = "NIS2"
	TemplateCategoryCustom   TemplateCategory = "CUSTOM"
)

// MarshalJSON converts TemplateCategory to lowercase for JSON serialization
func (tc TemplateCategory) MarshalJSON() ([]byte, error) {
	return json.Marshal(strings.ToLower(string(tc)))
}

// UnmarshalJSON converts lowercase JSON to TemplateCategory
func (tc *TemplateCategory) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	*tc = TemplateCategory(strings.ToUpper(s))
	return nil
}

// IsValid checks if the TemplateCategory is a valid value
func (tc TemplateCategory) IsValid() bool {
	switch tc {
	case TemplateCategoryISO27001, TemplateCategoryGDPR, TemplateCategoryNIS2, TemplateCategoryCustom:
		return true
	}
	return false
}

// IsSystemCategory returns true if this is a system-defined category
func (tc TemplateCategory) IsSystemCategory() bool {
	return tc != TemplateCategoryCustom
}

// TemplateVisibility represents the publishing scope of a template
// #IMPLEMENTATION_DECISION: DRAFT (unpublished), LOCAL (org only), GLOBAL (all orgs)
type TemplateVisibility string

const (
	TemplateVisibilityDraft  TemplateVisibility = "DRAFT"  // Not published yet
	TemplateVisibilityLocal  TemplateVisibility = "LOCAL"  // Only visible to owning organization
	TemplateVisibilityGlobal TemplateVisibility = "GLOBAL" // Available to all organizations
)

// MarshalJSON converts TemplateVisibility to lowercase for JSON serialization
func (tv TemplateVisibility) MarshalJSON() ([]byte, error) {
	return json.Marshal(strings.ToLower(string(tv)))
}

// UnmarshalJSON converts lowercase JSON to TemplateVisibility
func (tv *TemplateVisibility) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	*tv = TemplateVisibility(strings.ToUpper(s))
	return nil
}

// IsValid checks if the TemplateVisibility is a valid value
func (tv TemplateVisibility) IsValid() bool {
	switch tv {
	case TemplateVisibilityDraft, TemplateVisibilityLocal, TemplateVisibilityGlobal:
		return true
	}
	return false
}

// TemplateTopic represents a topic/section within a questionnaire template
// #NORMALIZATION_DECISION: Embedded as topics are intrinsic to template structure
type TemplateTopic struct {
	ID          string `bson:"id" json:"id"`
	Name        string `bson:"name" json:"name"`
	Description string `bson:"description,omitempty" json:"description,omitempty"`
	Order       int    `bson:"order" json:"order"`
}

// QuestionnaireTemplate represents a pre-defined questionnaire template
// #DATA_ASSUMPTION: System templates are read-only, managed by application deployment
// #DATA_ASSUMPTION: Companies can create custom templates (is_system=false, created_by_org_id set)
type QuestionnaireTemplate struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Name        string             `bson:"name" json:"name"`
	Description string             `bson:"description,omitempty" json:"description,omitempty"`
	Category    TemplateCategory   `bson:"category" json:"category"`
	Version     string             `bson:"version" json:"version"`

	// Ownership
	IsSystem       bool                `bson:"is_system" json:"is_system"`
	CreatedByOrgID *primitive.ObjectID `bson:"created_by_org_id,omitempty" json:"created_by_org_id,omitempty"`
	CreatedByUser  *primitive.ObjectID `bson:"created_by_user,omitempty" json:"created_by_user,omitempty"` // User who created (for custom templates)

	// Publishing
	Visibility  TemplateVisibility  `bson:"visibility" json:"visibility"`             // DRAFT, LOCAL, or GLOBAL
	PublishedBy *primitive.ObjectID `bson:"published_by,omitempty" json:"published_by,omitempty"` // User who published

	// Configuration
	DefaultPassingScore int `bson:"default_passing_score" json:"default_passing_score"`
	EstimatedMinutes    int `bson:"estimated_minutes" json:"estimated_minutes"`

	// Topics/Sections for organizing questions
	Topics []TemplateTopic `bson:"topics" json:"topics"`

	// Metadata
	Tags       []string `bson:"tags,omitempty" json:"tags,omitempty"`
	UsageCount int      `bson:"usage_count" json:"usage_count"`

	// Audit fields
	CreatedAt   time.Time  `bson:"created_at" json:"created_at"`
	UpdatedAt   time.Time  `bson:"updated_at" json:"updated_at"`
	PublishedAt *time.Time `bson:"published_at,omitempty" json:"published_at,omitempty"`
}

// CollectionName returns the MongoDB collection name for questionnaire templates
func (QuestionnaireTemplate) CollectionName() string {
	return "questionnaire_templates"
}

// BeforeCreate sets default values before inserting a new template
func (qt *QuestionnaireTemplate) BeforeCreate() {
	now := time.Now().UTC()
	if qt.ID.IsZero() {
		qt.ID = primitive.NewObjectID()
	}
	qt.CreatedAt = now
	qt.UpdatedAt = now
	qt.UsageCount = 0

	// Set defaults
	if qt.DefaultPassingScore == 0 {
		qt.DefaultPassingScore = 70
	}
	if qt.EstimatedMinutes == 0 {
		qt.EstimatedMinutes = 30
	}
	if qt.Version == "" {
		qt.Version = "1.0"
	}
	if qt.Topics == nil {
		qt.Topics = []TemplateTopic{}
	}
	if qt.Tags == nil {
		qt.Tags = []string{}
	}
	// Default visibility for new templates is DRAFT
	if qt.Visibility == "" {
		qt.Visibility = TemplateVisibilityDraft
	}
}

// BeforeUpdate sets the UpdatedAt timestamp
func (qt *QuestionnaireTemplate) BeforeUpdate() {
	qt.UpdatedAt = time.Now().UTC()
}

// Publish marks the template as published with the specified visibility
func (qt *QuestionnaireTemplate) Publish(visibility TemplateVisibility, publisherID primitive.ObjectID) {
	now := time.Now().UTC()
	qt.PublishedAt = &now
	qt.UpdatedAt = now
	qt.Visibility = visibility
	qt.PublishedBy = &publisherID
}

// Unpublish reverts the template to draft status
func (qt *QuestionnaireTemplate) Unpublish() {
	qt.PublishedAt = nil
	qt.PublishedBy = nil
	qt.Visibility = TemplateVisibilityDraft
	qt.UpdatedAt = time.Now().UTC()
}

// IsPublished returns true if the template has been published (LOCAL or GLOBAL)
func (qt *QuestionnaireTemplate) IsPublished() bool {
	return qt.Visibility == TemplateVisibilityLocal || qt.Visibility == TemplateVisibilityGlobal
}

// IsDraft returns true if the template is in draft status
func (qt *QuestionnaireTemplate) IsDraft() bool {
	return qt.Visibility == TemplateVisibilityDraft
}

// IsGlobal returns true if the template is published globally
func (qt *QuestionnaireTemplate) IsGlobal() bool {
	return qt.Visibility == TemplateVisibilityGlobal
}

// IsOwnedByUser returns true if the template was created by the specified user
func (qt *QuestionnaireTemplate) IsOwnedByUser(userID primitive.ObjectID) bool {
	return qt.CreatedByUser != nil && *qt.CreatedByUser == userID
}

// IsOwnedByOrg returns true if the template was created by the specified organization
func (qt *QuestionnaireTemplate) IsOwnedByOrg(orgID primitive.ObjectID) bool {
	return qt.CreatedByOrgID != nil && *qt.CreatedByOrgID == orgID
}

// IncrementUsage increments the usage count
func (qt *QuestionnaireTemplate) IncrementUsage() {
	qt.UsageCount++
	qt.UpdatedAt = time.Now().UTC()
}

// CanBeEdited returns true if the template can be edited
// System templates cannot be edited; custom templates can only be edited while in draft
func (qt *QuestionnaireTemplate) CanBeEdited() bool {
	return !qt.IsSystem && qt.IsDraft()
}

// CanBeDeleted returns true if the template can be deleted
// System templates cannot be deleted; custom templates can be deleted if draft or unused
func (qt *QuestionnaireTemplate) CanBeDeleted() bool {
	return !qt.IsSystem && (qt.IsDraft() || qt.UsageCount == 0)
}

// CanBeUnpublished returns true if the template can be reverted to draft
// Can only unpublish if not in use by any questionnaires
func (qt *QuestionnaireTemplate) CanBeUnpublished() bool {
	return !qt.IsSystem && qt.IsPublished() && qt.UsageCount == 0
}

// GetTopicByID returns a topic by its ID
func (qt *QuestionnaireTemplate) GetTopicByID(topicID string) *TemplateTopic {
	for i := range qt.Topics {
		if qt.Topics[i].ID == topicID {
			return &qt.Topics[i]
		}
	}
	return nil
}

// AddTopic adds a new topic to the template
func (qt *QuestionnaireTemplate) AddTopic(topic TemplateTopic) {
	if topic.Order == 0 {
		topic.Order = len(qt.Topics) + 1
	}
	qt.Topics = append(qt.Topics, topic)
	qt.UpdatedAt = time.Now().UTC()
}

// TopicCount returns the number of topics in the template
func (qt *QuestionnaireTemplate) TopicCount() int {
	return len(qt.Topics)
}
