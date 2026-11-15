# cascadepilot: Architecture and Engineering Master Plan
> Adaptive Speculative LLM Cascading Router & Cost-Minimization Gateway for Enterprise Workloads
> Target: High-Performance Single-Binary Go Reverse Proxy (< 20MB RAM, < 5ms routing overhead) with Embedded React 19 FinOps Studio & Quality Parity Tracker

---

## 1. Problem Statement and Motivation

### 1.1 The Unsustainable Economics of Enterprise LLM Workloads
As enterprises scale generative AI into production, API costs quickly become the primary operational bottleneck. Most organizations configure applications with a static model choice—typically a top-tier frontier model such as Claude 3.5 Sonnet, GPT-4o, or OpenAI o1:
1. **The 80/20 Enterprise Workload Imbalance**: Empirical analysis reveals that 75% to 85% of production prompts are routine tasks: entity extraction, text classification, simple FAQ summarization, or standardized formatting. These tasks can be answered by low-cost small language models (SLMs) such as Gemini 1.5 Flash, Llama 3.1 8B, or GPT-4o-mini at 1/50th of the cost.
2. **The Pitfall of Static Hardcoded Routing**: Hardcoding routing rules based on endpoint URLs or static regex patterns fails because user query complexity varies wildly within the same endpoint. A customer service query can range from a trivial "What is your return window?" to a multi-paragraph edge case involving conflicting contractual warranties.
3. **Quality Regression Fear**: Engineering teams default to expensive frontier models out of fear that smaller models will fail on difficult edge cases, leaving millions of dollars in unoptimized API expenditures on the table.
4. **Lack of Automated Fallback & Escalation**: In standard architectures, when a cheaper model generates an uncertain or low-confidence response, the application either delivers a subpar answer to the user or fails outright, because there is no transparent, in-flight escalation mechanism to a stronger model.

### 1.2 The cascadepilot Solution
`cascadepilot` is a speculative cascading reverse proxy written in Go. It operates transparently on the network layer, accepting standard OpenAI and Anthropic API requests, and executes dynamic two-tier speculative routing:
- **Zero-Latency Intent & Complexity Scoring**: Analyzes incoming prompts in under 5 milliseconds using lexical heuristics, token entropy, and lightweight embedding classifiers to predict required reasoning depth.
- **Speculative Tier 1 Execution**: Dispatches queries to ultra-fast, low-cost Tier 1 models (\$0.075 to \$0.15 per 1M tokens).
- **In-Flight Confidence & Grounding Verification**: Evaluates Tier 1 output quality using average log-probability margins, structural format adherence, and semantic certainty heuristics.
- **Transparent Tier 2 Escalation**: If Tier 1 confidence fails to clear the configured threshold ($\tau$), the proxy instantly and transparently escalates the query to a Tier 2 frontier model (\$3.00 to \$15.00 per 1M tokens), returning the premium completion without client application awareness.
- **Embedded FinOps Studio**: Bundles a React 19 analytics dashboard displaying cumulative dollar savings, routing ratios, latency distributions, and quality parity benchmarks.

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
       |  |  - Prompt Token Length & Lexical Density              |  |
       |  |  - Multi-Step Reasoning & Constraint Detection        |  |
       |  +-------------------------------------------------------+  |
       |                               |                             |
       |              Complexity Score < Threshold                   |
       |                               v                             |
       |  +-------------------------------------------------------+  |
       |  |              Tier 1: Fast & Low-Cost Engine           |  |
       |  |       (Gemini 1.5 Flash, Llama 3.1 8B, GPT-4o-mini)   |  |
       |  +-------------------------------------------------------+  |
       |                               |                             |
       |                               v                             |
       |  +-------------------------------------------------------+  |
       |  |            Speculative Quality & Confidence Gate      |  |
       |  |  - Average Log-Probability Margin                     |  |
       |  |  - JSON / Schema Validation (if applicable)           |  |
       |  |  - Length and Completeness Sanity                     |  |
       |  +-------------------------------------------------------+  |
       |               /                               \             |
       |   Confidence >= Threshold           Confidence < Threshold  |
       |             /                                   \           |
       |            v                                     v          |
       |   +------------------+                 +-----------------+  |
       |   | Return Tier 1    |                 | Escalate to     |  |
       |   | Direct to Client |                 | Tier 2 Frontier |  |
       |   | (Save 85-95%)    |                 | (Claude / GPT-4)|  |
       |   +------------------+                 +-----------------+  |
       |            \                                     /          |
       |             +-----------------+-----------------+           |
       |                               |                             |
       +-------------------------------|-----------------------------+
                                       |
                                       v
       +-------------------------------------------------------------+
       |                     FinOps & Telemetry                      |
       |  1. Transparent Response Delivery to Client                 |
       |  2. Async Telemetry Logged to SQLite (Cost / Latency / ROI) |
       |  3. Embedded React 19 FinOps Studio (Local HTTP :8088)      |
       +-------------------------------------------------------------+
```

---

## 3. Mathematical Formulation: Speculative Cascading Protocol

### 3.1 Prompt Complexity Scoring ($S_{\text{comp}}$)
Given prompt $P$ with token count $L_P$, syntactic constraint count $C_P$, and reasoning indicator density $R_P$:

$$S_{\text{comp}}(P) = w_1 \cdot \min\left(1.0, \frac{L_P}{2048}\right) + w_2 \cdot \sigma(C_P) + w_3 \cdot R_P$$

If $S_{\text{comp}}(P) > \theta_{\text{bypass}}$, Tier 1 is skipped entirely and routed directly to Tier 2 to prevent redundant latency.

### 3.2 Output Confidence Verification Metric ($Q_{\text{conf}}$)
For generated response $T_1 = \{t_1, t_2, \dots, t_N\}$ with corresponding token log-probabilities $\log p(t_i)$:

$$Q_{\text{conf}}(T_1) = \alpha \cdot \left( \frac{1}{N} \sum_{i=1}^{N} e^{\log p(t_i)} \right) + (1 - \alpha) \cdot V_{\text{schema}}(T_1)$$

Where $V_{\text{schema}} \in \{0, 1\}$ denotes deterministic schema validation adherence. The escalation condition is:

$$\text{Route}(P) = \begin{cases} 
\text{Accept } T_1, & \text{if } Q_{\text{conf}}(T_1) \ge \tau \\
\text{Dispatch Tier 2 } (T_2), & \text{if } Q_{\text{conf}}(T_1) < \tau 
\end{cases}$$

### 3.3 Cumulative FinOps Savings ($S_{\text{cost}}$)
Let $C_1$ and $C_2$ denote the cost per token of Tier 1 and Tier 2 respectively, with total requests $M$, Tier 1 acceptance rate $\rho$, and escalation rate $(1 - \rho)$:

$$S_{\text{cost}} = \sum_{k=1}^{M} \left( \text{Tokens}_k \cdot C_2 \right) - \sum_{k=1}^{M} \left[ \text{Tokens}_{k,1} \cdot C_1 + (1 - \rho_k) \cdot \text{Tokens}_{k,2} \cdot C_2 \right]$$

Net savings typically range between 70% and 85% on standard enterprise request distributions.

---

## 4. Repository Structure

```
cascadepilot/
├── cmd/
│   └── cascadepilot/
│       └── main.go                 # CLI entry point, subcommands: serve, benchmark, simulate
├── internal/
│   ├── config/
│   │   ├── config.go               # YAML configuration loader for model tiers and thresholds
│   │   └── schema.go               # Routing definitions, cost parameters ($/1M tokens), and timeouts
│   ├── router/
│   │   ├── engine.go               # Main cascading orchestrator and decision pipeline
│   │   ├── complexity.go           # Fast prompt complexity and intent evaluator
│   │   └── fallback.go             # Resilient cross-provider fallback handler
│   ├── verifier/
│   │   ├── confidence.go           # Token logprob entropy and probability analyzer
│   │   ├── heuristic.go            # Structural completeness and refusal detector
│   │   └── schema_gate.go          # Fast JSON/format compliance checker
│   ├── provider/
│   │   ├── client.go               # Unified upstream HTTP client (OpenAI, Anthropic, Gemini, Ollama)
│   │   ├── sse_forwarder.go        # Streaming chunk forwarder and buffer
│   │   └── pricing.go              # Real-time token cost calculation table per model
│   ├── finops/
│   │   ├── store.go                # Embedded SQLite store (modernc.org/sqlite) for cost records
│   │   ├── ledger.go               # Dollar savings, latency delta, and routing ratio aggregator
│   │   └── export.go               # CSV and JSON FinOps reporting exporter
│   └── server/
│       ├── proxy_handler.go        # HTTP handler matching /v1/chat/completions
│       ├── api.go                  # Telemetry and configuration REST API
│       ├── embed.go                # Go 1.16+ embed filesystem for React 19 studio
│       └── sse_hub.go              # Real-time SSE stream for dashboard charts
├── web/
│   ├── src/
│   │   ├── App.tsx                 # React 19 root application
│   │   ├── components/
│   │   │   ├── SavingsHero.tsx     # Cumulative dollar savings and ROI multiplier card
│   │   │   ├── RoutingWaterfall.tsx# Tier 1 vs Tier 2 volume and escalation ratio chart
│   │   │   ├── LatencyDiffChart.tsx# Latency distribution comparison (Tier 1 vs Escalated)
│   │   │   ├── ThresholdSlider.tsx # Interactive slider simulating threshold impacts on cost
│   │   │   └── RequestLiveLog.tsx  # Real-time table of routed queries with decision reasons
│   │   └── pages/
│   │       └── Dashboard.tsx       # Main FinOps executive cockpit
│   ├── package.json
│   ├── tsconfig.json
│   └── vite.config.ts
├── examples/
│   ├── cascadepilot.yaml           # Production configuration sample
│   └── query_benchmark.jsonl       # Sample test queries across diverse difficulty tiers
├── Makefile
├── go.mod
├── go.sum
├── LICENSE
└── README.md
```

---

## 5. Implementation Roadmap

### Phase 1: Go Engine Foundation and Provider Client
- Initialize Go module with minimal dependencies.
- Build unified client supporting OpenAI, Anthropic, and Gemini completion APIs.
- Implement cost tracking table calculating exact USD expenditures per request based on input/output tokens.

### Phase 2: Inbound Prompt Complexity Analyzer
- Implement `internal/router/complexity.go` computing prompt length, constraint markers, and semantic indicators in under 5 milliseconds.
- Build bypass rules allowing explicitly tagged prompts or queries with extreme complexity to skip directly to Tier 2.

### Phase 3: In-Flight Speculative Verification Engine
- Build `internal/verifier/confidence.go` evaluating token log-probabilities, formatting compliance, and absence of generic apologies/refusals.
- Set up configurable confidence threshold ($\tau$) in configuration YAML.

### Phase 4: Transparent Reverse Proxy and Escalation Pipeline
- Build HTTP reverse proxy intercepting `/v1/chat/completions`.
- Dispatch query to Tier 1; if verification passes, stream or return to client immediately.
- If verification fails, discard Tier 1 draft, seamlessly dispatch to Tier 2, and return final completion without breaking client connection.

### Phase 5: FinOps Accounting and SQLite Storage
- Implement `modernc.org/sqlite` database recording query timestamp, complexity score, Tier 1 model, Tier 2 model, escalation status, latency, and cost saved.
- Build REST API endpoints serving aggregated daily/monthly FinOps metrics.

### Phase 6: Embedded React 19 FinOps Studio
- Build modern web UI in `web/` featuring savings cards, routing percentage gauges, latency comparison charts, and live request inspector.
- Embed frontend static assets into the Go binary using `//go:embed` and serve locally on `:8088`.

### Phase 7: Verification, Benchmarking, and Documentation
- Run simulation on `examples/query_benchmark.jsonl` verifying cost reduction of at least 70% while maintaining 98%+ quality parity.
- Benchmark proxy overhead verifying added latency remains below 5ms on Tier 1 hits.
- Author humanized README with deployment guides, Dockerfile, and OpenAI/Anthropic drop-in replacement instructions.
