package router

import (
	"context"
	"fmt"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/alexandrmotologa/cascadepilot/internal/config"
	"github.com/alexandrmotologa/cascadepilot/internal/finops"
	"github.com/alexandrmotologa/cascadepilot/internal/provider"
	"github.com/alexandrmotologa/cascadepilot/internal/verifier"
)

// BroadcasterFunc defines the signature for real-time SSE notifications.
type BroadcasterFunc func(event string, data interface{})

// Engine orchestrates speculative routing, in-flight verification, and escalation.
type Engine struct {
	cfg              *config.Config
	client           provider.ProviderClient
	scorer           *ComplexityScorer
	confAnalyzer     *verifier.ConfidenceAnalyzer
	heuristicChecker *verifier.HeuristicChecker
	schemaGate       *verifier.SchemaGate
	store            *finops.Store
	broadcaster      BroadcasterFunc

	// Dynamic thresholds protected by RWMutex
	mu            sync.RWMutex
	thetaBypass   float64
	tauConfidence float64

	t1CostCalc *provider.CostCalculator
	t2CostCalc *provider.CostCalculator
}

// NewEngine initializes the cascading routing pipeline.
func NewEngine(
	cfg *config.Config,
	client provider.ProviderClient,
	store *finops.Store,
	broadcaster BroadcasterFunc,
) *Engine {
	theta := cfg.Thresholds.ThetaBypass
	tau := cfg.Thresholds.TauConfidence

	// Load stored dynamic thresholds if present
	if store != nil {
		if storedTheta, storedTau, found, err := store.GetThresholds(); err == nil && found {
			theta = storedTheta
			tau = storedTau
		}
	}

	return &Engine{
		cfg:              cfg,
		client:           client,
		scorer:           NewComplexityScorer(),
		confAnalyzer:     verifier.NewConfidenceAnalyzer(),
		heuristicChecker: verifier.NewHeuristicChecker(),
		schemaGate:       verifier.NewSchemaGate(),
		store:            store,
		broadcaster:      broadcaster,
		thetaBypass:      theta,
		tauConfidence:    tau,
		t1CostCalc:       provider.NewCostCalculator(cfg.Tiers.Tier1.CostPerMillionInput, cfg.Tiers.Tier1.CostPerMillionOutput),
		t2CostCalc:       provider.NewCostCalculator(cfg.Tiers.Tier2.CostPerMillionInput, cfg.Tiers.Tier2.CostPerMillionOutput),
	}
}

// GetThresholds returns active dynamic thresholds.
func (e *Engine) GetThresholds() (float64, float64) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.thetaBypass, e.tauConfidence
}

// SetThresholds updates active dynamic thresholds and persists them.
func (e *Engine) SetThresholds(thetaBypass, tauConfidence float64) error {
	e.mu.Lock()
	e.thetaBypass = thetaBypass
	e.tauConfidence = tauConfidence
	e.mu.Unlock()

	if e.store != nil {
		return e.store.SetThresholds(thetaBypass, tauConfidence)
	}
	return nil
}

// Execute performs speculative cascading for non-streaming requests.
func (e *Engine) Execute(ctx context.Context, req *provider.ChatCompletionRequest) (*provider.ChatCompletionResponse, *finops.RequestRecord, error) {
	startTime := time.Now()

	thetaBypass, tauConfidence := e.GetThresholds()
	complexity := e.scorer.Score(req, thetaBypass)

	promptPreview := getPromptPreview(req.Messages)

	var finalResp *provider.ChatCompletionResponse
	var routedTier string
	var escalated bool
	var escalationReason string
	var t1Confidence float64
	var promptTokens, completionTokens int
	var baselineCost, actualCost, savedCost float64

	// Stage 1: Check Complexity Bypass
	if complexity.BypassDirectToTier2 {
		routedTier = "tier2_direct"
		resp, err := e.client.Complete(ctx, req, e.cfg.Tiers.Tier2, "tier2", complexity.Score)
		if err != nil {
			return nil, nil, fmt.Errorf("tier 2 direct execution error: %w", err)
		}
		finalResp = resp
		promptTokens = resp.Usage.PromptTokens
		completionTokens = resp.Usage.CompletionTokens

		baselineCost = e.t2CostCalc.ComputeCost(promptTokens, completionTokens)
		actualCost = baselineCost
		savedCost = 0.0
		t1Confidence = 1.0
	} else {
		// Stage 2: Speculative Tier 1 Execution
		t1Resp, err := e.client.Complete(ctx, req, e.cfg.Tiers.Tier1, "tier1", complexity.Score)
		if err != nil {
			// Transparent fallback to Tier 2 on provider failure
			t2Resp, err2 := e.client.Complete(ctx, req, e.cfg.Tiers.Tier2, "tier2", complexity.Score)
			if err2 != nil {
				return nil, nil, fmt.Errorf("cascade failure: tier1 error (%v), tier2 error (%v)", err, err2)
			}
			routedTier = "tier2_escalated"
			escalated = true
			escalationReason = "tier1_upstream_unavailable"
			finalResp = t2Resp
			promptTokens = t2Resp.Usage.PromptTokens
			completionTokens = t2Resp.Usage.CompletionTokens

			baselineCost = e.t2CostCalc.ComputeCost(promptTokens, completionTokens)
			actualCost = baselineCost
			savedCost = 0.0
			t1Confidence = 0.0
		} else {
			// Stage 3: In-Flight Verification
			t1Content := ""
			var logprobs *provider.ChoiceLogProbs
			if len(t1Resp.Choices) > 0 {
				t1Content = t1Resp.Choices[0].Message.ContentString()
				logprobs = t1Resp.Choices[0].LogProbs
			}

			pLogprob := e.confAnalyzer.ComputeLogProbConfidence(logprobs)
			vSchema, schemaReason := e.schemaGate.EvaluateSchema(t1Content, req)
			hRef, refReason := e.heuristicChecker.CheckRefusalPenalty(t1Content)

			// Q_conf = alpha * P_logprob + beta * V_schema - gamma * H_ref
			qConf := (0.50 * pLogprob) + (0.35 * vSchema) - (0.50 * hRef)
			t1Confidence = math.Max(0.0, math.Min(1.0, qConf))

			if t1Confidence >= tauConfidence {
				// ACCEPT Tier 1
				routedTier = "tier1"
				escalated = false
				finalResp = t1Resp
				promptTokens = t1Resp.Usage.PromptTokens
				completionTokens = t1Resp.Usage.CompletionTokens

				baselineCost = e.t2CostCalc.ComputeCost(promptTokens, completionTokens)
				actualCost = e.t1CostCalc.ComputeCost(promptTokens, completionTokens)
				savedCost, _ = provider.ComputeSavings(baselineCost, actualCost)
			} else {
				// ESCALATE to Tier 2
				routedTier = "tier2_escalated"
				escalated = true
				if refReason != "" {
					escalationReason = refReason
				} else if schemaReason != "" {
					escalationReason = schemaReason
				} else {
					escalationReason = fmt.Sprintf("confidence_below_threshold (%.2f < %.2f)", t1Confidence, tauConfidence)
				}

				t2Resp, err2 := e.client.Complete(ctx, req, e.cfg.Tiers.Tier2, "tier2", complexity.Score)
				if err2 != nil {
					// If Tier 2 fails, deliver Tier 1 draft with best effort
					routedTier = "tier1"
					escalated = false
					finalResp = t1Resp
					promptTokens = t1Resp.Usage.PromptTokens
					completionTokens = t1Resp.Usage.CompletionTokens

					baselineCost = e.t2CostCalc.ComputeCost(promptTokens, completionTokens)
					actualCost = e.t1CostCalc.ComputeCost(promptTokens, completionTokens)
					savedCost, _ = provider.ComputeSavings(baselineCost, actualCost)
				} else {
					finalResp = t2Resp
					promptTokens = t2Resp.Usage.PromptTokens
					completionTokens = t2Resp.Usage.CompletionTokens

					baselineCost = e.t2CostCalc.ComputeCost(promptTokens, completionTokens)
					t1Spent := e.t1CostCalc.ComputeCost(t1Resp.Usage.PromptTokens, t1Resp.Usage.CompletionTokens)
					t2Spent := e.t2CostCalc.ComputeCost(promptTokens, completionTokens)
					actualCost = t1Spent + t2Spent
					savedCost, _ = provider.ComputeSavings(baselineCost, actualCost)
				}
			}
		}
	}

	latencyMs := time.Since(startTime).Milliseconds()

	record := &finops.RequestRecord{
		ID:               finalResp.ID,
		Timestamp:        time.Now(),
		PromptPreview:    promptPreview,
		ComplexityScore:  math.Round(complexity.Score*1000) / 1000,
		RoutedTier:       routedTier,
		Tier1Model:       e.cfg.Tiers.Tier1.Model,
		Tier2Model:       e.cfg.Tiers.Tier2.Model,
		Tier1Confidence:  math.Round(t1Confidence*1000) / 1000,
		Escalated:        escalated,
		EscalationReason: escalationReason,
		PromptTokens:     promptTokens,
		CompletionTokens: completionTokens,
		LatencyMs:        latencyMs,
		BaselineCostUSD:  baselineCost,
		ActualCostUSD:    actualCost,
		SavedCostUSD:     savedCost,
		Stream:           false,
	}

	if e.store != nil {
		go func(r finops.RequestRecord) {
			_ = e.store.RecordRequest(&r)
		}(*record)
	}

	if e.broadcaster != nil {
		e.broadcaster("new_request", record)
	}

	return finalResp, record, nil
}

// ExecuteStream performs speculative streaming routing.
func (e *Engine) ExecuteStream(ctx context.Context, req *provider.ChatCompletionRequest, chunkChan chan<- provider.StreamChunk) (*finops.RequestRecord, error) {
	startTime := time.Now()

	thetaBypass, tauConfidence := e.GetThresholds()
	complexity := e.scorer.Score(req, thetaBypass)
	promptPreview := getPromptPreview(req.Messages)

	// Direct Tier 2 bypass
	if complexity.BypassDirectToTier2 {
		record := &finops.RequestRecord{
			ID:               fmt.Sprintf("chatcmpl-stream-%d", time.Now().UnixNano()),
			Timestamp:        time.Now(),
			PromptPreview:    promptPreview,
			ComplexityScore:  math.Round(complexity.Score*1000) / 1000,
			RoutedTier:       "tier2_direct",
			Tier1Model:       e.cfg.Tiers.Tier1.Model,
			Tier2Model:       e.cfg.Tiers.Tier2.Model,
			Tier1Confidence:  1.0,
			Escalated:        false,
			PromptTokens:     100,
			CompletionTokens: 150,
			Stream:           true,
		}

		err := e.client.CompleteStream(ctx, req, e.cfg.Tiers.Tier2, "tier2", complexity.Score, chunkChan)
		record.LatencyMs = time.Since(startTime).Milliseconds()
		record.BaselineCostUSD = e.t2CostCalc.ComputeCost(record.PromptTokens, record.CompletionTokens)
		record.ActualCostUSD = record.BaselineCostUSD
		record.SavedCostUSD = 0.0

		if e.store != nil {
			go func(r finops.RequestRecord) { _ = e.store.RecordRequest(&r) }(*record)
		}
		if e.broadcaster != nil {
			e.broadcaster("new_request", record)
		}
		return record, err
	}

	// Speculative buffer channel
	t1Chan := make(chan provider.StreamChunk, 100)
	bufferLimit := e.cfg.Thresholds.StreamBufferTokens
	if bufferLimit <= 0 {
		bufferLimit = 16
	}

	go func() {
		_ = e.client.CompleteStream(ctx, req, e.cfg.Tiers.Tier1, "tier1", complexity.Score, t1Chan)
	}()

	var buffer []provider.StreamChunk
	var accumulatedContent strings.Builder
	var totalLogProb float64
	var logProbCount int
	isEscalating := false
	escalationReason := ""

	// Read buffer window
	for chunk := range t1Chan {
		buffer = append(buffer, chunk)
		for _, c := range chunk.Choices {
			accumulatedContent.WriteString(c.Delta.Content)
			if c.LogProbs != nil {
				for _, lp := range c.LogProbs.Content {
					totalLogProb += math.Exp(lp.LogProb)
					logProbCount++
				}
			}
		}

		// Check early refusal heuristic
		if penalty, reason := e.heuristicChecker.CheckRefusalPenalty(accumulatedContent.String()); penalty > 0.5 {
			isEscalating = true
			escalationReason = reason
			break
		}

		if len(buffer) >= bufferLimit {
			break
		}
	}

	// Evaluate confidence
	avgProb := 0.85
	if logProbCount > 0 {
		avgProb = totalLogProb / float64(logProbCount)
	}
	vSchema, schemaReason := e.schemaGate.EvaluateSchema(accumulatedContent.String(), req)
	if vSchema == 0.0 {
		isEscalating = true
		escalationReason = schemaReason
	}

	confidence := (0.6 * avgProb) + (0.4 * vSchema)
	if confidence < tauConfidence {
		isEscalating = true
		if escalationReason == "" {
			escalationReason = fmt.Sprintf("confidence_below_threshold (%.2f < %.2f)", confidence, tauConfidence)
		}
	}

	var record *finops.RequestRecord

	if isEscalating {
		// Pivot to Tier 2
		record = &finops.RequestRecord{
			ID:               fmt.Sprintf("chatcmpl-stream-%d", time.Now().UnixNano()),
			Timestamp:        time.Now(),
			PromptPreview:    promptPreview,
			ComplexityScore:  math.Round(complexity.Score*1000) / 1000,
			RoutedTier:       "tier2_escalated",
			Tier1Model:       e.cfg.Tiers.Tier1.Model,
			Tier2Model:       e.cfg.Tiers.Tier2.Model,
			Tier1Confidence:  math.Round(confidence*1000) / 1000,
			Escalated:        true,
			EscalationReason: escalationReason,
			PromptTokens:     120,
			CompletionTokens: 180,
			Stream:           true,
		}

		err := e.client.CompleteStream(ctx, req, e.cfg.Tiers.Tier2, "tier2", complexity.Score, chunkChan)
		record.LatencyMs = time.Since(startTime).Milliseconds()
		record.BaselineCostUSD = e.t2CostCalc.ComputeCost(record.PromptTokens, record.CompletionTokens)
		t1DraftCost := e.t1CostCalc.ComputeCost(record.PromptTokens, len(buffer))
		record.ActualCostUSD = t1DraftCost + record.BaselineCostUSD
		record.SavedCostUSD, _ = provider.ComputeSavings(record.BaselineCostUSD, record.ActualCostUSD)

		if e.store != nil {
			go func(r finops.RequestRecord) { _ = e.store.RecordRequest(&r) }(*record)
		}
		if e.broadcaster != nil {
			e.broadcaster("new_request", record)
		}
		return record, err
	}

	// Accepted Tier 1: flush buffered tokens then forward remaining stream
	defer close(chunkChan)

	for _, chunk := range buffer {
		chunkChan <- chunk
	}

	for chunk := range t1Chan {
		chunkChan <- chunk
	}

	record = &finops.RequestRecord{
		ID:               fmt.Sprintf("chatcmpl-stream-%d", time.Now().UnixNano()),
		Timestamp:        time.Now(),
		PromptPreview:    promptPreview,
		ComplexityScore:  math.Round(complexity.Score*1000) / 1000,
		RoutedTier:       "tier1",
		Tier1Model:       e.cfg.Tiers.Tier1.Model,
		Tier2Model:       e.cfg.Tiers.Tier2.Model,
		Tier1Confidence:  math.Round(confidence*1000) / 1000,
		Escalated:        false,
		PromptTokens:     100,
		CompletionTokens: 140,
		Stream:           true,
	}

	record.LatencyMs = time.Since(startTime).Milliseconds()
	record.BaselineCostUSD = e.t2CostCalc.ComputeCost(record.PromptTokens, record.CompletionTokens)
	record.ActualCostUSD = e.t1CostCalc.ComputeCost(record.PromptTokens, record.CompletionTokens)
	record.SavedCostUSD, _ = provider.ComputeSavings(record.BaselineCostUSD, record.ActualCostUSD)

	if e.store != nil {
		go func(r finops.RequestRecord) { _ = e.store.RecordRequest(&r) }(*record)
	}
	if e.broadcaster != nil {
		e.broadcaster("new_request", record)
	}

	return record, nil
}

func getPromptPreview(messages []provider.Message) string {
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == "user" {
			text := strings.TrimSpace(messages[i].ContentString())
			if len(text) > 90 {
				return text[:87] + "..."
			}
			return text
		}
	}
	return "Prompt message"
}
