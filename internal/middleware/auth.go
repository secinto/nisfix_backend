// Package middleware provides HTTP middleware for Gin framework.
// #IMPLEMENTATION_DECISION: Middleware chain for authentication, authorization, and logging
package middleware

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/checkfix-tools/nisfix_backend/internal/auth"
	"github.com/checkfix-tools/nisfix_backend/internal/models"
)

// Context keys for storing authenticated user data
// #INTEGRATION_POINT: Handlers extract user data using these keys
const (
	ContextKeyUserID  = "user_id"
	ContextKeyOrgID   = "org_id"
	ContextKeyRole    = "role"
	ContextKeyOrgType = "org_type"
	ContextKeyClaims  = "claims"
)

// Custom errors
var (
	ErrAuthHeaderMissing = errors.New("authorization header is required")
	ErrAuthHeaderFormat  = errors.New("authorization header format must be Bearer {token}")
	ErrInvalidToken      = errors.New("invalid or expired token")
	ErrForbidden         = errors.New("access denied")
)

// AuthMiddleware validates JWT tokens and extracts user claims
// #IMPLEMENTATION_DECISION: Bearer token authentication
func AuthMiddleware(jwtService auth.JWTService) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": ErrAuthHeaderMissing.Error(),
			})
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": ErrAuthHeaderFormat.Error(),
			})
			c.Abort()
			return
		}

		tokenString := parts[1]
		claims, err := jwtService.ValidateAccessToken(tokenString)
		if err != nil {
			statusCode := http.StatusUnauthorized
			message := ErrInvalidToken.Error()

			if errors.Is(err, auth.ErrTokenExpired) {
				message = "token has expired"
			}

			c.JSON(statusCode, gin.H{
				"error":   "unauthorized",
				"message": message,
			})
			c.Abort()
			return
		}

		// Store claims in context for downstream handlers
		c.Set(ContextKeyClaims, claims)
		c.Set(ContextKeyUserID, claims.UserID)
		c.Set(ContextKeyOrgID, claims.OrgID)
		c.Set(ContextKeyRole, claims.Role)
		c.Set(ContextKeyOrgType, claims.OrgType)

		c.Next()
	}
}

// OptionalAuthMiddleware extracts user claims if present but doesn't require authentication
// #IMPLEMENTATION_DECISION: For endpoints that behave differently for authenticated users
func OptionalAuthMiddleware(jwtService auth.JWTService) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.Next()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "bearer") {
			c.Next()
			return
		}

		tokenString := parts[1]
		claims, err := jwtService.ValidateAccessToken(tokenString)
		if err == nil {
			c.Set(ContextKeyClaims, claims)
			c.Set(ContextKeyUserID, claims.UserID)
			c.Set(ContextKeyOrgID, claims.OrgID)
			c.Set(ContextKeyRole, claims.Role)
			c.Set(ContextKeyOrgType, claims.OrgType)
		}

		c.Next()
	}
}

// RequireRole middleware checks if the user has one of the required roles
// #IMPLEMENTATION_DECISION: Role-based access control
func RequireRole(allowedRoles ...models.UserRole) gin.HandlerFunc {
	return func(c *gin.Context) {
		roleVal, exists := c.Get(ContextKeyRole)
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{
				"error":   "forbidden",
				"message": ErrForbidden.Error(),
			})
			c.Abort()
			return
		}

		roleStr, ok := roleVal.(string)
		if !ok {
			c.JSON(http.StatusForbidden, gin.H{
				"error":   "forbidden",
				"message": ErrForbidden.Error(),
			})
			c.Abort()
			return
		}
		userRole := models.UserRole(strings.ToUpper(roleStr))
		for _, allowed := range allowedRoles {
			if userRole == allowed {
				c.Next()
				return
			}
		}

		c.JSON(http.StatusForbidden, gin.H{
			"error":   "forbidden",
			"message": "insufficient role permissions",
		})
		c.Abort()
	}
}

// RequireOrgType middleware checks if the user belongs to an organization of the required type
// #IMPLEMENTATION_DECISION: Organization type guards for company/supplier specific endpoints
func RequireOrgType(allowedTypes ...models.OrganizationType) gin.HandlerFunc {
	return func(c *gin.Context) {
		orgTypeVal, exists := c.Get(ContextKeyOrgType)
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{
				"error":   "forbidden",
				"message": ErrForbidden.Error(),
			})
			c.Abort()
			return
		}

		orgTypeStr, ok := orgTypeVal.(string)
		if !ok {
			c.JSON(http.StatusForbidden, gin.H{
				"error":   "forbidden",
				"message": ErrForbidden.Error(),
			})
			c.Abort()
			return
		}
		orgType := models.OrganizationType(strings.ToUpper(orgTypeStr))
		for _, allowed := range allowedTypes {
			if orgType == allowed {
				c.Next()
				return
			}
		}

		c.JSON(http.StatusForbidden, gin.H{
			"error":   "forbidden",
			"message": "this endpoint is not available for your organization type",
		})
		c.Abort()
	}
}

// RequireAdmin is a shorthand for requiring admin role
func RequireAdmin() gin.HandlerFunc {
	return RequireRole(models.UserRoleAdmin)
}

// RequireCompany is a shorthand for requiring company organization type
func RequireCompany() gin.HandlerFunc {
	return RequireOrgType(models.OrganizationTypeCompany)
}

// RequireSupplier is a shorthand for requiring supplier organization type
func RequireSupplier() gin.HandlerFunc {
	return RequireOrgType(models.OrganizationTypeSupplier)
}

// Helper functions for extracting values from context

// GetUserID extracts the user ID from context
func GetUserID(c *gin.Context) (primitive.ObjectID, bool) {
	userIDVal, exists := c.Get(ContextKeyUserID)
	if !exists {
		return primitive.NilObjectID, false
	}

	userIDStr, ok := userIDVal.(string)
	if !ok {
		return primitive.NilObjectID, false
	}

	userID, err := primitive.ObjectIDFromHex(userIDStr)
	if err != nil {
		return primitive.NilObjectID, false
	}

	return userID, true
}

// GetOrgID extracts the organization ID from context
func GetOrgID(c *gin.Context) (primitive.ObjectID, bool) {
	orgIDVal, exists := c.Get(ContextKeyOrgID)
	if !exists {
		return primitive.NilObjectID, false
	}

	orgIDStr, ok := orgIDVal.(string)
	if !ok {
		return primitive.NilObjectID, false
	}

	orgID, err := primitive.ObjectIDFromHex(orgIDStr)
	if err != nil {
		return primitive.NilObjectID, false
	}

	return orgID, true
}

// GetRole extracts the user role from context
func GetRole(c *gin.Context) (models.UserRole, bool) {
	roleVal, exists := c.Get(ContextKeyRole)
	if !exists {
		return "", false
	}

	roleStr, ok := roleVal.(string)
	if !ok {
		return "", false
	}

	return models.UserRole(strings.ToUpper(roleStr)), true
}

// GetOrgType extracts the organization type from context
func GetOrgType(c *gin.Context) (models.OrganizationType, bool) {
	orgTypeVal, exists := c.Get(ContextKeyOrgType)
	if !exists {
		return "", false
	}

	orgTypeStr, ok := orgTypeVal.(string)
	if !ok {
		return "", false
	}

	return models.OrganizationType(strings.ToUpper(orgTypeStr)), true
}

// GetClaims extracts the full JWT claims from context
func GetClaims(c *gin.Context) (*auth.Claims, bool) {
	claimsVal, exists := c.Get(ContextKeyClaims)
	if !exists {
		return nil, false
	}

	claims, ok := claimsVal.(*auth.Claims)
	if !ok {
		return nil, false
	}

	return claims, true
}

// IsAdmin checks if the current user is an admin
func IsAdmin(c *gin.Context) bool {
	role, exists := GetRole(c)
	return exists && role == models.UserRoleAdmin
}

// IsCompanyUser checks if the current user belongs to a company
func IsCompanyUser(c *gin.Context) bool {
	orgType, exists := GetOrgType(c)
	return exists && orgType == models.OrganizationTypeCompany
}

// IsSupplierUser checks if the current user belongs to a supplier
func IsSupplierUser(c *gin.Context) bool {
	orgType, exists := GetOrgType(c)
	return exists && orgType == models.OrganizationTypeSupplier
}
