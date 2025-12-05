package binance

// ========== Binance API 端点常数 ==========

// Binance API 基础URL
const (
	SpotBaseURL       = "https://api.binance.com"
	FuturesBaseURL    = "https://fapi.binance.com"
	TestNetBaseURL    = "https://testnet.binance.vision"
	TestNetFuturesURL = "https://testnet.binancefuture.com"
)

// ========== Binance REST API 端点 ==========

// 现货公共端点
const (
	EndpointExchangeInfo = "/api/v3/exchangeInfo"
	EndpointTicker24hr   = "/api/v3/ticker/24hr"
	EndpointBookTicker   = "/api/v3/ticker/bookTicker"
	EndpointKlines       = "/api/v3/klines"
	EndpointServerTime   = "/api/v3/time"
)

// 期货公共端点
const (
	EndpointFuturesExchangeInfo = "/fapi/v1/exchangeInfo"
	EndpointFuturesTicker24hr   = "/fapi/v1/ticker/24hr"
	EndpointFuturesBookTicker   = "/fapi/v1/ticker/bookTicker"
	EndpointFuturesKlines       = "/fapi/v1/klines"
	EndpointFuturesPremiumIndex = "/fapi/v1/premiumIndex"
)

// ========== K线时间间隔 ==========

const (
	Interval1m  = "1m"
	Interval3m  = "3m"
	Interval5m  = "5m"
	Interval15m = "15m"
	Interval30m = "30m"
	Interval1h  = "1h"
	Interval2h  = "2h"
	Interval4h  = "4h"
	Interval6h  = "6h"
	Interval8h  = "8h"
	Interval12h = "12h"
	Interval1d  = "1d"
	Interval3d  = "3d"
	Interval1w  = "1w"
	Interval1M  = "1M"
)
