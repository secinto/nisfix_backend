// Package repository provides data access layer implementations.
package repository

import (
	"context"
	"time"

	"github.com/checkfix-tools/nisfix_backend/internal/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// AuditRepository defines operations for audit log management
// #IMPLEMENTATION_DECISION: Audit logs are append-only, no update/delete operations
type AuditRepository interface {
	// Create creates a new audit log entry
	Create(ctx context.Context, log *models.AuditLog) error

	// GetByID retrieves an audit log by ID
	GetByID(ctx context.Context, id primitive.ObjectID) (*models.AuditLog, error)

	// ListByResource lists audit logs for a specific resource
	ListByResource(ctx context.Context, resourceType string, resourceID primitive.ObjectID, opts PaginationOptions) (*PaginatedResult[models.AuditLog], error)

	// ListByActor lists audit logs by actor
	ListByActor(ctx context.Context, actorUserID primitive.ObjectID, opts PaginationOptions) (*PaginatedResult[models.AuditLog], error)

	// ListByOrganization lists audit logs for an organization
	ListByOrganization(ctx context.Context, orgID primitive.ObjectID, opts PaginationOptions) (*PaginatedResult[models.AuditLog], error)

	// ListByAction lists audit logs by action type
	ListByAction(ctx context.Context, action models.AuditAction, opts PaginationOptions) (*PaginatedResult[models.AuditLog], error)

	// ListByDateRange lists audit logs within a date range
	ListByDateRange(ctx context.Context, startDate, endDate time.Time, opts PaginationOptions) (*PaginatedResult[models.AuditLog], error)
}

// MongoAuditRepository implements AuditRepository for MongoDB
type MongoAuditRepository struct {
	collection *mongo.Collection
}

// NewMongoAuditRepository creates a new MongoDB audit repository
func NewMongoAuditRepository(db *mongo.Database) *MongoAuditRepository {
	return &MongoAuditRepository{
		collection: db.Collection(models.AuditLog{}.CollectionName()),
	}
}

// Create creates a new audit log entry
func (r *MongoAuditRepository) Create(ctx context.Context, log *models.AuditLog) error {
	log.BeforeCreate()
	_, err := r.collection.InsertOne(ctx, log)
	return err
}

// GetByID retrieves an audit log by ID
func (r *MongoAuditRepository) GetByID(ctx context.Context, id primitive.ObjectID) (*models.AuditLog, error) {
	var log models.AuditLog
	err := r.collection.FindOne(ctx, bson.M{"_id": id}).Decode(&log)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, models.ErrAuditLogNotFound
		}
		return nil, err
	}
	return &log, nil
}

// ListByResource lists audit logs for a specific resource
func (r *MongoAuditRepository) ListByResource(ctx context.Context, resourceType string, resourceID primitive.ObjectID, opts PaginationOptions) (*PaginatedResult[models.AuditLog], error) {
	filter := bson.M{
		"resource_type": resourceType,
		"resource_id":   resourceID,
	}
	return r.listWithPagination(ctx, filter, opts)
}

// ListByActor lists audit logs by actor
func (r *MongoAuditRepository) ListByActor(ctx context.Context, actorUserID primitive.ObjectID, opts PaginationOptions) (*PaginatedResult[models.AuditLog], error) {
	filter := bson.M{"actor_user_id": actorUserID}
	return r.listWithPagination(ctx, filter, opts)
}

// ListByOrganization lists audit logs for an organization
func (r *MongoAuditRepository) ListByOrganization(ctx context.Context, orgID primitive.ObjectID, opts PaginationOptions) (*PaginatedResult[models.AuditLog], error) {
	filter := bson.M{"actor_org_id": orgID}
	return r.listWithPagination(ctx, filter, opts)
}

// ListByAction lists audit logs by action type
func (r *MongoAuditRepository) ListByAction(ctx context.Context, action models.AuditAction, opts PaginationOptions) (*PaginatedResult[models.AuditLog], error) {
	filter := bson.M{"action": action}
	return r.listWithPagination(ctx, filter, opts)
}

// ListByDateRange lists audit logs within a date range
func (r *MongoAuditRepository) ListByDateRange(ctx context.Context, startDate, endDate time.Time, opts PaginationOptions) (*PaginatedResult[models.AuditLog], error) {
	filter := bson.M{
		"created_at": bson.M{
			"$gte": startDate,
			"$lte": endDate,
		},
	}
	return r.listWithPagination(ctx, filter, opts)
}

// listWithPagination is a helper for paginated queries
func (r *MongoAuditRepository) listWithPagination(ctx context.Context, filter bson.M, opts PaginationOptions) (*PaginatedResult[models.AuditLog], error) {
	// Count total
	total, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, err
	}

	// Find with pagination (default sort by created_at descending)
	sortBy := opts.SortBy
	if sortBy == "" {
		sortBy = "created_at"
	}
	sortDir := opts.SortDir
	if sortDir == 0 {
		sortDir = -1 // Default descending for audit logs
	}

	findOpts := options.Find().
		SetSort(bson.D{{Key: sortBy, Value: sortDir}}).
		SetSkip(int64((opts.Page - 1) * opts.Limit)).
		SetLimit(int64(opts.Limit))

	cursor, err := r.collection.Find(ctx, filter, findOpts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var logs []models.AuditLog
	if err := cursor.All(ctx, &logs); err != nil {
		return nil, err
	}

	totalPages := int(total) / opts.Limit
	if int(total)%opts.Limit > 0 {
		totalPages++
	}

	return &PaginatedResult[models.AuditLog]{
		Items:      logs,
		TotalCount: total,
		Page:       opts.Page,
		Limit:      opts.Limit,
		TotalPages: totalPages,
	}, nil
}

// Ensure MongoAuditRepository implements AuditRepository
var _ AuditRepository = (*MongoAuditRepository)(nil)
