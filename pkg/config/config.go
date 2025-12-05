package config

import (
	"os"
	"strconv"
	"time"

	"github.com/joho/godotenv"
	"github.com/sirupsen/logrus"
)

type Config struct {
	// Redis配置
	RedisHost     string
	RedisPort     string
	RedisPassword string
	RedisDB       int

	// 服务配置
	LogLevel string
	BaseURL  string

	ExchangeType string // 交易所类型: binance, bybit, okx, mexc
	MarketType   string // 市场类型: spot, future

	// 风险管理配置
	ShortFundingRateThreshold float64 // 做空资金费率阈值，低于此阈值不开空仓

	// 认证配置
	AdminUsername string // 管理员用户名
	AdminPassword string // 管理员密码
	JWTSecret     string // JWT密钥

	FreqtradeBaseURL  string // Freqtrade API 基础URL
	FreqtradeUsername string // Freqtrade 用户名
	FreqtradePassword string // Freqtrade 密码

	// 价格管理配置
	PriceUpdateInterval time.Duration // 价格更新间隔
}

var GlobalConfig *Config

func LoadConfig() {
	// 加载.env文件
	if err := godotenv.Load(); err != nil {
		logrus.Warn("未找到.env文件，使用环境变量")
	}

	GlobalConfig = &Config{
		RedisHost:     getEnv("REDIS_HOST", "localhost"),
		RedisPort:     getEnv("REDIS_PORT", "6379"),
		RedisPassword: getEnv("REDIS_PASSWORD", ""),
		RedisDB:       getEnvInt("REDIS_DB", 0),
		LogLevel:      getEnv("LOG_LEVEL", "info"),
		BaseURL:       getEnv("BASE_URL", "localhost"),

		ExchangeType: getEnv("EXCHANGE_TYPE", "binance"), // 默认使用 binance
		MarketType:   getEnv("MARKET_TYPE", "future"),    // 默认使用期货

		ShortFundingRateThreshold: getEnvFloat("SHORT_FUNDING_RATE_THRESHOLD", -0.002), // 默认-0.2%

		AdminUsername: getEnv("ADMIN_USERNAME", "admin"),
		AdminPassword: getEnv("ADMIN_PASSWORD", ""),
		JWTSecret:     getEnv("JWT_SECRET", "d4f8c1b2e3f4a5b6c7d8e9f0a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6q7r8s9t0"),

		FreqtradeBaseURL:  getEnv("FREQTRADE_BASE_URL", "http://localhost:8080"),
		FreqtradeUsername: getEnv("FREQTRADE_USERNAME", ""),
		FreqtradePassword: getEnv("FREQTRADE_PASSWORD", ""),

		PriceUpdateInterval: getEnvDuration("PRICE_UPDATE_INTERVAL", "15s"), // 默认15秒
	}

	// 设置日志级别
	level, err := logrus.ParseLevel(GlobalConfig.LogLevel)
	if err != nil {
		level = logrus.InfoLevel
	}
	logrus.SetLevel(level)

	logrus.Info("配置加载完成")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

func getEnvFloat(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if floatValue, err := strconv.ParseFloat(value, 64); err == nil {
			return floatValue
		}
	}
	return defaultValue
}

func getEnvDuration(key, defaultValue string) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
		logrus.Warnf("无法解析环境变量 %s 的时间间隔值: %s，使用默认值: %s", key, value, defaultValue)
	}

	if duration, err := time.ParseDuration(defaultValue); err == nil {
		return duration
	}

	logrus.Errorf("无法解析默认时间间隔值: %s，使用15秒", defaultValue)
	return 15 * time.Second
}
