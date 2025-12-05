package apis

import (
	"path/filepath"
	"trading_assistant/controllers"
	"trading_assistant/core"
	"trading_assistant/pkg/exchange_factory"
	"trading_assistant/pkg/freqtrade"
	"trading_assistant/pkg/middleware"
	"trading_assistant/pkg/websocket"

	"github.com/gin-gonic/gin"
)

func SetupRoutes(r *gin.Engine, exchangeClient exchange_factory.ExchangeInterface, marketManager *core.MarketManager, freqtradeController *freqtrade.Controller) {
	// 创建控制器实例
	coinController := controllers.NewCoinController(exchangeClient, marketManager)
	priceController := &controllers.PriceController{}
	authController := &controllers.AuthController{}
	configController := controllers.NewConfigController()
	klineController := controllers.NewKlineController(exchangeClient)
	positionController := controllers.NewPositionController(freqtradeController)

	// 初始化WebSocket管理器
	wsManager := websocket.GetGlobalWebSocketManager()

	// 静态文件服务
	webBuildPath := "./web/build"

	// 服务静态资源文件
	r.Static("/static", filepath.Join(webBuildPath, "static"))
	r.StaticFile("/favicon.ico", filepath.Join(webBuildPath, "favicon.ico"))
	r.StaticFile("/favicon.svg", filepath.Join(webBuildPath, "favicon.svg"))
	r.StaticFile("/manifest.json", filepath.Join(webBuildPath, "manifest.json"))

	// 健康检查
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "ok",
			"message": "Trading Assistant API is running",
		})
	})

	// 添加认证中间件
	r.Use(middleware.AuthMiddleware())

	// WebSocket路由
	r.GET("/ws", wsManager.HandleWebSocket)

	// 认证路由
	auth := r.Group("/api/v1/auth")
	{
		auth.POST("/login", authController.Login) // 用户登录
	}

	// API版本组
	v1 := r.Group("/api/v1")
	{
		// 用户信息路由
		user := v1.Group("/user")
		{
			user.GET("/profile", authController.GetProfile) // 获取用户信息
		}
		// 币种管理路由
		coins := v1.Group("/coins")
		{
			coins.GET("", coinController.GetCoins)                  // 获取所有币种
			coins.GET("/", coinController.GetCoins)                 // 获取币种列表
			coins.GET("/selected", coinController.GetSelectedCoins) // 获取选中的币种
			coins.POST("/select", coinController.SelectCoin)        // 筛选币种
			coins.POST("/sync", coinController.SyncCoins)           // 同步币种
			coins.PUT("/tier", coinController.UpdateCoinTier)       // 更新币种等级
		}

		// 价格预估路由
		estimates := v1.Group("/estimates")
		{
			estimates.GET("/all", priceController.GetAllPriceEstimates)       // 获取所有价格预估（Orders页面需要）
			estimates.POST("", priceController.CreatePriceEstimate)           // 创建价格预估
			estimates.DELETE("/:id", priceController.DeletePriceEstimate)     // 删除价格预估
			estimates.PUT("/:id/toggle", priceController.TogglePriceEstimate) // 切换价格预估监听状态
		}

		// K线分析路由
		klines := v1.Group("/klines")
		{
			klines.GET("", klineController.GetKlines) // 获取K线数据
		}

		// 持仓管理路由
		positions := v1.Group("/positions")
		{
			positions.GET("", positionController.GetPositions)               // 获取所有持仓
			positions.GET("/summary", positionController.GetPositionSummary) // 获取持仓摘要
		}

		// 系统配置路由
		v1.GET("/config", configController.GetSystemConfig) // 获取系统配置
	}

	// 服务前端应用（SPA路由）
	r.NoRoute(func(c *gin.Context) {
		// 如果是API路由，返回404
		if len(c.Request.URL.Path) > 4 && c.Request.URL.Path[:5] == "/api/" {
			c.JSON(404, gin.H{"error": "API endpoint not found"})
			return
		}

		// 否则返回前端index.html
		c.File(filepath.Join(webBuildPath, "index.html"))
	})
}
