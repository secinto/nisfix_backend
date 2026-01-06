// Package middleware provides HTTP middleware for Gin framework.
package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ContextKeyRequestID is the context key for request ID
const ContextKeyRequestID = "request_id"

// RequestID adds a unique request ID to each request
// #IMPLEMENTATION_DECISION: UUID v4 for traceability across logs
func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}

		c.Set(ContextKeyRequestID, requestID)
		c.Header("X-Request-ID", requestID)
		c.Next()
	}
}

// GetRequestID extracts the request ID from context
func GetRequestID(c *gin.Context) string {
	if requestIDVal, exists := c.Get(ContextKeyRequestID); exists {
		if requestID, ok := requestIDVal.(string); ok {
			return requestID
		}
	}
	return ""
}

// CORS configures Cross-Origin Resource Sharing
// #IMPLEMENTATION_DECISION: Configurable allowed origins for security
func CORS(allowedOrigins []string) gin.HandlerFunc {
	originsMap := make(map[string]bool)
	for _, origin := range allowedOrigins {
		originsMap[origin] = true
	}

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")

		// Check if origin is allowed
		if originsMap[origin] || (len(allowedOrigins) == 1 && allowedOrigins[0] == "*") {
			c.Header("Access-Control-Allow-Origin", origin)
		}

		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Content-Length, Accept-Encoding, Authorization, X-Request-ID")
		c.Header("Access-Control-Expose-Headers", "Content-Length, X-Request-ID")
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Max-Age", "86400")

		// Handle preflight requests
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// Logger provides structured logging middleware
// #IMPLEMENTATION_DECISION: Log request details for debugging and monitoring
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Start timer
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		// Process request
		c.Next()

		// Calculate latency
		latency := time.Since(start)

		// Get request ID
		requestID := GetRequestID(c)

		// Log format fields
		clientIP := c.ClientIP()
		method := c.Request.Method
		statusCode := c.Writer.Status()
		bodySize := c.Writer.Size()

		if raw != "" {
			path = path + "?" + raw
		}

		// Log entry using Gin's default logger format with additions
		//nolint:errcheck // Logging errors should not crash the request
		gin.DefaultWriter.Write([]byte(
			time.Now().Format("2006/01/02 - 15:04:05") + " | " +
				requestID[:8] + " | " +
				statusString(statusCode) + " | " +
				latency.String() + " | " +
				clientIP + " | " +
				method + " | " +
				path + " | " +
				bytesString(bodySize) + "\n",
		))
	}
}

// statusString returns colored status code
func statusString(code int) string {
	switch {
	case code >= 200 && code < 300:
		return "\033[32m" + string(rune(code/100+48)) + string(rune(code%100/10+48)) + string(rune(code%10+48)) + "\033[0m"
	case code >= 300 && code < 400:
		return "\033[36m" + string(rune(code/100+48)) + string(rune(code%100/10+48)) + string(rune(code%10+48)) + "\033[0m"
	case code >= 400 && code < 500:
		return "\033[33m" + string(rune(code/100+48)) + string(rune(code%100/10+48)) + string(rune(code%10+48)) + "\033[0m"
	default:
		return "\033[31m" + string(rune(code/100+48)) + string(rune(code%100/10+48)) + string(rune(code%10+48)) + "\033[0m"
	}
}

// bytesString formats bytes for display
func bytesString(size int) string {
	if size < 0 {
		return "-"
	}
	if size < 1024 {
		return intToString(size) + "B"
	}
	if size < 1024*1024 {
		return intToString(size/1024) + "KB"
	}
	return intToString(size/(1024*1024)) + "MB"
}

// intToString converts int to string without importing strconv
func intToString(i int) string {
	if i == 0 {
		return "0"
	}
	result := ""
	for i > 0 {
		result = string(rune(i%10+48)) + result
		i /= 10
	}
	return result
}

// Recovery recovers from panics and returns a 500 error
// #IMPLEMENTATION_DECISION: Custom recovery with request ID for debugging
func Recovery() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		requestID := GetRequestID(c)

		c.JSON(500, gin.H{
			"error":      "internal_server_error",
			"message":    "An unexpected error occurred",
			"request_id": requestID,
		})
	})
}

// SecureHeaders adds security-related headers
// #SECURITY_CONCERN: Helps prevent common web attacks
func SecureHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Header("Content-Security-Policy", "default-src 'self'")

		c.Next()
	}
}

// RateLimiter provides basic rate limiting
// #IMPLEMENTATION_DECISION: Simple in-memory rate limiting
// #TECHNICAL_DEBT: Should use Redis for distributed rate limiting
type RateLimiter struct {
	requests map[string][]time.Time
	limit    int
	window   time.Duration
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(limit int, window time.Duration) *RateLimiter {
	return &RateLimiter{
		requests: make(map[string][]time.Time),
		limit:    limit,
		window:   window,
	}
}

// RateLimit middleware function
func (rl *RateLimiter) RateLimit() gin.HandlerFunc {
	return func(c *gin.Context) {
		clientIP := c.ClientIP()
		now := time.Now()

		// Clean old entries
		windowStart := now.Add(-rl.window)
		var validRequests []time.Time
		for _, t := range rl.requests[clientIP] {
			if t.After(windowStart) {
				validRequests = append(validRequests, t)
			}
		}

		// Check limit
		if len(validRequests) >= rl.limit {
			c.JSON(429, gin.H{
				"error":   "too_many_requests",
				"message": "Rate limit exceeded. Please try again later.",
			})
			c.Abort()
			return
		}

		// Add current request
		rl.requests[clientIP] = append(validRequests, now)

		c.Next()
	}
}
