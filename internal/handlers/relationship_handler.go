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

// RelationshipHandler handles supplier relationship endpoints
// #INTEGRATION_POINT: Company portal uses these endpoints for supplier management
type RelationshipHandler struct {
	relationshipService services.RelationshipService
}

// NewRelationshipHandler creates a new relationship handler
func NewRelationshipHandler(relationshipService services.RelationshipService) *RelationshipHandler {
	return &RelationshipHandler{
		relationshipService: relationshipService,
	}
}

// InviteSupplierRequest represents the invite supplier request body
type InviteSupplierRequest struct {
	Email            string   `json:"email" binding:"required,email"`
	Classification   string   `json:"classification,omitempty"`
	Notes            string   `json:"notes,omitempty"`
	ServicesProvided []string `json:"services_provided,omitempty"`
	ContractRef      string   `json:"contract_ref,omitempty"`
}

// RelationshipResponse represents a relationship in API responses
type RelationshipResponse struct {
	ID               string                 `json:"id"`
	CompanyID        string                 `json:"company_id"`
	SupplierID       *string                `json:"supplier_id,omitempty"`
	InvitedEmail     string                 `json:"invited_email"`
	Status           string                 `json:"status"`
	Classification   string                 `json:"classification"`
	Notes            string                 `json:"notes,omitempty"`
	ServicesProvided []string               `json:"services_provided,omitempty"`
	ContractRef      string                 `json:"contract_ref,omitempty"`
	InvitedAt        time.Time              `json:"invited_at"`
	AcceptedAt       *time.Time             `json:"accepted_at,omitempty"`
	StatusHistory    []StatusChangeResponse `json:"status_history,omitempty"`
	CreatedAt        time.Time              `json:"created_at"`
	UpdatedAt        time.Time              `json:"updated_at"`
}

// StatusChangeResponse represents a status change in API responses
type StatusChangeResponse struct {
	FromStatus string    `json:"from_status"`
	ToStatus   string    `json:"to_status"`
	Reason     string    `json:"reason,omitempty"`
	ChangedAt  time.Time `json:"changed_at"`
}

// SupplierStatsResponse represents supplier statistics
type SupplierStatsResponse struct {
	Total     int64 `json:"total"`
	Active    int64 `json:"active"`
	Pending   int64 `json:"pending"`
	Suspended int64 `json:"suspended"`
	Critical  int64 `json:"critical"`
	Important int64 `json:"important"`
	Standard  int64 `json:"standard"`
}

// PaginatedRelationshipsResponse represents paginated relationships
type PaginatedRelationshipsResponse struct {
	Items      []RelationshipResponse `json:"items"`
	TotalCount int64                  `json:"total_count"`
	Page       int                    `json:"page"`
	Limit      int                    `json:"limit"`
	TotalPages int                    `json:"total_pages"`
}

// InviteSupplier handles POST /api/v1/suppliers
// @Summary Invite a supplier
// @Description Sends an invitation to a supplier by email
// @Tags Suppliers
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body InviteSupplierRequest true "Invite request"
// @Success 201 {object} RelationshipResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 409 {object} ErrorResponse
// @Router /suppliers [post]
func (h *RelationshipHandler) InviteSupplier(c *gin.Context) {
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

	var req InviteSupplierRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid request body",
		})
		return
	}

	// Convert classification
	classification := models.SupplierClassificationStandard
	if req.Classification != "" {
		classification = models.SupplierClassification(req.Classification)
	}

	serviceReq := services.InviteSupplierRequest{
		Email:            req.Email,
		Classification:   classification,
		Notes:            req.Notes,
		ServicesProvided: req.ServicesProvided,
		ContractRef:      req.ContractRef,
	}

	relationship, err := h.relationshipService.InviteSupplier(c.Request.Context(), companyID, userID, serviceReq)
	if err != nil {
		if errors.Is(err, services.ErrRelationshipExists) {
			c.JSON(http.StatusConflict, ErrorResponse{
				Error:   "relationship_exists",
				Message: "A relationship already exists with this supplier email",
			})
			return
		}
		if errors.Is(err, services.ErrInvalidClassification) {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "invalid_classification",
				Message: "Invalid supplier classification",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to invite supplier",
		})
		return
	}

	c.JSON(http.StatusCreated, toRelationshipResponse(relationship))
}

// ListSuppliers handles GET /api/v1/suppliers
// @Summary List suppliers
// @Description Lists all suppliers for the company
// @Tags Suppliers
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param status query string false "Filter by status"
// @Param classification query string false "Filter by classification"
// @Param page query int false "Page number" default(1)
// @Param limit query int false "Items per page" default(20)
// @Success 200 {object} PaginatedRelationshipsResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Router /suppliers [get]
func (h *RelationshipHandler) ListSuppliers(c *gin.Context) {
	companyID, ok := middleware.GetOrgID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "Invalid session",
		})
		return
	}

	// Parse query parameters
	filters := services.SupplierFilters{}
	if status := c.Query("status"); status != "" {
		s := models.RelationshipStatus(status)
		filters.Status = &s
	}
	if classification := c.Query("classification"); classification != "" {
		cl := models.SupplierClassification(classification)
		filters.Classification = &cl
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
	if sortDir := c.Query("sort_dir"); sortDir == sortDirectionAsc {
		opts.SortDir = 1
	}

	result, err := h.relationshipService.ListCompanySuppliers(c.Request.Context(), companyID, filters, opts)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to list suppliers",
		})
		return
	}

	items := make([]RelationshipResponse, len(result.Items))
	for i, r := range result.Items {
		items[i] = toRelationshipResponse(&r)
	}

	c.JSON(http.StatusOK, PaginatedRelationshipsResponse{
		Items:      items,
		TotalCount: result.TotalCount,
		Page:       result.Page,
		Limit:      result.Limit,
		TotalPages: result.TotalPages,
	})
}

// GetSupplier handles GET /api/v1/suppliers/:id
// @Summary Get supplier details
// @Description Gets details of a specific supplier relationship
// @Tags Suppliers
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Relationship ID"
// @Success 200 {object} RelationshipResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /suppliers/{id} [get]
func (h *RelationshipHandler) GetSupplier(c *gin.Context) {
	companyID, ok := middleware.GetOrgID(c)
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

	relationship, err := h.relationshipService.GetRelationship(c.Request.Context(), relationshipID, &companyID)
	if err != nil {
		if errors.Is(err, services.ErrRelationshipNotFound) {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "not_found",
				Message: "Supplier relationship not found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to get supplier",
		})
		return
	}

	c.JSON(http.StatusOK, toRelationshipResponse(relationship))
}

// UpdateClassificationRequest represents the update classification request
type UpdateClassificationRequest struct {
	Classification string `json:"classification" binding:"required"`
}

// UpdateClassification handles PATCH /api/v1/suppliers/:id/classification
// @Summary Update supplier classification
// @Description Updates the classification of a supplier
// @Tags Suppliers
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Relationship ID"
// @Param request body UpdateClassificationRequest true "Classification update"
// @Success 200 {object} RelationshipResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /suppliers/{id}/classification [patch]
func (h *RelationshipHandler) UpdateClassification(c *gin.Context) {
	companyID, ok := middleware.GetOrgID(c)
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

	var req UpdateClassificationRequest
	if bindErr := c.ShouldBindJSON(&req); bindErr != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: "Classification is required",
		})
		return
	}

	classification := models.SupplierClassification(req.Classification)
	relationship, err := h.relationshipService.UpdateClassification(c.Request.Context(), relationshipID, companyID, classification)
	if err != nil {
		if errors.Is(err, services.ErrRelationshipNotFound) {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "not_found",
				Message: "Supplier relationship not found",
			})
			return
		}
		if errors.Is(err, services.ErrInvalidClassification) {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "invalid_classification",
				Message: "Invalid supplier classification",
			})
			return
		}
		if errors.Is(err, services.ErrCannotModifyRelationship) {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "cannot_modify",
				Message: "Cannot modify terminated relationship",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to update classification",
		})
		return
	}

	c.JSON(http.StatusOK, toRelationshipResponse(relationship))
}

// UpdateDetailsRequest represents the update details request
type UpdateDetailsRequest struct {
	Notes            *string  `json:"notes,omitempty"`
	ServicesProvided []string `json:"services_provided,omitempty"`
	ContractRef      *string  `json:"contract_ref,omitempty"`
}

// UpdateDetails handles PATCH /api/v1/suppliers/:id
// @Summary Update supplier details
// @Description Updates details of a supplier relationship
// @Tags Suppliers
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Relationship ID"
// @Param request body UpdateDetailsRequest true "Details update"
// @Success 200 {object} RelationshipResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /suppliers/{id} [patch]
func (h *RelationshipHandler) UpdateDetails(c *gin.Context) {
	companyID, ok := middleware.GetOrgID(c)
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

	var req UpdateDetailsRequest
	if bindErr := c.ShouldBindJSON(&req); bindErr != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid request body",
		})
		return
	}

	serviceReq := services.UpdateRelationshipRequest{
		Notes:            req.Notes,
		ServicesProvided: req.ServicesProvided,
		ContractRef:      req.ContractRef,
	}

	relationship, err := h.relationshipService.UpdateDetails(c.Request.Context(), relationshipID, companyID, serviceReq)
	if err != nil {
		if errors.Is(err, services.ErrRelationshipNotFound) {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "not_found",
				Message: "Supplier relationship not found",
			})
			return
		}
		if errors.Is(err, services.ErrCannotModifyRelationship) {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "cannot_modify",
				Message: "Cannot modify terminated relationship",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to update details",
		})
		return
	}

	c.JSON(http.StatusOK, toRelationshipResponse(relationship))
}

// StatusActionRequest represents a status action request
type StatusActionRequest struct {
	Reason string `json:"reason,omitempty"`
}

// SuspendSupplier handles POST /api/v1/suppliers/:id/suspend
// @Summary Suspend supplier
// @Description Suspends an active supplier relationship
// @Tags Suppliers
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Relationship ID"
// @Param request body StatusActionRequest false "Suspension reason"
// @Success 200 {object} RelationshipResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /suppliers/{id}/suspend [post]
func (h *RelationshipHandler) SuspendSupplier(c *gin.Context) {
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

	relationshipID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid relationship ID",
		})
		return
	}

	var req StatusActionRequest
	//nolint:errcheck // Optional body - binding failure is acceptable
	c.ShouldBindJSON(&req)

	relationship, err := h.relationshipService.SuspendRelationship(c.Request.Context(), relationshipID, companyID, userID, req.Reason)
	if err != nil {
		if errors.Is(err, services.ErrRelationshipNotFound) {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "not_found",
				Message: "Supplier relationship not found",
			})
			return
		}
		if errors.Is(err, services.ErrInvalidStatusTransition) {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "invalid_transition",
				Message: "Cannot suspend this relationship",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to suspend supplier",
		})
		return
	}

	c.JSON(http.StatusOK, toRelationshipResponse(relationship))
}

// ReactivateSupplier handles POST /api/v1/suppliers/:id/reactivate
// @Summary Reactivate supplier
// @Description Reactivates a suspended supplier relationship
// @Tags Suppliers
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Relationship ID"
// @Param request body StatusActionRequest false "Reactivation reason"
// @Success 200 {object} RelationshipResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /suppliers/{id}/reactivate [post]
func (h *RelationshipHandler) ReactivateSupplier(c *gin.Context) {
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

	relationshipID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid relationship ID",
		})
		return
	}

	var req StatusActionRequest
	//nolint:errcheck // Optional body - binding failure is acceptable
	c.ShouldBindJSON(&req)

	relationship, err := h.relationshipService.ReactivateRelationship(c.Request.Context(), relationshipID, companyID, userID, req.Reason)
	if err != nil {
		if errors.Is(err, services.ErrRelationshipNotFound) {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "not_found",
				Message: "Supplier relationship not found",
			})
			return
		}
		if errors.Is(err, services.ErrInvalidStatusTransition) {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "invalid_transition",
				Message: "Cannot reactivate this relationship",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to reactivate supplier",
		})
		return
	}

	c.JSON(http.StatusOK, toRelationshipResponse(relationship))
}

// TerminateSupplier handles POST /api/v1/suppliers/:id/terminate
// @Summary Terminate supplier
// @Description Terminates a supplier relationship
// @Tags Suppliers
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Relationship ID"
// @Param request body StatusActionRequest false "Termination reason"
// @Success 200 {object} RelationshipResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /suppliers/{id}/terminate [post]
func (h *RelationshipHandler) TerminateSupplier(c *gin.Context) {
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

	relationshipID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid relationship ID",
		})
		return
	}

	var req StatusActionRequest
	//nolint:errcheck // Optional body - binding failure is acceptable
	c.ShouldBindJSON(&req)

	relationship, err := h.relationshipService.TerminateRelationship(c.Request.Context(), relationshipID, companyID, userID, req.Reason)
	if err != nil {
		if errors.Is(err, services.ErrRelationshipNotFound) {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "not_found",
				Message: "Supplier relationship not found",
			})
			return
		}
		if errors.Is(err, services.ErrInvalidStatusTransition) {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "invalid_transition",
				Message: "Cannot terminate this relationship",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to terminate supplier",
		})
		return
	}

	c.JSON(http.StatusOK, toRelationshipResponse(relationship))
}

// GetSupplierStats handles GET /api/v1/suppliers/stats
// @Summary Get supplier statistics
// @Description Gets supplier statistics for the company
// @Tags Suppliers
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} SupplierStatsResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Router /suppliers/stats [get]
func (h *RelationshipHandler) GetSupplierStats(c *gin.Context) {
	companyID, ok := middleware.GetOrgID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "Invalid session",
		})
		return
	}

	stats, err := h.relationshipService.GetSupplierStats(c.Request.Context(), companyID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to get supplier stats",
		})
		return
	}

	c.JSON(http.StatusOK, SupplierStatsResponse{
		Total:     stats.Total,
		Active:    stats.Active,
		Pending:   stats.Pending,
		Suspended: stats.Suspended,
		Critical:  stats.Critical,
		Important: stats.Important,
		Standard:  stats.Standard,
	})
}

// RegisterRoutes registers relationship handler routes
// #INTEGRATION_POINT: Routes require authentication and company organization type
func (h *RelationshipHandler) RegisterRoutes(rg *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	suppliers := rg.Group("/suppliers")
	suppliers.Use(authMiddleware)
	suppliers.Use(middleware.RequireCompany())
	suppliers.POST("", h.InviteSupplier)
	suppliers.GET("", h.ListSuppliers)
	suppliers.GET("/stats", h.GetSupplierStats)
	suppliers.GET("/:id", h.GetSupplier)
	suppliers.PATCH("/:id", h.UpdateDetails)
	suppliers.PATCH("/:id/classification", h.UpdateClassification)
	suppliers.POST("/:id/suspend", h.SuspendSupplier)
	suppliers.POST("/:id/reactivate", h.ReactivateSupplier)
	suppliers.POST("/:id/terminate", h.TerminateSupplier)
}

// toRelationshipResponse converts a relationship model to response
func toRelationshipResponse(r *models.CompanySupplierRelationship) RelationshipResponse {
	resp := RelationshipResponse{
		ID:               r.ID.Hex(),
		CompanyID:        r.CompanyID.Hex(),
		InvitedEmail:     r.InvitedEmail,
		Status:           string(r.Status),
		Classification:   string(r.Classification),
		Notes:            r.Notes,
		ServicesProvided: r.ServicesProvided,
		ContractRef:      r.ContractRef,
		InvitedAt:        r.InvitedAt,
		AcceptedAt:       r.AcceptedAt,
		CreatedAt:        r.CreatedAt,
		UpdatedAt:        r.UpdatedAt,
	}

	if r.SupplierID != nil {
		supplierID := r.SupplierID.Hex()
		resp.SupplierID = &supplierID
	}

	// Include status history
	resp.StatusHistory = make([]StatusChangeResponse, len(r.StatusHistory))
	for i, change := range r.StatusHistory {
		resp.StatusHistory[i] = StatusChangeResponse{
			FromStatus: string(change.FromStatus),
			ToStatus:   string(change.ToStatus),
			Reason:     change.Reason,
			ChangedAt:  change.ChangedAt,
		}
	}

	return resp
}
