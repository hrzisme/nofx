package market

import (
	"encoding/json"
	"fmt"
	"nofx/pool"
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
	Symbol  string   `json:"symbol"`
	Sources []string `json:"sources"` // 来源: "ai500" 和/或 "oi_top"
}

// OITopData 持仓量增长Top数据（用于AI决策参考）
type OITopData struct {
	Rank              int     // OI Top排名
	OIDeltaPercent    float64 // 持仓量变化百分比（1小时）
	OIDeltaValue      float64 // 持仓量变化价值
	PriceDeltaPercent float64 // 价格变化百分比
	NetLong           float64 // 净多仓
	NetShort          float64 // 净空仓
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
	OITopDataMap   map[string]*OITopData  `json:"-"` // OI Top数据映射
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

// fetchMarketDataForContext 为上下文中的所有币种获取市场数据和OI数据
func fetchMarketDataForContext(ctx *TradingContext) error {
	ctx.MarketDataMap = make(map[string]*MarketData)
	ctx.OITopDataMap = make(map[string]*OITopData)

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

	// 加载OI Top数据（不影响主流程）
	oiPositions, err := pool.GetOITopPositions()
	if err == nil {
		for _, pos := range oiPositions {
			// 标准化符号匹配
			symbol := pos.Symbol
			ctx.OITopDataMap[symbol] = &OITopData{
				Rank:              pos.Rank,
				OIDeltaPercent:    pos.OIDeltaPercent,
				OIDeltaValue:      pos.OIDeltaValue,
				PriceDeltaPercent: pos.PriceDeltaPercent,
				NetLong:           pos.NetLong,
				NetShort:          pos.NetShort,
			}
		}
	}

	return nil
}

// calculateMaxCandidates 根据账户状态计算需要分析的候选币种数量
func calculateMaxCandidates(ctx *TradingContext) int {
	// 直接返回候选池的全部币种数量
	// 因为候选池已经在 auto_trader.go 中筛选过了
	// 固定分析前20个评分最高的币种（来自AI500）
	return len(ctx.CandidateCoins)
}

// buildFullDecisionPrompt 构建完整的AI决策提示
func buildFullDecisionPrompt(ctx *TradingContext) string {
	var sb strings.Builder

	sb.WriteString("# 🤖 AI 摆动交易实盘竞赛系统\n\n")
	sb.WriteString("你是一个**激进型**专业加密货币摆动交易AI，正在参与实盘交易竞赛。\n\n")
	sb.WriteString("## ⚡ 交易风格定位\n")
	sb.WriteString("- **风格**: 极度激进型/重仓波段\n")
	sb.WriteString("- **目标**: 追求极致收益，充分利用每一分资金\n")
	sb.WriteString("- **仓位策略**: 单笔60-90%账户净值，极强信号可满仓100%\n")
	sb.WriteString("- **杠杆范围**: 积极使用15-20倍杠杆，放大收益\n")
	sb.WriteString("- **持仓数量**: 1-3个重仓持仓，保持极高资金使用率（目标80-95%）\n\n")

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
	sb.WriteString("## 🎯 候选币种池（AI500 + OI Top合并）\n")
	sb.WriteString(fmt.Sprintf("**总共 %d 个候选币种的市场数据**\n", len(ctx.MarketDataMap)))
	sb.WriteString("说明: [AI500]=AI评分高 | [OI_Top]=持仓量增长快 | [双标签]=两者都满足\n\n")

	displayedCount := 0
	for _, coin := range ctx.CandidateCoins {
		// 只显示已获取市场数据的币种
		marketData, hasData := ctx.MarketDataMap[coin.Symbol]
		if !hasData {
			continue
		}
		displayedCount++

		// 显示币种来源标签
		sourceTags := ""
		for _, source := range coin.Sources {
			if source == "ai500" {
				sourceTags += "[AI500] "
			} else if source == "oi_top" {
				sourceTags += "[OI_Top] "
			}
		}

		sb.WriteString(fmt.Sprintf("\n### 币种 #%d: %s %s\n", displayedCount, coin.Symbol, sourceTags))
		sb.WriteString(formatMarketDataBrief(marketData))

		// 如果有OI Top数据，也显示出来
		if oiTopData, hasOI := ctx.OITopDataMap[coin.Symbol]; hasOI {
			sb.WriteString(fmt.Sprintf("**市场热度** (OI Top排名 #%d):\n", oiTopData.Rank))
			sb.WriteString(fmt.Sprintf("  - 持仓量1h变化: %+.2f%% (价值: $%.0f)\n",
				oiTopData.OIDeltaPercent, oiTopData.OIDeltaValue))
			sb.WriteString(fmt.Sprintf("  - 价格1h变化: %+.2f%% | 净多仓: %.0f | 净空仓: %.0f\n",
				oiTopData.PriceDeltaPercent, oiTopData.NetLong, oiTopData.NetShort))
		}
	}

	// AI决策要求
	sb.WriteString("## 🎯 你的任务\n\n")
	sb.WriteString("**重要**：请按以下顺序进行决策分析：\n\n")
	sb.WriteString("### 第一步：分析现有持仓（如果有）\n")
	sb.WriteString("- 评估每个持仓的盈亏状态\n")
	sb.WriteString("- 判断技术指标是否支持继续持有\n")
	sb.WriteString("- 决定是否需要平仓/止盈/止损\n")
	sb.WriteString("- 计算平仓后可释放的资金\n\n")
	sb.WriteString("### 第二步：评估候选池中的新机会\n")
	sb.WriteString(fmt.Sprintf("- 从上面 **%d 个候选币种**中找出技术形态最强的2-5个标的\n", len(ctx.MarketDataMap)))
	sb.WriteString("- **综合评分维度**：\n")
	sb.WriteString("  1. 技术指标：价格趋势、RSI、MACD\n")
	sb.WriteString("  2. 市场热度：[OI_Top]标签的币种，持仓量增长说明资金流入\n")
	sb.WriteString("  3. AI评分：[AI500]标签的币种，AI评分高\n")
	sb.WriteString("  4. 双重信号：同时有[AI500]和[OI_Top]标签的，优先级最高\n")
	sb.WriteString("- 标注每个强势币种的信号强度（强/中/弱）\n\n")
	sb.WriteString("### 第三步：换仓机会评估（关键！）\n")
	sb.WriteString("- **对比现有持仓 vs 新发现的强势币种**\n")
	sb.WriteString("- 如果候选池中有明显更强的机会（信号更强、趋势更明确）：\n")
	sb.WriteString("  - 考虑平掉表现较弱/盈利已够/趋势转弱的持仓\n")
	sb.WriteString("  - 用释放的资金开仓更强的新机会\n")
	sb.WriteString("- **换仓条件**：新机会的综合评分明显优于现有持仓（至少高20%）\n")
	sb.WriteString("- 即使保证金使用率高（80-95%），也可以通过换仓优化持仓质量\n\n")
	sb.WriteString("### 第四步：账户风险状态与开仓决策\n")
	sb.WriteString("- 当前保证金使用率是否安全\n")
	sb.WriteString("- 是否有足够的可用余额\n")
	sb.WriteString("- **极度激进开仓条件**：\n")
	sb.WriteString("  - 可用余额至少10%以上即可考虑（极度激进策略）\n")
	sb.WriteString("  - 保证金使用率在95%以下都可以开仓\n")
	sb.WriteString("  - 发现高确定性机会时，大胆下单\n")
	sb.WriteString("- 如果没有可用余额但发现极强机会 → 优先考虑换仓策略\n\n")

	sb.WriteString("### 📋 决策原则（极度激进策略）\n")
	sb.WriteString(fmt.Sprintf("1. **仓位计算公式**: position_size_usd = 账户净值(%.2f USDT) × 仓位比例(60-100%%)\n", ctx.Account.TotalEquity))
	sb.WriteString("   - 中等信号（评分7-8分）：60-70%账户净值\n")
	sb.WriteString("   - 强信号（评分8-9分）：70-90%账户净值\n")
	sb.WriteString("   - 极强信号（评分9-10分）：80-100%账户净值（可满仓）\n")
	sb.WriteString("2. **风险控制**: 保证金使用率超过95%时不要开新仓，目标保持在80-95%\n")
	sb.WriteString("3. **持仓优先**: 先确保现有持仓健康，发现更强机会时积极换仓\n")
	sb.WriteString("4. **杠杆选择**: 积极使用15-20倍杠杆，小币种谨慎用10-15倍\n")
	sb.WriteString("5. **止损止盈**: 风险回报比至少1:2，追求更高收益\n")
	sb.WriteString("6. **重仓集中**: 可同时持有1-3个高确定性机会，保持极高资金使用率（80-95%）\n\n")

	sb.WriteString("### 📤 输出格式要求\n\n")
	sb.WriteString("**重要**: 请直接输出思维链和JSON，不要使用任何markdown代码块标记（不要用```或```json）\n\n")

	sb.WriteString("**第一部分**: 思维链分析（纯文本，按以下格式）\n\n")
	sb.WriteString("第一步：现有持仓分析\n")
	sb.WriteString("- [如无持仓，写\"无持仓\"并跳过]\n")
	sb.WriteString("- 逐个分析每个持仓的技术指标、盈亏状态、信号强度\n")
	sb.WriteString("- 给每个持仓打分（1-10分）\n\n")
	sb.WriteString("第二步：候选池强势币种筛选\n")
	sb.WriteString("- 从候选池中找出2-5个技术形态最强的币种\n")
	sb.WriteString("- 给每个强势币种打分（1-10分）\n")
	sb.WriteString("- 标注信号强度（强/中/弱）\n\n")
	sb.WriteString("第三步：换仓机会评估（关键！）\n")
	sb.WriteString("- 对比持仓评分 vs 候选币种评分\n")
	sb.WriteString("- 如果候选池有明显更强的机会（评分差距≥2分）：\n")
	sb.WriteString("  - 列出建议换仓的持仓和目标币种\n")
	sb.WriteString("  - 说明换仓理由（为什么新机会更好）\n")
	sb.WriteString("- 如果没有明显更好的机会：说明\"无需换仓\"\n\n")
	sb.WriteString("第四步：账户风险与最终决策\n")
	sb.WriteString("- 当前保证金使用率: X%\n")
	sb.WriteString("- 可用余额占比: X%\n")
	sb.WriteString("- 最终决策总结：\n")
	sb.WriteString("  - 换仓决策: X个（平A开B）\n")
	sb.WriteString("  - 平仓决策: X个（止盈/止损）\n")
	sb.WriteString("  - 开仓决策: X个（新增仓位）\n")
	sb.WriteString("  - 持有决策: X个\n\n")
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
	sb.WriteString("- **风险控制**: 如果保证金使用率>95%，不要输出任何开仓决策\n")
	sb.WriteString(fmt.Sprintf("- **仓位大小计算**: 必须按账户净值(%.2f)的百分比计算，不要用固定金额！\n", ctx.Account.TotalEquity))
	sb.WriteString(fmt.Sprintf("  - 示例：评分8分 → position_size_usd = %.2f × 0.75 = %.2f USDT\n",
		ctx.Account.TotalEquity, ctx.Account.TotalEquity*0.75))
	sb.WriteString(fmt.Sprintf("  - 示例：评分9分 → position_size_usd = %.2f × 0.85 = %.2f USDT\n",
		ctx.Account.TotalEquity, ctx.Account.TotalEquity*0.85))
	sb.WriteString("- 请确保JSON格式严格正确，可以被解析\n\n")

	sb.WriteString("### 📝 决策示例（注意：不要使用markdown代码块）\n\n")

	// 动态计算示例仓位（极度激进版本）
	exampleSize70 := ctx.Account.TotalEquity * 0.7  // 评分8分
	exampleSize85 := ctx.Account.TotalEquity * 0.85 // 评分9分
	exampleSize75 := ctx.Account.TotalEquity * 0.75 // 评分8分
	exampleSize90 := ctx.Account.TotalEquity * 0.9  // 评分9-10分

	sb.WriteString("**场景1 - 换仓（发现更好机会）**:\n")
	sb.WriteString("[\n")
	sb.WriteString("  {\"symbol\": \"BTCUSDT\", \"action\": \"close_long\", \"reasoning\": \"MACD死叉，趋势转弱（评分6分），平仓释放资金换仓\"},\n")
	sb.WriteString(fmt.Sprintf("  {\"symbol\": \"SOLUSDT\", \"action\": \"open_long\", \"leverage\": 18, \"position_size_usd\": %.0f, \"stop_loss\": 118.5, \"take_profit\": 135.0, \"reasoning\": \"RSI超卖+MACD金叉，信号极强（评分9分=85%%仓位），换仓机会\"},\n", exampleSize85))
	sb.WriteString("  {\"symbol\": \"ETHUSDT\", \"action\": \"hold\", \"reasoning\": \"趋势延续，继续持有（评分8分）\"}\n")
	sb.WriteString("]\n\n")
	sb.WriteString("**场景2 - 无持仓，寻找机会（开2个重仓）**:\n")
	sb.WriteString("[\n")
	sb.WriteString(fmt.Sprintf("  {\"symbol\": \"BNBUSDT\", \"action\": \"open_long\", \"leverage\": 18, \"position_size_usd\": %.0f, \"stop_loss\": 580.0, \"take_profit\": 650.0, \"reasoning\": \"突破关键阻力，多头信号强（评分9分=85%%仓位）\"},\n", exampleSize85))
	sb.WriteString(fmt.Sprintf("  {\"symbol\": \"AVAXUSDT\", \"action\": \"open_short\", \"leverage\": 15, \"position_size_usd\": %.0f, \"stop_loss\": 32.0, \"take_profit\": 26.0, \"reasoning\": \"空头趋势确认（评分8分=70%%仓位）\"}\n", exampleSize70))
	sb.WriteString("]\n\n")
	sb.WriteString("**场景3 - 保证金使用率高，通过换仓优化**:\n")
	sb.WriteString("[\n")
	sb.WriteString("  {\"symbol\": \"XRPUSDT\", \"action\": \"close_short\", \"reasoning\": \"小亏损，趋势减弱（评分5分），释放资金换仓\"},\n")
	sb.WriteString(fmt.Sprintf("  {\"symbol\": \"SOLUSDT\", \"action\": \"open_long\", \"leverage\": 20, \"position_size_usd\": %.0f, \"stop_loss\": 115.0, \"take_profit\": 145.0, \"reasoning\": \"候选池最强币种（评分10分=90%%仓位），换仓机会\"},\n", exampleSize90))
	sb.WriteString("  {\"symbol\": \"BTCUSDT\", \"action\": \"hold\", \"reasoning\": \"趋势良好，保留（评分8分）\"}\n")
	sb.WriteString("]\n\n")
	sb.WriteString("**场景4 - 激进重仓（1-2个极强机会）**:\n")
	sb.WriteString("[\n")
	sb.WriteString(fmt.Sprintf("  {\"symbol\": \"ETHUSDT\", \"action\": \"open_long\", \"leverage\": 18, \"position_size_usd\": %.0f, \"stop_loss\": 3200.0, \"take_profit\": 3800.0, \"reasoning\": \"主流币突破（评分8分=70%%）\"},\n", exampleSize70))
	sb.WriteString(fmt.Sprintf("  {\"symbol\": \"SOLUSDT\", \"action\": \"open_long\", \"leverage\": 20, \"position_size_usd\": %.0f, \"stop_loss\": 120.0, \"take_profit\": 145.0, \"reasoning\": \"技术形态极佳（评分9分=75%%）\"}\n", exampleSize75))
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
