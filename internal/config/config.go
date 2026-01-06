// Package config provides configuration loading from environment variables.
// #IMPLEMENTATION_DECISION: Using envconfig for type-safe environment variable parsing
// #CODE_ASSUMPTION: All secrets provided via environment variables (no secret manager integration)
package config

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/kelseyhightower/envconfig"
)

// Config holds all application configuration loaded from environment variables.
// #INTEGRATION_POINT: All services depend on this configuration
type Config struct {
	// Database configuration
	DatabaseURI  string `envconfig:"DATABASE_URI" required:"true"`
	DatabaseName string `envconfig:"DATABASE_NAME" default:"nisfix"`

	// JWT configuration
	JWTPrivateKeyPath  string        `envconfig:"JWT_PRIVATE_KEY_PATH" required:"true"`
	JWTPublicKeyPath   string        `envconfig:"JWT_PUBLIC_KEY_PATH" required:"true"`
	AccessTokenExpiry  time.Duration `envconfig:"ACCESS_TOKEN_EXPIRY" default:"1h"`
	RefreshTokenExpiry time.Duration `envconfig:"REFRESH_TOKEN_EXPIRY" default:"720h"` // 30 days

	// Mail service configuration
	MailServiceURL string `envconfig:"MAIL_SERVICE_URL" required:"true"`
	MailAPIKey     string `envconfig:"MAIL_API_KEY" required:"true"`

	// CheckFix API configuration
	CheckFixAPIURL string `envconfig:"CHECKFIX_API_URL"`
	CheckFixAPIKey string `envconfig:"CHECKFIX_API_KEY"`

	// Server configuration
	ServerPort  string `envconfig:"SERVER_PORT" default:"8080"`
	Environment string `envconfig:"ENVIRONMENT" default:"development"`

	// Magic link configuration
	MagicLinkBaseURL string        `envconfig:"MAGIC_LINK_BASE_URL" required:"true"`
	MagicLinkExpiry  time.Duration `envconfig:"MAGIC_LINK_EXPIRY" default:"15m"`
	InvitationExpiry time.Duration `envconfig:"INVITATION_EXPIRY" default:"168h"` // 7 days

	// CORS configuration
	AllowedOrigins []string `envconfig:"ALLOWED_ORIGINS" default:"http://localhost:3000"`

	// Rate limiting
	RateLimitRequests int           `envconfig:"RATE_LIMIT_REQUESTS" default:"100"`
	RateLimitWindow   time.Duration `envconfig:"RATE_LIMIT_WINDOW" default:"1m"`
}

var (
	instance *Config
	once     sync.Once
	errInit  error
)

// Load loads configuration from environment variables.
// #IMPLEMENTATION_DECISION: Singleton pattern ensures config is loaded once
func Load() (*Config, error) {
	once.Do(func() {
		instance = &Config{}
		errInit = envconfig.Process("NISFIX", instance)
		if errInit != nil {
			return
		}

		// Validate required file paths exist
		if _, err := os.Stat(instance.JWTPrivateKeyPath); os.IsNotExist(err) {
			errInit = fmt.Errorf("JWT private key file not found: %s", instance.JWTPrivateKeyPath)
			return
		}
		if _, err := os.Stat(instance.JWTPublicKeyPath); os.IsNotExist(err) {
			errInit = fmt.Errorf("JWT public key file not found: %s", instance.JWTPublicKeyPath)
			return
		}
	})

	return instance, errInit
}

// GetConfig returns the loaded configuration.
// Panics if configuration has not been loaded.
func GetConfig() *Config {
	if instance == nil {
		panic("config: Load() must be called before GetConfig()")
	}
	return instance
}

// IsDevelopment returns true if running in development mode
func (c *Config) IsDevelopment() bool {
	return c.Environment == "development"
}

// IsProduction returns true if running in production mode
func (c *Config) IsProduction() bool {
	return c.Environment == "production"
}
