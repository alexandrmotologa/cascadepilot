import React, { useState } from 'react';
import { Activity, Download, Search, X, CheckCircle, AlertTriangle, ArrowRight } from 'lucide-react';
import { RequestRecord } from '../types';

interface RequestLiveLogProps {
  requests: RequestRecord[];
  onExportCSV: () => void;
  onExportJSON: () => void;
  isConnected: boolean;
}

export const RequestLiveLog: React.FC<RequestLiveLogProps> = ({
  requests,
  onExportCSV,
  onExportJSON,
  isConnected,
}) => {
  const [searchTerm, setSearchTerm] = useState('');
  const [selectedReq, setSelectedReq] = useState<RequestRecord | null>(null);

  const filtered = requests.filter((r) => {
    if (!searchTerm) return true;
    const term = searchTerm.toLowerCase();
    return (
      r.prompt_preview.toLowerCase().includes(term) ||
      r.routed_tier.toLowerCase().includes(term) ||
      r.id.toLowerCase().includes(term)
    );
  });

  return (
    <div className="bg-[#0f172a] border border-slate-800 rounded-xl overflow-hidden">
      {/* Header bar */}
      <div className="p-4 border-b border-slate-800 flex flex-wrap items-center justify-between gap-3">
        <div className="flex items-center gap-2">
          <div className="p-2 rounded-lg bg-emerald-500/10 text-emerald-400">
            <Activity className="w-4 h-4" />
          </div>
          <div>
            <div className="flex items-center gap-2">
              <h3 className="text-sm font-semibold text-white">Live Query Stream</h3>
              <span className="flex items-center gap-1 text-[10px] font-mono px-2 py-0.5 rounded-full bg-slate-800 border border-slate-700 text-slate-300">
                <span className={`w-1.5 h-1.5 rounded-full ${isConnected ? 'bg-emerald-400 animate-pulse' : 'bg-red-400'}`} />
                {isConnected ? 'LIVE SSE' : 'OFFLINE'}
              </span>
            </div>
            <p className="text-xs text-slate-400">Real-time inspection of routed completions</p>
          </div>
        </div>

        <div className="flex items-center gap-2">
          <div className="relative">
            <Search className="w-3.5 h-3.5 text-slate-400 absolute left-2.5 top-1/2 -translate-y-1/2" />
            <input
              type="text"
              placeholder="Filter queries..."
              value={searchTerm}
              onChange={(e) => setSearchTerm(e.target.value)}
              className="text-xs bg-slate-900 border border-slate-800 rounded-lg pl-8 pr-3 py-1.5 text-slate-200 placeholder-slate-500 focus:outline-none focus:border-blue-500 transition w-44 md:w-56"
            />
          </div>

          <button
            onClick={onExportCSV}
            title="Download CSV"
            className="flex items-center gap-1 text-xs text-slate-300 px-2.5 py-1.5 rounded-lg bg-slate-900 border border-slate-800 hover:bg-slate-800 hover:text-white transition"
          >
            <Download className="w-3.5 h-3.5" />
            CSV
          </button>
          <button
            onClick={onExportJSON}
            title="Download JSON"
            className="flex items-center gap-1 text-xs text-slate-300 px-2.5 py-1.5 rounded-lg bg-slate-900 border border-slate-800 hover:bg-slate-800 hover:text-white transition"
          >
            <Download className="w-3.5 h-3.5" />
            JSON
          </button>
        </div>
      </div>

      {/* Table */}
      <div className="overflow-x-auto max-h-[380px] overflow-y-auto">
        <table className="w-full text-left text-xs border-collapse">
          <thead className="bg-slate-900/80 sticky top-0 border-b border-slate-800 text-slate-400 font-medium">
            <tr>
              <th className="py-2.5 px-3">Time</th>
              <th className="py-2.5 px-3">Prompt Preview</th>
              <th className="py-2.5 px-3">Complexity (S_comp)</th>
              <th className="py-2.5 px-3">Routed Tier</th>
              <th className="py-2.5 px-3">Confidence</th>
              <th className="py-2.5 px-3">Latency</th>
              <th className="py-2.5 px-3 text-right">Net Saved</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-slate-800/60 font-sans">
            {filtered.length === 0 ? (
              <tr>
                <td colSpan={7} className="py-8 text-center text-slate-400">
                  No requests recorded yet. Send queries to <code className="text-blue-400 bg-slate-900 px-1 py-0.5 rounded">/v1/chat/completions</code> or run a simulation.
                </td>
              </tr>
            ) : (
              filtered.map((req) => (
                <tr
                  key={req.id}
                  onClick={() => setSelectedReq(req)}
                  className="hover:bg-slate-800/50 cursor-pointer transition-colors group"
                >
                  <td className="py-2.5 px-3 font-mono text-[11px] text-slate-400 whitespace-nowrap">
                    {new Date(req.timestamp).toLocaleTimeString()}
                  </td>
                  <td className="py-2.5 px-3 max-w-[260px] truncate text-slate-200 group-hover:text-white font-normal">
                    {req.prompt_preview}
                  </td>
                  <td className="py-2.5 px-3 font-mono text-[11px] whitespace-nowrap">
                    <span
                      className={`px-1.5 py-0.5 rounded ${
                        req.complexity_score > 0.65
                          ? 'bg-purple-500/15 text-purple-400 border border-purple-500/20'
                          : 'bg-slate-800 text-slate-300'
                      }`}
                    >
                      {req.complexity_score.toFixed(3)}
                    </span>
                  </td>
                  <td className="py-2.5 px-3 whitespace-nowrap">
                    {req.routed_tier === 'tier1' ? (
                      <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-[11px] font-medium bg-emerald-500/15 text-emerald-400 border border-emerald-500/20">
                        <CheckCircle className="w-3 h-3" />
                        Tier 1 (SLM)
                      </span>
                    ) : req.routed_tier === 'tier2_escalated' ? (
                      <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-[11px] font-medium bg-amber-500/15 text-amber-400 border border-amber-500/20">
                        <AlertTriangle className="w-3 h-3" />
                        T1 Escalated
                      </span>
                    ) : (
                      <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full text-[11px] font-medium bg-purple-500/15 text-purple-400 border border-purple-500/20">
                        <ArrowRight className="w-3 h-3" />
                        Tier 2 Direct
                      </span>
                    )}
                  </td>
                  <td className="py-2.5 px-3 font-mono text-[11px] whitespace-nowrap text-slate-300">
                    {(req.tier1_confidence * 100).toFixed(1)}%
                  </td>
                  <td className="py-2.5 px-3 font-mono text-[11px] text-slate-400 whitespace-nowrap">
                    {req.latency_ms} ms
                  </td>
                  <td className="py-2.5 px-3 font-mono text-[11px] font-semibold text-right whitespace-nowrap">
                    {req.saved_cost_usd > 0 ? (
                      <span className="text-emerald-400">+${req.saved_cost_usd.toFixed(6)}</span>
                    ) : (
                      <span className="text-slate-400">${req.saved_cost_usd.toFixed(6)}</span>
                    )}
                  </td>
                </tr>
              ))
            )}
          </tbody>
        </table>
      </div>

      {/* Detail Drawer Modal */}
      {selectedReq && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60 backdrop-blur-sm animate-fadeIn">
          <div className="bg-[#0f172a] border border-slate-700 rounded-xl max-w-xl w-full p-6 shadow-2xl relative">
            <button
              onClick={() => setSelectedReq(null)}
              className="absolute top-4 right-4 p-1.5 rounded-lg text-slate-400 hover:text-white bg-slate-800/80 hover:bg-slate-700 transition"
            >
              <X className="w-4 h-4" />
            </button>

            <div className="flex items-center gap-2 mb-4">
              <span className="font-mono text-xs px-2 py-0.5 rounded bg-blue-500/15 text-blue-400 border border-blue-500/20">
                {selectedReq.id}
              </span>
              <span className="text-xs text-slate-400">
                {new Date(selectedReq.timestamp).toLocaleString()}
              </span>
            </div>

            <h4 className="text-xs font-semibold uppercase tracking-wider text-slate-400 mb-1">
              Prompt Preview
            </h4>
            <div className="p-3 rounded-lg bg-slate-900 border border-slate-800 text-xs font-mono text-slate-200 mb-4 whitespace-pre-wrap">
              {selectedReq.prompt_preview}
            </div>

            <div className="grid grid-cols-2 gap-3 text-xs mb-4">
              <div className="p-3 rounded-lg bg-slate-900/60 border border-slate-800">
                <span className="text-slate-400 block mb-1">Routing Path</span>
                <span className="font-medium text-white">{selectedReq.routed_tier}</span>
                {selectedReq.escalated && (
                  <span className="block text-[11px] text-amber-400 mt-1">
                    Reason: {selectedReq.escalation_reason || 'Low confidence'}
                  </span>
                )}
              </div>
              <div className="p-3 rounded-lg bg-slate-900/60 border border-slate-800">
                <span className="text-slate-400 block mb-1">Complexity & Confidence</span>
                <div className="flex justify-between font-mono">
                  <span>Score: {selectedReq.complexity_score}</span>
                  <span>Conf: {(selectedReq.tier1_confidence * 100).toFixed(1)}%</span>
                </div>
              </div>
              <div className="p-3 rounded-lg bg-slate-900/60 border border-slate-800">
                <span className="text-slate-400 block mb-1">Token Metrics</span>
                <div className="font-mono text-slate-300">
                  <span>{selectedReq.prompt_tokens} in / {selectedReq.completion_tokens} out</span>
                </div>
              </div>
              <div className="p-3 rounded-lg bg-slate-900/60 border border-slate-800">
                <span className="text-slate-400 block mb-1">FinOps Economics</span>
                <div className="font-mono text-emerald-400 font-semibold">
                  Saved: ${selectedReq.saved_cost_usd.toFixed(6)}
                </div>
                <div className="font-mono text-[10px] text-slate-400 mt-0.5">
                  Base: ${selectedReq.baseline_cost_usd.toFixed(6)} | Act: ${selectedReq.actual_cost_usd.toFixed(6)}
                </div>
              </div>
            </div>

            <div className="flex justify-end">
              <button
                onClick={() => setSelectedReq(null)}
                className="text-xs px-4 py-2 rounded-lg bg-slate-800 text-white hover:bg-slate-700 transition"
              >
                Close Inspector
              </button>
            </div>
          </div>
        </div>
      )}
    </div>
  );
};
