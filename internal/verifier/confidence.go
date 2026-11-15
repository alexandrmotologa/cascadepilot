package verifier

import (
	"math"

	"github.com/alexandrmotologa/cascadepilot/internal/provider"
)

// ConfidenceAnalyzer calculates confidence from token log-probabilities.
type ConfidenceAnalyzer struct{}

// NewConfidenceAnalyzer creates a new analyzer.
func NewConfidenceAnalyzer() *ConfidenceAnalyzer {
	return &ConfidenceAnalyzer{}
}

// ComputeLogProbConfidence computes average linear probability from token logprobs.
func (c *ConfidenceAnalyzer) ComputeLogProbConfidence(logprobs *provider.ChoiceLogProbs) float64 {
	if logprobs == nil || len(logprobs.Content) == 0 {
		// Neutral default confidence when provider omits token log probabilities
		return 0.85
	}

	var sumProb float64
	var count int

	for _, token := range logprobs.Content {
		if math.IsNaN(token.LogProb) || math.IsInf(token.LogProb, 0) {
			continue
		}
		// Clamp logprob to avoid numerical overflow: e^logprob
		prob := math.Exp(math.Max(-10.0, math.Min(0.0, token.LogProb)))
		sumProb += prob
		count++
	}

	if count == 0 {
		return 0.85
	}

	avgProb := sumProb / float64(count)
	return math.Max(0.0, math.Min(1.0, avgProb))
}
