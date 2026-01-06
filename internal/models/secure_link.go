package models

import (
	"encoding/json"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

// SecureLinkType represents the type of secure link
// #IMPLEMENTATION_DECISION: AUTH for magic links, INVITATION for supplier invites
type SecureLinkType string

const (
	SecureLinkTypeAuth       SecureLinkType = "AUTH"
	SecureLinkTypeInvitation SecureLinkType = "INVITATION"
)

// MarshalJSON converts SecureLinkType to lowercase for JSON serialization
func (slt SecureLinkType) MarshalJSON() ([]byte, error) {
	return json.Marshal(strings.ToLower(string(slt)))
}

// UnmarshalJSON converts lowercase JSON to SecureLinkType
func (slt *SecureLinkType) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	*slt = SecureLinkType(strings.ToUpper(s))
	return nil
}

// IsValid checks if the SecureLinkType is a valid value
func (slt SecureLinkType) IsValid() bool {
	switch slt {
	case SecureLinkTypeAuth, SecureLinkTypeInvitation:
		return true
	}
	return false
}

// SecureLink represents a magic link token for passwordless authentication
// #DATA_ASSUMPTION: SecureIdentifier is 64-character cryptographically random string
// #DATA_ASSUMPTION: Auth links expire in 15 minutes, invitation links in 7 days
// #INDEX_STRATEGY: TTL index on expires_at for automatic cleanup
type SecureLink struct {
	ID               primitive.ObjectID  `bson:"_id,omitempty" json:"id"`
	SecureIdentifier string              `bson:"secure_identifier" json:"secure_identifier"`
	Type             SecureLinkType      `bson:"type" json:"type"`

	// Target
	Email          string              `bson:"email" json:"email"`
	UserID         *primitive.ObjectID `bson:"user_id,omitempty" json:"user_id,omitempty"`
	RelationshipID *primitive.ObjectID `bson:"relationship_id,omitempty" json:"relationship_id,omitempty"`

	// Validity
	ExpiresAt time.Time  `bson:"expires_at" json:"expires_at"`
	UsedAt    *time.Time `bson:"used_at,omitempty" json:"used_at,omitempty"`
	IsValid   bool       `bson:"is_valid" json:"is_valid"`

	// Security tracking
	IPAddress string `bson:"ip_address,omitempty" json:"ip_address,omitempty"`
	UserAgent string `bson:"user_agent,omitempty" json:"user_agent,omitempty"`

	// Audit field (no update/delete - links are ephemeral)
	CreatedAt time.Time `bson:"created_at" json:"created_at"`
}

// CollectionName returns the MongoDB collection name for secure links
func (SecureLink) CollectionName() string {
	return "secure_links"
}

// AuthLinkExpiryDuration is the expiry duration for auth magic links (15 minutes)
const AuthLinkExpiryDuration = 15 * time.Minute

// InvitationLinkExpiryDuration is the expiry duration for invitation links (7 days)
const InvitationLinkExpiryDuration = 7 * 24 * time.Hour

// BeforeCreate sets default values before inserting a new secure link
func (sl *SecureLink) BeforeCreate() {
	now := time.Now().UTC()
	if sl.ID.IsZero() {
		sl.ID = primitive.NewObjectID()
	}
	sl.CreatedAt = now
	sl.IsValid = true

	// Set expiry based on type if not already set
	if sl.ExpiresAt.IsZero() {
		switch sl.Type {
		case SecureLinkTypeAuth:
			sl.ExpiresAt = now.Add(AuthLinkExpiryDuration)
		case SecureLinkTypeInvitation:
			sl.ExpiresAt = now.Add(InvitationLinkExpiryDuration)
		default:
			// Default to auth link expiry
			sl.ExpiresAt = now.Add(AuthLinkExpiryDuration)
		}
	}
}

// IsExpired returns true if the secure link has expired
func (sl *SecureLink) IsExpired() bool {
	return time.Now().UTC().After(sl.ExpiresAt)
}

// IsUsed returns true if the secure link has been used
func (sl *SecureLink) IsUsed() bool {
	return sl.UsedAt != nil
}

// CanBeUsed returns true if the secure link can still be used
func (sl *SecureLink) CanBeUsed() bool {
	return sl.IsValid && !sl.IsExpired() && !sl.IsUsed()
}

// MarkAsUsed marks the secure link as used
func (sl *SecureLink) MarkAsUsed() {
	now := time.Now().UTC()
	sl.UsedAt = &now
	sl.IsValid = false
}

// Invalidate marks the secure link as invalid without marking as used
func (sl *SecureLink) Invalidate() {
	sl.IsValid = false
}

// IsAuthLink returns true if this is an authentication magic link
func (sl *SecureLink) IsAuthLink() bool {
	return sl.Type == SecureLinkTypeAuth
}

// IsInvitationLink returns true if this is a supplier invitation link
func (sl *SecureLink) IsInvitationLink() bool {
	return sl.Type == SecureLinkTypeInvitation
}

// TimeUntilExpiry returns the duration until the link expires
func (sl *SecureLink) TimeUntilExpiry() time.Duration {
	return time.Until(sl.ExpiresAt)
}
