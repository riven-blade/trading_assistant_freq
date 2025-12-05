package mexc

// ========== MEXC API 基础URL ==========

const (
	BaseURL = "https://api.mexc.com"
)

// ========== MEXC 公共数据端点 ==========

const (
	EndpointExchangeInfo = "/api/v3/exchangeInfo"
	EndpointTicker24hr   = "/api/v3/ticker/24hr"
	EndpointTickerPrice  = "/api/v3/ticker/price"
	EndpointBookTicker   = "/api/v3/ticker/bookTicker"
	EndpointKlines       = "/api/v3/klines"
	EndpointServerTime   = "/api/v3/time"
)

// ========== MEXC 时间周期常数 ==========

const (
	Interval1m  = "1m"
	Interval5m  = "5m"
	Interval15m = "15m"
	Interval30m = "30m"
	Interval1h  = "60m"
	Interval4h  = "4h"
	Interval1d  = "1d"
	Interval1w  = "1W"
	Interval1M  = "1M"
)

