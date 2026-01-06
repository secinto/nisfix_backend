package repository

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/checkfix-tools/nisfix_backend/internal/models"
)

// MongoQuestionnaireTemplateRepository implements QuestionnaireTemplateRepository for MongoDB
// #ORM_INTEGRATION: MongoDB driver-based repository implementation
type MongoQuestionnaireTemplateRepository struct {
	collection *mongo.Collection
}

// NewMongoQuestionnaireTemplateRepository creates a new MongoDB questionnaire template repository
func NewMongoQuestionnaireTemplateRepository(db *mongo.Database) *MongoQuestionnaireTemplateRepository {
	return &MongoQuestionnaireTemplateRepository{
		collection: db.Collection(models.QuestionnaireTemplate{}.CollectionName()),
	}
}

// Create creates a new template
func (r *MongoQuestionnaireTemplateRepository) Create(ctx context.Context, template *models.QuestionnaireTemplate) error {
	template.BeforeCreate()
	_, err := r.collection.InsertOne(ctx, template)
	return err
}

// GetByID finds a template by ID
func (r *MongoQuestionnaireTemplateRepository) GetByID(ctx context.Context, id primitive.ObjectID) (*models.QuestionnaireTemplate, error) {
	var template models.QuestionnaireTemplate
	filter := bson.M{"_id": id}
	err := r.collection.FindOne(ctx, filter).Decode(&template)
	if err == mongo.ErrNoDocuments {
		return nil, models.ErrTemplateNotFound
	}
	if err != nil {
		return nil, err
	}
	return &template, nil
}

// Update updates a template
func (r *MongoQuestionnaireTemplateRepository) Update(ctx context.Context, template *models.QuestionnaireTemplate) error {
	template.BeforeUpdate()
	filter := bson.M{"_id": template.ID}
	update := bson.M{"$set": template}
	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return models.ErrTemplateNotFound
	}
	return nil
}

// Delete deletes a template
func (r *MongoQuestionnaireTemplateRepository) Delete(ctx context.Context, id primitive.ObjectID) error {
	filter := bson.M{"_id": id}
	result, err := r.collection.DeleteOne(ctx, filter)
	if err != nil {
		return err
	}
	if result.DeletedCount == 0 {
		return models.ErrTemplateNotFound
	}
	return nil
}

// IncrementUsageCount increments the usage count
func (r *MongoQuestionnaireTemplateRepository) IncrementUsageCount(ctx context.Context, id primitive.ObjectID) error {
	filter := bson.M{"_id": id}
	update := bson.M{
		"$inc": bson.M{"usage_count": 1},
	}
	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return models.ErrTemplateNotFound
	}
	return nil
}

// ListSystemTemplates lists all system templates
func (r *MongoQuestionnaireTemplateRepository) ListSystemTemplates(ctx context.Context, category *models.TemplateCategory) ([]models.QuestionnaireTemplate, error) {
	filter := bson.M{"is_system": true}
	if category != nil {
		filter["category"] = *category
	}

	findOpts := options.Find().SetSort(bson.D{{Key: "category", Value: 1}, {Key: "name", Value: 1}})

	cursor, err := r.collection.Find(ctx, filter, findOpts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var templates []models.QuestionnaireTemplate
	if err := cursor.All(ctx, &templates); err != nil {
		return nil, err
	}

	return templates, nil
}

// ListByOrganization lists templates created by an organization
func (r *MongoQuestionnaireTemplateRepository) ListByOrganization(ctx context.Context, orgID primitive.ObjectID, opts PaginationOptions) (*PaginatedResult[models.QuestionnaireTemplate], error) {
	filter := bson.M{"created_by_org_id": orgID}

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

	var templates []models.QuestionnaireTemplate
	if err := cursor.All(ctx, &templates); err != nil {
		return nil, err
	}

	totalPages := int(total) / opts.Limit
	if int(total)%opts.Limit > 0 {
		totalPages++
	}

	return &PaginatedResult[models.QuestionnaireTemplate]{
		Items:      templates,
		TotalCount: total,
		Page:       opts.Page,
		Limit:      opts.Limit,
		TotalPages: totalPages,
	}, nil
}

// SearchTemplates searches templates by name/description
func (r *MongoQuestionnaireTemplateRepository) SearchTemplates(ctx context.Context, query string, opts PaginationOptions) (*PaginatedResult[models.QuestionnaireTemplate], error) {
	filter := bson.M{
		"$text": bson.M{"$search": query},
	}

	// Count total
	total, err := r.collection.CountDocuments(ctx, filter)
	if err != nil {
		return nil, err
	}

	// Apply pagination with text score sorting
	skip := int64((opts.Page - 1) * opts.Limit)
	findOpts := options.Find().
		SetSkip(skip).
		SetLimit(int64(opts.Limit)).
		SetSort(bson.D{{Key: "score", Value: bson.M{"$meta": "textScore"}}}).
		SetProjection(bson.M{"score": bson.M{"$meta": "textScore"}})

	cursor, err := r.collection.Find(ctx, filter, findOpts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var templates []models.QuestionnaireTemplate
	if err := cursor.All(ctx, &templates); err != nil {
		return nil, err
	}

	totalPages := int(total) / opts.Limit
	if int(total)%opts.Limit > 0 {
		totalPages++
	}

	return &PaginatedResult[models.QuestionnaireTemplate]{
		Items:      templates,
		TotalCount: total,
		Page:       opts.Page,
		Limit:      opts.Limit,
		TotalPages: totalPages,
	}, nil
}

// Ensure MongoQuestionnaireTemplateRepository implements QuestionnaireTemplateRepository
var _ QuestionnaireTemplateRepository = (*MongoQuestionnaireTemplateRepository)(nil)

// MongoQuestionnaireRepository implements QuestionnaireRepository for MongoDB
// #ORM_INTEGRATION: MongoDB driver-based repository implementation
type MongoQuestionnaireRepository struct {
	collection *mongo.Collection
}

// NewMongoQuestionnaireRepository creates a new MongoDB questionnaire repository
func NewMongoQuestionnaireRepository(db *mongo.Database) *MongoQuestionnaireRepository {
	return &MongoQuestionnaireRepository{
		collection: db.Collection(models.Questionnaire{}.CollectionName()),
	}
}

// Create creates a new questionnaire
func (r *MongoQuestionnaireRepository) Create(ctx context.Context, questionnaire *models.Questionnaire) error {
	questionnaire.BeforeCreate()
	_, err := r.collection.InsertOne(ctx, questionnaire)
	return err
}

// GetByID finds a questionnaire by ID
func (r *MongoQuestionnaireRepository) GetByID(ctx context.Context, id primitive.ObjectID) (*models.Questionnaire, error) {
	var questionnaire models.Questionnaire
	filter := bson.M{"_id": id}
	err := r.collection.FindOne(ctx, filter).Decode(&questionnaire)
	if err == mongo.ErrNoDocuments {
		return nil, models.ErrQuestionnaireNotFound
	}
	if err != nil {
		return nil, err
	}
	return &questionnaire, nil
}

// Update updates a questionnaire
func (r *MongoQuestionnaireRepository) Update(ctx context.Context, questionnaire *models.Questionnaire) error {
	questionnaire.BeforeUpdate()
	filter := bson.M{"_id": questionnaire.ID}
	update := bson.M{"$set": questionnaire}
	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return models.ErrQuestionnaireNotFound
	}
	return nil
}

// Delete deletes a questionnaire (draft only)
func (r *MongoQuestionnaireRepository) Delete(ctx context.Context, id primitive.ObjectID) error {
	// Only allow deleting draft questionnaires
	filter := bson.M{
		"_id":    id,
		"status": models.QuestionnaireStatusDraft,
	}
	result, err := r.collection.DeleteOne(ctx, filter)
	if err != nil {
		return err
	}
	if result.DeletedCount == 0 {
		return models.ErrQuestionnaireNotDeletable
	}
	return nil
}

// UpdateStatistics updates question count and max score
func (r *MongoQuestionnaireRepository) UpdateStatistics(ctx context.Context, id primitive.ObjectID, questionCount, maxScore int) error {
	filter := bson.M{"_id": id}
	update := bson.M{
		"$set": bson.M{
			"question_count":     questionCount,
			"max_possible_score": maxScore,
		},
	}
	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return models.ErrQuestionnaireNotFound
	}
	return nil
}

// ListByCompany lists questionnaires for a company
func (r *MongoQuestionnaireRepository) ListByCompany(ctx context.Context, companyID primitive.ObjectID, status *models.QuestionnaireStatus, opts PaginationOptions) (*PaginatedResult[models.Questionnaire], error) {
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
		SetSort(bson.D{{Key: opts.SortBy, Value: opts.SortDir}})

	cursor, err := r.collection.Find(ctx, filter, findOpts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var questionnaires []models.Questionnaire
	if err := cursor.All(ctx, &questionnaires); err != nil {
		return nil, err
	}

	totalPages := int(total) / opts.Limit
	if int(total)%opts.Limit > 0 {
		totalPages++
	}

	return &PaginatedResult[models.Questionnaire]{
		Items:      questionnaires,
		TotalCount: total,
		Page:       opts.Page,
		Limit:      opts.Limit,
		TotalPages: totalPages,
	}, nil
}

// CountByCompany counts questionnaires for a company
func (r *MongoQuestionnaireRepository) CountByCompany(ctx context.Context, companyID primitive.ObjectID, status *models.QuestionnaireStatus) (int64, error) {
	filter := bson.M{"company_id": companyID}
	if status != nil {
		filter["status"] = *status
	}
	return r.collection.CountDocuments(ctx, filter)
}

// Ensure MongoQuestionnaireRepository implements QuestionnaireRepository
var _ QuestionnaireRepository = (*MongoQuestionnaireRepository)(nil)
