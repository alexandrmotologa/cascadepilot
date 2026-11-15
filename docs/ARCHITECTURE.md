# cascadepilot System Architecture

This document describes the internal architecture of cascadepilot, an adaptive speculative reverse proxy designed for enterprise language model workloads.

## Core Concepts

In enterprise environments, standard LLM applications invoke frontier models for all prompts. cascadepilot sits as a drop-in HTTP reverse proxy between client applications (OpenAI SDK, LangChain, LlamaIndex, custom agents) and upstream model providers.

The proxy routes requests through a two-tier speculative pipeline:
1. Inbound Complexity Scoring: Inspects the raw prompt for length, reasoning tokens, and syntactic constraints.
2. Speculative Dispatch: Sends suitable prompts to a low-cost Tier 1 model.
3. In-Flight Verification: Evaluates Tier 1 output confidence using log-probabilities, semantic heuristics, and schema validity.
4. Transparent Escalation: If Tier 1 confidence falls below the acceptance threshold, the proxy discards the draft and queries Tier 2 directly before responding to the client.

## Component Breakdown

### 1. Inbound Request Complexity Scorer (`internal/router/complexity.go`)
Before calling any model, cascadepilot scores prompt complexity. The scorer operates in under 2 milliseconds without network roundtrips.

Features evaluated:
- Token length: Normalizes token counts against an empirical upper bound of 2048 tokens.
- Constraint density: Detects JSON schemas, XML wrappers, markdown specifications, regex targets, and negative constraints ("never", "do not").
- Reasoning indicators: Detects chain of thought requests, algorithmic problems, mathematical formulations, and multi-step deduction requests.

If the prompt complexity exceeds the bypass threshold (theta_bypass), the query skips Tier 1 entirely, avoiding unnecessary speculative latency.

### 2. Provider Layer (`internal/provider/`)
The provider layer translates generic OpenAI-compatible chat completion requests to upstream provider protocols:
- OpenAI: Direct pass-through to `/v1/chat/completions`.
- Anthropic: Translates between OpenAI format and Anthropic `/v1/messages` format.
- Google Gemini: OpenAI-compatible endpoints or native REST format.
- Ollama / Local Models: Local inference endpoints for on-premise SLMs.
- Mock Engine: Deterministic simulation provider for testing, synthetic benchmarks, and demonstrations without API keys.

### 3. Verification Engine (`internal/verifier/`)
Once Tier 1 returns a response, the verification engine checks response validity before delivering it to the client:
- Confidence Scorer (`confidence.go`): Analyzes token log-probabilities when provided by the upstream engine.
- Heuristic Gate (`heuristic.go`): Detects standard refusal phrases, evasions, repetitive token loops, and unexpected truncated outputs.
- Schema Validator (`schema_gate.go`): Parses JSON schema responses to ensure valid syntax and complete keys when structured outputs are requested.

If the calculated score Q_conf is greater than or equal to tau, the response is accepted. Otherwise, the proxy initiates escalation.

### 4. Speculative Streaming Forwarder (`internal/provider/sse_forwarder.go`)
For streaming requests (`stream: true`), the proxy uses a speculative buffer. The first K tokens are held in a small ring buffer to evaluate early confidence and formatting signals. Once confidence is verified, the buffer flushes to the client and remaining chunks stream through with minimal Time to First Token (TTFT).

If an early refusal or low-confidence signal triggers escalation during buffering, the stream cleanly pivots to Tier 2 without delivering erroneous tokens to the client.

### 5. FinOps Ledger and Storage (`internal/finops/`)
All routing decisions, token counts, model latencies, and dollar savings are written asynchronously to an embedded SQLite database using `modernc.org/sqlite` (pure Go, zero CGO).

The ledger computes:
- Cumulative expenditure under static Tier 2 routing.
- Realized expenditure under cascadepilot speculative routing.
- Net savings in USD and percentage reduction.
- Quality parity metric based on acceptance rate and Tier 2 escalations.

### 6. Embedded FinOps Cockpit (`web/` and `internal/server/embed.go`)
The React 19 dashboard is compiled and embedded into the Go binary at build time. Users can access the studio by opening `http://localhost:8088` in any browser. The dashboard receives live events via Server-Sent Events (SSE) and exposes controls to adjust thresholds in real time.
