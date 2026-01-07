// Package main provides a CLI tool to create a company organization with an admin user.
// Usage: go run cmd/seed-company/main.go -name "Company Name" -email "admin@company.com"
package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
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
	name := flag.String("name", "", "Company name (required)")
	email := flag.String("email", "", "Admin user email (required)")
	slug := flag.String("slug", "", "URL-safe slug (auto-generated from name if not provided)")
	contactEmail := flag.String("contact-email", "", "Company contact email (defaults to admin email)")
	adminName := flag.String("admin-name", "", "Admin user display name (optional)")
	envFile := flag.String("env", "", "Path to .env file (defaults to .env in current dir or backend dir)")
	dryRun := flag.Bool("dry-run", false, "Print what would be created without writing to database")

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: %s [options]\n\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "Creates a company organization with an admin user in the NisFix database.\n\n")
		fmt.Fprintf(os.Stderr, "Configuration is loaded from .env file and/or environment variables.\n")
		fmt.Fprintf(os.Stderr, "Environment variables take precedence over .env file values.\n\n")
		fmt.Fprintf(os.Stderr, "Required config (via .env or environment):\n")
		fmt.Fprintf(os.Stderr, "  NISFIX_DATABASE_URI   MongoDB connection URI\n")
		fmt.Fprintf(os.Stderr, "  NISFIX_DATABASE_NAME  Database name (default: nisfix)\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nExamples:\n")
		fmt.Fprintf(os.Stderr, "  %s -name \"Acme Corp\" -email \"admin@acme.com\"\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -name \"Test Company\" -email \"test@example.com\" -env /path/to/.env\n", os.Args[0])
		fmt.Fprintf(os.Stderr, "  %s -name \"Test Company\" -email \"test@example.com\" -dry-run\n", os.Args[0])
	}

	flag.Parse()

	// Load .env file
	loadEnvFile(*envFile)

	// Validate required flags
	if *name == "" {
		log.Fatal("Error: -name is required")
	}
	if *email == "" {
		log.Fatal("Error: -email is required")
	}

	// Validate email format
	if !isValidEmail(*email) {
		log.Fatalf("Error: invalid email format: %s", *email)
	}

	// Auto-generate slug if not provided
	if *slug == "" {
		*slug = generateSlug(*name)
	}

	// Default contact email to admin email
	if *contactEmail == "" {
		*contactEmail = *email
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

	// Create organization and user objects
	now := time.Now().UTC()
	orgID := primitive.NewObjectID()
	userID := primitive.NewObjectID()

	org := &models.Organization{
		ID:           orgID,
		Type:         models.OrganizationTypeCompany,
		Name:         *name,
		Slug:         *slug,
		ContactEmail: *contactEmail,
		Settings:     models.DefaultOrganizationSettings(),
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	user := &models.User{
		ID:             userID,
		Email:          *email,
		Name:           *adminName,
		OrganizationID: orgID,
		Role:           models.UserRoleAdmin,
		IsActive:       true,
		Language:       "en",
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	// Print what will be created
	fmt.Println("=== Company Organization ===")
	fmt.Printf("  ID:            %s\n", org.ID.Hex())
	fmt.Printf("  Name:          %s\n", org.Name)
	fmt.Printf("  Slug:          %s\n", org.Slug)
	fmt.Printf("  Type:          %s\n", org.Type)
	fmt.Printf("  Contact Email: %s\n", org.ContactEmail)
	fmt.Println()
	fmt.Println("=== Admin User ===")
	fmt.Printf("  ID:              %s\n", user.ID.Hex())
	fmt.Printf("  Email:           %s\n", user.Email)
	fmt.Printf("  Name:            %s\n", user.Name)
	fmt.Printf("  Role:            %s\n", user.Role)
	fmt.Printf("  Organization ID: %s\n", user.OrganizationID.Hex())
	fmt.Println()

	if *dryRun {
		fmt.Println("[DRY RUN] No changes made to database")
		return
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

	// Check if organization with same slug already exists
	orgCollection := db.Collection(models.Organization{}.CollectionName())
	var existingOrg models.Organization
	err = orgCollection.FindOne(ctx, bson.M{"slug": org.Slug, "deleted_at": nil}).Decode(&existingOrg)
	if err == nil {
		log.Fatalf("Error: organization with slug '%s' already exists (ID: %s)", org.Slug, existingOrg.ID.Hex())
	} else if err != mongo.ErrNoDocuments {
		log.Fatalf("Error checking existing organization: %v", err)
	}

	// Check if user with same email already exists
	userCollection := db.Collection(models.User{}.CollectionName())
	var existingUser models.User
	err = userCollection.FindOne(ctx, bson.M{"email": user.Email, "deleted_at": nil}).Decode(&existingUser)
	if err == nil {
		log.Fatalf("Error: user with email '%s' already exists (ID: %s)", user.Email, existingUser.ID.Hex())
	} else if err != mongo.ErrNoDocuments {
		log.Fatalf("Error checking existing user: %v", err)
	}

	// Insert organization
	_, err = orgCollection.InsertOne(ctx, org)
	if err != nil {
		log.Fatalf("Failed to create organization: %v", err)
	}
	fmt.Printf("✓ Created organization: %s (%s)\n", org.Name, org.ID.Hex())

	// Insert user
	_, err = userCollection.InsertOne(ctx, user)
	if err != nil {
		// Rollback: delete the organization
		_, _ = orgCollection.DeleteOne(ctx, bson.M{"_id": org.ID})
		log.Fatalf("Failed to create user (organization rolled back): %v", err)
	}
	fmt.Printf("✓ Created admin user: %s (%s)\n", user.Email, user.ID.Hex())

	fmt.Println()
	fmt.Println("Company setup complete!")
	fmt.Printf("The admin can now log in at your frontend using: %s\n", user.Email)
}

// generateSlug creates a URL-safe slug from a company name
func generateSlug(name string) string {
	// Convert to lowercase
	slug := strings.ToLower(name)

	// Replace spaces and underscores with hyphens
	slug = strings.ReplaceAll(slug, " ", "-")
	slug = strings.ReplaceAll(slug, "_", "-")

	// Remove any character that isn't alphanumeric or hyphen
	reg := regexp.MustCompile(`[^a-z0-9-]`)
	slug = reg.ReplaceAllString(slug, "")

	// Replace multiple consecutive hyphens with single hyphen
	reg = regexp.MustCompile(`-+`)
	slug = reg.ReplaceAllString(slug, "-")

	// Trim hyphens from start and end
	slug = strings.Trim(slug, "-")

	return slug
}

// isValidEmail performs basic email validation
func isValidEmail(email string) bool {
	// Simple regex for email validation
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
			log.Printf("Error loading .env file: %v", err)
		}
	}
}
