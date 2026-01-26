package controllers

import (
	"fmt"
	"math"
	"net/http"
	"strconv"
	"time"
	"trading_assistant/models"
	"trading_assistant/pkg/config"
	"trading_assistant/pkg/exchanges/types"
	"trading_assistant/pkg/redis"
	"trading_assistant/pkg/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

type PriceController struct{}

// PriceEstimateRequest 价格预估请求结构
type PriceEstimateRequest struct {
	Symbol      string      `json:"symbol" binding:"required"`
	Side        string      `json:"side" binding:"required"`        // long, short
	ActionType  string      `json:"action_type" binding:"required"` // open, close
	TargetPrice float64     `json:"target_price"`
	Percentage  float64     `json:"percentage"`                     // 仓位比例 (加仓时必填)
	Leverage    int         `json:"leverage"`                       // 杠杆倍数
	OrderType   string      `json:"order_type"`                     // 订单类型：market, limit
	MarginMode  string      `json:"margin_mode"`                    // CROSS, ISOLATED (默认CROSS)
	TriggerType string      `json:"trigger_type"`                   // 触发类型
	Tag         interface{} `json:"tag"`                            // 交易标签（支持字符串和数字）
	StakeAmount float64     `json:"stake_amount"`                   // 操作金额 (USDT 保证金)
	Amount      float64     `json:"amount"`                         // 交易数量 (币的数量)
}

// isSpotMode 判断是否为现货模式
func (p *PriceController) isSpotMode() bool {
	return config.GlobalConfig != nil && config.GlobalConfig.MarketType == types.MarketTypeSpot
}

// validatePriceEstimateRequest 验证价格预估请求
func (p *PriceController) validatePriceEstimateRequest(req *PriceEstimateRequest) error {
	// 现货模式特殊处理
	if p.isSpotMode() {
		// 现货模式强制使用 long 方向
		req.Side = types.PositionSideLong
		// 现货模式杠杆固定为1
		req.Leverage = 1
	} else {
		// 期货模式验证交易方向
		if req.Side != types.PositionSideLong && req.Side != types.PositionSideShort {
			return fmt.Errorf("交易方向必须是 %s 或 %s", types.PositionSideLong, types.PositionSideShort)
		}
		// 设置默认杠杆
		if req.Leverage <= 0 {
			req.Leverage = 5 // 默认5倍杠杆
		}
	}

	// 验证操作类型
	validActionTypes := []string{
		models.ActionTypeOpen,
		models.ActionTypeAddition,
		models.ActionTypeTakeProfit,
	}
	isValidActionType := false
	for i := range validActionTypes {
		validType := validActionTypes[i]
		if req.ActionType == validType {
			isValidActionType = true
			break
		}
	}
	if !isValidActionType {
		return fmt.Errorf("操作类型必须是: %v", validActionTypes)
	}

	// 设置默认值并验证保证金模式
	if req.MarginMode == "" {
		req.MarginMode = types.MarginModeCross // 默认全仓
	}
	if req.MarginMode != types.MarginModeCross && req.MarginMode != types.MarginModeIsolated {
		return fmt.Errorf("保证金模式必须是 %s 或 %s", types.MarginModeCross, types.MarginModeIsolated)
	}

	// 设置默认值并验证订单类型
	if req.OrderType == "" {
		req.OrderType = types.OrderTypeLimit // 默认限价单
	}
	if req.OrderType != types.OrderTypeMarket && req.OrderType != types.OrderTypeLimit {
		return fmt.Errorf("订单类型必须是 %s 或 %s", types.OrderTypeMarket, types.OrderTypeLimit)
	}

	// 验证触发类型
	if req.TriggerType == "" {
		req.TriggerType = models.TriggerTypeCondition // 默认条件触发
	}
	if req.TriggerType != models.TriggerTypeCondition && req.TriggerType != models.TriggerTypeImmediate {
		return fmt.Errorf("触发类型必须是 %s 或 %s", models.TriggerTypeCondition, models.TriggerTypeImmediate)
	}

	// 根据操作类型验证必填字段
	switch req.ActionType {
	case models.ActionTypeAddition:
		// 加仓必须指定 Percentage
		if req.Percentage <= 0 {
			return fmt.Errorf("加仓操作必须指定有效的 Percentage (>0)，当前值: %.2f", req.Percentage)
		}
	case models.ActionTypeTakeProfit:
		// 止盈必须指定 Amount
		if req.Amount <= 0 {
			return fmt.Errorf("止盈操作必须指定 Amount > 0")
		}
	}

	// 条件触发时必须指定目标价格
	if req.TriggerType == models.TriggerTypeCondition && req.TargetPrice <= 0 {
		return fmt.Errorf("条件触发必须指定有效的目标价格 (target_price > 0)")
	}

	return nil
}

// formatPriceEstimatePrecision 格式化价格预估的精度
func (p *PriceController) formatPriceEstimatePrecision(req *PriceEstimateRequest) error {
	// 获取币种信息 (req.Symbol现在存储的就是MarketID)
	coin, err := redis.GlobalRedisClient.GetCoin(req.Symbol)
	if err != nil {
		logrus.Warnf("获取币种信息失败，使用默认精度: %s, error: %v", req.Symbol, err)
		// 使用默认精度
		req.Percentage = parseFloat(fmt.Sprintf("%.2f", req.Percentage))
		req.TargetPrice = parseFloat(fmt.Sprintf("%.4f", req.TargetPrice))
		return nil
	}

	// 格式化百分比精度，但允许为0
	if req.Percentage > 0 {
		req.Percentage = parseFloat(fmt.Sprintf("%.2f", req.Percentage))
	}

	// 格式化数量精度
	if req.Amount > 0 {
		quantityPrecision := coin.GetQuantityPrecisionFromStepSize()
		if quantityPrecision > 0 {
			quantityFormat := fmt.Sprintf("%%.%df", quantityPrecision)
			req.Amount = parseFloat(fmt.Sprintf(quantityFormat, req.Amount))
		}
		// 验证最小数量
		if coin.MinQty != "" {
			minQty := parseFloat(coin.MinQty)
			if minQty > 0 && req.Amount < minQty {
				return fmt.Errorf("交易数量 %.6f 小于最小数量 %.6f", req.Amount, minQty)
			}
		}
		// 验证最大数量
		if coin.MaxQty != "" {
			maxQty := parseFloat(coin.MaxQty)
			if maxQty > 0 && req.Amount > maxQty {
				return fmt.Errorf("交易数量 %.6f 大于最大数量 %.6f", req.Amount, maxQty)
			}
		}
		// 验证数量步长
		if coin.StepSize != "" {
			stepSize := parseFloat(coin.StepSize)
			if stepSize > 0 {
				// 使用 epsilon 避免浮点数计算误差
				epsilon := 1e-9
				steps := req.Amount / stepSize
				// 检查 steps 是否接近整数
				if math.Abs(steps-math.Round(steps)) > epsilon {
					adjustedQty := math.Floor(steps) * stepSize
					req.Amount = parseFloat(fmt.Sprintf("%%.%df", quantityPrecision, adjustedQty))
				}
			}
		}
	}

	// 格式化价格精度
	pricePrecision := coin.GetPricePrecisionFromTickSize()
	if pricePrecision > 0 {
		priceFormat := fmt.Sprintf("%%.%df", pricePrecision)
		req.TargetPrice = parseFloat(fmt.Sprintf(priceFormat, req.TargetPrice))

		// 验证最小价格（立即触发时跳过验证，因为 target_price 可以为 0）
		if coin.MinPrice != "" && req.TriggerType != models.TriggerTypeImmediate {
			minPrice := parseFloat(coin.MinPrice)
			if minPrice > 0 && req.TargetPrice < minPrice {
				return fmt.Errorf("目标价格 %.6f 小于最小价格 %.6f", req.TargetPrice, minPrice)
			}
		}

		// 验证价格步长
		if coin.TickSize != "" {
			tickSize := parseFloat(coin.TickSize)
			if tickSize > 0 {
				steps := req.TargetPrice / tickSize
				if steps != float64(int(steps)) {
					adjustedPrice := float64(int(steps)) * tickSize
					req.TargetPrice = parseFloat(fmt.Sprintf(priceFormat, adjustedPrice))
				}
			}
		}
	}

	logrus.WithFields(logrus.Fields{
		"symbol":       req.Symbol,
		"stake_amount": req.StakeAmount,
		"percentage":   req.Percentage,
		"target_price": req.TargetPrice,
		"min_price":    coin.MinPrice,
		"tick_size":    coin.TickSize,
	}).Debug("精度格式化完成")

	return nil
}

// parseFloat 解析格式化后的浮点数
func parseFloat(s string) float64 {
	val, _ := strconv.ParseFloat(s, 64)
	return val
}

// createPriceEstimateModel 创建价格预估模型
func (p *PriceController) createPriceEstimateModel(req *PriceEstimateRequest) *models.PriceEstimate {
	// 将Tag转换为字符串
	var tagStr string
	if req.Tag != nil {
		tagStr = fmt.Sprintf("%v", req.Tag)
	}

	// 初始状态为已启用，自动开始监听
	return &models.PriceEstimate{
		ID:          uuid.New().String(),
		Symbol:      req.Symbol,
		Side:        req.Side,
		ActionType:  req.ActionType,
		TargetPrice: req.TargetPrice,
		Percentage:  req.Percentage, // 恢复 Percentage 字段
		Leverage:    req.Leverage,
		OrderType:   req.OrderType,
		MarginMode:  req.MarginMode,
		TriggerType: req.TriggerType,
		Tag:         tagStr,                         // 交易标签（转换为字符串）
		StakeAmount: req.StakeAmount,                // 操作金额 (USDT 保证金)
		Amount:      req.Amount,                     // 交易数量 (币的数量)
		Status:      models.EstimateStatusListening, // 初始状态为监听状态
		Enabled:     true,                           // 默认启用，自动开始监听
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}
}

// CreatePriceEstimate 创建价格预估
func (p *PriceController) CreatePriceEstimate(ctx *gin.Context) {
	var req PriceEstimateRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		logrus.Warnf("价格预估参数错误: %v", err)
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "请求参数格式错误",
		})
		return
	}

	// 验证请求参数
	if err := p.validatePriceEstimateRequest(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": err.Error(),
		})
		return
	}

	// 格式化数量和价格精度
	if err := p.formatPriceEstimatePrecision(&req); err != nil {
		logrus.Errorf("格式化精度失败: %v", err)
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "格式化精度失败: " + err.Error(),
		})
		return
	}

	// 创建价格预估模型
	estimate := p.createPriceEstimateModel(&req)

	// 保存到Redis
	if redis.GlobalRedisClient == nil {
		ctx.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Redis服务不可用",
		})
		return
	}

	if err := redis.GlobalRedisClient.SetPriceEstimate(estimate); err != nil {
		logrus.Errorf("保存价格预估失败: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "保存价格预估失败",
		})
		return
	}

	// 自动选中币种（如果还未选中）
	if !redis.GlobalRedisClient.IsCoinSelected(req.Symbol) {
		err := redis.GlobalRedisClient.SetCoinSelection(req.Symbol, models.CoinSelectionActive)
		if err != nil {
			logrus.Warnf("自动选中币种失败: %s, error: %v", req.Symbol, err)
			// 不影响价格预估的创建，继续执行
		} else {
			logrus.Infof("币种 %s 已自动选中", req.Symbol)
		}
	}

	logrus.Infof("创建价格预估成功: %s %s %s %.4f",
		estimate.Symbol, estimate.Side, estimate.ActionType, estimate.TargetPrice)

	// 通过WebSocket广播价格预估更新
	go utils.BroadcastSymbolEstimatesUpdate()

	ctx.JSON(http.StatusOK, gin.H{
		"message": "价格预估创建成功",
		"data":    estimate,
	})
}

// DeletePriceEstimate 删除价格预估
func (p *PriceController) DeletePriceEstimate(ctx *gin.Context) {
	id := ctx.Param("id")

	if redis.GlobalRedisClient == nil {
		ctx.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Redis服务不可用",
		})
		return
	}

	// 直接删除预估记录
	err := redis.GlobalRedisClient.DeletePriceEstimate(id)
	if err != nil {
		logrus.Errorf("删除价格预估失败: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "删除价格预估失败",
		})
		return
	}

	logrus.Infof("删除价格预估成功: %s", id)

	// 通过WebSocket广播价格预估更新
	go utils.BroadcastSymbolEstimatesUpdate()

	ctx.JSON(http.StatusOK, gin.H{
		"message": "价格预估删除成功",
	})
}

// TogglePriceEstimate 切换价格预估监听状态
func (p *PriceController) TogglePriceEstimate(ctx *gin.Context) {
	id := ctx.Param("id")

	var req struct {
		Enabled bool `json:"enabled"`
	}

	if err := ctx.ShouldBindJSON(&req); err != nil {
		logrus.Warnf("价格预估切换参数错误: %v", err)
		ctx.JSON(http.StatusBadRequest, gin.H{
			"error": "请求参数格式错误",
		})
		return
	}

	if redis.GlobalRedisClient == nil {
		ctx.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "Redis服务不可用",
		})
		return
	}

	// 获取价格预估
	estimate, err := redis.GlobalRedisClient.GetEstimateById(id)
	if err != nil {
		ctx.JSON(http.StatusNotFound, gin.H{
			"error": "价格预估不存在",
		})
		return
	}

	estimate.Enabled = req.Enabled
	estimate.UpdatedAt = time.Now()

	if err := redis.GlobalRedisClient.SetPriceEstimate(estimate); err != nil {
		logrus.Errorf("更新价格预估状态失败: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "更新价格预估状态失败",
		})
		return
	}

	statusText := "暂停"
	if req.Enabled {
		statusText = "激活"
	}

	logrus.Infof("价格预估状态已更新: %s -> %s", id, statusText)

	// 通过WebSocket广播价格预估更新
	go utils.BroadcastSymbolEstimatesUpdate()

	ctx.JSON(http.StatusOK, gin.H{
		"message": "价格预估状态更新成功",
		"data":    estimate,
	})
}

// GetAllPriceEstimates 获取所有价格预估
func (p *PriceController) GetAllPriceEstimates(ctx *gin.Context) {
	symbol := ctx.Query("symbol")

	var estimates []*models.PriceEstimate
	var err error

	// 根据是否有symbol参数选择获取方法
	if symbol != "" {
		estimates, err = redis.GlobalRedisClient.GetAllEstimatesBySymbol(symbol)
	} else {
		estimates, err = redis.GlobalRedisClient.GetAllEstimates()
	}

	if err != nil {
		logrus.Errorf("获取价格预估失败: %v", err)
		ctx.JSON(http.StatusInternalServerError, gin.H{
			"error": "获取价格预估失败",
		})
		return
	}

	logrus.Debugf("获取到 %d 条价格预估数据 (symbol: %s)", len(estimates), symbol)

	ctx.JSON(http.StatusOK, gin.H{
		"data": estimates,
	})
}
