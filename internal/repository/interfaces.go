// Package repository defines interfaces for data access and their MongoDB implementations
// #ORM_PATTERN: Repository pattern with interfaces for testability and abstraction
package repository

import (
	"context"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/checkfix-tools/nisfix_backend/internal/models"
)

// PaginationOptions contains pagination parameters
type PaginationOptions struct {
	Page    int
	Limit   int
	SortBy  string
	SortDir int // 1 for ascending, -1 for descending
}

// DefaultPaginationOptions returns default pagination settings
// #DATA_ASSUMPTION: Pagination defaults to 20 items per page
func DefaultPaginationOptions() PaginationOptions {
	return PaginationOptions{
		Page:    1,
		Limit:   20,
		SortBy:  "created_at",
		SortDir: -1,
	}
}

// PaginatedResult contains paginated query results
type PaginatedResult[T any] struct {
	Items      []T   `json:"items"`
	TotalCount int64 `json:"total_count"`
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	TotalPages int   `json:"total_pages"`
}

// OrganizationRepository defines operations for organizations
// #QUERY_INTERFACE: Organization data access patterns
type OrganizationRepository interface {
	// Create creates a new organization
	Create(ctx context.Context, org *models.Organization) error

	// GetByID finds an organization by ID
	GetByID(ctx context.Context, id primitive.ObjectID) (*models.Organization, error)

	// GetBySlug finds an organization by slug
	GetBySlug(ctx context.Context, slug string) (*models.Organization, error)

	// GetByDomain finds an organization by domain
	GetByDomain(ctx context.Context, domain string) (*models.Organization, error)

	// Update updates an organization
	Update(ctx context.Context, org *models.Organization) error

	// SoftDelete soft deletes an organization
	SoftDelete(ctx context.Context, id primitive.ObjectID) error

	// List lists organizations with filtering and pagination
	List(ctx context.Context, orgType *models.OrganizationType, opts PaginationOptions) (*PaginatedResult[models.Organization], error)
}

// UserRepository defines operations for users
// #QUERY_INTERFACE: User data access patterns
type UserRepository interface {
	// Create creates a new user
	Create(ctx context.Context, user *models.User) error

	// GetByID finds a user by ID
	GetByID(ctx context.Context, id primitive.ObjectID) (*models.User, error)

	// GetByEmail finds a user by email
	GetByEmail(ctx context.Context, email string) (*models.User, error)

	// Update updates a user
	Update(ctx context.Context, user *models.User) error

	// SoftDelete soft deletes a user
	SoftDelete(ctx context.Context, id primitive.ObjectID) error

	// UpdateLastLogin updates the last login timestamp
	UpdateLastLogin(ctx context.Context, id primitive.ObjectID) error

	// ListByOrganization lists users in an organization
	ListByOrganization(ctx context.Context, orgID primitive.ObjectID, includeInactive bool, opts PaginationOptions) (*PaginatedResult[models.User], error)

	// CountByOrganization counts users in an organization
	CountByOrganization(ctx context.Context, orgID primitive.ObjectID) (int64, error)
}

// SecureLinkRepository defines operations for secure links
// #QUERY_INTERFACE: Secure link data access patterns
type SecureLinkRepository interface {
	// Create creates a new secure link
	Create(ctx context.Context, link *models.SecureLink) error

	// GetByIdentifier finds a secure link by its identifier
	GetByIdentifier(ctx context.Context, identifier string) (*models.SecureLink, error)

	// MarkAsUsed marks a secure link as used
	MarkAsUsed(ctx context.Context, id primitive.ObjectID) error

	// Invalidate invalidates a secure link
	Invalidate(ctx context.Context, id primitive.ObjectID) error

	// InvalidateAllForEmail invalidates all links for an email
	InvalidateAllForEmail(ctx context.Context, email string) error

	// CountRecentByEmail counts recent links for rate limiting
	CountRecentByEmail(ctx context.Context, email string, withinMinutes int) (int64, error)

	// DeleteExpired deletes expired links (TTL fallback)
	DeleteExpired(ctx context.Context) (int64, error)
}

// QuestionnaireTemplateRepository defines operations for questionnaire templates
// #QUERY_INTERFACE: Template data access patterns
type QuestionnaireTemplateRepository interface {
	// Create creates a new template
	Create(ctx context.Context, template *models.QuestionnaireTemplate) error

	// GetByID finds a template by ID
	GetByID(ctx context.Context, id primitive.ObjectID) (*models.QuestionnaireTemplate, error)

	// Update updates a template
	Update(ctx context.Context, template *models.QuestionnaireTemplate) error

	// Delete deletes a template
	Delete(ctx context.Context, id primitive.ObjectID) error

	// IncrementUsageCount increments the usage count
	IncrementUsageCount(ctx context.Context, id primitive.ObjectID) error

	// ListSystemTemplates lists all system templates
	ListSystemTemplates(ctx context.Context, category *models.TemplateCategory) ([]models.QuestionnaireTemplate, error)

	// ListByOrganization lists templates created by an organization
	ListByOrganization(ctx context.Context, orgID primitive.ObjectID, opts PaginationOptions) (*PaginatedResult[models.QuestionnaireTemplate], error)

	// SearchTemplates searches templates by name/description
	SearchTemplates(ctx context.Context, query string, opts PaginationOptions) (*PaginatedResult[models.QuestionnaireTemplate], error)

	// ListAvailableTemplates lists templates available to an organization
	// Returns: system templates + globally published + org's own templates (any visibility)
	ListAvailableTemplates(ctx context.Context, orgID primitive.ObjectID, category *models.TemplateCategory, opts PaginationOptions) (*PaginatedResult[models.QuestionnaireTemplate], error)

	// ListByUser lists templates created by a specific user
	ListByUser(ctx context.Context, userID primitive.ObjectID, opts PaginationOptions) (*PaginatedResult[models.QuestionnaireTemplate], error)
}

// QuestionnaireRepository defines operations for questionnaires
// #QUERY_INTERFACE: Questionnaire data access patterns
type QuestionnaireRepository interface {
	// Create creates a new questionnaire
	Create(ctx context.Context, questionnaire *models.Questionnaire) error

	// GetByID finds a questionnaire by ID
	GetByID(ctx context.Context, id primitive.ObjectID) (*models.Questionnaire, error)

	// Update updates a questionnaire
	Update(ctx context.Context, questionnaire *models.Questionnaire) error

	// Delete deletes a questionnaire (draft only)
	Delete(ctx context.Context, id primitive.ObjectID) error

	// UpdateStatistics updates question count and max score
	UpdateStatistics(ctx context.Context, id primitive.ObjectID, questionCount, maxScore int) error

	// ListByCompany lists questionnaires for a company
	ListByCompany(ctx context.Context, companyID primitive.ObjectID, status *models.QuestionnaireStatus, opts PaginationOptions) (*PaginatedResult[models.Questionnaire], error)

	// CountByCompany counts questionnaires for a company
	CountByCompany(ctx context.Context, companyID primitive.ObjectID, status *models.QuestionnaireStatus) (int64, error)
}

// QuestionRepository defines operations for questions
// #QUERY_INTERFACE: Question data access patterns
type QuestionRepository interface {
	// Create creates a new question
	Create(ctx context.Context, question *models.Question) error

	// GetByID finds a question by ID
	GetByID(ctx context.Context, id primitive.ObjectID) (*models.Question, error)

	// Update updates a question
	Update(ctx context.Context, question *models.Question) error

	// Delete deletes a question
	Delete(ctx context.Context, id primitive.ObjectID) error

	// ListByQuestionnaire lists all questions for a questionnaire
	ListByQuestionnaire(ctx context.Context, questionnaireID primitive.ObjectID) ([]models.Question, error)

	// ListByQuestionnaireAndTopic lists questions for a specific topic
	ListByQuestionnaireAndTopic(ctx context.Context, questionnaireID primitive.ObjectID, topicID string) ([]models.Question, error)

	// DeleteByQuestionnaire deletes all questions for a questionnaire
	DeleteByQuestionnaire(ctx context.Context, questionnaireID primitive.ObjectID) (int64, error)

	// UpdateOrder updates the order of questions
	UpdateOrder(ctx context.Context, questionnaireID primitive.ObjectID, orders map[primitive.ObjectID]int) error

	// CalculateMaxScore calculates the max possible score for a questionnaire
	CalculateMaxScore(ctx context.Context, questionnaireID primitive.ObjectID) (int, error)

	// CountByQuestionnaire counts questions for a questionnaire
	CountByQuestionnaire(ctx context.Context, questionnaireID primitive.ObjectID) (int64, error)
}

// RelationshipRepository defines operations for company-supplier relationships
// #QUERY_INTERFACE: Relationship data access patterns
type RelationshipRepository interface {
	// Create creates a new relationship
	Create(ctx context.Context, relationship *models.CompanySupplierRelationship) error

	// GetByID finds a relationship by ID
	GetByID(ctx context.Context, id primitive.ObjectID) (*models.CompanySupplierRelationship, error)

	// GetByCompanyAndSupplier finds a relationship by company and supplier IDs
	GetByCompanyAndSupplier(ctx context.Context, companyID, supplierID primitive.ObjectID) (*models.CompanySupplierRelationship, error)

	// GetByInvitedEmail finds a pending relationship by invited email
	GetByInvitedEmail(ctx context.Context, email string, companyID primitive.ObjectID) (*models.CompanySupplierRelationship, error)

	// Update updates a relationship
	Update(ctx context.Context, relationship *models.CompanySupplierRelationship) error

	// ListByCompany lists relationships for a company
	ListByCompany(ctx context.Context, companyID primitive.ObjectID, status *models.RelationshipStatus, classification *models.SupplierClassification, opts PaginationOptions) (*PaginatedResult[models.CompanySupplierRelationship], error)

	// ListBySupplier lists relationships for a supplier
	ListBySupplier(ctx context.Context, supplierID primitive.ObjectID, status *models.RelationshipStatus, opts PaginationOptions) (*PaginatedResult[models.CompanySupplierRelationship], error)

	// ListPendingByEmail lists pending invitations for an email
	ListPendingByEmail(ctx context.Context, email string) ([]models.CompanySupplierRelationship, error)

	// CountByCompany counts relationships for a company
	CountByCompany(ctx context.Context, companyID primitive.ObjectID, status *models.RelationshipStatus) (int64, error)

	// CountBySupplier counts relationships for a supplier
	CountBySupplier(ctx context.Context, supplierID primitive.ObjectID, status *models.RelationshipStatus) (int64, error)
}

// RequirementRepository defines operations for requirements
// #QUERY_INTERFACE: Requirement data access patterns
type RequirementRepository interface {
	// Create creates a new requirement
	Create(ctx context.Context, requirement *models.Requirement) error

	// GetByID finds a requirement by ID
	GetByID(ctx context.Context, id primitive.ObjectID) (*models.Requirement, error)

	// Update updates a requirement
	Update(ctx context.Context, requirement *models.Requirement) error

	// ListByCompany lists requirements for a company
	ListByCompany(ctx context.Context, companyID primitive.ObjectID, status *models.RequirementStatus, opts PaginationOptions) (*PaginatedResult[models.Requirement], error)

	// ListBySupplier lists requirements for a supplier
	ListBySupplier(ctx context.Context, supplierID primitive.ObjectID, status *models.RequirementStatus, opts PaginationOptions) (*PaginatedResult[models.Requirement], error)

	// ListByRelationship lists requirements for a relationship
	ListByRelationship(ctx context.Context, relationshipID primitive.ObjectID, status *models.RequirementStatus) ([]models.Requirement, error)

	// ListOverdue lists overdue requirements
	ListOverdue(ctx context.Context, companyID *primitive.ObjectID) ([]models.Requirement, error)

	// ListNeedingReminder lists requirements that need reminders
	ListNeedingReminder(ctx context.Context, reminderDaysBefore int) ([]models.Requirement, error)

	// MarkReminderSent marks a requirement's reminder as sent
	MarkReminderSent(ctx context.Context, id primitive.ObjectID) error

	// ExpireOverdue marks overdue requirements as expired
	ExpireOverdue(ctx context.Context) (int64, error)

	// CountByCompany counts requirements for a company
	CountByCompany(ctx context.Context, companyID primitive.ObjectID, status *models.RequirementStatus) (int64, error)

	// CountBySupplier counts requirements for a supplier
	CountBySupplier(ctx context.Context, supplierID primitive.ObjectID, status *models.RequirementStatus) (int64, error)
}

// ResponseRepository defines operations for supplier responses
// #QUERY_INTERFACE: Response data access patterns
type ResponseRepository interface {
	// Create creates a new response
	Create(ctx context.Context, response *models.SupplierResponse) error

	// GetByID finds a response by ID
	GetByID(ctx context.Context, id primitive.ObjectID) (*models.SupplierResponse, error)

	// GetByRequirement finds a response by requirement ID
	GetByRequirement(ctx context.Context, requirementID primitive.ObjectID) (*models.SupplierResponse, error)

	// Update updates a response
	Update(ctx context.Context, response *models.SupplierResponse) error

	// SaveDraftAnswer saves a draft answer
	SaveDraftAnswer(ctx context.Context, responseID primitive.ObjectID, answer models.DraftAnswer) error

	// ListBySupplier lists responses for a supplier
	ListBySupplier(ctx context.Context, supplierID primitive.ObjectID, opts PaginationOptions) (*PaginatedResult[models.SupplierResponse], error)

	// CountBySupplier counts responses for a supplier
	CountBySupplier(ctx context.Context, supplierID primitive.ObjectID) (int64, error)
}

// SubmissionRepository defines operations for questionnaire submissions
// #QUERY_INTERFACE: Submission data access patterns
type SubmissionRepository interface {
	// Create creates a new submission
	Create(ctx context.Context, submission *models.QuestionnaireSubmission) error

	// GetByID finds a submission by ID
	GetByID(ctx context.Context, id primitive.ObjectID) (*models.QuestionnaireSubmission, error)

	// GetByResponse finds a submission by response ID
	GetByResponse(ctx context.Context, responseID primitive.ObjectID) (*models.QuestionnaireSubmission, error)

	// ListByQuestionnaire lists submissions for a questionnaire
	ListByQuestionnaire(ctx context.Context, questionnaireID primitive.ObjectID, opts PaginationOptions) (*PaginatedResult[models.QuestionnaireSubmission], error)

	// GetPassRateByQuestionnaire calculates pass rate for a questionnaire
	GetPassRateByQuestionnaire(ctx context.Context, questionnaireID primitive.ObjectID) (float64, error)
}

// VerificationRepository defines operations for CheckFix verifications
// #QUERY_INTERFACE: Verification data access patterns
type VerificationRepository interface {
	// Create creates a new verification
	Create(ctx context.Context, verification *models.CheckFixVerification) error

	// GetByID finds a verification by ID
	GetByID(ctx context.Context, id primitive.ObjectID) (*models.CheckFixVerification, error)

	// GetByResponse finds a verification by response ID
	GetByResponse(ctx context.Context, responseID primitive.ObjectID) (*models.CheckFixVerification, error)

	// GetLatestBySupplier finds the latest verification for a supplier
	GetLatestBySupplier(ctx context.Context, supplierID primitive.ObjectID) (*models.CheckFixVerification, error)

	// Update updates a verification
	Update(ctx context.Context, verification *models.CheckFixVerification) error

	// ListExpiringVerifications lists verifications that are about to expire
	ListExpiringVerifications(ctx context.Context, daysBeforeExpiry int) ([]models.CheckFixVerification, error)
}

// AuditLogRepository defines operations for audit logs
// #QUERY_INTERFACE: Audit log data access patterns
type AuditLogRepository interface {
	// Create creates a new audit log entry
	Create(ctx context.Context, log *models.AuditLog) error

	// CreateAsync creates an audit log entry asynchronously
	CreateAsync(log *models.AuditLog)

	// ListByActor lists audit logs by actor
	ListByActor(ctx context.Context, userID primitive.ObjectID, opts PaginationOptions) (*PaginatedResult[models.AuditLog], error)

	// ListByResource lists audit logs by resource
	ListByResource(ctx context.Context, resourceType string, resourceID primitive.ObjectID, opts PaginationOptions) (*PaginatedResult[models.AuditLog], error)

	// ListByOrganization lists audit logs by organization
	ListByOrganization(ctx context.Context, orgID primitive.ObjectID, opts PaginationOptions) (*PaginatedResult[models.AuditLog], error)

	// ListByAction lists audit logs by action type
	ListByAction(ctx context.Context, action models.AuditAction, opts PaginationOptions) (*PaginatedResult[models.AuditLog], error)
}
