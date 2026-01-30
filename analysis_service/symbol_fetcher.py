"""
交易对获取器
从交易所 API 获取所有 USDT 交易对
"""
import ccxt.async_support as ccxt
from typing import List, Dict
import logging
import asyncio
from tenacity import retry, stop_after_attempt, wait_exponential, retry_if_exception_type

logger = logging.getLogger(__name__)


class SymbolFetcher:
    """交易对获取器"""
    
    def __init__(self):
        self.cache = {}  # 缓存市场数据
        self.cache_ttl = 3600  # 缓存有效期 1 小时
    
    def create_exchange(self, exchange_name: str, market_type: str):
        """
        创建交易所实例
        
        Args:
            exchange_name: 交易所名称 (binance, bybit)
            market_type: 市场类型 (spot, future)
        
        Returns:
            交易所实例
        """
        exchange_class = getattr(ccxt, exchange_name)
        
        config = {
            'enableRateLimit': True,
            'timeout': 30000,
        }
        
        # 设置市场类型
        if market_type == 'future':
            config['options'] = {'defaultType': 'swap'}  # 永续合约
        else:
            config['options'] = {'defaultType': 'spot'}
        
        return exchange_class(config)
    
    @retry(
        stop=stop_after_attempt(3),
        wait=wait_exponential(multiplier=1, min=4, max=10),
        retry=retry_if_exception_type((ccxt.NetworkError, ccxt.RequestTimeout))
    )
    async def load_markets_with_retry(self, exchange) -> Dict:
        """
        加载市场数据（带重试）
        
        Args:
            exchange: 交易所实例
        
        Returns:
            市场数据字典
        """
        return await exchange.load_markets()
    
    def is_usdt_pair(self, market: dict, market_type: str) -> bool:
        """
        判断是否为 USDT 交易对
        
        Args:
            market: 市场信息
            market_type: 市场类型 (spot, future)
        
        Returns:
            是否为 USDT 交易对
        """
        # 必须是活跃交易对
        if not market.get('active', False):
            return False
        
        if market_type == 'spot':
            # 现货: quote 必须是 USDT，且类型必须是 spot
            return market.get('quote') == 'USDT' and market.get('type') == 'spot'
        else:
            # 期货: settle (质押物) 必须是 USDT
            # 且类型必须是 swap (永续合约)
            # exclude options and dated futures
            return (
                market.get('settle') == 'USDT' and 
                (market.get('swap') is True or market.get('type') == 'swap')
            )
    
    async def fetch_symbols(self, exchange_name: str, market_type: str) -> List[str]:
        """
        获取指定交易所和市场类型的所有 USDT 交易对
        
        Args:
            exchange_name: 交易所名称 (binance, bybit)
            market_type: 市场类型 (spot, future)
        
        Returns:
            USDT 交易对列表，如 ['BTC/USDT', 'ETH/USDT', ...]
        """
        exchange = None
        try:
            logger.info(f"开始获取 {exchange_name} {market_type} 交易对列表")
            
            # 创建交易所实例
            exchange = self.create_exchange(exchange_name, market_type)
            
            # 加载市场数据
            markets = await self.load_markets_with_retry(exchange)
            
            # 过滤 USDT 交易对
            symbols = []
            for symbol, market in markets.items():
                if self.is_usdt_pair(market, market_type):
                    symbols.append(symbol)
            
            # 调试: 显示前几个交易对的格式
            if symbols:
                sample_symbols = symbols[:3]
                logger.info(f"{exchange_name} {market_type}: 找到 {len(symbols)} 个 USDT 交易对，示例: {sample_symbols}")
            else:
                logger.info(f"{exchange_name} {market_type}: 找到 {len(symbols)} 个 USDT 交易对")
            
            return symbols
            
        except ccxt.RateLimitExceeded as e:
            logger.warning(f"{exchange_name} 触发限流: {e}")
            await asyncio.sleep(60)  # 等待 60 秒
            return []
        except ccxt.ExchangeError as e:
            logger.error(f"{exchange_name} 交易所错误: {e}")
            return []
        except Exception as e:
            logger.error(f"获取 {exchange_name} {market_type} 交易对失败: {e}", exc_info=True)
            return []
        finally:
            if exchange:
                await exchange.close()
    
    async def fetch_all_symbols(self, exchanges: List[str], market_types: List[str]) -> Dict[str, Dict[str, List[str]]]:
        """
        获取所有交易所的所有市场类型的交易对
        
        Args:
            exchanges: 交易所列表
            market_types: 市场类型列表
        
        Returns:
            嵌套字典: {exchange: {market_type: [symbols]}}
        """
        result = {}
        
        for exchange in exchanges:
            result[exchange] = {}
            for market_type in market_types:
                symbols = await self.fetch_symbols(exchange, market_type)
                result[exchange][market_type] = symbols
        
        # 统计总数
        total = sum(len(symbols) for ex_data in result.values() for symbols in ex_data.values())
        logger.info(f"总共获取 {total} 个交易对")
        
        return result


# 全局实例
symbol_fetcher = SymbolFetcher()
