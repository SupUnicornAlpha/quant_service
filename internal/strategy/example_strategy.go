package strategy

import (
	"fmt"
	"log"
	"time"

	"agent-quant-system/internal/data"
)

// MovingAverageCrossStrategy 移动平均线交叉策略
type MovingAverageCrossStrategy struct {
	BaseStrategy
}

// NewMovingAverageCrossStrategy 创建移动平均线交叉策略
func NewMovingAverageCrossStrategy() *MovingAverageCrossStrategy {
	strategy := &MovingAverageCrossStrategy{
		BaseStrategy: BaseStrategy{
			Name:        "移动平均线交叉策略",
			Description: "基于短期和长期移动平均线交叉的交易策略",
			Parameters: StrategyParams{
				"short_period":        5.0,       // 短期移动平均线周期
				"long_period":         20.0,      // 长期移动平均线周期
				"volume_threshold":    1000000.0, // 成交量阈值
				"risk_percentage":     2.0,       // 风险百分比
				"stop_loss_percent":   5.0,       // 止损百分比
				"take_profit_percent": 10.0,      // 止盈百分比
			},
		},
	}
	return strategy
}

// ValidateParameters 验证策略参数
func (ma *MovingAverageCrossStrategy) ValidateParameters(params StrategyParams) error {
	shortPeriod := params["short_period"].(float64)
	longPeriod := params["long_period"].(float64)

	if shortPeriod >= longPeriod {
		return fmt.Errorf("短期周期 (%v) 必须小于长期周期 (%v)", shortPeriod, longPeriod)
	}

	if shortPeriod <= 0 || longPeriod <= 0 {
		return fmt.Errorf("移动平均线周期必须大于0")
	}

	return nil
}

// Initialize 初始化策略
func (ma *MovingAverageCrossStrategy) Initialize() error {
	if err := ma.ValidateParameters(ma.Parameters); err != nil {
		return fmt.Errorf("策略参数验证失败: %w", err)
	}

	ma.IsActive = true
	log.Printf("移动平均线交叉策略已初始化: 短期周期=%.0f, 长期周期=%.0f",
		ma.GetFloat64Param("short_period", 5),
		ma.GetFloat64Param("long_period", 20))

	return nil
}

// GenerateSignals 生成交易信号
func (ma *MovingAverageCrossStrategy) GenerateSignals(df data.DataFrame, guidance *AgentGuidance) ([]TradingSignal, error) {
	log.Printf("开始生成移动平均线交叉策略信号")

	if !ma.IsActive {
		return nil, fmt.Errorf("策略未激活")
	}

	// 验证数据
	if err := ma.validateData(df); err != nil {
		return nil, fmt.Errorf("数据验证失败: %w", err)
	}

	// 计算移动平均线
	shortMA, err := ma.calculateMovingAverage(df, int(ma.GetFloat64Param("short_period", 5)))
	if err != nil {
		return nil, fmt.Errorf("计算短期移动平均线失败: %w", err)
	}

	longMA, err := ma.calculateMovingAverage(df, int(ma.GetFloat64Param("long_period", 20)))
	if err != nil {
		return nil, fmt.Errorf("计算长期移动平均线失败: %w", err)
	}

	// 生成信号
	signals := ma.generateCrossSignals(shortMA, longMA, df, guidance)

	log.Printf("生成了 %d 个交易信号", len(signals))
	return signals, nil
}

// validateData 验证数据完整性
func (ma *MovingAverageCrossStrategy) validateData(df data.DataFrame) error {
	requiredColumns := []string{"close", "volume"}
	for _, col := range requiredColumns {
		if _, exists := df[col]; !exists {
			return fmt.Errorf("缺少必需的列: %s", col)
		}
	}

	dataLength := len(df["close"])
	if dataLength < int(ma.GetFloat64Param("long_period", 20)) {
		return fmt.Errorf("数据长度不足，需要至少 %v 个数据点", ma.GetFloat64Param("long_period", 20))
	}

	return nil
}

// calculateMovingAverage 计算移动平均线
func (ma *MovingAverageCrossStrategy) calculateMovingAverage(df data.DataFrame, period int) ([]float64, error) {
	closeData := df["close"]
	if len(closeData) < period {
		return nil, fmt.Errorf("数据长度不足")
	}

	var movingAverages []float64

	for i := period - 1; i < len(closeData); i++ {
		sum := 0.0
		for j := i - period + 1; j <= i; j++ {
			sum += closeData[j].(float64)
		}
		movingAverages = append(movingAverages, sum/float64(period))
	}

	return movingAverages, nil
}

// generateCrossSignals 生成交叉信号
func (ma *MovingAverageCrossStrategy) generateCrossSignals(shortMA, longMA []float64, df data.DataFrame, guidance *AgentGuidance) []TradingSignal {
	var signals []TradingSignal

	if len(shortMA) < 2 || len(longMA) < 2 {
		return signals
	}

	// 获取最新价格
	closeData := df["close"]
	currentPrice := closeData[len(closeData)-1].(float64)

	// 获取最新成交量
	volumeData := df["volume"]
	currentVolume := volumeData[len(volumeData)-1].(int64)

	// 检查成交量阈值
	volumeThreshold := int64(ma.GetFloat64Param("volume_threshold", 1000000))
	if currentVolume < volumeThreshold {
		log.Printf("成交量不足，跳过信号生成: 当前=%d, 阈值=%d", currentVolume, volumeThreshold)
		return signals
	}

	// 获取当前和前一个MA值
	currentShortMA := shortMA[len(shortMA)-1]
	currentLongMA := longMA[len(longMA)-1]
	prevShortMA := shortMA[len(shortMA)-2]
	prevLongMA := longMA[len(longMA)-2]

	// 金叉信号（短期MA上穿长期MA）
	if prevShortMA <= prevLongMA && currentShortMA > currentLongMA {
		confidence := 0.7
		reason := fmt.Sprintf("金叉信号: 短期MA(%.2f)上穿长期MA(%.2f)", currentShortMA, currentLongMA)

		// 结合Agent指导调整置信度
		if guidance != nil {
			if guidance.Sentiment == "Positive" {
				confidence += 0.1
				reason += fmt.Sprintf(" + Agent看多(%.2f)", guidance.Confidence)
			} else if guidance.Sentiment == "Negative" {
				confidence -= 0.1
				reason += fmt.Sprintf(" - Agent看空(%.2f)", guidance.Confidence)
			}
		}

		// 计算仓位大小
		quantity := ma.calculatePositionSize(currentPrice, confidence)

		// 计算止损止盈
		stopLoss := CalculateStopLoss(currentPrice, ma.GetFloat64Param("stop_loss_percent", 5), Buy)
		takeProfit := CalculateTakeProfit(currentPrice, ma.GetFloat64Param("take_profit_percent", 10), Buy)

		signal := TradingSignal{
			Symbol:     "DEFAULT_SYMBOL", // 实际应用中应该从参数或数据中获取
			Signal:     Buy,
			Price:      currentPrice,
			Quantity:   quantity,
			Confidence: confidence,
			Reason:     reason,
			Timestamp:  time.Now(),
			StopLoss:   stopLoss,
			TakeProfit: takeProfit,
		}

		signals = append(signals, signal)
		log.Printf("生成买入信号: 价格=%.2f, 数量=%.2f, 置信度=%.2f", currentPrice, quantity, confidence)
	}

	// 死叉信号（短期MA下穿长期MA）
	if prevShortMA >= prevLongMA && currentShortMA < currentLongMA {
		confidence := 0.7
		reason := fmt.Sprintf("死叉信号: 短期MA(%.2f)下穿长期MA(%.2f)", currentShortMA, currentLongMA)

		// 结合Agent指导调整置信度
		if guidance != nil {
			if guidance.Sentiment == "Negative" {
				confidence += 0.1
				reason += fmt.Sprintf(" + Agent看空(%.2f)", guidance.Confidence)
			} else if guidance.Sentiment == "Positive" {
				confidence -= 0.1
				reason += fmt.Sprintf(" - Agent看多(%.2f)", guidance.Confidence)
			}
		}

		// 计算仓位大小
		quantity := ma.calculatePositionSize(currentPrice, confidence)

		// 计算止损止盈
		stopLoss := CalculateStopLoss(currentPrice, ma.GetFloat64Param("stop_loss_percent", 5), Sell)
		takeProfit := CalculateTakeProfit(currentPrice, ma.GetFloat64Param("take_profit_percent", 10), Sell)

		signal := TradingSignal{
			Symbol:     "DEFAULT_SYMBOL", // 实际应用中应该从参数或数据中获取
			Signal:     Sell,
			Price:      currentPrice,
			Quantity:   quantity,
			Confidence: confidence,
			Reason:     reason,
			Timestamp:  time.Now(),
			StopLoss:   stopLoss,
			TakeProfit: takeProfit,
		}

		signals = append(signals, signal)
		log.Printf("生成卖出信号: 价格=%.2f, 数量=%.2f, 置信度=%.2f", currentPrice, quantity, confidence)
	}

	return signals
}

// calculatePositionSize 计算仓位大小
func (ma *MovingAverageCrossStrategy) calculatePositionSize(price, confidence float64) float64 {
	// 简化计算，实际应用中应该考虑账户余额和风险管理
	baseQuantity := 100.0
	adjustedQuantity := baseQuantity * confidence

	// 确保最小仓位
	if adjustedQuantity < 10.0 {
		adjustedQuantity = 10.0
	}

	return adjustedQuantity
}

// RSIStrategy RSI策略
type RSIStrategy struct {
	BaseStrategy
}

// NewRSIStrategy 创建RSI策略
func NewRSIStrategy() *RSIStrategy {
	strategy := &RSIStrategy{
		BaseStrategy: BaseStrategy{
			Name:        "RSI策略",
			Description: "基于相对强弱指数的交易策略",
			Parameters: StrategyParams{
				"rsi_period":       14.0, // RSI周期
				"oversold_level":   30.0, // 超卖水平
				"overbought_level": 70.0, // 超买水平
				"risk_percentage":  2.0,  // 风险百分比
			},
		},
	}
	return strategy
}

// GenerateSignals 生成RSI交易信号
func (rsi *RSIStrategy) GenerateSignals(df data.DataFrame, guidance *AgentGuidance) ([]TradingSignal, error) {
	log.Printf("开始生成RSI策略信号")

	if !rsi.IsActive {
		return nil, fmt.Errorf("策略未激活")
	}

	// 计算RSI
	rsiValues, err := rsi.calculateRSI(df, int(rsi.GetFloat64Param("rsi_period", 14)))
	if err != nil {
		return nil, fmt.Errorf("计算RSI失败: %w", err)
	}

	if len(rsiValues) == 0 {
		return []TradingSignal{}, nil
	}

	currentRSI := rsiValues[len(rsiValues)-1]
	oversoldLevel := rsi.GetFloat64Param("oversold_level", 30)
	overboughtLevel := rsi.GetFloat64Param("overbought_level", 70)

	// 获取最新价格
	closeData := df["close"]
	currentPrice := closeData[len(closeData)-1].(float64)

	var signals []TradingSignal

	// RSI超卖信号
	if currentRSI < oversoldLevel {
		confidence := (oversoldLevel - currentRSI) / oversoldLevel
		reason := fmt.Sprintf("RSI超卖信号: RSI=%.2f < %.2f", currentRSI, oversoldLevel)

		signal := CreateTradingSignal("DEFAULT_SYMBOL", Buy, currentPrice, 100.0, confidence, reason)
		signals = append(signals, signal)
		log.Printf("生成RSI买入信号: RSI=%.2f", currentRSI)
	}

	// RSI超买信号
	if currentRSI > overboughtLevel {
		confidence := (currentRSI - overboughtLevel) / (100 - overboughtLevel)
		reason := fmt.Sprintf("RSI超买信号: RSI=%.2f > %.2f", currentRSI, overboughtLevel)

		signal := CreateTradingSignal("DEFAULT_SYMBOL", Sell, currentPrice, 100.0, confidence, reason)
		signals = append(signals, signal)
		log.Printf("生成RSI卖出信号: RSI=%.2f", currentRSI)
	}

	return signals, nil
}

// calculateRSI 计算RSI指标
func (rsi *RSIStrategy) calculateRSI(df data.DataFrame, period int) ([]float64, error) {
	closeData := df["close"]
	if len(closeData) < period+1 {
		return nil, fmt.Errorf("数据长度不足")
	}

	var rsiValues []float64
	var gains, losses []float64

	// 计算价格变化
	for i := 1; i < len(closeData); i++ {
		change := closeData[i].(float64) - closeData[i-1].(float64)
		if change > 0 {
			gains = append(gains, change)
			losses = append(losses, 0)
		} else {
			gains = append(gains, 0)
			losses = append(losses, -change)
		}
	}

	// 计算RSI
	for i := period - 1; i < len(gains); i++ {
		avgGain := 0.0
		avgLoss := 0.0

		for j := i - period + 1; j <= i; j++ {
			avgGain += gains[j]
			avgLoss += losses[j]
		}

		avgGain /= float64(period)
		avgLoss /= float64(period)

		if avgLoss == 0 {
			rsiValues = append(rsiValues, 100)
		} else {
			rs := avgGain / avgLoss
			rsiValue := 100 - (100 / (1 + rs))
			rsiValues = append(rsiValues, rsiValue)
		}
	}

	return rsiValues, nil
}

// Initialize 初始化RSI策略
func (rsi *RSIStrategy) Initialize() error {
	rsi.IsActive = true
	log.Printf("RSI策略已初始化: 周期=%.0f, 超卖=%.0f, 超买=%.0f",
		rsi.GetFloat64Param("rsi_period", 14),
		rsi.GetFloat64Param("oversold_level", 30),
		rsi.GetFloat64Param("overbought_level", 70))
	return nil
}
