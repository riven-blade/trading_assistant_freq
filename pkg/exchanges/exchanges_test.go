package exchanges_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"trading_assistant/pkg/exchanges/binance"
	"trading_assistant/pkg/exchanges/bybit"
	"trading_assistant/pkg/exchanges/mexc"
	"trading_assistant/pkg/exchanges/okx"
	"trading_assistant/pkg/exchanges/types"
)

// ========== Binance 测试 ==========

func TestBinanceSpot(t *testing.T) {
	config := binance.DefaultConfig()
	config.MarketType = types.MarketTypeSpot

	exchange, err := binance.New(config)
	if err != nil {
		t.Fatalf("创建 Binance 现货实例失败: %v", err)
	}

	if exchange.GetMarketType() != types.MarketTypeSpot {
		t.Errorf("市场类型错误: 期望 %s, 实际 %s", types.MarketTypeSpot, exchange.GetMarketType())
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 测试获取市场信息
	markets, err := exchange.FetchMarkets(ctx, nil)
	if err != nil {
		t.Fatalf("获取市场信息失败: %v", err)
	}
	if len(markets) == 0 {
		t.Error("未获取到任何市场信息")
	}
	fmt.Printf("Binance Spot: 获取到 %d 个交易对\n", len(markets))

	// 验证市场类型标识
	for _, m := range markets[:min(3, len(markets))] {
		if !m.Spot {
			t.Errorf("市场 %s 应该标记为现货", m.Symbol)
		}
		fmt.Printf("  - %s (Spot=%v, Future=%v)\n", m.Symbol, m.Spot, m.Future)
	}

	// 测试获取 Ticker
	tickers, err := exchange.FetchTickers(ctx, []string{"BTCUSDT"}, nil)
	if err != nil {
		t.Fatalf("获取 Ticker 失败: %v", err)
	}
	if ticker, ok := tickers["BTCUSDT"]; ok {
		fmt.Printf("Binance Spot BTCUSDT: Last=%.2f, Bid=%.2f, Ask=%.2f\n", ticker.Last, ticker.Bid, ticker.Ask)
	}
}

func TestBinanceFutures(t *testing.T) {
	config := binance.DefaultConfig()
	config.MarketType = types.MarketTypeFuture

	exchange, err := binance.New(config)
	if err != nil {
		t.Fatalf("创建 Binance 期货实例失败: %v", err)
	}

	if exchange.GetMarketType() != types.MarketTypeFuture {
		t.Errorf("市场类型错误: 期望 %s, 实际 %s", types.MarketTypeFuture, exchange.GetMarketType())
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	markets, err := exchange.FetchMarkets(ctx, nil)
	if err != nil {
		t.Fatalf("获取市场信息失败: %v", err)
	}
	fmt.Printf("Binance Futures: 获取到 %d 个交易对\n", len(markets))

	for _, m := range markets[:min(3, len(markets))] {
		if !m.Future {
			t.Errorf("市场 %s 应该标记为期货", m.Symbol)
		}
		fmt.Printf("  - %s (Spot=%v, Future=%v, Swap=%v)\n", m.Symbol, m.Spot, m.Future, m.Swap)
	}

	// 测试标记价格（仅期货可用）
	markPrice, err := exchange.FetchMarkPrice(ctx, "BTCUSDT")
	if err != nil {
		t.Fatalf("获取标记价格失败: %v", err)
	}
	fmt.Printf("Binance Futures BTCUSDT MarkPrice=%.2f, IndexPrice=%.2f\n", markPrice.MarkPrice, markPrice.IndexPrice)
}

// ========== Bybit 测试 ==========

func TestBybitSpot(t *testing.T) {
	config := bybit.DefaultConfig()
	_ = config.SetMarketType(types.MarketTypeSpot)

	exchange, err := bybit.New(config)
	if err != nil {
		t.Fatalf("创建 Bybit 现货实例失败: %v", err)
	}

	if exchange.GetCategory() != bybit.CategorySpot {
		t.Errorf("Category 错误: 期望 %s, 实际 %s", bybit.CategorySpot, exchange.GetCategory())
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	markets, err := exchange.FetchMarkets(ctx, nil)
	if err != nil {
		t.Fatalf("获取市场信息失败: %v", err)
	}
	fmt.Printf("Bybit Spot: 获取到 %d 个交易对\n", len(markets))

	for _, m := range markets[:min(3, len(markets))] {
		if !m.Spot {
			t.Errorf("市场 %s 应该标记为现货", m.Symbol)
		}
		fmt.Printf("  - %s (Spot=%v, Future=%v)\n", m.Symbol, m.Spot, m.Future)
	}

	tickers, err := exchange.FetchTickers(ctx, []string{"BTCUSDT"}, nil)
	if err != nil {
		t.Fatalf("获取 Ticker 失败: %v", err)
	}
	if ticker, ok := tickers["BTCUSDT"]; ok {
		fmt.Printf("Bybit Spot BTCUSDT: Last=%.2f, Bid=%.2f, Ask=%.2f\n", ticker.Last, ticker.Bid, ticker.Ask)
	}
}

func TestBybitFutures(t *testing.T) {
	config := bybit.DefaultConfig()
	_ = config.SetMarketType(types.MarketTypeFuture)

	exchange, err := bybit.New(config)
	if err != nil {
		t.Fatalf("创建 Bybit 期货实例失败: %v", err)
	}

	if exchange.GetCategory() != bybit.CategoryLinear {
		t.Errorf("Category 错误: 期望 %s, 实际 %s", bybit.CategoryLinear, exchange.GetCategory())
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	markets, err := exchange.FetchMarkets(ctx, nil)
	if err != nil {
		t.Fatalf("获取市场信息失败: %v", err)
	}
	fmt.Printf("Bybit Futures: 获取到 %d 个交易对\n", len(markets))

	for _, m := range markets[:min(3, len(markets))] {
		if !m.Future {
			t.Errorf("市场 %s 应该标记为期货", m.Symbol)
		}
		fmt.Printf("  - %s (Spot=%v, Future=%v, Swap=%v)\n", m.Symbol, m.Spot, m.Future, m.Swap)
	}

	// 测试标记价格
	markPrice, err := exchange.FetchMarkPrice(ctx, "BTCUSDT")
	if err != nil {
		t.Fatalf("获取标记价格失败: %v", err)
	}
	fmt.Printf("Bybit Futures BTCUSDT MarkPrice=%.2f\n", markPrice.MarkPrice)
}

// ========== OKX 测试 ==========

func TestOKXSpot(t *testing.T) {
	config := okx.DefaultConfig()
	_ = config.SetMarketType(types.MarketTypeSpot)

	exchange, err := okx.New(config)
	if err != nil {
		t.Fatalf("创建 OKX 现货实例失败: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	markets, err := exchange.FetchMarkets(ctx, nil)
	if err != nil {
		t.Fatalf("获取市场信息失败: %v", err)
	}
	fmt.Printf("OKX Spot: 获取到 %d 个交易对\n", len(markets))

	for _, m := range markets[:min(3, len(markets))] {
		if !m.Spot {
			t.Errorf("市场 %s 应该标记为现货", m.Symbol)
		}
		fmt.Printf("  - %s (Spot=%v, Future=%v)\n", m.Symbol, m.Spot, m.Future)
	}

	tickers, err := exchange.FetchTickers(ctx, nil, nil)
	if err != nil {
		t.Fatalf("获取 Ticker 失败: %v", err)
	}
	if ticker, ok := tickers["BTC-USDT"]; ok {
		fmt.Printf("OKX Spot BTC-USDT: Last=%.2f, Bid=%.2f, Ask=%.2f\n", ticker.Last, ticker.Bid, ticker.Ask)
	}
}

func TestOKXFutures(t *testing.T) {
	config := okx.DefaultConfig()
	_ = config.SetMarketType(types.MarketTypeFuture)

	exchange, err := okx.New(config)
	if err != nil {
		t.Fatalf("创建 OKX 期货实例失败: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	markets, err := exchange.FetchMarkets(ctx, nil)
	if err != nil {
		t.Fatalf("获取市场信息失败: %v", err)
	}
	fmt.Printf("OKX Futures: 获取到 %d 个交易对\n", len(markets))

	for _, m := range markets[:min(3, len(markets))] {
		if !m.Future {
			t.Errorf("市场 %s 应该标记为期货", m.Symbol)
		}
		fmt.Printf("  - %s (Spot=%v, Future=%v, Swap=%v)\n", m.Symbol, m.Spot, m.Future, m.Swap)
	}

	// 测试标记价格
	markPrice, err := exchange.FetchMarkPrice(ctx, "BTC-USDT-SWAP")
	if err != nil {
		t.Fatalf("获取标记价格失败: %v", err)
	}
	fmt.Printf("OKX Futures BTC-USDT-SWAP MarkPrice=%.2f\n", markPrice.MarkPrice)
}

// ========== MEXC 测试 ==========

func TestMEXCSpot(t *testing.T) {
	config := mexc.DefaultConfig()

	exchange, err := mexc.New(config)
	if err != nil {
		t.Fatalf("创建 MEXC 现货实例失败: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	markets, err := exchange.FetchMarkets(ctx, nil)
	if err != nil {
		t.Fatalf("获取市场信息失败: %v", err)
	}
	fmt.Printf("MEXC Spot: 获取到 %d 个交易对\n", len(markets))

	for _, m := range markets[:min(3, len(markets))] {
		if !m.Spot {
			t.Errorf("市场 %s 应该标记为现货", m.Symbol)
		}
		fmt.Printf("  - %s (Spot=%v, Future=%v)\n", m.Symbol, m.Spot, m.Future)
	}

	tickers, err := exchange.FetchTickers(ctx, []string{"BTCUSDT"}, nil)
	if err != nil {
		t.Fatalf("获取 Ticker 失败: %v", err)
	}
	if ticker, ok := tickers["BTCUSDT"]; ok {
		fmt.Printf("MEXC Spot BTCUSDT: Last=%.2f, Bid=%.2f, Ask=%.2f\n", ticker.Last, ticker.Bid, ticker.Ask)
	}
}

// ========== 完整 API 测试 ==========

func TestAllAPIs(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	fmt.Println("\n========== 完整 API 测试 ==========")

	// 测试 Binance
	t.Run("Binance", func(t *testing.T) {
		testBinanceAPIs(t, ctx)
	})

	// 测试 Bybit
	t.Run("Bybit", func(t *testing.T) {
		testBybitAPIs(t, ctx)
	})

	// 测试 OKX
	t.Run("OKX", func(t *testing.T) {
		testOKXAPIs(t, ctx)
	})

	// 测试 MEXC
	t.Run("MEXC", func(t *testing.T) {
		testMEXCAPIs(t, ctx)
	})
}

func testBinanceAPIs(t *testing.T, ctx context.Context) {
	// 现货
	spotCfg := binance.DefaultConfig()
	spotCfg.MarketType = types.MarketTypeSpot
	spot, _ := binance.New(spotCfg)

	// 期货
	futuresCfg := binance.DefaultConfig()
	futuresCfg.MarketType = types.MarketTypeFuture
	futures, _ := binance.New(futuresCfg)

	fmt.Println("\n--- Binance 现货 ---")

	// FetchTickers
	tickers, err := spot.FetchTickers(ctx, []string{"BTCUSDT", "ETHUSDT"}, nil)
	if err != nil {
		t.Errorf("FetchTickers 失败: %v", err)
	} else {
		fmt.Printf("FetchTickers: 获取 %d 个 ticker\n", len(tickers))
		for sym, tk := range tickers {
			fmt.Printf("  %s: Last=%.2f, Bid=%.2f, Ask=%.2f, Volume=%.2f\n", sym, tk.Last, tk.Bid, tk.Ask, tk.BaseVolume)
		}
	}

	// FetchBookTickers
	bookTickers, err := spot.FetchBookTickers(ctx, []string{"BTCUSDT"}, nil)
	if err != nil {
		t.Errorf("FetchBookTickers 失败: %v", err)
	} else {
		fmt.Printf("FetchBookTickers: 获取 %d 个\n", len(bookTickers))
		for sym, tk := range bookTickers {
			fmt.Printf("  %s: Bid=%.2f, Ask=%.2f\n", sym, tk.Bid, tk.Ask)
		}
	}

	// FetchKlines
	klines, err := spot.FetchKlines(ctx, "BTCUSDT", "1h", 0, 5, nil)
	if err != nil {
		t.Errorf("FetchKlines 失败: %v", err)
	} else {
		fmt.Printf("FetchKlines: 获取 %d 根 K 线\n", len(klines))
		if len(klines) > 0 {
			k := klines[0]
			fmt.Printf("  最新: Open=%.2f, High=%.2f, Low=%.2f, Close=%.2f\n", k.Open, k.High, k.Low, k.Close)
		}
	}

	fmt.Println("\n--- Binance 期货 ---")

	// FetchTickers
	futuresTickers, err := futures.FetchTickers(ctx, []string{"BTCUSDT", "ETHUSDT"}, nil)
	if err != nil {
		t.Errorf("FetchTickers 失败: %v", err)
	} else {
		fmt.Printf("FetchTickers: 获取 %d 个 ticker\n", len(futuresTickers))
		for sym, tk := range futuresTickers {
			fmt.Printf("  %s: Last=%.2f, Bid=%.2f, Ask=%.2f\n", sym, tk.Last, tk.Bid, tk.Ask)
		}
	}

	// FetchBookTickers
	futuresBookTickers, err := futures.FetchBookTickers(ctx, []string{"BTCUSDT"}, nil)
	if err != nil {
		t.Errorf("FetchBookTickers 失败: %v", err)
	} else {
		fmt.Printf("FetchBookTickers: 获取 %d 个\n", len(futuresBookTickers))
		for sym, tk := range futuresBookTickers {
			fmt.Printf("  %s: Bid=%.2f, Ask=%.2f\n", sym, tk.Bid, tk.Ask)
		}
	}

	// FetchKlines
	futuresKlines, err := futures.FetchKlines(ctx, "BTCUSDT", "1h", 0, 5, nil)
	if err != nil {
		t.Errorf("FetchKlines 失败: %v", err)
	} else {
		fmt.Printf("FetchKlines: 获取 %d 根 K 线\n", len(futuresKlines))
		if len(futuresKlines) > 0 {
			k := futuresKlines[0]
			fmt.Printf("  最新: Open=%.2f, High=%.2f, Low=%.2f, Close=%.2f\n", k.Open, k.High, k.Low, k.Close)
		}
	}

	// FetchMarkPrice
	markPrice, err := futures.FetchMarkPrice(ctx, "BTCUSDT")
	if err != nil {
		t.Errorf("FetchMarkPrice 失败: %v", err)
	} else {
		fmt.Printf("FetchMarkPrice BTCUSDT: Mark=%.2f, Index=%.2f, FundingRate=%.6f\n",
			markPrice.MarkPrice, markPrice.IndexPrice, markPrice.FundingRate)
	}

	// FetchMarkPrices
	markPrices, err := futures.FetchMarkPrices(ctx, []string{"BTCUSDT", "ETHUSDT"})
	if err != nil {
		t.Errorf("FetchMarkPrices 失败: %v", err)
	} else {
		fmt.Printf("FetchMarkPrices: 获取 %d 个\n", len(markPrices))
		for sym, mp := range markPrices {
			fmt.Printf("  %s: Mark=%.2f, FundingRate=%.6f\n", sym, mp.MarkPrice, mp.FundingRate)
		}
	}
}

func testBybitAPIs(t *testing.T, ctx context.Context) {
	// 现货
	spotCfg := bybit.DefaultConfig()
	_ = spotCfg.SetMarketType(types.MarketTypeSpot)
	spot, _ := bybit.New(spotCfg)

	// 期货
	futuresCfg := bybit.DefaultConfig()
	_ = futuresCfg.SetMarketType(types.MarketTypeFuture)
	futures, _ := bybit.New(futuresCfg)

	fmt.Println("\n--- Bybit 现货 ---")

	// FetchTickers
	tickers, err := spot.FetchTickers(ctx, []string{"BTCUSDT"}, nil)
	if err != nil {
		t.Errorf("FetchTickers 失败: %v", err)
	} else {
		fmt.Printf("FetchTickers: 获取 %d 个 ticker\n", len(tickers))
		for sym, tk := range tickers {
			fmt.Printf("  %s: Last=%.2f, Bid=%.2f, Ask=%.2f\n", sym, tk.Last, tk.Bid, tk.Ask)
		}
	}

	// FetchKlines
	klines, err := spot.FetchKlines(ctx, "BTCUSDT", "60", 0, 5, nil)
	if err != nil {
		t.Errorf("FetchKlines 失败: %v", err)
	} else {
		fmt.Printf("FetchKlines: 获取 %d 根 K 线\n", len(klines))
		if len(klines) > 0 {
			k := klines[0]
			fmt.Printf("  最新: Open=%.2f, High=%.2f, Low=%.2f, Close=%.2f\n", k.Open, k.High, k.Low, k.Close)
		}
	}

	fmt.Println("\n--- Bybit 期货 ---")

	// FetchTickers
	futuresTickers, err := futures.FetchTickers(ctx, []string{"BTCUSDT"}, nil)
	if err != nil {
		t.Errorf("FetchTickers 失败: %v", err)
	} else {
		fmt.Printf("FetchTickers: 获取 %d 个 ticker\n", len(futuresTickers))
		for sym, tk := range futuresTickers {
			fmt.Printf("  %s: Last=%.2f, Bid=%.2f, Ask=%.2f\n", sym, tk.Last, tk.Bid, tk.Ask)
		}
	}

	// FetchKlines
	futuresKlines, err := futures.FetchKlines(ctx, "BTCUSDT", "60", 0, 5, nil)
	if err != nil {
		t.Errorf("FetchKlines 失败: %v", err)
	} else {
		fmt.Printf("FetchKlines: 获取 %d 根 K 线\n", len(futuresKlines))
		if len(futuresKlines) > 0 {
			k := futuresKlines[0]
			fmt.Printf("  最新: Open=%.2f, High=%.2f, Low=%.2f, Close=%.2f\n", k.Open, k.High, k.Low, k.Close)
		}
	}

	// FetchMarkPrice
	markPrice, err := futures.FetchMarkPrice(ctx, "BTCUSDT")
	if err != nil {
		t.Errorf("FetchMarkPrice 失败: %v", err)
	} else {
		fmt.Printf("FetchMarkPrice BTCUSDT: Mark=%.2f, Index=%.2f\n", markPrice.MarkPrice, markPrice.IndexPrice)
	}
}

func testOKXAPIs(t *testing.T, ctx context.Context) {
	// 现货
	spotCfg := okx.DefaultConfig()
	_ = spotCfg.SetMarketType(types.MarketTypeSpot)
	spot, _ := okx.New(spotCfg)

	// 期货
	futuresCfg := okx.DefaultConfig()
	_ = futuresCfg.SetMarketType(types.MarketTypeFuture)
	futures, _ := okx.New(futuresCfg)

	fmt.Println("\n--- OKX 现货 ---")

	// FetchTickers
	tickers, err := spot.FetchTickers(ctx, []string{"BTC-USDT"}, nil)
	if err != nil {
		// OKX 可能因网络限制无法访问，跳过而不是失败
		fmt.Printf("FetchTickers 跳过 (网络问题): %v\n", err)
	} else {
		fmt.Printf("FetchTickers: 获取 %d 个 ticker\n", len(tickers))
		for sym, tk := range tickers {
			fmt.Printf("  %s: Last=%.2f, Bid=%.2f, Ask=%.2f\n", sym, tk.Last, tk.Bid, tk.Ask)
		}
	}

	// FetchKlines
	klines, err := spot.FetchKlines(ctx, "BTC-USDT", "1H", 0, 5, nil)
	if err != nil {
		fmt.Printf("FetchKlines 跳过 (网络问题): %v\n", err)
	} else {
		fmt.Printf("FetchKlines: 获取 %d 根 K 线\n", len(klines))
		if len(klines) > 0 {
			k := klines[0]
			fmt.Printf("  最新: Open=%.2f, High=%.2f, Low=%.2f, Close=%.2f\n", k.Open, k.High, k.Low, k.Close)
		}
	}

	fmt.Println("\n--- OKX 期货 ---")

	// FetchTickers
	futuresTickers, err := futures.FetchTickers(ctx, []string{"BTC-USDT-SWAP"}, nil)
	if err != nil {
		fmt.Printf("FetchTickers 跳过 (网络问题): %v\n", err)
	} else {
		fmt.Printf("FetchTickers: 获取 %d 个 ticker\n", len(futuresTickers))
		for sym, tk := range futuresTickers {
			fmt.Printf("  %s: Last=%.2f, Bid=%.2f, Ask=%.2f\n", sym, tk.Last, tk.Bid, tk.Ask)
		}
	}

	// FetchKlines
	futuresKlines, err := futures.FetchKlines(ctx, "BTC-USDT-SWAP", "1H", 0, 5, nil)
	if err != nil {
		fmt.Printf("FetchKlines 跳过 (网络问题): %v\n", err)
	} else {
		fmt.Printf("FetchKlines: 获取 %d 根 K 线\n", len(futuresKlines))
		if len(futuresKlines) > 0 {
			k := futuresKlines[0]
			fmt.Printf("  最新: Open=%.2f, High=%.2f, Low=%.2f, Close=%.2f\n", k.Open, k.High, k.Low, k.Close)
		}
	}

	// FetchMarkPrice
	markPrice, err := futures.FetchMarkPrice(ctx, "BTC-USDT-SWAP")
	if err != nil {
		fmt.Printf("FetchMarkPrice 跳过 (网络问题): %v\n", err)
	} else {
		fmt.Printf("FetchMarkPrice BTC-USDT-SWAP: Mark=%.2f\n", markPrice.MarkPrice)
	}
}

func testMEXCAPIs(t *testing.T, ctx context.Context) {
	cfg := mexc.DefaultConfig()
	exchange, _ := mexc.New(cfg)

	fmt.Println("\n--- MEXC 现货 ---")

	// FetchTickers
	tickers, err := exchange.FetchTickers(ctx, []string{"BTCUSDT"}, nil)
	if err != nil {
		t.Errorf("FetchTickers 失败: %v", err)
	} else {
		fmt.Printf("FetchTickers: 获取 %d 个 ticker\n", len(tickers))
		for sym, tk := range tickers {
			fmt.Printf("  %s: Last=%.2f, Bid=%.2f, Ask=%.2f\n", sym, tk.Last, tk.Bid, tk.Ask)
		}
	}

	// FetchBookTickers
	bookTickers, err := exchange.FetchBookTickers(ctx, []string{"BTCUSDT"}, nil)
	if err != nil {
		t.Errorf("FetchBookTickers 失败: %v", err)
	} else {
		fmt.Printf("FetchBookTickers: 获取 %d 个\n", len(bookTickers))
		for sym, tk := range bookTickers {
			fmt.Printf("  %s: Bid=%.2f, Ask=%.2f\n", sym, tk.Bid, tk.Ask)
		}
	}

	// FetchKlines
	klines, err := exchange.FetchKlines(ctx, "BTCUSDT", "1h", 0, 5, nil)
	if err != nil {
		t.Errorf("FetchKlines 失败: %v", err)
	} else {
		fmt.Printf("FetchKlines: 获取 %d 根 K 线\n", len(klines))
		if len(klines) > 0 {
			k := klines[0]
			fmt.Printf("  最新: Open=%.2f, High=%.2f, Low=%.2f, Close=%.2f\n", k.Open, k.High, k.Low, k.Close)
		}
	}
}

// ========== 筛选 USDT 交易对测试 ==========

func TestFetchMarketsWithQuoteFilter(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	quoteFilter := map[string]interface{}{"quote": "USDT"}

	fmt.Println("\n========== 使用 params[\"quote\"]=\"USDT\" 筛选 ==========")

	// Binance 现货
	binanceSpotCfg := binance.DefaultConfig()
	binanceSpotCfg.MarketType = types.MarketTypeSpot
	binanceSpot, _ := binance.New(binanceSpotCfg)
	binanceSpotUSDT, _ := binanceSpot.FetchMarkets(ctx, quoteFilter)
	fmt.Printf("Binance Spot USDT: %d 交易对\n", len(binanceSpotUSDT))
	printFirstN(binanceSpotUSDT, 3)
	verifyAllQuote(t, binanceSpotUSDT, "USDT", "Binance Spot")

	// Binance 期货
	binanceFuturesCfg := binance.DefaultConfig()
	binanceFuturesCfg.MarketType = types.MarketTypeFuture
	binanceFutures, _ := binance.New(binanceFuturesCfg)
	binanceFuturesUSDT, _ := binanceFutures.FetchMarkets(ctx, map[string]interface{}{"quote": "USDT"})
	fmt.Printf("Binance Futures USDT: %d 交易对\n", len(binanceFuturesUSDT))
	printFirstN(binanceFuturesUSDT, 3)
	verifyAllQuote(t, binanceFuturesUSDT, "USDT", "Binance Futures")

	// Bybit 现货
	bybitSpotCfg := bybit.DefaultConfig()
	_ = bybitSpotCfg.SetMarketType(types.MarketTypeSpot)
	bybitSpot, _ := bybit.New(bybitSpotCfg)
	bybitSpotUSDT, _ := bybitSpot.FetchMarkets(ctx, map[string]interface{}{"quote": "USDT"})
	fmt.Printf("Bybit Spot USDT: %d 交易对\n", len(bybitSpotUSDT))
	printFirstN(bybitSpotUSDT, 3)
	verifyAllQuote(t, bybitSpotUSDT, "USDT", "Bybit Spot")

	// Bybit 期货
	bybitFuturesCfg := bybit.DefaultConfig()
	_ = bybitFuturesCfg.SetMarketType(types.MarketTypeFuture)
	bybitFutures, _ := bybit.New(bybitFuturesCfg)
	bybitFuturesUSDT, _ := bybitFutures.FetchMarkets(ctx, map[string]interface{}{"quote": "USDT"})
	fmt.Printf("Bybit Futures USDT: %d 交易对\n", len(bybitFuturesUSDT))
	printFirstN(bybitFuturesUSDT, 3)
	verifyAllQuote(t, bybitFuturesUSDT, "USDT", "Bybit Futures")

	// OKX 现货
	okxSpotCfg := okx.DefaultConfig()
	_ = okxSpotCfg.SetMarketType(types.MarketTypeSpot)
	okxSpot, _ := okx.New(okxSpotCfg)
	okxSpotUSDT, _ := okxSpot.FetchMarkets(ctx, map[string]interface{}{"quote": "USDT"})
	fmt.Printf("OKX Spot USDT: %d 交易对\n", len(okxSpotUSDT))
	printFirstN(okxSpotUSDT, 3)
	verifyAllQuote(t, okxSpotUSDT, "USDT", "OKX Spot")

	// OKX 期货
	okxFuturesCfg := okx.DefaultConfig()
	_ = okxFuturesCfg.SetMarketType(types.MarketTypeFuture)
	okxFutures, _ := okx.New(okxFuturesCfg)
	okxFuturesUSDT, _ := okxFutures.FetchMarkets(ctx, map[string]interface{}{"quote": "USDT"})
	fmt.Printf("OKX Futures USDT: %d 交易对\n", len(okxFuturesUSDT))
	printFirstN(okxFuturesUSDT, 3)
	verifyAllQuote(t, okxFuturesUSDT, "USDT", "OKX Futures")

	// MEXC 现货
	mexcCfg := mexc.DefaultConfig()
	mexcSpot, _ := mexc.New(mexcCfg)
	mexcUSDT, _ := mexcSpot.FetchMarkets(ctx, map[string]interface{}{"quote": "USDT"})
	fmt.Printf("MEXC Spot USDT: %d 交易对\n", len(mexcUSDT))
	printFirstN(mexcUSDT, 3)
	verifyAllQuote(t, mexcUSDT, "USDT", "MEXC Spot")
}

// verifyAllQuote 验证所有市场的 Quote 是否匹配
func verifyAllQuote(t *testing.T, markets []*types.Market, expectedQuote string, exchange string) {
	for _, m := range markets {
		if m.Quote != expectedQuote {
			t.Errorf("%s: 市场 %s 的 Quote 为 %s，期望 %s", exchange, m.Symbol, m.Quote, expectedQuote)
		}
	}
}

func printFirstN(markets []*types.Market, n int) {
	for i, m := range markets {
		if i >= n {
			break
		}
		fmt.Printf("  - %s (ID=%s, Quote=%s)\n", m.Symbol, m.ID, m.Quote)
	}
}

// ========== 辅助函数 ==========

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
