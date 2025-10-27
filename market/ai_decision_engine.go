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

	// 1. 优先获取持仓币种的数据（这是必须的）
	for _, pos := range ctx.Positions {
		symbolSet[pos.Symbol] = true
	}

	// 2. 候选币种数量根据账户状态动态调整
	maxCandidates := calculateMaxCandidates(ctx)
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

// calculateMaxCandidates 根据账户状态计算需要分析的候选币种数量
func calculateMaxCandidates(ctx *TradingContext) int {
	// 直接返回候选池的全部币种数量
	// 因为候选池已经在 auto_trader.go 中根据保证金使用率筛选过了
	// 无持仓时：30个评分最高币种
	// 有持仓时：15/25/35个评分最高币种（根据保证金使用率）
	return len(ctx.CandidateCoins)
}

// buildFullDecisionPrompt 构建完整的AI决策提示
func buildFullDecisionPrompt(ctx *TradingContext) string {
	var sb strings.Builder

	sb.WriteString("# 🤖 AI 摆动交易实盘竞赛系统\n\n")
	sb.WriteString("你是一个**激进型**专业加密货币摆动交易AI，正在参与实盘交易竞赛。\n\n")
	sb.WriteString("## ⚡ 交易风格定位\n")
	sb.WriteString("- **风格**: 激进型/高频波段\n")
	sb.WriteString("- **目标**: 追求高收益，把握每个高确定性机会\n")
	sb.WriteString("- **仓位策略**: 单笔20-50%账户净值，允许较高的资金使用率\n")
	sb.WriteString("- **杠杆范围**: 主要使用5-15倍，主流币可达20倍\n")
	sb.WriteString("- **持仓数量**: 2-4个并行持仓，充分利用资金\n\n")

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

	// 候选币种池 - 显示所有获取了市场数据的币种
	sb.WriteString("## 🎯 候选币种池\n")
	sb.WriteString(fmt.Sprintf("**总共 %d 个候选币种的市场数据**\n\n", len(ctx.MarketDataMap)))

	displayedCount := 0
	for _, coin := range ctx.CandidateCoins {
		// 只显示已获取市场数据的币种
		marketData, hasData := ctx.MarketDataMap[coin.Symbol]
		if !hasData {
			continue
		}
		displayedCount++
		sb.WriteString(fmt.Sprintf("\n### 币种 #%d: %s\n", displayedCount, coin.Symbol))
		sb.WriteString(formatMarketDataBrief(marketData))
	}

	// AI决策要求
	sb.WriteString("## 🎯 你的任务\n\n")
	sb.WriteString("**重要**：请按以下顺序进行决策分析：\n\n")
	sb.WriteString("### 第一步：分析现有持仓（如果有）\n")
	sb.WriteString("- 评估每个持仓的盈亏状态\n")
	sb.WriteString("- 判断技术指标是否支持继续持有\n")
	sb.WriteString("- 决定是否需要平仓/止盈/止损\n")
	sb.WriteString("- 计算平仓后可释放的资金\n\n")
	sb.WriteString("### 第二步：评估账户风险状态\n")
	sb.WriteString("- 当前保证金使用率是否安全\n")
	sb.WriteString("- 是否有足够的可用余额\n")
	sb.WriteString("- 总盈亏情况如何\n\n")
	sb.WriteString("### 第三步：考虑新机会（如果条件允许）\n")
	sb.WriteString("- **激进开仓条件**：\n")
	sb.WriteString("  - 可用余额至少15%以上即可考虑（激进策略）\n")
	sb.WriteString("  - 保证金使用率在85%以下都可以开仓\n")
	sb.WriteString("  - 发现高确定性机会时，大胆下单\n")
	sb.WriteString(fmt.Sprintf("- 从上面 **%d 个候选币种**中评估，优选技术形态最强的标的\n", len(ctx.MarketDataMap)))
	sb.WriteString("- 综合考虑价格趋势、RSI、MACD、持仓量等指标\n\n")

	sb.WriteString("### 📋 决策原则（激进策略）\n")
	sb.WriteString("1. **激进仓位**: 单笔仓位占账户净值的20-40%，把握高确定性机会时可达50%\n")
	sb.WriteString("2. **风险控制**: 保证金使用率超过85%时不要开新仓，允许较高的资金使用率\n")
	sb.WriteString("3. **持仓优先**: 先确保现有持仓健康，再考虑新机会\n")
	sb.WriteString("4. **杠杆选择**: 积极使用5-15倍杠杆，主流币可用10-20倍\n")
	sb.WriteString("5. **止损止盈**: 风险回报比至少1:2，追求更高收益\n")
	sb.WriteString("6. **多持仓**: 可同时持有2-4个高确定性机会，分散风险\n\n")

	sb.WriteString("### 📤 输出格式要求\n\n")
	sb.WriteString("**重要**: 请直接输出思维链和JSON，不要使用任何markdown代码块标记（不要用```或```json）\n\n")

	sb.WriteString("**第一部分**: 思维链分析（纯文本，按以下格式）\n\n")
	sb.WriteString("第一步：现有持仓分析\n")
	sb.WriteString("- [如无持仓，写\"无持仓\"并跳过]\n")
	sb.WriteString("- 逐个分析每个持仓的技术指标、盈亏状态\n")
	sb.WriteString("- 判断是否需要平仓及理由\n")
	sb.WriteString("- 预估平仓后可释放的资金\n\n")
	sb.WriteString("第二步：账户风险评估\n")
	sb.WriteString("- 当前保证金使用率: X%\n")
	sb.WriteString("- 可用余额占比: X%\n")
	sb.WriteString("- 总体风险等级: 高/中/低\n")
	sb.WriteString("- 是否适合开新仓: 是/否，原因...\n\n")
	sb.WriteString("第三步：新机会评估（仅在适合开仓时分析）\n")
	sb.WriteString("- 候选币种筛选标准\n")
	sb.WriteString("- 技术形态最优的2-3个币种\n")
	sb.WriteString("- 风险回报比计算\n")
	sb.WriteString("- 建议的仓位大小和杠杆\n\n")
	sb.WriteString("第四步：最终决策总结\n")
	sb.WriteString("- 平仓决策: X个\n")
	sb.WriteString("- 开仓决策: X个\n")
	sb.WriteString("- 持有决策: X个\n")
	sb.WriteString("- 整体策略: ...\n\n")
	sb.WriteString("---\n\n")

	sb.WriteString("**第二部分**: 决策JSON数组（纯JSON，不要任何代码块标记）\n\n")
	sb.WriteString("[\n")
	sb.WriteString("  {\n")
	sb.WriteString("    \"symbol\": \"BTCUSDT\",\n")
	sb.WriteString("    \"action\": \"open_long\",\n")
	sb.WriteString("    \"leverage\": 10,\n")
	sb.WriteString("    \"position_size_usd\": 300.0,\n")
	sb.WriteString("    \"stop_loss\": 45000.0,\n")
	sb.WriteString("    \"take_profit\": 50000.0,\n")
	sb.WriteString("    \"reasoning\": \"强势突破，激进做多\"\n")
	sb.WriteString("  },\n")
	sb.WriteString("  {\n")
	sb.WriteString("    \"symbol\": \"ETHUSDT\",\n")
	sb.WriteString("    \"action\": \"hold\",\n")
	sb.WriteString("    \"reasoning\": \"继续观察，趋势未明\"\n")
	sb.WriteString("  }\n")
	sb.WriteString("]\n\n")

	sb.WriteString("### ⚠️ 重要说明\n")
	sb.WriteString("- `action`类型: **open_long**(开多), **open_short**(开空), **close_long**(平多), **close_short**(平空), **hold**(持有), **wait**(观望)\n")
	sb.WriteString("- 开仓时必须提供: `leverage`, `position_size_usd`, `stop_loss`, `take_profit`\n")
	sb.WriteString("- 平仓/持有/观望时只需提供: `action`, `reasoning`\n")
	sb.WriteString("- `position_size_usd`是实际投入的USD金额，系统会根据杠杆计算实际买入数量\n")
	sb.WriteString("- **决策顺序很重要**: JSON数组中先列出平仓决策，再列出开仓决策\n")
	sb.WriteString("- **风险控制**: 如果保证金使用率>85%，不要输出任何开仓决策\n")
	sb.WriteString("- **激进建议**: 仓位大小建议200-500 USDT，高确定性机会可用更大仓位\n")
	sb.WriteString("- 请确保JSON格式严格正确，可以被解析\n\n")

	sb.WriteString("### 📝 决策示例（注意：不要使用markdown代码块）\n\n")
	sb.WriteString("**场景1 - 有持仓需要调整**:\n")
	sb.WriteString("[\n")
	sb.WriteString("  {\"symbol\": \"BTCUSDT\", \"action\": \"close_long\", \"reasoning\": \"RSI超买且MACD死叉，止盈离场\"},\n")
	sb.WriteString("  {\"symbol\": \"ETHUSDT\", \"action\": \"hold\", \"reasoning\": \"趋势延续，继续持有\"}\n")
	sb.WriteString("]\n\n")
	sb.WriteString("**场景2 - 无持仓，寻找机会**:\n")
	sb.WriteString("[\n")
	sb.WriteString("  {\"symbol\": \"SOLUSDT\", \"action\": \"open_long\", \"leverage\": 10, \"position_size_usd\": 300, \"stop_loss\": 18.5, \"take_profit\": 21.5, \"reasoning\": \"RSI超卖反弹，激进做多\"}\n")
	sb.WriteString("]\n\n")
	sb.WriteString("**场景3 - 保证金使用率高，只管理现有持仓**:\n")
	sb.WriteString("[\n")
	sb.WriteString("  {\"symbol\": \"BTCUSDT\", \"action\": \"hold\", \"reasoning\": \"趋势良好，继续持有\"},\n")
	sb.WriteString("  {\"symbol\": \"ETHUSDT\", \"action\": \"close_short\", \"reasoning\": \"止损，趋势反转\"}\n")
	sb.WriteString("]\n\n")

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
	// 查找JSON数组的开始位置
	jsonStart := strings.Index(response, "[")

	if jsonStart > 0 {
		// 思维链是JSON数组之前的内容
		return strings.TrimSpace(response[:jsonStart])
	}

	// 如果找不到JSON，整个响应都是思维链
	return strings.TrimSpace(response)
}

// extractDecisions 提取JSON决策列表
func extractDecisions(response string) ([]TradingDecision, error) {
	// 直接查找JSON数组 - 找第一个完整的JSON数组
	arrayStart := strings.Index(response, "[")
	if arrayStart == -1 {
		return nil, fmt.Errorf("无法找到JSON数组起始")
	}

	// 从 [ 开始，匹配括号找到对应的 ]
	arrayEnd := findMatchingBracket(response, arrayStart)
	if arrayEnd == -1 {
		return nil, fmt.Errorf("无法找到JSON数组结束")
	}

	jsonContent := strings.TrimSpace(response[arrayStart : arrayEnd+1])

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

// findMatchingBracket 查找匹配的右括号
func findMatchingBracket(s string, start int) int {
	if start >= len(s) || s[start] != '[' {
		return -1
	}

	depth := 0
	for i := start; i < len(s); i++ {
		switch s[i] {
		case '[':
			depth++
		case ']':
			depth--
			if depth == 0 {
				return i
			}
		}
	}

	return -1
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
