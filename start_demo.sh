#!/bin/bash

echo "=========================================="
echo "Agent Quant System 演示启动脚本"
echo "=========================================="

# 检查Go程序是否编译
if [ ! -f "./quant-system" ]; then
    echo "正在编译Go程序..."
    go build -o quant-system ./cmd/main.go
    if [ $? -ne 0 ]; then
        echo "编译失败！"
        exit 1
    fi
    echo "编译完成！"
fi

echo ""
echo "1. 系统健康检查..."
./quant-system health

echo ""
echo "2. 查看系统状态..."
./quant-system status

echo ""
echo "3. 执行单次交易循环测试..."
./quant-system single --symbol=AAPL

echo ""
echo "=========================================="
echo "演示完成！"
echo ""
echo "要启动Python Agent服务，请运行："
echo "cd py-agent && pip install -r requirements.txt && uvicorn main:app --reload"
echo ""
echo "要运行实时交易，请运行："
echo "./quant-system run --symbol=AAPL --interval=5m"
echo ""
echo "要运行回测，请运行："
echo "./quant-system backtest --symbol=AAPL --start=2023-01-01 --end=2023-12-31"
echo "=========================================="
