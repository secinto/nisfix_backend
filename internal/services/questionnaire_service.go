// Package services provides business logic implementations.
package services

import (
	"context"
	"errors"
	"fmt"

	"github.com/checkfix-tools/nisfix_backend/internal/models"
	"github.com/checkfix-tools/nisfix_backend/internal/repository"
	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Custom errors for questionnaire service
var (
	ErrQuestionnaireNotFound     = errors.New("questionnaire not found")
	ErrQuestionnaireNotEditable  = errors.New("questionnaire cannot be edited")
	ErrQuestionnaireNotDeletable = errors.New("questionnaire cannot be deleted")
	ErrTemplateNotFound          = errors.New("template not found")
	ErrQuestionNotFound          = errors.New("question not found")
	ErrInvalidQuestionType       = errors.New("invalid question type")
	ErrCannotPublish             = errors.New("cannot publish questionnaire")
)

// QuestionnaireService handles questionnaire business logic
// #INTEGRATION_POINT: Used by questionnaire handler for CRUD operations
type QuestionnaireService interface {
	// CreateQuestionnaire creates a new questionnaire from scratch
	CreateQuestionnaire(ctx context.Context, companyID primitive.ObjectID, req CreateQuestionnaireRequest) (*models.Questionnaire, error)

	// CreateFromTemplate creates a questionnaire from a template
	CreateFromTemplate(ctx context.Context, companyID primitive.ObjectID, templateID primitive.ObjectID, name string) (*models.Questionnaire, error)

	// GetQuestionnaire retrieves a questionnaire by ID
	GetQuestionnaire(ctx context.Context, id primitive.ObjectID, companyID *primitive.ObjectID) (*models.Questionnaire, error)

	// GetQuestionnaireWithQuestions retrieves a questionnaire with its questions
	GetQuestionnaireWithQuestions(ctx context.Context, id primitive.ObjectID, companyID *primitive.ObjectID) (*QuestionnaireWithQuestions, error)

	// ListQuestionnaires lists questionnaires for a company
	ListQuestionnaires(ctx context.Context, companyID primitive.ObjectID, filters QuestionnaireFilters, opts repository.PaginationOptions) (*repository.PaginatedResult[models.Questionnaire], error)

	// UpdateQuestionnaire updates questionnaire metadata
	UpdateQuestionnaire(ctx context.Context, id, companyID primitive.ObjectID, req UpdateQuestionnaireRequest) (*models.Questionnaire, error)

	// PublishQuestionnaire publishes a draft questionnaire
	PublishQuestionnaire(ctx context.Context, id, companyID primitive.ObjectID) (*models.Questionnaire, error)

	// ArchiveQuestionnaire archives a published questionnaire
	ArchiveQuestionnaire(ctx context.Context, id, companyID primitive.ObjectID) (*models.Questionnaire, error)

	// DeleteQuestionnaire deletes a draft questionnaire
	DeleteQuestionnaire(ctx context.Context, id, companyID primitive.ObjectID) error

	// AddQuestion adds a question to a questionnaire
	AddQuestion(ctx context.Context, questionnaireID, companyID primitive.ObjectID, req CreateQuestionRequest) (*models.Question, error)

	// UpdateQuestion updates a question
	UpdateQuestion(ctx context.Context, questionID, companyID primitive.ObjectID, req UpdateQuestionRequest) (*models.Question, error)

	// DeleteQuestion deletes a question from a questionnaire
	DeleteQuestion(ctx context.Context, questionID, companyID primitive.ObjectID) error

	// ReorderQuestions reorders questions in a questionnaire
	ReorderQuestions(ctx context.Context, questionnaireID, companyID primitive.ObjectID, questionOrders map[string]int) error

	// GetQuestionnaireStats returns questionnaire statistics for a company
	GetQuestionnaireStats(ctx context.Context, companyID primitive.ObjectID) (*QuestionnaireStats, error)
}

// CreateQuestionnaireRequest represents the request to create a questionnaire
type CreateQuestionnaireRequest struct {
	Name         string                      `json:"name" binding:"required"`
	Description  string                      `json:"description,omitempty"`
	PassingScore int                         `json:"passing_score,omitempty"`
	ScoringMode  models.ScoringMode          `json:"scoring_mode,omitempty"`
	Topics       []models.QuestionnaireTopic `json:"topics,omitempty"`
}

// UpdateQuestionnaireRequest represents the request to update a questionnaire
type UpdateQuestionnaireRequest struct {
	Name         *string                      `json:"name,omitempty"`
	Description  *string                      `json:"description,omitempty"`
	PassingScore *int                         `json:"passing_score,omitempty"`
	Topics       []models.QuestionnaireTopic `json:"topics,omitempty"`
}

// CreateQuestionRequest represents the request to create a question
type CreateQuestionRequest struct {
	TopicID     string                  `json:"topic_id,omitempty"`
	Text        string                  `json:"text" binding:"required"`
	Description string                  `json:"description,omitempty"`
	HelpText    string                  `json:"help_text,omitempty"`
	Type        models.QuestionType     `json:"type" binding:"required"`
	Weight      int                     `json:"weight,omitempty"`
	IsMustPass  bool                    `json:"is_must_pass,omitempty"`
	Options     []models.QuestionOption `json:"options,omitempty"`
}

// UpdateQuestionRequest represents the request to update a question
type UpdateQuestionRequest struct {
	TopicID     *string                  `json:"topic_id,omitempty"`
	Text        *string                  `json:"text,omitempty"`
	Description *string                  `json:"description,omitempty"`
	HelpText    *string                  `json:"help_text,omitempty"`
	Weight      *int                     `json:"weight,omitempty"`
	IsMustPass  *bool                    `json:"is_must_pass,omitempty"`
	Options     []models.QuestionOption `json:"options,omitempty"`
}

// QuestionnaireFilters contains filters for listing questionnaires
type QuestionnaireFilters struct {
	Status *models.QuestionnaireStatus
	Search string
}

// QuestionnaireWithQuestions combines questionnaire with its questions
type QuestionnaireWithQuestions struct {
	Questionnaire *models.Questionnaire `json:"questionnaire"`
	Questions     []models.Question     `json:"questions"`
}

// QuestionnaireStats contains questionnaire statistics
type QuestionnaireStats struct {
	Total     int64 `json:"total"`
	Draft     int64 `json:"draft"`
	Published int64 `json:"published"`
	Archived  int64 `json:"archived"`
}

// questionnaireService implements QuestionnaireService
type questionnaireService struct {
	questionnaireRepo repository.QuestionnaireRepository
	templateRepo      repository.QuestionnaireTemplateRepository
	questionRepo      repository.QuestionRepository
}

// NewQuestionnaireService creates a new questionnaire service
func NewQuestionnaireService(
	questionnaireRepo repository.QuestionnaireRepository,
	templateRepo repository.QuestionnaireTemplateRepository,
	questionRepo repository.QuestionRepository,
) QuestionnaireService {
	return &questionnaireService{
		questionnaireRepo: questionnaireRepo,
		templateRepo:      templateRepo,
		questionRepo:      questionRepo,
	}
}

// CreateQuestionnaire creates a new questionnaire from scratch
func (s *questionnaireService) CreateQuestionnaire(ctx context.Context, companyID primitive.ObjectID, req CreateQuestionnaireRequest) (*models.Questionnaire, error) {
	questionnaire := &models.Questionnaire{
		CompanyID:    companyID,
		Name:         req.Name,
		Description:  req.Description,
		PassingScore: req.PassingScore,
		ScoringMode:  req.ScoringMode,
		Topics:       req.Topics,
	}

	// Set defaults
	if questionnaire.PassingScore == 0 {
		questionnaire.PassingScore = 70
	}
	if questionnaire.ScoringMode == "" {
		questionnaire.ScoringMode = models.ScoringModePercentage
	}
	if questionnaire.Topics == nil {
		questionnaire.Topics = []models.QuestionnaireTopic{}
	}

	// Generate IDs for topics if not provided
	for i := range questionnaire.Topics {
		if questionnaire.Topics[i].ID == "" {
			questionnaire.Topics[i].ID = uuid.New().String()
		}
		if questionnaire.Topics[i].Order == 0 {
			questionnaire.Topics[i].Order = i + 1
		}
	}

	questionnaire.BeforeCreate()

	if err := s.questionnaireRepo.Create(ctx, questionnaire); err != nil {
		return nil, fmt.Errorf("failed to create questionnaire: %w", err)
	}

	return questionnaire, nil
}

// CreateFromTemplate creates a questionnaire from a template
// #BUSINESS_RULE: Template topics and default passing score are copied
func (s *questionnaireService) CreateFromTemplate(ctx context.Context, companyID primitive.ObjectID, templateID primitive.ObjectID, name string) (*models.Questionnaire, error) {
	// Get template
	template, err := s.templateRepo.GetByID(ctx, templateID)
	if err != nil {
		if errors.Is(err, models.ErrTemplateNotFound) {
			return nil, ErrTemplateNotFound
		}
		return nil, fmt.Errorf("failed to get template: %w", err)
	}

	// Create questionnaire from template
	questionnaire := &models.Questionnaire{
		CompanyID:    companyID,
		TemplateID:   &templateID,
		Name:         name,
		Description:  template.Description,
		PassingScore: template.DefaultPassingScore,
		ScoringMode:  models.ScoringModePercentage,
	}

	// Copy topics from template
	questionnaire.Topics = make([]models.QuestionnaireTopic, len(template.Topics))
	for i, t := range template.Topics {
		questionnaire.Topics[i] = models.QuestionnaireTopic{
			ID:          t.ID,
			Name:        t.Name,
			Description: t.Description,
			Order:       t.Order,
		}
	}

	questionnaire.BeforeCreate()

	if err := s.questionnaireRepo.Create(ctx, questionnaire); err != nil {
		return nil, fmt.Errorf("failed to create questionnaire: %w", err)
	}

	// Increment template usage count
	_ = s.templateRepo.IncrementUsageCount(ctx, templateID)

	return questionnaire, nil
}

// GetQuestionnaire retrieves a questionnaire by ID
func (s *questionnaireService) GetQuestionnaire(ctx context.Context, id primitive.ObjectID, companyID *primitive.ObjectID) (*models.Questionnaire, error) {
	questionnaire, err := s.questionnaireRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, models.ErrQuestionnaireNotFound) {
			return nil, ErrQuestionnaireNotFound
		}
		return nil, fmt.Errorf("failed to get questionnaire: %w", err)
	}

	// Verify company ownership if provided
	if companyID != nil && questionnaire.CompanyID != *companyID {
		return nil, ErrQuestionnaireNotFound
	}

	return questionnaire, nil
}

// GetQuestionnaireWithQuestions retrieves a questionnaire with its questions
func (s *questionnaireService) GetQuestionnaireWithQuestions(ctx context.Context, id primitive.ObjectID, companyID *primitive.ObjectID) (*QuestionnaireWithQuestions, error) {
	questionnaire, err := s.GetQuestionnaire(ctx, id, companyID)
	if err != nil {
		return nil, err
	}

	questions, err := s.questionRepo.ListByQuestionnaire(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get questions: %w", err)
	}

	return &QuestionnaireWithQuestions{
		Questionnaire: questionnaire,
		Questions:     questions,
	}, nil
}

// ListQuestionnaires lists questionnaires for a company
func (s *questionnaireService) ListQuestionnaires(ctx context.Context, companyID primitive.ObjectID, filters QuestionnaireFilters, opts repository.PaginationOptions) (*repository.PaginatedResult[models.Questionnaire], error) {
	return s.questionnaireRepo.ListByCompany(ctx, companyID, filters.Status, opts)
}

// UpdateQuestionnaire updates questionnaire metadata
// #BUSINESS_RULE: Only draft questionnaires can be edited
func (s *questionnaireService) UpdateQuestionnaire(ctx context.Context, id, companyID primitive.ObjectID, req UpdateQuestionnaireRequest) (*models.Questionnaire, error) {
	questionnaire, err := s.GetQuestionnaire(ctx, id, &companyID)
	if err != nil {
		return nil, err
	}

	if !questionnaire.CanBeEdited() {
		return nil, ErrQuestionnaireNotEditable
	}

	// Update fields if provided
	if req.Name != nil {
		questionnaire.Name = *req.Name
	}
	if req.Description != nil {
		questionnaire.Description = *req.Description
	}
	if req.PassingScore != nil {
		questionnaire.PassingScore = *req.PassingScore
	}
	if req.Topics != nil {
		// Generate IDs for new topics
		for i := range req.Topics {
			if req.Topics[i].ID == "" {
				req.Topics[i].ID = uuid.New().String()
			}
			if req.Topics[i].Order == 0 {
				req.Topics[i].Order = i + 1
			}
		}
		questionnaire.Topics = req.Topics
	}

	questionnaire.BeforeUpdate()

	if err := s.questionnaireRepo.Update(ctx, questionnaire); err != nil {
		return nil, fmt.Errorf("failed to update questionnaire: %w", err)
	}

	return questionnaire, nil
}

// PublishQuestionnaire publishes a draft questionnaire
// #BUSINESS_RULE: Questionnaire must have at least one question to be published
func (s *questionnaireService) PublishQuestionnaire(ctx context.Context, id, companyID primitive.ObjectID) (*models.Questionnaire, error) {
	questionnaire, err := s.GetQuestionnaire(ctx, id, &companyID)
	if err != nil {
		return nil, err
	}

	if !questionnaire.IsDraft() {
		return nil, ErrCannotPublish
	}

	// Check that questionnaire has at least one question
	count, err := s.questionRepo.CountByQuestionnaire(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to count questions: %w", err)
	}
	if count == 0 {
		return nil, ErrCannotPublish
	}

	// Update statistics before publishing
	maxScore, err := s.questionRepo.CalculateMaxScore(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate max score: %w", err)
	}
	questionnaire.UpdateStatistics(int(count), maxScore)

	if err := questionnaire.Publish(); err != nil {
		return nil, ErrCannotPublish
	}

	if err := s.questionnaireRepo.Update(ctx, questionnaire); err != nil {
		return nil, fmt.Errorf("failed to publish questionnaire: %w", err)
	}

	return questionnaire, nil
}

// ArchiveQuestionnaire archives a published questionnaire
func (s *questionnaireService) ArchiveQuestionnaire(ctx context.Context, id, companyID primitive.ObjectID) (*models.Questionnaire, error) {
	questionnaire, err := s.GetQuestionnaire(ctx, id, &companyID)
	if err != nil {
		return nil, err
	}

	if err := questionnaire.Archive(); err != nil {
		return nil, ErrInvalidStatusTransition
	}

	if err := s.questionnaireRepo.Update(ctx, questionnaire); err != nil {
		return nil, fmt.Errorf("failed to archive questionnaire: %w", err)
	}

	return questionnaire, nil
}

// DeleteQuestionnaire deletes a draft questionnaire
// #BUSINESS_RULE: Only draft questionnaires can be deleted
func (s *questionnaireService) DeleteQuestionnaire(ctx context.Context, id, companyID primitive.ObjectID) error {
	questionnaire, err := s.GetQuestionnaire(ctx, id, &companyID)
	if err != nil {
		return err
	}

	if !questionnaire.CanBeDeleted() {
		return ErrQuestionnaireNotDeletable
	}

	// Delete all questions first
	if _, err := s.questionRepo.DeleteByQuestionnaire(ctx, id); err != nil {
		return fmt.Errorf("failed to delete questions: %w", err)
	}

	if err := s.questionnaireRepo.Delete(ctx, id); err != nil {
		if errors.Is(err, models.ErrQuestionnaireNotDeletable) {
			return ErrQuestionnaireNotDeletable
		}
		return fmt.Errorf("failed to delete questionnaire: %w", err)
	}

	return nil
}

// AddQuestion adds a question to a questionnaire
// #BUSINESS_RULE: Questions can only be added to draft questionnaires
func (s *questionnaireService) AddQuestion(ctx context.Context, questionnaireID, companyID primitive.ObjectID, req CreateQuestionRequest) (*models.Question, error) {
	questionnaire, err := s.GetQuestionnaire(ctx, questionnaireID, &companyID)
	if err != nil {
		return nil, err
	}

	if !questionnaire.CanBeEdited() {
		return nil, ErrQuestionnaireNotEditable
	}

	if !req.Type.IsValid() {
		return nil, ErrInvalidQuestionType
	}

	// Get current question count for ordering
	count, err := s.questionRepo.CountByQuestionnaire(ctx, questionnaireID)
	if err != nil {
		return nil, fmt.Errorf("failed to count questions: %w", err)
	}

	// Generate option IDs if not provided
	for i := range req.Options {
		if req.Options[i].ID == "" {
			req.Options[i].ID = uuid.New().String()
		}
		if req.Options[i].Order == 0 {
			req.Options[i].Order = i + 1
		}
	}

	question := &models.Question{
		QuestionnaireID: questionnaireID,
		TopicID:         req.TopicID,
		Text:            req.Text,
		Description:     req.Description,
		HelpText:        req.HelpText,
		Type:            req.Type,
		Order:           int(count) + 1,
		Weight:          req.Weight,
		IsMustPass:      req.IsMustPass,
		Options:         req.Options,
	}

	question.BeforeCreate()

	if err := s.questionRepo.Create(ctx, question); err != nil {
		return nil, fmt.Errorf("failed to create question: %w", err)
	}

	// Update questionnaire statistics
	s.updateQuestionnaireStats(ctx, questionnaireID)

	return question, nil
}

// UpdateQuestion updates a question
func (s *questionnaireService) UpdateQuestion(ctx context.Context, questionID, companyID primitive.ObjectID, req UpdateQuestionRequest) (*models.Question, error) {
	question, err := s.questionRepo.GetByID(ctx, questionID)
	if err != nil {
		if errors.Is(err, models.ErrQuestionNotFound) {
			return nil, ErrQuestionNotFound
		}
		return nil, fmt.Errorf("failed to get question: %w", err)
	}

	// Verify company ownership through questionnaire
	questionnaire, err := s.GetQuestionnaire(ctx, question.QuestionnaireID, &companyID)
	if err != nil {
		return nil, ErrQuestionNotFound
	}

	if !questionnaire.CanBeEdited() {
		return nil, ErrQuestionnaireNotEditable
	}

	// Update fields if provided
	if req.TopicID != nil {
		question.TopicID = *req.TopicID
	}
	if req.Text != nil {
		question.Text = *req.Text
	}
	if req.Description != nil {
		question.Description = *req.Description
	}
	if req.HelpText != nil {
		question.HelpText = *req.HelpText
	}
	if req.Weight != nil {
		question.Weight = *req.Weight
	}
	if req.IsMustPass != nil {
		question.IsMustPass = *req.IsMustPass
	}
	if req.Options != nil {
		// Generate option IDs if not provided
		for i := range req.Options {
			if req.Options[i].ID == "" {
				req.Options[i].ID = uuid.New().String()
			}
			if req.Options[i].Order == 0 {
				req.Options[i].Order = i + 1
			}
		}
		question.Options = req.Options
		question.RecalculateMaxPoints()
	}

	question.BeforeUpdate()

	if err := s.questionRepo.Update(ctx, question); err != nil {
		return nil, fmt.Errorf("failed to update question: %w", err)
	}

	// Update questionnaire statistics
	s.updateQuestionnaireStats(ctx, question.QuestionnaireID)

	return question, nil
}

// DeleteQuestion deletes a question from a questionnaire
func (s *questionnaireService) DeleteQuestion(ctx context.Context, questionID, companyID primitive.ObjectID) error {
	question, err := s.questionRepo.GetByID(ctx, questionID)
	if err != nil {
		if errors.Is(err, models.ErrQuestionNotFound) {
			return ErrQuestionNotFound
		}
		return fmt.Errorf("failed to get question: %w", err)
	}

	// Verify company ownership through questionnaire
	questionnaire, err := s.GetQuestionnaire(ctx, question.QuestionnaireID, &companyID)
	if err != nil {
		return ErrQuestionNotFound
	}

	if !questionnaire.CanBeEdited() {
		return ErrQuestionnaireNotEditable
	}

	questionnaireID := question.QuestionnaireID

	if err := s.questionRepo.Delete(ctx, questionID); err != nil {
		return fmt.Errorf("failed to delete question: %w", err)
	}

	// Update questionnaire statistics
	s.updateQuestionnaireStats(ctx, questionnaireID)

	return nil
}

// ReorderQuestions reorders questions in a questionnaire
func (s *questionnaireService) ReorderQuestions(ctx context.Context, questionnaireID, companyID primitive.ObjectID, questionOrders map[string]int) error {
	questionnaire, err := s.GetQuestionnaire(ctx, questionnaireID, &companyID)
	if err != nil {
		return err
	}

	if !questionnaire.CanBeEdited() {
		return ErrQuestionnaireNotEditable
	}

	// Convert string IDs to ObjectIDs
	orders := make(map[primitive.ObjectID]int)
	for idStr, order := range questionOrders {
		id, err := primitive.ObjectIDFromHex(idStr)
		if err != nil {
			continue
		}
		orders[id] = order
	}

	if err := s.questionRepo.UpdateOrder(ctx, questionnaireID, orders); err != nil {
		return fmt.Errorf("failed to reorder questions: %w", err)
	}

	return nil
}

// GetQuestionnaireStats returns questionnaire statistics for a company
func (s *questionnaireService) GetQuestionnaireStats(ctx context.Context, companyID primitive.ObjectID) (*QuestionnaireStats, error) {
	total, err := s.questionnaireRepo.CountByCompany(ctx, companyID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to count total: %w", err)
	}

	draftStatus := models.QuestionnaireStatusDraft
	draft, err := s.questionnaireRepo.CountByCompany(ctx, companyID, &draftStatus)
	if err != nil {
		return nil, fmt.Errorf("failed to count draft: %w", err)
	}

	publishedStatus := models.QuestionnaireStatusPublished
	published, err := s.questionnaireRepo.CountByCompany(ctx, companyID, &publishedStatus)
	if err != nil {
		return nil, fmt.Errorf("failed to count published: %w", err)
	}

	archivedStatus := models.QuestionnaireStatusArchived
	archived, err := s.questionnaireRepo.CountByCompany(ctx, companyID, &archivedStatus)
	if err != nil {
		return nil, fmt.Errorf("failed to count archived: %w", err)
	}

	return &QuestionnaireStats{
		Total:     total,
		Draft:     draft,
		Published: published,
		Archived:  archived,
	}, nil
}

// updateQuestionnaireStats updates the questionnaire's denormalized statistics
func (s *questionnaireService) updateQuestionnaireStats(ctx context.Context, questionnaireID primitive.ObjectID) {
	count, err := s.questionRepo.CountByQuestionnaire(ctx, questionnaireID)
	if err != nil {
		return
	}

	maxScore, err := s.questionRepo.CalculateMaxScore(ctx, questionnaireID)
	if err != nil {
		return
	}

	_ = s.questionnaireRepo.UpdateStatistics(ctx, questionnaireID, int(count), maxScore)
}
