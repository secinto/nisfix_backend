package repository

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/checkfix-tools/nisfix_backend/internal/models"
)

// MongoResponseRepository implements ResponseRepository for MongoDB
// #ORM_INTEGRATION: MongoDB driver-based repository implementation
type MongoResponseRepository struct {
	collection *mongo.Collection
}

// NewMongoResponseRepository creates a new MongoDB response repository
func NewMongoResponseRepository(db *mongo.Database) *MongoResponseRepository {
	return &MongoResponseRepository{
		collection: db.Collection(models.SupplierResponse{}.CollectionName()),
	}
}

// Create creates a new response
func (r *MongoResponseRepository) Create(ctx context.Context, response *models.SupplierResponse) error {
	response.BeforeCreate()
	_, err := r.collection.InsertOne(ctx, response)
	if mongo.IsDuplicateKeyError(err) {
		return models.ErrResponseAlreadyExists
	}
	return err
}

// GetByID finds a response by ID
func (r *MongoResponseRepository) GetByID(ctx context.Context, id primitive.ObjectID) (*models.SupplierResponse, error) {
	var response models.SupplierResponse
	filter := bson.M{"_id": id}
	err := r.collection.FindOne(ctx, filter).Decode(&response)
	if err == mongo.ErrNoDocuments {
		return nil, models.ErrResponseNotFound
	}
	if err != nil {
		return nil, err
	}
	return &response, nil
}

// GetByRequirement finds a response by requirement ID
func (r *MongoResponseRepository) GetByRequirement(ctx context.Context, requirementID primitive.ObjectID) (*models.SupplierResponse, error) {
	var response models.SupplierResponse
	filter := bson.M{"requirement_id": requirementID}
	err := r.collection.FindOne(ctx, filter).Decode(&response)
	if err == mongo.ErrNoDocuments {
		return nil, models.ErrResponseNotFound
	}
	if err != nil {
		return nil, err
	}
	return &response, nil
}

// Update updates a response
func (r *MongoResponseRepository) Update(ctx context.Context, response *models.SupplierResponse) error {
	response.BeforeUpdate()
	filter := bson.M{"_id": response.ID}
	update := bson.M{"$set": response}
	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return models.ErrResponseNotFound
	}
	return nil
}

// SaveDraftAnswer saves a draft answer
func (r *MongoResponseRepository) SaveDraftAnswer(ctx context.Context, responseID primitive.ObjectID, answer models.DraftAnswer) error {
	now := time.Now().UTC()
	answer.SavedAt = now

	// First try to update existing answer
	filter := bson.M{
		"_id":                       responseID,
		"draft_answers.question_id": answer.QuestionID,
	}
	update := bson.M{
		"$set": bson.M{
			"draft_answers.$": answer,
			"updated_at":      now,
		},
	}
	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}

	// If no existing answer found, push new one
	if result.MatchedCount == 0 {
		filter = bson.M{"_id": responseID}
		update = bson.M{
			"$push": bson.M{"draft_answers": answer},
			"$set":  bson.M{"updated_at": now},
		}
		result, err = r.collection.UpdateOne(ctx, filter, update)
		if err != nil {
			return err
		}
		if result.MatchedCount == 0 {
			return models.ErrResponseNotFound
		}
	}

	return nil
}

// ListBySupplier lists responses for a supplier
func (r *MongoResponseRepository) ListBySupplier(ctx context.Context, supplierID primitive.ObjectID, opts PaginationOptions) (*PaginatedResult[models.SupplierResponse], error) {
	filter := bson.M{"supplier_id": supplierID}

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

	var responses []models.SupplierResponse
	if err := cursor.All(ctx, &responses); err != nil {
		return nil, err
	}

	totalPages := int(total) / opts.Limit
	if int(total)%opts.Limit > 0 {
		totalPages++
	}

	return &PaginatedResult[models.SupplierResponse]{
		Items:      responses,
		TotalCount: total,
		Page:       opts.Page,
		Limit:      opts.Limit,
		TotalPages: totalPages,
	}, nil
}

// CountBySupplier counts responses for a supplier
func (r *MongoResponseRepository) CountBySupplier(ctx context.Context, supplierID primitive.ObjectID) (int64, error) {
	filter := bson.M{"supplier_id": supplierID}
	return r.collection.CountDocuments(ctx, filter)
}

// Ensure MongoResponseRepository implements ResponseRepository
var _ ResponseRepository = (*MongoResponseRepository)(nil)

// MongoSubmissionRepository implements SubmissionRepository for MongoDB
// #ORM_INTEGRATION: MongoDB driver-based repository implementation
type MongoSubmissionRepository struct {
	collection *mongo.Collection
}

// NewMongoSubmissionRepository creates a new MongoDB submission repository
func NewMongoSubmissionRepository(db *mongo.Database) *MongoSubmissionRepository {
	return &MongoSubmissionRepository{
		collection: db.Collection(models.QuestionnaireSubmission{}.CollectionName()),
	}
}

// Create creates a new submission
func (r *MongoSubmissionRepository) Create(ctx context.Context, submission *models.QuestionnaireSubmission) error {
	submission.BeforeCreate()
	_, err := r.collection.InsertOne(ctx, submission)
	if mongo.IsDuplicateKeyError(err) {
		return models.ErrSubmissionAlreadyExists
	}
	return err
}

// GetByID finds a submission by ID
func (r *MongoSubmissionRepository) GetByID(ctx context.Context, id primitive.ObjectID) (*models.QuestionnaireSubmission, error) {
	var submission models.QuestionnaireSubmission
	filter := bson.M{"_id": id}
	err := r.collection.FindOne(ctx, filter).Decode(&submission)
	if err == mongo.ErrNoDocuments {
		return nil, models.ErrSubmissionNotFound
	}
	if err != nil {
		return nil, err
	}
	return &submission, nil
}

// GetByResponse finds a submission by response ID
func (r *MongoSubmissionRepository) GetByResponse(ctx context.Context, responseID primitive.ObjectID) (*models.QuestionnaireSubmission, error) {
	var submission models.QuestionnaireSubmission
	filter := bson.M{"response_id": responseID}
	err := r.collection.FindOne(ctx, filter).Decode(&submission)
	if err == mongo.ErrNoDocuments {
		return nil, models.ErrSubmissionNotFound
	}
	if err != nil {
		return nil, err
	}
	return &submission, nil
}

// ListByQuestionnaire lists submissions for a questionnaire
func (r *MongoSubmissionRepository) ListByQuestionnaire(ctx context.Context, questionnaireID primitive.ObjectID, opts PaginationOptions) (*PaginatedResult[models.QuestionnaireSubmission], error) {
	filter := bson.M{"questionnaire_id": questionnaireID}

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

	var submissions []models.QuestionnaireSubmission
	if err := cursor.All(ctx, &submissions); err != nil {
		return nil, err
	}

	totalPages := int(total) / opts.Limit
	if int(total)%opts.Limit > 0 {
		totalPages++
	}

	return &PaginatedResult[models.QuestionnaireSubmission]{
		Items:      submissions,
		TotalCount: total,
		Page:       opts.Page,
		Limit:      opts.Limit,
		TotalPages: totalPages,
	}, nil
}

// GetPassRateByQuestionnaire calculates pass rate for a questionnaire
func (r *MongoSubmissionRepository) GetPassRateByQuestionnaire(ctx context.Context, questionnaireID primitive.ObjectID) (float64, error) {
	pipeline := []bson.M{
		{
			"$match": bson.M{
				"questionnaire_id": questionnaireID,
				"submitted_at":     bson.M{"$ne": nil},
			},
		},
		{
			"$group": bson.M{
				"_id":    nil,
				"total":  bson.M{"$sum": 1},
				"passed": bson.M{"$sum": bson.M{"$cond": []interface{}{"$passed", 1, 0}}},
			},
		},
	}

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return 0, err
	}
	defer cursor.Close(ctx)

	var result struct {
		Total  int `bson:"total"`
		Passed int `bson:"passed"`
	}
	if cursor.Next(ctx) {
		if err := cursor.Decode(&result); err != nil {
			return 0, err
		}
		if result.Total == 0 {
			return 0, nil
		}
		return float64(result.Passed) / float64(result.Total) * 100, nil
	}

	return 0, nil
}

// Ensure MongoSubmissionRepository implements SubmissionRepository
var _ SubmissionRepository = (*MongoSubmissionRepository)(nil)

// MongoVerificationRepository implements VerificationRepository for MongoDB
// #ORM_INTEGRATION: MongoDB driver-based repository implementation
type MongoVerificationRepository struct {
	collection *mongo.Collection
}

// NewMongoVerificationRepository creates a new MongoDB verification repository
func NewMongoVerificationRepository(db *mongo.Database) *MongoVerificationRepository {
	return &MongoVerificationRepository{
		collection: db.Collection(models.CheckFixVerification{}.CollectionName()),
	}
}

// Create creates a new verification
func (r *MongoVerificationRepository) Create(ctx context.Context, verification *models.CheckFixVerification) error {
	verification.BeforeCreate()
	_, err := r.collection.InsertOne(ctx, verification)
	if mongo.IsDuplicateKeyError(err) {
		return models.ErrAlreadyExists
	}
	return err
}

// GetByID finds a verification by ID
func (r *MongoVerificationRepository) GetByID(ctx context.Context, id primitive.ObjectID) (*models.CheckFixVerification, error) {
	var verification models.CheckFixVerification
	filter := bson.M{"_id": id}
	err := r.collection.FindOne(ctx, filter).Decode(&verification)
	if err == mongo.ErrNoDocuments {
		return nil, models.ErrVerificationNotFound
	}
	if err != nil {
		return nil, err
	}
	return &verification, nil
}

// GetByResponse finds a verification by response ID
func (r *MongoVerificationRepository) GetByResponse(ctx context.Context, responseID primitive.ObjectID) (*models.CheckFixVerification, error) {
	var verification models.CheckFixVerification
	filter := bson.M{"response_id": responseID}
	err := r.collection.FindOne(ctx, filter).Decode(&verification)
	if err == mongo.ErrNoDocuments {
		return nil, models.ErrVerificationNotFound
	}
	if err != nil {
		return nil, err
	}
	return &verification, nil
}

// GetLatestBySupplier finds the latest verification for a supplier
func (r *MongoVerificationRepository) GetLatestBySupplier(ctx context.Context, supplierID primitive.ObjectID) (*models.CheckFixVerification, error) {
	var verification models.CheckFixVerification
	filter := bson.M{"supplier_id": supplierID}
	findOpts := options.FindOne().SetSort(bson.D{{Key: "verified_at", Value: -1}})
	err := r.collection.FindOne(ctx, filter, findOpts).Decode(&verification)
	if err == mongo.ErrNoDocuments {
		return nil, models.ErrVerificationNotFound
	}
	if err != nil {
		return nil, err
	}
	return &verification, nil
}

// Update updates a verification
func (r *MongoVerificationRepository) Update(ctx context.Context, verification *models.CheckFixVerification) error {
	verification.BeforeUpdate()
	filter := bson.M{"_id": verification.ID}
	update := bson.M{"$set": verification}
	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return models.ErrVerificationNotFound
	}
	return nil
}

// ListExpiringVerifications lists verifications that are about to expire
func (r *MongoVerificationRepository) ListExpiringVerifications(ctx context.Context, daysBeforeExpiry int) ([]models.CheckFixVerification, error) {
	expiryDate := time.Now().UTC().AddDate(0, 0, daysBeforeExpiry)
	filter := bson.M{
		"expires_at": bson.M{
			"$lte": expiryDate,
			"$gte": time.Now().UTC(),
		},
		"verification_valid": true,
	}

	cursor, err := r.collection.Find(ctx, filter)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var verifications []models.CheckFixVerification
	if err := cursor.All(ctx, &verifications); err != nil {
		return nil, err
	}

	return verifications, nil
}

// Ensure MongoVerificationRepository implements VerificationRepository
var _ VerificationRepository = (*MongoVerificationRepository)(nil)
