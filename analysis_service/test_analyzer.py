"""
测试优化后的分析器

运行测试：
cd /Users/doudou/trading_assistant_freq/analysis_service
python test_analyzer.py
"""
import asyncio
import pandas as pd
import numpy as np
from analyzer import analyzer
import logging

# 配置日志
logging.basicConfig(
    level=logging.DEBUG,
    format='%(asctime)s - %(name)s - %(levelname)s - %(message)s'
)

logger = logging.getLogger(__name__)


def create_test_data(num_candles=500):
    """创建测试用的K线数据"""
    # 生成模拟价格数据（带趋势和支撑/压力位）
    np.random.seed(42)
    
    # 基础趋势
    trend = np.linspace(100, 120, num_candles)
    
    # 添加噪声
    noise = np.random.normal(0, 2, num_candles)
    
    # 创建一些明显的支撑和压力位
    prices = trend + noise
    
    # 在特定价位增加反弹（模拟支撑）
    support_level = 105.0
    for i in range(100, 200):
        if prices[i] < support_level:
            prices[i] = support_level + np.random.uniform(0, 1)
    
    # 在特定价位增加回落（模拟压力）
    resistance_level = 115.0
    for i in range(300, 400):
        if prices[i] > resistance_level:
            prices[i] = resistance_level - np.random.uniform(0, 1)
    
    # 生成OHLC数据
    data = []
    for i, close in enumerate(prices):
        open_price = close + np.random.uniform(-0.5, 0.5)
        high = max(open_price, close) + abs(np.random.uniform(0, 1))
        low = min(open_price, close) - abs(np.random.uniform(0, 1))
        volume = np.random.uniform(1000, 10000)
        
        data.append({
            'timestamp': pd.Timestamp('2024-01-01') + pd.Timedelta(hours=i),
            'open': open_price,
            'high': high,
            'low': low,
            'close': close,
            'volume': volume
        })
    
    df = pd.DataFrame(data)
    logger.info(f"创建测试数据: {len(df)} 根K线")
    logger.info(f"预设支撑位: {support_level}")
    logger.info(f"预设压力位: {resistance_level}")
    logger.info(f"价格范围: {df['low'].min():.2f} - {df['high'].max():.2f}")
    
    return df, support_level, resistance_level


async def test_real_data():
    """测试真实数据"""
    logger.info("=" * 60)
    logger.info("测试1: 真实数据分析")
    logger.info("=" * 60)
    
    try:
        # 获取BTCUSDT 1h数据
        df = await analyzer.fetch_klines('binance', 'BTC/USDT', 'future', '1h')
        
        if df.empty:
            logger.error("无法获取数据")
            return
        
        logger.info(f"获取到 {len(df)} 根K线")
        logger.info(f"时间范围: {df['timestamp'].iloc[0]} 至 {df['timestamp'].iloc[-1]}")
        logger.info(f"当前价格: {df['close'].iloc[-1]:.2f}")
        
        # 分析支撑和压力位
        supports, resistances = analyzer.analyze_support_resistance(df, '1h')
        
        logger.info("=" * 60)
        logger.info("分析结果:")
        logger.info(f"识别到 {len(supports)} 个支撑位:")
        for i, s in enumerate(supports, 1):
            logger.info(f"  {i}. {s:.2f}")
        
        logger.info(f"识别到 {len(resistances)} 个压力位:")
        for i, r in enumerate(resistances, 1):
            logger.info(f"  {i}. {r:.2f}")
        
        logger.info("=" * 60)
        
    except Exception as e:
        logger.error(f"测试失败: {e}", exc_info=True)


def test_simulated_data():
    """测试模拟数据"""
    logger.info("=" * 60)
    logger.info("测试2: 模拟数据分析")
    logger.info("=" * 60)
    
    # 创建测试数据
    df, expected_support, expected_resistance = create_test_data()
    
    # 分析
    supports, resistances = analyzer.analyze_support_resistance(df, '1h')
    
    logger.info("=" * 60)
    logger.info("分析结果:")
    logger.info(f"识别到 {len(supports)} 个支撑位:")
    for i, s in enumerate(supports, 1):
        distance_to_expected = abs(s - expected_support)
        logger.info(f"  {i}. {s:.2f} (距离预设支撑: {distance_to_expected:.2f})")
    
    logger.info(f"识别到 {len(resistances)} 个压力位:")
    for i, r in enumerate(resistances, 1):
        distance_to_expected = abs(r - expected_resistance)
        logger.info(f"  {i}. {r:.2f} (距离预设压力: {distance_to_expected:.2f})")
    
    # 验证准确性
    support_found = any(abs(s - expected_support) < 2.0 for s in supports)
    resistance_found = any(abs(r - expected_resistance) < 2.0 for r in resistances)
    
    logger.info("=" * 60)
    logger.info("验证结果:")
    logger.info(f"支撑位识别: {'✓ 成功' if support_found else '✗ 失败'}")
    logger.info(f"压力位识别: {'✓ 成功' if resistance_found else '✗ 失败'}")
    logger.info("=" * 60)


def test_different_timeframes():
    """测试不同时间周期的动态窗口"""
    logger.info("=" * 60)
    logger.info("测试3: 动态窗口大小验证")
    logger.info("=" * 60)
    
    df, _, _ = create_test_data(1000)
    
    timeframes = ['15m', '1h', '4h', '1d']
    
    for tf in timeframes:
        windows = analyzer._get_dynamic_windows(tf, len(df))
        logger.info(f"{tf:4s} -> 窗口: {windows}")
    
    logger.info("=" * 60)


async def main():
    """运行所有测试"""
    logger.info("\n" + "=" * 60)
    logger.info("开始测试优化后的分析器")
    logger.info("=" * 60 + "\n")
    
    # 测试1: 模拟数据
    test_simulated_data()
    
    # 测试2: 动态窗口
    test_different_timeframes()
    
    # 测试3: 真实数据
    await test_real_data()
    
    logger.info("\n" + "=" * 60)
    logger.info("所有测试完成")
    logger.info("=" * 60 + "\n")


if __name__ == "__main__":
    asyncio.run(main())
