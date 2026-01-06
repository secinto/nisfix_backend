package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/checkfix-tools/nisfix_backend/internal/auth"
	"github.com/checkfix-tools/nisfix_backend/internal/models"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func init() {
	gin.SetMode(gin.TestMode)
}

// MockJWTService implements auth.JWTService for testing
type MockJWTService struct {
	ValidToken   string
	ValidClaims  *auth.Claims
	ExpiredError bool
}

func (m *MockJWTService) GenerateAccessToken(userID, orgID, role, orgType string) (string, time.Time, error) {
	return m.ValidToken, time.Now().Add(time.Hour), nil
}

func (m *MockJWTService) GenerateRefreshToken(userID string) (string, error) {
	return "refresh-token", nil
}

func (m *MockJWTService) GenerateTokenPair(userID, orgID, role, orgType string) (*auth.TokenPair, error) {
	return &auth.TokenPair{
		AccessToken:  m.ValidToken,
		RefreshToken: "refresh-token",
		ExpiresAt:    time.Now().Add(time.Hour),
		ExpiresIn:    3600,
	}, nil
}

func (m *MockJWTService) ValidateAccessToken(tokenString string) (*auth.Claims, error) {
	if m.ExpiredError {
		return nil, auth.ErrTokenExpired
	}
	// Empty token is always invalid
	if tokenString == "" {
		return nil, auth.ErrInvalidToken
	}
	if tokenString == m.ValidToken && m.ValidClaims != nil {
		return m.ValidClaims, nil
	}
	return nil, auth.ErrInvalidToken
}

func (m *MockJWTService) ValidateRefreshToken(tokenString string) (*auth.RefreshClaims, error) {
	return nil, nil
}

func TestAuthMiddleware_ValidToken(t *testing.T) {
	mockJWT := &MockJWTService{
		ValidToken: "valid-token",
		ValidClaims: &auth.Claims{
			UserID:  primitive.NewObjectID().Hex(),
			OrgID:   primitive.NewObjectID().Hex(),
			Role:    "ADMIN",
			OrgType: "COMPANY",
		},
	}

	router := gin.New()
	router.Use(AuthMiddleware(mockJWT))
	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestAuthMiddleware_MissingHeader(t *testing.T) {
	mockJWT := &MockJWTService{}

	router := gin.New()
	router.Use(AuthMiddleware(mockJWT))
	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestAuthMiddleware_InvalidFormat(t *testing.T) {
	mockJWT := &MockJWTService{}

	router := gin.New()
	router.Use(AuthMiddleware(mockJWT))
	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	tests := []struct {
		name   string
		header string
	}{
		{"Missing Bearer prefix", "token-only"},
		{"Wrong prefix", "Basic token"},
		{"Empty token", "Bearer "},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("Authorization", tt.header)
			w := httptest.NewRecorder()

			router.ServeHTTP(w, req)

			if w.Code != http.StatusUnauthorized {
				t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, w.Code)
			}
		})
	}
}

func TestAuthMiddleware_InvalidToken(t *testing.T) {
	mockJWT := &MockJWTService{
		ValidToken: "valid-token",
	}

	router := gin.New()
	router.Use(AuthMiddleware(mockJWT))
	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer invalid-token")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestAuthMiddleware_ExpiredToken(t *testing.T) {
	mockJWT := &MockJWTService{
		ExpiredError: true,
	}

	router := gin.New()
	router.Use(AuthMiddleware(mockJWT))
	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer expired-token")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status %d, got %d", http.StatusUnauthorized, w.Code)
	}
}

func TestOptionalAuthMiddleware_WithToken(t *testing.T) {
	mockJWT := &MockJWTService{
		ValidToken: "valid-token",
		ValidClaims: &auth.Claims{
			UserID: primitive.NewObjectID().Hex(),
			OrgID:  primitive.NewObjectID().Hex(),
		},
	}

	var capturedUserID string
	router := gin.New()
	router.Use(OptionalAuthMiddleware(mockJWT))
	router.GET("/test", func(c *gin.Context) {
		if userID, exists := c.Get(ContextKeyUserID); exists {
			capturedUserID = userID.(string)
		}
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer valid-token")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}
	if capturedUserID == "" {
		t.Error("Expected user ID to be set in context")
	}
}

func TestOptionalAuthMiddleware_WithoutToken(t *testing.T) {
	mockJWT := &MockJWTService{}

	var capturedUserID string
	router := gin.New()
	router.Use(OptionalAuthMiddleware(mockJWT))
	router.GET("/test", func(c *gin.Context) {
		if userID, exists := c.Get(ContextKeyUserID); exists {
			capturedUserID = userID.(string)
		}
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}
	if capturedUserID != "" {
		t.Error("Expected user ID to NOT be set in context")
	}
}

func TestRequireRole_Allowed(t *testing.T) {
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(ContextKeyRole, "ADMIN")
		c.Next()
	})
	router.Use(RequireRole(models.UserRoleAdmin))
	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestRequireRole_Denied(t *testing.T) {
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(ContextKeyRole, "VIEWER")
		c.Next()
	})
	router.Use(RequireRole(models.UserRoleAdmin))
	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status %d, got %d", http.StatusForbidden, w.Code)
	}
}

func TestRequireRole_NoRole(t *testing.T) {
	router := gin.New()
	router.Use(RequireRole(models.UserRoleAdmin))
	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status %d, got %d", http.StatusForbidden, w.Code)
	}
}

func TestRequireOrgType_Allowed(t *testing.T) {
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(ContextKeyOrgType, "COMPANY")
		c.Next()
	})
	router.Use(RequireOrgType(models.OrganizationTypeCompany))
	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}
}

func TestRequireOrgType_Denied(t *testing.T) {
	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(ContextKeyOrgType, "SUPPLIER")
		c.Next()
	})
	router.Use(RequireOrgType(models.OrganizationTypeCompany))
	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("Expected status %d, got %d", http.StatusForbidden, w.Code)
	}
}

func TestRequestID_GeneratesNew(t *testing.T) {
	router := gin.New()
	router.Use(RequestID())

	var capturedID string
	router.GET("/test", func(c *gin.Context) {
		capturedID = GetRequestID(c)
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if capturedID == "" {
		t.Error("Expected request ID to be generated")
	}

	// Check header is set
	if w.Header().Get("X-Request-ID") == "" {
		t.Error("Expected X-Request-ID header to be set")
	}
}

func TestRequestID_UsesExisting(t *testing.T) {
	router := gin.New()
	router.Use(RequestID())

	existingID := "existing-request-id"
	var capturedID string
	router.GET("/test", func(c *gin.Context) {
		capturedID = GetRequestID(c)
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Request-ID", existingID)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if capturedID != existingID {
		t.Errorf("Expected request ID %s, got %s", existingID, capturedID)
	}
}

func TestCORS_AllowedOrigin(t *testing.T) {
	router := gin.New()
	router.Use(CORS([]string{"http://localhost:3000"}))
	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Header().Get("Access-Control-Allow-Origin") != "http://localhost:3000" {
		t.Error("Expected CORS header to be set for allowed origin")
	}
}

func TestCORS_Preflight(t *testing.T) {
	router := gin.New()
	router.Use(CORS([]string{"*"}))
	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("OPTIONS", "/test", nil)
	req.Header.Set("Origin", "http://example.com")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("Expected status %d for OPTIONS, got %d", http.StatusNoContent, w.Code)
	}
}

func TestSecureHeaders(t *testing.T) {
	router := gin.New()
	router.Use(SecureHeaders())
	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	expectedHeaders := map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":        "DENY",
		"X-XSS-Protection":       "1; mode=block",
	}

	for header, expected := range expectedHeaders {
		if w.Header().Get(header) != expected {
			t.Errorf("Expected %s to be %s, got %s", header, expected, w.Header().Get(header))
		}
	}
}

func TestRecovery(t *testing.T) {
	router := gin.New()
	router.Use(RequestID())
	router.Use(Recovery())
	router.GET("/test", func(c *gin.Context) {
		panic("test panic")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("Expected status %d, got %d", http.StatusInternalServerError, w.Code)
	}
}

func TestRateLimiter(t *testing.T) {
	limiter := NewRateLimiter(2, time.Minute)

	router := gin.New()
	router.Use(limiter.RateLimit())
	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	// First request - should pass
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.1:1234"
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("First request: expected %d, got %d", http.StatusOK, w.Code)
	}

	// Second request - should pass
	req = httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.1:1234"
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("Second request: expected %d, got %d", http.StatusOK, w.Code)
	}

	// Third request - should be rate limited
	req = httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.1:1234"
	w = httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusTooManyRequests {
		t.Errorf("Third request: expected %d, got %d", http.StatusTooManyRequests, w.Code)
	}
}

func TestGetUserID_Valid(t *testing.T) {
	expectedID := primitive.NewObjectID()

	router := gin.New()
	router.Use(func(c *gin.Context) {
		c.Set(ContextKeyUserID, expectedID.Hex())
		c.Next()
	})

	var capturedID primitive.ObjectID
	var ok bool
	router.GET("/test", func(c *gin.Context) {
		capturedID, ok = GetUserID(c)
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if !ok {
		t.Error("Expected GetUserID to return true")
	}
	if capturedID != expectedID {
		t.Errorf("Expected user ID %s, got %s", expectedID.Hex(), capturedID.Hex())
	}
}

func TestGetUserID_NotSet(t *testing.T) {
	router := gin.New()

	var ok bool
	router.GET("/test", func(c *gin.Context) {
		_, ok = GetUserID(c)
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if ok {
		t.Error("Expected GetUserID to return false when not set")
	}
}

func TestIsAdmin(t *testing.T) {
	tests := []struct {
		name     string
		role     string
		expected bool
	}{
		{"Admin role", "ADMIN", true},
		{"Viewer role", "VIEWER", false},
		{"Empty role", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			if tt.role != "" {
				router.Use(func(c *gin.Context) {
					c.Set(ContextKeyRole, tt.role)
					c.Next()
				})
			}

			var result bool
			router.GET("/test", func(c *gin.Context) {
				result = IsAdmin(c)
				c.Status(http.StatusOK)
			})

			req := httptest.NewRequest("GET", "/test", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if result != tt.expected {
				t.Errorf("IsAdmin() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestIsCompanyUser(t *testing.T) {
	tests := []struct {
		name     string
		orgType  string
		expected bool
	}{
		{"Company type", "COMPANY", true},
		{"Supplier type", "SUPPLIER", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.Use(func(c *gin.Context) {
				c.Set(ContextKeyOrgType, tt.orgType)
				c.Next()
			})

			var result bool
			router.GET("/test", func(c *gin.Context) {
				result = IsCompanyUser(c)
				c.Status(http.StatusOK)
			})

			req := httptest.NewRequest("GET", "/test", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if result != tt.expected {
				t.Errorf("IsCompanyUser() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestBytesString(t *testing.T) {
	tests := []struct {
		size     int
		expected string
	}{
		{-1, "-"},
		{0, "0B"},
		{100, "100B"},
		{1024, "1KB"},
		{2048, "2KB"},
		{1048576, "1MB"},
	}

	for _, tt := range tests {
		result := bytesString(tt.size)
		if result != tt.expected {
			t.Errorf("bytesString(%d) = %s, want %s", tt.size, result, tt.expected)
		}
	}
}
