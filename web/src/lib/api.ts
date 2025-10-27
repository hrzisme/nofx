import type {
  SystemStatus,
  AccountInfo,
  Position,
  DecisionRecord,
  Statistics,
} from '../types';

const API_BASE = '/api';

export const api = {
  // 获取系统状态
  async getStatus(): Promise<SystemStatus> {
    const res = await fetch(`${API_BASE}/status`);
    if (!res.ok) throw new Error('获取系统状态失败');
    return res.json();
  },

  // 获取账户信息
  async getAccount(): Promise<AccountInfo> {
    const res = await fetch(`${API_BASE}/account`);
    if (!res.ok) throw new Error('获取账户信息失败');
    return res.json();
  },

  // 获取持仓列表
  async getPositions(): Promise<Position[]> {
    const res = await fetch(`${API_BASE}/positions`);
    if (!res.ok) throw new Error('获取持仓列表失败');
    return res.json();
  },

  // 获取决策日志
  async getDecisions(): Promise<DecisionRecord[]> {
    const res = await fetch(`${API_BASE}/decisions`);
    if (!res.ok) throw new Error('获取决策日志失败');
    return res.json();
  },

  // 获取最新决策
  async getLatestDecisions(): Promise<DecisionRecord[]> {
    const res = await fetch(`${API_BASE}/decisions/latest`);
    if (!res.ok) throw new Error('获取最新决策失败');
    return res.json();
  },

  // 获取统计信息
  async getStatistics(): Promise<Statistics> {
    const res = await fetch(`${API_BASE}/statistics`);
    if (!res.ok) throw new Error('获取统计信息失败');
    return res.json();
  },
};
