package models

import (
	"encoding/json"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestOrganizationType_MarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		ot       OrganizationType
		expected string
	}{
		{"Company lowercase", OrganizationTypeCompany, `"company"`},
		{"Supplier lowercase", OrganizationTypeSupplier, `"supplier"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := json.Marshal(tt.ot)
			if err != nil {
				t.Fatalf("MarshalJSON() error = %v", err)
			}
			if string(got) != tt.expected {
				t.Errorf("MarshalJSON() = %v, want %v", string(got), tt.expected)
			}
		})
	}
}

func TestOrganizationType_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected OrganizationType
	}{
		{"Company from lowercase", `"company"`, OrganizationTypeCompany},
		{"Supplier from lowercase", `"supplier"`, OrganizationTypeSupplier},
		{"Company from uppercase", `"COMPANY"`, OrganizationTypeCompany},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got OrganizationType
			err := json.Unmarshal([]byte(tt.input), &got)
			if err != nil {
				t.Fatalf("UnmarshalJSON() error = %v", err)
			}
			if got != tt.expected {
				t.Errorf("UnmarshalJSON() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestOrganizationType_IsValid(t *testing.T) {
	tests := []struct {
		name     string
		ot       OrganizationType
		expected bool
	}{
		{"Company is valid", OrganizationTypeCompany, true},
		{"Supplier is valid", OrganizationTypeSupplier, true},
		{"Invalid type", OrganizationType("INVALID"), false},
		{"Empty type", OrganizationType(""), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.ot.IsValid(); got != tt.expected {
				t.Errorf("IsValid() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestDefaultOrganizationSettings(t *testing.T) {
	settings := DefaultOrganizationSettings()

	if settings.DefaultDueDays != 30 {
		t.Errorf("DefaultDueDays = %v, want 30", settings.DefaultDueDays)
	}
	if settings.RequireCheckFix != false {
		t.Errorf("RequireCheckFix = %v, want false", settings.RequireCheckFix)
	}
	if settings.MinCheckFixGrade != "C" {
		t.Errorf("MinCheckFixGrade = %v, want C", settings.MinCheckFixGrade)
	}
	if settings.DefaultLanguage != "en" {
		t.Errorf("DefaultLanguage = %v, want en", settings.DefaultLanguage)
	}
	if settings.NotificationsEnabled != true {
		t.Errorf("NotificationsEnabled = %v, want true", settings.NotificationsEnabled)
	}
}

func TestOrganization_BeforeCreate(t *testing.T) {
	org := &Organization{
		Name: "Test Org",
		Type: OrganizationTypeCompany,
	}

	org.BeforeCreate()

	if org.ID.IsZero() {
		t.Error("ID should be set")
	}
	if org.CreatedAt.IsZero() {
		t.Error("CreatedAt should be set")
	}
	if org.UpdatedAt.IsZero() {
		t.Error("UpdatedAt should be set")
	}
	if org.Settings.DefaultLanguage != "en" {
		t.Errorf("Settings.DefaultLanguage = %v, want en", org.Settings.DefaultLanguage)
	}
}

func TestOrganization_BeforeCreate_PreservesExistingID(t *testing.T) {
	existingID := primitive.NewObjectID()
	org := &Organization{
		ID:   existingID,
		Name: "Test Org",
	}

	org.BeforeCreate()

	if org.ID != existingID {
		t.Error("BeforeCreate should preserve existing ID")
	}
}

func TestOrganization_BeforeUpdate(t *testing.T) {
	org := &Organization{
		Name: "Test Org",
	}
	org.BeforeCreate()
	originalUpdatedAt := org.UpdatedAt

	time.Sleep(1 * time.Millisecond)
	org.BeforeUpdate()

	if !org.UpdatedAt.After(originalUpdatedAt) {
		t.Error("UpdatedAt should be updated")
	}
}

func TestOrganization_SoftDelete(t *testing.T) {
	org := &Organization{
		Name: "Test Org",
	}
	org.BeforeCreate()

	if org.IsDeleted() {
		t.Error("Organization should not be deleted initially")
	}

	org.SoftDelete()

	if !org.IsDeleted() {
		t.Error("Organization should be deleted after SoftDelete")
	}
	if org.DeletedAt == nil {
		t.Error("DeletedAt should be set")
	}
}

func TestOrganization_IsCompany(t *testing.T) {
	tests := []struct {
		name     string
		orgType  OrganizationType
		expected bool
	}{
		{"Company returns true", OrganizationTypeCompany, true},
		{"Supplier returns false", OrganizationTypeSupplier, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			org := &Organization{Type: tt.orgType}
			if got := org.IsCompany(); got != tt.expected {
				t.Errorf("IsCompany() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestOrganization_IsSupplier(t *testing.T) {
	tests := []struct {
		name     string
		orgType  OrganizationType
		expected bool
	}{
		{"Supplier returns true", OrganizationTypeSupplier, true},
		{"Company returns false", OrganizationTypeCompany, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			org := &Organization{Type: tt.orgType}
			if got := org.IsSupplier(); got != tt.expected {
				t.Errorf("IsSupplier() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestOrganization_HasCheckFixLinked(t *testing.T) {
	now := time.Now()
	tests := []struct {
		name            string
		checkFixID      string
		checkFixLinked  *time.Time
		expected        bool
	}{
		{"Both set", "account123", &now, true},
		{"ID only", "account123", nil, false},
		{"Time only", "", &now, false},
		{"Neither set", "", nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			org := &Organization{
				CheckFixAccountID: tt.checkFixID,
				CheckFixLinkedAt:  tt.checkFixLinked,
			}
			if got := org.HasCheckFixLinked(); got != tt.expected {
				t.Errorf("HasCheckFixLinked() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestOrganization_CollectionName(t *testing.T) {
	org := Organization{}
	if got := org.CollectionName(); got != "organizations" {
		t.Errorf("CollectionName() = %v, want organizations", got)
	}
}
