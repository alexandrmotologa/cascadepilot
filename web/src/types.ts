export interface FinOpsSummary {
  total_requests: number;
  tier1_accepted: number;
  tier2_direct: number;
  tier2_escalated: number;
  tier1_success_rate: number;
  total_tokens_saved: number;
  total_baseline_cost_usd: number;
  total_actual_cost_usd: number;
  total_saved_usd: number;
  savings_percentage: number;
  average_latency_ms: number;
  quality_parity_score: number;
}

export interface WaterfallMetrics {
  total_queries: number;
  direct_tier2_count: number;
  speculative_tier1: number;
  tier1_passed: number;
  tier1_escalated: number;
}

export interface LatencyMetrics {
  avg_tier1_ms: number;
  avg_direct_t2_ms: number;
  avg_escalated_ms: number;
  avg_overall_ms: number;
}

export interface RequestRecord {
  id: string;
  timestamp: string;
  prompt_preview: string;
  complexity_score: number;
  routed_tier: 'tier1' | 'tier2_direct' | 'tier2_escalated';
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
}

export interface Thresholds {
  theta_bypass: number;
  tau_confidence: number;
}
