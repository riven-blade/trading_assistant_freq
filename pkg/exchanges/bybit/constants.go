package bybit

// ========== Bybit API 端点常数 ==========

// Bybit API 基础URL
const (
	SpotBaseURL       = "https://api.bybit.com"
	FuturesBaseURL    = "https://api.bybit.com"
	TestNetBaseURL    = "https://api-testnet.bybit.com"
	TestNetFuturesURL = "https://api-testnet.bybit.com"
)

// Bybit WebSocket URL
const (
	SpotWebSocketURL           = "wss://stream.bybit.com/v5/public/spot"
	FuturesWebSocketURL        = "wss://stream.bybit.com/v5/public/linear"
	TestNetWebSocketURL        = "wss://stream-testnet.bybit.com/v5/public/spot"
	TestNetFuturesWebSocketURL = "wss://stream-testnet.bybit.com/v5/public/linear"
)

// ========== Bybit WebSocket 事件类型常数 ==========

// Bybit WebSocket 响应消息中的主题类型
const (
	// 市场数据流事件
	TopicTrade       = "publicTrade" // 交易流
	TopicTicker      = "tickers"     // 24小时价格统计
	TopicOrderbook   = "orderbook"   // 订单簿
	TopicKline       = "kline"       // K线
	TopicLiquidation = "liquidation" // 强平

	// 用户数据流事件
	TopicWallet    = "wallet"    // 钱包余额更新
	TopicPosition  = "position"  // 持仓更新
	TopicExecution = "execution" // 成交更新
	TopicOrder     = "order"     // 订单更新
)

// ========== Bybit WebSocket 字段名称常量 ==========

// 通用字段
const (
	FieldTopic  = "topic"  // 主题
	FieldType   = "type"   // 类型
	FieldData   = "data"   // 数据
	FieldTS     = "ts"     // 时间戳
	FieldSymbol = "symbol" // 交易对
)

// Ticker 相关字段
const (
	FieldLastPrice    = "lastPrice"    // 最新价格
	FieldPrevPrice24h = "prevPrice24h" // 24小时前价格
	FieldPrice24hPcnt = "price24hPcnt" // 24小时涨跌幅
	FieldHighPrice24h = "highPrice24h" // 24小时最高价
	FieldLowPrice24h  = "lowPrice24h"  // 24小时最低价
	FieldVolume24h    = "volume24h"    // 24小时成交量
	FieldTurnover24h  = "turnover24h"  // 24小时成交额
	FieldBid1Price    = "bid1Price"    // 买一价
	FieldBid1Size     = "bid1Size"     // 买一量
	FieldAsk1Price    = "ask1Price"    // 卖一价
	FieldAsk1Size     = "ask1Size"     // 卖一量
)

// K线相关字段
const (
	FieldStart    = "start"    // 开始时间
	FieldEnd      = "end"      // 结束时间
	FieldInterval = "interval" // 时间间隔
	FieldOpen     = "open"     // 开盘价
	FieldHigh     = "high"     // 最高价
	FieldLow      = "low"      // 最低价
	FieldClose    = "close"    // 收盘价
	FieldVolume   = "volume"   // 成交量
	FieldTurnover = "turnover" // 成交额
	FieldConfirm  = "confirm"  // 是否确认
)

// 交易相关字段
const (
	FieldExecId       = "execId"       // 成交ID
	FieldPrice        = "price"        // 价格
	FieldSize         = "size"         // 数量
	FieldSide         = "side"         // 方向
	FieldExecTime     = "execTime"     // 成交时间
	FieldIsBlockTrade = "isBlockTrade" // 是否大宗交易
)

// 订单簿相关字段
const (
	FieldBids     = "b"   // 买盘
	FieldAsks     = "a"   // 卖盘
	FieldUpdateId = "u"   // 更新ID
	FieldSeq      = "seq" // 序列号
)

// WebSocket 协议字段
const (
	FieldOp      = "op"       // 操作类型
	FieldArgs    = "args"     // 参数
	FieldReqId   = "req_id"   // 请求ID
	FieldRetCode = "ret_code" // 返回码
	FieldRetMsg  = "ret_msg"  // 返回消息
	FieldConnId  = "conn_id"  // 连接ID
)

// WebSocket 操作类型常量
const (
	OpSubscribe   = "subscribe"   // 订阅
	OpUnsubscribe = "unsubscribe" // 取消订阅
	OpAuth        = "auth"        // 认证
	OpPing        = "ping"        // 心跳
)

// WebSocket 订阅流名称模板常量
const (
	StreamTemplateTicker       = "tickers.%s"       // 24小时ticker流模板
	StreamTemplateOrderbook    = "orderbook.1.%s"   // 订单簿流模板 (深度1)
	StreamTemplateOrderbook50  = "orderbook.50.%s"  // 订单簿流模板 (深度50)
	StreamTemplateOrderbook200 = "orderbook.200.%s" // 订单簿流模板 (深度200)
	StreamTemplateTrade        = "publicTrade.%s"   // 交易流模板
	StreamTemplateKline        = "kline.%s.%s"      // K线流模板 (interval, symbol)
	StreamTemplateLiquidation  = "liquidation.%s"   // 强平流模板
)

// 私有流模板
const (
	StreamTemplateWallet    = "wallet"    // 钱包流
	StreamTemplatePosition  = "position"  // 持仓流
	StreamTemplateExecution = "execution" // 成交流
	StreamTemplateOrder     = "order"     // 订单流
)

// ========== Bybit API 版本和路径常量 ==========

// API 版本
const (
	APIVersionV5 = "v5"
)

// API 路径前缀
const (
	PathMarket   = "/v5/market"   // 市场数据
	PathTrade    = "/v5/order"    // 交易
	PathAccount  = "/v5/account"  // 账户
	PathPosition = "/v5/position" // 持仓
	PathAsset    = "/v5/asset"    // 资产
)

// 具体 API 端点
const (
	// 市场数据
	EndpointInstrumentsInfo = "/v5/market/instruments-info" // 交易规则查询
	EndpointTickers         = "/v5/market/tickers"          // 24小时价格统计
	EndpointKline           = "/v5/market/kline"            // K线数据
	EndpointOrderbook       = "/v5/market/orderbook"        // 订单簿
	EndpointRecentTrade     = "/v5/market/recent-trade"     // 最新交易

	// 交易
	EndpointPlaceOrder    = "/v5/order/create"   // 下单
	EndpointCancelOrder   = "/v5/order/cancel"   // 撤单
	EndpointOrderHistory  = "/v5/order/history"  // 订单历史
	EndpointOrderRealtime = "/v5/order/realtime" // 实时订单

	// 账户
	EndpointWalletBalance = "/v5/account/wallet-balance" // 账户余额

	// 持仓
	EndpointPositionInfo = "/v5/position/list"           // 持仓信息
	EndpointSetLeverage  = "/v5/position/set-leverage"   // 设置杠杆
	EndpointSwitchMode   = "/v5/position/switch-mode"    // 切换持仓模式
	EndpointSetRiskLimit = "/v5/position/set-risk-limit" // 设置风险限额
)

// ========== Bybit 业务常量 ==========

// 产品类型
const (
	CategorySpot    = "spot"    // 现货
	CategoryLinear  = "linear"  // USDT永续
	CategoryInverse = "inverse" // 币本位永续
	CategoryOption  = "option"  // 期权
)

// 订单类型
const (
	OrderTypeMarket = "Market" // 市价单
	OrderTypeLimit  = "Limit"  // 限价单
)

// 订单方向
const (
	SideBuy  = "Buy"  // 买入
	SideSell = "Sell" // 卖出
)

// 持仓方向 (双向持仓模式)
const (
	PositionIdxBuy  = 1 // 做多
	PositionIdxSell = 2 // 做空
	PositionIdxBoth = 0 // 单向持仓
)

// 订单状态
const (
	OrderStatusNew             = "New"             // 新订单
	OrderStatusPartiallyFilled = "PartiallyFilled" // 部分成交
	OrderStatusFilled          = "Filled"          // 完全成交
	OrderStatusCancelled       = "Cancelled"       // 已取消
	OrderStatusRejected        = "Rejected"        // 已拒绝
)

// 时间有效性
const (
	TimeInForceGTC = "GTC"      // Good Till Cancelled
	TimeInForceIOC = "IOC"      // Immediate or Cancel
	TimeInForceFOK = "FOK"      // Fill or Kill
	TimeInForcePO  = "PostOnly" // Post Only
)

// 保证金模式
const (
	MarginModeIsolated = "ISOLATED_MARGIN" // 逐仓保证金
	MarginModeCross    = "REGULAR_MARGIN"  // 全仓保证金
)

// 持仓模式
const (
	PositionModeBothSide = 3 // 双向持仓
	PositionModeOneSide  = 0 // 单向持仓
)

// K线时间间隔
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
