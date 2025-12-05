package binance

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"
	"trading_assistant/pkg/exchanges/types"

	"trading_assistant/pkg/exchanges"
)

// ========== Binance 交易所实现（简化版 - 仅公共市场数据）==========

// Binance 实现交易所接口
type Binance struct {
	*exchanges.BaseExchange
	config     *Config
	marketType string // 市场类型：spot, futures

	// API端点缓存
	endpoints map[string]string
}

// ========== 构造函数 ==========

// New 创建新的Binance实例
func New(config *Config) (*Binance, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	base := exchanges.NewBaseExchange("binance", "Binance", "v3", []string{"JP", "MT"})
	binance := &Binance{
		BaseExchange: base,
		config:       config.Clone(),
		marketType:   config.MarketType,
		endpoints:    make(map[string]string),
	}

	// 设置基础信息
	binance.setBasicInfo()

	// 设置支持的功能
	binance.setCapabilities()

	// 设置API端点
	binance.setEndpoints()

	return binance, nil
}

// setBasicInfo 设置基础信息
func (b *Binance) setBasicInfo() {
	b.BaseExchange.SetRetryConfig(3, 100*time.Millisecond, 10*time.Second, true)
	b.BaseExchange.EnableRetry()
}

// setCapabilities 设置支持的功能
func (b *Binance) setCapabilities() {
	capabilities := map[string]bool{
		"fetchMarkets":    true,
		"fetchTicker":     true,
		"fetchBookTicker": true,
		"fetchKline":      true,
		"fetchMarkPrice":  b.marketType == types.MarketTypeFuture,
		"fetchMarkPrices": b.marketType == types.MarketTypeFuture,
	}

	// 设置时间周期
	timeframes := map[string]string{
		"1m":  Interval1m,
		"3m":  Interval3m,
		"5m":  Interval5m,
		"15m": Interval15m,
		"30m": Interval30m,
		"1h":  Interval1h,
		"2h":  Interval2h,
		"4h":  Interval4h,
		"6h":  Interval6h,
		"8h":  Interval8h,
		"12h": Interval12h,
		"1d":  Interval1d,
		"3d":  Interval3d,
		"1w":  Interval1w,
		"1M":  Interval1M,
	}

	// 直接设置功能和时间周期
	for k, v := range capabilities {
		b.BaseExchange.Has()[k] = v
	}
	for k, v := range timeframes {
		b.BaseExchange.GetTimeframes()[k] = v
	}
}

// setEndpoints 设置API端点
func (b *Binance) setEndpoints() {
	baseURL := b.config.GetBaseURL()
	futuresURL := b.config.GetFuturesURL()

	b.endpoints["base"] = baseURL
	b.endpoints["futures"] = futuresURL

	// 现货端点
	b.endpoints["exchangeInfo"] = baseURL + EndpointExchangeInfo
	b.endpoints["ticker24hr"] = baseURL + EndpointTicker24hr
	b.endpoints["bookTicker"] = baseURL + EndpointBookTicker
	b.endpoints["klines"] = baseURL + EndpointKlines

	// 期货端点
	if b.marketType == types.MarketTypeFuture {
		b.endpoints["futuresExchangeInfo"] = futuresURL + EndpointFuturesExchangeInfo
		b.endpoints["futuresTicker24hr"] = futuresURL + EndpointFuturesTicker24hr
		b.endpoints["futuresBookTicker"] = futuresURL + EndpointFuturesBookTicker
		b.endpoints["futuresKlines"] = futuresURL + EndpointFuturesKlines
		b.endpoints["futuresPremiumIndex"] = futuresURL + EndpointFuturesPremiumIndex
	}
}

// ========== 市场数据API ==========

// FetchMarkets 获取市场信息
// 支持 params["quote"] 筛选报价货币，如 params["quote"] = "USDT"
func (b *Binance) FetchMarkets(ctx context.Context, params map[string]interface{}) ([]*types.Market, error) {
	var endpoint string
	if b.marketType == types.MarketTypeFuture {
		endpoint = b.endpoints["futuresExchangeInfo"]
	} else {
		endpoint = b.endpoints["exchangeInfo"]
	}

	respStr, err := b.FetchWithRetry(ctx, endpoint, "GET", nil, "")
	if err != nil {
		return nil, err
	}

	var resp map[string]interface{}
	if err := json.Unmarshal([]byte(respStr), &resp); err != nil {
		return nil, err
	}

	symbols, ok := resp["symbols"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("invalid response format")
	}

	// 获取筛选参数
	var quoteFilter string
	if params != nil {
		if q, ok := params["quote"].(string); ok {
			quoteFilter = q
		}
	}

	var markets []*types.Market
	for _, symbolData := range symbols {
		symbolMap, ok := symbolData.(map[string]interface{})
		if !ok {
			continue
		}

		market := b.parseMarket(symbolMap)
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
func (b *Binance) parseMarket(data map[string]interface{}) *types.Market {
	symbol := b.SafeString(data, "symbol", "")
	if symbol == "" {
		return nil
	}

	status := b.SafeString(data, "status", "")
	if status != "TRADING" {
		return nil
	}

	baseAsset := b.SafeString(data, "baseAsset", "")
	quoteAsset := b.SafeString(data, "quoteAsset", "")

	// 从API数据中获取合约类型字段
	contractType := b.SafeString(data, "contractType", "")

	// 根据API提供的contractType字段判断是否为永续合约
	isSwap := false
	if b.marketType == types.MarketTypeFuture {
		isSwap = contractType == "PERPETUAL"
	}

	market := &types.Market{
		ID:     symbol,
		Symbol: fmt.Sprintf("%s/%s", baseAsset, quoteAsset),
		Base:   baseAsset,
		Quote:  quoteAsset,
		Type:   b.marketType,
		Active: status == "TRADING",
		Spot:   b.marketType == types.MarketTypeSpot,
		Future: b.marketType == types.MarketTypeFuture,
		Swap:   isSwap, // 根据API的contractType字段正确设置
		Info:   data,
	}

	// 解析精度信息
	if filters, ok := data["filters"].([]interface{}); ok {
		market.Precision = b.parseMarketPrecision(filters)
		market.Limits = b.parseMarketLimits(filters)
	}

	return market
}

// parseMarketPrecision 解析市场精度
func (b *Binance) parseMarketPrecision(filters []interface{}) types.MarketPrecision {
	precision := types.MarketPrecision{}

	for _, filterData := range filters {
		filter, ok := filterData.(map[string]interface{})
		if !ok {
			continue
		}

		filterType := b.SafeString(filter, "filterType", "")
		switch filterType {
		case "LOT_SIZE":
			stepSize := b.SafeString(filter, "stepSize", "")
			precision.Amount = b.PrecisionFromString(stepSize)
		case "PRICE_FILTER":
			tickSize := b.SafeString(filter, "tickSize", "")
			precision.Price = b.PrecisionFromString(tickSize)
		}
	}

	return precision
}

// parseMarketLimits 解析市场限制
func (b *Binance) parseMarketLimits(filters []interface{}) types.MarketLimits {
	limits := types.MarketLimits{}

	for _, filterData := range filters {
		filter, ok := filterData.(map[string]interface{})
		if !ok {
			continue
		}

		filterType := b.SafeString(filter, "filterType", "")
		switch filterType {
		case "LOT_SIZE":
			limits.Amount.Min = b.SafeFloat(filter, "minQty", 0)
			limits.Amount.Max = b.SafeFloat(filter, "maxQty", 0)
			limits.Amount.Step = b.SafeFloat(filter, "stepSize", 0)
		case "PRICE_FILTER":
			limits.Price.Min = b.SafeFloat(filter, "minPrice", 0)
			limits.Price.Max = b.SafeFloat(filter, "maxPrice", 0)
			limits.Price.Step = b.SafeFloat(filter, "tickSize", 0)
		case "MIN_NOTIONAL":
			limits.Cost.Min = b.SafeFloat(filter, "minNotional", 0)
		}
	}

	return limits
}

// FetchTickers 批量获取24小时价格统计
func (b *Binance) FetchTickers(ctx context.Context, symbols []string, params map[string]interface{}) (map[string]*types.Ticker, error) {
	// 如果没有symbols，获取所有ticker
	var endpoint string
	if b.marketType == types.MarketTypeFuture {
		endpoint = b.endpoints["futuresTicker24hr"]
	} else {
		endpoint = b.endpoints["ticker24hr"]
	}

	// 不传symbol参数，获取所有ticker数据
	respStr, err := b.FetchWithRetry(ctx, endpoint, "GET", nil, "")
	if err != nil {
		return nil, err
	}

	// 尝试解析为数组（所有ticker）
	var dataArray []interface{}
	if err := json.Unmarshal([]byte(respStr), &dataArray); err != nil {
		return nil, fmt.Errorf("解析ticker数组失败: %v", err)
	}

	// 转换为map，便于查找
	tickers := make(map[string]*types.Ticker)
	symbolsMap := make(map[string]bool)

	// 如果指定了symbols，创建查找map
	if len(symbols) > 0 {
		for _, symbol := range symbols {
			symbolsMap[symbol] = true
		}
	}

	for _, tickerData := range dataArray {
		tickerMap, ok := tickerData.(map[string]interface{})
		if !ok {
			continue
		}

		// 获取symbol
		symbol := b.SafeString(tickerMap, "symbol", "")
		if symbol == "" {
			continue
		}

		// 如果指定了symbols，只处理指定的symbols
		if len(symbols) > 0 && !symbolsMap[symbol] {
			continue
		}

		ticker := b.parseTicker(tickerMap, symbol)
		tickers[symbol] = ticker
	}

	return tickers, nil
}

// FetchBookTickers 获取最优买卖价（bookTicker）- 轻量级接口
func (b *Binance) FetchBookTickers(ctx context.Context, symbols []string, params map[string]interface{}) (map[string]*types.Ticker, error) {
	var endpoint string
	if b.marketType == types.MarketTypeFuture {
		endpoint = b.endpoints["futuresBookTicker"]
	} else {
		endpoint = b.endpoints["bookTicker"]
	}

	// 不传symbol参数，获取所有bookTicker数据
	respStr, err := b.FetchWithRetry(ctx, endpoint, "GET", nil, "")
	if err != nil {
		return nil, err
	}

	// 尝试解析为数组（所有bookTicker）
	var dataArray []interface{}
	if err := json.Unmarshal([]byte(respStr), &dataArray); err != nil {
		return nil, fmt.Errorf("解析bookTicker数组失败: %v", err)
	}

	// 转换为map，便于查找
	tickers := make(map[string]*types.Ticker)
	symbolsMap := make(map[string]bool)

	// 如果指定了symbols，创建查找map
	if len(symbols) > 0 {
		for _, symbol := range symbols {
			symbolsMap[symbol] = true
		}
	}

	for _, tickerData := range dataArray {
		tickerMap, ok := tickerData.(map[string]interface{})
		if !ok {
			continue
		}

		// 获取symbol
		symbol := b.SafeString(tickerMap, "symbol", "")
		if symbol == "" {
			continue
		}

		// 如果指定了symbols，只处理指定的symbols
		if len(symbols) > 0 && !symbolsMap[symbol] {
			continue
		}

		// 解析bookTicker数据
		ticker := &types.Ticker{
			Symbol:    symbol,
			TimeStamp: b.SafeInteger(tickerMap, "time", time.Now().UnixMilli()),
			Bid:       b.SafeFloat(tickerMap, "bidPrice", 0),
			BidVolume: b.SafeFloat(tickerMap, "bidQty", 0),
			Ask:       b.SafeFloat(tickerMap, "askPrice", 0),
			AskVolume: b.SafeFloat(tickerMap, "askQty", 0),
			Info:      tickerMap,
		}
		tickers[symbol] = ticker
	}

	return tickers, nil
}

// FetchTickersBatch 分批获取ticker数据 - 避免超时
func (b *Binance) FetchTickersBatch(ctx context.Context, symbols []string, batchSize int) (map[string]*types.Ticker, error) {
	if batchSize <= 0 {
		batchSize = 100 // 默认批次大小
	}

	// 如果symbols为空或很小，直接获取全部
	if len(symbols) == 0 || len(symbols) <= batchSize {
		return b.FetchTickers(ctx, symbols, nil)
	}

	allTickers := make(map[string]*types.Ticker)

	// 分批处理
	for i := 0; i < len(symbols); i += batchSize {
		end := i + batchSize
		if end > len(symbols) {
			end = len(symbols)
		}

		batch := symbols[i:end]

		// 获取这一批的ticker数据
		batchTickers, err := b.FetchTickers(ctx, batch, nil)
		if err != nil {
			return nil, fmt.Errorf("批次 %d-%d 获取失败: %v", i, end-1, err)
		}

		// 合并结果
		for symbol, ticker := range batchTickers {
			allTickers[symbol] = ticker
		}

		// 批次间延迟，避免rate limit
		if i+batchSize < len(symbols) {
			time.Sleep(100 * time.Millisecond)
		}
	}

	return allTickers, nil
}

// parseTicker 解析ticker数据
func (b *Binance) parseTicker(data map[string]interface{}, symbol string) *types.Ticker {
	timestamp := b.SafeInteger(data, "closeTime", time.Now().UnixMilli())

	return &types.Ticker{
		Symbol:      symbol,
		TimeStamp:   timestamp,
		Datetime:    b.ISO8601(timestamp),
		High:        b.SafeFloat(data, "highPrice", 0),
		Low:         b.SafeFloat(data, "lowPrice", 0),
		Bid:         b.SafeFloat(data, "bidPrice", 0),
		BidVolume:   b.SafeFloat(data, "bidQty", 0),
		Ask:         b.SafeFloat(data, "askPrice", 0),
		AskVolume:   b.SafeFloat(data, "askQty", 0),
		Open:        b.SafeFloat(data, "openPrice", 0),
		Close:       b.SafeFloat(data, "lastPrice", 0),
		Last:        b.SafeFloat(data, "lastPrice", 0),
		Change:      b.SafeFloat(data, "priceChange", 0),
		Percentage:  b.SafeFloat(data, "priceChangePercent", 0),
		BaseVolume:  b.SafeFloat(data, "volume", 0),
		QuoteVolume: b.SafeFloat(data, "quoteVolume", 0),
		Info:        data,
	}
}

// FetchKlines 获取K线数据
func (b *Binance) FetchKlines(ctx context.Context, symbol, interval string, since int64, limit int, params map[string]interface{}) ([]*types.Kline, error) {
	if symbol == "" {
		return nil, fmt.Errorf("symbol不能为空")
	}

	// 构建请求参数
	requestParams := map[string]interface{}{
		"symbol":   symbol,
		"interval": interval,
	}

	if limit > 0 {
		if limit > 1500 {
			limit = 1500 // Binance最大限制
		}
		requestParams["limit"] = limit
	} else {
		requestParams["limit"] = 500 // 默认值
	}

	// 如果指定了起始时间
	if since > 0 {
		requestParams["startTime"] = since
	}

	// 合并用户参数
	for k, v := range params {
		requestParams[k] = v
	}

	// 选择正确的端点
	var endpoint string
	if b.marketType == types.MarketTypeFuture {
		endpoint = b.endpoints["futuresKlines"]
	} else {
		endpoint = b.endpoints["klines"]
	}

	// 构建查询字符串
	queryParams := make([]string, 0, len(requestParams))
	for k, v := range requestParams {
		queryParams = append(queryParams, fmt.Sprintf("%s=%v", k, v))
	}

	if len(queryParams) > 0 {
		endpoint += "?" + strings.Join(queryParams, "&")
	}

	// 发送请求
	respStr, err := b.FetchWithRetry(ctx, endpoint, "GET", nil, "")
	if err != nil {
		return nil, fmt.Errorf("获取K线数据失败: %w", err)
	}

	// 解析响应
	var rawKlines [][]interface{}
	if err := json.Unmarshal([]byte(respStr), &rawKlines); err != nil {
		return nil, fmt.Errorf("解析K线数据失败: %w", err)
	}

	// 转换为标准格式
	klines := make([]*types.Kline, 0, len(rawKlines))
	for i := range rawKlines {
		rawKline := rawKlines[i]
		kline := b.parseKline(rawKline, symbol, interval)
		if kline != nil {
			klines = append(klines, kline)
		}
	}

	return klines, nil
}

// parseKline 解析K线数据
func (b *Binance) parseKline(data []interface{}, symbol, interval string) *types.Kline {
	if len(data) < 11 {
		return nil
	}

	// Binance K线数据格式:
	// [
	//   1499040000000,      // 开盘时间
	//   "0.01634790",       // 开盘价
	//   "0.80000000",       // 最高价
	//   "0.01575800",       // 最低价
	//   "0.01577100",       // 收盘价(当前K线未结束的即为最新价)
	//   "148976.11427815",  // 成交量
	//   1499644799999,      // 收盘时间
	//   "2434.19055334",    // 成交额
	//   308,                // 成交笔数
	//   "1756.87402397",    // 主动买入成交量
	//   "28.46694368",      // 主动买入成交额
	//   "17928899.62484339" // 请忽略该参数
	// ]

	// 安全的类型转换函数
	toInt64 := func(val interface{}) int64 {
		switch v := val.(type) {
		case float64:
			return int64(v)
		case int64:
			return v
		case int:
			return int64(v)
		case string:
			if parsed, err := strconv.ParseInt(v, 10, 64); err == nil {
				return parsed
			}
		}
		return time.Now().UnixMilli()
	}

	toFloat64 := func(val interface{}) float64 {
		switch v := val.(type) {
		case float64:
			return v
		case int64:
			return float64(v)
		case int:
			return float64(v)
		case string:
			if parsed, err := strconv.ParseFloat(v, 64); err == nil {
				return parsed
			}
		}
		return 0
	}

	timestamp := toInt64(data[0])
	closeTime := toInt64(data[6])

	return &types.Kline{
		Symbol:    symbol,
		Timeframe: interval,
		Timestamp: timestamp,
		Open:      toFloat64(data[1]),
		High:      toFloat64(data[2]),
		Low:       toFloat64(data[3]),
		Close:     toFloat64(data[4]),
		Volume:    toFloat64(data[5]),
		IsClosed:  closeTime <= time.Now().UnixMilli(), // 收盘时间小于等于当前时间表示已收盘
	}
}

// ========== 标记价格API ==========

// FetchMarkPrice 获取单个交易对的标记价格
func (b *Binance) FetchMarkPrice(ctx context.Context, symbol string) (*types.MarkPrice, error) {
	if b.marketType != types.MarketTypeFuture {
		return nil, fmt.Errorf("标记价格仅在期货模式下可用")
	}

	endpoint := b.endpoints["futuresPremiumIndex"]
	if symbol != "" {
		endpoint += "?symbol=" + symbol
	}

	respStr, err := b.FetchWithRetry(ctx, endpoint, "GET", nil, "")
	if err != nil {
		return nil, err
	}

	var data map[string]interface{}
	if err := json.Unmarshal([]byte(respStr), &data); err != nil {
		return nil, err
	}

	return b.parseMarkPrice(data), nil
}

// FetchMarkPrices 获取多个交易对的标记价格
func (b *Binance) FetchMarkPrices(ctx context.Context, symbols []string) (map[string]*types.MarkPrice, error) {
	if b.marketType != types.MarketTypeFuture {
		return nil, fmt.Errorf("标记价格仅在期货模式下可用")
	}

	endpoint := b.endpoints["futuresPremiumIndex"]

	respStr, err := b.FetchWithRetry(ctx, endpoint, "GET", nil, "")
	if err != nil {
		return nil, err
	}

	var dataArray []map[string]interface{}
	if err := json.Unmarshal([]byte(respStr), &dataArray); err != nil {
		return nil, err
	}

	markPrices := make(map[string]*types.MarkPrice)
	symbolsMap := make(map[string]bool)

	// 如果指定了symbols，创建查找map
	if len(symbols) > 0 {
		for _, symbol := range symbols {
			symbolsMap[symbol] = true
		}
	}

	for _, data := range dataArray {
		symbol := b.SafeString(data, "symbol", "")
		if symbol == "" {
			continue
		}

		// 如果指定了symbols，只处理指定的symbols
		if len(symbols) > 0 && !symbolsMap[symbol] {
			continue
		}

		markPrice := b.parseMarkPrice(data)
		markPrices[symbol] = markPrice
	}

	return markPrices, nil
}

// parseMarkPrice 解析标记价格数据
func (b *Binance) parseMarkPrice(data map[string]interface{}) *types.MarkPrice {
	return &types.MarkPrice{
		Symbol:               b.SafeString(data, "symbol", ""),
		MarkPrice:            b.SafeFloat(data, "markPrice", 0),
		IndexPrice:           b.SafeFloat(data, "indexPrice", 0),
		FundingRate:          b.SafeFloat(data, "lastFundingRate", 0),
		NextFundingTime:      b.SafeInteger(data, "nextFundingTime", 0),
		InterestRate:         b.SafeFloat(data, "interestRate", 0),
		EstimatedSettlePrice: b.SafeFloat(data, "estimatedSettlePrice", 0),
		Timestamp:            time.Now().UnixMilli(),
		Info:                 data,
	}
}

// ========== 实用方法 ==========

// GetMarketType 获取市场类型
func (b *Binance) GetMarketType() string {
	return b.marketType
}

// IsTestnet 是否测试网
func (b *Binance) IsTestnet() bool {
	return b.config.TestNet
}

// GetConfig 获取配置
func (b *Binance) GetConfig() *Config {
	return b.config
}
