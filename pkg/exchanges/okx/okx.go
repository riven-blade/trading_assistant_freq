package okx

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

// OKX 实现交易所接口 (仅公共市场数据)
type OKX struct {
	*exchanges.BaseExchange
	config   *Config
	instType string // 产品类型：SPOT, SWAP, FUTURES

	endpoints map[string]string
}

// New 创建新的OKX实例
func New(config *Config) (*OKX, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	base := exchanges.NewBaseExchange("okx", "OKX", "v5", []string{"SC"})
	okx := &OKX{
		BaseExchange: base,
		config:       config.Clone(),
		instType:     config.InstType,
		endpoints:    make(map[string]string),
	}

	okx.setCapabilities()
	okx.setEndpoints()
	okx.BaseExchange.SetRetryConfig(3, 100*time.Millisecond, 10*time.Second, true)
	okx.BaseExchange.EnableRetry()

	return okx, nil
}

// setCapabilities 设置支持的功能
func (o *OKX) setCapabilities() {
	capabilities := map[string]bool{
		"fetchMarkets":   true,
		"fetchTicker":    true,
		"fetchTickers":   true,
		"fetchKline":     true,
		"fetchMarkPrice": o.config.IsFutures(),
	}

	timeframes := map[string]string{
		"1m": Interval1m, "3m": Interval3m, "5m": Interval5m,
		"15m": Interval15m, "30m": Interval30m,
		"1h": Interval1H, "2h": Interval2H, "4h": Interval4H,
		"6h": Interval6H, "12h": Interval12H,
		"1d": Interval1D, "1w": Interval1W, "1M": Interval1M,
	}

	for k, v := range capabilities {
		o.BaseExchange.Has()[k] = v
	}
	for k, v := range timeframes {
		o.BaseExchange.GetTimeframes()[k] = v
	}
}

// setEndpoints 设置API端点
func (o *OKX) setEndpoints() {
	baseURL := o.config.GetBaseURL()
	o.endpoints["base"] = baseURL
	o.endpoints["instruments"] = baseURL + EndpointInstruments
	o.endpoints["tickers"] = baseURL + EndpointTickers
	o.endpoints["ticker"] = baseURL + EndpointTicker
	o.endpoints["klines"] = baseURL + EndpointKlines
	o.endpoints["markPrice"] = baseURL + EndpointMarkPrice
	o.endpoints["fundingRate"] = baseURL + EndpointFundingRate
}

// ========== 公共API方法 ==========

// buildQuery 构建查询字符串
func (o *OKX) buildQuery(params map[string]interface{}) string {
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
func (o *OKX) GetMarketType() string {
	return o.config.MarketType
}

// IsTestnet 是否测试网
func (o *OKX) IsTestnet() bool {
	return false // OKX公共API无测试网区分
}

// FetchMarkets 获取市场信息
// 支持 params["quote"] 筛选报价货币，如 params["quote"] = "USDT"
func (o *OKX) FetchMarkets(ctx context.Context, params map[string]interface{}) ([]*types.Market, error) {
	endpoint := o.endpoints["instruments"]

	// 获取筛选参数（在修改 params 之前）
	var quoteFilter string
	if params != nil {
		if q, ok := params["quote"].(string); ok {
			quoteFilter = q
			delete(params, "quote") // 从 params 中删除，避免传给 API
		}
	}

	if params == nil {
		params = make(map[string]interface{})
	}
	params["instType"] = o.instType

	query := o.buildQuery(params)
	if query != "" {
		endpoint += "?" + query
	}

	respStr, err := o.FetchWithRetry(ctx, endpoint, "GET", nil, "")
	if err != nil {
		return nil, err
	}

	var resp struct {
		Code string                   `json:"code"`
		Msg  string                   `json:"msg"`
		Data []map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal([]byte(respStr), &resp); err != nil {
		return nil, err
	}

	if resp.Code != "0" {
		return nil, fmt.Errorf("okx api error: %s", resp.Msg)
	}

	var markets []*types.Market
	for _, data := range resp.Data {
		market := o.parseMarket(data)
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
func (o *OKX) parseMarket(data map[string]interface{}) *types.Market {
	instId := o.SafeString(data, "instId", "")
	if instId == "" {
		return nil
	}

	state := o.SafeString(data, "state", "")
	if state != "live" {
		return nil
	}

	baseCcy := o.SafeString(data, "baseCcy", "")
	quoteCcy := o.SafeString(data, "quoteCcy", "")

	// 永续合约特殊处理
	if o.instType == InstTypeSwap {
		ctValCcy := o.SafeString(data, "ctValCcy", "")
		settleCcy := o.SafeString(data, "settleCcy", "")
		if baseCcy == "" {
			baseCcy = ctValCcy
		}
		if quoteCcy == "" {
			quoteCcy = settleCcy
		}
	}

	isSpot := o.instType == InstTypeSpot
	isFuture := o.instType == InstTypeSwap || o.instType == InstTypeFutures

	return &types.Market{
		ID:       instId,
		Symbol:   fmt.Sprintf("%s/%s", baseCcy, quoteCcy),
		Base:     baseCcy,
		Quote:    quoteCcy,
		Type:     o.config.MarketType,
		Active:   state == "live",
		Spot:     isSpot,
		Future:   isFuture,
		Swap:     o.instType == InstTypeSwap,
		Contract: isFuture,
		Linear:   isFuture && o.SafeString(data, "ctType", "") == "linear",
		Info:     data,
		Precision: types.MarketPrecision{
			Price:  o.SafeFloat(data, "tickSz", 0),
			Amount: o.SafeFloat(data, "lotSz", 0),
		},
	}
}

// FetchTickers 批量获取ticker
func (o *OKX) FetchTickers(ctx context.Context, symbols []string, params map[string]interface{}) (map[string]*types.Ticker, error) {
	endpoint := o.endpoints["tickers"]

	if params == nil {
		params = make(map[string]interface{})
	}
	params["instType"] = o.instType

	query := o.buildQuery(params)
	if query != "" {
		endpoint += "?" + query
	}

	respStr, err := o.FetchWithRetry(ctx, endpoint, "GET", nil, "")
	if err != nil {
		return nil, err
	}

	var resp struct {
		Code string                   `json:"code"`
		Msg  string                   `json:"msg"`
		Data []map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal([]byte(respStr), &resp); err != nil {
		return nil, err
	}

	if resp.Code != "0" {
		return nil, fmt.Errorf("okx api error: %s", resp.Msg)
	}

	tickers := make(map[string]*types.Ticker)
	symbolsMap := make(map[string]bool)
	for _, s := range symbols {
		symbolsMap[s] = true
	}

	for _, data := range resp.Data {
		instId := o.SafeString(data, "instId", "")
		if instId == "" {
			continue
		}
		if len(symbols) > 0 && !symbolsMap[instId] {
			continue
		}
		tickers[instId] = o.parseTicker(data, instId)
	}
	return tickers, nil
}

// FetchBookTickers 获取最优买卖价
func (o *OKX) FetchBookTickers(ctx context.Context, symbols []string, params map[string]interface{}) (map[string]*types.Ticker, error) {
	return o.FetchTickers(ctx, symbols, params)
}

// parseTicker 解析ticker数据
func (o *OKX) parseTicker(data map[string]interface{}, instId string) *types.Ticker {
	ts := o.SafeInteger(data, "ts", 0)
	lastPrice := o.SafeFloat(data, "last", 0)
	openPrice := o.SafeFloat(data, "open24h", 0)

	// 计算涨跌幅
	change := lastPrice - openPrice
	percentage := 0.0
	if openPrice > 0 {
		percentage = (change / openPrice) * 100
	}

	return &types.Ticker{
		Symbol:      instId,
		TimeStamp:   ts,
		Datetime:    o.ISO8601(ts),
		High:        o.SafeFloat(data, "high24h", 0),
		Low:         o.SafeFloat(data, "low24h", 0),
		Bid:         o.SafeFloat(data, "bidPx", 0),
		BidVolume:   o.SafeFloat(data, "bidSz", 0),
		Ask:         o.SafeFloat(data, "askPx", 0),
		AskVolume:   o.SafeFloat(data, "askSz", 0),
		Open:        openPrice,
		Last:        lastPrice,
		Close:       lastPrice,
		Change:      change,
		Percentage:  percentage,
		BaseVolume:  o.SafeFloat(data, "vol24h", 0),
		QuoteVolume: o.SafeFloat(data, "volCcy24h", 0),
		Info:        data,
	}
}

// FetchKlines 获取K线数据
func (o *OKX) FetchKlines(ctx context.Context, symbol, interval string, since int64, limit int, params map[string]interface{}) ([]*types.Kline, error) {
	if symbol == "" {
		return nil, fmt.Errorf("symbol不能为空")
	}

	endpoint := o.endpoints["klines"]
	if params == nil {
		params = make(map[string]interface{})
	}
	params["instId"] = symbol
	params["bar"] = o.convertInterval(interval)

	if limit > 0 {
		if limit > 300 {
			limit = 300 // OKX最大限制
		}
		params["limit"] = limit
	}

	if since > 0 {
		params["after"] = since
	}

	query := o.buildQuery(params)
	if query != "" {
		endpoint += "?" + query
	}

	respStr, err := o.FetchWithRetry(ctx, endpoint, "GET", nil, "")
	if err != nil {
		return nil, err
	}

	var resp struct {
		Code string          `json:"code"`
		Msg  string          `json:"msg"`
		Data [][]interface{} `json:"data"`
	}
	if err := json.Unmarshal([]byte(respStr), &resp); err != nil {
		return nil, err
	}

	if resp.Code != "0" {
		return nil, fmt.Errorf("okx api error: %s", resp.Msg)
	}

	// OKX返回数据是倒序的，需要反转
	klines := make([]*types.Kline, 0, len(resp.Data))
	for i := len(resp.Data) - 1; i >= 0; i-- {
		kline := o.parseKline(resp.Data[i], symbol, interval)
		if kline != nil {
			klines = append(klines, kline)
		}
	}
	return klines, nil
}

// parseKline 解析K线数据
func (o *OKX) parseKline(data []interface{}, symbol, interval string) *types.Kline {
	if len(data) < 6 {
		return nil
	}
	// OKX K线格式: [ts, o, h, l, c, vol, volCcy, volCcyQuote, confirm]
	toInt64 := func(v interface{}) int64 {
		switch val := v.(type) {
		case string:
			if n, err := strconv.ParseInt(val, 10, 64); err == nil {
				return n
			}
		case float64:
			return int64(val)
		}
		return 0
	}
	toFloat64 := func(v interface{}) float64 {
		switch val := v.(type) {
		case string:
			if n, err := strconv.ParseFloat(val, 64); err == nil {
				return n
			}
		case float64:
			return val
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
func (o *OKX) convertInterval(interval string) string {
	switch interval {
	case "1m":
		return Interval1m
	case "3m":
		return Interval3m
	case "5m":
		return Interval5m
	case "15m":
		return Interval15m
	case "30m":
		return Interval30m
	case "1h":
		return Interval1H
	case "2h":
		return Interval2H
	case "4h":
		return Interval4H
	case "6h":
		return Interval6H
	case "12h":
		return Interval12H
	case "1d":
		return Interval1D
	case "1w":
		return Interval1W
	case "1M":
		return Interval1M
	default:
		return interval
	}
}

// FetchMarkPrice 获取单个交易对的标记价格
func (o *OKX) FetchMarkPrice(ctx context.Context, symbol string) (*types.MarkPrice, error) {
	if !o.config.IsFutures() {
		return nil, fmt.Errorf("标记价格仅在期货模式下可用")
	}

	endpoint := o.endpoints["markPrice"]
	params := map[string]interface{}{
		"instType": o.instType,
		"instId":   symbol,
	}

	query := o.buildQuery(params)
	endpoint += "?" + query

	respStr, err := o.FetchWithRetry(ctx, endpoint, "GET", nil, "")
	if err != nil {
		return nil, err
	}

	var resp struct {
		Code string                   `json:"code"`
		Msg  string                   `json:"msg"`
		Data []map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal([]byte(respStr), &resp); err != nil {
		return nil, err
	}

	if resp.Code != "0" || len(resp.Data) == 0 {
		return nil, fmt.Errorf("okx api error: %s", resp.Msg)
	}

	return o.parseMarkPrice(resp.Data[0]), nil
}

// FetchMarkPrices 获取多个交易对的标记价格
func (o *OKX) FetchMarkPrices(ctx context.Context, symbols []string) (map[string]*types.MarkPrice, error) {
	if !o.config.IsFutures() {
		return nil, fmt.Errorf("标记价格仅在期货模式下可用")
	}

	endpoint := o.endpoints["markPrice"]
	params := map[string]interface{}{
		"instType": o.instType,
	}

	query := o.buildQuery(params)
	endpoint += "?" + query

	respStr, err := o.FetchWithRetry(ctx, endpoint, "GET", nil, "")
	if err != nil {
		return nil, err
	}

	var resp struct {
		Code string                   `json:"code"`
		Msg  string                   `json:"msg"`
		Data []map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal([]byte(respStr), &resp); err != nil {
		return nil, err
	}

	if resp.Code != "0" {
		return nil, fmt.Errorf("okx api error: %s", resp.Msg)
	}

	result := make(map[string]*types.MarkPrice)
	symbolsMap := make(map[string]bool)
	for _, s := range symbols {
		symbolsMap[s] = true
	}

	for _, data := range resp.Data {
		instId := o.SafeString(data, "instId", "")
		if len(symbols) > 0 && !symbolsMap[instId] {
			continue
		}
		result[instId] = o.parseMarkPrice(data)
	}
	return result, nil
}

// parseMarkPrice 解析标记价格
func (o *OKX) parseMarkPrice(data map[string]interface{}) *types.MarkPrice {
	return &types.MarkPrice{
		Symbol:    o.SafeString(data, "instId", ""),
		MarkPrice: o.SafeFloat(data, "markPx", 0),
		Timestamp: o.SafeInteger(data, "ts", 0),
		Info:      data,
	}
}
