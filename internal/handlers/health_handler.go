// Package handlers provides HTTP handlers for API endpoints.
package handlers

import (
	"context"
	"net/http"
	"runtime"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/checkfix-tools/nisfix_backend/internal/database"
)

// Health status constants
const (
	statusHealthy   = "healthy"
	statusUnhealthy = "unhealthy"
)

// HealthHandler handles health check endpoints
// #INTEGRATION_POINT: Used by load balancers and monitoring systems
type HealthHandler struct {
	dbClient  *database.Client
	version   string
	startTime time.Time
}

// NewHealthHandler creates a new health handler
func NewHealthHandler(dbClient *database.Client, version string) *HealthHandler {
	return &HealthHandler{
		dbClient:  dbClient,
		version:   version,
		startTime: time.Now(),
	}
}

// HealthResponse represents a health check response
type HealthResponse struct {
	Status    string            `json:"status"`
	Timestamp string            `json:"timestamp"`
	Version   string            `json:"version,omitempty"`
	Services  map[string]string `json:"services,omitempty"`
}

// DetailedHealthResponse includes more information
type DetailedHealthResponse struct {
	Status    string             `json:"status"`
	Timestamp string             `json:"timestamp"`
	Version   string             `json:"version"`
	Uptime    string             `json:"uptime"`
	Services  map[string]Service `json:"services"`
	System    SystemInfo         `json:"system"`
}

// Service represents service health
type Service struct {
	Status      string `json:"status"`
	Latency     string `json:"latency,omitempty"`
	Description string `json:"description,omitempty"`
}

// SystemInfo represents system information
type SystemInfo struct {
	GoVersion    string  `json:"go_version"`
	NumCPU       int     `json:"num_cpu"`
	NumGoroutine int     `json:"num_goroutine"`
	MemAllocMB   float64 `json:"mem_alloc_mb"`
}

// Ping handles GET /health/ping
// @Summary Ping endpoint
// @Description Simple ping endpoint for basic availability check
// @Tags Health
// @Produce json
// @Success 200 {object} map[string]string
// @Router /health/ping [get]
func (h *HealthHandler) Ping(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "pong",
	})
}

// Health handles GET /health
// @Summary Health check endpoint
// @Description Returns basic health status
// @Tags Health
// @Produce json
// @Success 200 {object} HealthResponse
// @Router /health [get]
func (h *HealthHandler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, HealthResponse{
		Status:    statusHealthy,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Version:   h.version,
	})
}

// Ready handles GET /health/ready
// @Summary Readiness check endpoint
// @Description Checks if the service is ready to receive traffic (dependencies are healthy)
// @Tags Health
// @Produce json
// @Success 200 {object} HealthResponse
// @Failure 503 {object} HealthResponse
// @Router /health/ready [get]
func (h *HealthHandler) Ready(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	services := make(map[string]string)
	allHealthy := true

	// Check database
	if err := h.dbClient.Ping(ctx); err != nil {
		services["mongodb"] = statusUnhealthy
		allHealthy = false
	} else {
		services["mongodb"] = statusHealthy
	}

	status := "ready"
	statusCode := http.StatusOK
	if !allHealthy {
		status = "not_ready"
		statusCode = http.StatusServiceUnavailable
	}

	c.JSON(statusCode, HealthResponse{
		Status:    status,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Version:   h.version,
		Services:  services,
	})
}

// Live handles GET /health/live
// @Summary Liveness check endpoint
// @Description Indicates the service is running (for Kubernetes liveness probe)
// @Tags Health
// @Produce json
// @Success 200 {object} HealthResponse
// @Router /health/live [get]
func (h *HealthHandler) Live(c *gin.Context) {
	c.JSON(http.StatusOK, HealthResponse{
		Status:    "alive",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}

// Detailed handles GET /health/detailed
// @Summary Detailed health check
// @Description Returns detailed health information including system stats
// @Tags Health
// @Produce json
// @Success 200 {object} DetailedHealthResponse
// @Router /health/detailed [get]
func (h *HealthHandler) Detailed(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
	defer cancel()

	services := make(map[string]Service)
	allHealthy := true

	// Check database with latency
	dbStart := time.Now()
	if err := h.dbClient.Ping(ctx); err != nil {
		services["mongodb"] = Service{
			Status:      statusUnhealthy,
			Description: err.Error(),
		}
		allHealthy = false
	} else {
		services["mongodb"] = Service{
			Status:  statusHealthy,
			Latency: time.Since(dbStart).String(),
		}
	}

	// Get memory stats
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	status := statusHealthy
	if !allHealthy {
		status = "degraded"
	}

	c.JSON(http.StatusOK, DetailedHealthResponse{
		Status:    status,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Version:   h.version,
		Uptime:    time.Since(h.startTime).String(),
		Services:  services,
		System: SystemInfo{
			GoVersion:    runtime.Version(),
			NumCPU:       runtime.NumCPU(),
			NumGoroutine: runtime.NumGoroutine(),
			MemAllocMB:   float64(memStats.Alloc) / 1024 / 1024,
		},
	})
}

// RegisterRoutes registers health handler routes
func (h *HealthHandler) RegisterRoutes(router *gin.Engine) {
	// Health endpoints at root level (not under /api/v1)
	router.GET("/health", h.Health)
	router.GET("/health/ping", h.Ping)
	router.GET("/health/ready", h.Ready)
	router.GET("/health/live", h.Live)
	router.GET("/health/detailed", h.Detailed)
}
