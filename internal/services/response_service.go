// Package services provides business logic implementations.
package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/checkfix-tools/nisfix_backend/internal/models"
	"github.com/checkfix-tools/nisfix_backend/internal/repository"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Custom errors for response service
var (
	ErrResponseNotFound       = errors.New("response not found")
	ErrResponseAlreadyExists  = errors.New("response already exists for this requirement")
	ErrResponseAlreadySubmitted = errors.New("response has already been submitted")
	ErrCannotStartResponse    = errors.New("cannot start response for this requirement")
	ErrSubmissionNotFound     = errors.New("submission not found")
	ErrInvalidAnswer          = errors.New("invalid answer")
)

// ResponseService handles supplier response business logic
// #INTEGRATION_POINT: Used by response handler for supplier response management
type ResponseService interface {
	// StartResponse creates a new response for a requirement
	StartResponse(ctx context.Context, requirementID, supplierID primitive.ObjectID) (*models.SupplierResponse, error)

	// GetResponse retrieves a response by ID
	GetResponse(ctx context.Context, id primitive.ObjectID, supplierID *primitive.ObjectID) (*models.SupplierResponse, error)

	// GetResponseByRequirement retrieves a response by requirement ID
	GetResponseByRequirement(ctx context.Context, requirementID primitive.ObjectID, supplierID *primitive.ObjectID) (*models.SupplierResponse, error)

	// SaveDraftAnswer saves a draft answer for a question
	SaveDraftAnswer(ctx context.Context, responseID, supplierID primitive.ObjectID, answer SaveDraftAnswerRequest) error

	// SaveMultipleDraftAnswers saves multiple draft answers at once
	SaveMultipleDraftAnswers(ctx context.Context, responseID, supplierID primitive.ObjectID, answers []SaveDraftAnswerRequest) error

	// SubmitQuestionnaireResponse submits a questionnaire response
	SubmitQuestionnaireResponse(ctx context.Context, responseID, supplierID primitive.ObjectID, answers []SubmitAnswerRequest) (*SubmissionResult, error)

	// GetSubmission retrieves a submission by ID
	GetSubmission(ctx context.Context, submissionID primitive.ObjectID) (*models.QuestionnaireSubmission, error)

	// GetSubmissionByResponse retrieves a submission by response ID
	GetSubmissionByResponse(ctx context.Context, responseID primitive.ObjectID) (*models.QuestionnaireSubmission, error)
}

// SaveDraftAnswerRequest represents a draft answer to save
type SaveDraftAnswerRequest struct {
	QuestionID      string   `json:"question_id" binding:"required"`
	SelectedOptions []string `json:"selected_options,omitempty"`
	TextAnswer      string   `json:"text_answer,omitempty"`
}

// SubmitAnswerRequest represents an answer to submit
type SubmitAnswerRequest struct {
	QuestionID      string   `json:"question_id" binding:"required"`
	SelectedOptions []string `json:"selected_options,omitempty"`
	TextAnswer      string   `json:"text_answer,omitempty"`
}

// SubmissionResult contains the result of a questionnaire submission
type SubmissionResult struct {
	Submission  *models.QuestionnaireSubmission `json:"submission"`
	Response    *models.SupplierResponse        `json:"response"`
	Requirement *models.Requirement             `json:"requirement"`
	Passed      bool                            `json:"passed"`
	Score       int                             `json:"score"`
	MaxScore    int                             `json:"max_score"`
	Percentage  float64                         `json:"percentage"`
}

// responseService implements ResponseService
type responseService struct {
	responseRepo      repository.ResponseRepository
	submissionRepo    repository.SubmissionRepository
	requirementRepo   repository.RequirementRepository
	questionnaireRepo repository.QuestionnaireRepository
	questionRepo      repository.QuestionRepository
}

// NewResponseService creates a new response service
func NewResponseService(
	responseRepo repository.ResponseRepository,
	submissionRepo repository.SubmissionRepository,
	requirementRepo repository.RequirementRepository,
	questionnaireRepo repository.QuestionnaireRepository,
	questionRepo repository.QuestionRepository,
) ResponseService {
	return &responseService{
		responseRepo:      responseRepo,
		submissionRepo:    submissionRepo,
		requirementRepo:   requirementRepo,
		questionnaireRepo: questionnaireRepo,
		questionRepo:      questionRepo,
	}
}

// StartResponse creates a new response for a requirement
// #BUSINESS_RULE: Response can only be started for pending requirements
// #BUSINESS_RULE: Only the assigned supplier can start a response
func (s *responseService) StartResponse(ctx context.Context, requirementID, supplierID primitive.ObjectID) (*models.SupplierResponse, error) {
	// Get requirement
	requirement, err := s.requirementRepo.GetByID(ctx, requirementID)
	if err != nil {
		if errors.Is(err, models.ErrRequirementNotFound) {
			return nil, ErrRequirementNotFound
		}
		return nil, fmt.Errorf("failed to get requirement: %w", err)
	}

	// Verify supplier ownership
	if requirement.SupplierID != supplierID {
		return nil, ErrRequirementNotFound
	}

	// Check if response can be started
	if !requirement.CanStartResponse() {
		return nil, ErrCannotStartResponse
	}

	// Check if response already exists
	existing, err := s.responseRepo.GetByRequirement(ctx, requirementID)
	if err == nil && existing != nil {
		// Return existing response if not submitted yet
		if !existing.IsSubmitted() {
			return existing, nil
		}
		return nil, ErrResponseAlreadyExists
	}

	// Create response
	response := &models.SupplierResponse{
		RequirementID: requirementID,
		SupplierID:    supplierID,
	}
	response.BeforeCreate()

	if err := s.responseRepo.Create(ctx, response); err != nil {
		if errors.Is(err, models.ErrResponseAlreadyExists) {
			return nil, ErrResponseAlreadyExists
		}
		return nil, fmt.Errorf("failed to create response: %w", err)
	}

	// Update requirement status to in progress
	if err := requirement.Start(supplierID); err == nil {
		_ = s.requirementRepo.Update(ctx, requirement)
	}

	return response, nil
}

// GetResponse retrieves a response by ID
func (s *responseService) GetResponse(ctx context.Context, id primitive.ObjectID, supplierID *primitive.ObjectID) (*models.SupplierResponse, error) {
	response, err := s.responseRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, models.ErrResponseNotFound) {
			return nil, ErrResponseNotFound
		}
		return nil, fmt.Errorf("failed to get response: %w", err)
	}

	// Verify supplier ownership if provided
	if supplierID != nil && response.SupplierID != *supplierID {
		return nil, ErrResponseNotFound
	}

	return response, nil
}

// GetResponseByRequirement retrieves a response by requirement ID
func (s *responseService) GetResponseByRequirement(ctx context.Context, requirementID primitive.ObjectID, supplierID *primitive.ObjectID) (*models.SupplierResponse, error) {
	response, err := s.responseRepo.GetByRequirement(ctx, requirementID)
	if err != nil {
		if errors.Is(err, models.ErrResponseNotFound) {
			return nil, ErrResponseNotFound
		}
		return nil, fmt.Errorf("failed to get response: %w", err)
	}

	// Verify supplier ownership if provided
	if supplierID != nil && response.SupplierID != *supplierID {
		return nil, ErrResponseNotFound
	}

	return response, nil
}

// SaveDraftAnswer saves a draft answer for a question
func (s *responseService) SaveDraftAnswer(ctx context.Context, responseID, supplierID primitive.ObjectID, answer SaveDraftAnswerRequest) error {
	// Verify response exists and belongs to supplier
	response, err := s.GetResponse(ctx, responseID, &supplierID)
	if err != nil {
		return err
	}

	// Cannot save draft after submission
	if response.IsSubmitted() {
		return ErrResponseAlreadySubmitted
	}

	// Parse question ID
	questionID, err := primitive.ObjectIDFromHex(answer.QuestionID)
	if err != nil {
		return ErrInvalidAnswer
	}

	// Create draft answer
	draftAnswer := models.DraftAnswer{
		QuestionID:      questionID,
		SelectedOptions: answer.SelectedOptions,
		TextAnswer:      answer.TextAnswer,
		SavedAt:         time.Now().UTC(),
	}

	if err := s.responseRepo.SaveDraftAnswer(ctx, responseID, draftAnswer); err != nil {
		return fmt.Errorf("failed to save draft answer: %w", err)
	}

	return nil
}

// SaveMultipleDraftAnswers saves multiple draft answers at once
func (s *responseService) SaveMultipleDraftAnswers(ctx context.Context, responseID, supplierID primitive.ObjectID, answers []SaveDraftAnswerRequest) error {
	for _, answer := range answers {
		if err := s.SaveDraftAnswer(ctx, responseID, supplierID, answer); err != nil {
			return err
		}
	}
	return nil
}

// SubmitQuestionnaireResponse submits a questionnaire response
// #BUSINESS_RULE: All answers are scored and saved to submission
// #BUSINESS_RULE: Requirement status is updated to submitted
func (s *responseService) SubmitQuestionnaireResponse(ctx context.Context, responseID, supplierID primitive.ObjectID, answers []SubmitAnswerRequest) (*SubmissionResult, error) {
	// Verify response exists and belongs to supplier
	response, err := s.GetResponse(ctx, responseID, &supplierID)
	if err != nil {
		return nil, err
	}

	// Cannot submit if already submitted
	if response.IsSubmitted() {
		return nil, ErrResponseAlreadySubmitted
	}

	// Get requirement
	requirement, err := s.requirementRepo.GetByID(ctx, response.RequirementID)
	if err != nil {
		return nil, fmt.Errorf("failed to get requirement: %w", err)
	}

	// Verify requirement is questionnaire type
	if !requirement.IsQuestionnaireRequirement() || requirement.QuestionnaireID == nil {
		return nil, errors.New("requirement is not a questionnaire requirement")
	}

	// Get questionnaire
	questionnaire, err := s.questionnaireRepo.GetByID(ctx, *requirement.QuestionnaireID)
	if err != nil {
		return nil, fmt.Errorf("failed to get questionnaire: %w", err)
	}

	// Get questions
	questions, err := s.questionRepo.ListByQuestionnaire(ctx, *requirement.QuestionnaireID)
	if err != nil {
		return nil, fmt.Errorf("failed to get questions: %w", err)
	}

	// Build question map for quick lookup
	questionMap := make(map[string]*models.Question)
	for i := range questions {
		questionMap[questions[i].ID.Hex()] = &questions[i]
	}

	// Create submission
	submission := &models.QuestionnaireSubmission{
		ResponseID:      responseID,
		QuestionnaireID: *requirement.QuestionnaireID,
		SupplierID:      supplierID,
		StartedAt:       response.StartedAt,
	}
	submission.BeforeCreate()

	// Build topic scores map
	topicScores := make(map[string]*models.TopicScore)
	for _, topic := range questionnaire.Topics {
		topicScores[topic.ID] = &models.TopicScore{
			TopicID:   topic.ID,
			TopicName: topic.Name,
			Score:     0,
			MaxScore:  0,
		}
	}

	// Score each answer
	for _, answerReq := range answers {
		question, exists := questionMap[answerReq.QuestionID]
		if !exists {
			continue // Skip unknown questions
		}

		// Calculate score for this answer
		var pointsEarned int
		if question.IsChoiceQuestion() {
			pointsEarned = question.CalculateScore(answerReq.SelectedOptions)
		} else if question.IsTextQuestion() {
			// Text questions get full points if answered
			if answerReq.TextAnswer != "" {
				pointsEarned = question.MaxPoints
			}
		}

		// Check must-pass
		var mustPassMet *bool
		if question.IsMustPass {
			passed := pointsEarned >= question.MaxPoints
			mustPassMet = &passed
		}

		// Create submission answer
		submissionAnswer := models.SubmissionAnswer{
			QuestionID:      question.ID,
			SelectedOptions: answerReq.SelectedOptions,
			TextAnswer:      answerReq.TextAnswer,
			PointsEarned:    pointsEarned,
			MaxPoints:       question.MaxPoints,
			IsMustPassMet:   mustPassMet,
		}
		submission.AddAnswer(submissionAnswer)

		// Update topic score
		if topic, exists := topicScores[question.TopicID]; exists {
			topic.Score += pointsEarned
			topic.MaxScore += question.MaxPoints
		}
	}

	// Add topic scores to submission
	for _, topic := range topicScores {
		if topic.MaxScore > 0 {
			submission.AddTopicScore(*topic)
		}
	}

	// Determine passing score
	passingScore := questionnaire.PassingScore
	if requirement.PassingScore != nil {
		passingScore = *requirement.PassingScore
	}

	// Calculate final scores
	submission.CalculateScores(passingScore)

	// Calculate completion time
	submission.CompletionTimeMinutes = int(time.Since(response.StartedAt).Minutes())

	// Submit
	submission.Submit()

	// Save submission
	if err := s.submissionRepo.Create(ctx, submission); err != nil {
		return nil, fmt.Errorf("failed to create submission: %w", err)
	}

	// Update response
	response.SetSubmission(submission.ID, submission.TotalScore, submission.MaxPossibleScore, submission.Passed)
	response.Submit()
	response.ClearDraftAnswers()

	if err := s.responseRepo.Update(ctx, response); err != nil {
		return nil, fmt.Errorf("failed to update response: %w", err)
	}

	// Update requirement status
	if err := requirement.Submit(supplierID); err == nil {
		_ = s.requirementRepo.Update(ctx, requirement)
	}

	return &SubmissionResult{
		Submission:  submission,
		Response:    response,
		Requirement: requirement,
		Passed:      submission.Passed,
		Score:       submission.TotalScore,
		MaxScore:    submission.MaxPossibleScore,
		Percentage:  submission.PercentageScore,
	}, nil
}

// GetSubmission retrieves a submission by ID
func (s *responseService) GetSubmission(ctx context.Context, submissionID primitive.ObjectID) (*models.QuestionnaireSubmission, error) {
	submission, err := s.submissionRepo.GetByID(ctx, submissionID)
	if err != nil {
		if errors.Is(err, models.ErrSubmissionNotFound) {
			return nil, ErrSubmissionNotFound
		}
		return nil, fmt.Errorf("failed to get submission: %w", err)
	}
	return submission, nil
}

// GetSubmissionByResponse retrieves a submission by response ID
func (s *responseService) GetSubmissionByResponse(ctx context.Context, responseID primitive.ObjectID) (*models.QuestionnaireSubmission, error) {
	submission, err := s.submissionRepo.GetByResponse(ctx, responseID)
	if err != nil {
		if errors.Is(err, models.ErrSubmissionNotFound) {
			return nil, ErrSubmissionNotFound
		}
		return nil, fmt.Errorf("failed to get submission: %w", err)
	}
	return submission, nil
}
