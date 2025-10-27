package market

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// PositionInfo 持仓信息
type PositionInfo struct {
	Symbol           string  `json:"symbol"`
	Side             string  `json:"side"` // "long" or "short"
	EntryPrice       float64 `json:"entry_price"`
	MarkPrice        float64 `json:"mark_price"`
	Quantity         float64 `json:"quantity"`
	Leverage         int     `json:"leverage"`
	UnrealizedPnL    float64 `json:"unrealized_pnl"`
	UnrealizedPnLPct float64 `json:"unrealized_pnl_pct"`
	LiquidationPrice float64 `json:"liquidation_price"`
	MarginUsed       float64 `json:"margin_used"`
}

// AccountInfo 账户信息
type AccountInfo struct {
	TotalEquity      float64 `json:"total_equity"`      // 账户净值
	AvailableBalance float64 `json:"available_balance"` // 可用余额
	TotalPnL         float64 `json:"total_pnl"`         // 总盈亏
	TotalPnLPct      float64 `json:"total_pnl_pct"`     // 总盈亏百分比
	MarginUsed       float64 `json:"margin_used"`       // 已用保证金
	MarginUsedPct    float64 `json:"margin_used_pct"`   // 保证金使用率
	PositionCount    int     `json:"position_count"`    // 持仓数量
}

// CandidateCoin 候选币种（来自币种池）
type CandidateCoin struct {
	Symbol string `json:"symbol"`
}

// TradingContext 交易上下文（传递给AI的完整信息）
type TradingContext struct {
	CurrentTime    string                 `json:"current_time"`
	RuntimeMinutes int                    `json:"runtime_minutes"`
	CallCount      int                    `json:"call_count"`
	Account        AccountInfo            `json:"account"`
	Positions      []PositionInfo         `json:"positions"`
	CandidateCoins []CandidateCoin        `json:"candidate_coins"`
	MarketDataMap  map[string]*MarketData `json:"-"` // 不序列化，但内部使用
}

// TradingDecision AI的交易决策
type TradingDecision struct {
	Symbol          string  `json:"symbol"`
	Action          string  `json:"action"` // "open_long", "open_short", "close_long", "close_short", "hold", "wait"
	Leverage        int     `json:"leverage,omitempty"`
	PositionSizeUSD float64 `json:"position_size_usd,omitempty"`
	StopLoss        float64 `json:"stop_loss,omitempty"`
	TakeProfit      float64 `json:"take_profit,omitempty"`
	Reasoning       string  `json:"reasoning"`
}

// AIFullDecision AI的完整决策（包含思维链）
type AIFullDecision struct {
	CoTTrace  string            `json:"cot_trace"` // 思维链分析
	Decisions []TradingDecision `json:"decisions"` // 具体决策列表
	Timestamp time.Time         `json:"timestamp"`
}

// GetFullTradingDecision 获取AI的完整交易决策（批量分析所有币种和持仓）
func GetFullTradingDecision(ctx *TradingContext) (*AIFullDecision, error) {
	// 1. 为所有币种获取市场数据
	if err := fetchMarketDataForContext(ctx); err != nil {
		return nil, fmt.Errorf("获取市场数据失败: %w", err)
	}

	// 2. 构建AI提示
	prompt := buildFullDecisionPrompt(ctx)

	// 3. 调用AI API
	aiResponse, err := callDeepSeekAPI(prompt)
	if err != nil {
		return nil, fmt.Errorf("调用AI API失败: %w", err)
	}

	// 4. 解析AI响应
	decision, err := parseFullDecisionResponse(aiResponse)
	if err != nil {
		return nil, fmt.Errorf("解析AI响应失败: %w", err)
	}

	decision.Timestamp = time.Now()
	return decision, nil
}

// fetchMarketDataForContext 为上下文中的所有币种获取市场数据
func fetchMarketDataForContext(ctx *TradingContext) error {
	ctx.MarketDataMap = make(map[string]*MarketData)

	// 收集所有需要获取数据的币种
	symbolSet := make(map[string]bool)

	// 持仓币种
	for _, pos := range ctx.Positions {
		symbolSet[pos.Symbol] = true
	}

	// 候选币种（限制数量，避免太多API调用）
	maxCandidates := 10
	for i, coin := range ctx.CandidateCoins {
		if i >= maxCandidates {
			break
		}
		symbolSet[coin.Symbol] = true
	}

	// 并发获取市场数据
	for symbol := range symbolSet {
		data, err := GetMarketData(symbol)
		if err != nil {
			// 单个币种失败不影响整体，只记录错误
			continue
		}
		ctx.MarketDataMap[symbol] = data
	}

	return nil
}

// buildFullDecisionPrompt 构建完整的AI决策提示
func buildFullDecisionPrompt(ctx *TradingContext) string {
	var sb strings.Builder

	sb.WriteString("# 🤖 AI 摆动交易实盘竞赛系统\n\n")
	sb.WriteString("你是一个专业的加密货币摆动交易AI，正在参与实盘交易竞赛。\n\n")

	// 系统状态
	sb.WriteString("## 📊 系统状态\n")
	sb.WriteString(fmt.Sprintf("- **当前时间**: %s\n", ctx.CurrentTime))
	sb.WriteString(fmt.Sprintf("- **运行时长**: %d 分钟\n", ctx.RuntimeMinutes))
	sb.WriteString(fmt.Sprintf("- **调用次数**: 第 %d 次\n\n", ctx.CallCount))

	// 账户信息
	sb.WriteString("## 💰 账户信息\n")
	sb.WriteString(fmt.Sprintf("- **账户净值**: %.2f USDT\n", ctx.Account.TotalEquity))
	sb.WriteString(fmt.Sprintf("- **可用余额**: %.2f USDT (%.1f%%)\n",
		ctx.Account.AvailableBalance,
		(ctx.Account.AvailableBalance/ctx.Account.TotalEquity)*100))
	sb.WriteString(fmt.Sprintf("- **总盈亏**: %.2f USDT (%+.2f%%)\n",
		ctx.Account.TotalPnL, ctx.Account.TotalPnLPct))
	sb.WriteString(fmt.Sprintf("- **已用保证金**: %.2f USDT (%.1f%%)\n",
		ctx.Account.MarginUsed, ctx.Account.MarginUsedPct))
	sb.WriteString(fmt.Sprintf("- **持仓数量**: %d\n\n", ctx.Account.PositionCount))

	// 当前持仓详情
	if len(ctx.Positions) > 0 {
		sb.WriteString("## 📈 当前持仓\n")
		for i, pos := range ctx.Positions {
			sb.WriteString(fmt.Sprintf("\n### 持仓 #%d: %s %s\n", i+1, pos.Symbol, strings.ToUpper(pos.Side)))
			sb.WriteString(fmt.Sprintf("- **入场价**: %.4f USDT\n", pos.EntryPrice))
			sb.WriteString(fmt.Sprintf("- **当前价**: %.4f USDT\n", pos.MarkPrice))
			sb.WriteString(fmt.Sprintf("- **数量**: %.4f\n", pos.Quantity))
			sb.WriteString(fmt.Sprintf("- **杠杆**: %dx\n", pos.Leverage))
			sb.WriteString(fmt.Sprintf("- **未实现盈亏**: %.2f USDT (%+.2f%%)\n",
				pos.UnrealizedPnL, pos.UnrealizedPnLPct))
			sb.WriteString(fmt.Sprintf("- **强平价**: %.4f USDT\n", pos.LiquidationPrice))
			sb.WriteString(fmt.Sprintf("- **占用保证金**: %.2f USDT\n", pos.MarginUsed))

			// 添加市场数据
			if marketData, ok := ctx.MarketDataMap[pos.Symbol]; ok {
				sb.WriteString(formatMarketDataBrief(marketData))
			}
		}
		sb.WriteString("\n")
	} else {
		sb.WriteString("## 📈 当前持仓\n")
		sb.WriteString("暂无持仓\n\n")
	}

	// 候选币种池
	sb.WriteString("## 🎯 候选币种池\n")
	for i, coin := range ctx.CandidateCoins {
		if i >= 10 { // 只显示前10个
			sb.WriteString(fmt.Sprintf("...还有 %d 个币种\n", len(ctx.CandidateCoins)-10))
			break
		}
		sb.WriteString(fmt.Sprintf("\n### 币种 #%d: %s\n", i+1, coin.Symbol))

		if marketData, ok := ctx.MarketDataMap[coin.Symbol]; ok {
			sb.WriteString(formatMarketDataBrief(marketData))
		} else {
			sb.WriteString("市场数据获取中...\n")
		}
	}
	sb.WriteString("\n")

	// AI决策要求
	sb.WriteString("## 🎯 你的任务\n\n")
	sb.WriteString("**对每个持仓币种做出决策**（持有/平仓），**对无持仓币种评估是否开仓**。\n\n")

	sb.WriteString("### 📋 决策原则\n")
	sb.WriteString("1. **仓位管理**: 根据账户净值和风险自主决定每笔交易的仓位大小（position_size_usd）\n")
	sb.WriteString("2. **杠杆选择**: 根据市场波动率和信心度自主选择杠杆倍数（1-20倍）\n")
	sb.WriteString("3. **风险控制**: 单笔风险建议不超过账户净值的2-5%\n")
	sb.WriteString("4. **趋势跟随**: 优先选择趋势明确、技术指标一致的机会\n")
	sb.WriteString("5. **止损止盈**: 风险回报比建议至少1:2\n")
	sb.WriteString("6. **分散投资**: 避免过度集中，建议同时持有2-5个不同方向的仓位\n\n")

	sb.WriteString("### 📤 输出格式要求\n\n")
	sb.WriteString("**第一部分**: `cot_trace` (思维链分析)\n")
	sb.WriteString("```\n")
	sb.WriteString("1. **市场概况**: 整体市场情绪、波动率、资金流向\n")
	sb.WriteString("2. **持仓分析**: 逐个分析每个持仓，评估是否继续持有\n")
	sb.WriteString("   - 技术指标解读（趋势/背离/超买超卖）\n")
	sb.WriteString("   - 盈亏情况与预期的匹配度\n")
	sb.WriteString("   - 止损止盈触发条件\n")
	sb.WriteString("   - 是否需要平仓的理由\n")
	sb.WriteString("3. **新机会扫描**: 评估候选币种中的开仓机会\n")
	sb.WriteString("   - 技术形态评分\n")
	sb.WriteString("   - 风险回报比计算\n")
	sb.WriteString("   - 与现有持仓的相关性\n")
	sb.WriteString("4. **风险评估**: \n")
	sb.WriteString("   - 当前账户风险暴露\n")
	sb.WriteString("   - 保证金使用率\n")
	sb.WriteString("   - 是否需要降低杠杆或减仓\n")
	sb.WriteString("5. **最终决策**: 总结本次交易计划\n")
	sb.WriteString("```\n\n")

	sb.WriteString("**第二部分**: `decisions` (JSON数组)\n")
	sb.WriteString("```json\n")
	sb.WriteString("[\n")
	sb.WriteString("  {\n")
	sb.WriteString("    \"symbol\": \"BTCUSDT\",\n")
	sb.WriteString("    \"action\": \"open_long\",  // 可选: open_long, open_short, close_long, close_short, hold, wait\n")
	sb.WriteString("    \"leverage\": 10,          // AI自主决定（1-20）\n")
	sb.WriteString("    \"position_size_usd\": 100.0,  // AI自主决定仓位大小（USD）\n")
	sb.WriteString("    \"stop_loss\": 45000.0,\n")
	sb.WriteString("    \"take_profit\": 50000.0,\n")
	sb.WriteString("    \"reasoning\": \"简短理由\"\n")
	sb.WriteString("  },\n")
	sb.WriteString("  {\n")
	sb.WriteString("    \"symbol\": \"ETHUSDT\",\n")
	sb.WriteString("    \"action\": \"hold\",\n")
	sb.WriteString("    \"reasoning\": \"继续观察，趋势未明\"\n")
	sb.WriteString("  }\n")
	sb.WriteString("]\n")
	sb.WriteString("```\n\n")

	sb.WriteString("### ⚠️ 重要说明\n")
	sb.WriteString("- `action`类型: **open_long**(开多), **open_short**(开空), **close_long**(平多), **close_short**(平空), **hold**(持有), **wait**(观望)\n")
	sb.WriteString("- 开仓时必须提供: `leverage`, `position_size_usd`, `stop_loss`, `take_profit`\n")
	sb.WriteString("- 平仓/持有/观望时只需提供: `action`, `reasoning`\n")
	sb.WriteString("- `position_size_usd`是实际投入的USD金额，系统会根据杠杆计算实际买入数量\n")
	sb.WriteString("- 请确保JSON格式严格正确，可以被解析\n\n")

	sb.WriteString("现在请开始分析并给出你的决策！\n")

	return sb.String()
}

// formatMarketDataBrief 格式化市场数据（简洁版）
func formatMarketDataBrief(data *MarketData) string {
	var sb strings.Builder

	sb.WriteString("**市场数据** (3分钟线):\n")
	sb.WriteString(fmt.Sprintf("  - 价格: %.4f | EMA20: %.4f (%s)\n",
		data.CurrentPrice, data.CurrentEMA20, pricePosition(data.CurrentPrice, data.CurrentEMA20)))
	sb.WriteString(fmt.Sprintf("  - MACD: %.4f (%s) | RSI(7): %.2f (%s)\n",
		data.CurrentMACD, macdTrend(data.CurrentMACD), data.CurrentRSI7, rsiStatus(data.CurrentRSI7)))

	if data.OpenInterest != nil {
		oiChange := ((data.OpenInterest.Latest - data.OpenInterest.Average) / data.OpenInterest.Average) * 100
		sb.WriteString(fmt.Sprintf("  - 持仓量: %+.2f%% | 资金费率: %.6f\n", oiChange, data.FundingRate))
	}

	return sb.String()
}

// parseFullDecisionResponse 解析AI的完整决策响应
func parseFullDecisionResponse(aiResponse string) (*AIFullDecision, error) {
	// 1. 提取 cot_trace（思维链）
	cotTrace := extractCoTTrace(aiResponse)

	// 2. 提取 JSON 决策列表
	decisions, err := extractDecisions(aiResponse)
	if err != nil {
		return nil, fmt.Errorf("提取决策失败: %w", err)
	}

	return &AIFullDecision{
		CoTTrace:  cotTrace,
		Decisions: decisions,
	}, nil
}

// extractCoTTrace 提取思维链分析
func extractCoTTrace(response string) string {
	// 尝试提取思维链部分（通常在JSON之前）
	jsonStart := strings.Index(response, "```json")
	if jsonStart == -1 {
		jsonStart = strings.Index(response, "[")
	}

	if jsonStart > 0 {
		return strings.TrimSpace(response[:jsonStart])
	}

	// 如果找不到JSON，整个响应都是思维链
	return strings.TrimSpace(response)
}

// extractDecisions 提取JSON决策列表
func extractDecisions(response string) ([]TradingDecision, error) {
	// 查找JSON数组
	jsonStart := strings.Index(response, "```json")
	jsonEnd := strings.LastIndex(response, "```")

	var jsonContent string
	if jsonStart != -1 && jsonEnd != -1 && jsonEnd > jsonStart {
		jsonContent = strings.TrimSpace(response[jsonStart+7 : jsonEnd])
	} else {
		// 尝试直接查找JSON数组
		arrayStart := strings.Index(response, "[")
		arrayEnd := strings.LastIndex(response, "]")

		if arrayStart == -1 || arrayEnd == -1 || arrayEnd <= arrayStart {
			return nil, fmt.Errorf("无法找到JSON数组")
		}

		jsonContent = strings.TrimSpace(response[arrayStart : arrayEnd+1])
	}

	// 解析JSON
	var decisions []TradingDecision
	if err := json.Unmarshal([]byte(jsonContent), &decisions); err != nil {
		return nil, fmt.Errorf("JSON解析失败: %w\nJSON内容: %s", err, jsonContent)
	}

	// 验证决策
	for i, decision := range decisions {
		if err := validateDecision(&decision); err != nil {
			return nil, fmt.Errorf("决策 #%d 验证失败: %w", i+1, err)
		}
	}

	return decisions, nil
}

// validateDecision 验证单个决策的有效性
func validateDecision(d *TradingDecision) error {
	// 验证action
	validActions := map[string]bool{
		"open_long":   true,
		"open_short":  true,
		"close_long":  true,
		"close_short": true,
		"hold":        true,
		"wait":        true,
	}

	if !validActions[d.Action] {
		return fmt.Errorf("无效的action: %s", d.Action)
	}

	// 开仓操作必须提供完整参数
	if d.Action == "open_long" || d.Action == "open_short" {
		if d.Leverage <= 0 || d.Leverage > 20 {
			return fmt.Errorf("杠杆必须在1-20之间: %d", d.Leverage)
		}
		if d.PositionSizeUSD <= 0 {
			return fmt.Errorf("仓位大小必须大于0: %.2f", d.PositionSizeUSD)
		}
		if d.StopLoss <= 0 || d.TakeProfit <= 0 {
			return fmt.Errorf("止损和止盈必须大于0")
		}

		// 验证止损止盈的合理性
		if d.Action == "open_long" {
			if d.StopLoss >= d.TakeProfit {
				return fmt.Errorf("做多时止损价必须小于止盈价")
			}
		} else {
			if d.StopLoss <= d.TakeProfit {
				return fmt.Errorf("做空时止损价必须大于止盈价")
			}
		}
	}

	return nil
}
