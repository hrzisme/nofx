import { useEffect, useState } from 'react';
import useSWR from 'swr';

interface TradeOutcome {
  symbol: string;
  side: string;
  open_price: number;
  close_price: number;
  pn_l: number;
  pn_l_pct: number;
  duration: string;
  open_time: string;
  close_time: string;
  was_stop_loss: boolean;
}

interface SymbolPerformance {
  symbol: string;
  total_trades: number;
  winning_trades: number;
  losing_trades: number;
  win_rate: number;
  total_pn_l: number;
  avg_pn_l: number;
}

interface PerformanceAnalysis {
  total_trades: number;
  winning_trades: number;
  losing_trades: number;
  win_rate: number;
  avg_win: number;
  avg_loss: number;
  profit_factor: number;
  recent_trades: TradeOutcome[];
  symbol_stats: { [key: string]: SymbolPerformance };
  best_symbol: string;
  worst_symbol: string;
}

interface AILearningProps {
  traderId: string;
}

interface DecisionRecord {
  timestamp: string;
  cycle_number: number;
  cot_trace: string;
  success: boolean;
}

const fetcher = (url: string) => fetch(url).then(res => res.json());

export default function AILearning({ traderId }: AILearningProps) {
  const { data: performance, error } = useSWR<PerformanceAnalysis>(
    `http://localhost:8080/api/performance?trader_id=${traderId}`,
    fetcher,
    { refreshInterval: 10000 }
  );

  // 获取最新的决策记录，查看AI的思考过程
  const { data: latestDecisions } = useSWR<DecisionRecord[]>(
    `http://localhost:8080/api/decisions/latest?trader_id=${traderId}`,
    fetcher,
    { refreshInterval: 10000 }
  );

  if (error) {
    return (
      <div className="rounded p-6" style={{ background: '#1E2329', border: '1px solid #2B3139' }}>
        <div style={{ color: '#F6465D' }}>⚠️ 加载AI学习数据失败</div>
      </div>
    );
  }

  if (!performance) {
    return (
      <div className="rounded p-6" style={{ background: '#1E2329', border: '1px solid #2B3139' }}>
        <div style={{ color: '#848E9C' }}>📊 加载中...</div>
      </div>
    );
  }

  if (!performance || performance.total_trades === 0) {
    return (
      <div className="rounded p-6" style={{ background: '#1E2329', border: '1px solid #2B3139' }}>
        <div className="flex items-center gap-2 mb-2">
          <span className="text-xl">🧠</span>
          <h2 className="text-lg font-bold" style={{ color: '#EAECEF' }}>AI 学习分析</h2>
        </div>
        <div style={{ color: '#848E9C' }}>
          暂无完整交易数据（需要完成开仓→平仓的完整周期）
        </div>
      </div>
    );
  }

  // 安全地获取symbol_stats
  const symbolStats = performance.symbol_stats || {};
  const symbolStatsList = Object.values(symbolStats).filter(stat => stat != null).sort(
    (a, b) => (b.total_pn_l || 0) - (a.total_pn_l || 0)
  );

  // 提取AI的最新反思（从CoT trace中）
  const latestReflection = extractReflectionFromCoT(latestDecisions?.[0]?.cot_trace);

  return (
    <div className="space-y-6">
      {/* 标题区 - 更简洁 */}
      <div className="flex items-center gap-4">
        <div className="w-12 h-12 rounded-xl flex items-center justify-center text-2xl" style={{
          background: 'linear-gradient(135deg, #8B5CF6 0%, #6366F1 100%)',
          boxShadow: '0 4px 14px rgba(139, 92, 246, 0.4)'
        }}>
          🧠
        </div>
        <div>
          <h2 className="text-2xl font-bold" style={{ color: '#EAECEF' }}>AI Learning & Reflection</h2>
          <p className="text-sm" style={{ color: '#848E9C' }}>
            {performance.total_trades} trades analyzed · Real-time evolution
          </p>
        </div>
      </div>

      {/* 主要内容：现代化网格布局 */}
      <div className="grid grid-cols-1 lg:grid-cols-12 gap-6">

        {/* 左侧大区域：AI反思 + 核心指标 (5列) */}
        <div className="lg:col-span-5 space-y-6">
          {/* AI最新反思 - 渐变卡片 */}
          {latestReflection && latestDecisions && latestDecisions.length > 0 && (
            <div className="rounded-2xl p-6 relative overflow-hidden" style={{
              background: 'linear-gradient(135deg, #1E1B4B 0%, #312E81 50%, #1E293B 100%)',
              border: '1px solid rgba(139, 92, 246, 0.2)',
              boxShadow: '0 10px 40px rgba(139, 92, 246, 0.15)'
            }}>
              {/* 背景装饰 */}
              <div className="absolute top-0 right-0 w-64 h-64 rounded-full opacity-10" style={{
                background: 'radial-gradient(circle, #8B5CF6 0%, transparent 70%)',
                filter: 'blur(40px)'
              }} />

              <div className="relative">
                <div className="flex items-center gap-3 mb-4">
                  <div className="w-10 h-10 rounded-lg flex items-center justify-center text-xl" style={{
                    background: 'rgba(139, 92, 246, 0.2)',
                    border: '1px solid rgba(139, 92, 246, 0.3)'
                  }}>
                    🎯
                  </div>
                  <div>
                    <h3 className="text-base font-bold" style={{ color: '#C4B5FD' }}>
                      Latest Reflection
                    </h3>
                    <p className="text-xs" style={{ color: '#94A3B8' }}>
                      Cycle #{latestDecisions[0].cycle_number} · {new Date(latestDecisions[0].timestamp).toLocaleTimeString()}
                    </p>
                  </div>
                </div>

                <div className="rounded-xl p-4 text-sm leading-relaxed whitespace-pre-wrap" style={{
                  background: 'rgba(0, 0, 0, 0.4)',
                  backdropFilter: 'blur(10px)',
                  border: '1px solid rgba(139, 92, 246, 0.1)',
                  color: '#DDD6FE',
                  fontFamily: 'ui-sans-serif, system-ui'
                }}>
                  {latestReflection}
                </div>

                {latestDecisions[0].cot_trace && (
                  <details className="mt-4">
                    <summary className="cursor-pointer text-sm font-semibold flex items-center gap-2 hover:opacity-80 transition-opacity" style={{ color: '#A78BFA' }}>
                      <span>📋 Full Chain of Thought</span>
                    </summary>
                    <div className="mt-3 rounded-xl p-4 text-xs leading-relaxed whitespace-pre-wrap max-h-80 overflow-y-auto" style={{
                      background: 'rgba(0, 0, 0, 0.5)',
                      border: '1px solid rgba(139, 92, 246, 0.15)',
                      color: '#A5B4FC',
                      fontFamily: 'ui-monospace, monospace'
                    }}>
                      {latestDecisions[0].cot_trace}
                    </div>
                  </details>
                )}
              </div>
            </div>
          )}

          {/* 核心指标网格 - 玻璃态设计 */}
          <div className="grid grid-cols-2 gap-4">
            {/* 总交易数 */}
            <div className="rounded-xl p-4 backdrop-blur-sm" style={{
              background: 'rgba(30, 35, 41, 0.6)',
              border: '1px solid rgba(99, 102, 241, 0.2)',
              boxShadow: '0 4px 12px rgba(0, 0, 0, 0.1)'
            }}>
              <div className="text-xs font-semibold mb-2" style={{ color: '#94A3B8' }}>Total Trades</div>
              <div className="text-3xl font-bold mono" style={{ color: '#E0E7FF' }}>
                {performance.total_trades}
              </div>
            </div>

            {/* 胜率 */}
            <div className="rounded-xl p-4 backdrop-blur-sm" style={{
              background: (performance.win_rate || 0) >= 50
                ? 'rgba(14, 203, 129, 0.1)'
                : 'rgba(246, 70, 93, 0.1)',
              border: `1px solid ${(performance.win_rate || 0) >= 50 ? 'rgba(14, 203, 129, 0.3)' : 'rgba(246, 70, 93, 0.3)'}`,
              boxShadow: '0 4px 12px rgba(0, 0, 0, 0.1)'
            }}>
              <div className="text-xs font-semibold mb-2" style={{ color: '#94A3B8' }}>Win Rate</div>
              <div className="text-3xl font-bold mono" style={{
                color: (performance.win_rate || 0) >= 50 ? '#10B981' : '#F87171'
              }}>
                {(performance.win_rate || 0).toFixed(1)}%
              </div>
              <div className="text-xs mt-1" style={{ color: '#94A3B8' }}>
                {performance.winning_trades || 0}W / {performance.losing_trades || 0}L
              </div>
            </div>

            {/* 平均盈利 */}
            <div className="rounded-xl p-4 backdrop-blur-sm" style={{
              background: 'rgba(14, 203, 129, 0.08)',
              border: '1px solid rgba(14, 203, 129, 0.2)',
              boxShadow: '0 4px 12px rgba(0, 0, 0, 0.1)'
            }}>
              <div className="text-xs font-semibold mb-2" style={{ color: '#94A3B8' }}>Avg Win</div>
              <div className="text-3xl font-bold mono" style={{ color: '#10B981' }}>
                +{(performance.avg_win || 0).toFixed(2)}%
              </div>
            </div>

            {/* 平均亏损 */}
            <div className="rounded-xl p-4 backdrop-blur-sm" style={{
              background: 'rgba(246, 70, 93, 0.08)',
              border: '1px solid rgba(246, 70, 93, 0.2)',
              boxShadow: '0 4px 12px rgba(0, 0, 0, 0.1)'
            }}>
              <div className="text-xs font-semibold mb-2" style={{ color: '#94A3B8' }}>Avg Loss</div>
              <div className="text-3xl font-bold mono" style={{ color: '#F87171' }}>
                {(performance.avg_loss || 0).toFixed(2)}%
              </div>
            </div>
          </div>

          {/* 盈亏比 - 突出显示 */}
          <div className="rounded-2xl p-6 relative overflow-hidden" style={{
            background: 'linear-gradient(135deg, rgba(240, 185, 11, 0.15) 0%, rgba(252, 213, 53, 0.05) 100%)',
            border: '1px solid rgba(240, 185, 11, 0.3)',
            boxShadow: '0 8px 24px rgba(240, 185, 11, 0.2)'
          }}>
            <div className="absolute top-0 right-0 w-40 h-40 rounded-full opacity-20" style={{
              background: 'radial-gradient(circle, #F0B90B 0%, transparent 70%)',
              filter: 'blur(30px)'
            }} />

            <div className="relative flex items-center justify-between">
              <div>
                <div className="text-sm font-semibold mb-1" style={{ color: '#FCD34D' }}>Profit Factor</div>
                <div className="text-xs" style={{ color: '#94A3B8' }}>
                  Avg Win ÷ Avg Loss
                </div>
              </div>
              <div className="text-5xl font-bold mono" style={{
                color: (performance.profit_factor || 0) >= 2.0 ? '#10B981' :
                       (performance.profit_factor || 0) >= 1.5 ? '#F0B90B' :
                       (performance.profit_factor || 0) >= 1.0 ? '#FB923C' : '#F87171'
              }}>
                {(performance.profit_factor || 0) > 0 ? (performance.profit_factor || 0).toFixed(2) : 'N/A'}
              </div>
            </div>

            <div className="mt-3 text-xs font-semibold" style={{
              color: (performance.profit_factor || 0) >= 2.0 ? '#10B981' :
                     (performance.profit_factor || 0) >= 1.5 ? '#F0B90B' : '#94A3B8'
            }}>
              {(performance.profit_factor || 0) >= 2.0 && '🔥 Excellent - Strong profitability'}
              {(performance.profit_factor || 0) >= 1.5 && (performance.profit_factor || 0) < 2.0 && '✓ Good - Stable profits'}
              {(performance.profit_factor || 0) >= 1.0 && (performance.profit_factor || 0) < 1.5 && '⚠️ Fair - Needs optimization'}
              {(performance.profit_factor || 0) > 0 && (performance.profit_factor || 0) < 1.0 && '❌ Poor - Losses exceed gains'}
            </div>
          </div>
        </div>
        {/* 左侧结束 */}

        {/* 中间列：数据表格 (4列) */}
        <div className="lg:col-span-4 space-y-6">
          {/* 最佳/最差币种 */}
          {(performance.best_symbol || performance.worst_symbol) && (
            <div className="grid grid-cols-2 gap-4">
              {performance.best_symbol && (
                <div className="rounded-xl p-5 backdrop-blur-sm" style={{
                  background: 'linear-gradient(135deg, rgba(16, 185, 129, 0.15) 0%, rgba(14, 203, 129, 0.05) 100%)',
                  border: '1px solid rgba(16, 185, 129, 0.3)',
                  boxShadow: '0 4px 16px rgba(16, 185, 129, 0.1)'
                }}>
                  <div className="flex items-center gap-2 mb-3">
                    <span className="text-xl">🏆</span>
                    <span className="text-xs font-semibold" style={{ color: '#6EE7B7' }}>Best Performer</span>
                  </div>
                  <div className="text-2xl font-bold mono mb-1" style={{ color: '#10B981' }}>
                    {performance.best_symbol}
                  </div>
                  {symbolStats[performance.best_symbol] && (
                    <div className="text-sm font-semibold" style={{ color: '#6EE7B7' }}>
                      {symbolStats[performance.best_symbol].total_pn_l > 0 ? '+' : ''}
                      {symbolStats[performance.best_symbol].total_pn_l.toFixed(2)}% P&L
                    </div>
                  )}
                </div>
              )}

              {performance.worst_symbol && (
                <div className="rounded-xl p-5 backdrop-blur-sm" style={{
                  background: 'linear-gradient(135deg, rgba(248, 113, 113, 0.15) 0%, rgba(246, 70, 93, 0.05) 100%)',
                  border: '1px solid rgba(248, 113, 113, 0.3)',
                  boxShadow: '0 4px 16px rgba(248, 113, 113, 0.1)'
                }}>
                  <div className="flex items-center gap-2 mb-3">
                    <span className="text-xl">📉</span>
                    <span className="text-xs font-semibold" style={{ color: '#FCA5A5' }}>Worst Performer</span>
                  </div>
                  <div className="text-2xl font-bold mono mb-1" style={{ color: '#F87171' }}>
                    {performance.worst_symbol}
                  </div>
                  {symbolStats[performance.worst_symbol] && (
                    <div className="text-sm font-semibold" style={{ color: '#FCA5A5' }}>
                      {symbolStats[performance.worst_symbol].total_pn_l > 0 ? '+' : ''}
                      {symbolStats[performance.worst_symbol].total_pn_l.toFixed(2)}% P&L
                    </div>
                  )}
                </div>
              )}
            </div>
          )}

          {/* 币种表现统计 - 现代化表格 */}
          {symbolStatsList.length > 0 && (
            <div className="rounded-2xl overflow-hidden" style={{
              background: 'rgba(30, 35, 41, 0.4)',
              border: '1px solid rgba(99, 102, 241, 0.2)',
              boxShadow: '0 4px 16px rgba(0, 0, 0, 0.2)'
            }}>
              <div className="p-5 border-b" style={{ borderColor: 'rgba(99, 102, 241, 0.2)', background: 'rgba(30, 35, 41, 0.6)' }}>
                <h3 className="font-bold flex items-center gap-2" style={{ color: '#E0E7FF' }}>
                  📊 Symbol Performance
                </h3>
              </div>
              <div className="overflow-x-auto">
                <table className="w-full">
                  <thead>
                    <tr style={{ background: 'rgba(15, 23, 42, 0.6)' }}>
                      <th className="text-left px-4 py-3 text-xs font-semibold" style={{ color: '#94A3B8' }}>Symbol</th>
                      <th className="text-right px-4 py-3 text-xs font-semibold" style={{ color: '#94A3B8' }}>Trades</th>
                      <th className="text-right px-4 py-3 text-xs font-semibold" style={{ color: '#94A3B8' }}>Win Rate</th>
                      <th className="text-right px-4 py-3 text-xs font-semibold" style={{ color: '#94A3B8' }}>Total P&L</th>
                      <th className="text-right px-4 py-3 text-xs font-semibold" style={{ color: '#94A3B8' }}>Avg P&L</th>
                    </tr>
                  </thead>
                  <tbody>
                    {symbolStatsList.map((stat, idx) => (
                      <tr key={stat.symbol} className="transition-colors hover:bg-white/5" style={{
                        borderTop: idx > 0 ? '1px solid rgba(99, 102, 241, 0.1)' : 'none'
                      }}>
                        <td className="px-4 py-3">
                          <span className="font-bold mono text-sm" style={{ color: '#E0E7FF' }}>{stat.symbol}</span>
                        </td>
                        <td className="px-4 py-3 text-right mono text-sm" style={{ color: '#CBD5E1' }}>
                          {stat.total_trades}
                        </td>
                        <td className="px-4 py-3 text-right mono text-sm font-semibold" style={{
                          color: (stat.win_rate || 0) >= 50 ? '#10B981' : '#F87171'
                        }}>
                          {(stat.win_rate || 0).toFixed(1)}%
                        </td>
                        <td className="px-4 py-3 text-right mono text-sm font-bold" style={{
                          color: (stat.total_pn_l || 0) > 0 ? '#10B981' : '#F87171'
                        }}>
                          {(stat.total_pn_l || 0) > 0 ? '+' : ''}{(stat.total_pn_l || 0).toFixed(2)}%
                        </td>
                        <td className="px-4 py-3 text-right mono text-sm" style={{
                          color: (stat.avg_pn_l || 0) > 0 ? '#10B981' : '#F87171'
                        }}>
                          {(stat.avg_pn_l || 0) > 0 ? '+' : ''}{(stat.avg_pn_l || 0).toFixed(2)}%
                        </td>
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
            </div>
          )}

        </div>
        {/* 中间列结束 */}

        {/* 右侧列：最近决策 (3列) */}
        <div className="lg:col-span-3 space-y-4">
          {latestDecisions && latestDecisions.length > 0 ? (
            <>
              {/* 标题卡 */}
              <div className="rounded-xl p-4 backdrop-blur-sm" style={{
                background: 'rgba(99, 102, 241, 0.1)',
                border: '1px solid rgba(99, 102, 241, 0.3)'
              }}>
                <div className="flex items-center gap-2">
                  <span className="text-xl">📝</span>
                  <div>
                    <h3 className="font-bold text-sm" style={{ color: '#C4B5FD' }}>Recent Decisions</h3>
                    <p className="text-xs" style={{ color: '#94A3B8' }}>
                      Last {latestDecisions.length} cycles
                    </p>
                  </div>
                </div>
              </div>

              {/* 决策卡片列表 */}
              <div className="space-y-3">
                {latestDecisions.map((decision, idx) => (
                  <div key={idx} className="rounded-xl p-4 backdrop-blur-sm transition-all hover:scale-[1.02]" style={{
                    background: idx === 0
                      ? 'linear-gradient(135deg, rgba(139, 92, 246, 0.15) 0%, rgba(99, 102, 241, 0.05) 100%)'
                      : 'rgba(30, 35, 41, 0.4)',
                    border: idx === 0
                      ? '1px solid rgba(139, 92, 246, 0.4)'
                      : '1px solid rgba(71, 85, 105, 0.3)',
                    boxShadow: idx === 0
                      ? '0 4px 16px rgba(139, 92, 246, 0.2)'
                      : '0 2px 8px rgba(0, 0, 0, 0.1)'
                  }}>
                    {/* 头部 */}
                    <div className="flex items-center justify-between mb-3">
                      <div className="flex items-center gap-2">
                        <span className="text-xs font-bold mono px-2 py-1 rounded" style={{
                          background: idx === 0 ? 'rgba(139, 92, 246, 0.3)' : 'rgba(99, 102, 241, 0.2)',
                          color: idx === 0 ? '#C4B5FD' : '#A5B4FC'
                        }}>
                          #{decision.cycle_number}
                        </span>
                        {idx === 0 && (
                          <span className="text-xs px-2 py-0.5 rounded font-semibold" style={{
                            background: 'rgba(139, 92, 246, 0.2)',
                            color: '#C4B5FD'
                          }}>
                            Latest
                          </span>
                        )}
                      </div>
                      <div className="text-xs px-2 py-1 rounded font-semibold" style={{
                        background: decision.success ? 'rgba(16, 185, 129, 0.2)' : 'rgba(248, 113, 113, 0.2)',
                        color: decision.success ? '#6EE7B7' : '#FCA5A5'
                      }}>
                        {decision.success ? '✓' : '✗'}
                      </div>
                    </div>

                    {/* 时间 */}
                    <div className="text-xs mb-3 font-mono" style={{ color: '#94A3B8' }}>
                      {new Date(decision.timestamp).toLocaleString('en-US', {
                        month: 'short',
                        day: '2-digit',
                        hour: '2-digit',
                        minute: '2-digit'
                      })}
                    </div>

                    {/* 思维链 */}
                    {decision.cot_trace && (
                      <details className="group">
                        <summary className="cursor-pointer text-xs font-semibold flex items-center gap-1 hover:opacity-70 transition-opacity" style={{
                          color: idx === 0 ? '#A78BFA' : '#818CF8'
                        }}>
                          <span>View CoT</span>
                          <span className="text-[10px]">▶</span>
                        </summary>
                        <div className="mt-2 rounded-lg p-3 text-xs leading-relaxed whitespace-pre-wrap max-h-48 overflow-y-auto scrollbar-thin" style={{
                          background: 'rgba(0, 0, 0, 0.4)',
                          border: '1px solid rgba(139, 92, 246, 0.2)',
                          color: '#C4B5FD',
                          fontFamily: 'ui-monospace, monospace'
                        }}>
                          {decision.cot_trace.length > 400
                            ? decision.cot_trace.substring(0, 400) + '...'
                            : decision.cot_trace}
                        </div>
                      </details>
                    )}
                  </div>
                ))}
              </div>
            </>
          ) : (
            <div className="rounded-xl p-6 text-center backdrop-blur-sm" style={{
              background: 'rgba(30, 35, 41, 0.4)',
              border: '1px solid rgba(71, 85, 105, 0.3)'
            }}>
              <div style={{ color: '#94A3B8' }}>No decisions yet</div>
            </div>
          )}
        </div>
        {/* 右侧列结束 */}

      </div>
      {/* 3列布局结束 */}

      {/* AI学习说明 - 现代化设计 */}
      <div className="rounded-2xl p-6 backdrop-blur-sm" style={{
        background: 'linear-gradient(135deg, rgba(240, 185, 11, 0.1) 0%, rgba(252, 213, 53, 0.05) 100%)',
        border: '1px solid rgba(240, 185, 11, 0.2)',
        boxShadow: '0 4px 16px rgba(240, 185, 11, 0.1)'
      }}>
        <div className="flex items-start gap-4">
          <div className="w-10 h-10 rounded-lg flex items-center justify-center text-xl flex-shrink-0" style={{
            background: 'rgba(240, 185, 11, 0.2)',
            border: '1px solid rgba(240, 185, 11, 0.3)'
          }}>
            💡
          </div>
          <div>
            <h3 className="font-bold mb-3 text-base" style={{ color: '#FCD34D' }}>How AI Learns & Evolves</h3>
            <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4 text-sm">
              <div className="flex items-start gap-2">
                <span style={{ color: '#F0B90B' }}>•</span>
                <span style={{ color: '#CBD5E1' }}>Analyzes last 20 trading cycles before each decision</span>
              </div>
              <div className="flex items-start gap-2">
                <span style={{ color: '#F0B90B' }}>•</span>
                <span style={{ color: '#CBD5E1' }}>Identifies best & worst performing symbols</span>
              </div>
              <div className="flex items-start gap-2">
                <span style={{ color: '#F0B90B' }}>•</span>
                <span style={{ color: '#CBD5E1' }}>Optimizes position sizing based on win rate</span>
              </div>
              <div className="flex items-start gap-2">
                <span style={{ color: '#F0B90B' }}>•</span>
                <span style={{ color: '#CBD5E1' }}>Avoids repeating past mistakes</span>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}

// 格式化持仓时长
function formatDuration(duration: string | undefined): string {
  if (!duration) return '-';

  // duration格式: "1h23m45s" or "23m45.123s"
  const match = duration.match(/(\d+h)?(\d+m)?(\d+\.?\d*s)?/);
  if (!match) return duration;

  const hours = match[1] || '';
  const minutes = match[2] || '';
  const seconds = match[3] || '';

  let result = '';
  if (hours) result += hours.replace('h', '小时');
  if (minutes) result += minutes.replace('m', '分');
  if (!hours && seconds) result += seconds.replace(/(\d+)\.?\d*s/, '$1秒');

  return result || duration;
}

// 从CoT trace中提取AI的历史表现分析和反思
function extractReflectionFromCoT(cotTrace: string | undefined): string | null {
  if (!cotTrace) return null;

  // 优先提取【历史经验反思】部分（新格式）
  const reflectionMatch = cotTrace.match(/【历史经验反思】\s*([\s\S]*?)(?=【|$)/);
  if (reflectionMatch) {
    const reflection = reflectionMatch[1].trim();
    if (reflection.length > 50) {
      return `🎯 AI历史经验总结\n\n${reflection}`;
    }
  }

  // 尝试提取"历史表现反馈"部分（兼容旧格式）
  const performanceSectionMatch = cotTrace.match(/## 📊 历史表现反馈[\s\S]*?(?=##|$)/);
  if (performanceSectionMatch) {
    const performanceSection = performanceSectionMatch[0];

    // 提取关键学习点
    const lines: string[] = [];

    // 提取总体统计
    const statsMatch = performanceSection.match(/总交易数：(\d+).*?胜率：([\d.]+)%.*?盈亏比：([\d.]+)/s);
    if (statsMatch) {
      const [, totalTrades, winRate, profitFactor] = statsMatch;
      lines.push(`📈 历史表现回顾：`);
      lines.push(`   • 完成了 ${totalTrades} 笔交易，胜率 ${winRate}%`);
      lines.push(`   • 盈亏比 ${profitFactor}（平均盈利/平均亏损）`);
      lines.push('');
    }

    // 提取最近交易
    const recentTradesMatch = performanceSection.match(/最近5笔交易[\s\S]*?(?=##|表现最好|$)/);
    if (recentTradesMatch) {
      const tradesText = recentTradesMatch[0];
      const tradeLines = tradesText.split('\n').filter(line => line.trim().startsWith('-'));

      if (tradeLines.length > 0) {
        lines.push(`🔍 最近交易分析：`);
        tradeLines.slice(0, 3).forEach(line => {
          lines.push(`   ${line.trim()}`);
        });
        lines.push('');
      }
    }

    // 提取最佳/最差币种
    const bestWorstMatch = performanceSection.match(/表现最好：([A-Z]+).*?\((.*?)\).*?表现最差：([A-Z]+).*?\((.*?)\)/s);
    if (bestWorstMatch) {
      const [, bestSymbol, bestPnl, worstSymbol, worstPnl] = bestWorstMatch;
      lines.push(`💡 币种表现洞察：`);
      lines.push(`   🏆 ${bestSymbol} 表现最佳 ${bestPnl}`);
      lines.push(`   💔 ${worstSymbol} 表现较差 ${worstPnl}`);
      lines.push('');
    }

    // 尝试提取AI的分析或决策理由
    const analysisMatch = cotTrace.match(/(?:分析|策略|决策)[:：]([\s\S]*?)(?:\n\n|##|$)/);
    if (analysisMatch) {
      const analysis = analysisMatch[1].trim();
      if (analysis.length > 20 && analysis.length < 500) {
        lines.push(`🎯 AI策略调整：`);
        lines.push(`   ${analysis.substring(0, 300)}${analysis.length > 300 ? '...' : ''}`);
      }
    }

    if (lines.length > 0) {
      return lines.join('\n');
    }
  }

  // 如果没有找到历史表现反馈，尝试提取整体思路
  const thinkingMatch = cotTrace.match(/(?:思考|分析|策略)[:：]([\s\S]{50,500}?)(?:\n\n|##|决策|$)/);
  if (thinkingMatch) {
    return `🤔 AI思考过程：\n\n${thinkingMatch[1].trim().substring(0, 400)}...`;
  }

  // 如果都没有，返回CoT的前面部分
  if (cotTrace.length > 100) {
    return `💭 AI分析摘要：\n\n${cotTrace.substring(0, 400).trim()}...`;
  }

  return null;
}
