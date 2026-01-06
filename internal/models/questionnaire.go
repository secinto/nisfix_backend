package models

import (
	"encoding/json"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// QuestionnaireStatus represents the status of a questionnaire
// #IMPLEMENTATION_DECISION: DRAFT -> PUBLISHED -> ARCHIVED lifecycle
type QuestionnaireStatus string

const (
	QuestionnaireStatusDraft     QuestionnaireStatus = "DRAFT"
	QuestionnaireStatusPublished QuestionnaireStatus = "PUBLISHED"
	QuestionnaireStatusArchived  QuestionnaireStatus = "ARCHIVED"
)

// MarshalJSON converts QuestionnaireStatus to lowercase for JSON serialization
func (qs QuestionnaireStatus) MarshalJSON() ([]byte, error) {
	return json.Marshal(strings.ToLower(string(qs)))
}

// UnmarshalJSON converts lowercase JSON to QuestionnaireStatus
func (qs *QuestionnaireStatus) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	*qs = QuestionnaireStatus(strings.ToUpper(s))
	return nil
}

// IsValid checks if the QuestionnaireStatus is a valid value
func (qs QuestionnaireStatus) IsValid() bool {
	switch qs {
	case QuestionnaireStatusDraft, QuestionnaireStatusPublished, QuestionnaireStatusArchived:
		return true
	}
	return false
}

// ScoringMode represents how the questionnaire is scored
// #IMPLEMENTATION_DECISION: PERCENTAGE for relative scoring, POINTS for absolute scoring
type ScoringMode string

const (
	ScoringModePercentage ScoringMode = "PERCENTAGE"
	ScoringModePoints     ScoringMode = "POINTS"
)

// MarshalJSON converts ScoringMode to lowercase for JSON serialization
func (sm ScoringMode) MarshalJSON() ([]byte, error) {
	return json.Marshal(strings.ToLower(string(sm)))
}

// UnmarshalJSON converts lowercase JSON to ScoringMode
func (sm *ScoringMode) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	*sm = ScoringMode(strings.ToUpper(s))
	return nil
}

// IsValid checks if the ScoringMode is a valid value
func (sm ScoringMode) IsValid() bool {
	switch sm {
	case ScoringModePercentage, ScoringModePoints:
		return true
	}
	return false
}

// QuestionnaireTopic represents a topic/section within a questionnaire
// #NORMALIZATION_DECISION: Copied from template, can be customized per questionnaire
type QuestionnaireTopic struct {
	ID          string `bson:"id" json:"id"`
	Name        string `bson:"name" json:"name"`
	Description string `bson:"description,omitempty" json:"description,omitempty"`
	Order       int    `bson:"order" json:"order"`
}

// Questionnaire represents a company-customized questionnaire instance
// #DATA_ASSUMPTION: Questionnaires are immutable once published (create new version instead)
// #CARDINALITY_ASSUMPTION: Company 1:N Questionnaires - Company owns multiple questionnaires
// #CARDINALITY_ASSUMPTION: Template 1:N Questionnaires - Template can spawn many questionnaire instances
type Questionnaire struct {
	ID         primitive.ObjectID  `bson:"_id,omitempty" json:"id"`
	CompanyID  primitive.ObjectID  `bson:"company_id" json:"company_id"`
	TemplateID *primitive.ObjectID `bson:"template_id,omitempty" json:"template_id,omitempty"`

	// Basic info
	Name        string              `bson:"name" json:"name"`
	Description string              `bson:"description,omitempty" json:"description,omitempty"`
	Status      QuestionnaireStatus `bson:"status" json:"status"`
	Version     int                 `bson:"version" json:"version"`

	// Scoring configuration
	PassingScore int         `bson:"passing_score" json:"passing_score"`
	ScoringMode  ScoringMode `bson:"scoring_mode" json:"scoring_mode"`

	// Topics (copied from template, can be customized)
	Topics []QuestionnaireTopic `bson:"topics" json:"topics"`

	// Statistics (denormalized for dashboard)
	// #NORMALIZATION_DECISION: QuestionCount denormalized for dashboard performance
	QuestionCount    int `bson:"question_count" json:"question_count"`
	MaxPossibleScore int `bson:"max_possible_score" json:"max_possible_score"`

	// Audit fields
	CreatedAt   time.Time  `bson:"created_at" json:"created_at"`
	UpdatedAt   time.Time  `bson:"updated_at" json:"updated_at"`
	PublishedAt *time.Time `bson:"published_at,omitempty" json:"published_at,omitempty"`
}

// CollectionName returns the MongoDB collection name for questionnaires
func (Questionnaire) CollectionName() string {
	return "questionnaires"
}

// BeforeCreate sets default values before inserting a new questionnaire
func (q *Questionnaire) BeforeCreate() {
	now := time.Now().UTC()
	if q.ID.IsZero() {
		q.ID = primitive.NewObjectID()
	}
	q.CreatedAt = now
	q.UpdatedAt = now
	q.Status = QuestionnaireStatusDraft
	q.Version = 1

	// Set defaults
	if q.PassingScore == 0 {
		q.PassingScore = 70
	}
	if q.ScoringMode == "" {
		q.ScoringMode = ScoringModePercentage
	}
	if q.Topics == nil {
		q.Topics = []QuestionnaireTopic{}
	}
}

// BeforeUpdate sets the UpdatedAt timestamp
func (q *Questionnaire) BeforeUpdate() {
	q.UpdatedAt = time.Now().UTC()
}

// Publish marks the questionnaire as published
func (q *Questionnaire) Publish() error {
	if q.Status != QuestionnaireStatusDraft {
		return ErrInvalidStatusTransition
	}
	now := time.Now().UTC()
	q.Status = QuestionnaireStatusPublished
	q.PublishedAt = &now
	q.UpdatedAt = now
	return nil
}

// Archive marks the questionnaire as archived
func (q *Questionnaire) Archive() error {
	if q.Status != QuestionnaireStatusPublished {
		return ErrInvalidStatusTransition
	}
	q.Status = QuestionnaireStatusArchived
	q.UpdatedAt = time.Now().UTC()
	return nil
}

// IsDraft returns true if the questionnaire is in draft status
func (q *Questionnaire) IsDraft() bool {
	return q.Status == QuestionnaireStatusDraft
}

// IsPublished returns true if the questionnaire is published
func (q *Questionnaire) IsPublished() bool {
	return q.Status == QuestionnaireStatusPublished
}

// IsArchived returns true if the questionnaire is archived
func (q *Questionnaire) IsArchived() bool {
	return q.Status == QuestionnaireStatusArchived
}

// CanBeEdited returns true if the questionnaire can be edited
// Only draft questionnaires can be edited
func (q *Questionnaire) CanBeEdited() bool {
	return q.IsDraft()
}

// CanBeDeleted returns true if the questionnaire can be deleted
// Only draft questionnaires can be deleted
func (q *Questionnaire) CanBeDeleted() bool {
	return q.IsDraft()
}

// CanBeAssigned returns true if the questionnaire can be assigned to suppliers
func (q *Questionnaire) CanBeAssigned() bool {
	return q.IsPublished()
}

// GetTopicByID returns a topic by its ID
func (q *Questionnaire) GetTopicByID(topicID string) *QuestionnaireTopic {
	for i := range q.Topics {
		if q.Topics[i].ID == topicID {
			return &q.Topics[i]
		}
	}
	return nil
}

// AddTopic adds a new topic to the questionnaire
func (q *Questionnaire) AddTopic(topic QuestionnaireTopic) {
	if topic.Order == 0 {
		topic.Order = len(q.Topics) + 1
	}
	q.Topics = append(q.Topics, topic)
	q.UpdatedAt = time.Now().UTC()
}

// UpdateStatistics updates the denormalized question count and max score
func (q *Questionnaire) UpdateStatistics(questionCount, maxPossibleScore int) {
	q.QuestionCount = questionCount
	q.MaxPossibleScore = maxPossibleScore
	q.UpdatedAt = time.Now().UTC()
}

// TopicCount returns the number of topics in the questionnaire
func (q *Questionnaire) TopicCount() int {
	return len(q.Topics)
}

// IsFromTemplate returns true if this questionnaire was created from a template
func (q *Questionnaire) IsFromTemplate() bool {
	return q.TemplateID != nil
}
