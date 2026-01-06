# Data Architecture Plan - B2B Supplier Security Portal

## Overview

This document defines the complete MongoDB data architecture for the B2B Supplier Security Portal (NisFix). The portal enables Companies to assess their Suppliers' security posture through questionnaires and CheckFix report integration.

#SCHEMA_RATIONALE: MongoDB chosen for flexible schema evolution, embedded documents for denormalization, and native JSON handling with Go driver. Document model suits the hierarchical nature of questionnaires and nested scoring structures.

#DATA_ASSUMPTION: All timestamps stored as UTC. MongoDB BSON dates used for automatic indexing efficiency.

#VOLUME_ASSUMPTION: Initial deployment targets 100-500 organizations, 1K-5K users, scaling to 10K+ organizations over 3 years.

---

## Entity Catalog

### Entity: Organization

**Purpose**: Represents both Company and Supplier entities. Core tenant boundary for multi-tenancy.

```go
type OrganizationType string

const (
    OrganizationTypeCompany  OrganizationType = "COMPANY"
    OrganizationTypeSupplier OrganizationType = "SUPPLIER"
)

type Organization struct {
    ID                primitive.ObjectID `bson:"_id,omitempty" json:"id"`
    Type              OrganizationType   `bson:"type" json:"type"`
    Name              string             `bson:"name" json:"name"`
    Slug              string             `bson:"slug" json:"slug"`
    Domain            string             `bson:"domain,omitempty" json:"domain,omitempty"`
    Description       string             `bson:"description,omitempty" json:"description,omitempty"`

    // Contact Information
    ContactEmail      string             `bson:"contact_email" json:"contact_email"`
    ContactPhone      string             `bson:"contact_phone,omitempty" json:"contact_phone,omitempty"`
    Address           *Address           `bson:"address,omitempty" json:"address,omitempty"`

    // CheckFix Integration (Suppliers only)
    CheckFixAccountID string             `bson:"checkfix_account_id,omitempty" json:"checkfix_account_id,omitempty"`
    CheckFixLinkedAt  *time.Time         `bson:"checkfix_linked_at,omitempty" json:"checkfix_linked_at,omitempty"`

    // Settings
    Settings          OrganizationSettings `bson:"settings" json:"settings"`

    // Audit fields
    CreatedAt         time.Time          `bson:"created_at" json:"created_at"`
    UpdatedAt         time.Time          `bson:"updated_at" json:"updated_at"`
    DeletedAt         *time.Time         `bson:"deleted_at,omitempty" json:"deleted_at,omitempty"`
}

type Address struct {
    Street     string `bson:"street,omitempty" json:"street,omitempty"`
    City       string `bson:"city,omitempty" json:"city,omitempty"`
    PostalCode string `bson:"postal_code,omitempty" json:"postal_code,omitempty"`
    Country    string `bson:"country,omitempty" json:"country,omitempty"`
}

type OrganizationSettings struct {
    DefaultLanguage       string `bson:"default_language" json:"default_language"`
    NotificationsEnabled  bool   `bson:"notifications_enabled" json:"notifications_enabled"`
    ReminderDaysBefore    int    `bson:"reminder_days_before" json:"reminder_days_before"`
}
```

#EXPORT_ENTITY: Organization{id, type, name, slug, domain, contact_email, settings, audit_fields}
#DATA_ASSUMPTION: Slug generated from name, must be URL-safe lowercase alphanumeric with hyphens
#DATA_ASSUMPTION: Domain field populated by supplier when linking CheckFix, used for verification
#NORMALIZATION_DECISION: Address embedded (1:1, rarely queried independently). Settings embedded for atomic reads.
#VOLUME_ASSUMPTION: 100-10K organizations, 50/50 split Company/Supplier initially

---

### Entity: User

**Purpose**: User accounts with role-based access within an organization.

```go
type UserRole string

const (
    UserRoleAdmin  UserRole = "ADMIN"
    UserRoleViewer UserRole = "VIEWER"
)

type User struct {
    ID             primitive.ObjectID `bson:"_id,omitempty" json:"id"`
    Email          string             `bson:"email" json:"email"`
    Name           string             `bson:"name,omitempty" json:"name,omitempty"`
    OrganizationID primitive.ObjectID `bson:"organization_id" json:"organization_id"`
    Role           UserRole           `bson:"role" json:"role"`

    // Status
    IsActive       bool               `bson:"is_active" json:"is_active"`
    LastLoginAt    *time.Time         `bson:"last_login_at,omitempty" json:"last_login_at,omitempty"`

    // Preferences
    Language       string             `bson:"language" json:"language"`
    Timezone       string             `bson:"timezone,omitempty" json:"timezone,omitempty"`

    // Audit fields
    CreatedAt      time.Time          `bson:"created_at" json:"created_at"`
    UpdatedAt      time.Time          `bson:"updated_at" json:"updated_at"`
    DeletedAt      *time.Time         `bson:"deleted_at,omitempty" json:"deleted_at,omitempty"`
}
```

#EXPORT_ENTITY: User{id, email, name, organization_id, role, is_active, audit_fields}
#DATA_ASSUMPTION: Email is unique across entire system (not per organization)
#DATA_ASSUMPTION: Users belong to exactly ONE organization (no multi-org membership)
#CARDINALITY_ASSUMPTION: Organization 1:N Users - One organization has many users
#VOLUME_ASSUMPTION: 5-50 users per organization, 1K-50K total users

---

### Entity: QuestionnaireTemplate

**Purpose**: Pre-defined questionnaire templates (ISO27001, GDPR, NIS2) that companies can clone and customize.

```go
type TemplateCategory string

const (
    TemplateCategoryISO27001  TemplateCategory = "ISO27001"
    TemplateCategoryGDPR      TemplateCategory = "GDPR"
    TemplateCategoryNIS2      TemplateCategory = "NIS2"
    TemplateCategoryCustom    TemplateCategory = "CUSTOM"
)

type QuestionnaireTemplate struct {
    ID             primitive.ObjectID `bson:"_id,omitempty" json:"id"`
    Name           string             `bson:"name" json:"name"`
    Description    string             `bson:"description,omitempty" json:"description,omitempty"`
    Category       TemplateCategory   `bson:"category" json:"category"`
    Version        string             `bson:"version" json:"version"`

    // Ownership
    IsSystem       bool               `bson:"is_system" json:"is_system"`
    CreatedByOrgID *primitive.ObjectID `bson:"created_by_org_id,omitempty" json:"created_by_org_id,omitempty"`

    // Configuration
    DefaultPassingScore int            `bson:"default_passing_score" json:"default_passing_score"`
    EstimatedMinutes    int            `bson:"estimated_minutes" json:"estimated_minutes"`

    // Topics/Sections for organizing questions
    Topics         []TemplateTopic    `bson:"topics" json:"topics"`

    // Metadata
    Tags           []string           `bson:"tags,omitempty" json:"tags,omitempty"`
    UsageCount     int                `bson:"usage_count" json:"usage_count"`

    // Audit fields
    CreatedAt      time.Time          `bson:"created_at" json:"created_at"`
    UpdatedAt      time.Time          `bson:"updated_at" json:"updated_at"`
    PublishedAt    *time.Time         `bson:"published_at,omitempty" json:"published_at,omitempty"`
}

type TemplateTopic struct {
    ID          string `bson:"id" json:"id"`
    Name        string `bson:"name" json:"name"`
    Description string `bson:"description,omitempty" json:"description,omitempty"`
    Order       int    `bson:"order" json:"order"`
}
```

#EXPORT_ENTITY: QuestionnaireTemplate{id, name, category, version, is_system, topics, default_passing_score}
#DATA_ASSUMPTION: System templates are read-only, managed by application deployment
#DATA_ASSUMPTION: Companies can create custom templates (is_system=false, created_by_org_id set)
#NORMALIZATION_DECISION: Topics embedded as they are intrinsic to template structure
#VOLUME_ASSUMPTION: 10-20 system templates, 0-50 custom templates per company

---

### Entity: Questionnaire

**Purpose**: Company-customized instance of a template, ready to be assigned as requirements.

```go
type QuestionnaireStatus string

const (
    QuestionnaireStatusDraft     QuestionnaireStatus = "DRAFT"
    QuestionnaireStatusPublished QuestionnaireStatus = "PUBLISHED"
    QuestionnaireStatusArchived  QuestionnaireStatus = "ARCHIVED"
)

type Questionnaire struct {
    ID               primitive.ObjectID  `bson:"_id,omitempty" json:"id"`
    CompanyID        primitive.ObjectID  `bson:"company_id" json:"company_id"`
    TemplateID       *primitive.ObjectID `bson:"template_id,omitempty" json:"template_id,omitempty"`

    // Basic info
    Name             string              `bson:"name" json:"name"`
    Description      string              `bson:"description,omitempty" json:"description,omitempty"`
    Status           QuestionnaireStatus `bson:"status" json:"status"`
    Version          int                 `bson:"version" json:"version"`

    // Scoring configuration
    PassingScore     int                 `bson:"passing_score" json:"passing_score"`
    ScoringMode      ScoringMode         `bson:"scoring_mode" json:"scoring_mode"`

    // Topics (copied from template, can be customized)
    Topics           []QuestionnaireTopic `bson:"topics" json:"topics"`

    // Statistics (denormalized for dashboard)
    QuestionCount    int                 `bson:"question_count" json:"question_count"`
    MaxPossibleScore int                 `bson:"max_possible_score" json:"max_possible_score"`

    // Audit fields
    CreatedAt        time.Time           `bson:"created_at" json:"created_at"`
    UpdatedAt        time.Time           `bson:"updated_at" json:"updated_at"`
    PublishedAt      *time.Time          `bson:"published_at,omitempty" json:"published_at,omitempty"`
}

type ScoringMode string

const (
    ScoringModePercentage ScoringMode = "PERCENTAGE"
    ScoringModePoints     ScoringMode = "POINTS"
)

type QuestionnaireTopic struct {
    ID          string `bson:"id" json:"id"`
    Name        string `bson:"name" json:"name"`
    Description string `bson:"description,omitempty" json:"description,omitempty"`
    Order       int    `bson:"order" json:"order"`
}
```

#EXPORT_ENTITY: Questionnaire{id, company_id, template_id, name, status, passing_score, topics, question_count}
#DATA_ASSUMPTION: Questionnaires are immutable once published (create new version instead)
#CARDINALITY_ASSUMPTION: Company 1:N Questionnaires - Company owns multiple questionnaires
#CARDINALITY_ASSUMPTION: Template 1:N Questionnaires - Template can spawn many questionnaire instances
#NORMALIZATION_DECISION: QuestionCount denormalized for dashboard performance
#VOLUME_ASSUMPTION: 5-50 questionnaires per company

---

### Entity: Question

**Purpose**: Individual question with options, scoring, and required flag.

```go
type QuestionType string

const (
    QuestionTypeSingleChoice   QuestionType = "SINGLE_CHOICE"
    QuestionTypeMultipleChoice QuestionType = "MULTIPLE_CHOICE"
    QuestionTypeText           QuestionType = "TEXT"
    QuestionTypeYesNo          QuestionType = "YES_NO"
)

type Question struct {
    ID               primitive.ObjectID `bson:"_id,omitempty" json:"id"`
    QuestionnaireID  primitive.ObjectID `bson:"questionnaire_id" json:"questionnaire_id"`
    TopicID          string             `bson:"topic_id" json:"topic_id"`

    // Content
    Text             string             `bson:"text" json:"text"`
    Description      string             `bson:"description,omitempty" json:"description,omitempty"`
    HelpText         string             `bson:"help_text,omitempty" json:"help_text,omitempty"`

    // Type and ordering
    Type             QuestionType       `bson:"type" json:"type"`
    Order            int                `bson:"order" json:"order"`

    // Scoring
    Weight           int                `bson:"weight" json:"weight"`
    MaxPoints        int                `bson:"max_points" json:"max_points"`
    IsMustPass       bool               `bson:"is_must_pass" json:"is_must_pass"`

    // Options (embedded for single/multiple choice)
    Options          []QuestionOption   `bson:"options,omitempty" json:"options,omitempty"`

    // Audit fields
    CreatedAt        time.Time          `bson:"created_at" json:"created_at"`
    UpdatedAt        time.Time          `bson:"updated_at" json:"updated_at"`
}

type QuestionOption struct {
    ID          string `bson:"id" json:"id"`
    Text        string `bson:"text" json:"text"`
    Points      int    `bson:"points" json:"points"`
    IsCorrect   bool   `bson:"is_correct" json:"is_correct"`
    Order       int    `bson:"order" json:"order"`
}
```

#EXPORT_ENTITY: Question{id, questionnaire_id, topic_id, text, type, weight, max_points, is_must_pass, options}
#DATA_ASSUMPTION: Weight defaults to 1, allows emphasizing critical questions
#DATA_ASSUMPTION: IsMustPass questions cause automatic fail regardless of total score
#NORMALIZATION_DECISION: QuestionOptions embedded (never queried independently, always with question)
#CARDINALITY_ASSUMPTION: Questionnaire 1:N Questions - Questionnaire contains many questions
#VOLUME_ASSUMPTION: 10-100 questions per questionnaire, 2-6 options per question

---

### Entity: CompanySupplierRelationship

**Purpose**: Tracks the business relationship between a Company and a Supplier, including invitation state.

```go
type RelationshipStatus string

const (
    RelationshipStatusPending    RelationshipStatus = "PENDING"
    RelationshipStatusActive     RelationshipStatus = "ACTIVE"
    RelationshipStatusRejected   RelationshipStatus = "REJECTED"
    RelationshipStatusSuspended  RelationshipStatus = "SUSPENDED"
    RelationshipStatusTerminated RelationshipStatus = "TERMINATED"
)

type SupplierClassification string

const (
    SupplierClassificationCritical SupplierClassification = "CRITICAL"
    SupplierClassificationHigh     SupplierClassification = "HIGH"
    SupplierClassificationMedium   SupplierClassification = "MEDIUM"
    SupplierClassificationLow      SupplierClassification = "LOW"
)

type CompanySupplierRelationship struct {
    ID               primitive.ObjectID      `bson:"_id,omitempty" json:"id"`
    CompanyID        primitive.ObjectID      `bson:"company_id" json:"company_id"`
    SupplierID       *primitive.ObjectID     `bson:"supplier_id,omitempty" json:"supplier_id,omitempty"`

    // Invitation details
    InvitedEmail     string                  `bson:"invited_email" json:"invited_email"`
    InvitedByUserID  primitive.ObjectID      `bson:"invited_by_user_id" json:"invited_by_user_id"`
    InvitedAt        time.Time               `bson:"invited_at" json:"invited_at"`

    // Status tracking
    Status           RelationshipStatus      `bson:"status" json:"status"`
    StatusHistory    []StatusChange          `bson:"status_history" json:"status_history"`

    // Classification
    Classification   SupplierClassification  `bson:"classification" json:"classification"`
    Notes            string                  `bson:"notes,omitempty" json:"notes,omitempty"`

    // Service details
    ServicesProvided []string                `bson:"services_provided,omitempty" json:"services_provided,omitempty"`
    ContractRef      string                  `bson:"contract_ref,omitempty" json:"contract_ref,omitempty"`

    // Response tracking (denormalized)
    AcceptedAt       *time.Time              `bson:"accepted_at,omitempty" json:"accepted_at,omitempty"`
    RejectedAt       *time.Time              `bson:"rejected_at,omitempty" json:"rejected_at,omitempty"`

    // Audit fields
    CreatedAt        time.Time               `bson:"created_at" json:"created_at"`
    UpdatedAt        time.Time               `bson:"updated_at" json:"updated_at"`
}

type StatusChange struct {
    FromStatus RelationshipStatus `bson:"from_status" json:"from_status"`
    ToStatus   RelationshipStatus `bson:"to_status" json:"to_status"`
    ChangedBy  primitive.ObjectID `bson:"changed_by" json:"changed_by"`
    Reason     string             `bson:"reason,omitempty" json:"reason,omitempty"`
    ChangedAt  time.Time          `bson:"changed_at" json:"changed_at"`
}
```

#EXPORT_ENTITY: CompanySupplierRelationship{id, company_id, supplier_id, invited_email, status, classification}
#DATA_ASSUMPTION: SupplierID is null until invitation accepted (email-based invite)
#DATA_ASSUMPTION: One relationship per Company-Supplier pair (enforced by unique index)
#RELATIONSHIP_PATTERN: Many-to-Many bridge table pattern between Company and Supplier orgs
#CARDINALITY_ASSUMPTION: Company N:M Suppliers via relationship table
#NORMALIZATION_DECISION: StatusHistory embedded for audit trail without separate collection
#VOLUME_ASSUMPTION: 10-500 supplier relationships per company

---

### Entity: Requirement

**Purpose**: A specific requirement that a Company assigns to a Supplier (questionnaire or CheckFix grade).

```go
type RequirementType string

const (
    RequirementTypeQuestionnaire RequirementType = "QUESTIONNAIRE"
    RequirementTypeCheckFix      RequirementType = "CHECKFIX"
)

type RequirementStatus string

const (
    RequirementStatusPending    RequirementStatus = "PENDING"
    RequirementStatusInProgress RequirementStatus = "IN_PROGRESS"
    RequirementStatusSubmitted  RequirementStatus = "SUBMITTED"
    RequirementStatusApproved   RequirementStatus = "APPROVED"
    RequirementStatusRejected   RequirementStatus = "REJECTED"
    RequirementStatusUnderReview RequirementStatus = "UNDER_REVIEW"
    RequirementStatusExpired    RequirementStatus = "EXPIRED"
)

type Requirement struct {
    ID               primitive.ObjectID  `bson:"_id,omitempty" json:"id"`
    RelationshipID   primitive.ObjectID  `bson:"relationship_id" json:"relationship_id"`
    CompanyID        primitive.ObjectID  `bson:"company_id" json:"company_id"`
    SupplierID       primitive.ObjectID  `bson:"supplier_id" json:"supplier_id"`

    // Requirement details
    Type             RequirementType     `bson:"type" json:"type"`
    Title            string              `bson:"title" json:"title"`
    Description      string              `bson:"description,omitempty" json:"description,omitempty"`

    // For Questionnaire requirements
    QuestionnaireID  *primitive.ObjectID `bson:"questionnaire_id,omitempty" json:"questionnaire_id,omitempty"`
    PassingScore     *int                `bson:"passing_score,omitempty" json:"passing_score,omitempty"`

    // For CheckFix requirements
    MinimumGrade     *string             `bson:"minimum_grade,omitempty" json:"minimum_grade,omitempty"`
    MaxReportAgeDays *int                `bson:"max_report_age_days,omitempty" json:"max_report_age_days,omitempty"`

    // Timing
    DueDate          *time.Time          `bson:"due_date,omitempty" json:"due_date,omitempty"`
    ReminderSentAt   *time.Time          `bson:"reminder_sent_at,omitempty" json:"reminder_sent_at,omitempty"`

    // Status tracking
    Status           RequirementStatus   `bson:"status" json:"status"`
    StatusHistory    []RequirementStatusChange `bson:"status_history" json:"status_history"`

    // Assignment
    AssignedByUserID primitive.ObjectID  `bson:"assigned_by_user_id" json:"assigned_by_user_id"`
    AssignedAt       time.Time           `bson:"assigned_at" json:"assigned_at"`

    // Audit fields
    CreatedAt        time.Time           `bson:"created_at" json:"created_at"`
    UpdatedAt        time.Time           `bson:"updated_at" json:"updated_at"`
}

type RequirementStatusChange struct {
    FromStatus RequirementStatus  `bson:"from_status" json:"from_status"`
    ToStatus   RequirementStatus  `bson:"to_status" json:"to_status"`
    ChangedBy  primitive.ObjectID `bson:"changed_by" json:"changed_by"`
    Reason     string             `bson:"reason,omitempty" json:"reason,omitempty"`
    ChangedAt  time.Time          `bson:"changed_at" json:"changed_at"`
}
```

#EXPORT_ENTITY: Requirement{id, relationship_id, type, questionnaire_id, minimum_grade, due_date, status}
#DATA_ASSUMPTION: SupplierID denormalized from relationship for efficient querying
#DATA_ASSUMPTION: CompanyID denormalized from relationship for efficient querying
#CARDINALITY_ASSUMPTION: Relationship 1:N Requirements - One relationship has many requirements
#CARDINALITY_ASSUMPTION: Questionnaire 1:N Requirements - One questionnaire assigned to many suppliers
#NORMALIZATION_DECISION: Company/SupplierID denormalized to avoid joins for common queries
#VOLUME_ASSUMPTION: 1-10 requirements per relationship, 100-5000 total per company

---

### Entity: SupplierResponse

**Purpose**: Supplier's response to a requirement, containing answers or verification results.

```go
type SupplierResponse struct {
    ID              primitive.ObjectID `bson:"_id,omitempty" json:"id"`
    RequirementID   primitive.ObjectID `bson:"requirement_id" json:"requirement_id"`
    SupplierID      primitive.ObjectID `bson:"supplier_id" json:"supplier_id"`

    // For questionnaire responses
    SubmissionID    *primitive.ObjectID `bson:"submission_id,omitempty" json:"submission_id,omitempty"`

    // For CheckFix responses
    VerificationID  *primitive.ObjectID `bson:"verification_id,omitempty" json:"verification_id,omitempty"`

    // Scoring (denormalized for quick access)
    Score           *int               `bson:"score,omitempty" json:"score,omitempty"`
    MaxScore        *int               `bson:"max_score,omitempty" json:"max_score,omitempty"`
    Passed          *bool              `bson:"passed,omitempty" json:"passed,omitempty"`
    Grade           *string            `bson:"grade,omitempty" json:"grade,omitempty"`

    // Review
    ReviewedByUserID *primitive.ObjectID `bson:"reviewed_by_user_id,omitempty" json:"reviewed_by_user_id,omitempty"`
    ReviewedAt       *time.Time         `bson:"reviewed_at,omitempty" json:"reviewed_at,omitempty"`
    ReviewNotes      string             `bson:"review_notes,omitempty" json:"review_notes,omitempty"`

    // Audit fields
    StartedAt       time.Time          `bson:"started_at" json:"started_at"`
    SubmittedAt     *time.Time         `bson:"submitted_at,omitempty" json:"submitted_at,omitempty"`
    CreatedAt       time.Time          `bson:"created_at" json:"created_at"`
    UpdatedAt       time.Time          `bson:"updated_at" json:"updated_at"`
}
```

#EXPORT_ENTITY: SupplierResponse{id, requirement_id, supplier_id, submission_id, verification_id, score, passed}
#DATA_ASSUMPTION: Either SubmissionID or VerificationID is set, not both (based on requirement type)
#CARDINALITY_ASSUMPTION: Requirement 1:1 SupplierResponse - One response per requirement
#NORMALIZATION_DECISION: Score/Passed denormalized from submission for dashboard performance
#VOLUME_ASSUMPTION: 1 response per requirement

---

### Entity: QuestionnaireSubmission

**Purpose**: Contains all answers for a questionnaire submission with calculated scores.

```go
type QuestionnaireSubmission struct {
    ID               primitive.ObjectID `bson:"_id,omitempty" json:"id"`
    ResponseID       primitive.ObjectID `bson:"response_id" json:"response_id"`
    QuestionnaireID  primitive.ObjectID `bson:"questionnaire_id" json:"questionnaire_id"`
    SupplierID       primitive.ObjectID `bson:"supplier_id" json:"supplier_id"`

    // Answers
    Answers          []SubmissionAnswer `bson:"answers" json:"answers"`

    // Calculated scores
    TotalScore       int                `bson:"total_score" json:"total_score"`
    MaxPossibleScore int                `bson:"max_possible_score" json:"max_possible_score"`
    PercentageScore  float64            `bson:"percentage_score" json:"percentage_score"`
    Passed           bool               `bson:"passed" json:"passed"`
    MustPassFailed   bool               `bson:"must_pass_failed" json:"must_pass_failed"`

    // Topic-level scores
    TopicScores      []TopicScore       `bson:"topic_scores" json:"topic_scores"`

    // Metadata
    CompletionTime   int                `bson:"completion_time_minutes" json:"completion_time_minutes"`

    // Audit fields
    StartedAt        time.Time          `bson:"started_at" json:"started_at"`
    SubmittedAt      *time.Time         `bson:"submitted_at,omitempty" json:"submitted_at,omitempty"`
    CreatedAt        time.Time          `bson:"created_at" json:"created_at"`
    UpdatedAt        time.Time          `bson:"updated_at" json:"updated_at"`
}

type SubmissionAnswer struct {
    QuestionID      primitive.ObjectID `bson:"question_id" json:"question_id"`
    SelectedOptions []string           `bson:"selected_options,omitempty" json:"selected_options,omitempty"`
    TextAnswer      string             `bson:"text_answer,omitempty" json:"text_answer,omitempty"`
    PointsEarned    int                `bson:"points_earned" json:"points_earned"`
    MaxPoints       int                `bson:"max_points" json:"max_points"`
    IsMustPassMet   *bool              `bson:"is_must_pass_met,omitempty" json:"is_must_pass_met,omitempty"`
}

type TopicScore struct {
    TopicID        string  `bson:"topic_id" json:"topic_id"`
    TopicName      string  `bson:"topic_name" json:"topic_name"`
    Score          int     `bson:"score" json:"score"`
    MaxScore       int     `bson:"max_score" json:"max_score"`
    PercentageScore float64 `bson:"percentage_score" json:"percentage_score"`
}
```

#EXPORT_ENTITY: QuestionnaireSubmission{id, response_id, questionnaire_id, answers, total_score, passed}
#DATA_ASSUMPTION: Answers stored as embedded array (10-100 items, acceptable for MongoDB)
#NORMALIZATION_DECISION: Answers embedded - always read together, never queried individually
#NORMALIZATION_DECISION: TopicScores calculated and stored at submission time for reporting
#CARDINALITY_ASSUMPTION: SupplierResponse 1:1 QuestionnaireSubmission
#VOLUME_ASSUMPTION: 1 submission per response, 10-100 answers per submission

---

### Entity: CheckFixVerification

**Purpose**: Stores verified CheckFix report data for a supplier domain.

```go
type CheckFixVerification struct {
    ID                primitive.ObjectID `bson:"_id,omitempty" json:"id"`
    ResponseID        primitive.ObjectID `bson:"response_id" json:"response_id"`
    SupplierID        primitive.ObjectID `bson:"supplier_id" json:"supplier_id"`

    // Domain verification
    Domain            string             `bson:"domain" json:"domain"`
    VerifiedDomain    string             `bson:"verified_domain" json:"verified_domain"`
    DomainMatch       bool               `bson:"domain_match" json:"domain_match"`

    // Report details
    ReportHash        string             `bson:"report_hash" json:"report_hash"`
    ReportDate        time.Time          `bson:"report_date" json:"report_date"`

    // Grades and scores
    OverallGrade      string             `bson:"overall_grade" json:"overall_grade"`
    OverallScore      int                `bson:"overall_score" json:"overall_score"`

    // Category grades
    CategoryGrades    []CategoryGrade    `bson:"category_grades" json:"category_grades"`

    // Finding counts
    CriticalFindings  int                `bson:"critical_findings" json:"critical_findings"`
    HighFindings      int                `bson:"high_findings" json:"high_findings"`
    MediumFindings    int                `bson:"medium_findings" json:"medium_findings"`
    LowFindings       int                `bson:"low_findings" json:"low_findings"`

    // Verification metadata
    VerifiedAt        time.Time          `bson:"verified_at" json:"verified_at"`
    VerificationValid bool               `bson:"verification_valid" json:"verification_valid"`
    ExpiresAt         time.Time          `bson:"expires_at" json:"expires_at"`

    // Audit fields
    CreatedAt         time.Time          `bson:"created_at" json:"created_at"`
    UpdatedAt         time.Time          `bson:"updated_at" json:"updated_at"`
}

type CategoryGrade struct {
    Category string `bson:"category" json:"category"`
    Grade    string `bson:"grade" json:"grade"`
    Score    int    `bson:"score" json:"score"`
}
```

#EXPORT_ENTITY: CheckFixVerification{id, response_id, domain, overall_grade, overall_score, category_grades, verified_at}
#DATA_ASSUMPTION: Grades are A, B, C, D, F (no E grade)
#DATA_ASSUMPTION: Report hash used to verify authenticity via CheckFix API
#CARDINALITY_ASSUMPTION: SupplierResponse 1:1 CheckFixVerification
#CACHE_ASSUMPTION: Verifications cached with ExpiresAt, re-verified when expired
#VOLUME_ASSUMPTION: 1 verification per CheckFix requirement response

---

### Entity: SecureLink

**Purpose**: Magic link tokens for passwordless authentication.

```go
type SecureLinkType string

const (
    SecureLinkTypeAuth       SecureLinkType = "AUTH"
    SecureLinkTypeInvitation SecureLinkType = "INVITATION"
)

type SecureLink struct {
    ID               primitive.ObjectID `bson:"_id,omitempty" json:"id"`
    SecureIdentifier string             `bson:"secure_identifier" json:"secure_identifier"`
    Type             SecureLinkType     `bson:"type" json:"type"`

    // Target
    Email            string             `bson:"email" json:"email"`
    UserID           *primitive.ObjectID `bson:"user_id,omitempty" json:"user_id,omitempty"`
    RelationshipID   *primitive.ObjectID `bson:"relationship_id,omitempty" json:"relationship_id,omitempty"`

    // Validity
    ExpiresAt        time.Time          `bson:"expires_at" json:"expires_at"`
    UsedAt           *time.Time         `bson:"used_at,omitempty" json:"used_at,omitempty"`
    IsValid          bool               `bson:"is_valid" json:"is_valid"`

    // Security
    IPAddress        string             `bson:"ip_address,omitempty" json:"ip_address,omitempty"`
    UserAgent        string             `bson:"user_agent,omitempty" json:"user_agent,omitempty"`

    // Audit fields
    CreatedAt        time.Time          `bson:"created_at" json:"created_at"`
}
```

#EXPORT_ENTITY: SecureLink{id, secure_identifier, type, email, expires_at, is_valid}
#DATA_ASSUMPTION: SecureIdentifier is 64-character cryptographically random string
#DATA_ASSUMPTION: Auth links expire in 15 minutes, invitation links in 7 days
#INDEX_STRATEGY: TTL index on expires_at for automatic cleanup
#VOLUME_ASSUMPTION: Ephemeral data, auto-deleted by TTL

---

### Entity: AuditLog

**Purpose**: Comprehensive activity audit trail for compliance and debugging.

```go
type AuditAction string

const (
    AuditActionCreate   AuditAction = "CREATE"
    AuditActionUpdate   AuditAction = "UPDATE"
    AuditActionDelete   AuditAction = "DELETE"
    AuditActionLogin    AuditAction = "LOGIN"
    AuditActionLogout   AuditAction = "LOGOUT"
    AuditActionApprove  AuditAction = "APPROVE"
    AuditActionReject   AuditAction = "REJECT"
    AuditActionSubmit   AuditAction = "SUBMIT"
    AuditActionInvite   AuditAction = "INVITE"
    AuditActionAccept   AuditAction = "ACCEPT"
)

type AuditLog struct {
    ID              primitive.ObjectID  `bson:"_id,omitempty" json:"id"`

    // Actor
    ActorUserID     *primitive.ObjectID `bson:"actor_user_id,omitempty" json:"actor_user_id,omitempty"`
    ActorEmail      string              `bson:"actor_email,omitempty" json:"actor_email,omitempty"`
    ActorOrgID      *primitive.ObjectID `bson:"actor_org_id,omitempty" json:"actor_org_id,omitempty"`

    // Action
    Action          AuditAction         `bson:"action" json:"action"`
    ResourceType    string              `bson:"resource_type" json:"resource_type"`
    ResourceID      primitive.ObjectID  `bson:"resource_id" json:"resource_id"`

    // Context
    Description     string              `bson:"description" json:"description"`
    Changes         map[string]interface{} `bson:"changes,omitempty" json:"changes,omitempty"`

    // Request metadata
    IPAddress       string              `bson:"ip_address,omitempty" json:"ip_address,omitempty"`
    UserAgent       string              `bson:"user_agent,omitempty" json:"user_agent,omitempty"`
    RequestID       string              `bson:"request_id,omitempty" json:"request_id,omitempty"`

    // Timestamp
    CreatedAt       time.Time           `bson:"created_at" json:"created_at"`
}
```

#EXPORT_ENTITY: AuditLog{id, actor_user_id, action, resource_type, resource_id, description, created_at}
#DATA_ASSUMPTION: Audit logs are append-only, never modified or deleted
#DATA_ASSUMPTION: Changes field stores before/after values for UPDATE actions
#VOLUME_ASSUMPTION: High volume - 10-100 logs per user per day
#GROWTH_ASSUMPTION: Consider time-based partitioning after 1M records
#RETENTION_POLICY: Keep for 2 years minimum for compliance

---

## Relationships

### Relationship: Organization has Users

```
Organization (1) <------ (N) User
             ↑
             └── organization_id
```

- **From**: Organization (1)
- **To**: User (N)
- **Type**: One-to-Many
- **Foreign Key**: User.organization_id -> Organization._id

#RELATIONSHIP_PATTERN: Ownership pattern - users belong to exactly one organization
#CARDINALITY_ASSUMPTION: 1:N - A user cannot belong to multiple organizations
#REFERENTIAL_INTEGRITY: Application-level enforcement, no CASCADE (soft delete)

---

### Relationship: Company has Supplier Relationships

```
Organization[Company] (1) <------ (N) CompanySupplierRelationship
                      ↑
                      └── company_id
```

- **From**: Organization (type=COMPANY) (1)
- **To**: CompanySupplierRelationship (N)
- **Type**: One-to-Many
- **Foreign Key**: CompanySupplierRelationship.company_id -> Organization._id

#RELATIONSHIP_PATTERN: Ownership pattern - company owns the relationship records
#CARDINALITY_ASSUMPTION: 1:N - Company has many supplier relationships

---

### Relationship: Supplier has Relationships

```
Organization[Supplier] (1) <------ (N) CompanySupplierRelationship
                       ↑
                       └── supplier_id
```

- **From**: Organization (type=SUPPLIER) (1)
- **To**: CompanySupplierRelationship (N)
- **Type**: One-to-Many
- **Foreign Key**: CompanySupplierRelationship.supplier_id -> Organization._id

#RELATIONSHIP_PATTERN: Reference pattern - supplier referenced by relationship
#CARDINALITY_ASSUMPTION: 1:N - Supplier can work with many companies
#DATA_ASSUMPTION: supplier_id is NULL until invitation accepted

---

### Relationship: Company owns Questionnaires

```
Organization[Company] (1) <------ (N) Questionnaire
                      ↑
                      └── company_id
```

- **From**: Organization (type=COMPANY) (1)
- **To**: Questionnaire (N)
- **Type**: One-to-Many
- **Foreign Key**: Questionnaire.company_id -> Organization._id

#RELATIONSHIP_PATTERN: Ownership pattern - company creates and owns questionnaires
#CARDINALITY_ASSUMPTION: 1:N - Company has many questionnaires
#CASCADE_STRATEGY: Soft delete - archive questionnaires when company deleted

---

### Relationship: Questionnaire derived from Template

```
QuestionnaireTemplate (1) <------ (N) Questionnaire
                      ↑
                      └── template_id (optional)
```

- **From**: QuestionnaireTemplate (1)
- **To**: Questionnaire (N)
- **Type**: One-to-Many (optional)
- **Foreign Key**: Questionnaire.template_id -> QuestionnaireTemplate._id

#RELATIONSHIP_PATTERN: Derivation pattern - questionnaire cloned from template
#CARDINALITY_ASSUMPTION: 1:N optional - questionnaire may not have template (custom)
#DATA_ASSUMPTION: Template changes do not affect derived questionnaires

---

### Relationship: Questionnaire contains Questions

```
Questionnaire (1) <------ (N) Question
              ↑
              └── questionnaire_id
```

- **From**: Questionnaire (1)
- **To**: Question (N)
- **Type**: One-to-Many
- **Foreign Key**: Question.questionnaire_id -> Questionnaire._id

#RELATIONSHIP_PATTERN: Composition pattern - questions are part of questionnaire
#CARDINALITY_ASSUMPTION: 1:N - Questionnaire has 10-100 questions
#CASCADE_STRATEGY: CASCADE DELETE - questions deleted with questionnaire

---

### Relationship: Relationship has Requirements

```
CompanySupplierRelationship (1) <------ (N) Requirement
                            ↑
                            └── relationship_id
```

- **From**: CompanySupplierRelationship (1)
- **To**: Requirement (N)
- **Type**: One-to-Many
- **Foreign Key**: Requirement.relationship_id -> CompanySupplierRelationship._id

#RELATIONSHIP_PATTERN: Ownership pattern - requirements tied to specific relationship
#CARDINALITY_ASSUMPTION: 1:N - Relationship has 1-10 active requirements
#REFERENTIAL_INTEGRITY: Application-level, no automatic cascade

---

### Relationship: Requirement has Response

```
Requirement (1) <------ (1) SupplierResponse
            ↑
            └── requirement_id
```

- **From**: Requirement (1)
- **To**: SupplierResponse (1)
- **Type**: One-to-One
- **Foreign Key**: SupplierResponse.requirement_id -> Requirement._id

#RELATIONSHIP_PATTERN: Response pattern - each requirement gets one response
#CARDINALITY_ASSUMPTION: 1:1 - Exactly one response per requirement
#DATA_ASSUMPTION: New requirement version creates new requirement, not new response

---

### Relationship: Response has Submission

```
SupplierResponse (1) <------ (0..1) QuestionnaireSubmission
                 ↑
                 └── response_id
```

- **From**: SupplierResponse (1)
- **To**: QuestionnaireSubmission (0..1)
- **Type**: One-to-One (optional)
- **Foreign Key**: QuestionnaireSubmission.response_id -> SupplierResponse._id

#RELATIONSHIP_PATTERN: Detail pattern - submission contains response details
#CARDINALITY_ASSUMPTION: 1:0..1 - Only questionnaire responses have submissions
#DATA_ASSUMPTION: Submission exists only for RequirementType=QUESTIONNAIRE

---

### Relationship: Response has Verification

```
SupplierResponse (1) <------ (0..1) CheckFixVerification
                 ↑
                 └── response_id
```

- **From**: SupplierResponse (1)
- **To**: CheckFixVerification (0..1)
- **Type**: One-to-One (optional)
- **Foreign Key**: CheckFixVerification.response_id -> SupplierResponse._id

#RELATIONSHIP_PATTERN: Detail pattern - verification contains CheckFix results
#CARDINALITY_ASSUMPTION: 1:0..1 - Only CheckFix responses have verifications
#DATA_ASSUMPTION: Verification exists only for RequirementType=CHECKFIX

---

## Index Definitions

### Organizations Collection

```javascript
// Unique slug for URL-friendly identifiers
db.organizations.createIndex({ "slug": 1 }, { unique: true })

// Unique domain for supplier verification (sparse - not all have domain)
db.organizations.createIndex({ "domain": 1 }, { unique: true, sparse: true })

// Query by type for listing
db.organizations.createIndex({ "type": 1, "created_at": -1 })

// Soft delete filtering
db.organizations.createIndex({ "deleted_at": 1 }, { sparse: true })
```

#INDEX_STRATEGY: Slug index for O(1) lookup by URL slug
#INDEX_STRATEGY: Domain sparse index for supplier verification without requiring all orgs have domain

---

### Users Collection

```javascript
// Unique email for authentication
db.users.createIndex({ "email": 1 }, { unique: true })

// Query users by organization
db.users.createIndex({ "organization_id": 1, "role": 1 })

// Active users filter
db.users.createIndex({ "organization_id": 1, "is_active": 1 })

// Soft delete filtering
db.users.createIndex({ "deleted_at": 1 }, { sparse: true })
```

#INDEX_STRATEGY: Email index critical for login performance
#INDEX_STRATEGY: Compound org+role for admin user lookups

---

### QuestionnaireTemplates Collection

```javascript
// Browse by category
db.questionnaire_templates.createIndex({ "category": 1, "is_system": 1 })

// Company's custom templates
db.questionnaire_templates.createIndex({ "created_by_org_id": 1 }, { sparse: true })

// Full-text search on name and description
db.questionnaire_templates.createIndex(
  { "name": "text", "description": "text", "tags": "text" },
  { weights: { name: 10, tags: 5, description: 1 } }
)
```

#INDEX_STRATEGY: Category index for template browser filtering
#INDEX_STRATEGY: Text index for template search functionality

---

### Questionnaires Collection

```javascript
// Company's questionnaires list
db.questionnaires.createIndex({ "company_id": 1, "status": 1, "created_at": -1 })

// Lookup by template origin
db.questionnaires.createIndex({ "template_id": 1 }, { sparse: true })
```

#INDEX_STRATEGY: Compound index for dashboard query optimization
#QUERY_PATTERN: Most common query is "list my published questionnaires"

---

### Questions Collection

```javascript
// Get all questions for a questionnaire (ordered)
db.questions.createIndex({ "questionnaire_id": 1, "order": 1 })

// Questions by topic for section display
db.questions.createIndex({ "questionnaire_id": 1, "topic_id": 1, "order": 1 })
```

#INDEX_STRATEGY: Compound index covers question ordering within questionnaire
#QUERY_PATTERN: Always fetch all questions for a questionnaire at once

---

### CompanySupplierRelationships Collection

```javascript
// Unique relationship per company-supplier pair (sparse for pending invites)
db.company_supplier_relationships.createIndex(
  { "company_id": 1, "supplier_id": 1 },
  { unique: true, sparse: true }
)

// Company's suppliers list with status filter
db.company_supplier_relationships.createIndex({ "company_id": 1, "status": 1, "classification": 1 })

// Supplier's companies list
db.company_supplier_relationships.createIndex({ "supplier_id": 1, "status": 1 })

// Find pending invites by email
db.company_supplier_relationships.createIndex({ "invited_email": 1, "status": 1 })
```

#INDEX_STRATEGY: Unique sparse prevents duplicate active relationships
#QUERY_PATTERN: Company dashboard queries by status and classification
#QUERY_PATTERN: Supplier lookup of pending invitations by email

---

### Requirements Collection

```javascript
// Company's requirements with status
db.requirements.createIndex({ "company_id": 1, "status": 1, "due_date": 1 })

// Supplier's requirements
db.requirements.createIndex({ "supplier_id": 1, "status": 1, "due_date": 1 })

// Requirements by relationship
db.requirements.createIndex({ "relationship_id": 1, "status": 1 })

// Due date monitoring for reminders
db.requirements.createIndex({ "due_date": 1, "status": 1 })

// Expiration handling
db.requirements.createIndex({ "status": 1, "due_date": 1 })
```

#INDEX_STRATEGY: Multiple indexes for different access patterns
#QUERY_PATTERN: Dashboard queries: "my pending requirements" and "overdue requirements"
#EXPORT_QUERY: SELECT * FROM requirements WHERE status='PENDING' AND due_date < NOW()

---

### SupplierResponses Collection

```javascript
// Response by requirement
db.supplier_responses.createIndex({ "requirement_id": 1 }, { unique: true })

// Supplier's responses
db.supplier_responses.createIndex({ "supplier_id": 1, "submitted_at": -1 })
```

#INDEX_STRATEGY: Unique index enforces one response per requirement
#QUERY_PATTERN: Response always fetched with requirement

---

### QuestionnaireSubmissions Collection

```javascript
// Submission by response
db.questionnaire_submissions.createIndex({ "response_id": 1 }, { unique: true })

// Supplier's submission history
db.questionnaire_submissions.createIndex({ "supplier_id": 1, "submitted_at": -1 })

// Analytics: questionnaire performance
db.questionnaire_submissions.createIndex({ "questionnaire_id": 1, "submitted_at": -1 })
```

#INDEX_STRATEGY: Unique index enforces one submission per response
#QUERY_PATTERN: Analytics query for questionnaire pass/fail rates

---

### CheckFixVerifications Collection

```javascript
// Verification by response
db.checkfix_verifications.createIndex({ "response_id": 1 }, { unique: true })

// Supplier's verifications
db.checkfix_verifications.createIndex({ "supplier_id": 1, "verified_at": -1 })

// Expiration check
db.checkfix_verifications.createIndex({ "expires_at": 1 })
```

#INDEX_STRATEGY: Unique index enforces one verification per response
#CACHE_ASSUMPTION: Verifications re-checked when expires_at passed

---

### SecureLinks Collection

```javascript
// Unique token lookup
db.secure_links.createIndex({ "secure_identifier": 1 }, { unique: true })

// TTL index for automatic expiration cleanup
db.secure_links.createIndex({ "expires_at": 1 }, { expireAfterSeconds: 0 })

// Find links by email for rate limiting
db.secure_links.createIndex({ "email": 1, "created_at": -1 })
```

#INDEX_STRATEGY: TTL index automatically removes expired tokens
#INDEX_STRATEGY: Email index for rate limiting (max 3 links per hour)
#DATA_ASSUMPTION: MongoDB TTL worker runs every 60 seconds

---

### AuditLogs Collection

```javascript
// Query by actor
db.audit_logs.createIndex({ "actor_user_id": 1, "created_at": -1 })

// Query by resource
db.audit_logs.createIndex({ "resource_type": 1, "resource_id": 1, "created_at": -1 })

// Query by organization
db.audit_logs.createIndex({ "actor_org_id": 1, "created_at": -1 })

// Query by action type
db.audit_logs.createIndex({ "action": 1, "created_at": -1 })

// Time-based archival
db.audit_logs.createIndex({ "created_at": 1 })
```

#INDEX_STRATEGY: Multiple indexes for different audit query patterns
#GROWTH_ASSUMPTION: Consider creating time-based collections (audit_logs_2024_q1) after 10M records
#PARTITION_RATIONALE: Time-based partitioning for archival and query performance

---

## API Query Patterns

### Pattern: User Authentication (Magic Link)

**Endpoint**: POST /api/v1/auth/request-link

```javascript
// 1. Find user by email
db.users.findOne({ email: "user@example.com", is_active: true, deleted_at: null })

// 2. Create secure link
db.secure_links.insertOne({
  secure_identifier: "<64-char-token>",
  type: "AUTH",
  email: "user@example.com",
  user_id: ObjectId("..."),
  expires_at: new Date(Date.now() + 15*60*1000),
  is_valid: true,
  created_at: new Date()
})
```

#QUERY_PATTERN: Single document lookup by indexed email field
#INDEX_STRATEGY: users.email index provides O(1) lookup
#THROUGHPUT_ASSUMPTION: 10-50 login requests per minute peak

---

### Pattern: Company Dashboard - Supplier List

**Endpoint**: GET /api/v1/suppliers

```javascript
// Get all supplier relationships with pagination
db.company_supplier_relationships.aggregate([
  { $match: {
    company_id: ObjectId("<company_id>"),
    status: { $in: ["ACTIVE", "PENDING"] }
  }},
  { $sort: { classification: 1, created_at: -1 } },
  { $skip: 0 },
  { $limit: 20 },
  { $lookup: {
    from: "organizations",
    localField: "supplier_id",
    foreignField: "_id",
    as: "supplier"
  }},
  { $unwind: { path: "$supplier", preserveNullAndEmptyArrays: true } }
])
```

#QUERY_PATTERN: Aggregation with lookup for joined data
#INDEX_STRATEGY: company_supplier_relationships compound index covers match and sort
#EXPORT_QUERY: Company supplier list with org details joined

---

### Pattern: Supplier Dashboard - Requirements List

**Endpoint**: GET /api/v1/requirements

```javascript
// Get supplier's pending requirements
db.requirements.aggregate([
  { $match: {
    supplier_id: ObjectId("<supplier_id>"),
    status: { $in: ["PENDING", "IN_PROGRESS", "REJECTED"] }
  }},
  { $sort: { due_date: 1 } },
  { $lookup: {
    from: "organizations",
    localField: "company_id",
    foreignField: "_id",
    as: "company"
  }},
  { $unwind: "$company" },
  { $lookup: {
    from: "questionnaires",
    localField: "questionnaire_id",
    foreignField: "_id",
    as: "questionnaire"
  }},
  { $unwind: { path: "$questionnaire", preserveNullAndEmptyArrays: true } }
])
```

#QUERY_PATTERN: Multi-lookup aggregation for requirement details
#INDEX_STRATEGY: requirements.supplier_id compound index for efficient filtering
#EXPORT_QUERY: Supplier requirements with company and questionnaire details

---

### Pattern: Fill Questionnaire - Get Questions

**Endpoint**: GET /api/v1/questionnaires/:id/questions

```javascript
// Get questionnaire with all questions
const questionnaire = db.questionnaires.findOne({ _id: ObjectId("<id>") })

const questions = db.questions.find({
  questionnaire_id: ObjectId("<id>")
}).sort({ topic_id: 1, order: 1 }).toArray()

// Group questions by topic
const groupedByTopic = groupBy(questions, "topic_id")
```

#QUERY_PATTERN: Two queries - questionnaire metadata + all questions
#INDEX_STRATEGY: questions compound index (questionnaire_id, order) for sorted fetch
#DATA_ASSUMPTION: Typical questionnaire has 10-100 questions, acceptable single fetch

---

### Pattern: Submit Questionnaire

**Endpoint**: POST /api/v1/responses/:id/submit

```javascript
// Transaction: Update response, create submission, update requirement
session.startTransaction()

// 1. Get questions for scoring
const questions = db.questions.find({ questionnaire_id: ObjectId("<id>") }).toArray()

// 2. Calculate scores (application logic)
const scores = calculateScores(answers, questions)

// 3. Create submission
db.questionnaire_submissions.insertOne({
  response_id: ObjectId("<response_id>"),
  questionnaire_id: ObjectId("<questionnaire_id>"),
  supplier_id: ObjectId("<supplier_id>"),
  answers: processedAnswers,
  total_score: scores.total,
  max_possible_score: scores.max,
  percentage_score: scores.percentage,
  passed: scores.passed,
  must_pass_failed: scores.mustPassFailed,
  topic_scores: scores.byTopic,
  submitted_at: new Date()
}, { session })

// 4. Update response
db.supplier_responses.updateOne(
  { _id: ObjectId("<response_id>") },
  {
    $set: {
      submission_id: newSubmissionId,
      score: scores.total,
      max_score: scores.max,
      passed: scores.passed,
      submitted_at: new Date()
    }
  },
  { session }
)

// 5. Update requirement status
db.requirements.updateOne(
  { _id: ObjectId("<requirement_id>") },
  {
    $set: { status: "SUBMITTED", updated_at: new Date() },
    $push: {
      status_history: {
        from_status: "IN_PROGRESS",
        to_status: "SUBMITTED",
        changed_by: ObjectId("<user_id>"),
        changed_at: new Date()
      }
    }
  },
  { session }
)

session.commitTransaction()
```

#QUERY_PATTERN: Multi-collection transaction for data consistency
#EXPORT_PIPELINE: Answers -> Scoring -> Submission -> Response Update -> Status Update
#DATA_ASSUMPTION: MongoDB 4.0+ supports multi-document transactions

---

### Pattern: Verify CheckFix Report

**Endpoint**: POST /api/v1/checkfix/verify

```javascript
// 1. Call CheckFix API to verify report (external)
const checkfixData = await checkfixAPI.verifyReport(reportHash)

// 2. Verify domain matches supplier
const org = db.organizations.findOne({ _id: ObjectId("<supplier_id>") })
const domainMatch = checkfixData.domain === org.domain

// 3. Create verification record
db.checkfix_verifications.insertOne({
  response_id: ObjectId("<response_id>"),
  supplier_id: ObjectId("<supplier_id>"),
  domain: org.domain,
  verified_domain: checkfixData.domain,
  domain_match: domainMatch,
  report_hash: reportHash,
  report_date: checkfixData.reportDate,
  overall_grade: checkfixData.grade,
  overall_score: checkfixData.score,
  category_grades: checkfixData.categories,
  critical_findings: checkfixData.findings.critical,
  high_findings: checkfixData.findings.high,
  medium_findings: checkfixData.findings.medium,
  low_findings: checkfixData.findings.low,
  verified_at: new Date(),
  verification_valid: domainMatch && meetsMinimumGrade,
  expires_at: new Date(Date.now() + 30*24*60*60*1000) // 30 days
})

// 4. Update response
db.supplier_responses.updateOne(
  { _id: ObjectId("<response_id>") },
  {
    $set: {
      verification_id: newVerificationId,
      grade: checkfixData.grade,
      passed: domainMatch && meetsMinimumGrade
    }
  }
)
```

#QUERY_PATTERN: External API call + document creation + update
#EXPORT_PIPELINE: External Verify -> Store Verification -> Update Response
#CACHE_ASSUMPTION: Verifications valid for 30 days before re-verification needed

---

### Pattern: Overdue Requirements Report

**Endpoint**: GET /api/v1/reports/overdue (Company Admin)

```javascript
db.requirements.aggregate([
  { $match: {
    company_id: ObjectId("<company_id>"),
    status: { $in: ["PENDING", "IN_PROGRESS"] },
    due_date: { $lt: new Date() }
  }},
  { $lookup: {
    from: "company_supplier_relationships",
    localField: "relationship_id",
    foreignField: "_id",
    as: "relationship"
  }},
  { $unwind: "$relationship" },
  { $lookup: {
    from: "organizations",
    localField: "supplier_id",
    foreignField: "_id",
    as: "supplier"
  }},
  { $unwind: "$supplier" },
  { $group: {
    _id: "$supplier_id",
    supplier_name: { $first: "$supplier.name" },
    classification: { $first: "$relationship.classification" },
    overdue_count: { $sum: 1 },
    oldest_due_date: { $min: "$due_date" },
    requirements: { $push: { id: "$_id", title: "$title", due_date: "$due_date" } }
  }},
  { $sort: { classification: 1, overdue_count: -1 } }
])
```

#QUERY_PATTERN: Complex aggregation for reporting
#EXPORT_QUERY: Overdue requirements grouped by supplier
#INDEX_STRATEGY: requirements compound index (company_id, status, due_date) covers query

---

## Data Flows

### Flow: User Registration via Invitation

```
1. Company admin invites supplier via email
   └─> Create CompanySupplierRelationship (status=PENDING)
   └─> Create SecureLink (type=INVITATION)
   └─> Send email via mail service

2. Supplier clicks invitation link
   └─> GET /auth/verify/:token
   └─> Validate SecureLink
   └─> Check if organization exists for email domain

3a. New Supplier Registration
   └─> Create Organization (type=SUPPLIER)
   └─> Create User (role=ADMIN)
   └─> Update CompanySupplierRelationship.supplier_id
   └─> Issue JWT tokens

3b. Existing Supplier User
   └─> Update CompanySupplierRelationship.supplier_id
   └─> Issue JWT tokens

4. Supplier accepts invitation
   └─> POST /companies/:id/accept
   └─> Update relationship status to ACTIVE
   └─> Log to audit_logs
```

#EXPORT_PIPELINE: Invitation -> SecureLink -> Registration -> Relationship Update
#THROUGHPUT_ASSUMPTION: 10-50 invitations per company per day
#DATA_ASSUMPTION: Email domain used to suggest organization, not enforce

---

### Flow: Questionnaire Creation

```
1. Company admin selects template
   └─> GET /questionnaire-templates/:id
   └─> Fetch template with topics

2. Clone template to questionnaire
   └─> POST /questionnaires
   └─> Create Questionnaire (status=DRAFT)
   └─> Copy topics to questionnaire

3. Add/edit questions
   └─> POST /questionnaires/:id/questions
   └─> Create Question documents
   └─> Embed QuestionOptions in each

4. Configure scoring
   └─> PATCH /questionnaires/:id
   └─> Set passing_score, scoring_mode
   └─> Mark must_pass questions

5. Publish questionnaire
   └─> POST /questionnaires/:id/publish
   └─> Update status to PUBLISHED
   └─> Calculate and store max_possible_score
   └─> Increment template usage_count
```

#EXPORT_PIPELINE: Template -> Clone -> Edit Questions -> Configure -> Publish
#DATA_ASSUMPTION: Questions can only be edited while questionnaire is DRAFT

---

### Flow: Requirement Assignment and Response

```
1. Company assigns requirement
   └─> POST /suppliers/:id/requirements
   └─> Create Requirement (status=PENDING)
   └─> Send notification email to supplier

2. Supplier starts response
   └─> POST /requirements/:id/responses
   └─> Create SupplierResponse (started_at=now)
   └─> Update Requirement status to IN_PROGRESS

3a. Questionnaire Response
   └─> Fetch all questions
   └─> Supplier fills answers
   └─> Auto-save progress to SupplierResponse.draft_answers

4a. Submit questionnaire
   └─> POST /responses/:id/submit
   └─> Calculate scores
   └─> Create QuestionnaireSubmission
   └─> Update SupplierResponse with scores
   └─> Update Requirement status to SUBMITTED

3b. CheckFix Response
   └─> POST /checkfix/verify
   └─> Call CheckFix API
   └─> Create CheckFixVerification
   └─> Update SupplierResponse with grade
   └─> Update Requirement status to SUBMITTED

5. Company reviews
   └─> POST /responses/:id/approve OR /reject
   └─> Update Requirement status
   └─> Log to audit_logs
   └─> Send notification email
```

#EXPORT_PIPELINE: Assign -> Start -> Fill/Verify -> Submit -> Review -> Complete
#THROUGHPUT_ASSUMPTION: 100-500 submissions per day at scale

---

### Flow: Reminder System (Background Job)

```
1. Scheduled job runs daily
   └─> Find requirements where:
       - status IN (PENDING, IN_PROGRESS)
       - due_date - NOW() <= reminder_days_before
       - reminder_sent_at IS NULL

2. For each requirement
   └─> Get supplier contact email
   └─> Get company settings for reminder template
   └─> Send reminder email
   └─> Update requirement.reminder_sent_at

3. Find expired requirements
   └─> status IN (PENDING, IN_PROGRESS)
   └─> due_date < NOW()
   └─> Update status to EXPIRED
   └─> Notify company admin
```

#EXPORT_PIPELINE: Scheduled -> Query Pending -> Send Reminders -> Update Status
#QUERY_PATTERN: Batch query for reminder-eligible requirements
#INDEX_STRATEGY: requirements compound index (status, due_date) optimizes batch query

---

## Performance Strategy

### Read Performance

| Query Type | Strategy |
|------------|----------|
| Single document by ID | Direct _id lookup, O(1) |
| User by email | Unique index, O(1) |
| Organization by slug | Unique index, O(1) |
| Supplier list (paginated) | Compound index + skip/limit |
| Requirements by supplier | Compound index, covered query |
| Questions by questionnaire | Compound index, sorted fetch |

#INDEX_STRATEGY: All high-frequency queries have covering indexes
#CACHE_ASSUMPTION: Application-level caching for organization and user lookups

### Write Performance

| Operation | Strategy |
|-----------|----------|
| Create user | Single insert, unique constraint |
| Update requirement status | Atomic update with $push for history |
| Submit questionnaire | Multi-document transaction |
| Audit logging | Async write, fire-and-forget |

#DATA_ASSUMPTION: Write volume is 10% of read volume
#THROUGHPUT_ASSUMPTION: 100-500 writes per minute at peak

### Aggregation Performance

| Report | Strategy |
|--------|----------|
| Supplier compliance summary | Pre-computed in SupplierResponse |
| Overdue requirements | Indexed aggregation |
| Questionnaire pass rates | Time-bounded aggregation |

#INDEX_STRATEGY: Aggregation queries use compound indexes for $match optimization
#CACHE_ASSUMPTION: Dashboard aggregations cached for 5 minutes

---

## Scale Projections

### Year 1

| Metric | Estimate |
|--------|----------|
| Organizations | 500 (250 companies, 250 suppliers) |
| Users | 2,500 (5 per org average) |
| Questionnaires | 1,250 (5 per company) |
| Questions | 62,500 (50 per questionnaire) |
| Relationships | 12,500 (50 suppliers per company) |
| Requirements | 37,500 (3 per relationship) |
| Submissions | 30,000 (80% response rate) |
| Audit logs | 500,000 |

#VOLUME_ASSUMPTION: Year 1 targets 500 organizations
#GROWTH_ASSUMPTION: 50% quarter-over-quarter growth

### Year 3

| Metric | Estimate |
|--------|----------|
| Organizations | 10,000 |
| Users | 50,000 |
| Questionnaires | 25,000 |
| Questions | 1,250,000 |
| Relationships | 500,000 |
| Requirements | 1,500,000 |
| Submissions | 1,200,000 |
| Audit logs | 50,000,000 |

#VOLUME_ASSUMPTION: Year 3 targets 10K organizations
#PARTITION_RATIONALE: Consider sharding by organization_id at 1M+ documents per collection

### Storage Estimates

| Collection | Avg Doc Size | Year 1 | Year 3 |
|------------|--------------|--------|--------|
| organizations | 2 KB | 1 MB | 20 MB |
| users | 1 KB | 2.5 MB | 50 MB |
| questionnaires | 3 KB | 3.75 MB | 75 MB |
| questions | 2 KB | 125 MB | 2.5 GB |
| relationships | 2 KB | 25 MB | 1 GB |
| requirements | 2 KB | 75 MB | 3 GB |
| submissions | 10 KB | 300 MB | 12 GB |
| audit_logs | 1 KB | 500 MB | 50 GB |

#GROWTH_ASSUMPTION: Audit logs are largest collection, need archival strategy
#ARCHIVE_STRATEGY: Archive audit logs older than 1 year to cold storage

---

## Database Choice and Configuration

### MongoDB Selection Rationale

#SCHEMA_RATIONALE: MongoDB selected for the following reasons:
1. **Flexible schema** - Questionnaire structures vary, embedded options/answers natural fit
2. **Document model** - Questionnaire with embedded questions/options reads as single unit
3. **Aggregation framework** - Complex reporting queries without joins
4. **Horizontal scaling** - Sharding capability for multi-tenant growth
5. **Go driver maturity** - Official driver with strong type mapping

### Recommended Configuration

```yaml
# MongoDB 6.0+ recommended
replicaSet: "rs0"
nodes: 3  # Primary + 2 secondaries for HA

# Connection string
mongodb://user:pass@host1:27017,host2:27017,host3:27017/supplier_portal?replicaSet=rs0&readPreference=secondaryPreferred

# Write concern for transactions
writeConcern: { w: "majority", j: true }

# Read preference
readPreference: secondaryPreferred  # Analytics queries to secondary
```

#DATA_ASSUMPTION: Production uses replica set for high availability
#DATA_ASSUMPTION: Read-heavy workloads benefit from secondary reads

---

## Data Retention and Archival

### Retention Policies

| Collection | Retention | Strategy |
|------------|-----------|----------|
| organizations | Indefinite | Soft delete |
| users | Indefinite | Soft delete |
| questionnaires | Indefinite | Archive to cold storage |
| questions | With questionnaire | Cascade with questionnaire |
| relationships | Indefinite | Soft delete |
| requirements | 5 years | Archive after completion |
| submissions | 5 years | Archive with requirement |
| verifications | 2 years | Delete after expiry |
| secure_links | TTL auto-delete | MongoDB TTL |
| audit_logs | 2 years active, 5 years archive | Time-based archival |

#RETENTION_POLICY: Compliance requires 5-year retention for security assessments
#ARCHIVE_STRATEGY: Move completed requirements older than 1 year to archive collection

### Soft Delete Implementation

```go
// All models with soft delete
type SoftDeletable struct {
    DeletedAt *time.Time `bson:"deleted_at,omitempty" json:"deleted_at,omitempty"`
}

// Repository query filter
func NotDeleted() bson.M {
    return bson.M{"deleted_at": nil}
}

// Soft delete operation
func SoftDelete(id primitive.ObjectID) error {
    _, err := collection.UpdateOne(ctx,
        bson.M{"_id": id},
        bson.M{"$set": bson.M{"deleted_at": time.Now()}})
    return err
}
```

#DATA_ASSUMPTION: Soft delete preserves referential integrity
#DATA_ASSUMPTION: Background job permanently deletes after 30 days

---

## Constraints and Business Rules

### Entity Constraints

#EXPORT_CONSTRAINT: Organization.slug must be unique and URL-safe (lowercase alphanumeric with hyphens)
#EXPORT_CONSTRAINT: Organization.domain must be unique and valid domain format when present
#EXPORT_CONSTRAINT: User.email must be unique across entire system
#EXPORT_CONSTRAINT: CompanySupplierRelationship (company_id, supplier_id) must be unique when supplier_id is set
#EXPORT_CONSTRAINT: Requirement can only be created for ACTIVE relationships
#EXPORT_CONSTRAINT: SupplierResponse can only be created when requirement status is PENDING
#EXPORT_CONSTRAINT: QuestionnaireSubmission can only be created once per response

### Business Rules

#BUSINESS_RULE: Users can only access data within their organization
#BUSINESS_RULE: Company admins can invite suppliers and create requirements
#BUSINESS_RULE: Supplier admins can respond to requirements and accept invitations
#BUSINESS_RULE: Questionnaires cannot be edited after publishing (create new version)
#BUSINESS_RULE: Requirements cannot be assigned to suppliers with TERMINATED relationships
#BUSINESS_RULE: IsMustPass questions cause automatic fail regardless of total score
#BUSINESS_RULE: CheckFix verification requires domain match with supplier organization
#BUSINESS_RULE: Expired requirements cannot receive responses (auto-expire via background job)

### State Machine Constraints

#BUSINESS_RULE: RelationshipStatus transitions:
- PENDING -> ACTIVE (accept) | REJECTED (decline)
- ACTIVE -> SUSPENDED (company action) | TERMINATED (either party)
- SUSPENDED -> ACTIVE (reactivate) | TERMINATED (finalize)
- REJECTED -> (terminal state)
- TERMINATED -> (terminal state)

#BUSINESS_RULE: RequirementStatus transitions:
- PENDING -> IN_PROGRESS (start) | EXPIRED (timeout)
- IN_PROGRESS -> SUBMITTED (submit) | EXPIRED (timeout)
- SUBMITTED -> APPROVED (company) | REJECTED (company) | UNDER_REVIEW (revision)
- UNDER_REVIEW -> SUBMITTED (resubmit)
- APPROVED -> (terminal state)
- REJECTED -> IN_PROGRESS (retry allowed)
- EXPIRED -> (terminal state)

---

## Migration Strategy

### Initial Schema Setup

```javascript
// Run on fresh database
use supplier_portal

// Create collections with validation (optional strict mode)
db.createCollection("organizations")
db.createCollection("users")
db.createCollection("questionnaire_templates")
db.createCollection("questionnaires")
db.createCollection("questions")
db.createCollection("company_supplier_relationships")
db.createCollection("requirements")
db.createCollection("supplier_responses")
db.createCollection("questionnaire_submissions")
db.createCollection("checkfix_verifications")
db.createCollection("secure_links")
db.createCollection("audit_logs")

// Create all indexes (see Index Definitions section)
// ... index creation scripts ...
```

### Seed Data

```javascript
// System questionnaire templates
db.questionnaire_templates.insertMany([
  {
    name: "ISO 27001 Basic Assessment",
    category: "ISO27001",
    version: "1.0",
    is_system: true,
    default_passing_score: 70,
    estimated_minutes: 30,
    topics: [
      { id: "access-control", name: "Access Control", order: 1 },
      { id: "data-protection", name: "Data Protection", order: 2 },
      { id: "incident-response", name: "Incident Response", order: 3 }
    ],
    created_at: new Date(),
    updated_at: new Date(),
    published_at: new Date()
  },
  {
    name: "GDPR Compliance Checklist",
    category: "GDPR",
    version: "1.0",
    is_system: true,
    default_passing_score: 80,
    estimated_minutes: 45,
    topics: [
      { id: "data-processing", name: "Data Processing", order: 1 },
      { id: "consent", name: "Consent Management", order: 2 },
      { id: "data-subject-rights", name: "Data Subject Rights", order: 3 }
    ],
    created_at: new Date(),
    updated_at: new Date(),
    published_at: new Date()
  },
  {
    name: "NIS2 Security Assessment",
    category: "NIS2",
    version: "1.0",
    is_system: true,
    default_passing_score: 75,
    estimated_minutes: 60,
    topics: [
      { id: "risk-management", name: "Risk Management", order: 1 },
      { id: "supply-chain", name: "Supply Chain Security", order: 2 },
      { id: "incident-handling", name: "Incident Handling", order: 3 }
    ],
    created_at: new Date(),
    updated_at: new Date(),
    published_at: new Date()
  }
])
```

---

## Quality Checklist

- [x] Every entity has clear purpose and attributes
- [x] All relationships documented with cardinality
- [x] API query patterns identified
- [x] Performance strategy defined
- [x] Scale projections provided
- [x] Database choice justified
- [x] All assumptions tagged
- [x] State machines documented
- [x] Index definitions complete
- [x] Data flows documented
- [x] Retention policies defined
- [x] Business rules captured

---

## Document Metadata

| Field | Value |
|-------|-------|
| Version | 1.0 |
| Created | 2024-12-29 |
| Author | Data Architecture AI Agent |
| Status | Draft |
| Review Required | Yes |

---

## Appendix: Go Model Files Structure

Recommended file organization for implementation:

```
internal/models/
├── organization.go      # Organization, Address, OrganizationSettings
├── user.go              # User, UserRole
├── template.go          # QuestionnaireTemplate, TemplateTopic
├── questionnaire.go     # Questionnaire, QuestionnaireTopic, ScoringMode
├── question.go          # Question, QuestionOption, QuestionType
├── relationship.go      # CompanySupplierRelationship, StatusChange
├── requirement.go       # Requirement, RequirementStatusChange
├── response.go          # SupplierResponse
├── submission.go        # QuestionnaireSubmission, SubmissionAnswer, TopicScore
├── verification.go      # CheckFixVerification, CategoryGrade
├── secure_link.go       # SecureLink
└── audit.go             # AuditLog, AuditAction
```
