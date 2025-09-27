# Agent Quant System

一个基于 Go 和 Python 的混合架构量化交易系统。Go 负责高性能的引擎和业务逻辑，Python 负责与大语言模型（LLM）交互的灵活性。

## 系统架构

```
┌─────────────────┐    HTTP API    ┌─────────────────┐
│   Go 主系统      │ ◄──────────► │  Python Agent   │
│                 │                │                 │
│ ┌─────────────┐ │                │ ┌─────────────┐ │
│ │ 量化引擎     │ │                │ │ LLM 分析    │ │
│ │ 策略管理     │ │                │ │ 新闻处理    │ │
│ │ 交易执行     │ │                │ │ 情绪分析    │ │
│ │ 数据管理     │ │                │ └─────────────┘ │
│ │ 账户管理     │ │                │                 │
│ │ 回测模块     │ │                │                 │
│ └─────────────┘ │                │                 │
└─────────────────┘                └─────────────────┘
```

## 主要特性

- **混合架构**: Go 高性能引擎 + Python LLM 智能分析
- **实时交易**: 支持实时市场数据分析和交易执行
- **智能分析**: 基于 LLM 的新闻情绪分析和投资建议
- **多策略支持**: 内置移动平均线交叉、RSI 等经典策略
- **回测功能**: 完整的历史数据回测和性能分析
- **风险控制**: 内置风险管理器和仓位控制
- **多账户支持**: 支持股票、加密货币等多种账户类型

## 项目结构

```
agent-quant-system/
├── cmd/                    # Go 程序入口
│   └── main.go
├── internal/               # Go 内部模块
│   ├── account/           # 账户管理
│   ├── agent/             # Agent 客户端
│   ├── backtest/          # 回测模块
│   ├── config/            # 配置管理
│   ├── core/              # 核心引擎
│   ├── data/              # 数据管理
│   ├── strategy/          # 策略管理
│   └── trading/           # 交易引擎
├── py-agent/              # Python Agent 服务
│   ├── main.py            # FastAPI 服务
│   ├── requirements.txt   # Python 依赖
│   └── .env.example       # 环境变量示例
├── config.toml            # 系统配置
├── go.mod                 # Go 模块文件
└── README.md              # 项目文档
```

## 快速开始

### 1. 环境要求

- Go 1.21+
- Python 3.8+
- OpenAI API Key (可选，用于真实 LLM 分析)

### 2. 启动 Python Agent 服务

```bash
cd py-agent
pip install -r requirements.txt

# 创建环境变量文件
cp .env.example .env
# 编辑 .env 文件，填入你的 OpenAI API Key

# 启动服务
uvicorn main:app --reload
```

服务将在 `http://localhost:8000` 上运行。

### 3. 启动 Go 主系统

```bash
# 下载 Go 依赖
go mod tidy

# 运行实时交易循环
go run ./cmd/main.go run --symbol=AAPL --interval=5m

# 运行回测
go run ./cmd/main.go backtest --symbol=TSLA --start=2023-01-01 --end=2023-12-31

# 查看系统状态
go run ./cmd/main.go status

# 健康检查
go run ./cmd/main.go health
```

## 配置说明

### config.toml 配置文件

```toml
[agent_service]
url = "http://localhost:8000"

[api_keys]
openai_key = "YOUR_OPENAI_API_KEY"

[accounts]
[accounts.my_stock_broker]
api_key = "STOCK_API_KEY"
api_secret = "STOCK_API_SECRET"
broker_type = "stock"

[accounts.my_crypto_exchange]
api_key = "CRYPTO_API_KEY"
api_secret = "CRYPTO_API_SECRET"
broker_type = "crypto"

[backtest]
initial_capital = 100000.0
commission_rate = 0.001
slippage_rate = 0.0005
```

### 环境变量

可以通过环境变量覆盖配置：

```bash
export OPENAI_API_KEY="your_openai_api_key"
export QUANT_AGENT_SERVICE_URL="http://localhost:8000"
```

## 使用示例

### 1. 实时交易

```bash
# 启动实时交易系统
go run ./cmd/main.go run --symbol=AAPL --interval=5m
```

系统将：
1. 每5分钟获取市场数据
2. 调用 Python Agent 分析新闻情绪
3. 根据策略生成交易信号
4. 执行交易操作

### 2. 策略回测

```bash
# 运行回测
go run ./cmd/main.go backtest --symbol=AAPL --start=2023-01-01 --end=2023-12-31
```

回测结果包括：
- 总收益率和年化收益率
- 最大回撤和夏普比率
- 胜率和盈亏比
- 交易统计和风险指标

### 3. 单次循环测试

```bash
# 执行单次交易循环（用于测试）
go run ./cmd/main.go single --symbol=AAPL
```

## API 接口

### Python Agent 服务

- `POST /analyze` - 新闻情绪分析
- `GET /health` - 健康检查
- `GET /status` - 服务状态
- `GET /models` - 可用模型列表

### 示例请求

```bash
curl -X POST "http://localhost:8000/analyze" \
     -H "Content-Type: application/json" \
     -d '{
       "symbol": "AAPL",
       "news_items": [
         "苹果发布新款iPhone，市场反应积极",
         "科技股整体上涨，投资者信心增强"
       ]
     }'
```

## 策略开发

### 自定义策略

实现 `Strategy` 接口：

```go
type MyStrategy struct {
    strategy.BaseStrategy
}

func (s *MyStrategy) GenerateSignals(df data.DataFrame, guidance *strategy.AgentGuidance) ([]strategy.TradingSignal, error) {
    // 实现你的策略逻辑
    // 可以结合 Agent 指导信息
    return signals, nil
}
```

### 注册策略

```go
strategyManager.RegisterStrategy("my_strategy", NewMyStrategy())
```

## 风险管理

系统内置了完整的风险管理功能：

- 最大仓位控制
- 止损止盈设置
- 日亏损限制
- 最大回撤控制

## 监控和日志

- 详细的交易日志
- 性能指标统计
- 健康状态监控
- 错误追踪和告警

## 部署建议

### 生产环境

1. 使用 Docker 容器化部署
2. 配置负载均衡和高可用
3. 设置监控和告警系统
4. 定期备份配置和数据

### 安全考虑

1. 保护 API 密钥和凭证
2. 使用 HTTPS 加密通信
3. 限制网络访问权限
4. 定期更新依赖包

## 故障排除

### 常见问题

1. **Agent 服务连接失败**
   - 检查 Python 服务是否正常运行
   - 验证配置文件中的 URL 设置

2. **策略执行失败**
   - 检查数据源是否正常
   - 验证策略参数配置

3. **交易执行失败**
   - 检查账户凭证是否正确
   - 验证经纪商连接状态

### 日志查看

```bash
# Go 系统日志
tail -f logs/quant_system.log

# Python Agent 日志
tail -f py-agent/logs/agent_service.log
```

## 贡献指南

1. Fork 项目
2. 创建功能分支
3. 提交更改
4. 创建 Pull Request

## 许可证

本项目采用 MIT 许可证 - 查看 [LICENSE](LICENSE) 文件了解详情。

## 联系方式

如有问题或建议，请创建 Issue 或联系开发团队。

---

**注意**: 本系统仅用于学习和研究目的，实际交易请谨慎使用并遵守相关法律法规。