# 🤖 NOFX - AI驱动的币安合约自动交易系统

[![Go Version](https://img.shields.io/badge/Go-1.21+-00ADD8?style=flat&logo=go)](https://golang.org/)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

一个基于 **DeepSeek/Qwen AI** 驱动的币安合约自动交易系统，具备完整的市场数据分析、技术指标计算、AI决策引擎和风险控制机制。

> ⚠️ **风险提示**：本系统为实验性项目，AI自动交易存在重大风险，强烈建议仅用于学习研究或小额资金测试！

---

## ✨ 核心特性

### 🧠 AI全权决策
- **DeepSeek/Qwen AI** 完全自主决策
- AI自动选择交易币种、方向（多/空）
- AI自主决定杠杆倍数（1-20倍）
- AI自主确定仓位大小、止损、止盈价格
- **Chain of Thought (CoT)** 思维链推理记录

### 📊 市场数据分析
- **3分钟K线**：实时价格、EMA20、MACD、RSI(7)
- **4小时K线**：长期趋势、EMA20/50、ATR、RSI(14)
- **持仓量与资金费率**：市场情绪判断
- **批量分析**：一次性分析多个候选币种

### 🎯 智能币种池
- 自动从API获取热门币种
- **主流币种默认加入**：BTC、ETH、SOL、BNB、XRP、DOGE
- 根据账户风险动态调整分析数量
- 持仓优先分析，确保现有仓位健康

### 📝 完整决策日志
- 每次AI决策完整记录（JSON格式）
- 包含：思维链、决策列表、账户快照、持仓快照、执行结果
- 便于复盘分析和策略优化
- 日志文件按时间戳自动命名

### 🛡️ 风险控制
- 最大日亏损限制（可配置）
- 最大回撤限制（可配置）
- 触发风控自动暂停交易
- 保证金使用率监控（>70%停止开新仓）
- 单笔仓位控制（建议5-15%账户净值）

---

## 🏗️ 技术架构

```
nofx/
├── main.go                          # 程序入口
├── trader/
│   ├── auto_trader.go              # 自动交易主控逻辑
│   └── binance_futures.go          # 币安合约API封装
├── market/
│   ├── market_data.go              # 市场数据获取
│   ├── ai_decision_engine.go       # AI决策引擎（核心）
│   └── ai_signal.go                # AI API调用封装
├── scanner/
│   └── coin_scanner.go             # 币种扫描器
├── pool/
│   └── coin_pool.go                # 币种池管理
├── logger/
│   └── decision_logger.go          # 决策日志系统
└── decision_logs/                  # 决策记录存储目录
    └── decision_YYYYMMDD_HHMMSS_cycleN.json
```

### 依赖库

- **github.com/adshao/go-binance/v2** - 币安API客户端
- **github.com/markcheno/go-talib** - 技术指标计算（TA-Lib）

---

## 🚀 快速开始

### 1. 环境要求

- **Go 1.21+**
- **TA-Lib** 库（技术指标计算）

#### 安装 TA-Lib

**macOS:**
```bash
brew install ta-lib
```

**Ubuntu/Debian:**
```bash
sudo apt-get install libta-lib0-dev
```

**其他系统**：参考 [TA-Lib官方文档](https://github.com/markcheno/go-talib)

### 2. 克隆项目

```bash
git clone <repository-url>
cd nofx
```

### 3. 安装依赖

```bash
go mod download
```

### 4. 配置系统

编辑 `main.go` 中的配置：

```go
config := trader.AutoTraderConfig{
    // 币安API密钥（必填）
    BinanceAPIKey:    "YOUR_BINANCE_API_KEY",
    BinanceSecretKey: "YOUR_BINANCE_SECRET_KEY",

    // 币种池API（可选）
    CoinPoolAPIURL: "http://your-pool-api/list",

    // AI配置（必填）
    UseQwen:     false,                          // false=DeepSeek, true=Qwen
    DeepSeekKey: "sk-xxxxx",                     // DeepSeek API密钥
    QwenKey:     "sk-xxxxx",                     // 阿里云Qwen API密钥

    // 交易配置
    ScanInterval: 3 * time.Minute,               // 决策周期：每3分钟

    // 风险控制（仅作提示，AI可自主决定）
    MaxDailyLoss:    5.0,                        // 最大日亏损 5%
    MaxDrawdown:     10.0,                       // 最大回撤 10%
    StopTradingTime: 30 * time.Minute,           // 风控暂停时间
}
```

### 5. 运行系统

```bash
go build -o nofx-auto
./nofx-auto
```

### 6. 停止系统

按 `Ctrl+C` 优雅停止

---

## 📖 使用说明

### AI决策流程

每个决策周期（默认3分钟），系统按以下流程运行：

```
1. 获取账户状态
   ├─ 账户净值、可用余额
   ├─ 持仓数量、总盈亏
   └─ 保证金使用率

2. 分析现有持仓（如果有）
   ├─ 获取每个持仓的市场数据
   ├─ 计算技术指标（RSI、MACD、EMA）
   └─ AI判断是否需要平仓/止盈/止损

3. 评估新机会
   ├─ 从币种池获取候选币种
   ├─ 主流币种（BTC/ETH等）默认加入
   ├─ 根据保证金使用率动态调整分析数量
   └─ 批量获取市场数据和技术指标

4. AI综合决策
   ├─ Chain of Thought 思维链分析
   ├─ 输出具体决策（平仓/开仓/持有/观望）
   └─ 包含杠杆、仓位、止损、止盈等参数

5. 执行交易
   ├─ 按顺序执行：先平仓，再开仓
   ├─ 精度自动适配（LOT_SIZE）
   └─ 平仓后自动取消所有挂单

6. 记录日志
   └─ 保存完整决策记录到 decision_logs/
```

### AI决策示例

**决策JSON格式：**
```json
[
  {
    "symbol": "BTCUSDT",
    "action": "close_long",
    "reasoning": "RSI超买且MACD死叉，止盈离场"
  },
  {
    "symbol": "ETHUSDT",
    "action": "open_short",
    "leverage": 5,
    "position_size_usd": 100.0,
    "stop_loss": 3500.0,
    "take_profit": 3200.0,
    "reasoning": "EMA死叉+持仓量下降，空头信号明确"
  }
]
```

**决策类型：**
- `open_long` - 开多仓
- `open_short` - 开空仓
- `close_long` - 平多仓
- `close_short` - 平空仓
- `hold` - 继续持有
- `wait` - 观望等待

---

## 📊 决策日志

每次AI决策都会生成详细的JSON日志，存储在 `decision_logs/` 目录：

### 日志文件命名
```
decision_20251027_220436_cycle1.json
        ↓        ↓        ↓
      日期      时间     周期编号
```

### 日志内容结构

```json
{
  "timestamp": "2025-10-27T22:04:36+08:00",
  "cycle_number": 1,
  "cot_trace": "第一步：现有持仓分析\n- 持仓#1 BTCUSDT LONG...",
  "decision_json": "[{\"symbol\":\"BTCUSDT\",\"action\":\"hold\"...}]",
  "account_state": {
    "total_balance": 997.11,
    "available_balance": 958.77,
    "total_unrealized_profit": 38.34,
    "position_count": 4,
    "margin_used_pct": 3.85
  },
  "positions": [
    {
      "symbol": "BTCUSDT",
      "side": "long",
      "position_amt": 0.001,
      "entry_price": 68500.0,
      "mark_price": 69200.0,
      "unrealized_profit": 0.70,
      "leverage": 10,
      "liquidation_price": 62000.0
    }
  ],
  "candidate_coins": ["BTCUSDT", "ETHUSDT", ...],
  "decisions": [
    {
      "action": "close_long",
      "symbol": "BTCUSDT",
      "quantity": 0.001,
      "price": 69200.0,
      "order_id": 123456789,
      "timestamp": "2025-10-27T22:04:40+08:00",
      "success": true,
      "error": ""
    }
  ],
  "execution_log": [
    "✓ 执行决策 #1: close_long BTCUSDT",
    "✓ 平多仓成功: BTCUSDT 数量: 0.001"
  ],
  "success": true,
  "error_message": ""
}
```

---

## 🎛️ 高级配置

### 调整AI模型

**使用DeepSeek（默认）：**
```go
config.UseQwen = false
config.DeepSeekKey = "sk-xxxxx"
```

**使用阿里云Qwen：**
```go
config.UseQwen = true
config.QwenKey = "sk-xxxxx"
```

### 调整决策周期

```go
config.ScanInterval = 5 * time.Minute  // 改为每5分钟
```

### 自定义风险参数

```go
config.MaxDailyLoss = 3.0              // 日亏损限制 3%
config.MaxDrawdown = 8.0               // 最大回撤 8%
config.StopTradingTime = 1 * time.Hour // 风控后暂停1小时
```

---

## 🔧 常见问题

### 1. Precision错误

**问题**：`Precision is over the maximum defined for this asset`

**解决**：系统已自动处理精度，从Binance的LOT_SIZE过滤器获取。如遇此错误请检查网络或API访问。

### 2. AI API超时

**问题**：`context deadline exceeded`

**解决**：系统已将超时时间设置为120秒。如仍超时，检查网络或AI API密钥。

### 3. JSON解析失败

**问题**：`invalid character '`' after top-level value`

**解决**：系统使用bracket matching算法严格提取JSON，不受markdown干扰。

### 4. 挂单未取消

**解决**：系统在平仓后会自动调用 `CancelAllOrders()` 取消所有挂单。

---

## 📈 性能优化建议

1. **合理设置决策周期**：建议不低于3分钟，避免过度交易
2. **控制候选币种数量**：系统会根据保证金使用率自动调整
3. **定期清理日志**：使用 `DecisionLogger.CleanOldRecords(30)` 清理30天前的日志
4. **监控账户余额**：建议小额资金测试，避免大额亏损

---

## ⚠️ 重要风险提示

1. **本系统仅供学习研究使用，不构成任何投资建议**
2. **加密货币交易风险极高，AI决策不保证盈利**
3. **合约交易使用杠杆，亏损可能超过本金**
4. **建议仅使用可承受损失的资金进行测试**
5. **定期检查系统运行状态和账户余额**
6. **市场极端行情下可能出现爆仓风险**

---

## 📄 开源协议

MIT License

---

## 🤝 贡献

欢迎提交Issue和Pull Request！

---

## 📬 联系方式

如有问题或建议，请提交GitHub Issue。

---

**最后更新**: 2025-10-27
