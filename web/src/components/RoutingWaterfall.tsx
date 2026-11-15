import React from 'react';
import { GitFork, CheckCircle2, AlertTriangle, ArrowRight, Database } from 'lucide-react';
import { WaterfallMetrics, FinOpsSummary } from '../types';

interface RoutingWaterfallProps {
  waterfall: WaterfallMetrics | null;
  summary: FinOpsSummary | null;
}

export const RoutingWaterfall: React.FC<RoutingWaterfallProps> = ({ waterfall, summary }) => {
  const total = waterfall?.total_queries ?? summary?.total_requests ?? 0;
  const t0Cache = waterfall?.tier0_cache_hits ?? summary?.tier0_cache_hits ?? 0;
  const t1Passed = waterfall?.tier1_passed ?? summary?.tier1_accepted ?? 0;
  const t2Mid = waterfall?.tier2_mid_passed ?? summary?.tier2_midtier_count ?? 0;
  const t3Direct = waterfall?.direct_tier3_count ?? summary?.tier3_direct ?? 0;
  const t3Escalated = waterfall?.tier3_escalated ?? summary?.tier3_escalated ?? 0;

  const t0Pct = total > 0 ? (t0Cache / total) * 100 : 0;
  const t1PassedPct = total > 0 ? (t1Passed / total) * 100 : 0;
  const t2MidPct = total > 0 ? (t2Mid / total) * 100 : 0;
  const t3Pct = total > 0 ? ((t3Direct + t3Escalated) / total) * 100 : 0;

  return (
    <div className="bg-[#0f172a] border border-slate-800 rounded-xl p-5 flex flex-col justify-between">
      <div className="flex items-center justify-between mb-4">
        <div className="flex items-center gap-2">
          <div className="p-2 rounded-lg bg-blue-500/10 text-blue-400">
            <GitFork className="w-4 h-4" />
          </div>
          <div>
            <h3 className="text-sm font-semibold text-white">Speculative 3-Tier Waterfall</h3>
            <p className="text-xs text-slate-400">Traffic distribution across Cache, SLM, Mid-tier, and Frontier</p>
          </div>
        </div>
        <span className="text-xs font-mono text-slate-400 bg-slate-800/80 px-2 py-1 rounded">
          {total} Total Queries
        </span>
      </div>

      {/* Waterfall Segmented Bar */}
      <div className="mb-5">
        <div className="h-4 w-full bg-slate-900 rounded-full overflow-hidden flex p-0.5 border border-slate-800">
          <div
            style={{ width: `${t0Pct}%` }}
            className="bg-cyan-500 h-full rounded-l-full transition-all duration-500 relative group"
            title={`Tier 0 Cache: ${t0Pct.toFixed(1)}%`}
          />
          <div
            style={{ width: `${t1PassedPct}%` }}
            className="bg-emerald-500 h-full transition-all duration-500 relative group"
            title={`Tier 1 Accepted: ${t1PassedPct.toFixed(1)}%`}
          />
          <div
            style={{ width: `${t2MidPct}%` }}
            className="bg-blue-500 h-full transition-all duration-500 relative group"
            title={`Tier 2 Mid-tier: ${t2MidPct.toFixed(1)}%`}
          />
          <div
            style={{ width: `${t3Pct}%` }}
            className="bg-purple-500 h-full rounded-r-full transition-all duration-500 relative group"
            title={`Tier 3 Frontier: ${t3Pct.toFixed(1)}%`}
          />
        </div>
        <div className="flex flex-wrap justify-between text-[11px] text-slate-400 mt-2 gap-2">
          <span className="flex items-center gap-1.5">
            <span className="w-2.5 h-2.5 rounded-full bg-cyan-500 inline-block" />
            T0 Cache ({t0Pct.toFixed(1)}%)
          </span>
          <span className="flex items-center gap-1.5">
            <span className="w-2.5 h-2.5 rounded-full bg-emerald-500 inline-block" />
            T1 SLM ({t1PassedPct.toFixed(1)}%)
          </span>
          <span className="flex items-center gap-1.5">
            <span className="w-2.5 h-2.5 rounded-full bg-blue-500 inline-block" />
            T2 Mid-tier ({t2MidPct.toFixed(1)}%)
          </span>
          <span className="flex items-center gap-1.5">
            <span className="w-2.5 h-2.5 rounded-full bg-purple-500 inline-block" />
            T3 Frontier ({t3Pct.toFixed(1)}%)
          </span>
        </div>
      </div>

      {/* Process Flow Cards */}
      <div className="grid grid-cols-2 md:grid-cols-4 gap-2.5">
        <div className="p-2.5 rounded-lg bg-slate-900/80 border border-slate-800">
          <div className="flex items-center justify-between text-xs text-slate-400 mb-1">
            <span className="font-medium text-cyan-400 flex items-center gap-1">
              <Database className="w-3 h-3" /> T0 Cache
            </span>
            <span className="font-mono text-white font-bold">{t0Cache}</span>
          </div>
          <p className="text-[10px] text-slate-400 leading-tight">
            Sub-millisecond hit at $0.00 cost.
          </p>
        </div>

        <div className="p-2.5 rounded-lg bg-slate-900/80 border border-slate-800">
          <div className="flex items-center justify-between text-xs text-slate-400 mb-1">
            <span className="font-medium text-emerald-400 flex items-center gap-1">
              <CheckCircle2 className="w-3 h-3" /> T1 SLM
            </span>
            <span className="font-mono text-white font-bold">{t1Passed}</span>
          </div>
          <p className="text-[10px] text-slate-400 leading-tight">
            Resolved by fast SLM ($0.075/1M).
          </p>
        </div>

        <div className="p-2.5 rounded-lg bg-slate-900/80 border border-slate-800">
          <div className="flex items-center justify-between text-xs text-slate-400 mb-1">
            <span className="font-medium text-blue-400 flex items-center gap-1">
              <ArrowRight className="w-3 h-3" /> T2 Mid-tier
            </span>
            <span className="font-mono text-white font-bold">{t2Mid}</span>
          </div>
          <p className="text-[10px] text-slate-400 leading-tight">
            Moderate reasoning ($0.15/1M).
          </p>
        </div>

        <div className="p-2.5 rounded-lg bg-slate-900/80 border border-slate-800">
          <div className="flex items-center justify-between text-xs text-slate-400 mb-1">
            <span className="font-medium text-purple-400 flex items-center gap-1">
              <AlertTriangle className="w-3 h-3" /> T3 Frontier
            </span>
            <span className="font-mono text-white font-bold">{t3Direct + t3Escalated}</span>
          </div>
          <p className="text-[10px] text-slate-400 leading-tight">
            Deep complexity & proof tasks.
          </p>
        </div>
      </div>
    </div>
  );
};
