# FinOps Cost Models and Formulas

This document details the mathematical formulations implemented in cascadepilot for evaluating prompt complexity, verification confidence, and cost savings.

## 1. Prompt Complexity Formulation

The complexity score S_comp(P) predicts whether a small model can handle prompt P without quality degradation:

```
S_comp(P) = w1 * min(1.0, L_P / 2048) + w2 * sigma(C_P) + w3 * R_P
```

Where:
- L_P: Estimated token length of prompt P.
- C_P: Number of structural and negative constraints detected in the prompt.
- sigma(x): Sigmoid squashing function sigma(x) = 1 / (1 + e^(-(x - 2))).
- R_P: Reasoning marker ratio (occurrences of mathematical, logical, and code analysis keywords normalized by prompt word count).
- Default weights: w1 = 0.35, w2 = 0.35, w3 = 0.30.

Routing Decision:
- If S_comp(P) > theta_bypass: Bypass Tier 1. Dispatch directly to Tier 2.
- If S_comp(P) <= theta_bypass: Dispatch speculatively to Tier 1.

## 2. In-Flight Verification Confidence Metric

When Tier 1 generates draft completion T1, cascadepilot scores response confidence Q_conf(T1):

```
Q_conf(T1) = alpha * P_logprob + beta * V_schema - gamma * H_ref
```

Where:
- P_logprob: Average token probability derived from log-probabilities: (1 / N) * sum(e^(log p(ti))). Defaults to 0.85 when log-probabilities are omitted by the upstream provider.
- V_schema: Schema adherence score. Equals 1.0 if JSON structure satisfies required schema, 0.0 if invalid JSON, and 1.0 if no schema was requested.
- H_ref: Refusal or uncertainty penalty. Ranges from 0.0 (no refusal markers) to 1.0 (clear refusal, apology, or uncertainty phrase detected).
- Default parameters: alpha = 0.50, beta = 0.35, gamma = 0.50.

Escalation Decision:
- If Q_conf(T1) >= tau: Accept completion T1. Return directly to client.
- If Q_conf(T1) < tau: Discard T1. Dispatch query to Tier 2 frontier model.

## 3. FinOps Savings Calculations

### Baseline Cost (Static Frontier Allocation)
The hypothetical expenditure if all requests were sent directly to the Tier 2 frontier model:

```
Cost_baseline = sum_{k=1}^{M} [ (Tokens_in,k * Price_in,T2) + (Tokens_out,k * Price_out,T2) ]
```

### Actual Realized Cost
The actual expenditure incurred across both tiers:

```
Cost_realized = sum_{k=1}^{M} Cost_actual,k
```

For request k:
- If routed directly to Tier 2 (bypassed):
  `Cost_actual,k = (Tokens_in,k * Price_in,T2) + (Tokens_out,k * Price_out,T2)`
- If routed to Tier 1 and accepted:
  `Cost_actual,k = (Tokens_in,k * Price_in,T1) + (Tokens_out,k * Price_out,T1)`
- If routed to Tier 1 and escalated to Tier 2:
  `Cost_actual,k = [Tokens_in,k * Price_in,T1 + Tokens_out,T1 * Price_out,T1] + [Tokens_in,k * Price_in,T2 + Tokens_out,T2 * Price_out,T2]`

### Net Dollar Savings
```
Savings_usd = Cost_baseline - Cost_realized
```

### Savings Percentage
```
Savings_pct = (Savings_usd / Cost_baseline) * 100
```

### Quality Parity Metric
The quality parity score approximates output quality retention relative to a 100% frontier baseline:

```
Parity_score = (N_tier1_accepted * W_t1 + N_escalated * 1.0 + N_direct_t2 * 1.0) / N_total
```

Where W_t1 represents the empirical benchmark accuracy ratio of the Tier 1 model on non-complex queries (typically 0.985 to 0.992 for models like Gemini 1.5 Flash and GPT-4o-mini).
