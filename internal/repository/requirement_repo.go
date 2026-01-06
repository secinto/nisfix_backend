package repository

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/checkfix-tools/nisfix_backend/internal/models"
)

// MongoRequirementRepository implements RequirementRepository for MongoDB
// #ORM_INTEGRATION: MongoDB driver-based repository implementation
type MongoRequirementRepository struct {
	collection *mongo.Collection
}

// NewMongoRequirementRepository creates a new MongoDB requirement repository
func NewMongoRequirementRepository(db *mongo.Database) *MongoRequirementRepository {
	return &MongoRequirementRepository{
		collection: db.Collection(models.Requirement{}.CollectionName()),
	}
}

// Create creates a new requirement
func (r *MongoRequirementRepository) Create(ctx context.Context, requirement *models.Requirement) error {
	requirement.BeforeCreate()
	_, err := r.collection.InsertOne(ctx, requirement)
	return err
}

// GetByID finds a requirement by ID
func (r *MongoRequirementRepository) GetByID(ctx context.Context, id primitive.ObjectID) (*models.Requirement, error) {
	var requirement models.Requirement
	filter := bson.M{"_id": id}
	err := r.collection.FindOne(ctx, filter).Decode(&requirement)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, models.ErrRequirementNotFound
	}
	if err != nil {
		return nil, err
	}
	return &requirement, nil
}

// Update updates a requirement
func (r *MongoRequirementRepository) Update(ctx context.Context, requirement *models.Requirement) error {
	requirement.BeforeUpdate()
	filter := bson.M{"_id": requirement.ID}
	update := bson.M{"$set": requirement}
	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return models.ErrRequirementNotFound
	}
	return nil
}

// ListByCompany lists requirements for a company
func (r *MongoRequirementRepository) ListByCompany(ctx context.Context, companyID primitive.ObjectID, status *models.RequirementStatus, opts PaginationOptions) (*PaginatedResult[models.Requirement], error) {
	filter := bson.M{"company_id": companyID}
	if status != nil {
		filter["status"] = *status
	}

	// Count total
	total, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, err
	}

	// Apply pagination
	skip := int64((opts.Page - 1) * opts.Limit)
	findOpts := options.Find().
		SetSkip(skip).
		SetLimit(int64(opts.Limit)).
		SetSort(bson.D{{Key: "due_date", Value: 1}, {Key: opts.SortBy, Value: opts.SortDir}})

	cursor, err := r.collection.Find(ctx, filter, findOpts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx) //nolint:errcheck // defer close

	var requirements []models.Requirement
	if err := cursor.All(ctx, &requirements); err != nil {
		return nil, err
	}

	totalPages := int(total) / opts.Limit
	if int(total)%opts.Limit > 0 {
		totalPages++
	}

	return &PaginatedResult[models.Requirement]{
		Items:      requirements,
		TotalCount: total,
		Page:       opts.Page,
		Limit:      opts.Limit,
		TotalPages: totalPages,
	}, nil
}

// ListBySupplier lists requirements for a supplier
func (r *MongoRequirementRepository) ListBySupplier(ctx context.Context, supplierID primitive.ObjectID, status *models.RequirementStatus, opts PaginationOptions) (*PaginatedResult[models.Requirement], error) {
	filter := bson.M{"supplier_id": supplierID}
	if status != nil {
		filter["status"] = *status
	}

	// Count total
	total, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, err
	}

	// Apply pagination
	skip := int64((opts.Page - 1) * opts.Limit)
	findOpts := options.Find().
		SetSkip(skip).
		SetLimit(int64(opts.Limit)).
		SetSort(bson.D{{Key: "due_date", Value: 1}, {Key: opts.SortBy, Value: opts.SortDir}})

	cursor, err := r.collection.Find(ctx, filter, findOpts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx) //nolint:errcheck // defer close

	var requirements []models.Requirement
	if err := cursor.All(ctx, &requirements); err != nil {
		return nil, err
	}

	totalPages := int(total) / opts.Limit
	if int(total)%opts.Limit > 0 {
		totalPages++
	}

	return &PaginatedResult[models.Requirement]{
		Items:      requirements,
		TotalCount: total,
		Page:       opts.Page,
		Limit:      opts.Limit,
		TotalPages: totalPages,
	}, nil
}

// ListByRelationship lists requirements for a relationship
func (r *MongoRequirementRepository) ListByRelationship(ctx context.Context, relationshipID primitive.ObjectID, status *models.RequirementStatus) ([]models.Requirement, error) {
	filter := bson.M{"relationship_id": relationshipID}
	if status != nil {
		filter["status"] = *status
	}

	findOpts := options.Find().SetSort(bson.D{{Key: "due_date", Value: 1}})

	cursor, err := r.collection.Find(ctx, filter, findOpts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx) //nolint:errcheck // defer close

	var requirements []models.Requirement
	if err := cursor.All(ctx, &requirements); err != nil {
		return nil, err
	}

	return requirements, nil
}

// ListOverdue lists overdue requirements
// #QUERY_PATTERN: Dashboard queries: "overdue requirements"
func (r *MongoRequirementRepository) ListOverdue(ctx context.Context, companyID *primitive.ObjectID) ([]models.Requirement, error) {
	filter := bson.M{
		"status": bson.M{
			"$in": []models.RequirementStatus{
				models.RequirementStatusPending,
				models.RequirementStatusInProgress,
			},
		},
		"due_date": bson.M{
			"$lt": time.Now().UTC(),
		},
	}
	if companyID != nil {
		filter["company_id"] = *companyID
	}

	findOpts := options.Find().SetSort(bson.D{{Key: "due_date", Value: 1}})

	cursor, err := r.collection.Find(ctx, filter, findOpts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx) //nolint:errcheck // defer close

	var requirements []models.Requirement
	if err := cursor.All(ctx, &requirements); err != nil {
		return nil, err
	}

	return requirements, nil
}

// ListNeedingReminder lists requirements that need reminders
func (r *MongoRequirementRepository) ListNeedingReminder(ctx context.Context, reminderDaysBefore int) ([]models.Requirement, error) {
	reminderDate := time.Now().UTC().AddDate(0, 0, reminderDaysBefore)
	filter := bson.M{
		"status": bson.M{
			"$in": []models.RequirementStatus{
				models.RequirementStatusPending,
				models.RequirementStatusInProgress,
			},
		},
		"due_date": bson.M{
			"$lte": reminderDate,
			"$gte": time.Now().UTC(),
		},
		"reminder_sent_at": nil,
	}

	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx) //nolint:errcheck // defer close

	var requirements []models.Requirement
	if err := cursor.All(ctx, &requirements); err != nil {
		return nil, err
	}

	return requirements, nil
}

// MarkReminderSent marks a requirement's reminder as sent
func (r *MongoRequirementRepository) MarkReminderSent(ctx context.Context, id primitive.ObjectID) error {
	now := time.Now().UTC()
	filter := bson.M{"_id": id}
	update := bson.M{
		"$set": bson.M{
			"reminder_sent_at": now,
			"updated_at":       now,
		},
	}
	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return models.ErrRequirementNotFound
	}
	return nil
}

// ExpireOverdue marks overdue requirements as expired
func (r *MongoRequirementRepository) ExpireOverdue(ctx context.Context) (int64, error) {
	now := time.Now().UTC()
	filter := bson.M{
		"status": bson.M{
			"$in": []models.RequirementStatus{
				models.RequirementStatusPending,
				models.RequirementStatusInProgress,
			},
		},
		"due_date": bson.M{
			"$lt": now,
		},
	}
	update := bson.M{
		"$set": bson.M{
			"status":     models.RequirementStatusExpired,
			"updated_at": now,
		},
		"$push": bson.M{
			"status_history": bson.M{
				"from_status": models.RequirementStatusPending,
				"to_status":   models.RequirementStatusExpired,
				"reason":      "Expired due to passing due date",
				"changed_at":  now,
			},
		},
	}
	result, err := r.collection.UpdateMany(ctx, filter, update)
	if err != nil {
		return 0, err
	}
	return result.ModifiedCount, nil
}

// CountByCompany counts requirements for a company
func (r *MongoRequirementRepository) CountByCompany(ctx context.Context, companyID primitive.ObjectID, status *models.RequirementStatus) (int64, error) {
	filter := bson.M{"company_id": companyID}
	if status != nil {
		filter["status"] = *status
	}
	return r.collection.CountDocuments(ctx, filter)
}

// CountBySupplier counts requirements for a supplier
func (r *MongoRequirementRepository) CountBySupplier(ctx context.Context, supplierID primitive.ObjectID, status *models.RequirementStatus) (int64, error) {
	filter := bson.M{"supplier_id": supplierID}
	if status != nil {
		filter["status"] = *status
	}
	return r.collection.CountDocuments(ctx, filter)
}

// Ensure MongoRequirementRepository implements RequirementRepository
var _ RequirementRepository = (*MongoRequirementRepository)(nil)
