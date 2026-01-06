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
	"github.com/checkfix-tools/nisfix_backend/internal/services"
)

// RequirementHandler handles requirement endpoints
// #INTEGRATION_POINT: Company portal uses these endpoints for requirement management
type RequirementHandler struct {
	requirementService services.RequirementService
}

// NewRequirementHandler creates a new requirement handler
func NewRequirementHandler(requirementService services.RequirementService) *RequirementHandler {
	return &RequirementHandler{
		requirementService: requirementService,
	}
}

// CreateRequirementAPIRequest represents the create requirement request body
type CreateRequirementAPIRequest struct {
	RelationshipID   string     `json:"relationship_id" binding:"required"`
	Type             string     `json:"type" binding:"required"`
	Title            string     `json:"title" binding:"required"`
	Description      string     `json:"description,omitempty"`
	Priority         string     `json:"priority,omitempty"`
	DueDate          *time.Time `json:"due_date,omitempty"`
	QuestionnaireID  *string    `json:"questionnaire_id,omitempty"`
	PassingScore     *int       `json:"passing_score,omitempty"`
	MinimumGrade     *string    `json:"minimum_grade,omitempty"`
	MaxReportAgeDays *int       `json:"max_report_age_days,omitempty"`
}

// RequirementResponse represents a requirement in API responses
type RequirementResponse struct {
	ID               string                        `json:"id"`
	RelationshipID   string                        `json:"relationship_id"`
	CompanyID        string                        `json:"company_id"`
	SupplierID       string                        `json:"supplier_id"`
	Type             string                        `json:"type"`
	Title            string                        `json:"title"`
	Description      string                        `json:"description,omitempty"`
	Priority         string                        `json:"priority"`
	Status           string                        `json:"status"`
	DueDate          *time.Time                    `json:"due_date,omitempty"`
	QuestionnaireID  *string                       `json:"questionnaire_id,omitempty"`
	PassingScore     *int                          `json:"passing_score,omitempty"`
	MinimumGrade     *string                       `json:"minimum_grade,omitempty"`
	MaxReportAgeDays *int                          `json:"max_report_age_days,omitempty"`
	AssignedAt       time.Time                     `json:"assigned_at"`
	StatusHistory    []RequirementStatusChangeResp `json:"status_history,omitempty"`
	IsOverdue        bool                          `json:"is_overdue"`
	DaysUntilDue     int                           `json:"days_until_due"`
	CreatedAt        time.Time                     `json:"created_at"`
	UpdatedAt        time.Time                     `json:"updated_at"`
}

// RequirementStatusChangeResp represents a status change in responses
type RequirementStatusChangeResp struct {
	FromStatus string    `json:"from_status"`
	ToStatus   string    `json:"to_status"`
	Reason     string    `json:"reason,omitempty"`
	ChangedAt  time.Time `json:"changed_at"`
}

// PaginatedRequirementsResponse represents paginated requirements
type PaginatedRequirementsResponse struct {
	Items      []RequirementResponse `json:"items"`
	TotalCount int64                 `json:"total_count"`
	Page       int                   `json:"page"`
	Limit      int                   `json:"limit"`
	TotalPages int                   `json:"total_pages"`
}

// RequirementStatsResponse represents requirement statistics
type RequirementStatsResponse struct {
	Total      int64 `json:"total"`
	Pending    int64 `json:"pending"`
	InProgress int64 `json:"in_progress"`
	Submitted  int64 `json:"submitted"`
	Approved   int64 `json:"approved"`
	Rejected   int64 `json:"rejected"`
	Expired    int64 `json:"expired"`
	Overdue    int64 `json:"overdue"`
}

// CreateRequirement handles POST /api/v1/requirements
// @Summary Create a requirement
// @Description Creates a new requirement for a supplier
// @Tags Requirements
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body CreateRequirementAPIRequest true "Create request"
// @Success 201 {object} RequirementResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /requirements [post]
func (h *RequirementHandler) CreateRequirement(c *gin.Context) {
	companyID, ok := middleware.GetOrgID(c)
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

	var req CreateRequirementAPIRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: "relationship_id, type, and title are required",
		})
		return
	}

	priority := models.PriorityMedium
	if req.Priority != "" {
		priority = models.Priority(req.Priority)
	}

	serviceReq := services.CreateRequirementRequest{
		RelationshipID:   req.RelationshipID,
		Type:             models.RequirementType(req.Type),
		Title:            req.Title,
		Description:      req.Description,
		Priority:         priority,
		DueDate:          req.DueDate,
		QuestionnaireID:  req.QuestionnaireID,
		PassingScore:     req.PassingScore,
		MinimumGrade:     req.MinimumGrade,
		MaxReportAgeDays: req.MaxReportAgeDays,
	}

	requirement, err := h.requirementService.CreateRequirement(c.Request.Context(), companyID, userID, serviceReq)
	if err != nil {
		if errors.Is(err, services.ErrRelationshipNotFound) {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "relationship_not_found",
				Message: "Relationship not found",
			})
			return
		}
		if errors.Is(err, services.ErrRelationshipNotActive) {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "relationship_not_active",
				Message: "Requirements can only be created for active relationships",
			})
			return
		}
		if errors.Is(err, services.ErrQuestionnaireNotFound) {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "questionnaire_not_found",
				Message: "Questionnaire not found",
			})
			return
		}
		if errors.Is(err, services.ErrQuestionnaireNotPublished) {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "questionnaire_not_published",
				Message: "Questionnaire must be published",
			})
			return
		}
		if errors.Is(err, services.ErrInvalidRequirementType) {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "invalid_type",
				Message: "Invalid requirement type",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to create requirement",
		})
		return
	}

	c.JSON(http.StatusCreated, toRequirementResponse(requirement))
}

// ListRequirements handles GET /api/v1/requirements
// @Summary List requirements
// @Description Lists all requirements for the company
// @Tags Requirements
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param status query string false "Filter by status"
// @Param relationship_id query string false "Filter by relationship"
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20)
// @Success 200 {object} PaginatedRequirementsResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Router /requirements [get]
func (h *RequirementHandler) ListRequirements(c *gin.Context) {
	companyID, ok := middleware.GetOrgID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "Invalid session",
		})
		return
	}

	// Check if filtering by relationship
	if relationshipIDStr := c.Query("relationship_id"); relationshipIDStr != "" {
		relationshipID, err := primitive.ObjectIDFromHex(relationshipIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "invalid_id",
				Message: "Invalid relationship ID",
			})
			return
		}

		var status *models.RequirementStatus
		if statusStr := c.Query("status"); statusStr != "" {
			s := models.RequirementStatus(statusStr)
			status = &s
		}

		requirements, err := h.requirementService.ListRequirementsByRelationship(c.Request.Context(), relationshipID, status)
		if err != nil {
			c.JSON(http.StatusInternalServerError, ErrorResponse{
				Error:   "internal_error",
				Message: "Failed to list requirements",
			})
			return
		}

		items := make([]RequirementResponse, len(requirements))
		for i, r := range requirements {
			items[i] = toRequirementResponse(&r)
		}

		c.JSON(http.StatusOK, PaginatedRequirementsResponse{
			Items:      items,
			TotalCount: int64(len(requirements)),
			Page:       1,
			Limit:      len(requirements),
			TotalPages: 1,
		})
		return
	}

	// Parse query parameters
	filters := services.RequirementFilters{}
	if status := c.Query("status"); status != "" {
		s := models.RequirementStatus(status)
		filters.Status = &s
	}

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
	if sortDir := c.Query("sort_dir"); sortDir == sortDirectionAsc {
		opts.SortDir = 1
	}

	result, err := h.requirementService.ListRequirementsByCompany(c.Request.Context(), companyID, filters, opts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to list requirements",
		})
		return
	}

	items := make([]RequirementResponse, len(result.Items))
	for i, r := range result.Items {
		items[i] = toRequirementResponse(&r)
	}

	c.JSON(http.StatusOK, PaginatedRequirementsResponse{
		Items:      items,
		TotalCount: result.TotalCount,
		Page:       result.Page,
		Limit:      result.Limit,
		TotalPages: result.TotalPages,
	})
}

// GetRequirement handles GET /api/v1/requirements/:id
// @Summary Get requirement details
// @Description Gets details of a specific requirement
// @Tags Requirements
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Requirement ID"
// @Success 200 {object} RequirementResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /requirements/{id} [get]
func (h *RequirementHandler) GetRequirement(c *gin.Context) {
	companyID, ok := middleware.GetOrgID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "Invalid session",
		})
		return
	}

	requirementID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid requirement ID",
		})
		return
	}

	requirement, err := h.requirementService.GetRequirement(c.Request.Context(), requirementID, &companyID)
	if err != nil {
		if errors.Is(err, services.ErrRequirementNotFound) {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "not_found",
				Message: "Requirement not found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to get requirement",
		})
		return
	}

	c.JSON(http.StatusOK, toRequirementResponse(requirement))
}

// UpdateRequirementAPIRequest represents the update requirement request
type UpdateRequirementAPIRequest struct {
	Title            *string    `json:"title,omitempty"`
	Description      *string    `json:"description,omitempty"`
	Priority         *string    `json:"priority,omitempty"`
	DueDate          *time.Time `json:"due_date,omitempty"`
	PassingScore     *int       `json:"passing_score,omitempty"`
	MinimumGrade     *string    `json:"minimum_grade,omitempty"`
	MaxReportAgeDays *int       `json:"max_report_age_days,omitempty"`
}

// UpdateRequirement handles PATCH /api/v1/requirements/:id
// @Summary Update requirement
// @Description Updates a requirement (while pending)
// @Tags Requirements
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Requirement ID"
// @Param request body UpdateRequirementAPIRequest true "Update request"
// @Success 200 {object} RequirementResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /requirements/{id} [patch]
func (h *RequirementHandler) UpdateRequirement(c *gin.Context) {
	companyID, ok := middleware.GetOrgID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "Invalid session",
		})
		return
	}

	requirementID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid requirement ID",
		})
		return
	}

	var req UpdateRequirementAPIRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid request body",
		})
		return
	}

	var priority *models.Priority
	if req.Priority != nil {
		p := models.Priority(*req.Priority)
		priority = &p
	}

	serviceReq := services.UpdateRequirementRequest{
		Title:            req.Title,
		Description:      req.Description,
		Priority:         priority,
		DueDate:          req.DueDate,
		PassingScore:     req.PassingScore,
		MinimumGrade:     req.MinimumGrade,
		MaxReportAgeDays: req.MaxReportAgeDays,
	}

	requirement, err := h.requirementService.UpdateRequirement(c.Request.Context(), requirementID, companyID, serviceReq)
	if err != nil {
		if errors.Is(err, services.ErrRequirementNotFound) {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "not_found",
				Message: "Requirement not found",
			})
			return
		}

		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "update_failed",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, toRequirementResponse(requirement))
}

// GetRequirementStats handles GET /api/v1/requirements/stats
// @Summary Get requirement statistics
// @Description Gets requirement statistics for the company
// @Tags Requirements
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} RequirementStatsResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Router /requirements/stats [get]
func (h *RequirementHandler) GetRequirementStats(c *gin.Context) {
	companyID, ok := middleware.GetOrgID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "Invalid session",
		})
		return
	}

	stats, err := h.requirementService.GetRequirementStats(c.Request.Context(), companyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to get requirement stats",
		})
		return
	}

	c.JSON(http.StatusOK, RequirementStatsResponse{
		Total:      stats.Total,
		Pending:    stats.Pending,
		InProgress: stats.InProgress,
		Submitted:  stats.Submitted,
		Approved:   stats.Approved,
		Rejected:   stats.Rejected,
		Expired:    stats.Expired,
		Overdue:    stats.Overdue,
	})
}

// RegisterRoutes registers requirement handler routes
// #INTEGRATION_POINT: Routes require authentication and company organization type
func (h *RequirementHandler) RegisterRoutes(rg *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	requirements := rg.Group("/requirements")
	requirements.Use(authMiddleware)
	requirements.Use(middleware.RequireCompany())
	{
		requirements.POST("", h.CreateRequirement)
		requirements.GET("", h.ListRequirements)
		requirements.GET("/stats", h.GetRequirementStats)
		requirements.GET("/:id", h.GetRequirement)
		requirements.PATCH("/:id", h.UpdateRequirement)
	}
}

// toRequirementResponse converts a requirement model to response
func toRequirementResponse(r *models.Requirement) RequirementResponse {
	resp := RequirementResponse{
		ID:               r.ID.Hex(),
		RelationshipID:   r.RelationshipID.Hex(),
		CompanyID:        r.CompanyID.Hex(),
		SupplierID:       r.SupplierID.Hex(),
		Type:             string(r.Type),
		Title:            r.Title,
		Description:      r.Description,
		Priority:         string(r.Priority),
		Status:           string(r.Status),
		DueDate:          r.DueDate,
		PassingScore:     r.PassingScore,
		MinimumGrade:     r.MinimumGrade,
		MaxReportAgeDays: r.MaxReportAgeDays,
		AssignedAt:       r.AssignedAt,
		IsOverdue:        r.IsOverdue(),
		DaysUntilDue:     r.DaysUntilDue(),
		CreatedAt:        r.CreatedAt,
		UpdatedAt:        r.UpdatedAt,
	}

	if r.QuestionnaireID != nil {
		qID := r.QuestionnaireID.Hex()
		resp.QuestionnaireID = &qID
	}

	// Include status history
	resp.StatusHistory = make([]RequirementStatusChangeResp, len(r.StatusHistory))
	for i, change := range r.StatusHistory {
		resp.StatusHistory[i] = RequirementStatusChangeResp{
			FromStatus: string(change.FromStatus),
			ToStatus:   string(change.ToStatus),
			Reason:     change.Reason,
			ChangedAt:  change.ChangedAt,
		}
	}

	return resp
}
