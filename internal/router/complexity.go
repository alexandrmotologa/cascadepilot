package router

import (
	"math"
	"regexp"
	"strings"

	"github.com/alexandrmotologa/cascadepilot/internal/provider"
)

var (
	constraintPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)\b(do not|never|must (not|only|contain)|cannot|strictly|exact(ly)?|without using)\b`),
		regexp.MustCompile(`(?i)\b(schema|regex|json format|format specifications?)\b`),
		regexp.MustCompile(`(?i)\b(under no circumstances?|forbidden|disallow)\b`),
	}

	reasoningPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)\b(step by step|prove|mathematical|formal proof|derive|derivation)\b`),
		regexp.MustCompile(`(?i)\b(concurrency|distributed|deadlock|race condition|split-brain|consensus)\b`),
		regexp.MustCompile(`(?i)\b(analyze tradeoffs?|reconcile|jurisdiction|precedents?|counterexample)\b`),
		regexp.MustCompile(`(?i)\b(algorithm|complexity|asymptotic|closed-form|markov chain|eigenvalue)\b`),
		regexp.MustCompile(`(?i)\b(memory leak|heap profile|pprof|goroutine stack|lock-free)\b`),
	}
)

// ComplexityResult contains the breakdown of the prompt complexity evaluation.
type ComplexityResult struct {
	Score              float64 `json:"score"`
	LengthRatio        float64 `json:"length_ratio"`
	ConstraintScore    float64 `json:"constraint_score"`
	ReasoningScore     float64 `json:"reasoning_score"`
	BypassDirectToTier2 bool    `json:"bypass_direct_to_tier2"`
	BypassReason       string  `json:"bypass_reason,omitempty"`
}

// ComplexityScorer evaluates inbound chat completion prompts in under 2ms.
type ComplexityScorer struct {
	w1 float64 // token length weight
	w2 float64 // constraint density weight
	w3 float64 // reasoning indicator weight
}

// NewComplexityScorer creates an inbound scorer with default weights.
func NewComplexityScorer() *ComplexityScorer {
	return &ComplexityScorer{
		w1: 0.35,
		w2: 0.35,
		w3: 0.30,
	}
}

// Score analyzes the prompt and returns a complexity score between 0.0 and 1.0.
func (s *ComplexityScorer) Score(req *provider.ChatCompletionRequest, thetaBypass float64) ComplexityResult {
	var fullPrompt strings.Builder
	for _, m := range req.Messages {
		fullPrompt.WriteString(m.ContentString())
		fullPrompt.WriteString(" ")
	}
	text := fullPrompt.String()

	// Check explicit bypass headers/tags
	if strings.Contains(text, "[bypass-cascade]") || strings.Contains(text, "cascadepilot-direct-tier2") {
		return ComplexityResult{
			Score:              1.0,
			LengthRatio:        1.0,
			ConstraintScore:    1.0,
			ReasoningScore:     1.0,
			BypassDirectToTier2: true,
			BypassReason:       "explicit_bypass_tag",
		}
	}

	words := strings.Fields(text)
	wordCount := len(words)
	charCount := len(text)

	// 1. Length ratio normalized to 2048 tokens (~1500 words or 6000 chars)
	estimatedTokens := math.Max(float64(wordCount)*1.3, float64(charCount)/4.0)
	lengthRatio := math.Min(1.0, estimatedTokens/2048.0)

	// 2. Constraint count and sigmoid squashing
	constraintCount := 0
	for _, pattern := range constraintPatterns {
		matches := pattern.FindAllStringIndex(text, -1)
		constraintCount += len(matches)
	}
	var constraintScore float64
	if constraintCount > 0 {
		constraintScore = math.Min(1.0, float64(constraintCount)*0.35)
	}

	// 3. Reasoning indicator density and intensity
	reasoningMatches := 0
	for _, pattern := range reasoningPatterns {
		matches := pattern.FindAllStringIndex(text, -1)
		reasoningMatches += len(matches)
	}

	var reasoningScore float64
	if reasoningMatches > 0 {
		density := float64(reasoningMatches) * 0.25
		reasoningScore = math.Min(1.0, density)
	}

	// Weighted linear combination
	baseScore := (s.w1 * lengthRatio) + (s.w2 * constraintScore) + (s.w3 * reasoningScore)
	
	// If reasoning intensity is high (multiple multi-step / mathematical markers), elevate score
	score := math.Max(baseScore, reasoningScore*0.80)
	score = math.Max(0.0, math.Min(1.0, score))

	bypass := score > thetaBypass
	var bypassReason string
	if bypass {
		bypassReason = "complexity_above_threshold"
	}

	return ComplexityResult{
		Score:              score,
		LengthRatio:        lengthRatio,
		ConstraintScore:    constraintScore,
		ReasoningScore:     reasoningScore,
		BypassDirectToTier2: bypass,
		BypassReason:       bypassReason,
	}
}
