package api

import (
	"fmt"
	"log"
	"net/http"
	"nofx/logger"
	"nofx/trader"

	"github.com/gin-gonic/gin"
)

// Server HTTP API服务器
type Server struct {
	router      *gin.Engine
	autoTrader  *trader.AutoTrader
	decisionLog *logger.DecisionLogger
	port        int
}

// NewServer 创建API服务器
func NewServer(autoTrader *trader.AutoTrader, decisionLog *logger.DecisionLogger, port int) *Server {
	// 设置为Release模式（减少日志输出）
	gin.SetMode(gin.ReleaseMode)

	router := gin.Default()

	// 启用CORS
	router.Use(corsMiddleware())

	s := &Server{
		router:      router,
		autoTrader:  autoTrader,
		decisionLog: decisionLog,
		port:        port,
	}

	// 设置路由
	s.setupRoutes()

	return s
}

// corsMiddleware CORS中间件
func corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusOK)
			return
		}

		c.Next()
	}
}

// setupRoutes 设置路由
func (s *Server) setupRoutes() {
	// 健康检查
	s.router.GET("/health", s.handleHealth)

	// API路由组
	api := s.router.Group("/api")
	{
		// 系统状态
		api.GET("/status", s.handleStatus)

		// 账户信息
		api.GET("/account", s.handleAccount)

		// 持仓列表
		api.GET("/positions", s.handlePositions)

		// 决策日志
		api.GET("/decisions", s.handleDecisions)
		api.GET("/decisions/latest", s.handleLatestDecisions)
		api.GET("/decisions/:filename", s.handleDecisionDetail)

		// 统计信息
		api.GET("/statistics", s.handleStatistics)

		// 收益率历史数据
		api.GET("/equity-history", s.handleEquityHistory)
	}
}

// handleHealth 健康检查
func (s *Server) handleHealth(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
		"time":   c.Request.Context().Value("time"),
	})
}

// handleStatus 系统状态
func (s *Server) handleStatus(c *gin.Context) {
	status := s.autoTrader.GetStatus()
	c.JSON(http.StatusOK, status)
}

// handleAccount 账户信息
func (s *Server) handleAccount(c *gin.Context) {
	log.Printf("📊 收到账户信息请求")
	account, err := s.autoTrader.GetAccountInfo()
	if err != nil {
		log.Printf("❌ 获取账户信息失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("获取账户信息失败: %v", err),
		})
		return
	}

	log.Printf("✓ 返回账户信息: 净值=%.2f, 可用=%.2f, 盈亏=%.2f (%.2f%%)",
		account["total_equity"],
		account["available_balance"],
		account["total_pnl"],
		account["total_pnl_pct"])
	c.JSON(http.StatusOK, account)
}

// handlePositions 持仓列表
func (s *Server) handlePositions(c *gin.Context) {
	positions, err := s.autoTrader.GetPositions()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("获取持仓列表失败: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, positions)
}

// handleDecisions 决策日志列表
func (s *Server) handleDecisions(c *gin.Context) {
	// 获取所有历史决策记录（无限制）
	records, err := s.decisionLog.GetLatestRecords(10000)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("获取决策日志失败: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, records)
}

// handleLatestDecisions 最新决策日志（最近5条，最新的在前）
func (s *Server) handleLatestDecisions(c *gin.Context) {
	records, err := s.decisionLog.GetLatestRecords(5)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("获取决策日志失败: %v", err),
		})
		return
	}

	// 反转数组，让最新的在前面（用于列表显示）
	// GetLatestRecords返回的是从旧到新（用于图表），这里需要从新到旧
	for i, j := 0, len(records)-1; i < j; i, j = i+1, j-1 {
		records[i], records[j] = records[j], records[i]
	}

	c.JSON(http.StatusOK, records)
}

// handleDecisionDetail 决策详情
func (s *Server) handleDecisionDetail(c *gin.Context) {
	filename := c.Param("filename")

	// TODO: 实现根据文件名获取决策详情
	c.JSON(http.StatusOK, gin.H{
		"filename": filename,
		"message":  "功能开发中",
	})
}

// handleStatistics 统计信息
func (s *Server) handleStatistics(c *gin.Context) {
	stats, err := s.decisionLog.GetStatistics()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("获取统计信息失败: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// handleEquityHistory 收益率历史数据
func (s *Server) handleEquityHistory(c *gin.Context) {
	// 获取尽可能多的历史数据（几天的数据）
	// 每3分钟一个周期：10000条 = 约20天的数据
	records, err := s.decisionLog.GetLatestRecords(10000)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("获取历史数据失败: %v", err),
		})
		return
	}

	// 构建收益率历史数据点
	type EquityPoint struct {
		Timestamp        string  `json:"timestamp"`
		TotalEquity      float64 `json:"total_equity"`      // 账户净值（wallet + unrealized）
		AvailableBalance float64 `json:"available_balance"` // 可用余额
		TotalPnL         float64 `json:"total_pnl"`         // 总盈亏（相对初始余额）
		TotalPnLPct      float64 `json:"total_pnl_pct"`     // 总盈亏百分比
		PositionCount    int     `json:"position_count"`    // 持仓数量
		MarginUsedPct    float64 `json:"margin_used_pct"`   // 保证金使用率
		CycleNumber      int     `json:"cycle_number"`
	}

	// 从第一条记录获取初始余额（用于计算盈亏百分比）
	initialBalance := 1047.0 // 默认值，如果有记录则从AutoTrader获取
	if at := s.autoTrader; at != nil {
		if status := at.GetStatus(); status != nil {
			if ib, ok := status["initial_balance"].(float64); ok {
				initialBalance = ib
			}
		}
	}

	var history []EquityPoint
	for _, record := range records {
		// TotalBalance字段实际存储的是TotalEquity
		totalEquity := record.AccountState.TotalBalance
		// TotalUnrealizedProfit字段实际存储的是TotalPnL（相对初始余额）
		totalPnL := record.AccountState.TotalUnrealizedProfit

		// 计算盈亏百分比
		totalPnLPct := 0.0
		if initialBalance > 0 {
			totalPnLPct = (totalPnL / initialBalance) * 100
		}

		history = append(history, EquityPoint{
			Timestamp:        record.Timestamp.Format("2006-01-02 15:04:05"),
			TotalEquity:      totalEquity,
			AvailableBalance: record.AccountState.AvailableBalance,
			TotalPnL:         totalPnL,
			TotalPnLPct:      totalPnLPct,
			PositionCount:    record.AccountState.PositionCount,
			MarginUsedPct:    record.AccountState.MarginUsedPct,
			CycleNumber:      record.CycleNumber,
		})
	}

	c.JSON(http.StatusOK, history)
}

// Start 启动服务器
func (s *Server) Start() error {
	addr := fmt.Sprintf(":%d", s.port)
	log.Printf("🌐 API服务器启动在 http://localhost%s", addr)
	log.Printf("📊 API文档:")
	log.Printf("  • GET  /api/status          - 系统状态")
	log.Printf("  • GET  /api/account         - 账户信息")
	log.Printf("  • GET  /api/positions       - 持仓列表")
	log.Printf("  • GET  /api/decisions       - 决策日志（最多10000条）")
	log.Printf("  • GET  /api/decisions/latest - 最新决策（最近5条）")
	log.Printf("  • GET  /api/statistics      - 统计信息")
	log.Printf("  • GET  /api/equity-history  - 收益率历史数据（最多10000条 ≈ 20天）")
	log.Printf("  • GET  /health              - 健康检查")
	log.Println()

	return s.router.Run(addr)
}
