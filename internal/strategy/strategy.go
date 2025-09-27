package strategy

import (
	"time"

	"agent-quant-system/internal/data"
)

// Signal 交易信号类型
type Signal int

const (
	Hold Signal = iota // 持有
	Buy                // 买入
	Sell               // 卖出
)

// String 返回信号的字符串表示
func (s Signal) String() string {
	switch s {
	case Buy:
		return "买入"
	case Sell:
		return "卖出"
	case Hold:
		return "持有"
	default:
		return "未知"
	}
}

// AgentGuidance Agent指导信息
type AgentGuidance struct {
	Sentiment  string    `json:"sentiment"`  // 情绪分析结果
	Reason     string    `json:"reason"`     // 分析原因
	Confidence float64   `json:"confidence"` // 置信度
	Timestamp  time.Time `json:"timestamp"`  // 时间戳
	Symbol     string    `json:"symbol"`     // 标的符号
}

// TradingSignal 交易信号
type TradingSignal struct {
	Symbol     string    `json:"symbol"`      // 标的符号
	Signal     Signal    `json:"signal"`      // 信号类型
	Price      float64   `json:"price"`       // 建议价格
	Quantity   float64   `json:"quantity"`    // 建议数量
	Confidence float64   `json:"confidence"`  // 信号置信度
	Reason     string    `json:"reason"`      // 信号原因
	Timestamp  time.Time `json:"timestamp"`   // 时间戳
	StopLoss   float64   `json:"stop_loss"`   // 止损价格
	TakeProfit float64   `json:"take_profit"` // 止盈价格
}

// StrategyParams 策略参数
type StrategyParams map[string]interface{}

// Strategy 策略接口
type Strategy interface {
	// GetName 获取策略名称
	GetName() string

	// GetDescription 获取策略描述
	GetDescription() string

	// GetParameters 获取策略参数
	GetParameters() StrategyParams

	// SetParameters 设置策略参数
	SetParameters(params StrategyParams) error

	// ValidateParameters 验证参数
	ValidateParameters(params StrategyParams) error

	// GenerateSignals 生成交易信号
	GenerateSignals(data data.DataFrame, guidance *AgentGuidance) ([]TradingSignal, error)

	// Initialize 初始化策略
	Initialize() error

	// Cleanup 清理资源
	Cleanup() error
}

// BaseStrategy 基础策略结构体，提供通用功能
type BaseStrategy struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  StrategyParams `json:"parameters"`
	IsActive    bool           `json:"is_active"`
}

// GetName 获取策略名称
func (bs *BaseStrategy) GetName() string {
	return bs.Name
}

// GetDescription 获取策略描述
func (bs *BaseStrategy) GetDescription() string {
	return bs.Description
}

// GetParameters 获取策略参数
func (bs *BaseStrategy) GetParameters() StrategyParams {
	return bs.Parameters
}

// SetParameters 设置策略参数
func (bs *BaseStrategy) SetParameters(params StrategyParams) error {
	bs.Parameters = params
	return nil
}

// ValidateParameters 验证参数（子类可重写）
func (bs *BaseStrategy) ValidateParameters(params StrategyParams) error {
	return nil
}

// Initialize 初始化策略（子类可重写）
func (bs *BaseStrategy) Initialize() error {
	bs.IsActive = true
	return nil
}

// Cleanup 清理资源（子类可重写）
func (bs *BaseStrategy) Cleanup() error {
	bs.IsActive = false
	return nil
}

// GetFloat64Param 获取float64类型参数
func (bs *BaseStrategy) GetFloat64Param(key string, defaultValue float64) float64 {
	if val, exists := bs.Parameters[key]; exists {
		if floatVal, ok := val.(float64); ok {
			return floatVal
		}
	}
	return defaultValue
}

// GetIntParam 获取int类型参数
func (bs *BaseStrategy) GetIntParam(key string, defaultValue int) int {
	if val, exists := bs.Parameters[key]; exists {
		if intVal, ok := val.(int); ok {
			return intVal
		}
	}
	return defaultValue
}

// GetStringParam 获取string类型参数
func (bs *BaseStrategy) GetStringParam(key string, defaultValue string) string {
	if val, exists := bs.Parameters[key]; exists {
		if strVal, ok := val.(string); ok {
			return strVal
		}
	}
	return defaultValue
}

// GetBoolParam 获取bool类型参数
func (bs *BaseStrategy) GetBoolParam(key string, defaultValue bool) bool {
	if val, exists := bs.Parameters[key]; exists {
		if boolVal, ok := val.(bool); ok {
			return boolVal
		}
	}
	return defaultValue
}

// CreateTradingSignal 创建交易信号
func CreateTradingSignal(symbol string, signal Signal, price, quantity, confidence float64, reason string) TradingSignal {
	return TradingSignal{
		Symbol:     symbol,
		Signal:     signal,
		Price:      price,
		Quantity:   quantity,
		Confidence: confidence,
		Reason:     reason,
		Timestamp:  time.Now(),
	}
}

// CalculatePositionSize 计算仓位大小
func CalculatePositionSize(accountBalance, riskPercentage, stopLossDistance, price float64) float64 {
	if stopLossDistance <= 0 || price <= 0 {
		return 0
	}

	riskAmount := accountBalance * riskPercentage / 100.0
	positionSize := riskAmount / stopLossDistance

	// 确保不超过账户余额
	maxPosition := accountBalance / price
	if positionSize > maxPosition {
		positionSize = maxPosition
	}

	return positionSize
}

// CalculateStopLoss 计算止损价格
func CalculateStopLoss(entryPrice float64, stopLossPercent float64, signal Signal) float64 {
	if signal == Buy {
		return entryPrice * (1 - stopLossPercent/100.0)
	} else if signal == Sell {
		return entryPrice * (1 + stopLossPercent/100.0)
	}
	return entryPrice
}

// CalculateTakeProfit 计算止盈价格
func CalculateTakeProfit(entryPrice float64, takeProfitPercent float64, signal Signal) float64 {
	if signal == Buy {
		return entryPrice * (1 + takeProfitPercent/100.0)
	} else if signal == Sell {
		return entryPrice * (1 - takeProfitPercent/100.0)
	}
	return entryPrice
}
