package mexc

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
	"trading_assistant/pkg/exchanges"
	"trading_assistant/pkg/exchanges/types"
)

// MEXC 实现交易所接口
type MEXC struct {
	*exchanges.BaseExchange
	config    *Config
	endpoints map[string]string
}

// New 创建新的MEXC实例
func New(config *Config) (*MEXC, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	base := exchanges.NewBaseExchange("mexc", "MEXC", "v3", []string{"CN", "SG"})
	mexc := &MEXC{
		BaseExchange: base,
		config:       config.Clone(),
		endpoints:    make(map[string]string),
	}

	mexc.setCapabilities()
	mexc.setEndpoints()
	mexc.BaseExchange.SetRetryConfig(3, 100*time.Millisecond, 10*time.Second, true)
	mexc.BaseExchange.EnableRetry()

	return mexc, nil
}

// setCapabilities 设置支持的功能
func (m *MEXC) setCapabilities() {
	capabilities := map[string]bool{
		"fetchMarkets":   true,
		"fetchTicker":    true,
		"fetchTickers":   true,
		"fetchKline":     true,
		"fetchMarkPrice": false,
	}

	timeframes := map[string]string{
		"1m": Interval1m, "5m": Interval5m, "15m": Interval15m, "30m": Interval30m,
		"1h": Interval1h, "4h": Interval4h, "1d": Interval1d, "1w": Interval1w, "1M": Interval1M,
	}

	for k, v := range capabilities {
		m.BaseExchange.Has()[k] = v
	}
	for k, v := range timeframes {
		m.BaseExchange.GetTimeframes()[k] = v
	}
}

// setEndpoints 设置API端点
func (m *MEXC) setEndpoints() {
	baseURL := m.config.GetBaseURL()
	m.endpoints["base"] = baseURL
	m.endpoints["exchangeInfo"] = baseURL + EndpointExchangeInfo
	m.endpoints["ticker24hr"] = baseURL + EndpointTicker24hr
	m.endpoints["tickerPrice"] = baseURL + EndpointTickerPrice
	m.endpoints["bookTicker"] = baseURL + EndpointBookTicker
	m.endpoints["klines"] = baseURL + EndpointKlines
}

// buildQuery 构建查询字符串
func (m *MEXC) buildQuery(params map[string]interface{}) string {
	if len(params) == 0 {
		return ""
	}
	var parts []string
	for k, v := range params {
		parts = append(parts, fmt.Sprintf("%s=%v", k, v))
	}
	return strings.Join(parts, "&")
}

// GetMarketType 获取市场类型
func (m *MEXC) GetMarketType() string {
	return m.config.MarketType
}

// IsTestnet 是否测试网
func (m *MEXC) IsTestnet() bool {
	return false
}

// FetchMarkets 获取市场信息
// 支持 params["quote"] 筛选报价货币，如 params["quote"] = "USDT"
func (m *MEXC) FetchMarkets(ctx context.Context, params map[string]interface{}) ([]*types.Market, error) {
	endpoint := m.endpoints["exchangeInfo"]

	// 获取筛选参数
	var quoteFilter string
	if params != nil {
		if q, ok := params["quote"].(string); ok {
			quoteFilter = q
		}
	}

	respStr, err := m.FetchWithRetry(ctx, endpoint, "GET", nil, "")
	if err != nil {
		return nil, err
	}

	var resp struct {
		Symbols []map[string]interface{} `json:"symbols"`
	}
	if err := json.Unmarshal([]byte(respStr), &resp); err != nil {
		return nil, err
	}

	var markets []*types.Market
	for _, data := range resp.Symbols {
		market := m.parseMarket(data)
		if market != nil {
			// 应用 quote 筛选
			if quoteFilter != "" && market.Quote != quoteFilter {
				continue
			}
			markets = append(markets, market)
		}
	}
	return markets, nil
}

// parseMarket 解析市场信息
func (m *MEXC) parseMarket(data map[string]interface{}) *types.Market {
	symbol := m.SafeString(data, "symbol", "")
	if symbol == "" {
		return nil
	}

	// MEXC API 返回 status="1" 表示活跃
	status := m.SafeString(data, "status", "")
	isActive := status == "1" || status == "ENABLED"
	if !isActive {
		return nil
	}

	// 检查是否允许现货交易
	isSpotAllowed := m.SafeBool(data, "isSpotTradingAllowed", false)
	if !isSpotAllowed {
		return nil
	}

	baseCcy := m.SafeString(data, "baseAsset", "")
	quoteCcy := m.SafeString(data, "quoteAsset", "")

	// 获取精度信息
	// MEXC 返回 quotePrecision (价格精度) 和 baseAssetPrecision (数量精度)
	quotePrecision := m.SafeFloat(data, "quotePrecision", 8)
	baseAssetPrecision := m.SafeFloat(data, "baseAssetPrecision", 8)

	return &types.Market{
		ID:       symbol,
		Symbol:   fmt.Sprintf("%s/%s", baseCcy, quoteCcy),
		Base:     baseCcy,
		Quote:    quoteCcy,
		Type:     types.MarketTypeSpot,
		Active:   isActive,
		Spot:     true,
		Future:   false,
		Swap:     false,
		Contract: false,
		Precision: types.MarketPrecision{
			Price:  quotePrecision,
			Amount: baseAssetPrecision,
		},
		Info: data,
	}
}

// FetchTickers 批量获取ticker
func (m *MEXC) FetchTickers(ctx context.Context, symbols []string, params map[string]interface{}) (map[string]*types.Ticker, error) {
	endpoint := m.endpoints["ticker24hr"]

	respStr, err := m.FetchWithRetry(ctx, endpoint, "GET", nil, "")
	if err != nil {
		return nil, err
	}

	var resp []map[string]interface{}
	if err := json.Unmarshal([]byte(respStr), &resp); err != nil {
		return nil, err
	}

	tickers := make(map[string]*types.Ticker)
	symbolsMap := make(map[string]bool)
	for _, s := range symbols {
		symbolsMap[s] = true
	}

	for _, data := range resp {
		symbol := m.SafeString(data, "symbol", "")
		if symbol == "" {
			continue
		}
		if len(symbols) > 0 && !symbolsMap[symbol] {
			continue
		}
		tickers[symbol] = m.parseTicker(data, symbol)
	}
	return tickers, nil
}

// FetchBookTickers 获取最优买卖价
func (m *MEXC) FetchBookTickers(ctx context.Context, symbols []string, params map[string]interface{}) (map[string]*types.Ticker, error) {
	endpoint := m.endpoints["bookTicker"]

	respStr, err := m.FetchWithRetry(ctx, endpoint, "GET", nil, "")
	if err != nil {
		return nil, err
	}

	var resp []map[string]interface{}
	if err := json.Unmarshal([]byte(respStr), &resp); err != nil {
		return nil, err
	}

	tickers := make(map[string]*types.Ticker)
	symbolsMap := make(map[string]bool)
	for _, s := range symbols {
		symbolsMap[s] = true
	}

	for _, data := range resp {
		symbol := m.SafeString(data, "symbol", "")
		if symbol == "" {
			continue
		}
		if len(symbols) > 0 && !symbolsMap[symbol] {
			continue
		}
		tickers[symbol] = &types.Ticker{
			Symbol:    symbol,
			Bid:       m.SafeFloat(data, "bidPrice", 0),
			BidVolume: m.SafeFloat(data, "bidQty", 0),
			Ask:       m.SafeFloat(data, "askPrice", 0),
			AskVolume: m.SafeFloat(data, "askQty", 0),
			Info:      data,
		}
	}
	return tickers, nil
}

// parseTicker 解析ticker数据
func (m *MEXC) parseTicker(data map[string]interface{}, symbol string) *types.Ticker {
	return &types.Ticker{
		Symbol:      symbol,
		High:        m.SafeFloat(data, "highPrice", 0),
		Low:         m.SafeFloat(data, "lowPrice", 0),
		Bid:         m.SafeFloat(data, "bidPrice", 0),
		Ask:         m.SafeFloat(data, "askPrice", 0),
		Open:        m.SafeFloat(data, "openPrice", 0),
		Last:        m.SafeFloat(data, "lastPrice", 0),
		Close:       m.SafeFloat(data, "lastPrice", 0),
		Change:      m.SafeFloat(data, "priceChange", 0),
		Percentage:  m.SafeFloat(data, "priceChangePercent", 0) * 100, // 转换为百分比
		BaseVolume:  m.SafeFloat(data, "volume", 0),
		QuoteVolume: m.SafeFloat(data, "quoteVolume", 0),
		Info:        data,
	}
}

// FetchKlines 获取K线数据
func (m *MEXC) FetchKlines(ctx context.Context, symbol, interval string, since int64, limit int, params map[string]interface{}) ([]*types.Kline, error) {
	if symbol == "" {
		return nil, fmt.Errorf("symbol不能为空")
	}

	endpoint := m.endpoints["klines"]
	if params == nil {
		params = make(map[string]interface{})
	}
	params["symbol"] = symbol
	params["interval"] = m.convertInterval(interval)

	if limit > 0 {
		if limit > 1000 {
			limit = 1000
		}
		params["limit"] = limit
	}

	if since > 0 {
		params["startTime"] = since
	}

	query := m.buildQuery(params)
	if query != "" {
		endpoint += "?" + query
	}

	respStr, err := m.FetchWithRetry(ctx, endpoint, "GET", nil, "")
	if err != nil {
		return nil, err
	}

	var resp [][]interface{}
	if err := json.Unmarshal([]byte(respStr), &resp); err != nil {
		return nil, err
	}

	klines := make([]*types.Kline, 0, len(resp))
	for _, data := range resp {
		kline := m.parseKline(data, symbol, interval)
		if kline != nil {
			klines = append(klines, kline)
		}
	}
	return klines, nil
}

// parseKline 解析K线数据
func (m *MEXC) parseKline(data []interface{}, symbol, interval string) *types.Kline {
	if len(data) < 6 {
		return nil
	}
	toInt64 := func(v interface{}) int64 {
		switch val := v.(type) {
		case float64:
			return int64(val)
		case string:
			if n, err := strconv.ParseInt(val, 10, 64); err == nil {
				return n
			}
		}
		return 0
	}
	toFloat64 := func(v interface{}) float64 {
		switch val := v.(type) {
		case float64:
			return val
		case string:
			if n, err := strconv.ParseFloat(val, 64); err == nil {
				return n
			}
		}
		return 0
	}

	return &types.Kline{
		Symbol:    symbol,
		Timeframe: interval,
		Timestamp: toInt64(data[0]),
		Open:      toFloat64(data[1]),
		High:      toFloat64(data[2]),
		Low:       toFloat64(data[3]),
		Close:     toFloat64(data[4]),
		Volume:    toFloat64(data[5]),
		IsClosed:  true,
	}
}

// convertInterval 转换时间周期格式
func (m *MEXC) convertInterval(interval string) string {
	switch interval {
	case "1m":
		return Interval1m
	case "5m":
		return Interval5m
	case "15m":
		return Interval15m
	case "30m":
		return Interval30m
	case "1h":
		return Interval1h
	case "4h":
		return Interval4h
	case "1d":
		return Interval1d
	case "1w":
		return Interval1w
	case "1M":
		return Interval1M
	default:
		return interval
	}
}

// FetchMarkPrice 获取标记价格
func (m *MEXC) FetchMarkPrice(ctx context.Context, symbol string) (*types.MarkPrice, error) {
	return nil, fmt.Errorf("MEXC现货不支持标记价格")
}

// FetchMarkPrices 获取多个标记价格
func (m *MEXC) FetchMarkPrices(ctx context.Context, symbols []string) (map[string]*types.MarkPrice, error) {
	return nil, fmt.Errorf("MEXC现货不支持标记价格")
}
