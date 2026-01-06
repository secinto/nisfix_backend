# Unified System Blueprint - NisFix B2B Supplier Security Portal

## Document Metadata
| Field | Value |
|-------|-------|
| Version | 1.0 |
| Created | 2025-12-29 |
| Status | Unified Blueprint |
| Source Plans | system-architecture-plan.md, ui-ux-plan.md, data-architecture-plan.md |

---

## Synthesis Summary

This unified blueprint integrates the system architecture, UI/UX, and data architecture plans for the NisFix B2B Supplier Security Portal. The synthesis resolves naming inconsistencies, aligns API contracts with frontend expectations, and establishes canonical data models.

#SYNTHESIS_RATIONALE: The synthesis prioritizes data layer naming conventions as the source of truth since the database schema forms the foundation. API contracts are then aligned to match, with frontend adapting to the canonical API. This bottom-up approach ensures consistency from storage through to presentation.

---

## Conflict Resolution

### CONFLICT 1: Organization Type Naming

#CONFLICT_DETECTED: Inconsistent naming for organization types across plans

| Plan | Naming Convention |
|------|-------------------|
| System Architecture | `company`, `supplier` (lowercase) |
| UI/UX Plan | `'company'` \| `'supplier'` (lowercase string literals) |
| Data Architecture | `COMPANY`, `SUPPLIER` (UPPERCASE constants) |

#RESOLUTION: Use UPPERCASE constants in Go code and database storage, lowercase in JSON API responses
#NAMING_CONFLICT: OrganizationType constants standardized to UPPERCASE in code, lowercase in JSON serialization
#UNIFIED_NAMING: Go code uses `OrganizationTypeCompany = "COMPANY"`, JSON output uses `"type": "company"`

**Canonical Definition:**
```go
type OrganizationType string

const (
    OrganizationTypeCompany  OrganizationType = "COMPANY"
    OrganizationTypeSupplier OrganizationType = "SUPPLIER"
)

// JSON marshaling converts to lowercase
func (ot OrganizationType) MarshalJSON() ([]byte, error) {
    return json.Marshal(strings.ToLower(string(ot)))
}
```

#SYNTHESIS_RATIONALE: UPPERCASE in code follows Go conventions for constants; lowercase in JSON follows REST API conventions and matches frontend expectations.

---

### CONFLICT 2: User Role Naming

#CONFLICT_DETECTED: Inconsistent role naming conventions

| Plan | Naming Convention |
|------|-------------------|
| System Architecture | `admin`, `viewer` (lowercase) |
| Data Architecture | `ADMIN`, `VIEWER` (UPPERCASE) |
| UI/UX Plan | Uses auth context helpers `isAdmin()` |

#RESOLUTION: Same pattern as OrganizationType - UPPERCASE in code, lowercase in JSON
#NAMING_CONFLICT: UserRole constants standardized
#UNIFIED_NAMING: `UserRoleAdmin = "ADMIN"` internally, `"role": "admin"` in JSON

**Canonical Definition:**
```go
type UserRole string

const (
    UserRoleAdmin  UserRole = "ADMIN"
    UserRoleViewer UserRole = "VIEWER"
)
```

---

### CONFLICT 3: Supplier Classification Values

#CONFLICT_DETECTED: Different classification categories across plans

| Plan | Classifications |
|------|-----------------|
| System Architecture | `critical`, `important`, `standard` |
| UI/UX Plan | `critical`, `standard` (only 2 options in InviteDialog) |
| Data Architecture | `CRITICAL`, `HIGH`, `MEDIUM`, `LOW` (4 levels) |

#RESOLUTION: Adopt 3-tier classification from System Architecture as compromise
#PARAMETER_MISMATCH: Supplier classification aligned to 3 values
#UNIFIED_NAMING: `CRITICAL`, `IMPORTANT`, `STANDARD` (JSON: `critical`, `important`, `standard`)

**Canonical Definition:**
```go
type SupplierClassification string

const (
    SupplierClassificationCritical  SupplierClassification = "CRITICAL"
    SupplierClassificationImportant SupplierClassification = "IMPORTANT"
    SupplierClassificationStandard  SupplierClassification = "STANDARD"
)
```

#SYNTHESIS_RATIONALE: Three tiers balance granularity needs (data plan's 4 levels too complex) with simplicity (UI plan's 2 levels insufficient for risk-based prioritization).
#TRADE_OFF_DECISION: Chose 3-tier over 4-tier to reduce UI complexity while maintaining meaningful classification

---

### CONFLICT 4: CheckFix API Endpoint Naming

#CONFLICT_DETECTED: Different endpoint paths for CheckFix linking

| Plan | Endpoint |
|------|----------|
| System Architecture | `POST /api/v1/checkfix/link` |
| UI/UX Plan | `POST /api/v1/checkfix/link-account` |

#RESOLUTION: Use System Architecture naming (shorter, consistent with REST conventions)
#ENDPOINT_ALIGNMENT: CheckFix endpoints aligned to `/api/v1/checkfix/link`
#UNIFIED_CONTRACT: Frontend must update API calls to use `/api/v1/checkfix/link`

**Canonical Endpoints:**
```
POST /api/v1/checkfix/link      # Link CheckFix account
POST /api/v1/checkfix/verify    # Verify/refresh report
GET  /api/v1/checkfix/status    # Get current status
DELETE /api/v1/checkfix/link    # Unlink account
```

---

### CONFLICT 5: Requirement Status Values

#CONFLICT_DETECTED: Slightly different status values across plans

| Plan | Statuses |
|------|----------|
| System Architecture | `pending`, `in_progress`, `submitted`, `approved`, `rejected`, `expired` |
| Data Architecture | `PENDING`, `IN_PROGRESS`, `SUBMITTED`, `APPROVED`, `REJECTED`, `UNDER_REVIEW`, `EXPIRED` |

#RESOLUTION: Include `UNDER_REVIEW` from data plan (supports revision workflow)
#TYPE_MISMATCH: RequirementStatus aligned to include all states
#UNIFIED_NAMING: Add `UNDER_REVIEW` to system architecture specification

**Canonical Definition:**
```go
type RequirementStatus string

const (
    RequirementStatusPending     RequirementStatus = "PENDING"
    RequirementStatusInProgress  RequirementStatus = "IN_PROGRESS"
    RequirementStatusSubmitted   RequirementStatus = "SUBMITTED"
    RequirementStatusUnderReview RequirementStatus = "UNDER_REVIEW"  // Added
    RequirementStatusApproved    RequirementStatus = "APPROVED"
    RequirementStatusRejected    RequirementStatus = "REJECTED"
    RequirementStatusExpired     RequirementStatus = "EXPIRED"
)
```

#SYNTHESIS_RATIONALE: `UNDER_REVIEW` supports the revision workflow mentioned in system architecture's `POST /api/v1/responses/:id/request-revision` endpoint.

---

### CONFLICT 6: Response Status vs Requirement Status

#CONFLICT_DETECTED: System Architecture uses `ResponseStatus` on SupplierResponse, Data Architecture merges status tracking

| Plan | Approach |
|------|----------|
| System Architecture | Separate `ResponseStatus` enum: `in_progress`, `submitted`, `approved`, `rejected`, `revision_requested` |
| Data Architecture | SupplierResponse has no status field; uses Requirement status |

#RESOLUTION: SupplierResponse does NOT have its own status; Requirement status is source of truth
#INTERFACE_RECONCILIATION: Response status derived from parent Requirement
#UNIFIED_STRUCTURE: Remove ResponseStatus from SupplierResponse model; query via Requirement.status

**Canonical Decision:**
- `Requirement.status` tracks the overall state of the requirement/response lifecycle
- `SupplierResponse` captures submission data and scores only
- API responses include status from the parent Requirement

#SYNTHESIS_RATIONALE: Single source of truth for status prevents synchronization issues. Requirement is the business entity; Response is the submission artifact.

---

### CONFLICT 7: Questions API Endpoint

#CONFLICT_DETECTED: UI plan expects separate questions endpoint not in system architecture

| Plan | Endpoint |
|------|----------|
| UI/UX Plan | `GET /api/v1/requirements/:id/questions` (for QuestionnaireFill page) |
| System Architecture | Questions returned embedded in `GET /api/v1/questionnaires/:id` |

#RESOLUTION: Use embedded questions approach from System Architecture
#ENDPOINT_ALIGNMENT: Questions fetched via questionnaire endpoint, not separate endpoint
#CONTRACT_ALIGNMENT: UI fetches `GET /api/v1/questionnaires/:id` to get questions

**Canonical Flow:**
1. `GET /api/v1/requirements/:id` returns requirement with `questionnaire_id`
2. `GET /api/v1/questionnaires/:questionnaire_id` returns questionnaire with embedded questions

#SYNTHESIS_RATIONALE: Two-call approach is cleaner than a requirement-specific questions endpoint. Questions belong to questionnaires, not requirements.

---

### CONFLICT 8: CheckFix Grade Values

#CONFLICT_DETECTED: Inconsistent grade scale definitions

| Plan | Grades |
|------|--------|
| System Architecture | `A`, `B`, `C`, `D`, `E`, `F` (implied by min_checkfix_grade: "C") |
| Data Architecture | `A`, `B`, `C`, `D`, `F` (explicitly "no E grade") |
| UI/UX Plan | `A`, `B`, `C`, `D`, `E`, `F` (GradeDisplay component) |

#RESOLUTION: Use standard letter grades WITHOUT 'E' (matches academic grading)
#TYPE_MISMATCH: CheckFix grades are A, B, C, D, F only
#UNIFIED_NAMING: Remove 'E' from grade options in UI GradeDisplay component

**Canonical Definition:**
```go
type CheckFixGrade string

const (
    CheckFixGradeA CheckFixGrade = "A"
    CheckFixGradeB CheckFixGrade = "B"
    CheckFixGradeC CheckFixGrade = "C"
    CheckFixGradeD CheckFixGrade = "D"
    CheckFixGradeF CheckFixGrade = "F"
)
```

#SYNTHESIS_RATIONALE: Data Architecture explicitly documents "no E grade" which aligns with standard academic grading (A-D, F). UI must be updated.

---

### CONFLICT 9: Organization Settings Structure

#CONFLICT_DETECTED: Different settings fields across plans

| Plan | Settings Fields |
|------|-----------------|
| System Architecture | `default_due_days`, `require_checkfix`, `min_checkfix_grade`, `notification_emails` |
| Data Architecture | `default_language`, `notifications_enabled`, `reminder_days_before` |

#RESOLUTION: Merge all settings into unified structure
#INTERFACE_RECONCILIATION: OrganizationSettings includes all fields from both plans

**Canonical Definition:**
```go
type OrganizationSettings struct {
    // From System Architecture
    DefaultDueDays       int      `bson:"default_due_days" json:"default_due_days"`
    RequireCheckFix      bool     `bson:"require_checkfix" json:"require_checkfix"`
    MinCheckFixGrade     string   `bson:"min_checkfix_grade" json:"min_checkfix_grade"`
    NotificationEmails   []string `bson:"notification_emails" json:"notification_emails"`

    // From Data Architecture
    DefaultLanguage      string   `bson:"default_language" json:"default_language"`
    NotificationsEnabled bool     `bson:"notifications_enabled" json:"notifications_enabled"`
    ReminderDaysBefore   int      `bson:"reminder_days_before" json:"reminder_days_before"`
}
```

#SYNTHESIS_RATIONALE: Both sets of settings are valid business requirements. Merged structure supports all use cases.

---

### CONFLICT 10: User Fields - Name vs No Name

#CONFLICT_DETECTED: User entity structure differs

| Plan | User Fields |
|------|-------------|
| System Architecture | `id`, `email`, `role`, `organization_id`, `created_at`, `updated_at`, `last_login_at` |
| Data Architecture | Same + `name`, `is_active`, `language`, `timezone`, `deleted_at` |

#RESOLUTION: Include all fields from Data Architecture (more complete model)
#UNIFIED_STRUCTURE: User model includes name, is_active, language, timezone, soft delete

**Canonical Definition:**
```go
type User struct {
    ID             primitive.ObjectID `bson:"_id,omitempty" json:"id"`
    Email          string             `bson:"email" json:"email"`
    Name           string             `bson:"name,omitempty" json:"name,omitempty"`
    OrganizationID primitive.ObjectID `bson:"organization_id" json:"organization_id"`
    Role           UserRole           `bson:"role" json:"role"`
    IsActive       bool               `bson:"is_active" json:"is_active"`
    Language       string             `bson:"language" json:"language"`
    Timezone       string             `bson:"timezone,omitempty" json:"timezone,omitempty"`
    LastLoginAt    *time.Time         `bson:"last_login_at,omitempty" json:"last_login_at,omitempty"`
    CreatedAt      time.Time          `bson:"created_at" json:"created_at"`
    UpdatedAt      time.Time          `bson:"updated_at" json:"updated_at"`
    DeletedAt      *time.Time         `bson:"deleted_at,omitempty" json:"deleted_at,omitempty"`
}
```

#SYNTHESIS_RATIONALE: Data Architecture provides a more complete user model supporting internationalization and user preferences.

---

## Unified Data Flow

### Authentication Flow
#DATA_FLOW_VALIDATION: UI -> API -> DB consistency verified

```
1. UI: MagicLinkForm component
   - Collects: email
   - Validates: email format (Zod schema)

2. API: POST /api/v1/auth/request-link
   - Receives: { "email": "user@example.com" }
   - Returns: { "message": "magic_link_sent", "email": "..." }

3. DB:
   - Find User by email (optional, for existing user)
   - Create SecureLink with hashed token
   - TTL index auto-expires after 15 minutes

4. Email: Magic link sent via MailService

5. UI: User clicks email link -> /auth/verify/:token

6. API: GET /api/v1/auth/verify/:token
   - Validates SecureLink exists, not expired, not used
   - Creates/finds User and Organization
   - Returns: { access_token, refresh_token, user: {...} }

7. UI: Store tokens in localStorage, redirect to portal
```

#STATE_SYNC_STRATEGY: JWT tokens stored in localStorage; user profile cached in AuthContext

---

### Supplier Invitation Flow
#DATA_FLOW_VALIDATION: Full lifecycle validated

```
1. UI: InviteDialog component (Company Portal)
   - Collects: email, classification, message

2. API: POST /api/v1/suppliers
   - Auth: Company admin required
   - Creates: CompanySupplierRelationship (status: PENDING, supplier_id: null)
   - Creates: SecureLink (type: INVITATION)
   - Returns: { id, supplier: {...}, status: "pending", ... }

3. DB:
   - Insert CompanySupplierRelationship
   - Insert SecureLink with 7-day expiry

4. Email: Invitation sent to supplier email

5. Supplier clicks link -> /auth/verify/:token
   - Creates/finds Supplier Organization
   - Creates User for supplier
   - Updates CompanySupplierRelationship.supplier_id

6. UI: Supplier Dashboard shows pending invitation

7. API: POST /api/v1/companies/:id/accept
   - Updates relationship status to ACTIVE

8. DB: Update CompanySupplierRelationship, add to StatusHistory
```

---

### Questionnaire Response Flow
#DATA_FLOW_VALIDATION: Full submission lifecycle validated

```
1. Company creates Requirement:
   API: POST /api/v1/suppliers/:id/requirements
   DB: Create Requirement (status: PENDING)

2. Supplier views requirements:
   API: GET /api/v1/requirements
   UI: RequirementsList page

3. Supplier starts response:
   API: POST /api/v1/requirements/:id/responses
   DB: Create SupplierResponse (started_at: now)
   DB: Update Requirement status to IN_PROGRESS

4. Supplier fills questionnaire:
   API: GET /api/v1/questionnaires/:id (get questions)
   UI: QuestionnaireForm component
   API: PATCH /api/v1/responses/:id (save draft)

5. Supplier submits:
   API: POST /api/v1/responses/:id/submit
   DB: Create QuestionnaireSubmission with calculated scores
   DB: Update SupplierResponse with score/passed
   DB: Update Requirement status to SUBMITTED

6. Company reviews:
   API: GET /api/v1/responses/:id
   UI: ResponseViewer component
   API: POST /api/v1/responses/:id/approve OR /reject
   DB: Update Requirement status to APPROVED/REJECTED
```

---

## Component-API-Data Mapping

### Feature: User Authentication

#CONTRACT_ALIGNMENT: All three layers aligned

| Layer | Element | Details |
|-------|---------|---------|
| UI Component | MagicLinkForm | Props: `onSuccess?: () => void`, State: email, submitting |
| API Endpoint | POST /api/v1/auth/request-link | Auth: None, Body: `{ email: string }` |
| DB Collection | secure_links | Query: Insert new SecureLink |

| Layer | Element | Details |
|-------|---------|---------|
| UI Component | EmailVerification page | Auto-calls verify on mount |
| API Endpoint | GET /api/v1/auth/verify/:token | Auth: None, Param: 64-char token |
| DB Query | SecureLink + User + Organization | Multi-collection transaction |

#UNIFIED_CONTRACT: AuthResponse structure:
```typescript
interface AuthResponse {
  access_token: string;
  refresh_token: string;
  expires_in: number;
  user: {
    id: string;
    email: string;
    name?: string;
    role: "admin" | "viewer";
    organization: {
      id: string;
      name: string;
      type: "company" | "supplier";
      slug: string;
    };
  };
}
```

---

### Feature: Supplier Management (Company Portal)

| Layer | Element | Details |
|-------|---------|---------|
| UI Component | SupplierList page | DataTable with filtering |
| API Endpoint | GET /api/v1/suppliers | Query: status, classification, search, page, limit |
| DB Query | company_supplier_relationships | Aggregation with Organization lookup |

| Layer | Element | Details |
|-------|---------|---------|
| UI Component | InviteDialog | Form: email, classification |
| API Endpoint | POST /api/v1/suppliers | Body: email, company_name, classification, message |
| DB Query | CompanySupplierRelationship + SecureLink | Multi-document insert |

| Layer | Element | Details |
|-------|---------|---------|
| UI Component | SupplierDetail page | Tabs: Overview, Requirements, Responses |
| API Endpoint | GET /api/v1/suppliers/:id | Returns relationship with requirements |
| DB Query | CompanySupplierRelationship + Requirements | Aggregation with lookups |

#UNIFIED_CONTRACT: SupplierResponse structure:
```typescript
interface SupplierRelationship {
  id: string;
  supplier: {
    id: string;
    name: string;
    domain?: string;
    slug: string;
  };
  status: "pending" | "active" | "suspended" | "terminated" | "rejected";
  classification: "critical" | "important" | "standard";
  invited_at: string;
  accepted_at?: string;
  checkfix_status?: {
    linked: boolean;
    grade?: "A" | "B" | "C" | "D" | "F";
    verified_at?: string;
  };
}
```

---

### Feature: Questionnaire Management

| Layer | Element | Details |
|-------|---------|---------|
| UI Component | QuestionnaireList page | DataTable with status filter |
| API Endpoint | GET /api/v1/questionnaires | Query: status, search, page, limit |
| DB Collection | questionnaires | Index on company_id + status |

| Layer | Element | Details |
|-------|---------|---------|
| UI Component | QuestionnaireBuilder | Drag-drop editor |
| API Endpoint | POST /api/v1/questionnaires | Body: name, description, template_id?, questions[] |
| DB Collections | questionnaires + questions | Multi-document transaction |

#UNIFIED_CONTRACT: Questionnaire structure:
```typescript
interface Questionnaire {
  id: string;
  name: string;
  description?: string;
  status: "draft" | "published" | "archived";
  passing_score: number;
  scoring_mode: "PERCENTAGE" | "POINTS";
  topics: { id: string; name: string; order: number }[];
  questions: Question[];
  question_count: number;
  max_possible_score: number;
  created_at: string;
  updated_at: string;
}

interface Question {
  id: string;
  text: string;
  type: "SINGLE_CHOICE" | "MULTIPLE_CHOICE" | "TEXT" | "YES_NO";
  topic_id: string;
  weight: number;
  max_points: number;
  is_must_pass: boolean;
  order: number;
  options?: QuestionOption[];
}

interface QuestionOption {
  id: string;
  text: string;
  points: number;
  is_correct: boolean;
  order: number;
}
```

---

### Feature: Requirements (Supplier Portal)

| Layer | Element | Details |
|-------|---------|---------|
| UI Component | RequirementsList page | DataTable with due date indicators |
| API Endpoint | GET /api/v1/requirements | Query: status, company_id, type, overdue |
| DB Collection | requirements | Index on supplier_id + status + due_date |

| Layer | Element | Details |
|-------|---------|---------|
| UI Component | QuestionnaireFill page | QuestionnaireForm with progress |
| API Endpoint | GET /api/v1/requirements/:id + GET /api/v1/questionnaires/:qid | Two calls for data |
| DB Collections | requirements + questionnaires + questions | Multiple lookups |

#UNIFIED_CONTRACT: Requirement structure:
```typescript
interface Requirement {
  id: string;
  type: "questionnaire" | "checkfix";
  title: string;
  description?: string;
  company: { id: string; name: string };
  questionnaire?: { id: string; name: string; question_count: number };
  minimum_grade?: "A" | "B" | "C" | "D" | "F";
  status: "pending" | "in_progress" | "submitted" | "under_review" | "approved" | "rejected" | "expired";
  priority: "low" | "medium" | "high";
  due_date?: string;
  created_at: string;
  response?: SupplierResponseSummary;
}
```

---

### Feature: CheckFix Integration

| Layer | Element | Details |
|-------|---------|---------|
| UI Component | CheckFixPage / CheckFixLinkForm | Link account form |
| API Endpoint | POST /api/v1/checkfix/link | Body: checkfix_email, report_hash |
| DB Collections | organizations + checkfix_verifications | Update org, create verification |

| Layer | Element | Details |
|-------|---------|---------|
| UI Component | GradeDisplay | Shows A-F grade with color |
| API Endpoint | GET /api/v1/checkfix/status | Returns verification status |
| DB Collection | checkfix_verifications | Query by supplier_id |

#UNIFIED_CONTRACT: CheckFixStatus structure:
```typescript
interface CheckFixStatus {
  linked: boolean;
  checkfix_email?: string;
  domain?: string;
  verification?: {
    grade: "A" | "B" | "C" | "D" | "F";
    overall_score: number;
    categories: {
      category: string;
      grade: string;
      score: number;
    }[];
    findings: {
      critical: number;
      high: number;
      medium: number;
      low: number;
    };
    report_date: string;
    verified_at: string;
    expires_at: string;
  };
}
```

---

## Unified Type Definitions

#UNIFIED_STRUCTURE: Canonical types for all layers

### Enums (Go -> JSON mapping)

```go
// Organization Types
type OrganizationType string
const (
    OrganizationTypeCompany  OrganizationType = "COMPANY"   // JSON: "company"
    OrganizationTypeSupplier OrganizationType = "SUPPLIER"  // JSON: "supplier"
)

// User Roles
type UserRole string
const (
    UserRoleAdmin  UserRole = "ADMIN"   // JSON: "admin"
    UserRoleViewer UserRole = "VIEWER"  // JSON: "viewer"
)

// Supplier Classification
type SupplierClassification string
const (
    SupplierClassificationCritical  SupplierClassification = "CRITICAL"   // JSON: "critical"
    SupplierClassificationImportant SupplierClassification = "IMPORTANT"  // JSON: "important"
    SupplierClassificationStandard  SupplierClassification = "STANDARD"   // JSON: "standard"
)

// Relationship Status
type RelationshipStatus string
const (
    RelationshipStatusPending    RelationshipStatus = "PENDING"     // JSON: "pending"
    RelationshipStatusActive     RelationshipStatus = "ACTIVE"      // JSON: "active"
    RelationshipStatusRejected   RelationshipStatus = "REJECTED"    // JSON: "rejected"
    RelationshipStatusSuspended  RelationshipStatus = "SUSPENDED"   // JSON: "suspended"
    RelationshipStatusTerminated RelationshipStatus = "TERMINATED"  // JSON: "terminated"
)

// Requirement Status
type RequirementStatus string
const (
    RequirementStatusPending     RequirementStatus = "PENDING"
    RequirementStatusInProgress  RequirementStatus = "IN_PROGRESS"
    RequirementStatusSubmitted   RequirementStatus = "SUBMITTED"
    RequirementStatusUnderReview RequirementStatus = "UNDER_REVIEW"
    RequirementStatusApproved    RequirementStatus = "APPROVED"
    RequirementStatusRejected    RequirementStatus = "REJECTED"
    RequirementStatusExpired     RequirementStatus = "EXPIRED"
)

// Requirement Type
type RequirementType string
const (
    RequirementTypeQuestionnaire RequirementType = "QUESTIONNAIRE"  // JSON: "questionnaire"
    RequirementTypeCheckFix      RequirementType = "CHECKFIX"       // JSON: "checkfix"
)

// Questionnaire Status
type QuestionnaireStatus string
const (
    QuestionnaireStatusDraft     QuestionnaireStatus = "DRAFT"
    QuestionnaireStatusPublished QuestionnaireStatus = "PUBLISHED"
    QuestionnaireStatusArchived  QuestionnaireStatus = "ARCHIVED"
)

// Question Type
type QuestionType string
const (
    QuestionTypeSingleChoice   QuestionType = "SINGLE_CHOICE"
    QuestionTypeMultipleChoice QuestionType = "MULTIPLE_CHOICE"
    QuestionTypeText           QuestionType = "TEXT"
    QuestionTypeYesNo          QuestionType = "YES_NO"
)

// Secure Link Type
type SecureLinkType string
const (
    SecureLinkTypeAuth       SecureLinkType = "AUTH"
    SecureLinkTypeInvitation SecureLinkType = "INVITATION"
)

// CheckFix Grade
type CheckFixGrade string
const (
    CheckFixGradeA CheckFixGrade = "A"
    CheckFixGradeB CheckFixGrade = "B"
    CheckFixGradeC CheckFixGrade = "C"
    CheckFixGradeD CheckFixGrade = "D"
    CheckFixGradeF CheckFixGrade = "F"
)

// Priority
type Priority string
const (
    PriorityLow    Priority = "LOW"
    PriorityMedium Priority = "MEDIUM"
    PriorityHigh   Priority = "HIGH"
)
```

---

## API Specification (Canonical)

#ENDPOINT_ALIGNMENT: All endpoints verified against UI needs and DB capabilities

### Authentication (Public)
| Method | Endpoint | Purpose |
|--------|----------|---------|
| POST | /api/v1/auth/request-link | Send magic link email |
| GET | /api/v1/auth/verify/:token | Verify token, return JWT |
| POST | /api/v1/auth/refresh | Refresh token pair |
| GET | /api/v1/auth/profile | Get user profile (authenticated) |

### Organizations
| Method | Endpoint | Purpose | Auth |
|--------|----------|---------|------|
| GET | /api/v1/organizations/current | Get current org | Any |
| PATCH | /api/v1/organizations/current | Update org settings | Admin |

### Users
| Method | Endpoint | Purpose | Auth |
|--------|----------|---------|------|
| GET | /api/v1/users | List org users | Admin |
| POST | /api/v1/users | Invite user | Admin |
| PATCH | /api/v1/users/:id | Update user | Admin |
| DELETE | /api/v1/users/:id | Remove user | Admin |

### Company Portal - Suppliers
| Method | Endpoint | Purpose | Auth |
|--------|----------|---------|------|
| GET | /api/v1/suppliers | List suppliers | Company |
| POST | /api/v1/suppliers | Invite supplier | Company Admin |
| GET | /api/v1/suppliers/:id | Get supplier details | Company |
| PATCH | /api/v1/suppliers/:id | Update classification/status | Company Admin |
| POST | /api/v1/suppliers/:id/requirements | Create requirement | Company Admin |

### Company Portal - Questionnaires
| Method | Endpoint | Purpose | Auth |
|--------|----------|---------|------|
| GET | /api/v1/questionnaires | List questionnaires | Company |
| POST | /api/v1/questionnaires | Create questionnaire | Company Admin |
| GET | /api/v1/questionnaires/:id | Get with questions | Company/Supplier |
| PATCH | /api/v1/questionnaires/:id | Update questionnaire | Company Admin |
| DELETE | /api/v1/questionnaires/:id | Delete (draft only) | Company Admin |
| POST | /api/v1/questionnaires/:id/publish | Publish | Company Admin |
| POST | /api/v1/questionnaires/:id/archive | Archive | Company Admin |

### Company Portal - Questions
| Method | Endpoint | Purpose | Auth |
|--------|----------|---------|------|
| POST | /api/v1/questionnaires/:id/questions | Add question | Company Admin |
| PATCH | /api/v1/questionnaires/:qid/questions/:id | Update question | Company Admin |
| DELETE | /api/v1/questionnaires/:qid/questions/:id | Delete question | Company Admin |
| POST | /api/v1/questionnaires/:id/questions/reorder | Reorder questions | Company Admin |

### Company Portal - Templates
| Method | Endpoint | Purpose | Auth |
|--------|----------|---------|------|
| GET | /api/v1/questionnaire-templates | List templates | Company |
| GET | /api/v1/questionnaire-templates/:id | Get template | Company |
| POST | /api/v1/questionnaire-templates/:id/clone | Clone to questionnaire | Company Admin |

### Company Portal - Responses
| Method | Endpoint | Purpose | Auth |
|--------|----------|---------|------|
| GET | /api/v1/responses | List responses | Company |
| GET | /api/v1/responses/:id | Get response details | Company/Supplier |
| POST | /api/v1/responses/:id/approve | Approve response | Company Admin |
| POST | /api/v1/responses/:id/reject | Reject response | Company Admin |
| POST | /api/v1/responses/:id/request-revision | Request changes | Company Admin |

### Supplier Portal - Companies
| Method | Endpoint | Purpose | Auth |
|--------|----------|---------|------|
| GET | /api/v1/companies | List company relationships | Supplier |
| POST | /api/v1/companies/:id/accept | Accept invitation | Supplier Admin |
| POST | /api/v1/companies/:id/decline | Decline invitation | Supplier Admin |

### Supplier Portal - Requirements
| Method | Endpoint | Purpose | Auth |
|--------|----------|---------|------|
| GET | /api/v1/requirements | List requirements | Supplier |
| GET | /api/v1/requirements/:id | Get requirement | Supplier |
| POST | /api/v1/requirements/:id/responses | Start response | Supplier Admin |

### Supplier Portal - Responses
| Method | Endpoint | Purpose | Auth |
|--------|----------|---------|------|
| PATCH | /api/v1/responses/:id | Save draft answers | Supplier Admin |
| POST | /api/v1/responses/:id/submit | Submit response | Supplier Admin |

### Supplier Portal - CheckFix
| Method | Endpoint | Purpose | Auth |
|--------|----------|---------|------|
| GET | /api/v1/checkfix/status | Get linking status | Supplier |
| POST | /api/v1/checkfix/link | Link account | Supplier Admin |
| POST | /api/v1/checkfix/verify | Verify/refresh report | Supplier Admin |
| DELETE | /api/v1/checkfix/link | Unlink account | Supplier Admin |

### Health & Utility (Public)
| Method | Endpoint | Purpose |
|--------|----------|---------|
| GET | /health | Health check |
| GET | /api/v1/config | Public configuration |

---

## Implementation Roadmap

#IMPLEMENTATION_GUIDANCE: Clear sequence for builders

### Phase 1: Foundation (Week 1-2)

**Priority 1: Core Infrastructure**
1. Set up Go project structure with Gin
2. Implement MongoDB connection and database helper
3. Create configuration loading from environment
4. Implement JWT RS512 authentication (JWTService)
5. Set up middleware chain (CORS, logging, rate limiting)

**Priority 2: Authentication System**
1. Implement SecureLink repository and model
2. Implement User repository and model
3. Implement Organization repository and model
4. Build AuthService (magic link flow)
5. Build AuthHandler with endpoints
6. Integrate mail service client

**Database Setup:**
- Create all collections
- Apply indexes from data architecture plan
- Seed system questionnaire templates

### Phase 2: Company Portal Core (Week 3-4)

**Priority 1: Organization & Users**
1. OrganizationService and handler
2. UserService and handler
3. Organization settings management

**Priority 2: Supplier Relationships**
1. CompanySupplierRelationship model and repository
2. RelationshipService (invite, accept, status management)
3. Supplier handlers (list, invite, update)
4. Invitation email integration

**Priority 3: Questionnaire Management**
1. QuestionnaireTemplate repository
2. Questionnaire and Question models and repositories
3. QuestionnaireService (CRUD, publish, archive)
4. QuestionService (CRUD, reorder)
5. Template cloning functionality

### Phase 3: Supplier Portal & Responses (Week 5-6)

**Priority 1: Supplier Views**
1. Companies list endpoint for suppliers
2. Accept/decline invitation endpoints
3. Requirements list endpoint for suppliers

**Priority 2: Response System**
1. Requirement model and repository
2. SupplierResponse model and repository
3. QuestionnaireSubmission model and repository
4. RequirementService (create, assign)
5. ResponseService (start, save draft, submit)
6. ScoreService (calculate scores)

**Priority 3: Review Workflow**
1. Response approval/rejection endpoints
2. Revision request workflow
3. Status history tracking
4. Email notifications for status changes

### Phase 4: CheckFix Integration (Week 7)

1. CheckFixVerification model and repository
2. CheckFix external API client
3. CheckFixService (link, verify, status)
4. CheckFix requirement type handling
5. Grade-based requirement validation

### Phase 5: Polish & Production (Week 8)

1. Comprehensive error handling
2. Request validation with detailed errors
3. Audit logging implementation
4. Rate limiting refinement
5. API documentation (OpenAPI/Swagger)
6. Health check endpoint with DB status
7. Production configuration
8. Security hardening

#DEFERRED_DECISION: Advanced features for later phases:
- Real-time notifications (WebSocket)
- Advanced reporting/analytics
- Bulk operations
- API versioning strategy (v2)
- Caching layer

---

## Consolidated Assumptions

### Authentication Assumptions
#ASSUMPTION: Magic link tokens are 64-character cryptographically random strings
#ASSUMPTION: Auth links expire in 15 minutes, invitation links in 7 days
#ASSUMPTION: Access tokens have 1-hour expiry, refresh tokens have 30-day expiry
#ASSUMPTION: Refresh tokens are single-use and rotated on each refresh
#ASSUMPTION: JWT RS512 private key stored on filesystem with 0600 permissions

### Data Assumptions
#ASSUMPTION: Email is unique across entire system (not per organization)
#ASSUMPTION: Users belong to exactly ONE organization
#ASSUMPTION: Slug generated from name, must be URL-safe lowercase alphanumeric with hyphens
#ASSUMPTION: All timestamps stored as UTC
#ASSUMPTION: MongoDB IDs serialized as 24-character hex strings in JSON

### Business Rule Assumptions
#ASSUMPTION: First user to register for an email domain becomes admin
#ASSUMPTION: Email domain maps to organization (may need manual override)
#ASSUMPTION: Suppliers can decline invitations without penalty
#ASSUMPTION: Questionnaires immutable once published (create new version)
#ASSUMPTION: IsMustPass questions cause automatic fail regardless of total score
#ASSUMPTION: Scoring snapshot stored at submission time

### Integration Assumptions
#ASSUMPTION: External mail service has compatible API
#ASSUMPTION: CheckFix API provides report verification by hash
#ASSUMPTION: CheckFix reports have unique, stable hashes
#ASSUMPTION: CheckFix API rate limits: 100 requests/hour
#ASSUMPTION: CheckFix verifications valid for 30 days

### UI/UX Assumptions
#ASSUMPTION: Primary users are on desktop/laptop devices
#ASSUMPTION: B2B users expect professional, business-focused interfaces
#ASSUMPTION: Users prefer explicit save over auto-save for questionnaires
#ASSUMPTION: All times displayed in user's local timezone
#ASSUMPTION: Pagination defaults to 20 items per page

### Performance Assumptions
#ASSUMPTION: Expected load: <1000 concurrent users, <100 requests/second
#ASSUMPTION: Write volume is 10% of read volume
#ASSUMPTION: Dashboard aggregations cached for 5 minutes

---

## Remaining Uncertainties

#SYNTHESIS_GAP: Areas needing further clarification

1. **Email Domain to Organization Mapping**
   - What happens when email domain is ambiguous (e.g., gmail.com)?
   - How to handle manual organization assignment?

2. **Questionnaire Versioning**
   - How to handle questionnaire updates after requirements assigned?
   - Should old submissions be linked to questionnaire version?

3. **CheckFix Integration Details**
   - Exact CheckFix API specification not confirmed
   - Handling of CheckFix API downtime

4. **Multi-language Support**
   - Are questionnaire templates translated?
   - How to handle language for email templates?

#SYNTHESIS_UNCERTAINTY: Unresolved questions

1. Should we show predicted pass/fail during questionnaire fill, or just score percentage?
2. What timezone to use for due dates - UTC or user's timezone?
3. How to handle supplier classification changes after requirements assigned?

#VALIDATION_NEEDED: Areas requiring testing

1. Token expiry edge cases (refresh while access expired)
2. Concurrent questionnaire submission handling
3. Large questionnaire performance (100+ questions)
4. Audit log write performance at scale

---

## Trade-Off Documentation

#TRADE_OFF_DECISION: 3-tier classification chosen over 4-tier
- Pro: Simpler UI, clearer mental model
- Con: Less granularity for risk assessment
- Decision: Simplicity wins for MVP; can expand later

#TRADE_OFF_DECISION: Embedded questions in API response vs separate endpoint
- Pro: Single request for questionnaire data
- Con: Larger response payload
- Decision: Embedded chosen for reduced latency

#TRADE_OFF_DECISION: Status on Requirement only, not on SupplierResponse
- Pro: Single source of truth, simpler state management
- Con: Additional join needed to get response with status
- Decision: Consistency wins over minor query complexity

#TRADE_OFF_DECISION: UPPERCASE enums in Go with lowercase JSON serialization
- Pro: Follows both Go and REST conventions
- Con: Requires custom marshaling
- Decision: Convention compliance worth the overhead

#PRIORITY_RESOLUTION: Database naming conventions (data plan) take precedence over API naming (system plan) when in conflict, as database is foundational.

---

## Quality Checklist

- [x] All three plans loaded and analyzed
- [x] Naming conflicts resolved (10 conflicts identified and resolved)
- [x] Type mismatches aligned
- [x] API endpoints matched to UI and DB
- [x] Data flow validated end-to-end
- [x] All critical exports incorporated
- [x] Implementation has clear guidance (8-week roadmap)
- [x] Unified blueprint saved to docs/unified-blueprint.md

---

## Appendix A: File Structure (Backend)

```
nisfix_backend/
├── cmd/
│   └── server/
│       └── main.go
├── internal/
│   ├── auth/
│   │   └── jwt.go
│   ├── config/
│   │   └── config.go
│   ├── database/
│   │   └── mongodb.go
│   ├── handlers/
│   │   ├── auth_handler.go
│   │   ├── organization_handler.go
│   │   ├── user_handler.go
│   │   ├── supplier_handler.go
│   │   ├── questionnaire_handler.go
│   │   ├── requirement_handler.go
│   │   ├── response_handler.go
│   │   └── checkfix_handler.go
│   ├── middleware/
│   │   ├── auth.go
│   │   ├── cors.go
│   │   ├── logger.go
│   │   ├── rate_limiter.go
│   │   └── guards.go
│   ├── models/
│   │   ├── organization.go
│   │   ├── user.go
│   │   ├── template.go
│   │   ├── questionnaire.go
│   │   ├── question.go
│   │   ├── relationship.go
│   │   ├── requirement.go
│   │   ├── response.go
│   │   ├── submission.go
│   │   ├── verification.go
│   │   ├── secure_link.go
│   │   └── audit.go
│   ├── repository/
│   │   ├── user_repo.go
│   │   ├── organization_repo.go
│   │   ├── questionnaire_repo.go
│   │   ├── relationship_repo.go
│   │   ├── requirement_repo.go
│   │   ├── response_repo.go
│   │   └── secure_link_repo.go
│   ├── services/
│   │   ├── auth_service.go
│   │   ├── organization_service.go
│   │   ├── user_service.go
│   │   ├── questionnaire_service.go
│   │   ├── relationship_service.go
│   │   ├── requirement_service.go
│   │   ├── response_service.go
│   │   ├── score_service.go
│   │   ├── checkfix_service.go
│   │   └── mail_service.go
│   └── dto/
│       ├── requests.go
│       └── responses.go
├── pkg/
│   └── validator/
│       └── validator.go
├── go.mod
├── go.sum
└── Makefile
```

---

## Appendix B: MongoDB Collections Summary

| Collection | Primary Index | Key Secondary Indexes |
|------------|---------------|----------------------|
| organizations | _id | slug (unique), domain (unique sparse), type |
| users | _id | email (unique), organization_id |
| questionnaire_templates | _id | category + is_system, text search |
| questionnaires | _id | company_id + status |
| questions | _id | questionnaire_id + order |
| company_supplier_relationships | _id | company_id + supplier_id (unique sparse), invited_email |
| requirements | _id | company_id + status + due_date, supplier_id + status |
| supplier_responses | _id | requirement_id (unique), supplier_id |
| questionnaire_submissions | _id | response_id (unique), questionnaire_id |
| checkfix_verifications | _id | response_id (unique), supplier_id, expires_at |
| secure_links | _id | secure_identifier (unique), email, expires_at (TTL) |
| audit_logs | _id | actor_user_id, resource_type + resource_id, created_at |

---

*Document generated by Plan Synthesis Specialist. All conflicts resolved and integration points validated.*
