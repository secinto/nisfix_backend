// Package services provides business logic implementations.
package services

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/checkfix-tools/nisfix_backend/internal/models"
	"github.com/checkfix-tools/nisfix_backend/internal/repository"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Custom errors for requirement service
var (
	ErrRequirementNotFound     = errors.New("requirement not found")
	ErrInvalidRequirementType  = errors.New("invalid requirement type")
	ErrRelationshipNotActive   = errors.New("relationship is not active")
	ErrQuestionnaireNotPublished = errors.New("questionnaire is not published")
)

// RequirementService handles requirement business logic
// #INTEGRATION_POINT: Used by requirement handler for requirement management
type RequirementService interface {
	// CreateRequirement creates a new requirement for a supplier
	CreateRequirement(ctx context.Context, companyID, userID primitive.ObjectID, req CreateRequirementRequest) (*models.Requirement, error)

	// GetRequirement retrieves a requirement by ID
	GetRequirement(ctx context.Context, id primitive.ObjectID, companyID *primitive.ObjectID) (*models.Requirement, error)

	// ListRequirementsByCompany lists requirements created by a company
	ListRequirementsByCompany(ctx context.Context, companyID primitive.ObjectID, filters RequirementFilters, opts repository.PaginationOptions) (*repository.PaginatedResult[models.Requirement], error)

	// ListRequirementsBySupplier lists requirements for a supplier
	ListRequirementsBySupplier(ctx context.Context, supplierID primitive.ObjectID, filters RequirementFilters, opts repository.PaginationOptions) (*repository.PaginatedResult[models.Requirement], error)

	// ListRequirementsByRelationship lists requirements for a specific relationship
	ListRequirementsByRelationship(ctx context.Context, relationshipID primitive.ObjectID, status *models.RequirementStatus) ([]models.Requirement, error)

	// UpdateRequirement updates requirement details (before supplier starts)
	UpdateRequirement(ctx context.Context, id, companyID primitive.ObjectID, req UpdateRequirementRequest) (*models.Requirement, error)

	// GetRequirementStats returns requirement statistics for a company
	GetRequirementStats(ctx context.Context, companyID primitive.ObjectID) (*RequirementStats, error)
}

// CreateRequirementRequest represents the request to create a requirement
type CreateRequirementRequest struct {
	RelationshipID   string            `json:"relationship_id" binding:"required"`
	Type             models.RequirementType `json:"type" binding:"required"`
	Title            string            `json:"title" binding:"required"`
	Description      string            `json:"description,omitempty"`
	Priority         models.Priority   `json:"priority,omitempty"`
	DueDate          *time.Time        `json:"due_date,omitempty"`

	// For Questionnaire requirements
	QuestionnaireID  *string           `json:"questionnaire_id,omitempty"`
	PassingScore     *int              `json:"passing_score,omitempty"`

	// For CheckFix requirements
	MinimumGrade     *string           `json:"minimum_grade,omitempty"`
	MaxReportAgeDays *int              `json:"max_report_age_days,omitempty"`
}

// UpdateRequirementRequest represents the request to update a requirement
type UpdateRequirementRequest struct {
	Title            *string           `json:"title,omitempty"`
	Description      *string           `json:"description,omitempty"`
	Priority         *models.Priority  `json:"priority,omitempty"`
	DueDate          *time.Time        `json:"due_date,omitempty"`
	PassingScore     *int              `json:"passing_score,omitempty"`
	MinimumGrade     *string           `json:"minimum_grade,omitempty"`
	MaxReportAgeDays *int              `json:"max_report_age_days,omitempty"`
}

// RequirementFilters contains filters for listing requirements
type RequirementFilters struct {
	Status   *models.RequirementStatus
	Type     *models.RequirementType
	Priority *models.Priority
}

// RequirementStats contains requirement statistics
type RequirementStats struct {
	Total       int64 `json:"total"`
	Pending     int64 `json:"pending"`
	InProgress  int64 `json:"in_progress"`
	Submitted   int64 `json:"submitted"`
	Approved    int64 `json:"approved"`
	Rejected    int64 `json:"rejected"`
	Expired     int64 `json:"expired"`
	Overdue     int64 `json:"overdue"`
}

// requirementService implements RequirementService
type requirementService struct {
	requirementRepo   repository.RequirementRepository
	relationshipRepo  repository.RelationshipRepository
	questionnaireRepo repository.QuestionnaireRepository
}

// NewRequirementService creates a new requirement service
func NewRequirementService(
	requirementRepo repository.RequirementRepository,
	relationshipRepo repository.RelationshipRepository,
	questionnaireRepo repository.QuestionnaireRepository,
) RequirementService {
	return &requirementService{
		requirementRepo:   requirementRepo,
		relationshipRepo:  relationshipRepo,
		questionnaireRepo: questionnaireRepo,
	}
}

// CreateRequirement creates a new requirement for a supplier
// #BUSINESS_RULE: Requirements can only be created for active relationships
// #BUSINESS_RULE: Questionnaire requirements must reference a published questionnaire
func (s *requirementService) CreateRequirement(ctx context.Context, companyID, userID primitive.ObjectID, req CreateRequirementRequest) (*models.Requirement, error) {
	// Validate requirement type
	if !req.Type.IsValid() {
		return nil, ErrInvalidRequirementType
	}

	// Parse and validate relationship
	relationshipID, err := primitive.ObjectIDFromHex(req.RelationshipID)
	if err != nil {
		return nil, ErrRelationshipNotFound
	}

	relationship, err := s.relationshipRepo.GetByID(ctx, relationshipID)
	if err != nil {
		if errors.Is(err, models.ErrRelationshipNotFound) {
			return nil, ErrRelationshipNotFound
		}
		return nil, fmt.Errorf("failed to get relationship: %w", err)
	}

	// Verify company ownership
	if relationship.CompanyID != companyID {
		return nil, ErrRelationshipNotFound
	}

	// Verify relationship is active
	if !relationship.CanReceiveRequirements() {
		return nil, ErrRelationshipNotActive
	}

	// Create requirement
	requirement := &models.Requirement{
		RelationshipID:   relationshipID,
		CompanyID:        companyID,
		SupplierID:       *relationship.SupplierID,
		Type:             req.Type,
		Title:            req.Title,
		Description:      req.Description,
		Priority:         req.Priority,
		DueDate:          req.DueDate,
		AssignedByUserID: userID,
	}

	// Set defaults
	if requirement.Priority == "" {
		requirement.Priority = models.PriorityMedium
	}

	// Handle type-specific fields
	if req.Type == models.RequirementTypeQuestionnaire {
		if req.QuestionnaireID == nil {
			return nil, errors.New("questionnaire_id is required for questionnaire requirements")
		}

		questionnaireID, err := primitive.ObjectIDFromHex(*req.QuestionnaireID)
		if err != nil {
			return nil, errors.New("invalid questionnaire_id")
		}

		// Verify questionnaire exists and is published
		questionnaire, err := s.questionnaireRepo.GetByID(ctx, questionnaireID)
		if err != nil {
			if errors.Is(err, models.ErrQuestionnaireNotFound) {
				return nil, ErrQuestionnaireNotFound
			}
			return nil, fmt.Errorf("failed to get questionnaire: %w", err)
		}

		// Verify company owns the questionnaire
		if questionnaire.CompanyID != companyID {
			return nil, ErrQuestionnaireNotFound
		}

		// Verify questionnaire is published
		if !questionnaire.IsPublished() {
			return nil, ErrQuestionnaireNotPublished
		}

		requirement.QuestionnaireID = &questionnaireID
		requirement.PassingScore = req.PassingScore
		if requirement.PassingScore == nil {
			ps := questionnaire.PassingScore
			requirement.PassingScore = &ps
		}
	} else if req.Type == models.RequirementTypeCheckFix {
		requirement.MinimumGrade = req.MinimumGrade
		requirement.MaxReportAgeDays = req.MaxReportAgeDays

		// Set defaults for CheckFix
		if requirement.MinimumGrade == nil {
			defaultGrade := "C"
			requirement.MinimumGrade = &defaultGrade
		}
		if requirement.MaxReportAgeDays == nil {
			defaultDays := 90
			requirement.MaxReportAgeDays = &defaultDays
		}
	}

	requirement.BeforeCreate()

	if err := s.requirementRepo.Create(ctx, requirement); err != nil {
		return nil, fmt.Errorf("failed to create requirement: %w", err)
	}

	return requirement, nil
}

// GetRequirement retrieves a requirement by ID
func (s *requirementService) GetRequirement(ctx context.Context, id primitive.ObjectID, companyID *primitive.ObjectID) (*models.Requirement, error) {
	requirement, err := s.requirementRepo.GetByID(ctx, id)
	if err != nil {
		if errors.Is(err, models.ErrRequirementNotFound) {
			return nil, ErrRequirementNotFound
		}
		return nil, fmt.Errorf("failed to get requirement: %w", err)
	}

	// Verify company ownership if provided
	if companyID != nil && requirement.CompanyID != *companyID {
		return nil, ErrRequirementNotFound
	}

	return requirement, nil
}

// ListRequirementsByCompany lists requirements created by a company
func (s *requirementService) ListRequirementsByCompany(ctx context.Context, companyID primitive.ObjectID, filters RequirementFilters, opts repository.PaginationOptions) (*repository.PaginatedResult[models.Requirement], error) {
	return s.requirementRepo.ListByCompany(ctx, companyID, filters.Status, opts)
}

// ListRequirementsBySupplier lists requirements for a supplier
func (s *requirementService) ListRequirementsBySupplier(ctx context.Context, supplierID primitive.ObjectID, filters RequirementFilters, opts repository.PaginationOptions) (*repository.PaginatedResult[models.Requirement], error) {
	return s.requirementRepo.ListBySupplier(ctx, supplierID, filters.Status, opts)
}

// ListRequirementsByRelationship lists requirements for a specific relationship
func (s *requirementService) ListRequirementsByRelationship(ctx context.Context, relationshipID primitive.ObjectID, status *models.RequirementStatus) ([]models.Requirement, error) {
	return s.requirementRepo.ListByRelationship(ctx, relationshipID, status)
}

// UpdateRequirement updates requirement details
// #BUSINESS_RULE: Requirements can only be updated while pending
func (s *requirementService) UpdateRequirement(ctx context.Context, id, companyID primitive.ObjectID, req UpdateRequirementRequest) (*models.Requirement, error) {
	requirement, err := s.GetRequirement(ctx, id, &companyID)
	if err != nil {
		return nil, err
	}

	// Only pending requirements can be updated
	if !requirement.IsPending() {
		return nil, errors.New("requirement can only be updated while pending")
	}

	// Update fields if provided
	if req.Title != nil {
		requirement.Title = *req.Title
	}
	if req.Description != nil {
		requirement.Description = *req.Description
	}
	if req.Priority != nil && req.Priority.IsValid() {
		requirement.Priority = *req.Priority
	}
	if req.DueDate != nil {
		requirement.DueDate = req.DueDate
	}
	if req.PassingScore != nil && requirement.IsQuestionnaireRequirement() {
		requirement.PassingScore = req.PassingScore
	}
	if req.MinimumGrade != nil && requirement.IsCheckFixRequirement() {
		requirement.MinimumGrade = req.MinimumGrade
	}
	if req.MaxReportAgeDays != nil && requirement.IsCheckFixRequirement() {
		requirement.MaxReportAgeDays = req.MaxReportAgeDays
	}

	requirement.BeforeUpdate()

	if err := s.requirementRepo.Update(ctx, requirement); err != nil {
		return nil, fmt.Errorf("failed to update requirement: %w", err)
	}

	return requirement, nil
}

// GetRequirementStats returns requirement statistics for a company
func (s *requirementService) GetRequirementStats(ctx context.Context, companyID primitive.ObjectID) (*RequirementStats, error) {
	total, err := s.requirementRepo.CountByCompany(ctx, companyID, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to count total: %w", err)
	}

	pendingStatus := models.RequirementStatusPending
	pending, err := s.requirementRepo.CountByCompany(ctx, companyID, &pendingStatus)
	if err != nil {
		return nil, fmt.Errorf("failed to count pending: %w", err)
	}

	inProgressStatus := models.RequirementStatusInProgress
	inProgress, err := s.requirementRepo.CountByCompany(ctx, companyID, &inProgressStatus)
	if err != nil {
		return nil, fmt.Errorf("failed to count in_progress: %w", err)
	}

	submittedStatus := models.RequirementStatusSubmitted
	submitted, err := s.requirementRepo.CountByCompany(ctx, companyID, &submittedStatus)
	if err != nil {
		return nil, fmt.Errorf("failed to count submitted: %w", err)
	}

	approvedStatus := models.RequirementStatusApproved
	approved, err := s.requirementRepo.CountByCompany(ctx, companyID, &approvedStatus)
	if err != nil {
		return nil, fmt.Errorf("failed to count approved: %w", err)
	}

	rejectedStatus := models.RequirementStatusRejected
	rejected, err := s.requirementRepo.CountByCompany(ctx, companyID, &rejectedStatus)
	if err != nil {
		return nil, fmt.Errorf("failed to count rejected: %w", err)
	}

	expiredStatus := models.RequirementStatusExpired
	expired, err := s.requirementRepo.CountByCompany(ctx, companyID, &expiredStatus)
	if err != nil {
		return nil, fmt.Errorf("failed to count expired: %w", err)
	}

	// Count overdue (would need a specific query)
	// #TECHNICAL_DEBT: Add overdue count method to repository
	overdue := int64(0)

	return &RequirementStats{
		Total:      total,
		Pending:    pending,
		InProgress: inProgress,
		Submitted:  submitted,
		Approved:   approved,
		Rejected:   rejected,
		Expired:    expired,
		Overdue:    overdue,
	}, nil
}
