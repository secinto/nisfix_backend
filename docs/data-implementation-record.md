# Data Implementation Record

**Date:** 2025-12-29
**Implementation:** MongoDB Data Layer for NisFix B2B Supplier Security Portal

---

## Schema Implementation

### Collections Created

| Collection | Description | Key Fields |
|------------|-------------|------------|
| `organizations` | Company and Supplier entities | id, type, name, slug, domain, settings |
| `users` | User accounts with roles | id, email, name, organization_id, role, is_active |
| `secure_links` | Magic link tokens | id, secure_identifier, type, email, expires_at |
| `questionnaire_templates` | Pre-defined templates (ISO27001, GDPR, NIS2) | id, name, category, is_system, topics |
| `questionnaires` | Company-customized questionnaires | id, company_id, name, status, passing_score |
| `questions` | Individual questions with options | id, questionnaire_id, text, type, options |
| `company_supplier_relationships` | Company-Supplier relationships | id, company_id, supplier_id, status, classification |
| `requirements` | Requirements assigned to suppliers | id, relationship_id, type, status, due_date |
| `supplier_responses` | Supplier responses to requirements | id, requirement_id, supplier_id, score, passed |
| `questionnaire_submissions` | Submitted questionnaire answers | id, response_id, answers, total_score, passed |
| `checkfix_verifications` | CheckFix report verifications | id, response_id, overall_grade, verified_at |
| `audit_logs` | Activity audit trail | id, actor_user_id, action, resource_type, resource_id |

### Model Files Created

- `/internal/models/organization.go` - Organization, Address, OrganizationSettings types
- `/internal/models/user.go` - User, UserRole types
- `/internal/models/secure_link.go` - SecureLink, SecureLinkType types
- `/internal/models/questionnaire_template.go` - QuestionnaireTemplate, TemplateTopic, TemplateCategory types
- `/internal/models/questionnaire.go` - Questionnaire, QuestionnaireTopic, QuestionnaireStatus, ScoringMode types
- `/internal/models/question.go` - Question, QuestionOption, QuestionType types
- `/internal/models/relationship.go` - CompanySupplierRelationship, StatusChange, RelationshipStatus, SupplierClassification types
- `/internal/models/requirement.go` - Requirement, RequirementStatusChange, RequirementType, RequirementStatus, Priority types
- `/internal/models/response.go` - SupplierResponse, DraftAnswer types
- `/internal/models/submission.go` - QuestionnaireSubmission, SubmissionAnswer, TopicScore types
- `/internal/models/verification.go` - CheckFixVerification, CategoryGrade, CheckFixGrade types
- `/internal/models/audit.go` - AuditLog, AuditAction, AuditLogBuilder types
- `/internal/models/errors.go` - Domain-specific error definitions

---

## Relationships

| From | To | Type | Foreign Key |
|------|-----|------|-------------|
| Organization | User | 1:N | user.organization_id |
| Organization (Company) | CompanySupplierRelationship | 1:N | relationship.company_id |
| Organization (Supplier) | CompanySupplierRelationship | 1:N | relationship.supplier_id |
| Organization (Company) | Questionnaire | 1:N | questionnaire.company_id |
| QuestionnaireTemplate | Questionnaire | 1:N | questionnaire.template_id |
| Questionnaire | Question | 1:N | question.questionnaire_id |
| CompanySupplierRelationship | Requirement | 1:N | requirement.relationship_id |
| Requirement | SupplierResponse | 1:1 | response.requirement_id |
| SupplierResponse | QuestionnaireSubmission | 1:0..1 | submission.response_id |
| SupplierResponse | CheckFixVerification | 1:0..1 | verification.response_id |

---

## Indexes

#INDEX_IMPLEMENTATION: All indexes created per data architecture plan

### Organizations Collection
- `idx_slug_unique` - Unique slug for URL-friendly identifiers
- `idx_domain_unique_sparse` - Unique domain for supplier verification (sparse)
- `idx_type_created` - Query by type for listing
- `idx_deleted_at_sparse` - Soft delete filtering

### Users Collection
- `idx_email_unique` - Unique email for authentication
- `idx_org_role` - Query users by organization and role
- `idx_org_active` - Active users filter
- `idx_deleted_at_sparse` - Soft delete filtering

### Secure Links Collection
- `idx_secure_identifier_unique` - Unique token lookup
- `idx_expires_at_ttl` - TTL index for automatic expiration
- `idx_email_created` - Rate limiting by email

### Questionnaire Templates Collection
- `idx_category_system` - Browse by category
- `idx_created_by_org_sparse` - Company's custom templates
- `idx_text_search` - Full-text search on name, description, tags

### Questionnaires Collection
- `idx_company_status_created` - Company's questionnaires list

### Questions Collection
- `idx_questionnaire_order` - Get all questions for a questionnaire (ordered)
- `idx_questionnaire_topic_order` - Questions by topic for section display

### Relationships Collection
- `idx_company_supplier_unique_sparse` - Unique relationship per company-supplier pair
- `idx_company_status_class` - Company's suppliers list with status filter
- `idx_supplier_status` - Supplier's companies list
- `idx_invited_email_status` - Find pending invites by email

### Requirements Collection
- `idx_company_status_due` - Company's requirements with status
- `idx_supplier_status_due` - Supplier's requirements
- `idx_relationship_status` - Requirements by relationship
- `idx_due_date_status` - Due date monitoring for reminders
- `idx_status_due_date` - Expiration handling

### Supplier Responses Collection
- `idx_requirement_unique` - Unique response by requirement
- `idx_supplier_submitted` - Supplier's responses

### Questionnaire Submissions Collection
- `idx_response_unique` - Unique submission by response
- `idx_supplier_submitted` - Supplier's submission history
- `idx_questionnaire_submitted` - Analytics: questionnaire performance

### CheckFix Verifications Collection
- `idx_response_unique` - Unique verification by response
- `idx_supplier_verified` - Supplier's verifications
- `idx_expires_at` - Expiration check

### Audit Logs Collection
- `idx_actor_created` - Query by actor
- `idx_resource_created` - Query by resource
- `idx_org_created` - Query by organization
- `idx_action_created` - Query by action type
- `idx_created_at` - Time-based archival

---

## Query Patterns

#QUERY_INTERFACE: Available queries for backend

### Organization Queries
- Get organization by ID, slug, or domain
- List organizations by type with pagination
- Soft delete support

### User Queries
- Get user by ID or email
- List users by organization
- Update last login timestamp
- Soft delete support

### Secure Link Queries
- Get valid link by identifier
- Mark as used
- Invalidate links
- Rate limiting by email

### Questionnaire Template Queries
- List system templates by category
- List custom templates by organization
- Full-text search

### Questionnaire Queries
- List by company with status filter
- Update statistics (question count, max score)

### Question Queries
- List by questionnaire (ordered by topic and order)
- Bulk update order
- Calculate max possible score

### Relationship Queries
- Get by company and supplier
- Get pending by invited email
- List by company with classification filter
- List by supplier

### Requirement Queries
- List by company/supplier with status
- List overdue requirements
- List needing reminder
- Expire overdue batch operation

### Response Queries
- Get by requirement (unique)
- Save draft answers (upsert pattern)
- List by supplier

### Submission Queries
- Get by response (unique)
- Calculate pass rate by questionnaire

### Verification Queries
- Get latest by supplier
- List expiring verifications

---

## Query Optimizations

#QUERY_OPTIMIZATION: Optimizations applied

1. **Denormalized Fields**
   - Questionnaire.question_count, max_possible_score
   - SupplierResponse.score, max_score, passed
   - Requirement.company_id, supplier_id (from relationship)

2. **Embedded Documents**
   - Organization.address, settings
   - Questionnaire.topics
   - Question.options
   - CompanySupplierRelationship.status_history
   - Requirement.status_history
   - QuestionnaireSubmission.answers, topic_scores

3. **Compound Indexes**
   - Support common query patterns without additional lookups
   - Cover sorting and filtering in single index scan

4. **TTL Index**
   - Automatic cleanup of expired secure links

---

## ORM Configuration

#ORM_PATTERN: Repository pattern with MongoDB driver

### Repository Pattern
Each entity has a corresponding repository interface in `/internal/repository/interfaces.go` with MongoDB implementation.

### Files Created
- `/internal/repository/interfaces.go` - All repository interfaces
- `/internal/repository/organization_repo.go` - MongoOrganizationRepository
- `/internal/repository/user_repo.go` - MongoUserRepository
- `/internal/repository/secure_link_repo.go` - MongoSecureLinkRepository
- `/internal/repository/questionnaire_repo.go` - MongoQuestionnaireTemplateRepository, MongoQuestionnaireRepository
- `/internal/repository/question_repo.go` - MongoQuestionRepository
- `/internal/repository/relationship_repo.go` - MongoRelationshipRepository
- `/internal/repository/requirement_repo.go` - MongoRequirementRepository
- `/internal/repository/response_repo.go` - MongoResponseRepository, MongoSubmissionRepository, MongoVerificationRepository

### Features Implemented
- Pagination support with `PaginatedResult[T]`
- Soft delete pattern
- Transaction support via `WithTransaction`
- BeforeCreate/BeforeUpdate hooks on models

---

## Migrations

#MIGRATION_DECISION: Index-based migration at startup

### Strategy
- Indexes created at application startup via `IndexManager.CreateAllIndexes()`
- Idempotent - only creates indexes that don't exist
- System templates seeded via `Seeder.SeedQuestionnaireTemplates()`

### Database Files
- `/internal/database/mongodb.go` - MongoDB client with connection pooling
- `/internal/database/indexes.go` - IndexManager with all index definitions
- `/internal/database/seed.go` - Seeder for system templates

### Initialization Order
1. Connect to MongoDB
2. Create all indexes
3. Seed system templates (if not exists)

---

## Constraints and Validation

#CONSTRAINT_IMPLEMENTATION: Constraints added

### Unique Constraints (via indexes)
- Organization.slug (unique)
- Organization.domain (unique, sparse)
- User.email (unique)
- CompanySupplierRelationship.company_id + supplier_id (unique, sparse)
- SupplierResponse.requirement_id (unique)
- QuestionnaireSubmission.response_id (unique)
- CheckFixVerification.response_id (unique)
- SecureLink.secure_identifier (unique)

#DATA_VALIDATION: Validation rules

### Model-Level Validation
- Type enums with IsValid() methods
- Status transition validation (CanTransitionTo)
- BeforeCreate hooks for defaults
- BeforeUpdate hooks for timestamps

---

## Performance Tuning

#PERFORMANCE_TRADEOFF: Tradeoffs made

1. **Denormalization vs Consistency**
   - Chose denormalization for dashboard performance
   - Status and scores cached on parent documents
   - Trade: Manual sync required on updates

2. **Embedded vs Referenced**
   - Embedded: Small, rarely-independent data (options, status_history)
   - Referenced: Large, frequently-updated data (questions, submissions)
   - Trade: Embedded limits document size; referenced requires lookups

3. **Index Coverage vs Write Performance**
   - Multiple compound indexes for read optimization
   - Trade: Slower writes, more storage

#CACHE_IMPLEMENTATION: Caching strategy

- TTL-based caching for secure links (automatic expiration)
- Verification validity period (30 days)
- Dashboard aggregations recommended: 5-minute cache at application layer

---

## Assumptions

#COMPLETION_DRIVE: Key assumptions made

1. MongoDB 4.0+ for multi-document transaction support
2. Replica set configuration for production (high availability)
3. Application-level soft delete handling (not database triggers)
4. UTC timestamps for all date/time fields
5. ObjectID serialization as 24-character hex strings in JSON

#DATA_ASSUMPTION: Data characteristics assumed

1. 100-10K organizations, 50/50 Company/Supplier split
2. 5-50 users per organization
3. 10-100 questions per questionnaire
4. 10-500 supplier relationships per company
5. Write volume is 10% of read volume
6. Audit logs highest volume - archive strategy needed

---

## Outstanding Issues

1. **Audit Log Archival** - Need time-based partitioning for logs older than 1 year
2. **Sharding Strategy** - Consider sharding by organization_id at 1M+ documents
3. **Full-Text Search** - Text index on templates only; may need external search service for scale
4. **Connection Pooling** - Default pool size (100) may need tuning based on load

---

## File Structure Summary

```
/Users/skraxberger/development/checkfix-tools/nisfix_backend/
├── go.mod
├── internal/
│   ├── models/
│   │   ├── organization.go
│   │   ├── user.go
│   │   ├── secure_link.go
│   │   ├── questionnaire_template.go
│   │   ├── questionnaire.go
│   │   ├── question.go
│   │   ├── relationship.go
│   │   ├── requirement.go
│   │   ├── response.go
│   │   ├── submission.go
│   │   ├── verification.go
│   │   ├── audit.go
│   │   └── errors.go
│   ├── repository/
│   │   ├── interfaces.go
│   │   ├── organization_repo.go
│   │   ├── user_repo.go
│   │   ├── secure_link_repo.go
│   │   ├── questionnaire_repo.go
│   │   ├── question_repo.go
│   │   ├── relationship_repo.go
│   │   ├── requirement_repo.go
│   │   └── response_repo.go
│   └── database/
│       ├── mongodb.go
│       ├── indexes.go
│       └── seed.go
└── docs/
    ├── unified-blueprint.md
    ├── data-architecture-plan.md
    └── data-implementation-record.md
```
