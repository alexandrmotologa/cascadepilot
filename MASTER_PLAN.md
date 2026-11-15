# cascadepilot: Architecture and Engineering Master Plan
Adaptive Speculative LLM Cascading Router and Cost-Minimization Gateway for Enterprise Workloads
Target: Single-Binary Go Reverse Proxy (< 20MB RAM, < 0.1ms routing overhead) with Embedded React 19 FinOps Studio, Tier 0 Cache, and Department Chargeback

---

## 1. Problem Statement and Motivation

### 1.1 The Economics of Enterprise LLM Workloads
As organizations run generative AI in production, API expenditures often turn into an operational challenge. Most setups rely on static model choices, configuring applications to query frontier models regardless of query difficulty:
1. Workload Distribution: About 75% to 85% of production prompts involve routine tasks such as data extraction, text classification, simple summarization, or formatting. Small language models (SLMs) like Gemini Flash, Llama 3.1 8B, or GPT-4o-mini can resolve these tasks at a fraction of the cost.
2. Limitations of Static Routing: Hardcoded rules based on endpoint paths or basic regex patterns fail because query complexity varies within the same endpoint.
3. Fear of Quality Degradation: Teams default to expensive models out of concern that smaller models will fail on difficult edge cases.
4. Absence of Automatic Escalation: Standard architectures lack an in-flight mechanism to evaluate confidence and escalate low-certainty outputs to frontier models transparently.

### 1.2 The cascadepilot Solution
cascadepilot is a speculative cascading reverse proxy in Go. It operates on the network layer, accepts standard OpenAI and Anthropic API requests, and executes dynamic multi-tier speculative routing:
- Tier 0 Exact & Normalized Prompt Caching: Returns cached completions in < 1ms at $0.00 cost for repetitive queries.
- Prompt Complexity Scoring: Evaluates incoming requests in under 0.1 milliseconds using token length, syntactic constraints, and reasoning markers.
- Speculative 3-Tier Execution: Dispatches queries across Tier 1 (SLM), Tier 2 (Mid-tier reasoning), and Tier 3 (Frontier flagship).
- In-Flight Quality and Confidence Verification: Inspects Tier 1 output quality using token log-probabilities, refusal detection, and schema validation.
- Transparent Escalation: If Tier 1 confidence is below the configured threshold, the proxy escalates the query progressively to Tier 2 Mid-tier or Tier 3 Frontier.
- Multi-Tenant Department Chargeback: Generates virtual gateway tokens (`cp-live-...`) and tracks spending against department budgets.
- Enterprise Budget Guardrail: Implements circuit breakers to protect against runaway cloud costs.
- Embedded FinOps Studio: Includes a React 19 dashboard displaying dollar savings, routing ratios, latency distributions, side-by-side model comparators, and executive ROI calculators.

---

## 2. High-Level System Architecture

```
Client Application / Agent (OpenAI SDK / Anthropic SDK / LangChain)
                                |
                  POST /v1/chat/completions (with Bearer cp-live-...)
                                v
               +---------------------------------+
               |      Tier 0 Prompt Cache        | ---> Cache Hit? ---> Return ($0.00, < 1ms)
               +---------------------------------+
                                | Miss
                                v
               +---------------------------------+
               |  Inbound Complexity Scorer      |  (S_comp < 0.1ms)
               |  - Token Length & Constraints   |
               |  - Reasoning Indicator Density  |
               +---------------------------------+
                    /                       \
           S_comp > theta_bypass       S_comp <= theta_bypass
                  /                           \
                 v                             v
           Tier 3 Frontier              Tier 1 Speculative Engine
           (Direct Bypass)              (Gemini Flash / SLM)
                 |                             |
                 |                             v
                 |                      Verification Gate
                 |                      - Token logprobs (P_logprob)
                 |                      - Refusal heuristic (H_ref)
                 |                      - JSON schema gate (V_schema)
                 |                             |
                 |                    +--------+--------+
                 |                    |                 |
                 |               Q_conf >= tau     Q_conf < tau
                 |                    |                 |
                 |                    v                 v
                 |               Accept Tier 1     Moderate or High Complexity?
                 |               (Save 95%)       /                            \
                 |                    |      Moderate (S_comp <= 0.65)    High (S_comp > 0.65)
                 |                    |             v                             v
                 |                    |       Tier 2 Mid-tier               Tier 3 Frontier
                 |                    |       (GPT-4o-mini, save 90%)       (Claude 3.5 Sonnet)
                 |                    |             |                             |
                 +--------------------+-------------+-----------------------------+
                                      |
                                      v
                        Client Delivery + Async FinOps Ledger
```

---

## 3. Mathematical Formulation

### 3.1 Prompt Complexity Scoring (S_comp)
For prompt P with estimated token count L_P, syntactic constraint count C_P, and reasoning indicator intensity R_P:

```
S_comp(P) = max( (w1 * min(1.0, L_P / 2048) + w2 * C_P + w3 * R_P), R_P * 0.80 )
```

Weights default to w1 = 0.35, w2 = 0.35, w3 = 0.30.
If S_comp(P) > theta_bypass (default 0.75), smaller tiers are bypassed directly to Tier 3 Frontier.

### 3.2 Output Confidence Verification Metric (Q_conf)
For generated response T1 with token log-probabilities, refusal penalty H_ref, and schema validity V_schema:

```
Q_conf(T1) = alpha * P_logprob + beta * V_schema - gamma * H_ref
```

Where alpha = 0.50, beta = 0.35, gamma = 0.50.

### 3.3 Progressive 3-Tier Routing Protocol
- If Q_conf(T1) >= tau: Accept T1 (Tier 1 SLM).
- If Q_conf(T1) < tau and S_comp <= 0.65: Escalate to Tier 2 Mid-tier (GPT-4o-mini).
- If Q_conf(T1) < tau and S_comp > 0.65: Escalate to Tier 3 Frontier (Claude 3.5 Sonnet).

---

## 4. Completed Implementation Milestones

- [x] Phase 1: Go Engine Foundation and Provider Client
- [x] Phase 2: Inbound Prompt Complexity Analyzer (< 0.1ms overhead)
- [x] Phase 3: In-Flight Speculative Verification Engine (Logprobs, Refusal DFA, JSON Gate)
- [x] Phase 4: Transparent Reverse Proxy and Streaming Escalation Pipeline
- [x] Phase 5: FinOps Accounting and Pure-Go Embedded SQLite Storage (`modernc.org/sqlite`)
- [x] Phase 6: Embedded React 19 FinOps Studio (`//go:embed all:dist`)
- [x] Phase 7: Tier 0 Exact & Normalized Prompt Caching (< 1ms, $0.00 cost)
- [x] Phase 8: Multi-Tenant Virtual API Keys & Department Chargeback
- [x] Phase 9: Progressive 3-Tier Speculative Cascade (SLM -> Mid-tier -> Frontier)
- [x] Phase 10: Enterprise Budget Caps & Circuit Breaker Guardrail
- [x] Phase 11: Side-by-Side Model Benchmark & Executive ROI Calculator
- [x] Phase 12: Expanded Prometheus Metrics & Empirical Benchmarking
