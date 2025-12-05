package binance

import (
	"fmt"
	"trading_assistant/pkg/exchanges/types"
)

// ========== Binance 配置 ==========

// Config Binance 交易所配置（简化版 - 仅公共市场数据）
type Config struct {
	// 环境配置
	TestNet bool `json:"testnet"` // 是否使用测试网

	// 网络配置
	Timeout int `json:"timeout"` // 超时时间(毫秒)

	// 市场类型配置
	MarketType string `json:"marketType"` // 市场类型: spot, futures
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		TestNet:    false,
		Timeout:    30000, // 30秒
		MarketType: types.MarketTypeSpot,
	}
}

// Validate 验证配置
func (c *Config) Validate() error {
	if c.Timeout < 0 {
		return fmt.Errorf("timeout cannot be negative")
	}

	// 验证市场类型
	validTypes := map[string]bool{
		types.MarketTypeSpot:   true,
		types.MarketTypeFuture: true,
	}

	if !validTypes[c.MarketType] {
		return fmt.Errorf("invalid marketType: %s, must be 'spot' or 'future'", c.MarketType)
	}

	return nil
}

// Clone 克隆配置
func (c *Config) Clone() *Config {
	clone := *c
	return &clone
}

// GetBaseURL 获取基础URL
func (c *Config) GetBaseURL() string {
	if c.TestNet {
		return TestNetBaseURL
	}
	return SpotBaseURL
}

// GetFuturesURL 获取期货URL
func (c *Config) GetFuturesURL() string {
	if c.TestNet {
		return TestNetFuturesURL
	}
	return FuturesBaseURL
}

// IsSpot 是否现货
func (c *Config) IsSpot() bool {
	return c.MarketType == types.MarketTypeSpot
}

// IsFutures 是否期货
func (c *Config) IsFutures() bool {
	return c.MarketType == types.MarketTypeFuture
}
