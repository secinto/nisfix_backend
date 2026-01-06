// Package services provides business logic implementations.
package services

import (
	"context"
	"fmt"
	"log"

	"github.com/checkfix-tools/nisfix_backend/internal/models"
	"github.com/checkfix-tools/nisfix_backend/internal/repository"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// AuditService handles audit logging
// #INTEGRATION_POINT: Used by all services for compliance logging
type AuditService interface {
	// Log creates an audit log entry
	Log(ctx context.Context, entry AuditEntry) error

	// LogAsync logs asynchronously (non-blocking)
	LogAsync(entry AuditEntry)

	// ListByResource lists audit logs for a resource
	ListByResource(ctx context.Context, resourceType string, resourceID primitive.ObjectID, opts repository.PaginationOptions) (*repository.PaginatedResult[models.AuditLog], error)

	// ListByOrganization lists audit logs for an organization
	ListByOrganization(ctx context.Context, orgID primitive.ObjectID, opts repository.PaginationOptions) (*repository.PaginatedResult[models.AuditLog], error)
}

// AuditEntry represents an audit log entry to be created
type AuditEntry struct {
	ActorUserID  *primitive.ObjectID
	ActorEmail   string
	ActorOrgID   *primitive.ObjectID
	Action       models.AuditAction
	ResourceType string
	ResourceID   primitive.ObjectID
	Description  string
	Changes      map[string]interface{}
	IPAddress    string
	UserAgent    string
	RequestID    string
}

// auditService implements AuditService
type auditService struct {
	auditRepo repository.AuditRepository
	logChan   chan AuditEntry
}

// NewAuditService creates a new audit service
func NewAuditService(auditRepo repository.AuditRepository) AuditService {
	svc := &auditService{
		auditRepo: auditRepo,
		logChan:   make(chan AuditEntry, 1000), // Buffer for async logging
	}

	// Start async worker
	go svc.asyncWorker()

	return svc
}

// asyncWorker processes audit entries asynchronously
func (s *auditService) asyncWorker() {
	for entry := range s.logChan {
		ctx := context.Background()
		if err := s.Log(ctx, entry); err != nil {
			log.Printf("Failed to log audit entry: %v", err)
		}
	}
}

// Log creates an audit log entry
func (s *auditService) Log(ctx context.Context, entry AuditEntry) error {
	auditLog := &models.AuditLog{
		ActorUserID:  entry.ActorUserID,
		ActorEmail:   entry.ActorEmail,
		ActorOrgID:   entry.ActorOrgID,
		Action:       entry.Action,
		ResourceType: entry.ResourceType,
		ResourceID:   entry.ResourceID,
		Description:  entry.Description,
		Changes:      entry.Changes,
		IPAddress:    entry.IPAddress,
		UserAgent:    entry.UserAgent,
		RequestID:    entry.RequestID,
	}

	if err := s.auditRepo.Create(ctx, auditLog); err != nil {
		return fmt.Errorf("failed to create audit log: %w", err)
	}

	return nil
}

// LogAsync logs asynchronously (non-blocking)
func (s *auditService) LogAsync(entry AuditEntry) {
	select {
	case s.logChan <- entry:
		// Successfully queued
	default:
		// Channel full, log synchronously as fallback
		log.Printf("Audit log channel full, logging synchronously")
		ctx := context.Background()
		if err := s.Log(ctx, entry); err != nil {
			log.Printf("Failed to log audit entry: %v", err)
		}
	}
}

// ListByResource lists audit logs for a resource
func (s *auditService) ListByResource(ctx context.Context, resourceType string, resourceID primitive.ObjectID, opts repository.PaginationOptions) (*repository.PaginatedResult[models.AuditLog], error) {
	return s.auditRepo.ListByResource(ctx, resourceType, resourceID, opts)
}

// ListByOrganization lists audit logs for an organization
func (s *auditService) ListByOrganization(ctx context.Context, orgID primitive.ObjectID, opts repository.PaginationOptions) (*repository.PaginatedResult[models.AuditLog], error) {
	return s.auditRepo.ListByOrganization(ctx, orgID, opts)
}

// AuditHelpers provides convenient methods for common audit operations
type AuditHelpers struct {
	service AuditService
}

// NewAuditHelpers creates audit helpers
func NewAuditHelpers(service AuditService) *AuditHelpers {
	return &AuditHelpers{service: service}
}

// LogUserLogin logs a user login event
func (h *AuditHelpers) LogUserLogin(userID, orgID primitive.ObjectID, email, ipAddress, userAgent, requestID string) {
	h.service.LogAsync(AuditEntry{
		ActorUserID:  &userID,
		ActorEmail:   email,
		ActorOrgID:   &orgID,
		Action:       models.AuditActionLogin,
		ResourceType: "user",
		ResourceID:   userID,
		Description:  fmt.Sprintf("User %s logged in", email),
		IPAddress:    ipAddress,
		UserAgent:    userAgent,
		RequestID:    requestID,
	})
}

// LogSupplierInvite logs a supplier invitation
func (h *AuditHelpers) LogSupplierInvite(actorUserID, actorOrgID, relationshipID primitive.ObjectID, supplierEmail, requestID string) {
	h.service.LogAsync(AuditEntry{
		ActorUserID:  &actorUserID,
		ActorOrgID:   &actorOrgID,
		Action:       models.AuditActionInvite,
		ResourceType: "relationship",
		ResourceID:   relationshipID,
		Description:  fmt.Sprintf("Invited supplier: %s", supplierEmail),
		RequestID:    requestID,
	})
}

// LogRequirementCreate logs requirement creation
func (h *AuditHelpers) LogRequirementCreate(actorUserID, actorOrgID, requirementID primitive.ObjectID, title, requestID string) {
	h.service.LogAsync(AuditEntry{
		ActorUserID:  &actorUserID,
		ActorOrgID:   &actorOrgID,
		Action:       models.AuditActionCreate,
		ResourceType: "requirement",
		ResourceID:   requirementID,
		Description:  fmt.Sprintf("Created requirement: %s", title),
		RequestID:    requestID,
	})
}

// LogSubmission logs a response submission
func (h *AuditHelpers) LogSubmission(actorUserID, actorOrgID, responseID primitive.ObjectID, requestID string) {
	h.service.LogAsync(AuditEntry{
		ActorUserID:  &actorUserID,
		ActorOrgID:   &actorOrgID,
		Action:       models.AuditActionSubmit,
		ResourceType: "response",
		ResourceID:   responseID,
		Description:  "Submitted response",
		RequestID:    requestID,
	})
}

// LogApproval logs a requirement approval
func (h *AuditHelpers) LogApproval(actorUserID, actorOrgID, requirementID primitive.ObjectID, notes, requestID string) {
	h.service.LogAsync(AuditEntry{
		ActorUserID:  &actorUserID,
		ActorOrgID:   &actorOrgID,
		Action:       models.AuditActionApprove,
		ResourceType: "requirement",
		ResourceID:   requirementID,
		Description:  "Approved requirement",
		Changes:      map[string]interface{}{"notes": notes},
		RequestID:    requestID,
	})
}

// LogRejection logs a requirement rejection
func (h *AuditHelpers) LogRejection(actorUserID, actorOrgID, requirementID primitive.ObjectID, reason, requestID string) {
	h.service.LogAsync(AuditEntry{
		ActorUserID:  &actorUserID,
		ActorOrgID:   &actorOrgID,
		Action:       models.AuditActionReject,
		ResourceType: "requirement",
		ResourceID:   requirementID,
		Description:  "Rejected requirement",
		Changes:      map[string]interface{}{"reason": reason},
		RequestID:    requestID,
	})
}

// LogCheckFixVerification logs a CheckFix verification
func (h *AuditHelpers) LogCheckFixVerification(actorUserID, actorOrgID, verificationID primitive.ObjectID, grade string, passed bool, requestID string) {
	h.service.LogAsync(AuditEntry{
		ActorUserID:  &actorUserID,
		ActorOrgID:   &actorOrgID,
		Action:       models.AuditActionVerify,
		ResourceType: "checkfix_verification",
		ResourceID:   verificationID,
		Description:  fmt.Sprintf("CheckFix verification: Grade %s, Passed: %v", grade, passed),
		Changes:      map[string]interface{}{"grade": grade, "passed": passed},
		RequestID:    requestID,
	})
}

// Ensure auditService implements AuditService
var _ AuditService = (*auditService)(nil)
