// Package services provides business logic implementations.
// mail_service.go implements email delivery via mailsendAPI following checkfix_backend patterns.
package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/checkfix-tools/nisfix_backend/internal/config"
)

// TemplateEmailRequest represents a template-based email request to mailsendAPI.
// #INTEGRATION_POINT: Maps to POST /email/template endpoint
type TemplateEmailRequest struct {
	Recipient  string                 `json:"recipient"`
	Subject    string                 `json:"subject"`
	Template   string                 `json:"template"`
	Variables  map[string]interface{} `json:"variables"`
	Project    string                 `json:"project,omitempty"`
	SenderName string                 `json:"sender_name,omitempty"`
}

// EmailResponse represents the API response after sending an email.
type EmailResponse struct {
	Message     string `json:"message"`
	ReceptionID string `json:"reception_id"`
}

// MailErrorResponse represents an error response from the mail API.
type MailErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// HTTPMailService implements MailService using HTTP calls to mailsendAPI.
// #INTEGRATION_POINT: Real mail service for production
type HTTPMailService struct {
	config *config.MailConfig
	client *http.Client
}

// NewHTTPMailService creates a new HTTP mail service.
func NewHTTPMailService(cfg *config.MailConfig) *HTTPMailService {
	return &HTTPMailService{
		config: cfg,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// SendMagicLink sends a magic link email via mailsendAPI template.
func (m *HTTPMailService) SendMagicLink(ctx context.Context, email, name, magicLink string) error {
	// Default to English template
	template := m.config.SecureLinkMailEN
	subject := "Your NisFix Login Link"

	variables := map[string]interface{}{
		"secure_link": magicLink,
	}

	return m.sendTemplateEmail(ctx, email, template, subject, variables)
}

// SendInvitation sends a supplier invitation email via mailsendAPI template.
func (m *HTTPMailService) SendInvitation(ctx context.Context, email, companyName, inviteLink string) error {
	// Default to English template
	template := m.config.InviteSupplierEN
	subject := fmt.Sprintf("%s has invited you to NisFix", companyName)

	variables := map[string]interface{}{
		"invite_link":  inviteLink,
		"company_name": companyName,
	}

	return m.sendTemplateEmail(ctx, email, template, subject, variables)
}

// sendTemplateEmail sends a template-based email to mailsendAPI.
func (m *HTTPMailService) sendTemplateEmail(ctx context.Context, recipient, template, subject string, variables map[string]interface{}) error {
	req := TemplateEmailRequest{
		Recipient:  recipient,
		Subject:    subject,
		Template:   template,
		Variables:  variables,
		Project:    m.config.Project,
		SenderName: m.config.SenderName,
	}

	url := m.config.BaseURL + "/email/template"

	reqBody, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal email request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(reqBody))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", m.config.APIKey)

	log.Printf("[MAIL] Sending template email: recipient=%s, template=%s, subject=%s", recipient, template, subject)

	resp, err := m.client.Do(httpReq)
	if err != nil {
		log.Printf("[MAIL] HTTP request failed: %v", err)
		return fmt.Errorf("mail API request failed: %w", err)
	}
	defer resp.Body.Close()

	// mailsendAPI returns 202 Accepted on success
	if resp.StatusCode != http.StatusAccepted {
		bodyBytes, _ := io.ReadAll(resp.Body)

		var errorResp MailErrorResponse
		if err := json.Unmarshal(bodyBytes, &errorResp); err == nil {
			log.Printf("[MAIL] API error (status %d): %s - %s", resp.StatusCode, errorResp.Error, errorResp.Message)
			return fmt.Errorf("mail API error: %s - %s", errorResp.Error, errorResp.Message)
		}

		log.Printf("[MAIL] API error (status %d): %s", resp.StatusCode, string(bodyBytes))
		return fmt.Errorf("mail API returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var emailResp EmailResponse
	if err := json.NewDecoder(resp.Body).Decode(&emailResp); err != nil {
		log.Printf("[MAIL] Failed to decode success response: %v", err)
		return fmt.Errorf("failed to decode mail API response: %w", err)
	}

	log.Printf("[MAIL] Email sent successfully: recipient=%s, reception_id=%s", recipient, emailResp.ReceptionID)
	return nil
}

