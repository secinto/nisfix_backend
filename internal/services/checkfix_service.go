// Package services provides business logic implementations.
package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"

	"github.com/checkfix-tools/nisfix_backend/internal/models"
	"github.com/checkfix-tools/nisfix_backend/internal/repository"
)

// Custom errors for CheckFix service
var (
	ErrCheckFixAPIError       = errors.New("checkfix API error")
	ErrCheckFixNotLinked      = errors.New("checkfix account not linked")
	ErrCheckFixReportNotFound = errors.New("checkfix report not found")
	ErrCheckFixDomainMismatch = errors.New("domain does not match organization")
	ErrCheckFixReportExpired  = errors.New("checkfix report is too old")
	ErrCheckFixGradeNotMet    = errors.New("checkfix grade does not meet requirement")
	ErrVerificationNotFound   = errors.New("verification not found")
)

// CheckFixAPIClient defines the interface for CheckFix API operations
// #INTEGRATION_POINT: External CheckFix API for report verification
type CheckFixAPIClient interface {
	// VerifyReport verifies a report by its hash and returns the report data
	VerifyReport(ctx context.Context, reportHash string) (*CheckFixReportData, error)

	// GetAccountDomain gets the domain associated with a CheckFix account
	GetAccountDomain(ctx context.Context, accountID string) (string, error)

	// ValidateAccountAccess validates that an account ID is valid and accessible
	ValidateAccountAccess(ctx context.Context, accountID string) (bool, error)
}

// CheckFixReportData represents data from a CheckFix report
type CheckFixReportData struct {
	ReportHash       string                 `json:"report_hash"`
	Domain           string                 `json:"domain"`
	ReportDate       time.Time              `json:"report_date"`
	OverallGrade     string                 `json:"overall_grade"`
	OverallScore     int                    `json:"overall_score"`
	CategoryGrades   []models.CategoryGrade `json:"category_grades"`
	CriticalFindings int                    `json:"critical_findings"`
	HighFindings     int                    `json:"high_findings"`
	MediumFindings   int                    `json:"medium_findings"`
	LowFindings      int                    `json:"low_findings"`
}

// CheckFixService handles CheckFix integration business logic
// #INTEGRATION_POINT: Used by handlers for CheckFix linking and verification
type CheckFixService interface {
	// LinkAccount links a supplier organization to their CheckFix account
	LinkAccount(ctx context.Context, supplierID primitive.ObjectID, accountID string) error

	// UnlinkAccount removes the CheckFix link from a supplier
	UnlinkAccount(ctx context.Context, supplierID primitive.ObjectID) error

	// GetLinkStatus gets the current CheckFix link status for a supplier
	GetLinkStatus(ctx context.Context, supplierID primitive.ObjectID) (*CheckFixLinkStatus, error)

	// VerifyReport verifies a CheckFix report and stores the verification
	VerifyReport(ctx context.Context, supplierID, responseID primitive.ObjectID, reportHash string) (*models.CheckFixVerification, error)

	// GetVerification retrieves a verification by response ID
	GetVerification(ctx context.Context, responseID primitive.ObjectID) (*models.CheckFixVerification, error)

	// GetLatestVerification gets the most recent verification for a supplier
	GetLatestVerification(ctx context.Context, supplierID primitive.ObjectID) (*models.CheckFixVerification, error)

	// CheckRequirementMet checks if a CheckFix requirement is met
	CheckRequirementMet(ctx context.Context, responseID primitive.ObjectID, minimumGrade models.CheckFixGrade, maxReportAgeDays int) (bool, error)

	// SubmitCheckFixResponse submits a CheckFix verification as a response
	SubmitCheckFixResponse(ctx context.Context, requirementID, supplierID primitive.ObjectID, reportHash string) (*CheckFixSubmissionResult, error)
}

// CheckFixLinkStatus represents the current CheckFix link status
type CheckFixLinkStatus struct {
	IsLinked         bool                         `json:"is_linked"`
	AccountID        string                       `json:"account_id,omitempty"`
	Domain           string                       `json:"domain,omitempty"`
	LinkedAt         *time.Time                   `json:"linked_at,omitempty"`
	LatestGrade      *models.CheckFixGrade        `json:"latest_grade,omitempty"`
	LatestVerifiedAt *time.Time                   `json:"latest_verified_at,omitempty"`
	Verification     *models.CheckFixVerification `json:"verification,omitempty"`
}

// CheckFixSubmissionResult represents the result of a CheckFix submission
type CheckFixSubmissionResult struct {
	Verification *models.CheckFixVerification `json:"verification"`
	Response     *models.SupplierResponse     `json:"response"`
	Requirement  *models.Requirement          `json:"requirement"`
	Passed       bool                         `json:"passed"`
	Grade        models.CheckFixGrade         `json:"grade"`
	Message      string                       `json:"message"`
}

// checkFixService implements CheckFixService
type checkFixService struct {
	apiClient        CheckFixAPIClient
	verificationRepo repository.VerificationRepository
	responseRepo     repository.ResponseRepository
	requirementRepo  repository.RequirementRepository
	orgRepo          repository.OrganizationRepository
}

// NewCheckFixService creates a new CheckFix service
func NewCheckFixService(
	apiClient CheckFixAPIClient,
	verificationRepo repository.VerificationRepository,
	responseRepo repository.ResponseRepository,
	requirementRepo repository.RequirementRepository,
	orgRepo repository.OrganizationRepository,
) CheckFixService {
	return &checkFixService{
		apiClient:        apiClient,
		verificationRepo: verificationRepo,
		responseRepo:     responseRepo,
		requirementRepo:  requirementRepo,
		orgRepo:          orgRepo,
	}
}

// LinkAccount links a supplier organization to their CheckFix account
// #BUSINESS_RULE: Validates account access before linking
func (s *checkFixService) LinkAccount(ctx context.Context, supplierID primitive.ObjectID, accountID string) error {
	// Get organization
	org, err := s.orgRepo.GetByID(ctx, supplierID)
	if err != nil {
		return fmt.Errorf("failed to get organization: %w", err)
	}

	if !org.IsSupplier() {
		return errors.New("only suppliers can link CheckFix accounts")
	}

	// Validate account access
	valid, err := s.apiClient.ValidateAccountAccess(ctx, accountID)
	if err != nil {
		return fmt.Errorf("failed to validate account: %w", err)
	}
	if !valid {
		return errors.New("invalid CheckFix account ID")
	}

	// Get domain from CheckFix
	domain, err := s.apiClient.GetAccountDomain(ctx, accountID)
	if err != nil {
		return fmt.Errorf("failed to get account domain: %w", err)
	}

	// Update organization
	now := time.Now().UTC()
	org.CheckFixAccountID = accountID
	org.CheckFixLinkedAt = &now
	org.Domain = domain

	if err := s.orgRepo.Update(ctx, org); err != nil {
		return fmt.Errorf("failed to update organization: %w", err)
	}

	return nil
}

// UnlinkAccount removes the CheckFix link from a supplier
func (s *checkFixService) UnlinkAccount(ctx context.Context, supplierID primitive.ObjectID) error {
	org, err := s.orgRepo.GetByID(ctx, supplierID)
	if err != nil {
		return fmt.Errorf("failed to get organization: %w", err)
	}

	org.CheckFixAccountID = ""
	org.CheckFixLinkedAt = nil

	if err := s.orgRepo.Update(ctx, org); err != nil {
		return fmt.Errorf("failed to update organization: %w", err)
	}

	return nil
}

// GetLinkStatus gets the current CheckFix link status for a supplier
func (s *checkFixService) GetLinkStatus(ctx context.Context, supplierID primitive.ObjectID) (*CheckFixLinkStatus, error) {
	org, err := s.orgRepo.GetByID(ctx, supplierID)
	if err != nil {
		return nil, fmt.Errorf("failed to get organization: %w", err)
	}

	status := &CheckFixLinkStatus{
		IsLinked:  org.HasCheckFixLinked(),
		AccountID: org.CheckFixAccountID,
		Domain:    org.Domain,
		LinkedAt:  org.CheckFixLinkedAt,
	}

	// Get latest verification if linked
	if status.IsLinked {
		verification, err := s.verificationRepo.GetLatestBySupplier(ctx, supplierID)
		if err == nil && verification != nil {
			status.LatestGrade = &verification.OverallGrade
			status.LatestVerifiedAt = &verification.VerifiedAt
			status.Verification = verification
		}
	}

	return status, nil
}

// VerifyReport verifies a CheckFix report and stores the verification
// #BUSINESS_RULE: Domain must match organization domain
func (s *checkFixService) VerifyReport(ctx context.Context, supplierID, responseID primitive.ObjectID, reportHash string) (*models.CheckFixVerification, error) {
	// Get organization
	org, err := s.orgRepo.GetByID(ctx, supplierID)
	if err != nil {
		return nil, fmt.Errorf("failed to get organization: %w", err)
	}

	if !org.HasCheckFixLinked() {
		return nil, ErrCheckFixNotLinked
	}

	// Verify report via API
	reportData, err := s.apiClient.VerifyReport(ctx, reportHash)
	if err != nil {
		return nil, fmt.Errorf("failed to verify report: %w", err)
	}

	// Check domain match
	domainMatch := strings.EqualFold(reportData.Domain, org.Domain)

	// Create verification
	verification := &models.CheckFixVerification{
		ResponseID:        responseID,
		SupplierID:        supplierID,
		Domain:            org.Domain,
		VerifiedDomain:    reportData.Domain,
		DomainMatch:       domainMatch,
		ReportHash:        reportData.ReportHash,
		ReportDate:        reportData.ReportDate,
		OverallGrade:      models.CheckFixGrade(reportData.OverallGrade),
		OverallScore:      reportData.OverallScore,
		CategoryGrades:    reportData.CategoryGrades,
		CriticalFindings:  reportData.CriticalFindings,
		HighFindings:      reportData.HighFindings,
		MediumFindings:    reportData.MediumFindings,
		LowFindings:       reportData.LowFindings,
		VerificationValid: true,
	}
	verification.BeforeCreate()

	if err := s.verificationRepo.Create(ctx, verification); err != nil {
		return nil, fmt.Errorf("failed to create verification: %w", err)
	}

	return verification, nil
}

// GetVerification retrieves a verification by response ID
func (s *checkFixService) GetVerification(ctx context.Context, responseID primitive.ObjectID) (*models.CheckFixVerification, error) {
	verification, err := s.verificationRepo.GetByResponse(ctx, responseID)
	if err != nil {
		if errors.Is(err, models.ErrVerificationNotFound) {
			return nil, ErrVerificationNotFound
		}
		return nil, fmt.Errorf("failed to get verification: %w", err)
	}
	return verification, nil
}

// GetLatestVerification gets the most recent verification for a supplier
func (s *checkFixService) GetLatestVerification(ctx context.Context, supplierID primitive.ObjectID) (*models.CheckFixVerification, error) {
	verification, err := s.verificationRepo.GetLatestBySupplier(ctx, supplierID)
	if err != nil {
		if errors.Is(err, models.ErrVerificationNotFound) {
			return nil, ErrVerificationNotFound
		}
		return nil, fmt.Errorf("failed to get verification: %w", err)
	}
	return verification, nil
}

// CheckRequirementMet checks if a CheckFix requirement is met
func (s *checkFixService) CheckRequirementMet(ctx context.Context, responseID primitive.ObjectID, minimumGrade models.CheckFixGrade, maxReportAgeDays int) (bool, error) {
	verification, err := s.verificationRepo.GetByResponse(ctx, responseID)
	if err != nil {
		return false, err
	}

	return verification.PassesRequirement(minimumGrade, maxReportAgeDays), nil
}

// SubmitCheckFixResponse submits a CheckFix verification as a response
// #BUSINESS_RULE: Creates response, verifies report, updates requirement status
func (s *checkFixService) SubmitCheckFixResponse(ctx context.Context, requirementID, supplierID primitive.ObjectID, reportHash string) (*CheckFixSubmissionResult, error) {
	// Get requirement
	requirement, err := s.requirementRepo.GetByID(ctx, requirementID)
	if err != nil {
		return nil, ErrRequirementNotFound
	}

	// Verify ownership
	if requirement.SupplierID != supplierID {
		return nil, ErrRequirementNotFound
	}

	// Check requirement type
	if !requirement.IsCheckFixRequirement() {
		return nil, errors.New("requirement is not a CheckFix requirement")
	}

	// Get or create response
	response, err := s.responseRepo.GetByRequirement(ctx, requirementID)
	if err != nil {
		// Create new response
		response = &models.SupplierResponse{
			RequirementID: requirementID,
			SupplierID:    supplierID,
		}
		response.BeforeCreate()
		if err := s.responseRepo.Create(ctx, response); err != nil {
			return nil, fmt.Errorf("failed to create response: %w", err)
		}
	}

	// Verify the report
	verification, err := s.VerifyReport(ctx, supplierID, response.ID, reportHash)
	if err != nil {
		return nil, err
	}

	// Determine if passed
	minimumGrade := models.CheckFixGradeC
	if requirement.MinimumGrade != nil {
		minimumGrade = models.CheckFixGrade(*requirement.MinimumGrade)
	}
	maxAgeDays := 90
	if requirement.MaxReportAgeDays != nil {
		maxAgeDays = *requirement.MaxReportAgeDays
	}

	passed := verification.PassesRequirement(minimumGrade, maxAgeDays)

	// Update response
	gradeStr := string(verification.OverallGrade)
	response.Grade = &gradeStr
	response.Passed = &passed
	response.Submit()

	if err := s.responseRepo.Update(ctx, response); err != nil {
		return nil, fmt.Errorf("failed to update response: %w", err)
	}

	// Update requirement status
	if err := requirement.Submit(supplierID); err == nil {
		_ = s.requirementRepo.Update(ctx, requirement)
	}

	// Build message
	message := "CheckFix verification successful"
	if !passed {
		if !verification.DomainMatch {
			message = "Domain does not match organization"
		} else if !verification.MeetsMinimumGrade(minimumGrade) {
			message = fmt.Sprintf("Grade %s does not meet minimum %s", verification.OverallGrade, minimumGrade)
		} else if verification.IsReportTooOld(maxAgeDays) {
			message = fmt.Sprintf("Report is %d days old, maximum is %d days", verification.ReportAgeDays(), maxAgeDays)
		}
	}

	return &CheckFixSubmissionResult{
		Verification: verification,
		Response:     response,
		Requirement:  requirement,
		Passed:       passed,
		Grade:        verification.OverallGrade,
		Message:      message,
	}, nil
}

// HTTPCheckFixAPIClient implements CheckFixAPIClient using HTTP
type HTTPCheckFixAPIClient struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

// NewHTTPCheckFixAPIClient creates a new HTTP-based CheckFix API client
func NewHTTPCheckFixAPIClient(baseURL, apiKey string) *HTTPCheckFixAPIClient {
	return &HTTPCheckFixAPIClient{
		baseURL: strings.TrimSuffix(baseURL, "/"),
		apiKey:  apiKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// VerifyReport verifies a report via the CheckFix API
func (c *HTTPCheckFixAPIClient) VerifyReport(ctx context.Context, reportHash string) (*CheckFixReportData, error) {
	url := fmt.Sprintf("%s/api/v1/reports/%s/verify", c.baseURL, reportHash)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, ErrCheckFixAPIError
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrCheckFixReportNotFound
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("%w: %s", ErrCheckFixAPIError, string(body))
	}

	var data CheckFixReportData
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &data, nil
}

// GetAccountDomain gets the domain for a CheckFix account
func (c *HTTPCheckFixAPIClient) GetAccountDomain(ctx context.Context, accountID string) (string, error) {
	url := fmt.Sprintf("%s/api/v1/accounts/%s", c.baseURL, accountID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", ErrCheckFixAPIError
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", ErrCheckFixAPIError
	}

	var data struct {
		Domain string `json:"domain"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return "", err
	}

	return data.Domain, nil
}

// ValidateAccountAccess validates account access
func (c *HTTPCheckFixAPIClient) ValidateAccountAccess(ctx context.Context, accountID string) (bool, error) {
	url := fmt.Sprintf("%s/api/v1/accounts/%s/validate", c.baseURL, accountID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return false, err
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false, nil // Treat network errors as invalid
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK, nil
}

// MockCheckFixAPIClient is a mock implementation for development/testing
type MockCheckFixAPIClient struct {
	MockDomain string
	MockGrade  string
}

// NewMockCheckFixAPIClient creates a mock CheckFix API client
func NewMockCheckFixAPIClient() *MockCheckFixAPIClient {
	return &MockCheckFixAPIClient{
		MockDomain: "example.com",
		MockGrade:  "B",
	}
}

// VerifyReport returns mock report data
func (c *MockCheckFixAPIClient) VerifyReport(ctx context.Context, reportHash string) (*CheckFixReportData, error) {
	return &CheckFixReportData{
		ReportHash:       reportHash,
		Domain:           c.MockDomain,
		ReportDate:       time.Now().AddDate(0, 0, -7), // 7 days ago
		OverallGrade:     c.MockGrade,
		OverallScore:     75,
		CategoryGrades:   []models.CategoryGrade{},
		CriticalFindings: 0,
		HighFindings:     2,
		MediumFindings:   5,
		LowFindings:      10,
	}, nil
}

// GetAccountDomain returns mock domain
func (c *MockCheckFixAPIClient) GetAccountDomain(ctx context.Context, accountID string) (string, error) {
	return c.MockDomain, nil
}

// ValidateAccountAccess always returns true for mock
func (c *MockCheckFixAPIClient) ValidateAccountAccess(ctx context.Context, accountID string) (bool, error) {
	return true, nil
}
