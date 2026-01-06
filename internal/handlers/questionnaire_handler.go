// Package handlers provides HTTP handlers for API endpoints.
package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/checkfix-tools/nisfix_backend/internal/middleware"
	"github.com/checkfix-tools/nisfix_backend/internal/models"
	"github.com/checkfix-tools/nisfix_backend/internal/repository"
	"github.com/checkfix-tools/nisfix_backend/internal/services"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// QuestionnaireHandler handles questionnaire endpoints
// #INTEGRATION_POINT: Company portal uses these endpoints for questionnaire management
type QuestionnaireHandler struct {
	questionnaireService services.QuestionnaireService
}

// NewQuestionnaireHandler creates a new questionnaire handler
func NewQuestionnaireHandler(questionnaireService services.QuestionnaireService) *QuestionnaireHandler {
	return &QuestionnaireHandler{
		questionnaireService: questionnaireService,
	}
}

// CreateQuestionnaireRequest represents the create questionnaire request body
type CreateQuestionnaireRequest struct {
	Name         string  `json:"name" binding:"required"`
	Description  string  `json:"description,omitempty"`
	PassingScore int     `json:"passing_score,omitempty"`
	ScoringMode  string  `json:"scoring_mode,omitempty"`
	TemplateID   *string `json:"template_id,omitempty"`
	Topics       []TopicRequest `json:"topics,omitempty"`
}

// TopicRequest represents a topic in requests
type TopicRequest struct {
	ID          string `json:"id,omitempty"`
	Name        string `json:"name" binding:"required"`
	Description string `json:"description,omitempty"`
	Order       int    `json:"order,omitempty"`
}

// QuestionnaireResponse represents a questionnaire in API responses
type QuestionnaireResponse struct {
	ID               string          `json:"id"`
	CompanyID        string          `json:"company_id"`
	TemplateID       *string         `json:"template_id,omitempty"`
	Name             string          `json:"name"`
	Description      string          `json:"description,omitempty"`
	Status           string          `json:"status"`
	Version          int             `json:"version"`
	PassingScore     int             `json:"passing_score"`
	ScoringMode      string          `json:"scoring_mode"`
	Topics           []TopicResponse `json:"topics"`
	QuestionCount    int             `json:"question_count"`
	MaxPossibleScore int             `json:"max_possible_score"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
	PublishedAt      *time.Time      `json:"published_at,omitempty"`
}

// TopicResponse represents a topic in responses
type TopicResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Order       int    `json:"order"`
}

// QuestionResponse represents a question in API responses
type QuestionResponse struct {
	ID              string           `json:"id"`
	QuestionnaireID string           `json:"questionnaire_id"`
	TopicID         string           `json:"topic_id,omitempty"`
	Text            string           `json:"text"`
	Description     string           `json:"description,omitempty"`
	HelpText        string           `json:"help_text,omitempty"`
	Type            string           `json:"type"`
	Order           int              `json:"order"`
	Weight          int              `json:"weight"`
	MaxPoints       int              `json:"max_points"`
	IsMustPass      bool             `json:"is_must_pass"`
	Options         []OptionResponse `json:"options,omitempty"`
	CreatedAt       time.Time        `json:"created_at"`
	UpdatedAt       time.Time        `json:"updated_at"`
}

// OptionResponse represents an option in API responses
type OptionResponse struct {
	ID        string `json:"id"`
	Text      string `json:"text"`
	Points    int    `json:"points"`
	IsCorrect bool   `json:"is_correct"`
	Order     int    `json:"order"`
}

// QuestionnaireWithQuestionsResponse represents questionnaire with questions
type QuestionnaireWithQuestionsResponse struct {
	Questionnaire QuestionnaireResponse `json:"questionnaire"`
	Questions     []QuestionResponse    `json:"questions"`
}

// PaginatedQuestionnairesResponse represents paginated questionnaires
type PaginatedQuestionnairesResponse struct {
	Items      []QuestionnaireResponse `json:"items"`
	TotalCount int64                   `json:"total_count"`
	Page       int                     `json:"page"`
	Limit      int                     `json:"limit"`
	TotalPages int                     `json:"total_pages"`
}

// QuestionnaireStatsResponse represents questionnaire statistics
type QuestionnaireStatsResponse struct {
	Total     int64 `json:"total"`
	Draft     int64 `json:"draft"`
	Published int64 `json:"published"`
	Archived  int64 `json:"archived"`
}

// CreateQuestionnaire handles POST /api/v1/questionnaires
// @Summary Create a questionnaire
// @Description Creates a new questionnaire, optionally from a template
// @Tags Questionnaires
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body CreateQuestionnaireRequest true "Create request"
// @Success 201 {object} QuestionnaireResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /questionnaires [post]
func (h *QuestionnaireHandler) CreateQuestionnaire(c *gin.Context) {
	companyID, ok := middleware.GetOrgID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "Invalid session",
		})
		return
	}

	var req CreateQuestionnaireRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: "Name is required",
		})
		return
	}

	var questionnaire *models.Questionnaire
	var err error

	if req.TemplateID != nil {
		// Create from template
		templateID, parseErr := primitive.ObjectIDFromHex(*req.TemplateID)
		if parseErr != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "invalid_template_id",
				Message: "Invalid template ID",
			})
			return
		}
		questionnaire, err = h.questionnaireService.CreateFromTemplate(c.Request.Context(), companyID, templateID, req.Name)
	} else {
		// Create from scratch
		topics := make([]models.QuestionnaireTopic, len(req.Topics))
		for i, t := range req.Topics {
			topics[i] = models.QuestionnaireTopic{
				ID:          t.ID,
				Name:        t.Name,
				Description: t.Description,
				Order:       t.Order,
			}
		}

		scoringMode := models.ScoringModePercentage
		if req.ScoringMode != "" {
			scoringMode = models.ScoringMode(req.ScoringMode)
		}

		serviceReq := services.CreateQuestionnaireRequest{
			Name:         req.Name,
			Description:  req.Description,
			PassingScore: req.PassingScore,
			ScoringMode:  scoringMode,
			Topics:       topics,
		}
		questionnaire, err = h.questionnaireService.CreateQuestionnaire(c.Request.Context(), companyID, serviceReq)
	}

	if err != nil {
		if errors.Is(err, services.ErrTemplateNotFound) {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "template_not_found",
				Message: "Template not found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to create questionnaire",
		})
		return
	}

	c.JSON(http.StatusCreated, toQuestionnaireResponse(questionnaire))
}

// ListQuestionnaires handles GET /api/v1/questionnaires
// @Summary List questionnaires
// @Description Lists all questionnaires for the company
// @Tags Questionnaires
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param status query string false "Filter by status"
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20)
// @Success 200 {object} PaginatedQuestionnairesResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Router /questionnaires [get]
func (h *QuestionnaireHandler) ListQuestionnaires(c *gin.Context) {
	companyID, ok := middleware.GetOrgID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "Invalid session",
		})
		return
	}

	// Parse query parameters
	filters := services.QuestionnaireFilters{}
	if status := c.Query("status"); status != "" {
		s := models.QuestionnaireStatus(status)
		filters.Status = &s
	}
	filters.Search = c.Query("search")

	opts := repository.DefaultPaginationOptions()
	if page, err := strconv.Atoi(c.Query("page")); err == nil && page > 0 {
		opts.Page = page
	}
	if limit, err := strconv.Atoi(c.Query("limit")); err == nil && limit > 0 && limit <= 100 {
		opts.Limit = limit
	}
	if sortBy := c.Query("sort_by"); sortBy != "" {
		opts.SortBy = sortBy
	}
	if sortDir := c.Query("sort_dir"); sortDir == "asc" {
		opts.SortDir = 1
	}

	result, err := h.questionnaireService.ListQuestionnaires(c.Request.Context(), companyID, filters, opts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to list questionnaires",
		})
		return
	}

	items := make([]QuestionnaireResponse, len(result.Items))
	for i, q := range result.Items {
		items[i] = toQuestionnaireResponse(&q)
	}

	c.JSON(http.StatusOK, PaginatedQuestionnairesResponse{
		Items:      items,
		TotalCount: result.TotalCount,
		Page:       result.Page,
		Limit:      result.Limit,
		TotalPages: result.TotalPages,
	})
}

// GetQuestionnaire handles GET /api/v1/questionnaires/:id
// @Summary Get questionnaire
// @Description Gets a questionnaire with its questions
// @Tags Questionnaires
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Questionnaire ID"
// @Success 200 {object} QuestionnaireWithQuestionsResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /questionnaires/{id} [get]
func (h *QuestionnaireHandler) GetQuestionnaire(c *gin.Context) {
	companyID, ok := middleware.GetOrgID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "Invalid session",
		})
		return
	}

	questionnaireID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid questionnaire ID",
		})
		return
	}

	result, err := h.questionnaireService.GetQuestionnaireWithQuestions(c.Request.Context(), questionnaireID, &companyID)
	if err != nil {
		if errors.Is(err, services.ErrQuestionnaireNotFound) {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "not_found",
				Message: "Questionnaire not found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to get questionnaire",
		})
		return
	}

	questions := make([]QuestionResponse, len(result.Questions))
	for i, q := range result.Questions {
		questions[i] = toQuestionResponse(&q)
	}

	c.JSON(http.StatusOK, QuestionnaireWithQuestionsResponse{
		Questionnaire: toQuestionnaireResponse(result.Questionnaire),
		Questions:     questions,
	})
}

// UpdateQuestionnaireRequest represents the update questionnaire request
type UpdateQuestionnaireRequest struct {
	Name         *string        `json:"name,omitempty"`
	Description  *string        `json:"description,omitempty"`
	PassingScore *int           `json:"passing_score,omitempty"`
	Topics       []TopicRequest `json:"topics,omitempty"`
}

// UpdateQuestionnaire handles PATCH /api/v1/questionnaires/:id
// @Summary Update questionnaire
// @Description Updates a questionnaire (draft only)
// @Tags Questionnaires
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Questionnaire ID"
// @Param request body UpdateQuestionnaireRequest true "Update request"
// @Success 200 {object} QuestionnaireResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /questionnaires/{id} [patch]
func (h *QuestionnaireHandler) UpdateQuestionnaire(c *gin.Context) {
	companyID, ok := middleware.GetOrgID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "Invalid session",
		})
		return
	}

	questionnaireID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid questionnaire ID",
		})
		return
	}

	var req UpdateQuestionnaireRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid request body",
		})
		return
	}

	var topics []models.QuestionnaireTopic
	if req.Topics != nil {
		topics = make([]models.QuestionnaireTopic, len(req.Topics))
		for i, t := range req.Topics {
			topics[i] = models.QuestionnaireTopic{
				ID:          t.ID,
				Name:        t.Name,
				Description: t.Description,
				Order:       t.Order,
			}
		}
	}

	serviceReq := services.UpdateQuestionnaireRequest{
		Name:         req.Name,
		Description:  req.Description,
		PassingScore: req.PassingScore,
		Topics:       topics,
	}

	questionnaire, err := h.questionnaireService.UpdateQuestionnaire(c.Request.Context(), questionnaireID, companyID, serviceReq)
	if err != nil {
		if errors.Is(err, services.ErrQuestionnaireNotFound) {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "not_found",
				Message: "Questionnaire not found",
			})
			return
		}
		if errors.Is(err, services.ErrQuestionnaireNotEditable) {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "not_editable",
				Message: "Only draft questionnaires can be edited",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to update questionnaire",
		})
		return
	}

	c.JSON(http.StatusOK, toQuestionnaireResponse(questionnaire))
}

// PublishQuestionnaire handles POST /api/v1/questionnaires/:id/publish
// @Summary Publish questionnaire
// @Description Publishes a draft questionnaire
// @Tags Questionnaires
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Questionnaire ID"
// @Success 200 {object} QuestionnaireResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /questionnaires/{id}/publish [post]
func (h *QuestionnaireHandler) PublishQuestionnaire(c *gin.Context) {
	companyID, ok := middleware.GetOrgID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "Invalid session",
		})
		return
	}

	questionnaireID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid questionnaire ID",
		})
		return
	}

	questionnaire, err := h.questionnaireService.PublishQuestionnaire(c.Request.Context(), questionnaireID, companyID)
	if err != nil {
		if errors.Is(err, services.ErrQuestionnaireNotFound) {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "not_found",
				Message: "Questionnaire not found",
			})
			return
		}
		if errors.Is(err, services.ErrCannotPublish) {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "cannot_publish",
				Message: "Cannot publish: questionnaire must be in draft status and have at least one question",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to publish questionnaire",
		})
		return
	}

	c.JSON(http.StatusOK, toQuestionnaireResponse(questionnaire))
}

// ArchiveQuestionnaire handles POST /api/v1/questionnaires/:id/archive
// @Summary Archive questionnaire
// @Description Archives a published questionnaire
// @Tags Questionnaires
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Questionnaire ID"
// @Success 200 {object} QuestionnaireResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /questionnaires/{id}/archive [post]
func (h *QuestionnaireHandler) ArchiveQuestionnaire(c *gin.Context) {
	companyID, ok := middleware.GetOrgID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "Invalid session",
		})
		return
	}

	questionnaireID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid questionnaire ID",
		})
		return
	}

	questionnaire, err := h.questionnaireService.ArchiveQuestionnaire(c.Request.Context(), questionnaireID, companyID)
	if err != nil {
		if errors.Is(err, services.ErrQuestionnaireNotFound) {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "not_found",
				Message: "Questionnaire not found",
			})
			return
		}
		if errors.Is(err, services.ErrInvalidStatusTransition) {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "invalid_transition",
				Message: "Only published questionnaires can be archived",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to archive questionnaire",
		})
		return
	}

	c.JSON(http.StatusOK, toQuestionnaireResponse(questionnaire))
}

// DeleteQuestionnaire handles DELETE /api/v1/questionnaires/:id
// @Summary Delete questionnaire
// @Description Deletes a draft questionnaire
// @Tags Questionnaires
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Questionnaire ID"
// @Success 204 "No Content"
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /questionnaires/{id} [delete]
func (h *QuestionnaireHandler) DeleteQuestionnaire(c *gin.Context) {
	companyID, ok := middleware.GetOrgID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "Invalid session",
		})
		return
	}

	questionnaireID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid questionnaire ID",
		})
		return
	}

	err = h.questionnaireService.DeleteQuestionnaire(c.Request.Context(), questionnaireID, companyID)
	if err != nil {
		if errors.Is(err, services.ErrQuestionnaireNotFound) {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "not_found",
				Message: "Questionnaire not found",
			})
			return
		}
		if errors.Is(err, services.ErrQuestionnaireNotDeletable) {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "not_deletable",
				Message: "Only draft questionnaires can be deleted",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to delete questionnaire",
		})
		return
	}

	c.Status(http.StatusNoContent)
}

// CreateQuestionAPIRequest represents the create question request body
type CreateQuestionAPIRequest struct {
	TopicID     string          `json:"topic_id,omitempty"`
	Text        string          `json:"text" binding:"required"`
	Description string          `json:"description,omitempty"`
	HelpText    string          `json:"help_text,omitempty"`
	Type        string          `json:"type" binding:"required"`
	Weight      int             `json:"weight,omitempty"`
	IsMustPass  bool            `json:"is_must_pass,omitempty"`
	Options     []OptionRequest `json:"options,omitempty"`
}

// OptionRequest represents an option in requests
type OptionRequest struct {
	ID        string `json:"id,omitempty"`
	Text      string `json:"text" binding:"required"`
	Points    int    `json:"points,omitempty"`
	IsCorrect bool   `json:"is_correct,omitempty"`
	Order     int    `json:"order,omitempty"`
}

// AddQuestion handles POST /api/v1/questionnaires/:id/questions
// @Summary Add question to questionnaire
// @Description Adds a question to a draft questionnaire
// @Tags Questionnaires
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Questionnaire ID"
// @Param request body CreateQuestionAPIRequest true "Question to add"
// @Success 201 {object} QuestionResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /questionnaires/{id}/questions [post]
func (h *QuestionnaireHandler) AddQuestion(c *gin.Context) {
	companyID, ok := middleware.GetOrgID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "Invalid session",
		})
		return
	}

	questionnaireID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid questionnaire ID",
		})
		return
	}

	var req CreateQuestionAPIRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: "Text and type are required",
		})
		return
	}

	options := make([]models.QuestionOption, len(req.Options))
	for i, o := range req.Options {
		options[i] = models.QuestionOption{
			ID:        o.ID,
			Text:      o.Text,
			Points:    o.Points,
			IsCorrect: o.IsCorrect,
			Order:     o.Order,
		}
	}

	serviceReq := services.CreateQuestionRequest{
		TopicID:     req.TopicID,
		Text:        req.Text,
		Description: req.Description,
		HelpText:    req.HelpText,
		Type:        models.QuestionType(req.Type),
		Weight:      req.Weight,
		IsMustPass:  req.IsMustPass,
		Options:     options,
	}

	question, err := h.questionnaireService.AddQuestion(c.Request.Context(), questionnaireID, companyID, serviceReq)
	if err != nil {
		if errors.Is(err, services.ErrQuestionnaireNotFound) {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "not_found",
				Message: "Questionnaire not found",
			})
			return
		}
		if errors.Is(err, services.ErrQuestionnaireNotEditable) {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "not_editable",
				Message: "Only draft questionnaires can be edited",
			})
			return
		}
		if errors.Is(err, services.ErrInvalidQuestionType) {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "invalid_type",
				Message: "Invalid question type",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to add question",
		})
		return
	}

	c.JSON(http.StatusCreated, toQuestionResponse(question))
}

// UpdateQuestionAPIRequest represents the update question request
type UpdateQuestionAPIRequest struct {
	TopicID     *string         `json:"topic_id,omitempty"`
	Text        *string         `json:"text,omitempty"`
	Description *string         `json:"description,omitempty"`
	HelpText    *string         `json:"help_text,omitempty"`
	Weight      *int            `json:"weight,omitempty"`
	IsMustPass  *bool           `json:"is_must_pass,omitempty"`
	Options     []OptionRequest `json:"options,omitempty"`
}

// UpdateQuestion handles PATCH /api/v1/questions/:id
// @Summary Update question
// @Description Updates a question in a draft questionnaire
// @Tags Questionnaires
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Question ID"
// @Param request body UpdateQuestionAPIRequest true "Update request"
// @Success 200 {object} QuestionResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /questions/{id} [patch]
func (h *QuestionnaireHandler) UpdateQuestion(c *gin.Context) {
	companyID, ok := middleware.GetOrgID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "Invalid session",
		})
		return
	}

	questionID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid question ID",
		})
		return
	}

	var req UpdateQuestionAPIRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid request body",
		})
		return
	}

	var options []models.QuestionOption
	if req.Options != nil {
		options = make([]models.QuestionOption, len(req.Options))
		for i, o := range req.Options {
			options[i] = models.QuestionOption{
				ID:        o.ID,
				Text:      o.Text,
				Points:    o.Points,
				IsCorrect: o.IsCorrect,
				Order:     o.Order,
			}
		}
	}

	serviceReq := services.UpdateQuestionRequest{
		TopicID:     req.TopicID,
		Text:        req.Text,
		Description: req.Description,
		HelpText:    req.HelpText,
		Weight:      req.Weight,
		IsMustPass:  req.IsMustPass,
		Options:     options,
	}

	question, err := h.questionnaireService.UpdateQuestion(c.Request.Context(), questionID, companyID, serviceReq)
	if err != nil {
		if errors.Is(err, services.ErrQuestionNotFound) {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "not_found",
				Message: "Question not found",
			})
			return
		}
		if errors.Is(err, services.ErrQuestionnaireNotEditable) {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "not_editable",
				Message: "Only draft questionnaires can be edited",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to update question",
		})
		return
	}

	c.JSON(http.StatusOK, toQuestionResponse(question))
}

// DeleteQuestion handles DELETE /api/v1/questions/:id
// @Summary Delete question
// @Description Deletes a question from a draft questionnaire
// @Tags Questionnaires
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Question ID"
// @Success 204 "No Content"
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /questions/{id} [delete]
func (h *QuestionnaireHandler) DeleteQuestion(c *gin.Context) {
	companyID, ok := middleware.GetOrgID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "Invalid session",
		})
		return
	}

	questionID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid question ID",
		})
		return
	}

	err = h.questionnaireService.DeleteQuestion(c.Request.Context(), questionID, companyID)
	if err != nil {
		if errors.Is(err, services.ErrQuestionNotFound) {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "not_found",
				Message: "Question not found",
			})
			return
		}
		if errors.Is(err, services.ErrQuestionnaireNotEditable) {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "not_editable",
				Message: "Only draft questionnaires can be edited",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to delete question",
		})
		return
	}

	c.Status(http.StatusNoContent)
}

// ReorderQuestionsRequest represents the reorder questions request
type ReorderQuestionsRequest struct {
	Orders map[string]int `json:"orders" binding:"required"`
}

// ReorderQuestions handles POST /api/v1/questionnaires/:id/questions/reorder
// @Summary Reorder questions
// @Description Reorders questions in a draft questionnaire
// @Tags Questionnaires
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Questionnaire ID"
// @Param request body ReorderQuestionsRequest true "Question orders"
// @Success 200 {object} map[string]string
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /questionnaires/{id}/questions/reorder [post]
func (h *QuestionnaireHandler) ReorderQuestions(c *gin.Context) {
	companyID, ok := middleware.GetOrgID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "Invalid session",
		})
		return
	}

	questionnaireID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid questionnaire ID",
		})
		return
	}

	var req ReorderQuestionsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: "Orders map is required",
		})
		return
	}

	err = h.questionnaireService.ReorderQuestions(c.Request.Context(), questionnaireID, companyID, req.Orders)
	if err != nil {
		if errors.Is(err, services.ErrQuestionnaireNotFound) {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "not_found",
				Message: "Questionnaire not found",
			})
			return
		}
		if errors.Is(err, services.ErrQuestionnaireNotEditable) {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "not_editable",
				Message: "Only draft questionnaires can be edited",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to reorder questions",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Questions reordered successfully"})
}

// GetQuestionnaireStats handles GET /api/v1/questionnaires/stats
// @Summary Get questionnaire statistics
// @Description Gets questionnaire statistics for the company
// @Tags Questionnaires
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} QuestionnaireStatsResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Router /questionnaires/stats [get]
func (h *QuestionnaireHandler) GetQuestionnaireStats(c *gin.Context) {
	companyID, ok := middleware.GetOrgID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "Invalid session",
		})
		return
	}

	stats, err := h.questionnaireService.GetQuestionnaireStats(c.Request.Context(), companyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to get questionnaire stats",
		})
		return
	}

	c.JSON(http.StatusOK, QuestionnaireStatsResponse{
		Total:     stats.Total,
		Draft:     stats.Draft,
		Published: stats.Published,
		Archived:  stats.Archived,
	})
}

// RegisterRoutes registers questionnaire handler routes
// #INTEGRATION_POINT: Routes require authentication and company organization type
func (h *QuestionnaireHandler) RegisterRoutes(rg *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	questionnaires := rg.Group("/questionnaires")
	questionnaires.Use(authMiddleware)
	questionnaires.Use(middleware.RequireCompany())
	{
		questionnaires.POST("", h.CreateQuestionnaire)
		questionnaires.GET("", h.ListQuestionnaires)
		questionnaires.GET("/stats", h.GetQuestionnaireStats)
		questionnaires.GET("/:id", h.GetQuestionnaire)
		questionnaires.PATCH("/:id", h.UpdateQuestionnaire)
		questionnaires.DELETE("/:id", h.DeleteQuestionnaire)
		questionnaires.POST("/:id/publish", h.PublishQuestionnaire)
		questionnaires.POST("/:id/archive", h.ArchiveQuestionnaire)
		questionnaires.POST("/:id/questions", h.AddQuestion)
		questionnaires.POST("/:id/questions/reorder", h.ReorderQuestions)
	}

	// Question routes (not nested under questionnaires for simpler URLs)
	questions := rg.Group("/questions")
	questions.Use(authMiddleware)
	questions.Use(middleware.RequireCompany())
	{
		questions.PATCH("/:id", h.UpdateQuestion)
		questions.DELETE("/:id", h.DeleteQuestion)
	}
}

// toQuestionnaireResponse converts a questionnaire model to response
func toQuestionnaireResponse(q *models.Questionnaire) QuestionnaireResponse {
	resp := QuestionnaireResponse{
		ID:               q.ID.Hex(),
		CompanyID:        q.CompanyID.Hex(),
		Name:             q.Name,
		Description:      q.Description,
		Status:           string(q.Status),
		Version:          q.Version,
		PassingScore:     q.PassingScore,
		ScoringMode:      string(q.ScoringMode),
		QuestionCount:    q.QuestionCount,
		MaxPossibleScore: q.MaxPossibleScore,
		CreatedAt:        q.CreatedAt,
		UpdatedAt:        q.UpdatedAt,
		PublishedAt:      q.PublishedAt,
	}

	if q.TemplateID != nil {
		templateID := q.TemplateID.Hex()
		resp.TemplateID = &templateID
	}

	resp.Topics = make([]TopicResponse, len(q.Topics))
	for i, t := range q.Topics {
		resp.Topics[i] = TopicResponse{
			ID:          t.ID,
			Name:        t.Name,
			Description: t.Description,
			Order:       t.Order,
		}
	}

	return resp
}

// toQuestionResponse converts a question model to response
func toQuestionResponse(q *models.Question) QuestionResponse {
	resp := QuestionResponse{
		ID:              q.ID.Hex(),
		QuestionnaireID: q.QuestionnaireID.Hex(),
		TopicID:         q.TopicID,
		Text:            q.Text,
		Description:     q.Description,
		HelpText:        q.HelpText,
		Type:            string(q.Type),
		Order:           q.Order,
		Weight:          q.Weight,
		MaxPoints:       q.MaxPoints,
		IsMustPass:      q.IsMustPass,
		CreatedAt:       q.CreatedAt,
		UpdatedAt:       q.UpdatedAt,
	}

	resp.Options = make([]OptionResponse, len(q.Options))
	for i, o := range q.Options {
		resp.Options[i] = OptionResponse{
			ID:        o.ID,
			Text:      o.Text,
			Points:    o.Points,
			IsCorrect: o.IsCorrect,
			Order:     o.Order,
		}
	}

	return resp
}
