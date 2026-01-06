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

// MongoOrganizationRepository implements OrganizationRepository for MongoDB
// #ORM_INTEGRATION: MongoDB driver-based repository implementation
type MongoOrganizationRepository struct {
	collection *mongo.Collection
}

// NewMongoOrganizationRepository creates a new MongoDB organization repository
func NewMongoOrganizationRepository(db *mongo.Database) *MongoOrganizationRepository {
	return &MongoOrganizationRepository{
		collection: db.Collection(models.Organization{}.CollectionName()),
	}
}

// Create creates a new organization
func (r *MongoOrganizationRepository) Create(ctx context.Context, org *models.Organization) error {
	org.BeforeCreate()
	_, err := r.collection.InsertOne(ctx, org)
	if mongo.IsDuplicateKeyError(err) {
		return models.ErrAlreadyExists
	}
	return err
}

// GetByID finds an organization by ID
func (r *MongoOrganizationRepository) GetByID(ctx context.Context, id primitive.ObjectID) (*models.Organization, error) {
	var org models.Organization
	filter := bson.M{
		"_id":        id,
		"deleted_at": nil,
	}
	err := r.collection.FindOne(ctx, filter).Decode(&org)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, models.ErrOrganizationNotFound
	}
	if err != nil {
		return nil, err
	}
	return &org, nil
}

// GetBySlug finds an organization by slug
func (r *MongoOrganizationRepository) GetBySlug(ctx context.Context, slug string) (*models.Organization, error) {
	var org models.Organization
	filter := bson.M{
		"slug":       slug,
		"deleted_at": nil,
	}
	err := r.collection.FindOne(ctx, filter).Decode(&org)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, models.ErrOrganizationNotFound
	}
	if err != nil {
		return nil, err
	}
	return &org, nil
}

// GetByDomain finds an organization by domain
func (r *MongoOrganizationRepository) GetByDomain(ctx context.Context, domain string) (*models.Organization, error) {
	var org models.Organization
	filter := bson.M{
		"domain":     domain,
		"deleted_at": nil,
	}
	err := r.collection.FindOne(ctx, filter).Decode(&org)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, models.ErrOrganizationNotFound
	}
	if err != nil {
		return nil, err
	}
	return &org, nil
}

// Update updates an organization
func (r *MongoOrganizationRepository) Update(ctx context.Context, org *models.Organization) error {
	org.BeforeUpdate()
	filter := bson.M{
		"_id":        org.ID,
		"deleted_at": nil,
	}
	update := bson.M{"$set": org}
	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return models.ErrAlreadyExists
		}
		return err
	}
	if result.MatchedCount == 0 {
		return models.ErrOrganizationNotFound
	}
	return nil
}

// SoftDelete soft deletes an organization
func (r *MongoOrganizationRepository) SoftDelete(ctx context.Context, id primitive.ObjectID) error {
	now := time.Now().UTC()
	filter := bson.M{
		"_id":        id,
		"deleted_at": nil,
	}
	update := bson.M{
		"$set": bson.M{
			"deleted_at": now,
			"updated_at": now,
		},
	}
	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return models.ErrOrganizationNotFound
	}
	return nil
}

// List lists organizations with filtering and pagination
func (r *MongoOrganizationRepository) List(ctx context.Context, orgType *models.OrganizationType, opts PaginationOptions) (*PaginatedResult[models.Organization], error) {
	filter := bson.M{"deleted_at": nil}
	if orgType != nil {
		filter["type"] = *orgType
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
	defer cursor.Close(ctx) //nolint:errcheck // defer close

	var orgs []models.Organization
	if err := cursor.All(ctx, &orgs); err != nil {
		return nil, err
	}

	totalPages := int(total) / opts.Limit
	if int(total)%opts.Limit > 0 {
		totalPages++
	}

	return &PaginatedResult[models.Organization]{
		Items:      orgs,
		TotalCount: total,
		Page:       opts.Page,
		Limit:      opts.Limit,
		TotalPages: totalPages,
	}, nil
}

// Ensure MongoOrganizationRepository implements OrganizationRepository
var _ OrganizationRepository = (*MongoOrganizationRepository)(nil)
