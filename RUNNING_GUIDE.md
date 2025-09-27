# Agent Quant System 运行指南

## 如何运行系统

### 第一步：启动 Python Agent 服务

```bash
# 进入 Python Agent 目录
cd py-agent

# 安装 Python 依赖
pip install -r requirements.txt

# 创建环境变量文件
cp .env.example .env

# 编辑 .env 文件，填入你的 OpenAI API Key
# 如果不填写，系统将使用模拟模式
nano .env

# 启动 Python Agent 服务
uvicorn main:app --reload
```

服务将在 `http://localhost:8000` 上运行。

你可以通过以下方式验证服务是否正常：
- 访问 `http://localhost:8000/docs` 查看 API 文档
- 访问 `http://localhost:8000/health` 检查健康状态

### 第二步：运行 Go 主程序

#### 1. 下载 Go 依赖

```bash
# 在项目根目录执行
go mod tidy
```

#### 2. 运行实时交易循环

```bash
# 基本运行（使用默认参数）
go run ./cmd/main.go run

# 指定交易标的和循环间隔
go run ./cmd/main.go run --symbol=AAPL --interval=5m

# 使用其他标的
go run ./cmd/main.go run --symbol=TSLA --interval=10m
```

#### 3. 运行回测

```bash
# 使用默认日期范围
go run ./cmd/main.go backtest --symbol=AAPL

# 指定日期范围
go run ./cmd/main.go backtest --symbol=TSLA --start=2023-01-01 --end=2023-12-31
```

#### 4. 查看系统状态

```bash
# 查看运行状态
go run ./cmd/main.go status

# 健康检查
go run ./cmd/main.go health
```

#### 5. 执行单次循环（测试用）

```bash
# 执行一次完整的交易决策流程
go run ./cmd/main.go single --symbol=AAPL
```

## 命令参数说明

### run 命令
- `--symbol, -s`: 交易标的（默认：AAPL）
- `--interval, -i`: 交易循环间隔（默认：5m）
- `--config, -c`: 配置文件路径（默认：config.toml）

### backtest 命令
- `--symbol, -s`: 回测标的（默认：AAPL）
- `--start`: 开始日期，格式：YYYY-MM-DD
- `--end`: 结束日期，格式：YYYY-MM-DD
- `--config, -c`: 配置文件路径

### 其他命令
- `status`: 查看系统状态
- `health`: 健康检查
- `single`: 执行单次循环

## 配置说明

### 1. 修改配置文件

编辑 `config.toml` 文件：

```toml
[agent_service]
url = "http://localhost:8000"  # Python Agent 服务地址

[accounts]
[accounts.my_stock_broker]
api_key = "YOUR_STOCK_API_KEY"
api_secret = "YOUR_STOCK_API_SECRET"
broker_type = "stock"
```

### 2. 环境变量配置

可以通过环境变量覆盖配置：

```bash
export OPENAI_API_KEY="your_openai_api_key"
export QUANT_AGENT_SERVICE_URL="http://localhost:8000"
```

## 运行示例

### 示例 1：完整的实时交易流程

```bash
# 终端 1：启动 Python Agent 服务
cd py-agent
uvicorn main:app --reload

# 终端 2：运行 Go 主程序
go run ./cmd/main.go run --symbol=AAPL --interval=5m
```

系统将：
1. 每5分钟获取 AAPL 的市场数据
2. 调用 Python Agent 分析相关新闻
3. 根据移动平均线交叉策略生成交易信号
4. 执行交易操作
5. 记录交易日志

### 示例 2：策略回测

```bash
# 运行一年的回测
go run ./cmd/main.go backtest --symbol=AAPL --start=2023-01-01 --end=2023-12-31
```

回测输出示例：
```
=== 回测结果 ===
策略名称: 移动平均线交叉策略
标的符号: AAPL
初始资金: 100000.00
最终资金: 105000.00
总收益率: 5.00%
年化收益率: 5.00%
最大回撤: 2.00%
夏普比率: 1.20
总交易次数: 10
胜率: 70.00%
平均盈利: 500.00
平均亏损: -300.00
盈亏比: 1.67
==================
```

### 示例 3：系统监控

```bash
# 查看系统状态
go run ./cmd/main.go status
```

输出示例：
```
=== 系统状态 ===
运行状态: true
启动时间: 2024-01-01 10:00:00
最后更新: 2024-01-01 10:05:00
总循环数: 5
成功循环: 4
失败循环: 1
总信号数: 8
已执行交易: 3
总盈亏: 150.00

=== 账户状态 ===
账户: my_stock_broker
  类型: stock
  状态: true
  余额: 100150.00
  可用余额: 99500.00
  持仓数量: 1
```

## 故障排除

### 1. Python Agent 服务启动失败

**问题**: `uvicorn: command not found`

**解决方案**:
```bash
pip install uvicorn
# 或者
pip install -r requirements.txt
```

**问题**: 端口被占用

**解决方案**:
```bash
# 查看端口占用
lsof -i :8000

# 修改端口
uvicorn main:app --reload --port 8001
```

### 2. Go 程序连接失败

**问题**: `Agent服务连接失败`

**解决方案**:
1. 检查 Python Agent 服务是否正常运行
2. 验证 `config.toml` 中的 URL 配置
3. 检查网络连接

### 3. 策略执行失败

**问题**: `数据验证失败`

**解决方案**:
1. 检查数据源是否正常
2. 验证日期格式是否正确
3. 确保有足够的历史数据

### 4. 交易执行失败

**问题**: `账户验证失败`

**解决方案**:
1. 检查账户凭证配置
2. 验证经纪商连接
3. 确认账户余额充足

## 日志查看

### Go 系统日志
```bash
# 查看实时日志
tail -f logs/quant_system.log

# 查看错误日志
grep "ERROR" logs/quant_system.log
```

### Python Agent 日志
```bash
# 查看实时日志
tail -f py-agent/logs/agent_service.log

# 查看 API 请求日志
grep "POST /analyze" py-agent/logs/agent_service.log
```

## 性能优化

### 1. 调整循环间隔
```bash
# 降低频率，减少资源消耗
go run ./cmd/main.go run --interval=30m

# 提高频率，增加交易机会
go run ./cmd/main.go run --interval=1m
```

### 2. 并发配置
修改 `config.toml` 中的并发设置：
```toml
[performance]
max_concurrent_requests = 10
request_timeout = 30s
```

### 3. 内存优化
```bash
# 设置 Go 程序内存限制
export GOGC=100
go run ./cmd/main.go run
```

## 安全建议

1. **保护 API 密钥**
   - 不要在代码中硬编码密钥
   - 使用环境变量或配置文件
   - 定期轮换密钥

2. **网络安全**
   - 使用 HTTPS 通信
   - 限制网络访问权限
   - 配置防火墙规则

3. **数据安全**
   - 定期备份配置和数据
   - 加密敏感信息
   - 监控异常访问

## 生产部署

### 1. 使用 Docker

创建 `Dockerfile`:
```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go mod tidy && go build -o main ./cmd/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/main .
COPY --from=builder /app/config.toml .
CMD ["./main", "run"]
```

### 2. 使用 Systemd

创建服务文件 `/etc/systemd/system/quant-system.service`:
```ini
[Unit]
Description=Agent Quant System
After=network.target

[Service]
Type=simple
User=quant
WorkingDirectory=/opt/quant-system
ExecStart=/opt/quant-system/main run
Restart=always
RestartSec=10

[Install]
WantedBy=multi-user.target
```

### 3. 监控和告警

- 设置系统资源监控
- 配置交易异常告警
- 建立日志分析系统
- 定期健康检查

---

**重要提醒**: 
- 本系统仅用于学习和研究目的
- 实际交易请谨慎使用并遵守相关法律法规
- 建议在模拟环境中充分测试后再考虑实盘交易
