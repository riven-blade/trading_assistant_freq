package core

import (
	"fmt"
	"time"
	"trading_assistant/models"
	"trading_assistant/pkg/exchanges/types"
	"trading_assistant/pkg/freqtrade"
	"trading_assistant/pkg/redis"
	"trading_assistant/pkg/utils"

	"github.com/sirupsen/logrus"
)

// OrderExecutor 订单执行器
type OrderExecutor struct {
	freqtradeClient *freqtrade.Controller
}

// NewOrderExecutor 创建订单执行器
func NewOrderExecutor(freqtradeClient *freqtrade.Controller) *OrderExecutor {
	return &OrderExecutor{
		freqtradeClient: freqtradeClient,
	}
}

// ExecuteOrder 执行订单
func (oe *OrderExecutor) ExecuteOrder(estimate *models.PriceEstimate, currentPrice float64) error {
	if oe.freqtradeClient == nil {
		return fmt.Errorf("freqtrade客户端未初始化")
	}

	logrus.WithFields(logrus.Fields{
		"symbol":        estimate.Symbol,
		"action_type":   estimate.ActionType,
		"side":          estimate.Side,
		"percentage":    estimate.Percentage,
		"target_price":  estimate.TargetPrice,
		"current_price": currentPrice,
	}).Info("开始执行Freqtrade订单")

	// 执行下单
	err := oe.executeFreqtradeOrder(estimate, currentPrice)
	if err != nil {
		return fmt.Errorf("freqtrade下单失败: %v", err)
	}

	// 更新预估状态
	if err := oe.updateEstimateStatus(estimate, "triggered"); err != nil {
		logrus.Errorf("更新预估状态失败: %v", err)
	}

	logrus.WithFields(logrus.Fields{
		"symbol":        estimate.Symbol,
		"action_type":   estimate.ActionType,
		"side":          estimate.Side,
		"target_price":  estimate.TargetPrice,
		"current_price": currentPrice,
	}).Info("Freqtrade订单执行成功")

	return nil
}

// executeFreqtradeOrder 执行下单
func (oe *OrderExecutor) executeFreqtradeOrder(estimate *models.PriceEstimate, currentPrice float64) error {
	switch estimate.ActionType {
	case models.ActionTypeOpen:
		return oe.executeOpenPosition(estimate, currentPrice)
	case models.ActionTypeAddition:
		return oe.executeAddPosition(estimate, currentPrice)
	case models.ActionTypeTakeProfit:
		return oe.executeTakeProfit(estimate, currentPrice)
	default:
		return fmt.Errorf("不支持的操作类型: %s", estimate.ActionType)
	}
}

// executeOpenPosition 开仓
func (oe *OrderExecutor) executeOpenPosition(estimate *models.PriceEstimate, currentPrice float64) error {
	futureSymbol := utils.ConvertMarketIDToFutureSymbol(estimate.Symbol)

	// 检查是否可以开仓
	if !oe.freqtradeClient.CheckForceBuy(futureSymbol) {
		return fmt.Errorf("无法开仓: 达到最大持仓数量或交易对已存在持仓")
	}

	orderType := "market"
	if estimate.OrderType == types.OrderTypeLimit {
		orderType = "limit"
	}

	orderPrice := currentPrice

	entryTag := estimate.Tag
	if entryTag == "" {
		entryTag = fmt.Sprintf("open_%s", estimate.Side)
	}

	// 确定开仓方向
	side := "long"
	if estimate.Side == types.PositionSideShort {
		side = "short"
	}

	payload := models.ForceBuyPayload{
		Pair:      futureSymbol,
		OrderType: orderType,
		EntryTag:  entryTag,
		Side:      side, // 设置开仓方向
		Leverage:  estimate.Leverage,
	}

	// 只有当开仓金额大于0时才设置
	if estimate.StakeAmount > 0 {
		payload.StakeAmount = &estimate.StakeAmount
	}

	// 设置订单价格
	if orderType == "limit" {
		payload.Price = orderPrice
	}

	logrus.WithFields(logrus.Fields{
		"symbol":        estimate.Symbol,
		"side":          side,
		"order_type":    orderType,
		"leverage":      estimate.Leverage,
		"stake_amount":  estimate.StakeAmount,
		"current_price": currentPrice,
		"order_price":   orderPrice,
		"target_price":  estimate.TargetPrice,
	}).Info("执行开仓订单")

	return oe.freqtradeClient.ForceBuy(payload)
}

// executeAddPosition 加仓
func (oe *OrderExecutor) executeAddPosition(estimate *models.PriceEstimate, currentPrice float64) error {
	positions, err := oe.freqtradeClient.GetPositions()
	if err != nil {
		return fmt.Errorf("获取仓位信息失败: %v", err)
	}

	futureSymbol := utils.ConvertMarketIDToFutureSymbol(estimate.Symbol)

	var existingPosition *models.TradePosition
	for i := range positions {
		pos := &positions[i]
		if pos.Pair == futureSymbol && pos.IsOpen {
			// 检查方向是否匹配
			isLongPosition := pos.TradeDirection == "long" || !pos.IsShort
			isEstimateLong := estimate.Side == types.PositionSideLong

			if isLongPosition == isEstimateLong {
				existingPosition = pos
				break
			}
		}
	}

	if existingPosition == nil {
		return fmt.Errorf("未找到对应的仓位用于加仓 %s %s", estimate.Symbol, estimate.Side)
	}

	cost := existingPosition.Orders[0].Cost
	if *cost <= 0 {
		return fmt.Errorf("获取不到原始投入金额")
	}

	stakeCost := *cost * (estimate.Percentage / 100.0) / *existingPosition.Leverage * 10

	orderPrice := currentPrice

	logrus.WithFields(logrus.Fields{
		"symbol":            estimate.Symbol,
		"side":              estimate.Side,
		"existing_position": existingPosition.Amount,
		"add_percentage":    estimate.Percentage,
		"add_stake_amount":  stakeCost,
		"current_price":     currentPrice,
		"order_price":       orderPrice,
		"target_price":      estimate.TargetPrice,
	}).Info("计算加仓金额")

	side := "long"
	if estimate.Side == types.PositionSideShort {
		side = "short"
	}

	entryTag := estimate.Tag
	if entryTag == "" {
		// 如果没有指定标签，使用默认格式
		entryTag = fmt.Sprintf("add_%s", estimate.Side)
	}

	return oe.freqtradeClient.ForceAdjustBuy(
		futureSymbol,
		orderPrice,
		side,
		stakeCost,
		entryTag,
	)
}

// executeTakeProfit 止盈
func (oe *OrderExecutor) executeTakeProfit(estimate *models.PriceEstimate, currentPrice float64) error {
	return oe.executeSellOperation(estimate, currentPrice, "take_profit")
}

// executeSellOperation 执行卖出操作
func (oe *OrderExecutor) executeSellOperation(estimate *models.PriceEstimate, currentPrice float64, operation string) error {
	// 获取当前交易状态
	trades, err := oe.freqtradeClient.GetTradeStatus()
	if err != nil {
		return fmt.Errorf("获取交易状态失败: %v", err)
	}

	futureSymbol := utils.ConvertMarketIDToFutureSymbol(estimate.Symbol)

	// 查找对应的开仓交易
	var targetTrade *models.TradePosition
	for i := range trades {
		trade := &trades[i]
		if trade.Pair == futureSymbol && trade.IsOpen {
			// 检查方向是否匹配
			isLongPosition := trade.TradeDirection == "long" || !trade.IsShort
			isEstimateLong := estimate.Side == types.PositionSideLong

			if isLongPosition == isEstimateLong {
				targetTrade = trade
				break
			}
		}
	}

	// 检查是否找到对应仓位
	if targetTrade == nil {
		return fmt.Errorf("未找到对应的仓位用于%s %s %s", operation, estimate.Symbol, estimate.Side)
	}

	// 计算卖出数量
	orderType := "market"
	if estimate.OrderType == types.OrderTypeLimit {
		orderType = "limit"
	}

	amount := "all" // 默认全部卖出
	var sellAmount float64
	if estimate.Percentage > 0 && estimate.Percentage < 100 {
		// 根据百分比计算卖出数量
		sellAmount = targetTrade.Amount * (estimate.Percentage / 100.0)
		amount = fmt.Sprintf("%.8f", sellAmount)
	} else {
		sellAmount = targetTrade.Amount
	}

	logrus.WithFields(logrus.Fields{
		"symbol":          estimate.Symbol,
		"side":            estimate.Side,
		"operation":       operation,
		"position_amount": targetTrade.Amount,
		"sell_percentage": estimate.Percentage,
		"sell_amount":     sellAmount,
		"trade_id":        targetTrade.TradeId,
		"current_price":   currentPrice,
		"target_price":    estimate.TargetPrice,
		"order_type":      orderType,
	}).Info("执行卖出操作")

	return oe.freqtradeClient.ForceSell(
		fmt.Sprintf("%d", targetTrade.TradeId),
		orderType,
		amount,
	)
}

// updateEstimateStatus 更新预估状态
func (oe *OrderExecutor) updateEstimateStatus(estimate *models.PriceEstimate, status string) error {
	logrus.WithFields(logrus.Fields{
		"estimate_id": estimate.ID,
		"old_status":  estimate.Status,
		"new_status":  status,
	}).Debug("更新预估状态")

	estimate.Status = status
	estimate.UpdatedAt = time.Now()

	err := redis.GlobalRedisClient.SetPriceEstimate(estimate)
	if err != nil {
		return err
	}

	// 广播价格预估更新
	go utils.BroadcastSymbolEstimatesUpdate()
	return nil
}

// getActionText 获取操作类型的中文描述（freqtrade的3种核心操作）
func (oe *OrderExecutor) getActionText(actionType string) string {
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
func (oe *OrderExecutor) getPositionText(side string) string {
	switch side {
	case types.PositionSideLong:
		return "做多"
	case types.PositionSideShort:
		return "做空"
	default:
		return "未知"
	}
}
