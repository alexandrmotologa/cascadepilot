package verifier

import (
	"regexp"
	"strings"
)

var (
	// Hedging and refusal patterns indicating model uncertainty or evasion
	refusalPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)\b(i am not (completely |entirely )?(sure|certain))\b`),
		regexp.MustCompile(`(?i)\b(i (cannot|can't) (fulfill|answer|guarantee|verify))\b`),
		regexp.MustCompile(`(?i)\b(as an? (ai|language model|artificial intelligence))\b`),
		regexp.MustCompile(`(?i)\b(i do not have (access to|information regarding))\b`),
		regexp.MustCompile(`(?i)\b(please consult (a|an) (professional|expert|lawyer|doctor))\b`),
		regexp.MustCompile(`(?i)\b(i apologize, but (i cannot|i am unable))\b`),
		regexp.MustCompile(`(?i)\b(it might be possible.*but you should verify)\b`),
	}
)

// HeuristicChecker evaluates completions for quality degradation signals.
type HeuristicChecker struct{}

// NewHeuristicChecker creates a new heuristic checker.
func NewHeuristicChecker() *HeuristicChecker {
	return &HeuristicChecker{}
}

// CheckRefusalPenalty returns a penalty score between 0.0 and 1.0.
func (h *HeuristicChecker) CheckRefusalPenalty(content string) (penalty float64, reason string) {
	trimmed := strings.TrimSpace(content)
	if len(trimmed) == 0 {
		return 1.0, "empty_completion"
	}

	for _, pattern := range refusalPatterns {
		if pattern.MatchString(trimmed) {
			return 0.75, "hedging_or_refusal_detected"
		}
	}

	// Check repetitive degeneracy (repetition loops)
	lines := strings.Split(trimmed, "\n")
	if len(lines) >= 4 {
		lineMap := make(map[string]int)
		for _, line := range lines {
			l := strings.TrimSpace(line)
			if len(l) > 10 {
				lineMap[l]++
				if lineMap[l] >= 3 {
					return 0.80, "repetitive_loop_detected"
				}
			}
		}
	}

	// Check truncated code block (odd number of triple backticks)
	if strings.Count(trimmed, "```")%2 != 0 {
		return 0.40, "truncated_code_block"
	}

	return 0.0, ""
}
