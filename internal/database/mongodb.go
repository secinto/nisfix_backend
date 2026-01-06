// Package database provides MongoDB connection and initialization utilities
// #SCHEMA_IMPLEMENTATION: Using MongoDB with connection pooling and replica set support
package database

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

// Collection names as constants
// #INTEGRATION_POINT: All repositories use these collection names
const (
	CollectionOrganizations                = "organizations"
	CollectionUsers                        = "users"
	CollectionSecureLinks                  = "secure_links"
	CollectionQuestionnaireTemplates       = "questionnaire_templates"
	CollectionQuestionnaires               = "questionnaires"
	CollectionQuestions                    = "questions"
	CollectionCompanySupplierRelationships = "company_supplier_relationships"
	CollectionRequirements                 = "requirements"
	CollectionSupplierResponses            = "supplier_responses"
	CollectionQuestionnaireSubmissions     = "questionnaire_submissions"
	CollectionCheckFixVerifications        = "checkfix_verifications"
	CollectionAuditLogs                    = "audit_logs"
)

// Config holds MongoDB connection configuration
// #DATA_ASSUMPTION: Production uses replica set for high availability
type Config struct {
	URI                    string
	Database               string
	MaxPoolSize            uint64
	MinPoolSize            uint64
	MaxConnIdleTime        time.Duration
	ConnectTimeout         time.Duration
	ServerSelectionTimeout time.Duration
}

// DefaultConfig returns default MongoDB configuration
func DefaultConfig() Config {
	return Config{
		URI:                    "mongodb://localhost:27017",
		Database:               "nisfix_portal",
		MaxPoolSize:            100,
		MinPoolSize:            10,
		MaxConnIdleTime:        30 * time.Minute,
		ConnectTimeout:         10 * time.Second,
		ServerSelectionTimeout: 10 * time.Second,
	}
}

// Client wraps the MongoDB client with helper methods
type Client struct {
	client   *mongo.Client
	database *mongo.Database
	config   Config
}

// NewClient creates a new MongoDB client
func NewClient(cfg Config) (*Client, error) {
	ctx, cancel := context.WithTimeout(context.Background(), cfg.ConnectTimeout)
	defer cancel()

	// Configure client options
	// #IMPLEMENTATION_DECISION: Using connection pooling for performance
	clientOpts := options.Client().
		ApplyURI(cfg.URI).
		SetMaxPoolSize(cfg.MaxPoolSize).
		SetMinPoolSize(cfg.MinPoolSize).
		SetMaxConnIdleTime(cfg.MaxConnIdleTime).
		SetServerSelectionTimeout(cfg.ServerSelectionTimeout)

	// Connect to MongoDB
	client, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	// Verify connection with ping
	if err := client.Ping(ctx, readpref.Primary()); err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	return &Client{
		client:   client,
		database: client.Database(cfg.Database),
		config:   cfg,
	}, nil
}

// Database returns the MongoDB database
func (c *Client) Database() *mongo.Database {
	return c.database
}

// Client returns the underlying MongoDB client
func (c *Client) Client() *mongo.Client {
	return c.client
}

// Collection returns a MongoDB collection
func (c *Client) Collection(name string) *mongo.Collection {
	return c.database.Collection(name)
}

// Ping verifies the MongoDB connection
func (c *Client) Ping(ctx context.Context) error {
	return c.client.Ping(ctx, readpref.Primary())
}

// Close disconnects from MongoDB
func (c *Client) Close(ctx context.Context) error {
	return c.client.Disconnect(ctx)
}

// WithTransaction executes a function within a MongoDB transaction
// #IMPLEMENTATION_DECISION: Multi-document transactions for data consistency
func (c *Client) WithTransaction(ctx context.Context, fn func(sessCtx mongo.SessionContext) error) error {
	session, err := c.client.StartSession()
	if err != nil {
		return fmt.Errorf("failed to start session: %w", err)
	}
	defer session.EndSession(ctx)

	_, err = session.WithTransaction(ctx, func(sessCtx mongo.SessionContext) (interface{}, error) {
		return nil, fn(sessCtx)
	})

	if err != nil {
		return fmt.Errorf("transaction failed: %w", err)
	}

	return nil
}

// HealthCheck performs a health check on the database connection
func (c *Client) HealthCheck(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := c.Ping(ctx); err != nil {
		return fmt.Errorf("database health check failed: %w", err)
	}

	return nil
}

// DatabaseStats returns statistics about the database
func (c *Client) DatabaseStats(ctx context.Context) (map[string]interface{}, error) {
	var result map[string]interface{}
	err := c.database.RunCommand(ctx, map[string]interface{}{"dbStats": 1}).Decode(&result)
	if err != nil {
		return nil, fmt.Errorf("failed to get database stats: %w", err)
	}
	return result, nil
}

// CollectionNames returns all collection names in the database
func (c *Client) CollectionNames(ctx context.Context) ([]string, error) {
	names, err := c.database.ListCollectionNames(ctx, map[string]interface{}{})
	if err != nil {
		return nil, fmt.Errorf("failed to list collections: %w", err)
	}
	return names, nil
}

// DropDatabase drops the database (use with caution!)
func (c *Client) DropDatabase(ctx context.Context) error {
	return c.database.Drop(ctx)
}

// EnsureIndexes creates all required database indexes
// #IMPLEMENTATION_DECISION: Indexes created on application startup
// #COMPLETION_DRIVE: Assuming index creation is idempotent
func (c *Client) EnsureIndexes(ctx context.Context) error {
	indexes := []struct {
		collection string
		models     []mongo.IndexModel
	}{
		{
			collection: CollectionOrganizations,
			models: []mongo.IndexModel{
				{
					Keys:    bson.D{{Key: "slug", Value: 1}},
					Options: options.Index().SetUnique(true),
				},
				{
					Keys:    bson.D{{Key: "domain", Value: 1}},
					Options: options.Index().SetUnique(true).SetSparse(true),
				},
				{
					Keys: bson.D{{Key: "type", Value: 1}},
				},
			},
		},
		{
			collection: CollectionUsers,
			models: []mongo.IndexModel{
				{
					Keys:    bson.D{{Key: "email", Value: 1}},
					Options: options.Index().SetUnique(true),
				},
				{
					Keys: bson.D{{Key: "organization_id", Value: 1}},
				},
			},
		},
		{
			collection: CollectionSecureLinks,
			models: []mongo.IndexModel{
				{
					Keys:    bson.D{{Key: "secure_identifier", Value: 1}},
					Options: options.Index().SetUnique(true),
				},
				{
					Keys: bson.D{{Key: "email", Value: 1}},
				},
				{
					Keys:    bson.D{{Key: "expires_at", Value: 1}},
					Options: options.Index().SetExpireAfterSeconds(0), // TTL index
				},
			},
		},
		{
			collection: CollectionQuestionnaireTemplates,
			models: []mongo.IndexModel{
				{
					Keys: bson.D{
						{Key: "category", Value: 1},
						{Key: "is_system", Value: 1},
					},
				},
			},
		},
		{
			collection: CollectionQuestionnaires,
			models: []mongo.IndexModel{
				{
					Keys: bson.D{
						{Key: "organization_id", Value: 1},
						{Key: "status", Value: 1},
					},
				},
			},
		},
		{
			collection: CollectionQuestions,
			models: []mongo.IndexModel{
				{
					Keys: bson.D{
						{Key: "questionnaire_id", Value: 1},
						{Key: "order", Value: 1},
					},
				},
			},
		},
		{
			collection: CollectionCompanySupplierRelationships,
			models: []mongo.IndexModel{
				{
					Keys: bson.D{
						{Key: "company_id", Value: 1},
						{Key: "supplier_id", Value: 1},
					},
					Options: options.Index().SetUnique(true).SetSparse(true),
				},
				{
					Keys: bson.D{{Key: "invited_email", Value: 1}},
				},
				{
					Keys: bson.D{{Key: "status", Value: 1}},
				},
			},
		},
		{
			collection: CollectionRequirements,
			models: []mongo.IndexModel{
				{
					Keys: bson.D{
						{Key: "company_id", Value: 1},
						{Key: "status", Value: 1},
						{Key: "due_date", Value: 1},
					},
				},
				{
					Keys: bson.D{
						{Key: "supplier_id", Value: 1},
						{Key: "status", Value: 1},
					},
				},
			},
		},
		{
			collection: CollectionSupplierResponses,
			models: []mongo.IndexModel{
				{
					Keys:    bson.D{{Key: "requirement_id", Value: 1}},
					Options: options.Index().SetUnique(true),
				},
				{
					Keys: bson.D{{Key: "supplier_id", Value: 1}},
				},
			},
		},
		{
			collection: CollectionQuestionnaireSubmissions,
			models: []mongo.IndexModel{
				{
					Keys:    bson.D{{Key: "response_id", Value: 1}},
					Options: options.Index().SetUnique(true),
				},
				{
					Keys: bson.D{{Key: "questionnaire_id", Value: 1}},
				},
			},
		},
		{
			collection: CollectionCheckFixVerifications,
			models: []mongo.IndexModel{
				{
					Keys:    bson.D{{Key: "supplier_id", Value: 1}},
					Options: options.Index().SetUnique(true),
				},
				{
					Keys: bson.D{{Key: "expires_at", Value: 1}},
				},
			},
		},
		{
			collection: CollectionAuditLogs,
			models: []mongo.IndexModel{
				{
					Keys: bson.D{{Key: "actor_user_id", Value: 1}},
				},
				{
					Keys: bson.D{
						{Key: "resource_type", Value: 1},
						{Key: "resource_id", Value: 1},
					},
				},
				{
					Keys: bson.D{{Key: "created_at", Value: -1}},
				},
			},
		},
	}

	for _, idx := range indexes {
		collection := c.Collection(idx.collection)
		_, err := collection.Indexes().CreateMany(ctx, idx.models)
		if err != nil {
			return fmt.Errorf("failed to create indexes for %s: %w", idx.collection, err)
		}
	}

	return nil
}

// SeedData seeds initial data including questionnaire templates
// #IMPLEMENTATION_DECISION: Only seeds if data doesn't exist (idempotent)
func (c *Client) SeedData(ctx context.Context) error {
	seeder := NewSeeder(c.database)
	return seeder.SeedAll(ctx)
}
