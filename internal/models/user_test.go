package models

import (
	"encoding/json"
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestUserRole_MarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		role     UserRole
		expected string
	}{
		{"Admin lowercase", UserRoleAdmin, `"admin"`},
		{"Viewer lowercase", UserRoleViewer, `"viewer"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := json.Marshal(tt.role)
			if err != nil {
				t.Fatalf("MarshalJSON() error = %v", err)
			}
			if string(got) != tt.expected {
				t.Errorf("MarshalJSON() = %v, want %v", string(got), tt.expected)
			}
		})
	}
}

func TestUserRole_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected UserRole
	}{
		{"Admin from lowercase", `"admin"`, UserRoleAdmin},
		{"Viewer from lowercase", `"viewer"`, UserRoleViewer},
		{"Admin from uppercase", `"ADMIN"`, UserRoleAdmin},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got UserRole
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

func TestUserRole_IsValid(t *testing.T) {
	tests := []struct {
		name     string
		role     UserRole
		expected bool
	}{
		{"Admin is valid", UserRoleAdmin, true},
		{"Viewer is valid", UserRoleViewer, true},
		{"Invalid role", UserRole("INVALID"), false},
		{"Empty role", UserRole(""), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.role.IsValid(); got != tt.expected {
				t.Errorf("IsValid() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestUser_BeforeCreate(t *testing.T) {
	user := &User{
		Email: "test@example.com",
		Role:  UserRoleAdmin,
	}

	user.BeforeCreate()

	if user.ID.IsZero() {
		t.Error("ID should be set")
	}
	if user.CreatedAt.IsZero() {
		t.Error("CreatedAt should be set")
	}
	if user.UpdatedAt.IsZero() {
		t.Error("UpdatedAt should be set")
	}
	if !user.IsActive {
		t.Error("IsActive should be true by default")
	}
	if user.Language != "en" {
		t.Errorf("Language = %v, want en", user.Language)
	}
}

func TestUser_BeforeCreate_PreservesExistingID(t *testing.T) {
	existingID := primitive.NewObjectID()
	user := &User{
		ID:    existingID,
		Email: "test@example.com",
	}

	user.BeforeCreate()

	if user.ID != existingID {
		t.Error("BeforeCreate should preserve existing ID")
	}
}

func TestUser_BeforeUpdate(t *testing.T) {
	user := &User{Email: "test@example.com"}
	user.BeforeCreate()
	originalUpdatedAt := user.UpdatedAt

	time.Sleep(1 * time.Millisecond)
	user.BeforeUpdate()

	if !user.UpdatedAt.After(originalUpdatedAt) {
		t.Error("UpdatedAt should be updated")
	}
}

func TestUser_SoftDelete(t *testing.T) {
	user := &User{Email: "test@example.com"}
	user.BeforeCreate()

	if user.IsDeleted() {
		t.Error("User should not be deleted initially")
	}
	if !user.IsActive {
		t.Error("User should be active initially")
	}

	user.SoftDelete()

	if !user.IsDeleted() {
		t.Error("User should be deleted after SoftDelete")
	}
	if user.DeletedAt == nil {
		t.Error("DeletedAt should be set")
	}
	if user.IsActive {
		t.Error("User should be inactive after SoftDelete")
	}
}

func TestUser_IsAdmin(t *testing.T) {
	tests := []struct {
		name     string
		role     UserRole
		expected bool
	}{
		{"Admin returns true", UserRoleAdmin, true},
		{"Viewer returns false", UserRoleViewer, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user := &User{Role: tt.role}
			if got := user.IsAdmin(); got != tt.expected {
				t.Errorf("IsAdmin() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestUser_IsViewer(t *testing.T) {
	tests := []struct {
		name     string
		role     UserRole
		expected bool
	}{
		{"Viewer returns true", UserRoleViewer, true},
		{"Admin returns false", UserRoleAdmin, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user := &User{Role: tt.role}
			if got := user.IsViewer(); got != tt.expected {
				t.Errorf("IsViewer() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestUser_UpdateLastLogin(t *testing.T) {
	user := &User{Email: "test@example.com"}
	user.BeforeCreate()

	if user.LastLoginAt != nil {
		t.Error("LastLoginAt should be nil initially")
	}

	user.UpdateLastLogin()

	if user.LastLoginAt == nil {
		t.Error("LastLoginAt should be set after UpdateLastLogin")
	}
}

func TestUser_CanManageOrganization(t *testing.T) {
	tests := []struct {
		name      string
		role      UserRole
		isActive  bool
		isDeleted bool
		expected  bool
	}{
		{"Active admin can manage", UserRoleAdmin, true, false, true},
		{"Inactive admin cannot manage", UserRoleAdmin, false, false, false},
		{"Deleted admin cannot manage", UserRoleAdmin, true, true, false},
		{"Active viewer cannot manage", UserRoleViewer, true, false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user := &User{
				Role:     tt.role,
				IsActive: tt.isActive,
			}
			if tt.isDeleted {
				now := time.Now()
				user.DeletedAt = &now
			}
			if got := user.CanManageOrganization(); got != tt.expected {
				t.Errorf("CanManageOrganization() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestUser_CanInviteSuppliers(t *testing.T) {
	tests := []struct {
		name     string
		role     UserRole
		isActive bool
		expected bool
	}{
		{"Active admin can invite", UserRoleAdmin, true, true},
		{"Inactive admin cannot invite", UserRoleAdmin, false, false},
		{"Active viewer cannot invite", UserRoleViewer, true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			user := &User{
				Role:     tt.role,
				IsActive: tt.isActive,
			}
			if got := user.CanInviteSuppliers(); got != tt.expected {
				t.Errorf("CanInviteSuppliers() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestUser_CollectionName(t *testing.T) {
	user := User{}
	if got := user.CollectionName(); got != "users" {
		t.Errorf("CollectionName() = %v, want users", got)
	}
}
