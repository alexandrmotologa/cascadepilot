import React, { useState, useEffect } from 'react';
import { Sliders, RefreshCw, Check, Info } from 'lucide-react';
import { Thresholds } from '../types';

interface ThresholdSliderProps {
  initialThresholds: Thresholds | null;
  onUpdate: (thresholds: Thresholds) => Promise<void>;
}

export const ThresholdSlider: React.FC<ThresholdSliderProps> = ({ initialThresholds, onUpdate }) => {
  const [theta, setTheta] = useState(0.75);
  const [tau, setTau] = useState(0.70);
  const [saving, setSaving] = useState(false);
  const [savedNotice, setSavedNotice] = useState(false);

  useEffect(() => {
    if (initialThresholds) {
      setTheta(initialThresholds.theta_bypass);
      setTau(initialThresholds.tau_confidence);
    }
  }, [initialThresholds]);

  const handleSave = async () => {
    setSaving(true);
    try {
      await onUpdate({ theta_bypass: theta, tau_confidence: tau });
      setSavedNotice(true);
      setTimeout(() => setSavedNotice(false), 2500);
    } finally {
      setSaving(false);
    }
  };

  const handleReset = () => {
    setTheta(0.75);
    setTau(0.70);
  };

  return (
    <div className="bg-[#0f172a] border border-slate-800 rounded-xl p-5">
      <div className="flex items-center justify-between mb-4">
        <div className="flex items-center gap-2">
          <div className="p-2 rounded-lg bg-indigo-500/10 text-indigo-400">
            <Sliders className="w-4 h-4" />
          </div>
          <div>
            <h3 className="text-sm font-semibold text-white">Dynamic Routing Hyperparameters</h3>
            <p className="text-xs text-slate-400">Live calibration without process restart</p>
          </div>
        </div>
        <div className="flex items-center gap-2">
          <button
            onClick={handleReset}
            className="text-xs text-slate-400 hover:text-white px-2.5 py-1 rounded bg-slate-800/80 hover:bg-slate-700 transition"
          >
            Reset Defaults
          </button>
          <button
            onClick={handleSave}
            disabled={saving}
            className="flex items-center gap-1.5 text-xs font-medium text-white px-3 py-1.5 rounded-lg bg-blue-600 hover:bg-blue-500 active:scale-95 transition disabled:opacity-50"
          >
            {saving ? (
              <RefreshCw className="w-3.5 h-3.5 animate-spin" />
            ) : savedNotice ? (
              <Check className="w-3.5 h-3.5 text-emerald-400" />
            ) : null}
            {savedNotice ? 'Applied Live' : 'Apply Changes'}
          </button>
        </div>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 gap-6">
        {/* Slider 1: Theta Bypass */}
        <div className="p-4 rounded-lg bg-slate-900/60 border border-slate-800/80">
          <div className="flex justify-between items-center mb-2">
            <label className="text-xs font-medium text-slate-200 flex items-center gap-1">
              <span>Bypass Threshold (&theta;<sub>bypass</sub>)</span>
            </label>
            <span className="font-mono text-xs font-bold px-2 py-0.5 rounded bg-blue-500/15 text-blue-400 border border-blue-500/30">
              {theta.toFixed(2)}
            </span>
          </div>
          <input
            type="range"
            min="0.10"
            max="1.00"
            step="0.05"
            value={theta}
            onChange={(e) => setTheta(parseFloat(e.target.value))}
            className="w-full h-1.5 bg-slate-800 rounded-lg appearance-none cursor-pointer accent-blue-500"
          />
          <div className="flex justify-between text-[10px] text-slate-400 mt-1 font-mono">
            <span>Aggressive (0.10)</span>
            <span>Balanced (0.75)</span>
            <span>Permissive (1.00)</span>
          </div>
          <p className="text-[11px] text-slate-400 mt-2 flex items-start gap-1">
            <Info className="w-3.5 h-3.5 text-blue-400 flex-shrink-0 mt-0.5" />
            Queries scoring above {theta.toFixed(2)} bypass Tier 1 directly to Tier 2 to eliminate draft latency.
          </p>
        </div>

        {/* Slider 2: Tau Confidence */}
        <div className="p-4 rounded-lg bg-slate-900/60 border border-slate-800/80">
          <div className="flex justify-between items-center mb-2">
            <label className="text-xs font-medium text-slate-200 flex items-center gap-1">
              <span>Confidence Gate (&tau;<sub>conf</sub>)</span>
            </label>
            <span className="font-mono text-xs font-bold px-2 py-0.5 rounded bg-purple-500/15 text-purple-400 border border-purple-500/30">
              {tau.toFixed(2)}
            </span>
          </div>
          <input
            type="range"
            min="0.10"
            max="1.00"
            step="0.05"
            value={tau}
            onChange={(e) => setTau(parseFloat(e.target.value))}
            className="w-full h-1.5 bg-slate-800 rounded-lg appearance-none cursor-pointer accent-purple-500"
          />
          <div className="flex justify-between text-[10px] text-slate-400 mt-1 font-mono">
            <span>Lenient (0.10)</span>
            <span>Balanced (0.70)</span>
            <span>Strict (1.00)</span>
          </div>
          <p className="text-[11px] text-slate-400 mt-2 flex items-start gap-1">
            <Info className="w-3.5 h-3.5 text-purple-400 flex-shrink-0 mt-0.5" />
            Tier 1 responses with confidence below {tau.toFixed(2)} automatically escalate to Tier 2.
          </p>
        </div>
      </div>
    </div>
  );
};
