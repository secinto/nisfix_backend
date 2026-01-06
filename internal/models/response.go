package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// SupplierResponse represents a supplier's response to a requirement
// #DATA_ASSUMPTION: Either SubmissionID or VerificationID is set, not both (based on requirement type)
// #CARDINALITY_ASSUMPTION: Requirement 1:1 SupplierResponse - One response per requirement
// #NORMALIZATION_DECISION: Score/Passed denormalized from submission for dashboard performance
type SupplierResponse struct {
	ID            primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	RequirementID primitive.ObjectID `bson:"requirement_id" json:"requirement_id"`
	SupplierID    primitive.ObjectID `bson:"supplier_id" json:"supplier_id"`

	// For questionnaire responses
	SubmissionID *primitive.ObjectID `bson:"submission_id,omitempty" json:"submission_id,omitempty"`

	// For CheckFix responses
	VerificationID *primitive.ObjectID `bson:"verification_id,omitempty" json:"verification_id,omitempty"`

	// Scoring (denormalized for quick access)
	Score    *int    `bson:"score,omitempty" json:"score,omitempty"`
	MaxScore *int    `bson:"max_score,omitempty" json:"max_score,omitempty"`
	Passed   *bool   `bson:"passed,omitempty" json:"passed,omitempty"`
	Grade    *string `bson:"grade,omitempty" json:"grade,omitempty"`

	// Draft answers (saved progress for questionnaire responses)
	DraftAnswers []DraftAnswer `bson:"draft_answers,omitempty" json:"draft_answers,omitempty"`

	// Review
	ReviewedByUserID *primitive.ObjectID `bson:"reviewed_by_user_id,omitempty" json:"reviewed_by_user_id,omitempty"`
	ReviewedAt       *time.Time          `bson:"reviewed_at,omitempty" json:"reviewed_at,omitempty"`
	ReviewNotes      string              `bson:"review_notes,omitempty" json:"review_notes,omitempty"`

	// Audit fields
	StartedAt   time.Time  `bson:"started_at" json:"started_at"`
	SubmittedAt *time.Time `bson:"submitted_at,omitempty" json:"submitted_at,omitempty"`
	CreatedAt   time.Time  `bson:"created_at" json:"created_at"`
	UpdatedAt   time.Time  `bson:"updated_at" json:"updated_at"`
}

// DraftAnswer represents a saved draft answer for a questionnaire question
type DraftAnswer struct {
	QuestionID      primitive.ObjectID `bson:"question_id" json:"question_id"`
	SelectedOptions []string           `bson:"selected_options,omitempty" json:"selected_options,omitempty"`
	TextAnswer      string             `bson:"text_answer,omitempty" json:"text_answer,omitempty"`
	SavedAt         time.Time          `bson:"saved_at" json:"saved_at"`
}

// CollectionName returns the MongoDB collection name for supplier responses
func (SupplierResponse) CollectionName() string {
	return "supplier_responses"
}

// BeforeCreate sets default values before inserting a new response
func (r *SupplierResponse) BeforeCreate() {
	now := time.Now().UTC()
	if r.ID.IsZero() {
		r.ID = primitive.NewObjectID()
	}
	r.CreatedAt = now
	r.UpdatedAt = now
	r.StartedAt = now

	if r.DraftAnswers == nil {
		r.DraftAnswers = []DraftAnswer{}
	}
}

// BeforeUpdate sets the UpdatedAt timestamp
func (r *SupplierResponse) BeforeUpdate() {
	r.UpdatedAt = time.Now().UTC()
}

// Submit marks the response as submitted
func (r *SupplierResponse) Submit() {
	now := time.Now().UTC()
	r.SubmittedAt = &now
	r.UpdatedAt = now
}

// SetSubmission links a questionnaire submission to this response
func (r *SupplierResponse) SetSubmission(submissionID primitive.ObjectID, score, maxScore int, passed bool) {
	r.SubmissionID = &submissionID
	r.Score = &score
	r.MaxScore = &maxScore
	r.Passed = &passed
	r.UpdatedAt = time.Now().UTC()
}

// SetVerification links a CheckFix verification to this response
func (r *SupplierResponse) SetVerification(verificationID primitive.ObjectID, grade string, passed bool) {
	r.VerificationID = &verificationID
	r.Grade = &grade
	r.Passed = &passed
	r.UpdatedAt = time.Now().UTC()
}

// MarkReviewed marks the response as reviewed
func (r *SupplierResponse) MarkReviewed(reviewerID primitive.ObjectID, notes string) {
	now := time.Now().UTC()
	r.ReviewedByUserID = &reviewerID
	r.ReviewedAt = &now
	r.ReviewNotes = notes
	r.UpdatedAt = now
}

// IsSubmitted returns true if the response has been submitted
func (r *SupplierResponse) IsSubmitted() bool {
	return r.SubmittedAt != nil
}

// IsReviewed returns true if the response has been reviewed
func (r *SupplierResponse) IsReviewed() bool {
	return r.ReviewedAt != nil
}

// HasSubmission returns true if a questionnaire submission is linked
func (r *SupplierResponse) HasSubmission() bool {
	return r.SubmissionID != nil
}

// HasVerification returns true if a CheckFix verification is linked
func (r *SupplierResponse) HasVerification() bool {
	return r.VerificationID != nil
}

// HasPassed returns true if the response passed (for both questionnaire and CheckFix)
func (r *SupplierResponse) HasPassed() bool {
	return r.Passed != nil && *r.Passed
}

// GetScorePercentage returns the score as a percentage
func (r *SupplierResponse) GetScorePercentage() float64 {
	if r.Score == nil || r.MaxScore == nil || *r.MaxScore == 0 {
		return 0
	}
	return float64(*r.Score) / float64(*r.MaxScore) * 100
}

// SaveDraftAnswer saves or updates a draft answer
func (r *SupplierResponse) SaveDraftAnswer(answer DraftAnswer) {
	answer.SavedAt = time.Now().UTC()

	// Update existing or append new
	found := false
	for i, a := range r.DraftAnswers {
		if a.QuestionID == answer.QuestionID {
			r.DraftAnswers[i] = answer
			found = true
			break
		}
	}
	if !found {
		r.DraftAnswers = append(r.DraftAnswers, answer)
	}
	r.UpdatedAt = time.Now().UTC()
}

// GetDraftAnswer returns the draft answer for a question
func (r *SupplierResponse) GetDraftAnswer(questionID primitive.ObjectID) *DraftAnswer {
	for i := range r.DraftAnswers {
		if r.DraftAnswers[i].QuestionID == questionID {
			return &r.DraftAnswers[i]
		}
	}
	return nil
}

// ClearDraftAnswers removes all draft answers (after submission)
func (r *SupplierResponse) ClearDraftAnswers() {
	r.DraftAnswers = []DraftAnswer{}
	r.UpdatedAt = time.Now().UTC()
}

// DraftAnswerCount returns the number of saved draft answers
func (r *SupplierResponse) DraftAnswerCount() int {
	return len(r.DraftAnswers)
}

// CompletionTimeMinutes returns the time from start to submission in minutes
func (r *SupplierResponse) CompletionTimeMinutes() int {
	if r.SubmittedAt == nil {
		return 0
	}
	return int(r.SubmittedAt.Sub(r.StartedAt).Minutes())
}
