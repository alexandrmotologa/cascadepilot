import React from 'react';
import { TrendingDown, Zap, ShieldCheck, Clock, ArrowUpRight, Database, ShieldAlert } from 'lucide-react';
import { FinOpsSummary } from '../types';

interface SavingsHeroProps {
  summary: FinOpsSummary | null;
}

export const SavingsHero: React.FC<SavingsHeroProps> = ({ summary }) => {
  const savedUSD = summary?.total_saved_usd ?? 0;
  const savedPct = summary?.savings_percentage ?? 0;
  const baselineCost = summary?.total_baseline_cost_usd ?? 0;
  const actualCost = summary?.total_actual_cost_usd ?? 0;
  const successRate = summary?.tier1_success_rate ?? 0;
  const qualityParity = summary?.quality_parity_score ?? 100;
  const avgLatency = summary?.average_latency_ms ?? 0;
  const cacheHits = summary?.tier0_cache_hits ?? 0;
  const circuitBreaker = summary?.circuit_breaker_active ?? false;
  const monthlySpend = summary?.monthly_spend_usd ?? 0;

  return (
    <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-5 gap-4">
      {/* Card 1: Cumulative Net Savings */}
      <div className="bg-[#0f172a] border border-slate-800 rounded-xl p-5 relative overflow-hidden group hover:border-blue-500/50 transition-all duration-300">
        <div className="absolute top-0 right-0 w-32 h-32 bg-blue-500/5 rounded-full blur-2xl group-hover:bg-blue-500/10 transition-colors pointer-events-none" />
        <div className="flex items-center justify-between text-slate-400 mb-2">
          <span className="text-xs font-medium uppercase tracking-wider">Net Dollar Savings</span>
          <div className="p-2 rounded-lg bg-emerald-500/10 text-emerald-400">
            <TrendingDown className="w-4 h-4" />
          </div>
        </div>
        <div className="flex items-baseline gap-2 mb-1">
          <span className="text-3xl font-bold font-mono tracking-tight text-white">
            ${savedUSD.toFixed(4)}
          </span>
          <span className="inline-flex items-center text-xs font-semibold px-2 py-0.5 rounded-full bg-emerald-500/15 text-emerald-400 border border-emerald-500/20">
            <ArrowUpRight className="w-3 h-3 mr-0.5" />
            {savedPct.toFixed(1)}%
          </span>
        </div>
        <div className="text-xs text-slate-400 flex items-center justify-between pt-2 border-t border-slate-800/80">
          <span>Base: <strong className="font-mono text-slate-300">${baselineCost.toFixed(4)}</strong></span>
          <span>Act: <strong className="font-mono text-slate-300">${actualCost.toFixed(4)}</strong></span>
        </div>
      </div>

      {/* Card 2: Tier 0 Cache Hits (Free Requests) */}
      <div className="bg-[#0f172a] border border-slate-800 rounded-xl p-5 relative overflow-hidden group hover:border-cyan-500/50 transition-all duration-300">
        <div className="absolute top-0 right-0 w-32 h-32 bg-cyan-500/5 rounded-full blur-2xl group-hover:bg-cyan-500/10 transition-colors pointer-events-none" />
        <div className="flex items-center justify-between text-slate-400 mb-2">
          <span className="text-xs font-medium uppercase tracking-wider">Tier 0 Cache Hits</span>
          <div className="p-2 rounded-lg bg-cyan-500/10 text-cyan-400">
            <Database className="w-4 h-4" />
          </div>
        </div>
        <div className="flex items-baseline gap-2 mb-1">
          <span className="text-3xl font-bold font-mono tracking-tight text-white">
            {cacheHits}
          </span>
          <span className="text-xs text-cyan-400 font-semibold">$0.00 cost</span>
        </div>
        <div className="text-xs text-slate-400 pt-2 border-t border-slate-800/80 flex items-center justify-between">
          <span>Sub-millisecond</span>
          <span className="text-cyan-400 font-mono font-medium">100% saved</span>
        </div>
      </div>

      {/* Card 3: Tier 1 Acceptance Ratio */}
      <div className="bg-[#0f172a] border border-slate-800 rounded-xl p-5 relative overflow-hidden group hover:border-emerald-500/50 transition-all duration-300">
        <div className="absolute top-0 right-0 w-32 h-32 bg-emerald-500/5 rounded-full blur-2xl group-hover:bg-emerald-500/10 transition-colors pointer-events-none" />
        <div className="flex items-center justify-between text-slate-400 mb-2">
          <span className="text-xs font-medium uppercase tracking-wider">Tier 1 Acceptance</span>
          <div className="p-2 rounded-lg bg-emerald-500/10 text-emerald-400">
            <Zap className="w-4 h-4" />
          </div>
        </div>
        <div className="flex items-baseline gap-2 mb-1">
          <span className="text-3xl font-bold font-mono tracking-tight text-white">
            {successRate.toFixed(1)}%
          </span>
          <span className="text-xs text-slate-400">
            ({summary?.tier1_accepted ?? 0})
          </span>
        </div>
        <div className="text-xs text-slate-400 pt-2 border-t border-slate-800/80 flex items-center justify-between">
          <span>Fast SLM Resolved</span>
          <span className="text-emerald-400 font-medium">95% cost cut</span>
        </div>
      </div>

      {/* Card 4: Quality Parity Score */}
      <div className="bg-[#0f172a] border border-slate-800 rounded-xl p-5 relative overflow-hidden group hover:border-purple-500/50 transition-all duration-300">
        <div className="absolute top-0 right-0 w-32 h-32 bg-purple-500/5 rounded-full blur-2xl group-hover:bg-purple-500/10 transition-colors pointer-events-none" />
        <div className="flex items-center justify-between text-slate-400 mb-2">
          <span className="text-xs font-medium uppercase tracking-wider">Quality Parity</span>
          <div className="p-2 rounded-lg bg-purple-500/10 text-purple-400">
            <ShieldCheck className="w-4 h-4" />
          </div>
        </div>
        <div className="flex items-baseline gap-2 mb-1">
          <span className="text-3xl font-bold font-mono tracking-tight text-white">
            {qualityParity.toFixed(1)}%
          </span>
          <span className="text-xs text-purple-400 font-medium">vs Frontier</span>
        </div>
        <div className="text-xs text-slate-400 pt-2 border-t border-slate-800/80 flex items-center justify-between">
          <span>Escalations: <strong className="font-mono text-amber-400">{summary?.tier3_escalated ?? 0}</strong></span>
          <span className="text-slate-400">Zero accuracy loss</span>
        </div>
      </div>

      {/* Card 5: Workload Latency & Spend */}
      <div className="bg-[#0f172a] border border-slate-800 rounded-xl p-5 relative overflow-hidden group hover:border-indigo-500/50 transition-all duration-300">
        <div className="absolute top-0 right-0 w-32 h-32 bg-indigo-500/5 rounded-full blur-2xl group-hover:bg-indigo-500/10 transition-colors pointer-events-none" />
        <div className="flex items-center justify-between text-slate-400 mb-2">
          <span className="text-xs font-medium uppercase tracking-wider">Latency & Month Cap</span>
          <div className="p-2 rounded-lg bg-indigo-500/10 text-indigo-400">
            {circuitBreaker ? <ShieldAlert className="w-4 h-4 text-red-400" /> : <Clock className="w-4 h-4" />}
          </div>
        </div>
        <div className="flex items-baseline gap-2 mb-1">
          <span className="text-3xl font-bold font-mono tracking-tight text-white">
            {avgLatency.toFixed(1)}
          </span>
          <span className="text-sm font-normal text-slate-400">ms</span>
        </div>
        <div className="text-xs text-slate-400 pt-2 border-t border-slate-800/80 flex items-center justify-between">
          <span>Month Spend: <strong className="font-mono text-slate-200">${monthlySpend.toFixed(2)}</strong></span>
          <span className={circuitBreaker ? 'text-red-400 font-bold' : 'text-emerald-400 font-medium'}>
            {circuitBreaker ? 'Tripped' : 'Normal'}
          </span>
        </div>
      </div>
    </div>
  );
};
