// Package services provides business logic implementations.
// #IMPLEMENTATION_DECISION: Services orchestrate repositories and external services
package services

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/checkfix-tools/nisfix_backend/internal/auth"
	"github.com/checkfix-tools/nisfix_backend/internal/models"
	"github.com/checkfix-tools/nisfix_backend/internal/repository"
)

// Custom errors for auth service
var (
	ErrUserNotFound         = errors.New("user not found")
	ErrUserInactive         = errors.New("user is inactive")
	ErrOrganizationNotFound = errors.New("organization not found")
	ErrOrganizationInactive = errors.New("organization is inactive")
	ErrInvalidSecureLink    = errors.New("invalid or expired secure link")
	ErrRateLimitExceeded    = errors.New("rate limit exceeded for magic links")
	ErrInvalidRefreshToken  = errors.New("invalid refresh token")
)

// AuthService handles authentication logic
// #INTEGRATION_POINT: Used by auth handler for login/logout flows
type AuthService interface {
	// RequestMagicLink sends a magic link to the user's email
	RequestMagicLink(ctx context.Context, email string) error

	// VerifyMagicLink validates a magic link and returns token pair
	VerifyMagicLink(ctx context.Context, identifier string) (*auth.TokenPair, *models.User, *models.Organization, error)

	// RefreshAccessToken refreshes an access token using a refresh token
	RefreshAccessToken(ctx context.Context, refreshToken string) (*auth.TokenPair, error)

	// InvalidateRefreshToken invalidates a refresh token (logout)
	InvalidateRefreshToken(ctx context.Context, userID primitive.ObjectID) error

	// GetUserContext retrieves user context from token claims
	GetUserContext(ctx context.Context, userID primitive.ObjectID) (*models.User, *models.Organization, error)
}

// MailService interface for sending emails
// #INTEGRATION_POINT: External mail service integration
type MailService interface {
	SendMagicLink(ctx context.Context, email, name, magicLink string) error
	SendInvitation(ctx context.Context, email, companyName, magicLink string) error
}

// authService implements AuthService
type authService struct {
	userRepo       repository.UserRepository
	orgRepo        repository.OrganizationRepository
	secureLinkRepo repository.SecureLinkRepository
	jwtService     auth.JWTService
	mailService    MailService
	magicLinkBase  string
	rateLimitCount int
	rateLimitMins  int
}

// AuthServiceConfig holds configuration for the auth service
type AuthServiceConfig struct {
	MagicLinkBaseURL    string
	RateLimitCount      int
	RateLimitWindowMins int
}

// NewAuthService creates a new auth service instance
// #IMPLEMENTATION_DECISION: Constructor injection for testability
func NewAuthService(
	userRepo repository.UserRepository,
	orgRepo repository.OrganizationRepository,
	secureLinkRepo repository.SecureLinkRepository,
	jwtService auth.JWTService,
	mailService MailService,
	cfg AuthServiceConfig,
) AuthService {
	return &authService{
		userRepo:       userRepo,
		orgRepo:        orgRepo,
		secureLinkRepo: secureLinkRepo,
		jwtService:     jwtService,
		mailService:    mailService,
		magicLinkBase:  cfg.MagicLinkBaseURL,
		rateLimitCount: cfg.RateLimitCount,
		rateLimitMins:  cfg.RateLimitWindowMins,
	}
}

// RequestMagicLink sends a magic link to the user's email
// #IMPLEMENTATION_DECISION: Rate limit of 5 requests per 15 minutes per email
// #SECURITY_CONCERN: Always return success even for non-existent emails to prevent enumeration
func (s *authService) RequestMagicLink(ctx context.Context, email string) error {
	// Check rate limit
	count, err := s.secureLinkRepo.CountRecentByEmail(ctx, email, s.rateLimitMins)
	if err != nil {
		return fmt.Errorf("failed to check rate limit: %w", err)
	}
	if count >= int64(s.rateLimitCount) {
		return ErrRateLimitExceeded
	}

	// Find user by email
	user, err := s.userRepo.GetByEmail(ctx, email)
	if err != nil || user == nil {
		// #SECURITY_CONCERN: Don't reveal if user exists - return success silently
		return nil //nolint:nilerr // Security: intentional to prevent user enumeration
	}

	if !user.IsActive || user.IsDeleted() {
		// #SECURITY_CONCERN: Don't reveal user status
		return nil
	}

	// Get organization
	org, err := s.orgRepo.GetByID(ctx, user.OrganizationID)
	if err != nil || org == nil || org.IsDeleted() {
		return nil //nolint:nilerr // Security: intentional to prevent org enumeration
	}

	// Invalidate existing links for this email
	if invalidateErr := s.secureLinkRepo.InvalidateAllForEmail(ctx, email); invalidateErr != nil {
		return fmt.Errorf("failed to invalidate existing links: %w", invalidateErr)
	}

	// Generate secure identifier
	identifier, err := generateSecureIdentifier()
	if err != nil {
		return fmt.Errorf("failed to generate secure identifier: %w", err)
	}

	// Create secure link
	link := &models.SecureLink{
		SecureIdentifier: identifier,
		Type:             models.SecureLinkTypeAuth,
		Email:            email,
		UserID:           &user.ID,
	}
	link.BeforeCreate()

	if err := s.secureLinkRepo.Create(ctx, link); err != nil {
		return fmt.Errorf("failed to create secure link: %w", err)
	}

	// Build magic link URL (path parameter to match frontend route /auth/verify/:token)
	magicLinkURL := fmt.Sprintf("%s/auth/verify/%s", s.magicLinkBase, identifier)

	// Send email
	if err := s.mailService.SendMagicLink(ctx, email, user.Name, magicLinkURL); err != nil {
		// #TECHNICAL_DEBT: Should handle email send failures with retry queue
		return fmt.Errorf("failed to send magic link email: %w", err)
	}

	return nil
}

// VerifyMagicLink validates a magic link and returns tokens
// #IMPLEMENTATION_DECISION: Single-use links - marked as used immediately
func (s *authService) VerifyMagicLink(ctx context.Context, identifier string) (*auth.TokenPair, *models.User, *models.Organization, error) {
	// Find secure link
	link, err := s.secureLinkRepo.GetByIdentifier(ctx, identifier)
	if err != nil {
		return nil, nil, nil, ErrInvalidSecureLink
	}

	// Validate link
	if link == nil || !link.CanBeUsed() {
		return nil, nil, nil, ErrInvalidSecureLink
	}

	// Mark as used immediately to prevent race conditions
	if markErr := s.secureLinkRepo.MarkAsUsed(ctx, link.ID); markErr != nil {
		return nil, nil, nil, fmt.Errorf("failed to mark link as used: %w", markErr)
	}

	// Get user
	if link.UserID == nil {
		return nil, nil, nil, ErrUserNotFound
	}

	user, err := s.userRepo.GetByID(ctx, *link.UserID)
	if err != nil || user == nil {
		return nil, nil, nil, ErrUserNotFound
	}

	if !user.IsActive || user.IsDeleted() {
		return nil, nil, nil, ErrUserInactive
	}

	// Get organization
	org, err := s.orgRepo.GetByID(ctx, user.OrganizationID)
	if err != nil || org == nil {
		return nil, nil, nil, ErrOrganizationNotFound
	}

	if org.IsDeleted() {
		return nil, nil, nil, ErrOrganizationInactive
	}

	// Update last login
	if updateErr := s.userRepo.UpdateLastLogin(ctx, user.ID); updateErr != nil { //nolint:staticcheck // Log error but don't fail login
		// #TECHNICAL_DEBT: Log error but don't fail login
	}

	// Generate token pair
	tokenPair, err := s.jwtService.GenerateTokenPair(
		user.ID.Hex(),
		org.ID.Hex(),
		string(user.Role),
		string(org.Type),
	)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	return tokenPair, user, org, nil
}

// RefreshAccessToken refreshes an access token
// #SECURITY_CONCERN: Refresh tokens should ideally be stored and tracked for rotation
func (s *authService) RefreshAccessToken(ctx context.Context, refreshToken string) (*auth.TokenPair, error) {
	claims, err := s.jwtService.ValidateRefreshToken(refreshToken)
	if err != nil {
		if errors.Is(err, auth.ErrTokenExpired) {
			return nil, ErrInvalidRefreshToken
		}
		return nil, ErrInvalidRefreshToken
	}

	// Parse user ID
	userID, err := primitive.ObjectIDFromHex(claims.UserID)
	if err != nil {
		return nil, ErrInvalidRefreshToken
	}

	// Get user
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil || user == nil {
		return nil, ErrUserNotFound
	}

	if !user.IsActive || user.IsDeleted() {
		return nil, ErrUserInactive
	}

	// Get organization
	org, err := s.orgRepo.GetByID(ctx, user.OrganizationID)
	if err != nil || org == nil || org.IsDeleted() {
		return nil, ErrOrganizationNotFound
	}

	// Generate new token pair
	tokenPair, err := s.jwtService.GenerateTokenPair(
		user.ID.Hex(),
		org.ID.Hex(),
		string(user.Role),
		string(org.Type),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to generate tokens: %w", err)
	}

	return tokenPair, nil
}

// InvalidateRefreshToken invalidates all tokens for a user (logout)
// #IMPLEMENTATION_DECISION: Stateless JWT - no server-side invalidation currently
// #SECURITY_CONCERN: For proper logout, would need token blacklist or short expiry
func (s *authService) InvalidateRefreshToken(ctx context.Context, userID primitive.ObjectID) error {
	// #TECHNICAL_DEBT: Implement token blacklist for proper server-side invalidation
	// For now, client should discard tokens and frontend handles session cleanup
	return nil
}

// GetUserContext retrieves full user context
func (s *authService) GetUserContext(ctx context.Context, userID primitive.ObjectID) (*models.User, *models.Organization, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil || user == nil {
		return nil, nil, ErrUserNotFound
	}

	org, err := s.orgRepo.GetByID(ctx, user.OrganizationID)
	if err != nil || org == nil {
		return nil, nil, ErrOrganizationNotFound
	}

	return user, org, nil
}

// generateSecureIdentifier generates a cryptographically secure random identifier
// #IMPLEMENTATION_DECISION: 32 bytes = 64 hex characters
func generateSecureIdentifier() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// NOTE: MailService implementations (HTTPMailService, MockMailService) are in mail_service.go
