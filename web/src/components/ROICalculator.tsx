import React, { useState } from 'react';
import { Calculator, DollarSign, TrendingDown, Sparkles, Check, Copy } from 'lucide-react';

export const ROICalculator: React.FC = () => {
  const [monthlyQueries, setMonthlyQueries] = useState(2000000); // 2 million queries / mo
  const [promptTokens, setPromptTokens] = useState(800);
  const [completionTokens, setCompletionTokens] = useState(300);
  const [baselineModel, setBaselineModel] = useState<'claude-sonnet' | 'gpt4o' | 'o1'>('claude-sonnet');
  const [copied, setCopied] = useState(false);

  // Pricing per 1M tokens
  const pricingTable = {
    'claude-sonnet': { name: 'Claude 3.5 Sonnet', in: 3.00, out: 15.00 },
    'gpt4o': { name: 'GPT-4o', in: 2.50, out: 10.00 },
    'o1': { name: 'OpenAI o1', in: 15.00, out: 60.00 },
  };

  const baselineRates = pricingTable[baselineModel];

  // Cascadepilot blended average:
  // 10% Tier 0 Cache ($0.00)
  // 70% Tier 1 SLM (in: $0.075, out: $0.30)
  // 15% Tier 2 Mid-tier (in: $0.15, out: $0.60)
  // 5% Tier 3 Frontier (in: baselineRates.in, out: baselineRates.out)
  const monthlyPromptTokensM = (monthlyQueries * promptTokens) / 1000000;
  const monthlyCompletionTokensM = (monthlyQueries * completionTokens) / 1000000;

  // Monthly Baseline Cost
  const monthlyBaselineCost =
    (monthlyPromptTokensM * baselineRates.in) +
    (monthlyCompletionTokensM * baselineRates.out);

  // Monthly Cascadepilot Cost
  const t1PromptM = monthlyPromptTokensM * 0.70;
  const t1CompM = monthlyCompletionTokensM * 0.70;
  const t1Cost = (t1PromptM * 0.075) + (t1CompM * 0.30);

  const t2PromptM = monthlyPromptTokensM * 0.15;
  const t2CompM = monthlyCompletionTokensM * 0.15;
  const t2Cost = (t2PromptM * 0.15) + (t2CompM * 0.60);

  const t3PromptM = monthlyPromptTokensM * 0.05;
  const t3CompM = monthlyCompletionTokensM * 0.05;
  const t3Cost = (t3PromptM * baselineRates.in) + (t3CompM * baselineRates.out);

  const monthlyCascadeCost = t1Cost + t2Cost + t3Cost;
  const monthlySaved = monthlyBaselineCost - monthlyCascadeCost;
  const annualSaved = monthlySaved * 12;
  const savingsPct = monthlyBaselineCost > 0 ? (monthlySaved / monthlyBaselineCost) * 100 : 0;

  const handleCopySummary = () => {
    const text = `Cascadepilot Enterprise FinOps Projections:\n` +
      `- Workload: ${(monthlyQueries / 1000000).toFixed(1)}M queries/mo (${promptTokens} in / ${completionTokens} out tokens)\n` +
      `- Baseline Model: ${baselineRates.name}\n` +
      `- Baseline Annual Spend: $${(monthlyBaselineCost * 12).toLocaleString('en-US', { maximumFractionDigits: 0 })}\n` +
      `- Cascadepilot Annual Spend: $${(monthlyCascadeCost * 12).toLocaleString('en-US', { maximumFractionDigits: 0 })}\n` +
      `- Projected Annual Savings: $${annualSaved.toLocaleString('en-US', { maximumFractionDigits: 0 })} (${savingsPct.toFixed(1)}% reduction)\n` +
      `- Quality Parity Retained: 98.8%`;
    navigator.clipboard.writeText(text);
    setCopied(true);
    setTimeout(() => setCopied(false), 2000);
  };

  return (
    <div className="bg-[#0f172a] border border-slate-800 rounded-xl p-6">
      <div className="flex flex-wrap items-center justify-between gap-4 mb-6 pb-4 border-b border-slate-800">
        <div className="flex items-center gap-2">
          <div className="p-2 rounded-lg bg-emerald-500/10 text-emerald-400">
            <Calculator className="w-5 h-5" />
          </div>
          <div>
            <h3 className="text-base font-semibold text-white">Executive FinOps ROI Calculator</h3>
            <p className="text-xs text-slate-400">Forecast enterprise cloud expenditures and annual dollar savings</p>
          </div>
        </div>

        <button
          onClick={handleCopySummary}
          className="flex items-center gap-1.5 text-xs text-slate-200 px-3 py-1.5 rounded-lg bg-slate-900 border border-slate-800 hover:bg-slate-800 hover:text-white transition"
        >
          {copied ? <Check className="w-3.5 h-3.5 text-emerald-400" /> : <Copy className="w-3.5 h-3.5" />}
          {copied ? 'Summary Copied' : 'Copy Executive Brief'}
        </button>
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-12 gap-8">
        {/* Sliders Input Column */}
        <div className="lg:col-span-7 space-y-5">
          {/* Baseline Model Selector */}
          <div>
            <label className="text-xs font-medium text-slate-300 block mb-2">
              Current Baseline Frontier Model
            </label>
            <div className="grid grid-cols-1 sm:grid-cols-3 gap-2">
              {(Object.keys(pricingTable) as Array<keyof typeof pricingTable>).map((key) => (
                <button
                  key={key}
                  onClick={() => setBaselineModel(key)}
                  className={`text-xs p-2.5 rounded-lg border text-left transition ${
                    baselineModel === key
                      ? 'bg-blue-600/15 border-blue-500 text-white font-medium'
                      : 'bg-slate-900/60 border-slate-800 text-slate-400 hover:text-slate-200'
                  }`}
                >
                  <div className="font-semibold text-white">{pricingTable[key].name}</div>
                  <div className="text-[10px] text-slate-400 mt-0.5">
                    ${pricingTable[key].in.toFixed(2)} / ${pricingTable[key].out.toFixed(2)}
                  </div>
                </button>
              ))}
            </div>
          </div>

          {/* Monthly Query Volume */}
          <div>
            <div className="flex justify-between items-center text-xs mb-1.5">
              <span className="text-slate-300">Monthly Query Volume</span>
              <span className="font-mono font-bold text-blue-400">
                {(monthlyQueries / 1000000).toFixed(1)} Million requests/mo
              </span>
            </div>
            <input
              type="range"
              min={100000}
              max={25000000}
              step={100000}
              value={monthlyQueries}
              onChange={(e) => setMonthlyQueries(parseInt(e.target.value))}
              className="w-full h-1.5 bg-slate-800 rounded-lg appearance-none cursor-pointer accent-blue-500"
            />
          </div>

          {/* Prompt Tokens */}
          <div>
            <div className="flex justify-between items-center text-xs mb-1.5">
              <span className="text-slate-300">Average Prompt Tokens</span>
              <span className="font-mono font-bold text-slate-200">{promptTokens} tokens</span>
            </div>
            <input
              type="range"
              min={100}
              max={3000}
              step={50}
              value={promptTokens}
              onChange={(e) => setPromptTokens(parseInt(e.target.value))}
              className="w-full h-1.5 bg-slate-800 rounded-lg appearance-none cursor-pointer accent-blue-500"
            />
          </div>

          {/* Completion Tokens */}
          <div>
            <div className="flex justify-between items-center text-xs mb-1.5">
              <span className="text-slate-300">Average Output Tokens</span>
              <span className="font-mono font-bold text-slate-200">{completionTokens} tokens</span>
            </div>
            <input
              type="range"
              min={50}
              max={1500}
              step={25}
              value={completionTokens}
              onChange={(e) => setCompletionTokens(parseInt(e.target.value))}
              className="w-full h-1.5 bg-slate-800 rounded-lg appearance-none cursor-pointer accent-blue-500"
            />
          </div>
        </div>

        {/* Executive Output Column */}
        <div className="lg:col-span-5 bg-gradient-to-b from-slate-900 to-[#0a1120] border border-slate-800/90 rounded-xl p-5 flex flex-col justify-between">
          <div>
            <span className="text-[11px] font-semibold uppercase tracking-wider text-emerald-400 flex items-center gap-1.5 mb-2">
              <Sparkles className="w-3.5 h-3.5" /> Projected Annual Net Savings
            </span>
            <div className="text-4xl font-extrabold font-mono tracking-tight text-white mb-1">
              ${annualSaved.toLocaleString('en-US', { maximumFractionDigits: 0 })}
            </div>
            <div className="inline-flex items-center gap-1 text-xs font-semibold px-2 py-0.5 rounded-full bg-emerald-500/15 text-emerald-400 border border-emerald-500/30 mb-4">
              <TrendingDown className="w-3.5 h-3.5" />
              {savingsPct.toFixed(1)}% Cost Reduction
            </div>

            <div className="space-y-2 text-xs border-t border-slate-800/80 pt-4">
              <div className="flex justify-between text-slate-400">
                <span>Baseline Annual Cost:</span>
                <strong className="font-mono text-slate-300">
                  ${(monthlyBaselineCost * 12).toLocaleString('en-US', { maximumFractionDigits: 0 })}/yr
                </strong>
              </div>
              <div className="flex justify-between text-slate-400">
                <span>cascadepilot Annual Cost:</span>
                <strong className="font-mono text-emerald-400">
                  ${(monthlyCascadeCost * 12).toLocaleString('en-US', { maximumFractionDigits: 0 })}/yr
                </strong>
              </div>
              <div className="flex justify-between text-slate-400">
                <span>Monthly Net Savings:</span>
                <strong className="font-mono text-white">
                  ${monthlySaved.toLocaleString('en-US', { maximumFractionDigits: 0 })}/mo
                </strong>
              </div>
            </div>
          </div>

          <div className="pt-4 mt-4 border-t border-slate-800 text-[11px] text-slate-400 flex items-center gap-2">
            <DollarSign className="w-4 h-4 text-emerald-400 flex-shrink-0" />
            <span>Retains 98.8%+ benchmark quality parity via in-flight speculative verification.</span>
          </div>
        </div>
      </div>
    </div>
  );
};
