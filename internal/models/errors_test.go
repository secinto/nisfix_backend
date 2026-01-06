package models

import (
	"testing"
)

func TestIsNotFoundError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"ErrUserNotFound", ErrUserNotFound, true},
		{"ErrOrganizationNotFound", ErrOrganizationNotFound, true},
		{"ErrSecureLinkNotFound", ErrSecureLinkNotFound, true},
		{"ErrRelationshipNotFound", ErrRelationshipNotFound, true},
		{"ErrQuestionnaireNotFound", ErrQuestionnaireNotFound, true},
		{"ErrQuestionNotFound", ErrQuestionNotFound, true},
		{"ErrTemplateNotFound", ErrTemplateNotFound, true},
		{"ErrRequirementNotFound", ErrRequirementNotFound, true},
		{"ErrResponseNotFound", ErrResponseNotFound, true},
		{"ErrSubmissionNotFound", ErrSubmissionNotFound, true},
		{"ErrVerificationNotFound", ErrVerificationNotFound, true},
		{"ErrAuditLogNotFound", ErrAuditLogNotFound, true},
		{"Non-NotFound error", ErrInvalidStatusTransition, false},
		{"nil error", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsNotFoundError(tt.err); got != tt.expected {
				t.Errorf("IsNotFoundError() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestIsValidationError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"ErrInvalidInput", ErrInvalidInput, true},
		{"ErrInvalidStatusTransition", ErrInvalidStatusTransition, true},
		{"ErrInvalidQuestionType", ErrInvalidQuestionType, true},
		{"Non-validation error", ErrUserNotFound, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidationError(tt.err); got != tt.expected {
				t.Errorf("IsValidationError() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestIsAuthError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"ErrUnauthorized", ErrUnauthorized, true},
		{"ErrForbidden", ErrForbidden, true},
		{"ErrSecureLinkExpired", ErrSecureLinkExpired, true},
		{"Non-auth error", ErrUserNotFound, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsAuthError(tt.err); got != tt.expected {
				t.Errorf("IsAuthError() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestIsConflictError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"ErrAlreadyExists", ErrAlreadyExists, true},
		{"ErrEmailAlreadyExists", ErrEmailAlreadyExists, true},
		{"ErrRelationshipExists", ErrRelationshipExists, true},
		{"Non-conflict error", ErrUserNotFound, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsConflictError(tt.err); got != tt.expected {
				t.Errorf("IsConflictError() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestErrorMessages(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		contains string
	}{
		{"ErrUserNotFound", ErrUserNotFound, "user not found"},
		{"ErrOrganizationNotFound", ErrOrganizationNotFound, "organization not found"},
		{"ErrInvalidStatusTransition", ErrInvalidStatusTransition, "invalid status transition"},
		{"ErrSecureLinkExpired", ErrSecureLinkExpired, "secure link has expired"},
		{"ErrSecureLinkUsed", ErrSecureLinkUsed, "secure link has already been used"},
		{"ErrEmailAlreadyExists", ErrEmailAlreadyExists, "email already exists"},
		{"ErrRelationshipExists", ErrRelationshipExists, "relationship already exists"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err.Error() != tt.contains {
				t.Errorf("Error() = %v, want %v", tt.err.Error(), tt.contains)
			}
		})
	}
}
