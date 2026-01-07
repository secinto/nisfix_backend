// Package services provides business logic implementations.
package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/checkfix-tools/nisfix_backend/internal/models"
	"github.com/checkfix-tools/nisfix_backend/internal/repository"
)

// CreateTemplateRequest contains data for creating a new template
type CreateTemplateRequest struct {
	Name                string               `json:"name"`
	Description         string               `json:"description,omitempty"`
	Category            string               `json:"category"`
	Version             string               `json:"version,omitempty"`
	DefaultPassingScore int                  `json:"default_passing_score,omitempty"`
	EstimatedMinutes    int                  `json:"estimated_minutes,omitempty"`
	Topics              []TemplateTopicInput `json:"topics,omitempty"`
	Tags                []string             `json:"tags,omitempty"`
}

// TemplateTopicInput represents topic data for template creation/update
type TemplateTopicInput struct {
	ID          string `json:"id,omitempty"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Order       int    `json:"order,omitempty"`
}

// UpdateTemplateRequest contains data for updating a template
type UpdateTemplateRequest struct {
	Name                *string              `json:"name,omitempty"`
	Description         *string              `json:"description,omitempty"`
	Version             *string              `json:"version,omitempty"`
	DefaultPassingScore *int                 `json:"default_passing_score,omitempty"`
	EstimatedMinutes    *int                 `json:"estimated_minutes,omitempty"`
	Topics              []TemplateTopicInput `json:"topics,omitempty"`
	Tags                []string             `json:"tags,omitempty"`
}

// TemplateService handles questionnaire template business logic
// #INTEGRATION_POINT: Used by template handler for CRUD operations
type TemplateService interface {
	// CreateTemplate creates a new draft template
	CreateTemplate(ctx context.Context, orgID, userID primitive.ObjectID, req CreateTemplateRequest) (*models.QuestionnaireTemplate, error)

	// ImportTemplate parses and creates a template from JSON content
	ImportTemplate(ctx context.Context, orgID, userID primitive.ObjectID, content []byte) (*models.QuestionnaireTemplate, error)

	// GetTemplate retrieves a template by ID (checks visibility permissions)
	GetTemplate(ctx context.Context, id primitive.ObjectID, orgID *primitive.ObjectID) (*models.QuestionnaireTemplate, error)

	// UpdateTemplate updates a draft template (user must be owner)
	UpdateTemplate(ctx context.Context, id, userID primitive.ObjectID, req UpdateTemplateRequest) (*models.QuestionnaireTemplate, error)

	// DeleteTemplate deletes a template (user must be owner, template must be deletable)
	DeleteTemplate(ctx context.Context, id, userID primitive.ObjectID) error

	// PublishTemplate publishes a template with specified visibility (user must be owner)
	PublishTemplate(ctx context.Context, id, userID primitive.ObjectID, visibility models.TemplateVisibility) (*models.QuestionnaireTemplate, error)

	// UnpublishTemplate reverts a template to draft (user must be owner, template must not be in use)
	UnpublishTemplate(ctx context.Context, id, userID primitive.ObjectID) (*models.QuestionnaireTemplate, error)

	// ListAvailableTemplates lists templates available to an organization
	ListAvailableTemplates(ctx context.Context, orgID primitive.ObjectID, category *models.TemplateCategory, opts repository.PaginationOptions) (*repository.PaginatedResult[models.QuestionnaireTemplate], error)

	// ListMyTemplates lists templates created by a user
	ListMyTemplates(ctx context.Context, userID primitive.ObjectID, opts repository.PaginationOptions) (*repository.PaginatedResult[models.QuestionnaireTemplate], error)
}

// templateService implements TemplateService
type templateService struct {
	templateRepo repository.QuestionnaireTemplateRepository
}

// NewTemplateService creates a new template service
func NewTemplateService(templateRepo repository.QuestionnaireTemplateRepository) TemplateService {
	return &templateService{
		templateRepo: templateRepo,
	}
}

// CreateTemplate creates a new draft template
// #BUSINESS_RULE: Templates are created as DRAFT, owned by user
func (s *templateService) CreateTemplate(ctx context.Context, orgID, userID primitive.ObjectID, req CreateTemplateRequest) (*models.QuestionnaireTemplate, error) {
	// Validate category
	category := models.TemplateCategory(strings.ToUpper(req.Category))
	if !category.IsValid() {
		return nil, fmt.Errorf("%w: %s", models.ErrTemplateInvalidFormat, "invalid category")
	}

	// Build template
	template := &models.QuestionnaireTemplate{
		Name:                req.Name,
		Description:         req.Description,
		Category:            category,
		Version:             req.Version,
		IsSystem:            false,
		CreatedByOrgID:      &orgID,
		CreatedByUser:       &userID,
		Visibility:          models.TemplateVisibilityDraft,
		DefaultPassingScore: req.DefaultPassingScore,
		EstimatedMinutes:    req.EstimatedMinutes,
		Tags:                req.Tags,
	}

	// Convert topics
	template.Topics = s.convertTopics(req.Topics)

	// Validate
	if err := s.validateTemplate(template); err != nil {
		return nil, err
	}

	// Create in repository
	if err := s.templateRepo.Create(ctx, template); err != nil {
		return nil, fmt.Errorf("failed to create template: %w", err)
	}

	return template, nil
}

// ImportTemplate parses and creates a template from JSON content
func (s *templateService) ImportTemplate(ctx context.Context, orgID, userID primitive.ObjectID, content []byte) (*models.QuestionnaireTemplate, error) {
	var req CreateTemplateRequest
	if err := json.Unmarshal(content, &req); err != nil {
		return nil, fmt.Errorf("%w: %v", models.ErrTemplateInvalidFormat, err)
	}

	// Validate required fields
	if req.Name == "" {
		return nil, fmt.Errorf("%w: name is required", models.ErrTemplateMissingFields)
	}
	if req.Category == "" {
		return nil, fmt.Errorf("%w: category is required", models.ErrTemplateMissingFields)
	}

	return s.CreateTemplate(ctx, orgID, userID, req)
}

// GetTemplate retrieves a template by ID
// #BUSINESS_RULE: Check visibility - system/global visible to all, org templates visible to owning org
func (s *templateService) GetTemplate(ctx context.Context, id primitive.ObjectID, orgID *primitive.ObjectID) (*models.QuestionnaireTemplate, error) {
	template, err := s.templateRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Check visibility
	if !s.canViewTemplate(template, orgID) {
		return nil, models.ErrTemplateNotFound
	}

	return template, nil
}

// UpdateTemplate updates a draft template
// #BUSINESS_RULE: Only owner can update, only drafts can be edited
func (s *templateService) UpdateTemplate(ctx context.Context, id, userID primitive.ObjectID, req UpdateTemplateRequest) (*models.QuestionnaireTemplate, error) {
	template, err := s.templateRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Check ownership
	if !template.IsOwnedByUser(userID) {
		return nil, models.ErrTemplateNotOwnedByUser
	}

	// Check if editable
	if !template.CanBeEdited() {
		return nil, models.ErrTemplateNotEditable
	}

	// Apply updates
	if req.Name != nil {
		template.Name = *req.Name
	}
	if req.Description != nil {
		template.Description = *req.Description
	}
	if req.Version != nil {
		template.Version = *req.Version
	}
	if req.DefaultPassingScore != nil {
		template.DefaultPassingScore = *req.DefaultPassingScore
	}
	if req.EstimatedMinutes != nil {
		template.EstimatedMinutes = *req.EstimatedMinutes
	}
	if req.Topics != nil {
		template.Topics = s.convertTopics(req.Topics)
	}
	if req.Tags != nil {
		template.Tags = req.Tags
	}

	// Validate
	if err := s.validateTemplate(template); err != nil {
		return nil, err
	}

	// Update in repository
	if err := s.templateRepo.Update(ctx, template); err != nil {
		return nil, fmt.Errorf("failed to update template: %w", err)
	}

	return template, nil
}

// DeleteTemplate deletes a template
// #BUSINESS_RULE: Only owner can delete, template must be deletable (draft or unused)
func (s *templateService) DeleteTemplate(ctx context.Context, id, userID primitive.ObjectID) error {
	template, err := s.templateRepo.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// Check ownership
	if !template.IsOwnedByUser(userID) {
		return models.ErrTemplateNotOwnedByUser
	}

	// Check if deletable
	if !template.CanBeDeleted() {
		if template.UsageCount > 0 {
			return models.ErrTemplateInUse
		}
		return models.ErrTemplateNotDeletable
	}

	return s.templateRepo.Delete(ctx, id)
}

// PublishTemplate publishes a template with specified visibility
// #BUSINESS_RULE: Only owner can publish, template must be draft
func (s *templateService) PublishTemplate(ctx context.Context, id, userID primitive.ObjectID, visibility models.TemplateVisibility) (*models.QuestionnaireTemplate, error) {
	// Validate visibility
	if visibility != models.TemplateVisibilityLocal && visibility != models.TemplateVisibilityGlobal {
		return nil, models.ErrTemplateInvalidVisibility
	}

	template, err := s.templateRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Check ownership
	if !template.IsOwnedByUser(userID) {
		return nil, models.ErrTemplateNotOwnedByUser
	}

	// Check if draft
	if !template.IsDraft() {
		return nil, models.ErrTemplateAlreadyPublished
	}

	// Validate for publishing
	if err := s.validateForPublish(template); err != nil {
		return nil, err
	}

	// Publish
	template.Publish(visibility, userID)

	// Update in repository
	if err := s.templateRepo.Update(ctx, template); err != nil {
		return nil, fmt.Errorf("failed to publish template: %w", err)
	}

	return template, nil
}

// UnpublishTemplate reverts a template to draft
// #BUSINESS_RULE: Only owner can unpublish, template must not be in use
func (s *templateService) UnpublishTemplate(ctx context.Context, id, userID primitive.ObjectID) (*models.QuestionnaireTemplate, error) {
	template, err := s.templateRepo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	// Check ownership
	if !template.IsOwnedByUser(userID) {
		return nil, models.ErrTemplateNotOwnedByUser
	}

	// Check if published
	if !template.IsPublished() {
		return nil, models.ErrTemplateNotPublished
	}

	// Check if can be unpublished
	if !template.CanBeUnpublished() {
		return nil, models.ErrTemplateInUse
	}

	// Unpublish
	template.Unpublish()

	// Update in repository
	if err := s.templateRepo.Update(ctx, template); err != nil {
		return nil, fmt.Errorf("failed to unpublish template: %w", err)
	}

	return template, nil
}

// ListAvailableTemplates lists templates available to an organization
func (s *templateService) ListAvailableTemplates(ctx context.Context, orgID primitive.ObjectID, category *models.TemplateCategory, opts repository.PaginationOptions) (*repository.PaginatedResult[models.QuestionnaireTemplate], error) {
	return s.templateRepo.ListAvailableTemplates(ctx, orgID, category, opts)
}

// ListMyTemplates lists templates created by a user
func (s *templateService) ListMyTemplates(ctx context.Context, userID primitive.ObjectID, opts repository.PaginationOptions) (*repository.PaginatedResult[models.QuestionnaireTemplate], error) {
	return s.templateRepo.ListByUser(ctx, userID, opts)
}

// Helper methods

// convertTopics converts topic inputs to model topics
func (s *templateService) convertTopics(inputs []TemplateTopicInput) []models.TemplateTopic {
	topics := make([]models.TemplateTopic, len(inputs))
	for i, input := range inputs {
		id := input.ID
		if id == "" {
			id = uuid.New().String()
		}
		order := input.Order
		if order == 0 {
			order = i + 1
		}
		topics[i] = models.TemplateTopic{
			ID:          id,
			Name:        input.Name,
			Description: input.Description,
			Order:       order,
		}
	}
	return topics
}

// validateTemplate validates a template's basic requirements
func (s *templateService) validateTemplate(template *models.QuestionnaireTemplate) error {
	var errs []string

	if template.Name == "" {
		errs = append(errs, "name is required")
	}
	if !template.Category.IsValid() {
		errs = append(errs, "invalid category")
	}
	if template.DefaultPassingScore < 0 || template.DefaultPassingScore > 100 {
		errs = append(errs, "default_passing_score must be between 0 and 100")
	}

	// Validate topics
	topicIDs := make(map[string]bool)
	for i, topic := range template.Topics {
		if topic.Name == "" {
			errs = append(errs, fmt.Sprintf("topic[%d].name is required", i))
		}
		if topic.ID != "" {
			if topicIDs[topic.ID] {
				errs = append(errs, fmt.Sprintf("duplicate topic ID: %s", topic.ID))
			}
			topicIDs[topic.ID] = true
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("%w: %s", models.ErrTemplateMissingFields, strings.Join(errs, "; "))
	}
	return nil
}

// validateForPublish validates a template is ready to be published
func (s *templateService) validateForPublish(template *models.QuestionnaireTemplate) error {
	if template.Name == "" {
		return fmt.Errorf("%w: template must have a name", models.ErrTemplateMissingFields)
	}
	if len(template.Topics) == 0 {
		return fmt.Errorf("%w: template must have at least one topic", models.ErrTemplateMissingFields)
	}
	return nil
}

// canViewTemplate checks if a template is visible to the given organization
func (s *templateService) canViewTemplate(template *models.QuestionnaireTemplate, orgID *primitive.ObjectID) bool {
	// System templates are always visible
	if template.IsSystem {
		return true
	}

	// Globally published templates are visible to all
	if template.IsGlobal() {
		return true
	}

	// Org's own templates are visible (any visibility)
	if orgID != nil && template.IsOwnedByOrg(*orgID) {
		return true
	}

	// Locally published templates from other orgs are not visible
	return false
}

// Ensure templateService implements TemplateService
var _ TemplateService = (*templateService)(nil)
