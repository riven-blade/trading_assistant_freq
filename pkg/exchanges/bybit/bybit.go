package bybit

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
	"trading_assistant/pkg/exchanges/types"

	"trading_assistant/pkg/exchanges"
)

// ========== Bybit 交易所实现 ==========

// Bybit 实现交易所接口
type Bybit struct {
	*exchanges.BaseExchange
	config   *Config
	category string // 产品类型：spot, linear, inverse, option

	// API端点缓存
	endpoints map[string]string

	// 缓存字段
	lastServerTimeRequest int64
	serverTimeOffset      int64
}

// ========== 构造函数 ==========

// New 创建新的Bybit实例
func New(config *Config) (*Bybit, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}

	base := exchanges.NewBaseExchange("bybit", "Bybit", "v5", []string{"VG"})
	bybit := &Bybit{
		BaseExchange: base,
		config:       config.Clone(),
		category:     config.Category,
		endpoints:    make(map[string]string),
	}

	// 设置基础信息
	bybit.setBasicInfo()

	// 设置支持的功能
	bybit.setCapabilities()

	// 设置API端点
	bybit.setEndpoints()

	// 设置凭证
	bybit.SetCredentials(config.APIKey, config.Secret, "", "")

	// 初始同步服务器时间
	go bybit.updateServerTimeOffset()

	return bybit, nil
}

// setBasicInfo 设置基础信息
func (b *Bybit) setBasicInfo() {
	b.BaseExchange.SetRetryConfig(3, 100*time.Millisecond, 10*time.Second, true)
	b.BaseExchange.EnableRetry()
}

// setCapabilities 设置支持的功能
func (b *Bybit) setCapabilities() {
	capabilities := map[string]bool{
		"fetchMarkets":    true,
		"fetchTicker":     true,
		"fetchKline":      true,
		"fetchTrades":     true,
		"fetchOrderBook":  true,
		"fetchBalance":    true,
		"createOrder":     true,
		"cancelOrder":     true,
		"fetchOrder":      true,
		"fetchOrders":     true,
		"fetchOpenOrders": true,
		"fetchPositions":  false,
		"setLeverage":     false,
		"setMarginMode":   false,
	}

	// 根据产品类型调整功能
	if b.category == CategoryLinear || b.category == CategoryInverse {
		capabilities["fetchPositions"] = true
		capabilities["setLeverage"] = true
		capabilities["setMarginMode"] = true
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
		"12h": Interval12h,
		"1d":  Interval1d,
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
func (b *Bybit) setEndpoints() {
	baseURL := b.config.GetBaseURL()

	b.endpoints["base"] = baseURL
	b.endpoints["websocket"] = b.config.GetWebSocketURL()

	// 市场数据端点
	b.endpoints["instrumentsInfo"] = baseURL + EndpointInstrumentsInfo
	b.endpoints["tickers"] = baseURL + EndpointTickers
	b.endpoints["kline"] = baseURL + EndpointKline
	b.endpoints["orderbook"] = baseURL + EndpointOrderbook
	b.endpoints["recentTrade"] = baseURL + EndpointRecentTrade

	// 交易端点
	b.endpoints["placeOrder"] = baseURL + EndpointPlaceOrder
	b.endpoints["cancelOrder"] = baseURL + EndpointCancelOrder
	b.endpoints["orderHistory"] = baseURL + EndpointOrderHistory
	b.endpoints["orderRealtime"] = baseURL + EndpointOrderRealtime

	// 账户端点
	b.endpoints["walletBalance"] = baseURL + EndpointWalletBalance

	// 持仓端点
	if b.category == CategoryLinear || b.category == CategoryInverse {
		b.endpoints["positionInfo"] = baseURL + EndpointPositionInfo
		b.endpoints["setLeverage"] = baseURL + EndpointSetLeverage
		b.endpoints["switchMode"] = baseURL + EndpointSwitchMode
	}
}

// ========== 签名和认证 ==========

// Sign 签名请求
func (b *Bybit) Sign(path, api, method string, params map[string]interface{}, headers map[string]string, body interface{}) (string, map[string]string, interface{}, error) {
	if headers == nil {
		headers = make(map[string]string)
	}

	// 公开API不需要签名
	if api == "public" {
		query := b.buildQuery(params)
		if query != "" {
			if strings.Contains(path, "?") {
				path += "&" + query
			} else {
				path += "?" + query
			}
		}
		return path, headers, body, nil
	}

	// 私有API需要签名
	if b.GetApiKey() == "" || b.GetSecret() == "" {
		return "", nil, nil, exchanges.NewAuthenticationError("API key and secret required")
	}

	// 添加时间戳
	if params == nil {
		params = make(map[string]interface{})
	}
	params["timestamp"] = b.GetServerTime()

	// 添加接收窗口
	if b.config.RecvWindow > 0 {
		params["recvWindow"] = b.config.RecvWindow
	}

	// 根据HTTP方法处理签名
	var signature string
	if method == "GET" || method == "DELETE" {
		// GET/DELETE请求：参数在查询字符串中
		query := b.buildQuery(params)
		signature = b.generateSignature(method, path, query, "")
		if query != "" {
			if strings.Contains(path, "?") {
				path += "&" + query
			} else {
				path += "?" + query
			}
		}
	} else {
		// POST/PUT请求：参数在请求体中
		bodyStr := ""
		if len(params) > 0 {
			bodyBytes, _ := json.Marshal(params)
			bodyStr = string(bodyBytes)
			body = bodyStr
		}
		signature = b.generateSignature(method, path, "", bodyStr)
		headers["Content-Type"] = "application/json"
	}

	// 添加认证头部
	headers["X-BAPI-API-KEY"] = b.GetApiKey()
	headers["X-BAPI-SIGN"] = signature
	headers["X-BAPI-TIMESTAMP"] = fmt.Sprintf("%d", b.GetServerTime())
	if b.config.RecvWindow > 0 {
		headers["X-BAPI-RECV-WINDOW"] = fmt.Sprintf("%d", b.config.RecvWindow)
	}

	return path, headers, body, nil
}

// buildQuery 构建查询字符串
func (b *Bybit) buildQuery(params map[string]interface{}) string {
	if len(params) == 0 {
		return ""
	}

	var keys []string
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var parts []string
	for _, k := range keys {
		v := params[k]
		parts = append(parts, fmt.Sprintf("%s=%v", k, v))
	}

	return strings.Join(parts, "&")
}

// generateSignature 生成HMAC SHA256签名
func (b *Bybit) generateSignature(method, path, query, body string) string {
	// Bybit v5 签名格式: timestamp + api_key + recv_window + query_string + body
	timestamp := fmt.Sprintf("%d", b.GetServerTime())
	recvWindow := ""
	if b.config.RecvWindow > 0 {
		recvWindow = fmt.Sprintf("%d", b.config.RecvWindow)
	}

	payload := timestamp + b.GetApiKey() + recvWindow + query + body

	mac := hmac.New(sha256.New, []byte(b.GetSecret()))
	mac.Write([]byte(payload))
	return hex.EncodeToString(mac.Sum(nil))
}

// GetServerTime 获取服务器时间
func (b *Bybit) GetServerTime() int64 {
	now := time.Now().UnixMilli()

	// 如果有时间偏移，应用偏移
	if b.serverTimeOffset != 0 {
		return now + b.serverTimeOffset
	}

	// 如果距离上次请求服务器时间超过5分钟，更新时间偏移
	if now-b.lastServerTimeRequest > 5*60*1000 {
		go b.updateServerTimeOffset()
	}

	return now
}

// updateServerTimeOffset 更新服务器时间偏移
func (b *Bybit) updateServerTimeOffset() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	url := b.endpoints["base"] + "/v5/market/time"
	resp, err := b.Fetch(ctx, url, "GET", nil, "")
	if err != nil {
		return
	}

	var timeResp struct {
		RetCode int    `json:"retCode"`
		RetMsg  string `json:"retMsg"`
		Result  struct {
			TimeSecond string `json:"timeSecond"`
			TimeNano   string `json:"timeNano"`
		} `json:"result"`
	}
	if err := json.Unmarshal([]byte(resp), &timeResp); err != nil {
		return
	}

	if timeResp.RetCode == 0 {
		if serverTime, err := strconv.ParseInt(timeResp.Result.TimeSecond, 10, 64); err == nil {
			localTime := time.Now().Unix()
			b.serverTimeOffset = (serverTime - localTime) * 1000 // 转换为毫秒
			b.lastServerTimeRequest = time.Now().UnixMilli()
		}
	}
}

// ========== 市场数据API ==========

// FetchMarkets 获取市场信息
func (b *Bybit) FetchMarkets(ctx context.Context, params map[string]interface{}) ([]*types.Market, error) {
	endpoint := b.endpoints["instrumentsInfo"]

	// 添加产品类型参数
	if params == nil {
		params = make(map[string]interface{})
	}
	params["category"] = b.category

	// 设置limit参数以获取更多数据
	if _, hasLimit := params["limit"]; !hasLimit {
		params["limit"] = 1000
	}

	// 构建查询参数
	query := b.buildQuery(params)
	if query != "" {
		endpoint += "?" + query
	}

	respStr, err := b.FetchWithRetry(ctx, endpoint, "GET", nil, "")
	if err != nil {
		return nil, err
	}

	var resp struct {
		RetCode int    `json:"retCode"`
		RetMsg  string `json:"retMsg"`
		Result  struct {
			Category       string                   `json:"category"`
			List           []map[string]interface{} `json:"list"`
			NextPageCursor string                   `json:"nextPageCursor"`
		} `json:"result"`
	}
	if err := json.Unmarshal([]byte(respStr), &resp); err != nil {
		return nil, err
	}

	if resp.RetCode != 0 {
		return nil, fmt.Errorf("bybit api error: %s", resp.RetMsg)
	}

	var markets []*types.Market
	for _, symbolData := range resp.Result.List {
		market := b.parseMarket(symbolData)
		if market != nil {
			markets = append(markets, market)
		}
	}

	return markets, nil
}

// parseMarket 解析市场信息
func (b *Bybit) parseMarket(data map[string]interface{}) *types.Market {
	symbol := b.SafeString(data, "symbol", "")
	if symbol == "" {
		return nil
	}

	status := b.SafeString(data, "status", "")
	if status != "Trading" {
		return nil
	}

	baseCoin := b.SafeString(data, "baseCoin", "")
	quoteCoin := b.SafeString(data, "quoteCoin", "")

	// 根据产品类型设置市场属性
	isSpot := b.category == CategorySpot
	isFuture := b.category == CategoryLinear || b.category == CategoryInverse
	isSwap := false // Bybit的linear和inverse都是永续合约
	if isFuture {
		contractType := b.SafeString(data, "contractType", "")
		isSwap = contractType == "LinearPerpetual" || contractType == "InversePerpetual"
	}

	market := &types.Market{
		ID:     symbol,
		Symbol: fmt.Sprintf("%s/%s", baseCoin, quoteCoin),
		Base:   baseCoin,
		Quote:  quoteCoin,
		Type:   b.config.MarketType,
		Active: status == "Trading",
		Spot:   isSpot,
		Future: isFuture,
		Swap:   isSwap,
		Info:   data,
	}

	// 解析精度信息
	market.Precision = b.parseMarketPrecision(data)
	market.Limits = b.parseMarketLimits(data)

	return market
}

// parseMarketPrecision 解析市场精度
func (b *Bybit) parseMarketPrecision(data map[string]interface{}) types.MarketPrecision {
	precision := types.MarketPrecision{}

	// 价格精度
	if priceScale, ok := data["priceScale"]; ok {
		if scale, ok := priceScale.(float64); ok {
			precision.Price = scale
		}
	}

	// 数量精度
	if lotSizeFilter, ok := data["lotSizeFilter"].(map[string]interface{}); ok {
		if qtyStep := b.SafeString(lotSizeFilter, "qtyStep", ""); qtyStep != "" {
			precision.Amount = b.PrecisionFromString(qtyStep)
		}
	}

	return precision
}

// parseMarketLimits 解析市场限制
func (b *Bybit) parseMarketLimits(data map[string]interface{}) types.MarketLimits {
	limits := types.MarketLimits{}

	// 价格限制
	if priceFilter, ok := data["priceFilter"].(map[string]interface{}); ok {
		limits.Price.Min = b.SafeFloat(priceFilter, "minPrice", 0)
		limits.Price.Max = b.SafeFloat(priceFilter, "maxPrice", 0)
		limits.Price.Step = b.SafeFloat(priceFilter, "tickSize", 0)
	}

	// 数量限制
	if lotSizeFilter, ok := data["lotSizeFilter"].(map[string]interface{}); ok {
		limits.Amount.Min = b.SafeFloat(lotSizeFilter, "minOrderQty", 0)
		limits.Amount.Max = b.SafeFloat(lotSizeFilter, "maxOrderQty", 0)
		limits.Amount.Step = b.SafeFloat(lotSizeFilter, "qtyStep", 0)
	}

	return limits
}

// FetchBookTickers 获取最优买卖价（bookTicker）- 轻量级接口
func (b *Bybit) FetchBookTickers(ctx context.Context, symbols []string, params map[string]interface{}) (map[string]*types.Ticker, error) {
	// Bybit 暂时使用 FetchTickers 实现（包含 bid/ask）
	return b.FetchTickers(ctx, symbols, params)
}

// FetchTickers 批量获取24小时价格统计
func (b *Bybit) FetchTickers(ctx context.Context, symbols []string, params map[string]interface{}) (map[string]*types.Ticker, error) {
	endpoint := b.endpoints["tickers"]

	// 添加产品类型参数
	if params == nil {
		params = make(map[string]interface{})
	}
	params["category"] = b.category

	// 如果指定了symbol，添加到参数中
	if len(symbols) == 1 {
		params["symbol"] = symbols[0]
	}

	// 构建查询参数
	query := b.buildQuery(params)
	if query != "" {
		endpoint += "?" + query
	}

	respStr, err := b.FetchWithRetry(ctx, endpoint, "GET", nil, "")
	if err != nil {
		return nil, err
	}

	var resp struct {
		RetCode int    `json:"retCode"`
		RetMsg  string `json:"retMsg"`
		Result  struct {
			Category string                   `json:"category"`
			List     []map[string]interface{} `json:"list"`
		} `json:"result"`
	}
	if err := json.Unmarshal([]byte(respStr), &resp); err != nil {
		return nil, err
	}

	if resp.RetCode != 0 {
		return nil, fmt.Errorf("bybit api error: %s", resp.RetMsg)
	}

	// 转换为map
	tickers := make(map[string]*types.Ticker)
	symbolsMap := make(map[string]bool)

	// 如果指定了symbols，创建查找map
	if len(symbols) > 0 {
		for _, symbol := range symbols {
			symbolsMap[symbol] = true
		}
	}

	for _, tickerData := range resp.Result.List {
		// 获取symbol
		symbol := b.SafeString(tickerData, "symbol", "")
		if symbol == "" {
			continue
		}

		// 如果指定了symbols，只处理指定的symbols
		if len(symbols) > 0 && !symbolsMap[symbol] {
			continue
		}

		ticker := b.parseTicker(tickerData, symbol)
		tickers[symbol] = ticker
	}

	return tickers, nil
}

// parseTicker 解析ticker数据
func (b *Bybit) parseTicker(data map[string]interface{}, symbol string) *types.Ticker {
	timestamp := time.Now().UnixMilli()

	return &types.Ticker{
		Symbol:      symbol,
		TimeStamp:   timestamp,
		Datetime:    b.ISO8601(timestamp),
		High:        b.SafeFloat(data, "highPrice24h", 0),
		Low:         b.SafeFloat(data, "lowPrice24h", 0),
		Bid:         b.SafeFloat(data, "bid1Price", 0),
		BidVolume:   b.SafeFloat(data, "bid1Size", 0),
		Ask:         b.SafeFloat(data, "ask1Price", 0),
		AskVolume:   b.SafeFloat(data, "ask1Size", 0),
		Open:        b.SafeFloat(data, "prevPrice24h", 0),
		Close:       b.SafeFloat(data, "lastPrice", 0),
		Last:        b.SafeFloat(data, "lastPrice", 0),
		Change:      0,                                          // 需要计算
		Percentage:  b.SafeFloat(data, "price24hPcnt", 0) * 100, // 转换为百分比
		BaseVolume:  b.SafeFloat(data, "volume24h", 0),
		QuoteVolume: b.SafeFloat(data, "turnover24h", 0),
		Info:        data,
	}
}

// FetchKlines 获取K线数据
func (b *Bybit) FetchKlines(ctx context.Context, symbol, interval string, since int64, limit int, params map[string]interface{}) ([]*types.Kline, error) {
	if symbol == "" {
		return nil, fmt.Errorf("symbol不能为空")
	}

	endpoint := b.endpoints["kline"]

	// 转换interval格式为bybit格式
	bybitInterval := b.convertInterval(interval)

	// 构建请求参数
	requestParams := map[string]interface{}{
		"category": b.category,
		"symbol":   symbol,
		"interval": bybitInterval,
	}

	if limit > 0 {
		if limit > 1000 {
			limit = 1000 // Bybit最大限制
		}
		requestParams["limit"] = limit
	} else {
		requestParams["limit"] = 200 // 默认值
	}

	// 如果指定了起始时间，使用start参数从该时间往后获取
	if since > 0 {
		requestParams["start"] = since
	} else {
		// 如果没有指定起始时间，使用end参数设置为当前时间，从当前时间往前获取最近的数据
		// 这样可以确保获取到最近的limit条K线数据
		requestParams["end"] = time.Now().UnixMilli()
	}

	// 合并用户参数
	for k, v := range params {
		requestParams[k] = v
	}

	// 构建查询字符串
	query := b.buildQuery(requestParams)
	if query != "" {
		endpoint += "?" + query
	}

	// 发送请求
	respStr, err := b.FetchWithRetry(ctx, endpoint, "GET", nil, "")
	if err != nil {
		return nil, fmt.Errorf("获取K线数据失败: %w", err)
	}

	// 解析响应
	var resp struct {
		RetCode int    `json:"retCode"`
		RetMsg  string `json:"retMsg"`
		Result  struct {
			Category string          `json:"category"`
			Symbol   string          `json:"symbol"`
			List     [][]interface{} `json:"list"`
		} `json:"result"`
	}
	if err := json.Unmarshal([]byte(respStr), &resp); err != nil {
		return nil, fmt.Errorf("解析K线数据失败: %w", err)
	}

	if resp.RetCode != 0 {
		return nil, fmt.Errorf("bybit api error: %s", resp.RetMsg)
	}

	// 转换为标准格式
	// 注意：Bybit返回的K线数据是倒序的（从新到旧），需要反转为正序（从旧到新）
	klines := make([]*types.Kline, 0, len(resp.Result.List))
	for i := len(resp.Result.List) - 1; i >= 0; i-- {
		rawKline := resp.Result.List[i]
		kline := b.parseKline(rawKline, symbol, interval)
		if kline != nil {
			klines = append(klines, kline)
		}
	}

	return klines, nil
}

// parseKline 解析K线数据
func (b *Bybit) parseKline(data []interface{}, symbol, interval string) *types.Kline {
	if len(data) < 7 {
		return nil
	}

	// Bybit K线数据格式:
	// [
	//   "1670608800000", // 开始时间
	//   "16493.50",      // 开盘价
	//   "16611.00",      // 最高价
	//   "16493.50",      // 最低价
	//   "16511.00",      // 收盘价
	//   "25.777",        // 成交量
	//   "426170.8199"    // 成交额
	// ]

	// 安全的类型转换函数
	toInt64 := func(val interface{}) int64 {
		switch v := val.(type) {
		case string:
			if parsed, err := strconv.ParseInt(v, 10, 64); err == nil {
				return parsed
			}
		case float64:
			return int64(v)
		case int64:
			return v
		case int:
			return int64(v)
		}
		return time.Now().UnixMilli()
	}

	toFloat64 := func(val interface{}) float64 {
		switch v := val.(type) {
		case string:
			if parsed, err := strconv.ParseFloat(v, 64); err == nil {
				return parsed
			}
		case float64:
			return v
		case int64:
			return float64(v)
		case int:
			return float64(v)
		}
		return 0
	}

	timestamp := toInt64(data[0])

	return &types.Kline{
		Symbol:    symbol,
		Timeframe: interval,
		Timestamp: timestamp,
		Open:      toFloat64(data[1]),
		High:      toFloat64(data[2]),
		Low:       toFloat64(data[3]),
		Close:     toFloat64(data[4]),
		Volume:    toFloat64(data[5]),
		IsClosed:  true, // Bybit返回的都是已关闭的K线
	}
}

// ========== 标记价格API ==========

// FetchMarkPrice 获取单个交易对的标记价格
func (b *Bybit) FetchMarkPrice(ctx context.Context, symbol string) (*types.MarkPrice, error) {
	if !b.config.IsFutures() {
		return nil, fmt.Errorf("标记价格仅在期货模式下可用")
	}

	endpoint := b.endpoints["base"] + "/v5/market/tickers"

	// 构建请求参数
	params := map[string]interface{}{
		"category": b.category,
	}
	if symbol != "" {
		params["symbol"] = symbol
	}

	// 构建查询参数
	query := b.buildQuery(params)
	if query != "" {
		endpoint += "?" + query
	}

	respStr, err := b.FetchWithRetry(ctx, endpoint, "GET", nil, "")
	if err != nil {
		return nil, err
	}

	var resp struct {
		RetCode int    `json:"retCode"`
		RetMsg  string `json:"retMsg"`
		Result  struct {
			Category string                   `json:"category"`
			List     []map[string]interface{} `json:"list"`
		} `json:"result"`
	}
	if err := json.Unmarshal([]byte(respStr), &resp); err != nil {
		return nil, err
	}

	if resp.RetCode != 0 {
		return nil, fmt.Errorf("bybit api error: %s", resp.RetMsg)
	}

	if len(resp.Result.List) == 0 {
		return nil, fmt.Errorf("未找到交易对 %s 的标记价格", symbol)
	}

	return b.parseMarkPrice(resp.Result.List[0]), nil
}

// FetchMarkPrices 获取多个交易对的标记价格
func (b *Bybit) FetchMarkPrices(ctx context.Context, symbols []string) (map[string]*types.MarkPrice, error) {
	if !b.config.IsFutures() {
		return nil, fmt.Errorf("标记价格仅在期货模式下可用")
	}

	endpoint := b.endpoints["base"] + "/v5/market/tickers"

	// 构建请求参数
	params := map[string]interface{}{
		"category": b.category,
	}

	// 构建查询参数
	query := b.buildQuery(params)
	if query != "" {
		endpoint += "?" + query
	}

	respStr, err := b.FetchWithRetry(ctx, endpoint, "GET", nil, "")
	if err != nil {
		return nil, err
	}

	var resp struct {
		RetCode int    `json:"retCode"`
		RetMsg  string `json:"retMsg"`
		Result  struct {
			Category string                   `json:"category"`
			List     []map[string]interface{} `json:"list"`
		} `json:"result"`
	}
	if err := json.Unmarshal([]byte(respStr), &resp); err != nil {
		return nil, err
	}

	if resp.RetCode != 0 {
		return nil, fmt.Errorf("bybit api error: %s", resp.RetMsg)
	}

	markPrices := make(map[string]*types.MarkPrice)
	symbolsMap := make(map[string]bool)

	// 如果指定了symbols，创建查找map
	if len(symbols) > 0 {
		for _, symbol := range symbols {
			symbolsMap[symbol] = true
		}
	}

	for _, data := range resp.Result.List {
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
func (b *Bybit) parseMarkPrice(data map[string]interface{}) *types.MarkPrice {
	return &types.MarkPrice{
		Symbol:               b.SafeString(data, "symbol", ""),
		MarkPrice:            b.SafeFloat(data, "markPrice", 0),
		IndexPrice:           b.SafeFloat(data, "indexPrice", 0),
		FundingRate:          b.SafeFloat(data, "fundingRate", 0),
		NextFundingTime:      b.SafeInteger(data, "nextFundingTime", 0),
		InterestRate:         0, // Bybit 不直接提供利率
		EstimatedSettlePrice: 0, // Bybit 不直接提供预估结算价
		Timestamp:            time.Now().UnixMilli(),
		Info:                 data,
	}
}

// ========== 实用方法 ==========

// convertInterval 转换interval格式为bybit格式
func (b *Bybit) convertInterval(interval string) string {
	// 将标准时间格式转换为bybit API格式
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
		return Interval1h
	case "2h":
		return Interval2h
	case "4h":
		return Interval4h
	case "6h":
		return Interval6h
	case "12h":
		return Interval12h
	case "1d":
		return Interval1d
	case "1w":
		return Interval1w
	case "1M":
		return Interval1M
	default:
		// 如果是bybit原生格式，直接返回
		return interval
	}
}

// GetMarketType 获取市场类型
func (b *Bybit) GetMarketType() string {
	return b.config.MarketType
}

// GetCategory 获取产品类型
func (b *Bybit) GetCategory() string {
	return b.category
}

// IsTestnet 是否测试网
func (b *Bybit) IsTestnet() bool {
	return b.config.TestNet
}

// GetConfig 获取配置
func (b *Bybit) GetConfig() *Config {
	return b.config
}
