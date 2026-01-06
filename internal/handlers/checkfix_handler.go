// Package handlers provides HTTP handlers for API endpoints.
package handlers

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/checkfix-tools/nisfix_backend/internal/middleware"
	"github.com/checkfix-tools/nisfix_backend/internal/models"
	"github.com/checkfix-tools/nisfix_backend/internal/services"
)

// CheckFixHandler handles CheckFix integration endpoints
// #INTEGRATION_POINT: Supplier portal uses these for CheckFix linking and verification
type CheckFixHandler struct {
	checkFixService services.CheckFixService
}

// NewCheckFixHandler creates a new CheckFix handler
func NewCheckFixHandler(checkFixService services.CheckFixService) *CheckFixHandler {
	return &CheckFixHandler{
		checkFixService: checkFixService,
	}
}

// CheckFixStatusResponse represents the CheckFix status response
type CheckFixStatusResponse struct {
	IsLinked         bool                          `json:"is_linked"`
	AccountID        string                        `json:"account_id,omitempty"`
	Domain           string                        `json:"domain,omitempty"`
	LinkedAt         *time.Time                    `json:"linked_at,omitempty"`
	LatestGrade      *string                       `json:"latest_grade,omitempty"`
	LatestVerifiedAt *time.Time                    `json:"latest_verified_at,omitempty"`
	LatestScore      *int                          `json:"latest_score,omitempty"`
	Verification     *CheckFixVerificationResponse `json:"verification,omitempty"`
}

// CheckFixVerificationResponse represents a verification in API responses
type CheckFixVerificationResponse struct {
	ID               string                  `json:"id"`
	Domain           string                  `json:"domain"`
	VerifiedDomain   string                  `json:"verified_domain"`
	DomainMatch      bool                    `json:"domain_match"`
	ReportHash       string                  `json:"report_hash"`
	ReportDate       time.Time               `json:"report_date"`
	OverallGrade     string                  `json:"overall_grade"`
	OverallScore     int                     `json:"overall_score"`
	CategoryGrades   []CategoryGradeResponse `json:"category_grades"`
	CriticalFindings int                     `json:"critical_findings"`
	HighFindings     int                     `json:"high_findings"`
	MediumFindings   int                     `json:"medium_findings"`
	LowFindings      int                     `json:"low_findings"`
	VerifiedAt       time.Time               `json:"verified_at"`
	ExpiresAt        time.Time               `json:"expires_at"`
	IsValid          bool                    `json:"is_valid"`
	DaysUntilExpiry  int                     `json:"days_until_expiry"`
}

// CategoryGradeResponse represents a category grade
type CategoryGradeResponse struct {
	Category string `json:"category"`
	Grade    string `json:"grade"`
	Score    int    `json:"score"`
}

// LinkAccountRequest represents the link account request
type LinkAccountRequest struct {
	AccountID string `json:"account_id" binding:"required"`
}

// VerifyReportRequest represents the verify report request
type VerifyReportRequest struct {
	ReportHash string `json:"report_hash" binding:"required"`
}

// SubmitCheckFixRequest represents a CheckFix submission request
type SubmitCheckFixRequest struct {
	ReportHash string `json:"report_hash" binding:"required"`
}

// CheckFixSubmissionResponse represents the submission result
type CheckFixSubmissionResponse struct {
	Passed       bool                          `json:"passed"`
	Grade        string                        `json:"grade"`
	Message      string                        `json:"message"`
	Verification *CheckFixVerificationResponse `json:"verification"`
}

// GetStatus handles GET /api/v1/supplier/checkfix/status
// @Summary Get CheckFix status
// @Description Gets the current CheckFix integration status for the supplier
// @Tags CheckFix
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} CheckFixStatusResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Router /supplier/checkfix/status [get]
func (h *CheckFixHandler) GetStatus(c *gin.Context) {
	supplierID, ok := middleware.GetOrgID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "Invalid session",
		})
		return
	}

	status, err := h.checkFixService.GetLinkStatus(c.Request.Context(), supplierID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to get CheckFix status",
		})
		return
	}

	resp := CheckFixStatusResponse{
		IsLinked:  status.IsLinked,
		AccountID: status.AccountID,
		Domain:    status.Domain,
		LinkedAt:  status.LinkedAt,
	}

	if status.LatestGrade != nil {
		gradeStr := string(*status.LatestGrade)
		resp.LatestGrade = &gradeStr
	}
	resp.LatestVerifiedAt = status.LatestVerifiedAt

	if status.Verification != nil {
		resp.LatestScore = &status.Verification.OverallScore
		resp.Verification = toCheckFixVerificationResponse(status.Verification)
	}

	c.JSON(http.StatusOK, resp)
}

// LinkAccount handles POST /api/v1/supplier/checkfix/link
// @Summary Link CheckFix account
// @Description Links the supplier's CheckFix account
// @Tags CheckFix
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body LinkAccountRequest true "Account to link"
// @Success 200 {object} CheckFixStatusResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Router /supplier/checkfix/link [post]
func (h *CheckFixHandler) LinkAccount(c *gin.Context) {
	supplierID, ok := middleware.GetOrgID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "Invalid session",
		})
		return
	}

	var req LinkAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: "Account ID is required",
		})
		return
	}

	if err := h.checkFixService.LinkAccount(c.Request.Context(), supplierID, req.AccountID); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "link_failed",
			Message: err.Error(),
		})
		return
	}

	// Return updated status
	status, err := h.checkFixService.GetLinkStatus(c.Request.Context(), supplierID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to get link status",
		})
		return
	}
	resp := CheckFixStatusResponse{
		IsLinked:  status.IsLinked,
		AccountID: status.AccountID,
		Domain:    status.Domain,
		LinkedAt:  status.LinkedAt,
	}

	c.JSON(http.StatusOK, resp)
}

// UnlinkAccount handles DELETE /api/v1/supplier/checkfix/link
// @Summary Unlink CheckFix account
// @Description Removes the CheckFix account link
// @Tags CheckFix
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} map[string]string
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Router /supplier/checkfix/link [delete]
func (h *CheckFixHandler) UnlinkAccount(c *gin.Context) {
	supplierID, ok := middleware.GetOrgID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "Invalid session",
		})
		return
	}

	if err := h.checkFixService.UnlinkAccount(c.Request.Context(), supplierID); err != nil {
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to unlink account",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Account unlinked successfully"})
}

// VerifyReport handles POST /api/v1/supplier/checkfix/verify
// @Summary Verify a CheckFix report
// @Description Verifies a CheckFix report and stores the verification
// @Tags CheckFix
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body VerifyReportRequest true "Report to verify"
// @Success 200 {object} CheckFixVerificationResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Router /supplier/checkfix/verify [post]
func (h *CheckFixHandler) VerifyReport(c *gin.Context) {
	supplierID, ok := middleware.GetOrgID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "Invalid session",
		})
		return
	}

	var req VerifyReportRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: "Report hash is required",
		})
		return
	}

	// Use a zero ObjectID for standalone verification (not tied to a response)
	verification, err := h.checkFixService.VerifyReport(c.Request.Context(), supplierID, primitive.NilObjectID, req.ReportHash)
	if err != nil {
		if errors.Is(err, services.ErrCheckFixNotLinked) {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "not_linked",
				Message: "CheckFix account is not linked",
			})
			return
		}
		if errors.Is(err, services.ErrCheckFixReportNotFound) {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "report_not_found",
				Message: "CheckFix report not found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "verification_failed",
			Message: "Failed to verify report",
		})
		return
	}

	c.JSON(http.StatusOK, toCheckFixVerificationResponse(verification))
}

// SubmitCheckFix handles POST /api/v1/supplier/requirements/:id/checkfix
// @Summary Submit CheckFix verification for a requirement
// @Description Submits a CheckFix report verification as a response to a requirement
// @Tags CheckFix
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Requirement ID"
// @Param request body SubmitCheckFixRequest true "CheckFix report to submit"
// @Success 200 {object} CheckFixSubmissionResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /supplier/requirements/{id}/checkfix [post]
func (h *CheckFixHandler) SubmitCheckFix(c *gin.Context) {
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

	var req SubmitCheckFixRequest
	if bindErr := c.ShouldBindJSON(&req); bindErr != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: "Report hash is required",
		})
		return
	}

	result, err := h.checkFixService.SubmitCheckFixResponse(c.Request.Context(), requirementID, supplierID, req.ReportHash)
	if err != nil {
		if errors.Is(err, services.ErrRequirementNotFound) {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "not_found",
				Message: "Requirement not found",
			})
			return
		}
		if errors.Is(err, services.ErrCheckFixNotLinked) {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "not_linked",
				Message: "CheckFix account is not linked",
			})
			return
		}
		if errors.Is(err, services.ErrCheckFixReportNotFound) {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "report_not_found",
				Message: "CheckFix report not found",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "submission_failed",
			Message: "Failed to submit CheckFix verification",
		})
		return
	}

	c.JSON(http.StatusOK, CheckFixSubmissionResponse{
		Passed:       result.Passed,
		Grade:        string(result.Grade),
		Message:      result.Message,
		Verification: toCheckFixVerificationResponse(result.Verification),
	})
}

// GetRequirementVerification handles GET /api/v1/requirements/:id/checkfix
// @Summary Get CheckFix verification for a requirement
// @Description Gets the CheckFix verification details for a requirement (company view)
// @Tags CheckFix
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Requirement ID"
// @Success 200 {object} CheckFixVerificationResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /requirements/{id}/checkfix [get]
func (h *CheckFixHandler) GetRequirementVerification(c *gin.Context) {
	companyID, ok := middleware.GetOrgID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "Invalid session",
		})
		return
	}

	// This is a placeholder - would need requirement service to verify ownership
	// For now, just return not found as this needs requirement lookup first
	_ = companyID

	c.JSON(http.StatusNotFound, ErrorResponse{
		Error:   "not_found",
		Message: "No CheckFix verification found",
	})
}

// RegisterRoutes registers CheckFix handler routes
// #INTEGRATION_POINT: Supplier routes for CheckFix management
func (h *CheckFixHandler) RegisterRoutes(rg *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	// Supplier CheckFix routes
	supplier := rg.Group("/supplier")
	supplier.Use(authMiddleware)
	supplier.Use(middleware.RequireSupplier())

	checkfix := supplier.Group("/checkfix")
	checkfix.GET("/status", h.GetStatus)
	checkfix.POST("/link", h.LinkAccount)
	checkfix.DELETE("/link", h.UnlinkAccount)
	checkfix.POST("/verify", h.VerifyReport)

	// Submit CheckFix for requirement
	supplier.POST("/requirements/:id/checkfix", h.SubmitCheckFix)

	// Company routes for viewing verifications
	requirements := rg.Group("/requirements")
	requirements.Use(authMiddleware)
	requirements.Use(middleware.RequireCompany())
	requirements.GET("/:id/checkfix", h.GetRequirementVerification)
}

// toCheckFixVerificationResponse converts a verification to API response
func toCheckFixVerificationResponse(v *models.CheckFixVerification) *CheckFixVerificationResponse {
	if v == nil {
		return nil
	}

	categoryGrades := make([]CategoryGradeResponse, len(v.CategoryGrades))
	for i, cg := range v.CategoryGrades {
		categoryGrades[i] = CategoryGradeResponse{
			Category: cg.Category,
			Grade:    cg.Grade,
			Score:    cg.Score,
		}
	}

	return &CheckFixVerificationResponse{
		ID:               v.ID.Hex(),
		Domain:           v.Domain,
		VerifiedDomain:   v.VerifiedDomain,
		DomainMatch:      v.DomainMatch,
		ReportHash:       v.ReportHash,
		ReportDate:       v.ReportDate,
		OverallGrade:     string(v.OverallGrade),
		OverallScore:     v.OverallScore,
		CategoryGrades:   categoryGrades,
		CriticalFindings: v.CriticalFindings,
		HighFindings:     v.HighFindings,
		MediumFindings:   v.MediumFindings,
		LowFindings:      v.LowFindings,
		VerifiedAt:       v.VerifiedAt,
		ExpiresAt:        v.ExpiresAt,
		IsValid:          v.IsValid(),
		DaysUntilExpiry:  v.DaysUntilExpiry(),
	}
}
