import React, { useState, useEffect, useCallback } from 'react';
import { Play, RefreshCw, Zap, LayoutDashboard, Users, Calculator, Database } from 'lucide-react';
import { SavingsHero } from '../components/SavingsHero';
import { RoutingWaterfall } from '../components/RoutingWaterfall';
import { LatencyDiffChart } from '../components/LatencyDiffChart';
import { ThresholdSlider } from '../components/ThresholdSlider';
import { RequestLiveLog } from '../components/RequestLiveLog';
import { ProxyPlayground } from '../components/ProxyPlayground';
import { TenantsView } from '../components/TenantsView';
import { ROICalculator } from '../components/ROICalculator';
import { FinOpsSummary, WaterfallMetrics, LatencyMetrics, RequestRecord, Thresholds } from '../types';

export const Dashboard: React.FC = () => {
  const [activeTab, setActiveTab] = useState<'overview' | 'tenants' | 'roi'>('overview');
  const [summary, setSummary] = useState<FinOpsSummary | null>(null);
  const [waterfall, setWaterfall] = useState<WaterfallMetrics | null>(null);
  const [latency, setLatency] = useState<LatencyMetrics | null>(null);
  const [requests, setRequests] = useState<RequestRecord[]>([]);
  const [thresholds, setThresholds] = useState<Thresholds | null>(null);
  const [isConnected, setIsConnected] = useState(false);
  const [simulating, setSimulating] = useState(false);

  const fetchAllData = useCallback(async () => {
    try {
      const [sumRes, watRes, latRes, reqRes, thrRes] = await Promise.all([
        fetch('/api/summary').then((r) => r.json()).catch(() => null),
        fetch('/api/waterfall').then((r) => r.json()).catch(() => null),
        fetch('/api/latency').then((r) => r.json()).catch(() => null),
        fetch('/api/requests?limit=50').then((r) => r.json()).catch(() => []),
        fetch('/api/thresholds').then((r) => r.json()).catch(() => null),
      ]);

      if (sumRes) setSummary(sumRes);
      if (watRes) setWaterfall(watRes);
      if (latRes) setLatency(latRes);
      if (reqRes && Array.isArray(reqRes)) setRequests(reqRes);
      if (thrRes) setThresholds(thrRes);
    } catch (e) {
      console.error('Error fetching dashboard telemetry:', e);
    }
  }, []);

  useEffect(() => {
    fetchAllData();

    // Setup Server-Sent Events stream
    const eventSource = new EventSource('/api/events');

    eventSource.onopen = () => {
      setIsConnected(true);
    };

    eventSource.onerror = () => {
      setIsConnected(false);
    };

    eventSource.addEventListener('new_request', (event: MessageEvent) => {
      try {
        const newReq = JSON.parse(event.data) as RequestRecord;
        setRequests((prev) => [newReq, ...prev.slice(0, 49)]);
        fetchAllData();
      } catch (err) {
        console.error('Error handling SSE new_request:', err);
      }
    });

    return () => {
      eventSource.close();
    };
  }, [fetchAllData]);

  const handleUpdateThresholds = async (newThresholds: Thresholds) => {
    await fetch('/api/thresholds', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(newThresholds),
    });
    setThresholds(newThresholds);
  };

  const handleSimulate = async () => {
    setSimulating(true);
    try {
      await fetch('/api/simulate', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ count: 12 }),
      });
      await fetchAllData();
    } finally {
      setSimulating(false);
    }
  };

  const handleExportCSV = () => {
    window.location.href = '/api/export/csv';
  };

  const handleExportJSON = () => {
    window.location.href = '/api/export/json';
  };

  return (
    <div className="min-h-screen bg-[#090d16] text-slate-100 flex flex-col font-sans">
      {/* Top Navbar */}
      <header className="border-b border-slate-800 bg-[#0f172a]/90 backdrop-blur sticky top-0 z-40">
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 h-16 flex items-center justify-between">
          <div className="flex items-center gap-3">
            <div className="w-9 h-9 rounded-xl bg-gradient-to-tr from-blue-600 to-indigo-500 flex items-center justify-center text-white shadow-lg shadow-blue-500/20">
              <Zap className="w-5 h-5 fill-white" />
            </div>
            <div>
              <div className="flex items-center gap-2">
                <h1 className="text-base font-bold tracking-tight text-white">cascadepilot</h1>
                <span className="text-[10px] uppercase font-mono px-1.5 py-0.5 rounded bg-blue-500/20 text-blue-400 font-semibold">
                  FinOps v1.2
                </span>
              </div>
              <p className="text-[11px] text-slate-400">Speculative LLM Cascading Router & FinOps Gateway</p>
            </div>
          </div>

          {/* Model tier pill indicators */}
          <div className="hidden xl:flex items-center gap-2 text-xs">
            <span className="flex items-center gap-1.5 px-2 py-0.5 rounded-lg bg-cyan-500/10 border border-cyan-500/20 text-cyan-400 font-medium">
              <Database className="w-3 h-3" />
              T0: Cache ($0)
            </span>
            <span className="flex items-center gap-1.5 px-2 py-0.5 rounded-lg bg-emerald-500/10 border border-emerald-500/20 text-emerald-400 font-medium">
              <span className="w-1.5 h-1.5 rounded-full bg-emerald-400" />
              T1: Flash ($0.075)
            </span>
            <span className="flex items-center gap-1.5 px-2 py-0.5 rounded-lg bg-blue-500/10 border border-blue-500/20 text-blue-400 font-medium">
              <span className="w-1.5 h-1.5 rounded-full bg-blue-400" />
              T2: 4o-mini ($0.15)
            </span>
            <span className="flex items-center gap-1.5 px-2 py-0.5 rounded-lg bg-purple-500/10 border border-purple-500/20 text-purple-400 font-medium">
              <span className="w-1.5 h-1.5 rounded-full bg-purple-400" />
              T3: Sonnet ($3.00)
            </span>
          </div>

          {/* Actions */}
          <div className="flex items-center gap-2">
            <button
              onClick={handleSimulate}
              disabled={simulating}
              className="flex items-center gap-1.5 text-xs font-semibold px-3 py-1.5 rounded-lg bg-gradient-to-r from-blue-600 to-indigo-600 text-white hover:opacity-90 active:scale-95 transition shadow-sm disabled:opacity-50"
            >
              {simulating ? (
                <RefreshCw className="w-3.5 h-3.5 animate-spin" />
              ) : (
                <Play className="w-3.5 h-3.5 fill-white" />
              )}
              {simulating ? 'Simulating...' : 'Simulate 12 Queries'}
            </button>
            <button
              onClick={() => fetchAllData()}
              title="Refresh Telemetry"
              className="p-1.5 rounded-lg bg-slate-800 text-slate-300 hover:text-white hover:bg-slate-700 transition"
            >
              <RefreshCw className="w-4 h-4" />
            </button>
          </div>
        </div>

        {/* Navigation Tabs */}
        <div className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 flex items-center gap-1 border-t border-slate-800/60 overflow-x-auto">
          <button
            onClick={() => setActiveTab('overview')}
            className={`flex items-center gap-2 text-xs font-medium py-2.5 px-3 border-b-2 transition whitespace-nowrap ${
              activeTab === 'overview'
                ? 'border-blue-500 text-white bg-blue-500/5'
                : 'border-transparent text-slate-400 hover:text-slate-200'
            }`}
          >
            <LayoutDashboard className="w-3.5 h-3.5" />
            Cockpit Overview
          </button>
          <button
            onClick={() => setActiveTab('tenants')}
            className={`flex items-center gap-2 text-xs font-medium py-2.5 px-3 border-b-2 transition whitespace-nowrap ${
              activeTab === 'tenants'
                ? 'border-blue-500 text-white bg-blue-500/5'
                : 'border-transparent text-slate-400 hover:text-slate-200'
            }`}
          >
            <Users className="w-3.5 h-3.5" />
            Department Chargeback & Virtual Keys
          </button>
          <button
            onClick={() => setActiveTab('roi')}
            className={`flex items-center gap-2 text-xs font-medium py-2.5 px-3 border-b-2 transition whitespace-nowrap ${
              activeTab === 'roi'
                ? 'border-blue-500 text-white bg-blue-500/5'
                : 'border-transparent text-slate-400 hover:text-slate-200'
            }`}
          >
            <Calculator className="w-3.5 h-3.5" />
            Executive ROI Calculator
          </button>
        </div>
      </header>

      {/* Main Content Area */}
      <main className="max-w-7xl mx-auto px-4 sm:px-6 lg:px-8 py-6 flex-1 w-full space-y-6">
        {activeTab === 'overview' && (
          <>
            {/* KPI Hero Section */}
            <SavingsHero summary={summary} />

            {/* Charts Grid */}
            <div className="grid grid-cols-1 lg:grid-cols-2 gap-6">
              <RoutingWaterfall waterfall={waterfall} summary={summary} />
              <LatencyDiffChart latency={latency} />
            </div>

            {/* Hyperparameter Controls */}
            <ThresholdSlider initialThresholds={thresholds} onUpdate={handleUpdateThresholds} />

            {/* Interactive Query Tester */}
            <ProxyPlayground onQueryComplete={fetchAllData} />

            {/* Live Transaction Ledger */}
            <RequestLiveLog
              requests={requests}
              onExportCSV={handleExportCSV}
              onExportJSON={handleExportJSON}
              isConnected={isConnected}
            />
          </>
        )}

        {activeTab === 'tenants' && (
          <TenantsView />
        )}

        {activeTab === 'roi' && (
          <ROICalculator />
        )}
      </main>

      {/* Footer */}
      <footer className="border-t border-slate-800/80 bg-[#090d16] py-4 text-center text-xs text-slate-400">
        <div className="max-w-7xl mx-auto px-4 flex flex-col sm:flex-row items-center justify-between gap-2">
          <span>
            cascadepilot Reverse Proxy &bull; Speculative 3-Tier Cascade &bull; Memory RSS &lt; 20MB &bull; Routing Overhead &lt; 0.1ms
          </span>
          <div className="flex items-center gap-4">
            <a href="/v1/models" target="_blank" rel="noreferrer" className="hover:text-slate-200 transition">
              /v1/models
            </a>
            <a href="/health" target="_blank" rel="noreferrer" className="hover:text-slate-200 transition">
              /health
            </a>
            <a href="/metrics" target="_blank" rel="noreferrer" className="hover:text-slate-200 transition">
              /metrics
            </a>
          </div>
        </div>
      </footer>
    </div>
  );
};
