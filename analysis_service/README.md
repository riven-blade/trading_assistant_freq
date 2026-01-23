# Trading Analysis Service

自动化市场分析服务 - 定时分析所有 USDT 交易对的支撑/压力位

## 功能特性

- ✅ **自动化分析**: 程序启动时立即执行一次，后续每小时自动执行
- ✅ **多交易所支持**: Binance 和 Bybit
- ✅ **全市场覆盖**: 现货和期货市场
- ✅ **多时间周期**: 1h 和 4h
- ✅ **专业算法**: 基于 2000 根 K 线的支撑/压力位分析
- ✅ **容错设计**: 单个交易对失败不影响整体
- ✅ **并发控制**: 避免触发交易所限流

## 架构设计

```
定时调度器 (每小时)
    ↓
分析协调器 (遍历交易所/市场/周期)
    ↓
交易对获取器 (获取所有 USDT 交易对)
    ↓
市场分析器 (2000根K线 → Top5支撑/压力位)
    ↓
数据库管理器 (Upsert 结果)
```

## 环境变量配置

创建 `.env` 文件（参考 `.env.example`）：

```bash
# 数据库配置
MYSQL_HOST=localhost
MYSQL_PORT=3306
MYSQL_USER=trading
MYSQL_PASSWORD=your_password
MYSQL_DB=trading_analysis

# 分析配置
ANALYSIS_INTERVAL_HOURS=1              # 分析间隔（小时）
ANALYSIS_EXCHANGES=binance,bybit       # 交易所列表
ANALYSIS_TIMEFRAMES=1h,4h              # 时间周期
ANALYSIS_KLINE_LIMIT=2000              # K线数量
ANALYSIS_RUN_ON_STARTUP=true           # 启动时立即执行

# 日志配置
LOG_LEVEL=INFO                         # 日志级别
```

## 本地开发

### 1. 安装依赖

```bash
pip install -r requirements.txt
```

### 2. 配置环境变量

```bash
cp .env.example .env
# 编辑 .env 文件，填入正确的配置
```

### 3. 启动服务

```bash
uvicorn main:app --host 0.0.0.0 --port 8000 --reload
```

### 4. 查看日志

服务启动后会输出详细的日志信息：

```
========== 分析服务启动 ==========
✓ 数据库表已创建
✓ 定时调度器已启动
→ 开始执行首次分析...
========== 开始市场分析 ==========
交易所: binance, bybit
市场类型: spot, future
时间周期: 1h, 4h
K线数量: 2000
...
```

## Docker 部署

### 1. 构建镜像

```bash
docker build -t trading-analysis:latest .
```

### 2. 使用 Docker Compose

```bash
docker-compose up -d analysis-service
```

### 3. 查看日志

```bash
docker logs -f trading-analysis
```

### 4. 健康检查

```bash
curl http://localhost:8000/health
```

## 模块说明

### config.py
配置管理模块，管理所有环境变量和配置项。

### symbol_fetcher.py
交易对获取器，从交易所 API 获取所有 USDT 交易对。

### analyzer.py
市场分析器，执行技术分析，计算支撑/压力位。

### db_manager.py
数据库管理器，管理分析结果的数据库操作。

### coordinator.py
分析协调器，协调整个市场分析流程。

### scheduler.py
定时任务调度器，管理定时分析任务的执行。

### main.py
主程序入口，FastAPI 应用和生命周期管理。

## 数据库表结构

### analysis_results

| 字段 | 类型 | 说明 |
|------|------|------|
| id | uint | 主键 |
| exchange | string | 交易所名称 |
| symbol | string | 交易对 |
| market_type | string | 市场类型 (spot/future) |
| timeframe | string | 时间周期 (1h/4h) |
| input_limit | int | K线数量 |
| support_levels | json | 支撑位数组 |
| resistance_levels | json | 压力位数组 |
| last_price | float64 | 最新价格 |
| created_at | timestamp | 创建时间 |
| updated_at | timestamp | 更新时间 |

**唯一约束**: (exchange, symbol, market_type, timeframe)

## 性能指标

- **单次完整分析**: < 30 分钟
- **分析成功率**: > 90%
- **内存使用**: < 2GB
- **并发请求**: 最多 5 个

## 监控和日志

### 日志级别

- **INFO**: 正常流程信息
- **WARNING**: 可恢复的异常
- **ERROR**: 错误但不影响整体
- **DEBUG**: 详细调试信息

### 统计信息

每次分析完成后会输出统计摘要：

```
========== 分析统计摘要 ==========
总数: 2200
成功: 2100
失败: 100
成功率: 95.45%
耗时: 1234.56 秒 (20.58 分钟)
==============================
```

## 故障排查

### 问题: 服务启动失败

**解决方案**:
1. 检查数据库连接配置
2. 确认 MySQL 服务已启动
3. 查看日志中的错误信息

### 问题: 分析失败率过高

**解决方案**:
1. 检查网络连接
2. 确认交易所 API 可访问
3. 检查是否触发限流
4. 查看错误日志详情

### 问题: 内存使用过高

**解决方案**:
1. 减少并发数（修改 semaphore 值）
2. 优化 K 线数据处理
3. 增加服务器内存

## 扩展开发

### 添加新交易所

1. 在 `config.py` 中添加交易所名称到 `valid_exchanges`
2. 确保 ccxt 库支持该交易所
3. 测试交易对获取和 K 线数据

### 添加新时间周期

1. 在 `config.py` 中添加时间周期到 `valid_timeframes`
2. 更新环境变量 `ANALYSIS_TIMEFRAMES`
3. 重启服务

### 自定义分析算法

修改 `analyzer.py` 中的 `analyze_support_resistance()` 方法。

## 许可证

MIT License

## 联系方式

如有问题，请联系开发团队。
