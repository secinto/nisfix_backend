// Package auth provides JWT RS512 authentication services.
// #IMPLEMENTATION_DECISION: RS512 chosen for asymmetric signing - allows public key distribution
// #SECURITY_ASSUMPTION: Private key stored securely on server filesystem with 0600 permissions
package auth

import (
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// Custom errors
var (
	ErrInvalidToken     = errors.New("invalid token")
	ErrTokenExpired     = errors.New("token has expired")
	ErrInvalidClaims    = errors.New("invalid token claims")
	ErrKeyNotFound      = errors.New("key file not found")
	ErrInvalidKeyFormat = errors.New("invalid key format")
)

// Claims represents the JWT claims for access tokens
// #INTEGRATION_POINT: Frontend uses these claims for authorization
type Claims struct {
	jwt.RegisteredClaims
	UserID  string `json:"user_id"`
	OrgID   string `json:"org_id"`
	Role    string `json:"role"`
	OrgType string `json:"org_type"`
}

// RefreshClaims represents the JWT claims for refresh tokens
type RefreshClaims struct {
	jwt.RegisteredClaims
	UserID    string `json:"user_id"`
	TokenType string `json:"type"`
}

// TokenPair represents an access and refresh token pair
type TokenPair struct {
	AccessToken  string    `json:"access_token"`
	RefreshToken string    `json:"refresh_token"`
	ExpiresAt    time.Time `json:"expires_at"`
	ExpiresIn    int64     `json:"expires_in"`
}

// JWTService handles JWT token generation and validation
// #IMPLEMENTATION_DECISION: Service interface for testability
type JWTService interface {
	GenerateAccessToken(userID, orgID, role, orgType string) (string, time.Time, error)
	GenerateRefreshToken(userID string) (string, error)
	GenerateTokenPair(userID, orgID, role, orgType string) (*TokenPair, error)
	ValidateAccessToken(tokenString string) (*Claims, error)
	ValidateRefreshToken(tokenString string) (*RefreshClaims, error)
}

// jwtService implements JWTService
type jwtService struct {
	privateKey         *rsa.PrivateKey
	publicKey          *rsa.PublicKey
	accessTokenExpiry  time.Duration
	refreshTokenExpiry time.Duration
	issuer             string
}

// JWTConfig holds JWT service configuration
type JWTConfig struct {
	PrivateKeyPath     string
	PublicKeyPath      string
	AccessTokenExpiry  time.Duration
	RefreshTokenExpiry time.Duration
	Issuer             string
}

// NewJWTService creates a new JWT service instance
// #LIBRARY_CHOICE: golang-jwt/jwt/v5 - well-maintained, supports RS512
func NewJWTService(cfg JWTConfig) (JWTService, error) {
	privateKey, err := loadPrivateKey(cfg.PrivateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load private key: %w", err)
	}

	publicKey, err := loadPublicKey(cfg.PublicKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load public key: %w", err)
	}

	return &jwtService{
		privateKey:         privateKey,
		publicKey:          publicKey,
		accessTokenExpiry:  cfg.AccessTokenExpiry,
		refreshTokenExpiry: cfg.RefreshTokenExpiry,
		issuer:             cfg.Issuer,
	}, nil
}

// GenerateAccessToken creates a new access token
// #IMPLEMENTATION_DECISION: 1-hour expiry for access tokens
func (s *jwtService) GenerateAccessToken(userID, orgID, role, orgType string) (string, time.Time, error) {
	now := time.Now()
	expiresAt := now.Add(s.accessTokenExpiry)

	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.issuer,
			Subject:   userID,
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
		},
		UserID:  userID,
		OrgID:   orgID,
		Role:    role,
		OrgType: orgType,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS512, claims)
	tokenString, err := token.SignedString(s.privateKey)
	if err != nil {
		return "", time.Time{}, fmt.Errorf("failed to sign access token: %w", err)
	}

	return tokenString, expiresAt, nil
}

// GenerateRefreshToken creates a new refresh token
// #IMPLEMENTATION_DECISION: 30-day expiry for refresh tokens
// #SECURITY_CONCERN: Refresh tokens are single-use and should be rotated
func (s *jwtService) GenerateRefreshToken(userID string) (string, error) {
	now := time.Now()
	expiresAt := now.Add(s.refreshTokenExpiry)

	claims := RefreshClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    s.issuer,
			Subject:   userID,
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
		},
		UserID:    userID,
		TokenType: "refresh",
	}

	token := jwt.NewWithClaims(jwt.SigningMethodRS512, claims)
	tokenString, err := token.SignedString(s.privateKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign refresh token: %w", err)
	}

	return tokenString, nil
}

// GenerateTokenPair creates both access and refresh tokens
func (s *jwtService) GenerateTokenPair(userID, orgID, role, orgType string) (*TokenPair, error) {
	accessToken, expiresAt, err := s.GenerateAccessToken(userID, orgID, role, orgType)
	if err != nil {
		return nil, err
	}

	refreshToken, err := s.GenerateRefreshToken(userID)
	if err != nil {
		return nil, err
	}

	return &TokenPair{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    expiresAt,
		ExpiresIn:    int64(s.accessTokenExpiry.Seconds()),
	}, nil
}

// ValidateAccessToken validates an access token and returns the claims
func (s *jwtService) ValidateAccessToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.publicKey, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, fmt.Errorf("%w: %w", ErrInvalidToken, err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, ErrInvalidClaims
	}

	return claims, nil
}

// ValidateRefreshToken validates a refresh token and returns the claims
func (s *jwtService) ValidateRefreshToken(tokenString string) (*RefreshClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &RefreshClaims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodRSA); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return s.publicKey, nil
	})

	if err != nil {
		if errors.Is(err, jwt.ErrTokenExpired) {
			return nil, ErrTokenExpired
		}
		return nil, fmt.Errorf("%w: %w", ErrInvalidToken, err)
	}

	claims, ok := token.Claims.(*RefreshClaims)
	if !ok || !token.Valid {
		return nil, ErrInvalidClaims
	}

	if claims.TokenType != "refresh" {
		return nil, ErrInvalidClaims
	}

	return claims, nil
}

// loadPrivateKey loads an RSA private key from a PEM file
func loadPrivateKey(path string) (*rsa.PrivateKey, error) {
	keyData, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrKeyNotFound
		}
		return nil, err
	}

	block, _ := pem.Decode(keyData)
	if block == nil {
		return nil, ErrInvalidKeyFormat
	}

	// Try PKCS#1 format first
	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err == nil {
		return privateKey, nil
	}

	// Try PKCS#8 format
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidKeyFormat, err)
	}

	rsaKey, ok := key.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("%w: not an RSA key", ErrInvalidKeyFormat)
	}

	return rsaKey, nil
}

// loadPublicKey loads an RSA public key from a PEM file
func loadPublicKey(path string) (*rsa.PublicKey, error) {
	keyData, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, ErrKeyNotFound
		}
		return nil, err
	}

	block, _ := pem.Decode(keyData)
	if block == nil {
		return nil, ErrInvalidKeyFormat
	}

	// Try PKIX format first
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err == nil {
		rsaKey, ok := pub.(*rsa.PublicKey)
		if !ok {
			return nil, fmt.Errorf("%w: not an RSA key", ErrInvalidKeyFormat)
		}
		return rsaKey, nil
	}

	// Try PKCS#1 format
	rsaKey, err := x509.ParsePKCS1PublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrInvalidKeyFormat, err)
	}

	return rsaKey, nil
}
