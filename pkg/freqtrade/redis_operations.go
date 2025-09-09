package freqtrade

import (
	"strings"
	"trading_assistant/models"

	"github.com/sirupsen/logrus"
)

// SetWatchedPairs 设置观察的交易对列表
func (fc *Controller) SetWatchedPairs(pairs []string) error {
	if fc.redisClient == nil {
		logrus.Error("Redis客户端未初始化")
		return nil // 不阻止程序运行
	}

	logrus.WithFields(logrus.Fields{
		"raw_pairs": pairs,
	}).Debug("收到 freqtrade 白名单")

	// 清除所有现有的选择状态（包括旧格式）
	existingSelections, err := fc.redisClient.GetAllCoinSelections()
	if err != nil {
		logrus.Warnf("获取现有币种选择状态失败: %v", err)
	} else {
		for i := range existingSelections {
			selection := existingSelections[i]
			if selection.Status == models.CoinSelectionActive {
				err := fc.redisClient.SetCoinSelection(selection.Symbol, models.CoinSelectionInactive)
				if err != nil {
					logrus.Warnf("取消选择币种 %s 失败: %v", selection.Symbol, err)
				}
			}
		}
	}

	// 将白名单中的币种设置为选中状态
	successCount := 0
	for i := range pairs {
		pair := pairs[i]

		// 从期货格式转换为MarketID格式
		marketID := fc.convertToMarketID(pair)
		if marketID == "" {
			logrus.Warnf("无法转换币种格式，跳过: %s", pair)
			continue
		}

		// 对于coin_selection，使用MarketID格式保持一致性
		err = fc.redisClient.SetCoinSelection(marketID, models.CoinSelectionActive)
		if err != nil {
			logrus.Warnf("选择币种 %s (MarketID: %s) 失败: %v", pair, marketID, err)
			continue
		}
		successCount++
	}

	logrus.WithFields(logrus.Fields{
		"total_pairs":   len(pairs),
		"success_count": successCount,
		"failed_count":  len(pairs) - successCount,
	}).Info("Freqtrade 白名单已同步到币种选择系统")

	return nil
}

// convertToMarketID 将期货格式转换为MarketID格式
func (fc *Controller) convertToMarketID(futuresSymbol string) string {
	result := strings.ReplaceAll(futuresSymbol, "/", "")
	result = strings.ReplaceAll(result, ":", "")

	if strings.HasSuffix(result, "USDTUSDT") {
		result = strings.TrimSuffix(result, "USDT")
	}

	return result
}
