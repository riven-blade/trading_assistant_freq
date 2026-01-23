from fastapi import FastAPI
from sqlalchemy.ext.asyncio import AsyncSession
from sqlalchemy.future import select
from sqlalchemy import func
from typing import List, Optional
from contextlib import asynccontextmanager
import logging

from database import engine, get_db, Base
from models import AnalysisResult
from config import config
from scheduler import start_scheduler, stop_scheduler, run_analysis

# 配置日志
logging.basicConfig(
    level=getattr(logging, config.LOG_LEVEL),
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)
logger = logging.getLogger(__name__)


@asynccontextmanager
async def lifespan(app: FastAPI):
    """应用生命周期管理"""
    # Startup
    logger.info("=" * 60)
    logger.info("分析服务启动")
    logger.info("=" * 60)
    
    # 1. 验证配置
    if not config.validate():
        logger.error("配置验证失败，服务无法启动")
        raise ValueError("Invalid configuration")
    
    config.log_config()
    
    # 2. 初始化数据库
    async with engine.begin() as conn:
        await conn.run_sync(Base.metadata.create_all)
    logger.info("✓ 数据库表已创建")
    
    # 3. 启动定时调度器
    start_scheduler()
    logger.info("✓ 定时调度器已启动")
    
    # 4. 立即执行首次分析（如果配置启用）
    if config.ANALYSIS_RUN_ON_STARTUP:
        logger.info("→ 开始执行首次分析...")
        try:
            await run_analysis()
            logger.info("✓ 首次分析完成")
        except Exception as e:
            logger.error(f"✗ 首次分析失败: {e}", exc_info=True)
            logger.info("服务将继续运行，定时任务将按计划执行")
    else:
        logger.info("跳过首次分析，等待定时任务触发")
    
    logger.info("=" * 60)
    logger.info("服务启动完成")
    logger.info(f"后续将每 {config.ANALYSIS_INTERVAL_HOURS} 小时自动执行分析任务")
    logger.info("=" * 60)
    
    yield
    
    # Shutdown
    logger.info("=" * 60)
    logger.info("服务关闭中...")
    logger.info("=" * 60)
    
    stop_scheduler()
    logger.info("✓ 定时调度器已停止")
    
    await engine.dispose()
    logger.info("✓ 数据库连接已关闭")
    
    logger.info("服务已关闭")
    logger.info("=" * 60)


app = FastAPI(
    title="Trading Analysis Service",
    description="自动化市场分析服务 - 定时分析所有 USDT 交易对的支撑/压力位",
    version="2.0.0",
    lifespan=lifespan
)


@app.get("/health")
async def health_check():
    """健康检查端点"""
    return {
        "status": "ok",
        "service": "Trading Analysis Service",
        "version": "2.0.0",
        "description": "定时分析服务正在运行"
    }
