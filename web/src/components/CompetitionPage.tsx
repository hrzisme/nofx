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
      <div className="text-center py-12" style={{ color: '#848E9C' }}>
        <div className="skeleton inline-block w-48 h-8 rounded"></div>
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
    <div className="space-y-6">
      {/* Competition Header - Binance Style */}
      <div className="rounded p-6" style={{ background: 'linear-gradient(135deg, rgba(240, 185, 11, 0.15) 0%, rgba(252, 213, 53, 0.05) 100%)', border: '1px solid rgba(240, 185, 11, 0.2)' }}>
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-2xl font-bold mb-1 flex items-center gap-3" style={{ color: '#EAECEF' }}>
              <span>🏆</span>
              AI Trading Competition
              <span className="text-sm font-normal px-2 py-1 rounded" style={{ background: 'rgba(240, 185, 11, 0.15)', color: '#F0B90B' }}>
                {competition.count} Traders
              </span>
            </h1>
            <p className="text-sm" style={{ color: '#848E9C' }}>
              Qwen vs DeepSeek · Real-time Performance Battle
            </p>
          </div>
          <div className="text-right">
            <div className="text-xs" style={{ color: '#848E9C' }}>Current Leader</div>
            <div className="text-lg font-bold" style={{ color: '#F0B90B' }}>{leader?.trader_name}</div>
          </div>
        </div>
      </div>

      {/* Leader Board - Binance Style */}
      <div className="binance-card p-5">
        <h2 className="text-lg font-bold mb-4 flex items-center gap-2" style={{ color: '#EAECEF' }}>
          🥇 Leaderboard
        </h2>
        <div className="space-y-3">
          {sortedTraders.map((trader, index) => {
            const isLeader = index === 0;
            const aiModelColor = trader.ai_model === 'qwen' ? '#c084fc' : '#60a5fa';

            return (
              <div
                key={trader.trader_id}
                className="rounded p-4"
                style={{
                  background: '#0B0E11',
                  border: `1px solid ${isLeader ? 'rgba(240, 185, 11, 0.3)' : '#2B3139'}`,
                  boxShadow: isLeader ? '0 0 0 1px rgba(240, 185, 11, 0.1)' : 'none'
                }}
              >
                <div className="flex items-center justify-between">
                  {/* Rank & Name */}
                  <div className="flex items-center gap-4">
                    <div className="text-3xl w-8">
                      {index === 0 ? '🥇' : index === 1 ? '🥈' : '🥉'}
                    </div>
                    <div>
                      <div className="font-bold text-base" style={{ color: '#EAECEF' }}>{trader.trader_name}</div>
                      <div className="text-xs mono font-semibold" style={{ color: aiModelColor }}>
                        {trader.ai_model.toUpperCase()} Model
                      </div>
                    </div>
                  </div>

                  {/* Stats */}
                  <div className="flex items-center gap-6">
                    {/* Total Equity */}
                    <div className="text-right">
                      <div className="text-xs" style={{ color: '#848E9C' }}>Total Equity</div>
                      <div className="text-base font-bold mono" style={{ color: '#EAECEF' }}>
                        {trader.total_equity.toFixed(2)} USDT
                      </div>
                    </div>

                    {/* P&L */}
                    <div className="text-right min-w-[120px]">
                      <div className="text-xs" style={{ color: '#848E9C' }}>Total P&L</div>
                      <div
                        className="text-2xl font-bold mono"
                        style={{ color: trader.total_pnl >= 0 ? '#0ECB81' : '#F6465D' }}
                      >
                        {trader.total_pnl >= 0 ? '+' : ''}
                        {trader.total_pnl_pct.toFixed(2)}%
                      </div>
                      <div className="text-xs mono" style={{ color: '#848E9C' }}>
                        {trader.total_pnl >= 0 ? '+' : ''}
                        {trader.total_pnl.toFixed(2)} USDT
                      </div>
                    </div>

                    {/* Positions */}
                    <div className="text-right">
                      <div className="text-xs" style={{ color: '#848E9C' }}>Positions</div>
                      <div className="text-base font-bold mono" style={{ color: '#EAECEF' }}>
                        {trader.position_count}
                      </div>
                      <div className="text-xs" style={{ color: '#848E9C' }}>
                        Margin: {trader.margin_used_pct.toFixed(1)}%
                      </div>
                    </div>

                    {/* Cycles */}
                    <div className="text-right">
                      <div className="text-xs" style={{ color: '#848E9C' }}>Cycles</div>
                      <div className="text-base font-bold mono" style={{ color: '#EAECEF' }}>{trader.call_count}</div>
                    </div>

                    {/* Status */}
                    <div>
                      <div
                        className="px-3 py-1 rounded text-xs font-bold"
                        style={trader.is_running
                          ? { background: 'rgba(14, 203, 129, 0.1)', color: '#0ECB81' }
                          : { background: 'rgba(246, 70, 93, 0.1)', color: '#F6465D' }
                        }
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
      <div className="binance-card p-5">
        <h2 className="text-lg font-bold mb-4" style={{ color: '#EAECEF' }}>📈 Performance Comparison</h2>
        <ComparisonChart traders={sortedTraders} />
      </div>

      {/* Head-to-Head Stats */}
      {competition.traders.length === 2 && (
        <div className="binance-card p-5">
          <h2 className="text-lg font-bold mb-4" style={{ color: '#EAECEF' }}>⚔️ Head-to-Head</h2>
          <div className="grid grid-cols-2 gap-6">
            {sortedTraders.map((trader, index) => {
              const isWinning = index === 0;
              const opponent = sortedTraders[1 - index];
              const gap = trader.total_pnl_pct - opponent.total_pnl_pct;

              return (
                <div
                  key={trader.trader_id}
                  className="p-6 rounded"
                  style={isWinning
                    ? { background: 'rgba(14, 203, 129, 0.05)', border: '1px solid rgba(14, 203, 129, 0.2)' }
                    : { background: '#0B0E11', border: '1px solid #2B3139' }
                  }
                >
                  <div className="text-center">
                    <div
                      className="text-lg font-bold mb-2"
                      style={{ color: trader.ai_model === 'qwen' ? '#c084fc' : '#60a5fa' }}
                    >
                      {trader.trader_name}
                    </div>
                    <div className="text-3xl font-bold mono mb-2" style={{ color: trader.total_pnl >= 0 ? '#0ECB81' : '#F6465D' }}>
                      {trader.total_pnl >= 0 ? '+' : ''}{trader.total_pnl_pct.toFixed(2)}%
                    </div>
                    {isWinning && gap > 0 && (
                      <div className="text-sm font-semibold" style={{ color: '#0ECB81' }}>
                        Leading by {gap.toFixed(2)}%
                      </div>
                    )}
                    {!isWinning && gap < 0 && (
                      <div className="text-sm font-semibold" style={{ color: '#F6465D' }}>
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
