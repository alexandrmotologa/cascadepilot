package finops

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"time"
)

// ExportCSV writes all transaction records in CSV format.
func (s *Store) ExportCSV(w io.Writer) error {
	records, err := s.GetRecentRequests(10000)
	if err != nil {
		return err
	}

	csvWriter := csv.NewWriter(w)
	defer csvWriter.Flush()

	header := []string{
		"id", "timestamp", "prompt_preview", "complexity_score", "routed_tier",
		"tier1_model", "tier2_model", "tier1_confidence", "escalated", "escalation_reason",
		"prompt_tokens", "completion_tokens", "latency_ms", "baseline_cost_usd",
		"actual_cost_usd", "saved_cost_usd", "stream",
	}
	if err := csvWriter.Write(header); err != nil {
		return err
	}

	for _, r := range records {
		row := []string{
			r.ID,
			r.Timestamp.Format(time.RFC3339),
			r.PromptPreview,
			strconv.FormatFloat(r.ComplexityScore, 'f', 4, 64),
			r.RoutedTier,
			r.Tier1Model,
			r.Tier2Model,
			strconv.FormatFloat(r.Tier1Confidence, 'f', 4, 64),
			strconv.FormatBool(r.Escalated),
			r.EscalationReason,
			strconv.Itoa(r.PromptTokens),
			strconv.Itoa(r.CompletionTokens),
			strconv.FormatInt(r.LatencyMs, 10),
			fmt.Sprintf("%.6f", r.BaselineCostUSD),
			fmt.Sprintf("%.6f", r.ActualCostUSD),
			fmt.Sprintf("%.6f", r.SavedCostUSD),
			strconv.FormatBool(r.Stream),
		}
		if err := csvWriter.Write(row); err != nil {
			return err
		}
	}

	return nil
}

// ExportJSON writes all transaction records in JSON array format.
func (s *Store) ExportJSON(w io.Writer) error {
	records, err := s.GetRecentRequests(10000)
	if err != nil {
		return err
	}

	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(records)
}
