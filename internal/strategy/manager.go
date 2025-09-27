package strategy

import (
	"fmt"
	"log"
	"sync"

	"agent-quant-system/internal/data"
)

// StrategyManager 策略管理器
type StrategyManager struct {
	strategies map[string]Strategy
	mutex      sync.RWMutex
}

// NewStrategyManager 创建策略管理器
func NewStrategyManager() *StrategyManager {
	manager := &StrategyManager{
		strategies: make(map[string]Strategy),
	}

	// 注册默认策略
	manager.registerDefaultStrategies()

	return manager
}

// registerDefaultStrategies 注册默认策略
func (sm *StrategyManager) registerDefaultStrategies() {
	// 注册移动平均线交叉策略
	maStrategy := NewMovingAverageCrossStrategy()
	if err := maStrategy.Initialize(); err != nil {
		log.Printf("移动平均线交叉策略初始化失败: %v", err)
	} else {
		sm.strategies["ma_cross"] = maStrategy
		log.Printf("已注册策略: %s", maStrategy.GetName())
	}

	// 注册RSI策略
	rsiStrategy := NewRSIStrategy()
	if err := rsiStrategy.Initialize(); err != nil {
		log.Printf("RSI策略初始化失败: %v", err)
	} else {
		sm.strategies["rsi"] = rsiStrategy
		log.Printf("已注册策略: %s", rsiStrategy.GetName())
	}
}

// RegisterStrategy 注册策略
func (sm *StrategyManager) RegisterStrategy(name string, strategy Strategy) error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	if name == "" {
		return fmt.Errorf("策略名称不能为空")
	}

	if strategy == nil {
		return fmt.Errorf("策略不能为nil")
	}

	// 初始化策略
	if err := strategy.Initialize(); err != nil {
		return fmt.Errorf("策略初始化失败: %w", err)
	}

	sm.strategies[name] = strategy
	log.Printf("成功注册策略: %s (%s)", name, strategy.GetName())

	return nil
}

// GetStrategy 获取策略
func (sm *StrategyManager) GetStrategy(name string) (Strategy, error) {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	strategy, exists := sm.strategies[name]
	if !exists {
		return nil, fmt.Errorf("策略 '%s' 不存在", name)
	}

	return strategy, nil
}

// GetAvailableStrategies 获取所有可用策略
func (sm *StrategyManager) GetAvailableStrategies() map[string]StrategyInfo {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	strategies := make(map[string]StrategyInfo)
	for name, strategy := range sm.strategies {
		strategies[name] = StrategyInfo{
			Name:        strategy.GetName(),
			Description: strategy.GetDescription(),
			Parameters:  strategy.GetParameters(),
		}
	}

	return strategies
}

// StrategyInfo 策略信息
type StrategyInfo struct {
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Parameters  StrategyParams `json:"parameters"`
}

// UnregisterStrategy 注销策略
func (sm *StrategyManager) UnregisterStrategy(name string) error {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	strategy, exists := sm.strategies[name]
	if !exists {
		return fmt.Errorf("策略 '%s' 不存在", name)
	}

	// 清理策略资源
	if err := strategy.Cleanup(); err != nil {
		log.Printf("策略清理失败: %v", err)
	}

	delete(sm.strategies, name)
	log.Printf("已注销策略: %s", name)

	return nil
}

// UpdateStrategyParameters 更新策略参数
func (sm *StrategyManager) UpdateStrategyParameters(name string, params StrategyParams) error {
	strategy, err := sm.GetStrategy(name)
	if err != nil {
		return err
	}

	// 验证参数
	if err := strategy.ValidateParameters(params); err != nil {
		return fmt.Errorf("参数验证失败: %w", err)
	}

	// 设置新参数
	if err := strategy.SetParameters(params); err != nil {
		return fmt.Errorf("设置参数失败: %w", err)
	}

	log.Printf("成功更新策略 '%s' 的参数", name)
	return nil
}

// ListStrategies 列出所有策略
func (sm *StrategyManager) ListStrategies() []string {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	var names []string
	for name := range sm.strategies {
		names = append(names, name)
	}

	return names
}

// GetStrategyCount 获取策略数量
func (sm *StrategyManager) GetStrategyCount() int {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	return len(sm.strategies)
}

// ValidateAllStrategies 验证所有策略
func (sm *StrategyManager) ValidateAllStrategies() map[string]error {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	errors := make(map[string]error)
	for name, strategy := range sm.strategies {
		if err := strategy.ValidateParameters(strategy.GetParameters()); err != nil {
			errors[name] = err
		}
	}

	return errors
}

// CleanupAllStrategies 清理所有策略
func (sm *StrategyManager) CleanupAllStrategies() {
	sm.mutex.Lock()
	defer sm.mutex.Unlock()

	for name, strategy := range sm.strategies {
		if err := strategy.Cleanup(); err != nil {
			log.Printf("策略 '%s' 清理失败: %v", name, err)
		}
	}

	log.Printf("已清理所有策略")
}

// ExecuteStrategy 执行策略
func (sm *StrategyManager) ExecuteStrategy(name string, data data.DataFrame, guidance *AgentGuidance) ([]TradingSignal, error) {
	strategy, err := sm.GetStrategy(name)
	if err != nil {
		return nil, err
	}

	log.Printf("开始执行策略: %s", name)
	signals, err := strategy.GenerateSignals(data, guidance)
	if err != nil {
		return nil, fmt.Errorf("策略执行失败: %w", err)
	}

	log.Printf("策略 '%s' 执行完成，生成 %d 个信号", name, len(signals))
	return signals, nil
}

// GetStrategyStatus 获取策略状态
func (sm *StrategyManager) GetStrategyStatus(name string) (*StrategyStatus, error) {
	strategy, err := sm.GetStrategy(name)
	if err != nil {
		return nil, err
	}

	// 通过反射或类型断言获取BaseStrategy字段
	// 这里我们简化处理，直接使用接口方法
	status := &StrategyStatus{
		Name:        strategy.GetName(),
		IsActive:    true, // 简化处理，假设策略都是激活的
		Parameters:  strategy.GetParameters(),
		Description: strategy.GetDescription(),
	}

	return status, nil
}

// StrategyStatus 策略状态
type StrategyStatus struct {
	Name        string         `json:"name"`
	IsActive    bool           `json:"is_active"`
	Parameters  StrategyParams `json:"parameters"`
	Description string         `json:"description"`
}

// GetAllStrategyStatuses 获取所有策略状态
func (sm *StrategyManager) GetAllStrategyStatuses() map[string]*StrategyStatus {
	sm.mutex.RLock()
	defer sm.mutex.RUnlock()

	statuses := make(map[string]*StrategyStatus)
	for name, strategy := range sm.strategies {
		statuses[name] = &StrategyStatus{
			Name:        strategy.GetName(),
			IsActive:    true, // 简化处理，假设策略都是激活的
			Parameters:  strategy.GetParameters(),
			Description: strategy.GetDescription(),
		}
	}

	return statuses
}

// RunStrategyBacktest 运行策略回测
func (sm *StrategyManager) RunStrategyBacktest(name string, data data.DataFrame, initialCapital float64) (*BacktestResult, error) {
	strategy, err := sm.GetStrategy(name)
	if err != nil {
		return nil, err
	}

	log.Printf("开始回测策略: %s", name)

	// 模拟回测逻辑
	result := &BacktestResult{
		StrategyName:   strategy.GetName(),
		InitialCapital: initialCapital,
		FinalCapital:   initialCapital * 1.05, // 模拟5%收益
		TotalReturn:    0.05,
		MaxDrawdown:    0.02,
		SharpeRatio:    1.2,
		TotalTrades:    10,
		WinningTrades:  7,
		LosingTrades:   3,
		WinRate:        0.7,
	}

	log.Printf("策略回测完成: 总收益=%.2f%%, 最大回撤=%.2f%%", result.TotalReturn*100, result.MaxDrawdown*100)

	return result, nil
}

// BacktestResult 回测结果
type BacktestResult struct {
	StrategyName   string  `json:"strategy_name"`
	InitialCapital float64 `json:"initial_capital"`
	FinalCapital   float64 `json:"final_capital"`
	TotalReturn    float64 `json:"total_return"`
	MaxDrawdown    float64 `json:"max_drawdown"`
	SharpeRatio    float64 `json:"sharpe_ratio"`
	TotalTrades    int     `json:"total_trades"`
	WinningTrades  int     `json:"winning_trades"`
	LosingTrades   int     `json:"losing_trades"`
	WinRate        float64 `json:"win_rate"`
}
