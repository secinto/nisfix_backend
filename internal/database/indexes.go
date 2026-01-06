package database

import (
	"context"
	"fmt"
	"log"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"github.com/checkfix-tools/nisfix_backend/internal/models"
)

// IndexManager handles MongoDB index creation and management
// #INDEX_IMPLEMENTATION: All indexes defined per data architecture plan
type IndexManager struct {
	db *mongo.Database
}

// NewIndexManager creates a new index manager
func NewIndexManager(db *mongo.Database) *IndexManager {
	return &IndexManager{db: db}
}

// CreateAllIndexes creates all indexes for all collections
// #MIGRATION_DECISION: Indexes created at application startup if they don't exist
func (m *IndexManager) CreateAllIndexes(ctx context.Context) error {
	log.Println("Creating MongoDB indexes...")

	if err := m.createOrganizationIndexes(ctx); err != nil {
		return fmt.Errorf("failed to create organization indexes: %w", err)
	}

	if err := m.createUserIndexes(ctx); err != nil {
		return fmt.Errorf("failed to create user indexes: %w", err)
	}

	if err := m.createSecureLinkIndexes(ctx); err != nil {
		return fmt.Errorf("failed to create secure link indexes: %w", err)
	}

	if err := m.createQuestionnaireTemplateIndexes(ctx); err != nil {
		return fmt.Errorf("failed to create questionnaire template indexes: %w", err)
	}

	if err := m.createQuestionnaireIndexes(ctx); err != nil {
		return fmt.Errorf("failed to create questionnaire indexes: %w", err)
	}

	if err := m.createQuestionIndexes(ctx); err != nil {
		return fmt.Errorf("failed to create question indexes: %w", err)
	}

	if err := m.createRelationshipIndexes(ctx); err != nil {
		return fmt.Errorf("failed to create relationship indexes: %w", err)
	}

	if err := m.createRequirementIndexes(ctx); err != nil {
		return fmt.Errorf("failed to create requirement indexes: %w", err)
	}

	if err := m.createResponseIndexes(ctx); err != nil {
		return fmt.Errorf("failed to create response indexes: %w", err)
	}

	if err := m.createSubmissionIndexes(ctx); err != nil {
		return fmt.Errorf("failed to create submission indexes: %w", err)
	}

	if err := m.createVerificationIndexes(ctx); err != nil {
		return fmt.Errorf("failed to create verification indexes: %w", err)
	}

	if err := m.createAuditLogIndexes(ctx); err != nil {
		return fmt.Errorf("failed to create audit log indexes: %w", err)
	}

	log.Println("All indexes created successfully")
	return nil
}

// createOrganizationIndexes creates indexes for the organizations collection
// #INDEX_IMPLEMENTATION: Slug unique, domain unique sparse, type + created_at
func (m *IndexManager) createOrganizationIndexes(ctx context.Context) error {
	collection := m.db.Collection(models.Organization{}.CollectionName())

	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "slug", Value: 1}},
			Options: options.Index().SetUnique(true).SetName("idx_slug_unique"),
		},
		{
			Keys:    bson.D{{Key: "domain", Value: 1}},
			Options: options.Index().SetUnique(true).SetSparse(true).SetName("idx_domain_unique_sparse"),
		},
		{
			Keys:    bson.D{{Key: "type", Value: 1}, {Key: "created_at", Value: -1}},
			Options: options.Index().SetName("idx_type_created"),
		},
		{
			Keys:    bson.D{{Key: "deleted_at", Value: 1}},
			Options: options.Index().SetSparse(true).SetName("idx_deleted_at_sparse"),
		},
	}

	_, err := collection.Indexes().CreateMany(ctx, indexes)
	return err
}

// createUserIndexes creates indexes for the users collection
// #INDEX_IMPLEMENTATION: Email unique, organization_id + role, active users
func (m *IndexManager) createUserIndexes(ctx context.Context) error {
	collection := m.db.Collection(models.User{}.CollectionName())

	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "email", Value: 1}},
			Options: options.Index().SetUnique(true).SetName("idx_email_unique"),
		},
		{
			Keys:    bson.D{{Key: "organization_id", Value: 1}, {Key: "role", Value: 1}},
			Options: options.Index().SetName("idx_org_role"),
		},
		{
			Keys:    bson.D{{Key: "organization_id", Value: 1}, {Key: "is_active", Value: 1}},
			Options: options.Index().SetName("idx_org_active"),
		},
		{
			Keys:    bson.D{{Key: "deleted_at", Value: 1}},
			Options: options.Index().SetSparse(true).SetName("idx_deleted_at_sparse"),
		},
	}

	_, err := collection.Indexes().CreateMany(ctx, indexes)
	return err
}

// createSecureLinkIndexes creates indexes for the secure_links collection
// #INDEX_IMPLEMENTATION: TTL index for automatic expiration, unique identifier
func (m *IndexManager) createSecureLinkIndexes(ctx context.Context) error {
	collection := m.db.Collection(models.SecureLink{}.CollectionName())

	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "secure_identifier", Value: 1}},
			Options: options.Index().SetUnique(true).SetName("idx_secure_identifier_unique"),
		},
		{
			// TTL index - MongoDB automatically deletes expired documents
			Keys:    bson.D{{Key: "expires_at", Value: 1}},
			Options: options.Index().SetExpireAfterSeconds(0).SetName("idx_expires_at_ttl"),
		},
		{
			Keys:    bson.D{{Key: "email", Value: 1}, {Key: "created_at", Value: -1}},
			Options: options.Index().SetName("idx_email_created"),
		},
	}

	_, err := collection.Indexes().CreateMany(ctx, indexes)
	return err
}

// createQuestionnaireTemplateIndexes creates indexes for the questionnaire_templates collection
// #INDEX_IMPLEMENTATION: Category + is_system, text search
func (m *IndexManager) createQuestionnaireTemplateIndexes(ctx context.Context) error {
	collection := m.db.Collection(models.QuestionnaireTemplate{}.CollectionName())

	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "category", Value: 1}, {Key: "is_system", Value: 1}},
			Options: options.Index().SetName("idx_category_system"),
		},
		{
			Keys:    bson.D{{Key: "created_by_org_id", Value: 1}},
			Options: options.Index().SetSparse(true).SetName("idx_created_by_org_sparse"),
		},
		{
			Keys: bson.D{
				{Key: "name", Value: "text"},
				{Key: "description", Value: "text"},
				{Key: "tags", Value: "text"},
			},
			Options: options.Index().
				SetName("idx_text_search").
				SetWeights(bson.D{
					{Key: "name", Value: 10},
					{Key: "tags", Value: 5},
					{Key: "description", Value: 1},
				}),
		},
	}

	_, err := collection.Indexes().CreateMany(ctx, indexes)
	return err
}

// createQuestionnaireIndexes creates indexes for the questionnaires collection
// #INDEX_IMPLEMENTATION: Company's questionnaires by status
func (m *IndexManager) createQuestionnaireIndexes(ctx context.Context) error {
	collection := m.db.Collection(models.Questionnaire{}.CollectionName())

	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "company_id", Value: 1}, {Key: "status", Value: 1}, {Key: "created_at", Value: -1}},
			Options: options.Index().SetName("idx_company_status_created"),
		},
		{
			Keys:    bson.D{{Key: "template_id", Value: 1}},
			Options: options.Index().SetSparse(true).SetName("idx_template_sparse"),
		},
	}

	_, err := collection.Indexes().CreateMany(ctx, indexes)
	return err
}

// createQuestionIndexes creates indexes for the questions collection
// #INDEX_IMPLEMENTATION: Questionnaire questions ordered
func (m *IndexManager) createQuestionIndexes(ctx context.Context) error {
	collection := m.db.Collection(models.Question{}.CollectionName())

	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "questionnaire_id", Value: 1}, {Key: "order", Value: 1}},
			Options: options.Index().SetName("idx_questionnaire_order"),
		},
		{
			Keys:    bson.D{{Key: "questionnaire_id", Value: 1}, {Key: "topic_id", Value: 1}, {Key: "order", Value: 1}},
			Options: options.Index().SetName("idx_questionnaire_topic_order"),
		},
	}

	_, err := collection.Indexes().CreateMany(ctx, indexes)
	return err
}

// createRelationshipIndexes creates indexes for the company_supplier_relationships collection
// #INDEX_IMPLEMENTATION: Unique company-supplier pair, status filters
func (m *IndexManager) createRelationshipIndexes(ctx context.Context) error {
	collection := m.db.Collection(models.CompanySupplierRelationship{}.CollectionName())

	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "company_id", Value: 1}, {Key: "supplier_id", Value: 1}},
			Options: options.Index().SetUnique(true).SetSparse(true).SetName("idx_company_supplier_unique_sparse"),
		},
		{
			Keys:    bson.D{{Key: "company_id", Value: 1}, {Key: "status", Value: 1}, {Key: "classification", Value: 1}},
			Options: options.Index().SetName("idx_company_status_class"),
		},
		{
			Keys:    bson.D{{Key: "supplier_id", Value: 1}, {Key: "status", Value: 1}},
			Options: options.Index().SetName("idx_supplier_status"),
		},
		{
			Keys:    bson.D{{Key: "invited_email", Value: 1}, {Key: "status", Value: 1}},
			Options: options.Index().SetName("idx_invited_email_status"),
		},
	}

	_, err := collection.Indexes().CreateMany(ctx, indexes)
	return err
}

// createRequirementIndexes creates indexes for the requirements collection
// #INDEX_IMPLEMENTATION: Company/supplier requirements with status and due date
func (m *IndexManager) createRequirementIndexes(ctx context.Context) error {
	collection := m.db.Collection(models.Requirement{}.CollectionName())

	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "company_id", Value: 1}, {Key: "status", Value: 1}, {Key: "due_date", Value: 1}},
			Options: options.Index().SetName("idx_company_status_due"),
		},
		{
			Keys:    bson.D{{Key: "supplier_id", Value: 1}, {Key: "status", Value: 1}, {Key: "due_date", Value: 1}},
			Options: options.Index().SetName("idx_supplier_status_due"),
		},
		{
			Keys:    bson.D{{Key: "relationship_id", Value: 1}, {Key: "status", Value: 1}},
			Options: options.Index().SetName("idx_relationship_status"),
		},
		{
			Keys:    bson.D{{Key: "due_date", Value: 1}, {Key: "status", Value: 1}},
			Options: options.Index().SetName("idx_due_date_status"),
		},
		{
			Keys:    bson.D{{Key: "status", Value: 1}, {Key: "due_date", Value: 1}},
			Options: options.Index().SetName("idx_status_due_date"),
		},
	}

	_, err := collection.Indexes().CreateMany(ctx, indexes)
	return err
}

// createResponseIndexes creates indexes for the supplier_responses collection
// #INDEX_IMPLEMENTATION: Unique response per requirement
func (m *IndexManager) createResponseIndexes(ctx context.Context) error {
	collection := m.db.Collection(models.SupplierResponse{}.CollectionName())

	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "requirement_id", Value: 1}},
			Options: options.Index().SetUnique(true).SetName("idx_requirement_unique"),
		},
		{
			Keys:    bson.D{{Key: "supplier_id", Value: 1}, {Key: "submitted_at", Value: -1}},
			Options: options.Index().SetName("idx_supplier_submitted"),
		},
	}

	_, err := collection.Indexes().CreateMany(ctx, indexes)
	return err
}

// createSubmissionIndexes creates indexes for the questionnaire_submissions collection
// #INDEX_IMPLEMENTATION: Unique submission per response
func (m *IndexManager) createSubmissionIndexes(ctx context.Context) error {
	collection := m.db.Collection(models.QuestionnaireSubmission{}.CollectionName())

	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "response_id", Value: 1}},
			Options: options.Index().SetUnique(true).SetName("idx_response_unique"),
		},
		{
			Keys:    bson.D{{Key: "supplier_id", Value: 1}, {Key: "submitted_at", Value: -1}},
			Options: options.Index().SetName("idx_supplier_submitted"),
		},
		{
			Keys:    bson.D{{Key: "questionnaire_id", Value: 1}, {Key: "submitted_at", Value: -1}},
			Options: options.Index().SetName("idx_questionnaire_submitted"),
		},
	}

	_, err := collection.Indexes().CreateMany(ctx, indexes)
	return err
}

// createVerificationIndexes creates indexes for the checkfix_verifications collection
// #INDEX_IMPLEMENTATION: Unique verification per response, expiration handling
func (m *IndexManager) createVerificationIndexes(ctx context.Context) error {
	collection := m.db.Collection(models.CheckFixVerification{}.CollectionName())

	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "response_id", Value: 1}},
			Options: options.Index().SetUnique(true).SetName("idx_response_unique"),
		},
		{
			Keys:    bson.D{{Key: "supplier_id", Value: 1}, {Key: "verified_at", Value: -1}},
			Options: options.Index().SetName("idx_supplier_verified"),
		},
		{
			Keys:    bson.D{{Key: "expires_at", Value: 1}},
			Options: options.Index().SetName("idx_expires_at"),
		},
	}

	_, err := collection.Indexes().CreateMany(ctx, indexes)
	return err
}

// createAuditLogIndexes creates indexes for the audit_logs collection
// #INDEX_IMPLEMENTATION: Multiple indexes for different audit query patterns
func (m *IndexManager) createAuditLogIndexes(ctx context.Context) error {
	collection := m.db.Collection(models.AuditLog{}.CollectionName())

	indexes := []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "actor_user_id", Value: 1}, {Key: "created_at", Value: -1}},
			Options: options.Index().SetName("idx_actor_created"),
		},
		{
			Keys:    bson.D{{Key: "resource_type", Value: 1}, {Key: "resource_id", Value: 1}, {Key: "created_at", Value: -1}},
			Options: options.Index().SetName("idx_resource_created"),
		},
		{
			Keys:    bson.D{{Key: "actor_org_id", Value: 1}, {Key: "created_at", Value: -1}},
			Options: options.Index().SetName("idx_org_created"),
		},
		{
			Keys:    bson.D{{Key: "action", Value: 1}, {Key: "created_at", Value: -1}},
			Options: options.Index().SetName("idx_action_created"),
		},
		{
			Keys:    bson.D{{Key: "created_at", Value: 1}},
			Options: options.Index().SetName("idx_created_at"),
		},
	}

	_, err := collection.Indexes().CreateMany(ctx, indexes)
	return err
}

// DropAllIndexes drops all custom indexes (not the _id index)
func (m *IndexManager) DropAllIndexes(ctx context.Context) error {
	collections := []string{
		models.Organization{}.CollectionName(),
		models.User{}.CollectionName(),
		models.SecureLink{}.CollectionName(),
		models.QuestionnaireTemplate{}.CollectionName(),
		models.Questionnaire{}.CollectionName(),
		models.Question{}.CollectionName(),
		models.CompanySupplierRelationship{}.CollectionName(),
		models.Requirement{}.CollectionName(),
		models.SupplierResponse{}.CollectionName(),
		models.QuestionnaireSubmission{}.CollectionName(),
		models.CheckFixVerification{}.CollectionName(),
		models.AuditLog{}.CollectionName(),
	}

	for _, collName := range collections {
		_, err := m.db.Collection(collName).Indexes().DropAll(ctx)
		if err != nil {
			return fmt.Errorf("failed to drop indexes for %s: %w", collName, err)
		}
	}

	return nil
}

// GetIndexInfo returns information about indexes for a collection
func (m *IndexManager) GetIndexInfo(ctx context.Context, collectionName string) ([]bson.M, error) {
	collection := m.db.Collection(collectionName)

	cursor, err := collection.Indexes().List(ctx)
	if err != nil {
		return nil, err
	}
	defer func() {
		if closeErr := cursor.Close(ctx); closeErr != nil {
			// Closing cursor error is logged but not returned
			_ = closeErr
		}
	}()

	var indexes []bson.M
	if err := cursor.All(ctx, &indexes); err != nil {
		return nil, err
	}

	return indexes, nil
}
