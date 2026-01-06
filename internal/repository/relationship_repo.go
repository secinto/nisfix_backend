package repository

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/checkfix-tools/nisfix_backend/internal/models"
)

// MongoRelationshipRepository implements RelationshipRepository for MongoDB
// #ORM_INTEGRATION: MongoDB driver-based repository implementation
type MongoRelationshipRepository struct {
	collection *mongo.Collection
}

// NewMongoRelationshipRepository creates a new MongoDB relationship repository
func NewMongoRelationshipRepository(db *mongo.Database) *MongoRelationshipRepository {
	return &MongoRelationshipRepository{
		collection: db.Collection(models.CompanySupplierRelationship{}.CollectionName()),
	}
}

// Create creates a new relationship
func (r *MongoRelationshipRepository) Create(ctx context.Context, relationship *models.CompanySupplierRelationship) error {
	relationship.BeforeCreate()
	_, err := r.collection.InsertOne(ctx, relationship)
	if mongo.IsDuplicateKeyError(err) {
		return models.ErrRelationshipExists
	}
	return err
}

// GetByID finds a relationship by ID
func (r *MongoRelationshipRepository) GetByID(ctx context.Context, id primitive.ObjectID) (*models.CompanySupplierRelationship, error) {
	var relationship models.CompanySupplierRelationship
	filter := bson.M{"_id": id}
	err := r.collection.FindOne(ctx, filter).Decode(&relationship)
	if err == mongo.ErrNoDocuments {
		return nil, models.ErrRelationshipNotFound
	}
	if err != nil {
		return nil, err
	}
	return &relationship, nil
}

// GetByCompanyAndSupplier finds a relationship by company and supplier IDs
func (r *MongoRelationshipRepository) GetByCompanyAndSupplier(ctx context.Context, companyID, supplierID primitive.ObjectID) (*models.CompanySupplierRelationship, error) {
	var relationship models.CompanySupplierRelationship
	filter := bson.M{
		"company_id":  companyID,
		"supplier_id": supplierID,
	}
	err := r.collection.FindOne(ctx, filter).Decode(&relationship)
	if err == mongo.ErrNoDocuments {
		return nil, models.ErrRelationshipNotFound
	}
	if err != nil {
		return nil, err
	}
	return &relationship, nil
}

// GetByInvitedEmail finds a pending relationship by invited email
func (r *MongoRelationshipRepository) GetByInvitedEmail(ctx context.Context, email string, companyID primitive.ObjectID) (*models.CompanySupplierRelationship, error) {
	var relationship models.CompanySupplierRelationship
	filter := bson.M{
		"company_id":    companyID,
		"invited_email": email,
		"status":        models.RelationshipStatusPending,
	}
	err := r.collection.FindOne(ctx, filter).Decode(&relationship)
	if err == mongo.ErrNoDocuments {
		return nil, models.ErrRelationshipNotFound
	}
	if err != nil {
		return nil, err
	}
	return &relationship, nil
}

// Update updates a relationship
func (r *MongoRelationshipRepository) Update(ctx context.Context, relationship *models.CompanySupplierRelationship) error {
	relationship.BeforeUpdate()
	filter := bson.M{"_id": relationship.ID}
	update := bson.M{"$set": relationship}
	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return models.ErrRelationshipNotFound
	}
	return nil
}

// ListByCompany lists relationships for a company
// #QUERY_PATTERN: Company dashboard queries by status and classification
func (r *MongoRelationshipRepository) ListByCompany(ctx context.Context, companyID primitive.ObjectID, status *models.RelationshipStatus, classification *models.SupplierClassification, opts PaginationOptions) (*PaginatedResult[models.CompanySupplierRelationship], error) {
	filter := bson.M{"company_id": companyID}
	if status != nil {
		filter["status"] = *status
	}
	if classification != nil {
		filter["classification"] = *classification
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
		SetSort(bson.D{{Key: "classification", Value: 1}, {Key: opts.SortBy, Value: opts.SortDir}})

	cursor, err := r.collection.Find(ctx, filter, findOpts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var relationships []models.CompanySupplierRelationship
	if err := cursor.All(ctx, &relationships); err != nil {
		return nil, err
	}

	totalPages := int(total) / opts.Limit
	if int(total)%opts.Limit > 0 {
		totalPages++
	}

	return &PaginatedResult[models.CompanySupplierRelationship]{
		Items:      relationships,
		TotalCount: total,
		Page:       opts.Page,
		Limit:      opts.Limit,
		TotalPages: totalPages,
	}, nil
}

// ListBySupplier lists relationships for a supplier
func (r *MongoRelationshipRepository) ListBySupplier(ctx context.Context, supplierID primitive.ObjectID, status *models.RelationshipStatus, opts PaginationOptions) (*PaginatedResult[models.CompanySupplierRelationship], error) {
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
		SetSort(bson.D{{Key: opts.SortBy, Value: opts.SortDir}})

	cursor, err := r.collection.Find(ctx, filter, findOpts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var relationships []models.CompanySupplierRelationship
	if err := cursor.All(ctx, &relationships); err != nil {
		return nil, err
	}

	totalPages := int(total) / opts.Limit
	if int(total)%opts.Limit > 0 {
		totalPages++
	}

	return &PaginatedResult[models.CompanySupplierRelationship]{
		Items:      relationships,
		TotalCount: total,
		Page:       opts.Page,
		Limit:      opts.Limit,
		TotalPages: totalPages,
	}, nil
}

// ListPendingByEmail lists pending invitations for an email
// #QUERY_PATTERN: Supplier lookup of pending invitations by email
func (r *MongoRelationshipRepository) ListPendingByEmail(ctx context.Context, email string) ([]models.CompanySupplierRelationship, error) {
	filter := bson.M{
		"invited_email": email,
		"status":        models.RelationshipStatusPending,
	}
	findOpts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}})

	cursor, err := r.collection.Find(ctx, filter, findOpts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var relationships []models.CompanySupplierRelationship
	if err := cursor.All(ctx, &relationships); err != nil {
		return nil, err
	}

	return relationships, nil
}

// CountByCompany counts relationships for a company
func (r *MongoRelationshipRepository) CountByCompany(ctx context.Context, companyID primitive.ObjectID, status *models.RelationshipStatus) (int64, error) {
	filter := bson.M{"company_id": companyID}
	if status != nil {
		filter["status"] = *status
	}
	return r.collection.CountDocuments(ctx, filter)
}

// CountBySupplier counts relationships for a supplier
func (r *MongoRelationshipRepository) CountBySupplier(ctx context.Context, supplierID primitive.ObjectID, status *models.RelationshipStatus) (int64, error) {
	filter := bson.M{"supplier_id": supplierID}
	if status != nil {
		filter["status"] = *status
	}
	return r.collection.CountDocuments(ctx, filter)
}

// Ensure MongoRelationshipRepository implements RelationshipRepository
var _ RelationshipRepository = (*MongoRelationshipRepository)(nil)
