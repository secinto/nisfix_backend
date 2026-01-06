package repository

import (
	"context"
	"errors"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"

	"github.com/checkfix-tools/nisfix_backend/internal/models"
)

// MongoSecureLinkRepository implements SecureLinkRepository for MongoDB
// #ORM_INTEGRATION: MongoDB driver-based repository implementation
type MongoSecureLinkRepository struct {
	collection *mongo.Collection
}

// NewMongoSecureLinkRepository creates a new MongoDB secure link repository
func NewMongoSecureLinkRepository(db *mongo.Database) *MongoSecureLinkRepository {
	return &MongoSecureLinkRepository{
		collection: db.Collection(models.SecureLink{}.CollectionName()),
	}
}

// Create creates a new secure link
func (r *MongoSecureLinkRepository) Create(ctx context.Context, link *models.SecureLink) error {
	link.BeforeCreate()
	_, err := r.collection.InsertOne(ctx, link)
	if mongo.IsDuplicateKeyError(err) {
		return models.ErrAlreadyExists
	}
	return err
}

// GetByIdentifier finds a secure link by its identifier
func (r *MongoSecureLinkRepository) GetByIdentifier(ctx context.Context, identifier string) (*models.SecureLink, error) {
	var link models.SecureLink
	filter := bson.M{
		"secure_identifier": identifier,
	}
	err := r.collection.FindOne(ctx, filter).Decode(&link)
	if errors.Is(err, mongo.ErrNoDocuments) {
		return nil, models.ErrSecureLinkNotFound
	}
	if err != nil {
		return nil, err
	}

	// Check validity
	if !link.CanBeUsed() {
		if link.IsExpired() {
			return nil, models.ErrSecureLinkExpired
		}
		if link.IsUsed() {
			return nil, models.ErrSecureLinkUsed
		}
		return nil, models.ErrSecureLinkInvalid
	}

	return &link, nil
}

// MarkAsUsed marks a secure link as used
func (r *MongoSecureLinkRepository) MarkAsUsed(ctx context.Context, id primitive.ObjectID) error {
	now := time.Now().UTC()
	filter := bson.M{
		"_id":      id,
		"is_valid": true,
		"used_at":  nil,
	}
	update := bson.M{
		"$set": bson.M{
			"used_at":  now,
			"is_valid": false,
		},
	}
	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return models.ErrSecureLinkNotFound
	}
	return nil
}

// Invalidate invalidates a secure link
func (r *MongoSecureLinkRepository) Invalidate(ctx context.Context, id primitive.ObjectID) error {
	filter := bson.M{
		"_id":      id,
		"is_valid": true,
	}
	update := bson.M{
		"$set": bson.M{
			"is_valid": false,
		},
	}
	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return models.ErrSecureLinkNotFound
	}
	return nil
}

// InvalidateAllForEmail invalidates all links for an email
func (r *MongoSecureLinkRepository) InvalidateAllForEmail(ctx context.Context, email string) error {
	filter := bson.M{
		"email":    email,
		"is_valid": true,
	}
	update := bson.M{
		"$set": bson.M{
			"is_valid": false,
		},
	}
	_, err := r.collection.UpdateMany(ctx, filter, update)
	return err
}

// CountRecentByEmail counts recent links for rate limiting
// #INDEX_STRATEGY: Email index for rate limiting (max 3 links per hour)
func (r *MongoSecureLinkRepository) CountRecentByEmail(ctx context.Context, email string, withinMinutes int) (int64, error) {
	since := time.Now().UTC().Add(-time.Duration(withinMinutes) * time.Minute)
	filter := bson.M{
		"email": email,
		"created_at": bson.M{
			"$gte": since,
		},
	}
	return r.collection.CountDocuments(ctx, filter)
}

// DeleteExpired deletes expired links (TTL fallback)
// #INDEX_STRATEGY: TTL index automatically removes expired tokens
// This is a fallback for manual cleanup if needed
func (r *MongoSecureLinkRepository) DeleteExpired(ctx context.Context) (int64, error) {
	filter := bson.M{
		"expires_at": bson.M{
			"$lt": time.Now().UTC(),
		},
	}
	result, err := r.collection.DeleteMany(ctx, filter)
	if err != nil {
		return 0, err
	}
	return result.DeletedCount, nil
}

// Ensure MongoSecureLinkRepository implements SecureLinkRepository
var _ SecureLinkRepository = (*MongoSecureLinkRepository)(nil)
