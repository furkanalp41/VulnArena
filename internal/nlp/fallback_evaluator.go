package nlp

import (
	"context"
	"log/slog"
)

// FallbackEvaluator chains a primary evaluator with a backup. If the primary
// (typically the Anthropic LLM) returns an error — network failure, parse
// failure on truncated JSON, rate limit, anything — the request is re-run
// against the backup (typically the keyword evaluator) so the user still
// gets a scored submission instead of a 500.
//
// The primary's failure is logged at WARN; we never surface the underlying
// error to the caller because the backup produced a valid result. The
// terminal_log of the backup result is annotated so reviewers can see the
// fallback path triggered without inspecting server logs.
type FallbackEvaluator struct {
	primary SemanticMatcher
	backup  SemanticMatcher
	logger  *slog.Logger
}

func NewFallbackEvaluator(primary, backup SemanticMatcher, logger *slog.Logger) *FallbackEvaluator {
	return &FallbackEvaluator{
		primary: primary,
		backup:  backup,
		logger:  logger,
	}
}

func (e *FallbackEvaluator) Name() string {
	return e.primary.Name() + "+fallback(" + e.backup.Name() + ")"
}

func (e *FallbackEvaluator) Evaluate(ctx context.Context, req EvaluationRequest) (*EvaluationResult, error) {
	res, err := e.primary.Evaluate(ctx, req)
	if err == nil {
		return res, nil
	}

	e.logger.Warn("semantic evaluator primary failed, using backup",
		slog.String("primary", e.primary.Name()),
		slog.String("backup", e.backup.Name()),
		slog.String("error", err.Error()),
	)

	res, backupErr := e.backup.Evaluate(ctx, req)
	if backupErr != nil {
		// Both evaluators broken. Surface the backup error since it's the
		// last thing that ran — the primary error is in the WARN log above.
		return nil, backupErr
	}

	// Annotate the result so the user knows degraded scoring is active.
	annotation := "> NOTICE: primary evaluator unavailable — keyword fallback scoring applied."
	res.TerminalLog = append([]string{annotation}, res.TerminalLog...)
	return res, nil
}
