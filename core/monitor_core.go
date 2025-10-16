package core

import (
	"fmt"
	"time"
	"trading_assistant/models"
	"trading_assistant/pkg/config"
	"trading_assistant/pkg/exchanges/types"
	"trading_assistant/pkg/freqtrade"
	"trading_assistant/pkg/redis"
	"trading_assistant/pkg/utils"

	"github.com/sirupsen/logrus"
)

type PriceMonitor struct {
	running       bool
	stopChan      chan bool
	tickInterval  time.Duration
	orderExecutor *OrderExecutor
}

var GlobalPriceMonitor *PriceMonitor

// InitPriceMonitor 初始化价格监控器
func InitPriceMonitor(freqtradeClient *freqtrade.Controller) {
	GlobalPriceMonitor = &PriceMonitor{
		running:       false,
		stopChan:      make(chan bool),
		tickInterval:  500 * time.Millisecond, // 0.5秒检查一次，更快响应价格变化
		orderExecutor: NewOrderExecutor(freqtradeClient),
	}
}

// Start 开始价格监控
func (pm *PriceMonitor) Start() {
	if pm.running {
		logrus.Warn("price monitor is already running")
		return
	}

	pm.running = true
	logrus.Info("price monitor started")

	go pm.monitorLoop()
}

// Stop 停止价格监控
func (pm *PriceMonitor) Stop() {
	if !pm.running {
		return
	}

	pm.running = false
	pm.stopChan <- true
	logrus.Info("价格监控已停止")
}

// IsRunning 检查是否在运行
func (pm *PriceMonitor) IsRunning() bool {
	return pm.running
}

// monitorLoop 监控循环
func (pm *PriceMonitor) monitorLoop() {
	ticker := time.NewTicker(pm.tickInterval)
	defer ticker.Stop()

	for {
		select {
		case <-pm.stopChan:
			return
		case <-ticker.C:
			pm.checkPriceTargets()
		}
	}
}

// checkPriceTargets 检查价格目标
func (pm *PriceMonitor) checkPriceTargets() {
	// 获取所有待处理的价格预估
	estimates, err := redis.GlobalRedisClient.GetActiveEstimates()
	if err != nil {
		logrus.Errorf("获取价格预估失败: %v", err)
		return
	}

	if len(estimates) == 0 {
		return
	}

	logrus.Debugf("检查 %d 个价格预估", len(estimates))

	for i := range estimates {
		estimate := estimates[i]
		pm.checkSingleEstimate(estimate)
	}
}

// checkSingleEstimate 检查单个价格预估
func (pm *PriceMonitor) checkSingleEstimate(estimate *models.PriceEstimate) {
	// 获取价格数据 (estimate.Symbol现在存储的就是MarketID)
	markPriceData, err := redis.GlobalRedisClient.GetMarkPrice(estimate.Symbol)
	if err != nil {
		logrus.Debugf("未找到 %s 的价格数据", estimate.Symbol)
		return
	}

	if markPriceData == nil {
		logrus.Debugf("价格数据为空 %s", estimate.Symbol)
		return
	}

	// 根据交易方向选择合适的实时价格
	// long（做多）- 需要买入，使用卖价（askPrice）
	// short（做空）- 需要卖出，使用买价（bidPrice）
	var currentPrice float64
	switch estimate.Side {
	case types.PositionSideLong:
		currentPrice = markPriceData.AskPrice // 做多使用卖价（买入时的成本）
		if currentPrice <= 0 {
			// 降级到标记价格
			currentPrice = markPriceData.MarkPrice
			logrus.Debugf("%s 卖价无效，降级使用标记价格: %f", estimate.Symbol, currentPrice)
		}
	case types.PositionSideShort:
		currentPrice = markPriceData.BidPrice // 做空使用买价（卖出时的价格）
		if currentPrice <= 0 {
			// 降级到标记价格
			currentPrice = markPriceData.MarkPrice
			logrus.Debugf("%s 买价无效，降级使用标记价格: %f", estimate.Symbol, currentPrice)
		}
	}

	if currentPrice <= 0 {
		logrus.Errorf("无效的价格 %s: bid=%f, ask=%f, mark=%f",
			estimate.Symbol, markPriceData.BidPrice, markPriceData.AskPrice, markPriceData.MarkPrice)
		return
	}

	// 根据操作类型和交易方向判断触发条件
	actionType := estimate.ActionType
	triggerType := estimate.TriggerType

	// 使用实时买卖价判断触发
	var shouldTrigger bool
	switch estimate.Side {
	case types.PositionSideLong:
		shouldTrigger = shouldTriggerLong(actionType, triggerType, currentPrice, estimate.TargetPrice)
	case types.PositionSideShort:
		shouldTrigger = shouldTriggerShort(actionType, triggerType, currentPrice, estimate.TargetPrice)
	default:
		logrus.Errorf("无效的交易方向: %s", estimate.Side)
		return
	}

	if shouldTrigger {
		// 根据交易方向确定价格类型描述
		var priceType string
		switch estimate.Side {
		case types.PositionSideLong:
			priceType = "卖价(ask)"
		case types.PositionSideShort:
			priceType = "买价(bid)"
		default:
			priceType = "未知价格"
		}

		logrus.Infof("价格目标触发: %s %s %s, 当前%s: %f, 目标价格: %f",
			estimate.Symbol, estimate.Side, actionType, priceType, currentPrice, estimate.TargetPrice)

		// 对于做空场景，检查资金费率
		if estimate.Side == types.PositionSideShort {
			if !pm.checkFundingRateForShort(estimate, markPriceData) {
				return
			}
		}

		pm.triggerEstimate(estimate, currentPrice)
	}
}

// triggerEstimate 触发价格预估
func (pm *PriceMonitor) triggerEstimate(estimate *models.PriceEstimate, currentPrice float64) {
	// 执行自动下单
	err := pm.orderExecutor.ExecuteOrder(estimate, currentPrice)
	if err != nil {
		logrus.Errorf("订单执行失败: %v", err)

		// 记录错误信息到日志
		actionText := getActionText(estimate.ActionType)
		positionText := getPositionText(estimate.Side)
		logrus.Errorf("订单执行失败: %s %s %s, 比例: %.2f%%, 目标价: %.4f, 当前价: %.6f, 错误: %v",
			estimate.Symbol, actionText, positionText, estimate.Percentage, estimate.TargetPrice, currentPrice, err)

		// 更新预估状态为失败，并保存错误信息
		estimate.Status = models.EstimateStatusFailed
		estimate.ErrorMessage = err.Error() // 保存失败原因
	} else {
		// 更新预估状态为已触发，清空错误信息
		estimate.Status = models.EstimateStatusTriggered
		estimate.ErrorMessage = "" // 清空之前的错误信息（如果有）
	}

	estimate.UpdatedAt = time.Now()
	err = redis.GlobalRedisClient.SetPriceEstimate(estimate)
	if err != nil {
		logrus.Errorf("更新价格预估状态失败: %v", err)
		return
	}

	// 广播价格预估更新
	go utils.BroadcastSymbolEstimatesUpdate()
}

// getActionText 获取操作类型的中文描述
func getActionText(actionType string) string {
	switch actionType {
	case models.ActionTypeOpen:
		return "开仓"
	case models.ActionTypeAddition:
		return "加仓"
	case models.ActionTypeTakeProfit:
		return "止盈"
	default:
		return "交易"
	}
}

// getPositionText 获取仓位方向的中文描述
func getPositionText(side string) string {
	switch side {
	case types.PositionSideLong:
		return "做多"
	case types.PositionSideShort:
		return "做空"
	default:
		return "未知"
	}
}

// checkFundingRateForShort 检查做空时的资金费率
func (pm *PriceMonitor) checkFundingRateForShort(estimate *models.PriceEstimate, markPriceData *types.WatchMarkPrice) bool {
	// 获取配置中的资金费率阈值
	threshold := config.GlobalConfig.ShortFundingRateThreshold
	currentFundingRate := markPriceData.FundingRate

	// 如果资金费率小于阈值
	if currentFundingRate < threshold {
		logrus.Warnf("做空触发失败: %s 资金费率 %f < 阈值 %f，不允许开空仓",
			estimate.Symbol, currentFundingRate, threshold)

		// 构建错误信息
		errorMsg := fmt.Sprintf("资金费率过低: 当前%.4f%% < 阈值%.4f%%，不允许开空仓",
			currentFundingRate*100, threshold*100)

		// 更新预估状态为失败，并保存错误信息
		estimate.Status = models.EstimateStatusFailed
		estimate.ErrorMessage = errorMsg
		estimate.UpdatedAt = time.Now()
		err := redis.GlobalRedisClient.SetPriceEstimate(estimate)
		if err != nil {
			logrus.Errorf("更新价格预估状态失败: %v", err)
		}

		// 记录资金费率检查失败信息到日志
		actionText := getActionText(estimate.ActionType)
		logrus.Warnf("做空触发失败 - 资金费率检查: 交易对=%s, 操作=%s, 当前资金费率=%.4f%%, 阈值=%.4f%%, 原因=资金费率过低",
			estimate.Symbol, actionText, currentFundingRate*100, threshold*100)

		// 通过WebSocket广播失败事件
		go pm.broadcastFundingRateFailEvent(estimate, currentFundingRate, threshold)

		// 广播预估更新
		go utils.BroadcastSymbolEstimatesUpdate()

		return false
	}

	logrus.Debugf("做空资金费率检查通过: %s 资金费率 %f >= 阈值 %f",
		estimate.Symbol, currentFundingRate, threshold)
	return true
}

// broadcastFundingRateFailEvent 记录资金费率检查失败事件
func (pm *PriceMonitor) broadcastFundingRateFailEvent(estimate *models.PriceEstimate, currentFundingRate, threshold float64) {
	// 记录资金费率检查失败事件
	logrus.Infof("资金费率检查失败事件: %s 资金费率 %.4f%%",
		estimate.Symbol, currentFundingRate*100)
}
