package nlp

import (
	"fmt"
	"sort"
)

// Scoring tunables for the keyword evaluator. These were calibrated against the
// test matrix in scoring_test.go so that a genuinely correct submission clears
// the pass threshold while vague/wrong submissions do not.
const (
	// canonWeight / keyWeight blend the two concept-coverage signals: matching
	// the canonical security vocabulary vs. reproducing the challenge's own
	// ground-truth wording. Keyword overlap carries slightly more weight so
	// that vulnerability classes absent from the canonical map are still fully
	// scorable.
	canonWeight = 0.45
	keyWeight   = 0.55

	// keywordDenomCap bounds how many of the reference's keywords a user must
	// reproduce for full credit. Without a cap, a verbose ground-truth
	// description would make a perfect answer impossible; with it, reproducing
	// the dozen most salient concepts is "full coverage".
	keywordDenomCap = 12

	// lineTolerance is the ± window (in lines) within which a user's flagged
	// line is treated as hitting a ground-truth line.
	lineTolerance = 2

	// regionMergeGap collapses near-adjacent ground-truth lines into a single
	// vulnerable "region". A user who flags any line inside a contiguous
	// multi-line vulnerable block has found that block — they should not be
	// penalised for not enumerating every line of it.
	regionMergeGap = 2
)

// conceptCoverage scores how well a user's (already-lowercased) answer covers
// the expected concepts for one axis (vulnerability or fix). It blends two
// signals:
//
//  1. Canonical coverage: the fraction of the challenge's relevant canonical
//     security concepts (with synonym expansion) that the answer mentions.
//  2. Keyword overlap: the fraction of the challenge's own ground-truth
//     vocabulary that the answer reproduces.
//
// When the canonical map has no entry for this challenge's class, the score
// rests entirely on keyword overlap — which is always category-appropriate, so
// no vulnerability class is ever mathematically unpassable. Returns the
// coverage in [0,1] plus the list of canonical concepts detected (for display).
func conceptCoverage(lowerAnswer string, canon map[string][]string, referenceText string) (float64, []string) {
	matchedCanon := extractMatchedTerms(lowerAnswer, canon)

	canonRatio := -1.0 // -1 signals "no canonical signal available"
	if len(canon) > 0 {
		canonRatio = float64(len(matchedCanon)) / float64(len(canon))
	}

	keyRatio := 0.0
	keywords := extractKeywords(referenceText)
	if len(keywords) > 0 {
		denom := len(keywords)
		if denom > keywordDenomCap {
			denom = keywordDenomCap
		}
		hits := countKeywordHits(lowerAnswer, keywords)
		if hits > denom {
			hits = denom
		}
		keyRatio = float64(hits) / float64(denom)
	}

	var coverage float64
	if canonRatio >= 0 {
		coverage = canonWeight*canonRatio + keyWeight*keyRatio
	} else {
		coverage = keyRatio
	}
	if coverage > 1 {
		coverage = 1
	}
	return coverage, matchedCanon
}

type lineRegion struct {
	lo, hi int
}

// mergeRegions collapses a sorted, de-duplicated list of ground-truth line
// numbers into vulnerable regions, merging entries that sit within mergeGap of
// each other. [47,48,49,50] becomes one region; [10,20,30,40] stays four.
func mergeRegions(sortedTruth []int, mergeGap int) []lineRegion {
	if len(sortedTruth) == 0 {
		return nil
	}
	regions := []lineRegion{{lo: sortedTruth[0], hi: sortedTruth[0]}}
	for _, l := range sortedTruth[1:] {
		last := &regions[len(regions)-1]
		if l-last.hi <= mergeGap {
			last.hi = l
		} else {
			regions = append(regions, lineRegion{lo: l, hi: l})
		}
	}
	return regions
}

// scoreLineAccuracy computes a region-based F1 (0-100) of the user's flagged
// lines against the ground-truth vulnerable lines, with a ±lineTolerance
// window and contiguous-block merging. It returns the score and a slice of
// terminal-log lines describing the per-line hit/miss outcome.
//
// recall    = vulnerable regions the user located / total regions
// precision = user lines that landed on some region / total user lines
func scoreLineAccuracy(truth, user []int) (float64, []string) {
	logs := []string{
		fmt.Sprintf("> Line targeting: user flagged %d line(s), ground truth has %d",
			len(user), len(truth)),
	}

	user = uniqSortInts(user)
	regions := mergeRegions(uniqSortInts(truth), regionMergeGap)
	if len(regions) > 1 {
		logs = append(logs, fmt.Sprintf("> Vulnerable code spans %d distinct region(s); ±%d line tolerance applied",
			len(regions), lineTolerance))
	}

	regionHit := make([]bool, len(regions))
	userHitsAny := 0
	for _, ul := range user {
		hitSomething := false
		hitRegion := -1
		for ri, r := range regions {
			if ul >= r.lo-lineTolerance && ul <= r.hi+lineTolerance {
				regionHit[ri] = true
				hitSomething = true
				if hitRegion == -1 {
					hitRegion = ri
				}
			}
		}
		if hitSomething {
			userHitsAny++
			region := regions[hitRegion]
			if region.lo == region.hi {
				logs = append(logs, fmt.Sprintf("  [+] Line %d — HIT (vulnerable line %d)", ul, region.lo))
			} else {
				logs = append(logs, fmt.Sprintf("  [+] Line %d — HIT (vulnerable region L%d–L%d)", ul, region.lo, region.hi))
			}
		} else {
			logs = append(logs, fmt.Sprintf("  [-] Line %d — MISS", ul))
		}
	}

	regionsFound := 0
	for _, ok := range regionHit {
		if ok {
			regionsFound++
		}
	}

	accuracy := 0.0
	if len(user) > 0 && len(regions) > 0 {
		precision := float64(userHitsAny) / float64(len(user))
		recall := float64(regionsFound) / float64(len(regions))
		if precision+recall > 0 {
			accuracy = 2 * precision * recall / (precision + recall) * 100
		}
	}
	logs = append(logs, fmt.Sprintf("> Line accuracy: %.1f%% (%d/%d region(s) located)",
		accuracy, regionsFound, len(regions)))
	return accuracy, logs
}

// uniqSortInts returns a sorted copy of xs with duplicates removed.
func uniqSortInts(xs []int) []int {
	if len(xs) == 0 {
		return nil
	}
	seen := make(map[int]bool, len(xs))
	out := make([]int, 0, len(xs))
	for _, x := range xs {
		if !seen[x] {
			seen[x] = true
			out = append(out, x)
		}
	}
	sort.Ints(out)
	return out
}
