# cascadepilot: Architecture and Engineering Master Plan
Adaptive Speculative LLM Cascading Router and Cost-Minimization Gateway for Enterprise Workloads
Target: Single-Binary Go Reverse Proxy (< 20MB RAM, < 5ms routing overhead) with Embedded React 19 FinOps Studio and Quality Parity Tracker

---

## 1. Problem Statement and Motivation

### 1.1 The Economics of Enterprise LLM Workloads
As organizations run generative AI in production, API expenditures often turn into an operational challenge. Most setups rely on static model choices, configuring applications to query frontier models regardless of query difficulty:
1. Workload Distribution: About 75% to 85% of production prompts involve routine tasks such as data extraction, text classification, simple summarization, or formatting. Small language models (SLMs) like Gemini Flash, Llama 3.1 8B, or GPT-4o-mini can resolve these tasks at a fraction of the cost.
2. Limitations of Static Routing: Hardcoded rules based on endpoint paths or basic regex patterns fail because query complexity varies within the same endpoint.
3. Fear of Quality Degradation: Teams default to expensive models out of concern that smaller models will fail on difficult edge cases.
4. Absence of Automatic Escalation: Standard architectures lack an in-flight mechanism to evaluate confidence and escalate low-certainty outputs to frontier models transparently.

### 1.2 The cascadepilot Solution
cascadepilot is a speculative cascading reverse proxy in Go. It operates on the network layer, accepts standard OpenAI and Anthropic API requests, and executes dynamic two-tier speculative routing:
- Prompt Complexity Scoring: Evaluates incoming requests in under 2 milliseconds using token length, syntactic constraints, and reasoning markers.
- Speculative Tier 1 Execution: Dispatches queries to low-cost Tier 1 models ($0.075 to $0.15 per 1M tokens).
- In-Flight Quality and Confidence Verification: Inspects Tier 1 output quality using token log-probabilities, refusal detection, and schema validation.
- Transparent Tier 2 Escalation: If Tier 1 confidence is below the configured threshold, the proxy escalates the query to a Tier 2 frontier model ($3.00 to $15.00 per 1M tokens), returning the completion without breaking client compatibility.
- Embedded FinOps Studio: Includes a React 19 dashboard displaying dollar savings, routing ratios, latency distributions, and interactive threshold tuning.

---

## 2. High-Level System Architecture

```
       +-------------------------------------------------------------+
       |               Client Application / Agent / UI               |
       |       (Configured with baseUrl: http://localhost:8088)      |
       +-------------------------------------------------------------+
                                      |
                         Standard Chat Completion API
                                      v
       +-------------------------------------------------------------+
       |                   cascadepilot Reverse Proxy                |
       |                   (High-Performance Go CLI)                |
       |                                                             |
       |  +-------------------------------------------------------+  |
       |  |             Inbound Request Complexity Scorer         |  |
       |  |  - Prompt Token Length and Lexical Density            |  |
       |  |  - Multi-Step Reasoning and Constraint Detection      |  |
       |  +-------------------------------------------------------+  |
       |                               |                             |
       |              Complexity Score < Threshold                   |
       |                               v                             |
       |  +-------------------------------------------------------+  |
       |  |              Tier 1: Fast and Low-Cost Engine         |  |
       |  |       (Gemini Flash, Llama 3.1 8B, GPT-4o-mini, Mock) |  |
       |  +-------------------------------------------------------+  |
       |                               |                             |
       |                               v                             |
       |  +-------------------------------------------------------+  |
       |  |            Speculative Quality and Confidence Gate    |  |
       |  |  - Average Log-Probability Margin                     |  |
       |  |  - Structural Refusal and Hedging Detection           |  |
       |  |  - JSON and Schema Format Validation                  |  |
       |  |  - Length and Completeness Verification               |  |
       |  +-------------------------------------------------------+  |
       |               /                               \             |
       |   Confidence >= Threshold           Confidence < Threshold  |
       |             /                                   \           |
       |            v                                     v          |
       |   +------------------+                 +-----------------+  |
       |   | Return Tier 1    |                 | Escalate to     |  |
       |   | Direct to Client |                 | Tier 2 Frontier |  |
       |   | (Save 80-90%)    |                 | (Claude / GPT-4)|  |
       |   +------------------+                 +-----------------+  |
       |            \                                     /          |
       |             +-----------------+-----------------+           |
       |                               |                             |
       +-------------------------------|-----------------------------+
                                       |
                                       v
       +-------------------------------------------------------------+
       |                     FinOps and Telemetry                    |
       |  1. Transparent Response Delivery to Client                 |
       |  2. Async Telemetry Logged to SQLite (modernc.org/sqlite)   |
       |  3. Embedded React 19 FinOps Studio (Local HTTP :8088)      |
       |  4. Live SSE Stream and Dynamic Threshold Configuration     |
       +-------------------------------------------------------------+
```

---

## 3. Mathematical Formulation

### 3.1 Prompt Complexity Scoring (S_comp)
For prompt P with estimated token count L_P, syntactic constraint count C_P, and reasoning indicator density R_P:

S_comp(P) = w1 * min(1.0, L_P / 2048) + w2 * sigma(C_P) + w3 * R_P

Weights default to w1 = 0.35, w2 = 0.35, w3 = 0.30.
If S_comp(P) > theta_bypass (default 0.75), Tier 1 is bypassed directly to Tier 2 to avoid redundant latency on complex tasks.

### 3.2 Output Confidence Verification Metric (Q_conf)
For generated response T1 = {t1, t2, ..., tN} with token log-probabilities log p(ti), refusal heuristic penalty H_ref, and schema validity V_schema:

Q_conf(T1) = alpha * (1 / N * sum(e^(log p(ti)))) + beta * V_schema - gamma * H_ref

The routing decision follows:
Route(P) = Accept T1 if Q_conf(T1) >= tau, else Dispatch Tier 2 (T2).

### 3.3 Cumulative FinOps Savings (S_cost)
Let C1 and C2 represent the cost per token of Tier 1 and Tier 2 respectively:

S_cost = sum(Tokens_k * C2) - sum(Tokens_{k,1} * C1 + (1 - rho_k) * Tokens_{k,2} * C2)

Net savings typically range between 70% and 85% for standard enterprise distributions.

---

## 4. Repository Structure

```
cascadepilot/
├── cmd/
│   └── cascadepilot/
│       └── main.go                 # CLI entry point (serve, benchmark, simulate)
├── internal/
│   ├── config/
│   │   ├── config.go               # YAML configuration loader
│   │   └── schema.go               # Model definitions, cost parameters, thresholds
│   ├── router/
│   │   ├── engine.go               # Cascading orchestrator and decision pipeline
│   │   ├── complexity.go           # Fast prompt complexity evaluator (< 2ms)
│   │   └── fallback.go             # Resilient cross-provider fallback handler
│   ├── verifier/
│   │   ├── confidence.go           # Token logprob entropy and probability analyzer
│   │   ├── heuristic.go            # Structural refusal and hedging detector
│   │   └── schema_gate.go          # Fast JSON and schema compliance checker
│   ├── provider/
│   │   ├── types.go                # Unified completion request and response types
│   │   ├── client.go               # Multi-provider client (OpenAI, Anthropic, Gemini, Ollama)
│   │   ├── mock.go                 # Synthetic provider for deterministic offline testing
│   │   ├── sse_forwarder.go        # Streaming chunk forwarder and buffer
│   │   └── pricing.go              # Real-time token cost calculation table per model
│   ├── finops/
│   │   ├── store.go                # Pure Go embedded SQLite store (modernc.org/sqlite)
│   │   ├── ledger.go               # Cost savings, latency delta, and ratio aggregator
│   │   └── export.go               # CSV and JSON FinOps reporting exporter
│   └── server/
│       ├── proxy_handler.go        # HTTP handler matching /v1/chat/completions and /v1/messages
│       ├── api.go                  # Telemetry, configuration, and simulation REST API
│       ├── embed.go                # Go embed filesystem for React 19 studio
│       └── sse_hub.go              # Real-time SSE stream for dashboard charts
├── web/
│   ├── src/
│   │   ├── App.tsx                 # React 19 root application
│   │   ├── components/
│   │   │   ├── SavingsHero.tsx     # Cumulative dollar savings and ROI multiplier card
│   │   │   ├── RoutingWaterfall.tsx# Tier 1 vs Tier 2 volume and escalation ratio chart
│   │   │   ├── LatencyDiffChart.tsx# Latency distribution comparison
│   │   │   ├── ThresholdSlider.tsx # Interactive threshold simulation and live updates
│   │   │   ├── RequestLiveLog.tsx  # Real-time table of routed queries with decision details
│   │   │   └── ProxyPlayground.tsx # Interactive query tester within the studio
│   │   └── pages/
│   │       └── Dashboard.tsx       # Main FinOps executive cockpit
│   ├── package.json
│   ├── tsconfig.json
│   ├── vite.config.ts
│   └── index.html
├── docs/
│   ├── ARCHITECTURE.md             # Detailed technical architecture guide
│   ├── BENCHMARKS.md               # Empirical benchmark results and latency overhead
│   └── FINOPS_FORMULAS.md          # Formal cost models and parity metrics
├── examples/
│   ├── cascadepilot.yaml           # Production configuration sample
│   └── query_benchmark.jsonl       # Test queries across diverse difficulty tiers
├── Makefile
├── Dockerfile
├── go.mod
├── go.sum
├── LICENSE
└── README.md
```

---

## 5. Detailed Implementation Roadmap

### Phase 1: Go Engine Foundation and Provider Client
- Initialize Go module with minimal dependencies.
- Build unified client supporting OpenAI, Anthropic, Gemini, Ollama, and a synthetic mock provider for deterministic offline tests.
- Implement cost tracking table calculating exact USD expenditures per request based on input and output tokens.

### Phase 2: Inbound Prompt Complexity Analyzer
- Implement `internal/router/complexity.go` computing prompt length, constraint markers, and semantic indicators in under 2 milliseconds.
- Build bypass rules allowing prompts with extreme complexity or explicit bypass tags to skip directly to Tier 2.

### Phase 3: In-Flight Speculative Verification Engine
- Build `internal/verifier/confidence.go` evaluating token log-probabilities, formatting compliance, and absence of generic refusals or evasions.
- Set up configurable confidence threshold (tau) and bypass threshold (theta_bypass) in configuration.

### Phase 4: Transparent Reverse Proxy and Escalation Pipeline
- Build HTTP reverse proxy intercepting `/v1/chat/completions` and `/v1/messages`.
- Dispatch query to Tier 1; if verification passes, stream or return to client immediately.
- If verification fails, discard Tier 1 draft, dispatch to Tier 2, and return final completion without breaking client connection.

### Phase 5: FinOps Accounting and SQLite Storage
- Implement `modernc.org/sqlite` database recording timestamp, complexity score, Tier 1 model, Tier 2 model, escalation status, latency, and cost saved.
- Build REST API endpoints serving aggregated FinOps metrics, live SSE updates, and CSV/JSON export.

### Phase 6: Embedded React 19 FinOps Studio
- Build modern web UI in `web/` featuring savings cards, routing percentage gauges, latency comparison charts, interactive threshold sliders, live request inspector, and direct proxy playground.
- Embed frontend static assets into the Go binary using `//go:embed` and serve locally on `:8088`.

### Phase 7: Verification, Benchmarking, and Documentation
- Run simulation on `examples/query_benchmark.jsonl` verifying cost reduction while maintaining quality parity.
- Benchmark proxy overhead verifying added latency remains below 5ms on Tier 1 hits.
- Author humanized documentation, ARCHITECTURE.md, BENCHMARKS.md, FINOPS_FORMULAS.md, and production README.md.
