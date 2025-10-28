import { useEffect, useState } from 'react';
import useSWR from 'swr';
import { api } from './lib/api';
import { EquityChart } from './components/EquityChart';
import { CompetitionPage } from './components/CompetitionPage';
import type {
  SystemStatus,
  AccountInfo,
  Position,
  DecisionRecord,
  Statistics,
  TraderInfo,
} from './types';

type Page = 'competition' | 'trader';

function App() {
  const [currentPage, setCurrentPage] = useState<Page>('competition');
  const [selectedTraderId, setSelectedTraderId] = useState<string | undefined>();
  const [lastUpdate, setLastUpdate] = useState<string>('--:--:--');

  // 获取trader列表
  const { data: traders } = useSWR<TraderInfo[]>('traders', api.getTraders, {
    refreshInterval: 10000,
  });

  // 当获取到traders后，设置默认选中第一个
  useEffect(() => {
    if (traders && traders.length > 0 && !selectedTraderId) {
      setSelectedTraderId(traders[0].trader_id);
    }
  }, [traders, selectedTraderId]);

  // 如果在trader页面，获取该trader的数据
  const { data: status } = useSWR<SystemStatus>(
    currentPage === 'trader' && selectedTraderId
      ? `status-${selectedTraderId}`
      : null,
    () => api.getStatus(selectedTraderId),
    {
      refreshInterval: 5000,
      revalidateOnFocus: true,
      dedupingInterval: 0,
    }
  );

  const { data: account } = useSWR<AccountInfo>(
    currentPage === 'trader' && selectedTraderId
      ? `account-${selectedTraderId}`
      : null,
    () => api.getAccount(selectedTraderId),
    {
      refreshInterval: 5000,
      revalidateOnFocus: true,
      dedupingInterval: 0,
    }
  );

  const { data: positions } = useSWR<Position[]>(
    currentPage === 'trader' && selectedTraderId
      ? `positions-${selectedTraderId}`
      : null,
    () => api.getPositions(selectedTraderId),
    {
      refreshInterval: 5000,
      revalidateOnFocus: true,
      dedupingInterval: 0,
    }
  );

  const { data: decisions } = useSWR<DecisionRecord[]>(
    currentPage === 'trader' && selectedTraderId
      ? `decisions/latest-${selectedTraderId}`
      : null,
    () => api.getLatestDecisions(selectedTraderId),
    { refreshInterval: 10000 }
  );

  const { data: stats } = useSWR<Statistics>(
    currentPage === 'trader' && selectedTraderId
      ? `statistics-${selectedTraderId}`
      : null,
    () => api.getStatistics(selectedTraderId),
    { refreshInterval: 10000 }
  );

  useEffect(() => {
    if (account) {
      const now = new Date().toLocaleTimeString();
      setLastUpdate(now);
    }
  }, [account]);

  const selectedTrader = traders?.find((t) => t.trader_id === selectedTraderId);

  return (
    <div className="min-h-screen text-gray-100">
      {/* Header */}
      <header className="glass sticky top-0 z-50 backdrop-blur-xl">
        <div className="max-w-7xl mx-auto px-4 py-6">
          <div className="flex items-center justify-between">
            <div>
              <h1 className="text-3xl font-bold text-white">
                🏆 AI Trading Competition
              </h1>
              <p className="text-gray-400 mt-1 mono text-sm">
                Qwen vs DeepSeek - Real-time Performance Tracking
              </p>
            </div>
            <div className="flex items-center gap-4">
              {/* Page Toggle */}
              <div className="flex gap-1 bg-gray-800/50 rounded-lg p-1">
                <button
                  onClick={() => setCurrentPage('competition')}
                  className={`px-4 py-2 rounded text-sm font-medium transition-all ${
                    currentPage === 'competition'
                      ? 'bg-purple-600 text-white'
                      : 'text-gray-400 hover:text-white'
                  }`}
                >
                  🏆 Competition
                </button>
                <button
                  onClick={() => setCurrentPage('trader')}
                  className={`px-4 py-2 rounded text-sm font-medium transition-all ${
                    currentPage === 'trader'
                      ? 'bg-blue-600 text-white'
                      : 'text-gray-400 hover:text-white'
                  }`}
                >
                  📊 Trader Details
                </button>
              </div>

              {/* Trader Selector (only show on trader page) */}
              {currentPage === 'trader' && traders && traders.length > 0 && (
                <select
                  value={selectedTraderId}
                  onChange={(e) => setSelectedTraderId(e.target.value)}
                  className="bg-gray-800 border border-gray-700 rounded-lg px-4 py-2 text-sm font-medium text-white cursor-pointer hover:border-gray-600 transition-colors"
                >
                  {traders.map((trader) => (
                    <option key={trader.trader_id} value={trader.trader_id}>
                      {trader.trader_name} ({trader.ai_model.toUpperCase()})
                    </option>
                  ))}
                </select>
              )}

              {/* Status Indicator (only show on trader page) */}
              {currentPage === 'trader' && status && (
                <div
                  className={`flex items-center gap-2 px-4 py-2 rounded-lg ${
                    status.is_running
                      ? 'bg-green-900/30 text-green-400 border border-green-900/50'
                      : 'bg-red-900/30 text-red-400 border border-red-900/50'
                  }`}
                >
                  <div
                    className={`w-2 h-2 rounded-full ${
                      status.is_running ? 'bg-green-400 pulse-glow' : 'bg-red-400'
                    }`}
                  />
                  <span className="font-semibold mono text-sm">
                    {status.is_running ? 'RUNNING' : 'STOPPED'}
                  </span>
                </div>
              )}
            </div>
          </div>
        </div>
      </header>

      {/* Main Content */}
      <main className="max-w-7xl mx-auto px-4 py-8">
        {currentPage === 'competition' ? (
          <CompetitionPage />
        ) : (
          <TraderDetailsPage
            selectedTrader={selectedTrader}
            status={status}
            account={account}
            positions={positions}
            decisions={decisions}
            stats={stats}
            lastUpdate={lastUpdate}
          />
        )}
      </main>

      {/* Footer */}
      <footer className="mt-16 border-t border-gray-800 bg-gray-900">
        <div className="max-w-7xl mx-auto px-4 py-6 text-center text-gray-500 text-sm">
          <p>NOFX - AI Trading Competition System</p>
          <p className="mt-1">⚠️ Trading involves risk. Use at your own discretion.</p>
        </div>
      </footer>
    </div>
  );
}

// Trader Details Page Component
function TraderDetailsPage({
  selectedTrader,
  status,
  account,
  positions,
  decisions,
  stats,
  lastUpdate,
}: {
  selectedTrader?: TraderInfo;
  status?: SystemStatus;
  account?: AccountInfo;
  positions?: Position[];
  decisions?: DecisionRecord[];
  stats?: Statistics;
  lastUpdate: string;
}) {
  if (!selectedTrader) {
    return <div className="text-center py-12 text-gray-500">Loading...</div>;
  }

  return (
    <div>
      {/* Trader Header */}
      <div className="mb-8 bg-gradient-to-r from-purple-900/30 to-blue-900/30 rounded-lg p-6 border border-purple-800/50">
        <h2 className="text-2xl font-bold mb-2">{selectedTrader.trader_name}</h2>
        <div className="flex items-center gap-4 text-sm text-gray-400">
          <span>AI Model: <span className={`font-semibold ${selectedTrader.ai_model === 'qwen' ? 'text-purple-400' : 'text-blue-400'}`}>{selectedTrader.ai_model.toUpperCase()}</span></span>
          {status && (
            <>
              <span>•</span>
              <span>Cycles: {status.call_count}</span>
              <span>•</span>
              <span>Runtime: {status.runtime_minutes} min</span>
            </>
          )}
        </div>
      </div>

      {/* Debug Info */}
      {account && (
        <div className="mb-4 p-3 bg-gray-900/30 border border-gray-800 rounded text-xs font-mono">
          <div className="text-gray-400">
            🔄 Last Update: {lastUpdate} | Total Equity: {account.total_equity.toFixed(2)} |
            Available: {account.available_balance.toFixed(2)} | P&L: {account.total_pnl.toFixed(2)}{' '}
            ({account.total_pnl_pct.toFixed(2)}%)
          </div>
        </div>
      )}

      {/* Account Overview */}
      <div className="grid grid-cols-1 md:grid-cols-4 gap-4 mb-8 animate-fade-in">
        <StatCard
          title="Total Equity"
          value={`${account?.total_equity.toFixed(2) || '0.00'} USDT`}
          change={account?.total_pnl_pct || 0}
          positive={account ? account.total_pnl > 0 : false}
        />
        <StatCard
          title="Available Balance"
          value={`${account?.available_balance.toFixed(2) || '0.00'} USDT`}
          subtitle={`${((account?.available_balance / account?.total_equity) * 100 || 0).toFixed(1)}% Free`}
        />
        <StatCard
          title="Total P&L"
          value={`${account?.total_pnl >= 0 ? '+' : ''}${account?.total_pnl.toFixed(2) || '0.00'} USDT`}
          change={account?.total_pnl_pct || 0}
          positive={account ? account.total_pnl >= 0 : false}
        />
        <StatCard
          title="Positions"
          value={`${account?.position_count || 0}`}
          subtitle={`Margin: ${account?.margin_used_pct.toFixed(1) || '0.0'}%`}
        />
      </div>

      {/* Equity Chart */}
      <div className="mb-8 animate-fade-in">
        <EquityChart traderId={selectedTrader.trader_id} />
      </div>

      {/* Statistics */}
      {stats && (
        <div className="bg-gray-900 rounded-lg p-6 mb-8 border border-gray-800">
          <h2 className="text-xl font-bold mb-4">Statistics</h2>
          <div className="grid grid-cols-2 md:grid-cols-5 gap-4">
            <div>
              <div className="text-sm text-gray-400">Total Cycles</div>
              <div className="text-2xl font-bold">{stats.total_cycles}</div>
            </div>
            <div>
              <div className="text-sm text-gray-400">Successful</div>
              <div className="text-2xl font-bold text-green-400">
                {stats.successful_cycles}
              </div>
            </div>
            <div>
              <div className="text-sm text-gray-400">Failed</div>
              <div className="text-2xl font-bold text-red-400">{stats.failed_cycles}</div>
            </div>
            <div>
              <div className="text-sm text-gray-400">Open Positions</div>
              <div className="text-2xl font-bold">{stats.total_open_positions}</div>
            </div>
            <div>
              <div className="text-sm text-gray-400">Close Positions</div>
              <div className="text-2xl font-bold">{stats.total_close_positions}</div>
            </div>
          </div>
        </div>
      )}

      {/* Positions */}
      <div className="bg-gray-900 rounded-lg p-6 mb-8 border border-gray-800">
        <h2 className="text-xl font-bold mb-4">Current Positions</h2>
        {positions && positions.length > 0 ? (
          <div className="overflow-x-auto">
            <table className="w-full text-sm">
              <thead className="text-left border-b border-gray-800">
                <tr>
                  <th className="pb-3 font-semibold text-gray-400">Symbol</th>
                  <th className="pb-3 font-semibold text-gray-400">Side</th>
                  <th className="pb-3 font-semibold text-gray-400">Entry Price</th>
                  <th className="pb-3 font-semibold text-gray-400">Mark Price</th>
                  <th className="pb-3 font-semibold text-gray-400">Quantity</th>
                  <th className="pb-3 font-semibold text-gray-400">Position Value</th>
                  <th className="pb-3 font-semibold text-gray-400">Leverage</th>
                  <th className="pb-3 font-semibold text-gray-400">Unrealized P&L</th>
                  <th className="pb-3 font-semibold text-gray-400">Liq. Price</th>
                </tr>
              </thead>
              <tbody>
                {positions.map((pos, i) => (
                  <tr key={i} className="border-b border-gray-800 last:border-0">
                    <td className="py-3 font-mono font-semibold">{pos.symbol}</td>
                    <td className="py-3">
                      <span
                        className={`px-2 py-1 rounded text-xs font-semibold ${
                          pos.side === 'long'
                            ? 'bg-green-900/30 text-green-400'
                            : 'bg-red-900/30 text-red-400'
                        }`}
                      >
                        {pos.side.toUpperCase()}
                      </span>
                    </td>
                    <td className="py-3 font-mono">{pos.entry_price.toFixed(4)}</td>
                    <td className="py-3 font-mono">{pos.mark_price.toFixed(4)}</td>
                    <td className="py-3 font-mono">{pos.quantity.toFixed(4)}</td>
                    <td className="py-3 font-mono font-semibold text-white">
                      {(pos.quantity * pos.mark_price).toFixed(2)} USDT
                    </td>
                    <td className="py-3 font-mono">{pos.leverage}x</td>
                    <td className="py-3 font-mono">
                      <span
                        className={pos.unrealized_pnl >= 0 ? 'text-green-400' : 'text-red-400'}
                      >
                        {pos.unrealized_pnl >= 0 ? '+' : ''}
                        {pos.unrealized_pnl.toFixed(2)} ({pos.unrealized_pnl_pct.toFixed(2)}%)
                      </span>
                    </td>
                    <td className="py-3 font-mono text-gray-400">
                      {pos.liquidation_price.toFixed(4)}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        ) : (
          <div className="text-center py-12 text-gray-500">No active positions</div>
        )}
      </div>

      {/* Recent Decisions */}
      <div className="bg-gray-900 rounded-lg p-6 border border-gray-800">
        <h2 className="text-xl font-bold mb-4">Recent Decisions</h2>
        {decisions && decisions.length > 0 ? (
          <div className="space-y-4">
            {decisions.map((decision, i) => (
              <DecisionCard key={i} decision={decision} />
            ))}
          </div>
        ) : (
          <div className="text-center py-12 text-gray-500">No decisions yet</div>
        )}
      </div>
    </div>
  );
}

// Stat Card Component
function StatCard({
  title,
  value,
  change,
  positive,
  subtitle,
}: {
  title: string;
  value: string;
  change?: number;
  positive?: boolean;
  subtitle?: string;
}) {
  return (
    <div className="bg-gray-900/50 rounded-lg p-6 border border-gray-800 hover:border-gray-700 transition-all">
      <div className="text-xs text-gray-400 mb-2 mono uppercase tracking-wider">{title}</div>
      <div className="text-2xl font-bold mb-1 mono">{value}</div>
      {change !== undefined && (
        <div
          className={`text-sm mono font-semibold ${positive ? 'text-green-400' : 'text-red-400'}`}
        >
          {positive ? '+' : ''}
          {change.toFixed(2)}%
        </div>
      )}
      {subtitle && <div className="text-xs text-gray-400 mt-2 mono">{subtitle}</div>}
    </div>
  );
}

// Decision Card Component with CoT Trace
function DecisionCard({ decision }: { decision: DecisionRecord }) {
  const [showCoT, setShowCoT] = useState(false);

  return (
    <div className="border border-gray-800 rounded-lg p-4 bg-gray-950">
      {/* Header */}
      <div className="flex items-start justify-between mb-3">
        <div>
          <div className="font-semibold">Cycle #{decision.cycle_number}</div>
          <div className="text-sm text-gray-400">
            {new Date(decision.timestamp).toLocaleString()}
          </div>
        </div>
        <div
          className={`px-3 py-1 rounded text-sm font-semibold ${
            decision.success
              ? 'bg-green-900/30 text-green-400'
              : 'bg-red-900/30 text-red-400'
          }`}
        >
          {decision.success ? 'Success' : 'Failed'}
        </div>
      </div>

      {/* AI Chain of Thought - Collapsible */}
      {decision.cot_trace && (
        <div className="mb-3">
          <button
            onClick={() => setShowCoT(!showCoT)}
            className="flex items-center gap-2 text-sm text-blue-400 hover:text-blue-300 transition-colors"
          >
            <span className="font-semibold">💭 AI思维链分析</span>
            <span className="text-xs">{showCoT ? '▼ 收起' : '▶ 展开'}</span>
          </button>
          {showCoT && (
            <div className="mt-2 bg-gray-900 rounded p-4 text-sm font-mono whitespace-pre-wrap text-gray-300 max-h-96 overflow-y-auto border border-gray-700">
              {decision.cot_trace}
            </div>
          )}
        </div>
      )}

      {/* Decisions Actions */}
      {decision.decisions && decision.decisions.length > 0 && (
        <div className="space-y-2 mb-3">
          {decision.decisions.map((action, j) => (
            <div key={j} className="flex items-center gap-2 text-sm bg-gray-900 rounded px-3 py-2">
              <span className="font-mono font-semibold">{action.symbol}</span>
              <span
                className={`px-2 py-0.5 rounded text-xs ${
                  action.action.includes('open')
                    ? 'bg-blue-900/30 text-blue-400'
                    : 'bg-yellow-900/30 text-yellow-400'
                }`}
              >
                {action.action}
              </span>
              {action.leverage > 0 && <span className="text-gray-400">{action.leverage}x</span>}
              {action.price > 0 && (
                <span className="text-gray-400 font-mono text-xs">@{action.price.toFixed(4)}</span>
              )}
              <span className={action.success ? 'text-green-400' : 'text-red-400'}>
                {action.success ? '✓' : '✗'}
              </span>
              {action.error && <span className="text-xs text-red-400 ml-2">{action.error}</span>}
            </div>
          ))}
        </div>
      )}

      {/* Account State Summary */}
      {decision.account_state && (
        <div className="flex gap-4 text-xs text-gray-400 mb-3 bg-gray-900/50 rounded px-3 py-2">
          <span>净值: {decision.account_state.total_balance.toFixed(2)} USDT</span>
          <span>可用: {decision.account_state.available_balance.toFixed(2)} USDT</span>
          <span>保证金率: {decision.account_state.margin_used_pct.toFixed(1)}%</span>
          <span>持仓: {decision.account_state.position_count}</span>
        </div>
      )}

      {/* Execution Logs */}
      {decision.execution_log && decision.execution_log.length > 0 && (
        <div className="space-y-1">
          {decision.execution_log.map((log, k) => (
            <div
              key={k}
              className={`text-xs font-mono ${
                log.includes('✓') || log.includes('成功') ? 'text-green-400' : 'text-red-400'
              }`}
            >
              {log}
            </div>
          ))}
        </div>
      )}

      {/* Error Message */}
      {decision.error_message && (
        <div className="text-sm text-red-400 bg-red-900/10 rounded px-3 py-2 mt-3">
          ❌ {decision.error_message}
        </div>
      )}
    </div>
  );
}

export default App;
