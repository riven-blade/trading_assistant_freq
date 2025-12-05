package exchange_factory

import (
	"context"
	"fmt"
	"os"
	"strings"

	"trading_assistant/pkg/config"
	"trading_assistant/pkg/exchanges/binance"
	"trading_assistant/pkg/exchanges/bybit"
	"trading_assistant/pkg/exchanges/mexc"
	"trading_assistant/pkg/exchanges/okx"
	"trading_assistant/pkg/exchanges/types"
)

// ExchangeInterface 定义交易所接口
type ExchangeInterface interface {
	// 基础信息
	GetID() string
	GetName() string
	GetMarketType() string
	IsTestnet() bool

	// 核心市场数据功能
	FetchMarkets(ctx context.Context, params map[string]interface{}) ([]*types.Market, error)
	FetchTickers(ctx context.Context, symbols []string, params map[string]interface{}) (map[string]*types.Ticker, error)
	FetchBookTickers(ctx context.Context, symbols []string, params map[string]interface{}) (map[string]*types.Ticker, error) // 获取最优买卖价
	FetchKlines(ctx context.Context, symbol, interval string, since int64, limit int, params map[string]interface{}) ([]*types.Kline, error)

	FetchMarkPrice(ctx context.Context, symbol string) (*types.MarkPrice, error)
	FetchMarkPrices(ctx context.Context, symbols []string) (map[string]*types.MarkPrice, error)
}

// ExchangeType 支持的交易所类型
type ExchangeType string

const (
	ExchangeTypeBinance ExchangeType = "binance"
	ExchangeTypeBybit   ExchangeType = "bybit"
	ExchangeTypeOKX     ExchangeType = "okx"
	ExchangeTypeMEXC    ExchangeType = "mexc"
)

// ExchangeFactory 交易所工厂
type ExchangeFactory struct{}

// NewExchangeFactory 创建新的交易所工厂
func NewExchangeFactory() *ExchangeFactory {
	return &ExchangeFactory{}
}

// CreateExchange 根据配置创建交易所实例
func (f *ExchangeFactory) CreateExchange(exchangeType string, marketType string) (ExchangeInterface, error) {
	exchangeType = strings.ToLower(strings.TrimSpace(exchangeType))

	switch ExchangeType(exchangeType) {
	case ExchangeTypeBinance:
		return f.createBinanceExchange(marketType)
	case ExchangeTypeBybit:
		return f.createBybitExchange(marketType)
	case ExchangeTypeOKX:
		return f.createOKXExchange(marketType)
	case ExchangeTypeMEXC:
		return f.createMEXCExchange(marketType)
	default:
		return nil, fmt.Errorf("不支持的交易所类型: %s", exchangeType)
	}
}

// CreateFromConfig 从全局配置创建交易所
func (f *ExchangeFactory) CreateFromConfig() (ExchangeInterface, error) {
	if config.GlobalConfig == nil {
		return nil, fmt.Errorf("全局配置未初始化")
	}

	exchangeType := config.GlobalConfig.ExchangeType
	marketType := config.GlobalConfig.MarketType
	if marketType == "" {
		marketType = types.MarketTypeFuture // 默认期货市场
	}

	return f.CreateExchange(exchangeType, marketType)
}

// createBinanceExchange 创建 Binance 交易所实例
func (f *ExchangeFactory) createBinanceExchange(marketType string) (*binance.Binance, error) {
	config := binance.DefaultConfig()

	// 设置市场类型
	config.MarketType = marketType

	// 设置测试网环境
	if testnet := os.Getenv("BINANCE_TESTNET"); testnet == "true" {
		config.TestNet = true
	}

	return binance.New(config)
}

// createBybitExchange 创建 Bybit 交易所实例
func (f *ExchangeFactory) createBybitExchange(marketType string) (*bybit.Bybit, error) {
	config := bybit.DefaultConfig()

	// 设置市场类型
	if err := config.SetMarketType(marketType); err != nil {
		return nil, fmt.Errorf("设置Bybit市场类型失败: %w", err)
	}

	// 设置测试网环境
	if testnet := os.Getenv("BYBIT_TESTNET"); testnet == "true" {
		config.TestNet = true
	}

	return bybit.New(config)
}

// createOKXExchange 创建 OKX 交易所实例
func (f *ExchangeFactory) createOKXExchange(marketType string) (*okx.OKX, error) {
	config := okx.DefaultConfig()

	// 设置市场类型
	if err := config.SetMarketType(marketType); err != nil {
		return nil, fmt.Errorf("设置OKX市场类型失败: %w", err)
	}

	return okx.New(config)
}

// createMEXCExchange 创建 MEXC 交易所实例
func (f *ExchangeFactory) createMEXCExchange(marketType string) (*mexc.MEXC, error) {
	config := mexc.DefaultConfig()
	config.MarketType = marketType
	return mexc.New(config)
}

// GetSupportedExchanges 获取支持的交易所列表
func (f *ExchangeFactory) GetSupportedExchanges() []string {
	return []string{
		string(ExchangeTypeBinance),
		string(ExchangeTypeBybit),
		string(ExchangeTypeOKX),
		string(ExchangeTypeMEXC),
	}
}

// ValidateExchangeType 验证交易所类型是否支持
func (f *ExchangeFactory) ValidateExchangeType(exchangeType string) error {
	exchangeType = strings.ToLower(strings.TrimSpace(exchangeType))

	supportedExchanges := f.GetSupportedExchanges()
	for _, supported := range supportedExchanges {
		if exchangeType == supported {
			return nil
		}
	}

	return fmt.Errorf("不支持的交易所类型: %s, 支持的类型: %v", exchangeType, supportedExchanges)
}

// GetExchangeInfo 获取交易所信息
func (f *ExchangeFactory) GetExchangeInfo(exchangeType string) (map[string]interface{}, error) {
	exchangeType = strings.ToLower(strings.TrimSpace(exchangeType))

	switch ExchangeType(exchangeType) {
	case ExchangeTypeBinance:
		return map[string]interface{}{
			"name": "Binance", "id": "binance", "countries": []string{"JP", "MT"},
			"version": "v3", "website": "https://www.binance.com",
			"spot": true, "futures": true,
		}, nil
	case ExchangeTypeBybit:
		return map[string]interface{}{
			"name": "Bybit", "id": "bybit", "countries": []string{"VG"},
			"version": "v5", "website": "https://www.bybit.com",
			"spot": true, "futures": true,
		}, nil
	case ExchangeTypeOKX:
		return map[string]interface{}{
			"name": "OKX", "id": "okx", "countries": []string{"SC"},
			"version": "v5", "website": "https://www.okx.com",
			"spot": true, "futures": true,
		}, nil
	case ExchangeTypeMEXC:
		return map[string]interface{}{
			"name": "MEXC", "id": "mexc", "countries": []string{"SG"},
			"version": "v3", "website": "https://www.mexc.com",
			"spot": true, "futures": false,
		}, nil
	default:
		return nil, fmt.Errorf("不支持的交易所类型: %s", exchangeType)
	}
}

// CreateDefaultExchange 创建默认交易所
func CreateDefaultExchange() (ExchangeInterface, error) {
	factory := NewExchangeFactory()

	// 如果有全局配置，使用配置的交易所
	if config.GlobalConfig != nil && config.GlobalConfig.ExchangeType != "" {
		return factory.CreateFromConfig()
	}

	// 否则默认使用 Binance
	return factory.CreateExchange(string(ExchangeTypeBinance), types.MarketTypeFuture)
}

// GetAvailableMarketTypes 获取交易所支持的市场类型
func (f *ExchangeFactory) GetAvailableMarketTypes(exchangeType string) ([]string, error) {
	exchangeType = strings.ToLower(strings.TrimSpace(exchangeType))

	switch ExchangeType(exchangeType) {
	case ExchangeTypeBinance, ExchangeTypeBybit:
		return []string{types.MarketTypeSpot, types.MarketTypeFuture}, nil
	case ExchangeTypeOKX:
		return []string{types.MarketTypeSpot, types.MarketTypeFuture}, nil
	case ExchangeTypeMEXC:
		return []string{types.MarketTypeSpot}, nil
	default:
		return nil, fmt.Errorf("不支持的交易所类型: %s", exchangeType)
	}
}
