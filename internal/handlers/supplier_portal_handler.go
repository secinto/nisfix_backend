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

// SupplierPortalHandler handles supplier-side endpoints
// #INTEGRATION_POINT: Supplier portal uses these endpoints for viewing and responding to requirements
type SupplierPortalHandler struct {
	relationshipRepo repository.RelationshipRepository
	requirementRepo  repository.RequirementRepository
	responseService  services.ResponseService
}

// NewSupplierPortalHandler creates a new supplier portal handler
func NewSupplierPortalHandler(
	relationshipRepo repository.RelationshipRepository,
	requirementRepo repository.RequirementRepository,
	responseService services.ResponseService,
) *SupplierPortalHandler {
	return &SupplierPortalHandler{
		relationshipRepo: relationshipRepo,
		requirementRepo:  requirementRepo,
		responseService:  responseService,
	}
}

// CompanyRelationshipResponse represents a company relationship for suppliers
type CompanyRelationshipResponse struct {
	ID               string     `json:"id"`
	CompanyID        string     `json:"company_id"`
	CompanyName      string     `json:"company_name,omitempty"`
	Status           string     `json:"status"`
	Classification   string     `json:"classification"`
	InvitedAt        time.Time  `json:"invited_at"`
	AcceptedAt       *time.Time `json:"accepted_at,omitempty"`
	PendingRequirements int     `json:"pending_requirements"`
}

// SupplierRequirementResponse represents a requirement from supplier's view
type SupplierRequirementResponse struct {
	ID              string     `json:"id"`
	CompanyID       string     `json:"company_id"`
	CompanyName     string     `json:"company_name,omitempty"`
	Type            string     `json:"type"`
	Title           string     `json:"title"`
	Description     string     `json:"description,omitempty"`
	Priority        string     `json:"priority"`
	Status          string     `json:"status"`
	DueDate         *time.Time `json:"due_date,omitempty"`
	QuestionnaireID *string    `json:"questionnaire_id,omitempty"`
	PassingScore    *int       `json:"passing_score,omitempty"`
	MinimumGrade    *string    `json:"minimum_grade,omitempty"`
	ResponseID      *string    `json:"response_id,omitempty"`
	IsOverdue       bool       `json:"is_overdue"`
	DaysUntilDue    int        `json:"days_until_due"`
	AssignedAt      time.Time  `json:"assigned_at"`
	CreatedAt       time.Time  `json:"created_at"`
}

// SupplierResponseResponse represents a response in API responses
type SupplierResponseResponse struct {
	ID               string             `json:"id"`
	RequirementID    string             `json:"requirement_id"`
	SupplierID       string             `json:"supplier_id"`
	Score            *int               `json:"score,omitempty"`
	MaxScore         *int               `json:"max_score,omitempty"`
	Passed           *bool              `json:"passed,omitempty"`
	Grade            *string            `json:"grade,omitempty"`
	DraftAnswerCount int                `json:"draft_answer_count"`
	IsSubmitted      bool               `json:"is_submitted"`
	StartedAt        time.Time          `json:"started_at"`
	SubmittedAt      *time.Time         `json:"submitted_at,omitempty"`
	DraftAnswers     []DraftAnswerResponse `json:"draft_answers,omitempty"`
}

// DraftAnswerResponse represents a draft answer
type DraftAnswerResponse struct {
	QuestionID      string    `json:"question_id"`
	SelectedOptions []string  `json:"selected_options,omitempty"`
	TextAnswer      string    `json:"text_answer,omitempty"`
	SavedAt         time.Time `json:"saved_at"`
}

// SupplierDashboardResponse represents the supplier dashboard
type SupplierDashboardResponse struct {
	TotalCompanies       int64                  `json:"total_companies"`
	PendingInvitations   int64                  `json:"pending_invitations"`
	PendingRequirements  int64                  `json:"pending_requirements"`
	OverdueRequirements  int64                  `json:"overdue_requirements"`
	SubmittedRequirements int64                 `json:"submitted_requirements"`
	RecentRequirements   []SupplierRequirementResponse `json:"recent_requirements"`
}

// PaginatedSupplierRequirementsResponse represents paginated requirements
type PaginatedSupplierRequirementsResponse struct {
	Items      []SupplierRequirementResponse `json:"items"`
	TotalCount int64                         `json:"total_count"`
	Page       int                           `json:"page"`
	Limit      int                           `json:"limit"`
	TotalPages int                           `json:"total_pages"`
}

// GetSupplierDashboard handles GET /api/v1/supplier/dashboard
// @Summary Get supplier dashboard
// @Description Gets the supplier dashboard with overview statistics
// @Tags Supplier Portal
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} SupplierDashboardResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Router /supplier/dashboard [get]
func (h *SupplierPortalHandler) GetSupplierDashboard(c *gin.Context) {
	supplierID, ok := middleware.GetOrgID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "Invalid session",
		})
		return
	}

	ctx := c.Request.Context()

	// Count total companies (active relationships)
	activeStatus := models.RelationshipStatusActive
	totalCompanies, _ := h.relationshipRepo.CountBySupplier(ctx, supplierID, &activeStatus)

	// Count pending invitations
	pendingStatus := models.RelationshipStatusPending
	pendingInvitations, _ := h.relationshipRepo.CountBySupplier(ctx, supplierID, &pendingStatus)

	// Count pending requirements
	pendingReqStatus := models.RequirementStatusPending
	pendingRequirements, _ := h.requirementRepo.CountBySupplier(ctx, supplierID, &pendingReqStatus)

	// Count in progress requirements
	inProgressStatus := models.RequirementStatusInProgress
	inProgress, _ := h.requirementRepo.CountBySupplier(ctx, supplierID, &inProgressStatus)
	pendingRequirements += inProgress

	// Count submitted requirements
	submittedStatus := models.RequirementStatusSubmitted
	submitted, _ := h.requirementRepo.CountBySupplier(ctx, supplierID, &submittedStatus)

	// Get recent requirements
	opts := repository.PaginationOptions{Page: 1, Limit: 5, SortBy: "created_at", SortDir: -1}
	result, _ := h.requirementRepo.ListBySupplier(ctx, supplierID, nil, opts)

	recentReqs := make([]SupplierRequirementResponse, len(result.Items))
	for i, r := range result.Items {
		recentReqs[i] = toSupplierRequirementResponse(&r)
	}

	// Count overdue (approximation)
	overdue := int64(0)
	for _, r := range result.Items {
		if r.IsOverdue() {
			overdue++
		}
	}

	c.JSON(http.StatusOK, SupplierDashboardResponse{
		TotalCompanies:        totalCompanies,
		PendingInvitations:    pendingInvitations,
		PendingRequirements:   pendingRequirements,
		OverdueRequirements:   overdue,
		SubmittedRequirements: submitted,
		RecentRequirements:    recentReqs,
	})
}

// ListCompanies handles GET /api/v1/supplier/companies
// @Summary List companies
// @Description Lists all companies that have relationships with this supplier
// @Tags Supplier Portal
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param status query string false "Filter by status"
// @Success 200 {object} []CompanyRelationshipResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Router /supplier/companies [get]
func (h *SupplierPortalHandler) ListCompanies(c *gin.Context) {
	supplierID, ok := middleware.GetOrgID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "Invalid session",
		})
		return
	}

	var status *models.RelationshipStatus
	if statusStr := c.Query("status"); statusStr != "" {
		s := models.RelationshipStatus(statusStr)
		status = &s
	}

	opts := repository.PaginationOptions{Page: 1, Limit: 100, SortBy: "created_at", SortDir: -1}
	result, err := h.relationshipRepo.ListBySupplier(c.Request.Context(), supplierID, status, opts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to list companies",
		})
		return
	}

	items := make([]CompanyRelationshipResponse, len(result.Items))
	for i, r := range result.Items {
		items[i] = CompanyRelationshipResponse{
			ID:             r.ID.Hex(),
			CompanyID:      r.CompanyID.Hex(),
			Status:         string(r.Status),
			Classification: string(r.Classification),
			InvitedAt:      r.InvitedAt,
			AcceptedAt:     r.AcceptedAt,
		}
	}

	c.JSON(http.StatusOK, items)
}

// ListPendingInvitations handles GET /api/v1/supplier/invitations
// @Summary List pending invitations
// @Description Lists all pending invitations for this supplier
// @Tags Supplier Portal
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} []CompanyRelationshipResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Router /supplier/invitations [get]
func (h *SupplierPortalHandler) ListPendingInvitations(c *gin.Context) {
	supplierID, ok := middleware.GetOrgID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "Invalid session",
		})
		return
	}

	pendingStatus := models.RelationshipStatusPending
	opts := repository.PaginationOptions{Page: 1, Limit: 100, SortBy: "created_at", SortDir: -1}
	result, err := h.relationshipRepo.ListBySupplier(c.Request.Context(), supplierID, &pendingStatus, opts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to list invitations",
		})
		return
	}

	items := make([]CompanyRelationshipResponse, len(result.Items))
	for i, r := range result.Items {
		items[i] = CompanyRelationshipResponse{
			ID:             r.ID.Hex(),
			CompanyID:      r.CompanyID.Hex(),
			Status:         string(r.Status),
			Classification: string(r.Classification),
			InvitedAt:      r.InvitedAt,
		}
	}

	c.JSON(http.StatusOK, items)
}

// AcceptInvitation handles POST /api/v1/supplier/invitations/:id/accept
// @Summary Accept invitation
// @Description Accepts a company invitation
// @Tags Supplier Portal
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Relationship ID"
// @Success 200 {object} CompanyRelationshipResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /supplier/invitations/{id}/accept [post]
func (h *SupplierPortalHandler) AcceptInvitation(c *gin.Context) {
	supplierID, ok := middleware.GetOrgID(c)
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

	relationshipID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid relationship ID",
		})
		return
	}

	// Get relationship
	relationship, err := h.relationshipRepo.GetByID(c.Request.Context(), relationshipID)
	if err != nil {
		if errors.Is(err, models.ErrRelationshipNotFound) {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "not_found",
				Message: "Invitation not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to get invitation",
		})
		return
	}

	// Accept
	if err := relationship.Accept(supplierID, userID); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_transition",
			Message: "Cannot accept this invitation",
		})
		return
	}

	if err := h.relationshipRepo.Update(c.Request.Context(), relationship); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to accept invitation",
		})
		return
	}

	c.JSON(http.StatusOK, CompanyRelationshipResponse{
		ID:             relationship.ID.Hex(),
		CompanyID:      relationship.CompanyID.Hex(),
		Status:         string(relationship.Status),
		Classification: string(relationship.Classification),
		InvitedAt:      relationship.InvitedAt,
		AcceptedAt:     relationship.AcceptedAt,
	})
}

// DeclineInvitationRequest represents the decline request
type DeclineInvitationRequest struct {
	Reason string `json:"reason,omitempty"`
}

// DeclineInvitation handles POST /api/v1/supplier/invitations/:id/decline
// @Summary Decline invitation
// @Description Declines a company invitation
// @Tags Supplier Portal
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Relationship ID"
// @Param request body DeclineInvitationRequest false "Decline reason"
// @Success 200 {object} CompanyRelationshipResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /supplier/invitations/{id}/decline [post]
func (h *SupplierPortalHandler) DeclineInvitation(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "Invalid session",
		})
		return
	}

	relationshipID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid relationship ID",
		})
		return
	}

	var req DeclineInvitationRequest
	_ = c.ShouldBindJSON(&req)

	// Get relationship
	relationship, err := h.relationshipRepo.GetByID(c.Request.Context(), relationshipID)
	if err != nil {
		if errors.Is(err, models.ErrRelationshipNotFound) {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "not_found",
				Message: "Invitation not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to get invitation",
		})
		return
	}

	// Decline
	if err := relationship.Decline(userID, req.Reason); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_transition",
			Message: "Cannot decline this invitation",
		})
		return
	}

	if err := h.relationshipRepo.Update(c.Request.Context(), relationship); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to decline invitation",
		})
		return
	}

	c.JSON(http.StatusOK, CompanyRelationshipResponse{
		ID:             relationship.ID.Hex(),
		CompanyID:      relationship.CompanyID.Hex(),
		Status:         string(relationship.Status),
		Classification: string(relationship.Classification),
		InvitedAt:      relationship.InvitedAt,
	})
}

// ListRequirements handles GET /api/v1/supplier/requirements
// @Summary List requirements
// @Description Lists all requirements assigned to this supplier
// @Tags Supplier Portal
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param status query string false "Filter by status"
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20)
// @Success 200 {object} PaginatedSupplierRequirementsResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Router /supplier/requirements [get]
func (h *SupplierPortalHandler) ListRequirements(c *gin.Context) {
	supplierID, ok := middleware.GetOrgID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "Invalid session",
		})
		return
	}

	var status *models.RequirementStatus
	if statusStr := c.Query("status"); statusStr != "" {
		s := models.RequirementStatus(statusStr)
		status = &s
	}

	opts := repository.DefaultPaginationOptions()
	if page, err := strconv.Atoi(c.Query("page")); err == nil && page > 0 {
		opts.Page = page
	}
	if limit, err := strconv.Atoi(c.Query("limit")); err == nil && limit > 0 && limit <= 100 {
		opts.Limit = limit
	}

	result, err := h.requirementRepo.ListBySupplier(c.Request.Context(), supplierID, status, opts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to list requirements",
		})
		return
	}

	items := make([]SupplierRequirementResponse, len(result.Items))
	for i, r := range result.Items {
		items[i] = toSupplierRequirementResponse(&r)
	}

	c.JSON(http.StatusOK, PaginatedSupplierRequirementsResponse{
		Items:      items,
		TotalCount: result.TotalCount,
		Page:       result.Page,
		Limit:      result.Limit,
		TotalPages: result.TotalPages,
	})
}

// GetRequirement handles GET /api/v1/supplier/requirements/:id
// @Summary Get requirement details
// @Description Gets details of a specific requirement
// @Tags Supplier Portal
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Requirement ID"
// @Success 200 {object} SupplierRequirementResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /supplier/requirements/{id} [get]
func (h *SupplierPortalHandler) GetRequirement(c *gin.Context) {
	supplierID, ok := middleware.GetOrgID(c)
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

	requirement, err := h.requirementRepo.GetByID(c.Request.Context(), requirementID)
	if err != nil {
		if errors.Is(err, models.ErrRequirementNotFound) {
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

	// Verify supplier ownership
	if requirement.SupplierID != supplierID {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "not_found",
			Message: "Requirement not found",
		})
		return
	}

	c.JSON(http.StatusOK, toSupplierRequirementResponse(requirement))
}

// StartResponse handles POST /api/v1/supplier/requirements/:id/start
// @Summary Start response
// @Description Starts a response for a requirement
// @Tags Supplier Portal
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Requirement ID"
// @Success 201 {object} SupplierResponseResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /supplier/requirements/{id}/start [post]
func (h *SupplierPortalHandler) StartResponse(c *gin.Context) {
	supplierID, ok := middleware.GetOrgID(c)
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

	response, err := h.responseService.StartResponse(c.Request.Context(), requirementID, supplierID)
	if err != nil {
		if errors.Is(err, services.ErrRequirementNotFound) {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "not_found",
				Message: "Requirement not found",
			})
			return
		}
		if errors.Is(err, services.ErrCannotStartResponse) {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "cannot_start",
				Message: "Cannot start response for this requirement",
			})
			return
		}
		if errors.Is(err, services.ErrResponseAlreadyExists) {
			c.JSON(http.StatusConflict, ErrorResponse{
				Error:   "response_exists",
				Message: "Response already exists",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to start response",
		})
		return
	}

	c.JSON(http.StatusCreated, toSupplierResponseResponse(response))
}

// GetResponse handles GET /api/v1/supplier/responses/:id
// @Summary Get response
// @Description Gets details of a response
// @Tags Supplier Portal
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Response ID"
// @Success 200 {object} SupplierResponseResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /supplier/responses/{id} [get]
func (h *SupplierPortalHandler) GetResponse(c *gin.Context) {
	supplierID, ok := middleware.GetOrgID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "Invalid session",
		})
		return
	}

	responseID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid response ID",
		})
		return
	}

	response, err := h.responseService.GetResponse(c.Request.Context(), responseID, &supplierID)
	if err != nil {
		if errors.Is(err, services.ErrResponseNotFound) {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "not_found",
				Message: "Response not found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to get response",
		})
		return
	}

	c.JSON(http.StatusOK, toSupplierResponseResponse(response))
}

// SaveDraftRequest represents a save draft request
type SaveDraftRequest struct {
	Answers []SaveDraftAnswerAPIRequest `json:"answers" binding:"required"`
}

// SaveDraftAnswerAPIRequest represents a draft answer in API requests
type SaveDraftAnswerAPIRequest struct {
	QuestionID      string   `json:"question_id" binding:"required"`
	SelectedOptions []string `json:"selected_options,omitempty"`
	TextAnswer      string   `json:"text_answer,omitempty"`
}

// SaveDraft handles POST /api/v1/supplier/responses/:id/draft
// @Summary Save draft answers
// @Description Saves draft answers for a response
// @Tags Supplier Portal
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Response ID"
// @Param request body SaveDraftRequest true "Draft answers"
// @Success 200 {object} map[string]string
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /supplier/responses/{id}/draft [post]
func (h *SupplierPortalHandler) SaveDraft(c *gin.Context) {
	supplierID, ok := middleware.GetOrgID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "Invalid session",
		})
		return
	}

	responseID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid response ID",
		})
		return
	}

	var req SaveDraftRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: "Answers are required",
		})
		return
	}

	// Convert to service format
	answers := make([]services.SaveDraftAnswerRequest, len(req.Answers))
	for i, a := range req.Answers {
		answers[i] = services.SaveDraftAnswerRequest{
			QuestionID:      a.QuestionID,
			SelectedOptions: a.SelectedOptions,
			TextAnswer:      a.TextAnswer,
		}
	}

	if err := h.responseService.SaveMultipleDraftAnswers(c.Request.Context(), responseID, supplierID, answers); err != nil {
		if errors.Is(err, services.ErrResponseNotFound) {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "not_found",
				Message: "Response not found",
			})
			return
		}
		if errors.Is(err, services.ErrResponseAlreadySubmitted) {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "already_submitted",
				Message: "Response has already been submitted",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to save draft",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Draft saved successfully"})
}

// SubmitResponseRequest represents a submit response request
type SubmitResponseRequest struct {
	Answers []SubmitAnswerAPIRequest `json:"answers" binding:"required"`
}

// SubmitAnswerAPIRequest represents an answer in submit request
type SubmitAnswerAPIRequest struct {
	QuestionID      string   `json:"question_id" binding:"required"`
	SelectedOptions []string `json:"selected_options,omitempty"`
	TextAnswer      string   `json:"text_answer,omitempty"`
}

// SubmitResponse handles POST /api/v1/supplier/responses/:id/submit
// @Summary Submit response
// @Description Submits a questionnaire response
// @Tags Supplier Portal
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Response ID"
// @Param request body SubmitResponseRequest true "Answers to submit"
// @Success 200 {object} SubmissionResultResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /supplier/responses/{id}/submit [post]
func (h *SupplierPortalHandler) SubmitResponse(c *gin.Context) {
	supplierID, ok := middleware.GetOrgID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "Invalid session",
		})
		return
	}

	responseID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid response ID",
		})
		return
	}

	var req SubmitResponseRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: "Answers are required",
		})
		return
	}

	// Convert to service format
	answers := make([]services.SubmitAnswerRequest, len(req.Answers))
	for i, a := range req.Answers {
		answers[i] = services.SubmitAnswerRequest{
			QuestionID:      a.QuestionID,
			SelectedOptions: a.SelectedOptions,
			TextAnswer:      a.TextAnswer,
		}
	}

	result, err := h.responseService.SubmitQuestionnaireResponse(c.Request.Context(), responseID, supplierID, answers)
	if err != nil {
		if errors.Is(err, services.ErrResponseNotFound) {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "not_found",
				Message: "Response not found",
			})
			return
		}
		if errors.Is(err, services.ErrResponseAlreadySubmitted) {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "already_submitted",
				Message: "Response has already been submitted",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to submit response",
		})
		return
	}

	c.JSON(http.StatusOK, SubmissionResultResponse{
		SubmissionID: result.Submission.ID.Hex(),
		Passed:       result.Passed,
		Score:        result.Score,
		MaxScore:     result.MaxScore,
		Percentage:   result.Percentage,
	})
}

// SubmissionResultResponse represents submission result
type SubmissionResultResponse struct {
	SubmissionID string  `json:"submission_id"`
	Passed       bool    `json:"passed"`
	Score        int     `json:"score"`
	MaxScore     int     `json:"max_score"`
	Percentage   float64 `json:"percentage"`
}

// RegisterRoutes registers supplier portal handler routes
// #INTEGRATION_POINT: Routes require authentication and supplier organization type
func (h *SupplierPortalHandler) RegisterRoutes(rg *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	supplier := rg.Group("/supplier")
	supplier.Use(authMiddleware)
	supplier.Use(middleware.RequireSupplier())
	{
		// Dashboard
		supplier.GET("/dashboard", h.GetSupplierDashboard)

		// Companies
		supplier.GET("/companies", h.ListCompanies)

		// Invitations
		supplier.GET("/invitations", h.ListPendingInvitations)
		supplier.POST("/invitations/:id/accept", h.AcceptInvitation)
		supplier.POST("/invitations/:id/decline", h.DeclineInvitation)

		// Requirements
		supplier.GET("/requirements", h.ListRequirements)
		supplier.GET("/requirements/:id", h.GetRequirement)
		supplier.POST("/requirements/:id/start", h.StartResponse)

		// Responses
		supplier.GET("/responses/:id", h.GetResponse)
		supplier.POST("/responses/:id/draft", h.SaveDraft)
		supplier.POST("/responses/:id/submit", h.SubmitResponse)
	}
}

// toSupplierRequirementResponse converts a requirement to supplier response format
func toSupplierRequirementResponse(r *models.Requirement) SupplierRequirementResponse {
	resp := SupplierRequirementResponse{
		ID:           r.ID.Hex(),
		CompanyID:    r.CompanyID.Hex(),
		Type:         string(r.Type),
		Title:        r.Title,
		Description:  r.Description,
		Priority:     string(r.Priority),
		Status:       string(r.Status),
		DueDate:      r.DueDate,
		PassingScore: r.PassingScore,
		MinimumGrade: r.MinimumGrade,
		IsOverdue:    r.IsOverdue(),
		DaysUntilDue: r.DaysUntilDue(),
		AssignedAt:   r.AssignedAt,
		CreatedAt:    r.CreatedAt,
	}

	if r.QuestionnaireID != nil {
		qID := r.QuestionnaireID.Hex()
		resp.QuestionnaireID = &qID
	}

	return resp
}

// toSupplierResponseResponse converts a response to API format
func toSupplierResponseResponse(r *models.SupplierResponse) SupplierResponseResponse {
	resp := SupplierResponseResponse{
		ID:               r.ID.Hex(),
		RequirementID:    r.RequirementID.Hex(),
		SupplierID:       r.SupplierID.Hex(),
		Score:            r.Score,
		MaxScore:         r.MaxScore,
		Passed:           r.Passed,
		Grade:            r.Grade,
		DraftAnswerCount: r.DraftAnswerCount(),
		IsSubmitted:      r.IsSubmitted(),
		StartedAt:        r.StartedAt,
		SubmittedAt:      r.SubmittedAt,
	}

	// Include draft answers
	resp.DraftAnswers = make([]DraftAnswerResponse, len(r.DraftAnswers))
	for i, a := range r.DraftAnswers {
		resp.DraftAnswers[i] = DraftAnswerResponse{
			QuestionID:      a.QuestionID.Hex(),
			SelectedOptions: a.SelectedOptions,
			TextAnswer:      a.TextAnswer,
			SavedAt:         a.SavedAt,
		}
	}

	return resp
}
