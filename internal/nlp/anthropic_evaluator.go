package nlp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const anthropicAPIURL = "https://api.anthropic.com/v1/messages"
const anthropicAPIVersion = "2023-06-01"

// AnthropicEvaluator uses Claude to perform deep semantic evaluation
// of user-submitted vulnerability assessments. It implements the
// SemanticMatcher interface and can be swapped in for the keyword-based
// LLMEvaluator via config.
type AnthropicEvaluator struct {
	apiKey     string
	model      string
	httpClient *http.Client
}

// NewAnthropicEvaluator creates an evaluator backed by the Anthropic Messages API.
func NewAnthropicEvaluator(apiKey, model string) *AnthropicEvaluator {
	return &AnthropicEvaluator{
		apiKey: apiKey,
		model:  model,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (e *AnthropicEvaluator) Name() string {
	return "anthropic-evaluator-" + e.model
}

// systemPrompt defines the LLM's evaluation persona and strict output contract.
const systemPrompt = `You are a Principal Application Security Engineer conducting a rigorous evaluation of a security assessment submission. You have 15+ years of experience in vulnerability research, penetration testing, and secure code review across all major languages and platforms.

## Your Task

You will receive:
1. TARGET_VULNERABILITY — the ground-truth description of the vulnerability in the code
2. CONCEPTUAL_FIX — the expected remediation approach
3. VULNERABLE_LINES — the ground-truth line numbers where the flaw exists (may be empty for legacy challenges)
4. USER_SUBMISSION — the candidate's assessment including their identified line numbers and explanation

## Evaluation Criteria

Score the submission on three axes (0-100 each):

### Vulnerability Identification (vuln_score) — Weight: 40%
- Did the user correctly identify the vulnerability class (e.g., SQL injection, buffer overflow)?
- Did they explain the attack vector or exploitation path?
- Did they identify the specific code location or pattern that creates the vulnerability?
- Award partial credit for identifying related but imprecise concepts.

### Remediation Analysis (fix_score) — Weight: 30%
- Did the user propose a valid fix or mitigation?
- Is the proposed fix technically correct for the identified vulnerability?
- Did they mention defense-in-depth measures?
- Award partial credit for correct general direction even if specifics are wrong.

### Line Accuracy (line_accuracy) — Weight: 30%
- Did the user correctly identify the vulnerable line numbers?
- Allow ±2 line tolerance (e.g., if truth is line 45, lines 43-47 count as hits).
- Score using F1 (precision × recall harmonic mean) of correct line hits.
- If no VULNERABLE_LINES are provided (legacy challenge), set line_accuracy to 0 and use the legacy 60/40 weighting for vuln_score/fix_score.

## Scoring Guidelines
- 90-100: Expert-level analysis. Precise identification, correct exploitation path, comprehensive fix, exact lines.
- 70-89: Strong analysis. Correct vulnerability class, reasonable attack scenario, valid fix, close lines.
- 50-69: Partial understanding. Right general area but missing key details.
- 25-49: Weak. Vaguely related concepts but fundamental misunderstanding.
- 0-24: Incorrect or irrelevant.

## Pass Threshold
Overall score >= 60 constitutes a pass.
With line targeting: overall = vuln_score * 0.40 + fix_score * 0.30 + line_accuracy * 0.30.
Without line targeting (legacy): overall = vuln_score * 0.60 + fix_score * 0.40.

## Output Format

You MUST respond with ONLY a valid JSON object — no markdown fences, no commentary, no explanation outside the JSON. The JSON must conform exactly to this schema:

{
  "vuln_score": <number 0-100>,
  "fix_score": <number 0-100>,
  "line_accuracy": <number 0-100>,
  "overall_score": <number 0-100>,
  "is_correct": <boolean>,
  "vuln_identified": <boolean>,
  "fix_identified": <boolean>,
  "matched_vuln_terms": [<string>, ...],
  "matched_fix_terms": [<string>, ...],
  "terminal_log": [<string>, ...]
}

Where:
- matched_vuln_terms: security concepts the user correctly identified
- matched_fix_terms: remediation concepts the user correctly proposed
- terminal_log: an array of 10-18 strings simulating a terminal-style SAST analysis output. Use ">" prefix. Include line analysis results, matched concepts, scores, and a final verdict. Example:
  "> Initializing SAST analysis engine [anthropic-evaluator]..."
  "> Parsing submission (247 characters, 3 target lines)..."
  "> Line 145: MATCH — within vulnerable region [+]"
  "> Line 280: MISS — not in vulnerable scope [-]"
  "> Vulnerability concept match: buffer overflow [CONFIRMED]"
  "> Overall score: 85.0/100"
  "> STATUS: PASSED — Code audit accepted."

Be rigorous but fair. Award credit for conceptual understanding even when terminology differs from the reference answer.`

// anthropicRequest is the Anthropic Messages API request format.
type anthropicRequest struct {
	Model     string            `json:"model"`
	MaxTokens int               `json:"max_tokens"`
	System    string            `json:"system"`
	Messages  []anthropicMsg    `json:"messages"`
}

type anthropicMsg struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// anthropicResponse is the Anthropic Messages API response format.
type anthropicResponse struct {
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	StopReason string `json:"stop_reason"`
	Usage      struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

// llmEvalResult is the structured JSON the LLM must return.
type llmEvalResult struct {
	VulnScore        float64  `json:"vuln_score"`
	FixScore         float64  `json:"fix_score"`
	LineAccuracy     float64  `json:"line_accuracy"`
	OverallScore     float64  `json:"overall_score"`
	IsCorrect        bool     `json:"is_correct"`
	VulnIdentified   bool     `json:"vuln_identified"`
	FixIdentified    bool     `json:"fix_identified"`
	MatchedVulnTerms []string `json:"matched_vuln_terms"`
	MatchedFixTerms  []string `json:"matched_fix_terms"`
	TerminalLog      []string `json:"terminal_log"`
}

func (e *AnthropicEvaluator) Evaluate(ctx context.Context, req EvaluationRequest) (*EvaluationResult, error) {
	// Format vulnerable lines for the prompt
	vulnLinesStr := "N/A (legacy challenge)"
	if len(req.VulnerableLines) > 0 {
		parts := make([]string, len(req.VulnerableLines))
		for i, l := range req.VulnerableLines {
			parts[i] = fmt.Sprintf("%d", l)
		}
		vulnLinesStr = fmt.Sprintf("[%s]", strings.Join(parts, ", "))
	}

	userLinesStr := "None provided"
	if len(req.UserTargetLines) > 0 {
		parts := make([]string, len(req.UserTargetLines))
		for i, l := range req.UserTargetLines {
			parts[i] = fmt.Sprintf("%d", l)
		}
		userLinesStr = fmt.Sprintf("[%s]", strings.Join(parts, ", "))
	}

	// Build the user message with the evaluation context
	userMsg := fmt.Sprintf(`## TARGET_VULNERABILITY
%s

## CONCEPTUAL_FIX
%s

## VULNERABLE_LINES
%s

## USER_SUBMISSION
Language: %s | Difficulty: %d/10
User Target Lines: %s

%s`, req.TargetVulnerability, req.ConceptualFix, vulnLinesStr, req.Language, req.Difficulty, userLinesStr, req.UserAnswer)

	// Build API request
	apiReq := anthropicRequest{
		Model:     e.model,
		MaxTokens: 1024,
		System:    systemPrompt,
		Messages: []anthropicMsg{
			{Role: "user", Content: userMsg},
		},
	}

	body, err := json.Marshal(apiReq)
	if err != nil {
		return nil, fmt.Errorf("marshaling request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, anthropicAPIURL, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("creating request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", e.apiKey)
	httpReq.Header.Set("anthropic-version", anthropicAPIVersion)

	resp, err := e.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("calling Anthropic API: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Anthropic API error (status %d): %s", resp.StatusCode, string(respBody))
	}

	var apiResp anthropicResponse
	if err := json.Unmarshal(respBody, &apiResp); err != nil {
		return nil, fmt.Errorf("decoding Anthropic response: %w", err)
	}

	if len(apiResp.Content) == 0 || apiResp.Content[0].Type != "text" {
		return nil, fmt.Errorf("unexpected Anthropic response format: no text content")
	}

	// Parse the structured JSON from Claude's response
	rawJSON := apiResp.Content[0].Text
	var evalResult llmEvalResult
	if err := json.Unmarshal([]byte(rawJSON), &evalResult); err != nil {
		return nil, fmt.Errorf("parsing evaluation JSON from Claude: %w (raw: %.200s)", err, rawJSON)
	}

	return &EvaluationResult{
		VulnScore:        evalResult.VulnScore,
		FixScore:         evalResult.FixScore,
		LineAccuracy:     evalResult.LineAccuracy,
		OverallScore:     evalResult.OverallScore,
		Passed:           evalResult.IsCorrect,
		VulnIdentified:   evalResult.VulnIdentified,
		FixIdentified:    evalResult.FixIdentified,
		MatchedVulnTerms: evalResult.MatchedVulnTerms,
		MatchedFixTerms:  evalResult.MatchedFixTerms,
		TerminalLog:      evalResult.TerminalLog,
	}, nil
}
