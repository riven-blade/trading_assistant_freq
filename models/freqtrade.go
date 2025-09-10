package models

import "time"

// FreqtradeController 相关模型定义

// LoginResponse Freqtrade 登录响应
type LoginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	TokenType    string `json:"token_type,omitempty"`
}

// ForceBuyPayload 强制买入载荷
type ForceBuyPayload struct {
	Pair        string  `json:"pair"`
	Price       float64 `json:"price,omitempty"`
	OrderType   string  `json:"ordertype,omitempty"`   // market, limit
	StakeAmount float64 `json:"stakeamount,omitempty"` // 投入金额
	EntryTag    string  `json:"entry_tag,omitempty"`   // 入场标签
	Side        string  `json:"side,omitempty"`        // long, short
	Leverage    int     `json:"leverage,omitempty"`    // 杠杆倍数
}

// ForceAdjustBuyPayload 强制调整买入载荷
type ForceAdjustBuyPayload struct {
	Pair        string  `json:"pair"`
	Price       float64 `json:"price"`
	OrderType   string  `json:"ordertype"`   // limit, market
	Side        string  `json:"side"`        // long, short
	EntryTag    string  `json:"entry_tag"`   // 入场标签
	StakeAmount float64 `json:"stakeamount"` // 投入金额
}

// ForceSellPayload 强制卖出载荷
type ForceSellPayload struct {
	TradeId   string `json:"tradeid"`   // 交易ID
	OrderType string `json:"ordertype"` // market, limit
	Amount    string `json:"amount"`    // 卖出数量，可以是 "half", "all" 或具体数字
}

// PositionStatus 持仓状态
type PositionStatus struct {
	DryRun          bool   `json:"dry_run"`
	MaxOpenTrades   int    `json:"max_open_trades"`
	MinimumBalance  int    `json:"minimum_balance"`
	OpenTradeCount  int    `json:"open_trade_count"`
	StakeAmount     int    `json:"stake_amount"`
	StakeCurrency   string `json:"stake_currency"`
	StartingBalance int    `json:"starting_balance"`
	StateSince      int64  `json:"state_since"`
	TradingMode     string `json:"trading_mode"`
	Max             int    `json:"max"` // 最大持仓数量
}

// TradePosition 交易持仓
type TradePosition struct {
	TradeId            int              `json:"trade_id"`
	Pair               string           `json:"pair"`
	IsOpen             bool             `json:"is_open"`
	ExchangeOrderId    string           `json:"exchange_order_id"`
	Strategy           string           `json:"strategy"`
	Timeframe          int              `json:"timeframe"` // freqtrade返回的是数字（分钟数）
	Amount             float64          `json:"amount"`
	AmountRequested    float64          `json:"amount_requested"`
	OpenDate           string           `json:"open_date"`
	OpenTimestamp      int64            `json:"open_timestamp"`
	OpenRate           float64          `json:"open_rate"`
	OpenOrderType      string           `json:"open_order_type"`
	OpenFee            float64          `json:"open_fee"`
	CloseDate          *string          `json:"close_date"`
	CloseTimestamp     *int64           `json:"close_timestamp"`
	CloseRate          *float64         `json:"close_rate"`
	CloseOrderType     *string          `json:"close_order_type"`
	CloseFee           *float64         `json:"close_fee"`
	CloseProfit        *float64         `json:"close_profit"`
	CloseProfitAbs     *float64         `json:"close_profit_abs"`
	TradeDirection     string           `json:"trade_direction"` // long, short
	Leverage           *float64         `json:"leverage"`
	InterestRate       *float64         `json:"interest_rate"`
	LiquidationPrice   *float64         `json:"liquidation_price"`
	IsShort            bool             `json:"is_short"`
	TradingMode        string           `json:"trading_mode"`
	FundingFees        *float64         `json:"funding_fees"`
	RealizedProfit     *float64         `json:"realized_profit"`
	CurrentProfit      float64          `json:"current_profit"`
	CurrentProfitAbs   float64          `json:"current_profit_abs"`
	CurrentProfitPct   float64          `json:"current_profit_pct"`
	CurrentRate        float64          `json:"current_rate"`
	InitialStopLoss    *float64         `json:"initial_stop_loss"`
	InitialStopLossPct *float64         `json:"initial_stop_loss_pct"`
	StopLoss           *float64         `json:"stop_loss"`
	StopLossPct        *float64         `json:"stop_loss_pct"`
	MinRate            float64          `json:"min_rate"`
	MaxRate            float64          `json:"max_rate"`
	EntryTag           *string          `json:"entry_tag"`
	ExitReason         *string          `json:"exit_reason"`
	ExitOrderStatus    *string          `json:"exit_order_status"`
	StakeAmount        float64          `json:"stake_amount"`
	HasOpenOrders      bool             `json:"has_open_orders"`
	Orders             []FreqtradeOrder `json:"orders"`
}

// FreqtradeOrder Freqtrade 订单信息
type FreqtradeOrder struct {
	OrderId              string   `json:"order_id"`
	OrderType            string   `json:"order_type"`
	OrderTimestamp       int64    `json:"order_timestamp"`
	OrderFilled          bool     `json:"order_filled"`
	OrderFillTimestamp   *int64   `json:"order_fill_timestamp"`
	OrderUpdateTimestamp *int64   `json:"order_update_timestamp"`
	Side                 string   `json:"side"`
	Amount               float64  `json:"amount"`
	Price                float64  `json:"price"`
	AveragePrice         *float64 `json:"average"`
	Cost                 *float64 `json:"cost"`
	Filled               float64  `json:"filled"`
	Remaining            float64  `json:"remaining"`
	Status               string   `json:"status"`
	Fee                  *float64 `json:"fee"`
	IsOpen               bool     `json:"is_open"`
}

// WhitelistResponse 白名单响应
type WhitelistResponse struct {
	Whitelist []string `json:"whitelist"`
	Length    int      `json:"length"`
	Method    []string `json:"method"`
}

// FreqtradeMonitorPair 监控交易对 (用于 Redis 存储)
type FreqtradeMonitorPair struct {
	Symbol    string    `json:"symbol"` // 交易对符号
	Side      string    `json:"side"`   // long, short
	Price     float64   `json:"price"`  // 目标价格
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
