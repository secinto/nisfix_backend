# System Architecture & API Plan - Nisfix Backend

## Executive Summary

Nisfix is a B2B Supplier Security Portal built with Go/Gin and MongoDB, enabling Companies to manage Supplier security compliance through questionnaires and CheckFix integration. The architecture follows a clean layered design with handlers, services, and repositories, using JWT RS512 passwordless authentication via magic links.

#PATH_RATIONALE: Chose layered architecture (handlers->services->repositories) for separation of concerns, testability, and consistency with existing checkfix_backend patterns. Passwordless auth chosen to reduce friction and security risks associated with password management.

---

## System Architecture

### High-Level Architecture Diagram

```
+------------------+     +------------------+     +------------------+
|   Frontend       |     |   Mail Service   |     |   CheckFix API   |
|  (nisfix_frontend)|     |   (External)     |     |   (External)     |
+--------+---------+     +--------+---------+     +--------+---------+
         |                        |                        |
         | HTTPS/JSON             | HTTP API               | HTTP API
         v                        v                        v
+--------+------------------------+------------------------+---------+
|                         Nisfix Backend                              |
|  +----------------------------------------------------------------+ |
|  |                     Gin HTTP Router                            | |
|  |  +------------------+  +------------------+  +----------------+ | |
|  |  | Auth Middleware  |->| Role Middleware  |->| Org Middleware | | |
|  +--+------------------+--+------------------+--+----------------+-+ |
|                                    |                                |
|  +----------------------------------------------------------------+ |
|  |                     Handler Layer                              | |
|  |  AuthHandler | OrgHandler | QuestionnaireHandler | ...         | |
|  +----------------------------------------------------------------+ |
|                                    |                                |
|  +----------------------------------------------------------------+ |
|  |                     Service Layer                              | |
|  |  AuthService | OrgService | QuestionnaireService | ScoreService| |
|  +----------------------------------------------------------------+ |
|                                    |                                |
|  +----------------------------------------------------------------+ |
|  |                    Repository Layer                            | |
|  |  UserRepo | OrgRepo | QuestionnaireRepo | RelationshipRepo     | |
|  +----------------------------------------------------------------+ |
|                                    |                                |
+------------------------------------+--------------------------------+
                                     |
                                     v
                        +------------------------+
                        |      MongoDB           |
                        |  (supplier_portal DB)  |
                        +------------------------+
```

#EXPORT_CRITICAL: All external communication uses HTTPS. Internal service communication is in-process (no microservices).

---

## Components

### 1. Configuration Component
- **Purpose**: Load and validate environment configuration
- **Responsibilities**:
  - Load environment variables
  - Validate required configuration
  - Provide typed config struct to other components
- **Technology**: Go standard library, envconfig

#EXPORT_INTERFACE: `Config` struct with GetConfig() singleton
#PLAN_UNCERTAINTY: Assumes environment variables for all secrets (no secret manager integration)

```go
type Config struct {
    DatabaseURI        string
    DatabaseName       string
    JWTPrivateKeyPath  string
    JWTPublicKeyPath   string
    MailServiceURL     string
    MailAPIKey         string
    CheckFixAPIURL     string
    CheckFixAPIKey     string
    ServerPort         string
    Environment        string  // development, staging, production
    MagicLinkBaseURL   string
    MagicLinkExpiry    time.Duration
    AccessTokenExpiry  time.Duration
    RefreshTokenExpiry time.Duration
}
```

### 2. Database Component
- **Purpose**: Manage MongoDB connections and provide database handles
- **Responsibilities**:
  - Establish MongoDB connection
  - Create indexes on startup
  - Provide collection handles
  - Health checking
- **Technology**: go.mongodb.org/mongo-driver

#EXPORT_INTERFACE: `Database` struct with GetCollection(name string) and EnsureIndexes()
#PLAN_UNCERTAINTY: Assumes MongoDB replica set for transaction support
#INTEGRATION_ASSUMPTION: Single MongoDB instance per environment

### 3. Auth Component (internal/auth)
- **Purpose**: JWT RS512 token generation and validation
- **Responsibilities**:
  - Generate access and refresh tokens
  - Validate and parse tokens
  - Extract claims from tokens
  - Key rotation support
- **Technology**: golang-jwt/jwt/v5

#EXPORT_INTERFACE:
```go
type JWTService interface {
    GenerateAccessToken(userID, orgID, role string, orgType OrgType) (string, error)
    GenerateRefreshToken(userID string) (string, error)
    ValidateAccessToken(token string) (*Claims, error)
    ValidateRefreshToken(token string) (*RefreshClaims, error)
}
```
#PATH_RATIONALE: RS512 chosen over HS256 for asymmetric signing - allows public key distribution for token validation in future service expansion
#SECURITY_ASSUMPTION: Private key stored securely on server filesystem with restricted permissions (0600)

### 4. Handler Layer (internal/handlers)
- **Purpose**: HTTP request handling, validation, and response formatting
- **Responsibilities**:
  - Parse and validate request data
  - Call appropriate service methods
  - Format and return responses
  - Error handling and status codes
- **Technology**: Gin framework

#PATH_RATIONALE: Thin handlers that delegate business logic to services - improves testability and reusability

#### Handler Files
| Handler | Purpose |
|---------|---------|
| `auth_handler.go` | Magic link request, verify, refresh, profile |
| `organization_handler.go` | Organization CRUD, settings |
| `questionnaire_handler.go` | Questionnaire and template management |
| `question_handler.go` | Question CRUD within questionnaires |
| `requirement_handler.go` | Company requirement creation/management |
| `response_handler.go` | Supplier responses, submissions, reviews |
| `relationship_handler.go` | Company-Supplier relationships, invitations |
| `checkfix_handler.go` | CheckFix account linking, verification |

### 5. Service Layer (internal/services)
- **Purpose**: Business logic and orchestration
- **Responsibilities**:
  - Implement business rules
  - Coordinate between repositories
  - Handle transactions when needed
  - Integrate with external services
- **Technology**: Go interfaces and structs

#EXPORT_INTERFACE: Each service defined as interface for testability

#### Services

##### AuthService
```go
type AuthService interface {
    RequestMagicLink(email string) error
    VerifyMagicLink(token string) (*TokenPair, *User, error)
    RefreshTokens(refreshToken string) (*TokenPair, error)
    GetProfile(userID string) (*UserProfile, error)
}
```
#PATH_RATIONALE: Separates authentication logic from HTTP handling
#INTEGRATION_ASSUMPTION: External mail service available via HTTP API

##### OrganizationService
```go
type OrganizationService interface {
    Create(input CreateOrgInput) (*Organization, error)
    GetByID(id string) (*Organization, error)
    GetBySlug(slug string) (*Organization, error)
    Update(id string, input UpdateOrgInput) (*Organization, error)
    Delete(id string) error
    GetByDomain(domain string) (*Organization, error)
}
```

##### UserService
```go
type UserService interface {
    Create(input CreateUserInput) (*User, error)
    GetByID(id string) (*User, error)
    GetByEmail(email string) (*User, error)
    Update(id string, input UpdateUserInput) (*User, error)
    GetOrCreate(email string, orgID string, role Role) (*User, error)
}
```
#PLAN_UNCERTAINTY: Assumes first user to register for domain becomes admin

##### QuestionnaireService
```go
type QuestionnaireService interface {
    // Templates
    ListTemplates(filter TemplateFilter) ([]QuestionnaireTemplate, error)
    GetTemplate(id string) (*QuestionnaireTemplate, error)
    CloneTemplate(templateID string, companyID string) (*Questionnaire, error)

    // Questionnaires
    Create(input CreateQuestionnaireInput) (*Questionnaire, error)
    GetByID(id string) (*Questionnaire, error)
    Update(id string, input UpdateQuestionnaireInput) (*Questionnaire, error)
    Delete(id string) error
    ListByCompany(companyID string, filter ListFilter) ([]Questionnaire, error)
    Publish(id string) error
    Archive(id string) error
}
```

##### QuestionService
```go
type QuestionService interface {
    Create(questionnaireID string, input CreateQuestionInput) (*Question, error)
    Update(id string, input UpdateQuestionInput) (*Question, error)
    Delete(id string) error
    ReorderQuestions(questionnaireID string, order []string) error
}
```

##### RelationshipService
```go
type RelationshipService interface {
    // Company actions
    InviteSupplier(companyID string, input InviteSupplierInput) (*Relationship, error)
    GetSuppliers(companyID string, filter RelationshipFilter) ([]Relationship, error)
    SuspendSupplier(companyID, supplierID string) error
    ReactivateSupplier(companyID, supplierID string) error
    TerminateSupplier(companyID, supplierID string) error

    // Supplier actions
    GetCompanies(supplierID string, filter RelationshipFilter) ([]Relationship, error)
    AcceptInvitation(supplierID, invitationID string) error
    DeclineInvitation(supplierID, invitationID string) error
}
```
#PLAN_UNCERTAINTY: Assumes supplier can decline invitations without penalty

##### RequirementService
```go
type RequirementService interface {
    Create(companyID string, input CreateRequirementInput) (*Requirement, error)
    GetByID(id string) (*Requirement, error)
    Update(id string, input UpdateRequirementInput) (*Requirement, error)
    Delete(id string) error
    ListByCompany(companyID string, filter ListFilter) ([]Requirement, error)
    ListBySupplier(supplierID string, filter ListFilter) ([]Requirement, error)
    AssignToSuppliers(requirementID string, supplierIDs []string) error
}
```

##### ResponseService
```go
type ResponseService interface {
    StartResponse(supplierID, requirementID string) (*SupplierResponse, error)
    SaveDraft(responseID string, answers []Answer) error
    Submit(responseID string) (*ScoreResult, error)
    GetByID(id string) (*SupplierResponse, error)
    ListByRequirement(requirementID string) ([]SupplierResponse, error)
    ListBySupplier(supplierID string, filter ListFilter) ([]SupplierResponse, error)

    // Review actions (Company)
    Approve(responseID string, feedback string) error
    Reject(responseID string, feedback string) error
    RequestRevision(responseID string, feedback string) error
}
```

##### ScoreService
```go
type ScoreService interface {
    CalculateScore(answers []Answer, questions []Question) (*ScoreResult, error)
    GetScoreBreakdown(responseID string) (*ScoreBreakdown, error)
}
```
#PATH_RATIONALE: Separate scoring service for complex calculation logic and potential future ML enhancements

##### CheckFixService
```go
type CheckFixService interface {
    LinkAccount(supplierID string, checkfixEmail string) error
    VerifyReport(supplierID string, reportHash string) (*CheckFixVerification, error)
    GetVerification(supplierID string) (*CheckFixVerification, error)
    RefreshVerification(supplierID string) (*CheckFixVerification, error)
    MeetsGradeRequirement(supplierID string, minGrade Grade) (bool, error)
}
```
#INTEGRATION_ASSUMPTION: CheckFix API provides report verification endpoint with hash-based lookup
#API_ASSUMPTION: CheckFix API uses API key authentication
#PLAN_UNCERTAINTY: Assumes CheckFix reports have unique, stable hashes

##### MailService
```go
type MailService interface {
    SendMagicLink(email string, magicLink string) error
    SendInvitation(email string, companyName string, inviteLink string) error
    SendRequirementNotification(email string, requirement RequirementNotification) error
    SendApprovalNotification(email string, response ResponseNotification) error
    SendRejectionNotification(email string, response ResponseNotification) error
}
```
#INTEGRATION_ASSUMPTION: External mail service has compatible API with checkfix_backend

### 6. Repository Layer (internal/repository)
- **Purpose**: Data persistence and retrieval
- **Responsibilities**:
  - MongoDB CRUD operations
  - Query building
  - Pagination
  - Index management
- **Technology**: MongoDB Go Driver

#EXPORT_INTERFACE: Each repository as interface

```go
type UserRepository interface {
    Create(user *User) error
    GetByID(id primitive.ObjectID) (*User, error)
    GetByEmail(email string) (*User, error)
    Update(id primitive.ObjectID, update *User) error
    Delete(id primitive.ObjectID) error
    ListByOrganization(orgID primitive.ObjectID, filter ListFilter) ([]User, int64, error)
}

type OrganizationRepository interface {
    Create(org *Organization) error
    GetByID(id primitive.ObjectID) (*Organization, error)
    GetBySlug(slug string) (*Organization, error)
    GetByDomain(domain string) (*Organization, error)
    Update(id primitive.ObjectID, update *Organization) error
    Delete(id primitive.ObjectID) error
    List(filter OrgFilter) ([]Organization, int64, error)
}

// Similar interfaces for:
// - QuestionnaireTemplateRepository
// - QuestionnaireRepository
// - QuestionRepository
// - RelationshipRepository
// - RequirementRepository
// - SupplierResponseRepository
// - SecureLinkRepository
// - CheckFixVerificationRepository
// - AuditLogRepository
```

### 7. Middleware Layer (internal/middleware)
- **Purpose**: Request processing pipeline
- **Responsibilities**:
  - Authentication verification
  - Authorization checks
  - Request logging
  - Rate limiting
  - CORS handling

#EXPORT_INTERFACE: Gin middleware functions

---

## Middleware Chain

```
Request
    |
    v
+-------------------+
| CORS Middleware   |  <- Handles preflight, sets headers
+-------------------+
    |
    v
+-------------------+
| Request Logger    |  <- Logs request/response, timing
+-------------------+
    |
    v
+-------------------+
| Rate Limiter      |  <- Per-IP rate limiting
+-------------------+
    |
    v
+-------------------+
| Auth Middleware   |  <- Validates JWT, sets user context
+-------------------+
    |
    v
+-------------------+
| Org Type Guard    |  <- Checks Company vs Supplier access
+-------------------+
    |
    v
+-------------------+
| Role Guard        |  <- Checks admin vs viewer permissions
+-------------------+
    |
    v
+-------------------+
| Resource Guard    |  <- Checks resource ownership
+-------------------+
    |
    v
Handler
```

#PATH_RATIONALE: Layered middleware allows flexible composition - public routes skip auth, role-specific routes add role guard

### Middleware Implementations

#### AuthMiddleware
```go
func AuthMiddleware(jwtService auth.JWTService) gin.HandlerFunc {
    return func(c *gin.Context) {
        token := extractBearerToken(c)
        if token == "" {
            c.AbortWithStatusJSON(401, ErrorResponse{Error: "missing_token"})
            return
        }
        claims, err := jwtService.ValidateAccessToken(token)
        if err != nil {
            c.AbortWithStatusJSON(401, ErrorResponse{Error: "invalid_token"})
            return
        }
        c.Set("userID", claims.UserID)
        c.Set("orgID", claims.OrgID)
        c.Set("role", claims.Role)
        c.Set("orgType", claims.OrgType)
        c.Next()
    }
}
```

#### OrgTypeGuard
```go
func RequireOrgType(allowedTypes ...OrgType) gin.HandlerFunc {
    return func(c *gin.Context) {
        orgType := c.GetString("orgType")
        for _, t := range allowedTypes {
            if orgType == string(t) {
                c.Next()
                return
            }
        }
        c.AbortWithStatusJSON(403, ErrorResponse{Error: "org_type_not_allowed"})
    }
}
```

#### RoleGuard
```go
func RequireRole(allowedRoles ...Role) gin.HandlerFunc {
    return func(c *gin.Context) {
        role := c.GetString("role")
        for _, r := range allowedRoles {
            if role == string(r) {
                c.Next()
                return
            }
        }
        c.AbortWithStatusJSON(403, ErrorResponse{Error: "insufficient_permissions"})
    }
}
```

#SECURITY_ASSUMPTION: Role checks happen after authentication - authenticated user always has valid role

---

## API Design

### Base URL Structure
```
/api/v1/...
```
#PATH_RATIONALE: Version prefix allows future API evolution without breaking existing clients

### Authentication Endpoints

#### RESOURCE: /api/v1/auth

##### POST /api/v1/auth/request-link
- **Purpose**: Request magic link for passwordless authentication
- **Authentication**: None (public)
- **Authorization**: None

#EXPORT_ENDPOINT: POST /api/v1/auth/request-link - Request magic link
#ENDPOINT_RATIONALE: Separate from verify to prevent timing attacks on token validation

**Request:**
#REQUEST_CONTRACT:
```json
{
  "email": "user@example.com"
}
```
| Field | Type | Required | Validation |
|-------|------|----------|------------|
| email | string | Yes | Valid email format |

**Response:**
#RESPONSE_CONTRACT:
```json
{
  "message": "magic_link_sent",
  "email": "user@example.com"
}
```
- Success: 200 OK
- Errors:
  - 400 Bad Request: Invalid email format
  - 429 Too Many Requests: Rate limit exceeded

#SECURITY_ASSUMPTION: Always return 200 even if email doesn't exist (prevents email enumeration)

##### GET /api/v1/auth/verify/:token
- **Purpose**: Verify magic link token and return JWT pair
- **Authentication**: Magic link token in URL
- **Authorization**: None

#EXPORT_ENDPOINT: GET /api/v1/auth/verify/:token - Verify magic link

**Request:**
#REQUEST_CONTRACT:
- URL Parameter: `token` (64-character hex string)

**Response:**
#RESPONSE_CONTRACT:
```json
{
  "access_token": "eyJhbG...",
  "refresh_token": "eyJhbG...",
  "expires_in": 3600,
  "user": {
    "id": "507f1f77bcf86cd799439011",
    "email": "user@example.com",
    "role": "admin",
    "organization": {
      "id": "507f1f77bcf86cd799439012",
      "name": "Acme Corp",
      "type": "company",
      "slug": "acme-corp"
    }
  }
}
```
- Success: 200 OK
- Errors:
  - 400 Bad Request: Invalid token format
  - 401 Unauthorized: Token expired or invalid
  - 404 Not Found: Token not found

##### POST /api/v1/auth/refresh
- **Purpose**: Exchange refresh token for new token pair
- **Authentication**: Refresh token in body
- **Authorization**: None

#EXPORT_ENDPOINT: POST /api/v1/auth/refresh - Refresh tokens

**Request:**
#REQUEST_CONTRACT:
```json
{
  "refresh_token": "eyJhbG..."
}
```

**Response:**
#RESPONSE_CONTRACT:
```json
{
  "access_token": "eyJhbG...",
  "refresh_token": "eyJhbG...",
  "expires_in": 3600
}
```
- Success: 200 OK
- Errors:
  - 400 Bad Request: Missing refresh token
  - 401 Unauthorized: Invalid or expired refresh token

##### GET /api/v1/auth/profile
- **Purpose**: Get current user profile
- **Authentication**: Required (Bearer token)
- **Authorization**: Any authenticated user

#EXPORT_ENDPOINT: GET /api/v1/auth/profile - Get user profile

**Request:**
#REQUEST_CONTRACT:
- Header: `Authorization: Bearer <access_token>`

**Response:**
#RESPONSE_CONTRACT:
```json
{
  "id": "507f1f77bcf86cd799439011",
  "email": "user@example.com",
  "role": "admin",
  "created_at": "2024-01-15T10:30:00Z",
  "organization": {
    "id": "507f1f77bcf86cd799439012",
    "name": "Acme Corp",
    "type": "company",
    "slug": "acme-corp",
    "domain": "acme.com",
    "settings": {
      "default_due_days": 30,
      "require_checkfix": true,
      "min_checkfix_grade": "C"
    }
  }
}
```
- Success: 200 OK
- Errors:
  - 401 Unauthorized: Invalid or missing token

---

### Organization Endpoints

#### RESOURCE: /api/v1/organizations

##### GET /api/v1/organizations/current
- **Purpose**: Get current user's organization details
- **Authentication**: Required
- **Authorization**: Any authenticated user

#EXPORT_ENDPOINT: GET /api/v1/organizations/current - Get current organization

**Response:**
#RESPONSE_CONTRACT:
```json
{
  "id": "507f1f77bcf86cd799439012",
  "name": "Acme Corp",
  "type": "company",
  "slug": "acme-corp",
  "domain": "acme.com",
  "logo_url": "https://...",
  "settings": {
    "default_due_days": 30,
    "require_checkfix": true,
    "min_checkfix_grade": "C",
    "notification_emails": ["security@acme.com"]
  },
  "created_at": "2024-01-15T10:30:00Z",
  "updated_at": "2024-01-20T14:00:00Z"
}
```

##### PATCH /api/v1/organizations/current
- **Purpose**: Update current organization settings
- **Authentication**: Required
- **Authorization**: Admin role only

#EXPORT_ENDPOINT: PATCH /api/v1/organizations/current - Update organization

**Request:**
#REQUEST_CONTRACT:
```json
{
  "name": "Acme Corporation",
  "settings": {
    "default_due_days": 45,
    "min_checkfix_grade": "B"
  }
}
```

---

### Company Portal Endpoints (OrgType: Company)

#### RESOURCE: /api/v1/suppliers

##### GET /api/v1/suppliers
- **Purpose**: List suppliers for company
- **Authentication**: Required
- **Authorization**: Company org type, any role

#EXPORT_ENDPOINT: GET /api/v1/suppliers - List company suppliers

**Request:**
#REQUEST_CONTRACT:
- Query Parameters:
  - `status`: Filter by status (pending, active, suspended, terminated)
  - `classification`: Filter by classification (critical, important, standard)
  - `search`: Search by name or domain
  - `page`: Page number (default: 1)
  - `limit`: Items per page (default: 20, max: 100)
  - `sort`: Sort field (name, created_at, status)
  - `order`: Sort order (asc, desc)

**Response:**
#RESPONSE_CONTRACT:
```json
{
  "data": [
    {
      "id": "507f1f77bcf86cd799439013",
      "supplier": {
        "id": "507f1f77bcf86cd799439014",
        "name": "Supplier Inc",
        "domain": "supplier.com",
        "slug": "supplier-inc"
      },
      "status": "active",
      "classification": "critical",
      "invited_at": "2024-01-10T10:00:00Z",
      "accepted_at": "2024-01-12T14:00:00Z",
      "requirements_count": 3,
      "pending_requirements": 1,
      "checkfix_status": {
        "linked": true,
        "grade": "B",
        "verified_at": "2024-01-15T09:00:00Z"
      }
    }
  ],
  "pagination": {
    "page": 1,
    "limit": 20,
    "total": 45,
    "total_pages": 3
  }
}
```

##### POST /api/v1/suppliers
- **Purpose**: Invite a new supplier
- **Authentication**: Required
- **Authorization**: Company org type, Admin role

#EXPORT_ENDPOINT: POST /api/v1/suppliers - Invite supplier

**Request:**
#REQUEST_CONTRACT:
```json
{
  "email": "contact@supplier.com",
  "company_name": "Supplier Inc",
  "classification": "critical",
  "message": "Please complete our security assessment"
}
```

**Response:**
#RESPONSE_CONTRACT:
```json
{
  "id": "507f1f77bcf86cd799439013",
  "supplier": {
    "id": "507f1f77bcf86cd799439014",
    "name": "Supplier Inc",
    "domain": "supplier.com"
  },
  "status": "pending",
  "classification": "critical",
  "invited_at": "2024-01-20T10:00:00Z",
  "invitation_sent_to": "contact@supplier.com"
}
```
- Success: 201 Created
- Errors:
  - 400 Bad Request: Invalid input
  - 409 Conflict: Supplier already invited

##### GET /api/v1/suppliers/:id
- **Purpose**: Get supplier relationship details
- **Authentication**: Required
- **Authorization**: Company org type, any role

#EXPORT_ENDPOINT: GET /api/v1/suppliers/:id - Get supplier details

**Response:**
#RESPONSE_CONTRACT:
```json
{
  "id": "507f1f77bcf86cd799439013",
  "supplier": {
    "id": "507f1f77bcf86cd799439014",
    "name": "Supplier Inc",
    "domain": "supplier.com",
    "slug": "supplier-inc"
  },
  "status": "active",
  "classification": "critical",
  "invited_at": "2024-01-10T10:00:00Z",
  "accepted_at": "2024-01-12T14:00:00Z",
  "requirements": [
    {
      "id": "507f1f77bcf86cd799439015",
      "type": "questionnaire",
      "questionnaire": {
        "id": "507f1f77bcf86cd799439016",
        "name": "Annual Security Assessment"
      },
      "status": "submitted",
      "due_date": "2024-02-15T23:59:59Z",
      "response": {
        "id": "507f1f77bcf86cd799439017",
        "status": "submitted",
        "score": {
          "percentage": 85,
          "passed": true,
          "must_pass_met": true
        },
        "submitted_at": "2024-02-10T16:30:00Z"
      }
    }
  ],
  "checkfix_status": {
    "linked": true,
    "grade": "B",
    "overall_score": 78,
    "categories": {
      "ssl": "A",
      "headers": "B",
      "reputation": "B",
      "email": "C"
    },
    "verified_at": "2024-01-15T09:00:00Z"
  }
}
```

##### PATCH /api/v1/suppliers/:id
- **Purpose**: Update supplier relationship (classification, status)
- **Authentication**: Required
- **Authorization**: Company org type, Admin role

#EXPORT_ENDPOINT: PATCH /api/v1/suppliers/:id - Update supplier relationship

**Request:**
#REQUEST_CONTRACT:
```json
{
  "classification": "important",
  "status": "suspended"
}
```

##### POST /api/v1/suppliers/:id/requirements
- **Purpose**: Create new requirement for supplier
- **Authentication**: Required
- **Authorization**: Company org type, Admin role

#EXPORT_ENDPOINT: POST /api/v1/suppliers/:id/requirements - Create requirement

**Request:**
#REQUEST_CONTRACT:
```json
{
  "type": "questionnaire",
  "questionnaire_id": "507f1f77bcf86cd799439016",
  "due_date": "2024-02-28T23:59:59Z",
  "priority": "high",
  "message": "Please complete by end of month"
}
```

For CheckFix requirement:
```json
{
  "type": "checkfix",
  "min_grade": "B",
  "due_date": "2024-02-28T23:59:59Z",
  "priority": "high"
}
```

**Response:**
#RESPONSE_CONTRACT:
```json
{
  "id": "507f1f77bcf86cd799439018",
  "type": "questionnaire",
  "questionnaire": {
    "id": "507f1f77bcf86cd799439016",
    "name": "Annual Security Assessment"
  },
  "status": "pending",
  "due_date": "2024-02-28T23:59:59Z",
  "priority": "high",
  "created_at": "2024-01-20T10:00:00Z"
}
```

---

#### RESOURCE: /api/v1/questionnaires

##### GET /api/v1/questionnaires
- **Purpose**: List company's questionnaires
- **Authentication**: Required
- **Authorization**: Company org type, any role

#EXPORT_ENDPOINT: GET /api/v1/questionnaires - List questionnaires

**Request:**
#REQUEST_CONTRACT:
- Query Parameters:
  - `status`: Filter by status (draft, published, archived)
  - `search`: Search by name
  - `page`, `limit`, `sort`, `order`: Pagination

**Response:**
#RESPONSE_CONTRACT:
```json
{
  "data": [
    {
      "id": "507f1f77bcf86cd799439016",
      "name": "Annual Security Assessment",
      "description": "Comprehensive security evaluation",
      "status": "published",
      "question_count": 25,
      "topics": ["access_control", "data_protection", "incident_response"],
      "scoring": {
        "pass_threshold": 70,
        "must_pass_count": 5
      },
      "template_source": {
        "id": "507f1f77bcf86cd799439019",
        "name": "ISO 27001 Basics"
      },
      "usage_count": 12,
      "created_at": "2024-01-05T10:00:00Z",
      "updated_at": "2024-01-18T14:00:00Z"
    }
  ],
  "pagination": {...}
}
```

##### POST /api/v1/questionnaires
- **Purpose**: Create new questionnaire
- **Authentication**: Required
- **Authorization**: Company org type, Admin role

#EXPORT_ENDPOINT: POST /api/v1/questionnaires - Create questionnaire

**Request:**
#REQUEST_CONTRACT:
```json
{
  "name": "Vendor Security Assessment",
  "description": "Security assessment for new vendors",
  "template_id": "507f1f77bcf86cd799439019",
  "scoring": {
    "pass_threshold": 75,
    "must_pass_count": 3
  }
}
```

Or without template:
```json
{
  "name": "Custom Assessment",
  "description": "Custom security questions",
  "scoring": {
    "pass_threshold": 80,
    "must_pass_count": 2
  },
  "questions": [
    {
      "text": "Do you have a documented security policy?",
      "type": "single_choice",
      "topic": "governance",
      "weight": 2,
      "is_must_pass": true,
      "options": [
        {"text": "Yes, reviewed annually", "points": 10, "is_correct": true},
        {"text": "Yes, but outdated", "points": 5, "is_correct": false},
        {"text": "No", "points": 0, "is_correct": false}
      ]
    }
  ]
}
```

##### GET /api/v1/questionnaires/:id
- **Purpose**: Get questionnaire with all questions
- **Authentication**: Required
- **Authorization**: Company org type, any role

#EXPORT_ENDPOINT: GET /api/v1/questionnaires/:id - Get questionnaire details

**Response:**
#RESPONSE_CONTRACT:
```json
{
  "id": "507f1f77bcf86cd799439016",
  "name": "Annual Security Assessment",
  "description": "Comprehensive security evaluation",
  "status": "published",
  "scoring": {
    "pass_threshold": 70,
    "must_pass_count": 5,
    "max_possible_score": 250
  },
  "questions": [
    {
      "id": "507f1f77bcf86cd799439020",
      "text": "Do you have a documented security policy?",
      "type": "single_choice",
      "topic": "governance",
      "weight": 2,
      "max_points": 10,
      "is_must_pass": true,
      "order": 1,
      "options": [
        {"id": "opt1", "text": "Yes, reviewed annually", "points": 10, "is_correct": true},
        {"id": "opt2", "text": "Yes, but outdated", "points": 5, "is_correct": false},
        {"id": "opt3", "text": "No", "points": 0, "is_correct": false}
      ]
    }
  ],
  "topics": ["governance", "access_control", "data_protection"],
  "created_at": "2024-01-05T10:00:00Z",
  "updated_at": "2024-01-18T14:00:00Z"
}
```

##### PATCH /api/v1/questionnaires/:id
- **Purpose**: Update questionnaire
- **Authentication**: Required
- **Authorization**: Company org type, Admin role

#EXPORT_ENDPOINT: PATCH /api/v1/questionnaires/:id - Update questionnaire

**Request:**
#REQUEST_CONTRACT:
```json
{
  "name": "Updated Assessment Name",
  "scoring": {
    "pass_threshold": 80
  }
}
```

##### DELETE /api/v1/questionnaires/:id
- **Purpose**: Delete questionnaire (only if draft)
- **Authentication**: Required
- **Authorization**: Company org type, Admin role

#EXPORT_ENDPOINT: DELETE /api/v1/questionnaires/:id - Delete questionnaire

- Success: 204 No Content
- Errors:
  - 400 Bad Request: Cannot delete published questionnaire with responses

##### POST /api/v1/questionnaires/:id/publish
- **Purpose**: Publish questionnaire for use
- **Authentication**: Required
- **Authorization**: Company org type, Admin role

#EXPORT_ENDPOINT: POST /api/v1/questionnaires/:id/publish - Publish questionnaire

##### POST /api/v1/questionnaires/:id/archive
- **Purpose**: Archive questionnaire
- **Authentication**: Required
- **Authorization**: Company org type, Admin role

#EXPORT_ENDPOINT: POST /api/v1/questionnaires/:id/archive - Archive questionnaire

---

#### RESOURCE: /api/v1/questionnaires/:id/questions

##### POST /api/v1/questionnaires/:id/questions
- **Purpose**: Add question to questionnaire
- **Authentication**: Required
- **Authorization**: Company org type, Admin role

#EXPORT_ENDPOINT: POST /api/v1/questionnaires/:id/questions - Add question

**Request:**
#REQUEST_CONTRACT:
```json
{
  "text": "How do you handle access revocation?",
  "type": "multiple_choice",
  "topic": "access_control",
  "weight": 1,
  "is_must_pass": false,
  "options": [
    {"text": "Automated within 24 hours", "points": 10},
    {"text": "Manual within 48 hours", "points": 7},
    {"text": "Manual within 1 week", "points": 3},
    {"text": "No formal process", "points": 0}
  ]
}
```

##### PATCH /api/v1/questionnaires/:qid/questions/:id
- **Purpose**: Update question
- **Authentication**: Required
- **Authorization**: Company org type, Admin role

#EXPORT_ENDPOINT: PATCH /api/v1/questionnaires/:qid/questions/:id - Update question

##### DELETE /api/v1/questionnaires/:qid/questions/:id
- **Purpose**: Delete question
- **Authentication**: Required
- **Authorization**: Company org type, Admin role

#EXPORT_ENDPOINT: DELETE /api/v1/questionnaires/:qid/questions/:id - Delete question

##### POST /api/v1/questionnaires/:id/questions/reorder
- **Purpose**: Reorder questions
- **Authentication**: Required
- **Authorization**: Company org type, Admin role

#EXPORT_ENDPOINT: POST /api/v1/questionnaires/:id/questions/reorder - Reorder questions

**Request:**
#REQUEST_CONTRACT:
```json
{
  "question_ids": ["id3", "id1", "id4", "id2"]
}
```

---

#### RESOURCE: /api/v1/questionnaire-templates

##### GET /api/v1/questionnaire-templates
- **Purpose**: List available templates
- **Authentication**: Required
- **Authorization**: Company org type, any role

#EXPORT_ENDPOINT: GET /api/v1/questionnaire-templates - List templates

**Request:**
#REQUEST_CONTRACT:
- Query Parameters:
  - `category`: Filter by category (iso27001, gdpr, nis2, hipaa, custom)
  - `search`: Search by name

**Response:**
#RESPONSE_CONTRACT:
```json
{
  "data": [
    {
      "id": "507f1f77bcf86cd799439019",
      "name": "ISO 27001 Basics",
      "description": "Core ISO 27001 security controls assessment",
      "category": "iso27001",
      "question_count": 30,
      "topics": ["governance", "access_control", "cryptography"],
      "difficulty": "intermediate",
      "estimated_time_minutes": 45,
      "is_system": true,
      "usage_count": 1250
    }
  ]
}
```

##### GET /api/v1/questionnaire-templates/:id
- **Purpose**: Get template with preview of questions
- **Authentication**: Required
- **Authorization**: Company org type, any role

#EXPORT_ENDPOINT: GET /api/v1/questionnaire-templates/:id - Get template

##### POST /api/v1/questionnaire-templates/:id/clone
- **Purpose**: Clone template to create company questionnaire
- **Authentication**: Required
- **Authorization**: Company org type, Admin role

#EXPORT_ENDPOINT: POST /api/v1/questionnaire-templates/:id/clone - Clone template

**Response:**
#RESPONSE_CONTRACT:
```json
{
  "id": "507f1f77bcf86cd799439025",
  "name": "ISO 27001 Basics (Copy)",
  "status": "draft",
  "template_source": {
    "id": "507f1f77bcf86cd799439019",
    "name": "ISO 27001 Basics"
  }
}
```

---

#### RESOURCE: /api/v1/responses (Company view)

##### GET /api/v1/responses
- **Purpose**: List all responses for company's requirements
- **Authentication**: Required
- **Authorization**: Company org type, any role

#EXPORT_ENDPOINT: GET /api/v1/responses - List responses (company view)

**Request:**
#REQUEST_CONTRACT:
- Query Parameters:
  - `status`: Filter by status (pending, in_progress, submitted, approved, rejected)
  - `supplier_id`: Filter by supplier
  - `questionnaire_id`: Filter by questionnaire
  - `needs_review`: Boolean, filter submitted responses needing review

**Response:**
#RESPONSE_CONTRACT:
```json
{
  "data": [
    {
      "id": "507f1f77bcf86cd799439017",
      "requirement": {
        "id": "507f1f77bcf86cd799439015",
        "due_date": "2024-02-15T23:59:59Z"
      },
      "supplier": {
        "id": "507f1f77bcf86cd799439014",
        "name": "Supplier Inc"
      },
      "questionnaire": {
        "id": "507f1f77bcf86cd799439016",
        "name": "Annual Security Assessment"
      },
      "status": "submitted",
      "score": {
        "percentage": 85,
        "passed": true,
        "must_pass_met": true,
        "breakdown": {
          "governance": 90,
          "access_control": 80,
          "data_protection": 85
        }
      },
      "submitted_at": "2024-02-10T16:30:00Z"
    }
  ],
  "pagination": {...}
}
```

##### GET /api/v1/responses/:id
- **Purpose**: Get response details with answers
- **Authentication**: Required
- **Authorization**: Company org type (for own requirements), any role

#EXPORT_ENDPOINT: GET /api/v1/responses/:id - Get response details

**Response:**
#RESPONSE_CONTRACT:
```json
{
  "id": "507f1f77bcf86cd799439017",
  "requirement": {...},
  "supplier": {...},
  "questionnaire": {...},
  "status": "submitted",
  "score": {
    "percentage": 85,
    "passed": true,
    "must_pass_met": true,
    "points_earned": 212,
    "max_points": 250,
    "breakdown": {
      "governance": {"percentage": 90, "earned": 45, "max": 50},
      "access_control": {"percentage": 80, "earned": 80, "max": 100},
      "data_protection": {"percentage": 87, "earned": 87, "max": 100}
    },
    "must_pass_results": [
      {"question_id": "q1", "passed": true},
      {"question_id": "q5", "passed": true}
    ]
  },
  "answers": [
    {
      "question_id": "507f1f77bcf86cd799439020",
      "question_text": "Do you have a documented security policy?",
      "selected_options": ["opt1"],
      "points_earned": 10,
      "max_points": 10,
      "is_must_pass": true,
      "passed": true
    }
  ],
  "submitted_at": "2024-02-10T16:30:00Z",
  "history": [
    {"status": "pending", "timestamp": "2024-01-20T10:00:00Z"},
    {"status": "in_progress", "timestamp": "2024-02-01T09:00:00Z"},
    {"status": "submitted", "timestamp": "2024-02-10T16:30:00Z"}
  ]
}
```

##### POST /api/v1/responses/:id/approve
- **Purpose**: Approve submitted response
- **Authentication**: Required
- **Authorization**: Company org type, Admin role

#EXPORT_ENDPOINT: POST /api/v1/responses/:id/approve - Approve response

**Request:**
#REQUEST_CONTRACT:
```json
{
  "feedback": "Great job on the security documentation!"
}
```

##### POST /api/v1/responses/:id/reject
- **Purpose**: Reject response with feedback
- **Authentication**: Required
- **Authorization**: Company org type, Admin role

#EXPORT_ENDPOINT: POST /api/v1/responses/:id/reject - Reject response

**Request:**
#REQUEST_CONTRACT:
```json
{
  "feedback": "Missing evidence for data encryption claims. Please provide documentation."
}
```

##### POST /api/v1/responses/:id/request-revision
- **Purpose**: Request revision of response
- **Authentication**: Required
- **Authorization**: Company org type, Admin role

#EXPORT_ENDPOINT: POST /api/v1/responses/:id/request-revision - Request revision

**Request:**
#REQUEST_CONTRACT:
```json
{
  "feedback": "Please clarify answers to questions 5 and 12."
}
```

---

### Supplier Portal Endpoints (OrgType: Supplier)

#### RESOURCE: /api/v1/companies

##### GET /api/v1/companies
- **Purpose**: List companies with relationships to supplier
- **Authentication**: Required
- **Authorization**: Supplier org type, any role

#EXPORT_ENDPOINT: GET /api/v1/companies - List company relationships

**Request:**
#REQUEST_CONTRACT:
- Query Parameters:
  - `status`: Filter by relationship status

**Response:**
#RESPONSE_CONTRACT:
```json
{
  "data": [
    {
      "id": "507f1f77bcf86cd799439013",
      "company": {
        "id": "507f1f77bcf86cd799439012",
        "name": "Acme Corp",
        "domain": "acme.com"
      },
      "status": "active",
      "classification": "critical",
      "invited_at": "2024-01-10T10:00:00Z",
      "accepted_at": "2024-01-12T14:00:00Z",
      "pending_requirements": 2,
      "total_requirements": 5
    }
  ]
}
```

##### POST /api/v1/companies/:id/accept
- **Purpose**: Accept company invitation
- **Authentication**: Required
- **Authorization**: Supplier org type, Admin role

#EXPORT_ENDPOINT: POST /api/v1/companies/:id/accept - Accept invitation

##### POST /api/v1/companies/:id/decline
- **Purpose**: Decline company invitation
- **Authentication**: Required
- **Authorization**: Supplier org type, Admin role

#EXPORT_ENDPOINT: POST /api/v1/companies/:id/decline - Decline invitation

---

#### RESOURCE: /api/v1/requirements (Supplier view)

##### GET /api/v1/requirements
- **Purpose**: List requirements assigned to supplier
- **Authentication**: Required
- **Authorization**: Supplier org type, any role

#EXPORT_ENDPOINT: GET /api/v1/requirements - List requirements (supplier view)

**Request:**
#REQUEST_CONTRACT:
- Query Parameters:
  - `status`: Filter by status
  - `company_id`: Filter by company
  - `type`: Filter by type (questionnaire, checkfix)
  - `overdue`: Boolean, show only overdue

**Response:**
#RESPONSE_CONTRACT:
```json
{
  "data": [
    {
      "id": "507f1f77bcf86cd799439015",
      "type": "questionnaire",
      "company": {
        "id": "507f1f77bcf86cd799439012",
        "name": "Acme Corp"
      },
      "questionnaire": {
        "id": "507f1f77bcf86cd799439016",
        "name": "Annual Security Assessment",
        "question_count": 25,
        "estimated_time_minutes": 45
      },
      "status": "pending",
      "due_date": "2024-02-28T23:59:59Z",
      "priority": "high",
      "response": null,
      "created_at": "2024-01-20T10:00:00Z"
    }
  ]
}
```

##### GET /api/v1/requirements/:id
- **Purpose**: Get requirement details
- **Authentication**: Required
- **Authorization**: Supplier org type (assigned to them), any role

#EXPORT_ENDPOINT: GET /api/v1/requirements/:id - Get requirement details (supplier view)

##### POST /api/v1/requirements/:id/responses
- **Purpose**: Start response to requirement
- **Authentication**: Required
- **Authorization**: Supplier org type (assigned), Admin role

#EXPORT_ENDPOINT: POST /api/v1/requirements/:id/responses - Start response

**Response:**
#RESPONSE_CONTRACT:
```json
{
  "id": "507f1f77bcf86cd799439017",
  "requirement_id": "507f1f77bcf86cd799439015",
  "status": "in_progress",
  "questionnaire": {
    "id": "507f1f77bcf86cd799439016",
    "name": "Annual Security Assessment",
    "questions": [...]
  },
  "started_at": "2024-02-01T09:00:00Z"
}
```

---

#### RESOURCE: /api/v1/responses (Supplier actions)

##### PATCH /api/v1/responses/:id
- **Purpose**: Save draft answers
- **Authentication**: Required
- **Authorization**: Supplier org type (own response), Admin role

#EXPORT_ENDPOINT: PATCH /api/v1/responses/:id - Save draft

**Request:**
#REQUEST_CONTRACT:
```json
{
  "answers": [
    {
      "question_id": "507f1f77bcf86cd799439020",
      "selected_options": ["opt1"]
    },
    {
      "question_id": "507f1f77bcf86cd799439021",
      "selected_options": ["opt2", "opt3"]
    }
  ]
}
```

##### POST /api/v1/responses/:id/submit
- **Purpose**: Submit completed response
- **Authentication**: Required
- **Authorization**: Supplier org type (own response), Admin role

#EXPORT_ENDPOINT: POST /api/v1/responses/:id/submit - Submit response

**Response:**
#RESPONSE_CONTRACT:
```json
{
  "id": "507f1f77bcf86cd799439017",
  "status": "submitted",
  "score": {
    "percentage": 85,
    "passed": true,
    "must_pass_met": true,
    "breakdown": {...}
  },
  "submitted_at": "2024-02-10T16:30:00Z"
}
```

---

#### RESOURCE: /api/v1/checkfix

##### GET /api/v1/checkfix/status
- **Purpose**: Get CheckFix linking status
- **Authentication**: Required
- **Authorization**: Supplier org type, any role

#EXPORT_ENDPOINT: GET /api/v1/checkfix/status - Get CheckFix status

**Response:**
#RESPONSE_CONTRACT:
```json
{
  "linked": true,
  "checkfix_email": "user@supplier.com",
  "domain": "supplier.com",
  "verification": {
    "grade": "B",
    "overall_score": 78,
    "categories": {
      "ssl": {"grade": "A", "score": 95},
      "headers": {"grade": "B", "score": 75},
      "reputation": {"grade": "B", "score": 80},
      "email": {"grade": "C", "score": 65}
    },
    "findings": {
      "critical": 0,
      "high": 2,
      "medium": 5,
      "low": 8
    },
    "report_date": "2024-01-15T00:00:00Z",
    "verified_at": "2024-01-15T09:00:00Z",
    "expires_at": "2024-02-15T09:00:00Z"
  }
}
```

##### POST /api/v1/checkfix/link
- **Purpose**: Link CheckFix account
- **Authentication**: Required
- **Authorization**: Supplier org type, Admin role

#EXPORT_ENDPOINT: POST /api/v1/checkfix/link - Link CheckFix account

**Request:**
#REQUEST_CONTRACT:
```json
{
  "checkfix_email": "user@supplier.com",
  "report_hash": "abc123def456..."
}
```

#API_ASSUMPTION: CheckFix provides a way to get a shareable report hash
#INTEGRATION_ASSUMPTION: Report hash can be used to verify report ownership via CheckFix API

##### POST /api/v1/checkfix/verify
- **Purpose**: Verify/refresh CheckFix report
- **Authentication**: Required
- **Authorization**: Supplier org type, Admin role

#EXPORT_ENDPOINT: POST /api/v1/checkfix/verify - Verify CheckFix report

**Request:**
#REQUEST_CONTRACT:
```json
{
  "report_hash": "abc123def456..."
}
```

##### DELETE /api/v1/checkfix/link
- **Purpose**: Unlink CheckFix account
- **Authentication**: Required
- **Authorization**: Supplier org type, Admin role

#EXPORT_ENDPOINT: DELETE /api/v1/checkfix/link - Unlink CheckFix

---

### Shared Endpoints

#### RESOURCE: /api/v1/users

##### GET /api/v1/users
- **Purpose**: List users in organization
- **Authentication**: Required
- **Authorization**: Admin role

#EXPORT_ENDPOINT: GET /api/v1/users - List organization users

##### POST /api/v1/users
- **Purpose**: Invite user to organization
- **Authentication**: Required
- **Authorization**: Admin role

#EXPORT_ENDPOINT: POST /api/v1/users - Invite user

**Request:**
#REQUEST_CONTRACT:
```json
{
  "email": "newuser@company.com",
  "role": "viewer"
}
```

##### PATCH /api/v1/users/:id
- **Purpose**: Update user role
- **Authentication**: Required
- **Authorization**: Admin role

#EXPORT_ENDPOINT: PATCH /api/v1/users/:id - Update user

##### DELETE /api/v1/users/:id
- **Purpose**: Remove user from organization
- **Authentication**: Required
- **Authorization**: Admin role

#EXPORT_ENDPOINT: DELETE /api/v1/users/:id - Remove user

---

### Health & Utility Endpoints

##### GET /health
- **Purpose**: Health check endpoint
- **Authentication**: None

#EXPORT_ENDPOINT: GET /health - Health check

**Response:**
```json
{
  "status": "healthy",
  "version": "1.0.0",
  "database": "connected"
}
```

##### GET /api/v1/config
- **Purpose**: Get public configuration
- **Authentication**: None

#EXPORT_ENDPOINT: GET /api/v1/config - Public config

**Response:**
```json
{
  "version": "1.0.0",
  "features": {
    "checkfix_integration": true
  }
}
```

---

## Authentication & Authorization Strategy

#AUTH_STRATEGY: JWT RS512 with passwordless magic link authentication

### Token Structure

#### Access Token Claims
```json
{
  "sub": "user_id",
  "org_id": "organization_id",
  "role": "admin",
  "org_type": "company",
  "exp": 1706104800,
  "iat": 1706101200
}
```

#### Refresh Token Claims
```json
{
  "sub": "user_id",
  "type": "refresh",
  "exp": 1708693200,
  "iat": 1706101200
}
```

#SECURITY_ASSUMPTION: Access tokens have 1-hour expiry, refresh tokens have 30-day expiry
#SECURITY_ASSUMPTION: Refresh tokens are single-use and rotated on each refresh

### Magic Link Flow

```
1. POST /auth/request-link { email }
   |
   v
2. Generate SecureLink (64-char random token, 15-min expiry)
   |
   v
3. Store in secure_links collection with hashed token
   |
   v
4. Send email with link: {BASE_URL}/auth/verify/{token}
   |
   v
5. GET /auth/verify/{token}
   |
   v
6. Validate: token exists, not expired, not used
   |
   v
7. Find or create user for email
   |
   v
8. Find or create organization from email domain
   |
   v
9. Generate JWT pair
   |
   v
10. Mark SecureLink as used
    |
    v
11. Return tokens + user profile
```

#PLAN_UNCERTAINTY: Assumes email domain maps to organization - may need manual org assignment
#SECURITY_ASSUMPTION: SecureLink tokens are hashed in database (bcrypt)

### Role-Based Access Control

| Role | Permissions |
|------|-------------|
| admin | Full access to organization resources, can manage users |
| viewer | Read-only access to organization resources |

#PATH_RATIONALE: Simple two-role model sufficient for initial release - can extend later

### Organization Type Access Control

| Org Type | Accessible Resources |
|----------|---------------------|
| company | Suppliers, Questionnaires, Templates, Requirements (create), Responses (review) |
| supplier | Companies, Requirements (assigned), Responses (fill), CheckFix linking |

---

## Error Handling Strategy

#ERROR_CONVENTION: Consistent error response format across all endpoints

### Error Response Format
```json
{
  "error": {
    "code": "validation_failed",
    "message": "Invalid input data",
    "details": [
      {
        "field": "email",
        "message": "Invalid email format"
      }
    ]
  },
  "request_id": "abc-123-def"
}
```

### Error Codes

| HTTP Status | Error Code | Description |
|-------------|------------|-------------|
| 400 | `validation_failed` | Request validation failed |
| 400 | `invalid_input` | Malformed request body |
| 400 | `invalid_state` | Operation not allowed in current state |
| 401 | `missing_token` | Authorization header missing |
| 401 | `invalid_token` | Token invalid or expired |
| 401 | `token_expired` | Access token has expired |
| 403 | `insufficient_permissions` | User lacks required role |
| 403 | `org_type_not_allowed` | Organization type cannot access resource |
| 403 | `resource_not_owned` | Resource belongs to another organization |
| 404 | `not_found` | Requested resource not found |
| 409 | `already_exists` | Resource already exists (duplicate) |
| 409 | `conflict` | Operation conflicts with current state |
| 422 | `business_rule_violation` | Business logic constraint violated |
| 429 | `rate_limit_exceeded` | Too many requests |
| 500 | `internal_error` | Unexpected server error |
| 503 | `service_unavailable` | External service unavailable |

#PATH_RATIONALE: Structured error codes enable frontend to handle errors programmatically

---

## Integration Points

### Frontend-Backend Interface

#EXPORT_INTERFACE: REST API with JSON payloads over HTTPS

#### Request Conventions
- Content-Type: application/json
- Authorization: Bearer {access_token}
- Accept-Language: en (or de for German)

#### Response Conventions
- Pagination: Standard format for all list endpoints
- Timestamps: ISO 8601 format (UTC)
- IDs: MongoDB ObjectID as 24-char hex string

#INTEGRATION_ASSUMPTION: Frontend handles token refresh automatically on 401 responses

### Backend-Database Interface

#EXPORT_INTERFACE: MongoDB Go Driver with repository pattern

#### Collections
| Collection | Indexes |
|------------|---------|
| organizations | slug (unique), domain (unique sparse) |
| users | email (unique), organization_id |
| secure_links | secure_identifier (unique), expires_at (TTL) |
| questionnaire_templates | category, is_system |
| questionnaires | organization_id, status |
| questions | questionnaire_id, order |
| company_supplier_relationships | company_id+supplier_id (unique sparse), status |
| requirements | company_id, supplier_id, status, due_date |
| supplier_responses | requirement_id, supplier_id, status |
| checkfix_verifications | supplier_id (unique), expires_at |
| audit_logs | organization_id, created_at, action |

#PLAN_UNCERTAINTY: Index strategy may need tuning based on query patterns

### Backend-Mail Service Interface

#EXPORT_INTERFACE: HTTP REST API to external mail service

```go
type MailRequest struct {
    To       string            `json:"to"`
    Template string            `json:"template"`
    Data     map[string]string `json:"data"`
}
```

#INTEGRATION_ASSUMPTION: Mail service supports templates: magic_link, supplier_invitation, requirement_notification, response_approved, response_rejected

### Backend-CheckFix Interface

#EXPORT_INTERFACE: HTTP REST API to CheckFix service

```go
// Verify report endpoint
// GET {CHECKFIX_API_URL}/api/v1/reports/verify/{hash}
// Authorization: X-API-Key {CHECKFIX_API_KEY}

type CheckFixVerifyResponse struct {
    Valid       bool                   `json:"valid"`
    Domain      string                 `json:"domain"`
    Grade       string                 `json:"grade"`
    Score       int                    `json:"score"`
    Categories  map[string]CategoryScore `json:"categories"`
    Findings    FindingCounts          `json:"findings"`
    ReportDate  time.Time              `json:"report_date"`
}
```

#API_ASSUMPTION: CheckFix API provides report verification by hash
#INTEGRATION_ASSUMPTION: CheckFix API rate limits: 100 requests/hour
#PLAN_UNCERTAINTY: Need to confirm exact CheckFix API specification

---

## Architectural Patterns

### Dependency Injection Pattern
- **Pattern**: Constructor injection for all services and repositories
- **Implementation**: Wire dependencies in main.go
#PATH_RATIONALE: Enables easy testing with mock implementations
#PATTERN_CONFLICT: Considered using wire for DI generation, but manual wiring sufficient for project size

### Repository Pattern
- **Pattern**: Interface-based repositories for data access
- **Implementation**: Each repository defines interface, MongoDB implementation
#PATH_RATIONALE: Decouples business logic from database implementation

### Service Layer Pattern
- **Pattern**: Services encapsulate business logic
- **Implementation**: Services depend on repositories and other services via interfaces
#PATH_RATIONALE: Separates HTTP handling from business rules

### DTO Pattern
- **Pattern**: Separate request/response DTOs from domain models
- **Implementation**: handlers/dto package with validation tags
#PATH_RATIONALE: Prevents leaking internal structure to API consumers

### Context Propagation Pattern
- **Pattern**: Pass context.Context through all layers
- **Implementation**: First parameter to all repository and service methods
#PATH_RATIONALE: Enables cancellation, timeouts, and distributed tracing

---

## Security Considerations

### Authentication Security

#THREAT_VECTOR: Brute force magic link token guessing
#CONTROL_RATIONALE: 64-char random token provides 256-bit entropy; 15-min expiry limits window

#THREAT_VECTOR: Token replay attacks
#CONTROL_RATIONALE: Magic links are single-use; refresh tokens rotated on use

#THREAT_VECTOR: JWT token theft
#CONTROL_RATIONALE: Short access token expiry (1 hour); refresh tokens are HTTP-only in production

### Authorization Security

#THREAT_VECTOR: Privilege escalation
#CONTROL_RATIONALE: Role and org type checked at middleware level before handlers

#THREAT_VECTOR: Cross-organization data access (IDOR)
#CONTROL_RATIONALE: All data queries include organization_id filter from JWT claims

#SECURITY_ASSUMPTION: Organization ID in JWT cannot be manipulated (RS512 signature)

### Data Security

#THREAT_VECTOR: Sensitive data exposure in logs
#CONTROL_RATIONALE: Structured logging with PII redaction

#THREAT_VECTOR: SQL/NoSQL injection
#CONTROL_RATIONALE: MongoDB driver handles parameter binding; never construct queries from strings

### API Security

#THREAT_VECTOR: Rate limit bypass
#CONTROL_RATIONALE: Per-IP and per-user rate limiting at middleware level

#THREAT_VECTOR: CORS misconfiguration
#CONTROL_RATIONALE: Explicit allowed origins; no wildcards in production

---

## Risk Register

### Architectural Risks

#ARCHITECTURE_CONCERN: Email domain to organization mapping may be ambiguous
#MITIGATION_STRATEGY: Implement manual organization assignment flow as fallback

#ARCHITECTURE_CONCERN: CheckFix API changes could break integration
#MITIGATION_STRATEGY: Abstract CheckFix client behind interface; comprehensive error handling

#ARCHITECTURE_CONCERN: Scoring algorithm changes after responses submitted
#MITIGATION_STRATEGY: Store score snapshot at submission time; questionnaire versioning

### Technical Debt Risks

#TECHNICAL_DEBT_RISK: Manual DI wiring becomes unwieldy as service count grows
#MITIGATION_STRATEGY: Consider wire or dig if >20 services

#TECHNICAL_DEBT_RISK: Lack of comprehensive API versioning strategy
#MITIGATION_STRATEGY: v1 prefix allows future breaking changes in v2

### Performance Risks

#SCALE_ASSUMPTION: Expected load: <1000 concurrent users, <100 requests/second
#ARCHITECTURE_CONCERN: MongoDB queries without pagination could timeout
#MITIGATION_STRATEGY: Enforce pagination on all list endpoints; max 100 items per page

---

## Exported Decisions Summary

#EXPORT_CRITICAL: JWT RS512 authentication with 1-hour access tokens and 30-day refresh tokens
#EXPORT_CRITICAL: Magic link authentication with 15-minute expiry
#EXPORT_CRITICAL: Organization types: Company and Supplier with distinct permissions
#EXPORT_CRITICAL: Roles: admin (full access) and viewer (read-only)
#EXPORT_CRITICAL: Scoring: percentage threshold + must-pass questions for pass/fail

#EXPORT_CONSTRAINT: All list endpoints must support pagination
#EXPORT_CONSTRAINT: All requests must include organization_id filter from JWT
#EXPORT_CONSTRAINT: CheckFix integration must handle API unavailability gracefully
#EXPORT_CONSTRAINT: No PII in logs; use structured logging

#EXPORT_INTERFACE: REST API with JSON, versioned at /api/v1
#EXPORT_INTERFACE: Standard pagination: page, limit, total, total_pages
#EXPORT_INTERFACE: Standard error format with code, message, details

---

## Data Models (Reference)

### Organization
```go
type Organization struct {
    ID        primitive.ObjectID `bson:"_id,omitempty"`
    Name      string             `bson:"name"`
    Type      OrgType            `bson:"type"` // company, supplier
    Slug      string             `bson:"slug"`
    Domain    string             `bson:"domain,omitempty"`
    LogoURL   string             `bson:"logo_url,omitempty"`
    Settings  OrgSettings        `bson:"settings"`
    CreatedAt time.Time          `bson:"created_at"`
    UpdatedAt time.Time          `bson:"updated_at"`
}

type OrgSettings struct {
    DefaultDueDays     int      `bson:"default_due_days"`
    RequireCheckFix    bool     `bson:"require_checkfix"`
    MinCheckFixGrade   string   `bson:"min_checkfix_grade"`
    NotificationEmails []string `bson:"notification_emails"`
}
```

### User
```go
type User struct {
    ID             primitive.ObjectID `bson:"_id,omitempty"`
    Email          string             `bson:"email"`
    OrganizationID primitive.ObjectID `bson:"organization_id"`
    Role           Role               `bson:"role"` // admin, viewer
    CreatedAt      time.Time          `bson:"created_at"`
    UpdatedAt      time.Time          `bson:"updated_at"`
    LastLoginAt    *time.Time         `bson:"last_login_at,omitempty"`
}
```

### SecureLink
```go
type SecureLink struct {
    ID               primitive.ObjectID `bson:"_id,omitempty"`
    SecureIdentifier string             `bson:"secure_identifier"` // hashed token
    Email            string             `bson:"email"`
    Purpose          string             `bson:"purpose"` // magic_link, invitation
    ExpiresAt        time.Time          `bson:"expires_at"`
    UsedAt           *time.Time         `bson:"used_at,omitempty"`
    CreatedAt        time.Time          `bson:"created_at"`
}
```

### Questionnaire
```go
type Questionnaire struct {
    ID              primitive.ObjectID  `bson:"_id,omitempty"`
    OrganizationID  primitive.ObjectID  `bson:"organization_id"`
    Name            string              `bson:"name"`
    Description     string              `bson:"description"`
    Status          QuestionnaireStatus `bson:"status"` // draft, published, archived
    TemplateID      *primitive.ObjectID `bson:"template_id,omitempty"`
    Scoring         ScoringConfig       `bson:"scoring"`
    Topics          []string            `bson:"topics"`
    CreatedAt       time.Time           `bson:"created_at"`
    UpdatedAt       time.Time           `bson:"updated_at"`
    PublishedAt     *time.Time          `bson:"published_at,omitempty"`
}

type ScoringConfig struct {
    PassThreshold int `bson:"pass_threshold"` // Minimum percentage to pass
    MustPassCount int `bson:"must_pass_count"` // Number of must-pass questions
}
```

### Question
```go
type Question struct {
    ID              primitive.ObjectID `bson:"_id,omitempty"`
    QuestionnaireID primitive.ObjectID `bson:"questionnaire_id"`
    Text            string             `bson:"text"`
    Type            QuestionType       `bson:"type"` // single_choice, multiple_choice
    Topic           string             `bson:"topic"`
    Weight          int                `bson:"weight"` // Multiplier for points
    MaxPoints       int                `bson:"max_points"`
    IsMustPass      bool               `bson:"is_must_pass"`
    Order           int                `bson:"order"`
    Options         []QuestionOption   `bson:"options"`
}

type QuestionOption struct {
    ID        string `bson:"id"`
    Text      string `bson:"text"`
    Points    int    `bson:"points"`
    IsCorrect bool   `bson:"is_correct"` // For must-pass determination
}
```

### CompanySupplierRelationship
```go
type CompanySupplierRelationship struct {
    ID             primitive.ObjectID `bson:"_id,omitempty"`
    CompanyID      primitive.ObjectID `bson:"company_id"`
    SupplierID     primitive.ObjectID `bson:"supplier_id,omitempty"` // null until accepted
    InvitedEmail   string             `bson:"invited_email"`
    Status         RelationshipStatus `bson:"status"` // pending, active, suspended, terminated, rejected
    Classification Classification     `bson:"classification"` // critical, important, standard
    InvitedAt      time.Time          `bson:"invited_at"`
    AcceptedAt     *time.Time         `bson:"accepted_at,omitempty"`
    UpdatedAt      time.Time          `bson:"updated_at"`
}
```

### Requirement
```go
type Requirement struct {
    ID              primitive.ObjectID  `bson:"_id,omitempty"`
    CompanyID       primitive.ObjectID  `bson:"company_id"`
    SupplierID      primitive.ObjectID  `bson:"supplier_id"`
    RelationshipID  primitive.ObjectID  `bson:"relationship_id"`
    Type            RequirementType     `bson:"type"` // questionnaire, checkfix
    QuestionnaireID *primitive.ObjectID `bson:"questionnaire_id,omitempty"`
    MinCheckFixGrade *string            `bson:"min_checkfix_grade,omitempty"`
    Status          RequirementStatus   `bson:"status"` // pending, in_progress, submitted, approved, rejected, expired
    Priority        Priority            `bson:"priority"` // low, medium, high
    DueDate         time.Time           `bson:"due_date"`
    Message         string              `bson:"message"`
    CreatedAt       time.Time           `bson:"created_at"`
    UpdatedAt       time.Time           `bson:"updated_at"`
}
```

### SupplierResponse
```go
type SupplierResponse struct {
    ID            primitive.ObjectID `bson:"_id,omitempty"`
    RequirementID primitive.ObjectID `bson:"requirement_id"`
    SupplierID    primitive.ObjectID `bson:"supplier_id"`
    Status        ResponseStatus     `bson:"status"` // in_progress, submitted, approved, rejected, revision_requested
    Answers       []Answer           `bson:"answers"`
    Score         *ScoreResult       `bson:"score,omitempty"` // Calculated on submission
    Feedback      string             `bson:"feedback,omitempty"`
    StartedAt     time.Time          `bson:"started_at"`
    SubmittedAt   *time.Time         `bson:"submitted_at,omitempty"`
    ReviewedAt    *time.Time         `bson:"reviewed_at,omitempty"`
    ReviewedBy    *primitive.ObjectID `bson:"reviewed_by,omitempty"`
    UpdatedAt     time.Time          `bson:"updated_at"`
}

type Answer struct {
    QuestionID      primitive.ObjectID `bson:"question_id"`
    SelectedOptions []string           `bson:"selected_options"`
}

type ScoreResult struct {
    Percentage    float64                   `bson:"percentage"`
    Passed        bool                      `bson:"passed"`
    MustPassMet   bool                      `bson:"must_pass_met"`
    PointsEarned  int                       `bson:"points_earned"`
    MaxPoints     int                       `bson:"max_points"`
    TopicBreakdown map[string]TopicScore    `bson:"topic_breakdown"`
    MustPassResults []MustPassResult        `bson:"must_pass_results"`
}
```

### CheckFixVerification
```go
type CheckFixVerification struct {
    ID           primitive.ObjectID `bson:"_id,omitempty"`
    SupplierID   primitive.ObjectID `bson:"supplier_id"`
    Domain       string             `bson:"domain"`
    CheckFixEmail string            `bson:"checkfix_email"`
    ReportHash   string             `bson:"report_hash"`
    Grade        string             `bson:"grade"`
    OverallScore int                `bson:"overall_score"`
    Categories   map[string]CategoryScore `bson:"categories"`
    Findings     FindingCounts      `bson:"findings"`
    ReportDate   time.Time          `bson:"report_date"`
    VerifiedAt   time.Time          `bson:"verified_at"`
    ExpiresAt    time.Time          `bson:"expires_at"`
}
```

---

## Appendix: Router Setup Example

```go
func SetupRouter(
    authHandler *handlers.AuthHandler,
    orgHandler *handlers.OrganizationHandler,
    questionnaireHandler *handlers.QuestionnaireHandler,
    supplierHandler *handlers.SupplierHandler,
    requirementHandler *handlers.RequirementHandler,
    responseHandler *handlers.ResponseHandler,
    checkfixHandler *handlers.CheckFixHandler,
    userHandler *handlers.UserHandler,
    jwtService auth.JWTService,
) *gin.Engine {
    r := gin.Default()

    // Global middleware
    r.Use(middleware.CORS())
    r.Use(middleware.RequestLogger())
    r.Use(middleware.RateLimiter())

    // Health check
    r.GET("/health", handlers.HealthCheck)

    // API v1
    v1 := r.Group("/api/v1")
    {
        // Public config
        v1.GET("/config", handlers.GetConfig)

        // Auth (public)
        auth := v1.Group("/auth")
        {
            auth.POST("/request-link", authHandler.RequestLink)
            auth.GET("/verify/:token", authHandler.Verify)
            auth.POST("/refresh", authHandler.Refresh)
        }

        // Authenticated routes
        authenticated := v1.Group("")
        authenticated.Use(middleware.AuthMiddleware(jwtService))
        {
            // Profile
            authenticated.GET("/auth/profile", authHandler.GetProfile)

            // Organization
            authenticated.GET("/organizations/current", orgHandler.GetCurrent)
            authenticated.PATCH("/organizations/current", middleware.RequireRole(models.RoleAdmin), orgHandler.UpdateCurrent)

            // Users
            users := authenticated.Group("/users")
            users.Use(middleware.RequireRole(models.RoleAdmin))
            {
                users.GET("", userHandler.List)
                users.POST("", userHandler.Create)
                users.PATCH("/:id", userHandler.Update)
                users.DELETE("/:id", userHandler.Delete)
            }

            // Company Portal
            company := authenticated.Group("")
            company.Use(middleware.RequireOrgType(models.OrgTypeCompany))
            {
                // Suppliers
                suppliers := company.Group("/suppliers")
                {
                    suppliers.GET("", supplierHandler.List)
                    suppliers.POST("", middleware.RequireRole(models.RoleAdmin), supplierHandler.Create)
                    suppliers.GET("/:id", supplierHandler.Get)
                    suppliers.PATCH("/:id", middleware.RequireRole(models.RoleAdmin), supplierHandler.Update)
                    suppliers.POST("/:id/requirements", middleware.RequireRole(models.RoleAdmin), requirementHandler.Create)
                }

                // Questionnaires
                questionnaires := company.Group("/questionnaires")
                {
                    questionnaires.GET("", questionnaireHandler.List)
                    questionnaires.POST("", middleware.RequireRole(models.RoleAdmin), questionnaireHandler.Create)
                    questionnaires.GET("/:id", questionnaireHandler.Get)
                    questionnaires.PATCH("/:id", middleware.RequireRole(models.RoleAdmin), questionnaireHandler.Update)
                    questionnaires.DELETE("/:id", middleware.RequireRole(models.RoleAdmin), questionnaireHandler.Delete)
                    questionnaires.POST("/:id/publish", middleware.RequireRole(models.RoleAdmin), questionnaireHandler.Publish)
                    questionnaires.POST("/:id/archive", middleware.RequireRole(models.RoleAdmin), questionnaireHandler.Archive)

                    // Questions
                    questionnaires.POST("/:id/questions", middleware.RequireRole(models.RoleAdmin), questionnaireHandler.AddQuestion)
                    questionnaires.PATCH("/:qid/questions/:id", middleware.RequireRole(models.RoleAdmin), questionnaireHandler.UpdateQuestion)
                    questionnaires.DELETE("/:qid/questions/:id", middleware.RequireRole(models.RoleAdmin), questionnaireHandler.DeleteQuestion)
                    questionnaires.POST("/:id/questions/reorder", middleware.RequireRole(models.RoleAdmin), questionnaireHandler.ReorderQuestions)
                }

                // Templates
                templates := company.Group("/questionnaire-templates")
                {
                    templates.GET("", questionnaireHandler.ListTemplates)
                    templates.GET("/:id", questionnaireHandler.GetTemplate)
                    templates.POST("/:id/clone", middleware.RequireRole(models.RoleAdmin), questionnaireHandler.CloneTemplate)
                }

                // Responses (company view)
                responses := company.Group("/responses")
                {
                    responses.GET("", responseHandler.ListForCompany)
                    responses.GET("/:id", responseHandler.GetForCompany)
                    responses.POST("/:id/approve", middleware.RequireRole(models.RoleAdmin), responseHandler.Approve)
                    responses.POST("/:id/reject", middleware.RequireRole(models.RoleAdmin), responseHandler.Reject)
                    responses.POST("/:id/request-revision", middleware.RequireRole(models.RoleAdmin), responseHandler.RequestRevision)
                }
            }

            // Supplier Portal
            supplier := authenticated.Group("")
            supplier.Use(middleware.RequireOrgType(models.OrgTypeSupplier))
            {
                // Companies
                companies := supplier.Group("/companies")
                {
                    companies.GET("", supplierHandler.ListCompanies)
                    companies.POST("/:id/accept", middleware.RequireRole(models.RoleAdmin), supplierHandler.AcceptInvitation)
                    companies.POST("/:id/decline", middleware.RequireRole(models.RoleAdmin), supplierHandler.DeclineInvitation)
                }

                // Requirements
                requirements := supplier.Group("/requirements")
                {
                    requirements.GET("", requirementHandler.ListForSupplier)
                    requirements.GET("/:id", requirementHandler.GetForSupplier)
                    requirements.POST("/:id/responses", middleware.RequireRole(models.RoleAdmin), responseHandler.Start)
                }

                // Responses (supplier actions)
                responses := supplier.Group("/responses")
                {
                    responses.PATCH("/:id", middleware.RequireRole(models.RoleAdmin), responseHandler.SaveDraft)
                    responses.POST("/:id/submit", middleware.RequireRole(models.RoleAdmin), responseHandler.Submit)
                }

                // CheckFix
                checkfix := supplier.Group("/checkfix")
                {
                    checkfix.GET("/status", checkfixHandler.GetStatus)
                    checkfix.POST("/link", middleware.RequireRole(models.RoleAdmin), checkfixHandler.Link)
                    checkfix.POST("/verify", middleware.RequireRole(models.RoleAdmin), checkfixHandler.Verify)
                    checkfix.DELETE("/link", middleware.RequireRole(models.RoleAdmin), checkfixHandler.Unlink)
                }
            }
        }
    }

    return r
}
```

---

## Quality Checklist

- [x] Every major decision has `#PATH_RATIONALE`
- [x] Every assumption has `#PLAN_UNCERTAINTY` or `#API_ASSUMPTION`
- [x] Critical decisions marked with `#EXPORT_CRITICAL`
- [x] All interfaces tagged with `#EXPORT_INTERFACE`
- [x] All endpoints tagged with `#EXPORT_ENDPOINT`
- [x] Security risks identified and tagged
- [x] No implementation details included (only specifications)
- [x] Plan saved to docs/system-architecture-plan.md
