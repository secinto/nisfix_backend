package repository

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/checkfix-tools/nisfix_backend/internal/models"
)

// MongoQuestionRepository implements QuestionRepository for MongoDB
// #ORM_INTEGRATION: MongoDB driver-based repository implementation
type MongoQuestionRepository struct {
	collection *mongo.Collection
}

// NewMongoQuestionRepository creates a new MongoDB question repository
func NewMongoQuestionRepository(db *mongo.Database) *MongoQuestionRepository {
	return &MongoQuestionRepository{
		collection: db.Collection(models.Question{}.CollectionName()),
	}
}

// Create creates a new question
func (r *MongoQuestionRepository) Create(ctx context.Context, question *models.Question) error {
	question.BeforeCreate()
	_, err := r.collection.InsertOne(ctx, question)
	return err
}

// GetByID finds a question by ID
func (r *MongoQuestionRepository) GetByID(ctx context.Context, id primitive.ObjectID) (*models.Question, error) {
	var question models.Question
	filter := bson.M{"_id": id}
	err := r.collection.FindOne(ctx, filter).Decode(&question)
	if err == mongo.ErrNoDocuments {
		return nil, models.ErrQuestionNotFound
	}
	if err != nil {
		return nil, err
	}
	return &question, nil
}

// Update updates a question
func (r *MongoQuestionRepository) Update(ctx context.Context, question *models.Question) error {
	question.BeforeUpdate()
	filter := bson.M{"_id": question.ID}
	update := bson.M{"$set": question}
	result, err := r.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return err
	}
	if result.MatchedCount == 0 {
		return models.ErrQuestionNotFound
	}
	return nil
}

// Delete deletes a question
func (r *MongoQuestionRepository) Delete(ctx context.Context, id primitive.ObjectID) error {
	filter := bson.M{"_id": id}
	result, err := r.collection.DeleteOne(ctx, filter)
	if err != nil {
		return err
	}
	if result.DeletedCount == 0 {
		return models.ErrQuestionNotFound
	}
	return nil
}

// ListByQuestionnaire lists all questions for a questionnaire
// #QUERY_PATTERN: Fetch all questions for a questionnaire at once, sorted by order
func (r *MongoQuestionRepository) ListByQuestionnaire(ctx context.Context, questionnaireID primitive.ObjectID) ([]models.Question, error) {
	filter := bson.M{"questionnaire_id": questionnaireID}
	findOpts := options.Find().SetSort(bson.D{{Key: "topic_id", Value: 1}, {Key: "order", Value: 1}})

	cursor, err := r.collection.Find(ctx, filter, findOpts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var questions []models.Question
	if err := cursor.All(ctx, &questions); err != nil {
		return nil, err
	}

	return questions, nil
}

// ListByQuestionnaireAndTopic lists questions for a specific topic
func (r *MongoQuestionRepository) ListByQuestionnaireAndTopic(ctx context.Context, questionnaireID primitive.ObjectID, topicID string) ([]models.Question, error) {
	filter := bson.M{
		"questionnaire_id": questionnaireID,
		"topic_id":         topicID,
	}
	findOpts := options.Find().SetSort(bson.D{{Key: "order", Value: 1}})

	cursor, err := r.collection.Find(ctx, filter, findOpts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var questions []models.Question
	if err := cursor.All(ctx, &questions); err != nil {
		return nil, err
	}

	return questions, nil
}

// DeleteByQuestionnaire deletes all questions for a questionnaire
// #CASCADE_STRATEGY: CASCADE DELETE - questions deleted with questionnaire
func (r *MongoQuestionRepository) DeleteByQuestionnaire(ctx context.Context, questionnaireID primitive.ObjectID) (int64, error) {
	filter := bson.M{"questionnaire_id": questionnaireID}
	result, err := r.collection.DeleteMany(ctx, filter)
	if err != nil {
		return 0, err
	}
	return result.DeletedCount, nil
}

// UpdateOrder updates the order of questions
func (r *MongoQuestionRepository) UpdateOrder(ctx context.Context, questionnaireID primitive.ObjectID, orders map[primitive.ObjectID]int) error {
	// Use bulk write for efficiency
	var operations []mongo.WriteModel
	for questionID, order := range orders {
		filter := bson.M{
			"_id":              questionID,
			"questionnaire_id": questionnaireID,
		}
		update := bson.M{"$set": bson.M{"order": order}}
		operations = append(operations, mongo.NewUpdateOneModel().SetFilter(filter).SetUpdate(update))
	}

	if len(operations) == 0 {
		return nil
	}

	_, err := r.collection.BulkWrite(ctx, operations)
	return err
}

// CalculateMaxScore calculates the max possible score for a questionnaire
func (r *MongoQuestionRepository) CalculateMaxScore(ctx context.Context, questionnaireID primitive.ObjectID) (int, error) {
	pipeline := []bson.M{
		{
			"$match": bson.M{"questionnaire_id": questionnaireID},
		},
		{
			"$group": bson.M{
				"_id": nil,
				"total": bson.M{
					"$sum": bson.M{
						"$multiply": []string{"$max_points", "$weight"},
					},
				},
			},
		},
	}

	cursor, err := r.collection.Aggregate(ctx, pipeline)
	if err != nil {
		return 0, err
	}
	defer cursor.Close(ctx)

	var result struct {
		Total int `bson:"total"`
	}
	if cursor.Next(ctx) {
		if err := cursor.Decode(&result); err != nil {
			return 0, err
		}
		return result.Total, nil
	}

	return 0, nil
}

// CountByQuestionnaire counts questions for a questionnaire
func (r *MongoQuestionRepository) CountByQuestionnaire(ctx context.Context, questionnaireID primitive.ObjectID) (int64, error) {
	filter := bson.M{"questionnaire_id": questionnaireID}
	return r.collection.CountDocuments(ctx, filter)
}

// Ensure MongoQuestionRepository implements QuestionRepository
var _ QuestionRepository = (*MongoQuestionRepository)(nil)
