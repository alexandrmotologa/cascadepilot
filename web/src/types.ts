export interface FinOpsSummary {
  total_requests: number;
  tier0_cache_hits: number;
  tier1_accepted: number;
  tier2_midtier_count: number;
  tier3_direct: number;
  tier3_escalated: number;
  tier1_success_rate: number;
  total_baseline_cost_usd: number;
  total_actual_cost_usd: number;
  total_saved_usd: number;
  savings_percentage: number;
  average_latency_ms: number;
  quality_parity_score: number;
  daily_spend_usd: number;
  monthly_spend_usd: number;
  circuit_breaker_active: boolean;
}

export interface WaterfallMetrics {
  total_queries: number;
  tier0_cache_hits: number;
  direct_tier3_count: number;
  speculative_tier1: number;
  tier1_passed: number;
  tier2_mid_passed: number;
  tier3_escalated: number;
}

export interface LatencyMetrics {
  avg_cache_ms: number;
  avg_tier1_ms: number;
  avg_tier2_mid_ms: number;
  avg_direct_t3_ms: number;
  avg_escalated_ms: number;
  avg_overall_ms: number;
}

export interface RequestRecord {
  id: string;
  timestamp: string;
  prompt_preview: string;
  complexity_score: number;
  routed_tier: 'tier0_cache' | 'tier1' | 'tier2_mid' | 'tier3_direct' | 'tier3_escalated' | 'tier2_direct' | 'tier2_escalated';
  tier1_model: string;
  tier2_model: string;
  tier1_confidence: number;
  escalated: boolean;
  escalation_reason?: string;
  prompt_tokens: number;
  completion_tokens: number;
  latency_ms: number;
  baseline_cost_usd: number;
  actual_cost_usd: number;
  saved_cost_usd: number;
  stream: boolean;
  tenant_id?: string;
  department?: string;
  cache_hit?: boolean;
}

export interface TenantSummary {
  id: string;
  name: string;
  department: string;
  api_key: string;
  monthly_budget_usd: number;
  created_at: string;
  active: boolean;
  total_requests: number;
  total_tokens: number;
  total_actual_cost_usd: number;
  total_saved_cost_usd: number;
  budget_utilization_pct: number;
}

export interface Thresholds {
  theta_bypass: number;
  tau_confidence: number;
}

export interface TierModelEvaluation {
  model: string;
  tier: string;
  content: string;
  latency_ms: number;
  prompt_tokens: number;
  completion_tokens: number;
  cost_usd: number;
}

export interface CompareResult {
  prompt: string;
  cascade_route: RequestRecord;
  tier1_result: TierModelEvaluation;
  tier2_result: TierModelEvaluation;
  tier3_result: TierModelEvaluation;
}
