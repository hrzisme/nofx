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
      api.getEquityHistory(trader.trader_id),
      { refreshInterval: 10000 }
    );
  });

  useEffect(() => {
    // 等待所有数据加载完成
    const allLoaded = traderHistories.every((h) => h.data);
    if (!allLoaded) return;

    // 合并所有trader的数据 - 使用cycle_number作为key确保数据对齐
    const cycleMap = new Map<number, any>();

    traderHistories.forEach((history, index) => {
      const trader = traders[index];
      history.data?.forEach((point: any) => {
        const cycleNumber = point.cycle_number || 0;
        const time = new Date(point.timestamp).toLocaleTimeString('zh-CN', {
          hour: '2-digit',
          minute: '2-digit',
        });

        if (!cycleMap.has(cycleNumber)) {
          cycleMap.set(cycleNumber, {
            cycle: cycleNumber,
            time,
            timestamp: point.timestamp
          });
        }

        const entry = cycleMap.get(cycleNumber);
        entry[`${trader.trader_id}_pnl_pct`] = point.total_pnl_pct;
        entry[`${trader.trader_id}_equity`] = point.total_equity;
      });
    });

    // 转换为数组并按cycle排序
    const combined = Array.from(cycleMap.values())
      .filter(item => {
        // 只保留所有trader都有数据的点
        return traders.every(t => item[`${t.trader_id}_pnl_pct`] !== undefined);
      })
      .sort((a, b) => a.cycle - b.cycle);

    setCombinedData(combined);
  }, [traderHistories.map((h) => h.data).join(',')]);

  const isLoading = traderHistories.some((h) => !h.data);

  if (isLoading) {
    return (
      <div className="text-center py-12 text-gray-500">
        <div className="animate-pulse">Loading comparison data...</div>
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

  // 计算Y轴范围
  const calculateYDomain = () => {
    const allValues: number[] = [];
    displayData.forEach((point) => {
      traders.forEach((trader) => {
        const value = point[`${trader.trader_id}_pnl_pct`];
        if (value !== undefined) {
          allValues.push(value);
        }
      });
    });

    if (allValues.length === 0) return [-5, 5];

    const minVal = Math.min(...allValues);
    const maxVal = Math.max(...allValues);
    const range = Math.max(Math.abs(maxVal), Math.abs(minVal));
    const padding = Math.max(range * 0.2, 1); // 至少留1%余量

    return [
      Math.floor(minVal - padding),
      Math.ceil(maxVal + padding)
    ];
  };

  // Trader颜色配置 - 使用更鲜艳对比度更高的颜色
  const getTraderColor = (traderId: string) => {
    const trader = traders.find((t) => t.trader_id === traderId);
    if (trader?.ai_model === 'qwen') {
      return '#c084fc'; // purple-400 (更亮)
    } else {
      return '#60a5fa'; // blue-400 (更亮)
    }
  };

  // 自定义Tooltip
  const CustomTooltip = ({ active, payload }: any) => {
    if (active && payload && payload.length) {
      const data = payload[0].payload;
      return (
        <div className="bg-gray-900 border border-gray-700 rounded-lg p-3 shadow-xl">
          <div className="text-xs text-gray-400 mb-2">
            Cycle #{data.cycle} - {data.time}
          </div>
          {traders.map((trader) => {
            const pnlPct = data[`${trader.trader_id}_pnl_pct`];
            const equity = data[`${trader.trader_id}_equity`];
            if (pnlPct === undefined) return null;

            return (
              <div key={trader.trader_id} className="mb-1.5 last:mb-0">
                <div
                  className="text-xs font-semibold mb-0.5"
                  style={{ color: getTraderColor(trader.trader_id) }}
                >
                  {trader.trader_name}
                </div>
                <div className={`text-sm mono font-bold ${pnlPct >= 0 ? 'text-green-400' : 'text-red-400'}`}>
                  {pnlPct >= 0 ? '+' : ''}{pnlPct.toFixed(2)}%
                  <span className="text-xs text-gray-500 ml-2 font-normal">
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

  // 计算当前差距
  const currentGap = displayData.length > 0 ? (() => {
    const lastPoint = displayData[displayData.length - 1];
    const values = traders.map(t => lastPoint[`${t.trader_id}_pnl_pct`] || 0);
    return Math.abs(values[0] - values[1]);
  })() : 0;

  return (
    <div>
      <ResponsiveContainer width="100%" height={500}>
        <LineChart data={displayData} margin={{ top: 20, right: 30, left: 20, bottom: 40 }}>
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
                <stop offset="5%" stopColor={getTraderColor(trader.trader_id)} stopOpacity={0.9} />
                <stop offset="95%" stopColor={getTraderColor(trader.trader_id)} stopOpacity={0.2} />
              </linearGradient>
            ))}
          </defs>

          <CartesianGrid strokeDasharray="3 3" stroke="#27272a" />

          <XAxis
            dataKey="time"
            stroke="#71717a"
            tick={{ fill: '#71717a', fontSize: 11 }}
            tickLine={{ stroke: '#27272a' }}
            interval={Math.floor(displayData.length / 12)}
            angle={-15}
            textAnchor="end"
            height={60}
          />

          <YAxis
            stroke="#71717a"
            tick={{ fill: '#71717a', fontSize: 12 }}
            tickLine={{ stroke: '#27272a' }}
            domain={calculateYDomain()}
            tickFormatter={(value) => `${value.toFixed(1)}%`}
            width={60}
          />

          <Tooltip content={<CustomTooltip />} />

          <ReferenceLine
            y={0}
            stroke="#6b7280"
            strokeDasharray="5 5"
            strokeWidth={2}
            label={{
              value: 'Break Even',
              fill: '#9ca3af',
              fontSize: 11,
              position: 'right',
            }}
          />

          {traders.map((trader, index) => (
            <Line
              key={trader.trader_id}
              type="monotone"
              dataKey={`${trader.trader_id}_pnl_pct`}
              stroke={getTraderColor(trader.trader_id)}
              strokeWidth={3}
              dot={displayData.length < 50 ? { fill: getTraderColor(trader.trader_id), r: 3 } : false}
              activeDot={{ r: 6, fill: getTraderColor(trader.trader_id), stroke: '#fff', strokeWidth: 2 }}
              name={trader.trader_name}
              connectNulls
            />
          ))}

          <Legend
            wrapperStyle={{ paddingTop: '20px' }}
            iconType="line"
            formatter={(value, entry: any) => {
              const traderId = traders.find((t) => value === t.trader_name)?.trader_id;
              const trader = traders.find((t) => t.trader_id === traderId);
              return (
                <span style={{ color: entry.color, fontWeight: 600, fontSize: '14px' }}>
                  {trader?.trader_name} ({trader?.ai_model.toUpperCase()})
                </span>
              );
            }}
          />
        </LineChart>
      </ResponsiveContainer>

      {/* Stats */}
      <div className="mt-6 grid grid-cols-4 gap-4 pt-4 border-t border-gray-800">
        <div>
          <div className="text-xs text-gray-400">对比模式</div>
          <div className="text-sm font-semibold">收益率百分比 (%)</div>
        </div>
        <div>
          <div className="text-xs text-gray-400">数据点数</div>
          <div className="text-sm font-semibold mono">{combinedData.length} 个周期</div>
        </div>
        <div>
          <div className="text-xs text-gray-400">当前差距</div>
          <div className="text-sm font-semibold mono">{currentGap.toFixed(2)}%</div>
        </div>
        <div>
          <div className="text-xs text-gray-400">显示范围</div>
          <div className="text-sm font-semibold mono">
            {combinedData.length > MAX_DISPLAY_POINTS
              ? `最近 ${MAX_DISPLAY_POINTS}`
              : '全部数据'}
          </div>
        </div>
      </div>
    </div>
  );
}
