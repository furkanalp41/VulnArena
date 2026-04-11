package model

import (
	"time"

	"github.com/google/uuid"
)

const (
	CommunityStatusPending   = "pending"
	CommunityStatusApproved  = "approved"
	CommunityStatusRejected  = "rejected"
	CommunityStatusPublished = "published"
)

// MinXPForCommunitySubmission is the XP threshold required to submit community challenges.
// Corresponds to "Pro Hacker" tier (tier 3).
const MinXPForCommunitySubmission = 600

// CommunityChallenge represents a user-submitted vulnerable code challenge.
type CommunityChallenge struct {
	ID                  uuid.UUID  `json:"id"`
	AuthorID            uuid.UUID  `json:"author_id"`
	Title               string     `json:"title"`
	Description         string     `json:"description"`
	Difficulty          int        `json:"difficulty"`
	LanguageSlug        string     `json:"language_slug"`
	VulnCategorySlug    string     `json:"vuln_category_slug"`
	VulnerableCode      string     `json:"vulnerable_code"`
	TargetVulnerability string     `json:"target_vulnerability"`
	ConceptualFix       string     `json:"conceptual_fix"`
	VulnerableLines     string     `json:"vulnerable_lines"`
	Hints               []string   `json:"hints"`
	Points              int        `json:"points"`
	Status              string     `json:"status"`
	ReviewerID          *uuid.UUID `json:"reviewer_id,omitempty"`
	ReviewerNotes       string     `json:"reviewer_notes,omitempty"`
	ChallengeID         *uuid.UUID `json:"challenge_id,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`

	// Joined fields
	AuthorUsername string `json:"author_username,omitempty"`
}
