package nlp

import (
	"context"
	"math"
	"testing"
)

// approxEq returns true when a and b agree to two decimal places. The
// evaluator rounds scores to two decimals before returning, so test
// assertions don't need finer tolerance.
func approxEq(a, b float64) bool {
	return math.Abs(a-b) < 0.01
}

func runEval(t *testing.T, req EvaluationRequest) *EvaluationResult {
	t.Helper()
	e := NewLLMEvaluator(0)
	res, err := e.Evaluate(context.Background(), req)
	if err != nil {
		t.Fatalf("Evaluate returned error: %v", err)
	}
	return res
}

// ── Line accuracy (F1) tests ──

func TestLineF1_ExactMatch(t *testing.T) {
	res := runEval(t, EvaluationRequest{
		UserAnswer:          "sql injection from string concatenation; use parameterized queries",
		TargetVulnerability: "sql injection via string concatenation",
		ConceptualFix:       "use parameterized queries",
		VulnerableLines:     []int{10, 11, 12},
		UserTargetLines:     []int{10, 11, 12},
		Difficulty:          5,
	})
	if !approxEq(res.LineAccuracy, 100.0) {
		t.Errorf("exact match: want 100.0, got %v", res.LineAccuracy)
	}
}

func TestLineF1_OffByOneWithinTolerance(t *testing.T) {
	// ±2 tolerance: user submits 11 when truth is 10 — should be a hit.
	res := runEval(t, EvaluationRequest{
		TargetVulnerability: "sql injection",
		ConceptualFix:       "parameterized queries",
		VulnerableLines:     []int{10},
		UserTargetLines:     []int{11},
	})
	if !approxEq(res.LineAccuracy, 100.0) {
		t.Errorf("off-by-one within tolerance: want 100.0, got %v", res.LineAccuracy)
	}
}

func TestLineF1_OffByThreeMisses(t *testing.T) {
	// ±2 tolerance: user submits 13 when truth is 10 — should miss.
	res := runEval(t, EvaluationRequest{
		TargetVulnerability: "sql injection",
		ConceptualFix:       "parameterized queries",
		VulnerableLines:     []int{10},
		UserTargetLines:     []int{13},
	})
	if !approxEq(res.LineAccuracy, 0.0) {
		t.Errorf("off-by-three: want 0.0, got %v", res.LineAccuracy)
	}
}

func TestLineF1_DuplicateUserLineNotDoubleCredit(t *testing.T) {
	// User submits the same line twice. Only one truth match can be claimed.
	// precision = 1/2, recall = 1/1, F1 = 2*0.5*1.0/1.5 = 0.6667.
	res := runEval(t, EvaluationRequest{
		TargetVulnerability: "sql injection",
		ConceptualFix:       "parameterized queries",
		VulnerableLines:     []int{10},
		UserTargetLines:     []int{10, 10},
	})
	want := 2 * (1.0 / 2.0) * (1.0 / 1.0) / (1.0/2.0 + 1.0/1.0) * 100
	if !approxEq(res.LineAccuracy, want) {
		t.Errorf("duplicate user line: want %.2f, got %v", want, res.LineAccuracy)
	}
}

func TestLineF1_PrecisionPenalty(t *testing.T) {
	// User over-claims: truth=[10], user=[10, 20, 30, 40].
	// precision = 1/4, recall = 1/1, F1 = 2*0.25*1/1.25 = 0.4 → 40%.
	res := runEval(t, EvaluationRequest{
		TargetVulnerability: "sql injection",
		ConceptualFix:       "parameterized queries",
		VulnerableLines:     []int{10},
		UserTargetLines:     []int{10, 20, 30, 40},
	})
	if !approxEq(res.LineAccuracy, 40.0) {
		t.Errorf("over-claim precision penalty: want 40.0, got %v", res.LineAccuracy)
	}
}

func TestLineF1_RecallPenalty(t *testing.T) {
	// User under-claims: truth=[10, 20, 30, 40], user=[10].
	// precision = 1/1, recall = 1/4, F1 = 2*1*0.25/1.25 = 0.4 → 40%.
	res := runEval(t, EvaluationRequest{
		TargetVulnerability: "sql injection",
		ConceptualFix:       "parameterized queries",
		VulnerableLines:     []int{10, 20, 30, 40},
		UserTargetLines:     []int{10},
	})
	if !approxEq(res.LineAccuracy, 40.0) {
		t.Errorf("under-claim recall penalty: want 40.0, got %v", res.LineAccuracy)
	}
}

func TestLineF1_NoUserLinesWithTruth(t *testing.T) {
	// Truth exists, user submitted none. lineAccuracy=0.
	res := runEval(t, EvaluationRequest{
		TargetVulnerability: "sql injection",
		ConceptualFix:       "parameterized queries",
		VulnerableLines:     []int{10},
		UserTargetLines:     nil,
	})
	if res.LineAccuracy != 0 {
		t.Errorf("no user lines: want 0, got %v", res.LineAccuracy)
	}
}

func TestLineF1_LegacyChallengeNoTruth(t *testing.T) {
	// Legacy challenges have no VulnerableLines — the evaluator must fall
	// back to the 60/40 vuln/fix weighting and not let line_accuracy
	// (which stays 0) drag the overall score down.
	res := runEval(t, EvaluationRequest{
		UserAnswer:          "sql injection via string concatenation; fix with parameterized queries",
		TargetVulnerability: "sql injection via string concatenation",
		ConceptualFix:       "use parameterized queries",
		VulnerableLines:     nil,
		UserTargetLines:     []int{10, 20, 30}, // ignored
	})
	if res.LineAccuracy != 0 {
		t.Errorf("legacy: line_accuracy should be 0, got %v", res.LineAccuracy)
	}
	// Overall should be vuln*0.60 + fix*0.40 — not the lined-weighted formula.
	want := res.VulnScore*0.60 + res.FixScore*0.40
	want = math.Round(want*100) / 100
	if !approxEq(res.OverallScore, want) {
		t.Errorf("legacy weighting: want %.2f, got %v", want, res.OverallScore)
	}
}

// ── Pass-threshold and identification flags ──

func TestPassThreshold(t *testing.T) {
	// Exactly at threshold (60) should pass.
	if PassThreshold != 60.0 {
		t.Fatalf("PassThreshold changed from 60 to %v — update tests intentionally", PassThreshold)
	}
}

func TestVulnIdentified_LowFidelity(t *testing.T) {
	// A user answer that name-drops the wrong vulnerability class should
	// not get a passing vuln_identified flag.
	res := runEval(t, EvaluationRequest{
		UserAnswer:          "this looks like a privilege escalation problem; add MFA",
		TargetVulnerability: "buffer overflow via strcpy without bounds checking",
		ConceptualFix:       "replace strcpy with strncpy and validate input length",
		VulnerableLines:     []int{},
	})
	if res.VulnIdentified {
		t.Errorf("vuln_identified should be false for clearly wrong answer; got score=%v", res.VulnScore)
	}
}

func TestKeywordMatching_BufferOverflow(t *testing.T) {
	// Verify keyword vocabulary picks up canonical buffer-overflow concepts.
	res := runEval(t, EvaluationRequest{
		UserAnswer:          "buffer overflow caused by strcpy without bounds checking; use strncpy and validate length",
		TargetVulnerability: "buffer overflow via strcpy in parse_log_entry; no bounds check",
		ConceptualFix:       "replace strcpy with strncpy and add bounds checking on input length",
		VulnerableLines:     []int{},
		Difficulty:          8,
	})
	if res.VulnScore < 50 {
		t.Errorf("expected strong vuln_score for matching keywords, got %v", res.VulnScore)
	}
	if res.FixScore < 40 {
		t.Errorf("expected reasonable fix_score for matching keywords, got %v", res.FixScore)
	}
}
