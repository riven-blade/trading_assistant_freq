package controllers

import (
	"net/http"
	"trading_assistant/pkg/freqtrade"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type PositionController struct {
	freqtradeController *freqtrade.Controller
}

// NewPositionController 创建新的持仓控制器
func NewPositionController(freqtradeController *freqtrade.Controller) *PositionController {
	return &PositionController{
		freqtradeController: freqtradeController,
	}
}

// GetPositions 获取当前持仓
func (pc *PositionController) GetPositions(c *gin.Context) {
	if pc.freqtradeController == nil {
		logrus.Error("Freqtrade控制器未初始化")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Freqtrade控制器未初始化",
		})
		return
	}

	// 从freqtrade获取持仓数据
	positions, err := pc.freqtradeController.GetPositions()
	if err != nil {
		logrus.Errorf("获取持仓数据失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "获取持仓数据失败",
			"details": err.Error(),
		})
		return
	}

	// 计算统计信息
	totalPnl := 0.0
	totalStakeAmount := 0.0
	for i := range positions {
		position := &positions[i]
		totalPnl += position.CurrentProfitAbs
		totalStakeAmount += position.StakeAmount
	}

	response := gin.H{
		"success": true,
		"data": gin.H{
			"positions":      positions,
			"total_pnl":      totalPnl,
			"position_count": len(positions),
			"total_stake":    totalStakeAmount,
			"last_updated":   nil, // freqtrade会提供实时数据
		},
	}

	c.JSON(http.StatusOK, response)
}

// GetPositionSummary 获取持仓摘要信息
func (pc *PositionController) GetPositionSummary(c *gin.Context) {
	if pc.freqtradeController == nil {
		logrus.Error("Freqtrade控制器未初始化")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Freqtrade控制器未初始化",
		})
		return
	}

	// 获取持仓数据
	positions, err := pc.freqtradeController.GetPositions()
	if err != nil {
		logrus.Errorf("获取持仓摘要失败: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "获取持仓摘要失败",
			"details": err.Error(),
		})
		return
	}

	// 统计数据
	totalPnl := 0.0
	totalStakeAmount := 0.0
	profitableCount := 0

	for i := range positions {
		position := &positions[i]
		totalPnl += position.CurrentProfitAbs
		totalStakeAmount += position.StakeAmount

		if position.CurrentProfitAbs > 0 {
			profitableCount++
		}
	}

	summary := gin.H{
		"success": true,
		"data": gin.H{
			"position_count":   len(positions),
			"total_pnl":        totalPnl,
			"total_stake":      totalStakeAmount,
			"profitable_count": profitableCount,
			"loss_count":       len(positions) - profitableCount,
		},
	}

	c.JSON(http.StatusOK, summary)
}
