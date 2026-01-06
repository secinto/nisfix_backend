// Package services provides business logic implementations.
package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/checkfix-tools/nisfix_backend/internal/models"
)

// NotificationService handles all email notifications
// #INTEGRATION_POINT: Used by various services to send event notifications
type NotificationService interface {
	// Supplier invitation notifications
	SendSupplierInvitation(ctx context.Context, email, companyName, supplierName, inviteLink string) error

	// Requirement notifications
	SendRequirementAssigned(ctx context.Context, email, supplierName, companyName, requirementTitle string, dueDate *time.Time) error
	SendRequirementReminder(ctx context.Context, email, supplierName, requirementTitle string, daysUntilDue int) error
	SendRequirementOverdue(ctx context.Context, email, supplierName, requirementTitle string) error

	// Submission notifications
	SendSubmissionReceived(ctx context.Context, email, companyName, supplierName, requirementTitle string) error
	SendSubmissionApproved(ctx context.Context, email, supplierName, companyName, requirementTitle string, notes string) error
	SendSubmissionRejected(ctx context.Context, email, supplierName, companyName, requirementTitle string, reason string) error
	SendRevisionRequested(ctx context.Context, email, supplierName, companyName, requirementTitle string, reason string) error

	// CheckFix notifications
	SendCheckFixLinked(ctx context.Context, email, supplierName, domain string) error
	SendCheckFixVerified(ctx context.Context, email, supplierName string, grade string, passed bool) error
}

// notificationService implements NotificationService
type notificationService struct {
	mailServiceURL string
	apiKey         string
	httpClient     *http.Client
	fromEmail      string
	fromName       string
}

// NewNotificationService creates a new notification service
func NewNotificationService(mailServiceURL, apiKey, fromEmail, fromName string) NotificationService {
	return &notificationService{
		mailServiceURL: mailServiceURL,
		apiKey:         apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		fromEmail: fromEmail,
		fromName:  fromName,
	}
}

// EmailRequest represents an email send request
type EmailRequest struct {
	To       string            `json:"to"`
	From     string            `json:"from"`
	FromName string            `json:"from_name"`
	Subject  string            `json:"subject"`
	Template string            `json:"template"`
	Data     map[string]string `json:"data"`
}

// sendEmail sends an email via the mail service
func (s *notificationService) sendEmail(ctx context.Context, req EmailRequest) error {
	body, err := json.Marshal(req)
	if err != nil {
		return err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", s.mailServiceURL+"/api/v1/send", bytes.NewReader(body))
	if err != nil {
		return err
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+s.apiKey)

	resp, err := s.httpClient.Do(httpReq)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("mail service returned status %d", resp.StatusCode)
	}

	return nil
}

// SendSupplierInvitation sends a supplier invitation email
func (s *notificationService) SendSupplierInvitation(ctx context.Context, email, companyName, supplierName, inviteLink string) error {
	return s.sendEmail(ctx, EmailRequest{
		To:       email,
		From:     s.fromEmail,
		FromName: s.fromName,
		Subject:  fmt.Sprintf("%s has invited you to their supplier portal", companyName),
		Template: "supplier_invitation",
		Data: map[string]string{
			"company_name":  companyName,
			"supplier_name": supplierName,
			"invite_link":   inviteLink,
		},
	})
}

// SendRequirementAssigned sends a notification when a requirement is assigned
func (s *notificationService) SendRequirementAssigned(ctx context.Context, email, supplierName, companyName, requirementTitle string, dueDate *time.Time) error {
	dueDateStr := "No due date"
	if dueDate != nil {
		dueDateStr = dueDate.Format("January 2, 2006")
	}

	return s.sendEmail(ctx, EmailRequest{
		To:       email,
		From:     s.fromEmail,
		FromName: s.fromName,
		Subject:  fmt.Sprintf("New requirement from %s: %s", companyName, requirementTitle),
		Template: "requirement_assigned",
		Data: map[string]string{
			"supplier_name":     supplierName,
			"company_name":      companyName,
			"requirement_title": requirementTitle,
			"due_date":          dueDateStr,
		},
	})
}

// SendRequirementReminder sends a reminder for an upcoming due date
func (s *notificationService) SendRequirementReminder(ctx context.Context, email, supplierName, requirementTitle string, daysUntilDue int) error {
	daysText := fmt.Sprintf("%d days", daysUntilDue)
	if daysUntilDue == 1 {
		daysText = "1 day"
	}

	return s.sendEmail(ctx, EmailRequest{
		To:       email,
		From:     s.fromEmail,
		FromName: s.fromName,
		Subject:  fmt.Sprintf("Reminder: %s is due in %s", requirementTitle, daysText),
		Template: "requirement_reminder",
		Data: map[string]string{
			"supplier_name":     supplierName,
			"requirement_title": requirementTitle,
			"days_until_due":    daysText,
		},
	})
}

// SendRequirementOverdue sends a notification when a requirement is overdue
func (s *notificationService) SendRequirementOverdue(ctx context.Context, email, supplierName, requirementTitle string) error {
	return s.sendEmail(ctx, EmailRequest{
		To:       email,
		From:     s.fromEmail,
		FromName: s.fromName,
		Subject:  fmt.Sprintf("Overdue: %s", requirementTitle),
		Template: "requirement_overdue",
		Data: map[string]string{
			"supplier_name":     supplierName,
			"requirement_title": requirementTitle,
		},
	})
}

// SendSubmissionReceived sends a notification when a submission is received
func (s *notificationService) SendSubmissionReceived(ctx context.Context, email, companyName, supplierName, requirementTitle string) error {
	return s.sendEmail(ctx, EmailRequest{
		To:       email,
		From:     s.fromEmail,
		FromName: s.fromName,
		Subject:  fmt.Sprintf("Submission received from %s: %s", supplierName, requirementTitle),
		Template: "submission_received",
		Data: map[string]string{
			"company_name":      companyName,
			"supplier_name":     supplierName,
			"requirement_title": requirementTitle,
		},
	})
}

// SendSubmissionApproved sends a notification when a submission is approved
func (s *notificationService) SendSubmissionApproved(ctx context.Context, email, supplierName, companyName, requirementTitle string, notes string) error {
	return s.sendEmail(ctx, EmailRequest{
		To:       email,
		From:     s.fromEmail,
		FromName: s.fromName,
		Subject:  fmt.Sprintf("Approved: %s", requirementTitle),
		Template: "submission_approved",
		Data: map[string]string{
			"supplier_name":     supplierName,
			"company_name":      companyName,
			"requirement_title": requirementTitle,
			"notes":             notes,
		},
	})
}

// SendSubmissionRejected sends a notification when a submission is rejected
func (s *notificationService) SendSubmissionRejected(ctx context.Context, email, supplierName, companyName, requirementTitle string, reason string) error {
	return s.sendEmail(ctx, EmailRequest{
		To:       email,
		From:     s.fromEmail,
		FromName: s.fromName,
		Subject:  fmt.Sprintf("Action Required: %s submission rejected", requirementTitle),
		Template: "submission_rejected",
		Data: map[string]string{
			"supplier_name":     supplierName,
			"company_name":      companyName,
			"requirement_title": requirementTitle,
			"reason":            reason,
		},
	})
}

// SendRevisionRequested sends a notification when revision is requested
func (s *notificationService) SendRevisionRequested(ctx context.Context, email, supplierName, companyName, requirementTitle string, reason string) error {
	return s.sendEmail(ctx, EmailRequest{
		To:       email,
		From:     s.fromEmail,
		FromName: s.fromName,
		Subject:  fmt.Sprintf("Revision requested: %s", requirementTitle),
		Template: "revision_requested",
		Data: map[string]string{
			"supplier_name":     supplierName,
			"company_name":      companyName,
			"requirement_title": requirementTitle,
			"reason":            reason,
		},
	})
}

// SendCheckFixLinked sends a notification when CheckFix is linked
func (s *notificationService) SendCheckFixLinked(ctx context.Context, email, supplierName, domain string) error {
	return s.sendEmail(ctx, EmailRequest{
		To:       email,
		From:     s.fromEmail,
		FromName: s.fromName,
		Subject:  "CheckFix account linked successfully",
		Template: "checkfix_linked",
		Data: map[string]string{
			"supplier_name": supplierName,
			"domain":        domain,
		},
	})
}

// SendCheckFixVerified sends a notification when CheckFix verification completes
func (s *notificationService) SendCheckFixVerified(ctx context.Context, email, supplierName string, grade string, passed bool) error {
	status := "passed"
	if !passed {
		status = "did not meet requirements"
	}

	return s.sendEmail(ctx, EmailRequest{
		To:       email,
		From:     s.fromEmail,
		FromName: s.fromName,
		Subject:  fmt.Sprintf("CheckFix verification: Grade %s", grade),
		Template: "checkfix_verified",
		Data: map[string]string{
			"supplier_name": supplierName,
			"grade":         grade,
			"status":        status,
		},
	})
}

// MockNotificationService is a mock implementation for development/testing
type MockNotificationService struct {
	SentNotifications []MockNotification
}

// MockNotification represents a sent notification for testing
type MockNotification struct {
	Type    string
	Email   string
	Subject string
	Data    map[string]string
}

// NewMockNotificationService creates a mock notification service
func NewMockNotificationService() *MockNotificationService {
	return &MockNotificationService{
		SentNotifications: []MockNotification{},
	}
}

func (s *MockNotificationService) log(notifType, email, subject string, data map[string]string) {
	s.SentNotifications = append(s.SentNotifications, MockNotification{
		Type:    notifType,
		Email:   email,
		Subject: subject,
		Data:    data,
	})
	fmt.Printf("[MOCK NOTIFICATION] Type: %s, To: %s, Subject: %s\n", notifType, email, subject)
}

func (s *MockNotificationService) SendSupplierInvitation(ctx context.Context, email, companyName, supplierName, inviteLink string) error {
	s.log("supplier_invitation", email, fmt.Sprintf("%s has invited you", companyName), map[string]string{"company": companyName, "link": inviteLink})
	return nil
}

func (s *MockNotificationService) SendRequirementAssigned(ctx context.Context, email, supplierName, companyName, requirementTitle string, dueDate *time.Time) error {
	s.log("requirement_assigned", email, requirementTitle, map[string]string{"company": companyName, "title": requirementTitle})
	return nil
}

func (s *MockNotificationService) SendRequirementReminder(ctx context.Context, email, supplierName, requirementTitle string, daysUntilDue int) error {
	s.log("requirement_reminder", email, requirementTitle, map[string]string{"days": fmt.Sprintf("%d", daysUntilDue)})
	return nil
}

func (s *MockNotificationService) SendRequirementOverdue(ctx context.Context, email, supplierName, requirementTitle string) error {
	s.log("requirement_overdue", email, requirementTitle, nil)
	return nil
}

func (s *MockNotificationService) SendSubmissionReceived(ctx context.Context, email, companyName, supplierName, requirementTitle string) error {
	s.log("submission_received", email, requirementTitle, map[string]string{"supplier": supplierName})
	return nil
}

func (s *MockNotificationService) SendSubmissionApproved(ctx context.Context, email, supplierName, companyName, requirementTitle string, notes string) error {
	s.log("submission_approved", email, requirementTitle, map[string]string{"notes": notes})
	return nil
}

func (s *MockNotificationService) SendSubmissionRejected(ctx context.Context, email, supplierName, companyName, requirementTitle string, reason string) error {
	s.log("submission_rejected", email, requirementTitle, map[string]string{"reason": reason})
	return nil
}

func (s *MockNotificationService) SendRevisionRequested(ctx context.Context, email, supplierName, companyName, requirementTitle string, reason string) error {
	s.log("revision_requested", email, requirementTitle, map[string]string{"reason": reason})
	return nil
}

func (s *MockNotificationService) SendCheckFixLinked(ctx context.Context, email, supplierName, domain string) error {
	s.log("checkfix_linked", email, "CheckFix linked", map[string]string{"domain": domain})
	return nil
}

func (s *MockNotificationService) SendCheckFixVerified(ctx context.Context, email, supplierName string, grade string, passed bool) error {
	s.log("checkfix_verified", email, fmt.Sprintf("Grade %s", grade), map[string]string{"passed": fmt.Sprintf("%v", passed)})
	return nil
}

// Ensure MockNotificationService implements NotificationService
var _ NotificationService = (*MockNotificationService)(nil)
var _ NotificationService = (*notificationService)(nil)

// Ensure models.AuditLog is referenced for compilation check
var _ = models.AuditLog{}
