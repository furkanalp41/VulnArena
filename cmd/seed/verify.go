package main

import (
	"fmt"
	"sort"
	"strings"
)

// verifyChallengeLines is the deterministic line verifier. It splits each
// challenge's `code` field on "\n" and asserts that every index in
// `vulnerableLines` points to a line that is:
//
//  1. In range (1 <= idx <= line count)
//  2. Non-empty after trimming whitespace
//  3. Not a pure comment line for the challenge's language. This includes
//     single-line comments AND continuations of multi-line block comments
//     (e.g. " * foo" or " bar */" where the line lives entirely inside an
//     unclosed /* ... */ region).
//  4. Not a duplicate
//  5. Sorted ascending (catches paste-order mistakes)
//
// This runs before any DB write and on every -verify pass. If any challenge
// fails, the seed binary aborts. New challenges added to buildChallenges()
// inherit this gate automatically — there is no path to insert a challenge
// whose VulnerableLines are wrong.
func verifyChallengeLines(challenges []challengeSeed, verbose bool) error {
	var failures []string

	for _, ch := range challenges {
		// Legacy challenges without line targeting are exempt by design
		// (see EvaluationRequest.VulnerableLines: "Empty for legacy challenges").
		if len(ch.vulnerableLines) == 0 {
			continue
		}

		lines := strings.Split(ch.code, "\n")
		commentMask := classifyComments(ch.code, ch.langSlug)
		seen := make(map[int]bool, len(ch.vulnerableLines))

		// Duplicate detection
		for _, l := range ch.vulnerableLines {
			if seen[l] {
				failures = append(failures, fmt.Sprintf(
					"  [%s] duplicate line %d in VulnerableLines", ch.slug, l))
			}
			seen[l] = true
		}

		// Ascending-sorted check (mirrors the canonical input to F1 scoring)
		if !sort.IntsAreSorted(ch.vulnerableLines) {
			failures = append(failures, fmt.Sprintf(
				"  [%s] VulnerableLines not sorted ascending: %v",
				ch.slug, ch.vulnerableLines))
		}

		for _, n := range ch.vulnerableLines {
			if n < 1 || n > len(lines) {
				failures = append(failures, fmt.Sprintf(
					"  [%s] line %d out of range (file has %d lines)",
					ch.slug, n, len(lines)))
				continue
			}

			raw := lines[n-1]
			trimmed := strings.TrimSpace(raw)

			if trimmed == "" {
				failures = append(failures, fmt.Sprintf(
					"  [%s] line %d is EMPTY", ch.slug, n))
				continue
			}

			if commentMask[n-1] {
				failures = append(failures, fmt.Sprintf(
					"  [%s] line %d is COMMENT-ONLY (%s): %q",
					ch.slug, n, ch.langSlug, raw))
				continue
			}

			if verbose {
				fmt.Printf("  [%s] L%-4d %s\n", ch.slug, n, raw)
			}
		}
	}

	if len(failures) == 0 {
		return nil
	}

	return fmt.Errorf("%d failure(s):\n%s",
		len(failures), strings.Join(failures, "\n"))
}

// classifyComments returns a bool slice where index i is true iff line i+1
// is a pure comment line in the given language. It walks the code once,
// tracking block-comment state so continuations like
//
//	/* opens here
//	   continues
//	   closes */
//
// are all flagged as comments. String-literal awareness is intentionally
// omitted — challenge code rarely contains `/*` inside a string, and false
// positives there are easy to fix manually when the verifier flags them.
func classifyComments(code, langSlug string) []bool {
	lines := strings.Split(code, "\n")
	result := make([]bool, len(lines))

	blockOpen, blockClose := blockCommentDelimiters(langSlug)
	hasBlock := blockOpen != ""

	inBlock := false
	for i, raw := range lines {
		trimmed := strings.TrimSpace(raw)
		if trimmed == "" {
			// Blank lines are caught separately by the EMPTY check.
			result[i] = false
			continue
		}

		if hasBlock && inBlock {
			closeIdx := strings.Index(raw, blockClose)
			if closeIdx < 0 {
				// Still entirely inside the block.
				result[i] = true
				continue
			}

			rest := raw[closeIdx+len(blockClose):]
			restTrimmed := strings.TrimSpace(rest)
			inBlock = false

			// If anything past "*/" re-opens an unclosed block, we go back in.
			if openIdx := strings.Index(rest, blockOpen); openIdx >= 0 {
				if !strings.Contains(rest[openIdx+len(blockOpen):], blockClose) {
					inBlock = true
				}
			}

			// If "*/" is followed only by whitespace or another comment, the
			// whole line is comment. Otherwise the trailing text is code.
			if restTrimmed == "" || isSingleLineCommentOnly(restTrimmed, langSlug) {
				result[i] = true
				continue
			}
			result[i] = false
			continue
		}

		// Not currently inside a block comment.
		if isSingleLineCommentOnly(trimmed, langSlug) {
			result[i] = true
			// Did this comment line open a still-unclosed block?
			if hasBlock && strings.HasPrefix(trimmed, blockOpen) &&
				!strings.Contains(trimmed[len(blockOpen):], blockClose) {
				inBlock = true
			}
			continue
		}

		// This is a code line, but it may open a multi-line block at its tail.
		if hasBlock {
			if openIdx := strings.Index(raw, blockOpen); openIdx >= 0 {
				if !strings.Contains(raw[openIdx+len(blockOpen):], blockClose) {
					inBlock = true
				}
			}
		}
		result[i] = false
	}
	return result
}

// isSingleLineCommentOnly answers: does this trimmed line consist entirely
// of a single-line-style comment (or block-comment delimiter) in the given
// language? This is the per-line classifier used by classifyComments.
func isSingleLineCommentOnly(trimmed, langSlug string) bool {
	if trimmed == "" {
		return true
	}

	hashLangs := map[string]bool{
		"python": true, "bash": true, "shell": true, "ruby": true,
		"yaml": true, "yml": true, "terraform": true, "hcl": true,
		"toml": true, "ini": true, "conf": true, "dockerfile": true,
	}

	cFamilyLangs := map[string]bool{
		"go": true, "c": true, "cpp": true, "rust": true, "java": true,
		"javascript": true, "typescript": true, "nodejs": true, "node": true,
		"csharp": true, "cs": true, "kotlin": true, "swift": true, "scala": true,
		"php": true, "dart": true, "groovy": true, "solidity": true, "css": true,
	}

	if cFamilyLangs[langSlug] {
		if strings.HasPrefix(trimmed, "//") {
			return true
		}
		// "/* short */" entirely on one line.
		if strings.HasPrefix(trimmed, "/*") && strings.HasSuffix(trimmed, "*/") {
			return true
		}
		// Block-comment middle continuation: "* foo" or just "*"
		if trimmed == "*" || strings.HasPrefix(trimmed, "* ") {
			return true
		}
		// Block-comment closer on its own line.
		if trimmed == "*/" {
			return true
		}
	}

	if hashLangs[langSlug] || langSlug == "php" {
		if strings.HasPrefix(trimmed, "#") {
			return true
		}
	}

	if langSlug == "python" {
		if trimmed == `"""` || trimmed == "'''" {
			return true
		}
	}

	if langSlug == "sql" || langSlug == "haskell" {
		if strings.HasPrefix(trimmed, "--") {
			return true
		}
	}

	if langSlug == "html" || langSlug == "xml" ||
		langSlug == "svelte" || langSlug == "vue" {
		if strings.HasPrefix(trimmed, "<!--") && strings.HasSuffix(trimmed, "-->") {
			return true
		}
	}

	return false
}

// blockCommentDelimiters returns the (open, close) tokens for a multi-line
// block comment in the given language, or empty strings when the language
// has no block-comment syntax we care to track.
func blockCommentDelimiters(langSlug string) (open, close string) {
	cFamily := map[string]bool{
		"go": true, "c": true, "cpp": true, "rust": true, "java": true,
		"javascript": true, "typescript": true, "nodejs": true, "node": true,
		"csharp": true, "cs": true, "kotlin": true, "swift": true, "scala": true,
		"php": true, "dart": true, "groovy": true, "solidity": true, "css": true,
	}
	if cFamily[langSlug] {
		return "/*", "*/"
	}
	if langSlug == "html" || langSlug == "xml" {
		return "<!--", "-->"
	}
	return "", ""
}
