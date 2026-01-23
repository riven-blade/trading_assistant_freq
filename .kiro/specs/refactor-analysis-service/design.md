# 重构分析服务 - 设计文档

## 1. 系统架构设计

### 1.1 整体架构

```
┌─────────────────────────────────────────────────────────────┐
│                    Python 分析服务                            │
│  ┌──────────────────────────────────────────────────────┐   │
│  │           定时任务调度器 (APScheduler)                 │   │
│  │              每小时触发一次                            │   │
│  └────────────────┬─────────────────────────────────────┘   │
│                   │                                          │
│                   ▼                                          │
│  ┌──────────────────────────────────────────────────────┐   │
│  │           市场分析协调器 (Coordinator)                 │   │
│  │  - 遍历交易所 [Binance, Bybit]                        │   │
│  │  - 遍历市场类型 [spot, future]                        │   │
│  │  - 遍历时间周期 [1h, 4h]                              │   │
│  └────────────────┬─────────────────────────────────────┘   │
│                   │                                          │
│                   ▼                                          │
│  ┌──────────────────────────────────────────────────────┐   │
│  │         交易对获取器 (Symbol Fetcher)                  │   │
│  │  - 调用交易所 API 获取所有交易对                       │   │
│  │  - 过滤 USDT 质押物交易对                             │   │
│  └────────────────┬─────────────────────────────────────┘   │
│                   │                                          │
│                   ▼                                          │
│  ┌──────────────────────────────────────────────────────┐   │
│  │         市场分析器 (Market Analyzer)                   │   │
│  │  - 获取 2000 根 K 线数据                              │   │
│  │  - 计算支撑/压力位 (Top 5)                            │   │
│  └────────────────┬─────────────────────────────────────┘   │
│                   │                                          │
│                   ▼                                          │
│  ┌──────────────────────────────────────────────────────┐   │
│  │         数据库管理器 (DB Manager)                      │   │
│  │  - Upsert 分析结果                                    │   │
│  │  - 批量提交事务                                        │   │
│  └──────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
                    ┌──────────────────┐
                    │   MySQL 数据库    │
                    │ analysis_results │
                    └──────────────────┘
                              ▲
                              │
┌─────────────────────────────────────────────────────────────┐
│                      Go 后端服务                              │
│  ┌──────────────────────────────────────────────────────┐   │
│  │        Analysis Controller                           │   │
│  │  - GET /api/v1/analysis/results                      │   │
│  │  - GET /api/v1/analysis/:id                          │   │
│  └──────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
                    ┌──────────────────┐
                    │   前端 (React)    │
                    │  Analysis.js     │
                    │  ChartPage.js    │
                    └──────────────────┘
```

### 1.2 服务职责

#### Python 分析服务
- **定时任务**: 每小时自动触发分析
- **数据采集**: 从交易所获取交易对和 K 线数据
- **数据分析**: 计算支撑/压力位
- **数据存储**: 写入 MySQL 数据库
- **健康检查**: 提供 HTTP 健康检查端点

#### Go 后端服务
- **API 网关**: 提供 RESTful API
- **数据查询**: 从数据库读取分析结果
- **前端服务**: 服务前端静态文件

#### 前端应用
- **数据展示**: 显示分析结果列表
- **图表可视化**: 在 K 线图上绘制支撑/压力位

## 2. 核心模块设计

### 2.0 启动流程设计

**文件**: `analysis_service/main.py`

**启动顺序**:
```python
@asynccontextmanager
async def lifespan(app: FastAPI):
    # Startup
    logger.info("========== 分析服务启动 ==========")
    
    # 1. 初始化数据库
    async with engine.begin() as conn:
        await conn.run_sync(Base.metadata.create_all)
    logger.info("✓ 数据库表已创建")
    
    # 2. 启动定时调度器
    scheduler.start()
    logger.info("✓ 定时调度器已启动 (每小时执行)")
    
    # 3. 立即执行首次分析
    logger.info("→ 开始执行首次分析...")
    try:
        await run_market_analysis()
        logger.info("✓ 首次分析完成")
    except Exception as e:
        logger.error(f"✗ 首次分析失败: {e}", exc_info=True)
    
    logger.info("========== 服务启动完成 ==========")
    logger.info("后续将每小时自动执行分析任务")
    
    yield
    
    # Shutdown
    scheduler.shutdown()
    logger.info("定时调度器已停止")
    await engine.dispose()
    logger.info("数据库连接已关闭")
```

**关键点**:
1. 服务启动后立即执行一次完整分析
2. 首次分析失败不影响服务启动
3. 定时任务继续按计划执行
4. 清晰的日志输出，便于监控

### 2.1 定时任务调度器 (Scheduler)

**文件**: `analysis_service/scheduler.py`

**职责**: 管理定时任务的执行


**技术选型**: APScheduler

**配置**:
```python
from apscheduler.schedulers.asyncio import AsyncIOScheduler
from apscheduler.triggers.interval import IntervalTrigger

scheduler = AsyncIOScheduler()

# 添加定时任务，每小时执行一次
scheduler.add_job(
    run_analysis,
    trigger=IntervalTrigger(hours=1),
    id='market_analysis',
    name='Market Analysis Job',
    replace_existing=True
)

# 启动时立即执行一次
async def start_scheduler():
    """启动调度器并立即执行一次分析"""
    logger.info("启动定时任务调度器")
    scheduler.start()
    
    # 立即执行一次分析
    logger.info("程序启动，立即执行首次分析")
    await run_analysis()
    logger.info("首次分析完成，后续将每小时自动执行")
```

**关键方法**:
- `start_scheduler()`: 启动调度器并立即执行一次分析
- `stop_scheduler()`: 停止调度器
- `run_analysis()`: 执行分析任务的入口函数

**执行时机**:
1. **程序启动时**: 立即执行一次完整分析
2. **后续执行**: 每隔 1 小时自动执行一次

### 2.2 市场分析协调器 (Coordinator)

**文件**: `analysis_service/coordinator.py`

**职责**: 协调整个分析流程

**核心逻辑**:
```python
async def run_market_analysis():
    """
    执行完整的市场分析流程
    """
    exchanges = ['binance', 'bybit']
    market_types = ['spot', 'future']
    timeframes = ['1h', '4h']
    
    stats = {
        'total': 0,
        'success': 0,
        'failed': 0,
        'start_time': datetime.now()
    }
    
    for exchange in exchanges:
        for market_type in market_types:
            # 获取交易对列表
            symbols = await fetch_symbols(exchange, market_type)
            
            for symbol in symbols:
                for timeframe in timeframes:
                    try:
                        # 分析单个交易对
                        await analyze_symbol(
                            exchange, symbol, 
                            market_type, timeframe
                        )
                        stats['success'] += 1
                    except Exception as e:
                        logger.error(f"分析失败: {symbol}", exc_info=True)
                        stats['failed'] += 1
                    finally:
                        stats['total'] += 1
    
    # 记录统计信息
    log_statistics(stats)
```

**关键方法**:
- `run_market_analysis()`: 主入口函数
- `analyze_symbol()`: 分析单个交易对
- `log_statistics()`: 记录统计信息

### 2.3 交易对获取器 (Symbol Fetcher)

**文件**: `analysis_service/symbol_fetcher.py`

**职责**: 从交易所获取 USDT 交易对列表

**实现逻辑**:
```python
async def fetch_symbols(exchange_name: str, market_type: str) -> List[str]:
    """
    获取指定交易所和市场类型的所有 USDT 交易对
    
    Args:
        exchange_name: 交易所名称 (binance, bybit)
        market_type: 市场类型 (spot, future)
    
    Returns:
        USDT 交易对列表，如 ['BTC/USDT', 'ETH/USDT', ...]
    """
    exchange = create_exchange(exchange_name, market_type)
    
    try:
        # 加载市场数据
        await exchange.load_markets()
        
        # 过滤 USDT 交易对
        symbols = []
        for symbol, market in exchange.markets.items():
            if is_usdt_pair(market, market_type):
                symbols.append(symbol)
        
        logger.info(f"{exchange_name} {market_type}: 找到 {len(symbols)} 个交易对")
        return symbols
        
    finally:
        await exchange.close()

def is_usdt_pair(market: dict, market_type: str) -> bool:
    """
    判断是否为 USDT 交易对
    
    现货: quote = 'USDT'
    期货: settle = 'USDT' (质押物)
    """
    if market_type == 'spot':
        return market.get('quote') == 'USDT' and market.get('active', False)
    else:  # future
        return market.get('settle') == 'USDT' and market.get('active', False)
```

**关键方法**:
- `fetch_symbols()`: 获取交易对列表
- `is_usdt_pair()`: 判断是否为 USDT 交易对
- `create_exchange()`: 创建交易所实例

### 2.4 市场分析器 (Market Analyzer)

**文件**: `analysis_service/analyzer.py` (已存在，需要优化)

**职责**: 执行技术分析，计算支撑/压力位

**优化点**:
1. 固定 K 线数量为 2000
2. 优化批量获取逻辑
3. 添加错误处理和重试

**关键方法**:
- `fetch_klines()`: 获取 2000 根 K 线
- `analyze_support_resistance()`: 计算支撑/压力位

### 2.5 数据库管理器 (DB Manager)

**文件**: `analysis_service/db_manager.py`

**职责**: 管理数据库操作

**核心逻辑**:
```python
async def upsert_analysis_result(
    db: AsyncSession,
    exchange: str,
    symbol: str,
    market_type: str,
    timeframe: str,
    support_levels: List[float],
    resistance_levels: List[float],
    last_price: float,
    input_limit: int
):
    """
    插入或更新分析结果
    """
    # 查询是否存在
    query = select(AnalysisResult).where(
        AnalysisResult.exchange == exchange,
        AnalysisResult.symbol == symbol,
        AnalysisResult.market_type == market_type,
        AnalysisResult.timeframe == timeframe
    )
    result = await db.execute(query)
    existing = result.scalar_one_or_none()
    
    if existing:
        # 更新
        existing.support_levels = support_levels
        existing.resistance_levels = resistance_levels
        existing.last_price = last_price
        existing.input_limit = input_limit
        existing.updated_at = func.now()
    else:
        # 插入
        new_record = AnalysisResult(
            exchange=exchange,
            symbol=symbol,
            market_type=market_type,
            timeframe=timeframe,
            support_levels=support_levels,
            resistance_levels=resistance_levels,
            last_price=last_price,
            input_limit=input_limit
        )
        db.add(new_record)
    
    await db.commit()
```

**关键方法**:
- `upsert_analysis_result()`: 插入或更新分析结果
- `batch_commit()`: 批量提交事务

## 3. 数据流设计

### 3.1 完整分析流程

```
1. 触发时机
   ├─> 程序启动时 (立即执行)
   └─> 定时触发 (每小时)
       └─> run_market_analysis()

2. 遍历交易所
   ├─> Binance
   │   ├─> Spot
   │   │   ├─> 获取所有 USDT 交易对
   │   │   ├─> 遍历每个交易对
   │   │   │   ├─> 分析 1h 周期
   │   │   │   └─> 分析 4h 周期
   │   │   └─> 写入数据库
   │   └─> Future
   │       └─> (同上)
   └─> Bybit
       └─> (同上)

3. 记录统计信息
   └─> 总数、成功数、失败数、耗时
```

### 3.2 单个交易对分析流程

```
输入: (exchange, symbol, market_type, timeframe)
例如: ('binance', 'BTC/USDT', 'spot', '1h')

1. 获取 K 线数据
   ├─> 调用 ccxt API
   ├─> 获取 2000 根 K 线
   └─> 转换为 DataFrame

2. 技术分析
   ├─> 多窗口局部极值检测 [5, 10, 20, 30]
   ├─> 价格聚类 (0.5% 阈值)
   ├─> 综合评分
   │   ├─> 触及次数得分 (权重 3.0)
   │   ├─> 成交量得分 (权重 1.5)
   │   ├─> 距离得分
   │   └─> 方向得分
   └─> 返回 Top 5 支撑位和压力位

3. 数据存储
   ├─> 查询是否存在记录
   ├─> 存在则更新，不存在则插入
   └─> 提交事务

输出: 分析结果写入数据库
```

## 4. 配置设计

### 4.1 环境变量

**文件**: `analysis_service/.env`

```bash
# 数据库配置
MYSQL_HOST=localhost
MYSQL_PORT=3306
MYSQL_USER=trading
MYSQL_PASSWORD=your_password
MYSQL_DB=trading_analysis

# 分析配置
ANALYSIS_INTERVAL_HOURS=1
ANALYSIS_EXCHANGES=binance,bybit
ANALYSIS_TIMEFRAMES=1h,4h
ANALYSIS_KLINE_LIMIT=2000
ANALYSIS_RUN_ON_STARTUP=true

# 日志配置
LOG_LEVEL=INFO
```

### 4.2 配置类

**文件**: `analysis_service/config.py`

```python
import os
from typing import List

class Config:
    # 数据库配置
    MYSQL_HOST = os.getenv("MYSQL_HOST", "localhost")
    MYSQL_PORT = int(os.getenv("MYSQL_PORT", "3306"))
    MYSQL_USER = os.getenv("MYSQL_USER", "root")
    MYSQL_PASSWORD = os.getenv("MYSQL_PASSWORD", "root")
    MYSQL_DB = os.getenv("MYSQL_DB", "trading_analysis")
    
    # 分析配置
    ANALYSIS_INTERVAL_HOURS = int(os.getenv("ANALYSIS_INTERVAL_HOURS", "1"))
    ANALYSIS_EXCHANGES = os.getenv("ANALYSIS_EXCHANGES", "binance,bybit").split(",")
    ANALYSIS_TIMEFRAMES = os.getenv("ANALYSIS_TIMEFRAMES", "1h,4h").split(",")
    ANALYSIS_KLINE_LIMIT = int(os.getenv("ANALYSIS_KLINE_LIMIT", "2000"))
    ANALYSIS_RUN_ON_STARTUP = os.getenv("ANALYSIS_RUN_ON_STARTUP", "true").lower() == "true"  # 默认启动时执行
    
    # 日志配置
    LOG_LEVEL = os.getenv("LOG_LEVEL", "INFO")
    
    @property
    def database_url(self) -> str:
        return f"mysql+aiomysql://{self.MYSQL_USER}:{self.MYSQL_PASSWORD}@{self.MYSQL_HOST}:{self.MYSQL_PORT}/{self.MYSQL_DB}"

config = Config()
```

## 5. 错误处理和容错设计

### 5.1 错误分类

1. **交易所 API 错误**
   - 限流 (RateLimitExceeded)
   - 网络超时 (NetworkError)
   - 交易对不存在 (BadSymbol)

2. **数据库错误**
   - 连接失败
   - 事务冲突
   - 超时

3. **数据分析错误**
   - K 线数据不足
   - 计算异常

### 5.2 容错策略

```python
# 1. 交易所 API 重试
@retry(
    stop=stop_after_attempt(3),
    wait=wait_exponential(multiplier=1, min=4, max=10),
    retry=retry_if_exception_type((NetworkError, RequestTimeout))
)
async def fetch_klines_with_retry(...):
    pass

# 2. 限流处理
async def fetch_with_rate_limit(exchange, ...):
    try:
        return await exchange.fetch_ohlcv(...)
    except RateLimitExceeded:
        logger.warning("触发限流，等待 60 秒")
        await asyncio.sleep(60)
        return await exchange.fetch_ohlcv(...)

# 3. 单个交易对失败不影响其他
for symbol in symbols:
    try:
        await analyze_symbol(symbol)
    except Exception as e:
        logger.error(f"分析 {symbol} 失败: {e}")
        continue  # 继续下一个

# 4. 数据库连接池
engine = create_async_engine(
    DATABASE_URL,
    pool_size=10,
    max_overflow=20,
    pool_pre_ping=True,  # 自动检测连接有效性
    pool_recycle=3600    # 1小时回收连接
)
```

## 6. 性能优化设计

### 6.1 并发控制

```python
# 使用信号量控制并发数
semaphore = asyncio.Semaphore(5)  # 最多 5 个并发请求

async def analyze_with_semaphore(symbol):
    async with semaphore:
        await analyze_symbol(symbol)

# 批量处理
tasks = [analyze_with_semaphore(s) for s in symbols]
await asyncio.gather(*tasks, return_exceptions=True)
```

### 6.2 数据库批量操作

```python
# 批量提交，每 50 条提交一次
batch_size = 50
for i in range(0, len(results), batch_size):
    batch = results[i:i+batch_size]
    async with AsyncSessionLocal() as session:
        for result in batch:
            await upsert_analysis_result(session, **result)
        await session.commit()
```

### 6.3 缓存优化

```python
# 缓存交易所市场数据（1小时有效）
market_cache = {}

async def get_markets(exchange_name):
    cache_key = f"{exchange_name}_markets"
    if cache_key in market_cache:
        cached_time, markets = market_cache[cache_key]
        if time.time() - cached_time < 3600:
            return markets
    
    # 重新获取
    exchange = create_exchange(exchange_name)
    markets = await exchange.load_markets()
    market_cache[cache_key] = (time.time(), markets)
    return markets
```

## 7. 日志和监控设计

### 7.1 日志级别

```python
# INFO: 正常流程
logger.info(f"开始分析 {exchange} {market_type}")
logger.info(f"找到 {len(symbols)} 个交易对")
logger.info(f"分析完成: {symbol} - 支撑位: {len(supports)}, 压力位: {len(resistances)}")

# WARNING: 可恢复的异常
logger.warning(f"触发限流，等待重试")
logger.warning(f"{symbol} K线数据不足")

# ERROR: 错误但不影响整体
logger.error(f"分析 {symbol} 失败: {e}", exc_info=True)

# CRITICAL: 严重错误
logger.critical(f"数据库连接失败", exc_info=True)
```

### 7.2 统计信息

```python
class AnalysisStats:
    def __init__(self):
        self.total = 0
        self.success = 0
        self.failed = 0
        self.start_time = None
        self.end_time = None
        self.errors = []
    
    def log_summary(self):
        duration = (self.end_time - self.start_time).total_seconds()
        logger.info(f"""
        ========== 分析统计 ==========
        总数: {self.total}
        成功: {self.success}
        失败: {self.failed}
        成功率: {self.success/self.total*100:.2f}%
        耗时: {duration:.2f} 秒
        ==============================
        """)
```

## 8. Go 后端改动

### 8.1 移除无用配置

**文件**: `deploy/docker-compose.yml`

```yaml
# 移除这一行
- ANALYSIS_SERVICE_URL=http://analysis-service:8000

# 移除端口映射
# ports:
#   - "8000:8000"
```

### 8.2 保持现有 API

**无需改动**:
- `controllers/analysis_controller.go`
- `pkg/models/analysis.go`
- `apis/routes.go`

Go 后端继续提供读取 API，前端功能保持不变。

## 9. Python 服务改动清单

### 9.1 新增文件

```
analysis_service/
├── scheduler.py          # 定时任务调度器
├── coordinator.py        # 分析协调器
├── symbol_fetcher.py     # 交易对获取器
├── db_manager.py         # 数据库管理器
└── config.py             # 配置管理
```

### 9.2 修改文件

```
analysis_service/
├── main.py               # 移除 HTTP 端点，添加调度器启动
├── analyzer.py           # 优化 K 线获取逻辑
└── requirements.txt      # 添加 APScheduler 依赖
```

### 9.3 删除文件

```
analysis_service/
└── schemas.py            # 不再需要 Pydantic 模型
```

## 10. 部署架构

### 10.1 Docker Compose 配置

```yaml
analysis-service:
  image: ddhdocker/trading-analysis:latest
  container_name: trading-analysis
  restart: unless-stopped
  environment:
    - TZ=Asia/Shanghai
    - MYSQL_HOST=mysql
    - MYSQL_PORT=3306
    - MYSQL_USER=trading
    - MYSQL_PASSWORD=your_password
    - MYSQL_DB=trading_analysis
    - ANALYSIS_INTERVAL_HOURS=1
    - ANALYSIS_EXCHANGES=binance,bybit
    - ANALYSIS_TIMEFRAMES=1h,4h
    - ANALYSIS_KLINE_LIMIT=2000
    - ANALYSIS_RUN_ON_STARTUP=true  # 启动时立即执行
  depends_on:
    - mysql
  networks:
    - trading-network
  labels:
    - "traefik.enable=false"
  # 移除端口映射，不对外暴露
```

### 10.2 健康检查

```yaml
healthcheck:
  test: ["CMD", "curl", "-f", "http://localhost:8000/health"]
  interval: 30s
  timeout: 10s
  retries: 3
  start_period: 40s
```

## 11. 测试策略

### 11.1 单元测试

```python
# test_symbol_fetcher.py
async def test_fetch_binance_spot_symbols():
    symbols = await fetch_symbols('binance', 'spot')
    assert len(symbols) > 0
    assert all('USDT' in s for s in symbols)

# test_analyzer.py
async def test_analyze_support_resistance():
    df = create_test_dataframe()
    supports, resistances = analyzer.analyze_support_resistance(df)
    assert len(supports) <= 5
    assert len(resistances) <= 5
```

### 11.2 集成测试

```python
# test_integration.py
async def test_full_analysis_flow():
    # 测试完整流程
    await run_market_analysis()
    
    # 验证数据库
    async with AsyncSessionLocal() as session:
        result = await session.execute(
            select(AnalysisResult).limit(1)
        )
        record = result.scalar_one()
        assert record is not None
        assert len(record.support_levels) > 0
```

### 11.3 性能测试

```python
# test_performance.py
async def test_analysis_performance():
    start = time.time()
    await analyze_symbol('binance', 'BTC/USDT', 'spot', '1h')
    duration = time.time() - start
    assert duration < 10  # 单个交易对分析应在 10 秒内完成
```

## 12. 实施步骤

### 阶段 1: 清理 Go 后端 (30 分钟)
1. 修改 `deploy/docker-compose.yml`
2. 搜索并确认无 ANALYSIS_SERVICE_URL 使用
3. 测试 Go 服务启动

### 阶段 2: 重构 Python 服务 (4 小时)
1. 创建新模块文件
2. 修改 main.py
3. 优化 analyzer.py
4. 更新 requirements.txt
5. 本地测试

### 阶段 3: 集成测试 (2 小时)
1. Docker 构建测试
2. 端到端测试
3. 性能测试

### 阶段 4: 部署上线 (1 小时)
1. 构建镜像
2. 推送到仓库
3. 更新生产环境
4. 监控运行状态

## 13. 回滚计划

如果出现问题，可以快速回滚：

1. **回滚 Docker 镜像**
   ```bash
   docker-compose down
   docker-compose up -d --force-recreate
   ```

2. **恢复旧版本代码**
   ```bash
   git revert <commit-hash>
   ```

3. **数据库无需回滚**
   - 数据库 schema 未改变
   - 只是数据更新方式改变

## 14. 监控指标

### 14.1 关键指标

- **分析成功率**: success / total
- **平均分析时间**: total_duration / total
- **失败率**: failed / total
- **数据库写入延迟**: db_write_time
- **交易所 API 调用次数**: api_call_count

### 14.2 告警规则

- 分析成功率 < 90% → 告警
- 单次分析时间 > 30 分钟 → 告警
- 连续 3 次失败 → 告警
- 数据库连接失败 → 紧急告警

## 15. 正确性属性

### 15.1 数据完整性
**属性**: 每个 USDT 交易对在每个时间周期都有分析结果
**验证**: 
```sql
SELECT COUNT(*) FROM analysis_results 
WHERE exchange = 'binance' 
  AND market_type = 'spot' 
  AND timeframe = '1h'
```

### 15.2 数据新鲜度
**属性**: 所有分析结果的 updated_at 在 2 小时内
**验证**:
```sql
SELECT COUNT(*) FROM analysis_results 
WHERE updated_at < NOW() - INTERVAL 2 HOUR
```

### 15.3 数据准确性
**属性**: 支撑位价格 < 当前价格 < 压力位价格
**验证**: 在分析逻辑中添加断言检查

## 16. 总结

本设计文档详细描述了如何重构分析服务，主要改动包括：

1. **Go 后端**: 移除无用的 ANALYSIS_SERVICE_URL 配置
2. **Python 服务**: 
   - 移除 HTTP 接口
   - 添加定时任务
   - 自动分析所有 USDT 交易对
   - 支持 Binance 和 Bybit
   - 固定分析 1h 和 4h 周期
   - 每次获取 2000 根 K 线

3. **架构优势**:
   - 完全自动化，无需手动触发
   - 服务解耦，职责清晰
   - 容错性强，单点失败不影响整体
   - 可扩展，易于添加新交易所

4. **性能目标**:
   - 单次完整分析 < 30 分钟
   - 分析成功率 > 90%
   - 内存使用 < 2GB
