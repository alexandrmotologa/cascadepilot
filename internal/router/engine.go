package router

import (
	"context"
	"fmt"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/alexandrmotologa/cascadepilot/internal/cache"
	"github.com/alexandrmotologa/cascadepilot/internal/config"
	"github.com/alexandrmotologa/cascadepilot/internal/finops"
	"github.com/alexandrmotologa/cascadepilot/internal/provider"
	"github.com/alexandrmotologa/cascadepilot/internal/verifier"
)

// BroadcasterFunc defines the signature for real-time SSE notifications.
type BroadcasterFunc func(event string, data interface{})

// CompareResult captures parallel evaluation results across tiers.
type CompareResult struct {
	Prompt       string              `json:"prompt"`
	CascadeRoute *finops.RequestRecord `json:"cascade_route"`
	Tier1Result  TierModelEvaluation `json:"tier1_result"`
	Tier2Result  TierModelEvaluation `json:"tier2_result"`
	Tier3Result  TierModelEvaluation `json:"tier3_result"`
}

// TierModelEvaluation holds response and cost metrics for a specific tier.
type TierModelEvaluation struct {
	Model            string  `json:"model"`
	Tier             string  `json:"tier"`
	Content          string  `json:"content"`
	LatencyMs        int64   `json:"latency_ms"`
	PromptTokens     int     `json:"prompt_tokens"`
	CompletionTokens int     `json:"completion_tokens"`
	CostUSD          float64 `json:"cost_usd"`
}

// Engine orchestrates speculative routing, Tier 0 caching, 3-tier cascade, and FinOps.
type Engine struct {
	cfg              *config.Config
	client           provider.ProviderClient
	scorer           *ComplexityScorer
	confAnalyzer     *verifier.ConfidenceAnalyzer
	heuristicChecker *verifier.HeuristicChecker
	schemaGate       *verifier.SchemaGate
	store            *finops.Store
	cache            *cache.PromptCache
	broadcaster      BroadcasterFunc

	// Dynamic thresholds
	mu            sync.RWMutex
	thetaBypass   float64
	tauConfidence float64

	t1CostCalc *provider.CostCalculator
	t2CostCalc *provider.CostCalculator
	t3CostCalc *provider.CostCalculator
}

// NewEngine initializes the cascading routing pipeline.
func NewEngine(
	cfg *config.Config,
	client provider.ProviderClient,
	store *finops.Store,
	promptCache *cache.PromptCache,
	broadcaster BroadcasterFunc,
) *Engine {
	theta := cfg.Thresholds.ThetaBypass
	tau := cfg.Thresholds.TauConfidence

	if store != nil {
		if storedTheta, storedTau, found, err := store.GetThresholds(); err == nil && found {
			theta = storedTheta
			tau = storedTau
		}
	}

	frontierTier := cfg.Tiers.Tier3
	if frontierTier.Model == "" {
		frontierTier = cfg.Tiers.Tier2
	}

	return &Engine{
		cfg:              cfg,
		client:           client,
		scorer:           NewComplexityScorer(),
		confAnalyzer:     verifier.NewConfidenceAnalyzer(),
		heuristicChecker: verifier.NewHeuristicChecker(),
		schemaGate:       verifier.NewSchemaGate(),
		store:            store,
		cache:            promptCache,
		broadcaster:      broadcaster,
		thetaBypass:      theta,
		tauConfidence:    tau,
		t1CostCalc:       provider.NewCostCalculator(cfg.Tiers.Tier1.CostPerMillionInput, cfg.Tiers.Tier1.CostPerMillionOutput),
		t2CostCalc:       provider.NewCostCalculator(cfg.Tiers.Tier2.CostPerMillionInput, cfg.Tiers.Tier2.CostPerMillionOutput),
		t3CostCalc:       provider.NewCostCalculator(frontierTier.CostPerMillionInput, frontierTier.CostPerMillionOutput),
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
	return e.ExecuteWithTenant(ctx, req, "tenant_default", "General")
}

// ExecuteWithTenant executes routing with explicit multi-tenant attribution.
func (e *Engine) ExecuteWithTenant(ctx context.Context, req *provider.ChatCompletionRequest, tenantID, department string) (*provider.ChatCompletionResponse, *finops.RequestRecord, error) {
	startTime := time.Now()
	promptPreview := getPromptPreview(req.Messages)

	// Step 1: Tier 0 Cache Inspection
	if e.cache != nil && e.cfg.Cache.Enabled && !req.Stream {
		if cachedResp, hit := e.cache.Get(req); hit {
			baselineCost := e.t3CostCalc.ComputeCost(cachedResp.Usage.PromptTokens, cachedResp.Usage.CompletionTokens)
			record := &finops.RequestRecord{
				ID:               cachedResp.ID,
				Timestamp:        time.Now(),
				PromptPreview:    promptPreview,
				ComplexityScore:  0.0,
				RoutedTier:       "tier0_cache",
				Tier1Model:       "cache-hit",
				Tier2Model:       "cache-hit",
				Tier1Confidence:  1.0,
				Escalated:        false,
				PromptTokens:     cachedResp.Usage.PromptTokens,
				CompletionTokens: cachedResp.Usage.CompletionTokens,
				LatencyMs:        time.Since(startTime).Milliseconds(),
				BaselineCostUSD:  baselineCost,
				ActualCostUSD:    0.0,
				SavedCostUSD:     baselineCost,
				Stream:           false,
				TenantID:         tenantID,
				Department:       department,
				CacheHit:         true,
			}

			if e.store != nil {
				go func(r finops.RequestRecord) { _ = e.store.RecordRequest(&r) }(*record)
			}
			if e.broadcaster != nil {
				e.broadcaster("new_request", record)
			}
			return cachedResp, record, nil
		}
	}

	thetaBypass, tauConfidence := e.GetThresholds()
	complexity := e.scorer.Score(req, thetaBypass)

	// Step 2: Circuit Breaker Check
	circuitBreakerTripped := false
	if e.store != nil && e.cfg.Budget.MonthlyCapUSD > 0 {
		if monthSpend, err := e.store.GetGlobalSpendThisMonth(); err == nil && monthSpend >= e.cfg.Budget.MonthlyCapUSD {
			circuitBreakerTripped = true
		}
	}

	frontierTier := e.cfg.Tiers.Tier3
	if frontierTier.Model == "" {
		frontierTier = e.cfg.Tiers.Tier2
	}

	var finalResp *provider.ChatCompletionResponse
	var routedTier string
	var escalated bool
	var escalationReason string
	var t1Confidence float64
	var promptTokens, completionTokens int
	var baselineCost, actualCost, savedCost float64

	// Stage 1: Complexity Bypass (unless circuit breaker forces Tier 1 downgrade)
	if complexity.BypassDirectToTier2 && !circuitBreakerTripped {
		routedTier = "tier3_direct"
		resp, err := e.client.Complete(ctx, req, frontierTier, "tier3", complexity.Score)
		if err != nil {
			return nil, nil, fmt.Errorf("tier 3 direct execution error: %w", err)
		}
		finalResp = resp
		promptTokens = resp.Usage.PromptTokens
		completionTokens = resp.Usage.CompletionTokens

		baselineCost = e.t3CostCalc.ComputeCost(promptTokens, completionTokens)
		actualCost = baselineCost
		savedCost = 0.0
		t1Confidence = 1.0
	} else {
		// Stage 2: Speculative Tier 1 Execution (SLM)
		t1Resp, err := e.client.Complete(ctx, req, e.cfg.Tiers.Tier1, "tier1", complexity.Score)
		if err != nil {
			// Failover to Tier 2 Mid-tier
			t2Resp, err2 := e.client.Complete(ctx, req, e.cfg.Tiers.Tier2, "tier2", complexity.Score)
			if err2 != nil {
				// Fallback to Frontier
				t3Resp, err3 := e.client.Complete(ctx, req, frontierTier, "tier3", complexity.Score)
				if err3 != nil {
					return nil, nil, fmt.Errorf("all tiers failed: %v, %v, %v", err, err2, err3)
				}
				finalResp = t3Resp
				routedTier = "tier3_escalated"
			} else {
				finalResp = t2Resp
				routedTier = "tier2_mid"
			}
			escalated = true
			escalationReason = "tier1_upstream_unavailable"
			promptTokens = finalResp.Usage.PromptTokens
			completionTokens = finalResp.Usage.CompletionTokens
			baselineCost = e.t3CostCalc.ComputeCost(promptTokens, completionTokens)
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

			qConf := (0.50 * pLogprob) + (0.35 * vSchema) - (0.50 * hRef)
			t1Confidence = math.Max(0.0, math.Min(1.0, qConf))

			if t1Confidence >= tauConfidence || circuitBreakerTripped {
				// Accept Tier 1
				routedTier = "tier1"
				escalated = false
				finalResp = t1Resp
				promptTokens = t1Resp.Usage.PromptTokens
				completionTokens = t1Resp.Usage.CompletionTokens

				baselineCost = e.t3CostCalc.ComputeCost(promptTokens, completionTokens)
				actualCost = e.t1CostCalc.ComputeCost(promptTokens, completionTokens)
				savedCost, _ = provider.ComputeSavings(baselineCost, actualCost)
			} else {
				// Step 4: Progressive 3-Tier Escalation
				if refReason != "" {
					escalationReason = refReason
				} else if schemaReason != "" {
					escalationReason = schemaReason
				} else {
					escalationReason = fmt.Sprintf("confidence_below_threshold (%.2f < %.2f)", t1Confidence, tauConfidence)
				}

				// If moderate complexity (<= 0.65), escalate first to Tier 2 Mid-tier ($0.15/1M)
				if complexity.Score <= 0.65 && e.cfg.Tiers.Tier2.Model != "" {
					t2Resp, err2 := e.client.Complete(ctx, req, e.cfg.Tiers.Tier2, "tier2", complexity.Score)
					if err2 == nil {
						routedTier = "tier2_mid"
						escalated = true
						finalResp = t2Resp
						promptTokens = t2Resp.Usage.PromptTokens
						completionTokens = t2Resp.Usage.CompletionTokens

						baselineCost = e.t3CostCalc.ComputeCost(promptTokens, completionTokens)
						t1Spent := e.t1CostCalc.ComputeCost(t1Resp.Usage.PromptTokens, t1Resp.Usage.CompletionTokens)
						t2Spent := e.t2CostCalc.ComputeCost(promptTokens, completionTokens)
						actualCost = t1Spent + t2Spent
						savedCost, _ = provider.ComputeSavings(baselineCost, actualCost)
					} else {
						// Fallback directly to Frontier Tier 3
						routedTier = "tier3_escalated"
						escalated = true
						t3Resp, _ := e.client.Complete(ctx, req, frontierTier, "tier3", complexity.Score)
						finalResp = t3Resp
						promptTokens = t3Resp.Usage.PromptTokens
						completionTokens = t3Resp.Usage.CompletionTokens
						baselineCost = e.t3CostCalc.ComputeCost(promptTokens, completionTokens)
						t1Spent := e.t1CostCalc.ComputeCost(t1Resp.Usage.PromptTokens, t1Resp.Usage.CompletionTokens)
						t3Spent := e.t3CostCalc.ComputeCost(promptTokens, completionTokens)
						actualCost = t1Spent + t3Spent
						savedCost, _ = provider.ComputeSavings(baselineCost, actualCost)
					}
				} else {
					// High complexity: escalate directly to Tier 3 Frontier
					routedTier = "tier3_escalated"
					escalated = true
					t3Resp, err3 := e.client.Complete(ctx, req, frontierTier, "tier3", complexity.Score)
					if err3 != nil {
						finalResp = t1Resp
						routedTier = "tier1"
						escalated = false
					} else {
						finalResp = t3Resp
						promptTokens = t3Resp.Usage.PromptTokens
						completionTokens = t3Resp.Usage.CompletionTokens
						baselineCost = e.t3CostCalc.ComputeCost(promptTokens, completionTokens)
						t1Spent := e.t1CostCalc.ComputeCost(t1Resp.Usage.PromptTokens, t1Resp.Usage.CompletionTokens)
						t3Spent := e.t3CostCalc.ComputeCost(promptTokens, completionTokens)
						actualCost = t1Spent + t3Spent
						savedCost, _ = provider.ComputeSavings(baselineCost, actualCost)
					}
				}
			}
		}
	}

	latencyMs := time.Since(startTime).Milliseconds()

	// Store in Tier 0 cache for future identical requests
	if e.cache != nil && e.cfg.Cache.Enabled && finalResp != nil {
		e.cache.Set(req, finalResp)
	}

	record := &finops.RequestRecord{
		ID:               finalResp.ID,
		Timestamp:        time.Now(),
		PromptPreview:    promptPreview,
		ComplexityScore:  math.Round(complexity.Score*1000) / 1000,
		RoutedTier:       routedTier,
		Tier1Model:       e.cfg.Tiers.Tier1.Model,
		Tier2Model:       frontierTier.Model,
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
		TenantID:         tenantID,
		Department:       department,
		CacheHit:         false,
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

// CompareModels executes the query against Tier 1, Tier 2, Tier 3, and Cascadepilot in parallel.
func (e *Engine) CompareModels(ctx context.Context, req *provider.ChatCompletionRequest) (*CompareResult, error) {
	var wg sync.WaitGroup
	wg.Add(4)

	frontierTier := e.cfg.Tiers.Tier3
	if frontierTier.Model == "" {
		frontierTier = e.cfg.Tiers.Tier2
	}

	var cascadeRec *finops.RequestRecord
	var t1Eval, t2Eval, t3Eval TierModelEvaluation

	// 1. Cascadepilot Cascade
	go func() {
		defer wg.Done()
		_, rec, _ := e.Execute(ctx, req)
		cascadeRec = rec
	}()

	// 2. Pure Tier 1
	go func() {
		defer wg.Done()
		start := time.Now()
		resp, err := e.client.Complete(ctx, req, e.cfg.Tiers.Tier1, "tier1", 0.3)
		if err == nil {
			cost := e.t1CostCalc.ComputeCost(resp.Usage.PromptTokens, resp.Usage.CompletionTokens)
			t1Eval = TierModelEvaluation{
				Model:            e.cfg.Tiers.Tier1.Model,
				Tier:             "Tier 1 (SLM)",
				Content:          resp.Choices[0].Message.ContentString(),
				LatencyMs:        time.Since(start).Milliseconds(),
				PromptTokens:     resp.Usage.PromptTokens,
				CompletionTokens: resp.Usage.CompletionTokens,
				CostUSD:          cost,
			}
		}
	}()

	// 3. Pure Tier 2 (Mid-tier)
	go func() {
		defer wg.Done()
		start := time.Now()
		resp, err := e.client.Complete(ctx, req, e.cfg.Tiers.Tier2, "tier2", 0.5)
		if err == nil {
			cost := e.t2CostCalc.ComputeCost(resp.Usage.PromptTokens, resp.Usage.CompletionTokens)
			t2Eval = TierModelEvaluation{
				Model:            e.cfg.Tiers.Tier2.Model,
				Tier:             "Tier 2 (Mid-tier)",
				Content:          resp.Choices[0].Message.ContentString(),
				LatencyMs:        time.Since(start).Milliseconds(),
				PromptTokens:     resp.Usage.PromptTokens,
				CompletionTokens: resp.Usage.CompletionTokens,
				CostUSD:          cost,
			}
		}
	}()

	// 4. Pure Tier 3 (Frontier)
	go func() {
		defer wg.Done()
		start := time.Now()
		resp, err := e.client.Complete(ctx, req, frontierTier, "tier3", 0.9)
		if err == nil {
			cost := e.t3CostCalc.ComputeCost(resp.Usage.PromptTokens, resp.Usage.CompletionTokens)
			t3Eval = TierModelEvaluation{
				Model:            frontierTier.Model,
				Tier:             "Tier 3 (Frontier)",
				Content:          resp.Choices[0].Message.ContentString(),
				LatencyMs:        time.Since(start).Milliseconds(),
				PromptTokens:     resp.Usage.PromptTokens,
				CompletionTokens: resp.Usage.CompletionTokens,
				CostUSD:          cost,
			}
		}
	}()

	wg.Wait()

	promptText := ""
	if len(req.Messages) > 0 {
		promptText = req.Messages[len(req.Messages)-1].ContentString()
	}

	return &CompareResult{
		Prompt:       promptText,
		CascadeRoute: cascadeRec,
		Tier1Result:  t1Eval,
		Tier2Result:  t2Eval,
		Tier3Result:  t3Eval,
	}, nil
}

// ExecuteStream performs speculative streaming routing.
func (e *Engine) ExecuteStream(ctx context.Context, req *provider.ChatCompletionRequest, chunkChan chan<- provider.StreamChunk) (*finops.RequestRecord, error) {
	startTime := time.Now()

	thetaBypass, tauConfidence := e.GetThresholds()
	complexity := e.scorer.Score(req, thetaBypass)
	promptPreview := getPromptPreview(req.Messages)

	frontierTier := e.cfg.Tiers.Tier3
	if frontierTier.Model == "" {
		frontierTier = e.cfg.Tiers.Tier2
	}

	// Direct Frontier bypass
	if complexity.BypassDirectToTier2 {
		record := &finops.RequestRecord{
			ID:               fmt.Sprintf("chatcmpl-stream-%d", time.Now().UnixNano()),
			Timestamp:        time.Now(),
			PromptPreview:    promptPreview,
			ComplexityScore:  math.Round(complexity.Score*1000) / 1000,
			RoutedTier:       "tier3_direct",
			Tier1Model:       e.cfg.Tiers.Tier1.Model,
			Tier2Model:       frontierTier.Model,
			Tier1Confidence:  1.0,
			Escalated:        false,
			PromptTokens:     100,
			CompletionTokens: 150,
			Stream:           true,
		}

		err := e.client.CompleteStream(ctx, req, frontierTier, "tier3", complexity.Score, chunkChan)
		record.LatencyMs = time.Since(startTime).Milliseconds()
		record.BaselineCostUSD = e.t3CostCalc.ComputeCost(record.PromptTokens, record.CompletionTokens)
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

		if penalty, reason := e.heuristicChecker.CheckRefusalPenalty(accumulatedContent.String()); penalty > 0.5 {
			isEscalating = true
			escalationReason = reason
			break
		}

		if len(buffer) >= bufferLimit {
			break
		}
	}

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
		record = &finops.RequestRecord{
			ID:               fmt.Sprintf("chatcmpl-stream-%d", time.Now().UnixNano()),
			Timestamp:        time.Now(),
			PromptPreview:    promptPreview,
			ComplexityScore:  math.Round(complexity.Score*1000) / 1000,
			RoutedTier:       "tier3_escalated",
			Tier1Model:       e.cfg.Tiers.Tier1.Model,
			Tier2Model:       frontierTier.Model,
			Tier1Confidence:  math.Round(confidence*1000) / 1000,
			Escalated:        true,
			EscalationReason: escalationReason,
			PromptTokens:     120,
			CompletionTokens: 180,
			Stream:           true,
		}

		err := e.client.CompleteStream(ctx, req, frontierTier, "tier3", complexity.Score, chunkChan)
		record.LatencyMs = time.Since(startTime).Milliseconds()
		record.BaselineCostUSD = e.t3CostCalc.ComputeCost(record.PromptTokens, record.CompletionTokens)
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

	// Accepted Tier 1
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
		Tier2Model:       frontierTier.Model,
		Tier1Confidence:  math.Round(confidence*1000) / 1000,
		Escalated:        false,
		PromptTokens:     100,
		CompletionTokens: 140,
		Stream:           true,
	}

	record.LatencyMs = time.Since(startTime).Milliseconds()
	record.BaselineCostUSD = e.t3CostCalc.ComputeCost(record.PromptTokens, record.CompletionTokens)
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
