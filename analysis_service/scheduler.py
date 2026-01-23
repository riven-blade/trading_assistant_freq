"""
定时任务调度器
管理定时分析任务的执行
"""
from apscheduler.schedulers.asyncio import AsyncIOScheduler
from apscheduler.triggers.interval import IntervalTrigger
import logging

from config import config
from coordinator import coordinator

logger = logging.getLogger(__name__)

# 创建调度器实例
scheduler = AsyncIOScheduler()


async def run_analysis():
    """执行分析任务的入口函数"""
    try:
        await coordinator.run_market_analysis()
    except Exception as e:
        logger.error(f"分析任务执行失败: {e}", exc_info=True)


def start_scheduler():
    """启动定时调度器"""
    try:
        logger.info("=" * 60)
        logger.info("启动定时任务调度器")
        logger.info("=" * 60)
        
        # 添加定时任务
        scheduler.add_job(
            run_analysis,
            trigger=IntervalTrigger(hours=config.ANALYSIS_INTERVAL_HOURS),
            id='market_analysis',
            name='Market Analysis Job',
            replace_existing=True
        )
        
        logger.info(f"定时任务已配置: 每 {config.ANALYSIS_INTERVAL_HOURS} 小时执行一次")
        
        # 启动调度器
        scheduler.start()
        logger.info("定时调度器已启动")
        logger.info("=" * 60)
        
    except Exception as e:
        logger.error(f"启动调度器失败: {e}", exc_info=True)
        raise


def stop_scheduler():
    """停止定时调度器"""
    try:
        logger.info("停止定时调度器...")
        scheduler.shutdown()
        logger.info("定时调度器已停止")
    except Exception as e:
        logger.error(f"停止调度器失败: {e}", exc_info=True)
