package core

import (
	"fmt"
	"log"
	"sync"
	"time"

	"agent-quant-system/internal/account"
	"agent-quant-system/internal/agent"
	"agent-quant-system/internal/backtest"
	"agent-quant-system/internal/config"
	"agent-quant-system/internal/data"
	"agent-quant-system/internal/strategy"
	"agent-quant-system/internal/trading"
)

// QuantEngine 量化引擎
type QuantEngine struct {
	config          *config.Config
	dataManager     *data.DataManager
	strategyManager *strategy.StrategyManager
	agentClient     agent.ClientInterface
	tradingEngine   *trading.TradingEngine
	accountManager  *account.AccountManager

	isRunning bool
	mutex     sync.RWMutex
	stopChan  chan struct{}

	// 统计信息
	stats *EngineStats
}

// EngineStats 引擎统计信息
type EngineStats struct {
	StartTime        time.Time `json:"start_time"`
	LastUpdateTime   time.Time `json:"last_update_time"`
	TotalCycles      int       `json:"total_cycles"`
	SuccessfulCycles int       `json:"successful_cycles"`
	FailedCycles     int       `json:"failed_cycles"`
	TotalSignals     int       `json:"total_signals"`
	ExecutedTrades   int       `json:"executed_trades"`
	TotalPnL         float64   `json:"total_pnl"`
}

// NewQuantEngine 创建量化引擎
func NewQuantEngine(cfg *config.Config) (*QuantEngine, error) {
	log.Printf("初始化量化引擎")

	// 创建数据管理器
	dataManager := data.NewDataManager()

	// 创建策略管理器
	strategyManager := strategy.NewStrategyManager()

	// 创建账户管理器
	accountManager := account.NewAccountManager(cfg)

	// 创建交易引擎
	tradingEngine := trading.NewTradingEngine(cfg, accountManager)

	// 创建Agent客户端
	agentClient := agent.CreateClient(cfg.AgentService.URL, false) // 使用真实客户端

	engine := &QuantEngine{
		config:          cfg,
		dataManager:     dataManager,
		strategyManager: strategyManager,
		agentClient:     agentClient,
		tradingEngine:   tradingEngine,
		accountManager:  accountManager,
		isRunning:       false,
		stopChan:        make(chan struct{}),
		stats: &EngineStats{
			StartTime: time.Now(),
		},
	}

	// 验证Agent服务连接
	if err := engine.agentClient.HealthCheck(); err != nil {
		log.Printf("Agent服务连接失败，将使用模拟客户端: %v", err)
		engine.agentClient = agent.CreateClient(cfg.AgentService.URL, true)
	}

	log.Printf("量化引擎初始化完成")
	return engine, nil
}

// Start 启动量化引擎
func (qe *QuantEngine) Start() error {
	qe.mutex.Lock()
	defer qe.mutex.Unlock()

	if qe.isRunning {
		return fmt.Errorf("量化引擎已在运行")
	}

	log.Printf("启动量化引擎")

	// 启动交易引擎
	if err := qe.tradingEngine.Start(); err != nil {
		return fmt.Errorf("启动交易引擎失败: %w", err)
	}

	qe.isRunning = true
	qe.stats.StartTime = time.Now()

	log.Printf("量化引擎启动成功")
	return nil
}

// Stop 停止量化引擎
func (qe *QuantEngine) Stop() error {
	qe.mutex.Lock()
	defer qe.mutex.Unlock()

	if !qe.isRunning {
		return fmt.Errorf("量化引擎未运行")
	}

	log.Printf("停止量化引擎")

	// 发送停止信号
	close(qe.stopChan)

	// 停止交易引擎
	if err := qe.tradingEngine.Stop(); err != nil {
		log.Printf("停止交易引擎失败: %v", err)
	}

	qe.isRunning = false

	log.Printf("量化引擎已停止")
	return nil
}

// RunSingleLoop 运行单次循环
func (qe *QuantEngine) RunSingleLoop() error {
	log.Printf("开始执行单次交易循环")

	qe.stats.TotalCycles++
	qe.stats.LastUpdateTime = time.Now()

	defer func() {
		if r := recover(); r != nil {
			qe.stats.FailedCycles++
			log.Printf("交易循环发生panic: %v", r)
		}
	}()

	// 1. 模拟获取新闻数据
	newsItems := qe.getMockNews()
	log.Printf("获取到 %d 条新闻", len(newsItems))

	// 2. 调用Agent分析新闻
	symbol := "AAPL" // 默认标的
	analysis, err := qe.agentClient.AnalyzeNews(symbol, newsItems)
	if err != nil {
		qe.stats.FailedCycles++
		return fmt.Errorf("Agent分析失败: %w", err)
	}
	log.Printf("Agent分析完成: 情绪=%s, 置信度=%.2f, 原因=%s",
		analysis.Sentiment, analysis.ConfidenceScore, analysis.Reason)

	// 3. 获取市场数据
	df, err := qe.dataManager.GetMarketData(symbol,
		time.Now().AddDate(0, 0, -30).Format("2006-01-02"),
		time.Now().Format("2006-01-02"))
	if err != nil {
		qe.stats.FailedCycles++
		return fmt.Errorf("获取市场数据失败: %w", err)
	}
	log.Printf("获取到 %d 条市场数据", len(df["close"]))

	// 4. 转换Agent指导为策略指导
	guidance := &strategy.AgentGuidance{
		Sentiment:  analysis.Sentiment,
		Reason:     analysis.Reason,
		Confidence: analysis.ConfidenceScore,
		Timestamp:  analysis.Timestamp,
		Symbol:     symbol,
	}

	// 5. 生成交易信号
	signals, err := qe.strategyManager.ExecuteStrategy("ma_cross", df, guidance)
	if err != nil {
		qe.stats.FailedCycles++
		return fmt.Errorf("策略执行失败: %w", err)
	}
	log.Printf("策略生成 %d 个交易信号", len(signals))

	qe.stats.TotalSignals += len(signals)

	// 6. 执行交易
	for _, signal := range signals {
		if err := qe.executeTrade(signal); err != nil {
			log.Printf("执行交易失败: %v", err)
			continue
		}
		qe.stats.ExecutedTrades++
	}

	qe.stats.SuccessfulCycles++
	log.Printf("交易循环执行完成")
	return nil
}

// RunContinuous 运行连续循环
func (qe *QuantEngine) RunContinuous(interval time.Duration) error {
	log.Printf("开始连续运行，间隔: %v", interval)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-qe.stopChan:
			log.Printf("收到停止信号，退出连续运行")
			return nil
		case <-ticker.C:
			if err := qe.RunSingleLoop(); err != nil {
				log.Printf("交易循环执行失败: %v", err)
			}
		}
	}
}

// executeTrade 执行交易
func (qe *QuantEngine) executeTrade(signal strategy.TradingSignal) error {
	log.Printf("执行交易信号: %s %s %.2f @ %.2f",
		signal.Symbol, signal.Signal.String(), signal.Quantity, signal.Price)

	// 选择账户（简化处理，使用第一个账户）
	accounts := qe.accountManager.GetAllAccounts()
	if len(accounts) == 0 {
		return fmt.Errorf("没有可用的交易账户")
	}

	var accountName string
	for name := range accounts {
		accountName = name
		break
	}

	// 执行交易
	order, err := qe.tradingEngine.ExecuteSignal(signal, accountName)
	if err != nil {
		return fmt.Errorf("交易执行失败: %w", err)
	}

	log.Printf("交易执行成功: 订单ID=%s, 状态=%s", order.ID, order.Status)
	return nil
}

// getMockNews 获取模拟新闻
func (qe *QuantEngine) getMockNews() []string {
	newsItems := []string{
		"苹果公司发布新款iPhone，市场反应积极",
		"科技股整体上涨，投资者信心增强",
		"美联储维持利率不变，市场预期稳定",
		"分析师上调苹果目标价格至200美元",
		"全球供应链问题得到缓解，利好科技股",
	}

	return newsItems
}

// GetStats 获取引擎统计信息
func (qe *QuantEngine) GetStats() *EngineStats {
	qe.mutex.RLock()
	defer qe.mutex.RUnlock()

	// 返回副本
	stats := *qe.stats
	return &stats
}

// GetStatus 获取引擎状态
func (qe *QuantEngine) GetStatus() *EngineStatus {
	qe.mutex.RLock()
	defer qe.mutex.RUnlock()

	status := &EngineStatus{
		IsRunning:        qe.isRunning,
		StartTime:        qe.stats.StartTime,
		LastUpdateTime:   qe.stats.LastUpdateTime,
		TotalCycles:      qe.stats.TotalCycles,
		SuccessfulCycles: qe.stats.SuccessfulCycles,
		FailedCycles:     qe.stats.FailedCycles,
		TotalSignals:     qe.stats.TotalSignals,
		ExecutedTrades:   qe.stats.ExecutedTrades,
		TotalPnL:         qe.stats.TotalPnL,
	}

	// 获取账户状态
	status.Accounts = qe.accountManager.GetAllAccountStatuses()

	// 获取交易引擎状态
	status.TradingStatus = qe.tradingEngine.GetTradingStatus()

	// 获取策略状态
	status.Strategies = qe.strategyManager.GetAllStrategyStatuses()

	return status
}

// EngineStatus 引擎状态
type EngineStatus struct {
	IsRunning        bool                                `json:"is_running"`
	StartTime        time.Time                           `json:"start_time"`
	LastUpdateTime   time.Time                           `json:"last_update_time"`
	TotalCycles      int                                 `json:"total_cycles"`
	SuccessfulCycles int                                 `json:"successful_cycles"`
	FailedCycles     int                                 `json:"failed_cycles"`
	TotalSignals     int                                 `json:"total_signals"`
	ExecutedTrades   int                                 `json:"executed_trades"`
	TotalPnL         float64                             `json:"total_pnl"`
	Accounts         map[string]*account.AccountStatus   `json:"accounts"`
	TradingStatus    *trading.TradingStatus              `json:"trading_status"`
	Strategies       map[string]*strategy.StrategyStatus `json:"strategies"`
}

// RunBacktest 运行回测
func (qe *QuantEngine) RunBacktest(symbol, startDate, endDate string) error {
	log.Printf("开始运行回测: 标的=%s, 开始=%s, 结束=%s", symbol, startDate, endDate)

	// 获取策略
	strategy, err := qe.strategyManager.GetStrategy("ma_cross")
	if err != nil {
		return fmt.Errorf("获取策略失败: %w", err)
	}

	// 创建回测器
	backtester := backtest.NewBacktester(strategy, qe.dataManager,
		qe.config.Backtest.InitialCapital,
		qe.config.Backtest.CommissionRate,
		qe.config.Backtest.SlippageRate)

	// 运行回测
	result, err := backtester.Run(symbol, startDate, endDate)
	if err != nil {
		return fmt.Errorf("回测执行失败: %w", err)
	}

	// 打印回测结果
	qe.printBacktestResult(result)

	return nil
}

// printBacktestResult 打印回测结果
func (qe *QuantEngine) printBacktestResult(result *backtest.BacktestResult) {
	log.Printf("=== 回测结果 ===")
	log.Printf("策略名称: %s", result.StrategyName)
	log.Printf("标的符号: %s", result.Symbol)
	log.Printf("初始资金: %.2f", result.InitialCapital)
	log.Printf("最终资金: %.2f", result.FinalCapital)
	log.Printf("总收益率: %.2f%%", result.TotalReturn*100)
	log.Printf("年化收益率: %.2f%%", result.AnnualReturn*100)
	log.Printf("最大回撤: %.2f%%", result.MaxDrawdown*100)
	log.Printf("夏普比率: %.2f", result.SharpeRatio)
	log.Printf("索提诺比率: %.2f", result.SortinoRatio)
	log.Printf("总交易次数: %d", result.TotalTrades)
	log.Printf("胜率: %.2f%%", result.WinRate*100)
	log.Printf("平均盈利: %.2f", result.AvgWin)
	log.Printf("平均亏损: %.2f", result.AvgLoss)
	log.Printf("盈亏比: %.2f", result.ProfitFactor)
	log.Printf("最大连续盈利: %d", result.MaxConsecutiveWins)
	log.Printf("最大连续亏损: %d", result.MaxConsecutiveLosses)
	log.Printf("总佣金: %.2f", result.Commission)
	log.Printf("总滑点: %.2f", result.Slippage)
	log.Printf("==================")
}

// GetAccountBalance 获取账户余额
func (qe *QuantEngine) GetAccountBalance(accountName string) (float64, error) {
	return qe.tradingEngine.GetAccountBalance(accountName)
}

// GetAccountPositions 获取账户持仓
func (qe *QuantEngine) GetAccountPositions(accountName string) (map[string]trading.Position, error) {
	return qe.tradingEngine.GetAccountPositions(accountName)
}

// GetAccountOrders 获取账户订单
func (qe *QuantEngine) GetAccountOrders(accountName string, symbol string, status trading.OrderStatus) ([]trading.Order, error) {
	return qe.tradingEngine.GetAccountOrders(accountName, symbol, status)
}

// GetAccountTrades 获取账户成交记录
func (qe *QuantEngine) GetAccountTrades(accountName string, symbol string, limit int) ([]trading.Trade, error) {
	return qe.tradingEngine.GetAccountTrades(accountName, symbol, limit)
}

// RefreshAccountData 刷新账户数据
func (qe *QuantEngine) RefreshAccountData(accountName string) error {
	return qe.accountManager.RefreshAccountData(accountName)
}

// IsRunning 检查是否运行中
func (qe *QuantEngine) IsRunning() bool {
	qe.mutex.RLock()
	defer qe.mutex.RUnlock()
	return qe.isRunning
}

// UpdateStrategyParameters 更新策略参数
func (qe *QuantEngine) UpdateStrategyParameters(strategyName string, params strategy.StrategyParams) error {
	return qe.strategyManager.UpdateStrategyParameters(strategyName, params)
}

// GetAvailableStrategies 获取可用策略
func (qe *QuantEngine) GetAvailableStrategies() map[string]strategy.StrategyInfo {
	return qe.strategyManager.GetAvailableStrategies()
}

// HealthCheck 健康检查
func (qe *QuantEngine) HealthCheck() *HealthStatus {
	status := &HealthStatus{
		Timestamp: time.Now(),
		Services:  make(map[string]ServiceStatus),
	}

	// 检查Agent服务
	if err := qe.agentClient.HealthCheck(); err != nil {
		status.Services["agent"] = ServiceStatus{
			Name:   "Agent服务",
			Status: "unhealthy",
			Error:  err.Error(),
		}
	} else {
		status.Services["agent"] = ServiceStatus{
			Name:   "Agent服务",
			Status: "healthy",
		}
	}

	// 检查交易引擎
	if qe.tradingEngine.IsRunning() {
		status.Services["trading"] = ServiceStatus{
			Name:   "交易引擎",
			Status: "healthy",
		}
	} else {
		status.Services["trading"] = ServiceStatus{
			Name:   "交易引擎",
			Status: "unhealthy",
			Error:  "交易引擎未运行",
		}
	}

	// 检查数据管理器
	status.Services["data"] = ServiceStatus{
		Name:   "数据管理器",
		Status: "healthy",
	}

	// 检查策略管理器
	status.Services["strategy"] = ServiceStatus{
		Name:   "策略管理器",
		Status: "healthy",
	}

	// 检查账户管理器
	status.Services["account"] = ServiceStatus{
		Name:   "账户管理器",
		Status: "healthy",
	}

	// 计算总体健康状态
	allHealthy := true
	for _, service := range status.Services {
		if service.Status != "healthy" {
			allHealthy = false
			break
		}
	}

	if allHealthy {
		status.Overall = "healthy"
	} else {
		status.Overall = "unhealthy"
	}

	return status
}

// HealthStatus 健康状态
type HealthStatus struct {
	Timestamp time.Time                `json:"timestamp"`
	Overall   string                   `json:"overall"`
	Services  map[string]ServiceStatus `json:"services"`
}

// ServiceStatus 服务状态
type ServiceStatus struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Error  string `json:"error,omitempty"`
}
