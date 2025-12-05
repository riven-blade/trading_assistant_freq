package utils

import (
	"strings"
)

// ConvertFutureSymbolToMarketID 将期货格式的symbol转换为marketid
// 例如: "DOGE/USDT:USDT" -> "DOGEUSDT"
func ConvertFutureSymbolToMarketID(symbol string) string {
	if symbol == "" {
		return ""
	}

	// 处理期货格式: "DOGE/USDT:USDT"
	// 分离基础交易对和结算币种
	parts := strings.Split(symbol, ":")
	baseSymbol := parts[0] // "DOGE/USDT"

	// 移除斜杠，转换为市场ID格式
	marketID := strings.ReplaceAll(baseSymbol, "/", "")

	return marketID
}

// ConvertSpotSymbolToMarketID 将现货格式的symbol转换为marketid
// 例如: "DOGE/USDT" -> "DOGEUSDT"
func ConvertSpotSymbolToMarketID(symbol string) string {
	if symbol == "" {
		return ""
	}

	// 移除斜杠
	return strings.ReplaceAll(symbol, "/", "")
}

// ConvertSymbolToMarketID 通用的symbol到marketid转换函数
// 自动检测是期货还是现货格式并进行相应转换
func ConvertSymbolToMarketID(symbol string) string {
	if symbol == "" {
		return ""
	}

	// 检查是否为期货格式 (包含冒号)
	if strings.Contains(symbol, ":") {
		return ConvertFutureSymbolToMarketID(symbol)
	}

	// 现货格式
	return ConvertSpotSymbolToMarketID(symbol)
}

// ExtractBaseAndQuote 从symbol中提取基础货币和计价货币
// 例如: "DOGE/USDT:USDT" -> ("DOGE", "USDT", "USDT")
// 例如: "DOGE/USDT" -> ("DOGE", "USDT", "")
func ExtractBaseAndQuote(symbol string) (base, quote, settle string) {
	if symbol == "" {
		return "", "", ""
	}

	// 处理期货格式
	if strings.Contains(symbol, ":") {
		parts := strings.Split(symbol, ":")
		baseSymbol := parts[0] // "DOGE/USDT"
		if len(parts) > 1 {
			settle = parts[1] // "USDT"
		}

		// 分离基础货币和计价货币
		if strings.Contains(baseSymbol, "/") {
			baseParts := strings.Split(baseSymbol, "/")
			if len(baseParts) >= 2 {
				base = baseParts[0]  // "DOGE"
				quote = baseParts[1] // "USDT"
			}
		}
	} else {
		// 现货格式
		if strings.Contains(symbol, "/") {
			parts := strings.Split(symbol, "/")
			if len(parts) >= 2 {
				base = parts[0]  // "DOGE"
				quote = parts[1] // "USDT"
			}
		}
	}

	return base, quote, settle
}

// IsSymbolFuture 检查symbol是否为期货格式
func IsSymbolFuture(symbol string) bool {
	return strings.Contains(symbol, ":")
}

// IsSymbolSpot 检查symbol是否为现货格式
func IsSymbolSpot(symbol string) bool {
	return strings.Contains(symbol, "/") && !strings.Contains(symbol, ":")
}

// ConvertMarketIDToFutureSymbol 将marketid转换为期货格式的symbol
// 例如: "DOGEUSDT" -> "DOGE/USDT:USDT"
// 默认假设以USDT结尾的都是USDT结算的期货
func ConvertMarketIDToFutureSymbol(marketID string) string {
	if marketID == "" {
		return ""
	}

	// 大部分都是以USDT结尾，特殊处理
	if strings.HasSuffix(marketID, "USDT") {
		base := strings.TrimSuffix(marketID, "USDT")
		return base + "/USDT:USDT"
	}

	// 如果以其他结尾，需要根据实际情况处理
	// 这里先简化处理，可以根据需要扩展
	if strings.HasSuffix(marketID, "USDC") {
		base := strings.TrimSuffix(marketID, "USDC")
		return base + "/USDC:USDC"
	}

	if strings.HasSuffix(marketID, "BTC") {
		base := strings.TrimSuffix(marketID, "BTC")
		return base + "/BTC:BTC"
	}

	if strings.HasSuffix(marketID, "ETH") {
		base := strings.TrimSuffix(marketID, "ETH")
		return base + "/ETH:ETH"
	}

	// 默认按USDT处理
	return marketID + "/USDT:USDT"
}

// ConvertMarketIDToSpotSymbol 将marketid转换为现货格式的symbol
// 例如: "DOGEUSDT" -> "DOGE/USDT"
func ConvertMarketIDToSpotSymbol(marketID string) string {
	if marketID == "" {
		return ""
	}

	// 大部分都是以USDT结尾
	if strings.HasSuffix(marketID, "USDT") {
		base := strings.TrimSuffix(marketID, "USDT")
		return base + "/USDT"
	}

	if strings.HasSuffix(marketID, "USDC") {
		base := strings.TrimSuffix(marketID, "USDC")
		return base + "/USDC"
	}

	if strings.HasSuffix(marketID, "BTC") {
		base := strings.TrimSuffix(marketID, "BTC")
		return base + "/BTC"
	}

	if strings.HasSuffix(marketID, "ETH") {
		base := strings.TrimSuffix(marketID, "ETH")
		return base + "/ETH"
	}

	// 默认按USDT处理
	return marketID + "/USDT"
}

// ConvertMarketIDToSymbol 根据市场类型将marketid转换为对应格式的symbol
// marketType: "spot" 或 "future"
// 例如: "DOGEUSDT", "spot" -> "DOGE/USDT"
// 例如: "DOGEUSDT", "future" -> "DOGE/USDT:USDT"
func ConvertMarketIDToSymbol(marketID string, marketType string) string {
	if marketType == "spot" {
		return ConvertMarketIDToSpotSymbol(marketID)
	}
	return ConvertMarketIDToFutureSymbol(marketID)
}
