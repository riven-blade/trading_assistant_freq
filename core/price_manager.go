package core

import (
	"context"
	"fmt"
	"strconv"
	"time"
	"trading_assistant/pkg/config"
	"trading_assistant/pkg/exchange_factory"
	"trading_assistant/pkg/exchanges/types"
	"trading_assistant/pkg/redis"
	"trading_assistant/pkg/websocket"

	"github.com/sirupsen/logrus"
)

// PriceManager REST API 定时价格管理器
type PriceManager struct {
	exchangeClient exchange_factory.ExchangeInterface
	ctx            context.Context
	cancel         context.CancelFunc
	isRunning      bool
	ticker         *time.Ticker  // 定时器
	startTime      time.Time     // 启动时间
	lastFetchTime  time.Time     // 最后获取时间
	fetchCount     int64         // 获取次数
	updateInterval time.Duration // 更新间隔
}

// NewPriceManager 创建价格管理器
func NewPriceManager(exchangeClient exchange_factory.ExchangeInterface) *PriceManager {
	ctx, cancel := context.WithCancel(context.Background())

	return &PriceManager{
		exchangeClient: exchangeClient,
		ctx:            ctx,
		cancel:         cancel,
		updateInterval: config.GlobalConfig.PriceUpdateInterval,
	}
}

// Start 启动定时价格获取
func (pm *PriceManager) Start() error {
	if pm.isRunning {
		return fmt.Errorf("价格管理器已在运行")
	}

	pm.isRunning = true
	pm.startTime = time.Now()
	pm.fetchCount = 0

	// 立即获取一次价格数据
	go pm.fetchPricesOnce()

	// 启动定时器
	pm.ticker = time.NewTicker(pm.updateInterval)
	go pm.run()

	logrus.Infof("价格管理器已启动，更新间隔: %v", pm.updateInterval)
	return nil
}

// Stop 停止定时价格获取
func (pm *PriceManager) Stop() {
	if !pm.isRunning {
		return
	}

	logrus.Info("停止价格管理器...")

	pm.cancel()
	pm.isRunning = false

	// 停止定时器
	if pm.ticker != nil {
		pm.ticker.Stop()
		pm.ticker = nil
	}

	logrus.Info("价格管理器已停止")
}

// IsRunning 检查管理器是否在运行
func (pm *PriceManager) IsRunning() bool {
	return pm.isRunning
}

// GetStatus 获取管理器状态信息
func (pm *PriceManager) GetStatus() map[string]interface{} {
	return map[string]interface{}{
		"running":         pm.isRunning,
		"start_time":      pm.startTime.Unix(),
		"last_fetch_time": pm.lastFetchTime.Unix(),
		"fetch_count":     pm.fetchCount,
		"update_interval": pm.updateInterval.String(),
		"mode":            "rest_api_timer",
		"exchange":        pm.exchangeClient.GetName(),
	}
}

// run 主运行循环
func (pm *PriceManager) run() {
	defer func() {
		if r := recover(); r != nil {
			logrus.Errorf("价格管理器运行时发生异常: %v", r)
		}
	}()

	for {
		select {
		case <-pm.ctx.Done():
			logrus.Info("价格管理器收到停止信号")
			return
		case <-pm.ticker.C:
			pm.fetchPricesOnce()
		}
	}
}

// fetchPricesOnce 执行一次价格获取
func (pm *PriceManager) fetchPricesOnce() {
	defer func() {
		if r := recover(); r != nil {
			logrus.Errorf("获取价格数据时发生异常: %v", r)
		}
	}()

	startTime := time.Now()
	pm.fetchCount++

	// 直接从Redis获取选中的币种
	selectedSymbols, err := redis.GlobalRedisClient.GetSelectedCoinMarketIDs()
	if err != nil {
		logrus.Errorf("获取选中币种列表失败: %v", err)
		return
	}

	if len(selectedSymbols) == 0 {
		logrus.Debug("没有选中的币种，跳过价格获取")
		return
	}

	// 获取实时买卖价（bookTicker）和资金费率（premiumIndex）
	ctx, cancel := context.WithTimeout(pm.ctx, 10*time.Second)
	defer cancel()

	// 1. 获取实时BookTicker数据（只包含bid/ask价格，权重更低）
	tickers, err := pm.exchangeClient.FetchBookTickers(ctx, selectedSymbols, nil)
	if err != nil {
		logrus.Errorf("获取BookTicker数据失败: %v", err)
		return
	}

	// 2. 获取资金费率数据
	markPrices, err := pm.exchangeClient.FetchMarkPrices(ctx, selectedSymbols)
	if err != nil {
		logrus.Errorf("获取标记价格失败: %v", err)
		return
	}

	pm.lastFetchTime = time.Now()
	processedCount := 0
	pricesData := make(map[string]interface{}) // 用于广播的价格数据

	// 3. 合并两个数据源
	for _, symbol := range selectedSymbols {
		ticker := tickers[symbol]
		markPrice := markPrices[symbol]

		// 确保至少有一个数据源有效
		if ticker == nil && markPrice == nil {
			continue
		}

		// 构建完整的价格数据结构
		watchMarkPrice := &types.WatchMarkPrice{
			Symbol:    symbol,
			TimeStamp: time.Now().UnixMilli(),
		}

		// 从 Ticker 获取实时买卖价（优先使用）
		if ticker != nil {
			watchMarkPrice.BidPrice = ticker.Bid // 最优买价（实时）
			watchMarkPrice.AskPrice = ticker.Ask // 最优卖价（实时）
		}

		// 从 MarkPrice 获取资金费率等信息
		if markPrice != nil {
			watchMarkPrice.MarkPrice = markPrice.MarkPrice         // 标记价格（作为参考）
			watchMarkPrice.IndexPrice = markPrice.IndexPrice       // 指数价格
			watchMarkPrice.FundingRate = markPrice.FundingRate     // 资金费率
			watchMarkPrice.FundingTime = markPrice.NextFundingTime // 下次资金费时间
		}

		// 如果没有bid/ask，降级使用标记价格
		if watchMarkPrice.BidPrice <= 0 && watchMarkPrice.MarkPrice > 0 {
			watchMarkPrice.BidPrice = watchMarkPrice.MarkPrice
		}
		if watchMarkPrice.AskPrice <= 0 && watchMarkPrice.MarkPrice > 0 {
			watchMarkPrice.AskPrice = watchMarkPrice.MarkPrice
		}

		// 验证数据有效性
		if watchMarkPrice.BidPrice <= 0 || watchMarkPrice.AskPrice <= 0 {
			logrus.Warnf("跳过 %s: 买卖价无效 (bid=%f, ask=%f)", symbol, watchMarkPrice.BidPrice, watchMarkPrice.AskPrice)
			continue
		}

		// 保存到Redis缓存
		if err := pm.saveToCache(watchMarkPrice); err != nil {
			logrus.Errorf("保存 %s 价格数据到缓存失败: %v", symbol, err)
		}

		// 获取价格变化信息用于广播
		priceChange := 0.0
		priceChangePercent := 0.0
		if coin, err := redis.GlobalRedisClient.GetCoin(symbol); err == nil {
			if change, parseErr := strconv.ParseFloat(coin.PriceChange, 64); parseErr == nil {
				priceChange = change
			}
			if changePercent, parseErr := strconv.ParseFloat(coin.PriceChangePercent, 64); parseErr == nil {
				priceChangePercent = changePercent
			}
		}

		// 构建广播数据（包含实时买卖价）
		pricesData[symbol] = map[string]interface{}{
			"symbol":             symbol,
			"bidPrice":           watchMarkPrice.BidPrice,    // 实时买价
			"askPrice":           watchMarkPrice.AskPrice,    // 实时卖价
			"markPrice":          watchMarkPrice.MarkPrice,   // 标记价格（参考）
			"indexPrice":         watchMarkPrice.IndexPrice,  // 指数价格
			"fundingRate":        watchMarkPrice.FundingRate, // 资金费率
			"fundingTime":        watchMarkPrice.FundingTime, // 下次资金费时间
			"updateTime":         watchMarkPrice.TimeStamp,   // 更新时间
			"priceChange":        priceChange,
			"priceChangePercent": priceChangePercent,
		}

		processedCount++
	}

	duration := time.Since(startTime)
	logrus.Debugf("获取价格完成: %d/%d 个币种，耗时: %v", processedCount, len(selectedSymbols), duration)

	// 直接广播已获取的价格数据给前端
	if processedCount > 0 {
		go pm.broadcastPrices(pricesData)
	}

	// 每100次获取记录一次统计日志
	if pm.fetchCount%100 == 0 {
		logrus.Infof("价格获取统计: 总次数=%d, 平均处理币种数=%d, 运行时间=%v",
			pm.fetchCount, processedCount, time.Since(pm.startTime))
	}
}

// saveToCache 保存价格数据到Redis缓存
func (pm *PriceManager) saveToCache(markPrice *types.WatchMarkPrice) error {
	if redis.GlobalRedisClient == nil {
		return fmt.Errorf("redis客户端未初始化")
	}

	return redis.GlobalRedisClient.SetMarkPrice(markPrice)
}

// broadcastPrices 广播价格数据给前端
func (pm *PriceManager) broadcastPrices(pricesData map[string]interface{}) {
	wsManager := websocket.GetGlobalWebSocketManager()
	if wsManager == nil {
		logrus.Debug("WebSocket管理器未初始化")
		return
	}

	wsManager.BroadcastPrices(pricesData)
	logrus.Debugf("通过WebSocket广播价格数据，包含 %d 个币种", len(pricesData))
}
