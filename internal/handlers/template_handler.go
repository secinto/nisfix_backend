// Package handlers provides HTTP handlers for API endpoints.
package handlers

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/checkfix-tools/nisfix_backend/internal/middleware"
	"github.com/checkfix-tools/nisfix_backend/internal/models"
	"github.com/checkfix-tools/nisfix_backend/internal/repository"
)

// TemplateHandler handles questionnaire template endpoints
// #INTEGRATION_POINT: Company portal uses these endpoints for template browsing
type TemplateHandler struct {
	templateRepo repository.QuestionnaireTemplateRepository
}

// NewTemplateHandler creates a new template handler
func NewTemplateHandler(templateRepo repository.QuestionnaireTemplateRepository) *TemplateHandler {
	return &TemplateHandler{
		templateRepo: templateRepo,
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
	DefaultPassingScore int                     `json:"default_passing_score"`
	EstimatedMinutes    int                     `json:"estimated_minutes"`
	Topics              []TemplateTopicResponse `json:"topics"`
	Tags                []string                `json:"tags,omitempty"`
	UsageCount          int                     `json:"usage_count"`
	CreatedAt           time.Time               `json:"created_at"`
	UpdatedAt           time.Time               `json:"updated_at"`
	PublishedAt         *time.Time              `json:"published_at,omitempty"`
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

// RegisterRoutes registers template handler routes
// #INTEGRATION_POINT: Routes require authentication
func (h *TemplateHandler) RegisterRoutes(rg *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	templates := rg.Group("/templates")
	templates.Use(authMiddleware)
	{
		templates.GET("", h.ListSystemTemplates)
		templates.GET("/search", h.SearchTemplates)
		templates.GET("/organization", middleware.RequireCompany(), h.ListOrganizationTemplates)
		templates.GET("/:id", h.GetTemplate)
	}
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
		DefaultPassingScore: t.DefaultPassingScore,
		EstimatedMinutes:    t.EstimatedMinutes,
		Tags:                t.Tags,
		UsageCount:          t.UsageCount,
		CreatedAt:           t.CreatedAt,
		UpdatedAt:           t.UpdatedAt,
		PublishedAt:         t.PublishedAt,
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
