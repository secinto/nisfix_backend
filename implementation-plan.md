# Supplier Security Portal - Implementation Plan

## Overview

A B2B portal where **Companies** require their **Suppliers** to meet security standards through:
1. **Questionnaires** - Custom or template-based assessments with scoring
2. **CheckFix Reports** - Integration with checkfix.io for domain security grades (A-F)

**Key Decisions:**
- Separate standalone applications (not extensions of existing checkfix_backend/checkfix_frontend)
- Independent organization types: Company and Supplier (no relation to existing types)
- Scoring: Both percentage thresholds AND required questions for pass/fail
- CheckFix integration uses existing A-F grades

---

## Project Locations

- **Backend**: `/Users/skraxberger/development/checkfix-tools/nisfix_backend`
- **Frontend**: `/Users/skraxberger/development/checkfix-tools/nisfix_frontend`

## Implementation Approach

- **Backend-first**: Complete backend API with Swagger docs, then build frontend
- **Email Service**: Use existing mail service (same as checkfix_backend via MAIL_BASE_URL)

---

## 1. Backend Architecture (Go/Gin/MongoDB)

### 1.1 Project Structure

```
nisfix_backend/
├── cmd/server/main.go              # Entry point
├── internal/
│   ├── auth/                       # JWT RS512 service
│   ├── config/                     # Environment config
│   ├── database/                   # MongoDB connection
│   ├── handlers/                   # HTTP handlers (Gin)
│   │   ├── auth_handler.go
│   │   ├── organization_handler.go
│   │   ├── questionnaire_handler.go
│   │   ├── requirement_handler.go
│   │   ├── response_handler.go
│   │   ├── relationship_handler.go
│   │   └── checkfix_handler.go
│   ├── middleware/                 # Auth, role, org type checks
│   ├── models/                     # Domain models + DTOs
│   ├── repository/                 # MongoDB operations
│   ├── services/                   # Business logic
│   └── validators/                 # Email validation
├── keys/                           # JWT RSA keys
├── docs/                           # Swagger docs
└── go.mod
```

### 1.2 Core Data Models

| Model | Purpose |
|-------|---------|
| `Organization` | Company or Supplier with domain, settings, CheckFix link |
| `User` | Email, role (admin/viewer), org membership |
| `QuestionnaireTemplate` | Pre-defined templates (ISO27001, GDPR, NIS2) |
| `Questionnaire` | Company-customized instance from template |
| `Question` | Single/multiple choice with scoring, required flag |
| `QuestionOption` | Answer options with points |
| `CompanySupplierRelationship` | Invitation, status, classification |
| `Requirement` | What company requires (questionnaire or CheckFix) |
| `SupplierResponse` | Supplier's response to requirement |
| `SecureLink` | Magic link tokens for passwordless auth |

### 1.3 Key API Endpoints

**Authentication (Passwordless)**
```
POST /api/v1/auth/request-link     # Request magic link
GET  /api/v1/auth/verify/:token    # Verify & get JWT
POST /api/v1/auth/refresh          # Refresh tokens
GET  /api/v1/auth/profile          # Get user profile
```

**Company Portal**
```
GET/POST   /api/v1/suppliers                    # List/invite suppliers
GET/PATCH  /api/v1/suppliers/:id                # Supplier details
POST       /api/v1/suppliers/:id/requirements   # Create requirement
GET/POST   /api/v1/questionnaires               # List/create questionnaires
GET        /api/v1/questionnaire-templates      # Browse templates
POST       /api/v1/responses/:id/approve        # Approve response
POST       /api/v1/responses/:id/reject         # Reject response
```

**Supplier Portal**
```
GET  /api/v1/companies                          # List company relationships
POST /api/v1/companies/:id/accept               # Accept invitation
GET  /api/v1/requirements                       # My requirements
POST /api/v1/requirements/:id/responses         # Start response
POST /api/v1/responses/:id/submit               # Submit response
POST /api/v1/checkfix/link-account              # Link CheckFix
```

### 1.4 Scoring Algorithm

```go
// Calculate questionnaire score
func CalculateScore(answers []Answer, questions []Question) ScoreResult {
    totalPoints := 0
    maxPoints := 0
    mustPassFailed := false

    for _, q := range questions {
        answer := findAnswer(answers, q.ID)
        earned := calculatePointsForAnswer(answer, q)
        totalPoints += earned * q.Weight
        maxPoints += q.MaxPoints * q.Weight

        if q.IsMustPass && !isCorrectAnswer(answer, q) {
            mustPassFailed = true
        }
    }

    percentage := (totalPoints / maxPoints) * 100
    passed := percentage >= minScore && !mustPassFailed

    return ScoreResult{Percentage: percentage, Passed: passed}
}
```

### 1.5 CheckFix Integration

- Verify reports via CheckFix API using report hash
- Store: grades (A-F), scores (0-100), finding counts
- Validate: domain match, report age, minimum grade
- Cache verified reports with expiration

---

## 2. Frontend Architecture (React/Vite/Tailwind)

### 2.1 Project Structure

```
nisfix_frontend/
├── src/
│   ├── components/
│   │   ├── auth/                   # RouteGuard, MagicLinkForm
│   │   ├── layout/                 # AppLayout, Sidebars, Header
│   │   ├── common/                 # DataTable, StatusBadge, ScoreGauge
│   │   ├── questionnaire/
│   │   │   ├── builder/            # QuestionnaireBuilder, QuestionEditor
│   │   │   └── viewer/             # QuestionnaireForm, ScoreDisplay
│   │   ├── supplier/               # SupplierCard, InviteDialog
│   │   ├── company/                # RequestCard, CheckFixLinkForm
│   │   └── ui/                     # Shadcn/ui components
│   ├── hooks/                      # useAuth, useSuppliers, useQuestionnaires
│   ├── pages/
│   │   ├── company/                # CompanyDashboard, SupplierList, QuestionnaireEdit
│   │   └── supplier/               # SupplierDashboard, QuestionnaireFill
│   ├── services/apiClient.ts       # API client with token refresh
│   ├── types/                      # TypeScript interfaces
│   └── context/LanguageContext.tsx # i18n (en/de)
├── tailwind.config.ts
└── vite.config.ts
```

### 2.2 Routing

```typescript
// Company Portal
/company                    # Dashboard
/company/suppliers          # Supplier list
/company/suppliers/:id      # Supplier detail
/company/questionnaires     # Questionnaire list
/company/questionnaires/new # Create questionnaire
/company/questionnaires/:id # Edit questionnaire
/company/templates          # Browse templates

// Supplier Portal
/supplier                   # Dashboard
/supplier/requests          # Company requests
/supplier/requests/:id      # Request detail
/supplier/questionnaire/:id # Fill questionnaire
/supplier/checkfix          # CheckFix status/linking

// Shared
/                           # Landing/login
/auth/verify/:token         # Magic link verification
/profile                    # User profile
/settings                   # Organization settings
```

### 2.3 Key Components

| Component | Purpose |
|-----------|---------|
| `QuestionnaireBuilder` | Drag-drop question editor with scoring config |
| `QuestionnaireForm` | Fillable form with validation |
| `ScoreGauge` | Circular gauge for score visualization |
| `GradeDisplay` | A-F grade with color coding |
| `SupplierInviteDialog` | Email invite with classification |
| `RequirementsChecklist` | Show requirements with status |
| `DataTable` | Sortable, filterable, paginated table |

### 2.4 Tech Stack

- React 19 + TypeScript (strict)
- Vite 5+ for build
- Tailwind CSS + Shadcn/ui
- React Query v5 for data fetching
- React Router v7 for routing
- React Hook Form + Zod for forms
- @dnd-kit for drag-drop in builder
- Sonner for toasts
- Lucide React for icons

---

## 3. Database Schema (MongoDB)

### 3.1 Collections

```
organizations              # Company and Supplier orgs
users                      # User accounts
questionnaire_templates    # System and custom templates
questionnaires            # Company questionnaire instances
questions                 # Question definitions
question_options          # Answer options with scoring
company_supplier_relationships  # Invitations and relationships
requirements              # What companies require
supplier_responses        # Responses to requirements
questionnaire_submissions # Questionnaire answers
checkfix_verifications    # CheckFix report verifications
secure_links              # Magic link tokens
audit_logs                # Activity audit trail
```

### 3.2 Key Indexes

```javascript
// Organizations
{ slug: 1 } unique
{ domain: 1 } unique sparse

// Relationships
{ company_id: 1, supplier_id: 1 } unique sparse
{ company_id: 1, status: 1 }

// Requirements
{ company_id: 1, status: 1 }
{ due_date: 1, status: 1 }

// Secure Links
{ secure_identifier: 1 } unique
{ expires_at: 1 } TTL
```

---

## 4. Authentication Flow (Passwordless)

```
1. User enters email → POST /auth/request-link
2. Backend generates SecureLink (64-char token, 15-min expiry)
3. Email sent with link: /auth/verify/{token}
4. User clicks → GET /auth/verify/{token}
5. Backend validates, returns JWT pair (access + refresh)
6. Frontend stores tokens, redirects to role-based dashboard
```

---

## 5. Status State Machines

### Supplier Invitation
```
PENDING → ACTIVE (accept) → SUSPENDED → ACTIVE (reactivate)
        → REJECTED (decline)           → TERMINATED
```

### Requirement Response
```
PENDING → IN_PROGRESS → SUBMITTED → APPROVED
                                  → REJECTED
                                  → UNDER_REVIEW (revision)
        → EXPIRED
```

---

## 6. Implementation Phases

### Phase 1: Foundation (Backend)
1. Project scaffolding with Go modules
2. MongoDB connection and index creation
3. JWT RS512 authentication service
4. Magic link service with email
5. Organization and User models/repos/services
6. Auth handlers and middleware

### Phase 2: Foundation (Frontend)
1. Vite + React + TypeScript scaffolding
2. Tailwind + Shadcn/ui setup
3. Auth context with token management
4. API client with refresh logic
5. Layout components and routing
6. Magic link login flow

### Phase 3: Company Portal Core
1. Supplier list and invite flow
2. Relationship management
3. Questionnaire template browser
4. Basic questionnaire CRUD
5. Requirement creation

### Phase 4: Questionnaire Builder
1. Drag-drop question list
2. Question type editors (single/multiple choice)
3. Scoring configuration UI
4. Topic/category management
5. Template cloning

### Phase 5: Supplier Portal Core
1. Dashboard with pending requests
2. Request list and details
3. Questionnaire fill form
4. Response submission
5. Status tracking

### Phase 6: Scoring & Review
1. Backend scoring calculation
2. Score preview in fill form
3. Score display after submission
4. Company review workflow
5. Approve/reject with feedback

### Phase 7: CheckFix Integration
1. CheckFix API client service
2. Account linking flow
3. Report verification
4. Grade comparison logic
5. Status display in dashboard

### Phase 8: Polish
1. Email notifications
2. Reminder system
3. Audit logging
4. Swagger documentation
5. i18n (German translations)
6. Mobile responsiveness

---

## 7. Reference Files (Patterns to Follow)

### Backend Patterns (from checkfix_backend)
- `internal/auth/jwt_service.go` - JWT RS512 implementation
- `internal/middleware/auth.go` - Auth middleware chain
- `internal/services/auth_service.go` - Service layer pattern
- `cmd/server/main.go` - Gin router setup, DI

### Frontend Patterns (from checkfix_frontend)
- `src/hooks/useAuth.tsx` - Auth context with multi-org
- `src/services/apiClient.ts` - API client with token refresh
- `src/App.tsx` - Routing and provider hierarchy
- `src/components/profile/OrganizationForm.tsx` - Form patterns

---

## 8. Environment Variables

### Backend
```
DATABASE_URI=mongodb://localhost:27017
DATABASE_NAME=supplier_portal
JWT_PRIVATE_KEY_PATH=./keys/jwt-rs512-private.pem
JWT_PUBLIC_KEY_PATH=./keys/jwt-rs512-public.pem
MAIL_SERVICE_URL=https://mail.example.com
MAIL_API_KEY=xxx
CHECKFIX_API_URL=https://api.checkfix.io
CHECKFIX_API_KEY=xxx
```

### Frontend
```
VITE_API_BASE_URL=http://localhost:8080
VITE_APP_NAME=Supplier Portal
```

---

## 9. Confirmed Decisions

- **Project Names**: `nisfix_backend` and `nisfix_frontend`
- **Email Service**: Existing mail service (same as checkfix_backend)
- **Implementation Order**: Backend-first with Swagger docs

## 10. Next Steps (After Plan Approval)

1. Create `nisfix_backend` directory and initialize Go module
2. Set up MongoDB connection and configuration
3. Implement JWT RS512 authentication service
4. Create core models (Organization, User, SecureLink)
5. Build auth handlers and middleware
6. Add questionnaire and requirement models
7. Implement scoring service
8. Add CheckFix integration service
9. Generate Swagger documentation
10. Then proceed to frontend implementation
