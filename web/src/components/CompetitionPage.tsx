import useSWR from 'swr';
import { api } from '../lib/api';
import type { CompetitionData } from '../types';
import { ComparisonChart } from './ComparisonChart';

export function CompetitionPage() {
  const { data: competition } = useSWR<CompetitionData>(
    'competition',
    api.getCompetition,
    {
      refreshInterval: 5000,
      revalidateOnFocus: true,
    }
  );

  if (!competition || !competition.traders) {
    return (
      <div className="text-center py-12 text-gray-500">
        Loading competition data...
      </div>
    );
  }

  // 按收益率排序
  const sortedTraders = [...competition.traders].sort(
    (a, b) => b.total_pnl_pct - a.total_pnl_pct
  );

  // 找出领先者
  const leader = sortedTraders[0];

  return (
    <div className="space-y-8">
      {/* Competition Header */}
      <div className="bg-gradient-to-r from-purple-900/30 to-blue-900/30 rounded-lg p-6 border border-purple-800/50">
        <h1 className="text-3xl font-bold mb-2 flex items-center gap-3">
          🏆 AI Trading Competition
          <span className="text-sm font-normal text-gray-400">
            {competition.count} Traders
          </span>
        </h1>
        <p className="text-gray-400">
          Qwen vs DeepSeek - Real-time Performance Comparison
        </p>
      </div>

      {/* Leader Board */}
      <div className="bg-gray-900 rounded-lg p-6 border border-gray-800">
        <h2 className="text-xl font-bold mb-4 flex items-center gap-2">
          🥇 Leaderboard
        </h2>
        <div className="space-y-3">
          {sortedTraders.map((trader, index) => {
            const isLeader = index === 0;
            const aiModelColor =
              trader.ai_model === 'qwen' ? 'text-purple-400' : 'text-blue-400';
            const borderColor =
              trader.ai_model === 'qwen'
                ? 'border-purple-900/50'
                : 'border-blue-900/50';

            return (
              <div
                key={trader.trader_id}
                className={`bg-gray-950 border ${borderColor} rounded-lg p-4 ${
                  isLeader ? 'ring-2 ring-yellow-500/50' : ''
                }`}
              >
                <div className="flex items-center justify-between">
                  {/* Rank & Name */}
                  <div className="flex items-center gap-4">
                    <div className="text-3xl font-bold text-gray-600 w-8">
                      {index === 0 ? '🥇' : index === 1 ? '🥈' : '🥉'}
                    </div>
                    <div>
                      <div className="font-bold text-lg">{trader.trader_name}</div>
                      <div className={`text-sm mono font-semibold ${aiModelColor}`}>
                        {trader.ai_model.toUpperCase()} Model
                      </div>
                    </div>
                  </div>

                  {/* Stats */}
                  <div className="flex items-center gap-8">
                    {/* Total Equity */}
                    <div className="text-right">
                      <div className="text-xs text-gray-400">Total Equity</div>
                      <div className="text-lg font-bold mono">
                        {trader.total_equity.toFixed(2)} USDT
                      </div>
                    </div>

                    {/* P&L */}
                    <div className="text-right">
                      <div className="text-xs text-gray-400">Total P&L</div>
                      <div
                        className={`text-2xl font-bold mono ${
                          trader.total_pnl >= 0 ? 'text-green-400' : 'text-red-400'
                        }`}
                      >
                        {trader.total_pnl >= 0 ? '+' : ''}
                        {trader.total_pnl_pct.toFixed(2)}%
                      </div>
                      <div className="text-xs text-gray-400 mono">
                        {trader.total_pnl >= 0 ? '+' : ''}
                        {trader.total_pnl.toFixed(2)} USDT
                      </div>
                    </div>

                    {/* Positions */}
                    <div className="text-right">
                      <div className="text-xs text-gray-400">Positions</div>
                      <div className="text-lg font-bold mono">
                        {trader.position_count}
                      </div>
                      <div className="text-xs text-gray-400">
                        Margin: {trader.margin_used_pct.toFixed(1)}%
                      </div>
                    </div>

                    {/* Cycles */}
                    <div className="text-right">
                      <div className="text-xs text-gray-400">Cycles</div>
                      <div className="text-lg font-bold mono">{trader.call_count}</div>
                    </div>

                    {/* Status */}
                    <div>
                      <div
                        className={`px-3 py-1 rounded text-sm font-semibold ${
                          trader.is_running
                            ? 'bg-green-900/30 text-green-400'
                            : 'bg-red-900/30 text-red-400'
                        }`}
                      >
                        {trader.is_running ? 'RUNNING' : 'STOPPED'}
                      </div>
                    </div>
                  </div>
                </div>
              </div>
            );
          })}
        </div>
      </div>

      {/* Performance Comparison Chart */}
      <div className="bg-gray-900 rounded-lg p-6 border border-gray-800">
        <h2 className="text-xl font-bold mb-4">📈 Performance Comparison</h2>
        <ComparisonChart traders={sortedTraders} />
      </div>

      {/* Head-to-Head Stats */}
      {competition.traders.length === 2 && (
        <div className="bg-gray-900 rounded-lg p-6 border border-gray-800">
          <h2 className="text-xl font-bold mb-4">⚔️ Head-to-Head</h2>
          <div className="grid grid-cols-2 gap-8">
            {sortedTraders.map((trader, index) => {
              const isWinning = index === 0;
              const opponent = sortedTraders[1 - index];
              const gap = trader.total_pnl_pct - opponent.total_pnl_pct;

              return (
                <div
                  key={trader.trader_id}
                  className={`p-6 rounded-lg border ${
                    isWinning
                      ? 'bg-green-900/10 border-green-900/50'
                      : 'bg-gray-950 border-gray-800'
                  }`}
                >
                  <div className="text-center">
                    <div
                      className={`text-xl font-bold mb-2 ${
                        trader.ai_model === 'qwen'
                          ? 'text-purple-400'
                          : 'text-blue-400'
                      }`}
                    >
                      {trader.trader_name}
                    </div>
                    <div className="text-3xl font-bold mono mb-2">
                      {trader.total_pnl_pct.toFixed(2)}%
                    </div>
                    {isWinning && gap > 0 && (
                      <div className="text-green-400 text-sm">
                        Leading by {gap.toFixed(2)}%
                      </div>
                    )}
                    {!isWinning && gap < 0 && (
                      <div className="text-red-400 text-sm">
                        Behind by {Math.abs(gap).toFixed(2)}%
                      </div>
                    )}
                  </div>
                </div>
              );
            })}
          </div>
        </div>
      )}
    </div>
  );
}
