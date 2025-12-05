package mexc

import (
	"trading_assistant/pkg/exchanges/types"
)

// Config MEXC 交易所配置 (仅公共市场数据)
type Config struct {
	Timeout    int    `json:"timeout"`
	MarketType string `json:"marketType"`
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		Timeout:    30000, // 30秒
		MarketType: types.MarketTypeSpot,
	}
}

// Validate 验证配置
func (c *Config) Validate() error {
	return nil
}

// Clone 克隆配置
func (c *Config) Clone() *Config {
	clone := *c
	return &clone
}

// GetBaseURL 获取基础URL
func (c *Config) GetBaseURL() string {
	return BaseURL
}

