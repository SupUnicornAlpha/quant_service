package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"agent-quant-system/internal/config"
	"agent-quant-system/internal/core"

	"github.com/spf13/cobra"
)

var (
	configFile string
	symbol     string
	startDate  string
	endDate    string
	interval   time.Duration
)

// rootCmd 根命令
var rootCmd = &cobra.Command{
	Use:   "quant-system",
	Short: "Agent Quant System - 混合架构量化交易系统",
	Long: `Agent Quant System 是一个基于 Go 和 Python 的混合架构量化交易系统。
Go 负责高性能的引擎和业务逻辑，Python 负责与大语言模型交互的灵活性。

主要功能：
- 实时交易执行
- 历史数据回测
- Agent 智能分析
- 多策略支持
- 风险控制`,
	Version: "1.0.0",
}

// runCmd 运行命令
var runCmd = &cobra.Command{
	Use:   "run",
	Short: "运行量化交易系统",
	Long:  `启动量化交易系统，执行实时交易循环`,
	RunE:  runSystem,
}

// backtestCmd 回测命令
var backtestCmd = &cobra.Command{
	Use:   "backtest",
	Short: "运行策略回测",
	Long:  `对指定策略进行历史数据回测分析`,
	RunE:  runBacktest,
}

// statusCmd 状态命令
var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "查看系统状态",
	Long:  `查看量化交易系统的运行状态和统计信息`,
	RunE:  showStatus,
}

// healthCmd 健康检查命令
var healthCmd = &cobra.Command{
	Use:   "health",
	Short: "系统健康检查",
	Long:  `检查系统各个组件的健康状态`,
	RunE:  checkHealth,
}

func init() {
	// 添加全局标志
	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "config.toml", "配置文件路径")

	// 添加 run 命令标志
	runCmd.Flags().StringVarP(&symbol, "symbol", "s", "AAPL", "交易标的")
	runCmd.Flags().DurationVarP(&interval, "interval", "i", 5*time.Minute, "交易循环间隔")

	// 添加 backtest 命令标志
	backtestCmd.Flags().StringVarP(&symbol, "symbol", "s", "AAPL", "回测标的")
	backtestCmd.Flags().StringVar(&startDate, "start", "", "开始日期 (YYYY-MM-DD)")
	backtestCmd.Flags().StringVar(&endDate, "end", "", "结束日期 (YYYY-MM-DD)")

	// 添加子命令
	rootCmd.AddCommand(runCmd)
	rootCmd.AddCommand(backtestCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(healthCmd)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		log.Fatalf("命令执行失败: %v", err)
	}
}

// runSystem 运行系统
func runSystem(cmd *cobra.Command, args []string) error {
	log.Printf("启动 Agent Quant System")

	// 加载配置
	cfg, err := config.LoadConfig(configFile)
	if err != nil {
		return fmt.Errorf("加载配置失败: %w", err)
	}

	// 验证配置
	if err := cfg.Validate(); err != nil {
		return fmt.Errorf("配置验证失败: %w", err)
	}

	// 创建量化引擎
	engine, err := core.NewQuantEngine(cfg)
	if err != nil {
		return fmt.Errorf("创建量化引擎失败: %w", err)
	}

	// 启动引擎
	if err := engine.Start(); err != nil {
		return fmt.Errorf("启动量化引擎失败: %w", err)
	}
	defer func() {
		if err := engine.Stop(); err != nil {
			log.Printf("停止量化引擎失败: %v", err)
		}
	}()

	log.Printf("量化引擎已启动，交易标的: %s, 循环间隔: %v", symbol, interval)

	// 设置信号处理
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// 启动连续运行
	go func() {
		if err := engine.RunContinuous(interval); err != nil {
			log.Printf("连续运行失败: %v", err)
		}
	}()

	// 等待停止信号
	<-sigChan
	log.Printf("收到停止信号，正在关闭系统...")

	return nil
}

// runBacktest 运行回测
func runBacktest(cmd *cobra.Command, args []string) error {
	log.Printf("开始运行回测")

	// 设置默认日期
	if startDate == "" {
		startDate = time.Now().AddDate(0, 0, -90).Format("2006-01-02")
	}
	if endDate == "" {
		endDate = time.Now().Format("2006-01-02")
	}

	// 加载配置
	cfg, err := config.LoadConfig(configFile)
	if err != nil {
		return fmt.Errorf("加载配置失败: %w", err)
	}

	// 创建量化引擎
	engine, err := core.NewQuantEngine(cfg)
	if err != nil {
		return fmt.Errorf("创建量化引擎失败: %w", err)
	}

	log.Printf("回测参数: 标的=%s, 开始日期=%s, 结束日期=%s", symbol, startDate, endDate)

	// 运行回测
	if err := engine.RunBacktest(symbol, startDate, endDate); err != nil {
		return fmt.Errorf("回测执行失败: %w", err)
	}

	log.Printf("回测完成")
	return nil
}

// showStatus 显示状态
func showStatus(cmd *cobra.Command, args []string) error {
	log.Printf("查看系统状态")

	// 加载配置
	cfg, err := config.LoadConfig(configFile)
	if err != nil {
		return fmt.Errorf("加载配置失败: %w", err)
	}

	// 创建量化引擎
	engine, err := core.NewQuantEngine(cfg)
	if err != nil {
		return fmt.Errorf("创建量化引擎失败: %w", err)
	}

	// 获取状态
	status := engine.GetStatus()

	// 打印状态信息
	fmt.Printf("\n=== 系统状态 ===\n")
	fmt.Printf("运行状态: %v\n", status.IsRunning)
	fmt.Printf("启动时间: %s\n", status.StartTime.Format("2006-01-02 15:04:05"))
	fmt.Printf("最后更新: %s\n", status.LastUpdateTime.Format("2006-01-02 15:04:05"))
	fmt.Printf("总循环数: %d\n", status.TotalCycles)
	fmt.Printf("成功循环: %d\n", status.SuccessfulCycles)
	fmt.Printf("失败循环: %d\n", status.FailedCycles)
	fmt.Printf("总信号数: %d\n", status.TotalSignals)
	fmt.Printf("已执行交易: %d\n", status.ExecutedTrades)
	fmt.Printf("总盈亏: %.2f\n", status.TotalPnL)

	// 打印账户状态
	fmt.Printf("\n=== 账户状态 ===\n")
	for name, account := range status.Accounts {
		fmt.Printf("账户: %s\n", name)
		fmt.Printf("  类型: %s\n", account.BrokerType)
		fmt.Printf("  状态: %v\n", account.IsActive)
		fmt.Printf("  余额: %.2f\n", account.Balance)
		fmt.Printf("  可用余额: %.2f\n", account.AvailableBalance)
		fmt.Printf("  持仓数量: %d\n", account.PositionCount)
		fmt.Printf("  最后更新: %s\n", account.LastUpdate.Format("2006-01-02 15:04:05"))
	}

	// 打印策略状态
	fmt.Printf("\n=== 策略状态 ===\n")
	for name, strategy := range status.Strategies {
		fmt.Printf("策略: %s\n", name)
		fmt.Printf("  名称: %s\n", strategy.Name)
		fmt.Printf("  状态: %v\n", strategy.IsActive)
		fmt.Printf("  描述: %s\n", strategy.Description)
	}

	// 打印交易引擎状态
	fmt.Printf("\n=== 交易引擎状态 ===\n")
	fmt.Printf("运行状态: %v\n", status.TradingStatus.IsRunning)
	fmt.Printf("经纪商数量: %d\n", len(status.TradingStatus.Brokers))
	for name, broker := range status.TradingStatus.Brokers {
		fmt.Printf("  经纪商: %s (%s)\n", name, broker.Status)
	}

	return nil
}

// checkHealth 健康检查
func checkHealth(cmd *cobra.Command, args []string) error {
	log.Printf("执行系统健康检查")

	// 加载配置
	cfg, err := config.LoadConfig(configFile)
	if err != nil {
		return fmt.Errorf("加载配置失败: %w", err)
	}

	// 创建量化引擎
	engine, err := core.NewQuantEngine(cfg)
	if err != nil {
		return fmt.Errorf("创建量化引擎失败: %w", err)
	}

	// 执行健康检查
	health := engine.HealthCheck()

	// 打印健康状态
	fmt.Printf("\n=== 系统健康检查 ===\n")
	fmt.Printf("检查时间: %s\n", health.Timestamp.Format("2006-01-02 15:04:05"))
	fmt.Printf("总体状态: %s\n", health.Overall)

	fmt.Printf("\n=== 服务状态 ===\n")
	for _, service := range health.Services {
		statusIcon := "✓"
		if service.Status != "healthy" {
			statusIcon = "✗"
		}
		fmt.Printf("%s %s: %s\n", statusIcon, service.Name, service.Status)
		if service.Error != "" {
			fmt.Printf("   错误: %s\n", service.Error)
		}
	}

	if health.Overall == "healthy" {
		fmt.Printf("\n系统状态良好，所有服务正常运行\n")
		return nil
	} else {
		fmt.Printf("\n系统存在健康问题，请检查上述错误\n")
		return fmt.Errorf("系统健康检查失败")
	}
}

// runSingleLoop 运行单次循环（用于测试）
func runSingleLoop(cmd *cobra.Command, args []string) error {
	log.Printf("执行单次交易循环")

	// 加载配置
	cfg, err := config.LoadConfig(configFile)
	if err != nil {
		return fmt.Errorf("加载配置失败: %w", err)
	}

	// 创建量化引擎
	engine, err := core.NewQuantEngine(cfg)
	if err != nil {
		return fmt.Errorf("创建量化引擎失败: %w", err)
	}

	// 启动引擎
	if err := engine.Start(); err != nil {
		return fmt.Errorf("启动量化引擎失败: %w", err)
	}
	defer func() {
		if err := engine.Stop(); err != nil {
			log.Printf("停止量化引擎失败: %v", err)
		}
	}()

	// 执行单次循环
	if err := engine.RunSingleLoop(); err != nil {
		return fmt.Errorf("执行单次循环失败: %w", err)
	}

	log.Printf("单次交易循环执行完成")
	return nil
}

// init 初始化函数
func init() {
	// 设置日志格式
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	// 添加单次循环命令（用于测试）
	singleLoopCmd := &cobra.Command{
		Use:   "single",
		Short: "执行单次交易循环",
		Long:  `执行一次完整的交易决策流程，用于测试和调试`,
		RunE:  runSingleLoop,
	}
	singleLoopCmd.Flags().StringVarP(&symbol, "symbol", "s", "AAPL", "交易标的")
	rootCmd.AddCommand(singleLoopCmd)
}
