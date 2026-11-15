# FinOps Cost Models and Mathematical Formulations

This document details the mathematical formulations implemented in cascadepilot for evaluating prompt complexity, verification confidence, multi-tier cost savings, and department chargebacks.

## 1. Prompt Complexity Formulation

The complexity score S_comp(P) predicts whether a small model can handle prompt P without quality degradation:

```
S_comp(P) = max( (w1 * min(1.0, L_P / 2048) + w2 * C_P + w3 * R_P), R_P * 0.80 )
```

Where:
- L_P: Estimated token length of prompt P.
- C_P: Syntactic constraint density (JSON schema, formatting guidelines, negative constraints).
- R_P: Reasoning indicator intensity (occurrences of mathematical, distributed system, and formal logic keywords).
- Weights default to: w1 = 0.35, w2 = 0.35, w3 = 0.30.

Routing Decision:
- If S_comp(P) > theta_bypass: Bypass smaller tiers. Dispatch directly to Tier 3 Frontier.
- If S_comp(P) <= theta_bypass: Dispatch speculatively to Tier 1 SLM.

## 2. In-Flight Verification Confidence Metric

When Tier 1 generates draft completion T1, cascadepilot scores response confidence Q_conf(T1):

```
Q_conf(T1) = alpha * P_logprob + beta * V_schema - gamma * H_ref
```

Where:
- P_logprob: Average token probability derived from log-probabilities: (1 / N) * sum(e^(log p(ti))).
- V_schema: Schema adherence score (1.0 for valid JSON or non-structured text, 0.0 for invalid syntax).
- H_ref: Refusal or uncertainty penalty (0.0 to 1.0).
- Default weights: alpha = 0.50, beta = 0.35, gamma = 0.50.

Escalation Decision:
- If Q_conf(T1) >= tau: Accept completion T1.
- If Q_conf(T1) < tau and S_comp <= 0.65: Escalate to Tier 2 Mid-tier.
- If Q_conf(T1) < tau and S_comp > 0.65: Escalate directly to Tier 3 Frontier.

## 3. FinOps Savings Calculations

### Baseline Cost (Static Frontier Allocation)
Hypothetical expenditure if all queries were routed to Tier 3 Frontier:

```
Cost_baseline = sum_{k=1}^{M} [ (Tokens_in,k * Price_in,T3) + (Tokens_out,k * Price_out,T3) ]
```

### Actual Realized Cost
The actual expenditure incurred across all four stages:

```
Cost_realized = sum_{k=1}^{M} Cost_actual,k
```

For request k:
- Tier 0 Cache Hit: `Cost_actual,k = $0.00`
- Tier 1 Accepted: `Cost_actual,k = (Tokens_in * Price_in,T1) + (Tokens_out * Price_out,T1)`
- Tier 2 Mid-tier Escalation: `Cost_actual,k = Cost_T1_draft + Cost_T2_completion`
- Tier 3 Frontier Escalation: `Cost_actual,k = Cost_T1_draft + Cost_T3_completion`
- Tier 3 Frontier Direct Bypass: `Cost_actual,k = (Tokens_in * Price_in,T3) + (Tokens_out * Price_out,T3)`

### Net Dollar Savings
```
Savings_usd = Cost_baseline - Cost_realized
```

### Quality Parity Metric
The quality parity score approximates quality retention relative to a 100% frontier baseline:

```
Parity_score = [ (N_cache * 1.0) + (N_t1 * 0.988) + (N_t2_mid * 0.994) + (N_t3 * 1.0) ] / N_total
```

## 4. Multi-Tenant Department Chargeback

For department D with allocated monthly budget B_D and requests R_D:

```
Spend_D = sum_{r in R_D} Cost_actual,r
Budget_utilization_D = (Spend_D / B_D) * 100
Savings_D = sum_{r in R_D} Saved_cost,r
```

If Spend_D exceeds B_D, circuit breaker policies automatically throttle or downgrade non-critical traffic to Tier 1.
