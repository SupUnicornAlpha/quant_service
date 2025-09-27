package account

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"sync"
	"time"

	"agent-quant-system/internal/config"
)

// AccountManager 账户管理器
type AccountManager struct {
	config   *config.Config
	accounts map[string]*Account
	mutex    sync.RWMutex
}

// Account 账户信息
type Account struct {
	Name        string              `json:"name"`
	BrokerType  string              `json:"broker_type"`
	APIKey      string              `json:"api_key"`
	APISecret   string              `json:"api_secret"`
	Credentials AccountCredentials  `json:"credentials"`
	Balance     float64             `json:"balance"`
	Positions   map[string]Position `json:"positions"`
	IsActive    bool                `json:"is_active"`
	LastUpdate  time.Time           `json:"last_update"`
}

// AccountCredentials 账户凭证
type AccountCredentials struct {
	APIKey     string `json:"api_key"`
	APISecret  string `json:"api_secret"`
	BrokerType string `json:"broker_type"`
	// 其他特定经纪商的凭证
	Passphrase string `json:"passphrase,omitempty"` // 用于某些交易所
	Sandbox    bool   `json:"sandbox,omitempty"`    // 是否使用沙盒环境
}

// Position 持仓信息
type Position struct {
	Symbol       string    `json:"symbol"`
	Quantity     float64   `json:"quantity"`
	AvgPrice     float64   `json:"avg_price"`
	MarketValue  float64   `json:"market_value"`
	UnrealizedPL float64   `json:"unrealized_pl"`
	RealizedPL   float64   `json:"realized_pl"`
	OpenTime     time.Time `json:"open_time"`
	LastUpdate   time.Time `json:"last_update"`
}

// BalanceInfo 余额信息
type BalanceInfo struct {
	TotalBalance     float64   `json:"total_balance"`
	AvailableBalance float64   `json:"available_balance"`
	FrozenBalance    float64   `json:"frozen_balance"`
	Currency         string    `json:"currency"`
	LastUpdate       time.Time `json:"last_update"`
}

// NewAccountManager 创建账户管理器
func NewAccountManager(cfg *config.Config) *AccountManager {
	manager := &AccountManager{
		config:   cfg,
		accounts: make(map[string]*Account),
	}

	// 初始化账户
	manager.initializeAccounts()

	return manager
}

// initializeAccounts 初始化账户
func (am *AccountManager) initializeAccounts() {
	log.Printf("初始化账户管理器")

	for name, accountConfig := range am.config.Accounts {
		account := &Account{
			Name:       name,
			BrokerType: accountConfig.BrokerType,
			APIKey:     accountConfig.APIKey,
			APISecret:  accountConfig.APISecret,
			Credentials: AccountCredentials{
				APIKey:     accountConfig.APIKey,
				APISecret:  accountConfig.APISecret,
				BrokerType: accountConfig.BrokerType,
			},
			Balance:    100000.0, // 模拟初始余额
			Positions:  make(map[string]Position),
			IsActive:   true,
			LastUpdate: time.Now(),
		}

		am.accounts[name] = account
		log.Printf("已初始化账户: %s (%s)", name, accountConfig.BrokerType)
	}
}

// GetAccount 获取账户
func (am *AccountManager) GetAccount(name string) (*Account, error) {
	am.mutex.RLock()
	defer am.mutex.RUnlock()

	account, exists := am.accounts[name]
	if !exists {
		return nil, fmt.Errorf("账户 '%s' 不存在", name)
	}

	return account, nil
}

// GetAccountCredentials 获取账户凭证
func (am *AccountManager) GetAccountCredentials(name string) (*AccountCredentials, error) {
	account, err := am.GetAccount(name)
	if err != nil {
		return nil, err
	}

	return &account.Credentials, nil
}

// GetAllAccounts 获取所有账户
func (am *AccountManager) GetAllAccounts() map[string]*Account {
	am.mutex.RLock()
	defer am.mutex.RUnlock()

	// 返回副本以避免并发修改
	accounts := make(map[string]*Account)
	for name, account := range am.accounts {
		accounts[name] = account
	}

	return accounts
}

// UpdateAccountBalance 更新账户余额
func (am *AccountManager) UpdateAccountBalance(name string, balance float64) error {
	am.mutex.Lock()
	defer am.mutex.Unlock()

	account, exists := am.accounts[name]
	if !exists {
		return fmt.Errorf("账户 '%s' 不存在", name)
	}

	account.Balance = balance
	account.LastUpdate = time.Now()

	log.Printf("已更新账户 '%s' 余额: %.2f", name, balance)
	return nil
}

// AddPosition 添加持仓
func (am *AccountManager) AddPosition(accountName, symbol string, quantity, avgPrice float64) error {
	am.mutex.Lock()
	defer am.mutex.Unlock()

	account, exists := am.accounts[accountName]
	if !exists {
		return fmt.Errorf("账户 '%s' 不存在", accountName)
	}

	position := Position{
		Symbol:       symbol,
		Quantity:     quantity,
		AvgPrice:     avgPrice,
		MarketValue:  quantity * avgPrice,
		UnrealizedPL: 0.0,
		RealizedPL:   0.0,
		OpenTime:     time.Now(),
		LastUpdate:   time.Now(),
	}

	account.Positions[symbol] = position
	account.LastUpdate = time.Now()

	log.Printf("已添加持仓: 账户=%s, 标的=%s, 数量=%.2f, 均价=%.2f",
		accountName, symbol, quantity, avgPrice)

	return nil
}

// UpdatePosition 更新持仓
func (am *AccountManager) UpdatePosition(accountName, symbol string, quantity, avgPrice float64) error {
	am.mutex.Lock()
	defer am.mutex.Unlock()

	account, exists := am.accounts[accountName]
	if !exists {
		return fmt.Errorf("账户 '%s' 不存在", accountName)
	}

	position, exists := account.Positions[symbol]
	if !exists {
		return fmt.Errorf("持仓 '%s' 不存在", symbol)
	}

	position.Quantity = quantity
	position.AvgPrice = avgPrice
	position.MarketValue = quantity * avgPrice
	position.LastUpdate = time.Now()

	account.Positions[symbol] = position
	account.LastUpdate = time.Now()

	log.Printf("已更新持仓: 账户=%s, 标的=%s, 数量=%.2f, 均价=%.2f",
		accountName, symbol, quantity, avgPrice)

	return nil
}

// RemovePosition 移除持仓
func (am *AccountManager) RemovePosition(accountName, symbol string) error {
	am.mutex.Lock()
	defer am.mutex.Unlock()

	account, exists := am.accounts[accountName]
	if !exists {
		return fmt.Errorf("账户 '%s' 不存在", accountName)
	}

	if _, exists := account.Positions[symbol]; !exists {
		return fmt.Errorf("持仓 '%s' 不存在", symbol)
	}

	delete(account.Positions, symbol)
	account.LastUpdate = time.Now()

	log.Printf("已移除持仓: 账户=%s, 标的=%s", accountName, symbol)

	return nil
}

// GetPosition 获取持仓
func (am *AccountManager) GetPosition(accountName, symbol string) (*Position, error) {
	am.mutex.RLock()
	defer am.mutex.RUnlock()

	account, exists := am.accounts[accountName]
	if !exists {
		return nil, fmt.Errorf("账户 '%s' 不存在", accountName)
	}

	position, exists := account.Positions[symbol]
	if !exists {
		return nil, fmt.Errorf("持仓 '%s' 不存在", symbol)
	}

	return &position, nil
}

// GetAllPositions 获取所有持仓
func (am *AccountManager) GetAllPositions(accountName string) (map[string]Position, error) {
	am.mutex.RLock()
	defer am.mutex.RUnlock()

	account, exists := am.accounts[accountName]
	if !exists {
		return nil, fmt.Errorf("账户 '%s' 不存在", accountName)
	}

	positions := make(map[string]Position)
	for symbol, position := range account.Positions {
		positions[symbol] = position
	}

	return positions, nil
}

// GetBalanceInfo 获取余额信息
func (am *AccountManager) GetBalanceInfo(accountName string) (*BalanceInfo, error) {
	account, err := am.GetAccount(accountName)
	if err != nil {
		return nil, err
	}

	// 计算持仓市值
	totalPositionValue := 0.0
	for _, position := range account.Positions {
		totalPositionValue += position.MarketValue
	}

	balanceInfo := &BalanceInfo{
		TotalBalance:     account.Balance,
		AvailableBalance: account.Balance - totalPositionValue,
		FrozenBalance:    0.0, // 模拟冻结余额
		Currency:         "USD",
		LastUpdate:       time.Now(),
	}

	return balanceInfo, nil
}

// ValidateAccountCredentials 验证账户凭证
func (am *AccountManager) ValidateAccountCredentials(accountName string) error {
	account, err := am.GetAccount(accountName)
	if err != nil {
		return err
	}

	if account.APIKey == "" || account.APISecret == "" {
		return fmt.Errorf("账户 '%s' 的API凭证不完整", accountName)
	}

	if account.BrokerType == "" {
		return fmt.Errorf("账户 '%s' 的经纪商类型未设置", accountName)
	}

	log.Printf("账户 '%s' 凭证验证通过", accountName)
	return nil
}

// GetAccountHash 获取账户哈希（用于安全标识）
func (am *AccountManager) GetAccountHash(accountName string) (string, error) {
	account, err := am.GetAccount(accountName)
	if err != nil {
		return "", err
	}

	// 使用API Key生成哈希
	hash := sha256.Sum256([]byte(account.APIKey))
	return hex.EncodeToString(hash[:]), nil
}

// SetAccountActive 设置账户激活状态
func (am *AccountManager) SetAccountActive(accountName string, active bool) error {
	am.mutex.Lock()
	defer am.mutex.Unlock()

	account, exists := am.accounts[accountName]
	if !exists {
		return fmt.Errorf("账户 '%s' 不存在", accountName)
	}

	account.IsActive = active
	account.LastUpdate = time.Now()

	log.Printf("账户 '%s' 状态已更新为: %v", accountName, active)
	return nil
}

// GetAccountStatus 获取账户状态
func (am *AccountManager) GetAccountStatus(accountName string) (*AccountStatus, error) {
	account, err := am.GetAccount(accountName)
	if err != nil {
		return nil, err
	}

	balanceInfo, err := am.GetBalanceInfo(accountName)
	if err != nil {
		return nil, err
	}

	status := &AccountStatus{
		Name:             account.Name,
		BrokerType:       account.BrokerType,
		IsActive:         account.IsActive,
		Balance:          account.Balance,
		AvailableBalance: balanceInfo.AvailableBalance,
		PositionCount:    len(account.Positions),
		LastUpdate:       account.LastUpdate,
	}

	return status, nil
}

// AccountStatus 账户状态
type AccountStatus struct {
	Name             string    `json:"name"`
	BrokerType       string    `json:"broker_type"`
	IsActive         bool      `json:"is_active"`
	Balance          float64   `json:"balance"`
	AvailableBalance float64   `json:"available_balance"`
	PositionCount    int       `json:"position_count"`
	LastUpdate       time.Time `json:"last_update"`
}

// GetAllAccountStatuses 获取所有账户状态
func (am *AccountManager) GetAllAccountStatuses() map[string]*AccountStatus {
	am.mutex.RLock()
	defer am.mutex.RUnlock()

	statuses := make(map[string]*AccountStatus)
	for name := range am.accounts {
		status, err := am.GetAccountStatus(name)
		if err != nil {
			log.Printf("获取账户 '%s' 状态失败: %v", name, err)
			continue
		}
		statuses[name] = status
	}

	return statuses
}

// RefreshAccountData 刷新账户数据
func (am *AccountManager) RefreshAccountData(accountName string) error {
	account, err := am.GetAccount(accountName)
	if err != nil {
		return err
	}

	// 模拟从经纪商API获取最新数据
	log.Printf("正在刷新账户 '%s' 的数据", accountName)

	// 更新余额（模拟）
	account.Balance += float64(time.Now().Unix() % 100) // 模拟余额变化
	account.LastUpdate = time.Now()

	// 更新持仓市值（模拟）
	for symbol, position := range account.Positions {
		// 模拟价格变化
		priceChange := (float64(time.Now().Unix()%100) - 50) / 1000.0
		newPrice := position.AvgPrice * (1 + priceChange)
		position.MarketValue = position.Quantity * newPrice
		position.UnrealizedPL = position.MarketValue - (position.Quantity * position.AvgPrice)
		position.LastUpdate = time.Now()
		account.Positions[symbol] = position
	}

	log.Printf("账户 '%s' 数据刷新完成", accountName)
	return nil
}
