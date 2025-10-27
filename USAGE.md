# 使用指南

## 🎯 快速开始

### 1. 配置 API 密钥

编辑 `main_auto.go` 第 22-23 行：

```go
BinanceAPIKey:    "YOUR_BINANCE_API_KEY",
BinanceSecretKey: "YOUR_BINANCE_SECRET_KEY",
```

### 2. 运行

```bash
./nofx-auto
```

## 📁 项目结构（模块化设计）

```
nofx/
├── main_auto.go           # 主程序入口
├── trader/                # 交易模块
│   ├── auto_trader.go    # 自动交易核心逻辑
│   └── binance_futures.go# 币安合约交易接口
├── scanner/               # 扫描模块
│   └── ai_scanner.go     # AI币种扫描器
├── pool/                  # 币种池模块
│   └── coin_pool.go      # 币种池API
├── market/                # 市场模块
│   ├── ai_signal.go      # AI信号生成
│   └── market_data.go    # 市场数据获取
└── *.md                   # 文档文件
```

## ⚙️ 配置参数

在 `main_auto.go` 中修改：

```go
RiskPercentPerTrade: 2.0,   // 每笔风险2%
MaxPositions:        3,     // 最多3个仓位
DefaultLeverage:     10,    // 10倍杠杆
ScanInterval:        5 * time.Minute,  // 5分钟扫描
MinConfidence:       70.0,  // 最小信心度70%
MinPriority:         65,    // 最小优先级65分
MinRiskReward:       2.0,   // 风险回报比1:2
```

## 📦 模块说明

### trader 包
- **auto_trader.go**: 自动交易主循环、持仓管理、风险控制
- **binance_futures.go**: 币安合约API封装（开仓、平仓、杠杆）

### scanner 包
- **ai_scanner.go**: 并发扫描市场、AI信号筛选、优先级评分

### pool 包
- **coin_pool.go**: 从外部API获取可交易币种列表

### market 包  
- **ai_signal.go**: AI交易信号生成（DeepSeek/Qwen）
- **market_data.go**: K线数据、技术指标计算

## 📊 代码统计

- 总代码: ~3500 行
- 模块: 4 个包
- 文件: 7 个核心文件

详细文档: [AUTO_TRADING_README.md](AUTO_TRADING_README.md)
