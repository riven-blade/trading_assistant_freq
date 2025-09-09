package core

import (
	"fmt"
	"time"
	"trading_assistant/models"
	"trading_assistant/pkg/config"
	"trading_assistant/pkg/exchanges/types"
	"trading_assistant/pkg/freqtrade"
	"trading_assistant/pkg/redis"
	"trading_assistant/pkg/telegram"
	"trading_assistant/pkg/utils"
	"trading_assistant/pkg/websocket"

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
		tickInterval:  1 * time.Second,
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

	// 发送Telegram通知
	if telegram.GlobalTelegramClient != nil {
		err := telegram.GlobalTelegramClient.SendMessage("监控停止")
		if err != nil {
			logrus.Errorf("发送Telegram通知失败: %v", err)
		}
	}
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
	// 获取标记价格 (estimate.Symbol现在存储的就是MarketID)
	markPriceData, err := redis.GlobalRedisClient.GetMarkPrice(estimate.Symbol)
	if err != nil {
		logrus.Debugf("未找到 %s 的标记价格数据", estimate.Symbol)
		return
	}

	if markPriceData == nil {
		logrus.Debugf("标记价格数据为空 %s", estimate.Symbol)
		return
	}

	// 使用标记价格作为当前价格
	currentPrice := markPriceData.MarkPrice
	if currentPrice <= 0 {
		logrus.Errorf("无效的标记价格 %s: %f", estimate.Symbol, currentPrice)
		return
	}

	// 根据操作类型和交易方向判断触发条件
	actionType := estimate.ActionType
	triggerType := estimate.TriggerType

	// 统一使用markPrice
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
		logrus.Infof("价格目标触发: %s %s %s, 当前标记价格: %f, 目标价格: %f",
			estimate.Symbol, estimate.Side, actionType, currentPrice, estimate.TargetPrice)

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

		// 发送错误通知，包含详细的交易信息
		if telegram.GlobalTelegramClient != nil {
			// 构建详细的错误消息
			actionText := getActionText(estimate.ActionType)
			positionText := getPositionText(estimate.Side)

			errorMessage := fmt.Sprintf("%s %s %s\n比例: %.2f%%\n目标价: %.4f\n当前价: %.6f",
				estimate.Symbol, actionText, positionText,
				estimate.Percentage, estimate.TargetPrice, currentPrice)

			telegram.GlobalTelegramClient.SendError(errorMessage, err)
		}

		// 更新预估状态为失败
		estimate.Status = models.EstimateStatusFailed
	} else {
		// 更新预估状态为已触发
		estimate.Status = models.EstimateStatusTriggered
	}

	estimate.UpdatedAt = time.Now()
	err = redis.GlobalRedisClient.SetPriceEstimate(estimate)
	if err != nil {
		logrus.Errorf("更新价格预估状态失败: %v", err)
		return
	}

	// 通过WebSocket广播价格预估更新
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

		// 更新预估状态为失败
		estimate.Status = models.EstimateStatusFailed
		estimate.UpdatedAt = time.Now()
		err := redis.GlobalRedisClient.SetPriceEstimate(estimate)
		if err != nil {
			logrus.Errorf("更新价格预估状态失败: %v", err)
		}

		// 发送Telegram通知
		if telegram.GlobalTelegramClient != nil {
			actionText := getActionText(estimate.ActionType)
			message := fmt.Sprintf("做空触发失败 - 资金费率检查\n交易对: %s\n操作: %s\n当前资金费率: %.4f%%\n阈值: %.4f%%\n原因: 资金费率过低，不允许开空仓",
				estimate.Symbol, actionText, currentFundingRate*100, threshold*100)
			err := telegram.GlobalTelegramClient.SendMessage(message)
			if err != nil {
				logrus.Errorf("发送Telegram通知失败: %v", err)
			}
		}

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

// broadcastFundingRateFailEvent 广播资金费率检查失败事件到WebSocket客户端
func (pm *PriceMonitor) broadcastFundingRateFailEvent(estimate *models.PriceEstimate, currentFundingRate, threshold float64) {
	// 获取WebSocket管理器
	wsManager := websocket.GetGlobalWebSocketManager()
	if wsManager == nil {
		return
	}

	// 事件广播功能已移除
	logrus.Infof("资金费率检查失败事件: %s 资金费率 %.4f%%",
		estimate.Symbol, currentFundingRate*100)
}
