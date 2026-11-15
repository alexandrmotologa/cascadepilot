# cascadepilot

Adaptive speculative LLM cascading reverse proxy and FinOps gateway in Go with embedded React 19 cockpit.

cascadepilot routes incoming prompt traffic dynamically between Tier 0 Prompt Cache ($0.00), low-cost small language models (Gemini Flash, Llama 3.1 8B at $0.075 to $0.15 per 1M tokens), mid-tier reasoning models (GPT-4o-mini at $0.15 to $0.60 per 1M tokens), and expensive frontier models (Claude 3.5 Sonnet, GPT-4o, o1 at $3.00 to $15.00 per 1M tokens).

Routine queries are resolved by Tier 1 at 95% lower cost. If in-flight verification detects uncertainty, schema violation, or hedging, the proxy escalates the query to Tier 2 (Mid-tier) or Tier 3 (Frontier) without breaking client compatibility.

## Key capabilities

- Tier 0 Exact and Normalized Prompt Cache: Returns identical or normalized prompt completions in under 1 millisecond at zero cost ($0.00).
- Speculative 3-Tier Cascade: Dynamically orchestrates traffic across Tier 1 (SLM), Tier 2 (Mid-tier reasoning), and Tier 3 (Frontier flagship).
- Inbound Complexity Scorer: Evaluates token length, syntactic constraints, and reasoning markers in under 0.1 milliseconds.
- In-Flight Verification Gate: Evaluates token log-probabilities, refusal patterns, and JSON schema compliance before socket delivery.
- Multi-Tenant Cost Allocation and Department Chargeback: Generates virtual gateway API keys (`cp-live-...`) and tracks spending and savings by department (Engineering, Customer Support, Marketing, Data Science).
- Enterprise Budget Caps and Circuit Breaker: Enforces daily and monthly dollar caps with automatic fallback to Tier 1 to prevent runaway cloud charges.
- Side-by-Side Model Benchmark: Compares Tier 1, Mid-tier, Frontier, and Cascadepilot in parallel with token and cost diffs.
- Executive FinOps ROI Calculator: Interactively models enterprise annual dollar savings and quality parity retention.
- Drop-In API Compatibility: Matches OpenAI `/v1/chat/completions` and Anthropic `/v1/messages` endpoints.
- Embedded React 19 Cockpit: Single binary with embedded web dashboard on port `:8088` featuring live SSE streaming, waterfall distributions, and live threshold tuning.
- Minimal Footprint: Operates within 18MB RAM and adds under 0.1ms of proxy routing overhead.

## Architecture

```
Client Application / Agent (OpenAI or Anthropic SDK)
                     |
         POST /v1/chat/completions (with Bearer cp-live-...)
                     v
        +-------------------------+
        |  Tier 0 Prompt Cache    | ---> Cache Hit? ---> Return ($0.00, < 1ms)
        +-------------------------+
                     | Miss
                     v
        +-------------------------+
        |  Complexity Scorer      |  (S_comp < 0.1ms)
        +-------------------------+
           /                     \
   S_comp > theta_bypass     S_comp <= theta_bypass
         /                         \
        v                           v
  Tier 3 Frontier             Tier 1 Speculative Engine
  (Direct Bypass)             (Gemini Flash / SLM)
        |                           |
        |                           v
        |                     Verification Gate
        |                     - Token logprobs (P_logprob)
        |                     - Refusal heuristic (H_ref)
        |                     - JSON schema gate (V_schema)
        |                           |
        |                  +--------+--------+
        |                  |                 |
        |             Q_conf >= tau     Q_conf < tau
        |                  |                 |
        |                  v                 v
        |             Accept Tier 1     Moderate or High Complexity?
        |             (Save 95%)       /                            \
        |                  |      Moderate (S_comp <= 0.65)    High (S_comp > 0.65)
        |                  |             v                             v
        |                  |       Tier 2 Mid-tier               Tier 3 Frontier
        |                  |       (GPT-4o-mini, save 90%)       (Claude 3.5 Sonnet)
        |                  |             |                             |
        +------------------+-------------+-----------------------------+
                           |
                           v
              Client Delivery + Async FinOps Ledger
```

## Quick start

### 1. Build and run from source

Requires Go 1.24+ and Node 20+.

```bash
# Clone the repository
git clone https://github.com/alexandrmotologa/cascadepilot.git
cd cascadepilot

# Build frontend and Go binary
make build

# Start the reverse proxy and dashboard
./bin/cascadepilot serve --config examples/cascadepilot.yaml
```

Open `http://localhost:8088` to access the FinOps Cockpit.

### 2. Run with Docker

```bash
docker build -t cascadepilot .
docker run -p 8088:8088 -v $(pwd)/data:/app/data cascadepilot
```

## Drop-in client configuration

Point existing OpenAI SDK or Anthropic SDK clients to `http://localhost:8088`:

### Python (OpenAI SDK with Department Attribution)

```python
from openai import OpenAI

# Connect using a department virtual key for chargeback
client = OpenAI(
    base_url="http://localhost:8088/v1",
    api_key="cp-live-eng-9f821a", # Core Engineering virtual key
)

# Standard completion
response = client.chat.completions.create(
    model="cascadepilot-auto",
    messages=[
        {"role": "user", "content": "Extract company and price: Acme Corp bought WidgetLabs for $45M in 2023."}
    ],
)
print(response.choices[0].message.content)

# Streaming completion with speculative verification
stream = client.chat.completions.create(
    model="cascadepilot-auto",
    messages=[
        {"role": "user", "content": "Classify sentiment: Delivery arrived two days ahead of schedule."}
    ],
    stream=True,
)
for chunk in stream:
    if chunk.choices[0].delta.content:
        print(chunk.choices[0].delta.content, end="", flush=True)
```

### TypeScript / Node.js

```typescript
import OpenAI from "openai";

const openai = new OpenAI({
  baseURL: "http://localhost:8088/v1",
  apiKey: "cp-live-sup-3b714c", // Customer Support virtual key
});

const completion = await openai.chat.completions.create({
  model: "cascadepilot-auto",
  messages: [{ role: "user", content: "Summarize this quarterly revenue statement in one sentence." }],
});

console.log(completion.choices[0].message.content);
```

## CLI reference

```bash
# Start proxy server
cascadepilot serve --host 0.0.0.0 --port 8088 --config examples/cascadepilot.yaml

# Run latency and verification overhead benchmarks
cascadepilot benchmark

# Run batch workload simulation
cascadepilot simulate --queries examples/query_benchmark.jsonl --workers 4
```

## Benchmark results

Empirical measurements on an AMD Ryzen 7 / Apple M-series host:

| Component | Target Latency | Measured Latency | Status |
| :--- | :--- | :--- | :--- |
| Inbound Complexity Scorer | &lt; 2,000 µs | 65.50 µs | PASS |
| Refusal and Hedging Heuristic | &lt; 1,000 µs | 18.01 µs | PASS |
| JSON Schema Validator | &lt; 1,000 µs | 2.80 µs | PASS |
| LogProb Confidence Evaluator | &lt; 500 µs | 0.01 µs | PASS |
| Total Proxy Overhead | &lt; 5.0 ms | **0.086 ms** | **PASS** |

Memory resident set size (RSS) under steady state: 17.4 MB (target &lt; 20 MB).

## Configuration reference

```yaml
server:
  host: "0.0.0.0"
  port: 8088
  read_timeout_ms: 60000
  write_timeout_ms: 60000

thresholds:
  theta_bypass: 0.75          # Prompts with S_comp > 0.75 bypass Tier 1 directly
  tau_confidence: 0.70        # Tier 1 outputs with Q_conf < 0.70 escalate to Tier 2
  stream_buffer_tokens: 16    # Speculative token buffer window for streaming requests
  auto_tune_enabled: true

tiers:
  tier1:
    model: "gemini-1.5-flash"
    provider: "mock"
    endpoint: "https://generativelanguage.googleapis.com/v1beta/openai"
    api_key_env: "GEMINI_API_KEY"
    cost_per_million_input: 0.075
    cost_per_million_output: 0.30
    timeout_ms: 8000

  tier2:
    model: "gpt-4o-mini"
    provider: "mock"
    endpoint: "https://api.openai.com/v1"
    api_key_env: "OPENAI_API_KEY"
    cost_per_million_input: 0.150
    cost_per_million_output: 0.60
    timeout_ms: 15000

  tier3:
    model: "claude-3-5-sonnet"
    provider: "mock"
    endpoint: "https://api.anthropic.com/v1"
    api_key_env: "ANTHROPIC_API_KEY"
    cost_per_million_input: 3.00
    cost_per_million_output: 15.00
    timeout_ms: 35000

cache:
  enabled: true
  max_size: 5000
  ttl_minutes: 1440

budget:
  daily_cap_usd: 100.0
  monthly_cap_usd: 2500.0
  circuit_breaker_action: "downgrade_to_tier1"

storage:
  db_path: "data/cascadepilot.db"

simulation:
  mock_enabled: true
  simulated_network_latency_ms: 12
```

## API endpoints

- `POST /v1/chat/completions`: OpenAI compatible completion endpoint.
- `POST /v1/messages`: Anthropic compatible messages endpoint.
- `GET /v1/models`: Virtual and tier model listings.
- `GET /health`: Health probe with uptime and active thresholds.
- `GET /metrics`: Prometheus formatted metrics.
- `GET /api/events`: Server-Sent Events stream for real-time telemetry.
- `GET /api/summary`: Aggregated FinOps totals and percentage savings.
- `GET /api/waterfall`: Query routing distribution counts across all 4 stages.
- `GET /api/latency`: Latency distribution across tiers.
- `GET /api/requests`: Recent request ledger records with tenant attribution.
- `GET /api/tenants`: List departments with spend vs monthly budget.
- `POST /api/tenants`: Create a new department virtual API key.
- `POST /api/compare`: Execute parallel benchmark across all tiers.
- `GET /api/cache`: Cache statistics and hit ratios.
- `POST /api/cache`: Clear Tier 0 cache.
- `GET /api/thresholds`: Current dynamic thresholds.
- `POST /api/thresholds`: Update thresholds dynamically without restarting.
- `GET /api/export/csv`: Export all transaction records as CSV.
- `GET /api/export/json`: Export all transaction records as JSON.
- `POST /api/simulate`: Trigger simulated query batch.

## License

MIT License. See LICENSE for details.
