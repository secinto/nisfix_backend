// Package handlers provides HTTP handlers for API endpoints.
package handlers

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/checkfix-tools/nisfix_backend/internal/middleware"
	"github.com/checkfix-tools/nisfix_backend/internal/models"
	"github.com/checkfix-tools/nisfix_backend/internal/repository"
)

// OrganizationHandler handles organization management endpoints
// #INTEGRATION_POINT: Used by both company and supplier portals for settings
type OrganizationHandler struct {
	orgRepo repository.OrganizationRepository
}

// NewOrganizationHandler creates a new organization handler
func NewOrganizationHandler(orgRepo repository.OrganizationRepository) *OrganizationHandler {
	return &OrganizationHandler{
		orgRepo: orgRepo,
	}
}

// OrganizationResponse represents an organization in API responses
type OrganizationResponse struct {
	ID           string                       `json:"id"`
	Type         string                       `json:"type"`
	Name         string                       `json:"name"`
	Slug         string                       `json:"slug"`
	Domain       string                       `json:"domain,omitempty"`
	ContactEmail string                       `json:"contact_email,omitempty"`
	ContactPhone string                       `json:"contact_phone,omitempty"`
	Address      *AddressResponse             `json:"address,omitempty"`
	Settings     OrganizationSettingsResponse `json:"settings"`
	CreatedAt    time.Time                    `json:"created_at"`
	UpdatedAt    time.Time                    `json:"updated_at"`
}

// AddressResponse represents an address in API responses
type AddressResponse struct {
	Street     string `json:"street,omitempty"`
	City       string `json:"city,omitempty"`
	PostalCode string `json:"postal_code,omitempty"`
	Country    string `json:"country,omitempty"`
}

// OrganizationSettingsResponse represents organization settings
type OrganizationSettingsResponse struct {
	DefaultDueDays       int      `json:"default_due_days"`
	RequireCheckFix      bool     `json:"require_checkfix"`
	MinCheckFixGrade     string   `json:"min_checkfix_grade"`
	NotificationEmails   []string `json:"notification_emails"`
	DefaultLanguage      string   `json:"default_language"`
	NotificationsEnabled bool     `json:"notifications_enabled"`
}

// UpdateOrganizationRequest represents an organization update request
type UpdateOrganizationRequest struct {
	Name         *string                `json:"name,omitempty"`
	Domain       *string                `json:"domain,omitempty"`
	ContactEmail *string                `json:"contact_email,omitempty"`
	ContactPhone *string                `json:"contact_phone,omitempty"`
	Address      *UpdateAddressRequest  `json:"address,omitempty"`
	Settings     *UpdateSettingsRequest `json:"settings,omitempty"`
}

// UpdateAddressRequest represents an address update
type UpdateAddressRequest struct {
	Street     *string `json:"street,omitempty"`
	City       *string `json:"city,omitempty"`
	PostalCode *string `json:"postal_code,omitempty"`
	Country    *string `json:"country,omitempty"`
}

// UpdateSettingsRequest represents settings update
type UpdateSettingsRequest struct {
	DefaultDueDays       *int     `json:"default_due_days,omitempty"`
	RequireCheckFix      *bool    `json:"require_checkfix,omitempty"`
	MinCheckFixGrade     *string  `json:"min_checkfix_grade,omitempty"`
	NotificationEmails   []string `json:"notification_emails,omitempty"`
	DefaultLanguage      *string  `json:"default_language,omitempty"`
	NotificationsEnabled *bool    `json:"notifications_enabled,omitempty"`
}

// GetOrganization handles GET /api/v1/organization
// @Summary Get current organization
// @Description Gets the current user's organization details
// @Tags Organization
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} OrganizationResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /organization [get]
func (h *OrganizationHandler) GetOrganization(c *gin.Context) {
	orgID, ok := middleware.GetOrgID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "Invalid session",
		})
		return
	}

	org, err := h.orgRepo.GetByID(c.Request.Context(), orgID)
	if err != nil {
		if errors.Is(err, models.ErrOrganizationNotFound) {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "not_found",
				Message: "Organization not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to get organization",
		})
		return
	}

	c.JSON(http.StatusOK, toOrganizationResponse(org))
}

// UpdateOrganization handles PATCH /api/v1/organization
// @Summary Update organization
// @Description Updates the current user's organization
// @Tags Organization
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body UpdateOrganizationRequest true "Organization updates"
// @Success 200 {object} OrganizationResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /organization [patch]
func (h *OrganizationHandler) UpdateOrganization(c *gin.Context) {
	orgID, ok := middleware.GetOrgID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "Invalid session",
		})
		return
	}

	var req UpdateOrganizationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid request body",
		})
		return
	}

	org, err := h.orgRepo.GetByID(c.Request.Context(), orgID)
	if err != nil {
		if errors.Is(err, models.ErrOrganizationNotFound) {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "not_found",
				Message: "Organization not found",
			})
			return
		}
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to get organization",
		})
		return
	}

	// Apply updates
	if req.Name != nil {
		org.Name = *req.Name
	}
	if req.Domain != nil {
		org.Domain = *req.Domain
	}
	if req.ContactEmail != nil {
		org.ContactEmail = *req.ContactEmail
	}
	if req.ContactPhone != nil {
		org.ContactPhone = *req.ContactPhone
	}

	// Update address
	if req.Address != nil {
		if org.Address == nil {
			org.Address = &models.Address{}
		}
		if req.Address.Street != nil {
			org.Address.Street = *req.Address.Street
		}
		if req.Address.City != nil {
			org.Address.City = *req.Address.City
		}
		if req.Address.PostalCode != nil {
			org.Address.PostalCode = *req.Address.PostalCode
		}
		if req.Address.Country != nil {
			org.Address.Country = *req.Address.Country
		}
	}

	// Update settings
	if req.Settings != nil {
		if req.Settings.DefaultDueDays != nil {
			org.Settings.DefaultDueDays = *req.Settings.DefaultDueDays
		}
		if req.Settings.RequireCheckFix != nil {
			org.Settings.RequireCheckFix = *req.Settings.RequireCheckFix
		}
		if req.Settings.MinCheckFixGrade != nil {
			org.Settings.MinCheckFixGrade = *req.Settings.MinCheckFixGrade
		}
		if req.Settings.NotificationEmails != nil {
			org.Settings.NotificationEmails = req.Settings.NotificationEmails
		}
		if req.Settings.DefaultLanguage != nil {
			org.Settings.DefaultLanguage = *req.Settings.DefaultLanguage
		}
		if req.Settings.NotificationsEnabled != nil {
			org.Settings.NotificationsEnabled = *req.Settings.NotificationsEnabled
		}
	}

	org.BeforeUpdate()

	if err := h.orgRepo.Update(c.Request.Context(), org); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to update organization",
		})
		return
	}

	c.JSON(http.StatusOK, toOrganizationResponse(org))
}

// GetOrganizationSettings handles GET /api/v1/organization/settings
// @Summary Get organization settings
// @Description Gets the current organization's settings
// @Tags Organization
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} OrganizationSettingsResponse
// @Failure 401 {object} ErrorResponse
// @Router /organization/settings [get]
func (h *OrganizationHandler) GetOrganizationSettings(c *gin.Context) {
	orgID, ok := middleware.GetOrgID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "Invalid session",
		})
		return
	}

	org, err := h.orgRepo.GetByID(c.Request.Context(), orgID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to get organization",
		})
		return
	}

	c.JSON(http.StatusOK, OrganizationSettingsResponse{
		DefaultDueDays:       org.Settings.DefaultDueDays,
		RequireCheckFix:      org.Settings.RequireCheckFix,
		MinCheckFixGrade:     org.Settings.MinCheckFixGrade,
		NotificationEmails:   org.Settings.NotificationEmails,
		DefaultLanguage:      org.Settings.DefaultLanguage,
		NotificationsEnabled: org.Settings.NotificationsEnabled,
	})
}

// UpdateOrganizationSettings handles PATCH /api/v1/organization/settings
// @Summary Update organization settings
// @Description Updates the current organization's settings
// @Tags Organization
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body UpdateSettingsRequest true "Settings updates"
// @Success 200 {object} OrganizationSettingsResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Router /organization/settings [patch]
func (h *OrganizationHandler) UpdateOrganizationSettings(c *gin.Context) {
	orgID, ok := middleware.GetOrgID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "Invalid session",
		})
		return
	}

	var req UpdateSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid request body",
		})
		return
	}

	org, err := h.orgRepo.GetByID(c.Request.Context(), orgID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to get organization",
		})
		return
	}

	// Apply updates
	if req.DefaultDueDays != nil {
		org.Settings.DefaultDueDays = *req.DefaultDueDays
	}
	if req.RequireCheckFix != nil {
		org.Settings.RequireCheckFix = *req.RequireCheckFix
	}
	if req.MinCheckFixGrade != nil {
		org.Settings.MinCheckFixGrade = *req.MinCheckFixGrade
	}
	if req.NotificationEmails != nil {
		org.Settings.NotificationEmails = req.NotificationEmails
	}
	if req.DefaultLanguage != nil {
		org.Settings.DefaultLanguage = *req.DefaultLanguage
	}
	if req.NotificationsEnabled != nil {
		org.Settings.NotificationsEnabled = *req.NotificationsEnabled
	}

	org.BeforeUpdate()

	if err := h.orgRepo.Update(c.Request.Context(), org); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to update settings",
		})
		return
	}

	c.JSON(http.StatusOK, OrganizationSettingsResponse{
		DefaultDueDays:       org.Settings.DefaultDueDays,
		RequireCheckFix:      org.Settings.RequireCheckFix,
		MinCheckFixGrade:     org.Settings.MinCheckFixGrade,
		NotificationEmails:   org.Settings.NotificationEmails,
		DefaultLanguage:      org.Settings.DefaultLanguage,
		NotificationsEnabled: org.Settings.NotificationsEnabled,
	})
}

// RegisterRoutes registers organization handler routes
func (h *OrganizationHandler) RegisterRoutes(rg *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	org := rg.Group("/organization")
	org.Use(authMiddleware)
	org.GET("", h.GetOrganization)
	org.PATCH("", h.UpdateOrganization)
	org.GET("/settings", h.GetOrganizationSettings)
	org.PATCH("/settings", h.UpdateOrganizationSettings)
}

// toOrganizationResponse converts an organization to API response
func toOrganizationResponse(org *models.Organization) OrganizationResponse {
	resp := OrganizationResponse{
		ID:           org.ID.Hex(),
		Type:         string(org.Type),
		Name:         org.Name,
		Slug:         org.Slug,
		Domain:       org.Domain,
		ContactEmail: org.ContactEmail,
		ContactPhone: org.ContactPhone,
		Settings: OrganizationSettingsResponse{
			DefaultDueDays:       org.Settings.DefaultDueDays,
			RequireCheckFix:      org.Settings.RequireCheckFix,
			MinCheckFixGrade:     org.Settings.MinCheckFixGrade,
			NotificationEmails:   org.Settings.NotificationEmails,
			DefaultLanguage:      org.Settings.DefaultLanguage,
			NotificationsEnabled: org.Settings.NotificationsEnabled,
		},
		CreatedAt: org.CreatedAt,
		UpdatedAt: org.UpdatedAt,
	}

	if org.Address != nil {
		resp.Address = &AddressResponse{
			Street:     org.Address.Street,
			City:       org.Address.City,
			PostalCode: org.Address.PostalCode,
			Country:    org.Address.Country,
		}
	}

	return resp
}
