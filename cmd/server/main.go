// @title NisFix Backend API
// @version 1.0
// @description B2B Supplier Security Portal API - Companies manage supplier security requirements through questionnaires and CheckFix reports
// @termsOfService http://swagger.io/terms/

// @contact.name NisFix Support
// @contact.email support@nisfix.io

// @license.name Apache 2.0
// @license.url http://www.apache.org/licenses/LICENSE-2.0.html

// @host localhost:8080
// @BasePath /api/v1

// @securityDefinitions.apikey BearerAuth
// @in header
// @name Authorization
// @description Enter your bearer token in the format: Bearer {token}

// Package main is the entry point for the NisFix Backend API server.
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/checkfix-tools/nisfix_backend/internal/auth"
	"github.com/checkfix-tools/nisfix_backend/internal/config"
	"github.com/checkfix-tools/nisfix_backend/internal/database"
	"github.com/checkfix-tools/nisfix_backend/internal/handlers"
	"github.com/checkfix-tools/nisfix_backend/internal/middleware"
	"github.com/checkfix-tools/nisfix_backend/internal/repository"
	"github.com/checkfix-tools/nisfix_backend/internal/services"

	// Swagger docs
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	_ "github.com/checkfix-tools/nisfix_backend/docs"
)

// Build-time variables (set via ldflags)
var (
	Version   = "0.1.0-dev"
	BuildTime = "unknown"
	GitCommit = "unknown"
	GitBranch = "unknown"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Set Gin mode based on environment
	if cfg.IsProduction() {
		gin.SetMode(gin.ReleaseMode)
	}

	// Initialize database connection
	ctx := context.Background()
	dbCfg := database.Config{
		URI:                    cfg.DatabaseURI,
		Database:               cfg.DatabaseName,
		MaxPoolSize:            100,
		MinPoolSize:            10,
		MaxConnIdleTime:        30 * time.Minute,
		ConnectTimeout:         10 * time.Second,
		ServerSelectionTimeout: 10 * time.Second,
	}

	dbClient, err := database.NewClient(dbCfg)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Initialize JWT service early (before defer) to avoid exitAfterDefer issue
	jwtCfg := auth.JWTConfig{
		PrivateKeyPath:     cfg.JWTPrivateKeyPath,
		PublicKeyPath:      cfg.JWTPublicKeyPath,
		AccessTokenExpiry:  cfg.AccessTokenExpiry,
		RefreshTokenExpiry: cfg.RefreshTokenExpiry,
		Issuer:             "nisfix-backend",
	}

	jwtService, err := auth.NewJWTService(jwtCfg)
	if err != nil {
		if closeErr := dbClient.Close(ctx); closeErr != nil {
			log.Printf("Error closing database connection: %v", closeErr)
		}
		log.Fatalf("Failed to initialize JWT service: %v", err)
	}

	defer func() {
		if closeErr := dbClient.Close(ctx); closeErr != nil {
			log.Printf("Error closing database connection: %v", closeErr)
		}
	}()

	// Ensure indexes
	log.Println("Creating database indexes...")
	if indexErr := dbClient.EnsureIndexes(ctx); indexErr != nil {
		log.Printf("Warning: Failed to create indexes: %v", indexErr)
	}

	// Seed initial data (questionnaire templates)
	log.Println("Seeding initial data...")
	if seedErr := dbClient.SeedData(ctx); seedErr != nil {
		log.Printf("Warning: Failed to seed data: %v", seedErr)
	}

	// Initialize repositories
	userRepo := repository.NewUserRepository(dbClient)
	orgRepo := repository.NewOrganizationRepository(dbClient)
	secureLinkRepo := repository.NewSecureLinkRepository(dbClient)
	relationshipRepo := repository.NewRelationshipRepository(dbClient)
	questionnaireRepo := repository.NewQuestionnaireRepository(dbClient)
	templateRepo := repository.NewQuestionnaireTemplateRepository(dbClient)
	questionRepo := repository.NewQuestionRepository(dbClient)
	requirementRepo := repository.NewRequirementRepository(dbClient)
	responseRepo := repository.NewResponseRepository(dbClient)
	submissionRepo := repository.NewSubmissionRepository(dbClient)
	verificationRepo := repository.NewVerificationRepository(dbClient)

	// Initialize mail service (always use HTTP service)
	mailService := services.NewHTTPMailService(&cfg.Mail)

	// Initialize auth service
	authServiceCfg := services.AuthServiceConfig{
		MagicLinkBaseURL:    cfg.MagicLinkBaseURL,
		RateLimitCount:      5,
		RateLimitWindowMins: 15,
	}
	authService := services.NewAuthService(
		userRepo,
		orgRepo,
		secureLinkRepo,
		jwtService,
		mailService,
		authServiceCfg,
	)

	// Initialize relationship service
	relationshipService := services.NewRelationshipService(
		relationshipRepo,
		orgRepo,
		userRepo,
		mailService,
		cfg.MagicLinkBaseURL,
	)

	// Initialize questionnaire service
	questionnaireService := services.NewQuestionnaireService(
		questionnaireRepo,
		templateRepo,
		questionRepo,
	)

	// Initialize template service
	templateService := services.NewTemplateService(templateRepo)

	// Initialize requirement service
	requirementService := services.NewRequirementService(
		requirementRepo,
		relationshipRepo,
		questionnaireRepo,
	)

	// Initialize response service
	responseService := services.NewResponseService(
		responseRepo,
		submissionRepo,
		requirementRepo,
		questionnaireRepo,
		questionRepo,
	)

	// Initialize review service
	reviewService := services.NewReviewService(
		requirementRepo,
		responseRepo,
		submissionRepo,
	)

	// Initialize CheckFix API client
	// #IMPLEMENTATION_DECISION: Use mock in development, HTTP client in production
	var checkFixAPIClient services.CheckFixAPIClient
	if cfg.IsDevelopment() || cfg.CheckFixAPIURL == "" {
		log.Println("Using mock CheckFix API client in development mode")
		checkFixAPIClient = services.NewMockCheckFixAPIClient()
	} else {
		checkFixAPIClient = services.NewHTTPCheckFixAPIClient(cfg.CheckFixAPIURL, cfg.CheckFixAPIKey)
	}

	// Initialize CheckFix service
	checkFixService := services.NewCheckFixService(
		checkFixAPIClient,
		verificationRepo,
		responseRepo,
		requirementRepo,
		orgRepo,
	)

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(authService)
	healthHandler := handlers.NewHealthHandler(dbClient, Version)
	relationshipHandler := handlers.NewRelationshipHandler(relationshipService)
	questionnaireHandler := handlers.NewQuestionnaireHandler(questionnaireService)
	templateHandler := handlers.NewTemplateHandler(templateRepo, templateService)
	requirementHandler := handlers.NewRequirementHandler(requirementService)
	supplierPortalHandler := handlers.NewSupplierPortalHandler(relationshipRepo, requirementRepo, responseService)
	reviewHandler := handlers.NewReviewHandler(reviewService)
	checkFixHandler := handlers.NewCheckFixHandler(checkFixService)
	organizationHandler := handlers.NewOrganizationHandler(orgRepo)

	// Create Gin router
	router := gin.New()

	// Apply global middleware
	router.Use(middleware.Recovery())
	router.Use(middleware.RequestID())
	router.Use(middleware.Logger())
	router.Use(middleware.CORS(cfg.AllowedOrigins))
	router.Use(middleware.SecureHeaders())

	// Register health routes (not under /api/v1)
	healthHandler.RegisterRoutes(router)

	// Register Swagger documentation route
	router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// Create API v1 group
	apiV1 := router.Group("/api/v1")

	// Create auth middleware
	authMiddleware := middleware.AuthMiddleware(jwtService)

	// Register routes
	authHandler.RegisterRoutes(apiV1, authMiddleware)
	relationshipHandler.RegisterRoutes(apiV1, authMiddleware)
	questionnaireHandler.RegisterRoutes(apiV1, authMiddleware)
	templateHandler.RegisterRoutes(apiV1, authMiddleware)
	requirementHandler.RegisterRoutes(apiV1, authMiddleware)
	supplierPortalHandler.RegisterRoutes(apiV1, authMiddleware)
	reviewHandler.RegisterRoutes(apiV1, authMiddleware)
	checkFixHandler.RegisterRoutes(apiV1, authMiddleware)
	organizationHandler.RegisterRoutes(apiV1, authMiddleware)

	// Create HTTP server
	server := &http.Server{
		Addr:         ":" + cfg.ServerPort,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start server in goroutine
	go func() {
		log.Printf("Starting NisFix Backend API server v%s on port %s", Version, cfg.ServerPort)
		log.Printf("Build: %s | Commit: %s | Branch: %s", BuildTime, GitCommit, GitBranch)
		log.Printf("Environment: %s", cfg.Environment)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	// Create shutdown context with timeout
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Shutdown server gracefully
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	log.Println("Server shutdown complete")
}
