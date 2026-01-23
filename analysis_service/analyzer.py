import ccxt.async_support as ccxt
import pandas as pd
import numpy as np
from typing import List, Tuple, Dict
from scipy.signal import argrelextrema
from sklearn.cluster import DBSCAN
import logging
import asyncio
from tenacity import retry, stop_after_attempt, wait_exponential, retry_if_exception_type

logger = logging.getLogger(__name__)

class MarketAnalyzer:
    """专业的市场分析器，用于识别关键支撑位和压力位 - 优化版本"""
    
    def __init__(self):
        self.default_total_candles = 2000  # 默认获取2000根K线
        self.batch_size = 1000  # 每批1000根
        self.top_n_levels = 5  # 返回top 5个支撑/压力位
    
    @retry(
        stop=stop_after_attempt(3),
        wait=wait_exponential(multiplier=1, min=4, max=10),
        retry=retry_if_exception_type((ccxt.NetworkError, ccxt.RequestTimeout))
    )
    async def _fetch_ohlcv_with_retry(self, exchange, symbol: str, timeframe: str, **kwargs):
        """获取 K 线数据（带重试）"""
        return await exchange.fetch_ohlcv(symbol, timeframe, **kwargs)
    
    async def fetch_klines(self, exchange_name: str, symbol: str, market_type: str, timeframe: str) -> pd.DataFrame:
        """
        分两批获取K线数据，总共2000根
        第一批：最新的1000根
        第二批：更早的1000根
        """
        exchange_class = getattr(ccxt, exchange_name)
        
        # 根据市场类型设置配置
        config = {
            'enableRateLimit': True,
            'timeout': 30000,
        }
        
        if market_type == 'future':
            config['options'] = {'defaultType': 'swap'}
        else:
            config['options'] = {'defaultType': 'spot'}
        
        exchange = exchange_class(config)
        
        try:
            logger.debug(f"开始获取 {exchange_name} {symbol} {market_type} {timeframe} 的K线数据")
            
            # 第一批：获取最新的1000根K线
            batch1 = await self._fetch_ohlcv_with_retry(exchange, symbol, timeframe, limit=self.batch_size)
            logger.debug(f"第1批：获取 {len(batch1)} 根K线")
            
            if len(batch1) == 0:
                raise ValueError(f"无法获取 {symbol} 的K线数据")
            
            # 第二批：获取更早的1000根K线
            # 计算时间偏移量
            timeframe_ms = self._timeframe_to_ms(timeframe)
            since_timestamp = batch1[0][0] - (self.batch_size * timeframe_ms)
            
            batch2 = await self._fetch_ohlcv_with_retry(
                exchange, symbol, timeframe, 
                since=since_timestamp, 
                limit=self.batch_size
            )
            logger.debug(f"第2批：获取 {len(batch2)} 根K线")
            
            # 合并两批数据，去重
            all_candles = batch2 + batch1
            unique_candles = {candle[0]: candle for candle in all_candles}  # 用时间戳去重
            sorted_candles = sorted(unique_candles.values(), key=lambda x: x[0])
            
            # 转换为DataFrame
            df = pd.DataFrame(sorted_candles, columns=['timestamp', 'open', 'high', 'low', 'close', 'volume'])
            df['timestamp'] = pd.to_datetime(df['timestamp'], unit='ms')
            
            logger.debug(f"最终获取 {len(df)} 根K线数据")
            return df
            
        except ccxt.RateLimitExceeded as e:
            logger.warning(f"{exchange_name} {symbol} 触发限流: {e}")
            await asyncio.sleep(60)  # 等待 60 秒
            return pd.DataFrame()  # 返回空 DataFrame
        except ccxt.BadSymbol as e:
            logger.warning(f"{exchange_name} {symbol} 交易对不存在: {e}")
            return pd.DataFrame()
        except Exception as e:
            logger.error(f"获取 {exchange_name} {symbol} K线失败: {e}")
            return pd.DataFrame()
        finally:
            await exchange.close()
    
    def _timeframe_to_ms(self, timeframe: str) -> int:
        """将时间周期转换为毫秒"""
        unit = timeframe[-1]
        value = int(timeframe[:-1])
        
        multipliers = {
            'm': 60 * 1000,
            'h': 60 * 60 * 1000,
            'd': 24 * 60 * 60 * 1000,
            'w': 7 * 24 * 60 * 60 * 1000,
        }
        
        return value * multipliers.get(unit, 60 * 1000)
    
    def _timeframe_to_hours(self, timeframe: str) -> float:
        """将时间周期转换为小时数"""
        unit = timeframe[-1]
        value = int(timeframe[:-1])
        
        multipliers = {
            'm': 1/60,
            'h': 1,
            'd': 24,
            'w': 24 * 7,
        }
        
        return value * multipliers.get(unit, 1)
    
    def _get_dynamic_windows(self, timeframe: str, data_length: int) -> List[int]:
        """
        优化1: 根据时间周期动态计算窗口大小
        
        策略：基于时间跨度而非固定K线数量
        - 5小时、12小时、1天、2天、3天的时间窗口
        - 自动转换为对应时间周期的K线数量
        """
        # 基础时间窗口（小时）
        base_windows_hours = [5, 12, 24, 48, 72]
        
        # 转换为对应时间周期的K线数量
        timeframe_hours = self._timeframe_to_hours(timeframe)
        windows = [max(3, int(w / timeframe_hours)) for w in base_windows_hours]
        
        # 限制窗口大小不超过数据长度的1/10，确保足够数据用于后续分析
        max_window = data_length // 10
        windows = [w for w in windows if 3 <= w <= max_window]
        
        # 确保至少有一个窗口
        if not windows:
            windows = [min(5, data_length // 20)]
        
        logger.debug(f"时间周期 {timeframe}: 动态窗口 = {windows}")
        return windows
    
    def analyze_support_resistance(self, df: pd.DataFrame, timeframe: str = '1h') -> Tuple[List[float], List[float]]:
        """
        专业算法识别支撑位和压力位 - 完全优化版
        
        新增优化：
        1. 动态窗口大小（基于时间周期）
        2. DBSCAN智能聚类
        3. 时间衰减权重
        4. 强度验证系统
        5. 优化距离评分
        6. 技术指标辅助
        """
        if len(df) < 50:
            logger.warning(f"数据量不足（{len(df)}根），无法进行可靠分析")
            return [], []
        
        current_price = float(df['close'].iloc[-1])
        logger.debug(f"分析 {len(df)} 根K线，当前价格: {current_price:.6f}")
        
        # 步骤1：使用动态窗口检测局部极值点
        support_candidates = self._find_local_extrema(df, timeframe, is_support=True)
        resistance_candidates = self._find_local_extrema(df, timeframe, is_support=False)
        
        logger.debug(f"检测到 {len(support_candidates)} 个支撑候选，{len(resistance_candidates)} 个压力候选")
        
        # 步骤2：DBSCAN聚类并评分（包含时间衰减）
        top_supports = self._cluster_and_rank_advanced(
            support_candidates, df, current_price, is_support=True
        )
        top_resistances = self._cluster_and_rank_advanced(
            resistance_candidates, df, current_price, is_support=False
        )
        
        # 清理候选列表，释放内存
        del support_candidates
        del resistance_candidates
        
        # 步骤3：强度验证
        validated_supports = self._validate_levels(top_supports, df, is_support=True)
        validated_resistances = self._validate_levels(top_resistances, df, is_support=False)
        
        del top_supports
        del top_resistances
        
        # 步骤4：添加技术指标识别的关键位
        supports_with_tech, resistances_with_tech = self._add_technical_levels(
            df, validated_supports, validated_resistances, current_price
        )
        
        # 步骤5：最终排序并提取top N
        supports = [level['price'] for level in supports_with_tech[:self.top_n_levels]]
        resistances = [level['price'] for level in resistances_with_tech[:self.top_n_levels]]
        
        logger.debug(f"最终识别: {len(supports)} 个支撑位, {len(resistances)} 个压力位")
        
        return supports, resistances
    
    def _find_local_extrema(self, df: pd.DataFrame, timeframe: str, is_support: bool) -> List[Dict]:
        """使用动态窗口检测局部极值点"""
        candidates = []
        windows = self._get_dynamic_windows(timeframe, len(df))
        
        for window in windows:
            if is_support:
                # 找局部最小值
                indices = argrelextrema(df['low'].values, np.less, order=window)[0]
                for idx in indices:
                    candidates.append({
                        'price': float(df['low'].iloc[idx]),
                        'volume': float(df['volume'].iloc[idx]),
                        'index': idx,
                        'window': window,
                        'timestamp': df['timestamp'].iloc[idx]
                    })
            else:
                # 找局部最大值
                indices = argrelextrema(df['high'].values, np.greater, order=window)[0]
                for idx in indices:
                    candidates.append({
                        'price': float(df['high'].iloc[idx]),
                        'volume': float(df['volume'].iloc[idx]),
                        'index': idx,
                        'window': window,
                        'timestamp': df['timestamp'].iloc[idx]
                    })
        
        return candidates
    
    def _cluster_prices_dbscan(self, prices: np.ndarray, current_price: float) -> np.ndarray:
        """
        优化2: 使用DBSCAN进行智能聚类
        
        优势：
        - 自动确定聚类数量
        - 可以识别噪声点
        - 适应不同密度的聚类
        """
        if len(prices) == 0:
            return np.array([])
        
        # 动态计算eps（基于价格大小，使用0.3%作为聚类范围）
        eps = abs(current_price) * 0.003
        
        # DBSCAN聚类
        prices_2d = prices.reshape(-1, 1)
        clustering = DBSCAN(eps=eps, min_samples=2).fit(prices_2d)
        
        return clustering.labels_
    
    def _calculate_time_decay_factor(self, index: int, total_length: int) -> float:
        """
        优化3: 计算时间衰减因子
        
        策略：线性衰减，最新的数据权重为1.0，最早的为0.3
        确保最近的支撑/压力位更重要
        """
        # 归一化位置: 0 (最早) -> 1 (最新)
        normalized_position = index / (total_length - 1) if total_length > 1 else 1.0
        
       # 线性衰减: 0.3 -> 1.0
        decay_factor = 0.3 + 0.7 * normalized_position
        
        return decay_factor
    
    def _cluster_and_rank_advanced(self, candidates: List[Dict], df: pd.DataFrame, 
                                   current_price: float, is_support: bool) -> List[Dict]:
        """
        高级聚类和排序（整合优化2和优化3）
        
        新功能：
        1. 使用DBSCAN代替简单阈值聚类
        2. 应用时间衰减权重
        3. 改进的评分系统
        """
        if not candidates:
            return []
        
        # 提取价格数组
        prices = np.array([c['price'] for c in candidates])
        
        # DBSCAN聚类
        labels = self._cluster_prices_dbscan(prices, current_price)
        
        # 按聚类分组
        clusters_dict = {}
        for i, label in enumerate(labels):
            if label == -1:  # 噪声点，单独成组
                clusters_dict[f'noise_{i}'] = [candidates[i]]
            else:
                if label not in clusters_dict:
                    clusters_dict[label] = []
                clusters_dict[label].append(candidates[i])
        
        # 为每个聚类计算综合得分
        scored_levels = []
        avg_volume = df['volume'].mean()
        total_length = len(df)
        
        for cluster_id, cluster in clusters_dict.items():
            # 聚类平均价格
            avg_price = sum(c['price'] for c in cluster) / len(cluster)
            
            # 触及次数
            touch_count = len(cluster)
            
            # 时间加权成交量
            weighted_volume = 0
            total_weight = 0
            for candidate in cluster:
                time_decay = self._calculate_time_decay_factor(candidate['index'], total_length)
                weighted_volume += candidate['volume'] * time_decay
                total_weight += time_decay
            
            cluster_volume = weighted_volume / total_weight if total_weight > 0 else avg_volume
            
            # 距离当前价格
            distance_pct = abs(avg_price - current_price) / abs(current_price)
            
            # 最近触及时间（最大的index）
            latest_index = max(c['index'] for c in cluster)
            recency_factor = self._calculate_time_decay_factor(latest_index, total_length)
            
            # === 评分因素 ===
            
            # 1. 触及次数得分（权重最高）
            touch_score = touch_count * 3.0
            
            # 2. 时间加权成交量得分
            volume_score = (cluster_volume / avg_volume) * 2.0
            
            # 3. 优化的距离得分（考虑最近触及时间）
            # 距离1%-10%最优，但最近触及的给予额外加成
            if 0.01 <= distance_pct <= 0.10:
                distance_score = 2.5 * recency_factor
            elif distance_pct < 0.01:
                # 非常接近当前价格，最近触及的很重要
                distance_score = 2.0 * recency_factor
            else:
                # 距离较远，根据距离递减
                distance_score = max(0.2, (2.0 - (distance_pct - 0.10) * 5)) * recency_factor
            
            # 4. 方向得分
            if is_support:
                direction_score = 2.0 if avg_price < current_price else 0.5
            else:
                direction_score = 2.0 if avg_price > current_price else 0.5
            
            # 5. 窗口多样性得分（在多个窗口中出现的更可靠）
            unique_windows = len(set(c['window'] for c in cluster))
            window_diversity_score = unique_windows * 0.5
            
            # 综合得分
            total_score = (touch_score + volume_score + distance_score + 
                          direction_score + window_diversity_score)
            
            scored_levels.append({
                'price': avg_price,
                'score': total_score,
                'touch_count': touch_count,
                'volume': cluster_volume,
                'distance_pct': distance_pct * 100,
                'recency_factor': recency_factor,
                'latest_index': latest_index
            })
        
        # 按得分降序排序
        scored_levels.sort(key=lambda x: x['score'], reverse=True)
        
        return scored_levels
    
    def _validate_levels(self, levels: List[Dict], df: pd.DataFrame, is_support: bool) -> List[Dict]:
        """
        优化4: 验证支撑/压力位的强度
        
        验证指标：
        - 触及次数
        - 反弹次数
        - 突破次数
        - 强度得分（反弹率 - 突破率）
        """
        validated_levels = []
        
        for level in levels:
            level_price = level['price']
            tolerance = abs(level_price) * 0.01  # 1%容差
            
            touches = 0
            bounces = 0
            breaks = 0
            
            for i in range(len(df) - 1):  # 排除最后一根，因为需要看下一根
                price_low = df['low'].iloc[i]
                price_high = df['high'].iloc[i]
                
                # 检查是否触及该价位
                touched = False
                if is_support:
                    # 支撑位：低点接近该价位
                    if abs(price_low - level_price) <= tolerance:
                        touched = True
                        touches += 1
                        
                        # 检查下一根是否反弹
                        next_close = df['close'].iloc[i + 1]
                        current_close = df['close'].iloc[i]
                        if next_close > current_close:
                            bounces += 1
                        
                        # 检查是否突破
                        next_low = df['low'].iloc[i + 1]
                        if next_low < level_price - tolerance:
                            breaks += 1
                else:
                    # 压力位：高点接近该价位
                    if abs(price_high - level_price) <= tolerance:
                        touched = True
                        touches += 1
                        
                        # 检查下一根是否回落
                        next_close = df['close'].iloc[i + 1]
                        current_close = df['close'].iloc[i]
                        if next_close < current_close:
                            bounces += 1
                        
                        # 检查是否突破
                        next_high = df['high'].iloc[i + 1]
                        if next_high > level_price + tolerance:
                            breaks += 1
            
            # 计算强度指标
            bounce_rate = bounces / touches if touches > 0 else 0
            break_rate = breaks / touches if touches > 0 else 0
            strength = bounce_rate - break_rate  # 范围: -1 to 1
            
            # 强度得分加成（强度越高，得分越高）
            strength_bonus = strength * 2.0
            
            # 更新得分
            level['original_score'] = level['score']
            level['score'] = level['score'] + strength_bonus
            level['touches'] = touches
            level['bounces'] = bounces
            level['breaks'] = breaks
            level['bounce_rate'] = bounce_rate
            level['strength'] = strength
            
            validated_levels.append(level)
        
        # 重新排序
        validated_levels.sort(key=lambda x: x['score'], reverse=True)
        
        return validated_levels
    
    def _add_technical_levels(self, df: pd.DataFrame, supports: List[Dict], 
                             resistances: List[Dict], current_price: float) -> Tuple[List[Dict], List[Dict]]:
        """
        优化6: 添加技术指标识别的关键位
        
        指标：
        1. 布林带上下轨
        2. 斐波那契回撤位
        """
        # 1. 布林带
        df_copy = df.copy()
        df_copy['MA20'] = df_copy['close'].rolling(window=20).mean()
        df_copy['std20'] = df_copy['close'].rolling(window=20).std()
        df_copy['upper_band'] = df_copy['MA20'] + 2 * df_copy['std20']
        df_copy['lower_band'] = df_copy['MA20'] - 2 * df_copy['std20']
        
        if len(df_copy) >= 20:
            current_upper = float(df_copy['upper_band'].iloc[-1])
            current_lower = float(df_copy['lower_band'].iloc[-1])
            
            # 检查是否已存在类似的价位（避免重复）
            def is_duplicate(price, existing_levels, threshold=0.005):
                for level in existing_levels:
                    if abs(price - level['price']) / abs(price) < threshold:
                        return True
                return False
            
            # 添加布林带上轨
            if current_upper > current_price and not is_duplicate(current_upper, resistances):
                resistances.append({
                    'price': current_upper,
                    'score': 5.0,  # 技术指标给予固定得分
                    'source': 'Bollinger_Upper',
                    'touch_count': 1,
                    'volume': 0,
                    'distance_pct': abs(current_upper - current_price) / abs(current_price) * 100
                })
            
            # 添加布林带下轨
            if current_lower < current_price and not is_duplicate(current_lower, supports):
                supports.append({
                    'price': current_lower,
                    'score': 5.0,
                    'source': 'Bollinger_Lower',
                    'touch_count': 1,
                    'volume': 0,
                    'distance_pct': abs(current_lower - current_price) / abs(current_price) * 100
                })
        
        # 2. 斐波那契回撤位
        high = df['high'].max()
        low = df['low'].min()
        diff = high - low
        
        fib_ratios = {
            'Fib_23.6': 0.236,
            'Fib_38.2': 0.382,
            'Fib_50.0': 0.500,
            'Fib_61.8': 0.618,
            'Fib_78.6': 0.786,
        }
        
        for name, ratio in fib_ratios.items():
            fib_price = low + diff * ratio
            
            # 根据当前价格决定是支撑还是压力
            if fib_price < current_price * 0.99:  # 低于当前价格1%以上
                if not is_duplicate(fib_price, supports):
                    supports.append({
                        'price': fib_price,
                        'score': 4.0,  # 斐波那契得分略低于布林带
                        'source': name,
                        'touch_count': 1,
                        'volume': 0,
                        'distance_pct': abs(fib_price - current_price) / abs(current_price) * 100
                    })
            elif fib_price > current_price * 1.01:  # 高于当前价格1%以上
                if not is_duplicate(fib_price, resistances):
                    resistances.append({
                        'price': fib_price,
                        'score': 4.0,
                        'source': name,
                        'touch_count': 1,
                        'volume': 0,
                        'distance_pct': abs(fib_price - current_price) / abs(current_price) * 100
                    })
        
        # 重新排序
        supports.sort(key=lambda x: x['score'], reverse=True)
        resistances.sort(key=lambda x: x['score'], reverse=True)
        
        # 清理临时DataFrame
        del df_copy
        
        return supports, resistances

analyzer = MarketAnalyzer()
