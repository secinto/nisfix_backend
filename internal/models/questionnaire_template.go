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
}

// BeforeUpdate sets the UpdatedAt timestamp
func (qt *QuestionnaireTemplate) BeforeUpdate() {
	qt.UpdatedAt = time.Now().UTC()
}

// Publish marks the template as published
func (qt *QuestionnaireTemplate) Publish() {
	now := time.Now().UTC()
	qt.PublishedAt = &now
	qt.UpdatedAt = now
}

// IsPublished returns true if the template has been published
func (qt *QuestionnaireTemplate) IsPublished() bool {
	return qt.PublishedAt != nil
}

// IncrementUsage increments the usage count
func (qt *QuestionnaireTemplate) IncrementUsage() {
	qt.UsageCount++
	qt.UpdatedAt = time.Now().UTC()
}

// CanBeEdited returns true if the template can be edited
// System templates cannot be edited; custom templates can be edited before publishing
func (qt *QuestionnaireTemplate) CanBeEdited() bool {
	return !qt.IsSystem
}

// CanBeDeleted returns true if the template can be deleted
// System templates cannot be deleted; custom templates can be deleted if not used
func (qt *QuestionnaireTemplate) CanBeDeleted() bool {
	return !qt.IsSystem && qt.UsageCount == 0
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
