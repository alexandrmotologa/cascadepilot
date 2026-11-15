import React from 'react';
import { Gauge, Clock, Zap } from 'lucide-react';
import { LatencyMetrics } from '../types';

interface LatencyDiffChartProps {
  latency: LatencyMetrics | null;
}

export const LatencyDiffChart: React.FC<LatencyDiffChartProps> = ({ latency }) => {
  const cache = latency?.avg_cache_ms ?? 0.5;
  const t1 = latency?.avg_tier1_ms ?? 14.5;
  const t2Mid = latency?.avg_tier2_mid_ms ?? 18.0;
  const directT3 = latency?.avg_direct_t3_ms ?? 34.0;
  const escalated = latency?.avg_escalated_ms ?? 38.2;
  const overall = latency?.avg_overall_ms ?? 16.8;

  const maxVal = Math.max(cache, t1, t2Mid, directT3, escalated, overall, 50);

  return (
    <div className="bg-[#0f172a] border border-slate-800 rounded-xl p-5 flex flex-col justify-between">
      <div className="flex items-center justify-between mb-4">
        <div className="flex items-center gap-2">
          <div className="p-2 rounded-lg bg-cyan-500/10 text-cyan-400">
            <Gauge className="w-4 h-4" />
          </div>
          <div>
            <h3 className="text-sm font-semibold text-white">Latency Distribution Across Tiers</h3>
            <p className="text-xs text-slate-400">Response time profile: Cache vs SLM vs Mid-tier vs Frontier</p>
          </div>
        </div>
        <span className="text-xs font-mono text-cyan-400 bg-cyan-500/10 border border-cyan-500/20 px-2 py-0.5 rounded flex items-center gap-1">
          <Zap className="w-3 h-3" />
          {overall > 0 ? (directT3 / Math.max(1, overall)).toFixed(1) : '2.1'}x Speedup
        </span>
      </div>

      {/* Latency Bars */}
      <div className="space-y-2.5 mb-2">
        {/* Tier 0 Cache */}
        <div>
          <div className="flex justify-between text-xs mb-1">
            <span className="text-slate-300 flex items-center gap-1">
              <span className="w-2 h-2 rounded-full bg-cyan-400 inline-block" />
              Tier 0 Cache Hit
            </span>
            <span className="font-mono text-cyan-400 font-bold">{cache.toFixed(1)} ms</span>
          </div>
          <div className="h-1.5 w-full bg-slate-900 rounded-full overflow-hidden">
            <div
              style={{ width: `${Math.min(100, (cache / maxVal) * 100)}%` }}
              className="bg-cyan-500 h-full rounded-full transition-all duration-500"
            />
          </div>
        </div>

        {/* Tier 1 SLM */}
        <div>
          <div className="flex justify-between text-xs mb-1">
            <span className="text-slate-300 flex items-center gap-1">
              <span className="w-2 h-2 rounded-full bg-emerald-400 inline-block" />
              Tier 1 SLM Direct
            </span>
            <span className="font-mono text-emerald-400 font-bold">{t1.toFixed(1)} ms</span>
          </div>
          <div className="h-1.5 w-full bg-slate-900 rounded-full overflow-hidden">
            <div
              style={{ width: `${Math.min(100, (t1 / maxVal) * 100)}%` }}
              className="bg-emerald-500 h-full rounded-full transition-all duration-500"
            />
          </div>
        </div>

        {/* Tier 2 Mid-tier */}
        <div>
          <div className="flex justify-between text-xs mb-1">
            <span className="text-slate-300 flex items-center gap-1">
              <span className="w-2 h-2 rounded-full bg-blue-400 inline-block" />
              Tier 2 Mid-tier
            </span>
            <span className="font-mono text-blue-400 font-bold">{t2Mid.toFixed(1)} ms</span>
          </div>
          <div className="h-1.5 w-full bg-slate-900 rounded-full overflow-hidden">
            <div
              style={{ width: `${Math.min(100, (t2Mid / maxVal) * 100)}%` }}
              className="bg-blue-500 h-full rounded-full transition-all duration-500"
            />
          </div>
        </div>

        {/* Tier 3 Frontier Direct */}
        <div>
          <div className="flex justify-between text-xs mb-1">
            <span className="text-slate-300 flex items-center gap-1">
              <span className="w-2 h-2 rounded-full bg-purple-400 inline-block" />
              Tier 3 Frontier Direct
            </span>
            <span className="font-mono text-purple-400 font-bold">{directT3.toFixed(1)} ms</span>
          </div>
          <div className="h-1.5 w-full bg-slate-900 rounded-full overflow-hidden">
            <div
              style={{ width: `${Math.min(100, (directT3 / maxVal) * 100)}%` }}
              className="bg-purple-500 h-full rounded-full transition-all duration-500"
            />
          </div>
        </div>

        {/* Tier 3 Escalated */}
        <div>
          <div className="flex justify-between text-xs mb-1">
            <span className="text-slate-300 flex items-center gap-1">
              <span className="w-2 h-2 rounded-full bg-amber-400 inline-block" />
              Tier 3 Escalated (T1 + T3)
            </span>
            <span className="font-mono text-amber-400 font-bold">{escalated.toFixed(1)} ms</span>
          </div>
          <div className="h-1.5 w-full bg-slate-900 rounded-full overflow-hidden">
            <div
              style={{ width: `${Math.min(100, (escalated / maxVal) * 100)}%` }}
              className="bg-amber-500 h-full rounded-full transition-all duration-500"
            />
          </div>
        </div>
      </div>

      <div className="pt-3 border-t border-slate-800 text-[11px] text-slate-400 flex items-center justify-between">
        <span className="flex items-center gap-1">
          <Clock className="w-3.5 h-3.5 text-slate-400" />
          Weighted overall response time: <strong className="font-mono text-white">{overall.toFixed(1)} ms</strong>
        </span>
        <span className="font-mono text-slate-300">Proxy overhead: &lt; 0.1ms</span>
      </div>
    </div>
  );
};
