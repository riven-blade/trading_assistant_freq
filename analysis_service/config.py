"""
配置管理模块
管理所有环境变量和配置项
"""
import os
from typing import List
import logging

logger = logging.getLogger(__name__)


class Config:
    """应用配置类"""
    
    # 数据库配置
    MYSQL_HOST = os.getenv("MYSQL_HOST", "localhost")
    MYSQL_PORT = int(os.getenv("MYSQL_PORT", "3306"))
    MYSQL_USER = os.getenv("MYSQL_USER", "root")
    MYSQL_PASSWORD = os.getenv("MYSQL_PASSWORD", "root")
    MYSQL_DB = os.getenv("MYSQL_DB", "trading_analysis")
    
    # 分析配置
    ANALYSIS_INTERVAL_HOURS = int(os.getenv("ANALYSIS_INTERVAL_HOURS", "4"))  # 默认4小时分析一次
    ANALYSIS_EXCHANGES = os.getenv("ANALYSIS_EXCHANGES", "binance,bybit").split(",")
    ANALYSIS_MARKET_TYPES = os.getenv("ANALYSIS_MARKET_TYPES", "future").split(",")  # 可选: spot, future 或 spot,future
    ANALYSIS_TIMEFRAMES = os.getenv("ANALYSIS_TIMEFRAMES", "1h").split(",")
    ANALYSIS_KLINE_LIMIT = int(os.getenv("ANALYSIS_KLINE_LIMIT", "2000"))
    ANALYSIS_RUN_ON_STARTUP = os.getenv("ANALYSIS_RUN_ON_STARTUP", "true").lower() == "true"
    
    # 性能配置
    ANALYSIS_CONCURRENCY = int(os.getenv("ANALYSIS_CONCURRENCY", "3"))  # 并发数
    ANALYSIS_BATCH_SIZE = int(os.getenv("ANALYSIS_BATCH_SIZE", "20"))  # 批次大小
    
    # 日志配置
    LOG_LEVEL = os.getenv("LOG_LEVEL", "INFO")
    
    @property
    def database_url(self) -> str:
        """生成数据库连接 URL"""
        return f"mysql+aiomysql://{self.MYSQL_USER}:{self.MYSQL_PASSWORD}@{self.MYSQL_HOST}:{self.MYSQL_PORT}/{self.MYSQL_DB}"
    
    def validate(self) -> bool:
        """验证配置有效性"""
        try:
            # 验证必需配置
            assert self.MYSQL_HOST, "MYSQL_HOST 不能为空"
            assert self.MYSQL_USER, "MYSQL_USER 不能为空"
            assert self.MYSQL_PASSWORD, "MYSQL_PASSWORD 不能为空"
            assert self.MYSQL_DB, "MYSQL_DB 不能为空"
            
            # 验证数值范围
            assert 1 <= self.MYSQL_PORT <= 65535, "MYSQL_PORT 必须在 1-65535 之间"
            assert self.ANALYSIS_INTERVAL_HOURS > 0, "ANALYSIS_INTERVAL_HOURS 必须大于 0"
            assert self.ANALYSIS_KLINE_LIMIT > 0, "ANALYSIS_KLINE_LIMIT 必须大于 0"
            
            # 验证交易所列表
            valid_exchanges = ["binance", "bybit", "okx", "mexc"]
            for exchange in self.ANALYSIS_EXCHANGES:
                assert exchange.strip() in valid_exchanges, f"不支持的交易所: {exchange}"
            
            # 验证市场类型
            valid_market_types = ["spot", "future"]
            for market_type in self.ANALYSIS_MARKET_TYPES:
                assert market_type.strip() in valid_market_types, f"不支持的市场类型: {market_type}"
            
            # 验证时间周期
            valid_timeframes = ["1m", "5m", "15m", "30m", "1h", "4h", "1d", "1w"]
            for timeframe in self.ANALYSIS_TIMEFRAMES:
                assert timeframe.strip() in valid_timeframes, f"不支持的时间周期: {timeframe}"
            
            logger.info("配置验证通过")
            return True
            
        except AssertionError as e:
            logger.error(f"配置验证失败: {e}")
            return False
    
    def log_config(self):
        """记录配置信息（隐藏敏感信息）"""
        logger.info("========== 配置信息 ==========")
        logger.info(f"数据库地址: {self.MYSQL_HOST}:{self.MYSQL_PORT}")
        logger.info(f"数据库名称: {self.MYSQL_DB}")
        logger.info(f"数据库用户: {self.MYSQL_USER}")
        logger.info(f"分析间隔: {self.ANALYSIS_INTERVAL_HOURS} 小时")
        logger.info(f"交易所列表: {', '.join(self.ANALYSIS_EXCHANGES)}")
        logger.info(f"市场类型: {', '.join(self.ANALYSIS_MARKET_TYPES)}")
        logger.info(f"时间周期: {', '.join(self.ANALYSIS_TIMEFRAMES)}")
        logger.info(f"K线数量: {self.ANALYSIS_KLINE_LIMIT}")
        logger.info(f"启动时执行: {self.ANALYSIS_RUN_ON_STARTUP}")
        logger.info(f"并发数: {self.ANALYSIS_CONCURRENCY}")
        logger.info(f"批次大小: {self.ANALYSIS_BATCH_SIZE}")
        logger.info(f"日志级别: {self.LOG_LEVEL}")
        logger.info("==============================")


# 全局配置实例
config = Config()
