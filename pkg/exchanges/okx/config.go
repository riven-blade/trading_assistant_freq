package okx

import (
	"fmt"
	"trading_assistant/pkg/exchanges/types"
)

// Config OKX 交易所配置 (仅公共市场数据)
type Config struct {
	// 网络配置
	Timeout int    `json:"timeout"` // 超时时间(毫秒)
	UseAWS  bool   `json:"useAWS"`  // 是否使用AWS线路
	Proxy   string `json:"proxy,omitempty"`

	// 市场类型配置
	MarketType string `json:"marketType"` // 市场类型: spot, future
	InstType   string `json:"instType"`   // OKX产品类型: SPOT, SWAP, FUTURES
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		Timeout:    30000, // 30秒
		UseAWS:     false,
		MarketType: types.MarketTypeSpot,
		InstType:   InstTypeSpot,
	}
}

// Validate 验证配置
func (c *Config) Validate() error {
	switch c.InstType {
	case InstTypeSpot, InstTypeSwap, InstTypeFutures:
		return nil
	default:
		return fmt.Errorf("无效的产品类型: %s", c.InstType)
	}
}

// Clone 克隆配置
func (c *Config) Clone() *Config {
	clone := *c
	return &clone
}

// SetMarketType 设置市场类型
func (c *Config) SetMarketType(marketType string) error {
	c.MarketType = marketType
	switch marketType {
	case types.MarketTypeSpot:
		c.InstType = InstTypeSpot
	case types.MarketTypeFuture, types.MarketTypeSwap:
		c.InstType = InstTypeSwap
	default:
		return fmt.Errorf("不支持的市场类型: %s", marketType)
	}
	return nil
}

// IsFutures 是否期货/合约模式
func (c *Config) IsFutures() bool {
	return c.InstType == InstTypeSwap || c.InstType == InstTypeFutures
}

// GetBaseURL 获取基础URL
func (c *Config) GetBaseURL() string {
	if c.UseAWS {
		return AWSBaseURL
	}
	return BaseURL
}
