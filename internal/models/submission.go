package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// QuestionnaireSubmission contains all answers for a questionnaire submission with calculated scores
// #DATA_ASSUMPTION: Answers stored as embedded array (10-100 items, acceptable for MongoDB)
// #NORMALIZATION_DECISION: Answers embedded - always read together, never queried individually
// #NORMALIZATION_DECISION: TopicScores calculated and stored at submission time for reporting
// #CARDINALITY_ASSUMPTION: SupplierResponse 1:1 QuestionnaireSubmission
type QuestionnaireSubmission struct {
	ID              primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	ResponseID      primitive.ObjectID `bson:"response_id" json:"response_id"`
	QuestionnaireID primitive.ObjectID `bson:"questionnaire_id" json:"questionnaire_id"`
	SupplierID      primitive.ObjectID `bson:"supplier_id" json:"supplier_id"`

	// Answers
	Answers []SubmissionAnswer `bson:"answers" json:"answers"`

	// Calculated scores
	TotalScore       int     `bson:"total_score" json:"total_score"`
	MaxPossibleScore int     `bson:"max_possible_score" json:"max_possible_score"`
	PercentageScore  float64 `bson:"percentage_score" json:"percentage_score"`
	Passed           bool    `bson:"passed" json:"passed"`
	MustPassFailed   bool    `bson:"must_pass_failed" json:"must_pass_failed"`

	// Topic-level scores
	TopicScores []TopicScore `bson:"topic_scores" json:"topic_scores"`

	// Metadata
	CompletionTimeMinutes int `bson:"completion_time_minutes" json:"completion_time_minutes"`

	// Audit fields
	StartedAt   time.Time  `bson:"started_at" json:"started_at"`
	SubmittedAt *time.Time `bson:"submitted_at,omitempty" json:"submitted_at,omitempty"`
	CreatedAt   time.Time  `bson:"created_at" json:"created_at"`
	UpdatedAt   time.Time  `bson:"updated_at" json:"updated_at"`
}

// SubmissionAnswer represents a single answer in a submission
type SubmissionAnswer struct {
	QuestionID      primitive.ObjectID `bson:"question_id" json:"question_id"`
	SelectedOptions []string           `bson:"selected_options,omitempty" json:"selected_options,omitempty"`
	TextAnswer      string             `bson:"text_answer,omitempty" json:"text_answer,omitempty"`
	PointsEarned    int                `bson:"points_earned" json:"points_earned"`
	MaxPoints       int                `bson:"max_points" json:"max_points"`
	IsMustPassMet   *bool              `bson:"is_must_pass_met,omitempty" json:"is_must_pass_met,omitempty"`
}

// TopicScore represents the score for a specific topic
type TopicScore struct {
	TopicID         string  `bson:"topic_id" json:"topic_id"`
	TopicName       string  `bson:"topic_name" json:"topic_name"`
	Score           int     `bson:"score" json:"score"`
	MaxScore        int     `bson:"max_score" json:"max_score"`
	PercentageScore float64 `bson:"percentage_score" json:"percentage_score"`
}

// CollectionName returns the MongoDB collection name for questionnaire submissions
func (QuestionnaireSubmission) CollectionName() string {
	return "questionnaire_submissions"
}

// BeforeCreate sets default values before inserting a new submission
func (s *QuestionnaireSubmission) BeforeCreate() {
	now := time.Now().UTC()
	if s.ID.IsZero() {
		s.ID = primitive.NewObjectID()
	}
	s.CreatedAt = now
	s.UpdatedAt = now

	if s.Answers == nil {
		s.Answers = []SubmissionAnswer{}
	}
	if s.TopicScores == nil {
		s.TopicScores = []TopicScore{}
	}
}

// BeforeUpdate sets the UpdatedAt timestamp
func (s *QuestionnaireSubmission) BeforeUpdate() {
	s.UpdatedAt = time.Now().UTC()
}

// Submit marks the submission as submitted and calculates final scores
func (s *QuestionnaireSubmission) Submit() {
	now := time.Now().UTC()
	s.SubmittedAt = &now
	s.UpdatedAt = now
}

// CalculateScores calculates all scores from answers
// This should be called after all answers are added
func (s *QuestionnaireSubmission) CalculateScores(passingScore int) {
	s.TotalScore = 0
	s.MaxPossibleScore = 0
	s.MustPassFailed = false

	for _, answer := range s.Answers {
		s.TotalScore += answer.PointsEarned
		s.MaxPossibleScore += answer.MaxPoints

		// Check must-pass questions
		if answer.IsMustPassMet != nil && !*answer.IsMustPassMet {
			s.MustPassFailed = true
		}
	}

	// Calculate percentage
	if s.MaxPossibleScore > 0 {
		s.PercentageScore = float64(s.TotalScore) / float64(s.MaxPossibleScore) * 100
	}

	// Determine if passed
	// #BUSINESS_RULE: IsMustPass questions cause automatic fail regardless of total score
	s.Passed = !s.MustPassFailed && s.PercentageScore >= float64(passingScore)
}

// AddAnswer adds an answer to the submission
func (s *QuestionnaireSubmission) AddAnswer(answer SubmissionAnswer) {
	s.Answers = append(s.Answers, answer)
	s.UpdatedAt = time.Now().UTC()
}

// GetAnswer returns the answer for a specific question
func (s *QuestionnaireSubmission) GetAnswer(questionID primitive.ObjectID) *SubmissionAnswer {
	for i := range s.Answers {
		if s.Answers[i].QuestionID == questionID {
			return &s.Answers[i]
		}
	}
	return nil
}

// AddTopicScore adds a topic score
func (s *QuestionnaireSubmission) AddTopicScore(score TopicScore) {
	// Calculate percentage for the topic
	if score.MaxScore > 0 {
		score.PercentageScore = float64(score.Score) / float64(score.MaxScore) * 100
	}
	s.TopicScores = append(s.TopicScores, score)
	s.UpdatedAt = time.Now().UTC()
}

// GetTopicScore returns the score for a specific topic
func (s *QuestionnaireSubmission) GetTopicScore(topicID string) *TopicScore {
	for i := range s.TopicScores {
		if s.TopicScores[i].TopicID == topicID {
			return &s.TopicScores[i]
		}
	}
	return nil
}

// IsSubmitted returns true if the submission has been submitted
func (s *QuestionnaireSubmission) IsSubmitted() bool {
	return s.SubmittedAt != nil
}

// AnswerCount returns the number of answers
func (s *QuestionnaireSubmission) AnswerCount() int {
	return len(s.Answers)
}

// HasPassedAllMustPass returns true if all must-pass questions were answered correctly
func (s *QuestionnaireSubmission) HasPassedAllMustPass() bool {
	return !s.MustPassFailed
}

// GetFailedMustPassCount returns the count of failed must-pass questions
func (s *QuestionnaireSubmission) GetFailedMustPassCount() int {
	count := 0
	for _, answer := range s.Answers {
		if answer.IsMustPassMet != nil && !*answer.IsMustPassMet {
			count++
		}
	}
	return count
}

// GetWeakestTopics returns topics with the lowest percentage scores
func (s *QuestionnaireSubmission) GetWeakestTopics(limit int) []TopicScore {
	if len(s.TopicScores) == 0 {
		return []TopicScore{}
	}

	// Copy and sort by percentage score ascending
	sorted := make([]TopicScore, len(s.TopicScores))
	copy(sorted, s.TopicScores)

	// Simple bubble sort for small arrays
	for i := 0; i < len(sorted)-1; i++ {
		for j := 0; j < len(sorted)-i-1; j++ {
			if sorted[j].PercentageScore > sorted[j+1].PercentageScore {
				sorted[j], sorted[j+1] = sorted[j+1], sorted[j]
			}
		}
	}

	if limit > len(sorted) {
		limit = len(sorted)
	}
	return sorted[:limit]
}
