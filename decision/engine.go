package decision

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"nofx/market"
	"nofx/mcp"
	"nofx/pool"
	"strings"
	"time"
)

// PositionInfo 持仓信息
type PositionInfo struct {
	Symbol           string                 `json:"symbol"`
	Side             string                 `json:"side"` // "long" or "short"
	EntryPrice       float64                `json:"entry_price"`
	MarkPrice        float64                `json:"mark_price"`
	Quantity         float64                `json:"quantity"`
	Leverage         int                    `json:"leverage"`
	UnrealizedPnL    float64                `json:"unrealized_pnl"`
	UnrealizedPnLPct float64                `json:"unrealized_pnl_pct"`
	LiquidationPrice float64                `json:"liquidation_price"`
	MarginUsed       float64                `json:"margin_used"`
	EntryTime        int64                  `json:"entry_time"` // 持仓首次出现时间戳（毫秒）
	HoldMinutes      int                    `json:"hold_minutes"`
	RiskUSD          float64                `json:"risk_usd,omitempty"`
	RiskPct          float64                `json:"risk_pct,omitempty"`
	Protection       *ProtectionPlan        `json:"protection,omitempty"`
	ProtectionStatus *PositionProtectionLog `json:"protection_status,omitempty"`
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
	CurrentTime     string                  `json:"current_time"`
	RuntimeMinutes  int                     `json:"runtime_minutes"`
	CallCount       int                     `json:"call_count"`
	Account         AccountInfo             `json:"account"`
	Positions       []PositionInfo          `json:"positions"`
	CandidateCoins  []CandidateCoin         `json:"candidate_coins"`
	MarketDataMap   map[string]*market.Data `json:"-"` // 不序列化，但内部使用
	OITopDataMap    map[string]*OITopData   `json:"-"` // OI Top数据映射
	Performance     interface{}             `json:"-"` // 历史表现分析（logger.PerformanceAnalysis）
	BTCETHLeverage  int                     `json:"-"` // BTC/ETH杠杆倍数（从配置读取）
	AltcoinLeverage int                     `json:"-"` // 山寨币杠杆倍数（从配置读取）
	Risk            *RiskSnapshot           `json:"risk_snapshot,omitempty"`
}

// Decision AI的交易决策
type Decision struct {
	Symbol          string          `json:"symbol"`
	Action          string          `json:"action"` // "open_long", "open_short", "close_long", "close_short", "hold", "wait"
	Leverage        int             `json:"leverage,omitempty"`
	PositionSizeUSD float64         `json:"position_size_usd,omitempty"`
	StopLoss        float64         `json:"stop_loss,omitempty"`
	TakeProfit      float64         `json:"take_profit,omitempty"`
	Confidence      int             `json:"confidence,omitempty"` // 信心度 (0-100)
	RiskUSD         float64         `json:"risk_usd,omitempty"`   // 最大美元风险
	Reasoning       string          `json:"reasoning"`
	Protection      *ProtectionPlan `json:"protection,omitempty"`
}

// ProtectionPlan 趋势持仓的风险保护与跟踪参数
type ProtectionPlan struct {
	Strategy            string  `json:"strategy,omitempty"`              // 例如: "trend_follow"
	MinHoldMinutes      int     `json:"min_hold_minutes,omitempty"`      // 最少持仓时长
	BreakEvenTriggerPct float64 `json:"breakeven_trigger_pct,omitempty"` // 价格相对入场上涨/下跌多少%(不含杠杆)触发保本
	BreakEvenOffsetPct  float64 `json:"breakeven_offset_pct,omitempty"`  // 保本止损相对入场价偏移百分比
	TrailActivationPct  float64 `json:"trail_activation_pct,omitempty"`  // 盈利达到多少%启动追踪止损
	TrailDistancePct    float64 `json:"trail_distance_pct,omitempty"`    // 追踪止损与价格的距离百分比
	ExitMode            string  `json:"exit_mode,omitempty"`             // fixed | trailing | reversal
	ReversalTriggerPct  float64 `json:"reversal_trigger_pct,omitempty"`  // 反转止盈触发阈值
	Notes               string  `json:"notes,omitempty"`                 // 可选说明
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
	systemPrompt := buildSystemPrompt(ctx.Account.TotalEquity, ctx.BTCETHLeverage, ctx.AltcoinLeverage)
	userPrompt := buildUserPrompt(ctx)

	// 3. 调用AI API（使用 system + user prompt）
	aiResponse, err := mcp.CallWithMessages(systemPrompt, userPrompt)
	if err != nil {
		return nil, fmt.Errorf("调用AI API失败: %w", err)
	}

	// 4. 解析AI响应
	decision, err := parseFullDecisionResponse(aiResponse, ctx)
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
func buildSystemPrompt(accountEquity float64, btcEthLeverage, altcoinLeverage int) string {
	var sb strings.Builder

	// === 核心使命 ===
	sb.WriteString("你是专业的加密货币交易AI，在币安合约市场进行自主交易。\n\n")
	sb.WriteString("# 🎯 核心目标\n\n")
	sb.WriteString("**最大化夏普比率（Sharpe Ratio）**\n\n")
	sb.WriteString("夏普比率 = 平均收益 / 收益波动率\n\n")
	sb.WriteString("**这意味着**：\n")
	sb.WriteString("- ✅ 高质量交易（高胜率、大盈亏比）→ 提升夏普\n")
	sb.WriteString("- ✅ 稳定收益、控制回撤 → 提升夏普\n")
	sb.WriteString("- ✅ 耐心持仓、让利润奔跑 → 提升夏普\n")
	sb.WriteString("- ❌ 频繁交易、小盈小亏 → 增加波动，严重降低夏普\n")
	sb.WriteString("- ❌ 过度交易、手续费损耗 → 直接亏损\n")
	sb.WriteString("- ❌ 过早平仓、频繁进出 → 错失大行情\n\n")
	sb.WriteString("**关键认知**: 系统每3分钟扫描一次，但不意味着每次都要交易！\n")
	sb.WriteString("大多数时候应该是 `wait` 或 `hold`，只在极佳机会时才开仓。\n\n")

	// === 硬约束（风险控制）===
	sb.WriteString("# ⚖️ 硬约束（风险控制）\n\n")
	sb.WriteString("1. **风险回报比**: 必须 ≥ 1:3（冒1%风险，赚3%+收益）\n")
	sb.WriteString("2. **最多持仓**: 3个币种（质量>数量）\n")
	sb.WriteString(fmt.Sprintf("3. **单币仓位**: 山寨%.0f-%.0f U(%dx杠杆) | BTC/ETH %.0f-%.0f U(%dx杠杆)\n",
		accountEquity*0.8, accountEquity*1.5, altcoinLeverage, accountEquity*5, accountEquity*10, btcEthLeverage))
	sb.WriteString("4. **保证金**: 总使用率 ≤ 90%\n\n")

	// === 做空激励 ===
	sb.WriteString("# 📉 做多做空平衡\n\n")
	sb.WriteString("**重要**: 下跌趋势做空的利润 = 上涨趋势做多的利润\n\n")
	sb.WriteString("- 上涨趋势 → 做多\n")
	sb.WriteString("- 下跌趋势 → 做空\n")
	sb.WriteString("- 震荡市场 → 观望\n\n")
	sb.WriteString("**不要有做多偏见！做空是你的核心工具之一**\n\n")

	// === 交易频率认知 ===
	sb.WriteString("# ⏱️ 交易频率认知\n\n")
	sb.WriteString("**量化标准**:\n")
	sb.WriteString("- 优秀交易员：每天2-4笔 = 每小时0.1-0.2笔\n")
	sb.WriteString("- 过度交易：每小时>2笔 = 严重问题\n")
	sb.WriteString("- 最佳节奏：开仓后持有至少30-60分钟\n\n")
	sb.WriteString("**自查**:\n")
	sb.WriteString("如果你发现自己每个周期都在交易 → 说明标准太低\n")
	sb.WriteString("如果你发现持仓<30分钟就平仓 → 说明太急躁\n\n")

	// === 趋势持仓纪律 ===
	sb.WriteString("# 📈 趋势持仓纪律\n\n")
	sb.WriteString("你是一名顺势交易员，目标是在趋势中“拿得住”。\n\n")
	sb.WriteString("- 趋势未反转前不要提前离场，避免短线频繁换手\n")
	sb.WriteString("- 参考 3min / 4h / 1d 多周期，只有级别共振才开仓\n")
	sb.WriteString("- 默认策略：除非出现强烈反向信号或风控触发，目标持仓 30-60 分钟，并在行情反转或达标时主动处理\n")
	sb.WriteString("- 收益≥预期后，通过追踪止损锁定利润，让盈利继续扩张\n")
	sb.WriteString("- 如果信号减弱但仍在趋势中，降低仓位而不是立刻清仓\n\n")

	// === 风险保护 ===
	sb.WriteString("# 🛡️ 风险保护与跟踪计划\n\n")
	sb.WriteString("开仓时必须提供 `protection` 字段，指导系统动态风控。所有百分比均指标的价格相对入场价的变动（不放大杠杆）。\n\n")
	sb.WriteString("- `min_hold_minutes` 建议≥30，根据趋势力度设定，并说明何时允许提前退出\n")
	sb.WriteString("- `breakeven_trigger_pct` ≈ 3，表示行情顺利走出约3%后，把止损抬到保本\n")
	sb.WriteString("- `breakeven_offset_pct` 用于留出缓冲（如0.2表示保本止损设置在入场价上方0.2%）\n")
	sb.WriteString("- `trail_activation_pct` ≥ breakeven_trigger_pct，盈利达到该阈值后启用追踪止损\n")
	sb.WriteString("- `trail_distance_pct` 建议 0.8~1.5，表示追踪止损与最新价保持的百分比距离\n")
	sb.WriteString("- `exit_mode`: `fixed`（使用明确止盈价）、`trailing`（依赖追踪止损）、`reversal`（等待趋势反转时由你主动发出平仓指令，可结合 `reversal_trigger_pct`）\n")
	sb.WriteString("- `reversal_trigger_pct`：当选择`reversal`时，用于量化“确认反转”所需的回撤或背离幅度\n")
	sb.WriteString("- 盈利扩张过程中必须持续动态更新止损：`trailing` 模式要随着价格推升止损，`reversal` 模式也需在条件满足时主动下移止损并准备平仓\n")
	sb.WriteString("- 根据趋势强弱可在 `notes` 中说明调整逻辑，并在每轮输出时写明是否调整保护计划及原因\n\n")

	// === 开仓信号强度 ===
	sb.WriteString("# 🎯 开仓标准（严格）\n\n")
	sb.WriteString("只在**强信号**时开仓，不确定就观望。\n\n")
	sb.WriteString("**你拥有的完整数据**：\n")
	sb.WriteString("- 📊 **原始序列**：3分钟价格序列(MidPrices数组) + 4小时/日线结构化上下文\n")
	sb.WriteString("- 📈 **技术序列**：EMA20 / MACD / RSI7 / RSI14 等多周期指标\n")
	sb.WriteString("- 💰 **资金序列**：成交量序列、持仓量(OI)序列、资金费率\n")
	sb.WriteString("- 🧭 **派生信号**：系统提供的趋势JSON，包括日线/4小时趋势、斜率、波动率比等\n")
	sb.WriteString("- 🎯 **筛选标记**：AI500评分 / OI_Top排名（如果有标注）\n\n")
	sb.WriteString("**分析方法**（完全由你自主决定）：\n")
	sb.WriteString("- 自由运用序列数据，你可以做但不限于趋势分析、K线形态识别、支撑阻力、技术阻力位、斐波那契、波动带计算\n")
	sb.WriteString("- 多维度交叉验证（价格+量+OI+指标+序列形态）\n")
	sb.WriteString("- 用你认为最有效的方法发现高确定性机会\n")
	sb.WriteString("- 综合信心度 ≥ 75 才开仓\n\n")
	sb.WriteString("**避免低质量信号**：\n")
	sb.WriteString("- 单一维度（只看一个指标）\n")
	sb.WriteString("- 相互矛盾（涨但量萎缩）\n")
	sb.WriteString("- 横盘震荡\n")
	sb.WriteString("- 刚平仓不久（<15分钟）\n\n")

	// === 夏普比率自我进化 ===
	sb.WriteString("# 🧬 夏普比率自我进化\n\n")
	sb.WriteString("每次你会收到**夏普比率**作为绩效反馈（周期级别）：\n\n")
	sb.WriteString("**夏普比率 < -0.5** (持续亏损):\n")
	sb.WriteString("  → 🛑 停止交易，连续观望至少6个周期（18分钟）\n")
	sb.WriteString("  → 🔍 深度反思：\n")
	sb.WriteString("     • 交易频率过高？（每小时>2次就是过度）\n")
	sb.WriteString("     • 持仓时间过短？（<30分钟就是过早平仓）\n")
	sb.WriteString("     • 信号强度不足？（信心度<75）\n")
	sb.WriteString("     • 是否在做空？（单边做多是错误的）\n\n")
	sb.WriteString("**夏普比率 -0.5 ~ 0** (轻微亏损):\n")
	sb.WriteString("  → ⚠️ 严格控制：只做信心度>80的交易\n")
	sb.WriteString("  → 减少交易频率：每小时最多1笔新开仓\n")
	sb.WriteString("  → 耐心持仓：至少持有30分钟以上\n\n")
	sb.WriteString("**夏普比率 0 ~ 0.7** (正收益):\n")
	sb.WriteString("  → ✅ 维持当前策略\n\n")
	sb.WriteString("**夏普比率 > 0.7** (优异表现):\n")
	sb.WriteString("  → 🚀 可适度扩大仓位\n\n")
	sb.WriteString("**关键**: 夏普比率是唯一指标，它会自然惩罚频繁交易和过度进出。\n\n")

	// === 决策流程 ===
	sb.WriteString("# 📋 决策流程\n\n")
	sb.WriteString("1. **分析夏普比率**: 当前策略是否有效？需要调整吗？\n")
	sb.WriteString("2. **评估持仓**: 趋势是否改变？是否需要抬升保本、缩紧追踪止损、部分减仓？\n")
	sb.WriteString("3. **寻找新机会**: 有强信号吗？多空机会？\n")
	sb.WriteString("4. **输出决策**: 明确说明保护计划是否调整，并输出思维链 + JSON\n\n")

	// === 输出格式 ===
	sb.WriteString("# 📤 输出格式\n\n")
	sb.WriteString("**第一步: 思维链（纯文本）**\n")
	sb.WriteString("简洁分析你的思考过程\n\n")
	sb.WriteString("**第二步: JSON决策数组**\n\n")
	sb.WriteString("```json\n[\n")
	sb.WriteString(fmt.Sprintf("  {\"symbol\": \"BTCUSDT\", \"action\": \"open_short\", \"leverage\": %d, \"position_size_usd\": %.0f, \"stop_loss\": 97000, \"take_profit\": 91000, \"confidence\": 85, \"risk_usd\": 300, \"protection\": {\"strategy\": \"trend_follow\", \"min_hold_minutes\": 60, \"breakeven_trigger_pct\": 3.0, \"breakeven_offset_pct\": 0.2, \"trail_activation_pct\": 4.0, \"trail_distance_pct\": 1.0, \"exit_mode\": \"trailing\"}, \"reasoning\": \"下跌趋势+MACD死叉\"},\n", btcEthLeverage, accountEquity*5))
	sb.WriteString("  {\"symbol\": \"ETHUSDT\", \"action\": \"close_long\", \"reasoning\": \"止盈离场\"}\n")
	sb.WriteString("]\n```\n\n")
	sb.WriteString("**字段说明**:\n")
	sb.WriteString("- `action`: open_long | open_short | close_long | close_short | hold | wait\n")
	sb.WriteString("- `confidence`: 0-100（开仓建议≥75）\n")
	sb.WriteString("- 开仓时必填: leverage, position_size_usd, stop_loss, take_profit, confidence, risk_usd, reasoning, protection\n")
	sb.WriteString("- `protection`: 趋势风控计划，须包含 `min_hold_minutes`、`breakeven_trigger_pct`、`breakeven_offset_pct`、`trail_activation_pct`、`trail_distance_pct`，并根据需要设置 `exit_mode`/`reversal_trigger_pct`\n\n")

	// === 关键提醒 ===
	sb.WriteString("---\n\n")
	sb.WriteString("**记住**: \n")
	sb.WriteString("- 目标是夏普比率，不是交易频率\n")
	sb.WriteString("- 做空 = 做多，都是赚钱工具\n")
	sb.WriteString("- 宁可错过，不做低质量交易\n")
	sb.WriteString("- 风险回报比1:3是底线\n")

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
	if ctx.Risk != nil && ctx.Risk.PositionCount > 0 {
		sb.WriteString("## 风险快照\n")
		sb.WriteString(fmt.Sprintf("- 总风险: %.2f USDT (%.2f%% 账户净值)\n", ctx.Risk.TotalRiskUSD, ctx.Risk.TotalRiskPct))
		sb.WriteString(fmt.Sprintf("- 最大单笔风险: %.2f%%\n", ctx.Risk.MaxSingleRiskPct))
		sb.WriteString(fmt.Sprintf("- 多头敞口: %.2f USDT | 空头敞口: %.2f USDT | 净敞口: %.2f USDT\n\n",
			ctx.Risk.LongExposureUSD, ctx.Risk.ShortExposureUSD, ctx.Risk.NetExposureUSD))

		if riskJSON, err := json.MarshalIndent(ctx.Risk, "", "  "); err == nil {
			sb.WriteString("```json\n")
			sb.WriteString(string(riskJSON))
			sb.WriteString("\n```\n\n")
		}
	}

	if len(ctx.Positions) > 0 {
		sb.WriteString("## 当前持仓\n")
		structuredPositions := make([]map[string]interface{}, 0, len(ctx.Positions))

		for i, pos := range ctx.Positions {
			// 计算持仓时长
			holdingDuration := ""
			holdMinutes := 0
			if pos.EntryTime > 0 {
				durationMs := time.Now().UnixMilli() - pos.EntryTime
				holdMinutes = int(durationMs / (1000 * 60))
				if holdMinutes < 60 {
					holdingDuration = fmt.Sprintf(" | 持仓时长%d分钟", holdMinutes)
				} else {
					durationHour := holdMinutes / 60
					durationMinRemainder := holdMinutes % 60
					holdingDuration = fmt.Sprintf(" | 持仓时长%d小时%d分钟", durationHour, durationMinRemainder)
				}
			}

			sb.WriteString(fmt.Sprintf("%d. %s %s | 入场价%.4f 当前价%.4f | 盈亏%+.2f%% | 杠杆%dx | 保证金%.0f | 强平价%.4f%s\n\n",
				i+1, pos.Symbol, strings.ToUpper(pos.Side),
				pos.EntryPrice, pos.MarkPrice, pos.UnrealizedPnLPct,
				pos.Leverage, pos.MarginUsed, pos.LiquidationPrice, holdingDuration))

			// 使用FormatMarketData输出完整市场数据
			if marketData, ok := ctx.MarketDataMap[pos.Symbol]; ok {
				sb.WriteString(market.Format(marketData))
				sb.WriteString("\n")
			}

			positionMap := map[string]interface{}{
				"symbol":         pos.Symbol,
				"side":           pos.Side,
				"entry_price":    pos.EntryPrice,
				"mark_price":     pos.MarkPrice,
				"leverage":       pos.Leverage,
				"quantity":       pos.Quantity,
				"hold_minutes":   holdMinutes,
				"risk_usd":       pos.RiskUSD,
				"risk_pct":       pos.RiskPct,
				"unrealized_pct": pos.UnrealizedPnLPct,
			}

			if pos.Protection != nil {
				positionMap["protection"] = pos.Protection
			}
			if pos.ProtectionStatus != nil {
				positionMap["protection_status"] = pos.ProtectionStatus
			}

			structuredPositions = append(structuredPositions, positionMap)
		}

		if posJSON, err := json.MarshalIndent(structuredPositions, "", "  "); err == nil {
			sb.WriteString("```json\n")
			sb.WriteString(string(posJSON))
			sb.WriteString("\n```\n\n")
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
func parseFullDecisionResponse(aiResponse string, ctx *Context) (*FullDecision, error) {
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
	if err := validateDecisions(decisions, ctx); err != nil {
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

// validateDecisions 验证所有决策（需要账户信息和杠杆配置）
func validateDecisions(decisions []Decision, ctx *Context) error {
	for i := range decisions {
		if err := validateDecision(&decisions[i], ctx); err != nil {
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
func validateDecision(d *Decision, ctx *Context) error {
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
		// 根据币种使用配置的杠杆上限
		maxLeverage := ctx.AltcoinLeverage                // 山寨币使用配置的杠杆
		maxPositionValue := ctx.Account.TotalEquity * 1.5 // 山寨币最多1.5倍账户净值
		if d.Symbol == "BTCUSDT" || d.Symbol == "ETHUSDT" {
			maxLeverage = ctx.BTCETHLeverage                // BTC和ETH使用配置的杠杆
			maxPositionValue = ctx.Account.TotalEquity * 10 // BTC/ETH最多10倍账户净值
		}

		if d.Leverage <= 0 {
			return fmt.Errorf("杠杆必须在1-%d之间（%s，当前配置上限%d倍）: %d", maxLeverage, d.Symbol, maxLeverage, d.Leverage)
		}
		if d.Leverage > maxLeverage {
			log.Printf("⚠️  杠杆超出上限，已将 %s 的杠杆从 %dx 调整为配置上限 %dx", d.Symbol, d.Leverage, maxLeverage)
			d.Leverage = maxLeverage
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

		// 验证风险回报比（必须≥1:3）
		// 计算入场价（假设当前市价）
		var entryPrice float64
		if d.Action == "open_long" {
			// 做多：入场价在止损和止盈之间
			entryPrice = d.StopLoss + (d.TakeProfit-d.StopLoss)*0.2 // 假设在20%位置入场
		} else {
			// 做空：入场价在止损和止盈之间
			entryPrice = d.StopLoss - (d.StopLoss-d.TakeProfit)*0.2 // 假设在20%位置入场
		}

		var riskPercent, rewardPercent, riskRewardRatio float64
		if d.Action == "open_long" {
			riskPercent = (entryPrice - d.StopLoss) / entryPrice * 100
			rewardPercent = (d.TakeProfit - entryPrice) / entryPrice * 100
			if riskPercent > 0 {
				riskRewardRatio = rewardPercent / riskPercent
			}
		} else {
			riskPercent = (d.StopLoss - entryPrice) / entryPrice * 100
			rewardPercent = (entryPrice - d.TakeProfit) / entryPrice * 100
			if riskPercent > 0 {
				riskRewardRatio = rewardPercent / riskPercent
			}
		}

		// 硬约束：风险回报比必须≥3.0
		if riskRewardRatio < 3.0 {
			return fmt.Errorf("风险回报比过低(%.2f:1)，必须≥3.0:1 [风险:%.2f%% 收益:%.2f%%] [止损:%.2f 止盈:%.2f]",
				riskRewardRatio, riskPercent, rewardPercent, d.StopLoss, d.TakeProfit)
		}

		if d.Protection == nil {
			return fmt.Errorf("开仓必须提供protection风控计划")
		}
		if err := validateProtectionPlan(d.Protection); err != nil {
			return fmt.Errorf("protection设置无效: %w", err)
		}
		// 验证风险敞口不超过总资金3%
		marketData := ctx.MarketDataMap[d.Symbol]
		if marketData == nil {
			return fmt.Errorf("缺少%s的市场数据，无法验证风险敞口", d.Symbol)
		}

		currentPrice := marketData.CurrentPrice
		if currentPrice <= 0 {
			return fmt.Errorf("%s 当前价格无效，无法验证风险敞口", d.Symbol)
		}

		priceDiff := math.Abs(currentPrice - d.StopLoss)
		if priceDiff == 0 {
			return fmt.Errorf("%s 止损价与入场价相同，风险计算无意义", d.Symbol)
		}

		riskUSD := (priceDiff / currentPrice) * d.PositionSizeUSD
		maxAllowedRisk := ctx.Account.TotalEquity * 0.03
		if riskUSD > maxAllowedRisk {
			return fmt.Errorf("%s 风险敞口 %.2f USDT 超出账户净值3%%限制 (最大允许 %.2f)", d.Symbol, riskUSD, maxAllowedRisk)
		}
		d.RiskUSD = riskUSD
	}

	return nil
}

// validateProtectionPlan 验证趋势风控计划
func validateProtectionPlan(plan *ProtectionPlan) error {
	if plan.Strategy == "" {
		plan.Strategy = "trend_follow"
	}
	exitMode := strings.ToLower(plan.ExitMode)
	if exitMode == "" {
		exitMode = "trailing"
	}
	switch exitMode {
	case "fixed", "trailing", "reversal":
		plan.ExitMode = exitMode
	default:
		return fmt.Errorf("exit_mode 无效: %s，可选 fixed/trailing/reversal", plan.ExitMode)
	}
	if plan.MinHoldMinutes < 20 {
		return fmt.Errorf("min_hold_minutes 过低(%d)，建议至少20分钟以避免噪音交易", plan.MinHoldMinutes)
	}
	if plan.BreakEvenTriggerPct < 2.0 || plan.BreakEvenTriggerPct > 10 {
		return fmt.Errorf("breakeven_trigger_pct %.2f%% 不在合理范围(2-10%%)", plan.BreakEvenTriggerPct)
	}
	if plan.BreakEvenOffsetPct < -1.0 || plan.BreakEvenOffsetPct > 1.0 {
		return fmt.Errorf("breakeven_offset_pct %.2f%% 超出范围(-1%%~1%%)", plan.BreakEvenOffsetPct)
	}
	if plan.TrailActivationPct == 0 {
		plan.TrailActivationPct = plan.BreakEvenTriggerPct
	}
	if plan.TrailActivationPct < plan.BreakEvenTriggerPct {
		return fmt.Errorf("trail_activation_pct %.2f%% 必须 ≥ breakeven_trigger_pct %.2f%%", plan.TrailActivationPct, plan.BreakEvenTriggerPct)
	}
	if exitMode == "reversal" {
		if plan.ReversalTriggerPct <= 0 {
			return fmt.Errorf("reversal_trigger_pct 必须大于0，用于量化反转止盈触发条件")
		}
		// 反转策略下若未使用追踪止损，可允许distance为0
		if plan.TrailDistancePct < 0 || plan.TrailDistancePct > 5 {
			return fmt.Errorf("trail_distance_pct %.2f%% 无效，需在0-5%%之间", plan.TrailDistancePct)
		}
	} else {
		if plan.TrailDistancePct <= 0 || plan.TrailDistancePct > 5 {
			return fmt.Errorf("trail_distance_pct %.2f%% 无效，需在0-5%%之间", plan.TrailDistancePct)
		}
	}
	return nil
}

// PositionProtectionLog 记录保护计划执行状态
type PositionProtectionLog struct {
	ExitMode           string  `json:"exit_mode,omitempty"`
	BreakevenApplied   bool    `json:"breakeven_applied"`
	CurrentStop        float64 `json:"current_stop,omitempty"`
	RegisteredAt       int64   `json:"registered_at,omitempty"`
	TrailActivationPct float64 `json:"trail_activation_pct,omitempty"`
	TrailDistancePct   float64 `json:"trail_distance_pct,omitempty"`
	Notes              string  `json:"notes,omitempty"`
}

// PositionExposure 风险敞口
type PositionExposure struct {
	Symbol      string  `json:"symbol"`
	Side        string  `json:"side"`
	NotionalUSD float64 `json:"notional_usd"`
	RiskUSD     float64 `json:"risk_usd"`
	RiskPct     float64 `json:"risk_pct"`
}

// RiskSnapshot 总风险快照
type RiskSnapshot struct {
	TotalRiskUSD     float64                      `json:"total_risk_usd"`
	TotalRiskPct     float64                      `json:"total_risk_pct"`
	MaxSingleRiskPct float64                      `json:"max_single_risk_pct"`
	LongExposureUSD  float64                      `json:"long_exposure_usd"`
	ShortExposureUSD float64                      `json:"short_exposure_usd"`
	NetExposureUSD   float64                      `json:"net_exposure_usd"`
	PositionCount    int                          `json:"position_count"`
	SymbolRisks      map[string]*PositionExposure `json:"symbol_risks"`
}
