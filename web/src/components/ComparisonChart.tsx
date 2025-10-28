import { useState, useEffect } from 'react';
import {
  LineChart,
  Line,
  XAxis,
  YAxis,
  CartesianGrid,
  Tooltip,
  ResponsiveContainer,
  ReferenceLine,
  Legend,
} from 'recharts';
import useSWR from 'swr';
import { api } from '../lib/api';
import type { CompetitionTraderData } from '../types';

interface ComparisonChartProps {
  traders: CompetitionTraderData[];
}

export function ComparisonChart({ traders }: ComparisonChartProps) {
  const [combinedData, setCombinedData] = useState<any[]>([]);

  // 获取所有trader的历史数据
  const traderHistories = traders.map((trader) => {
    // eslint-disable-next-line react-hooks/rules-of-hooks
    return useSWR(`equity-history-${trader.trader_id}`, () =>
      api.getEquityHistory(trader.trader_id)
    );
  });

  useEffect(() => {
    // 等待所有数据加载完成
    const allLoaded = traderHistories.every((h) => h.data);
    if (!allLoaded) return;

    // 合并所有trader的数据
    const timestampMap = new Map<string, any>();

    traderHistories.forEach((history, index) => {
      const trader = traders[index];
      history.data?.forEach((point: any) => {
        const time = new Date(point.timestamp).toLocaleTimeString('zh-CN', {
          hour: '2-digit',
          minute: '2-digit',
        });
        const timestamp = point.timestamp;

        if (!timestampMap.has(timestamp)) {
          timestampMap.set(timestamp, { time, timestamp });
        }

        const entry = timestampMap.get(timestamp);
        entry[`${trader.trader_id}_pnl_pct`] = point.total_pnl_pct;
        entry[`${trader.trader_id}_equity`] = point.total_equity;
      });
    });

    // 转换为数组并排序
    const combined = Array.from(timestampMap.values()).sort(
      (a, b) => new Date(a.timestamp).getTime() - new Date(b.timestamp).getTime()
    );

    setCombinedData(combined);
  }, [traderHistories.map((h) => h.data).join(',')]);

  const isLoading = traderHistories.some((h) => !h.data);

  if (isLoading) {
    return (
      <div className="text-center py-12 text-gray-500">
        Loading comparison data...
      </div>
    );
  }

  if (combinedData.length === 0) {
    return (
      <div className="text-center py-12 text-gray-500">
        <div className="text-4xl mb-2">📊</div>
        <div>暂无历史数据</div>
        <div className="text-sm mt-1">运行几个周期后将显示对比曲线</div>
      </div>
    );
  }

  // 限制显示数据点
  const MAX_DISPLAY_POINTS = 2000;
  const displayData =
    combinedData.length > MAX_DISPLAY_POINTS
      ? combinedData.slice(-MAX_DISPLAY_POINTS)
      : combinedData;

  // Trader颜色配置
  const getTraderColor = (traderId: string) => {
    const trader = traders.find((t) => t.trader_id === traderId);
    if (trader?.ai_model === 'qwen') {
      return '#a855f7'; // purple-500
    } else {
      return '#3b82f6'; // blue-500
    }
  };

  // 自定义Tooltip
  const CustomTooltip = ({ active, payload }: any) => {
    if (active && payload && payload.length) {
      return (
        <div className="bg-gray-900 border border-gray-700 rounded-lg p-3 shadow-xl">
          <div className="text-xs text-gray-400 mb-2">{payload[0].payload.time}</div>
          {traders.map((trader) => {
            const pnlPct = payload[0].payload[`${trader.trader_id}_pnl_pct`];
            const equity = payload[0].payload[`${trader.trader_id}_equity`];
            if (pnlPct === undefined) return null;

            return (
              <div key={trader.trader_id} className="mb-1">
                <div
                  className="text-xs font-semibold"
                  style={{ color: getTraderColor(trader.trader_id) }}
                >
                  {trader.trader_name}
                </div>
                <div className={`text-sm mono font-semibold ${pnlPct >= 0 ? 'text-green-400' : 'text-red-400'}`}>
                  {pnlPct >= 0 ? '+' : ''}{pnlPct.toFixed(2)}%
                  <span className="text-xs text-gray-400 ml-2">
                    ({equity?.toFixed(2)} USDT)
                  </span>
                </div>
              </div>
            );
          })}
        </div>
      );
    }
    return null;
  };

  return (
    <div>
      <ResponsiveContainer width="100%" height={450}>
        <LineChart data={displayData} margin={{ top: 20, right: 30, left: 10, bottom: 40 }}>
          <defs>
            {traders.map((trader) => (
              <linearGradient
                key={`gradient-${trader.trader_id}`}
                id={`gradient-${trader.trader_id}`}
                x1="0"
                y1="0"
                x2="0"
                y2="1"
              >
                <stop offset="5%" stopColor={getTraderColor(trader.trader_id)} stopOpacity={0.8} />
                <stop offset="95%" stopColor={getTraderColor(trader.trader_id)} stopOpacity={0.1} />
              </linearGradient>
            ))}
          </defs>

          <CartesianGrid strokeDasharray="3 3" stroke="#27272a" />
          <XAxis
            dataKey="time"
            stroke="#71717a"
            tick={{ fill: '#71717a', fontSize: 11 }}
            tickLine={{ stroke: '#27272a' }}
            interval={Math.floor(displayData.length / 10)}
            angle={-15}
            textAnchor="end"
            height={60}
          />
          <YAxis
            stroke="#71717a"
            tick={{ fill: '#71717a', fontSize: 12 }}
            tickLine={{ stroke: '#27272a' }}
            tickFormatter={(value) => `${value}%`}
          />
          <Tooltip content={<CustomTooltip />} />
          <ReferenceLine
            y={0}
            stroke="#6b7280"
            strokeDasharray="3 3"
            label={{
              value: '0%',
              fill: '#9ca3af',
              fontSize: 12,
            }}
          />

          {traders.map((trader) => (
            <Line
              key={trader.trader_id}
              type="monotone"
              dataKey={`${trader.trader_id}_pnl_pct`}
              stroke={getTraderColor(trader.trader_id)}
              strokeWidth={3}
              dot={false}
              activeDot={{ r: 6 }}
              name={trader.trader_name}
            />
          ))}

          <Legend
            wrapperStyle={{ paddingTop: '20px' }}
            formatter={(value, entry: any) => {
              const trader = traders.find((t) => value.includes(t.trader_id));
              return (
                <span style={{ color: entry.color, fontWeight: 600 }}>
                  {trader?.trader_name} ({trader?.ai_model.toUpperCase()})
                </span>
              );
            }}
          />
        </LineChart>
      </ResponsiveContainer>

      {/* Stats */}
      <div className="mt-4 grid grid-cols-3 gap-4 pt-4 border-t border-gray-800">
        <div>
          <div className="text-xs text-gray-400">对比模式</div>
          <div className="text-sm font-semibold">收益率百分比 (%)</div>
        </div>
        <div>
          <div className="text-xs text-gray-400">历史周期</div>
          <div className="text-sm font-semibold mono">{combinedData.length} 个</div>
        </div>
        <div>
          <div className="text-xs text-gray-400">显示范围</div>
          <div className="text-sm font-semibold mono">
            {combinedData.length > MAX_DISPLAY_POINTS
              ? `最近 ${MAX_DISPLAY_POINTS}`
              : '全部'}
          </div>
        </div>
      </div>
    </div>
  );
}
