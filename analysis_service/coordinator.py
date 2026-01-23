"""
分析协调器
协调整个市场分析流程
"""
import asyncio
from datetime import datetime
from typing import List, Dict
import logging

from config import config
from symbol_fetcher import symbol_fetcher
from analyzer import analyzer
from db_manager import db_manager
from database import AsyncSessionLocal

logger = logging.getLogger(__name__)


class AnalysisStats:
    """分析统计信息"""
    
    def __init__(self):
        self.total = 0
        self.success = 0
        self.failed = 0
        self.start_time = None
        self.end_time = None
        self.errors = []
    
    def add_success(self):
        """记录成功"""
        self.total += 1
        self.success += 1
    
    def add_failure(self, error: str):
        """记录失败"""
        self.total += 1
        self.failed += 1
        self.errors.append(error)
    
    def log_summary(self):
        """输出统计摘要"""
        if self.start_time and self.end_time:
            duration = (self.end_time - self.start_time).total_seconds()
        else:
            duration = 0
        
        success_rate = (self.success / self.total * 100) if self.total > 0 else 0
        
        logger.info("=" * 60)
        logger.info("分析统计摘要")
        logger.info("=" * 60)
        logger.info(f"总数: {self.total}")
        logger.info(f"成功: {self.success}")
        logger.info(f"失败: {self.failed}")
        logger.info(f"成功率: {success_rate:.2f}%")
        logger.info(f"耗时: {duration:.2f} 秒 ({duration/60:.2f} 分钟)")
        
        if self.errors:
            logger.info(f"错误数: {len(self.errors)}")
            # 只显示前 10 个错误
            for i, error in enumerate(self.errors[:10], 1):
                logger.info(f"  {i}. {error}")
            if len(self.errors) > 10:
                logger.info(f"  ... 还有 {len(self.errors) - 10} 个错误")
        
        logger.info("=" * 60)


class MarketAnalysisCoordinator:
    """市场分析协调器"""
    
    def __init__(self):
        # 从配置读取并发数和批次大小
        self.semaphore = asyncio.Semaphore(config.ANALYSIS_CONCURRENCY)
        self.batch_size = config.ANALYSIS_BATCH_SIZE
    
    async def analyze_symbol(
        self,
        exchange: str,
        symbol: str,
        market_type: str,
        timeframe: str,
        stats: AnalysisStats
    ):
        """
        分析单个交易对
        
        Args:
            exchange: 交易所名称
            symbol: 交易对符号
            market_type: 市场类型
            timeframe: 时间周期
            stats: 统计信息对象
        """
        async with self.semaphore:
            db_session = None
            try:
                logger.debug(f"开始分析: {exchange} {symbol} (原始格式) {market_type} {timeframe}")
                
                # 1. 获取 K 线数据
                df = await analyzer.fetch_klines(
                    exchange,
                    symbol,
                    market_type,
                    timeframe
                )
                
                if df.empty:
                    error_msg = f"{exchange} {symbol} {market_type} {timeframe}: 没有数据"
                    logger.warning(error_msg)
                    stats.add_failure(error_msg)
                    return
                
                # 2. 分析支撑位和压力位（传递timeframe参数）
                supports, resistances = analyzer.analyze_support_resistance(df, timeframe)
                
                # 3. 获取最新价格
                last_price = float(df['close'].iloc[-1])
                
                # 4. 释放 DataFrame 内存
                del df
                
                # 5. 写入数据库
                db_session = AsyncSessionLocal()
                success = await db_manager.upsert_analysis_result(
                    db_session,
                    exchange=exchange,
                    symbol=symbol,
                    market_type=market_type,
                    timeframe=timeframe,
                    support_levels=supports,
                    resistance_levels=resistances,
                    last_price=last_price,
                    input_limit=2000
                )
                
                if success:
                    stats.add_success()
                    logger.debug(
                        f"分析完成: {exchange} {symbol} {market_type} {timeframe} - "
                        f"支撑位: {len(supports)}, 压力位: {len(resistances)}"
                    )
                else:
                    error_msg = f"{exchange} {symbol} {market_type} {timeframe}: 数据库写入失败"
                    stats.add_failure(error_msg)
                
            except Exception as e:
                error_msg = f"{exchange} {symbol} {market_type} {timeframe}: {str(e)}"
                logger.error(f"分析失败: {error_msg}", exc_info=True)
                stats.add_failure(error_msg)
            finally:
                # 确保关闭数据库会话
                if db_session:
                    await db_session.close()
    
    async def run_market_analysis(self):
        """
        执行完整的市场分析流程
        """
        logger.info("=" * 60)
        logger.info("开始市场分析")
        logger.info("=" * 60)
        
        stats = AnalysisStats()
        stats.start_time = datetime.now()
        
        try:
            # 获取配置
            exchanges = config.ANALYSIS_EXCHANGES
            timeframes = config.ANALYSIS_TIMEFRAMES
            market_types = config.ANALYSIS_MARKET_TYPES  # 从配置读取市场类型
            
            logger.info(f"交易所: {', '.join(exchanges)}")
            logger.info(f"市场类型: {', '.join(market_types)}")
            logger.info(f"时间周期: {', '.join(timeframes)}")
            logger.info(f"K线数量: {config.ANALYSIS_KLINE_LIMIT}")
            
            # 遍历交易所
            for exchange in exchanges:
                logger.info(f"\n处理交易所: {exchange}")
                
                # 遍历市场类型
                for market_type in market_types:
                    logger.info(f"  处理市场类型: {market_type}")
                    
                    # 获取交易对列表
                    symbols = await symbol_fetcher.fetch_symbols(exchange, market_type)
                    
                    if not symbols:
                        logger.warning(f"    {exchange} {market_type}: 没有找到交易对")
                        continue
                    
                    logger.info(f"    找到 {len(symbols)} 个交易对")
                    
                    # 分批处理交易对，避免内存峰值
                    for batch_start in range(0, len(symbols), self.batch_size):
                        batch_end = min(batch_start + self.batch_size, len(symbols))
                        batch_symbols = symbols[batch_start:batch_end]
                        
                        logger.info(f"    处理批次 {batch_start//self.batch_size + 1}: {batch_start+1}-{batch_end}/{len(symbols)}")
                        
                        # 创建分析任务
                        tasks = []
                        for symbol in batch_symbols:
                            for timeframe in timeframes:
                                task = self.analyze_symbol(
                                    exchange,
                                    symbol,
                                    market_type,
                                    timeframe,
                                    stats
                                )
                                tasks.append(task)
                        
                        # 并发执行当前批次
                        await asyncio.gather(*tasks, return_exceptions=True)
                        
                        # 批次完成后，强制垃圾回收
                        import gc
                        gc.collect()
                        
                        logger.info(f"    批次 {batch_start//self.batch_size + 1} 完成")
                    
                    logger.info(f"    {exchange} {market_type} 分析完成")
            
        except Exception as e:
            logger.error(f"市场分析过程中发生错误: {e}", exc_info=True)
        finally:
            stats.end_time = datetime.now()
            stats.log_summary()
            logger.info("市场分析结束")
            logger.info("=" * 60)


# 全局实例
coordinator = MarketAnalysisCoordinator()
