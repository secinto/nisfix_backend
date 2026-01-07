// Package main provides a CLI tool to generate a magic link for user authentication.
// Usage: go run cmd/generate-magic-link/main.go -email "user@example.com"
// This is useful for development when email service is not configured.
package main

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"time"

	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/checkfix-tools/nisfix_backend/internal/models"
)

func main() {
	// Define command line flags
	email := flag.String("email", "", "User email to generate magic link for (required)")
	envFile := flag.String("env", "", "Path to .env file (defaults to .env in current dir or backend dir)")
	baseURL := flag.String("base-url", "", "Override NISFIX_MAGIC_LINK_BASE_URL from environment")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Generates a magic link for user authentication (development use).\n\n")
		fmt.Fprintf(os.Stderr, "Configuration is loaded from .env file and/or environment variables.\n\n")
		fmt.Fprintf(os.Stderr, "Required config (via .env or environment):\n")
		fmt.Fprintf(os.Stderr, "  NISFIX_DATABASE_URI        MongoDB connection URI\n")
		fmt.Fprintf(os.Stderr, "  NISFIX_DATABASE_NAME       Database name (default: nisfix)\n")
		fmt.Fprintf(os.Stderr, "  NISFIX_MAGIC_LINK_BASE_URL Frontend base URL for magic links\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  %s -email \"admin@company.com\"\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -email \"user@example.com\" -base-url \"http://localhost:3000\"\n", os.Args[0])
	}

	flag.Parse()

	// Load .env file
	loadEnvFile(*envFile)

	// Validate required flags
	if *email == "" {
		log.Fatal("Error: -email is required")
	}

	// Validate email format
	if !isValidEmail(*email) {
		log.Fatalf("Error: invalid email format: %s", *email)
	}

	// Load database configuration from environment
	dbURI := os.Getenv("NISFIX_DATABASE_URI")
	if dbURI == "" {
		log.Fatal("Error: NISFIX_DATABASE_URI environment variable is required")
	}
	dbName := os.Getenv("NISFIX_DATABASE_NAME")
	if dbName == "" {
		dbName = "nisfix"
	}

	// Get magic link base URL
	magicLinkBaseURL := *baseURL
	if magicLinkBaseURL == "" {
		magicLinkBaseURL = os.Getenv("NISFIX_MAGIC_LINK_BASE_URL")
	}
	if magicLinkBaseURL == "" {
		magicLinkBaseURL = "http://localhost:3000" // Default for development
	}

	// Connect to MongoDB
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	clientOpts := options.Client().ApplyURI(dbURI)
	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		log.Fatalf("Failed to connect to MongoDB: %v", err)
	}
	defer func() {
		if disconnectErr := client.Disconnect(ctx); disconnectErr != nil {
			log.Printf("Error disconnecting from MongoDB: %v", disconnectErr)
		}
	}()

	// Ping database
	if err := client.Ping(ctx, nil); err != nil {
		log.Fatalf("Failed to ping MongoDB: %v", err)
	}

	db := client.Database(dbName)

	// Find user by email
	userCollection := db.Collection(models.User{}.CollectionName())
	var user models.User
	err = userCollection.FindOne(ctx, bson.M{
		"email":      *email,
		"deleted_at": nil,
	}).Decode(&user)
	if err == mongo.ErrNoDocuments {
		log.Fatalf("Error: no user found with email '%s'", *email)
	} else if err != nil {
		log.Fatalf("Error finding user: %v", err)
	}

	if !user.IsActive {
		log.Fatalf("Error: user '%s' is inactive", *email)
	}

	// Verify organization exists
	orgCollection := db.Collection(models.Organization{}.CollectionName())
	var org models.Organization
	err = orgCollection.FindOne(ctx, bson.M{
		"_id":        user.OrganizationID,
		"deleted_at": nil,
	}).Decode(&org)
	if err == mongo.ErrNoDocuments {
		log.Fatalf("Error: organization not found for user '%s'", *email)
	} else if err != nil {
		log.Fatalf("Error finding organization: %v", err)
	}

	// Invalidate existing magic links for this email
	secureLinkCollection := db.Collection(models.SecureLink{}.CollectionName())
	_, err = secureLinkCollection.UpdateMany(ctx,
		bson.M{
			"email":    *email,
			"is_valid": true,
			"type":     models.SecureLinkTypeAuth,
		},
		bson.M{"$set": bson.M{"is_valid": false}},
	)
	if err != nil {
		log.Printf("Warning: failed to invalidate existing links: %v", err)
	}

	// Generate secure identifier (32 bytes = 64 hex characters)
	identifier, err := generateSecureIdentifier()
	if err != nil {
		log.Fatalf("Failed to generate secure identifier: %v", err)
	}

	// Create secure link
	now := time.Now().UTC()
	link := models.SecureLink{
		ID:               primitive.NewObjectID(),
		SecureIdentifier: identifier,
		Type:             models.SecureLinkTypeAuth,
		Email:            *email,
		UserID:           &user.ID,
		ExpiresAt:        now.Add(models.AuthLinkExpiryDuration),
		IsValid:          true,
		CreatedAt:        now,
	}

	_, err = secureLinkCollection.InsertOne(ctx, link)
	if err != nil {
		log.Fatalf("Failed to create secure link: %v", err)
	}

	// Build magic link URL (path parameter to match frontend route /auth/verify/:token)
	magicLinkURL := fmt.Sprintf("%s/auth/verify/%s", magicLinkBaseURL, identifier)

	// Output results
	fmt.Println()
	fmt.Println("=== Magic Link Generated ===")
	fmt.Printf("  User:         %s\n", user.Email)
	fmt.Printf("  Name:         %s\n", user.Name)
	fmt.Printf("  Organization: %s\n", org.Name)
	fmt.Printf("  Expires:      %s (%d minutes)\n", link.ExpiresAt.Format(time.RFC3339), int(models.AuthLinkExpiryDuration.Minutes()))
	fmt.Println()
	fmt.Println("Magic Link URL:")
	fmt.Println(magicLinkURL)
	fmt.Println()
	fmt.Println("Note: This link can only be used once and expires in 15 minutes.")
}

// generateSecureIdentifier generates a cryptographically secure random identifier
func generateSecureIdentifier() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

// isValidEmail performs basic email validation
func isValidEmail(email string) bool {
	pattern := `^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`
	matched, _ := regexp.MatchString(pattern, email)
	return matched
}

// loadEnvFile loads environment variables from a .env file
func loadEnvFile(path string) {
	if path == "" {
		// Try to find .env in current dir or backend dir
		cwd, _ := os.Getwd()
		if _, err := os.Stat(filepath.Join(cwd, ".env")); err == nil {
			path = ".env"
		} else if _, err := os.Stat(filepath.Join(cwd, "backend", ".env")); err == nil {
			path = filepath.Join(cwd, "backend", ".env")
		}
	}

	if path != "" {
		if err := godotenv.Load(path); err != nil {
			log.Printf("Warning: Error loading .env file: %v", err)
		}
	}
}
