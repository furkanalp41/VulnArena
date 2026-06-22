package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// runAudit writes one annotated listing per challenge into outDir. The line
// numbers it prints are computed with the SAME strings.Split(code, "\n")
// 1-indexing that verifyChallengeLines and the Monaco editor use, so an auditor
// reading "L42| system(cmd)" is looking at the exact number a user would click.
//
// This is a review aid only — it touches no DB and is invoked via `-audit DIR`.
func runAudit(challenges []challengeSeed, outDir string) error {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}

	var index strings.Builder
	index.WriteString("file\tslug\tlang\tcat\tdifficulty\tnum_lines\tvulnerable_lines\n")

	for i, ch := range challenges {
		lineSet := make(map[int]bool, len(ch.vulnerableLines))
		for _, l := range ch.vulnerableLines {
			lineSet[l] = true
		}

		var b strings.Builder
		fmt.Fprintf(&b, "SLUG: %s\n", ch.slug)
		fmt.Fprintf(&b, "TITLE: %s\n", ch.title)
		fmt.Fprintf(&b, "LANG: %s | CAT: %s | DIFFICULTY: %d/10 | POINTS: %d\n",
			ch.langSlug, ch.catSlug, ch.difficulty, ch.points)
		fmt.Fprintf(&b, "CURRENT vulnerableLines: %v   (count=%d)\n", ch.vulnerableLines, len(ch.vulnerableLines))
		if ch.cveReference != "" {
			fmt.Fprintf(&b, "CVE: %s\n", ch.cveReference)
		}

		fmt.Fprintf(&b, "\n========== TARGET VULNERABILITY (ground truth) ==========\n%s\n", ch.targetVuln)
		fmt.Fprintf(&b, "\n========== CONCEPTUAL FIX (ground truth) ==========\n%s\n", ch.conceptualFix)

		fmt.Fprintf(&b, "\n========== CODE ==========\n")
		fmt.Fprintf(&b, "(1-indexed exactly as the verifier/editor see it; '>>>' marks a line currently in vulnerableLines)\n\n")
		lines := strings.Split(ch.code, "\n")
		for n, raw := range lines {
			ln := n + 1
			marker := "    "
			if lineSet[ln] {
				marker = ">>> "
			}
			fmt.Fprintf(&b, "%sL%-4d| %s\n", marker, ln, raw)
		}

		name := fmt.Sprintf("%03d-%s.txt", i+1, ch.slug)
		if err := os.WriteFile(filepath.Join(outDir, name), []byte(b.String()), 0o644); err != nil {
			return err
		}
		fmt.Fprintf(&index, "%s\t%s\t%s\t%s\t%d\t%d\t%v\n",
			name, ch.slug, ch.langSlug, ch.catSlug, ch.difficulty, len(lines), ch.vulnerableLines)
	}

	if err := os.WriteFile(filepath.Join(outDir, "INDEX.tsv"), []byte(index.String()), 0o644); err != nil {
		return err
	}
	fmt.Printf("[+] Wrote audit listings for %d challenges to %s\n", len(challenges), outDir)
	return nil
}
