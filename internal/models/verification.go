package models

import (
	"encoding/json"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// CheckFixGrade represents a CheckFix security grade
// #IMPLEMENTATION_DECISION: Grades are A, B, C, D, F (no E grade per unified blueprint)
type CheckFixGrade string

const (
	CheckFixGradeA CheckFixGrade = "A"
	CheckFixGradeB CheckFixGrade = "B"
	CheckFixGradeC CheckFixGrade = "C"
	CheckFixGradeD CheckFixGrade = "D"
	CheckFixGradeF CheckFixGrade = "F"
)

// MarshalJSON converts CheckFixGrade to string for JSON serialization
func (g CheckFixGrade) MarshalJSON() ([]byte, error) {
	return json.Marshal(string(g))
}

// UnmarshalJSON converts JSON string to CheckFixGrade
func (g *CheckFixGrade) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	*g = CheckFixGrade(strings.ToUpper(s))
	return nil
}

// IsValid checks if the CheckFixGrade is a valid value
func (g CheckFixGrade) IsValid() bool {
	switch g {
	case CheckFixGradeA, CheckFixGradeB, CheckFixGradeC, CheckFixGradeD, CheckFixGradeF:
		return true
	}
	return false
}

// Score returns a numeric score for the grade (higher is better)
func (g CheckFixGrade) Score() int {
	switch g {
	case CheckFixGradeA:
		return 5
	case CheckFixGradeB:
		return 4
	case CheckFixGradeC:
		return 3
	case CheckFixGradeD:
		return 2
	case CheckFixGradeF:
		return 1
	}
	return 0
}

// MeetsMinimum returns true if this grade meets or exceeds the minimum grade
func (g CheckFixGrade) MeetsMinimum(minimum CheckFixGrade) bool {
	return g.Score() >= minimum.Score()
}

// IsPassing returns true if this is a passing grade (C or better by default)
func (g CheckFixGrade) IsPassing() bool {
	return g.Score() >= CheckFixGradeC.Score()
}

// CategoryGrade represents a grade for a specific security category
type CategoryGrade struct {
	Category string `bson:"category" json:"category"`
	Grade    string `bson:"grade" json:"grade"`
	Score    int    `bson:"score" json:"score"`
}

// CheckFixVerification stores verified CheckFix report data for a supplier domain
// #DATA_ASSUMPTION: Grades are A, B, C, D, F (no E grade)
// #DATA_ASSUMPTION: Report hash used to verify authenticity via CheckFix API
// #CARDINALITY_ASSUMPTION: SupplierResponse 1:1 CheckFixVerification
// #CACHE_ASSUMPTION: Verifications cached with ExpiresAt, re-verified when expired
type CheckFixVerification struct {
	ID         primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	ResponseID primitive.ObjectID `bson:"response_id" json:"response_id"`
	SupplierID primitive.ObjectID `bson:"supplier_id" json:"supplier_id"`

	// Domain verification
	Domain         string `bson:"domain" json:"domain"`
	VerifiedDomain string `bson:"verified_domain" json:"verified_domain"`
	DomainMatch    bool   `bson:"domain_match" json:"domain_match"`

	// Report details
	ReportHash string    `bson:"report_hash" json:"report_hash"`
	ReportDate time.Time `bson:"report_date" json:"report_date"`

	// Grades and scores
	OverallGrade CheckFixGrade `bson:"overall_grade" json:"overall_grade"`
	OverallScore int           `bson:"overall_score" json:"overall_score"`

	// Category grades
	CategoryGrades []CategoryGrade `bson:"category_grades" json:"category_grades"`

	// Finding counts
	CriticalFindings int `bson:"critical_findings" json:"critical_findings"`
	HighFindings     int `bson:"high_findings" json:"high_findings"`
	MediumFindings   int `bson:"medium_findings" json:"medium_findings"`
	LowFindings      int `bson:"low_findings" json:"low_findings"`

	// Verification metadata
	VerifiedAt        time.Time `bson:"verified_at" json:"verified_at"`
	VerificationValid bool      `bson:"verification_valid" json:"verification_valid"`
	ExpiresAt         time.Time `bson:"expires_at" json:"expires_at"`

	// Audit fields
	CreatedAt time.Time `bson:"created_at" json:"created_at"`
	UpdatedAt time.Time `bson:"updated_at" json:"updated_at"`
}

// CollectionName returns the MongoDB collection name for CheckFix verifications
func (CheckFixVerification) CollectionName() string {
	return "checkfix_verifications"
}

// VerificationValidityDuration is the duration a verification is valid (30 days)
const VerificationValidityDuration = 30 * 24 * time.Hour

// BeforeCreate sets default values before inserting a new verification
func (v *CheckFixVerification) BeforeCreate() {
	now := time.Now().UTC()
	if v.ID.IsZero() {
		v.ID = primitive.NewObjectID()
	}
	v.CreatedAt = now
	v.UpdatedAt = now
	v.VerifiedAt = now

	// Set expiry if not already set
	if v.ExpiresAt.IsZero() {
		v.ExpiresAt = now.Add(VerificationValidityDuration)
	}

	if v.CategoryGrades == nil {
		v.CategoryGrades = []CategoryGrade{}
	}
}

// BeforeUpdate sets the UpdatedAt timestamp
func (v *CheckFixVerification) BeforeUpdate() {
	v.UpdatedAt = time.Now().UTC()
}

// IsExpired returns true if the verification has expired
func (v *CheckFixVerification) IsExpired() bool {
	return time.Now().UTC().After(v.ExpiresAt)
}

// IsValid returns true if the verification is still valid
func (v *CheckFixVerification) IsValid() bool {
	return v.VerificationValid && !v.IsExpired() && v.DomainMatch
}

// MeetsMinimumGrade returns true if the overall grade meets the minimum
func (v *CheckFixVerification) MeetsMinimumGrade(minimum CheckFixGrade) bool {
	return v.OverallGrade.MeetsMinimum(minimum)
}

// TotalFindings returns the total number of findings
func (v *CheckFixVerification) TotalFindings() int {
	return v.CriticalFindings + v.HighFindings + v.MediumFindings + v.LowFindings
}

// HasCriticalFindings returns true if there are critical findings
func (v *CheckFixVerification) HasCriticalFindings() bool {
	return v.CriticalFindings > 0
}

// HasHighFindings returns true if there are high severity findings
func (v *CheckFixVerification) HasHighFindings() bool {
	return v.HighFindings > 0
}

// GetCategoryGrade returns the grade for a specific category
func (v *CheckFixVerification) GetCategoryGrade(category string) *CategoryGrade {
	for i := range v.CategoryGrades {
		if v.CategoryGrades[i].Category == category {
			return &v.CategoryGrades[i]
		}
	}
	return nil
}

// AddCategoryGrade adds a category grade
func (v *CheckFixVerification) AddCategoryGrade(grade CategoryGrade) {
	v.CategoryGrades = append(v.CategoryGrades, grade)
	v.UpdatedAt = time.Now().UTC()
}

// DaysUntilExpiry returns the number of days until expiry
func (v *CheckFixVerification) DaysUntilExpiry() int {
	duration := time.Until(v.ExpiresAt)
	return int(duration.Hours() / 24)
}

// NeedsRefresh returns true if the verification is expired or about to expire
func (v *CheckFixVerification) NeedsRefresh(daysBeforeExpiry int) bool {
	return v.IsExpired() || v.DaysUntilExpiry() <= daysBeforeExpiry
}

// Refresh extends the verification expiry
func (v *CheckFixVerification) Refresh() {
	now := time.Now().UTC()
	v.VerifiedAt = now
	v.ExpiresAt = now.Add(VerificationValidityDuration)
	v.UpdatedAt = now
}

// ReportAgeDays returns the age of the report in days
func (v *CheckFixVerification) ReportAgeDays() int {
	duration := time.Since(v.ReportDate)
	return int(duration.Hours() / 24)
}

// IsReportTooOld returns true if the report exceeds the maximum age
func (v *CheckFixVerification) IsReportTooOld(maxAgeDays int) bool {
	return v.ReportAgeDays() > maxAgeDays
}

// PassesRequirement checks if the verification passes a CheckFix requirement
// #BUSINESS_RULE: CheckFix verification requires domain match with supplier organization
func (v *CheckFixVerification) PassesRequirement(minimumGrade CheckFixGrade, maxReportAgeDays int) bool {
	if !v.IsValid() {
		return false
	}
	if !v.MeetsMinimumGrade(minimumGrade) {
		return false
	}
	if maxReportAgeDays > 0 && v.IsReportTooOld(maxReportAgeDays) {
		return false
	}
	return true
}
