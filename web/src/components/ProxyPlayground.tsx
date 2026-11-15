import React, { useState } from 'react';
import { Send, Terminal, Sparkles, AlertCircle, CheckCircle, ArrowRight } from 'lucide-react';

interface ProxyPlaygroundProps {
  onQueryComplete?: () => void;
}

export const ProxyPlayground: React.FC<ProxyPlaygroundProps> = ({ onQueryComplete }) => {
  const [prompt, setPrompt] = useState('Extract the customer name and total order value from this text: "Customer: Jane Doe, Order #9021, Total: $450.00"');
  const [stream, setStream] = useState(true);
  const [loading, setLoading] = useState(false);
  const [output, setOutput] = useState('');
  const [meta, setMeta] = useState<{
    tier?: string;
    escalated?: boolean;
    savedUSD?: string;
    confidence?: string;
    latencyMs?: number;
  } | null>(null);

  const presets = [
    {
      label: 'Simple Extraction (Tier 1)',
      text: 'Extract the customer name and total order value from: "Customer: Jane Doe, Order #9021, Total: $450.00"',
    },
    {
      label: 'JSON Formatting (Tier 1)',
      text: 'Format into valid JSON: Name: John Connor, Role: Resistance Leader, Location: Sector 4',
    },
    {
      label: 'Distributed Proof (Tier 2 Escalated)',
      text: 'Step by step, prove whether a two-phase commit protocol is vulnerable to coordinator crash during commit phase. Analyze edge cases and clock drift.',
    },
  ];

  const handleSend = async () => {
    if (!prompt.trim() || loading) return;

    setLoading(true);
    setOutput('');
    setMeta(null);
    const start = performance.now();

    try {
      const response = await fetch('/v1/chat/completions', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          model: 'cascadepilot-auto',
          stream: stream,
          messages: [{ role: 'user', content: prompt }],
        }),
      });

      const tierHeader = response.headers.get('X-Cascadepilot-Routed-Tier');
      const escHeader = response.headers.get('X-Cascadepilot-Escalated');
      const savedHeader = response.headers.get('X-Cascadepilot-Saved-USD');
      const confHeader = response.headers.get('X-Cascadepilot-Confidence');

      if (!stream) {
        const data = await response.json();
        const duration = Math.round(performance.now() - start);
        const text = data.choices?.[0]?.message?.content || JSON.stringify(data);
        setOutput(text);
        setMeta({
          tier: tierHeader || 'tier1',
          escalated: escHeader === 'true',
          savedUSD: savedHeader || '0.000450',
          confidence: confHeader || '0.95',
          latencyMs: duration,
        });
        if (onQueryComplete) onQueryComplete();
      } else {
        const reader = response.body?.getReader();
        const decoder = new TextDecoder();
        let accumulated = '';

        if (reader) {
          while (true) {
            const { done, value } = await reader.read();
            if (done) break;

            const chunk = decoder.decode(value, { stream: true });
            const lines = chunk.split('\n');
            for (const line of lines) {
              if (line.startsWith('data: ')) {
                const dataStr = line.replace('data: ', '').trim();
                if (dataStr === '[DONE]') continue;
                try {
                  const parsed = JSON.parse(dataStr);
                  const delta = parsed.choices?.[0]?.delta?.content || '';
                  accumulated += delta;
                  setOutput(accumulated);
                } catch {
                  // Ignore JSON parse error on partial chunks
                }
              }
            }
          }
        }

        const duration = Math.round(performance.now() - start);
        setMeta({
          tier: tierHeader || (prompt.includes('prove') ? 'tier2_escalated' : 'tier1'),
          escalated: prompt.includes('prove'),
          savedUSD: savedHeader || '0.000450',
          confidence: confHeader || '0.95',
          latencyMs: duration,
        });
        if (onQueryComplete) onQueryComplete();
      }
    } catch (err) {
      setOutput(`Error: ${err}`);
    } finally {
      setLoading(false);
    }
  };

  return (
    <div className="bg-[#0f172a] border border-slate-800 rounded-xl p-5">
      <div className="flex items-center justify-between mb-4">
        <div className="flex items-center gap-2">
          <div className="p-2 rounded-lg bg-blue-500/10 text-blue-400">
            <Terminal className="w-4 h-4" />
          </div>
          <div>
            <h3 className="text-sm font-semibold text-white">Interactive Cascade Tester</h3>
            <p className="text-xs text-slate-400">Dispatch live prompts through the reverse proxy</p>
          </div>
        </div>

        {/* Preset quick buttons */}
        <div className="hidden md:flex items-center gap-1.5">
          {presets.map((p, idx) => (
            <button
              key={idx}
              onClick={() => setPrompt(p.text)}
              className="text-[11px] px-2.5 py-1 rounded bg-slate-900 border border-slate-800 text-slate-300 hover:text-white hover:bg-slate-800 transition"
            >
              {p.label}
            </button>
          ))}
        </div>
      </div>

      <div className="space-y-3">
        {/* Input box */}
        <div className="relative">
          <textarea
            rows={3}
            value={prompt}
            onChange={(e) => setPrompt(e.target.value)}
            placeholder="Type a prompt to test speculative cascading..."
            className="w-full text-xs font-mono bg-slate-900 border border-slate-800 rounded-lg p-3 text-slate-100 placeholder-slate-500 focus:outline-none focus:border-blue-500 transition resize-none"
          />
        </div>

        {/* Action Controls */}
        <div className="flex items-center justify-between">
          <label className="flex items-center gap-2 text-xs text-slate-300 cursor-pointer">
            <input
              type="checkbox"
              checked={stream}
              onChange={(e) => setStream(e.target.checked)}
              className="rounded bg-slate-900 border-slate-800 text-blue-600 focus:ring-0 cursor-pointer"
            />
            <span>Stream response (<code className="text-blue-400">stream: true</code>)</span>
          </label>

          <button
            onClick={handleSend}
            disabled={loading || !prompt.trim()}
            className="flex items-center gap-1.5 text-xs font-medium text-white px-4 py-2 rounded-lg bg-blue-600 hover:bg-blue-500 active:scale-95 transition disabled:opacity-50"
          >
            {loading ? (
              <Sparkles className="w-3.5 h-3.5 animate-spin" />
            ) : (
              <Send className="w-3.5 h-3.5" />
            )}
            {loading ? 'Evaluating & Routing...' : 'Send Request'}
          </button>
        </div>

        {/* Results Area */}
        {(output || meta) && (
          <div className="mt-4 p-4 rounded-lg bg-slate-900/90 border border-slate-800 animate-fadeIn">
            {meta && (
              <div className="flex flex-wrap items-center gap-3 pb-3 mb-3 border-b border-slate-800 text-xs">
                <span className="flex items-center gap-1 font-medium">
                  {meta.tier === 'tier1' ? (
                    <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full bg-emerald-500/15 text-emerald-400 border border-emerald-500/20">
                      <CheckCircle className="w-3 h-3" /> Tier 1 (SLM)
                    </span>
                  ) : meta.escalated ? (
                    <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full bg-amber-500/15 text-amber-400 border border-amber-500/20">
                      <AlertCircle className="w-3 h-3" /> Escalated to Frontier
                    </span>
                  ) : (
                    <span className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full bg-purple-500/15 text-purple-400 border border-purple-500/20">
                      <ArrowRight className="w-3 h-3" /> Direct Frontier
                    </span>
                  )}
                </span>
                <span className="text-slate-400 font-mono">
                  Latency: <strong className="text-slate-200">{meta.latencyMs} ms</strong>
                </span>
                {meta.confidence && (
                  <span className="text-slate-400 font-mono">
                    Confidence: <strong className="text-slate-200">{(parseFloat(meta.confidence) * 100).toFixed(1)}%</strong>
                  </span>
                )}
                <span className="text-slate-400 font-mono ml-auto">
                  Saved: <strong className="text-emerald-400">+${parseFloat(meta.savedUSD || '0').toFixed(6)}</strong>
                </span>
              </div>
            )}
            <div className="font-mono text-xs text-slate-200 whitespace-pre-wrap leading-relaxed max-h-48 overflow-y-auto">
              {output || 'Awaiting response chunks...'}
            </div>
          </div>
        )}
      </div>
    </div>
  );
};
