import React from 'react';
import { GitFork, CheckCircle2, AlertTriangle, ArrowRight } from 'lucide-react';
import { WaterfallMetrics, FinOpsSummary } from '../types';

interface RoutingWaterfallProps {
  waterfall: WaterfallMetrics | null;
  summary: FinOpsSummary | null;
}

export const RoutingWaterfall: React.FC<RoutingWaterfallProps> = ({ waterfall, summary }) => {
  const total = waterfall?.total_queries ?? summary?.total_requests ?? 0;
  const directT2 = waterfall?.direct_tier2_count ?? summary?.tier2_direct ?? 0;
  const t1Passed = waterfall?.tier1_passed ?? summary?.tier1_accepted ?? 0;
  const t1Escalated = waterfall?.tier1_escalated ?? summary?.tier2_escalated ?? 0;

  const t1PassedPct = total > 0 ? (t1Passed / total) * 100 : 0;
  const t1EscalatedPct = total > 0 ? (t1Escalated / total) * 100 : 0;
  const directT2Pct = total > 0 ? (directT2 / total) * 100 : 0;

  return (
    <div className="bg-[#0f172a] border border-slate-800 rounded-xl p-5 flex flex-col justify-between">
      <div className="flex items-center justify-between mb-4">
        <div className="flex items-center gap-2">
          <div className="p-2 rounded-lg bg-blue-500/10 text-blue-400">
            <GitFork className="w-4 h-4" />
          </div>
          <div>
            <h3 className="text-sm font-semibold text-white">Cascading Routing Waterfall</h3>
            <p className="text-xs text-slate-400">Speculative execution distribution</p>
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
            style={{ width: `${t1PassedPct}%` }}
            className="bg-emerald-500 h-full rounded-l-full transition-all duration-500 relative group"
            title={`Tier 1 Accepted: ${t1PassedPct.toFixed(1)}%`}
          />
          <div
            style={{ width: `${t1EscalatedPct}%` }}
            className="bg-amber-500 h-full transition-all duration-500 relative group"
            title={`Tier 1 Escalated: ${t1EscalatedPct.toFixed(1)}%`}
          />
          <div
            style={{ width: `${directT2Pct}%` }}
            className="bg-purple-500 h-full rounded-r-full transition-all duration-500 relative group"
            title={`Tier 2 Direct: ${directT2Pct.toFixed(1)}%`}
          />
        </div>
        <div className="flex justify-between text-[11px] text-slate-400 mt-2">
          <span className="flex items-center gap-1.5">
            <span className="w-2.5 h-2.5 rounded-full bg-emerald-500 inline-block" />
            Tier 1 Accepted ({t1PassedPct.toFixed(1)}%)
          </span>
          <span className="flex items-center gap-1.5">
            <span className="w-2.5 h-2.5 rounded-full bg-amber-500 inline-block" />
            Tier 1 Escalated ({t1EscalatedPct.toFixed(1)}%)
          </span>
          <span className="flex items-center gap-1.5">
            <span className="w-2.5 h-2.5 rounded-full bg-purple-500 inline-block" />
            Direct Frontier ({directT2Pct.toFixed(1)}%)
          </span>
        </div>
      </div>

      {/* Process Flow Cards */}
      <div className="grid grid-cols-1 md:grid-cols-3 gap-3">
        {/* Step 1 */}
        <div className="p-3 rounded-lg bg-slate-900/80 border border-slate-800">
          <div className="flex items-center justify-between text-xs text-slate-400 mb-1">
            <span className="font-medium text-emerald-400 flex items-center gap-1">
              <CheckCircle2 className="w-3.5 h-3.5" /> Direct Tier 1
            </span>
            <span className="font-mono text-white font-bold">{t1Passed}</span>
          </div>
          <p className="text-[11px] text-slate-400 leading-tight">
            Resolved by fast SLM at $0.075/1M tokens. Delivered with minimum TTFT.
          </p>
        </div>

        {/* Step 2 */}
        <div className="p-3 rounded-lg bg-slate-900/80 border border-slate-800">
          <div className="flex items-center justify-between text-xs text-slate-400 mb-1">
            <span className="font-medium text-amber-400 flex items-center gap-1">
              <AlertTriangle className="w-3.5 h-3.5" /> Escalated T1→T2
            </span>
            <span className="font-mono text-white font-bold">{t1Escalated}</span>
          </div>
          <p className="text-[11px] text-slate-400 leading-tight">
            Flagged by confidence or refusal gate. Dispatched seamlessly to frontier.
          </p>
        </div>

        {/* Step 3 */}
        <div className="p-3 rounded-lg bg-slate-900/80 border border-slate-800">
          <div className="flex items-center justify-between text-xs text-slate-400 mb-1">
            <span className="font-medium text-purple-400 flex items-center gap-1">
              <ArrowRight className="w-3.5 h-3.5" /> Direct Frontier
            </span>
            <span className="font-mono text-white font-bold">{directT2}</span>
          </div>
          <p className="text-[11px] text-slate-400 leading-tight">
            Bypassed Tier 1 due to high complexity score (&gt; &theta;<sub>bypass</sub>).
          </p>
        </div>
      </div>
    </div>
  );
};
