package bybit

import (
	"fmt"
	"trading_assistant/pkg/exchanges/types"
)

// ========== Bybit 配置（简化版 - 仅公共市场数据）==========

// Config Bybit 交易所配置
type Config struct {
	// 环境配置
	TestNet bool `json:"testnet"` // 是否使用测试网

	// 网络配置
	Timeout int `json:"timeout"` // 超时时间(毫秒)

	// 市场类型配置
	MarketType string `json:"marketType"` // 市场类型: spot, future

	// Bybit 特有配置
	Category string `json:"category"` // 产品类型: spot, linear, inverse
}

// DefaultConfig 返回默认配置
func DefaultConfig() *Config {
	return &Config{
		TestNet:    false,
		Timeout:    30000, // 30秒
		MarketType: types.MarketTypeSpot,
		Category:   CategorySpot,
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

	// 验证产品类型
	validCategories := map[string]bool{
		CategorySpot:    true,
		CategoryLinear:  true,
		CategoryInverse: true,
	}

	if !validCategories[c.Category] {
		return fmt.Errorf("invalid category: %s", c.Category)
	}

	// 市场类型和产品类型的映射验证
	typeMapping := map[string][]string{
		types.MarketTypeSpot:   {CategorySpot},
		types.MarketTypeFuture: {CategoryLinear, CategoryInverse},
	}

	if validCats, exists := typeMapping[c.MarketType]; exists {
		valid := false
		for _, validCat := range validCats {
			if c.Category == validCat {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("category %s is not valid for marketType %s", c.Category, c.MarketType)
		}
	}

	return nil
}

// Clone 克隆配置
func (c *Config) Clone() *Config {
	clone := *c
	return &clone
}

// SetMarketType 设置市场类型和产品类型
func (c *Config) SetMarketType(marketType string) error {
	c.MarketType = marketType

	// 自动设置对应的产品类型
	switch marketType {
	case types.MarketTypeSpot:
		c.Category = CategorySpot
	case types.MarketTypeFuture:
		c.Category = CategoryLinear // 默认使用USDT永续
	default:
		return fmt.Errorf("unsupported market type: %s, must be 'spot' or 'future'", marketType)
	}

	return nil
}

// GetBaseURL 获取基础URL
func (c *Config) GetBaseURL() string {
	if c.TestNet {
		return TestNetBaseURL
	}
	return BaseURL
}

// IsSpot 是否现货
func (c *Config) IsSpot() bool {
	return c.Category == CategorySpot
}

// IsFutures 是否期货
func (c *Config) IsFutures() bool {
	return c.Category == CategoryLinear || c.Category == CategoryInverse
}

// IsLinear 是否USDT永续
func (c *Config) IsLinear() bool {
	return c.Category == CategoryLinear
}

// IsInverse 是否币本位永续
func (c *Config) IsInverse() bool {
	return c.Category == CategoryInverse
}
