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

  // 自定义Tooltip
  const CustomTooltip = ({ active, payload }: any) => {
    if (active && payload && payload.length) {
      const data = payload[0].payload;
      return (
        <div className="bg-gray-900 border border-gray-700 rounded-lg p-3 shadow-xl">
          <div className="text-xs text-gray-400 mb-1">Cycle #{data.cycle}</div>
          <div className="font-semibold mono">
            {data.raw_equity.toFixed(2)} USDT
          </div>
          <div
            className={`text-sm mono ${
              data.raw_pnl >= 0 ? 'text-green-400' : 'text-red-400'
            }`}
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
    <div className="bg-gray-900/50 border border-gray-800 rounded-lg p-6">
      {/* Header */}
      <div className="flex items-center justify-between mb-6">
        <div>
          <h3 className="text-lg font-semibold mb-1">账户净值曲线</h3>
          <div className="flex items-baseline gap-3">
            <span className="text-2xl font-bold mono">
              {account?.total_equity.toFixed(2) || '0.00'} USDT
            </span>
            <span
              className={`text-sm font-semibold mono ${
                isProfit ? 'text-green-400' : 'text-red-400'
              }`}
            >
              {isProfit ? '+' : ''}
              {currentValue.raw_pnl.toFixed(2)} USDT ({isProfit ? '+' : ''}
              {currentValue.raw_pnl_pct}%)
            </span>
          </div>
        </div>

        {/* Display Mode Toggle */}
        <div className="flex gap-1 bg-gray-800/50 rounded-lg p-1">
          <button
            onClick={() => setDisplayMode('dollar')}
            className={`px-3 py-1.5 rounded text-sm font-medium transition-all ${
              displayMode === 'dollar'
                ? 'bg-gray-700 text-white'
                : 'text-gray-400 hover:text-white'
            }`}
          >
            美元 ($)
          </button>
          <button
            onClick={() => setDisplayMode('percent')}
            className={`px-3 py-1.5 rounded text-sm font-medium transition-all ${
              displayMode === 'percent'
                ? 'bg-gray-700 text-white'
                : 'text-gray-400 hover:text-white'
            }`}
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
              <stop offset="5%" stopColor="#ededed" stopOpacity={0.9} />
              <stop offset="95%" stopColor="#a1a1aa" stopOpacity={0.3} />
            </linearGradient>
          </defs>
          <CartesianGrid strokeDasharray="3 3" stroke="#27272a" />
          <XAxis
            dataKey="time"
            stroke="#71717a"
            tick={{ fill: '#71717a', fontSize: 11 }}
            tickLine={{ stroke: '#27272a' }}
            interval={Math.floor(chartData.length / 10)}
            angle={-15}
            textAnchor="end"
            height={60}
          />
          <YAxis
            stroke="#71717a"
            tick={{ fill: '#71717a', fontSize: 12 }}
            tickLine={{ stroke: '#27272a' }}
            domain={calculateYDomain()}
            tickFormatter={(value) =>
              displayMode === 'dollar' ? `$${value.toFixed(0)}` : `${value}%`
            }
          />
          <Tooltip content={<CustomTooltip />} />
          <ReferenceLine
            y={displayMode === 'dollar' ? initialBalance : 0}
            stroke="#6b7280"
            strokeDasharray="3 3"
            label={{
              value: displayMode === 'dollar' ? '初始' : '0%',
              fill: '#9ca3af',
              fontSize: 12,
            }}
          />
          <Line
            type="monotone"
            dataKey="value"
            stroke="url(#colorGradient)"
            strokeWidth={3}
            dot={chartData.length > 50 ? false : { fill: '#ededed', r: 4 }}
            activeDot={{ r: 6, fill: '#ffffff' }}
          />
        </LineChart>
      </ResponsiveContainer>

      {/* Footer Stats */}
      <div className="mt-4 grid grid-cols-4 gap-4 pt-4 border-t border-gray-800">
        <div>
          <div className="text-xs text-gray-400">初始余额</div>
          <div className="text-sm font-semibold mono">
            {initialBalance.toFixed(2)} USDT
          </div>
        </div>
        <div>
          <div className="text-xs text-gray-400">当前净值</div>
          <div className="text-sm font-semibold mono">
            {currentValue.raw_equity.toFixed(2)} USDT
          </div>
        </div>
        <div>
          <div className="text-xs text-gray-400">历史周期</div>
          <div className="text-sm font-semibold mono">{history.length} 个</div>
        </div>
        <div>
          <div className="text-xs text-gray-400">显示范围</div>
          <div className="text-sm font-semibold mono">
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
