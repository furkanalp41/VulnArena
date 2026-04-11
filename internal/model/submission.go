package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type Submission struct {
	ID           uuid.UUID       `json:"id"`
	UserID       uuid.UUID       `json:"user_id"`
	ChallengeID  uuid.UUID       `json:"challenge_id"`
	AnswerText   string          `json:"answer_text"`
	TargetLines  []int           `json:"target_lines,omitempty"`
	Score        float64         `json:"score"`
	IsCorrect    bool            `json:"is_correct"`
	IsFirstBlood bool            `json:"is_first_blood"`
	Feedback     json.RawMessage `json:"feedback"`
	TimeSpentSec *int            `json:"time_spent_sec,omitempty"`
	CreatedAt    time.Time       `json:"created_at"`
}

type EvaluationFeedback struct {
	VulnIdentified     bool     `json:"vuln_identified"`
	VulnScore          float64  `json:"vuln_score"`
	FixIdentified      bool     `json:"fix_identified"`
	FixScore           float64  `json:"fix_score"`
	LineAccuracy       float64  `json:"line_accuracy"`
	OverallScore       float64  `json:"overall_score"`
	Passed             bool     `json:"passed"`
	TerminalLog        []string `json:"terminal_log"`
	MatchedVulnTerms   []string `json:"matched_vuln_terms,omitempty"`
	MatchedFixTerms    []string `json:"matched_fix_terms,omitempty"`
}

type UserChallengeProgress struct {
	UserID        uuid.UUID  `json:"user_id"`
	ChallengeID   uuid.UUID  `json:"challenge_id"`
	Status        string     `json:"status"`
	BestScore     float64    `json:"best_score"`
	AttemptCount  int        `json:"attempt_count"`
	FirstSolvedAt *time.Time `json:"first_solved_at,omitempty"`
	LastAttempted *time.Time `json:"last_attempted,omitempty"`
}

const (
	ProgressNotStarted = "not_started"
	ProgressAttempted  = "attempted"
	ProgressSolved     = "solved"
)
