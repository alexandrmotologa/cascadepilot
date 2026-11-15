import React, { useState, useEffect, useCallback } from 'react';
import { Users, Plus, Key, Copy, Check, ShieldAlert, Sparkles } from 'lucide-react';
import { TenantSummary } from '../types';

export const TenantsView: React.FC = () => {
  const [tenants, setTenants] = useState<TenantSummary[]>([]);
  const [loading, setLoading] = useState(true);
  const [copiedKey, setCopiedKey] = useState<string | null>(null);
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [name, setName] = useState('');
  const [department, setDepartment] = useState('Engineering');
  const [budget, setBudget] = useState(500);
  const [creating, setCreating] = useState(false);

  const fetchTenants = useCallback(async () => {
    try {
      const res = await fetch('/api/tenants');
      const data = await res.json();
      if (Array.isArray(data)) {
        setTenants(data);
      }
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    fetchTenants();
  }, [fetchTenants]);

  const handleCopy = (key: string) => {
    navigator.clipboard.writeText(key);
    setCopiedKey(key);
    setTimeout(() => setCopiedKey(null), 2000);
  };

  const handleCreateTenant = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!name.trim()) return;

    setCreating(true);
    try {
      await fetch('/api/tenants', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          name: name.trim(),
          department: department.trim(),
          monthly_budget_usd: budget,
        }),
      });
      setName('');
      setIsModalOpen(false);
      await fetchTenants();
    } finally {
      setCreating(false);
    }
  };

  return (
    <div className="bg-[#0f172a] border border-slate-800 rounded-xl overflow-hidden">
      {/* Header */}
      <div className="p-5 border-b border-slate-800 flex flex-wrap items-center justify-between gap-4">
        <div className="flex items-center gap-2">
          <div className="p-2 rounded-lg bg-purple-500/10 text-purple-400">
            <Users className="w-4 h-4" />
          </div>
          <div>
            <h3 className="text-sm font-semibold text-white">Department Chargeback & Virtual Keys</h3>
            <p className="text-xs text-slate-400">Cost allocation and budget guardrails by business unit</p>
          </div>
        </div>

        <button
          onClick={() => setIsModalOpen(true)}
          className="flex items-center gap-1.5 text-xs font-semibold px-3 py-1.5 rounded-lg bg-blue-600 hover:bg-blue-500 text-white transition active:scale-95"
        >
          <Plus className="w-3.5 h-3.5" />
          Provision Virtual Key
        </button>
      </div>

      {/* Tenants Table */}
      <div className="overflow-x-auto">
        <table className="w-full text-left text-xs border-collapse">
          <thead className="bg-slate-900/80 border-b border-slate-800 text-slate-400 font-medium">
            <tr>
              <th className="py-2.5 px-4">Department / Team</th>
              <th className="py-2.5 px-4">Virtual API Key</th>
              <th className="py-2.5 px-4">Budget Utilization</th>
              <th className="py-2.5 px-4">Total Requests</th>
              <th className="py-2.5 px-4">Realized Spend</th>
              <th className="py-2.5 px-4 text-right">Net Saved</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-slate-800/60 font-sans">
            {loading ? (
              <tr>
                <td colSpan={6} className="py-6 text-center text-slate-400">
                  Loading department chargeback metrics...
                </td>
              </tr>
            ) : tenants.length === 0 ? (
              <tr>
                <td colSpan={6} className="py-6 text-center text-slate-400">
                  No tenants provisioned yet.
                </td>
              </tr>
            ) : (
              tenants.map((t) => {
                const util = t.budget_utilization_pct || 0;
                const isOverBudget = util >= 100;
                const isWarning = util >= 75 && util < 100;

                return (
                  <tr key={t.id} className="hover:bg-slate-800/40 transition-colors">
                    <td className="py-3 px-4">
                      <div className="font-semibold text-white">{t.name}</div>
                      <div className="text-[11px] text-slate-400">{t.department}</div>
                    </td>
                    <td className="py-3 px-4 font-mono text-[11px]">
                      <div className="flex items-center gap-1.5">
                        <Key className="w-3 h-3 text-slate-400" />
                        <span className="text-slate-300">{t.api_key}</span>
                        <button
                          onClick={() => handleCopy(t.api_key)}
                          className="p-1 hover:bg-slate-800 rounded text-slate-400 hover:text-white transition"
                          title="Copy Key"
                        >
                          {copiedKey === t.api_key ? (
                            <Check className="w-3 h-3 text-emerald-400" />
                          ) : (
                            <Copy className="w-3 h-3" />
                          )}
                        </button>
                      </div>
                    </td>
                    <td className="py-3 px-4 min-w-[170px]">
                      <div className="flex justify-between text-[11px] mb-1 font-mono">
                        <span className={isOverBudget ? 'text-red-400 font-bold flex items-center gap-0.5' : isWarning ? 'text-amber-400' : 'text-slate-300'}>
                          {isOverBudget && <ShieldAlert className="w-3 h-3 inline" />}
                          {util.toFixed(1)}%
                        </span>
                        <span className="text-slate-400">${t.monthly_budget_usd.toFixed(0)} cap</span>
                      </div>
                      <div className="h-1.5 w-full bg-slate-900 rounded-full overflow-hidden">
                        <div
                          style={{ width: `${Math.min(100, util)}%` }}
                          className={`h-full rounded-full transition-all duration-500 ${
                            isOverBudget ? 'bg-red-500' : isWarning ? 'bg-amber-500' : 'bg-blue-500'
                          }`}
                        />
                      </div>
                    </td>
                    <td className="py-3 px-4 font-mono text-slate-300">
                      {t.total_requests.toLocaleString()}
                    </td>
                    <td className="py-3 px-4 font-mono text-slate-200">
                      ${t.total_actual_cost_usd.toFixed(4)}
                    </td>
                    <td className="py-3 px-4 font-mono font-semibold text-emerald-400 text-right">
                      +${t.total_saved_cost_usd.toFixed(4)}
                    </td>
                  </tr>
                );
              })
            )}
          </tbody>
        </table>
      </div>

      {/* Provision Key Modal */}
      {isModalOpen && (
        <div className="fixed inset-0 z-50 flex items-center justify-center p-4 bg-black/60 backdrop-blur-sm animate-fadeIn">
          <div className="bg-[#0f172a] border border-slate-700 rounded-xl max-w-md w-full p-6 shadow-2xl relative">
            <h4 className="text-sm font-semibold text-white mb-1 flex items-center gap-2">
              <Sparkles className="w-4 h-4 text-blue-400" />
              Provision New Department Key
            </h4>
            <p className="text-xs text-slate-400 mb-4">
              Generates a virtual gateway token with isolated cost allocation.
            </p>

            <form onSubmit={handleCreateTenant} className="space-y-4">
              <div>
                <label className="text-xs font-medium text-slate-300 block mb-1">
                  Team / Service Name
                </label>
                <input
                  type="text"
                  required
                  placeholder="e.g. Mobile Checkout Service"
                  value={name}
                  onChange={(e) => setName(e.target.value)}
                  className="w-full text-xs bg-slate-900 border border-slate-800 rounded-lg p-2.5 text-white placeholder-slate-500 focus:outline-none focus:border-blue-500"
                />
              </div>

              <div>
                <label className="text-xs font-medium text-slate-300 block mb-1">
                  Department
                </label>
                <select
                  value={department}
                  onChange={(e) => setDepartment(e.target.value)}
                  className="w-full text-xs bg-slate-900 border border-slate-800 rounded-lg p-2.5 text-white focus:outline-none focus:border-blue-500"
                >
                  <option value="Engineering">Engineering</option>
                  <option value="Customer Support">Customer Support</option>
                  <option value="Marketing">Marketing</option>
                  <option value="Data Science">Data Science</option>
                  <option value="Operations">Operations</option>
                </select>
              </div>

              <div>
                <label className="text-xs font-medium text-slate-300 block mb-1">
                  Monthly Budget Cap (USD)
                </label>
                <input
                  type="number"
                  min="50"
                  max="50000"
                  step="50"
                  value={budget}
                  onChange={(e) => setBudget(parseFloat(e.target.value))}
                  className="w-full text-xs font-mono bg-slate-900 border border-slate-800 rounded-lg p-2.5 text-white focus:outline-none focus:border-blue-500"
                />
              </div>

              <div className="flex justify-end gap-2 pt-2">
                <button
                  type="button"
                  onClick={() => setIsModalOpen(false)}
                  className="text-xs px-3 py-1.5 rounded-lg bg-slate-800 text-slate-300 hover:text-white"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  disabled={creating}
                  className="text-xs font-semibold px-4 py-1.5 rounded-lg bg-blue-600 hover:bg-blue-500 text-white disabled:opacity-50"
                >
                  {creating ? 'Generating...' : 'Create Key'}
                </button>
              </div>
            </form>
          </div>
        </div>
      )}
    </div>
  );
};
