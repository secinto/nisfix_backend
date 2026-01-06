// Package handlers provides HTTP handlers for API endpoints.
// #IMPLEMENTATION_DECISION: Handlers are thin - delegate business logic to services
package handlers

import (
	"errors"
	"net/http"

	"github.com/checkfix-tools/nisfix_backend/internal/middleware"
	"github.com/checkfix-tools/nisfix_backend/internal/models"
	"github.com/checkfix-tools/nisfix_backend/internal/services"
	"github.com/gin-gonic/gin"
)

// AuthHandler handles authentication endpoints
// #INTEGRATION_POINT: Frontend auth flow uses these endpoints
type AuthHandler struct {
	authService services.AuthService
}

// NewAuthHandler creates a new auth handler
func NewAuthHandler(authService services.AuthService) *AuthHandler {
	return &AuthHandler{
		authService: authService,
	}
}

// RequestMagicLinkRequest represents the magic link request body
type RequestMagicLinkRequest struct {
	Email string `json:"email" binding:"required,email"`
}

// RequestMagicLinkResponse represents the magic link response
type RequestMagicLinkResponse struct {
	Message string `json:"message"`
}

// RequestMagicLink handles POST /api/v1/auth/magic-link
// @Summary Request a magic link
// @Description Sends a magic link to the provided email for passwordless authentication
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body RequestMagicLinkRequest true "Magic link request"
// @Success 200 {object} RequestMagicLinkResponse
// @Failure 400 {object} ErrorResponse
// @Failure 429 {object} ErrorResponse
// @Router /auth/magic-link [post]
func (h *AuthHandler) RequestMagicLink(c *gin.Context) {
	var req RequestMagicLinkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid email address",
		})
		return
	}

	err := h.authService.RequestMagicLink(c.Request.Context(), req.Email)
	if err != nil {
		if errors.Is(err, services.ErrRateLimitExceeded) {
			c.JSON(http.StatusTooManyRequests, ErrorResponse{
				Error:   "rate_limit_exceeded",
				Message: "Too many magic link requests. Please try again later.",
			})
			return
		}

		// #SECURITY_CONCERN: Don't reveal internal errors
		// Log the error internally but return generic success
	}

	// #SECURITY_CONCERN: Always return success to prevent email enumeration
	c.JSON(http.StatusOK, RequestMagicLinkResponse{
		Message: "If an account exists with this email, a magic link has been sent.",
	})
}

// VerifyMagicLinkRequest represents the verify request body
type VerifyMagicLinkRequest struct {
	Token string `json:"token" binding:"required"`
}

// VerifyMagicLinkResponse represents the verify response
type VerifyMagicLinkResponse struct {
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
	ExpiresAt    int64        `json:"expires_at"`
	ExpiresIn    int64        `json:"expires_in"`
	User         UserResponse `json:"user"`
	Organization OrgResponse  `json:"organization"`
}

// UserResponse represents user data in API responses
type UserResponse struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Name  string `json:"name,omitempty"`
	Role  string `json:"role"`
}

// OrgResponse represents organization data in API responses
type OrgResponse struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Slug string `json:"slug"`
	Type string `json:"type"`
}

// VerifyMagicLink handles POST /api/v1/auth/verify
// @Summary Verify a magic link token
// @Description Validates the magic link token and returns access/refresh tokens
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body VerifyMagicLinkRequest true "Token verification request"
// @Success 200 {object} VerifyMagicLinkResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Router /auth/verify [post]
func (h *AuthHandler) VerifyMagicLink(c *gin.Context) {
	var req VerifyMagicLinkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: "Token is required",
		})
		return
	}

	tokenPair, user, org, err := h.authService.VerifyMagicLink(c.Request.Context(), req.Token)
	if err != nil {
		statusCode := http.StatusUnauthorized
		message := "Invalid or expired magic link"

		if errors.Is(err, services.ErrUserNotFound) ||
			errors.Is(err, services.ErrUserInactive) ||
			errors.Is(err, services.ErrOrganizationNotFound) ||
			errors.Is(err, services.ErrOrganizationInactive) {
			message = "Account is not available"
		}

		c.JSON(statusCode, ErrorResponse{
			Error:   "authentication_failed",
			Message: message,
		})
		return
	}

	c.JSON(http.StatusOK, VerifyMagicLinkResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresAt:    tokenPair.ExpiresAt.Unix(),
		ExpiresIn:    tokenPair.ExpiresIn,
		User: UserResponse{
			ID:    user.ID.Hex(),
			Email: user.Email,
			Name:  user.Name,
			Role:  string(user.Role),
		},
		Organization: OrgResponse{
			ID:   org.ID.Hex(),
			Name: org.Name,
			Slug: org.Slug,
			Type: string(org.Type),
		},
	})
}

// RefreshTokenRequest represents the refresh token request body
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// RefreshTokenResponse represents the refresh token response
type RefreshTokenResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresAt    int64  `json:"expires_at"`
	ExpiresIn    int64  `json:"expires_in"`
}

// RefreshToken handles POST /api/v1/auth/refresh
// @Summary Refresh access token
// @Description Uses refresh token to generate new access/refresh token pair
// @Tags Auth
// @Accept json
// @Produce json
// @Param request body RefreshTokenRequest true "Refresh token request"
// @Success 200 {object} RefreshTokenResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Router /auth/refresh [post]
func (h *AuthHandler) RefreshToken(c *gin.Context) {
	var req RefreshTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: "Refresh token is required",
		})
		return
	}

	tokenPair, err := h.authService.RefreshAccessToken(c.Request.Context(), req.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "invalid_refresh_token",
			Message: "Invalid or expired refresh token",
		})
		return
	}

	c.JSON(http.StatusOK, RefreshTokenResponse{
		AccessToken:  tokenPair.AccessToken,
		RefreshToken: tokenPair.RefreshToken,
		ExpiresAt:    tokenPair.ExpiresAt.Unix(),
		ExpiresIn:    tokenPair.ExpiresIn,
	})
}

// LogoutResponse represents the logout response
type LogoutResponse struct {
	Message string `json:"message"`
}

// Logout handles POST /api/v1/auth/logout
// @Summary Logout user
// @Description Invalidates the user's session (client should discard tokens)
// @Tags Auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} LogoutResponse
// @Failure 401 {object} ErrorResponse
// @Router /auth/logout [post]
func (h *AuthHandler) Logout(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "Invalid session",
		})
		return
	}

	// Invalidate refresh token (server-side cleanup if implemented)
	_ = h.authService.InvalidateRefreshToken(c.Request.Context(), userID)

	c.JSON(http.StatusOK, LogoutResponse{
		Message: "Successfully logged out",
	})
}

// GetMeResponse represents the current user response
type GetMeResponse struct {
	User         UserResponse `json:"user"`
	Organization OrgResponse  `json:"organization"`
}

// GetMe handles GET /api/v1/auth/me
// @Summary Get current user
// @Description Returns the current authenticated user and their organization
// @Tags Auth
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} GetMeResponse
// @Failure 401 {object} ErrorResponse
// @Router /auth/me [get]
func (h *AuthHandler) GetMe(c *gin.Context) {
	userID, ok := middleware.GetUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "Invalid session",
		})
		return
	}

	user, org, err := h.authService.GetUserContext(c.Request.Context(), userID)
	if err != nil {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "unauthorized",
			Message: "User not found",
		})
		return
	}

	c.JSON(http.StatusOK, GetMeResponse{
		User: UserResponse{
			ID:    user.ID.Hex(),
			Email: user.Email,
			Name:  user.Name,
			Role:  string(user.Role),
		},
		Organization: OrgResponse{
			ID:   org.ID.Hex(),
			Name: org.Name,
			Slug: org.Slug,
			Type: string(org.Type),
		},
	})
}

// RegisterRoutes registers auth handler routes
func (h *AuthHandler) RegisterRoutes(rg *gin.RouterGroup, authMiddleware gin.HandlerFunc) {
	auth := rg.Group("/auth")
	{
		// Public endpoints
		auth.POST("/magic-link", h.RequestMagicLink)
		auth.POST("/verify", h.VerifyMagicLink)
		auth.POST("/refresh", h.RefreshToken)

		// Protected endpoints
		auth.POST("/logout", authMiddleware, h.Logout)
		auth.GET("/me", authMiddleware, h.GetMe)
	}
}

// ErrorResponse represents an API error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// ToUserResponse converts a User model to UserResponse
func ToUserResponse(user *models.User) UserResponse {
	return UserResponse{
		ID:    user.ID.Hex(),
		Email: user.Email,
		Name:  user.Name,
		Role:  string(user.Role),
	}
}

// ToOrgResponse converts an Organization model to OrgResponse
func ToOrgResponse(org *models.Organization) OrgResponse {
	return OrgResponse{
		ID:   org.ID.Hex(),
		Name: org.Name,
		Slug: org.Slug,
		Type: string(org.Type),
	}
}
