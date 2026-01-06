// Package services provides business logic implementations.
package services

import (
	"context"
	"errors"
	"fmt"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/checkfix-tools/nisfix_backend/internal/models"
	"github.com/checkfix-tools/nisfix_backend/internal/repository"
)

// Custom errors for review service
var (
	ErrCannotReview    = errors.New("cannot review this requirement")
	ErrAlreadyReviewed = errors.New("requirement has already been reviewed")
	ErrNoSubmission    = errors.New("no submission to review")
)

// ReviewService handles requirement review business logic
// #INTEGRATION_POINT: Used by review handler for company review workflow
type ReviewService interface {
	// ApproveRequirement approves a submitted requirement
	ApproveRequirement(ctx context.Context, requirementID, companyID, userID primitive.ObjectID, notes string) (*models.Requirement, error)

	// RejectRequirement rejects a submitted requirement
	RejectRequirement(ctx context.Context, requirementID, companyID, userID primitive.ObjectID, reason string) (*models.Requirement, error)

	// RequestRevision requests revision for a submitted requirement
	RequestRevision(ctx context.Context, requirementID, companyID, userID primitive.ObjectID, reason string) (*models.Requirement, error)

	// GetSubmissionForReview gets the submission for a requirement
	GetSubmissionForReview(ctx context.Context, requirementID, companyID primitive.ObjectID) (*ReviewSubmission, error)
}

// ReviewSubmission combines submission with response for review
type ReviewSubmission struct {
	Requirement *models.Requirement             `json:"requirement"`
	Response    *models.SupplierResponse        `json:"response"`
	Submission  *models.QuestionnaireSubmission `json:"submission,omitempty"`
}

// reviewService implements ReviewService
type reviewService struct {
	requirementRepo repository.RequirementRepository
	responseRepo    repository.ResponseRepository
	submissionRepo  repository.SubmissionRepository
}

// NewReviewService creates a new review service
func NewReviewService(
	requirementRepo repository.RequirementRepository,
	responseRepo repository.ResponseRepository,
	submissionRepo repository.SubmissionRepository,
) ReviewService {
	return &reviewService{
		requirementRepo: requirementRepo,
		responseRepo:    responseRepo,
		submissionRepo:  submissionRepo,
	}
}

// ApproveRequirement approves a submitted requirement
// #BUSINESS_RULE: Only submitted requirements can be approved
func (s *reviewService) ApproveRequirement(ctx context.Context, requirementID, companyID, userID primitive.ObjectID, notes string) (*models.Requirement, error) {
	// Get requirement
	requirement, err := s.requirementRepo.GetByID(ctx, requirementID)
	if err != nil {
		if errors.Is(err, models.ErrRequirementNotFound) {
			return nil, ErrRequirementNotFound
		}
		return nil, fmt.Errorf("failed to get requirement: %w", err)
	}

	// Verify company ownership
	if requirement.CompanyID != companyID {
		return nil, ErrRequirementNotFound
	}

	// Check if can be reviewed
	if !requirement.CanBeReviewed() {
		return nil, ErrCannotReview
	}

	// Get response and mark as reviewed
	response, getErr := s.responseRepo.GetByRequirement(ctx, requirementID)
	if getErr == nil && response != nil {
		response.MarkReviewed(userID, notes)
		//nolint:errcheck // Best-effort update
		s.responseRepo.Update(ctx, response)
	}

	// Approve
	if err := requirement.Approve(userID, notes); err != nil {
		return nil, ErrCannotReview
	}

	if err := s.requirementRepo.Update(ctx, requirement); err != nil {
		return nil, fmt.Errorf("failed to update requirement: %w", err)
	}

	return requirement, nil
}

// RejectRequirement rejects a submitted requirement
// #BUSINESS_RULE: Only submitted requirements can be rejected
// #BUSINESS_RULE: Rejection allows supplier to retry
func (s *reviewService) RejectRequirement(ctx context.Context, requirementID, companyID, userID primitive.ObjectID, reason string) (*models.Requirement, error) {
	// Get requirement
	requirement, err := s.requirementRepo.GetByID(ctx, requirementID)
	if err != nil {
		if errors.Is(err, models.ErrRequirementNotFound) {
			return nil, ErrRequirementNotFound
		}
		return nil, fmt.Errorf("failed to get requirement: %w", err)
	}

	// Verify company ownership
	if requirement.CompanyID != companyID {
		return nil, ErrRequirementNotFound
	}

	// Check if can be reviewed
	if !requirement.CanBeReviewed() {
		return nil, ErrCannotReview
	}

	// Get response and mark as reviewed
	response, getErr := s.responseRepo.GetByRequirement(ctx, requirementID)
	if getErr == nil && response != nil {
		response.MarkReviewed(userID, reason)
		//nolint:errcheck // Best-effort update
		s.responseRepo.Update(ctx, response)
	}

	// Reject
	if err := requirement.Reject(userID, reason); err != nil {
		return nil, ErrCannotReview
	}

	if err := s.requirementRepo.Update(ctx, requirement); err != nil {
		return nil, fmt.Errorf("failed to update requirement: %w", err)
	}

	return requirement, nil
}

// RequestRevision requests revision for a submitted requirement
// #BUSINESS_RULE: Puts requirement in under_review status for supplier to resubmit
func (s *reviewService) RequestRevision(ctx context.Context, requirementID, companyID, userID primitive.ObjectID, reason string) (*models.Requirement, error) {
	// Get requirement
	requirement, err := s.requirementRepo.GetByID(ctx, requirementID)
	if err != nil {
		if errors.Is(err, models.ErrRequirementNotFound) {
			return nil, ErrRequirementNotFound
		}
		return nil, fmt.Errorf("failed to get requirement: %w", err)
	}

	// Verify company ownership
	if requirement.CompanyID != companyID {
		return nil, ErrRequirementNotFound
	}

	// Check if can be reviewed
	if !requirement.CanBeReviewed() {
		return nil, ErrCannotReview
	}

	// Request revision
	if err := requirement.RequestRevision(userID, reason); err != nil {
		return nil, ErrCannotReview
	}

	if err := s.requirementRepo.Update(ctx, requirement); err != nil {
		return nil, fmt.Errorf("failed to update requirement: %w", err)
	}

	return requirement, nil
}

// GetSubmissionForReview gets the submission for a requirement
func (s *reviewService) GetSubmissionForReview(ctx context.Context, requirementID, companyID primitive.ObjectID) (*ReviewSubmission, error) {
	// Get requirement
	requirement, err := s.requirementRepo.GetByID(ctx, requirementID)
	if err != nil {
		if errors.Is(err, models.ErrRequirementNotFound) {
			return nil, ErrRequirementNotFound
		}
		return nil, fmt.Errorf("failed to get requirement: %w", err)
	}

	// Verify company ownership
	if requirement.CompanyID != companyID {
		return nil, ErrRequirementNotFound
	}

	// Get response
	response, err := s.responseRepo.GetByRequirement(ctx, requirementID)
	if err != nil {
		if errors.Is(err, models.ErrResponseNotFound) {
			return nil, ErrNoSubmission
		}
		return nil, fmt.Errorf("failed to get response: %w", err)
	}

	result := &ReviewSubmission{
		Requirement: requirement,
		Response:    response,
	}

	// Get submission if it exists
	if response.SubmissionID != nil {
		submission, err := s.submissionRepo.GetByID(ctx, *response.SubmissionID)
		if err == nil {
			result.Submission = submission
		}
	}

	return result, nil
}
