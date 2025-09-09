package core

import (
	"context"
	"fmt"
	"time"
	"trading_assistant/models"
	"trading_assistant/pkg/exchanges/binance"
	"trading_assistant/pkg/redis"

	"github.com/sirupsen/logrus"
)

// MarketManager 市场数据管理器
type MarketManager struct {
	binanceClient *binance.Binance
	priceManager  *PriceManager
}

// NewMarketManager 创建市场数据管理器
func NewMarketManager(binanceClient *binance.Binance) *MarketManager {
	return &MarketManager{
		binanceClient: binanceClient,
		priceManager:  NewPriceManager(binanceClient),
	}
}

// StartPriceSubscriptions 启动全局markPrice订阅
func (mm *MarketManager) StartPriceSubscriptions() error {
	logrus.Info("开始启动全局markPrice订阅...")

	// 启动价格管理器
	if err := mm.priceManager.Start(); err != nil {
		return fmt.Errorf("启动价格管理器失败: %v", err)
	}

	logrus.Info("markPrice订阅启动完成")
	return nil
}

// StopPriceSubscriptions 停止全局markPrice订阅
func (mm *MarketManager) StopPriceSubscriptions() {
	if mm.priceManager != nil {
		mm.priceManager.Stop()
		logrus.Info("全局价格订阅已停止")
	}
}

// GetPriceSubscriptionStatus 获取价格订阅状态
func (mm *MarketManager) GetPriceSubscriptionStatus() map[string]interface{} {
	if mm.priceManager == nil {
		return map[string]interface{}{
			"error": "价格管理器未初始化",
		}
	}

	return mm.priceManager.GetStatus()
}

// RefreshSelectedCoinsCache 刷新价格管理器的选中币种缓存
func (mm *MarketManager) RefreshSelectedCoinsCache() {
	if mm.priceManager != nil {
		mm.priceManager.RefreshSelectedCoinsCache()
	}
}

// SyncMarketAndPriceData 同步市场数据和价格数据
func (mm *MarketManager) SyncMarketAndPriceData() error {
	logrus.Info("开始同步市场数据和价格数据...")

	if err := mm.syncMarketData(); err != nil {
		return fmt.Errorf("同步市场数据失败: %w", err)
	}

	if err := mm.syncPriceData(); err != nil {
		return fmt.Errorf("同步价格数据失败: %w", err)
	}

	logrus.Info("市场数据和价格数据同步完成")
	return nil
}

// syncMarketData 同步市场数据
func (mm *MarketManager) syncMarketData() error {
	logrus.Info("开始同步市场数据...")

	// 获取所有USDT期货交易对
	markets, err := mm.binanceClient.FetchMarkets(context.Background(), nil)
	if err != nil {
		return fmt.Errorf("获取市场数据失败: %v", err)
	}

	// 统计计数器
	var syncedCount int
	var usdtCount int
	validSymbols := make(map[string]bool) // 记录有效的symbol

	for i := range markets {
		market := markets[i]
		// 只处理活跃的USDT永续合约
		if !market.Active || market.Quote != "USDT" || !market.Swap {
			logrus.Debugf("跳过非永续合约: %s (Active: %v, Quote: %s, Swap: %v)",
				market.ID, market.Active, market.Quote, market.Swap)
			continue
		}

		usdtCount++

		// 使用MarketID作为有效标识符
		validSymbols[market.ID] = true

		// 创建币种信息（统一使用MarketID）
		coin := &models.Coin{
			Symbol:     market.ID, // 统一使用MarketID: BTCUSDT
			MarketID:   market.ID, // binance原始ID: BTCUSDT
			BaseAsset:  market.Base,
			QuoteAsset: market.Quote,
			Status:     "active",
			TickSize:   fmt.Sprintf("%.8f", market.Limits.Price.Step),
			StepSize:   fmt.Sprintf("%.8f", market.Limits.Amount.Step),
			MinPrice:   fmt.Sprintf("%.8f", market.Limits.Price.Min),
			MaxPrice:   fmt.Sprintf("%.8f", market.Limits.Price.Max),
			MinQty:     fmt.Sprintf("%.8f", market.Limits.Amount.Min),
			MaxQty:     fmt.Sprintf("%.8f", market.Limits.Amount.Max),
			CreatedAt:  time.Now(),
			UpdatedAt:  time.Now(),
		}

		// 计算并设置正确的精度值
		coin.PricePrecision = coin.GetPricePrecisionFromTickSize()
		coin.QuantityPrecision = coin.GetQuantityPrecisionFromStepSize()

		logrus.WithFields(logrus.Fields{
			"symbol":             coin.Symbol,
			"tick_size":          coin.TickSize,
			"price_precision":    coin.PricePrecision,
			"step_size":          coin.StepSize,
			"quantity_precision": coin.QuantityPrecision,
		}).Debug("币种精度计算完成")

		// 保存到Redis
		if err := redis.GlobalRedisClient.SetCoin(coin); err != nil {
			logrus.Errorf("保存币种 %s 失败: %v", market.ID, err)
			continue
		}

		syncedCount++
	}

	if err := mm.cleanupInvalidCoins(validSymbols); err != nil {
		logrus.Warnf("清理无效币种失败: %v", err)
	}

	logrus.WithFields(logrus.Fields{
		"total_markets": len(markets),
		"usdt_markets":  usdtCount,
		"synced_count":  syncedCount,
	}).Info("市场数据同步完成")

	return nil
}

// cleanupInvalidCoins 清理不再有效的币种
func (mm *MarketManager) cleanupInvalidCoins(validSymbols map[string]bool) error {
	// 获取所有现有币种
	existingCoins, err := redis.GlobalRedisClient.GetAllCoins()
	if err != nil {
		return err
	}

	var deletedCount int
	for _, coin := range existingCoins {
		if !validSymbols[coin.Symbol] {
			// 这个币种不再有效，删除它
			if err := redis.GlobalRedisClient.DeleteCoin(coin.Symbol); err != nil {
				logrus.Errorf("删除无效币种 %s 失败: %v", coin.Symbol, err)
			} else {
				deletedCount++
			}
		}
	}

	if deletedCount > 0 {
		logrus.WithFields(logrus.Fields{
			"deleted_count": deletedCount,
		}).Info("清理无效币种完成")
	}

	return nil
}

// syncPriceData 同步价格数据
func (mm *MarketManager) syncPriceData() error {
	logrus.Info("开始同步价格数据...")

	// 获取所有币种列表
	coins, err := redis.GlobalRedisClient.GetAllCoins()
	if err != nil {
		return fmt.Errorf("获取币种列表失败: %v", err)
	}

	if len(coins) == 0 {
		logrus.Warn("没有找到币种数据，请先初始化市场数据")
		return nil
	}

	// 提取所有MarketID用于API调用，构建MarketID到Coin的映射
	var symbols []string
	marketIDMap := make(map[string]*models.Coin) // MarketID -> Coin的映射

	for i := range coins {
		coin := coins[i]
		symbols = append(symbols, coin.MarketID)
		marketIDMap[coin.MarketID] = coin
	}

	logrus.WithFields(logrus.Fields{
		"total_symbols": len(symbols),
	}).Info("开始批量获取ticker数据...")

	if len(symbols) != len(marketIDMap) {
		logrus.Warnf("symbols和marketIDMap数量不一致: symbols=%d, marketIDMap=%d", len(symbols), len(marketIDMap))
	}

	tickers, err := mm.binanceClient.FetchTickers(context.Background(), symbols, nil)
	if err != nil {
		logrus.Errorf("批量获取ticker数据失败: %v", err)
		return fmt.Errorf("批量获取ticker数据失败: %v", err)
	}

	logrus.WithFields(logrus.Fields{
		"received_tickers":  len(tickers),
		"requested_symbols": len(symbols),
		"coins_from_redis":  len(coins),
	}).Info("ticker数据获取完成")

	// 更新币种价格信息
	var successCount, errorCount int
	now := time.Now()

	// 遍历返回的ticker数据，通过MarketID映射到coin
	for marketID, ticker := range tickers {
		coin, exists := marketIDMap[marketID]
		if !exists {
			// 这个MarketID不在我们的监控列表中，跳过
			continue
		}

		// 更新币种的价格和交易信息（从 ticker 数据获取）
		coin.Price = fmt.Sprintf("%.8f", ticker.Last)
		coin.PriceChange = fmt.Sprintf("%.8f", ticker.Change)
		coin.PriceChangePercent = fmt.Sprintf("%.2f", ticker.Percentage)
		coin.Volume = fmt.Sprintf("%.8f", ticker.BaseVolume)
		coin.QuoteVolume = fmt.Sprintf("%.8f", ticker.QuoteVolume)
		coin.UpdatedAt = now

		// 确保精度信息仍然正确（防止被覆盖）
		if coin.PricePrecision == 0 {
			coin.PricePrecision = coin.GetPricePrecisionFromTickSize()
		}
		if coin.QuantityPrecision == 0 {
			coin.QuantityPrecision = coin.GetQuantityPrecisionFromStepSize()
		}

		// 保存更新后的币种信息
		if err := redis.GlobalRedisClient.SetCoin(coin); err != nil {
			logrus.Errorf("保存 %s 价格数据失败: %v", coin.Symbol, err)
			errorCount++
			continue
		}

		successCount++
	}

	logrus.WithFields(logrus.Fields{
		"total_coins":   len(coins),
		"success_count": successCount,
		"error_count":   errorCount,
		"api_requests":  1, // 只用了1次API请求
	}).Info("价格数据同步完成")

	return nil
}
