package trading

import (
	"fmt"
	"log"
	"time"
)

// OrderType 订单类型
type OrderType string

const (
	MarketOrder OrderType = "market" // 市价单
	LimitOrder  OrderType = "limit"  // 限价单
	StopOrder   OrderType = "stop"   // 止损单
)

// OrderSide 订单方向
type OrderSide string

const (
	BuySide  OrderSide = "buy"  // 买入
	SellSide OrderSide = "sell" // 卖出
)

// OrderStatus 订单状态
type OrderStatus string

const (
	Pending   OrderStatus = "pending"   // 待处理
	Submitted OrderStatus = "submitted" // 已提交
	Filled    OrderStatus = "filled"    // 已成交
	Cancelled OrderStatus = "cancelled" // 已取消
	Rejected  OrderStatus = "rejected"  // 已拒绝
)

// Order 订单结构体
type Order struct {
	ID          string      `json:"id"`
	Symbol      string      `json:"symbol"`
	Side        OrderSide   `json:"side"`
	Type        OrderType   `json:"type"`
	Quantity    float64     `json:"quantity"`
	Price       float64     `json:"price"`
	StopPrice   float64     `json:"stop_price,omitempty"`
	Status      OrderStatus `json:"status"`
	FilledQty   float64     `json:"filled_quantity"`
	AvgPrice    float64     `json:"average_price"`
	Commission  float64     `json:"commission"`
	CreateTime  time.Time   `json:"create_time"`
	UpdateTime  time.Time   `json:"update_time"`
	AccountName string      `json:"account_name"`
	Strategy    string      `json:"strategy"`
}

// Trade 成交记录
type Trade struct {
	ID          string    `json:"id"`
	OrderID     string    `json:"order_id"`
	Symbol      string    `json:"symbol"`
	Side        OrderSide `json:"side"`
	Quantity    float64   `json:"quantity"`
	Price       float64   `json:"price"`
	Commission  float64   `json:"commission"`
	Timestamp   time.Time `json:"timestamp"`
	AccountName string    `json:"account_name"`
}

// BrokerAPI 经纪商API接口
type BrokerAPI interface {
	// PlaceOrder 下单
	PlaceOrder(order Order) (*Order, error)

	// CancelOrder 撤单
	CancelOrder(orderID string) error

	// GetOrder 查询订单
	GetOrder(orderID string) (*Order, error)

	// GetOrders 查询订单列表
	GetOrders(symbol string, status OrderStatus) ([]Order, error)

	// GetBalance 获取余额
	GetBalance() (float64, error)

	// GetPositions 获取持仓
	GetPositions() (map[string]Position, error)

	// GetTrades 获取成交记录
	GetTrades(symbol string, limit int) ([]Trade, error)

	// Connect 连接经纪商
	Connect() error

	// Disconnect 断开连接
	Disconnect() error
}

// Position 持仓信息
type Position struct {
	Symbol       string    `json:"symbol"`
	Quantity     float64   `json:"quantity"`
	AvgPrice     float64   `json:"average_price"`
	MarketValue  float64   `json:"market_value"`
	UnrealizedPL float64   `json:"unrealized_pnl"`
	RealizedPL   float64   `json:"realized_pnl"`
	UpdateTime   time.Time `json:"update_time"`
}

// MockStockBroker 模拟股票经纪商
type MockStockBroker struct {
	name        string
	balance     float64
	positions   map[string]Position
	orders      map[string]Order
	trades      []Trade
	isConnected bool
}

// NewMockStockBroker 创建模拟股票经纪商
func NewMockStockBroker(name string) *MockStockBroker {
	return &MockStockBroker{
		name:      name,
		balance:   100000.0,
		positions: make(map[string]Position),
		orders:    make(map[string]Order),
		trades:    make([]Trade, 0),
	}
}

// Connect 连接经纪商
func (b *MockStockBroker) Connect() error {
	log.Printf("连接到股票经纪商: %s", b.name)
	b.isConnected = true
	return nil
}

// Disconnect 断开连接
func (b *MockStockBroker) Disconnect() error {
	log.Printf("断开股票经纪商连接: %s", b.name)
	b.isConnected = false
	return nil
}

// PlaceOrder 下单
func (b *MockStockBroker) PlaceOrder(order Order) (*Order, error) {
	if !b.isConnected {
		return nil, fmt.Errorf("经纪商未连接")
	}

	log.Printf("股票经纪商 %s 收到订单: %s %s %.2f @ %.2f",
		b.name, order.Side, order.Symbol, order.Quantity, order.Price)

	// 模拟订单处理
	order.ID = fmt.Sprintf("STOCK_%d", time.Now().UnixNano())
	order.Status = Submitted
	order.CreateTime = time.Now()
	order.UpdateTime = time.Now()

	// 模拟订单成交
	if order.Type == MarketOrder {
		// 市价单立即成交
		order.Status = Filled
		order.FilledQty = order.Quantity
		order.AvgPrice = order.Price * 1.001 // 模拟滑点
		order.Commission = order.Quantity * order.AvgPrice * 0.001

		// 更新持仓和余额
		b.updatePosition(order)
		b.updateBalance(order)

		// 记录成交
		trade := Trade{
			ID:          fmt.Sprintf("TRADE_%d", time.Now().UnixNano()),
			OrderID:     order.ID,
			Symbol:      order.Symbol,
			Side:        order.Side,
			Quantity:    order.Quantity,
			Price:       order.AvgPrice,
			Commission:  order.Commission,
			Timestamp:   time.Now(),
			AccountName: order.AccountName,
		}
		b.trades = append(b.trades, trade)

		log.Printf("订单已成交: ID=%s, 成交价=%.2f", order.ID, order.AvgPrice)
	} else {
		// 限价单待成交
		b.orders[order.ID] = order
		log.Printf("限价单已提交: ID=%s", order.ID)
	}

	return &order, nil
}

// CancelOrder 撤单
func (b *MockStockBroker) CancelOrder(orderID string) error {
	if !b.isConnected {
		return fmt.Errorf("经纪商未连接")
	}

	order, exists := b.orders[orderID]
	if !exists {
		return fmt.Errorf("订单不存在: %s", orderID)
	}

	order.Status = Cancelled
	order.UpdateTime = time.Now()
	b.orders[orderID] = order

	log.Printf("订单已取消: ID=%s", orderID)
	return nil
}

// GetOrder 查询订单
func (b *MockStockBroker) GetOrder(orderID string) (*Order, error) {
	if !b.isConnected {
		return nil, fmt.Errorf("经纪商未连接")
	}

	order, exists := b.orders[orderID]
	if !exists {
		return nil, fmt.Errorf("订单不存在: %s", orderID)
	}

	return &order, nil
}

// GetOrders 查询订单列表
func (b *MockStockBroker) GetOrders(symbol string, status OrderStatus) ([]Order, error) {
	if !b.isConnected {
		return nil, fmt.Errorf("经纪商未连接")
	}

	var orders []Order
	for _, order := range b.orders {
		if symbol != "" && order.Symbol != symbol {
			continue
		}
		if status != "" && order.Status != status {
			continue
		}
		orders = append(orders, order)
	}

	return orders, nil
}

// GetBalance 获取余额
func (b *MockStockBroker) GetBalance() (float64, error) {
	if !b.isConnected {
		return 0, fmt.Errorf("经纪商未连接")
	}

	return b.balance, nil
}

// GetPositions 获取持仓
func (b *MockStockBroker) GetPositions() (map[string]Position, error) {
	if !b.isConnected {
		return nil, fmt.Errorf("经纪商未连接")
	}

	positions := make(map[string]Position)
	for symbol, position := range b.positions {
		positions[symbol] = position
	}

	return positions, nil
}

// GetTrades 获取成交记录
func (b *MockStockBroker) GetTrades(symbol string, limit int) ([]Trade, error) {
	if !b.isConnected {
		return nil, fmt.Errorf("经纪商未连接")
	}

	var trades []Trade
	count := 0
	for i := len(b.trades) - 1; i >= 0 && count < limit; i-- {
		if symbol != "" && b.trades[i].Symbol != symbol {
			continue
		}
		trades = append([]Trade{b.trades[i]}, trades...)
		count++
	}

	return trades, nil
}

// updatePosition 更新持仓
func (b *MockStockBroker) updatePosition(order Order) {
	position, exists := b.positions[order.Symbol]

	if !exists {
		position = Position{
			Symbol:      order.Symbol,
			Quantity:    0,
			AvgPrice:    0,
			MarketValue: 0,
			UpdateTime:  time.Now(),
		}
	}

	if order.Side == BuySide {
		// 买入
		totalCost := position.Quantity*position.AvgPrice + order.Quantity*order.AvgPrice
		position.Quantity += order.Quantity
		if position.Quantity > 0 {
			position.AvgPrice = totalCost / position.Quantity
		}
	} else {
		// 卖出
		position.Quantity -= order.Quantity
		if position.Quantity <= 0 {
			delete(b.positions, order.Symbol)
			return
		}
	}

	position.MarketValue = position.Quantity * order.AvgPrice
	position.UpdateTime = time.Now()
	b.positions[order.Symbol] = position
}

// updateBalance 更新余额
func (b *MockStockBroker) updateBalance(order Order) {
	if order.Side == BuySide {
		// 买入减少余额
		b.balance -= order.Quantity*order.AvgPrice + order.Commission
	} else {
		// 卖出增加余额
		b.balance += order.Quantity*order.AvgPrice - order.Commission
	}
}

// MockCryptoBroker 模拟加密货币交易所
type MockCryptoBroker struct {
	name        string
	balance     float64
	positions   map[string]Position
	orders      map[string]Order
	trades      []Trade
	isConnected bool
}

// NewMockCryptoBroker 创建模拟加密货币交易所
func NewMockCryptoBroker(name string) *MockCryptoBroker {
	return &MockCryptoBroker{
		name:      name,
		balance:   100000.0,
		positions: make(map[string]Position),
		orders:    make(map[string]Order),
		trades:    make([]Trade, 0),
	}
}

// Connect 连接交易所
func (b *MockCryptoBroker) Connect() error {
	log.Printf("连接到加密货币交易所: %s", b.name)
	b.isConnected = true
	return nil
}

// Disconnect 断开连接
func (b *MockCryptoBroker) Disconnect() error {
	log.Printf("断开加密货币交易所连接: %s", b.name)
	b.isConnected = false
	return nil
}

// PlaceOrder 下单
func (b *MockCryptoBroker) PlaceOrder(order Order) (*Order, error) {
	if !b.isConnected {
		return nil, fmt.Errorf("交易所未连接")
	}

	log.Printf("加密货币交易所 %s 收到订单: %s %s %.2f @ %.2f",
		b.name, order.Side, order.Symbol, order.Quantity, order.Price)

	// 模拟订单处理
	order.ID = fmt.Sprintf("CRYPTO_%d", time.Now().UnixNano())
	order.Status = Submitted
	order.CreateTime = time.Now()
	order.UpdateTime = time.Now()

	// 模拟订单成交
	if order.Type == MarketOrder {
		// 市价单立即成交
		order.Status = Filled
		order.FilledQty = order.Quantity
		order.AvgPrice = order.Price * 1.002 // 模拟更大的滑点
		order.Commission = order.Quantity * order.AvgPrice * 0.001

		// 更新持仓和余额
		b.updatePosition(order)
		b.updateBalance(order)

		// 记录成交
		trade := Trade{
			ID:          fmt.Sprintf("TRADE_%d", time.Now().UnixNano()),
			OrderID:     order.ID,
			Symbol:      order.Symbol,
			Side:        order.Side,
			Quantity:    order.Quantity,
			Price:       order.AvgPrice,
			Commission:  order.Commission,
			Timestamp:   time.Now(),
			AccountName: order.AccountName,
		}
		b.trades = append(b.trades, trade)

		log.Printf("订单已成交: ID=%s, 成交价=%.2f", order.ID, order.AvgPrice)
	} else {
		// 限价单待成交
		b.orders[order.ID] = order
		log.Printf("限价单已提交: ID=%s", order.ID)
	}

	return &order, nil
}

// CancelOrder 撤单
func (b *MockCryptoBroker) CancelOrder(orderID string) error {
	if !b.isConnected {
		return fmt.Errorf("交易所未连接")
	}

	order, exists := b.orders[orderID]
	if !exists {
		return fmt.Errorf("订单不存在: %s", orderID)
	}

	order.Status = Cancelled
	order.UpdateTime = time.Now()
	b.orders[orderID] = order

	log.Printf("订单已取消: ID=%s", orderID)
	return nil
}

// GetOrder 查询订单
func (b *MockCryptoBroker) GetOrder(orderID string) (*Order, error) {
	if !b.isConnected {
		return nil, fmt.Errorf("交易所未连接")
	}

	order, exists := b.orders[orderID]
	if !exists {
		return nil, fmt.Errorf("订单不存在: %s", orderID)
	}

	return &order, nil
}

// GetOrders 查询订单列表
func (b *MockCryptoBroker) GetOrders(symbol string, status OrderStatus) ([]Order, error) {
	if !b.isConnected {
		return nil, fmt.Errorf("交易所未连接")
	}

	var orders []Order
	for _, order := range b.orders {
		if symbol != "" && order.Symbol != symbol {
			continue
		}
		if status != "" && order.Status != status {
			continue
		}
		orders = append(orders, order)
	}

	return orders, nil
}

// GetBalance 获取余额
func (b *MockCryptoBroker) GetBalance() (float64, error) {
	if !b.isConnected {
		return 0, fmt.Errorf("交易所未连接")
	}

	return b.balance, nil
}

// GetPositions 获取持仓
func (b *MockCryptoBroker) GetPositions() (map[string]Position, error) {
	if !b.isConnected {
		return nil, fmt.Errorf("交易所未连接")
	}

	positions := make(map[string]Position)
	for symbol, position := range b.positions {
		positions[symbol] = position
	}

	return positions, nil
}

// GetTrades 获取成交记录
func (b *MockCryptoBroker) GetTrades(symbol string, limit int) ([]Trade, error) {
	if !b.isConnected {
		return nil, fmt.Errorf("交易所未连接")
	}

	var trades []Trade
	count := 0
	for i := len(b.trades) - 1; i >= 0 && count < limit; i-- {
		if symbol != "" && b.trades[i].Symbol != symbol {
			continue
		}
		trades = append([]Trade{b.trades[i]}, trades...)
		count++
	}

	return trades, nil
}

// updatePosition 更新持仓
func (b *MockCryptoBroker) updatePosition(order Order) {
	position, exists := b.positions[order.Symbol]

	if !exists {
		position = Position{
			Symbol:      order.Symbol,
			Quantity:    0,
			AvgPrice:    0,
			MarketValue: 0,
			UpdateTime:  time.Now(),
		}
	}

	if order.Side == BuySide {
		// 买入
		totalCost := position.Quantity*position.AvgPrice + order.Quantity*order.AvgPrice
		position.Quantity += order.Quantity
		if position.Quantity > 0 {
			position.AvgPrice = totalCost / position.Quantity
		}
	} else {
		// 卖出
		position.Quantity -= order.Quantity
		if position.Quantity <= 0 {
			delete(b.positions, order.Symbol)
			return
		}
	}

	position.MarketValue = position.Quantity * order.AvgPrice
	position.UpdateTime = time.Now()
	b.positions[order.Symbol] = position
}

// updateBalance 更新余额
func (b *MockCryptoBroker) updateBalance(order Order) {
	if order.Side == BuySide {
		// 买入减少余额
		b.balance -= order.Quantity*order.AvgPrice + order.Commission
	} else {
		// 卖出增加余额
		b.balance += order.Quantity*order.AvgPrice - order.Commission
	}
}
