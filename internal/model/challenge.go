package model

import (
	"time"

	"github.com/google/uuid"
)

type VulnCategory struct {
	ID          int    `json:"id"`
	Slug        string `json:"slug"`
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	OWASPRef    string `json:"owasp_ref,omitempty"`
}

type Language struct {
	ID   int    `json:"id"`
	Slug string `json:"slug"`
	Name string `json:"name"`
}

type Challenge struct {
	ID                  uuid.UUID  `json:"id"`
	Title               string     `json:"title"`
	Slug                string     `json:"slug"`
	Description         string     `json:"description"`
	Difficulty          int        `json:"difficulty"`
	LanguageID          int        `json:"-"`
	VulnCategoryID      int        `json:"-"`
	VulnerableCode      string     `json:"vulnerable_code"`
	TargetVulnerability string     `json:"-"`    // Never sent to client
	ConceptualFix       string     `json:"-"`    // Never sent to client
	VulnerableLines     []int      `json:"-"`    // Ground-truth line numbers, never sent to client
	CVEReference        *string    `json:"cve_reference,omitempty"`
	Hints               []string   `json:"hints,omitempty"`
	Points              int        `json:"points"`
	LineCount           int        `json:"line_count"`
	IsPublished         bool       `json:"-"`
	FirstBloodUserID    *uuid.UUID `json:"first_blood_user_id,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`

	// Joined fields
	Language     *Language     `json:"language,omitempty"`
	VulnCategory *VulnCategory `json:"vuln_category,omitempty"`
}

// ChallengeListItem is a lighter representation for list endpoints.
type ChallengeListItem struct {
	ID           uuid.UUID     `json:"id"`
	Title        string        `json:"title"`
	Slug         string        `json:"slug"`
	Description  string        `json:"description"`
	Difficulty   int           `json:"difficulty"`
	Points       int           `json:"points"`
	LineCount    int           `json:"line_count"`
	Language     *Language     `json:"language"`
	VulnCategory *VulnCategory `json:"vuln_category"`
}

type ChallengeFilter struct {
	LanguageSlug    string
	VulnCategorySlug string
	DifficultyMin   int
	DifficultyMax   int
	Page            int
	Limit           int
}
