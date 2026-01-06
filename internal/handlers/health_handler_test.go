package handlers

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestHealthHandler_Ping(t *testing.T) {
	handler := &HealthHandler{
		version: "1.0.0",
	}

	router := gin.New()
	router.GET("/health/ping", handler.Ping)

	req := httptest.NewRequest("GET", "/health/ping", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]string
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response["status"] != "pong" {
		t.Errorf("Expected status 'pong', got '%s'", response["status"])
	}
}

func TestHealthHandler_Health(t *testing.T) {
	handler := &HealthHandler{
		version: "1.0.0",
	}

	router := gin.New()
	router.GET("/health", handler.Health)

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response HealthResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.Status != "healthy" {
		t.Errorf("Expected status 'healthy', got '%s'", response.Status)
	}

	if response.Version != "1.0.0" {
		t.Errorf("Expected version '1.0.0', got '%s'", response.Version)
	}

	if response.Timestamp == "" {
		t.Error("Expected timestamp to be set")
	}
}

func TestHealthHandler_Live(t *testing.T) {
	handler := &HealthHandler{
		version: "1.0.0",
	}

	router := gin.New()
	router.GET("/health/live", handler.Live)

	req := httptest.NewRequest("GET", "/health/live", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response HealthResponse
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.Status != "alive" {
		t.Errorf("Expected status 'alive', got '%s'", response.Status)
	}
}

func TestNewHealthHandler(t *testing.T) {
	handler := NewHealthHandler(nil, "1.2.3")

	if handler == nil {
		t.Fatal("Expected handler to be created")
	}

	if handler.version != "1.2.3" {
		t.Errorf("Expected version '1.2.3', got '%s'", handler.version)
	}

	if handler.startTime.IsZero() {
		t.Error("Expected startTime to be set")
	}
}
