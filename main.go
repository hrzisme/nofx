package main

import (
	"fmt"
	"log"
	"nofx/api"
	"nofx/logger"
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
		BinanceAPIKey:    "7xPCTcqquHXdiVCCKvkC3Vx6AUObNmw6YvuYtCd6FGLvoA2erBf2IXu0yx8lXBq8",
		BinanceSecretKey: "pEeN2Py0MgZoIowDe968pmQFYssnWXdywM5JoVUchVEYLH2ApPbAJUzLTeMZe1jL",
		CoinPoolAPIURL:   "http://43.128.34.180:30006/api/ai500/list?auth=admin123sadasd3r323",

		// AI配置
		UseQwen:     false, // true=使用Qwen, false=使用DeepSeek
		DeepSeekKey: "sk-44ac4e74ef184461800fccd15685ad30",
		QwenKey:     "sk-c02cc4678f094a72b5513e92889eda04",

		// 扫描配置
		ScanInterval: 3 * time.Minute, // 每3分钟一次AI决策

		// 账户配置
		InitialBalance: 1000.0, // 初始金额（USDT），用于计算盈亏百分比

		// 风险控制（仅作为提示，AI可自主决定）
		MaxDailyLoss:    5.0,              // 最大日亏损5%（提示）
		MaxDrawdown:     10.0,             // 最大回撤10%（提示）
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

	fmt.Println("✓ AI驱动自动交易器初始化成功")
	fmt.Println()
	fmt.Println("📋 系统配置:")
	fmt.Printf("  • AI模型: %s\n", getAIName(config.UseQwen))
	fmt.Printf("  • 决策周期: %v (每3分钟)\n", config.ScanInterval)
	fmt.Printf("  • 初始金额: %.2f USDT\n", config.InitialBalance)
	fmt.Println()
	fmt.Println("🤖 AI全权决策模式:")
	fmt.Println("  • AI将自主决定每笔交易的杠杆倍数（1-20倍）")
	fmt.Println("  • AI将自主决定每笔交易的仓位大小")
	fmt.Println("  • AI将自主设置止损和止盈价格")
	fmt.Println("  • AI将基于市场数据、技术指标、账户状态做出全面分析")
	fmt.Println()
	fmt.Println("⚠️  风险提示: AI自动交易有风险，建议小额资金测试！")
	fmt.Println()
	fmt.Println("按 Ctrl+C 停止运行")
	fmt.Println(strings.Repeat("=", 60))
	fmt.Println()

	// 创建决策日志记录器（用于API）
	decisionLogger := logger.NewDecisionLogger("decision_logs")

	// 创建并启动API服务器（在goroutine中运行）
	apiServer := api.NewServer(autoTrader, decisionLogger, 8080)
	go func() {
		if err := apiServer.Start(); err != nil {
			log.Printf("❌ API服务器错误: %v", err)
		}
	}()

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
