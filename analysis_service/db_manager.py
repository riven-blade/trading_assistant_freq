"""
数据库管理器
管理分析结果的数据库操作
"""
from sqlalchemy.ext.asyncio import AsyncSession
from sqlalchemy.future import select
from sqlalchemy import func
from sqlalchemy.dialects.mysql import insert
from typing import List
import logging

from models import AnalysisResult

logger = logging.getLogger(__name__)


def normalize_symbol(symbol: str) -> str:
    """
    规范化交易对名称，统一存储为 BTCUSDT 格式（无斜杠）
    
    处理各种格式:
    - BTC/USDT -> BTCUSDT
    - BTC/USDT:USDT -> BTCUSDT
    - BTCUSDT -> BTCUSDT
    
    Args:
        symbol: 原始交易对名称
    
    Returns:
        规范化后的交易对名称 (BTCUSDT 格式)
    """
    # 移除质押物后缀 (:USDT)
    if ':' in symbol:
        symbol = symbol.split(':')[0]
    
    # 移除斜杠
    symbol = symbol.replace('/', '')
    
    return symbol


class DBManager:
    """数据库管理器"""
    
    async def upsert_analysis_result(
        self,
        db: AsyncSession,
        exchange: str,
        symbol: str,
        market_type: str,
        timeframe: str,
        support_levels: List[float],
        resistance_levels: List[float],
        last_price: float,
        input_limit: int
    ) -> bool:
        """
        插入或更新分析结果（使用 MySQL 的 ON DUPLICATE KEY UPDATE）
        
        Args:
            db: 数据库会话
            exchange: 交易所名称
            symbol: 交易对符号
            market_type: 市场类型
            timeframe: 时间周期
            support_levels: 支撑位列表
            resistance_levels: 压力位列表
            last_price: 最新价格
            input_limit: K线数量
        
        Returns:
            是否成功
        """
        try:
            # 规范化交易对名称: BTC/USDT:USDT -> BTCUSDT
            normalized_symbol = normalize_symbol(symbol)
            logger.debug(f"Symbol normalization: '{symbol}' -> '{normalized_symbol}'")
            
            # 使用 MySQL 的 INSERT ... ON DUPLICATE KEY UPDATE
            stmt = insert(AnalysisResult).values(
                exchange=exchange,
                symbol=normalized_symbol,  # 使用规范化后的名称
                market_type=market_type,
                timeframe=timeframe,
                support_levels=support_levels,
                resistance_levels=resistance_levels,
                last_price=last_price,
                input_limit=input_limit
            )
            
            # 如果唯一键冲突，则更新这些字段
            stmt = stmt.on_duplicate_key_update(
                support_levels=stmt.inserted.support_levels,
                resistance_levels=stmt.inserted.resistance_levels,
                last_price=stmt.inserted.last_price,
                input_limit=stmt.inserted.input_limit,
                updated_at=func.now()
            )
            
            await db.execute(stmt)
            await db.commit()
            
            logger.debug(f"Upsert 成功: {exchange} {normalized_symbol} {market_type} {timeframe}")
            return True
            
        except Exception as e:
            logger.error(f"数据库操作失败: {exchange} {symbol} - {e}", exc_info=True)
            await db.rollback()
            return False
    
    async def batch_upsert(
        self,
        db: AsyncSession,
        results: List[dict]
    ) -> tuple:
        """
        批量插入或更新分析结果
        
        Args:
            db: 数据库会话
            results: 分析结果列表
        
        Returns:
            (成功数, 失败数)
        """
        success_count = 0
        failed_count = 0
        
        for result in results:
            success = await self.upsert_analysis_result(db, **result)
            if success:
                success_count += 1
            else:
                failed_count += 1
        
        logger.info(f"批量操作完成: 成功 {success_count}, 失败 {failed_count}")
        return success_count, failed_count


# 全局实例
db_manager = DBManager()
