# cascadepilot Benchmarking and Performance Metrics

This document outlines performance requirements, benchmark methodology, and latency profiles for cascadepilot.

## Performance Targets

- Routing Decision Overhead: < 5.0 ms for inbound complexity analysis and dispatch.
- Verification Overhead: < 3.0 ms for token logprob evaluation and refusal regex heuristics.
- Process Resident Memory (RSS): < 20 MB RAM under steady state.
- Throughput: > 5,000 requests per second per core on standard x86-64 / ARM64 instances.
- Cost Reduction: 70% to 85% compared to static frontier model allocation.
- Quality Parity: >= 98.5% compared to static frontier routing.

## Benchmark Methodology

cascadepilot includes a dedicated benchmark suite accessible through the CLI:

```bash
# Run internal routing and verification latency benchmarks
cascadepilot benchmark

# Run batch query simulation with synthetic or historical traffic
cascadepilot simulate --queries=examples/query_benchmark.jsonl --concurrency=10
```

### 1. Inbound Complexity Scoring Overhead
The complexity scorer uses pre-compiled regular expressions and character-based token estimators to compute prompt complexity without external API calls or embeddings.

Target latency:
- Prompts < 500 tokens: < 0.25 ms
- Prompts 500 to 2,000 tokens: < 0.85 ms
- Prompts > 2,000 tokens: < 1.80 ms

### 2. In-Flight Verification Latency
Verification checks run immediately as response chunks or full completions arrive:
- Log-probability margin extraction: O(N) where N is token count (< 0.15 ms for 500 tokens).
- Refusal detection: Single-pass DFA regex match (< 0.40 ms).
- JSON schema check: Fast byte scanner with standard library parser (< 0.80 ms).

### 3. Latency Tradeoff and Net User Experience
When Tier 1 generates an acceptable response (75% to 85% of traffic), the user experiences lower end-to-end latency because small models generate tokens 2x to 3x faster than frontier models.

When escalation occurs (15% to 25% of traffic), total latency equals Tier 1 draft latency plus Tier 2 completion latency plus proxy verification overhead (< 5 ms). Because escalation is restricted to uncertain or complex queries, average overall latency across the entire workload remains comparable or faster than pure frontier routing.
