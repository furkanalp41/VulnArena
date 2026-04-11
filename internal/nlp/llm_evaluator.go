package nlp

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"
)

// PassThreshold is the minimum overall score (0-100) to pass a challenge.
const PassThreshold = 60.0

// LLMEvaluator is the v1 semantic matcher using weighted keyword/concept scoring.
//
// Architecture note: This struct implements SemanticMatcher. When we integrate
// a real LLM (Anthropic/OpenAI), we create a new struct (e.g., AnthropicEvaluator)
// implementing the same interface. The service layer selects which evaluator to
// use via config, with zero changes to handler or repository code.
//
// The keyword approach is intentionally generous — it rewards users who
// demonstrate conceptual understanding even if their phrasing differs
// from the reference answer.
type LLMEvaluator struct {
	// simulateLatency adds realistic delay to mimic LLM processing time.
	// Set to 0 for tests.
	simulateLatency time.Duration
}

// NewLLMEvaluator creates a keyword-based evaluator.
// Pass 0 for latency in tests, or ~800ms for production feel.
func NewLLMEvaluator(simulateLatency time.Duration) *LLMEvaluator {
	return &LLMEvaluator{simulateLatency: simulateLatency}
}

func (e *LLMEvaluator) Name() string {
	return "llm-evaluator-v1-keyword"
}

func (e *LLMEvaluator) Evaluate(ctx context.Context, req EvaluationRequest) (*EvaluationResult, error) {
	log := []string{}
	logf := func(format string, args ...any) {
		log = append(log, fmt.Sprintf(format, args...))
	}

	logf("> Initializing semantic analysis engine [%s]...", e.Name())
	logf("> Target language: %s | Difficulty: %d/10", req.Language, req.Difficulty)
	logf("> Parsing user submission (%d characters)...", len(req.UserAnswer))

	// Simulate processing time for realistic UX
	if e.simulateLatency > 0 {
		select {
		case <-time.After(e.simulateLatency):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	// Build relevant term sets from the challenge's ground truth
	vulnTerms, fixTerms := buildRelevantTerms(req.TargetVulnerability, req.ConceptualFix)

	logf("> Loaded %d vulnerability concept signatures", len(vulnTerms))
	logf("> Loaded %d remediation concept signatures", len(fixTerms))
	logf("> Running concept extraction on submission...")

	// Extract matched concepts from user's answer
	matchedVuln := extractMatchedTerms(req.UserAnswer, vulnTerms)
	matchedFix := extractMatchedTerms(req.UserAnswer, fixTerms)

	logf("> Vulnerability concepts matched: %d/%d", len(matchedVuln), len(vulnTerms))
	for _, term := range matchedVuln {
		logf("  [+] Detected: %s", term)
	}

	logf("> Remediation concepts matched: %d/%d", len(matchedFix), len(fixTerms))
	for _, term := range matchedFix {
		logf("  [+] Detected: %s", term)
	}

	// --- Line accuracy scoring ---
	lineAccuracy := 0.0
	hasLineTargeting := len(req.VulnerableLines) > 0
	if hasLineTargeting && len(req.UserTargetLines) > 0 {
		logf("> Line targeting: user flagged %d line(s), ground truth has %d", len(req.UserTargetLines), len(req.VulnerableLines))
		hits := 0
		truthSet := make(map[int]bool, len(req.VulnerableLines))
		for _, l := range req.VulnerableLines {
			truthSet[l] = true
		}
		for _, ul := range req.UserTargetLines {
			// Allow ±2 line tolerance for minor off-by-one
			if truthSet[ul] || truthSet[ul-1] || truthSet[ul+1] || truthSet[ul-2] || truthSet[ul+2] {
				hits++
				logf("  [+] Line %d — HIT (matches vulnerable region)", ul)
			} else {
				logf("  [-] Line %d — MISS", ul)
			}
		}
		precision := float64(hits) / float64(len(req.UserTargetLines))
		recall := float64(hits) / float64(len(req.VulnerableLines))
		if precision+recall > 0 {
			lineAccuracy = 2 * precision * recall / (precision + recall) * 100 // F1 score
		}
		logf("> Line accuracy (F1): %.1f%%", lineAccuracy)
	} else if hasLineTargeting {
		logf("> Line targeting: no lines submitted by user (0%%)")
	}

	// --- Score calculation ---
	vulnRatio := 0.0
	if len(vulnTerms) > 0 {
		vulnRatio = float64(len(matchedVuln)) / float64(len(vulnTerms))
	}

	fixRatio := 0.0
	if len(fixTerms) > 0 {
		fixRatio = float64(len(matchedFix)) / float64(len(fixTerms))
	}

	wordCount := len(strings.Fields(req.UserAnswer))
	detailBonus := math.Min(float64(wordCount)/100.0, 0.15)

	logf("> Submission detail factor: %.0f words (bonus: +%.1f%%)", float64(wordCount), detailBonus*100)

	difficultyMultiplier := 1.0 + float64(req.Difficulty-1)*0.02

	vulnScore := math.Min((vulnRatio+detailBonus)*difficultyMultiplier*100, 100)
	fixScore := math.Min((fixRatio+detailBonus)*difficultyMultiplier*100, 100)

	// Weighted combination: with line targeting, lines contribute 20%
	var overallScore float64
	if hasLineTargeting {
		overallScore = vulnScore*0.40 + fixScore*0.30 + lineAccuracy*0.30
	} else {
		overallScore = vulnScore*0.60 + fixScore*0.40
	}
	overallScore = math.Round(overallScore*100) / 100

	vulnIdentified := vulnScore >= 40
	fixIdentified := fixScore >= 40
	passed := overallScore >= PassThreshold

	logf("> ─────────────────────────────────────")
	logf("> Vulnerability analysis:  %.1f/100 %s", vulnScore, statusIcon(vulnIdentified))
	logf("> Remediation analysis:    %.1f/100 %s", fixScore, statusIcon(fixIdentified))
	if hasLineTargeting {
		logf("> Line accuracy:           %.1f/100", lineAccuracy)
	}
	logf("> Overall score:           %.1f/100", overallScore)
	logf("> ─────────────────────────────────────")

	if passed {
		logf("> STATUS: PASSED — Vulnerability assessment accepted.")
		logf("> Points awarded to operator profile.")
	} else {
		logf("> STATUS: INSUFFICIENT — Refine your analysis and resubmit.")
		if !vulnIdentified {
			logf("> HINT: Focus on identifying the core vulnerability class.")
		}
		if !fixIdentified {
			logf("> HINT: Describe a concrete remediation approach.")
		}
		if hasLineTargeting && lineAccuracy < 40 {
			logf("> HINT: Review the source code carefully and pinpoint the exact vulnerable lines.")
		}
	}

	return &EvaluationResult{
		VulnScore:        math.Round(vulnScore*100) / 100,
		FixScore:         math.Round(fixScore*100) / 100,
		LineAccuracy:     math.Round(lineAccuracy*100) / 100,
		OverallScore:     overallScore,
		Passed:           passed,
		VulnIdentified:   vulnIdentified,
		FixIdentified:    fixIdentified,
		MatchedVulnTerms: matchedVuln,
		MatchedFixTerms:  matchedFix,
		TerminalLog:      log,
	}, nil
}

func statusIcon(ok bool) string {
	if ok {
		return "[IDENTIFIED]"
	}
	return "[INSUFFICIENT]"
}
