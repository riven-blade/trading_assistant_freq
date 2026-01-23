# 内存优化指南

## 问题分析

原始版本内存使用约 **1.76GB**，主要原因：

1. **并发任务过多**: 5 个并发 × 数百个交易对 = 大量内存占用
2. **DataFrame 累积**: 每个任务持有 2000 根 K 线的 DataFrame
3. **数据库会话**: 每个任务创建独立会话
4. **无内存释放**: 分析完成后数据未及时释放

## 优化措施

### 1. 降低并发数
```python
# 从 5 降低到 3
ANALYSIS_CONCURRENCY=3
```
**效果**: 减少同时运行的任务数，降低内存峰值

### 2. 分批处理
```python
# 每批处理 20 个交易对
ANALYSIS_BATCH_SIZE=20

# 处理流程
for batch in batches:
    await process_batch(batch)
    gc.collect()  # 强制垃圾回收
```
**效果**: 避免一次性加载所有交易对数据

### 3. 及时释放内存
```python
# 分析完成后立即删除 DataFrame
del df

# 关闭数据库会话
await db_session.close()

# 清理候选列表
del support_candidates
del resistance_candidates
```
**效果**: 主动释放不再使用的对象

### 4. 批次间垃圾回收
```python
import gc
gc.collect()  # 每批次完成后强制回收
```
**效果**: 确保内存及时回收

## 配置参数

### 性能 vs 内存权衡

| 配置 | 内存占用 | 分析速度 | 推荐场景 |
|------|---------|---------|---------|
| 并发=5, 批次=50 | ~1.8GB | 快 | 内存充足 |
| 并发=3, 批次=20 | ~800MB | 中等 | **推荐** |
| 并发=2, 批次=10 | ~500MB | 慢 | 内存紧张 |
| 并发=1, 批次=5  | ~300MB | 很慢 | 极度紧张 |

### 环境变量配置

```bash
# 标准配置（推荐）
ANALYSIS_CONCURRENCY=3
ANALYSIS_BATCH_SIZE=20

# 低内存配置
ANALYSIS_CONCURRENCY=2
ANALYSIS_BATCH_SIZE=10

# 高性能配置（需要更多内存）
ANALYSIS_CONCURRENCY=5
ANALYSIS_BATCH_SIZE=50
```

## 预期效果

### 优化前
- 内存使用: **1.76GB**
- 并发数: 5
- 批次处理: 无
- 内存释放: 被动

### 优化后
- 内存使用: **~800MB** (降低 55%)
- 并发数: 3
- 批次处理: 20 个/批
- 内存释放: 主动

## 监控建议

### 1. 查看容器内存使用
```bash
docker stats trading-analysis
```

### 2. 查看 Python 进程内存
```bash
docker exec trading-analysis ps aux | grep python
```

### 3. 查看详细内存分配
```bash
docker exec trading-analysis python -c "
import psutil
import os
process = psutil.Process(os.getpid())
print(f'内存使用: {process.memory_info().rss / 1024 / 1024:.2f} MB')
"
```

## 进一步优化

### 1. 使用生成器
```python
# 不要一次性加载所有交易对
symbols = await fetch_symbols()  # ❌

# 使用生成器逐个获取
async for symbol in fetch_symbols_generator():  # ✅
    await analyze(symbol)
```

### 2. 限制 K 线数量
```python
# 如果内存仍然紧张，可以减少 K 线数量
ANALYSIS_KLINE_LIMIT=1000  # 从 2000 降到 1000
```

### 3. 使用更轻量的数据结构
```python
# 不要保留完整 DataFrame
df = await fetch_klines()
last_price = df['close'].iloc[-1]
del df  # 立即删除

# 只保留必要数据
data = {
    'last_price': last_price,
    'supports': supports,
    'resistances': resistances
}
```

### 4. 数据库连接池优化
```python
# 在 database.py 中配置
engine = create_async_engine(
    DATABASE_URL,
    pool_size=5,        # 连接池大小
    max_overflow=10,    # 最大溢出连接
    pool_recycle=3600,  # 连接回收时间
    pool_pre_ping=True  # 连接健康检查
)
```

## 故障排查

### 问题: 内存仍然很高

**检查项**:
1. 确认环境变量已生效
   ```bash
   docker exec trading-analysis env | grep ANALYSIS
   ```

2. 查看日志中的配置
   ```bash
   docker logs trading-analysis | grep "配置信息"
   ```

3. 检查是否有内存泄漏
   ```bash
   # 多次查看内存，看是否持续增长
   watch -n 5 'docker stats trading-analysis --no-stream'
   ```

### 问题: 分析速度太慢

**解决方案**:
1. 适当增加并发数
   ```bash
   ANALYSIS_CONCURRENCY=4
   ```

2. 增加批次大小
   ```bash
   ANALYSIS_BATCH_SIZE=30
   ```

3. 只分析重要交易对
   ```python
   # 过滤低交易量的交易对
   if volume < threshold:
       continue
   ```

## 总结

通过以上优化，内存使用可以从 **1.76GB 降低到 ~800MB**，降低约 **55%**。

关键优化点：
- ✅ 降低并发数（5 → 3）
- ✅ 分批处理（无 → 20/批）
- ✅ 主动释放内存
- ✅ 批次间垃圾回收

如果内存仍然紧张，可以进一步降低并发数和批次大小。
