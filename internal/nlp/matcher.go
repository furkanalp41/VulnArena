// Package nlp provides the semantic evaluation engine for scoring
// user-submitted vulnerability assessments against expected answers.
//
// Architecture: The SemanticMatcher interface defines the contract.
// Implementations can range from keyword-based scoring (v1) to
// full LLM-powered evaluation (v2+). The interface is designed
// so swapping the backend requires zero changes to calling code.
package nlp

import "context"

// EvaluationRequest contains everything needed to score a user's answer.
type EvaluationRequest struct {
	// UserAnswer is the free-text explanation submitted by the user.
	UserAnswer string

	// TargetVulnerability is the ground-truth vulnerability description
	// that the challenge author defined.
	TargetVulnerability string

	// ConceptualFix is the expected remediation approach.
	ConceptualFix string

	// Language is the programming language of the challenge code (e.g., "go", "c").
	Language string

	// Difficulty is 1-10, used to adjust scoring strictness.
	Difficulty int

	// VulnerableLines are the ground-truth line numbers where the flaw exists.
	// Empty for legacy challenges that don't specify lines.
	VulnerableLines []int

	// UserTargetLines are the line numbers the user identified as vulnerable.
	UserTargetLines []int
}

// EvaluationResult is the scored output from the matcher.
type EvaluationResult struct {
	// VulnScore is 0-100 for vulnerability identification accuracy.
	VulnScore float64 `json:"vuln_score"`

	// FixScore is 0-100 for remediation/fix explanation accuracy.
	FixScore float64 `json:"fix_score"`

	// LineAccuracy is 0-100 for how accurately the user pinpointed the vulnerable lines.
	LineAccuracy float64 `json:"line_accuracy"`

	// OverallScore is the weighted combination (0-100).
	OverallScore float64 `json:"overall_score"`

	// Passed indicates whether the score meets the passing threshold.
	Passed bool `json:"passed"`

	// VulnIdentified indicates the user found the core vulnerability.
	VulnIdentified bool `json:"vuln_identified"`

	// FixIdentified indicates the user proposed a valid fix.
	FixIdentified bool `json:"fix_identified"`

	// MatchedVulnTerms are the security concepts detected in the answer
	// that matched the vulnerability description.
	MatchedVulnTerms []string `json:"matched_vuln_terms"`

	// MatchedFixTerms are the remediation concepts detected.
	MatchedFixTerms []string `json:"matched_fix_terms"`

	// TerminalLog is a sequence of human-readable processing steps
	// shown to the user in the terminal-style output panel.
	TerminalLog []string `json:"terminal_log"`
}

// SemanticMatcher is the interface all evaluation backends must implement.
// This allows hot-swapping between keyword scoring, embedding similarity,
// and full LLM evaluation without changing any calling code.
type SemanticMatcher interface {
	Evaluate(ctx context.Context, req EvaluationRequest) (*EvaluationResult, error)
	Name() string
}
