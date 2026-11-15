# cascadepilot

Adaptive speculative LLM cascading reverse proxy and FinOps gateway in Go with embedded React 19 cockpit.

cascadepilot routes incoming prompt traffic dynamically between low-cost small language models (Gemini Flash, Llama 3.1 8B, GPT-4o-mini at $0.075 to $0.15 per 1M tokens) and expensive frontier models (Claude 3.5 Sonnet, GPT-4o, o1 at $3.00 to $15.00 per 1M tokens). Routine queries are resolved by Tier 1 at 95% lower cost. If in-flight verification detects uncertainty, schema violation, or hedging, the proxy escalates the query to Tier 2 without breaking client compatibility.

## Key capabilities

- High-speed complexity analyzer: Evaluates token length, syntactic constraints, and reasoning markers in under 0.1 milliseconds.
- Speculative verification gate: Analyzes average token log-probabilities, refusal patterns, and JSON schema adherence before delivering output to the client.
- In-flight transparent escalation: Discards uncertain Tier 1 drafts and queries Tier 2 directly without client disruption.
- Streaming speculative buffer: Buffers the first K tokens during streaming requests to verify confidence before flushing to the client socket.
- Drop-in API compatibility: Matches OpenAI `/v1/chat/completions` and Anthropic `/v1/messages` endpoints.
- Pure Go embedded SQLite ledger: Tracks prompt tokens, completion tokens, latency, and exact dollar savings using `modernc.org/sqlite` (zero CGO required).
- Embedded React 19 FinOps studio: Single binary with embedded web dashboard accessible on `:8088` featuring live SSE streaming, routing waterfalls, latency histograms, and live threshold tuning.
- Minimal footprint: Consumes under 20MB of memory and adds under 0.1ms of proxy routing overhead.

## Architecture

```
Client Application (OpenAI SDK / LangChain / Agent)
                     |
         POST /v1/chat/completions
                     v
        +-------------------------+
        |  Complexity Scorer      |  (S_comp < 0.1ms)
        +-------------------------+
           /                     \
   S_comp > theta_bypass     S_comp <= theta_bypass
         /                         \
        v                           v
  Tier 2 Frontier             Tier 1 Speculative Engine
  (Direct Bypass)             (Gemini Flash / GPT-4o-mini)
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
        |             Accept Tier 1     Escalate to Tier 2
        |             (Save 80-95%)     (Claude 3.5 Sonnet)
        |                  |                 |
        +------------------+-----------------+
                           |
                           v
              Client Delivery + Async FinOps
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

The gateway listens on `http://localhost:8088`.
Open your browser at `http://localhost:8088` to access the FinOps Studio.

### 2. Run with Docker

```bash
docker build -t cascadepilot .
docker run -p 8088:8088 -v $(pwd)/data:/app/data cascadepilot
```

## Drop-in client configuration

Point existing OpenAI SDK or Anthropic SDK clients to `http://localhost:8088`:

### Python (OpenAI SDK)

```python
from openai import OpenAI

client = OpenAI(
    base_url="http://localhost:8088/v1",
    api_key="not-needed-if-using-mock",
)

# Standard non-streaming completion
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
  apiKey: "not-needed",
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
| Inbound Complexity Scorer | &lt; 2,000 µs | 58.89 µs | PASS |
| Refusal and Hedging Heuristic | &lt; 1,000 µs | 21.32 µs | PASS |
| JSON Schema Validator | &lt; 1,000 µs | 1.72 µs | PASS |
| LogProb Confidence Evaluator | &lt; 500 µs | 0.01 µs | PASS |
| Total Proxy Overhead | &lt; 5.0 ms | **0.082 ms** | **PASS** |

Resident set size (RSS) memory under steady state: 16.8 MB (target &lt; 20 MB).

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

tiers:
  tier1:
    model: "gemini-1.5-flash"
    provider: "mock"          # "google", "openai", "ollama", or "mock"
    endpoint: "https://generativelanguage.googleapis.com/v1beta/openai"
    api_key_env: "GEMINI_API_KEY"
    cost_per_million_input: 0.075
    cost_per_million_output: 0.30
    timeout_ms: 8000

  tier2:
    model: "claude-3-5-sonnet"
    provider: "mock"          # "anthropic", "openai", or "mock"
    endpoint: "https://api.anthropic.com/v1"
    api_key_env: "ANTHROPIC_API_KEY"
    cost_per_million_input: 3.00
    cost_per_million_output: 15.00
    timeout_ms: 35000

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
- `GET /api/waterfall`: Query routing distribution counts.
- `GET /api/latency`: Latency distribution across tiers.
- `GET /api/requests`: Recent request ledger records.
- `GET /api/thresholds`: Current dynamic thresholds.
- `POST /api/thresholds`: Update thresholds dynamically without restarting.
- `GET /api/export/csv`: Export all transaction records as CSV.
- `GET /api/export/json`: Export all transaction records as JSON.
- `POST /api/simulate`: Trigger simulated query batch.

## License

MIT License. See LICENSE for details.
