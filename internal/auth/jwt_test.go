package auth

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// testKeyPair holds the paths to test keys
type testKeyPair struct {
	privateKeyPath string
	publicKeyPath  string
	cleanup        func()
}

// generateTestKeys creates temporary RSA key files for testing
func generateTestKeys(t *testing.T) *testKeyPair {
	t.Helper()

	// Generate RSA key pair
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("Failed to generate RSA key: %v", err)
	}

	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "jwt_test_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	// Write private key
	privateKeyPath := filepath.Join(tmpDir, "private.pem")
	privateKeyBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	privatePEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privateKeyBytes,
	})
	if writeErr := os.WriteFile(privateKeyPath, privatePEM, 0o600); writeErr != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to write private key: %v", writeErr)
	}

	// Write public key
	publicKeyPath := filepath.Join(tmpDir, "public.pem")
	publicKeyBytes, err := x509.MarshalPKIXPublicKey(&privateKey.PublicKey)
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to marshal public key: %v", err)
	}
	publicPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicKeyBytes,
	})
	if err := os.WriteFile(publicKeyPath, publicPEM, 0o644); err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to write public key: %v", err)
	}

	return &testKeyPair{
		privateKeyPath: privateKeyPath,
		publicKeyPath:  publicKeyPath,
		cleanup: func() {
			os.RemoveAll(tmpDir)
		},
	}
}

func createTestJWTService(t *testing.T) (JWTService, func()) {
	t.Helper()

	keys := generateTestKeys(t)
	cfg := JWTConfig{
		PrivateKeyPath:     keys.privateKeyPath,
		PublicKeyPath:      keys.publicKeyPath,
		AccessTokenExpiry:  1 * time.Hour,
		RefreshTokenExpiry: 24 * time.Hour * 30,
		Issuer:             "test-issuer",
	}

	svc, err := NewJWTService(cfg)
	if err != nil {
		keys.cleanup()
		t.Fatalf("Failed to create JWT service: %v", err)
	}

	return svc, keys.cleanup
}

func TestNewJWTService(t *testing.T) {
	keys := generateTestKeys(t)
	defer keys.cleanup()

	tests := []struct {
		name        string
		cfg         JWTConfig
		expectError bool
	}{
		{
			name: "Valid config",
			cfg: JWTConfig{
				PrivateKeyPath:     keys.privateKeyPath,
				PublicKeyPath:      keys.publicKeyPath,
				AccessTokenExpiry:  1 * time.Hour,
				RefreshTokenExpiry: 24 * time.Hour,
				Issuer:             "test",
			},
			expectError: false,
		},
		{
			name: "Missing private key",
			cfg: JWTConfig{
				PrivateKeyPath:     "/nonexistent/private.pem",
				PublicKeyPath:      keys.publicKeyPath,
				AccessTokenExpiry:  1 * time.Hour,
				RefreshTokenExpiry: 24 * time.Hour,
				Issuer:             "test",
			},
			expectError: true,
		},
		{
			name: "Missing public key",
			cfg: JWTConfig{
				PrivateKeyPath:     keys.privateKeyPath,
				PublicKeyPath:      "/nonexistent/public.pem",
				AccessTokenExpiry:  1 * time.Hour,
				RefreshTokenExpiry: 24 * time.Hour,
				Issuer:             "test",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewJWTService(tt.cfg)
			if (err != nil) != tt.expectError {
				t.Errorf("NewJWTService() error = %v, expectError %v", err, tt.expectError)
			}
		})
	}
}

func TestJWTService_GenerateAccessToken(t *testing.T) {
	svc, cleanup := createTestJWTService(t)
	defer cleanup()

	userID := "user123"
	orgID := "org456"
	role := "ADMIN"
	orgType := "COMPANY"

	token, expiresAt, err := svc.GenerateAccessToken(userID, orgID, role, orgType)
	if err != nil {
		t.Fatalf("GenerateAccessToken() error = %v", err)
	}

	if token == "" {
		t.Error("GenerateAccessToken() returned empty token")
	}

	if expiresAt.Before(time.Now()) {
		t.Error("GenerateAccessToken() returned past expiration time")
	}

	// Verify token is valid
	claims, err := svc.ValidateAccessToken(token)
	if err != nil {
		t.Fatalf("ValidateAccessToken() error = %v", err)
	}

	if claims.UserID != userID {
		t.Errorf("Claims.UserID = %v, want %v", claims.UserID, userID)
	}
	if claims.OrgID != orgID {
		t.Errorf("Claims.OrgID = %v, want %v", claims.OrgID, orgID)
	}
	if claims.Role != role {
		t.Errorf("Claims.Role = %v, want %v", claims.Role, role)
	}
	if claims.OrgType != orgType {
		t.Errorf("Claims.OrgType = %v, want %v", claims.OrgType, orgType)
	}
}

func TestJWTService_GenerateRefreshToken(t *testing.T) {
	svc, cleanup := createTestJWTService(t)
	defer cleanup()

	userID := "user123"

	token, err := svc.GenerateRefreshToken(userID)
	if err != nil {
		t.Fatalf("GenerateRefreshToken() error = %v", err)
	}

	if token == "" {
		t.Error("GenerateRefreshToken() returned empty token")
	}

	// Verify token is valid
	claims, err := svc.ValidateRefreshToken(token)
	if err != nil {
		t.Fatalf("ValidateRefreshToken() error = %v", err)
	}

	if claims.UserID != userID {
		t.Errorf("Claims.UserID = %v, want %v", claims.UserID, userID)
	}
	if claims.TokenType != "refresh" {
		t.Errorf("Claims.TokenType = %v, want refresh", claims.TokenType)
	}
}

func TestJWTService_GenerateTokenPair(t *testing.T) {
	svc, cleanup := createTestJWTService(t)
	defer cleanup()

	userID := "user123"
	orgID := "org456"
	role := "ADMIN"
	orgType := "COMPANY"

	pair, err := svc.GenerateTokenPair(userID, orgID, role, orgType)
	if err != nil {
		t.Fatalf("GenerateTokenPair() error = %v", err)
	}

	if pair.AccessToken == "" {
		t.Error("TokenPair.AccessToken is empty")
	}
	if pair.RefreshToken == "" {
		t.Error("TokenPair.RefreshToken is empty")
	}
	if pair.ExpiresAt.Before(time.Now()) {
		t.Error("TokenPair.ExpiresAt is in the past")
	}
	if pair.ExpiresIn <= 0 {
		t.Error("TokenPair.ExpiresIn should be positive")
	}

	// Verify both tokens work
	_, err = svc.ValidateAccessToken(pair.AccessToken)
	if err != nil {
		t.Errorf("AccessToken validation failed: %v", err)
	}

	_, err = svc.ValidateRefreshToken(pair.RefreshToken)
	if err != nil {
		t.Errorf("RefreshToken validation failed: %v", err)
	}
}

func TestJWTService_ValidateAccessToken_Invalid(t *testing.T) {
	svc, cleanup := createTestJWTService(t)
	defer cleanup()

	tests := []struct {
		name  string
		token string
	}{
		{"Empty token", ""},
		{"Malformed token", "not.a.valid.token"},
		{"Invalid signature", "eyJhbGciOiJSUzUxMiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiYWRtaW4iOnRydWUsImlhdCI6MTUxNjIzOTAyMn0.invalid"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.ValidateAccessToken(tt.token)
			if err == nil {
				t.Error("ValidateAccessToken() should return error for invalid token")
			}
		})
	}
}

func TestJWTService_ValidateRefreshToken_Invalid(t *testing.T) {
	svc, cleanup := createTestJWTService(t)
	defer cleanup()

	tests := []struct {
		name  string
		token string
	}{
		{"Empty token", ""},
		{"Malformed token", "not.a.valid.token"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.ValidateRefreshToken(tt.token)
			if err == nil {
				t.Error("ValidateRefreshToken() should return error for invalid token")
			}
		})
	}
}

func TestJWTService_ValidateRefreshToken_WrongType(t *testing.T) {
	svc, cleanup := createTestJWTService(t)
	defer cleanup()

	// Generate an access token and try to use it as a refresh token
	accessToken, _, err := svc.GenerateAccessToken("user123", "org456", "ADMIN", "COMPANY")
	if err != nil {
		t.Fatalf("GenerateAccessToken() error = %v", err)
	}

	_, err = svc.ValidateRefreshToken(accessToken)
	if err == nil {
		t.Error("ValidateRefreshToken() should reject access tokens")
	}
}

func TestLoadPrivateKey_InvalidFormat(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "invalid_key_*.pem")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	// Write invalid PEM data
	if _, writeErr := tmpFile.WriteString("not valid pem data"); writeErr != nil {
		t.Fatalf("Failed to write temp file: %v", writeErr)
	}
	tmpFile.Close()

	_, err = loadPrivateKey(tmpFile.Name())
	if !errors.Is(err, ErrInvalidKeyFormat) {
		t.Errorf("loadPrivateKey() error = %v, want ErrInvalidKeyFormat", err)
	}
}

func TestLoadPublicKey_InvalidFormat(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "invalid_key_*.pem")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())

	// Write invalid PEM data
	if _, writeErr := tmpFile.WriteString("not valid pem data"); writeErr != nil {
		t.Fatalf("Failed to write temp file: %v", writeErr)
	}
	tmpFile.Close()

	_, err = loadPublicKey(tmpFile.Name())
	if !errors.Is(err, ErrInvalidKeyFormat) {
		t.Errorf("loadPublicKey() error = %v, want ErrInvalidKeyFormat", err)
	}
}

func TestLoadPrivateKey_NotFound(t *testing.T) {
	_, err := loadPrivateKey("/nonexistent/path/to/key.pem")
	if !errors.Is(err, ErrKeyNotFound) {
		t.Errorf("loadPrivateKey() error = %v, want ErrKeyNotFound", err)
	}
}

func TestLoadPublicKey_NotFound(t *testing.T) {
	_, err := loadPublicKey("/nonexistent/path/to/key.pem")
	if !errors.Is(err, ErrKeyNotFound) {
		t.Errorf("loadPublicKey() error = %v, want ErrKeyNotFound", err)
	}
}
