# NisFix Frontend - UI/UX Architecture Plan

## Document Metadata
- **Project**: nisfix_frontend (Supplier Security Portal)
- **Created**: 2025-12-29
- **Reference**: checkfix_frontend patterns
- **Tech Stack**: React 19, TypeScript, Vite, Tailwind CSS, Shadcn/ui, React Query v5, React Router v7

---

## 1. Design Vision

#DESIGN_RATIONALE: The UI follows a split-portal approach where Companies and Suppliers have distinct experiences tailored to their workflows. The design prioritizes clarity in security compliance status, progressive disclosure of complexity (simple dashboard to detailed questionnaire builder), and consistent patterns from checkfix_frontend for developer familiarity.

#UI_ASSUMPTION: B2B users expect professional, business-focused interfaces over consumer-style designs. Users prioritize efficiency and clarity over visual flair.

#PERSONA_ASSUMPTION: Company users are security/compliance managers who manage multiple suppliers. Supplier users are IT/security contacts responding to compliance requests.

---

## 2. User Flows

### 2.1 Primary Flow: Company Onboarding
1. **Entry**: User lands on `/` (landing page with magic link form)
2. **Action**: Enter email -> API: `POST /api/v1/auth/request-link`
3. **Response**: Success toast, email sent notification
4. **Action**: Click email link -> API: `GET /api/v1/auth/verify/:token`
5. **Response**: JWT tokens returned, stored in localStorage
6. **Route**: Redirect to `/company` (Company Dashboard)

#EXPORT_FLOW: company_onboarding_flow
#API_DEPENDENCY: POST /api/v1/auth/request-link, GET /api/v1/auth/verify/:token, GET /api/v1/auth/profile
#UX_UNCERTAINTY: Should we show organization type selection during first login, or detect from email domain?

### 2.2 Primary Flow: Company Invites Supplier
1. **Entry**: Company Dashboard `/company`
2. **Action**: Click "Invite Supplier" -> Opens InviteDialog
3. **Input**: Supplier email, classification (critical/standard)
4. **Action**: Submit -> API: `POST /api/v1/suppliers`
5. **Response**: Supplier created with PENDING status
6. **UI Update**: Supplier appears in list with "Pending" badge
7. **Backend**: Email sent to supplier with magic link

#EXPORT_FLOW: supplier_invitation_flow
#API_DEPENDENCY: POST /api/v1/suppliers, GET /api/v1/suppliers
#DESIGN_RATIONALE: Classification at invite time helps companies prioritize critical suppliers for stricter requirements.

### 2.3 Primary Flow: Company Creates Questionnaire
1. **Entry**: `/company/questionnaires`
2. **Action**: Click "Create Questionnaire" or select template
3. **Route**: Navigate to `/company/questionnaires/new`
4. **Action**: Drag-drop questions, set scoring, configure pass criteria
5. **Action**: Save -> API: `POST /api/v1/questionnaires`
6. **Response**: Questionnaire created, redirect to list
7. **Action**: Assign to suppliers -> API: `POST /api/v1/suppliers/:id/requirements`

#EXPORT_FLOW: questionnaire_creation_flow
#API_DEPENDENCY: GET /api/v1/questionnaire-templates, POST /api/v1/questionnaires, POST /api/v1/suppliers/:id/requirements
#UX_UNCERTAINTY: Should questionnaire builder auto-save or require explicit save? Auto-save may cause confusion with unfinished questions.

### 2.4 Primary Flow: Supplier Accepts Invitation
1. **Entry**: Email with magic link
2. **Action**: Click link -> API: `GET /api/v1/auth/verify/:token`
3. **Response**: JWT tokens, organization info
4. **Route**: Redirect to `/supplier` (first-time) or dashboard
5. **UI**: Show pending company relationships

#EXPORT_FLOW: supplier_accept_flow
#API_DEPENDENCY: GET /api/v1/auth/verify/:token, POST /api/v1/companies/:id/accept
#UI_ASSUMPTION: Suppliers may be invited by multiple companies. UI should handle multi-company context.

### 2.5 Primary Flow: Supplier Completes Questionnaire
1. **Entry**: `/supplier/requests`
2. **Action**: Click requirement row -> Navigate to `/supplier/questionnaire/:id`
3. **UI**: Display QuestionnaireForm with progress indicator
4. **Action**: Answer questions, see live score preview
5. **Action**: Submit -> API: `POST /api/v1/responses/:id/submit`
6. **Response**: Score calculated, status updated
7. **Route**: Return to requests list with updated status

#EXPORT_FLOW: questionnaire_completion_flow
#API_DEPENDENCY: GET /api/v1/requirements, POST /api/v1/requirements/:id/responses, POST /api/v1/responses/:id/submit
#ACCESSIBILITY_REQUIREMENT: Form must support keyboard navigation for all question types.

### 2.6 Primary Flow: Company Reviews Submission
1. **Entry**: `/company/suppliers/:id` (Supplier Detail)
2. **UI**: Show submitted responses with scores
3. **Action**: Review answers, provide feedback
4. **Action**: Approve/Reject -> API: `POST /api/v1/responses/:id/approve` or `POST /api/v1/responses/:id/reject`
5. **Response**: Status updated, notification sent to supplier

#EXPORT_FLOW: response_review_flow
#API_DEPENDENCY: GET /api/v1/responses/:id, POST /api/v1/responses/:id/approve, POST /api/v1/responses/:id/reject

### 2.7 Primary Flow: Supplier Links CheckFix Account
1. **Entry**: `/supplier/checkfix`
2. **Action**: Enter CheckFix account email or report hash
3. **Action**: Submit -> API: `POST /api/v1/checkfix/link-account`
4. **Response**: Account linked, grades fetched
5. **UI Update**: Show CheckFix grades (A-F) with status indicators

#EXPORT_FLOW: checkfix_linking_flow
#API_DEPENDENCY: POST /api/v1/checkfix/link-account, GET /api/v1/checkfix/status
#UX_UNCERTAINTY: What happens if CheckFix grade expires? Should we show warning or auto-prompt refresh?

---

## 3. Page Structure

### 3.1 Public Pages (No Auth)

#### Page: Landing / Login
- **Route**: `/`
- **Purpose**: Magic link authentication entry point
- **Data needs**: None (stateless)
- **Components**: MagicLinkForm, LanguageSwitcher

#### Page: Email Verification
- **Route**: `/auth/verify/:token`
- **Purpose**: Validate magic link token, establish session
- **Data needs**: API: `GET /api/v1/auth/verify/:token`
- **Components**: LoadingSpinner, ErrorMessage, auto-redirect

### 3.2 Company Portal Pages

#### Page: Company Dashboard
- **Route**: `/company`
- **Purpose**: Overview of supplier compliance status
- **Data needs**:
  - API: `GET /api/v1/suppliers` (with stats)
  - API: `GET /api/v1/auth/profile`
- **Components**: StatCards, SupplierOverviewChart, RecentActivityList

#VISUAL_HIERARCHY: Dashboard prioritizes aggregate compliance metrics at top, followed by actionable items (pending reviews), then recent activity.

#### Page: Supplier List
- **Route**: `/company/suppliers`
- **Purpose**: Manage all supplier relationships
- **Data needs**: API: `GET /api/v1/suppliers` (paginated)
- **Components**: DataTable, StatusBadge, InviteDialog, FilterBar

#### Page: Supplier Detail
- **Route**: `/company/suppliers/:id`
- **Purpose**: View supplier compliance details, manage requirements
- **Data needs**:
  - API: `GET /api/v1/suppliers/:id`
  - API: `GET /api/v1/suppliers/:id/requirements`
- **Components**: SupplierHeader, RequirementsChecklist, ResponseViewer, GradeDisplay

#### Page: Questionnaire List
- **Route**: `/company/questionnaires`
- **Purpose**: Browse and manage questionnaire templates
- **Data needs**:
  - API: `GET /api/v1/questionnaires`
  - API: `GET /api/v1/questionnaire-templates`
- **Components**: DataTable, TemplateCard, CreateButton

#### Page: Questionnaire Builder
- **Route**: `/company/questionnaires/new`, `/company/questionnaires/:id`
- **Purpose**: Create or edit questionnaires with drag-drop
- **Data needs**:
  - API: `GET /api/v1/questionnaires/:id` (edit mode)
  - API: `GET /api/v1/questionnaire-templates/:id` (from template)
- **Components**: QuestionnaireBuilder, QuestionEditor, ScoringConfig, DragDropContext

#### Page: Template Browser
- **Route**: `/company/templates`
- **Purpose**: Browse pre-built questionnaire templates (ISO27001, GDPR, NIS2)
- **Data needs**: API: `GET /api/v1/questionnaire-templates`
- **Components**: TemplateGrid, TemplatePreview, CloneButton

### 3.3 Supplier Portal Pages

#### Page: Supplier Dashboard
- **Route**: `/supplier`
- **Purpose**: Overview of pending requests and compliance status
- **Data needs**:
  - API: `GET /api/v1/requirements` (with status filter)
  - API: `GET /api/v1/companies`
- **Components**: StatCards, PendingRequestsList, CheckFixStatus

#VISUAL_HIERARCHY: Urgent items (pending/overdue) at top, followed by in-progress work, then completed items.

#### Page: Company Relationships
- **Route**: `/supplier/companies`
- **Purpose**: Manage relationships with requesting companies
- **Data needs**: API: `GET /api/v1/companies`
- **Components**: CompanyCard, AcceptRejectDialog, StatusBadge

#### Page: Requirements List
- **Route**: `/supplier/requests`
- **Purpose**: View and respond to compliance requirements
- **Data needs**: API: `GET /api/v1/requirements`
- **Components**: DataTable, StatusBadge, DueDateIndicator

#### Page: Questionnaire Fill
- **Route**: `/supplier/questionnaire/:id`
- **Purpose**: Complete a questionnaire requirement
- **Data needs**:
  - API: `GET /api/v1/requirements/:id`
  - API: `GET /api/v1/requirements/:id/questions`
- **Components**: QuestionnaireForm, ProgressBar, ScorePreview, SubmitButton

#### Page: CheckFix Integration
- **Route**: `/supplier/checkfix`
- **Purpose**: Link and display CheckFix security grades
- **Data needs**: API: `GET /api/v1/checkfix/status`
- **Components**: CheckFixLinkForm, GradeDisplay, ReportHistory

### 3.4 Shared Pages

#### Page: Profile
- **Route**: `/profile`
- **Purpose**: User profile and preferences
- **Data needs**: API: `GET /api/v1/auth/profile`
- **Components**: ProfileForm, AvatarUpload, LanguageSelector

#### Page: Settings
- **Route**: `/settings`
- **Purpose**: Organization settings (admin only)
- **Data needs**: API: `GET /api/v1/organizations/:id`
- **Components**: OrganizationForm, UserManagement, NotificationSettings

---

## 4. Component Hierarchy

### 4.1 Layout Components

#### Component: AppLayout
- **Purpose**: Main layout wrapper with sidebar and content area
- **Props**: None (uses Outlet for children)
- **State**: Sidebar collapsed state
- **API calls**: None
#COMPONENT_PURPOSE: Provides consistent layout structure for all authenticated routes
#INTERACTION_PATTERN: Collapsible sidebar with icon-only mode on collapse
#EXPORT_COMPONENT: AppLayout

#### Component: CompanySidebar
- **Parent**: AppLayout
- **Purpose**: Navigation for Company Portal
- **Props**: None (uses auth context)
- **State**: Active route highlight
- **API calls**: None
- **Menu Items**: Dashboard, Suppliers, Questionnaires, Templates, Settings
#COMPONENT_PURPOSE: Company-specific navigation with role-based menu items
#EXPORT_COMPONENT: CompanySidebar

#### Component: SupplierSidebar
- **Parent**: AppLayout
- **Purpose**: Navigation for Supplier Portal
- **Props**: None (uses auth context)
- **State**: Active route highlight
- **API calls**: None
- **Menu Items**: Dashboard, Companies, Requests, CheckFix, Settings
#COMPONENT_PURPOSE: Supplier-specific navigation optimized for response workflow
#EXPORT_COMPONENT: SupplierSidebar

### 4.2 Authentication Components

#### Component: RouteGuard
- **Purpose**: Protect routes based on auth state and organization type
- **Props**: `requiredRole?: 'company' | 'supplier'`
- **State**: Loading state during auth check
- **API calls**: Uses auth context (which calls profile API)
#COMPONENT_PURPOSE: Redirects unauthorized users and enforces portal separation
#STATE_ASSUMPTION: Auth context provides user, loading, isCompany(), isSupplier() helpers
#EXPORT_COMPONENT: RouteGuard

#### Component: MagicLinkForm
- **Purpose**: Email input for passwordless authentication
- **Props**: `onSuccess?: () => void`
- **State**: Email input, submitting state, mode (login/register)
- **API calls**: `POST /api/v1/auth/request-link`
#COMPONENT_PURPOSE: Single entry point for authentication without passwords
#API_DEPENDENCY: POST /api/v1/auth/request-link
#EXPORT_COMPONENT: MagicLinkForm

### 4.3 Common Components

#### Component: DataTable
- **Purpose**: Reusable table with sorting, filtering, pagination
- **Props**: `columns`, `data`, `onRowClick?`, `pagination?`, `sorting?`
- **State**: Local sort/filter state (or controlled via props)
- **API calls**: None (data provided via props)
#COMPONENT_PURPOSE: Consistent table display across all list views
#ACCESSIBILITY_REQUIREMENT: Keyboard navigable rows, screen reader announcements for sort changes
#EXPORT_COMPONENT: DataTable

#### Component: StatusBadge
- **Purpose**: Display status with color-coded badge
- **Props**: `status: 'pending' | 'active' | 'approved' | 'rejected' | etc.`
- **State**: None (stateless)
- **API calls**: None
#COMPONENT_PURPOSE: Visual indicator for various entity statuses
#EXPORT_COMPONENT: StatusBadge

#### Component: ScoreGauge
- **Purpose**: Circular gauge showing percentage score
- **Props**: `score: number`, `maxScore?: number`, `threshold?: number`
- **State**: None (stateless)
- **API calls**: None
#COMPONENT_PURPOSE: Visual representation of questionnaire scores
#ACCESSIBILITY_REQUIREMENT: Include aria-label with score percentage
#EXPORT_COMPONENT: ScoreGauge

#### Component: GradeDisplay
- **Purpose**: Display CheckFix letter grade (A-F) with color
- **Props**: `grade: 'A' | 'B' | 'C' | 'D' | 'E' | 'F'`, `score?: number`
- **State**: None (stateless)
- **API calls**: None
#COMPONENT_PURPOSE: Consistent display of CheckFix security grades
#EXPORT_COMPONENT: GradeDisplay

### 4.4 Questionnaire Components

#### Component: QuestionnaireBuilder
- **Purpose**: Drag-drop editor for creating questionnaires
- **Props**: `questionnaire?: Questionnaire`, `onSave: (q: Questionnaire) => void`
- **State**: Questions array, selected question, drag state
- **API calls**: None (parent handles save)
- **Children**: QuestionEditor, DragDropContext, ScoringConfig
#COMPONENT_PURPOSE: Full questionnaire editing experience with drag-drop reordering
#STATE_ASSUMPTION: Uses @dnd-kit for drag-drop functionality
#INTERACTION_PATTERN: Click question to edit, drag handle to reorder, delete button to remove
#EXPORT_COMPONENT: QuestionnaireBuilder

#### Component: QuestionEditor
- **Parent**: QuestionnaireBuilder
- **Purpose**: Edit single question details
- **Props**: `question: Question`, `onChange: (q: Question) => void`
- **State**: Form state for question fields
- **API calls**: None
#COMPONENT_PURPOSE: Form for editing question text, type, options, scoring
#EXPORT_COMPONENT: QuestionEditor

#### Component: ScoringConfig
- **Parent**: QuestionnaireBuilder
- **Purpose**: Configure pass/fail thresholds and required questions
- **Props**: `config: ScoringConfig`, `onChange: (c: ScoringConfig) => void`
- **State**: Form state
- **API calls**: None
#COMPONENT_PURPOSE: Define minimum score percentage and must-pass questions
#EXPORT_COMPONENT: ScoringConfig

#### Component: QuestionnaireForm
- **Purpose**: Fillable questionnaire for suppliers
- **Props**: `questionnaire: Questionnaire`, `onSubmit: (answers: Answer[]) => void`
- **State**: Answers array, current section, validation errors
- **API calls**: None (parent handles submit)
#COMPONENT_PURPOSE: Step-by-step questionnaire completion with validation
#ACCESSIBILITY_REQUIREMENT: Form labels, error messages, focus management
#INTERACTION_PATTERN: Section-by-section navigation with progress indicator
#EXPORT_COMPONENT: QuestionnaireForm

#### Component: ScorePreview
- **Parent**: QuestionnaireForm
- **Purpose**: Live score calculation as user answers questions
- **Props**: `answers: Answer[]`, `questions: Question[]`
- **State**: None (computed from props)
- **API calls**: None
#COMPONENT_PURPOSE: Real-time feedback on questionnaire progress and score
#UX_UNCERTAINTY: Should we show predicted pass/fail status or just percentage?
#EXPORT_COMPONENT: ScorePreview

### 4.5 Supplier Management Components

#### Component: SupplierCard
- **Purpose**: Compact display of supplier with status
- **Props**: `supplier: Supplier`, `onClick?: () => void`
- **State**: None
- **API calls**: None
#COMPONENT_PURPOSE: Grid-view representation of supplier for dashboard
#EXPORT_COMPONENT: SupplierCard

#### Component: InviteDialog
- **Purpose**: Modal for inviting new supplier
- **Props**: `open: boolean`, `onClose: () => void`, `onSuccess: () => void`
- **State**: Form state (email, classification)
- **API calls**: `POST /api/v1/suppliers`
#COMPONENT_PURPOSE: Streamlined supplier invitation flow
#API_DEPENDENCY: POST /api/v1/suppliers
#EXPORT_COMPONENT: InviteDialog

#### Component: RequirementsChecklist
- **Purpose**: Display requirements for a supplier with status
- **Props**: `requirements: Requirement[]`, `onAddRequirement?: () => void`
- **State**: Expanded requirement ID
- **API calls**: None
#COMPONENT_PURPOSE: Overview of all requirements assigned to a supplier
#EXPORT_COMPONENT: RequirementsChecklist

### 4.6 Company Relationship Components

#### Component: CompanyCard
- **Purpose**: Display company relationship for supplier view
- **Props**: `relationship: CompanySupplierRelationship`, `onAccept?: () => void`, `onReject?: () => void`
- **State**: None
- **API calls**: None (parent handles actions)
#COMPONENT_PURPOSE: Supplier view of company relationships with action buttons
#EXPORT_COMPONENT: CompanyCard

#### Component: RequestCard
- **Purpose**: Display requirement request for supplier
- **Props**: `requirement: Requirement`, `onClick?: () => void`
- **State**: None
- **API calls**: None
#COMPONENT_PURPOSE: Compact requirement display with due date and status
#EXPORT_COMPONENT: RequestCard

#### Component: CheckFixLinkForm
- **Purpose**: Form to link CheckFix account
- **Props**: `onSuccess: (result: CheckFixLinkResult) => void`
- **State**: Form state (email or hash), submitting
- **API calls**: `POST /api/v1/checkfix/link-account`
#COMPONENT_PURPOSE: Connect supplier to their CheckFix security assessment
#API_DEPENDENCY: POST /api/v1/checkfix/link-account
#EXPORT_COMPONENT: CheckFixLinkForm

---

## 5. State Management

### 5.1 Global State (Context)

#### AuthContext
- **Purpose**: Authentication state and user profile
- **State**:
  - `user: User | null`
  - `session: Session | null`
  - `userProfile: UserProfile | null`
  - `loading: boolean`
  - `isSwitchingOrg: boolean`
- **Actions**:
  - `signOut(): Promise<void>`
  - `refreshProfile(): Promise<void>`
  - `isCompany(): boolean`
  - `isSupplier(): boolean`
  - `isAdmin(): boolean`
#EXPORT_STATE: AuthContext
#STATE_ASSUMPTION: Profile includes organization_type to determine portal routing

#### LanguageContext
- **Purpose**: i18n language selection and translation
- **State**:
  - `language: 'de' | 'en'`
  - `translations: Record<string, string>`
- **Actions**:
  - `setLanguage(lang: 'de' | 'en'): void`
  - `t(key: string, variables?: Record<string, string | number>): string`
#EXPORT_STATE: LanguageContext
#STATE_ASSUMPTION: Default language detected from browser, persisted to localStorage

### 5.2 Server State (React Query)

#### Query Keys Structure
```typescript
const queryKeys = {
  // Auth
  profile: ['profile'] as const,

  // Company Portal
  suppliers: (filters?: SupplierFilters) => ['suppliers', filters] as const,
  supplier: (id: string) => ['suppliers', id] as const,
  supplierRequirements: (id: string) => ['suppliers', id, 'requirements'] as const,

  questionnaires: (filters?: QFilters) => ['questionnaires', filters] as const,
  questionnaire: (id: string) => ['questionnaires', id] as const,
  templates: () => ['questionnaire-templates'] as const,
  template: (id: string) => ['questionnaire-templates', id] as const,

  // Supplier Portal
  companies: () => ['companies'] as const,
  requirements: (filters?: ReqFilters) => ['requirements', filters] as const,
  requirement: (id: string) => ['requirements', id] as const,
  checkfixStatus: () => ['checkfix', 'status'] as const,

  // Responses
  responses: (requirementId: string) => ['responses', requirementId] as const,
  response: (id: string) => ['responses', id] as const,
};
```
#EXPORT_STATE: queryKeys
#STATE_ASSUMPTION: React Query handles caching, background refresh, and optimistic updates

#### Cache Configuration
```typescript
const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      staleTime: 5 * 60 * 1000, // 5 minutes
      gcTime: 10 * 60 * 1000,   // 10 minutes (garbage collection)
      refetchOnWindowFocus: false,
      refetchOnReconnect: true,
      retry: 1,
    },
    mutations: {
      retry: 1,
    },
  },
});
```
#EXPORT_STATE: queryClientConfig

### 5.3 Component State

#### Form State Pattern
- Use React Hook Form with Zod validation
- Schema defined per form component
- Errors displayed inline with FormMessage
- Submit disabled during loading

#### UI State Patterns
- Modal open/close: useState with boolean
- Selected items: useState with ID
- Filter state: useState or URL search params
- Pagination: URL search params for shareable links

#STATE_ASSUMPTION: Filter and pagination state stored in URL for bookmarkable/shareable views

---

## 6. Design System

### 6.1 Typography

#EXPORT_STYLE: typography_hierarchy
```
- Page Title: text-2xl font-bold (24px, 700)
- Section Title: text-lg font-medium (18px, 500)
- Card Title: text-base font-semibold (16px, 600)
- Body Text: text-sm (14px, 400)
- Small Text: text-xs (12px, 400)
- Labels: text-sm font-medium (14px, 500)
```

### 6.2 Colors

#EXPORT_STYLE: color_palette
```
Primary: Tailwind Blue (blue-600, blue-700 hover)
Success: Tailwind Green (green-600)
Warning: Tailwind Amber (amber-500)
Danger: Tailwind Red (red-600)
Neutral: Tailwind Gray scale

Grade Colors:
- A: green-600
- B: green-500
- C: amber-500
- D: orange-500
- E: red-500
- F: red-700

Status Colors:
- Pending: amber-500
- Active: blue-600
- Approved: green-600
- Rejected: red-600
- Expired: gray-500
```

### 6.3 Spacing

#EXPORT_STYLE: spacing_system
```
- Page padding: p-6 (24px)
- Card padding: p-4 (16px)
- Section gap: space-y-6 (24px)
- Form field gap: space-y-4 (16px)
- Button padding: px-4 py-2 (16px horizontal, 8px vertical)
- Icon size: 16px (inline), 20px (buttons), 24px (headers)
```

### 6.4 Component Patterns

#EXPORT_STYLE: component_patterns

#### Cards
```typescript
<Card>
  <CardHeader>
    <CardTitle>{title}</CardTitle>
    <CardDescription>{description}</CardDescription>
  </CardHeader>
  <CardContent>
    {content}
  </CardContent>
  <CardFooter>
    {actions}
  </CardFooter>
</Card>
```

#### Forms
```typescript
<Form {...form}>
  <form onSubmit={form.handleSubmit(onSubmit)}>
    <FormField
      control={form.control}
      name="fieldName"
      render={({ field }) => (
        <FormItem>
          <FormLabel>{label}</FormLabel>
          <FormControl>
            <Input {...field} />
          </FormControl>
          <FormMessage />
        </FormItem>
      )}
    />
    <Button type="submit" disabled={submitting}>
      {submitting ? t('common.saving') : t('common.save')}
    </Button>
  </form>
</Form>
```

#### Dialogs
```typescript
<Dialog open={open} onOpenChange={setOpen}>
  <DialogTrigger asChild>
    <Button>{triggerText}</Button>
  </DialogTrigger>
  <DialogContent>
    <DialogHeader>
      <DialogTitle>{title}</DialogTitle>
      <DialogDescription>{description}</DialogDescription>
    </DialogHeader>
    {content}
    <DialogFooter>
      <Button variant="outline" onClick={() => setOpen(false)}>
        {t('common.cancel')}
      </Button>
      <Button onClick={handleConfirm}>
        {t('common.confirm')}
      </Button>
    </DialogFooter>
  </DialogContent>
</Dialog>
```

---

## 7. Responsive Strategy

#RESPONSIVE_STRATEGY: mobile_first_breakpoints

### Breakpoints
- **Mobile**: < 640px (sm)
- **Tablet**: 640px - 1024px (sm to lg)
- **Desktop**: > 1024px (lg+)

### Mobile Adaptations
- Sidebar collapses to hamburger menu
- DataTables become card lists
- Multi-column forms become single column
- Dialogs become full-screen sheets

### Tablet Adaptations
- Sidebar remains visible but icon-only
- Tables have horizontal scroll
- Forms maintain 2-column layout

### Desktop Adaptations
- Full sidebar with labels
- Multi-column layouts
- Side-by-side panels for builder

#UI_ASSUMPTION: Primary users are on desktop/laptop devices. Mobile is secondary but must be functional.

---

## 8. Accessibility Plan

#ACCESSIBILITY_REQUIREMENT: WCAG 2.1 AA compliance

### Keyboard Navigation
- All interactive elements focusable
- Tab order follows visual order
- Focus visible on all elements
- Escape closes modals/dialogs
- Arrow keys navigate within components (radio groups, menus)

#ARIA_STRATEGY: semantic_html_first
- Use semantic HTML elements (button, nav, main, aside)
- Add ARIA only where HTML semantics insufficient
- Live regions for dynamic content updates
- Labels for all form inputs

### Screen Reader Support
- All images have alt text
- Icon-only buttons have aria-label
- Form errors announced on change
- Page title updates on navigation
- Loading states announced

### Color Contrast
- Minimum 4.5:1 for normal text
- Minimum 3:1 for large text and UI components
- Status colors have icon/text backup, not color alone

### Focus Management
- Focus trapped in modals
- Focus returned to trigger on modal close
- Focus moved to first error on form validation fail

---

## 9. API Integration Points

### 9.1 Data Requirements by Component

#API_DEPENDENCY: critical_api_endpoints

| Component | Endpoint | Method | Purpose |
|-----------|----------|--------|---------|
| MagicLinkForm | /api/v1/auth/request-link | POST | Send magic link |
| EmailVerification | /api/v1/auth/verify/:token | GET | Validate token |
| RouteGuard | /api/v1/auth/profile | GET | Get user profile |
| SupplierList | /api/v1/suppliers | GET | List suppliers |
| InviteDialog | /api/v1/suppliers | POST | Create supplier |
| SupplierDetail | /api/v1/suppliers/:id | GET | Supplier info |
| RequirementCreate | /api/v1/suppliers/:id/requirements | POST | Add requirement |
| QuestionnaireList | /api/v1/questionnaires | GET | List questionnaires |
| QuestionnaireBuilder | /api/v1/questionnaires | POST | Create questionnaire |
| QuestionnaireBuilder | /api/v1/questionnaires/:id | PATCH | Update questionnaire |
| TemplateList | /api/v1/questionnaire-templates | GET | List templates |
| CompanyList | /api/v1/companies | GET | Supplier's companies |
| AcceptInvite | /api/v1/companies/:id/accept | POST | Accept invite |
| RequirementsList | /api/v1/requirements | GET | Supplier's requirements |
| QuestionnaireForm | /api/v1/requirements/:id/responses | POST | Start response |
| QuestionnaireForm | /api/v1/responses/:id/submit | POST | Submit response |
| ResponseReview | /api/v1/responses/:id/approve | POST | Approve response |
| ResponseReview | /api/v1/responses/:id/reject | POST | Reject response |
| CheckFixLink | /api/v1/checkfix/link-account | POST | Link CheckFix |

### 9.2 Error Handling Patterns

```typescript
// API Client error handling (from checkfix_frontend pattern)
try {
  const result = await apiClient.createSupplier(data);
  toast.success(t('supplier.created'));
  queryClient.invalidateQueries({ queryKey: queryKeys.suppliers() });
} catch (error) {
  if (error instanceof ApiClientError) {
    if (error.statusCode === 400) {
      toast.error(t('supplier.validation_error'));
    } else if (error.statusCode === 401) {
      // Auth context will handle redirect
    } else if (error.statusCode === 403) {
      toast.error(t('common.permission_denied'));
    } else {
      toast.error(t('common.error_generic'));
    }
  }
}
```

### 9.3 Token Management

#STATE_ASSUMPTION: Token refresh handled automatically by apiClient

- Access token stored in localStorage
- Refresh token stored in localStorage
- Token expiry checked before each request
- Automatic refresh when expired
- Clear tokens on 401 response
- Redirect to login when refresh fails

---

## 10. Form Validation Patterns

### 10.1 Validation Schema Examples

```typescript
// Supplier Invite Form
const inviteSchema = z.object({
  email: z.string()
    .min(1, t('validation.email_required'))
    .email(t('validation.email_invalid')),
  classification: z.enum(['critical', 'standard']),
  notes: z.string().optional(),
});

// Questionnaire Question
const questionSchema = z.object({
  text: z.string().min(1, t('validation.question_required')),
  type: z.enum(['single_choice', 'multiple_choice']),
  options: z.array(z.object({
    text: z.string().min(1),
    points: z.number().min(0),
    isCorrect: z.boolean().optional(),
  })).min(2, t('validation.min_options')),
  weight: z.number().min(1).max(10).default(1),
  isMustPass: z.boolean().default(false),
});

// Scoring Configuration
const scoringSchema = z.object({
  minPassScore: z.number().min(0).max(100),
  showScorePreview: z.boolean(),
  allowRetry: z.boolean(),
});
```

### 10.2 Validation UX Patterns

- Validate on blur for individual fields
- Validate all on submit
- Show inline errors below fields
- Scroll to first error on submit fail
- Disable submit while loading
- Clear field error on change

---

## 11. UI Assumptions Summary

#UI_ASSUMPTION: Users prefer explicit save over auto-save for questionnaires
#UI_ASSUMPTION: Email is primary identifier (no username system)
#UI_ASSUMPTION: Organizations are single-type (Company OR Supplier, not both)
#UI_ASSUMPTION: All times displayed in user's local timezone
#UI_ASSUMPTION: Pagination defaults to 20 items per page
#UI_ASSUMPTION: English is fallback language when translation missing
#UI_ASSUMPTION: Toast notifications auto-dismiss after 5 seconds
#UI_ASSUMPTION: Modals close on backdrop click unless form has changes
#UI_ASSUMPTION: Date format follows locale (DD.MM.YYYY for German, MM/DD/YYYY for English)

#PERSONA_ASSUMPTION: Company users check portal 1-2x daily for updates
#PERSONA_ASSUMPTION: Supplier users visit primarily when notified of new requirements
#PERSONA_ASSUMPTION: Admin users need access to all organization settings
#PERSONA_ASSUMPTION: Viewer users only need read access to dashboards

#UX_UNCERTAINTY: Should questionnaire responses allow partial saves as draft?
#UX_UNCERTAINTY: How to handle timezone differences for due dates?
#UX_UNCERTAINTY: Should we show competitor comparison in dashboard?
#UX_UNCERTAINTY: What happens when CheckFix integration fails validation?

---

## 12. Quality Checklist

- [x] All user flows mapped with API dependencies
- [x] Component hierarchy defined
- [x] State management planned (Context + React Query)
- [x] API integration points identified
- [x] Accessibility requirements documented (WCAG 2.1 AA)
- [x] Every assumption tagged (#UI_ASSUMPTION, #PERSONA_ASSUMPTION)
- [x] Critical components exported (#EXPORT_COMPONENT)
- [x] Plan saved to docs/ui-ux-plan.md

---

## Appendix A: File Structure

```
nisfix_frontend/
├── src/
│   ├── components/
│   │   ├── auth/
│   │   │   ├── RouteGuard.tsx
│   │   │   └── MagicLinkForm.tsx
│   │   ├── layout/
│   │   │   ├── AppLayout.tsx
│   │   │   ├── CompanySidebar.tsx
│   │   │   └── SupplierSidebar.tsx
│   │   ├── common/
│   │   │   ├── DataTable.tsx
│   │   │   ├── StatusBadge.tsx
│   │   │   ├── ScoreGauge.tsx
│   │   │   ├── GradeDisplay.tsx
│   │   │   └── LoadingSpinner.tsx
│   │   ├── questionnaire/
│   │   │   ├── builder/
│   │   │   │   ├── QuestionnaireBuilder.tsx
│   │   │   │   ├── QuestionEditor.tsx
│   │   │   │   └── ScoringConfig.tsx
│   │   │   └── viewer/
│   │   │       ├── QuestionnaireForm.tsx
│   │   │       ├── ScorePreview.tsx
│   │   │       └── ProgressBar.tsx
│   │   ├── supplier/
│   │   │   ├── SupplierCard.tsx
│   │   │   ├── InviteDialog.tsx
│   │   │   └── RequirementsChecklist.tsx
│   │   ├── company/
│   │   │   ├── CompanyCard.tsx
│   │   │   ├── RequestCard.tsx
│   │   │   └── CheckFixLinkForm.tsx
│   │   └── ui/
│   │       └── (shadcn components)
│   ├── hooks/
│   │   ├── useAuth.tsx
│   │   ├── useSuppliers.ts
│   │   ├── useQuestionnaires.ts
│   │   ├── useRequirements.ts
│   │   └── useCheckFix.ts
│   ├── pages/
│   │   ├── Index.tsx
│   │   ├── EmailVerification.tsx
│   │   ├── company/
│   │   │   ├── CompanyDashboard.tsx
│   │   │   ├── SupplierList.tsx
│   │   │   ├── SupplierDetail.tsx
│   │   │   ├── QuestionnaireList.tsx
│   │   │   ├── QuestionnaireBuilder.tsx
│   │   │   └── TemplateBrowser.tsx
│   │   ├── supplier/
│   │   │   ├── SupplierDashboard.tsx
│   │   │   ├── CompanyList.tsx
│   │   │   ├── RequirementsList.tsx
│   │   │   ├── QuestionnaireFill.tsx
│   │   │   └── CheckFixPage.tsx
│   │   ├── ProfilePage.tsx
│   │   └── SettingsPage.tsx
│   ├── services/
│   │   └── apiClient.ts
│   ├── types/
│   │   └── types.ts
│   ├── context/
│   │   └── LanguageContext.tsx
│   ├── lib/
│   │   ├── queryClient.ts
│   │   ├── queryKeys.ts
│   │   └── utils.ts
│   ├── i18n/
│   │   ├── en.ts
│   │   └── de.ts
│   ├── App.tsx
│   └── main.tsx
├── tailwind.config.ts
├── vite.config.ts
├── tsconfig.json
└── package.json
```

---

## Appendix B: Translation Keys Structure

```typescript
// i18n/en.ts (sample structure)
export const en = {
  common: {
    save: 'Save',
    cancel: 'Cancel',
    delete: 'Delete',
    edit: 'Edit',
    loading: 'Loading...',
    error_generic: 'An error occurred. Please try again.',
    permission_denied: 'You do not have permission to perform this action.',
  },
  auth: {
    magic_link: {
      title: 'Sign in with email',
      description: 'Enter your email to receive a secure sign-in link',
      send: 'Send link',
      sent: 'Check your email for the sign-in link',
    },
  },
  company: {
    dashboard: {
      title: 'Company Dashboard',
      total_suppliers: 'Total Suppliers',
      pending_reviews: 'Pending Reviews',
      compliance_rate: 'Compliance Rate',
    },
    suppliers: {
      invite: 'Invite Supplier',
      invite_success: 'Supplier invited successfully',
      classification: {
        critical: 'Critical',
        standard: 'Standard',
      },
    },
  },
  supplier: {
    dashboard: {
      title: 'Supplier Dashboard',
      pending_requests: 'Pending Requests',
      companies: 'Companies',
    },
    checkfix: {
      link_account: 'Link CheckFix Account',
      link_success: 'CheckFix account linked successfully',
    },
  },
  questionnaire: {
    builder: {
      add_question: 'Add Question',
      save: 'Save Questionnaire',
      preview: 'Preview',
    },
    form: {
      submit: 'Submit Answers',
      next_section: 'Next Section',
      previous_section: 'Previous Section',
    },
  },
  validation: {
    email_required: 'Email is required',
    email_invalid: 'Please enter a valid email address',
    question_required: 'Question text is required',
    min_options: 'At least 2 options are required',
  },
};
```

---

*Document generated for nisfix_frontend UI/UX planning. All assumptions should be validated with stakeholders before implementation.*
