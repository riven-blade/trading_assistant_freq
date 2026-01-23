# Go 后端与 Python 分析服务交互分析

## 1. 系统架构概述

### 1.1 服务组成
系统由以下几个核心服务组成：

1. **Go 后端服务** (trading-assistant)
   - 端口: 8080
   - 职责: 主要业务逻辑、API 网关、前端服务
   - 技术栈: Go + Gin + GORM

2. **Python 分析服务** (analysis-service)
   - 端口: 8000
   - 职责: 市场数据分析、支撑/压力位计算
   - 技术栈: Python + FastAPI + SQLAlchemy + ccxt

3. **MySQL 数据库**
   - 端口: 3306
   - 职责: 数据持久化存储
   - 共享数据库: 两个服务共享同一个 MySQL 实例

4. **Redis 缓存**
   - 端口: 6379
   - 职责: 缓存和实时数据存储

5. **前端应用** (React)
   - 集成在 Go 服务中
   - 通过 Go API 与后端交互

### 1.2 部署架构
```
[Traefik 反向代理]
        |
        ├─> [Go 后端服务:8080] ──┐
        |                        |
        └─> [Python 分析服务:8000] (内部服务，不对外暴露)
                                  |
                    ┌─────────────┴─────────────┐
                    |                           |
              [MySQL:3306]                [Redis:6379]
```

## 2. 交互模式分析

### 2.1 **数据库共享模式** (当前实现)

#### 核心特点
- **异步解耦**: Go 和 Python 服务通过共享 MySQL 数据库进行数据交换
- **无直接 HTTP 调用**: 两个服务之间没有直接的 HTTP API 调用
- **独立运行**: 两个服务可以独立启动和停止

#### 数据流向

```
1. 分析请求流程:
   外部触发 → Python 分析服务 → 获取 K 线数据 → 计算支撑/压力位 → 写入 MySQL

2. 数据查询流程:
   前端 → Go 后端 API → 读取 MySQL → 返回分析结果 → 前端展示
```

### 2.2 数据表结构

**analysis_results 表** (共享表)

| 字段 | 类型 | 说明 |
|------|------|------|
| id | uint | 主键 |
| exchange | string | 交易所名称 (binance, bybit 等) |
| symbol | string | 交易对 (BTCUSDT) |
| market_type | string | 市场类型 (spot/future) |
| timeframe | string | 时间周期 (1h, 4h, 1d) |
| input_limit | int | 分析使用的 K 线数量 |
| support_levels | json | 支撑位数组 [price1, price2, ...] |
| resistance_levels | json | 压力位数组 [price1, price2, ...] |
| last_price | float64 | 最新价格 |
| created_at | timestamp | 创建时间 |
| updated_at | timestamp | 更新时间 |

**唯一约束**: (exchange, symbol, market_type, timeframe)

## 3. Python 分析服务详解

### 3.1 核心功能

#### 3.1.1 K 线数据获取
```python
# 分两批获取 2000 根 K 线
batch1 = 最新 1000 根
batch2 = 更早 1000 根
合并去重 → 总共约 2000 根
```

#### 3.1.2 支撑/压力位算法

**算法特点**:
1. **多窗口局部极值检测**: 使用 scipy 的 `argrelextrema` 函数
   - 窗口大小: [5, 10, 20, 30]
   - 捕获不同级别的关键价格位

2. **价格聚类**: 合并 0.5% 以内的相近价格

3. **综合评分系统**:
   - 触及次数得分 (权重 3.0)
   - 成交量得分 (权重 1.5)
   - 距离当前价格得分 (1%-10% 最优)
   - 方向正确性得分 (支撑在下，压力在上)

4. **返回 Top 5**: 按得分排序，返回最重要的 5 个支撑位和 5 个压力位

### 3.2 API 端点

#### POST /analyze
- **功能**: 提交分析任务
- **特点**: 异步执行，立即返回
- **请求体**:
```json
{
  "exchange": "binance",
  "symbol": "BTCUSDT",
  "market_type": "spot",
  "timeframe": "1h",
  "limit": 100
}
```
- **响应**:
```json
{
  "status": "accepted",
  "message": "分析任务已提交: BTCUSDT",
  "symbol": "BTCUSDT",
  "timeframe": "1h"
}
```

#### GET /results
- **功能**: 获取分析结果列表
- **参数**: symbol (可选), limit (默认 10)

#### GET /health
- **功能**: 健康检查

### 3.3 后台任务处理

```python
# 使用 FastAPI BackgroundTasks
background_tasks.add_task(perform_analysis, request, db)

# 任务执行流程:
1. 获取 K 线数据 (2000 根)
2. 计算支撑/压力位
3. 检查数据库是否存在记录
4. 存在则更新，不存在则插入
5. 提交事务
```

## 4. Go 后端服务详解

### 4.1 分析控制器 (AnalysisController)

#### GET /api/v1/analysis/results
- **功能**: 获取分析结果列表
- **参数**:
  - symbol: 交易对过滤
  - page: 页码 (默认 1)
  - pageSize: 每页数量 (默认 10, 最大 100)
- **响应**:
```json
{
  "data": [...],
  "total": 100,
  "page": 1,
  "pageSize": 10,
  "totalPages": 10
}
```

#### GET /api/v1/analysis/:id
- **功能**: 获取单个分析详情
- **用途**: 图表页面根据 ID 加载分析数据

### 4.2 数据模型

```go
type AnalysisResult struct {
    ID               uint            `json:"id"`
    Exchange         string          `json:"exchange"`
    Symbol           string          `json:"symbol"`
    MarketType       string          `json:"market_type"`
    Timeframe        string          `json:"timeframe"`
    InputLimit       int             `json:"input_limit"`
    SupportLevels    json.RawMessage `json:"support_levels"`
    ResistanceLevels json.RawMessage `json:"resistance_levels"`
    LastPrice        float64         `json:"last_price"`
    CreatedAt        time.Time       `json:"created_at"`
    UpdatedAt        time.Time       `json:"updated_at"`
}
```

## 5. 前端集成

### 5.1 分析结果页面 (Analysis.js)

**功能**:
- 展示所有分析结果
- 支持按交易对搜索
- 分页显示
- 点击"详情"跳转到图表页面

**数据流**:
```
前端 → GET /api/v1/analysis/results → Go 后端 → MySQL → 返回数据
```

### 5.2 图表页面 (ChartPage.js)

**功能**:
1. 根据 URL 参数加载分析数据
   - `?id=123&interval=1h`: 根据分析 ID 加载
   - `?symbol=BTCUSDT&interval=15m`: 根据交易对加载

2. 在图表上绘制支撑/压力位
   - 支撑位: 绿色虚线
   - 压力位: 红色虚线

3. 点击价格线创建监听
   - 点击支撑位 → 默认做多
   - 点击压力位 → 默认做空

**数据流**:
```
1. 加载分析数据:
   前端 → GET /api/v1/analysis/:id → Go 后端 → MySQL → 返回分析数据

2. 加载 K 线数据:
   前端 → GET /api/v1/klines → Go 后端 → 交易所 API → 返回 K 线

3. 绘制图表:
   使用 lightweight-charts 库绘制 K 线和价格线
```

## 6. 关键设计决策

### 6.1 为什么使用数据库共享而不是 HTTP 调用？

**优点**:
1. **解耦**: 两个服务完全独立，互不依赖
2. **异步**: Python 分析可以在后台慢慢执行，不阻塞 Go 服务
3. **容错**: 一个服务挂掉不影响另一个
4. **简单**: 不需要处理 HTTP 超时、重试等问题

**缺点**:
1. **实时性**: 数据不是实时的，需要等待 Python 写入数据库
2. **耦合**: 两个服务共享数据库 schema，修改需要同步

### 6.2 为什么 Python 服务不对外暴露？

从 docker-compose.yml 可以看到:
```yaml
analysis-service:
  labels:
    - "traefik.enable=false"  # 内部服务，不对外暴露
```

**原因**:
1. **安全**: 分析服务只需要内部使用，不需要外部访问
2. **简化**: 前端只需要对接 Go API，统一入口
3. **控制**: Go 服务可以控制何时触发分析

### 6.3 当前缺失的触发机制

**问题**: 代码中没有看到 Go 服务调用 Python 分析服务的代码

**可能的触发方式**:
1. **手动触发**: 通过外部脚本或工具直接调用 Python API
2. **定时任务**: 使用 cron 或其他调度工具定期触发
3. **待实现**: 可能计划在 Go 服务中添加触发逻辑

**建议实现**:
```go
// 在 Go 服务中添加触发分析的端点
POST /api/v1/analysis/trigger
{
  "symbol": "BTCUSDT",
  "timeframe": "1h"
}

// 内部调用 Python 服务
http.Post("http://analysis-service:8000/analyze", ...)
```

## 7. 数据一致性保证

### 7.1 唯一约束
```sql
UNIQUE KEY (exchange, symbol, market_type, timeframe)
```

**作用**:
- 防止重复分析
- 更新而不是插入新记录
- 保证每个交易对+时间周期只有一条最新记录

### 7.2 更新策略
```python
if existing_record:
    # 更新现有记录
    existing_record.support_levels = supports
    existing_record.resistance_levels = resistances
    existing_record.last_price = last_price
    existing_record.updated_at = func.now()
else:
    # 插入新记录
    new_record = AnalysisResult(...)
    db.add(new_record)
```

## 8. 性能考虑

### 8.1 Python 分析服务
- **异步执行**: 使用 BackgroundTasks 避免阻塞
- **批量获取**: 一次获取 2000 根 K 线，减少 API 调用
- **多 worker**: uvicorn --workers 2

### 8.2 Go 后端服务
- **分页查询**: 避免一次加载大量数据
- **索引优化**: 在 symbol, updated_at 等字段上建立索引

## 9. 改进建议

### 9.1 添加触发机制
在 Go 服务中添加触发 Python 分析的功能:
```go
// 新增控制器方法
func (ac *AnalysisController) TriggerAnalysis(c *gin.Context) {
    // 调用 Python 服务
    resp, err := http.Post(
        "http://analysis-service:8000/analyze",
        "application/json",
        bytes.NewBuffer(jsonData),
    )
    // 处理响应
}
```

### 9.2 添加缓存层
使用 Redis 缓存热门交易对的分析结果:
```go
// 先查 Redis
if cached, err := redis.Get(key); err == nil {
    return cached
}
// 再查 MySQL
result := db.Find(...)
// 写入 Redis
redis.Set(key, result, 5*time.Minute)
```

### 9.3 添加消息队列
使用 RabbitMQ 或 Kafka 解耦服务:
```
Go 服务 → 发送消息到队列 → Python 服务消费消息 → 执行分析 → 写入数据库
```

### 9.4 添加监控和日志
- 分析任务执行时间
- 失败率统计
- 数据更新频率监控

## 10. 总结

### 10.1 当前架构特点
- **松耦合**: 通过数据库共享数据，服务间无直接依赖
- **异步处理**: Python 分析在后台执行，不阻塞主服务
- **职责分离**: Go 负责业务逻辑，Python 负责数据分析
- **简单可靠**: 架构简单，易于维护

### 10.2 适用场景
- 分析结果不需要实时更新
- 分析任务耗时较长
- 服务需要独立部署和扩展

### 10.3 不适用场景
- 需要实时分析结果
- 需要同步返回分析数据
- 需要复杂的服务间通信
