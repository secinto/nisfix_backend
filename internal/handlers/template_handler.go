// Package handlers provides HTTP handlers for API endpoints.
package handlers

import (
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/checkfix-tools/nisfix_backend/internal/middleware"
	"github.com/checkfix-tools/nisfix_backend/internal/models"
	"github.com/checkfix-tools/nisfix_backend/internal/repository"
	"github.com/checkfix-tools/nisfix_backend/internal/services"
)

// TemplateHandler handles questionnaire template endpoints
// #INTEGRATION_POINT: Company portal uses these endpoints for template browsing
type TemplateHandler struct {
	templateRepo    repository.QuestionnaireTemplateRepository
	templateService services.TemplateService
}

// NewTemplateHandler creates a new template handler
func NewTemplateHandler(templateRepo repository.QuestionnaireTemplateRepository, templateService services.TemplateService) *TemplateHandler {
	return &TemplateHandler{
		templateRepo:    templateRepo,
		templateService: templateService,
	}
}

// TemplateResponse represents a template in API responses
type TemplateResponse struct {
	ID                  string                  `json:"id"`
	Name                string                  `json:"name"`
	Description         string                  `json:"description,omitempty"`
	Category            string                  `json:"category"`
	Version             string                  `json:"version"`
	IsSystem            bool                    `json:"is_system"`
	Visibility          string                  `json:"visibility"`
	DefaultPassingScore int                     `json:"default_passing_score"`
	EstimatedMinutes    int                     `json:"estimated_minutes"`
	Topics              []TemplateTopicResponse `json:"topics"`
	Tags                []string                `json:"tags,omitempty"`
	UsageCount          int                     `json:"usage_count"`
	CreatedByOrgID      string                  `json:"created_by_org_id,omitempty"`
	CreatedByUser       string                  `json:"created_by_user,omitempty"`
	CreatedAt           time.Time               `json:"created_at"`
	UpdatedAt           time.Time               `json:"updated_at"`
	PublishedAt         *time.Time              `json:"published_at,omitempty"`
}

// CreateTemplateAPIRequest represents a template creation request
type CreateTemplateAPIRequest struct {
	Name                string                          `json:"name" binding:"required"`
	Description         string                          `json:"description,omitempty"`
	Category            string                          `json:"category" binding:"required"`
	Version             string                          `json:"version,omitempty"`
	DefaultPassingScore int                             `json:"default_passing_score,omitempty"`
	EstimatedMinutes    int                             `json:"estimated_minutes,omitempty"`
	Topics              []services.TemplateTopicInput   `json:"topics,omitempty"`
	Tags                []string                        `json:"tags,omitempty"`
}

// UpdateTemplateAPIRequest represents a template update request
type UpdateTemplateAPIRequest struct {
	Name                *string                       `json:"name,omitempty"`
	Description         *string                       `json:"description,omitempty"`
	Version             *string                       `json:"version,omitempty"`
	DefaultPassingScore *int                          `json:"default_passing_score,omitempty"`
	EstimatedMinutes    *int                          `json:"estimated_minutes,omitempty"`
	Topics              []services.TemplateTopicInput `json:"topics,omitempty"`
	Tags                []string                      `json:"tags,omitempty"`
}

// PublishTemplateAPIRequest represents a template publish request
type PublishTemplateAPIRequest struct {
	Visibility string `json:"visibility" binding:"required"`
}

// TemplateTopicResponse represents a template topic in responses
type TemplateTopicResponse struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Order       int    `json:"order"`
}

// PaginatedTemplatesResponse represents paginated templates
type PaginatedTemplatesResponse struct {
	Items      []TemplateResponse `json:"items"`
	TotalCount int64              `json:"total_count"`
	Page       int                `json:"page"`
	Limit      int                `json:"limit"`
	TotalPages int                `json:"total_pages"`
}

// ListSystemTemplates handles GET /api/v1/templates
// @Summary List system templates
// @Description Lists all available system questionnaire templates
// @Tags Templates
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param category query string false "Filter by category (ISO27001, GDPR, NIS2)"
// @Success 200 {object} []TemplateResponse
// @Failure 401 {object} ErrorResponse
// @Router /templates [get]
func (h *TemplateHandler) ListSystemTemplates(c *gin.Context) {
	var category *models.TemplateCategory
	if cat := c.Query("category"); cat != "" {
		tc := models.TemplateCategory(cat)
		if tc.IsValid() {
			category = &tc
		}
	}

	templates, err := h.templateRepo.ListSystemTemplates(c.Request.Context(), category)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to list templates",
		})
		return
	}

	responses := make([]TemplateResponse, len(templates))
	for i, t := range templates {
		responses[i] = toTemplateResponse(&t)
	}

	c.JSON(http.StatusOK, responses)
}

// GetTemplate handles GET /api/v1/templates/:id
// @Summary Get template details
// @Description Gets details of a specific template
// @Tags Templates
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Template ID"
// @Success 200 {object} TemplateResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /templates/{id} [get]
func (h *TemplateHandler) GetTemplate(c *gin.Context) {
	templateID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid template ID",
		})
		return
	}

	template, err := h.templateRepo.GetByID(c.Request.Context(), templateID)
	if err != nil {
		if errors.Is(err, models.ErrTemplateNotFound) {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "not_found",
				Message: "Template not found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to get template",
		})
		return
	}

	c.JSON(http.StatusOK, toTemplateResponse(template))
}

// SearchTemplates handles GET /api/v1/templates/search
// @Summary Search templates
// @Description Searches templates by name or description
// @Tags Templates
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param q query string true "Search query"
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20)
// @Success 200 {object} PaginatedTemplatesResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Router /templates/search [get]
func (h *TemplateHandler) SearchTemplates(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: "Search query is required",
		})
		return
	}

	opts := repository.DefaultPaginationOptions()
	if page, err := strconv.Atoi(c.Query("page")); err == nil && page > 0 {
		opts.Page = page
	}
	if limit, err := strconv.Atoi(c.Query("limit")); err == nil && limit > 0 && limit <= 100 {
		opts.Limit = limit
	}

	result, err := h.templateRepo.SearchTemplates(c.Request.Context(), query, opts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to search templates",
		})
		return
	}

	items := make([]TemplateResponse, len(result.Items))
	for i, t := range result.Items {
		items[i] = toTemplateResponse(&t)
	}

	c.JSON(http.StatusOK, PaginatedTemplatesResponse{
		Items:      items,
		TotalCount: result.TotalCount,
		Page:       result.Page,
		Limit:      result.Limit,
		TotalPages: result.TotalPages,
	})
}

// ListOrganizationTemplates handles GET /api/v1/templates/organization
// @Summary List organization templates
// @Description Lists custom templates created by the company
// @Tags Templates
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20)
// @Success 200 {object} PaginatedTemplatesResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Router /templates/organization [get]
func (h *TemplateHandler) ListOrganizationTemplates(c *gin.Context) {
	companyID, ok := middleware.GetOrgID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "Invalid session",
		})
		return
	}

	opts := repository.DefaultPaginationOptions()
	if page, err := strconv.Atoi(c.Query("page")); err == nil && page > 0 {
		opts.Page = page
	}
	if limit, err := strconv.Atoi(c.Query("limit")); err == nil && limit > 0 && limit <= 100 {
		opts.Limit = limit
	}

	result, err := h.templateRepo.ListByOrganization(c.Request.Context(), companyID, opts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to list templates",
		})
		return
	}

	items := make([]TemplateResponse, len(result.Items))
	for i, t := range result.Items {
		items[i] = toTemplateResponse(&t)
	}

	c.JSON(http.StatusOK, PaginatedTemplatesResponse{
		Items:      items,
		TotalCount: result.TotalCount,
		Page:       result.Page,
		Limit:      result.Limit,
		TotalPages: result.TotalPages,
	})
}

// CreateTemplate handles POST /api/v1/templates
// @Summary Create a new template
// @Description Creates a new questionnaire template (draft)
// @Tags Templates
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body CreateTemplateAPIRequest true "Template data"
// @Success 201 {object} TemplateResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Router /templates [post]
func (h *TemplateHandler) CreateTemplate(c *gin.Context) {
	orgID, ok := middleware.GetOrgID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "Invalid session",
		})
		return
	}

	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "Invalid session",
		})
		return
	}

	var req CreateTemplateAPIRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid request body",
		})
		return
	}

	serviceReq := services.CreateTemplateRequest{
		Name:                req.Name,
		Description:         req.Description,
		Category:            req.Category,
		Version:             req.Version,
		DefaultPassingScore: req.DefaultPassingScore,
		EstimatedMinutes:    req.EstimatedMinutes,
		Topics:              req.Topics,
		Tags:                req.Tags,
	}

	template, err := h.templateService.CreateTemplate(c.Request.Context(), orgID, userID, serviceReq)
	if err != nil {
		h.handleTemplateError(c, err)
		return
	}

	c.JSON(http.StatusCreated, toTemplateResponse(template))
}

// ImportTemplate handles POST /api/v1/templates/import
// @Summary Import a template from JSON file
// @Description Imports a questionnaire template from a JSON file
// @Tags Templates
// @Accept multipart/form-data
// @Produce json
// @Security BearerAuth
// @Param file formData file true "JSON template file"
// @Success 201 {object} TemplateResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Router /templates/import [post]
func (h *TemplateHandler) ImportTemplate(c *gin.Context) {
	orgID, ok := middleware.GetOrgID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "Invalid session",
		})
		return
	}

	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "Invalid session",
		})
		return
	}

	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: "File is required",
		})
		return
	}

	// Validate file extension
	if !strings.HasSuffix(strings.ToLower(file.Filename), ".json") {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: "Only JSON files are supported",
		})
		return
	}

	// Read file content
	f, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to read file",
		})
		return
	}
	defer f.Close()

	content, err := io.ReadAll(f)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to read file content",
		})
		return
	}

	template, err := h.templateService.ImportTemplate(c.Request.Context(), orgID, userID, content)
	if err != nil {
		h.handleTemplateError(c, err)
		return
	}

	c.JSON(http.StatusCreated, toTemplateResponse(template))
}

// UpdateTemplate handles PUT /api/v1/templates/:id
// @Summary Update a template
// @Description Updates a draft template (owner only)
// @Tags Templates
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Template ID"
// @Param request body UpdateTemplateAPIRequest true "Update data"
// @Success 200 {object} TemplateResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /templates/{id} [put]
func (h *TemplateHandler) UpdateTemplate(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "Invalid session",
		})
		return
	}

	templateID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid template ID",
		})
		return
	}

	var req UpdateTemplateAPIRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid request body",
		})
		return
	}

	serviceReq := services.UpdateTemplateRequest{
		Name:                req.Name,
		Description:         req.Description,
		Version:             req.Version,
		DefaultPassingScore: req.DefaultPassingScore,
		EstimatedMinutes:    req.EstimatedMinutes,
		Topics:              req.Topics,
		Tags:                req.Tags,
	}

	template, err := h.templateService.UpdateTemplate(c.Request.Context(), templateID, userID, serviceReq)
	if err != nil {
		h.handleTemplateError(c, err)
		return
	}

	c.JSON(http.StatusOK, toTemplateResponse(template))
}

// DeleteTemplate handles DELETE /api/v1/templates/:id
// @Summary Delete a template
// @Description Deletes a template (owner only, must be draft or unused)
// @Tags Templates
// @Produce json
// @Security BearerAuth
// @Param id path string true "Template ID"
// @Success 204 "No content"
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /templates/{id} [delete]
func (h *TemplateHandler) DeleteTemplate(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "Invalid session",
		})
		return
	}

	templateID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid template ID",
		})
		return
	}

	if err := h.templateService.DeleteTemplate(c.Request.Context(), templateID, userID); err != nil {
		h.handleTemplateError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// PublishTemplate handles POST /api/v1/templates/:id/publish
// @Summary Publish a template
// @Description Publishes a draft template with specified visibility (owner only)
// @Tags Templates
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Template ID"
// @Param request body PublishTemplateAPIRequest true "Publish options"
// @Success 200 {object} TemplateResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /templates/{id}/publish [post]
func (h *TemplateHandler) PublishTemplate(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "Invalid session",
		})
		return
	}

	templateID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid template ID",
		})
		return
	}

	var req PublishTemplateAPIRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: "Visibility is required (local or global)",
		})
		return
	}

	// Parse visibility
	visibility := models.TemplateVisibility(strings.ToUpper(req.Visibility))
	if visibility != models.TemplateVisibilityLocal && visibility != models.TemplateVisibilityGlobal {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: "Visibility must be 'local' or 'global'",
		})
		return
	}

	template, err := h.templateService.PublishTemplate(c.Request.Context(), templateID, userID, visibility)
	if err != nil {
		h.handleTemplateError(c, err)
		return
	}

	c.JSON(http.StatusOK, toTemplateResponse(template))
}

// UnpublishTemplate handles POST /api/v1/templates/:id/unpublish
// @Summary Unpublish a template
// @Description Reverts a published template to draft (owner only, must not be in use)
// @Tags Templates
// @Produce json
// @Security BearerAuth
// @Param id path string true "Template ID"
// @Success 200 {object} TemplateResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /templates/{id}/unpublish [post]
func (h *TemplateHandler) UnpublishTemplate(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "Invalid session",
		})
		return
	}

	templateID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid template ID",
		})
		return
	}

	template, err := h.templateService.UnpublishTemplate(c.Request.Context(), templateID, userID)
	if err != nil {
		h.handleTemplateError(c, err)
		return
	}

	c.JSON(http.StatusOK, toTemplateResponse(template))
}

// ListMyTemplates handles GET /api/v1/templates/mine
// @Summary List my templates
// @Description Lists templates created by the current user
// @Tags Templates
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20)
// @Success 200 {object} PaginatedTemplatesResponse
// @Failure 401 {object} ErrorResponse
// @Router /templates/mine [get]
func (h *TemplateHandler) ListMyTemplates(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "Invalid session",
		})
		return
	}

	opts := repository.DefaultPaginationOptions()
	if page, err := strconv.Atoi(c.Query("page")); err == nil && page > 0 {
		opts.Page = page
	}
	if limit, err := strconv.Atoi(c.Query("limit")); err == nil && limit > 0 && limit <= 100 {
		opts.Limit = limit
	}

	result, err := h.templateService.ListMyTemplates(c.Request.Context(), userID, opts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to list templates",
		})
		return
	}

	items := make([]TemplateResponse, len(result.Items))
	for i, t := range result.Items {
		items[i] = toTemplateResponse(&t)
	}

	c.JSON(http.StatusOK, PaginatedTemplatesResponse{
		Items:      items,
		TotalCount: result.TotalCount,
		Page:       result.Page,
		Limit:      result.Limit,
		TotalPages: result.TotalPages,
	})
}

// handleTemplateError maps service errors to HTTP responses
func (h *TemplateHandler) handleTemplateError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, models.ErrTemplateNotFound):
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "not_found",
			Message: "Template not found",
		})
	case errors.Is(err, models.ErrTemplateNotOwnedByUser):
		c.JSON(http.StatusForbidden, ErrorResponse{
			Error:   "forbidden",
			Message: "You can only modify templates you created",
		})
	case errors.Is(err, models.ErrTemplateNotEditable):
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "not_editable",
			Message: "Template cannot be edited (already published or system template)",
		})
	case errors.Is(err, models.ErrTemplateNotDeletable):
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "not_deletable",
			Message: "Template cannot be deleted",
		})
	case errors.Is(err, models.ErrTemplateAlreadyPublished):
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "already_published",
			Message: "Template is already published",
		})
	case errors.Is(err, models.ErrTemplateNotPublished):
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "not_published",
			Message: "Template is not published",
		})
	case errors.Is(err, models.ErrTemplateInUse):
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "in_use",
			Message: "Template is in use and cannot be modified",
		})
	case errors.Is(err, models.ErrTemplateInvalidFormat):
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_format",
			Message: err.Error(),
		})
	case errors.Is(err, models.ErrTemplateMissingFields):
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "missing_fields",
			Message: err.Error(),
		})
	case errors.Is(err, models.ErrTemplateInvalidVisibility):
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_visibility",
			Message: "Invalid visibility value",
		})
	default:
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "An unexpected error occurred",
		})
	}
}

// RegisterRoutes registers template handler routes
// #INTEGRATION_POINT: Routes require authentication
func (h *TemplateHandler) RegisterRoutes(rg *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	templates := rg.Group("/templates")
	templates.Use(authMiddleware)

	// Read-only endpoints (all authenticated users)
	templates.GET("", h.ListSystemTemplates)
	templates.GET("/search", h.SearchTemplates)
	templates.GET("/:id", h.GetTemplate)

	// Organization-level endpoints
	templates.GET("/organization", middleware.RequireCompany(), h.ListOrganizationTemplates)
	templates.GET("/mine", middleware.RequireCompany(), h.ListMyTemplates)

	// Write endpoints (company users only)
	templates.POST("", middleware.RequireCompany(), h.CreateTemplate)
	templates.POST("/import", middleware.RequireCompany(), h.ImportTemplate)
	templates.PUT("/:id", middleware.RequireCompany(), h.UpdateTemplate)
	templates.DELETE("/:id", middleware.RequireCompany(), h.DeleteTemplate)
	templates.POST("/:id/publish", middleware.RequireCompany(), h.PublishTemplate)
	templates.POST("/:id/unpublish", middleware.RequireCompany(), h.UnpublishTemplate)
}

// toTemplateResponse converts a template model to response
func toTemplateResponse(t *models.QuestionnaireTemplate) TemplateResponse {
	resp := TemplateResponse{
		ID:                  t.ID.Hex(),
		Name:                t.Name,
		Description:         t.Description,
		Category:            string(t.Category),
		Version:             t.Version,
		IsSystem:            t.IsSystem,
		Visibility:          strings.ToLower(string(t.Visibility)),
		DefaultPassingScore: t.DefaultPassingScore,
		EstimatedMinutes:    t.EstimatedMinutes,
		Tags:                t.Tags,
		UsageCount:          t.UsageCount,
		CreatedAt:           t.CreatedAt,
		UpdatedAt:           t.UpdatedAt,
		PublishedAt:         t.PublishedAt,
	}

	// Set optional fields
	if t.CreatedByOrgID != nil {
		resp.CreatedByOrgID = t.CreatedByOrgID.Hex()
	}
	if t.CreatedByUser != nil {
		resp.CreatedByUser = t.CreatedByUser.Hex()
	}

	resp.Topics = make([]TemplateTopicResponse, len(t.Topics))
	for i, topic := range t.Topics {
		resp.Topics[i] = TemplateTopicResponse{
			ID:          topic.ID,
			Name:        topic.Name,
			Description: topic.Description,
			Order:       topic.Order,
		}
	}

	return resp
}
