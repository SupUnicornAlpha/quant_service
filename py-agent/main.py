"""
Agent Quant System - Python Agent Service
混合架构量化交易系统的Python Agent服务
"""

import os
import logging
import time
from typing import List, Optional
from datetime import datetime

import uvicorn
from fastapi import FastAPI, HTTPException, Depends, status
from fastapi.middleware.cors import CORSMiddleware
from fastapi.responses import JSONResponse
from pydantic import BaseModel, Field
from dotenv import load_dotenv

# 加载环境变量
load_dotenv()

# 配置日志
logging.basicConfig(
    level=logging.INFO,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s',
    handlers=[
        logging.FileHandler('logs/agent_service.log', encoding='utf-8'),
        logging.StreamHandler()
    ]
)
logger = logging.getLogger(__name__)

# 创建FastAPI应用
app = FastAPI(
    title="Agent Quant System",
    description="混合架构量化交易系统的Python Agent服务",
    version="1.0.0",
    docs_url="/docs",
    redoc_url="/redoc"
)

# 添加CORS中间件
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],  # 生产环境中应该限制具体域名
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

# Pydantic模型定义
class NewsAnalysisRequest(BaseModel):
    """新闻分析请求模型"""
    symbol: str = Field(..., description="股票代码", example="AAPL")
    news_items: List[str] = Field(..., description="新闻列表", example=["苹果发布新iPhone", "科技股上涨"])

class NewsAnalysisResponse(BaseModel):
    """新闻分析响应模型"""
    symbol: str = Field(..., description="股票代码")
    sentiment: str = Field(..., description="情绪分析结果", example="Positive")
    reason: str = Field(..., description="分析原因", example="新产品发布显示强劲创新力")
    confidence_score: float = Field(..., description="置信度分数", example=0.85, ge=0.0, le=1.0)

class HealthResponse(BaseModel):
    """健康检查响应模型"""
    status: str = Field(..., description="服务状态", example="healthy")
    timestamp: datetime = Field(..., description="检查时间")
    version: str = Field(..., description="服务版本", example="1.0.0")
    uptime: float = Field(..., description="运行时间（秒）")

# 全局变量
start_time = time.time()
openai_client = None

# 初始化OpenAI客户端
def init_openai_client():
    """初始化OpenAI客户端"""
    global openai_client
    try:
        from openai import OpenAI
        api_key = os.getenv("OPENAI_API_KEY")
        if not api_key or api_key == "your_openai_api_key_here":
            logger.warning("OpenAI API密钥未设置，将使用模拟响应")
            openai_client = None
        else:
            openai_client = OpenAI(api_key=api_key)
            logger.info("OpenAI客户端初始化成功")
    except Exception as e:
        logger.error(f"OpenAI客户端初始化失败: {e}")
        openai_client = None

# 模拟LLM分析函数
def mock_analyze_news(symbol: str, news_items: List[str]) -> NewsAnalysisResponse:
    """模拟新闻分析"""
    logger.info(f"模拟分析新闻: 标的={symbol}, 新闻数量={len(news_items)}")
    
    # 简单的关键词分析逻辑
    positive_keywords = ["涨", "上涨", "利好", "买入", "推荐", "增长", "收益", "创新", "突破", "强劲"]
    negative_keywords = ["跌", "下跌", "利空", "卖出", "警告", "亏损", "风险", "危机", "衰退", "疲软"]
    
    sentiment_score = 0
    positive_count = 0
    negative_count = 0
    
    for item in news_items:
        item_lower = item.lower()
        for keyword in positive_keywords:
            if keyword in item_lower:
                positive_count += 1
                sentiment_score += 1
        for keyword in negative_keywords:
            if keyword in item_lower:
                negative_count += 1
                sentiment_score -= 1
    
    # 确定情绪
    if sentiment_score > 0:
        sentiment = "Positive"
        confidence = min(0.9, 0.6 + positive_count * 0.1)
        reason = f"基于{positive_count}个正面信号的分析结果"
    elif sentiment_score < 0:
        sentiment = "Negative"
        confidence = min(0.9, 0.6 + negative_count * 0.1)
        reason = f"基于{negative_count}个负面信号的分析结果"
    else:
        sentiment = "Neutral"
        confidence = 0.5
        reason = "未发现明确的正面或负面信号"
    
    return NewsAnalysisResponse(
        symbol=symbol,
        sentiment=sentiment,
        reason=reason,
        confidence_score=confidence
    )

# 真实LLM分析函数
async def real_analyze_news(symbol: str, news_items: List[str]) -> NewsAnalysisResponse:
    """使用真实LLM分析新闻"""
    global openai_client
    
    if not openai_client:
        return mock_analyze_news(symbol, news_items)
    
    try:
        # 构建提示词
        news_text = "\n".join([f"- {item}" for item in news_items])
        prompt = f"""
请分析以下关于股票 {symbol} 的新闻，并给出情绪分析结果。

新闻内容：
{news_text}

请从以下角度进行分析：
1. 对股票价格的潜在影响
2. 市场情绪倾向
3. 投资建议倾向

请以JSON格式返回结果，包含以下字段：
- sentiment: "Positive", "Negative", 或 "Neutral"
- reason: 详细的分析原因
- confidence_score: 置信度分数（0.0-1.0）

分析要客观、专业，基于新闻内容的事实进行判断。
"""

        # 调用OpenAI API
        response = openai_client.chat.completions.create(
            model=os.getenv("DEFAULT_MODEL", "gpt-3.5-turbo"),
            messages=[
                {"role": "system", "content": "你是一个专业的金融分析师，擅长分析新闻对股票价格的影响。"},
                {"role": "user", "content": prompt}
            ],
            max_tokens=int(os.getenv("MAX_TOKENS", 1000)),
            temperature=float(os.getenv("TEMPERATURE", 0.7))
        )
        
        # 解析响应
        content = response.choices[0].message.content.strip()
        
        # 尝试解析JSON响应
        import json
        try:
            result = json.loads(content)
            return NewsAnalysisResponse(
                symbol=symbol,
                sentiment=result.get("sentiment", "Neutral"),
                reason=result.get("reason", "基于LLM分析的结果"),
                confidence_score=float(result.get("confidence_score", 0.7))
            )
        except json.JSONDecodeError:
            # 如果不是JSON格式，使用文本分析
            if "positive" in content.lower():
                sentiment = "Positive"
            elif "negative" in content.lower():
                sentiment = "Negative"
            else:
                sentiment = "Neutral"
            
            return NewsAnalysisResponse(
                symbol=symbol,
                sentiment=sentiment,
                reason=content[:200] + "..." if len(content) > 200 else content,
                confidence_score=0.8
            )
            
    except Exception as e:
        logger.error(f"LLM分析失败: {e}")
        return mock_analyze_news(symbol, news_items)

# API端点定义
@app.get("/", response_model=dict)
async def root():
    """根端点"""
    return {
        "message": "Agent Quant System - Python Agent Service",
        "version": "1.0.0",
        "status": "running",
        "timestamp": datetime.now()
    }

@app.post("/analyze", response_model=NewsAnalysisResponse)
async def analyze_news(request: NewsAnalysisRequest):
    """
    分析新闻情绪
    
    接收股票代码和新闻列表，返回情绪分析结果
    """
    try:
        logger.info(f"收到分析请求: 标的={request.symbol}, 新闻数量={len(request.news_items)}")
        
        # 验证输入
        if not request.symbol:
            raise HTTPException(status_code=400, detail="股票代码不能为空")
        
        if not request.news_items:
            raise HTTPException(status_code=400, detail="新闻列表不能为空")
        
        # 执行分析
        result = await real_analyze_news(request.symbol, request.news_items)
        
        logger.info(f"分析完成: 标的={result.symbol}, 情绪={result.sentiment}, 置信度={result.confidence_score}")
        
        return result
        
    except HTTPException:
        raise
    except Exception as e:
        logger.error(f"分析请求处理失败: {e}")
        raise HTTPException(status_code=500, detail=f"内部服务器错误: {str(e)}")

@app.get("/health", response_model=HealthResponse)
async def health_check():
    """健康检查端点"""
    uptime = time.time() - start_time
    
    return HealthResponse(
        status="healthy",
        timestamp=datetime.now(),
        version="1.0.0",
        uptime=uptime
    )

@app.get("/status")
async def get_status():
    """获取服务状态"""
    return {
        "service": "Agent Quant System",
        "version": "1.0.0",
        "status": "running",
        "uptime": time.time() - start_time,
        "openai_available": openai_client is not None,
        "timestamp": datetime.now()
    }

@app.get("/models")
async def get_available_models():
    """获取可用的模型列表"""
    if openai_client:
        try:
            models = openai_client.models.list()
            return {
                "models": [model.id for model in models.data],
                "default_model": os.getenv("DEFAULT_MODEL", "gpt-3.5-turbo")
            }
        except Exception as e:
            logger.error(f"获取模型列表失败: {e}")
            return {
                "models": ["gpt-3.5-turbo", "gpt-4"],
                "default_model": os.getenv("DEFAULT_MODEL", "gpt-3.5-turbo"),
                "error": str(e)
            }
    else:
        return {
            "models": ["mock-model"],
            "default_model": "mock-model",
            "note": "OpenAI客户端未初始化，使用模拟模式"
        }

# 启动事件
@app.on_event("startup")
async def startup_event():
    """应用启动事件"""
    logger.info("Agent Quant System Python服务启动中...")
    
    # 创建日志目录
    os.makedirs("logs", exist_ok=True)
    
    # 初始化OpenAI客户端
    init_openai_client()
    
    logger.info("Agent Quant System Python服务启动完成")

# 关闭事件
@app.on_event("shutdown")
async def shutdown_event():
    """应用关闭事件"""
    logger.info("Agent Quant System Python服务正在关闭...")

# 异常处理器
@app.exception_handler(Exception)
async def global_exception_handler(request, exc):
    """全局异常处理器"""
    logger.error(f"未处理的异常: {exc}")
    return JSONResponse(
        status_code=500,
        content={
            "detail": "内部服务器错误",
            "error": str(exc),
            "timestamp": datetime.now()
        }
    )

# 主函数
def main():
    """主函数"""
    host = os.getenv("HOST", "0.0.0.0")
    port = int(os.getenv("PORT", 8000))
    debug = os.getenv("DEBUG", "false").lower() == "true"
    
    logger.info(f"启动服务: {host}:{port}, 调试模式: {debug}")
    
    uvicorn.run(
        "main:app",
        host=host,
        port=port,
        reload=debug,
        log_level="info"
    )

if __name__ == "__main__":
    main()
