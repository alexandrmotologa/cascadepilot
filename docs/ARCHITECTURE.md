# cascadepilot System Architecture

This document describes the architectural design of cascadepilot, an adaptive speculative reverse proxy designed for enterprise language model workloads.

## Core Concepts

In enterprise environments, standard LLM applications invoke expensive frontier models for all prompts. cascadepilot operates as an inline HTTP reverse proxy between client applications (OpenAI SDK, LangChain, LlamaIndex, custom agents) and upstream model providers.

The gateway routes requests through a four-stage optimization pipeline:
1. Tier 0 Prompt Caching: Normalized SHA-256 cache evaluation returning instant completions at zero cost ($0.00) in under 1 millisecond.
2. Inbound Complexity Scoring: Evaluates prompt length, reasoning indicators, and syntactic constraints in under 0.1 milliseconds.
3. Speculative Multi-Tier Execution: Dispatches queries to Tier 1 (SLM). If uncertain, cascades to Tier 2 (Mid-tier reasoning) or Tier 3 (Frontier flagship).
4. In-Flight Verification: Evaluates output confidence using token log-probabilities, semantic heuristics, and schema validity before socket delivery.
5. Department Chargeback and FinOps: Maps usage to virtual API keys (`cp-live-...`), tracking spend against team budgets.

## Component Breakdown

### 1. Tier 0 Normalized Prompt Cache (`internal/cache/cache.go`)
Before calling any model, the gateway hashes incoming messages and parameters (temperature, response format) using SHA-256. If a matching entry exists within the TTL window, the cached completion is returned immediately:
- Cost: $0.00
- Latency: < 1.0 ms
- Routing tier tagged: `tier0_cache`

### 2. Inbound Complexity Scorer (`internal/router/complexity.go`)
When a cache miss occurs, the scorer evaluates prompt complexity S_comp without external network roundtrips:
- Token length ratio: Normalized against an empirical ceiling of 2048 tokens.
- Constraint density: Detects JSON schemas, XML wrappers, markdown specifications, and negative constraints.
- Reasoning intensity: Matches keywords associated with mathematical proofs, distributed systems, concurrency, formal analysis, and multi-step deduction.

If S_comp exceeds the bypass threshold (theta_bypass), the query skips smaller tiers and routes directly to Tier 3 Frontier.

### 3. Progressive 3-Tier Cascade (`internal/router/engine.go`)
- Tier 1 (SLM): Rapid small language models (Gemini Flash, Llama 8B) handling 70% to 80% of routine enterprise traffic at $0.075 to $0.15 per 1M tokens.
- Tier 2 (Mid-tier Reasoning): Capable models (GPT-4o-mini, Claude 3.5 Haiku) resolving moderate reasoning tasks at $0.15 to $0.60 per 1M tokens.
- Tier 3 (Frontier Flagship): Deep frontier models (Claude 3.5 Sonnet, GPT-4o, o1) reserved for intricate edge cases at $3.00 to $15.00 per 1M tokens.

When Tier 1 confidence fails to clear tau:
- If prompt complexity is moderate (S_comp <= 0.65): Escalates to Tier 2 Mid-tier, saving 90% compared to pure frontier routing.
- If prompt complexity is high (S_comp > 0.65): Escalates directly to Tier 3 Frontier.

### 4. Verification Engine (`internal/verifier/`)
Evaluates generated drafts before delivery:
- Confidence Scorer (`confidence.go`): Computes average linear probability from token log-probabilities.
- Heuristic Gate (`heuristic.go`): Detects evasion, refusal phrases, apologies, and repetitive loops.
- Schema Validator (`schema_gate.go`): Validates JSON syntax and schema completeness.

### 5. Multi-Tenant Cost Allocation (`internal/finops/`)
The proxy supports virtual API keys formatted as `Bearer cp-live-<dept>-<hash>`:
- Maps requests to organizational departments (Engineering, Customer Support, Marketing, Data Science).
- Monitors monthly expenditures against allocated department budget caps.
- Triggers circuit breaker protection if total spend approaches configured thresholds.

### 6. Embedded FinOps Cockpit (`web/` and `internal/server/embed.go`)
A single Go binary embeds the React 19 dashboard, exposing:
- Overview: Live SSE telemetry, waterfall progression, latency histograms, and hyperparameter sliders.
- Department Chargeback: Real-time spending versus budget bars and virtual key provisioning.
- Executive ROI Calculator: Interactive models projecting annual dollar savings and quality parity retention.
- Side-by-Side Comparator: Parallel execution comparing Tier 1, Mid-tier, Frontier, and Cascadepilot on identical prompts.
