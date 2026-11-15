import React from 'react';
import { X, CheckCircle, Zap, TrendingDown } from 'lucide-react';
import { CompareResult } from '../types';

interface CompareModalProps {
  result: CompareResult | null;
  onClose: () => void;
}

export const CompareModal: React.FC<CompareModalProps> = ({ result, onClose }) => {
  if (!result) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/70 backdrop-blur-sm animate-fadeIn">
      <div className="bg-[#0f172a] border border-slate-700 rounded-2xl max-w-5xl w-full p-6 shadow-2xl relative max-h-[90vh] flex flex-col">
        <button
          onClick={onClose}
          className="absolute top-4 right-4 p-1.5 rounded-lg text-slate-400 hover:text-white bg-slate-800/80 hover:bg-slate-700 transition"
        >
          <X className="w-5 h-5" />
        </button>

        <div className="mb-4">
          <div className="flex items-center gap-2 mb-1">
            <span className="p-1.5 rounded-lg bg-blue-500/10 text-blue-400">
              <Zap className="w-4 h-4" />
            </span>
            <h3 className="text-base font-bold text-white">Side-by-Side Model Benchmark</h3>
          </div>
          <p className="text-xs text-slate-400">
            Parallel execution across all tiers compared with Cascadepilot's dynamic decision
          </p>
        </div>

        {/* Prompt Header */}
        <div className="p-3 rounded-lg bg-slate-900 border border-slate-800 text-xs font-mono text-slate-300 mb-4 max-h-20 overflow-y-auto">
          <span className="text-slate-500 font-sans block text-[10px] uppercase font-bold mb-0.5">Prompt Tested</span>
          {result.prompt}
        </div>

        {/* 4-Column Comparison Grid */}
        <div className="grid grid-cols-1 md:grid-cols-4 gap-3 flex-1 overflow-y-auto pr-1">
          {/* Card 1: Tier 1 SLM */}
          <div className="bg-slate-900/70 border border-slate-800 rounded-xl p-3.5 flex flex-col justify-between">
            <div>
              <div className="flex items-center justify-between text-xs mb-2">
                <span className="font-semibold text-emerald-400">Tier 1: SLM</span>
                <span className="font-mono text-[10px] text-slate-400">{result.tier1_result.latency_ms} ms</span>
              </div>
              <div className="text-[11px] font-mono text-slate-300 mb-2">{result.tier1_result.model}</div>
              <div className="text-xs font-mono text-slate-300 bg-black/40 p-2.5 rounded-lg border border-slate-800/80 max-h-48 overflow-y-auto whitespace-pre-wrap leading-relaxed">
                {result.tier1_result.content}
              </div>
            </div>
            <div className="mt-3 pt-2 border-t border-slate-800/80 text-[11px] flex justify-between font-mono text-slate-400">
              <span>Tokens: {result.tier1_result.prompt_tokens + result.tier1_result.completion_tokens}</span>
              <span className="text-emerald-400 font-semibold">${result.tier1_result.cost_usd.toFixed(6)}</span>
            </div>
          </div>

          {/* Card 2: Tier 2 Mid-tier */}
          <div className="bg-slate-900/70 border border-slate-800 rounded-xl p-3.5 flex flex-col justify-between">
            <div>
              <div className="flex items-center justify-between text-xs mb-2">
                <span className="font-semibold text-blue-400">Tier 2: Mid-tier</span>
                <span className="font-mono text-[10px] text-slate-400">{result.tier2_result.latency_ms} ms</span>
              </div>
              <div className="text-[11px] font-mono text-slate-300 mb-2">{result.tier2_result.model}</div>
              <div className="text-xs font-mono text-slate-300 bg-black/40 p-2.5 rounded-lg border border-slate-800/80 max-h-48 overflow-y-auto whitespace-pre-wrap leading-relaxed">
                {result.tier2_result.content}
              </div>
            </div>
            <div className="mt-3 pt-2 border-t border-slate-800/80 text-[11px] flex justify-between font-mono text-slate-400">
              <span>Tokens: {result.tier2_result.prompt_tokens + result.tier2_result.completion_tokens}</span>
              <span className="text-blue-400 font-semibold">${result.tier2_result.cost_usd.toFixed(6)}</span>
            </div>
          </div>

          {/* Card 3: Tier 3 Frontier */}
          <div className="bg-slate-900/70 border border-slate-800 rounded-xl p-3.5 flex flex-col justify-between">
            <div>
              <div className="flex items-center justify-between text-xs mb-2">
                <span className="font-semibold text-purple-400">Tier 3: Frontier</span>
                <span className="font-mono text-[10px] text-slate-400">{result.tier3_result.latency_ms} ms</span>
              </div>
              <div className="text-[11px] font-mono text-slate-300 mb-2">{result.tier3_result.model}</div>
              <div className="text-xs font-mono text-slate-300 bg-black/40 p-2.5 rounded-lg border border-slate-800/80 max-h-48 overflow-y-auto whitespace-pre-wrap leading-relaxed">
                {result.tier3_result.content}
              </div>
            </div>
            <div className="mt-3 pt-2 border-t border-slate-800/80 text-[11px] flex justify-between font-mono text-slate-400">
              <span>Tokens: {result.tier3_result.prompt_tokens + result.tier3_result.completion_tokens}</span>
              <span className="text-purple-400 font-semibold">${result.tier3_result.cost_usd.toFixed(6)}</span>
            </div>
          </div>

          {/* Card 4: Cascadepilot Smart Cascade */}
          <div className="bg-gradient-to-b from-blue-950/40 to-slate-900 border-2 border-blue-500/60 rounded-xl p-3.5 flex flex-col justify-between shadow-lg shadow-blue-500/10">
            <div>
              <div className="flex items-center justify-between text-xs mb-2">
                <span className="font-bold text-white flex items-center gap-1">
                  <CheckCircle className="w-3.5 h-3.5 text-blue-400" />
                  cascadepilot
                </span>
                <span className="font-mono text-[10px] text-blue-300">{result.cascade_route?.latency_ms} ms</span>
              </div>
              <div className="text-[11px] font-mono text-blue-400 mb-2 flex items-center justify-between">
                <span>Routed: {result.cascade_route?.routed_tier}</span>
                <span>Conf: {((result.cascade_route?.tier1_confidence || 1) * 100).toFixed(0)}%</span>
              </div>
              <div className="p-2 rounded-lg bg-blue-900/30 border border-blue-500/30 text-xs text-blue-200 mb-2">
                {result.cascade_route?.routed_tier === 'tier0_cache' && 'Resolved at zero cost from Tier 0 Cache.'}
                {result.cascade_route?.routed_tier === 'tier1' && 'Confidence gate passed. Tier 1 completion accepted.'}
                {result.cascade_route?.routed_tier === 'tier2_mid' && 'Escalated to mid-tier. Saved 90% vs frontier.'}
                {(result.cascade_route?.routed_tier === 'tier3_direct' || result.cascade_route?.routed_tier === 'tier3_escalated') && 'Complexity/uncertainty flagged. Escalated to frontier.'}
              </div>
            </div>
            <div className="mt-3 pt-2 border-t border-slate-800/80 text-xs flex items-center justify-between font-mono">
              <span className="text-slate-400">Net Saved:</span>
              <span className="text-emerald-400 font-bold flex items-center gap-0.5">
                <TrendingDown className="w-3.5 h-3.5" />
                +${result.cascade_route?.saved_cost_usd.toFixed(6)}
              </span>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
};
