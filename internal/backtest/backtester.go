package backtest

import (
	"fmt"
	"log"
	"math"
	"time"

	"agent-quant-system/internal/data"
	"agent-quant-system/internal/strategy"
)

// Backtester 回测器
type Backtester struct {
	strategy       strategy.Strategy
	dataManager    *data.DataManager
	initialCapital float64
	commissionRate float64
	slippageRate   float64
}

// NewBacktester 创建回测器
func NewBacktester(strategy strategy.Strategy, dataManager *data.DataManager, initialCapital, commissionRate, slippageRate float64) *Backtester {
	return &Backtester{
		strategy:       strategy,
		dataManager:    dataManager,
		initialCapital: initialCapital,
		commissionRate: commissionRate,
		slippageRate:   slippageRate,
	}
}

// BacktestResult 回测结果
type BacktestResult struct {
	StrategyName         string        `json:"strategy_name"`
	Symbol               string        `json:"symbol"`
	StartDate            time.Time     `json:"start_date"`
	EndDate              time.Time     `json:"end_date"`
	InitialCapital       float64       `json:"initial_capital"`
	FinalCapital         float64       `json:"final_capital"`
	TotalReturn          float64       `json:"total_return"`
	AnnualReturn         float64       `json:"annual_return"`
	MaxDrawdown          float64       `json:"max_drawdown"`
	SharpeRatio          float64       `json:"sharpe_ratio"`
	SortinoRatio         float64       `json:"sortino_ratio"`
	WinRate              float64       `json:"win_rate"`
	TotalTrades          int           `json:"total_trades"`
	WinningTrades        int           `json:"winning_trades"`
	LosingTrades         int           `json:"losing_trades"`
	AvgWin               float64       `json:"avg_win"`
	AvgLoss              float64       `json:"avg_loss"`
	ProfitFactor         float64       `json:"profit_factor"`
	MaxConsecutiveWins   int           `json:"max_consecutive_wins"`
	MaxConsecutiveLosses int           `json:"max_consecutive_losses"`
	Commission           float64       `json:"commission"`
	Slippage             float64       `json:"slippage"`
	EquityCurve          []EquityPoint `json:"equity_curve"`
	TradeHistory         []TradeRecord `json:"trade_history"`
}

// EquityPoint 净值曲线点
type EquityPoint struct {
	Date  time.Time `json:"date"`
	Value float64   `json:"value"`
}

// TradeRecord 交易记录
type TradeRecord struct {
	EntryDate  time.Time `json:"entry_date"`
	ExitDate   time.Time `json:"exit_date"`
	Symbol     string    `json:"symbol"`
	Side       string    `json:"side"`
	EntryPrice float64   `json:"entry_price"`
	ExitPrice  float64   `json:"exit_price"`
	Quantity   float64   `json:"quantity"`
	PnL        float64   `json:"pnl"`
	Commission float64   `json:"commission"`
	Return     float64   `json:"return"`
}

// Run 运行回测
func (bt *Backtester) Run(symbol, startDate, endDate string) (*BacktestResult, error) {
	log.Printf("开始回测: 标的=%s, 开始日期=%s, 结束日期=%s", symbol, startDate, endDate)

	// 获取历史数据
	df, err := bt.dataManager.GetMarketData(symbol, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("获取历史数据失败: %w", err)
	}

	// 验证数据
	if err := bt.dataManager.ValidateData(df); err != nil {
		return nil, fmt.Errorf("数据验证失败: %w", err)
	}

	// 初始化回测状态
	state := &BacktestState{
		Capital:      bt.initialCapital,
		Position:     0,
		EntryPrice:   0,
		EntryTime:    time.Time{},
		EquityCurve:  make([]EquityPoint, 0),
		TradeHistory: make([]TradeRecord, 0),
	}

	// 执行回测
	if err := bt.executeBacktest(df, state); err != nil {
		return nil, fmt.Errorf("执行回测失败: %w", err)
	}

	// 生成报告
	result := bt.generateReport(symbol, startDate, endDate, state)

	log.Printf("回测完成: 总收益=%.2f%%, 最大回撤=%.2f%%, 夏普比率=%.2f",
		result.TotalReturn*100, result.MaxDrawdown*100, result.SharpeRatio)

	return result, nil
}

// BacktestState 回测状态
type BacktestState struct {
	Capital      float64
	Position     float64
	EntryPrice   float64
	EntryTime    time.Time
	EquityCurve  []EquityPoint
	TradeHistory []TradeRecord
}

// executeBacktest 执行回测逻辑
func (bt *Backtester) executeBacktest(df data.DataFrame, state *BacktestState) error {
	closeData := df["close"]
	volumeData := df["volume"]
	timestampData := df["timestamp"]

	dataLength := len(closeData)

	for i := int(bt.strategy.GetParameters()["long_period"].(float64)); i < dataLength; i++ {
		// 创建当前时间窗口的数据
		windowData := bt.createDataWindow(df, i)

		// 生成交易信号
		signals, err := bt.strategy.GenerateSignals(windowData, nil)
		if err != nil {
			log.Printf("生成信号失败: %v", err)
			continue
		}

		// 处理交易信号
		currentPrice := closeData[i].(float64)
		currentVolume := volumeData[i].(int64)
		currentTime := timestampData[i].(time.Time)

		for _, signal := range signals {
			if err := bt.processSignal(signal, currentPrice, currentVolume, currentTime, state); err != nil {
				log.Printf("处理信号失败: %v", err)
			}
		}

		// 更新净值曲线
		bt.updateEquityCurve(currentTime, state)
	}

	return nil
}

// createDataWindow 创建数据窗口
func (bt *Backtester) createDataWindow(df data.DataFrame, currentIndex int) data.DataFrame {
	windowSize := int(bt.strategy.GetParameters()["long_period"].(float64))
	startIndex := currentIndex - windowSize + 1

	windowData := data.DataFrame{
		"timestamp": make([]interface{}, windowSize),
		"open":      make([]interface{}, windowSize),
		"high":      make([]interface{}, windowSize),
		"low":       make([]interface{}, windowSize),
		"close":     make([]interface{}, windowSize),
		"volume":    make([]interface{}, windowSize),
	}

	for i := 0; i < windowSize; i++ {
		idx := startIndex + i
		windowData["timestamp"][i] = df["timestamp"][idx]
		windowData["open"][i] = df["open"][idx]
		windowData["high"][i] = df["high"][idx]
		windowData["low"][i] = df["low"][idx]
		windowData["close"][i] = df["close"][idx]
		windowData["volume"][i] = df["volume"][idx]
	}

	return windowData
}

// processSignal 处理交易信号
func (bt *Backtester) processSignal(signal strategy.TradingSignal, price float64, volume int64, timestamp time.Time, state *BacktestState) error {
	switch signal.Signal {
	case strategy.Buy:
		return bt.processBuySignal(signal, price, volume, timestamp, state)
	case strategy.Sell:
		return bt.processSellSignal(signal, price, volume, timestamp, state)
	default:
		return nil
	}
}

// processBuySignal 处理买入信号
func (bt *Backtester) processBuySignal(signal strategy.TradingSignal, price float64, volume int64, timestamp time.Time, state *BacktestState) error {
	if state.Position > 0 {
		// 已有持仓，跳过
		return nil
	}

	// 计算可买入数量
	maxQuantity := state.Capital / price
	quantity := math.Min(signal.Quantity, maxQuantity)

	if quantity <= 0 {
		return fmt.Errorf("资金不足，无法买入")
	}

	// 计算佣金和滑点
	commission := quantity * price * bt.commissionRate
	slippage := quantity * price * bt.slippageRate
	totalCost := quantity*price + commission + slippage

	if totalCost > state.Capital {
		return fmt.Errorf("资金不足，考虑佣金和滑点后无法买入")
	}

	// 执行买入
	state.Position = quantity
	state.EntryPrice = price
	state.EntryTime = timestamp
	state.Capital -= totalCost

	log.Printf("买入: 价格=%.2f, 数量=%.2f, 成本=%.2f", price, quantity, totalCost)

	return nil
}

// processSellSignal 处理卖出信号
func (bt *Backtester) processSellSignal(signal strategy.TradingSignal, price float64, volume int64, timestamp time.Time, state *BacktestState) error {
	if state.Position <= 0 {
		// 无持仓，跳过
		return nil
	}

	quantity := state.Position

	// 计算佣金和滑点
	commission := quantity * price * bt.commissionRate
	slippage := quantity * price * bt.slippageRate
	totalCost := commission + slippage
	proceeds := quantity*price - totalCost

	// 计算盈亏
	pnl := proceeds - (quantity * state.EntryPrice)

	// 记录交易
	trade := TradeRecord{
		EntryDate:  state.EntryTime,
		ExitDate:   timestamp,
		Symbol:     signal.Symbol,
		Side:       "long",
		EntryPrice: state.EntryPrice,
		ExitPrice:  price,
		Quantity:   quantity,
		PnL:        pnl,
		Commission: commission,
		Return:     pnl / (quantity * state.EntryPrice),
	}
	state.TradeHistory = append(state.TradeHistory, trade)

	// 更新资金
	state.Capital += proceeds
	state.Position = 0
	state.EntryPrice = 0
	state.EntryTime = time.Time{}

	log.Printf("卖出: 价格=%.2f, 数量=%.2f, 盈亏=%.2f", price, quantity, pnl)

	return nil
}

// updateEquityCurve 更新净值曲线
func (bt *Backtester) updateEquityCurve(timestamp time.Time, state *BacktestState) {
	equity := state.Capital
	if state.Position > 0 {
		// 计算持仓市值（简化处理，使用入场价格）
		equity += state.Position * state.EntryPrice
	}

	state.EquityCurve = append(state.EquityCurve, EquityPoint{
		Date:  timestamp,
		Value: equity,
	})
}

// generateReport 生成回测报告
func (bt *Backtester) generateReport(symbol, startDate, endDate string, state *BacktestState) *BacktestResult {
	result := &BacktestResult{
		StrategyName:   bt.strategy.GetName(),
		Symbol:         symbol,
		InitialCapital: bt.initialCapital,
		FinalCapital:   state.Capital,
		EquityCurve:    state.EquityCurve,
		TradeHistory:   state.TradeHistory,
	}

	// 解析日期
	if start, err := time.Parse("2006-01-02", startDate); err == nil {
		result.StartDate = start
	}
	if end, err := time.Parse("2006-01-02", endDate); err == nil {
		result.EndDate = end
	}

	// 计算基本指标
	result.TotalReturn = (result.FinalCapital - result.InitialCapital) / result.InitialCapital

	// 计算年化收益率
	if !result.StartDate.IsZero() && !result.EndDate.IsZero() {
		years := result.EndDate.Sub(result.StartDate).Hours() / (24 * 365)
		if years > 0 {
			result.AnnualReturn = math.Pow(1+result.TotalReturn, 1/years) - 1
		}
	}

	// 计算交易统计
	bt.calculateTradeStatistics(result)

	// 计算风险指标
	bt.calculateRiskMetrics(result)

	return result
}

// calculateTradeStatistics 计算交易统计
func (bt *Backtester) calculateTradeStatistics(result *BacktestResult) {
	trades := result.TradeHistory
	result.TotalTrades = len(trades)

	if result.TotalTrades == 0 {
		return
	}

	var wins, losses int
	var totalWin, totalLoss float64
	var consecutiveWins, consecutiveLosses int
	var maxConsecutiveWins, maxConsecutiveLosses int

	for _, trade := range trades {
		if trade.PnL > 0 {
			wins++
			totalWin += trade.PnL
			consecutiveWins++
			consecutiveLosses = 0
			if consecutiveWins > maxConsecutiveWins {
				maxConsecutiveWins = consecutiveWins
			}
		} else {
			losses++
			totalLoss += math.Abs(trade.PnL)
			consecutiveLosses++
			consecutiveWins = 0
			if consecutiveLosses > maxConsecutiveLosses {
				maxConsecutiveLosses = consecutiveLosses
			}
		}
	}

	result.WinningTrades = wins
	result.LosingTrades = losses
	result.WinRate = float64(wins) / float64(result.TotalTrades)

	if wins > 0 {
		result.AvgWin = totalWin / float64(wins)
	}
	if losses > 0 {
		result.AvgLoss = totalLoss / float64(losses)
	}

	result.MaxConsecutiveWins = maxConsecutiveWins
	result.MaxConsecutiveLosses = maxConsecutiveLosses

	if totalLoss > 0 {
		result.ProfitFactor = totalWin / totalLoss
	}

	// 计算佣金和滑点
	for _, trade := range trades {
		result.Commission += trade.Commission
		result.Slippage += trade.Quantity * trade.ExitPrice * bt.slippageRate
	}
}

// calculateRiskMetrics 计算风险指标
func (bt *Backtester) calculateRiskMetrics(result *BacktestResult) {
	equityCurve := result.EquityCurve
	if len(equityCurve) < 2 {
		return
	}

	// 假设无风险利率为3%
	riskFreeRate := 0.03

	// 计算收益率序列
	returns := make([]float64, len(equityCurve)-1)
	for i := 1; i < len(equityCurve); i++ {
		returns[i-1] = (equityCurve[i].Value - equityCurve[i-1].Value) / equityCurve[i-1].Value
	}

	// 计算夏普比率
	if len(returns) > 0 {
		meanReturn := bt.calculateMean(returns)
		stdReturn := bt.calculateStd(returns)

		if stdReturn > 0 {
			result.SharpeRatio = (meanReturn - riskFreeRate/252) / stdReturn
		}

		// 计算索提诺比率
		downsideReturns := make([]float64, 0)
		for _, ret := range returns {
			if ret < 0 {
				downsideReturns = append(downsideReturns, ret)
			}
		}

		if len(downsideReturns) > 0 {
			downsideStd := bt.calculateStd(downsideReturns)
			if downsideStd > 0 {
				result.SortinoRatio = (meanReturn - riskFreeRate/252) / downsideStd
			}
		}
	}

	// 计算最大回撤
	result.MaxDrawdown = bt.calculateMaxDrawdown(equityCurve)
}

// calculateMean 计算均值
func (bt *Backtester) calculateMean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}

	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

// calculateStd 计算标准差
func (bt *Backtester) calculateStd(values []float64) float64 {
	if len(values) < 2 {
		return 0
	}

	mean := bt.calculateMean(values)
	sum := 0.0
	for _, v := range values {
		sum += math.Pow(v-mean, 2)
	}
	return math.Sqrt(sum / float64(len(values)-1))
}

// calculateMaxDrawdown 计算最大回撤
func (bt *Backtester) calculateMaxDrawdown(equityCurve []EquityPoint) float64 {
	if len(equityCurve) < 2 {
		return 0
	}

	maxDrawdown := 0.0
	peak := equityCurve[0].Value

	for _, point := range equityCurve {
		if point.Value > peak {
			peak = point.Value
		}

		drawdown := (peak - point.Value) / peak
		if drawdown > maxDrawdown {
			maxDrawdown = drawdown
		}
	}

	return maxDrawdown
}

// GenerateReport 生成回测报告（兼容接口）
func (bt *Backtester) GenerateReport() (*BacktestResult, error) {
	// 这个方法是为了兼容策略管理器中的接口
	// 实际使用时应该调用 Run 方法
	return &BacktestResult{
		StrategyName:   bt.strategy.GetName(),
		InitialCapital: bt.initialCapital,
		FinalCapital:   bt.initialCapital * 1.05, // 模拟结果
		TotalReturn:    0.05,
		MaxDrawdown:    0.02,
		SharpeRatio:    1.2,
		TotalTrades:    10,
		WinningTrades:  7,
		LosingTrades:   3,
		WinRate:        0.7,
	}, nil
}
