package models

import "errors"

// Model validation and operation errors
var (
	// General errors
	ErrNotFound                = errors.New("resource not found")
	ErrAlreadyExists           = errors.New("resource already exists")
	ErrInvalidInput            = errors.New("invalid input")
	ErrUnauthorized            = errors.New("unauthorized")
	ErrForbidden               = errors.New("forbidden")
	ErrInvalidStatusTransition = errors.New("invalid status transition")

	// Organization errors
	ErrOrganizationNotFound    = errors.New("organization not found")
	ErrOrganizationDeleted     = errors.New("organization has been deleted")
	ErrInvalidOrganizationType = errors.New("invalid organization type")
	ErrSlugAlreadyExists       = errors.New("organization slug already exists")
	ErrDomainAlreadyExists     = errors.New("domain already exists")

	// User errors
	ErrUserNotFound       = errors.New("user not found")
	ErrUserDeleted        = errors.New("user has been deleted")
	ErrUserInactive       = errors.New("user is inactive")
	ErrEmailAlreadyExists = errors.New("email already exists")
	ErrInvalidUserRole    = errors.New("invalid user role")

	// Secure link errors
	ErrSecureLinkNotFound = errors.New("secure link not found")
	ErrSecureLinkExpired  = errors.New("secure link has expired")
	ErrSecureLinkUsed     = errors.New("secure link has already been used")
	ErrSecureLinkInvalid  = errors.New("secure link is invalid")

	// Questionnaire template errors
	ErrTemplateNotFound         = errors.New("questionnaire template not found")
	ErrTemplateNotEditable      = errors.New("template cannot be edited")
	ErrTemplateNotDeletable     = errors.New("template cannot be deleted")
	ErrTemplateNotOwnedByUser   = errors.New("template not owned by user")
	ErrTemplateAlreadyPublished = errors.New("template is already published")
	ErrTemplateNotPublished     = errors.New("template is not published")
	ErrTemplateInUse            = errors.New("template is in use and cannot be modified")
	ErrTemplateInvalidFormat    = errors.New("invalid template format")
	ErrTemplateMissingFields    = errors.New("template missing required fields")
	ErrTemplateInvalidVisibility = errors.New("invalid template visibility")

	// Questionnaire errors
	ErrQuestionnaireNotFound     = errors.New("questionnaire not found")
	ErrQuestionnaireNotDraft     = errors.New("questionnaire is not in draft status")
	ErrQuestionnaireNotPublished = errors.New("questionnaire is not published")
	ErrQuestionnaireNotEditable  = errors.New("questionnaire cannot be edited (not draft)")
	ErrQuestionnaireNotDeletable = errors.New("questionnaire cannot be deleted (not draft)")

	// Question errors
	ErrQuestionNotFound       = errors.New("question not found")
	ErrInvalidQuestionType    = errors.New("invalid question type")
	ErrMissingQuestionOptions = errors.New("choice questions require options")
	ErrInvalidOptionID        = errors.New("invalid option ID")
	ErrInvalidAnswerFormat    = errors.New("invalid answer format")

	// Relationship errors
	ErrRelationshipNotFound       = errors.New("relationship not found")
	ErrRelationshipExists         = errors.New("relationship already exists")
	ErrRelationshipNotActive      = errors.New("relationship is not active")
	ErrRelationshipTerminated     = errors.New("relationship has been terminated")
	ErrCannotAssignToRelationship = errors.New("cannot assign requirements to this relationship")

	// Requirement errors
	ErrRequirementNotFound       = errors.New("requirement not found")
	ErrRequirementExpired        = errors.New("requirement has expired")
	ErrRequirementNotPending     = errors.New("requirement is not pending")
	ErrRequirementNotSubmittable = errors.New("requirement cannot be submitted")
	ErrRequirementNotReviewable  = errors.New("requirement cannot be reviewed")

	// Response errors
	ErrResponseNotFound         = errors.New("response not found")
	ErrResponseAlreadyExists    = errors.New("response already exists for this requirement")
	ErrResponseNotSubmitted     = errors.New("response has not been submitted")
	ErrResponseAlreadySubmitted = errors.New("response has already been submitted")

	// Submission errors
	ErrSubmissionNotFound      = errors.New("submission not found")
	ErrSubmissionAlreadyExists = errors.New("submission already exists")

	// Verification errors
	ErrVerificationNotFound = errors.New("verification not found")
	ErrVerificationExpired  = errors.New("verification has expired")
	ErrVerificationInvalid  = errors.New("verification is invalid")
	ErrDomainMismatch       = errors.New("domain does not match")
	ErrGradeNotMet          = errors.New("minimum grade requirement not met")
	ErrReportTooOld         = errors.New("report is too old")

	// Audit log errors
	ErrAuditLogNotFound = errors.New("audit log not found")
)

// IsNotFoundError returns true if the error is a not found error
func IsNotFoundError(err error) bool {
	return errors.Is(err, ErrNotFound) ||
		errors.Is(err, ErrOrganizationNotFound) ||
		errors.Is(err, ErrUserNotFound) ||
		errors.Is(err, ErrSecureLinkNotFound) ||
		errors.Is(err, ErrTemplateNotFound) ||
		errors.Is(err, ErrQuestionnaireNotFound) ||
		errors.Is(err, ErrQuestionNotFound) ||
		errors.Is(err, ErrRelationshipNotFound) ||
		errors.Is(err, ErrRequirementNotFound) ||
		errors.Is(err, ErrResponseNotFound) ||
		errors.Is(err, ErrSubmissionNotFound) ||
		errors.Is(err, ErrVerificationNotFound) ||
		errors.Is(err, ErrAuditLogNotFound)
}

// IsValidationError returns true if the error is a validation error
func IsValidationError(err error) bool {
	return errors.Is(err, ErrInvalidInput) ||
		errors.Is(err, ErrInvalidStatusTransition) ||
		errors.Is(err, ErrInvalidOrganizationType) ||
		errors.Is(err, ErrInvalidUserRole) ||
		errors.Is(err, ErrInvalidQuestionType) ||
		errors.Is(err, ErrMissingQuestionOptions) ||
		errors.Is(err, ErrInvalidOptionID) ||
		errors.Is(err, ErrInvalidAnswerFormat) ||
		errors.Is(err, ErrTemplateInvalidFormat) ||
		errors.Is(err, ErrTemplateMissingFields) ||
		errors.Is(err, ErrTemplateInvalidVisibility)
}

// IsAuthError returns true if the error is an authentication/authorization error
func IsAuthError(err error) bool {
	return errors.Is(err, ErrUnauthorized) ||
		errors.Is(err, ErrForbidden) ||
		errors.Is(err, ErrUserInactive) ||
		errors.Is(err, ErrUserDeleted) ||
		errors.Is(err, ErrSecureLinkExpired) ||
		errors.Is(err, ErrSecureLinkUsed) ||
		errors.Is(err, ErrSecureLinkInvalid)
}

// IsConflictError returns true if the error is a conflict/duplicate error
func IsConflictError(err error) bool {
	return errors.Is(err, ErrAlreadyExists) ||
		errors.Is(err, ErrSlugAlreadyExists) ||
		errors.Is(err, ErrDomainAlreadyExists) ||
		errors.Is(err, ErrEmailAlreadyExists) ||
		errors.Is(err, ErrRelationshipExists) ||
		errors.Is(err, ErrResponseAlreadyExists) ||
		errors.Is(err, ErrSubmissionAlreadyExists)
}
