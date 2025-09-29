package main

import (
	"os"
	"os/signal"
	"syscall"
	"trading_assistant/core"
	"trading_assistant/pkg/config"
	"trading_assistant/pkg/exchange_factory"
	"trading_assistant/pkg/freqtrade"
	"trading_assistant/pkg/redis"
	"trading_assistant/servers"

	"github.com/sirupsen/logrus"
)

func main() {
	// 设置日志级别
	logrus.SetLevel(logrus.InfoLevel)
	logrus.Info("启动交易助手...")

	// 加载配置
	config.LoadConfig()

	// 初始化Redis
	if err := redis.InitRedis(); err != nil {
		logrus.Fatalf("Redis init fail: %v", err)
	}

	// 初始化交易所客户端
	factory := exchange_factory.NewExchangeFactory()
	exchangeClient, err := factory.CreateFromConfig()
	if err != nil {
		logrus.Fatalf("交易所客户端初始化失败: %v", err)
	}
	logrus.Infof("%s 客户端已初始化", exchangeClient.GetName())

	// 初始化市场数据管理器并同步数据
	marketManager := core.NewMarketManager(exchangeClient)
	if err := marketManager.SyncMarketAndPriceData(); err != nil {
		logrus.Errorf("同步市场数据和价格数据失败: %v", err)
	}

	// 初始化 Freqtrade 控制器
	if config.GlobalConfig.FreqtradeBaseURL == "" || config.GlobalConfig.FreqtradeUsername == "" || config.GlobalConfig.FreqtradePassword == "" {
		logrus.Fatal("Freqtrade 已启用但配置不完整，请检查 FREQTRADE_BASE_URL, FREQTRADE_USERNAME, FREQTRADE_PASSWORD")
	}

	freqtradeController := freqtrade.NewController(
		config.GlobalConfig.FreqtradeBaseURL,
		config.GlobalConfig.FreqtradeUsername,
		config.GlobalConfig.FreqtradePassword,
		redis.GlobalRedisClient,
	)

	// 创建消息通道用于 freqtrade 通知
	freqtradeMessageChan := make(chan string, 100)
	go func() {
		for range freqtradeMessageChan {
			// Telegram通知已移除
		}
	}()

	// 初始化 freqtrade 连接
	if err := freqtradeController.Init(freqtradeMessageChan); err != nil {
		logrus.Fatalf("Freqtrade 初始化失败: %v", err)
	}
	logrus.Info("Freqtrade 控制器已初始化")

	// 初始化核心组件
	core.InitPriceMonitor(freqtradeController)

	// 启动价格订阅
	if err := marketManager.StartPriceSubscriptions(); err != nil {
		logrus.Errorf("启动价格订阅失败: %v", err)
	}

	// 启动价格监控
	core.GlobalPriceMonitor.Start()

	// 创建HTTP服务器
	server := servers.NewHTTPServer(exchangeClient, marketManager, freqtradeController)
	go func() {
		server.Start()
	}()

	logrus.Info("交易助手启动完成!")

	// 优雅关闭
	gracefulShutdown(server, exchangeClient, marketManager, freqtradeController)
}

// gracefulShutdown 优雅关闭
func gracefulShutdown(server *servers.HTTPServer, exchangeClient exchange_factory.ExchangeInterface, marketManager *core.MarketManager, freqtradeController *freqtrade.Controller) {
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logrus.Info("正在关闭交易助手...")

	// 停止HTTP服务器 (当前实现没有优雅关闭，直接退出)
	logrus.Info("HTTP服务器将随程序退出关闭")

	// 停止 Freqtrade 控制器
	if freqtradeController != nil {
		freqtradeController.Stop()
	}

	// 停止价格订阅
	if marketManager != nil {
		marketManager.StopPriceSubscriptions()
	}

	// 停止核心组件
	if core.GlobalPriceMonitor != nil {
		core.GlobalPriceMonitor.Stop()
	}

	logrus.Info("交易助手已关闭")
}
