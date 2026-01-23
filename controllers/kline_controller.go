package controllers

import (
	"fmt"
	"net/http"
	"strconv"
	"trading_assistant/pkg/exchange_factory"
	"trading_assistant/pkg/exchanges/types"
	"trading_assistant/pkg/redis"

	"strings"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type KlineController struct {
	exchangeClient exchange_factory.ExchangeInterface
}

// NewKlineController 创建K线控制器
func NewKlineController(exchangeClient exchange_factory.ExchangeInterface) *KlineController {
	return &KlineController{
		exchangeClient: exchangeClient,
	}
}

// GetKlines 获取K线数据
func (k *KlineController) GetKlines(ctx *gin.Context) {
	if k.exchangeClient == nil {
		ctx.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "交易所客户端未初始化",
		})
		return
	}

	// 获取参数
	symbol := ctx.Query("symbol")
	if symbol == "" {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "symbol参数不能为空",
		})
		return
	}
	
	// 规范化symbol格式：移除斜杠 (BTC/USDT -> BTCUSDT)
	// Binance API需要无斜杠的格式，但前端可能传递带斜杠的格式
	symbol = strings.ReplaceAll(symbol, "/", "")
	
	// 检查symbol是否包含非ASCII字符（如中文）
	// 如果包含，需要从数据库查询对应的market_id
	hasNonASCII := false
	for _, r := range symbol {
		if r > 127 {
			hasNonASCII = true
			break
		}
	}
	
	if hasNonASCII {
		// 从数据库查询对应的market_id
		marketID, err := k.getMarketIDFromSymbol(symbol)
		if err != nil {
			logrus.Errorf("无法找到symbol对应的market_id: %s, error: %v", symbol, err)
			ctx.JSON(http.StatusBadRequest, gin.H{
				"error": fmt.Sprintf("无法识别的交易对: %s", symbol),
			})
			return
		}
		logrus.Infof("将symbol %s 转换为 market_id %s", symbol, marketID)
		symbol = marketID
	}

	interval := ctx.DefaultQuery("interval", "5m")
	limitStr := ctx.DefaultQuery("limit", "1000")

	limit, err := strconv.Atoi(limitStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "limit参数格式错误",
		})
		return
	}

	// 获取可选参数
	var since int64
	if sinceStr := ctx.Query("since"); sinceStr != "" {
		if parsed, err := strconv.ParseInt(sinceStr, 10, 64); err == nil {
			since = parsed
		}
	}

	// 构建缓存键
	cacheKey := fmt.Sprintf("%s:%s:%s:%d:%d", redis.CacheKeyKLines, symbol, interval, limit, since)

	// 检查Redis缓存
	var cachedKlines []*types.Kline
	if redis.GlobalRedisClient != nil {
		if err := redis.GlobalRedisClient.GetCache(cacheKey, &cachedKlines); err == nil {
			logrus.Debugf("从缓存获取K线数据: %s", cacheKey)
			ctx.JSON(http.StatusOK, gin.H{
				"success": true,
				"data":    cachedKlines,
				"count":   len(cachedKlines),
				"cached":  true,
				"source":  "cache",
				"params": gin.H{
					"symbol":   symbol,
					"interval": interval,
					"limit":    limit,
					"since":    since,
				},
			})
			return
		}
	}

	logrus.Infof("缓存中无K线数据，实时获取: symbol=%s, interval=%s, limit=%d, since=%d", symbol, interval, limit, since)

	// 从Binance获取K线数据
	klines, err := k.exchangeClient.FetchKlines(ctx.Request.Context(), symbol, interval, since, limit, nil)
	if err != nil {
		logrus.Errorf("获取K线数据失败: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error":   "获取K线数据失败",
			"details": err.Error(),
		})
		return
	}

	// 缓存K线数据
	if redis.GlobalRedisClient != nil && len(klines) > 0 {
		if err := redis.GlobalRedisClient.SetCache(cacheKey, klines); err != nil {
			logrus.Errorf("缓存K线数据失败: %v", err)
		} else {
			logrus.Debugf("已缓存K线数据5分钟: %s", cacheKey)
		}
	} else if len(klines) == 0 {
		logrus.Warnf("K线数据为空，不进行缓存: symbol=%s, interval=%s", symbol, interval)
	}

	logrus.Infof("成功获取K线数据: %d条", len(klines))

	// 返回K线数据
	ctx.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    klines,
		"count":   len(klines),
		"cached":  false,
		"source":  "real_time",
		"params": gin.H{
			"symbol":   symbol,
			"interval": interval,
			"limit":    limit,
			"since":    since,
		},
	})
}

// getMarketIDFromSymbol 从数据库查询symbol对应的market_id
func (k *KlineController) getMarketIDFromSymbol(symbol string) (string, error) {
	// 这里需要查询coin表，根据symbol模糊匹配找到对应的market_id
	// 暂时返回错误，需要实现数据库查询逻辑
	return "", fmt.Errorf("symbol包含非ASCII字符，需要从数据库查询market_id")
}
