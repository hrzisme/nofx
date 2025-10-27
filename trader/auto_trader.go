package trader

import (
	"fmt"
	"log"
	"nofx/market"
	"nofx/pool"
	"nofx/scanner"
	"strings"
	"time"
)

// AutoTraderConfig 自动交易配置
type AutoTraderConfig struct {
	// API配置
	BinanceAPIKey    string
	BinanceSecretKey string
	CoinPoolAPIURL   string

	// AI配置
	UseQwen     bool
	DeepSeekKey string
	QwenKey     string

	// 交易配置
	RiskPercentPerTrade float64 // 每笔交易风险百分比
	MaxPositions        int     // 最大持仓数量
	DefaultLeverage     int     // 默认杠杆倍数

	// 扫描配置
	ScanInterval  time.Duration // 扫描间隔
	TopN          int           // 选择前N个机会
	MinConfidence float64       // 最小信心度
	MinPriority   int           // 最小优先级
	MinRiskReward float64       // 最小风险回报比

	// 风险控制
	MaxDailyLoss    float64       // 最大日亏损百分比
	MaxDrawdown     float64       // 最大回撤百分比
	StopTradingTime time.Duration // 亏损后停止交易时长
}

// AutoTrader 自动交易器
type AutoTrader struct {
	config         AutoTraderConfig
	trader         *FuturesTrader
	initialBalance float64
	dailyPnL       float64
	lastResetTime  time.Time
	stopUntil      time.Time
	isRunning      bool
}

// NewAutoTrader 创建自动交易器
func NewAutoTrader(config AutoTraderConfig) (*AutoTrader, error) {
	// 初始化AI
	if config.UseQwen {
		market.SetQwenAPIKey(config.QwenKey, "")
		log.Println("🤖 使用阿里云Qwen AI")
	} else {
		market.SetDeepSeekAPIKey(config.DeepSeekKey)
		log.Println("🤖 使用DeepSeek AI")
	}

	// 初始化币种池API
	if config.CoinPoolAPIURL != "" {
		pool.SetCoinPoolAPI(config.CoinPoolAPIURL)
	}

	// 初始化币安合约交易器
	trader := NewFuturesTrader(config.BinanceAPIKey, config.BinanceSecretKey)

	// 获取初始余额
	balance, err := trader.GetBalance()
	if err != nil {
		return nil, fmt.Errorf("获取余额失败: %w", err)
	}

	initialBalance := 0.0
	if avail, ok := balance["availableBalance"].(float64); ok {
		initialBalance = avail
	}

	// 设置扫描配置
	scanner.SetScanConfig(scanner.ScanConfig{
		MinConfidence:      config.MinConfidence,
		MaxConcurrent:      10,
		Timeout:            60 * time.Second,
		MinPriority:        config.MinPriority,
		EnableLong:         true,
		EnableShort:        true,
		MinRiskRewardRatio: config.MinRiskReward,
	})

	return &AutoTrader{
		config:         config,
		trader:         trader,
		initialBalance: initialBalance,
		lastResetTime:  time.Now(),
		isRunning:      false,
	}, nil
}

// Run 运行自动交易主循环
func (at *AutoTrader) Run() error {
	at.isRunning = true
	log.Println("🚀 自动交易系统启动")
	log.Printf("💰 初始余额: %.2f USDT", at.initialBalance)
	log.Printf("⚙️  配置: 风险 %.1f%%, 杠杆 %dx, 最大持仓 %d",
		at.config.RiskPercentPerTrade, at.config.DefaultLeverage, at.config.MaxPositions)

	ticker := time.NewTicker(at.config.ScanInterval)
	defer ticker.Stop()

	// 首次立即执行
	if err := at.runCycle(); err != nil {
		log.Printf("❌ 执行失败: %v", err)
	}

	for at.isRunning {
		select {
		case <-ticker.C:
			if err := at.runCycle(); err != nil {
				log.Printf("❌ 执行失败: %v", err)
			}
		}
	}

	return nil
}

// Stop 停止自动交易
func (at *AutoTrader) Stop() {
	at.isRunning = false
	log.Println("⏹ 自动交易系统停止")
}

// runCycle 运行一个交易周期
func (at *AutoTrader) runCycle() error {
	log.Printf("\n" + strings.Repeat("=", 60))
	log.Printf("⏰ %s - 开始新的交易周期", time.Now().Format("2006-01-02 15:04:05"))
	log.Printf(strings.Repeat("=", 60))

	// 1. 检查是否需要停止交易
	if time.Now().Before(at.stopUntil) {
		remaining := at.stopUntil.Sub(time.Now())
		log.Printf("⏸ 风险控制：暂停交易中，剩余 %.0f 分钟", remaining.Minutes())
		return nil
	}

	// 2. 重置日盈亏（每天重置）
	if time.Since(at.lastResetTime) > 24*time.Hour {
		at.dailyPnL = 0
		at.lastResetTime = time.Now()
		log.Println("📅 日盈亏已重置")
	}

	// 3. 检查现有持仓
	if err := at.managePositions(); err != nil {
		log.Printf("⚠ 管理持仓失败: %v", err)
	}

	// 4. 获取当前持仓数量
	positions, err := at.trader.GetPositions()
	if err != nil {
		return fmt.Errorf("获取持仓失败: %w", err)
	}

	currentPositions := len(positions)
	log.Printf("📊 当前持仓: %d/%d", currentPositions, at.config.MaxPositions)

	// 5. 如果已达到最大持仓，不再开新仓
	if currentPositions >= at.config.MaxPositions {
		log.Println("⚠ 已达最大持仓数量，跳过扫描")
		return nil
	}

	// 6. 获取币种池
	symbols, err := pool.GetAvailableCoins()
	if err != nil {
		return fmt.Errorf("获取币种池失败: %w", err)
	}

	log.Printf("🎯 币种池: %d 个币种", len(symbols))

	// 7. 扫描市场机会
	opportunities, err := scanner.ScanMarket(symbols)
	if err != nil {
		return fmt.Errorf("扫描市场失败: %w", err)
	}

	if len(opportunities) == 0 {
		log.Println("😴 未找到合适的交易机会")
		return nil
	}

	// 8. 选择前N个机会
	topOpps := scanner.FilterTopN(opportunities, at.config.TopN)
	log.Printf("\n📈 找到 %d 个高质量机会:", len(topOpps))
	for i, opp := range topOpps {
		scanner.PrintOpportunity(opp, i)
	}

	// 9. 执行交易
	availableSlots := at.config.MaxPositions - currentPositions
	for i, opp := range topOpps {
		if i >= availableSlots {
			break
		}

		if err := at.executeOpportunity(opp); err != nil {
			log.Printf("❌ 执行交易失败 %s: %v", opp.Symbol, err)
		} else {
			// 成功开仓后短暂延迟
			time.Sleep(2 * time.Second)
		}
	}

	return nil
}

// managePositions 管理现有持仓
func (at *AutoTrader) managePositions() error {
	positions, err := at.trader.GetPositions()
	if err != nil {
		return err
	}

	if len(positions) == 0 {
		return nil
	}

	log.Printf("\n💼 检查 %d 个持仓...", len(positions))

	for _, pos := range positions {
		symbol := pos["symbol"].(string)
		side := pos["side"].(string)
		entryPrice := pos["entryPrice"].(float64)
		markPrice := pos["markPrice"].(float64)
		unrealizedPnl := pos["unRealizedProfit"].(float64)
		pnlPercent := 0.0
		if side == "long" {
			pnlPercent = ((markPrice - entryPrice) / entryPrice) * 100
		} else {
			pnlPercent = ((entryPrice - markPrice) / entryPrice) * 100
		}

		log.Printf("  %s %s: 入场 %.4f, 当前 %.4f, 盈亏 %.2f USDT (%.2f%%)",
			symbol, side, entryPrice, markPrice, unrealizedPnl, pnlPercent)

		// 检查是否需要平仓
		shouldClose, reason, err := at.shouldClosePosition(symbol, side, entryPrice, markPrice)
		if err != nil {
			log.Printf("    ⚠ 检查平仓失败: %v", err)
			continue
		}

		if shouldClose {
			log.Printf("    📌 平仓原因: %s", reason)
			if err := at.closePosition(symbol, side); err != nil {
				log.Printf("    ❌ 平仓失败: %v", err)
			}
		}
	}

	return nil
}

// shouldClosePosition 判断是否应该平仓
func (at *AutoTrader) shouldClosePosition(symbol, side string, entryPrice, markPrice float64) (bool, string, error) {
	// 获取市场数据和AI信号
	_, err := market.GetMarketData(symbol)
	if err != nil {
		return false, "", err
	}

	signal, err := market.GetAITradingSignal(symbol)
	if err != nil {
		return false, "", err
	}

	// 计算盈亏百分比
	var pnlPercent float64
	if side == "long" {
		pnlPercent = ((markPrice - entryPrice) / entryPrice) * 100
	} else {
		pnlPercent = ((entryPrice - markPrice) / entryPrice) * 100
	}

	// 1. AI建议平仓
	if side == "long" && (signal.Signal == market.SignalCloseLong || signal.Signal == market.SignalOpenShort) {
		if signal.Confidence > 70 {
			return true, fmt.Sprintf("AI强烈建议平多仓 (信心度: %.1f%%)", signal.Confidence), nil
		}
	}
	if side == "short" && (signal.Signal == market.SignalCloseShort || signal.Signal == market.SignalOpenLong) {
		if signal.Confidence > 70 {
			return true, fmt.Sprintf("AI强烈建议平空仓 (信心度: %.1f%%)", signal.Confidence), nil
		}
	}

	// 2. 止损检查
	if side == "long" && markPrice <= signal.StopLoss {
		return true, fmt.Sprintf("触发止损 (%.4f <= %.4f)", markPrice, signal.StopLoss), nil
	}
	if side == "short" && markPrice >= signal.StopLoss {
		return true, fmt.Sprintf("触发止损 (%.4f >= %.4f)", markPrice, signal.StopLoss), nil
	}

	// 3. 止盈检查
	if side == "long" && markPrice >= signal.TakeProfit {
		return true, fmt.Sprintf("触发止盈 (%.4f >= %.4f)", markPrice, signal.TakeProfit), nil
	}
	if side == "short" && markPrice <= signal.TakeProfit {
		return true, fmt.Sprintf("触发止盈 (%.4f <= %.4f)", markPrice, signal.TakeProfit), nil
	}

	// 4. 大额亏损强制平仓
	if pnlPercent < -5.0 {
		return true, fmt.Sprintf("大额亏损强制平仓 (%.2f%%)", pnlPercent), nil
	}

	return false, "", nil
}

// closePosition 平仓
func (at *AutoTrader) closePosition(symbol, side string) error {
	log.Printf("    🔄 正在平仓 %s %s...", symbol, side)

	var err error
	if side == "long" {
		_, err = at.trader.CloseLong(symbol, 0) // 0 = 全部平仓
	} else {
		_, err = at.trader.CloseShort(symbol, 0)
	}

	if err != nil {
		return err
	}

	log.Printf("    ✓ 平仓成功")
	return nil
}

// executeOpportunity 执行交易机会
func (at *AutoTrader) executeOpportunity(opp *scanner.TradingOpportunity) error {
	log.Printf("\n💫 正在执行交易: %s %s", opp.Symbol, scanner.GetSignalText(opp.Signal))

	// 1. 获取当前余额
	balance, err := at.trader.GetBalance()
	if err != nil {
		return err
	}

	freeBalance := 0.0
	if avail, ok := balance["availableBalance"].(float64); ok {
		freeBalance = avail
	}

	log.Printf("   可用余额: %.2f USDT", freeBalance)

	// 2. 计算仓位大小
	quantity := at.trader.CalculatePositionSize(
		freeBalance,
		at.config.RiskPercentPerTrade,
		opp.CurrentPrice,
		at.config.DefaultLeverage,
	)

	log.Printf("   开仓数量: %.4f (杠杆: %dx)", quantity, at.config.DefaultLeverage)

	// 3. 执行开仓
	var order map[string]interface{}
	if opp.Signal == market.SignalOpenLong {
		order, err = at.trader.OpenLong(opp.Symbol, quantity, at.config.DefaultLeverage)
	} else if opp.Signal == market.SignalOpenShort {
		order, err = at.trader.OpenShort(opp.Symbol, quantity, at.config.DefaultLeverage)
	}

	if err != nil {
		return err
	}

	// 4. 设置止损止盈
	positionSide := "LONG"
	if opp.Signal == market.SignalOpenShort {
		positionSide = "SHORT"
	}

	if err := at.trader.SetStopLoss(opp.Symbol, positionSide, quantity, opp.StopLoss); err != nil {
		log.Printf("   ⚠ 设置止损失败: %v", err)
	}

	if err := at.trader.SetTakeProfit(opp.Symbol, positionSide, quantity, opp.TakeProfit); err != nil {
		log.Printf("   ⚠ 设置止盈失败: %v", err)
	}

	log.Printf("   ✓ 交易执行成功，订单ID: %v", order["orderId"])
	return nil
}
