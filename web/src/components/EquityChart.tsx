import { useState } from 'react';
import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  ReferenceLine,
} from 'recharts';
import useSWR from 'swr';
import { api } from '../lib/api';

interface EquityPoint {
  timestamp: string;
  total_equity: number;
  pnl: number;
  pnl_pct: number;
  cycle_number: number;
}

interface EquityChartProps {
  traderId?: string;
}

export function EquityChart({ traderId }: EquityChartProps) {
  const [displayMode, setDisplayMode] = useState<'dollar' | 'percent'>('dollar');

  const { data: history, error } = useSWR<EquityPoint[]>(
    traderId ? `equity-history-${traderId}` : 'equity-history',
    () => api.getEquityHistory(traderId),
    {
      refreshInterval: 10000, // 每10秒刷新
    }
  );

  const { data: account } = useSWR(
    traderId ? `account-${traderId}` : 'account',
    () => api.getAccount(traderId),
    {
      refreshInterval: 5000,
    }
  );

  if (error) {
    return (
      <div className="bg-gray-900/50 border border-gray-800 rounded-lg p-6">
        <div className="text-red-400 text-sm">
          ⚠️ 无法加载收益率数据: {error.message}
        </div>
      </div>
    );
  }

  if (!history || history.length === 0) {
    return (
      <div className="bg-gray-900/50 border border-gray-800 rounded-lg p-6">
        <h3 className="text-lg font-semibold mb-4">账户净值曲线</h3>
        <div className="text-center py-12 text-gray-500">
          <div className="text-4xl mb-2">📊</div>
          <div>暂无历史数据</div>
          <div className="text-sm mt-1">运行几个周期后将显示收益率曲线</div>
        </div>
      </div>
    );
  }

  // 限制显示最近的数据点（性能优化）
  // 如果数据超过2000个点，只显示最近2000个
  const MAX_DISPLAY_POINTS = 2000;
  const displayHistory = history.length > MAX_DISPLAY_POINTS
    ? history.slice(-MAX_DISPLAY_POINTS)
    : history;

  // 计算初始余额（使用第一个数据点）
  const initialBalance = history[0]?.total_equity || 1000;

  // 转换数据格式
  const chartData = displayHistory.map((point) => {
    const pnl = point.total_equity - initialBalance;
    const pnlPct = ((pnl / initialBalance) * 100).toFixed(2);
    return {
      time: new Date(point.timestamp).toLocaleTimeString('zh-CN', {
        hour: '2-digit',
        minute: '2-digit',
      }),
      value: displayMode === 'dollar' ? point.total_equity : parseFloat(pnlPct),
      cycle: point.cycle_number,
      raw_equity: point.total_equity,
      raw_pnl: pnl,
      raw_pnl_pct: parseFloat(pnlPct),
    };
  });

  const currentValue = chartData[chartData.length - 1];
  const isProfit = currentValue.raw_pnl >= 0;

  // 计算Y轴范围
  const calculateYDomain = () => {
    if (displayMode === 'percent') {
      // 百分比模式：找到最大最小值，留20%余量
      const values = chartData.map(d => d.value);
      const minVal = Math.min(...values);
      const maxVal = Math.max(...values);
      const range = Math.max(Math.abs(maxVal), Math.abs(minVal));
      const padding = Math.max(range * 0.2, 1); // 至少留1%余量
      return [Math.floor(minVal - padding), Math.ceil(maxVal + padding)];
    } else {
      // 美元模式：以初始余额为基准，上下留10%余量
      const values = chartData.map(d => d.value);
      const minVal = Math.min(...values, initialBalance);
      const maxVal = Math.max(...values, initialBalance);
      const range = maxVal - minVal;
      const padding = Math.max(range * 0.15, initialBalance * 0.01); // 至少留1%余量
      return [
        Math.floor(minVal - padding),
        Math.ceil(maxVal + padding)
      ];
    }
  };

  // 自定义Tooltip - Binance Style
  const CustomTooltip = ({ active, payload }: any) => {
    if (active && payload && payload.length) {
      const data = payload[0].payload;
      return (
        <div className="rounded p-3 shadow-xl" style={{ background: '#1E2329', border: '1px solid #2B3139' }}>
          <div className="text-xs mb-1" style={{ color: '#848E9C' }}>Cycle #{data.cycle}</div>
          <div className="font-bold mono" style={{ color: '#EAECEF' }}>
            {data.raw_equity.toFixed(2)} USDT
          </div>
          <div
            className="text-sm mono font-bold"
            style={{ color: data.raw_pnl >= 0 ? '#0ECB81' : '#F6465D' }}
          >
            {data.raw_pnl >= 0 ? '+' : ''}
            {data.raw_pnl.toFixed(2)} USDT ({data.raw_pnl_pct >= 0 ? '+' : ''}
            {data.raw_pnl_pct}%)
          </div>
        </div>
      );
    }
    return null;
  };

  return (
    <div className="binance-card p-5">
      {/* Header */}
      <div className="flex items-center justify-between mb-5">
        <div>
          <h3 className="text-base font-semibold mb-1" style={{ color: '#EAECEF' }}>账户净值曲线</h3>
          <div className="flex items-baseline gap-3">
            <span className="text-2xl font-bold mono" style={{ color: '#EAECEF' }}>
              {account?.total_equity.toFixed(2) || '0.00'} USDT
            </span>
            <span
              className="text-sm font-bold mono"
              style={{ color: isProfit ? '#0ECB81' : '#F6465D' }}
            >
              {isProfit ? '+' : ''}
              {currentValue.raw_pnl.toFixed(2)} USDT ({isProfit ? '+' : ''}
              {currentValue.raw_pnl_pct}%)
            </span>
          </div>
        </div>

        {/* Display Mode Toggle */}
        <div className="flex gap-1 rounded p-1" style={{ background: '#0B0E11' }}>
          <button
            onClick={() => setDisplayMode('dollar')}
            className="px-3 py-1.5 rounded text-sm font-semibold transition-all"
            style={displayMode === 'dollar'
              ? { background: '#F0B90B', color: '#000' }
              : { background: 'transparent', color: '#848E9C' }
            }
          >
            美元 ($)
          </button>
          <button
            onClick={() => setDisplayMode('percent')}
            className="px-3 py-1.5 rounded text-sm font-semibold transition-all"
            style={displayMode === 'percent'
              ? { background: '#F0B90B', color: '#000' }
              : { background: 'transparent', color: '#848E9C' }
            }
          >
            百分比 (%)
          </button>
        </div>
      </div>

      {/* Chart */}
      <ResponsiveContainer width="100%" height={400}>
        <LineChart data={chartData} margin={{ top: 20, right: 30, left: 10, bottom: 40 }}>
          <defs>
            <linearGradient id="colorGradient" x1="0" y1="0" x2="0" y2="1">
              <stop offset="5%" stopColor="#F0B90B" stopOpacity={0.8} />
              <stop offset="95%" stopColor="#FCD535" stopOpacity={0.2} />
            </linearGradient>
          </defs>
          <CartesianGrid strokeDasharray="3 3" stroke="#2B3139" />
          <XAxis
            dataKey="time"
            stroke="#5E6673"
            tick={{ fill: '#848E9C', fontSize: 11 }}
            tickLine={{ stroke: '#2B3139' }}
            interval={Math.floor(chartData.length / 10)}
            angle={-15}
            textAnchor="end"
            height={60}
          />
          <YAxis
            stroke="#5E6673"
            tick={{ fill: '#848E9C', fontSize: 12 }}
            tickLine={{ stroke: '#2B3139' }}
            domain={calculateYDomain()}
            tickFormatter={(value) =>
              displayMode === 'dollar' ? `$${value.toFixed(0)}` : `${value}%`
            }
          />
          <Tooltip content={<CustomTooltip />} />
          <ReferenceLine
            y={displayMode === 'dollar' ? initialBalance : 0}
            stroke="#474D57"
            strokeDasharray="3 3"
            label={{
              value: displayMode === 'dollar' ? '初始' : '0%',
              fill: '#848E9C',
              fontSize: 12,
            }}
          />
          <Line
            type="monotone"
            dataKey="value"
            stroke="url(#colorGradient)"
            strokeWidth={2.5}
            dot={chartData.length > 50 ? false : { fill: '#F0B90B', r: 3 }}
            activeDot={{ r: 6, fill: '#FCD535', stroke: '#F0B90B', strokeWidth: 2 }}
          />
        </LineChart>
      </ResponsiveContainer>

      {/* Footer Stats */}
      <div className="mt-4 grid grid-cols-4 gap-4 pt-4" style={{ borderTop: '1px solid #2B3139' }}>
        <div>
          <div className="text-xs" style={{ color: '#848E9C' }}>初始余额</div>
          <div className="text-sm font-bold mono" style={{ color: '#EAECEF' }}>
            {initialBalance.toFixed(2)} USDT
          </div>
        </div>
        <div>
          <div className="text-xs" style={{ color: '#848E9C' }}>当前净值</div>
          <div className="text-sm font-bold mono" style={{ color: '#EAECEF' }}>
            {currentValue.raw_equity.toFixed(2)} USDT
          </div>
        </div>
        <div>
          <div className="text-xs" style={{ color: '#848E9C' }}>历史周期</div>
          <div className="text-sm font-bold mono" style={{ color: '#EAECEF' }}>{history.length} 个</div>
        </div>
        <div>
          <div className="text-xs" style={{ color: '#848E9C' }}>显示范围</div>
          <div className="text-sm font-bold mono" style={{ color: '#EAECEF' }}>
            {history.length > MAX_DISPLAY_POINTS
              ? `最近 ${MAX_DISPLAY_POINTS}`
              : '全部'
            }
          </div>
        </div>
      </div>
    </div>
  );
}
