package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
)

// Config 系统配置结构体
type Config struct {
	AgentService AgentServiceConfig       `mapstructure:"agent_service"`
	APIKeys      APIKeysConfig            `mapstructure:"api_keys"`
	Accounts     map[string]AccountConfig `mapstructure:"accounts"`
	Database     DatabaseConfig           `mapstructure:"database"`
	Logging      LoggingConfig            `mapstructure:"logging"`
	Backtest     BacktestConfig           `mapstructure:"backtest"`
}

// AgentServiceConfig Agent服务配置
type AgentServiceConfig struct {
	URL string `mapstructure:"url"`
}

// APIKeysConfig API密钥配置
type APIKeysConfig struct {
	OpenAIKey string `mapstructure:"openai_key"`
}

// AccountConfig 账户配置
type AccountConfig struct {
	APIKey     string `mapstructure:"api_key"`
	APISecret  string `mapstructure:"api_secret"`
	BrokerType string `mapstructure:"broker_type"`
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Host         string `mapstructure:"host"`
	Port         int    `mapstructure:"port"`
	Username     string `mapstructure:"username"`
	Password     string `mapstructure:"password"`
	DatabaseName string `mapstructure:"database_name"`
}

// LoggingConfig 日志配置
type LoggingConfig struct {
	Level string `mapstructure:"level"`
	File  string `mapstructure:"file"`
}

// BacktestConfig 回测配置
type BacktestConfig struct {
	InitialCapital float64 `mapstructure:"initial_capital"`
	CommissionRate float64 `mapstructure:"commission_rate"`
	SlippageRate   float64 `mapstructure:"slippage_rate"`
}

// LoadConfig 加载配置文件
func LoadConfig(path string) (*Config, error) {
	viper.SetConfigFile(path)
	viper.SetConfigType("toml")

	// 设置环境变量前缀
	viper.SetEnvPrefix("QUANT")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	// 设置默认值
	setDefaults()

	// 读取配置文件
	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("读取配置文件失败: %w", err)
	}

	var config Config
	if err := viper.Unmarshal(&config); err != nil {
		return nil, fmt.Errorf("解析配置文件失败: %w", err)
	}

	// 从环境变量覆盖敏感信息
	overrideFromEnv(&config)

	return &config, nil
}

// setDefaults 设置默认配置值
func setDefaults() {
	viper.SetDefault("agent_service.url", "http://localhost:8000")
	viper.SetDefault("logging.level", "info")
	viper.SetDefault("logging.file", "logs/quant_system.log")
	viper.SetDefault("backtest.initial_capital", 100000.0)
	viper.SetDefault("backtest.commission_rate", 0.001)
	viper.SetDefault("backtest.slippage_rate", 0.0005)
}

// overrideFromEnv 从环境变量覆盖敏感配置
func overrideFromEnv(config *Config) {
	if openaiKey := os.Getenv("OPENAI_API_KEY"); openaiKey != "" {
		config.APIKeys.OpenAIKey = openaiKey
	}

	// 可以添加更多环境变量覆盖逻辑
}

// GetAccountConfig 获取指定账户的配置
func (c *Config) GetAccountConfig(accountName string) (*AccountConfig, error) {
	account, exists := c.Accounts[accountName]
	if !exists {
		return nil, fmt.Errorf("账户 '%s' 不存在", accountName)
	}
	return &account, nil
}

// Validate 验证配置的有效性
func (c *Config) Validate() error {
	if c.AgentService.URL == "" {
		return fmt.Errorf("agent_service.url 不能为空")
	}

	if c.APIKeys.OpenAIKey == "" {
		return fmt.Errorf("openai_key 不能为空")
	}

	if len(c.Accounts) == 0 {
		return fmt.Errorf("至少需要配置一个账户")
	}

	for name, account := range c.Accounts {
		if account.APIKey == "" || account.APISecret == "" {
			return fmt.Errorf("账户 '%s' 的 API 密钥不能为空", name)
		}
		if account.BrokerType == "" {
			return fmt.Errorf("账户 '%s' 的经纪商类型不能为空", name)
		}
	}

	return nil
}
