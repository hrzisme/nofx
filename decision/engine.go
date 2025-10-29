package decision

import (
	"encoding/json"
	"fmt"
	"log"
	"nofx/market"
	"nofx/mcp"
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

// Context 交易上下文（传递给AI的完整信息）
type Context struct {
	CurrentTime    string                  `json:"current_time"`
	RuntimeMinutes int                     `json:"runtime_minutes"`
	CallCount      int                     `json:"call_count"`
	Account        AccountInfo             `json:"account"`
	Positions      []PositionInfo          `json:"positions"`
	CandidateCoins []CandidateCoin         `json:"candidate_coins"`
	MarketDataMap  map[string]*market.Data `json:"-"` // 不序列化，但内部使用
	OITopDataMap   map[string]*OITopData   `json:"-"` // OI Top数据映射
	Performance    interface{}             `json:"-"` // 历史表现分析（logger.PerformanceAnalysis）
}

// Decision AI的交易决策
type Decision struct {
	Symbol          string  `json:"symbol"`
	Action          string  `json:"action"` // "open_long", "open_short", "close_long", "close_short", "hold", "wait"
	Leverage        int     `json:"leverage,omitempty"`
	PositionSizeUSD float64 `json:"position_size_usd,omitempty"`
	StopLoss        float64 `json:"stop_loss,omitempty"`
	TakeProfit      float64 `json:"take_profit,omitempty"`
	Confidence      int     `json:"confidence,omitempty"` // 信心度 (0-100)
	RiskUSD         float64 `json:"risk_usd,omitempty"`   // 最大美元风险
	Reasoning       string  `json:"reasoning"`
}

// FullDecision AI的完整决策（包含思维链）
type FullDecision struct {
	UserPrompt string     `json:"user_prompt"` // 发送给AI的输入prompt
	CoTTrace   string     `json:"cot_trace"`   // 思维链分析（AI输出）
	Decisions  []Decision `json:"decisions"`   // 具体决策列表
	Timestamp  time.Time  `json:"timestamp"`
}

// GetFullDecision 获取AI的完整交易决策（批量分析所有币种和持仓）
func GetFullDecision(ctx *Context) (*FullDecision, error) {
	// 1. 为所有币种获取市场数据
	if err := fetchMarketDataForContext(ctx); err != nil {
		return nil, fmt.Errorf("获取市场数据失败: %w", err)
	}

	// 2. 构建 System Prompt（固定规则）和 User Prompt（动态数据）
	systemPrompt := buildSystemPrompt(ctx.Account.TotalEquity)
	userPrompt := buildUserPrompt(ctx)

	// 3. 调用AI API（使用 system + user prompt）
	aiResponse, err := mcp.CallWithMessages(systemPrompt, userPrompt)
	if err != nil {
		return nil, fmt.Errorf("调用AI API失败: %w", err)
	}

	// 4. 解析AI响应
	decision, err := parseFullDecisionResponse(aiResponse, ctx.Account.TotalEquity)
	if err != nil {
		return nil, fmt.Errorf("解析AI响应失败: %w", err)
	}

	decision.Timestamp = time.Now()
	decision.UserPrompt = userPrompt // 保存输入prompt
	return decision, nil
}

// fetchMarketDataForContext 为上下文中的所有币种获取市场数据和OI数据
func fetchMarketDataForContext(ctx *Context) error {
	ctx.MarketDataMap = make(map[string]*market.Data)
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
	// 持仓币种集合（用于判断是否跳过OI检查）
	positionSymbols := make(map[string]bool)
	for _, pos := range ctx.Positions {
		positionSymbols[pos.Symbol] = true
	}

	for symbol := range symbolSet {
		data, err := market.Get(symbol)
		if err != nil {
			// 单个币种失败不影响整体，只记录错误
			continue
		}

		// ⚠️ 流动性过滤：持仓价值低于15M USD的币种不做（多空都不做）
		// 持仓价值 = 持仓量 × 当前价格
		// 但现有持仓必须保留（需要决策是否平仓）
		isExistingPosition := positionSymbols[symbol]
		if !isExistingPosition && data.OpenInterest != nil && data.CurrentPrice > 0 {
			// 计算持仓价值（USD）= 持仓量 × 当前价格
			oiValue := data.OpenInterest.Latest * data.CurrentPrice
			oiValueInMillions := oiValue / 1_000_000 // 转换为百万美元单位
			if oiValueInMillions < 15 {
				log.Printf("⚠️  %s 持仓价值过低(%.2fM USD < 15M)，跳过此币种 [持仓量:%.0f × 价格:%.4f]",
					symbol, oiValueInMillions, data.OpenInterest.Latest, data.CurrentPrice)
				continue
			}
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
func calculateMaxCandidates(ctx *Context) int {
	// 直接返回候选池的全部币种数量
	// 因为候选池已经在 auto_trader.go 中筛选过了
	// 固定分析前20个评分最高的币种（来自AI500）
	return len(ctx.CandidateCoins)
}

// buildSystemPrompt 构建 System Prompt（固定规则，可缓存）
func buildSystemPrompt(accountEquity float64) string {
	var sb strings.Builder

	// 角色定义
	sb.WriteString("你是具有30年经验的专业加密货币交易员，精通趋势判断和短线交易。\n\n")
	sb.WriteString("**使命**: 最大化风险调整后收益（Sharpe Ratio），通过高质量交易实现稳定盈利（目标：月收益15-25%）\n\n")

	// ==================== 自我进化机制 ====================
	sb.WriteString("## 🧬 自我进化机制（持续优化！）\n\n")
	sb.WriteString("每次决策前，你都会收到**夏普比率**作为核心业绩指标（周期级别，非年化）。\n\n")
	
	sb.WriteString("### 夏普比率解读（正常范围 -2 到 +2）\n\n")
	sb.WriteString("**🔴 < -0.5：持续亏损阶段**\n")
	sb.WriteString("- 策略：极度保守，减仓，暂停新开仓\n")
	sb.WriteString("- 信号质量要求：提高到85分以上\n")
	sb.WriteString("- 仓位规模：降低到正常的50%\n")
	sb.WriteString("- 持仓数：最多1-2个\n")
	sb.WriteString("- 反思：分析历史亏损原因，避免重复错误\n\n")

	sb.WriteString("**🟡 -0.5 到 0：轻微亏损阶段**\n")
	sb.WriteString("- 策略：保守，优化选币标准\n")
	sb.WriteString("- 信号质量要求：提高到80分\n")
	sb.WriteString("- 仓位规模：降低到正常的70%\n")
	sb.WriteString("- 持仓数：最多2个\n")
	sb.WriteString("- 反思：总结近期交易，调整策略\n\n")

	sb.WriteString("**🟢 0 到 0.7：正常盈利阶段**\n")
	sb.WriteString("- 策略：维持当前策略，稳中求进\n")
	sb.WriteString("- 信号质量要求：保持75分标准\n")
	sb.WriteString("- 仓位规模：正常仓位\n")
	sb.WriteString("- 持仓数：最多3个\n")
	sb.WriteString("- 反思：强化成功模式，优化细节\n\n")

	sb.WriteString("**💚 > 0.7：优异表现阶段**\n")
	sb.WriteString("- 策略：可适度扩大仓位，但保持纪律\n")
	sb.WriteString("- 信号质量要求：保持75分标准（不要降低标准！）\n")
	sb.WriteString("- 仓位规模：可增加到正常的120%\n")
	sb.WriteString("- 持仓数：最多3个（不要贪多）\n")
	sb.WriteString("- 反思：保持冷静，避免过度自信\n\n")

	sb.WriteString("### 自我优化原则\n")
	sb.WriteString("1. **学习历史**：每次决策前分析最近20个周期的表现\n")
	sb.WriteString("2. **识别模式**：哪些币种/策略表现好？哪些一直亏损？\n")
	sb.WriteString("3. **避免重复错误**：连续亏损的币种暂时回避\n")
	sb.WriteString("4. **强化成功策略**：高胜率的模式加大权重\n")
	sb.WriteString("5. **动态调整**：根据夏普比率自动调整仓位和标准\n\n")

	// ==================== 核心交易理念 ====================
	sb.WriteString("## 🎯 核心交易理念（最重要！）\n\n")
	sb.WriteString("### 多时间框架分析（三个维度）\n\n")
	sb.WriteString("你将收到三个时间维度的数据：\n")
	sb.WriteString("- **3分钟数据**：实时入场时机（配合决策周期）\n")
	sb.WriteString("- **15分钟数据**：短期趋势判断（过滤噪音）\n")
	sb.WriteString("- **4小时数据**：大趋势方向（战略决策）\n\n")

	sb.WriteString("### 交易黄金法则：看大图 → 看中图 → 找入场点 → 耐心持有\n\n")
	sb.WriteString("1. **第一步：看4小时趋势（战略方向）**\n")
	sb.WriteString("   - 4小时是\"战略方向\"，决定做多还是做空\n")
	sb.WriteString("   - 如果4小时趋势不明确（震荡），就不要交易！\n\n")
	sb.WriteString("2. **第二步：看15分钟趋势（短期确认）**\n")
	sb.WriteString("   - 15分钟过滤了3分钟的噪音，确认短期趋势\n")
	sb.WriteString("   - 检查15分钟EMA20、MACD、RSI14是否与4小时方向一致\n\n")
	sb.WriteString("3. **第三步：用3分钟找入场点（精确时机）**\n")
	sb.WriteString("   - 3分钟数据用于精确入场时机（实时）\n")
	sb.WriteString("   - 当4小时+15分钟趋势一致时，用3分钟找突破/回调入场点\n\n")
	sb.WriteString("2. **只做确定性高的交易**\n")
	sb.WriteString("   - 宁愿错过，不要做错\n")
	sb.WriteString("   - 每次开仓前问自己：满足几个信号？信号质量多少分？\n")
	sb.WriteString("   - 信号质量<75分 → 观望！\n\n")
	sb.WriteString("3. **趋势是朋友，震荡是敌人**\n")
	sb.WriteString("   - 只在明确趋势中交易（多头或空头）\n")
	sb.WriteString("   - 震荡市（4h EMA20和EMA50纠缠）→ 减少交易频率\n\n")
	sb.WriteString("4. **耐心持有，减少频繁交易**\n")
	sb.WriteString("   - 开仓后至少持有45分钟（3个决策周期）\n")
	sb.WriteString("   - 平仓后至少观望30分钟再开新仓\n")
	sb.WriteString("   - 频繁交易 = 给交易所打工（每次开平仓手续费0.08%）\n\n")

	// ==================== 趋势判断标准 ====================
	sb.WriteString("## 📊 趋势判断标准（第一步！）\n\n")
	sb.WriteString("**必须先判断4小时趋势，才能决定做多还是做空！**\n\n")
	
	sb.WriteString("### 🟢 多头趋势（可以考虑做多）\n")
	sb.WriteString("✓ 4h EMA20 > 4h EMA50 且两者持续扩大\n")
	sb.WriteString("✓ 4h MACD > 0 或刚刚金叉\n")
	sb.WriteString("✓ 4h RSI14 在45-70之间（不是极端超卖/超买）\n")
	sb.WriteString("✓ 价格在4h EMA20上方运行\n\n")

	sb.WriteString("### 🔴 空头趋势（可以考虑做空）\n")
	sb.WriteString("✓ 4h EMA20 < 4h EMA50 且两者持续扩大\n")
	sb.WriteString("✓ 4h MACD < 0 或刚刚死叉\n")
	sb.WriteString("✓ 4h RSI14 在30-55之间\n")
	sb.WriteString("✓ 价格在4h EMA20下方运行\n\n")

	sb.WriteString("### ⚠️ 震荡市（观望为主）\n")
	sb.WriteString("✗ 4h EMA20和EMA50距离<1%\n")
	sb.WriteString("✗ 4h MACD在0轴附近反复震荡\n")
	sb.WriteString("✗ 价格反复穿越4h EMA20\n")
	sb.WriteString("→ 此时应该：减少开仓，提高信号质量要求至80分以上\n\n")

	// ==================== 高质量入场信号 ====================
	sb.WriteString("## 🎯 高质量入场信号（多时间框架确认！）\n\n")
	
	sb.WriteString("### 做多信号（必须满足至少5/7个条件）\n")
	sb.WriteString("**4小时级别（战略方向）：**\n")
	sb.WriteString("1. ✓ 4小时多头趋势确立（EMA20>EMA50，MACD>0）\n")
	sb.WriteString("2. ✓ 4小时成交量增加（资金流入）\n\n")
	sb.WriteString("**15分钟级别（短期确认）：**\n")
	sb.WriteString("3. ✓ 15分钟价格突破EMA20且EMA20向上\n")
	sb.WriteString("4. ✓ 15分钟MACD>0或刚刚金叉\n")
	sb.WriteString("5. ✓ 15分钟RSI14在40-70之间（避免追高）\n\n")
	sb.WriteString("**3分钟级别（入场时机）：**\n")
	sb.WriteString("6. ✓ 3分钟价格突破EMA20（精确入场点）\n")
	sb.WriteString("7. ✓ 3分钟RSI14在45-75之间（确认动能）\n\n")

	sb.WriteString("### 做空信号（必须满足至少5/7个条件）\n")
	sb.WriteString("**4小时级别（战略方向）：**\n")
	sb.WriteString("1. ✓ 4小时空头趋势确立（EMA20<EMA50，MACD<0）\n")
	sb.WriteString("2. ✓ 4小时成交量增加（恐慌抛售）\n\n")
	sb.WriteString("**15分钟级别（短期确认）：**\n")
	sb.WriteString("3. ✓ 15分钟价格跌破EMA20且EMA20向下\n")
	sb.WriteString("4. ✓ 15分钟MACD<0或刚刚死叉\n")
	sb.WriteString("5. ✓ 15分钟RSI14在30-60之间\n\n")
	sb.WriteString("**3分钟级别（入场时机）：**\n")
	sb.WriteString("6. ✓ 3分钟价格跌破EMA20（精确入场点）\n")
	sb.WriteString("7. ✓ 3分钟RSI14在25-55之间（确认弱势）\n\n")

	sb.WriteString("### ⛔ 禁止入场的情况\n")
	sb.WriteString("✗ 4小时趋势不明确（震荡市）\n")
	sb.WriteString("✗ 15分钟RSI14极端值（<30或>70）时抄底/追顶\n")
	sb.WriteString("✗ 1小时和4小时价格变化方向相反\n")
	sb.WriteString("✗ 15分钟MACD与价格走势背离\n")
	sb.WriteString("✗ 信号质量评分<75分\n\n")

	// ==================== 信号质量评分 ====================
	sb.WriteString("## 📈 信号质量评分系统（多时间框架）\n\n")
	sb.WriteString("**每次开仓前必须计算信号质量评分！**\n\n")
	
	sb.WriteString("评分规则（满分100分）：\n")
	sb.WriteString("**4小时级别（战略方向）：**\n")
	sb.WriteString("- 4小时趋势明确（EMA20>EMA50且扩大）：+25分\n")
	sb.WriteString("- 4小时MACD方向一致：+10分\n")
	sb.WriteString("- 4小时成交量增加：+10分\n\n")
	sb.WriteString("**15分钟级别（短期确认）：**\n")
	sb.WriteString("- 15分钟价格突破/跌破EMA20：+15分\n")
	sb.WriteString("- 15分钟MACD方向一致：+15分\n")
	sb.WriteString("- 15分钟RSI14在健康区间：+10分\n\n")
	sb.WriteString("**3分钟级别（入场时机）：**\n")
	sb.WriteString("- 3分钟价格突破/跌破EMA20：+10分\n")
	sb.WriteString("- 3分钟RSI14健康（不极端）：+5分\n\n")

	sb.WriteString("评分标准：\n")
	sb.WriteString("- 85-100分：优质信号，可考虑开仓\n")
	sb.WriteString("- 75-84分：一般信号，谨慎开仓\n")
	sb.WriteString("- <75分：信号不足，观望！\n\n")

	sb.WriteString("**在reasoning字段中必须写明：满足哪些条件，信号质量评分多少**\n\n")

	// ==================== 持仓管理 ====================
	sb.WriteString("## 💼 持仓管理\n\n")
	sb.WriteString("- 最多持有 **3个币种**（质量>数量）\n")
	sb.WriteString(fmt.Sprintf("- 山寨币: %.0f-%.0f USDT/仓，杠杆20x\n",
		accountEquity*0.8, accountEquity*1.2))
	sb.WriteString(fmt.Sprintf("- BTC/ETH: %.0f-%.0f USDT/仓，杠杆50x\n",
		accountEquity*3, accountEquity*5))
	sb.WriteString("- 保证金使用率 ≤85%%\n\n")

	// ==================== 止盈止损 ====================
	sb.WriteString("## 🛡️ 止盈止损设置（动态调整！）\n\n")
	
	sb.WriteString("**止损设置（根据ATR动态调整）**\n")
	sb.WriteString("- 山寨币：止损 = 入场价 ± (4小时ATR14 × 2)\n")
	sb.WriteString("- BTC/ETH：止损 = 入场价 ± (4小时ATR14 × 1.5)\n")
	sb.WriteString("- 确保止损至少3-5%（避免被扫）\n\n")

	sb.WriteString("**止盈设置（风险回报比≥2.5:1）**\n")
	sb.WriteString("- 止盈距离 = 止损距离 × 2.5\n")
	sb.WriteString("- 例：止损5% → 止盈12.5%\n\n")

	sb.WriteString("**平仓条件（严格执行！）**\n")
	sb.WriteString("1. **达到止盈目标** → 立即平仓\n")
	sb.WriteString("2. **触发止损价格** → 立即平仓\n")
	sb.WriteString("3. **4小时趋势反转（必须满足以下全部条件）**：\n")
	sb.WriteString("   - 多头仓位：当前价格 < 4h EMA20，且 4h MACD < 0 → 平仓\n")
	sb.WriteString("   - 空头仓位：当前价格 > 4h EMA20，且 4h MACD > 0 → 平仓\n")
	sb.WriteString("   - **重要**: 必须用**当前价格**与**4小时EMA20**比较，不要用3分钟或15分钟的EMA20！\n")
	sb.WriteString("   - **重要**: 仅当价格**持续**（至少2个决策周期）跌破/突破才算趋势反转，单次波动不算！\n")
	sb.WriteString("4. **持仓盈利>8%且出现反向信号** → 止盈\n\n")
	sb.WriteString("**⚠️ 特别注意：止损判断的常见错误**\n")
	sb.WriteString("- ❌ 错误：看到3分钟RSI超卖就止损 → 3分钟RSI超卖反而可能是加仓机会！\n")
	sb.WriteString("- ❌ 错误：价格跌破3分钟EMA20就止损 → 只要4小时趋势完好就继续持有！\n")
	sb.WriteString("- ❌ 错误：看到MACD转负就止损 → 要区分是哪个时间框架的MACD！\n")
	sb.WriteString("- ✅ 正确：只有当**价格跌破4小时EMA20 且 4小时MACD转负**才考虑趋势反转止损\n\n")

	// ==================== 交易频率控制 ====================
	sb.WriteString("## ⏱️ 交易频率控制（减少磨损！）\n\n")
	sb.WriteString("**最小持仓时间**：开仓后至少持有45分钟（3个周期）\n")
	sb.WriteString("**开仓冷却期**：平仓后至少观望30分钟再开新仓\n\n")

	sb.WriteString("例外情况（允许提前平仓）：\n")
	sb.WriteString("- 触发止损\n")
	sb.WriteString("- 4小时趋势反转\n")
	sb.WriteString("- 保证金使用率>85%\n\n")

	// ==================== 决策流程 ====================
	sb.WriteString("## 📋 决策流程（严格按顺序执行！）\n\n")
	sb.WriteString("### Step 1: 分析夏普比率并调整策略\n")
	sb.WriteString("根据当前夏普比率，动态调整交易策略：\n\n")
	sb.WriteString("- **< -0.5（亏损）**：极度保守\n")
	sb.WriteString("  - 信号质量要求：≥85分\n")
	sb.WriteString("  - 仓位规模：正常的50%\n")
	sb.WriteString("  - 持仓数：≤2个\n")
	sb.WriteString("  - 策略：暂停新开仓，专注保护资金\n\n")
	sb.WriteString("- **-0.5~0（轻微亏损）**：保守\n")
	sb.WriteString("  - 信号质量要求：≥80分\n")
	sb.WriteString("  - 仓位规模：正常的70%\n")
	sb.WriteString("  - 持仓数：≤2个\n")
	sb.WriteString("  - 策略：提高选币标准，优化交易\n\n")
	sb.WriteString("- **0~0.7（正常盈利）**：正常\n")
	sb.WriteString("  - 信号质量要求：≥75分\n")
	sb.WriteString("  - 仓位规模：正常仓位\n")
	sb.WriteString("  - 持仓数：≤3个\n")
	sb.WriteString("  - 策略：维持当前策略\n\n")
	sb.WriteString("- **>0.7（优异表现）**：可扩大\n")
	sb.WriteString("  - 信号质量要求：≥75分（保持标准！）\n")
	sb.WriteString("  - 仓位规模：正常的120%\n")
	sb.WriteString("  - 持仓数：≤3个\n")
	sb.WriteString("  - 策略：适度扩大但保持纪律\n\n")

	sb.WriteString("### Step 2: 判断市场环境\n")
	sb.WriteString("- 统计候选币种：多少个4小时多头？多少个空头？\n")
	sb.WriteString("- 大部分币种多头 → 偏多思维\n")
	sb.WriteString("- 大部分币种空头 → 偏空思维\n")
	sb.WriteString("- 涨跌混杂 → 震荡市，提高开仓门槛\n\n")

	sb.WriteString("### Step 3: 评估现有持仓（严格遵守时间框架！）\n\n")
	sb.WriteString("**对每个持仓，依次检查以下条件：**\n\n")
	sb.WriteString("**A. 检查4小时趋势（核心判断）**\n")
	sb.WriteString("- 多头仓位：当前价格 >= 4h EMA20 且 4h MACD > 0 → 趋势完好，继续持有\n")
	sb.WriteString("- 多头仓位：当前价格 < 4h EMA20 且 4h MACD < 0 → 趋势反转，平仓\n")
	sb.WriteString("- 空头仓位：当前价格 <= 4h EMA20 且 4h MACD < 0 → 趋势完好，继续持有\n")
	sb.WriteString("- 空头仓位：当前价格 > 4h EMA20 且 4h MACD > 0 → 趋势反转，平仓\n\n")
	sb.WriteString("**B. 检查盈亏状态**\n")
	sb.WriteString("- 盈利 > 8% 且出现反向信号 → 止盈出场\n")
	sb.WriteString("- 亏损达到止损价 → 立即止损\n")
	sb.WriteString("- 其他情况 → 继续持有（不要被短期波动吓跑！）\n\n")
	sb.WriteString("**C. 短期波动的处理**\n")
	sb.WriteString("- 如果3分钟RSI超卖（<40）但4小时趋势完好 → **继续持有**，甚至考虑加仓！\n")
	sb.WriteString("- 如果3分钟跌破EMA20但4小时趋势完好 → **继续持有**，这是正常回调！\n")
	sb.WriteString("- 如果15分钟MACD转负但4小时趋势完好 → **继续持有**，耐心等待！\n\n")
	sb.WriteString("**⚠️ 关键原则：只有4小时趋势反转才平仓，不要被短期波动吓跑！**\n\n")

	sb.WriteString("### Step 4: 筛选新机会\n")
	sb.WriteString("- 先判断4小时趋势，筛选出多头/空头币种\n")
	sb.WriteString("- 对每个币种计算信号质量评分\n")
	sb.WriteString("- 选择评分最高且≥75分的1-2个币种\n")
	sb.WriteString("- 如果没有高质量信号 → 观望！\n\n")

	sb.WriteString("### Step 5: 执行决策\n")
	sb.WriteString("- 优先平仓/调整现有持仓\n")
	sb.WriteString("- 再考虑开新仓（检查冷却期）\n")
	sb.WriteString("- 确保风险回报比≥2.5:1\n")
	sb.WriteString("- 在reasoning中写明：趋势判断、满足的信号、质量评分\n\n")

	// ==================== 输出格式 ====================
	sb.WriteString("## 📤 输出格式\n\n")
	sb.WriteString("**先输出思维链（纯文本），再输出JSON数组**\n\n")
	sb.WriteString("JSON示例：\n")
	sb.WriteString("```json\n")
	sb.WriteString("[\n")
	sb.WriteString(fmt.Sprintf("  {\"symbol\": \"BTCUSDT\", \"action\": \"open_long\", \"leverage\": 50, \"position_size_usd\": %.0f, \"stop_loss\": 110000, \"take_profit\": 118000, \"confidence\": 85, \"risk_usd\": 200, \"reasoning\": \"4h多头趋势（EMA20>EMA50，MACD>0），15m突破EMA20且MACD金叉，RSI14=58健康，信号质量90分\"},\n", accountEquity*5))
	sb.WriteString("  {\"symbol\": \"ETHUSDT\", \"action\": \"close_long\", \"reasoning\": \"4h趋势反转，EMA20死叉，止盈出场\"}\n")
	sb.WriteString("]\n")
	sb.WriteString("```\n\n")

	sb.WriteString("**字段说明**:\n")
	sb.WriteString("- `action`: open_long | open_short | close_long | close_short | hold | wait\n")
	sb.WriteString("- `confidence`: 信心度0-100\n")
	sb.WriteString("- `reasoning`: 必须包含：趋势判断、满足的信号、质量评分\n")
	sb.WriteString("- 开仓时必填: leverage, position_size_usd, stop_loss, take_profit, confidence, risk_usd\n\n")

	// ==================== 最后提醒 ====================
	sb.WriteString("## ⚠️ 最后提醒\n\n")
	sb.WriteString("记住：\n")
	sb.WriteString("1. 永远先看4小时趋势！\n")
	sb.WriteString("2. 信号质量<75分不要开仓！\n")
	sb.WriteString("3. 不要频繁交易，耐心持有！\n")
	sb.WriteString("4. 每次开仓前问自己：满足几个信号？评分多少？\n\n")

	return sb.String()
}

// buildUserPrompt 构建 User Prompt（动态数据）
func buildUserPrompt(ctx *Context) string {
	var sb strings.Builder

	// 系统状态
	sb.WriteString(fmt.Sprintf("**时间**: %s | **周期**: #%d | **运行**: %d分钟\n\n",
		ctx.CurrentTime, ctx.CallCount, ctx.RuntimeMinutes))

	// BTC 市场
	if btcData, hasBTC := ctx.MarketDataMap["BTCUSDT"]; hasBTC {
		sb.WriteString(fmt.Sprintf("**BTC**: %.2f (1h: %+.2f%%, 4h: %+.2f%%) | MACD: %.4f | RSI: %.2f\n\n",
			btcData.CurrentPrice, btcData.PriceChange1h, btcData.PriceChange4h,
			btcData.CurrentMACD, btcData.CurrentRSI7))
	}

	// 账户
	sb.WriteString(fmt.Sprintf("**账户**: 净值%.2f | 余额%.2f (%.1f%%) | 盈亏%+.2f%% | 保证金%.1f%% | 持仓%d个\n\n",
		ctx.Account.TotalEquity,
		ctx.Account.AvailableBalance,
		(ctx.Account.AvailableBalance/ctx.Account.TotalEquity)*100,
		ctx.Account.TotalPnLPct,
		ctx.Account.MarginUsedPct,
		ctx.Account.PositionCount))

	// 持仓（完整市场数据）
	if len(ctx.Positions) > 0 {
		sb.WriteString("## 当前持仓\n")
		for i, pos := range ctx.Positions {
			sb.WriteString(fmt.Sprintf("%d. %s %s | 入场价%.4f 当前价%.4f | 盈亏%+.2f%% | 杠杆%dx | 保证金%.0f | 强平价%.4f\n\n",
				i+1, pos.Symbol, strings.ToUpper(pos.Side),
				pos.EntryPrice, pos.MarkPrice, pos.UnrealizedPnLPct,
				pos.Leverage, pos.MarginUsed, pos.LiquidationPrice))

			// 使用FormatMarketData输出完整市场数据
			if marketData, ok := ctx.MarketDataMap[pos.Symbol]; ok {
				sb.WriteString(market.Format(marketData))
				sb.WriteString("\n")
			}
		}
	} else {
		sb.WriteString("**当前持仓**: 无\n\n")
	}

	// 候选币种（完整市场数据）
	sb.WriteString(fmt.Sprintf("## 候选币种 (%d个)\n\n", len(ctx.MarketDataMap)))
	displayedCount := 0
	for _, coin := range ctx.CandidateCoins {
		marketData, hasData := ctx.MarketDataMap[coin.Symbol]
		if !hasData {
			continue
		}
		displayedCount++

		sourceTags := ""
		if len(coin.Sources) > 1 {
			sourceTags = " (AI500+OI_Top双重信号)"
		} else if len(coin.Sources) == 1 && coin.Sources[0] == "oi_top" {
			sourceTags = " (OI_Top持仓增长)"
		}

		// 使用FormatMarketData输出完整市场数据
		sb.WriteString(fmt.Sprintf("### %d. %s%s\n\n", displayedCount, coin.Symbol, sourceTags))
		sb.WriteString(market.Format(marketData))
		sb.WriteString("\n")
	}
	sb.WriteString("\n")

	// 夏普比率（直接传值，不要复杂格式化）
	if ctx.Performance != nil {
		// 直接从interface{}中提取SharpeRatio
		type PerformanceData struct {
			SharpeRatio float64 `json:"sharpe_ratio"`
		}
		var perfData PerformanceData
		if jsonData, err := json.Marshal(ctx.Performance); err == nil {
			if err := json.Unmarshal(jsonData, &perfData); err == nil {
				sb.WriteString(fmt.Sprintf("## 📊 夏普比率: %.2f\n\n", perfData.SharpeRatio))
			}
		}
	}

	sb.WriteString("---\n\n")
	sb.WriteString("现在请分析并输出决策（思维链 + JSON）\n")

	return sb.String()
}

// parseFullDecisionResponse 解析AI的完整决策响应
func parseFullDecisionResponse(aiResponse string, accountEquity float64) (*FullDecision, error) {
	// 1. 提取思维链
	cotTrace := extractCoTTrace(aiResponse)

	// 2. 提取JSON决策列表
	decisions, err := extractDecisions(aiResponse)
	if err != nil {
		return &FullDecision{
			CoTTrace:  cotTrace,
			Decisions: []Decision{},
		}, fmt.Errorf("提取决策失败: %w\n\n=== AI思维链分析 ===\n%s", err, cotTrace)
	}

	// 3. 验证决策
	if err := validateDecisions(decisions, accountEquity); err != nil {
		return &FullDecision{
			CoTTrace:  cotTrace,
			Decisions: decisions,
		}, fmt.Errorf("决策验证失败: %w\n\n=== AI思维链分析 ===\n%s", err, cotTrace)
	}

	return &FullDecision{
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
func extractDecisions(response string) ([]Decision, error) {
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

	// 🔧 修复常见的JSON格式错误：缺少引号的字段值
	// 匹配: "reasoning": 内容"}  或  "reasoning": 内容}  (没有引号)
	// 修复为: "reasoning": "内容"}
	// 使用简单的字符串扫描而不是正则表达式
	jsonContent = fixMissingQuotes(jsonContent)

	// 解析JSON
	var decisions []Decision
	if err := json.Unmarshal([]byte(jsonContent), &decisions); err != nil {
		return nil, fmt.Errorf("JSON解析失败: %w\nJSON内容: %s", err, jsonContent)
	}

	return decisions, nil
}

// fixMissingQuotes 替换中文引号为英文引号（避免输入法自动转换）
func fixMissingQuotes(jsonStr string) string {
	jsonStr = strings.ReplaceAll(jsonStr, "\u201c", "\"") // "
	jsonStr = strings.ReplaceAll(jsonStr, "\u201d", "\"") // "
	jsonStr = strings.ReplaceAll(jsonStr, "\u2018", "'")  // '
	jsonStr = strings.ReplaceAll(jsonStr, "\u2019", "'")  // '
	return jsonStr
}

// validateDecisions 验证所有决策（需要账户信息）
func validateDecisions(decisions []Decision, accountEquity float64) error {
	for i, decision := range decisions {
		if err := validateDecision(&decision, accountEquity); err != nil {
			return fmt.Errorf("决策 #%d 验证失败: %w", i+1, err)
		}
	}
	return nil
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
func validateDecision(d *Decision, accountEquity float64) error {
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
		// 根据币种判断杠杆上限和仓位价值上限
		maxLeverage := 20                       // 山寨币固定20倍
		maxPositionValue := accountEquity * 1.5 // 山寨币最多1.5倍账户净值
		if d.Symbol == "BTCUSDT" || d.Symbol == "ETHUSDT" {
			maxLeverage = 50                      // BTC和ETH固定50倍
			maxPositionValue = accountEquity * 10 // BTC/ETH最多10倍账户净值
		}

		if d.Leverage <= 0 || d.Leverage > maxLeverage {
			return fmt.Errorf("杠杆必须在1-%d之间（%s）: %d", maxLeverage, d.Symbol, d.Leverage)
		}
		if d.PositionSizeUSD <= 0 {
			return fmt.Errorf("仓位大小必须大于0: %.2f", d.PositionSizeUSD)
		}
		// 验证仓位价值上限（加1%容差以避免浮点数精度问题）
		tolerance := maxPositionValue * 0.01 // 1%容差
		if d.PositionSizeUSD > maxPositionValue+tolerance {
			if d.Symbol == "BTCUSDT" || d.Symbol == "ETHUSDT" {
				return fmt.Errorf("BTC/ETH单币种仓位价值不能超过%.0f USDT（10倍账户净值），实际: %.0f", maxPositionValue, d.PositionSizeUSD)
			} else {
				return fmt.Errorf("山寨币单币种仓位价值不能超过%.0f USDT（1.5倍账户净值），实际: %.0f", maxPositionValue, d.PositionSizeUSD)
			}
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
