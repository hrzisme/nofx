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
	account, err := s.autoTrader.GetAccountInfo()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("获取账户信息失败: %v", err),
		})
		return
	}

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
	// 获取最近30条记录
	records, err := s.decisionLog.GetLatestRecords(30)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("获取决策日志失败: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, records)
}

// handleLatestDecisions 最新决策日志（最近5条）
func (s *Server) handleLatestDecisions(c *gin.Context) {
	records, err := s.decisionLog.GetLatestRecords(5)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("获取决策日志失败: %v", err),
		})
		return
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

// Start 启动服务器
func (s *Server) Start() error {
	addr := fmt.Sprintf(":%d", s.port)
	log.Printf("🌐 API服务器启动在 http://localhost%s", addr)
	log.Printf("📊 API文档:")
	log.Printf("  • GET  /api/status          - 系统状态")
	log.Printf("  • GET  /api/account         - 账户信息")
	log.Printf("  • GET  /api/positions       - 持仓列表")
	log.Printf("  • GET  /api/decisions       - 决策日志（最近30条）")
	log.Printf("  • GET  /api/decisions/latest - 最新决策（最近5条）")
	log.Printf("  • GET  /api/statistics      - 统计信息")
	log.Printf("  • GET  /health              - 健康检查")
	log.Println()

	return s.router.Run(addr)
}
