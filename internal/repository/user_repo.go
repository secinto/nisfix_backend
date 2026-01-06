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

// MongoUserRepository implements UserRepository for MongoDB
// #ORM_INTEGRATION: MongoDB driver-based repository implementation
type MongoUserRepository struct {
	collection *mongo.Collection
}

// NewMongoUserRepository creates a new MongoDB user repository
func NewMongoUserRepository(db *mongo.Database) *MongoUserRepository {
	return &MongoUserRepository{
		collection: db.Collection(models.User{}.CollectionName()),
	}
}

// Create creates a new user
func (r *MongoUserRepository) Create(ctx context.Context, user *models.User) error {
	user.BeforeCreate()
	_, err := r.collection.InsertOne(ctx, user)
	if mongo.IsDuplicateKeyError(err) {
		return models.ErrEmailAlreadyExists
	}
	return err
}

// GetByID finds a user by ID
func (r *MongoUserRepository) GetByID(ctx context.Context, id primitive.ObjectID) (*models.User, error) {
	var user models.User
	filter := bson.M{
		"_id":        id,
		"deleted_at": nil,
	}
	err := r.collection.FindOne(ctx, filter).Decode(&user)
	if err == mongo.ErrNoDocuments {
		return nil, models.ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// GetByEmail finds a user by email
func (r *MongoUserRepository) GetByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	filter := bson.M{
		"email":      email,
		"deleted_at": nil,
	}
	err := r.collection.FindOne(ctx, filter).Decode(&user)
	if err == mongo.ErrNoDocuments {
		return nil, models.ErrUserNotFound
	}
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// Update updates a user
func (r *MongoUserRepository) Update(ctx context.Context, user *models.User) error {
	user.BeforeUpdate()
	filter := bson.M{
		"_id":        user.ID,
		"deleted_at": nil,
	}
	update := bson.M{"$set": user}
	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return models.ErrEmailAlreadyExists
		}
		return err
	}
	if result.MatchedCount == 0 {
		return models.ErrUserNotFound
	}
	return nil
}

// SoftDelete soft deletes a user
func (r *MongoUserRepository) SoftDelete(ctx context.Context, id primitive.ObjectID) error {
	now := time.Now().UTC()
	filter := bson.M{
		"_id":        id,
		"deleted_at": nil,
	}
	update := bson.M{
		"$set": bson.M{
			"deleted_at": now,
			"updated_at": now,
			"is_active":  false,
		},
	}
	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return models.ErrUserNotFound
	}
	return nil
}

// UpdateLastLogin updates the last login timestamp
func (r *MongoUserRepository) UpdateLastLogin(ctx context.Context, id primitive.ObjectID) error {
	now := time.Now().UTC()
	filter := bson.M{
		"_id":        id,
		"deleted_at": nil,
	}
	update := bson.M{
		"$set": bson.M{
			"last_login_at": now,
			"updated_at":    now,
		},
	}
	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return models.ErrUserNotFound
	}
	return nil
}

// ListByOrganization lists users in an organization
func (r *MongoUserRepository) ListByOrganization(ctx context.Context, orgID primitive.ObjectID, includeInactive bool, opts PaginationOptions) (*PaginatedResult[models.User], error) {
	filter := bson.M{
		"organization_id": orgID,
		"deleted_at":      nil,
	}
	if !includeInactive {
		filter["is_active"] = true
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

	var users []models.User
	if err := cursor.All(ctx, &users); err != nil {
		return nil, err
	}

	totalPages := int(total) / opts.Limit
	if int(total)%opts.Limit > 0 {
		totalPages++
	}

	return &PaginatedResult[models.User]{
		Items:      users,
		TotalCount: total,
		Page:       opts.Page,
		Limit:      opts.Limit,
		TotalPages: totalPages,
	}, nil
}

// CountByOrganization counts users in an organization
func (r *MongoUserRepository) CountByOrganization(ctx context.Context, orgID primitive.ObjectID) (int64, error) {
	filter := bson.M{
		"organization_id": orgID,
		"deleted_at":      nil,
		"is_active":       true,
	}
	return r.collection.CountDocuments(ctx, filter)
}

// Ensure MongoUserRepository implements UserRepository
var _ UserRepository = (*MongoUserRepository)(nil)
