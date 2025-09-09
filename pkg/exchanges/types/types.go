package types

import (
	"time"
)

// MetaData 元数据结构
type MetaData struct {
	Exchange  string `json:"exchange"`
	Market    string `json:"market"`
	Symbol    string `json:"symbol"`
	MarketID  string `json:"market_id"`
	DataType  string `json:"data_type"`
	Timeframe string `json:"timeframe,omitempty"`
	Subject   string `json:"subject"`
	Stream    string `json:"stream"`
	Timestamp int64  `json:"timestamp"`
}

// Response HTTP响应
type Response struct {
	StatusCode int               `json:"status_code"`
	Headers    map[string]string `json:"headers"`
	Body       []byte            `json:"body"`
}

// ========== 核心数据类型 ==========

// Market 市场信息
type Market struct {
	ID             string                 `json:"id"`               // 交易所内部ID
	Symbol         string                 `json:"symbol"`           // 标准化符号 (BTC/USDT)
	Base           string                 `json:"base"`             // 基础货币
	Quote          string                 `json:"quote"`            // 计价货币
	Settle         string                 `json:"settle,omitempty"` // 结算货币 (期货)
	Type           string                 `json:"type"`             // spot, swap, future, option
	Spot           bool                   `json:"spot"`             // 是否现货
	Margin         bool                   `json:"margin"`           // 是否支持保证金
	Swap           bool                   `json:"swap"`             // 是否永续合约
	Future         bool                   `json:"future"`           // 是否期货
	Option         bool                   `json:"option"`           // 是否期权
	Active         bool                   `json:"active"`           // 是否活跃
	Contract       bool                   `json:"contract"`         // 是否合约
	Linear         bool                   `json:"linear"`           // 是否线性合约
	Inverse        bool                   `json:"inverse"`          // 是否反向合约
	Taker          float64                `json:"taker"`            // Taker 费率
	Maker          float64                `json:"maker"`            // Maker 费率
	ContractSize   float64                `json:"contractSize"`     // 合约大小
	Expiry         int64                  `json:"expiry,omitempty"` // 到期时间
	ExpiryDatetime string                 `json:"expiryDatetime,omitempty"`
	Strike         float64                `json:"strike,omitempty"`     // 行权价 (期权)
	OptionType     string                 `json:"optionType,omitempty"` // call/put (期权)
	Precision      MarketPrecision        `json:"precision"`            // 精度信息
	Limits         MarketLimits           `json:"limits"`               // 限制信息
	Info           map[string]interface{} `json:"info"`                 // 原始信息
}

// MarketPrecision 市场精度信息
type MarketPrecision struct {
	Amount float64 `json:"amount"` // 数量精度
	Price  float64 `json:"price"`  // 价格精度
	Cost   float64 `json:"cost"`   // 成本精度
}

// MarketLimits 市场限制信息
type MarketLimits struct {
	Leverage LimitRange `json:"leverage"` // 杠杆范围
	Amount   LimitRange `json:"amount"`   // 数量范围
	Price    LimitRange `json:"price"`    // 价格范围
	Cost     LimitRange `json:"cost"`     // 成本范围
}

// LimitRange 范围限制
type LimitRange struct {
	Min  float64 `json:"min"`
	Max  float64 `json:"max"`
	Step float64 `json:"step,omitempty"` // 步长
}

// Currency 货币信息
type Currency struct {
	ID        string                 `json:"id"`        // 交易所内部ID
	Code      string                 `json:"code"`      // 标准化代码
	Name      string                 `json:"name"`      // 全名
	Active    bool                   `json:"active"`    // 是否活跃
	Deposit   bool                   `json:"deposit"`   // 是否支持充值
	Withdraw  bool                   `json:"withdraw"`  // 是否支持提现
	Fee       float64                `json:"fee"`       // 提现费用
	Precision int                    `json:"precision"` // 精度
	Limits    CurrencyLimits         `json:"limits"`    // 限制
	Networks  map[string]interface{} `json:"networks"`  // 网络信息
	Info      map[string]interface{} `json:"info"`      // 原始信息
}

// CurrencyLimits 货币限制
type CurrencyLimits struct {
	Amount   LimitRange `json:"amount"`   // 数量范围
	Withdraw LimitRange `json:"withdraw"` // 提现范围
	Deposit  LimitRange `json:"deposit"`  // 充值范围
}

// Ticker 24小时行情数据
type Ticker struct {
	Symbol        string                 `json:"symbol"`        // 交易对符号
	TimeStamp     int64                  `json:"timestamp"`     // 时间戳
	Datetime      string                 `json:"datetime"`      // ISO8601 时间
	High          float64                `json:"high"`          // 24h最高价
	Low           float64                `json:"low"`           // 24h最低价
	Bid           float64                `json:"bid"`           // 买一价
	BidVolume     float64                `json:"bidVolume"`     // 买一量
	Ask           float64                `json:"ask"`           // 卖一价
	AskVolume     float64                `json:"askVolume"`     // 卖一量
	Vwap          float64                `json:"vwap"`          // 加权平均价
	Open          float64                `json:"open"`          // 开盘价
	Close         float64                `json:"close"`         // 收盘价
	Last          float64                `json:"last"`          // 最新价
	PreviousClose float64                `json:"previousClose"` // 前收盘价
	Change        float64                `json:"change"`        // 价格变化
	Percentage    float64                `json:"percentage"`    // 变化百分比
	Average       float64                `json:"average"`       // 平均价
	BaseVolume    float64                `json:"baseVolume"`    // 基础货币成交量
	QuoteVolume   float64                `json:"quoteVolume"`   // 计价货币成交额
	MarkPrice     float64                `json:"markPrice"`     // 标记价格 (期货)
	IndexPrice    float64                `json:"indexPrice"`    // 指数价格 (期货)
	FundingRate   float64                `json:"fundingRate"`   // 资金费率 (期货)
	NextFundingAt int64                  `json:"nextFundingAt"` // 下次资金费率时间
	Info          map[string]interface{} `json:"info"`          // 原始信息
}

// Kline K线数据
type Kline struct {
	Symbol    string  `json:"symbol"`    // 交易对符号
	Timeframe string  `json:"timeframe"` // 时间周期
	Timestamp int64   `json:"timestamp"` // 开盘时间戳
	Open      float64 `json:"open"`      // 开盘价
	High      float64 `json:"high"`      // 最高价
	Low       float64 `json:"low"`       // 最低价
	Close     float64 `json:"close"`     // 收盘价
	Volume    float64 `json:"volume"`    // 成交量
	IsClosed  bool    `json:"is_closed"` // 是否已关闭
}

// Trade 交易记录
type Trade struct {
	ID           string                 `json:"id"`           // 交易ID
	Symbol       string                 `json:"symbol"`       // 交易对
	Order        string                 `json:"order"`        // 订单ID
	Type         string                 `json:"type"`         // 订单类型
	Side         string                 `json:"side"`         // buy/sell
	Amount       float64                `json:"amount"`       // 数量
	Price        float64                `json:"price"`        // 价格
	Cost         float64                `json:"cost"`         // 成本
	Fee          Fee                    `json:"fee"`          // 手续费
	Timestamp    int64                  `json:"timestamp"`    // 时间戳
	Datetime     string                 `json:"datetime"`     // ISO8601 时间
	TakerOrMaker string                 `json:"takerOrMaker"` // taker/maker
	Info         map[string]interface{} `json:"info"`         // 原始信息
}

// OrderBookSide 订单簿一侧
type OrderBookSide struct {
	Price []float64 `json:"price"` // 价格数组
	Size  []float64 `json:"size"`  // 数量数组
}

// OrderBook 订单簿
type OrderBook struct {
	Symbol    string                 `json:"symbol"`    // 交易对
	Bids      OrderBookSide          `json:"bids"`      // 买单
	Asks      OrderBookSide          `json:"asks"`      // 卖单
	TimeStamp int64                  `json:"timestamp"` // 时间戳
	Datetime  string                 `json:"datetime"`  // ISO8601 时间
	Nonce     int64                  `json:"nonce"`     // 序列号
	Info      map[string]interface{} `json:"info"`      // 原始信息
}

// Balance 账户余额
type Balance struct {
	Free  float64 `json:"free"`  // 可用余额
	Used  float64 `json:"used"`  // 冻结余额
	Total float64 `json:"total"` // 总余额
}

// Account 完整账户信息
type Account struct {
	Info      map[string]interface{} `json:"info"`      // 原始信息
	ID        string                 `json:"id"`        // 账户ID
	Type      string                 `json:"type"`      // 账户类型
	Code      string                 `json:"code"`      // 货币代码
	Balances  map[string]Balance     `json:"balances"`  // 余额信息
	Free      map[string]float64     `json:"free"`      // 可用余额汇总
	Used      map[string]float64     `json:"used"`      // 冻结余额汇总
	Total     map[string]float64     `json:"total"`     // 总余额汇总
	Timestamp int64                  `json:"timestamp"` // 时间戳
	Datetime  string                 `json:"datetime"`  // ISO8601 时间
}

// Order 订单信息
type Order struct {
	ID                 string                 `json:"id"`                 // 订单ID
	ClientOrderId      string                 `json:"clientOrderId"`      // 客户端订单ID
	Timestamp          int64                  `json:"timestamp"`          // 创建时间戳
	Datetime           string                 `json:"datetime"`           // ISO8601 时间
	LastTradeTimestamp int64                  `json:"lastTradeTimestamp"` // 最后交易时间
	Symbol             string                 `json:"symbol"`             // 交易对
	Type               string                 `json:"type"`               // 订单类型
	TimeInForce        string                 `json:"timeInForce"`        // 时效类型
	Side               string                 `json:"side"`               // buy/sell (订单方向)
	PositionSide       string                 `json:"positionSide"`       // LONG/SHORT/BOTH (持仓方向) - 双向持仓关键字段
	Amount             float64                `json:"amount"`             // 数量
	Price              float64                `json:"price"`              // 价格
	Average            float64                `json:"average"`            // 平均成交价
	Filled             float64                `json:"filled"`             // 已成交数量
	Remaining          float64                `json:"remaining"`          // 剩余数量
	Cost               float64                `json:"cost"`               // 成交金额
	Status             string                 `json:"status"`             // 订单状态
	Fee                Fee                    `json:"fee"`                // 手续费
	Trades             []Trade                `json:"trades"`             // 成交记录
	StopPrice          float64                `json:"stopPrice"`          // 止损价
	TriggerPrice       float64                `json:"triggerPrice"`       // 触发价
	TakeProfitPrice    float64                `json:"takeProfitPrice"`    // 止盈价
	StopLossPrice      float64                `json:"stopLossPrice"`      // 止损价
	Info               map[string]interface{} `json:"info"`               // 原始信息
}

// Fee 手续费信息
type Fee struct {
	Currency string  `json:"currency"` // 手续费货币
	Cost     float64 `json:"cost"`     // 手续费数额
	Rate     float64 `json:"rate"`     // 手续费率
}

// Position 持仓信息 (期货/合约)
type Position struct {
	Info                        map[string]interface{} `json:"info"`                          // 原始信息
	ID                          string                 `json:"id"`                            // 持仓ID
	Symbol                      string                 `json:"symbol"`                        // 交易对
	Timestamp                   int64                  `json:"timestamp"`                     // 时间戳
	Datetime                    string                 `json:"datetime"`                      // ISO8601 时间
	Side                        string                 `json:"side"`                          // long/short
	Size                        float64                `json:"size"`                          // 持仓大小
	Contracts                   float64                `json:"contracts"`                     // 合约数量
	ContractSize                float64                `json:"contract_size"`                 // 合约大小
	MarkPrice                   float64                `json:"mark_price"`                    // 标记价格
	EntryPrice                  float64                `json:"entry_price"`                   // 开仓价格
	NotionalValue               float64                `json:"notional"`                      // 名义价值
	Leverage                    float64                `json:"leverage"`                      // 杠杆倍数
	Collateral                  float64                `json:"collateral"`                    // 保证金
	InitialMargin               float64                `json:"initial_margin"`                // 初始保证金
	MaintenanceMargin           float64                `json:"maintenance_margin"`            // 维持保证金
	InitialMarginPercentage     float64                `json:"initial_margin_percentage"`     // 初始保证金率
	MaintenanceMarginPercentage float64                `json:"maintenance_margin_percentage"` // 维持保证金率
	MarginRatio                 float64                `json:"margin_ratio"`                  // 保证金率
	UnrealizedPnl               float64                `json:"unrealized_pnl"`                // 未实现盈亏
	RealizedPnl                 float64                `json:"realized_pnl"`                  // 已实现盈亏
	RoiPercentage               float64                `json:"roi_percentage"`                // ROI百分比
	LiquidationPrice            float64                `json:"liquidation_price"`             // 强平价格
	PositionRisk                float64                `json:"position_risk"`                 // 持仓风险
	MarginType                  string                 `json:"margin_mode"`                   // 保证金模式: ISOLATED, CROSSED
	IsolatedMargin              float64                `json:"isolated_margin"`               // 逐仓保证金
}

// FundingRate 资金费率信息
type FundingRate struct {
	Symbol               string                 `json:"symbol"`               // 交易对
	MarkPrice            float64                `json:"markPrice"`            // 标记价格
	IndexPrice           float64                `json:"indexPrice"`           // 指数价格
	InterestRate         float64                `json:"interestRate"`         // 利率
	EstimatedSettlePrice float64                `json:"estimatedSettlePrice"` // 预估结算价
	Timestamp            int64                  `json:"timestamp"`            // 时间戳
	Datetime             string                 `json:"datetime"`             // ISO8601 时间
	FundingRate          float64                `json:"fundingRate"`          // 资金费率
	FundingTimestamp     int64                  `json:"fundingTimestamp"`     // 资金费率时间戳
	FundingDatetime      string                 `json:"fundingDatetime"`      // 资金费率时间
	NextFundingRate      float64                `json:"nextFundingRate"`      // 下期资金费率
	NextFundingTimestamp int64                  `json:"nextFundingTimestamp"` // 下期资金费率时间
	NextFundingDatetime  string                 `json:"nextFundingDatetime"`  // 下期资金费率时间
	Info                 map[string]interface{} `json:"info"`                 // 原始信息
}

// TradingFee 交易费率信息
type TradingFee struct {
	Info       map[string]interface{} `json:"info"`       // 原始信息
	Symbol     string                 `json:"symbol"`     // 交易对
	Maker      float64                `json:"maker"`      // Maker费率
	Taker      float64                `json:"taker"`      // Taker费率
	Percentage bool                   `json:"percentage"` // 是否百分比
	TierBased  bool                   `json:"tierBased"`  // 是否阶梯费率
}

// DepositAddress 充值地址信息
type DepositAddress struct {
	Currency string                 `json:"currency"` // 货币
	Address  string                 `json:"address"`  // 地址
	Tag      string                 `json:"tag"`      // 标签/备注
	Network  string                 `json:"network"`  // 网络
	Info     map[string]interface{} `json:"info"`     // 原始信息
}

// Transaction 资金记录 (充值/提现)
type Transaction struct {
	Info        map[string]interface{} `json:"info"`        // 原始信息
	ID          string                 `json:"id"`          // 记录ID
	TxID        string                 `json:"txid"`        // 区块链交易ID
	Timestamp   int64                  `json:"timestamp"`   // 时间戳
	Datetime    string                 `json:"datetime"`    // ISO8601 时间
	Currency    string                 `json:"currency"`    // 货币
	Amount      float64                `json:"amount"`      // 数量
	Address     string                 `json:"address"`     // 地址
	AddressTo   string                 `json:"addressTo"`   // 目标地址
	AddressFrom string                 `json:"addressFrom"` // 来源地址
	Tag         string                 `json:"tag"`         // 标签
	TagTo       string                 `json:"tagTo"`       // 目标标签
	TagFrom     string                 `json:"tagFrom"`     // 来源标签
	Type        string                 `json:"type"`        // deposit/withdrawal
	Status      string                 `json:"status"`      // 状态
	Updated     int64                  `json:"updated"`     // 更新时间
	Fee         Fee                    `json:"fee"`         // 手续费
	Network     string                 `json:"network"`     // 网络
}

// Leverage 杠杆信息
type Leverage struct {
	Symbol        string                 `json:"symbol"`        // 交易对
	Leverage      float64                `json:"leverage"`      // 杠杆倍数
	LongLeverage  float64                `json:"longLeverage"`  // 多头杠杆
	ShortLeverage float64                `json:"shortLeverage"` // 空头杠杆
	Info          map[string]interface{} `json:"info"`          // 原始信息
}

// MarginMode 保证金模式
type MarginMode struct {
	Symbol      string                 `json:"symbol"`
	MarginMode  string                 `json:"marginMode"`
	Settle      string                 `json:"settle"`
	MaxLeverage float64                `json:"maxLeverage"`
	Info        map[string]interface{} `json:"info"`
}

// MarginModeInfo 保证金模式信息 - 别名
type MarginModeInfo = MarginMode

// ========== WebSocket 相关类型 ==========

type WatchMiniTicker struct {
	Symbol      string  `json:"symbol"`       // 交易对符号
	TimeStamp   int64   `json:"timestamp"`    // 时间戳
	Open        float64 `json:"open"`         // 开盘价
	High        float64 `json:"high"`         // 最高价
	Low         float64 `json:"low"`          // 最低价
	Close       float64 `json:"close"`        // 收盘价(最新价)
	Volume      float64 `json:"volume"`       // 成交量
	QuoteVolume float64 `json:"quote_volume"` // 计价资产成交量
}

// WatchMarkPrice WebSocket 标记价格数据 (仅期货)
type WatchMarkPrice struct {
	Symbol               string  `json:"symbol"`                 // 交易对符号
	TimeStamp            int64   `json:"timestamp"`              // 时间戳
	MarkPrice            float64 `json:"mark_price"`             // 标记价格
	IndexPrice           float64 `json:"index_price"`            // 指数价格
	FundingRate          float64 `json:"funding_rate"`           // 资金费率
	FundingTime          int64   `json:"funding_time"`           // 下次资金费用时间
	EstimatedSettlePrice float64 `json:"estimated_settle_price"` // 预估结算价
}

// WatchBookTicker WebSocket 最优买卖价数据
type WatchBookTicker struct {
	Symbol      string  `json:"symbol"`       // 交易对符号
	TimeStamp   int64   `json:"timestamp"`    // 时间戳
	BidPrice    float64 `json:"bid_price"`    // 最优买价
	BidQuantity float64 `json:"bid_quantity"` // 买量
	AskPrice    float64 `json:"ask_price"`    // 最优卖价
	AskQuantity float64 `json:"ask_quantity"` // 卖量
}

// WatchOrderBook WebSocket 订单簿数据
type WatchOrderBook struct {
	Symbol    string      `json:"symbol"`    // 交易对符号
	TimeStamp int64       `json:"timestamp"` // 时间戳
	Bids      [][]float64 `json:"bids"`      // 买盘 [价格, 数量]
	Asks      [][]float64 `json:"asks"`      // 卖盘 [价格, 数量]
	Nonce     int64       `json:"nonce"`     // 序列号
}

// WatchTrade WebSocket 交易数据
type WatchTrade struct {
	ID           string  `json:"id"`           // 交易ID
	Symbol       string  `json:"symbol"`       // 交易对符号
	Timestamp    int64   `json:"timestamp"`    // 时间戳
	Price        float64 `json:"price"`        // 价格
	Amount       float64 `json:"amount"`       // 数量
	Cost         float64 `json:"cost"`         // 成本
	Side         string  `json:"side"`         // buy/sell
	Type         string  `json:"type"`         // 订单类型
	TakerOrMaker string  `json:"takerOrMaker"` // taker/maker
	Fee          float64 `json:"fee"`          // 手续费
	FeeCurrency  string  `json:"feeCurrency"`  // 手续费货币
}

// WatchBalance WebSocket 余额数据
type WatchBalance struct {
	Account
}

// WatchOrder WebSocket 订单数据
type WatchOrder struct {
	Order
}

// ========== 交易所状态和信息 ==========

// ExchangeStatus 交易所状态
type ExchangeStatus struct {
	Status  string                 `json:"status"`  // ok/maintenance
	Updated int64                  `json:"updated"` // 更新时间
	Eta     int64                  `json:"eta"`     // 预计恢复时间
	URL     string                 `json:"url"`     // 状态页面URL
	Info    map[string]interface{} `json:"info"`    // 原始信息
}

// ExchangeTime 交易所时间
type ExchangeTime struct {
	Timestamp int64  `json:"timestamp"` // 时间戳
	Datetime  string `json:"datetime"`  // ISO8601 时间
}

// ========== 常量定义 ==========

// 市场类型
const (
	MarketTypeSpot       = "spot"
	MarketTypeMargin     = "margin"
	MarketTypeFuture     = "future"
	MarketTypeSwap       = "swap"
	MarketTypeOption     = "option"
	MarketTypeDerivative = "derivative"
	MarketTypeContract   = "contract"
	MarketTypeIndex      = "index"
	MarketTypeDelivery   = "delivery" // 交割期货
)

// 订单类型
const (
	OrderTypeMarket       = "market"
	OrderTypeLimit        = "limit"
	OrderTypeStopMarket   = "stop_market"
	OrderTypeStopLimit    = "stop_limit"
	OrderTypeTakeProfit   = "take_profit"
	OrderTypeTrailingStop = "trailing_stop"
)

// 订单方向
const (
	OrderSideBuy  = "BUY"
	OrderSideSell = "SELL"
)

// 订单状态
const (
	OrderStatusOpen            = "open"
	OrderStatusClosed          = "closed"
	OrderStatusCanceled        = "canceled"
	OrderStatusPartiallyFilled = "partially_filled"
	OrderStatusFilled          = "filled"
	OrderStatusRejected        = "rejected"
	OrderStatusExpired         = "expired"
)

// 时效类型
const (
	TimeInForceGTC = "GTC" // Good Till Canceled
	TimeInForceIOC = "IOC" // Immediate Or Cancel
	TimeInForceFOK = "FOK" // Fill Or Kill
	TimeInForceGTD = "GTD" // Good Till Date
	TimeInForcePO  = "PO"  // Post Only
)

// 持仓方向
const (
	PositionSideLong  = "long"
	PositionSideShort = "short"
	PositionSideBoth  = "both"
)

// 保证金模式
const (
	MarginModeIsolated = "ISOLATED"
	MarginModeCross    = "CROSS"
	MarginModeCrossed  = "CROSSED"
)

// 交易方向
const (
	TradeSideBuy  = "BUY"
	TradeSideSell = "SELL"
)

// 交易记录类型
const (
	TakerOrMakerTaker = "taker"
	TakerOrMakerMaker = "maker"
)

// 精度模式
const (
	PrecisionModeDecimalPlaces     = 0
	PrecisionModeSignificantDigits = 1
	PrecisionModeTickSize          = 2
)

// 填充模式
const (
	PaddingModeNone = 0
	PaddingModeZero = 1
)

// 合约类型
const (
	ContractTypeSpot    = "spot"
	ContractTypeLinear  = "linear"
	ContractTypeInverse = "inverse"
	ContractTypeOption  = "option"
)

// WebSocket
const (
	UserStreamBalance = "outboundAccountPosition"
	UserStreamOrders  = "executionReport"
	UserStreamPrefix  = "user@"
)

// PriceLevel 价格层级
type PriceLevel struct {
	Price  float64 `json:"price"`
	Amount float64 `json:"amount"`
}

// LeverageInfo 杠杆信息
type LeverageInfo struct {
	Symbol   string                 `json:"symbol"`   // 交易对
	Leverage int                    `json:"leverage"` // 杠杆倍数
	Info     map[string]interface{} `json:"info"`     // 原始信息
}

// ========== 辅助函数 ==========

// IsExpired 检查是否已过期 (期货)
func (m *Market) IsExpired() bool {
	if m.Expiry == 0 {
		return false
	}
	return time.Now().Unix() > m.Expiry
}

// GetContractValue 计算合约价值
func (p *Position) GetContractValue() float64 {
	return p.Contracts * p.ContractSize * p.MarkPrice
}

// CalculatePnl 计算盈亏
func (p *Position) CalculatePnl() float64 {
	if p.Side == PositionSideLong {
		return (p.MarkPrice - p.EntryPrice) * p.Size
	}
	return (p.EntryPrice - p.MarkPrice) * p.Size
}

// IsLiquidationRisk 检查是否有强平风险
func (p *Position) IsLiquidationRisk(threshold float64) bool {
	if p.LiquidationPrice == 0 {
		return false
	}

	if p.Side == PositionSideLong {
		return p.MarkPrice <= p.LiquidationPrice*(1+threshold)
	}
	return p.MarkPrice >= p.LiquidationPrice*(1-threshold)
}

// ========== 用户数据流结构体 ==========

// WatchBalanceUpdate 账户余额更新数据
type WatchBalanceUpdate struct {
	EventType          string                 `json:"eventType"`          // 事件类型
	EventTime          int64                  `json:"eventTime"`          // 事件时间
	Symbol             string                 `json:"symbol"`             // 交易对
	Free               float64                `json:"free"`               // 可用余额
	Locked             float64                `json:"locked"`             // 冻结余额
	WalletBalance      float64                `json:"walletBalance"`      // 钱包余额 (期货)
	CrossWalletBalance float64                `json:"crossWalletBalance"` // 全仓钱包余额 (期货)
	BalanceChange      float64                `json:"balanceChange"`      // 余额变化量 (期货)
	Asset              string                 `json:"asset"`              // 资产名称
	ClearTime          int64                  `json:"clearTime"`          // 清算时间 (期货)
	Info               map[string]interface{} `json:"info"`               // 原始信息
}

// WatchOrderUpdate 订单更新数据
type WatchOrderUpdate struct {
	EventType          string                 `json:"eventType"`          // 事件类型
	EventTime          int64                  `json:"eventTime"`          // 事件时间
	Symbol             string                 `json:"symbol"`             // 交易对
	ClientOrderID      string                 `json:"clientOrderId"`      // 客户端订单ID
	Side               string                 `json:"side"`               // 买卖方向
	OrderType          string                 `json:"orderType"`          // 订单类型
	TimeInForce        string                 `json:"timeInForce"`        // 有效时间类型
	OriginalQuantity   float64                `json:"originalQuantity"`   // 原始数量
	OriginalPrice      float64                `json:"originalPrice"`      // 原始价格
	AveragePrice       float64                `json:"averagePrice"`       // 平均成交价格
	StopPrice          string                 `json:"stopPrice"`          // 止损价格
	ExecutionType      string                 `json:"executionType"`      // 本次事件的具体执行类型
	OrderStatus        string                 `json:"orderStatus"`        // 订单的当前状态
	OrderID            int64                  `json:"orderId"`            // 订单ID
	LastQuantityFilled float64                `json:"lastQuantityFilled"` // 成交数量
	FilledAccumulated  float64                `json:"filledAccumulated"`  // 累计成交数量
	LastPriceFilled    float64                `json:"lastPriceFilled"`    // 成交价格
	CommissionAmount   string                 `json:"commissionAmount"`   // 手续费数量
	CommissionAsset    string                 `json:"commissionAsset"`    // 手续费资产类型
	TradeTime          int64                  `json:"tradeTime"`          // 成交时间
	TradeID            int64                  `json:"tradeId"`            // 成交ID
	BidsNotional       string                 `json:"bidsNotional"`       // 买单净值
	AsksNotional       string                 `json:"asksNotional"`       // 卖单净值
	IsMakerSide        bool                   `json:"isMakerSide"`        // 该成交是作为挂单成交吗？
	IsReduceOnly       bool                   `json:"isReduceOnly"`       // 是否为只减仓单
	WorkingType        string                 `json:"workingType"`        // 条件价格触发类型
	OriginalOrderType  string                 `json:"originalOrderType"`  // 原始订单类型
	PositionSide       string                 `json:"positionSide"`       // 持仓方向
	IsClosePosition    bool                   `json:"isClosePosition"`    // 是否条件全平仓
	ActivationPrice    string                 `json:"activationPrice"`    // 跟踪止损激活价格
	CallbackRate       string                 `json:"callbackRate"`       // 跟踪止损回调比例
	RealizedProfit     float64                `json:"realizedProfit"`     // 该交易实现盈亏
	Info               map[string]interface{} `json:"info"`               // 原始信息
}

// WatchPositionUpdate 仓位更新数据
type WatchPositionUpdate struct {
	EventType              string                 `json:"eventType"`              // 事件类型
	EventTime              int64                  `json:"eventTime"`              // 事件时间
	Symbol                 string                 `json:"symbol"`                 // 交易对
	PositionAmount         float64                `json:"positionAmount"`         // 持仓数量
	EntryPrice             float64                `json:"entryPrice"`             // 持仓成本
	PreAccumulatedRealized float64                `json:"preAccumulatedRealized"` // 历史累计实现盈亏
	UnrealizedPnl          float64                `json:"unrealizedPnl"`          // 持仓未实现盈亏
	MarginType             string                 `json:"marginType"`             // 保证金模式
	IsolatedWallet         float64                `json:"isolatedWallet"`         // 逐仓钱包余额
	PositionSide           string                 `json:"positionSide"`           // 持仓方向
	Info                   map[string]interface{} `json:"info"`                   // 原始信息
}

// WatchAccountUpdate 账户配置更新数据
type WatchAccountUpdate struct {
	EventType       string                 `json:"eventType"`       // 事件类型
	EventTime       int64                  `json:"eventTime"`       // 事件时间
	TransactionTime int64                  `json:"transactionTime"` // 交易时间
	Balances        []WatchBalanceUpdate   `json:"balances"`        // 余额信息
	Positions       []WatchPositionUpdate  `json:"positions"`       // 持仓信息
	Info            map[string]interface{} `json:"info"`            // 原始信息
}
