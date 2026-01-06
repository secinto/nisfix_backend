// Package repository provides data access layer factories
// #IMPLEMENTATION_DECISION: Factory functions wrap raw MongoDB constructors for our database.Client
package repository

import (
	"github.com/checkfix-tools/nisfix_backend/internal/database"
)

// NewUserRepository creates a new user repository using our database client
func NewUserRepository(client *database.Client) UserRepository {
	return NewMongoUserRepository(client.Database())
}

// NewOrganizationRepository creates a new organization repository using our database client
func NewOrganizationRepository(client *database.Client) OrganizationRepository {
	return NewMongoOrganizationRepository(client.Database())
}

// NewSecureLinkRepository creates a new secure link repository using our database client
func NewSecureLinkRepository(client *database.Client) SecureLinkRepository {
	return NewMongoSecureLinkRepository(client.Database())
}

// NewQuestionnaireTemplateRepository creates a new questionnaire template repository
func NewQuestionnaireTemplateRepository(client *database.Client) QuestionnaireTemplateRepository {
	return NewMongoQuestionnaireTemplateRepository(client.Database())
}

// NewQuestionnaireRepository creates a new questionnaire repository
func NewQuestionnaireRepository(client *database.Client) QuestionnaireRepository {
	return NewMongoQuestionnaireRepository(client.Database())
}

// NewQuestionRepository creates a new question repository
func NewQuestionRepository(client *database.Client) QuestionRepository {
	return NewMongoQuestionRepository(client.Database())
}

// NewRelationshipRepository creates a new relationship repository
func NewRelationshipRepository(client *database.Client) RelationshipRepository {
	return NewMongoRelationshipRepository(client.Database())
}

// NewRequirementRepository creates a new requirement repository
func NewRequirementRepository(client *database.Client) RequirementRepository {
	return NewMongoRequirementRepository(client.Database())
}

// NewResponseRepository creates a new response repository
func NewResponseRepository(client *database.Client) ResponseRepository {
	return NewMongoResponseRepository(client.Database())
}

// NewSubmissionRepository creates a new submission repository
func NewSubmissionRepository(client *database.Client) SubmissionRepository {
	return NewMongoSubmissionRepository(client.Database())
}

// NewVerificationRepository creates a new verification repository
func NewVerificationRepository(client *database.Client) VerificationRepository {
	return NewMongoVerificationRepository(client.Database())
}

// NewAuditRepository creates a new audit repository
func NewAuditRepository(client *database.Client) AuditRepository {
	return NewMongoAuditRepository(client.Database())
}
