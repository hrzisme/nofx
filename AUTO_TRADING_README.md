# 🤖 AI驱动的币安合约自动交易系统

完全基于AI决策的加密货币自动交易系统，使用CCXT进行币安合约交易。

## ✨ 核心功能

### 1. 智能币种筛选
- 从API获取可交易币种池
- AI批量分析多个币种的市场数据
- 综合技术指标（RSI、MACD、EMA、成交量、资金费率）
- 优先级评分系统，自动筛选最佳交易机会

### 2. 全自动交易执行
- 自动开仓（做多/做空）
- 自动设置止损止盈
- 自动持仓管理和平仓
- 支持币安合约交易

### 3. 风险管理
- 可配置的单笔风险百分比
- 最大持仓数量控制
- 最大日亏损和回撤保护
- 自动风险控制暂停交易

### 4. AI决策
- 支持阿里云Qwen和DeepSeek AI
- 实时市场数据分析
- 综合技术面分析
- 自动生成交易信号和价格目标

## 📁 文件说明

```
├── main_auto.go        # 主程序入口
├── auto_trader.go      # 自动交易核心逻辑
├── ai_scanner.go       # AI币种扫描器
├── ccxt_trading.go     # CCXT交易接口
├── coin_pool.go        # 币种池API
├── market_data.go      # 市场数据获取 (原有)
├── ai_signal.go        # AI信号生成 (原有)
└── binance_account.go  # 账户管理 (原有)
```

## 🚀 快速开始

### 1. 安装依赖

```bash
go get github.com/ccxt/ccxt
```

### 2. 配置API密钥

编辑 `main_auto.go` 文件，修改配置：

```go
config := AutoTraderConfig{
    // 币安API密钥（必填）
    BinanceAPIKey:    "YOUR_BINANCE_API_KEY",
    BinanceSecretKey: "YOUR_BINANCE_SECRET_KEY",

    // 币种池API（已配置）
    CoinPoolAPIURL: "http://43.128.34.180:30006/api/ai500/list?auth=admin123sadasd3r323",

    // AI选择
    UseQwen:     true, // true=Qwen, false=DeepSeek
    DeepSeekKey: "your-deepseek-key",
    QwenKey:     "your-qwen-key",

    // 交易参数
    RiskPercentPerTrade: 2.0, // 每笔交易风险2%
    MaxPositions:        3,   // 最多3个仓位
    DefaultLeverage:     10,  // 10倍杠杆

    // 扫描设置
    ScanInterval:  5 * time.Minute, // 5分钟扫描一次
    TopN:          5,                // 选择前5个机会
    MinConfidence: 70.0,             // 最小信心度70%
    MinPriority:   65,               // 最小优先级65分
    MinRiskReward: 2.0,              // 最小风险回报比1:2
}
```

### 3. 运行

```bash
# 运行自动交易系统
go run main_auto.go auto_trader.go ai_scanner.go ccxt_trading.go coin_pool.go

# 或者编译后运行
go build -o nofx-auto *.go
./nofx-auto
```

## ⚙️ 配置参数详解

### 交易配置

| 参数 | 说明 | 推荐值 |
|------|------|--------|
| `RiskPercentPerTrade` | 单笔交易风险占账户百分比 | 1-3% |
| `MaxPositions` | 最大同时持仓数量 | 3-5 |
| `DefaultLeverage` | 默认杠杆倍数 | 5-10x |

### 扫描配置

| 参数 | 说明 | 推荐值 |
|------|------|--------|
| `ScanInterval` | 扫描间隔 | 5-10分钟 |
| `TopN` | 选择前N个最佳机会 | 3-5 |
| `MinConfidence` | 最小AI信心度 | 65-75% |
| `MinPriority` | 最小优先级评分 | 60-70 |
| `MinRiskReward` | 最小风险回报比 | 1.5-2.5 |

### 风险控制

| 参数 | 说明 | 推荐值 |
|------|------|--------|
| `MaxDailyLoss` | 最大日亏损百分比 | 5-10% |
| `MaxDrawdown` | 最大回撤百分比 | 10-15% |
| `StopTradingTime` | 触发风控后暂停时长 | 30-60分钟 |

## 📊 工作流程

```
1. 从API获取可交易币种列表
   ↓
2. AI并发扫描所有币种
   - 获取市场数据（K线、指标、资金费率等）
   - AI分析生成交易信号
   - 计算优先级评分
   ↓
3. 筛选高质量机会
   - 信心度过滤
   - 优先级排序
   - 选择TopN
   ↓
4. 执行交易
   - 计算仓位大小
   - 设置杠杆
   - 开仓
   - 设置止损止盈
   ↓
5. 持仓管理
   - AI持续监控
   - 检查止损止盈
   - 自动平仓
   ↓
6. 循环 (按扫描间隔重复)
```

## 🎯 AI优先级评分系统

总分100分，由以下部分组成：

- **信心度** (0-40分): AI对信号的信心程度
- **风险回报比** (0-25分): 止盈/止损比率
- **技术指标** (0-25分):
  - RSI超买超卖确认
  - MACD趋势确认
  - EMA趋势确认
  - 资金费率确认
- **成交量** (0-10分): 放量确认

## 💡 使用建议

### 新手模式
```go
RiskPercentPerTrade: 1.0,  // 1%风险
MaxPositions:        2,    // 最多2仓位
DefaultLeverage:     5,    // 5倍杠杆
MinConfidence:       75.0, // 高信心度
MinRiskReward:       2.5,  // 高回报比
```

### 激进模式
```go
RiskPercentPerTrade: 3.0,  // 3%风险
MaxPositions:        5,    // 最多5仓位
DefaultLeverage:     15,   // 15倍杠杆
MinConfidence:       65.0, // 中等信心度
MinRiskReward:       1.5,  // 较低回报比
```

## ⚠️ 风险提示

1. **加密货币交易风险极高**，可能导致本金全部损失
2. **杠杆交易放大风险**，请谨慎使用
3. **建议先在测试网测试**
4. **从小资金开始**，熟悉系统后再增加投入
5. **AI决策不保证盈利**，市场风险无法完全规避
6. **定期监控系统运行**，防止异常情况

## 🔧 高级功能

### 自定义扫描配置

```go
SetScanConfig(ScanConfig{
    MinConfidence:      70.0,
    MaxConcurrent:      10,
    Timeout:            60 * time.Second,
    MinPriority:        65,
    EnableLong:         true,
    EnableShort:        true,
    MinRiskRewardRatio: 2.0,
})
```

### 修改币种池API

```go
SetCoinPoolAPI("your-api-url")
```

## 📝 日志说明

系统会输出详细的运行日志：

```
🔍 开始扫描 50 个币种...
✓ 扫描完成，耗时 12.3s，找到 8 个交易机会

【机会 #1】BTCUSDT
  信号: 开多 🟢
  信心度: 85.5%  |  优先级: 88/100
  当前价: 42350.5000 USDT
  入场价: 42400.0000 USDT
  止损价: 41800.0000 USDT  (风险: 1.42%)
  止盈价: 43600.0000 USDT  (收益: 2.83%)
  风险回报比: 1:2.00
  分析: MACD金叉，RSI处于健康区间，价格突破EMA20...

💫 正在执行交易: BTCUSDT 开多 🟢
   可用余额: 1000.00 USDT
   开仓数量: 0.0471 (杠杆: 10x)
✓ 开多仓成功: BTCUSDT 数量: 0.0471
  订单ID: 12345678
  止损价设置: 41800.0000
  止盈价设置: 43600.0000
   ✓ 交易执行成功
```

## 🆘 常见问题

### Q: 如何停止运行？
A: 按 `Ctrl+C` 优雅退出。

### Q: 可以修改扫描间隔吗？
A: 可以，修改 `ScanInterval` 参数。建议不低于3分钟。

### Q: 如何只做多或只做空？
A: 修改 `ScanConfig` 中的 `EnableLong` 和 `EnableShort`。

### Q: AI信号不准确怎么办？
A: 提高 `MinConfidence` 和 `MinPriority` 阈值，减少交易频率但提高质量。

## 📞 技术支持

如遇问题，请检查：
1. API密钥是否正确
2. 网络连接是否正常
3. 币安账户是否有足够余额
4. 是否开通了合约交易权限

## 📄 许可证

本项目仅供学习研究使用，使用本系统进行实盘交易的任何盈亏由使用者自行承担。
