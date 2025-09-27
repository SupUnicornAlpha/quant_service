package trading

import (
	"fmt"
	"log"
	"sync"
	"time"

	"agent-quant-system/internal/account"
	"agent-quant-system/internal/config"
	"agent-quant-system/internal/strategy"
)

// TradingEngine 交易引擎
type TradingEngine struct {
	config         *config.Config
	accountManager *account.AccountManager
	brokers        map[string]BrokerAPI
	mutex          sync.RWMutex
	isRunning      bool
}

// NewTradingEngine 创建交易引擎
func NewTradingEngine(cfg *config.Config, accountManager *account.AccountManager) *TradingEngine {
	engine := &TradingEngine{
		config:         cfg,
		accountManager: accountManager,
		brokers:        make(map[string]BrokerAPI),
		isRunning:      false,
	}

	// 初始化经纪商连接
	engine.initializeBrokers()

	return engine
}

// initializeBrokers 初始化经纪商连接
func (te *TradingEngine) initializeBrokers() {
	log.Printf("初始化经纪商连接")

	for accountName, accountConfig := range te.config.Accounts {
		var broker BrokerAPI

		switch accountConfig.BrokerType {
		case "stock":
			broker = NewMockStockBroker(accountName)
		case "crypto":
			broker = NewMockCryptoBroker(accountName)
		default:
			log.Printf("未知的经纪商类型: %s", accountConfig.BrokerType)
			continue
		}

		// 连接经纪商
		if err := broker.Connect(); err != nil {
			log.Printf("连接经纪商 %s 失败: %v", accountName, err)
			continue
		}

		te.brokers[accountName] = broker
		log.Printf("已连接经纪商: %s (%s)", accountName, accountConfig.BrokerType)
	}
}

// GetBroker 获取经纪商实例
func (te *TradingEngine) GetBroker(accountName string) (BrokerAPI, error) {
	te.mutex.RLock()
	defer te.mutex.RUnlock()

	broker, exists := te.brokers[accountName]
	if !exists {
		return nil, fmt.Errorf("经纪商 '%s' 不存在或未连接", accountName)
	}

	return broker, nil
}

// ExecuteTrade 执行交易
func (te *TradingEngine) ExecuteTrade(order Order, accountName string) (*Order, error) {
	log.Printf("开始执行交易: 账户=%s, 标的=%s, 方向=%s, 数量=%.2f, 价格=%.2f",
		accountName, order.Symbol, order.Side, order.Quantity, order.Price)

	// 获取经纪商
	broker, err := te.GetBroker(accountName)
	if err != nil {
		return nil, fmt.Errorf("获取经纪商失败: %w", err)
	}

	// 验证账户
	if err := te.validateAccount(accountName); err != nil {
		return nil, fmt.Errorf("账户验证失败: %w", err)
	}

	// 设置订单信息
	order.AccountName = accountName
	order.CreateTime = time.Now()
	order.UpdateTime = time.Now()

	// 执行订单
	resultOrder, err := broker.PlaceOrder(order)
	if err != nil {
		return nil, fmt.Errorf("下单失败: %w", err)
	}

	// 更新账户信息
	if err := te.updateAccountAfterTrade(resultOrder, accountName); err != nil {
		log.Printf("更新账户信息失败: %v", err)
	}

	log.Printf("交易执行完成: 订单ID=%s, 状态=%s", resultOrder.ID, resultOrder.Status)
	return resultOrder, nil
}

// ExecuteSignal 执行交易信号
func (te *TradingEngine) ExecuteSignal(signal strategy.TradingSignal, accountName string) (*Order, error) {
	log.Printf("开始执行交易信号: 账户=%s, 标的=%s, 信号=%s, 数量=%.2f",
		accountName, signal.Symbol, signal.Signal.String(), signal.Quantity)

	// 转换信号为订单
	order := te.convertSignalToOrder(signal)

	// 执行交易
	return te.ExecuteTrade(order, accountName)
}

// convertSignalToOrder 将交易信号转换为订单
func (te *TradingEngine) convertSignalToOrder(signal strategy.TradingSignal) Order {
	var side OrderSide
	switch signal.Signal {
	case strategy.Buy:
		side = BuySide
	case strategy.Sell:
		side = SellSide
	default:
		side = BuySide // 默认买入
	}

	order := Order{
		Symbol:     signal.Symbol,
		Side:       side,
		Type:       MarketOrder, // 默认市价单
		Quantity:   signal.Quantity,
		Price:      signal.Price,
		Status:     Pending,
		CreateTime: time.Now(),
		UpdateTime: time.Now(),
	}

	// 设置止损和止盈价格
	if signal.StopLoss > 0 {
		order.StopPrice = signal.StopLoss
	}

	return order
}

// validateAccount 验证账户
func (te *TradingEngine) validateAccount(accountName string) error {
	// 检查账户是否存在
	_, err := te.accountManager.GetAccount(accountName)
	if err != nil {
		return fmt.Errorf("账户不存在: %w", err)
	}

	// 检查账户是否激活
	status, err := te.accountManager.GetAccountStatus(accountName)
	if err != nil {
		return fmt.Errorf("获取账户状态失败: %w", err)
	}

	if !status.IsActive {
		return fmt.Errorf("账户未激活")
	}

	// 验证账户凭证
	if err := te.accountManager.ValidateAccountCredentials(accountName); err != nil {
		return fmt.Errorf("账户凭证验证失败: %w", err)
	}

	return nil
}

// updateAccountAfterTrade 交易后更新账户信息
func (te *TradingEngine) updateAccountAfterTrade(order *Order, accountName string) error {
	// 获取经纪商
	broker, err := te.GetBroker(accountName)
	if err != nil {
		return err
	}

	// 更新余额
	balance, err := broker.GetBalance()
	if err != nil {
		log.Printf("获取余额失败: %v", err)
	} else {
		if err := te.accountManager.UpdateAccountBalance(accountName, balance); err != nil {
			log.Printf("更新账户余额失败: %v", err)
		}
	}

	// 更新持仓
	positions, err := broker.GetPositions()
	if err != nil {
		log.Printf("获取持仓失败: %v", err)
		return err
	}

	for symbol, position := range positions {
		if position.Quantity > 0 {
			// 更新或添加持仓
			_, err := te.accountManager.GetPosition(accountName, symbol)
			if err != nil {
				// 添加新持仓
				err = te.accountManager.AddPosition(accountName, symbol, position.Quantity, position.AvgPrice)
			} else {
				// 更新现有持仓
				err = te.accountManager.UpdatePosition(accountName, symbol, position.Quantity, position.AvgPrice)
			}

			if err != nil {
				log.Printf("更新持仓失败: %v", err)
			}
		} else {
			// 移除持仓
			te.accountManager.RemovePosition(accountName, symbol)
		}
	}

	return nil
}

// GetAccountBalance 获取账户余额
func (te *TradingEngine) GetAccountBalance(accountName string) (float64, error) {
	broker, err := te.GetBroker(accountName)
	if err != nil {
		return 0, err
	}

	return broker.GetBalance()
}

// GetAccountPositions 获取账户持仓
func (te *TradingEngine) GetAccountPositions(accountName string) (map[string]Position, error) {
	broker, err := te.GetBroker(accountName)
	if err != nil {
		return nil, err
	}

	return broker.GetPositions()
}

// GetAccountOrders 获取账户订单
func (te *TradingEngine) GetAccountOrders(accountName string, symbol string, status OrderStatus) ([]Order, error) {
	broker, err := te.GetBroker(accountName)
	if err != nil {
		return nil, err
	}

	return broker.GetOrders(symbol, status)
}

// GetAccountTrades 获取账户成交记录
func (te *TradingEngine) GetAccountTrades(accountName string, symbol string, limit int) ([]Trade, error) {
	broker, err := te.GetBroker(accountName)
	if err != nil {
		return nil, err
	}

	return broker.GetTrades(symbol, limit)
}

// CancelOrder 取消订单
func (te *TradingEngine) CancelOrder(accountName, orderID string) error {
	log.Printf("取消订单: 账户=%s, 订单ID=%s", accountName, orderID)

	broker, err := te.GetBroker(accountName)
	if err != nil {
		return err
	}

	return broker.CancelOrder(orderID)
}

// GetTradingStatus 获取交易状态
func (te *TradingEngine) GetTradingStatus() *TradingStatus {
	te.mutex.RLock()
	defer te.mutex.RUnlock()

	status := &TradingStatus{
		IsRunning: te.isRunning,
		Brokers:   make(map[string]BrokerStatus),
	}

	for name := range te.brokers {
		// 这里可以添加更多状态信息
		status.Brokers[name] = BrokerStatus{
			Name:   name,
			Status: "connected", // 简化状态
		}
	}

	return status
}

// Start 启动交易引擎
func (te *TradingEngine) Start() error {
	te.mutex.Lock()
	defer te.mutex.Unlock()

	if te.isRunning {
		return fmt.Errorf("交易引擎已在运行")
	}

	log.Printf("启动交易引擎")
	te.isRunning = true

	return nil
}

// Stop 停止交易引擎
func (te *TradingEngine) Stop() error {
	te.mutex.Lock()
	defer te.mutex.Unlock()

	if !te.isRunning {
		return fmt.Errorf("交易引擎未运行")
	}

	log.Printf("停止交易引擎")
	te.isRunning = false

	// 断开所有经纪商连接
	for name, broker := range te.brokers {
		if err := broker.Disconnect(); err != nil {
			log.Printf("断开经纪商 %s 连接失败: %v", name, err)
		}
	}

	return nil
}

// IsRunning 检查是否运行中
func (te *TradingEngine) IsRunning() bool {
	te.mutex.RLock()
	defer te.mutex.RUnlock()
	return te.isRunning
}

// TradingStatus 交易状态
type TradingStatus struct {
	IsRunning bool                    `json:"is_running"`
	Brokers   map[string]BrokerStatus `json:"brokers"`
}

// BrokerStatus 经纪商状态
type BrokerStatus struct {
	Name   string `json:"name"`
	Status string `json:"status"`
}

// RiskManager 风险管理器
type RiskManager struct {
	maxPositionSize float64 // 最大单笔仓位
	maxDailyLoss    float64 // 最大日亏损
	maxDrawdown     float64 // 最大回撤
}

// NewRiskManager 创建风险管理器
func NewRiskManager(maxPositionSize, maxDailyLoss, maxDrawdown float64) *RiskManager {
	return &RiskManager{
		maxPositionSize: maxPositionSize,
		maxDailyLoss:    maxDailyLoss,
		maxDrawdown:     maxDrawdown,
	}
}

// ValidateTrade 验证交易风险
func (rm *RiskManager) ValidateTrade(order Order, accountBalance float64, currentPositions map[string]Position) error {
	// 检查单笔仓位大小
	positionValue := order.Quantity * order.Price
	if positionValue > accountBalance*rm.maxPositionSize {
		return fmt.Errorf("单笔仓位过大: %.2f > %.2f", positionValue, accountBalance*rm.maxPositionSize)
	}

	// 检查总仓位
	totalPositionValue := positionValue
	for _, position := range currentPositions {
		totalPositionValue += position.MarketValue
	}

	if totalPositionValue > accountBalance {
		return fmt.Errorf("总仓位超过账户余额")
	}

	log.Printf("交易风险验证通过: 单笔仓位=%.2f, 总仓位=%.2f", positionValue, totalPositionValue)
	return nil
}

// CalculatePositionSize 计算仓位大小
func (rm *RiskManager) CalculatePositionSize(accountBalance, riskAmount, stopLossDistance float64) float64 {
	if stopLossDistance <= 0 {
		return 0
	}

	positionSize := riskAmount / stopLossDistance

	// 限制最大仓位
	maxPosition := accountBalance * rm.maxPositionSize / riskAmount
	if positionSize > maxPosition {
		positionSize = maxPosition
	}

	return positionSize
}
