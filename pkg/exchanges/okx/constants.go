package okx

// ========== OKX API 基础URL ==========

const (
	BaseURL    = "https://www.okx.com"
	AWSBaseURL = "https://aws.okx.com"
)

// ========== OKX 公共数据端点 ==========

const (
	EndpointInstruments = "/api/v5/public/instruments"
	EndpointTickers     = "/api/v5/market/tickers"
	EndpointTicker      = "/api/v5/market/ticker"
	EndpointKlines      = "/api/v5/market/candles"
	EndpointMarkPrice   = "/api/v5/public/mark-price"
	EndpointFundingRate = "/api/v5/public/funding-rate"
)

// ========== OKX 产品类型常数 ==========

const (
	InstTypeSpot    = "SPOT"
	InstTypeSwap    = "SWAP"
	InstTypeFutures = "FUTURES"
)

// ========== OKX 时间周期常数 ==========

const (
	Interval1m  = "1m"
	Interval3m  = "3m"
	Interval5m  = "5m"
	Interval15m = "15m"
	Interval30m = "30m"
	Interval1H  = "1H"
	Interval2H  = "2H"
	Interval4H  = "4H"
	Interval6H  = "6Hutc"
	Interval12H = "12Hutc"
	Interval1D  = "1Dutc"
	Interval1W  = "1Wutc"
	Interval1M  = "1Mutc"
)
