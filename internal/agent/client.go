package agent

import (
	"fmt"
	"log"
	"time"

	"github.com/go-resty/resty/v2"
)

// Client Agent客户端
type Client struct {
	httpClient *resty.Client
	baseURL    string
	timeout    time.Duration
}

// NewClient 创建Agent客户端
func NewClient(baseURL string) *Client {
	client := resty.New()
	client.SetTimeout(30 * time.Second)
	client.SetHeader("Content-Type", "application/json")
	client.SetHeader("Accept", "application/json")

	return &Client{
		httpClient: client,
		baseURL:    baseURL,
		timeout:    30 * time.Second,
	}
}

// AnalysisRequest 分析请求
type AnalysisRequest struct {
	Symbol    string    `json:"symbol"`
	NewsItems []string  `json:"news_items"`
	Timestamp time.Time `json:"timestamp"`
}

// AnalysisResponse 分析响应
type AnalysisResponse struct {
	Symbol          string    `json:"symbol"`
	Sentiment       string    `json:"sentiment"`
	Reason          string    `json:"reason"`
	ConfidenceScore float64   `json:"confidence_score"`
	Timestamp       time.Time `json:"timestamp"`
	AnalysisID      string    `json:"analysis_id"`
}

// NewsAnalysisRequest 新闻分析请求（与Python端匹配）
type NewsAnalysisRequest struct {
	Symbol    string   `json:"symbol"`
	NewsItems []string `json:"news_items"`
}

// NewsAnalysisResponse 新闻分析响应（与Python端匹配）
type NewsAnalysisResponse struct {
	Symbol          string  `json:"symbol"`
	Sentiment       string  `json:"sentiment"`
	Reason          string  `json:"reason"`
	ConfidenceScore float64 `json:"confidence_score"`
}

// AnalyzeNews 分析新闻
func (c *Client) AnalyzeNews(symbol string, newsItems []string) (*AnalysisResponse, error) {
	log.Printf("开始分析新闻: 标的=%s, 新闻数量=%d", symbol, len(newsItems))

	// 构建请求
	request := NewsAnalysisRequest{
		Symbol:    symbol,
		NewsItems: newsItems,
	}

	// 发送请求
	resp, err := c.httpClient.R().
		SetBody(request).
		SetResult(&NewsAnalysisResponse{}).
		Post(c.baseURL + "/analyze")

	if err != nil {
		return nil, fmt.Errorf("发送请求失败: %w", err)
	}

	if resp.StatusCode() != 200 {
		return nil, fmt.Errorf("请求失败，状态码: %d, 响应: %s", resp.StatusCode(), resp.String())
	}

	// 转换响应
	response, ok := resp.Result().(*NewsAnalysisResponse)
	if !ok {
		return nil, fmt.Errorf("响应解析失败")
	}

	// 转换为内部格式
	analysisResponse := &AnalysisResponse{
		Symbol:          response.Symbol,
		Sentiment:       response.Sentiment,
		Reason:          response.Reason,
		ConfidenceScore: response.ConfidenceScore,
		Timestamp:       time.Now(),
		AnalysisID:      fmt.Sprintf("ANALYSIS_%d", time.Now().UnixNano()),
	}

	log.Printf("新闻分析完成: 标的=%s, 情绪=%s, 置信度=%.2f",
		symbol, response.Sentiment, response.ConfidenceScore)

	return analysisResponse, nil
}

// AnalyzeMarketSentiment 分析市场情绪
func (c *Client) AnalyzeMarketSentiment(symbol string, marketData map[string]interface{}) (*AnalysisResponse, error) {
	log.Printf("开始分析市场情绪: 标的=%s", symbol)

	// 构建市场数据摘要
	dataSummary := fmt.Sprintf("价格: %.2f, 成交量: %.0f",
		marketData["price"], marketData["volume"])

	// 模拟新闻项目（实际应用中应该从市场数据中提取关键信息）
	newsItems := []string{
		fmt.Sprintf("标的 %s 当前价格 %s", symbol, dataSummary),
		"市场波动性增加",
		"成交量异常活跃",
	}

	return c.AnalyzeNews(symbol, newsItems)
}

// AnalyzeTechnicalIndicators 分析技术指标
func (c *Client) AnalyzeTechnicalIndicators(symbol string, indicators map[string]float64) (*AnalysisResponse, error) {
	log.Printf("开始分析技术指标: 标的=%s", symbol)

	// 构建指标摘要
	summary := "技术指标分析: "
	for name, value := range indicators {
		summary += fmt.Sprintf("%s=%.2f, ", name, value)
	}

	newsItems := []string{
		fmt.Sprintf("标的 %s 技术分析: %s", symbol, summary),
		"基于技术指标的交易信号",
	}

	return c.AnalyzeNews(symbol, newsItems)
}

// BatchAnalyze 批量分析
func (c *Client) BatchAnalyze(symbols []string, newsItems []string) (map[string]*AnalysisResponse, error) {
	log.Printf("开始批量分析: 标的数量=%d, 新闻数量=%d", len(symbols), len(newsItems))

	results := make(map[string]*AnalysisResponse)

	for _, symbol := range symbols {
		response, err := c.AnalyzeNews(symbol, newsItems)
		if err != nil {
			log.Printf("分析标的 %s 失败: %v", symbol, err)
			continue
		}
		results[symbol] = response
	}

	log.Printf("批量分析完成: 成功分析 %d/%d 个标的", len(results), len(symbols))
	return results, nil
}

// GetAnalysisHistory 获取分析历史
func (c *Client) GetAnalysisHistory(symbol string, limit int) ([]*AnalysisResponse, error) {
	log.Printf("获取分析历史: 标的=%s, 限制=%d", symbol, limit)

	// 模拟获取历史数据
	history := make([]*AnalysisResponse, 0)

	for i := 0; i < limit && i < 5; i++ {
		response := &AnalysisResponse{
			Symbol:          symbol,
			Sentiment:       []string{"Positive", "Negative", "Neutral"}[i%3],
			Reason:          fmt.Sprintf("历史分析 %d", i+1),
			ConfidenceScore: 0.7 + float64(i)*0.05,
			Timestamp:       time.Now().Add(-time.Duration(i) * time.Hour),
			AnalysisID:      fmt.Sprintf("HISTORY_%d", i),
		}
		history = append(history, response)
	}

	log.Printf("获取到 %d 条历史分析记录", len(history))
	return history, nil
}

// HealthCheck 健康检查
func (c *Client) HealthCheck() error {
	log.Printf("检查Agent服务健康状态")

	resp, err := c.httpClient.R().
		Get(c.baseURL + "/health")

	if err != nil {
		return fmt.Errorf("健康检查失败: %w", err)
	}

	if resp.StatusCode() != 200 {
		return fmt.Errorf("Agent服务不健康，状态码: %d", resp.StatusCode())
	}

	log.Printf("Agent服务健康状态正常")
	return nil
}

// SetTimeout 设置超时时间
func (c *Client) SetTimeout(timeout time.Duration) {
	c.timeout = timeout
	c.httpClient.SetTimeout(timeout)
}

// SetBaseURL 设置基础URL
func (c *Client) SetBaseURL(baseURL string) {
	c.baseURL = baseURL
}

// GetBaseURL 获取基础URL
func (c *Client) GetBaseURL() string {
	return c.baseURL
}

// MockClient 模拟客户端（用于测试）
type MockClient struct {
	baseURL string
}

// NewMockClient 创建模拟客户端
func NewMockClient(baseURL string) *MockClient {
	return &MockClient{
		baseURL: baseURL,
	}
}

// AnalyzeNews 模拟新闻分析
func (mc *MockClient) AnalyzeNews(symbol string, newsItems []string) (*AnalysisResponse, error) {
	log.Printf("模拟分析新闻: 标的=%s, 新闻数量=%d", symbol, len(newsItems))

	// 模拟分析逻辑
	sentiment := "Neutral"
	confidence := 0.5

	// 简单的关键词分析
	for _, item := range newsItems {
		if contains(item, []string{"涨", "上涨", "利好", "买入", "推荐"}) {
			sentiment = "Positive"
			confidence = 0.8
			break
		} else if contains(item, []string{"跌", "下跌", "利空", "卖出", "警告"}) {
			sentiment = "Negative"
			confidence = 0.8
			break
		}
	}

	response := &AnalysisResponse{
		Symbol:          symbol,
		Sentiment:       sentiment,
		Reason:          fmt.Sprintf("基于 %d 条新闻的分析结果", len(newsItems)),
		ConfidenceScore: confidence,
		Timestamp:       time.Now(),
		AnalysisID:      fmt.Sprintf("MOCK_%d", time.Now().UnixNano()),
	}

	log.Printf("模拟分析完成: 标的=%s, 情绪=%s, 置信度=%.2f",
		symbol, sentiment, confidence)

	return response, nil
}

// AnalyzeMarketSentiment 模拟市场情绪分析
func (mc *MockClient) AnalyzeMarketSentiment(symbol string, marketData map[string]interface{}) (*AnalysisResponse, error) {
	newsItems := []string{
		fmt.Sprintf("标的 %s 市场数据分析", symbol),
		"技术指标显示趋势变化",
	}

	return mc.AnalyzeNews(symbol, newsItems)
}

// AnalyzeTechnicalIndicators 模拟技术指标分析
func (mc *MockClient) AnalyzeTechnicalIndicators(symbol string, indicators map[string]float64) (*AnalysisResponse, error) {
	newsItems := []string{
		fmt.Sprintf("标的 %s 技术指标分析", symbol),
		"移动平均线交叉信号",
	}

	return mc.AnalyzeNews(symbol, newsItems)
}

// BatchAnalyze 模拟批量分析
func (mc *MockClient) BatchAnalyze(symbols []string, newsItems []string) (map[string]*AnalysisResponse, error) {
	results := make(map[string]*AnalysisResponse)

	for _, symbol := range symbols {
		response, err := mc.AnalyzeNews(symbol, newsItems)
		if err != nil {
			continue
		}
		results[symbol] = response
	}

	return results, nil
}

// GetAnalysisHistory 模拟获取分析历史
func (mc *MockClient) GetAnalysisHistory(symbol string, limit int) ([]*AnalysisResponse, error) {
	history := make([]*AnalysisResponse, 0)

	for i := 0; i < limit && i < 3; i++ {
		response := &AnalysisResponse{
			Symbol:          symbol,
			Sentiment:       []string{"Positive", "Negative", "Neutral"}[i%3],
			Reason:          fmt.Sprintf("模拟历史分析 %d", i+1),
			ConfidenceScore: 0.6 + float64(i)*0.1,
			Timestamp:       time.Now().Add(-time.Duration(i) * time.Hour),
			AnalysisID:      fmt.Sprintf("MOCK_HISTORY_%d", i),
		}
		history = append(history, response)
	}

	return history, nil
}

// HealthCheck 模拟健康检查
func (mc *MockClient) HealthCheck() error {
	log.Printf("模拟Agent服务健康检查通过")
	return nil
}

// SetTimeout 模拟设置超时
func (mc *MockClient) SetTimeout(timeout time.Duration) {
	// 模拟实现
}

// SetBaseURL 模拟设置基础URL
func (mc *MockClient) SetBaseURL(baseURL string) {
	mc.baseURL = baseURL
}

// GetBaseURL 获取基础URL
func (mc *MockClient) GetBaseURL() string {
	return mc.baseURL
}

// contains 检查字符串是否包含任何关键词
func contains(text string, keywords []string) bool {
	for _, keyword := range keywords {
		if len(text) >= len(keyword) {
			for i := 0; i <= len(text)-len(keyword); i++ {
				if text[i:i+len(keyword)] == keyword {
					return true
				}
			}
		}
	}
	return false
}

// ClientInterface Agent客户端接口
type ClientInterface interface {
	AnalyzeNews(symbol string, newsItems []string) (*AnalysisResponse, error)
	AnalyzeMarketSentiment(symbol string, marketData map[string]interface{}) (*AnalysisResponse, error)
	AnalyzeTechnicalIndicators(symbol string, indicators map[string]float64) (*AnalysisResponse, error)
	BatchAnalyze(symbols []string, newsItems []string) (map[string]*AnalysisResponse, error)
	GetAnalysisHistory(symbol string, limit int) ([]*AnalysisResponse, error)
	HealthCheck() error
	SetTimeout(timeout time.Duration)
	SetBaseURL(baseURL string)
	GetBaseURL() string
}

// CreateClient 创建客户端（工厂方法）
func CreateClient(baseURL string, useMock bool) ClientInterface {
	if useMock {
		return NewMockClient(baseURL)
	}
	return NewClient(baseURL)
}
