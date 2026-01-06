package models

import (
	"encoding/json"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// QuestionType represents the type of question
// #IMPLEMENTATION_DECISION: Support single/multiple choice, text, and yes/no questions
type QuestionType string

const (
	QuestionTypeSingleChoice   QuestionType = "SINGLE_CHOICE"
	QuestionTypeMultipleChoice QuestionType = "MULTIPLE_CHOICE"
	QuestionTypeText           QuestionType = "TEXT"
	QuestionTypeYesNo          QuestionType = "YES_NO"
)

// MarshalJSON converts QuestionType to lowercase with underscores for JSON serialization
func (qt QuestionType) MarshalJSON() ([]byte, error) {
	return json.Marshal(strings.ToLower(string(qt)))
}

// UnmarshalJSON converts lowercase JSON to QuestionType
func (qt *QuestionType) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	*qt = QuestionType(strings.ToUpper(s))
	return nil
}

// IsValid checks if the QuestionType is a valid value
func (qt QuestionType) IsValid() bool {
	switch qt {
	case QuestionTypeSingleChoice, QuestionTypeMultipleChoice, QuestionTypeText, QuestionTypeYesNo:
		return true
	}
	return false
}

// RequiresOptions returns true if this question type requires options
func (qt QuestionType) RequiresOptions() bool {
	return qt == QuestionTypeSingleChoice || qt == QuestionTypeMultipleChoice
}

// IsChoiceType returns true if this is a choice-based question
func (qt QuestionType) IsChoiceType() bool {
	return qt == QuestionTypeSingleChoice || qt == QuestionTypeMultipleChoice || qt == QuestionTypeYesNo
}

// QuestionOption represents an answer option for choice-based questions
// #NORMALIZATION_DECISION: Options embedded as they are never queried independently
type QuestionOption struct {
	ID        string `bson:"id" json:"id"`
	Text      string `bson:"text" json:"text"`
	Points    int    `bson:"points" json:"points"`
	IsCorrect bool   `bson:"is_correct" json:"is_correct"`
	Order     int    `bson:"order" json:"order"`
}

// Question represents an individual question with options, scoring, and required flag
// #DATA_ASSUMPTION: Weight defaults to 1, allows emphasizing critical questions
// #DATA_ASSUMPTION: IsMustPass questions cause automatic fail regardless of total score
// #CARDINALITY_ASSUMPTION: Questionnaire 1:N Questions - Questionnaire contains many questions
type Question struct {
	ID              primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	QuestionnaireID primitive.ObjectID `bson:"questionnaire_id" json:"questionnaire_id"`
	TopicID         string             `bson:"topic_id" json:"topic_id"`

	// Content
	Text        string `bson:"text" json:"text"`
	Description string `bson:"description,omitempty" json:"description,omitempty"`
	HelpText    string `bson:"help_text,omitempty" json:"help_text,omitempty"`

	// Type and ordering
	Type  QuestionType `bson:"type" json:"type"`
	Order int          `bson:"order" json:"order"`

	// Scoring
	Weight     int  `bson:"weight" json:"weight"`
	MaxPoints  int  `bson:"max_points" json:"max_points"`
	IsMustPass bool `bson:"is_must_pass" json:"is_must_pass"`

	// Options (embedded for single/multiple choice)
	Options []QuestionOption `bson:"options,omitempty" json:"options,omitempty"`

	// Audit fields
	CreatedAt time.Time `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time `bson:"updated_at" json:"updated_at"`
}

// CollectionName returns the MongoDB collection name for questions
func (Question) CollectionName() string {
	return "questions"
}

// BeforeCreate sets default values before inserting a new question
func (q *Question) BeforeCreate() {
	now := time.Now().UTC()
	if q.ID.IsZero() {
		q.ID = primitive.NewObjectID()
	}
	q.CreatedAt = now
	q.UpdatedAt = now

	// Set defaults
	if q.Weight == 0 {
		q.Weight = 1
	}
	if q.MaxPoints == 0 {
		q.MaxPoints = q.calculateMaxPoints()
	}
	if q.Options == nil {
		q.Options = []QuestionOption{}
	}
}

// BeforeUpdate sets the UpdatedAt timestamp
func (q *Question) BeforeUpdate() {
	q.UpdatedAt = time.Now().UTC()
}

// calculateMaxPoints determines the maximum points based on question type and options
func (q *Question) calculateMaxPoints() int {
	if len(q.Options) == 0 {
		return 1 // Default for text questions
	}

	maxPoints := 0
	switch q.Type {
	case QuestionTypeSingleChoice, QuestionTypeYesNo:
		// Max points is the highest option score
		for _, opt := range q.Options {
			if opt.Points > maxPoints {
				maxPoints = opt.Points
			}
		}
	case QuestionTypeMultipleChoice:
		// Max points is sum of all correct option scores
		for _, opt := range q.Options {
			if opt.IsCorrect && opt.Points > 0 {
				maxPoints += opt.Points
			}
		}
	case QuestionTypeText:
		// Text questions use default maxPoints of 0 (will be set to 1 below)
	}
	if maxPoints == 0 {
		maxPoints = 1
	}
	return maxPoints
}

// RecalculateMaxPoints recalculates and updates the max points
func (q *Question) RecalculateMaxPoints() {
	q.MaxPoints = q.calculateMaxPoints()
	q.UpdatedAt = time.Now().UTC()
}

// GetOptionByID returns an option by its ID
func (q *Question) GetOptionByID(optionID string) *QuestionOption {
	for i := range q.Options {
		if q.Options[i].ID == optionID {
			return &q.Options[i]
		}
	}
	return nil
}

// AddOption adds a new option to the question
func (q *Question) AddOption(option QuestionOption) {
	if option.Order == 0 {
		option.Order = len(q.Options) + 1
	}
	q.Options = append(q.Options, option)
	q.MaxPoints = q.calculateMaxPoints()
	q.UpdatedAt = time.Now().UTC()
}

// OptionCount returns the number of options
func (q *Question) OptionCount() int {
	return len(q.Options)
}

// HasOptions returns true if the question has options
func (q *Question) HasOptions() bool {
	return len(q.Options) > 0
}

// IsTextQuestion returns true if this is a text-based question
func (q *Question) IsTextQuestion() bool {
	return q.Type == QuestionTypeText
}

// IsChoiceQuestion returns true if this is a choice-based question
func (q *Question) IsChoiceQuestion() bool {
	return q.Type.IsChoiceType()
}

// IsSingleChoice returns true if this is a single choice question
func (q *Question) IsSingleChoice() bool {
	return q.Type == QuestionTypeSingleChoice
}

// IsMultipleChoice returns true if this is a multiple choice question
func (q *Question) IsMultipleChoice() bool {
	return q.Type == QuestionTypeMultipleChoice
}

// IsYesNo returns true if this is a yes/no question
func (q *Question) IsYesNo() bool {
	return q.Type == QuestionTypeYesNo
}

// WeightedMaxPoints returns the max points multiplied by weight
func (q *Question) WeightedMaxPoints() int {
	return q.MaxPoints * q.Weight
}

// CalculateScore calculates the score for given selected option IDs
func (q *Question) CalculateScore(selectedOptionIDs []string) int {
	if len(selectedOptionIDs) == 0 {
		return 0
	}

	totalScore := 0
	selectedSet := make(map[string]bool)
	for _, id := range selectedOptionIDs {
		selectedSet[id] = true
	}

	switch q.Type {
	case QuestionTypeSingleChoice, QuestionTypeYesNo:
		// For single choice, return the points of the selected option
		if len(selectedOptionIDs) == 1 {
			for _, opt := range q.Options {
				if opt.ID == selectedOptionIDs[0] {
					return opt.Points
				}
			}
		}
	case QuestionTypeMultipleChoice:
		// For multiple choice, sum points of all selected correct options
		for _, opt := range q.Options {
			if selectedSet[opt.ID] && opt.IsCorrect {
				totalScore += opt.Points
			}
		}
	case QuestionTypeText:
		// Text questions have no options to score
	}

	return totalScore
}

// ValidateAnswer validates if the answer is appropriate for this question type
func (q *Question) ValidateAnswer(selectedOptionIDs []string, textAnswer string) error {
	switch q.Type {
	case QuestionTypeSingleChoice, QuestionTypeYesNo:
		if len(selectedOptionIDs) != 1 {
			return ErrInvalidAnswerFormat
		}
		if q.GetOptionByID(selectedOptionIDs[0]) == nil {
			return ErrInvalidOptionID
		}
	case QuestionTypeMultipleChoice:
		if len(selectedOptionIDs) == 0 {
			return ErrInvalidAnswerFormat
		}
		for _, id := range selectedOptionIDs {
			if q.GetOptionByID(id) == nil {
				return ErrInvalidOptionID
			}
		}
	case QuestionTypeText:
		if textAnswer == "" {
			return ErrInvalidAnswerFormat
		}
	}
	return nil
}
