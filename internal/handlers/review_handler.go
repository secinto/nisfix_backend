// Package handlers provides HTTP handlers for API endpoints.
package handlers

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/checkfix-tools/nisfix_backend/internal/middleware"
	"github.com/checkfix-tools/nisfix_backend/internal/services"
)

// ReviewHandler handles requirement review endpoints
// #INTEGRATION_POINT: Company portal uses these endpoints for reviewing supplier submissions
type ReviewHandler struct {
	reviewService services.ReviewService
}

// NewReviewHandler creates a new review handler
func NewReviewHandler(reviewService services.ReviewService) *ReviewHandler {
	return &ReviewHandler{
		reviewService: reviewService,
	}
}

// ReviewActionRequest represents a review action request
type ReviewActionRequest struct {
	Notes  string `json:"notes,omitempty"`
	Reason string `json:"reason,omitempty"`
}

// ReviewSubmissionResponse represents a submission for review
type ReviewSubmissionResponse struct {
	Requirement RequirementResponse      `json:"requirement"`
	Response    *ReviewResponseDetails   `json:"response,omitempty"`
	Submission  *ReviewSubmissionDetails `json:"submission,omitempty"`
}

// ReviewResponseDetails represents response details in review
type ReviewResponseDetails struct {
	ID          string     `json:"id"`
	Score       *int       `json:"score,omitempty"`
	MaxScore    *int       `json:"max_score,omitempty"`
	Passed      *bool      `json:"passed,omitempty"`
	Grade       *string    `json:"grade,omitempty"`
	IsSubmitted bool       `json:"is_submitted"`
	StartedAt   time.Time  `json:"started_at"`
	SubmittedAt *time.Time `json:"submitted_at,omitempty"`
	IsReviewed  bool       `json:"is_reviewed"`
	ReviewedAt  *time.Time `json:"reviewed_at,omitempty"`
	ReviewNotes string     `json:"review_notes,omitempty"`
}

// ReviewSubmissionDetails represents submission details in review
type ReviewSubmissionDetails struct {
	ID               string                     `json:"id"`
	TotalScore       int                        `json:"total_score"`
	MaxPossibleScore int                        `json:"max_possible_score"`
	PercentageScore  float64                    `json:"percentage_score"`
	Passed           bool                       `json:"passed"`
	MustPassFailed   bool                       `json:"must_pass_failed"`
	TopicScores      []TopicScoreResponse       `json:"topic_scores"`
	Answers          []SubmissionAnswerResponse `json:"answers"`
	CompletionMins   int                        `json:"completion_time_minutes"`
}

// TopicScoreResponse represents a topic score
type TopicScoreResponse struct {
	TopicID         string  `json:"topic_id"`
	TopicName       string  `json:"topic_name"`
	Score           int     `json:"score"`
	MaxScore        int     `json:"max_score"`
	PercentageScore float64 `json:"percentage_score"`
}

// SubmissionAnswerResponse represents an answer in submission
type SubmissionAnswerResponse struct {
	QuestionID      string   `json:"question_id"`
	SelectedOptions []string `json:"selected_options,omitempty"`
	TextAnswer      string   `json:"text_answer,omitempty"`
	PointsEarned    int      `json:"points_earned"`
	MaxPoints       int      `json:"max_points"`
	IsMustPassMet   *bool    `json:"is_must_pass_met,omitempty"`
}

// GetSubmissionForReview handles GET /api/v1/requirements/:id/review
// @Summary Get submission for review
// @Description Gets the submission details for reviewing
// @Tags Review
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Requirement ID"
// @Success 200 {object} ReviewSubmissionResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /requirements/{id}/review [get]
func (h *ReviewHandler) GetSubmissionForReview(c *gin.Context) {
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

	result, err := h.reviewService.GetSubmissionForReview(c.Request.Context(), requirementID, companyID)
	if err != nil {
		if errors.Is(err, services.ErrRequirementNotFound) {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "not_found",
				Message: "Requirement not found",
			})
			return
		}
		if errors.Is(err, services.ErrNoSubmission) {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "no_submission",
				Message: "No submission to review",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to get submission",
		})
		return
	}

	resp := ReviewSubmissionResponse{
		Requirement: toRequirementResponse(result.Requirement),
	}

	if result.Response != nil {
		resp.Response = &ReviewResponseDetails{
			ID:          result.Response.ID.Hex(),
			Score:       result.Response.Score,
			MaxScore:    result.Response.MaxScore,
			Passed:      result.Response.Passed,
			Grade:       result.Response.Grade,
			IsSubmitted: result.Response.IsSubmitted(),
			StartedAt:   result.Response.StartedAt,
			SubmittedAt: result.Response.SubmittedAt,
			IsReviewed:  result.Response.IsReviewed(),
			ReviewedAt:  result.Response.ReviewedAt,
			ReviewNotes: result.Response.ReviewNotes,
		}
	}

	if result.Submission != nil {
		topicScores := make([]TopicScoreResponse, len(result.Submission.TopicScores))
		for i, ts := range result.Submission.TopicScores {
			topicScores[i] = TopicScoreResponse{
				TopicID:         ts.TopicID,
				TopicName:       ts.TopicName,
				Score:           ts.Score,
				MaxScore:        ts.MaxScore,
				PercentageScore: ts.PercentageScore,
			}
		}

		answers := make([]SubmissionAnswerResponse, len(result.Submission.Answers))
		for i, a := range result.Submission.Answers {
			answers[i] = SubmissionAnswerResponse{
				QuestionID:      a.QuestionID.Hex(),
				SelectedOptions: a.SelectedOptions,
				TextAnswer:      a.TextAnswer,
				PointsEarned:    a.PointsEarned,
				MaxPoints:       a.MaxPoints,
				IsMustPassMet:   a.IsMustPassMet,
			}
		}

		resp.Submission = &ReviewSubmissionDetails{
			ID:               result.Submission.ID.Hex(),
			TotalScore:       result.Submission.TotalScore,
			MaxPossibleScore: result.Submission.MaxPossibleScore,
			PercentageScore:  result.Submission.PercentageScore,
			Passed:           result.Submission.Passed,
			MustPassFailed:   result.Submission.MustPassFailed,
			TopicScores:      topicScores,
			Answers:          answers,
			CompletionMins:   result.Submission.CompletionTimeMinutes,
		}
	}

	c.JSON(http.StatusOK, resp)
}

// ApproveRequirement handles POST /api/v1/requirements/:id/approve
// @Summary Approve requirement
// @Description Approves a submitted requirement
// @Tags Review
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Requirement ID"
// @Param request body ReviewActionRequest false "Approval notes"
// @Success 200 {object} RequirementResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /requirements/{id}/approve [post]
func (h *ReviewHandler) ApproveRequirement(c *gin.Context) {
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

	requirementID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid requirement ID",
		})
		return
	}

	var req ReviewActionRequest
	_ = c.ShouldBindJSON(&req)

	requirement, err := h.reviewService.ApproveRequirement(c.Request.Context(), requirementID, companyID, userID, req.Notes)
	if err != nil {
		if errors.Is(err, services.ErrRequirementNotFound) {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "not_found",
				Message: "Requirement not found",
			})
			return
		}
		if errors.Is(err, services.ErrCannotReview) {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "cannot_review",
				Message: "Cannot approve this requirement",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to approve requirement",
		})
		return
	}

	c.JSON(http.StatusOK, toRequirementResponse(requirement))
}

// RejectRequirement handles POST /api/v1/requirements/:id/reject
// @Summary Reject requirement
// @Description Rejects a submitted requirement
// @Tags Review
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Requirement ID"
// @Param request body ReviewActionRequest true "Rejection reason"
// @Success 200 {object} RequirementResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /requirements/{id}/reject [post]
func (h *ReviewHandler) RejectRequirement(c *gin.Context) {
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

	requirementID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid requirement ID",
		})
		return
	}

	var req ReviewActionRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Reason == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: "Rejection reason is required",
		})
		return
	}

	requirement, err := h.reviewService.RejectRequirement(c.Request.Context(), requirementID, companyID, userID, req.Reason)
	if err != nil {
		if errors.Is(err, services.ErrRequirementNotFound) {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "not_found",
				Message: "Requirement not found",
			})
			return
		}
		if errors.Is(err, services.ErrCannotReview) {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "cannot_review",
				Message: "Cannot reject this requirement",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to reject requirement",
		})
		return
	}

	c.JSON(http.StatusOK, toRequirementResponse(requirement))
}

// RequestRevision handles POST /api/v1/requirements/:id/request-revision
// @Summary Request revision
// @Description Requests revision for a submitted requirement
// @Tags Review
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Requirement ID"
// @Param request body ReviewActionRequest true "Revision reason"
// @Success 200 {object} RequirementResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 403 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Router /requirements/{id}/request-revision [post]
func (h *ReviewHandler) RequestRevision(c *gin.Context) {
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

	requirementID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid requirement ID",
		})
		return
	}

	var req ReviewActionRequest
	if err := c.ShouldBindJSON(&req); err != nil || req.Reason == "" {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: "Revision reason is required",
		})
		return
	}

	requirement, err := h.reviewService.RequestRevision(c.Request.Context(), requirementID, companyID, userID, req.Reason)
	if err != nil {
		if errors.Is(err, services.ErrRequirementNotFound) {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "not_found",
				Message: "Requirement not found",
			})
			return
		}
		if errors.Is(err, services.ErrCannotReview) {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "cannot_review",
				Message: "Cannot request revision for this requirement",
			})
			return
		}

		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "internal_error",
			Message: "Failed to request revision",
		})
		return
	}

	c.JSON(http.StatusOK, toRequirementResponse(requirement))
}

// RegisterRoutes registers review handler routes
// #INTEGRATION_POINT: Routes require authentication and company organization type
func (h *ReviewHandler) RegisterRoutes(rg *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	// Review routes are under requirements
	requirements := rg.Group("/requirements")
	requirements.Use(authMiddleware)
	requirements.Use(middleware.RequireCompany())
	{
		requirements.GET("/:id/review", h.GetSubmissionForReview)
		requirements.POST("/:id/approve", h.ApproveRequirement)
		requirements.POST("/:id/reject", h.RejectRequirement)
		requirements.POST("/:id/request-revision", h.RequestRevision)
	}
}
