package main

import (
	"fmt"
	"log"
	"nofx/trader"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

func main() {
	fmt.Println("╔════════════════════════════════════════════════════════════╗")
	fmt.Println("║       🤖 AI驱动的币安合约自动交易系统                        ║")
	fmt.Println("╚════════════════════════════════════════════════════════════╝")
	fmt.Println()

	// ========== 配置区 ==========
	config := trader.AutoTraderConfig{
		// API密钥配置
		BinanceAPIKey:    "YOUR_BINANCE_API_KEY",
		BinanceSecretKey: "YOUR_BINANCE_SECRET_KEY",
		CoinPoolAPIURL:   "http://43.128.34.180:30006/api/ai500/list?auth=admin123sadasd3r323",

		// AI配置
		UseQwen:     true, // true=使用Qwen, false=使用DeepSeek
		DeepSeekKey: "sk-44ac4e74ef184461800fccd15685ad30",
		QwenKey:     "sk-c02cc4678f094a72b5513e92889eda04",

		// 交易配置
		RiskPercentPerTrade: 2.0, // 每笔交易风险2%
		MaxPositions:        3,   // 最多同时持有3个仓位
		DefaultLeverage:     10,  // 默认10倍杠杆

		// 扫描配置
		ScanInterval:  5 * time.Minute, // 每5分钟扫描一次
		TopN:          5,               // 选择前5个最佳机会
		MinConfidence: 70.0,            // 最小信心度70%
		MinPriority:   65,              // 最小优先级65分
		MinRiskReward: 2.0,             // 最小风险回报比1:2

		// 风险控制
		MaxDailyLoss:    5.0,              // 最大日亏损5%
		MaxDrawdown:     10.0,             // 最大回撤10%
		StopTradingTime: 30 * time.Minute, // 触发风控后暂停30分钟
	}
	// =============================

	// 验证API密钥配置
	if config.BinanceAPIKey == "YOUR_BINANCE_API_KEY" {
		log.Fatal("❌ 请先配置币安API密钥！")
	}

	// 创建自动交易器
	autoTrader, err := trader.NewAutoTrader(config)
	if err != nil {
		log.Fatalf("❌ 初始化失败: %v", err)
	}

	fmt.Println("✓ 自动交易器初始化成功")
	fmt.Println()
	fmt.Println("📋 配置信息:")
	fmt.Printf("  • AI模型: %s\n", getAIName(config.UseQwen))
	fmt.Printf("  • 扫描间隔: %v\n", config.ScanInterval)
	fmt.Printf("  • 每笔风险: %.1f%%\n", config.RiskPercentPerTrade)
	fmt.Printf("  • 默认杠杆: %dx\n", config.DefaultLeverage)
	fmt.Printf("  • 最大持仓: %d\n", config.MaxPositions)
	fmt.Printf("  • 最小信心度: %.0f%%\n", config.MinConfidence)
	fmt.Printf("  • 最小优先级: %d\n", config.MinPriority)
	fmt.Printf("  • 风险回报比: 1:%.1f\n", config.MinRiskReward)
	fmt.Println()
	fmt.Println("⚠️  风险提示: 自动交易有风险，请谨慎使用！")
	fmt.Println()
	fmt.Println("按 Ctrl+C 停止运行")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println()

	// 设置优雅退出
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// 启动自动交易（在goroutine中运行）
	go func() {
		if err := autoTrader.Run(); err != nil {
			log.Printf("❌ 运行错误: %v", err)
		}
	}()

	// 等待退出信号
	<-sigChan
	fmt.Println()
	fmt.Println()
	log.Println("📛 收到退出信号，正在停止...")
	autoTrader.Stop()

	fmt.Println()
	fmt.Println("👋 感谢使用！")
}

func getAIName(useQwen bool) string {
	if useQwen {
		return "阿里云Qwen"
	}
	return "DeepSeek"
}
