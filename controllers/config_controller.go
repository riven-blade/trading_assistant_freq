package controllers

import (
	"net/http"
	"trading_assistant/pkg/config"

	"github.com/gin-gonic/gin"
)

// ConfigController 系统配置控制器
type ConfigController struct{}

// NewConfigController 创建配置控制器
func NewConfigController() *ConfigController {
	return &ConfigController{}
}

// SystemConfigResponse 系统配置响应
type SystemConfigResponse struct {
	ExchangeType string `json:"exchange_type"` // 交易所类型: binance, bybit, okx, mexc
	MarketType   string `json:"market_type"`   // 市场类型: spot, future
}

// GetSystemConfig 获取系统配置
func (c *ConfigController) GetSystemConfig(ctx *gin.Context) {
	cfg := config.GlobalConfig

	response := SystemConfigResponse{
		ExchangeType: cfg.ExchangeType,
		MarketType:   cfg.MarketType,
	}

	ctx.JSON(http.StatusOK, gin.H{
		"data": response,
	})
}

