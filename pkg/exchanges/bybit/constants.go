package bybit

// ========== Bybit API 端点常数 ==========

// Bybit API 基础URL
const (
	BaseURL        = "https://api.bybit.com"
	TestNetBaseURL = "https://api-testnet.bybit.com"
)

// ========== Bybit REST API 端点 ==========

// 市场数据端点
const (
	EndpointInstrumentsInfo = "/v5/market/instruments-info" // 交易规则查询
	EndpointTickers         = "/v5/market/tickers"          // 24小时价格统计
	EndpointKline           = "/v5/market/kline"            // K线数据
	EndpointServerTime      = "/v5/market/time"             // 服务器时间
)

// ========== Bybit 业务常量 ==========

// 产品类型
const (
	CategorySpot    = "spot"    // 现货
	CategoryLinear  = "linear"  // USDT永续
	CategoryInverse = "inverse" // 币本位永续
)

// ========== K线时间间隔 ==========

const (
	Interval1m  = "1"   // 1分钟
	Interval3m  = "3"   // 3分钟
	Interval5m  = "5"   // 5分钟
	Interval15m = "15"  // 15分钟
	Interval30m = "30"  // 30分钟
	Interval1h  = "60"  // 1小时
	Interval2h  = "120" // 2小时
	Interval4h  = "240" // 4小时
	Interval6h  = "360" // 6小时
	Interval12h = "720" // 12小时
	Interval1d  = "D"   // 1天
	Interval1w  = "W"   // 1周
	Interval1M  = "M"   // 1月
)
