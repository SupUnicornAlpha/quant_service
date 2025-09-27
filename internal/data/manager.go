package data

import (
	"fmt"
	"log"
	"time"
)

// DataFrame 数据框架构体，用于存储市场数据
type DataFrame map[string][]interface{}

// DataPoint 数据点结构体
type DataPoint struct {
	Timestamp time.Time
	Open      float64
	High      float64
	Low       float64
	Close     float64
	Volume    int64
}

// MarketData 市场数据结构体
type MarketData struct {
	Symbol    string
	StartDate time.Time
	EndDate   time.Time
	Data      []DataPoint
}

// DataManager 数据管理器
type DataManager struct {
	// 可以添加数据库连接、API客户端等
	// db *sql.DB
	// apiClient *http.Client
}

// NewDataManager 创建新的数据管理器
func NewDataManager() *DataManager {
	return &DataManager{}
}

// GetMarketData 获取市场数据
func (dm *DataManager) GetMarketData(symbol, startDate, endDate string) (DataFrame, error) {
	log.Printf("获取市场数据: 符号=%s, 开始日期=%s, 结束日期=%s", symbol, startDate, endDate)

	// 解析日期
	start, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		return nil, fmt.Errorf("解析开始日期失败: %w", err)
	}

	end, err := time.Parse("2006-01-02", endDate)
	if err != nil {
		return nil, fmt.Errorf("解析结束日期失败: %w", err)
	}

	// 模拟数据生成（实际应用中应该从数据库或API获取）
	data := dm.generateMockData(symbol, start, end)

	// 转换为DataFrame格式
	dataFrame := dm.convertToDataFrame(data)

	log.Printf("成功获取 %d 条市场数据记录", len(data))
	return dataFrame, nil
}

// GetLatestPrice 获取最新价格
func (dm *DataManager) GetLatestPrice(symbol string) (float64, error) {
	log.Printf("获取最新价格: 符号=%s", symbol)

	// 模拟获取最新价格
	mockPrice := 150.25 + float64(time.Now().Unix()%100)/100.0

	log.Printf("最新价格: %.2f", mockPrice)
	return mockPrice, nil
}

// GetHistoricalData 获取历史数据（支持不同时间周期）
func (dm *DataManager) GetHistoricalData(symbol string, interval string, limit int) (*MarketData, error) {
	log.Printf("获取历史数据: 符号=%s, 周期=%s, 限制=%d", symbol, interval, limit)

	// 计算时间范围
	endTime := time.Now()
	var startTime time.Time

	switch interval {
	case "1m":
		startTime = endTime.Add(-time.Duration(limit) * time.Minute)
	case "5m":
		startTime = endTime.Add(-time.Duration(limit) * 5 * time.Minute)
	case "1h":
		startTime = endTime.Add(-time.Duration(limit) * time.Hour)
	case "1d":
		startTime = endTime.Add(-time.Duration(limit) * 24 * time.Hour)
	default:
		return nil, fmt.Errorf("不支持的时间周期: %s", interval)
	}

	// 生成模拟数据
	data := dm.generateMockData(symbol, startTime, endTime)

	return &MarketData{
		Symbol:    symbol,
		StartDate: startTime,
		EndDate:   endTime,
		Data:      data,
	}, nil
}

// generateMockData 生成模拟市场数据
func (dm *DataManager) generateMockData(symbol string, start, end time.Time) []DataPoint {
	var data []DataPoint
	current := start
	basePrice := 100.0

	for current.Before(end) {
		// 模拟价格波动
		priceChange := (float64(current.Unix()%100) - 50) / 100.0
		open := basePrice + priceChange
		high := open + float64(current.Unix()%10)/100.0
		low := open - float64(current.Unix()%10)/100.0
		close := open + (float64(current.Unix()%20)-10)/100.0
		volume := int64(1000000 + current.Unix()%500000)

		data = append(data, DataPoint{
			Timestamp: current,
			Open:      open,
			High:      high,
			Low:       low,
			Close:     close,
			Volume:    volume,
		})

		basePrice = close
		current = current.Add(time.Hour) // 每小时一个数据点
	}

	return data
}

// convertToDataFrame 将市场数据转换为DataFrame格式
func (dm *DataManager) convertToDataFrame(data []DataPoint) DataFrame {
	if len(data) == 0 {
		return DataFrame{}
	}

	df := DataFrame{
		"timestamp": make([]interface{}, len(data)),
		"open":      make([]interface{}, len(data)),
		"high":      make([]interface{}, len(data)),
		"low":       make([]interface{}, len(data)),
		"close":     make([]interface{}, len(data)),
		"volume":    make([]interface{}, len(data)),
	}

	for i, point := range data {
		df["timestamp"][i] = point.Timestamp
		df["open"][i] = point.Open
		df["high"][i] = point.High
		df["low"][i] = point.Low
		df["close"][i] = point.Close
		df["volume"][i] = point.Volume
	}

	return df
}

// ValidateData 验证数据完整性
func (dm *DataManager) ValidateData(df DataFrame) error {
	if len(df) == 0 {
		return fmt.Errorf("数据为空")
	}

	requiredColumns := []string{"timestamp", "open", "high", "low", "close", "volume"}
	for _, col := range requiredColumns {
		if _, exists := df[col]; !exists {
			return fmt.Errorf("缺少必需的列: %s", col)
		}
	}

	// 检查数据长度一致性
	dataLength := len(df["close"])
	for _, col := range requiredColumns {
		if len(df[col]) != dataLength {
			return fmt.Errorf("列 '%s' 的数据长度不一致", col)
		}
	}

	return nil
}

// GetDataStats 获取数据统计信息
func (dm *DataManager) GetDataStats(df DataFrame) map[string]interface{} {
	closeData := df["close"]
	if len(closeData) == 0 {
		return map[string]interface{}{}
	}

	var min, max, sum float64
	min = closeData[0].(float64)
	max = closeData[0].(float64)

	for _, val := range closeData {
		price := val.(float64)
		if price < min {
			min = price
		}
		if price > max {
			max = price
		}
		sum += price
	}

	avg := sum / float64(len(closeData))

	return map[string]interface{}{
		"count": len(closeData),
		"min":   min,
		"max":   max,
		"avg":   avg,
		"range": max - min,
	}
}
