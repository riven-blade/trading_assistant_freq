package servers

import (
	"fmt"
	"trading_assistant/apis"
	"trading_assistant/core"
	"trading_assistant/pkg/config"
	"trading_assistant/pkg/exchange_factory"
	"trading_assistant/pkg/freqtrade"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

type HTTPServer struct {
	engine              *gin.Engine
	port                string
	exchangeClient      exchange_factory.ExchangeInterface
	marketManager       *core.MarketManager
	freqtradeController *freqtrade.Controller
}

// NewHTTPServer 创建HTTP服务器
func NewHTTPServer(exchangeClient exchange_factory.ExchangeInterface, marketManager *core.MarketManager, freqtradeController *freqtrade.Controller) *HTTPServer {
	// 设置Gin模式
	if config.GlobalConfig.LogLevel == "debug" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	engine := gin.Default()

	// 设置路由
	apis.SetupRoutes(engine, exchangeClient, marketManager, freqtradeController)

	return &HTTPServer{
		engine:              engine,
		port:                "8080",
		exchangeClient:      exchangeClient,
		marketManager:       marketManager,
		freqtradeController: freqtradeController,
	}
}

// Start 启动HTTP服务器
func (s *HTTPServer) Start() {
	addr := fmt.Sprintf(":%s", s.port)
	logrus.Infof("HTTP服务器启动在端口 %s", s.port)

	if err := s.engine.Run(addr); err != nil {
		logrus.Fatalf("HTTP服务器启动失败: %v", err)
	}
}
