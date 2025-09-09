package telegram

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
	"trading_assistant/models"
	"trading_assistant/pkg/config"
	"trading_assistant/pkg/exchanges/types"
	"trading_assistant/pkg/redis"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"github.com/sirupsen/logrus"
)

const (
	MaxMessageLength = 4096 // Telegram单条消息最大长度
)

type TelegramClient struct {
	bot    *tgbotapi.BotAPI
	chatID int64
}

var GlobalTelegramClient *TelegramClient

// checkRedisClient 检查Redis客户端是否可用
func (t *TelegramClient) checkRedisClient() bool {
	if redis.GlobalRedisClient == nil {
		t.SendMessage("错误: Redis客户端未初始化")
		return false
	}
	return true
}

// 获取中国时区
func getChinaLocation() *time.Location {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		logrus.Warnf("无法加载中国时区，使用UTC: %v", err)
		return time.UTC
	}
	return loc
}

// 格式化创建时间为完整的年月日时间格式
func formatCreationTime(t time.Time) string {
	chinaLoc := getChinaLocation()
	localTime := t.In(chinaLoc)
	return localTime.Format("2006-01-02 15:04:05")
}

// 安全发送消息，处理长消息分割
func (t *TelegramClient) sendMessageSafely(text string) error {
	if t == nil || t.bot == nil {
		return fmt.Errorf("Telegram客户端未初始化")
	}

	// 如果消息长度超过限制，进行分割
	if len(text) <= MaxMessageLength {
		return t.SendMessage(text)
	}

	// 分割长消息
	parts := splitLongMessage(text, MaxMessageLength)
	for i, part := range parts {
		if i > 0 {
			time.Sleep(100 * time.Millisecond) // 避免发送过快
		}
		if err := t.SendMessage(part); err != nil {
			return fmt.Errorf("发送消息第%d部分失败: %v", i+1, err)
		}
	}
	return nil
}

// 分割长消息
func splitLongMessage(text string, maxLen int) []string {
	if len(text) <= maxLen {
		return []string{text}
	}

	var parts []string
	lines := strings.Split(text, "\n")
	currentPart := ""

	for i := range lines {
		line := lines[i]
		if len(line) > maxLen {
			if currentPart != "" {
				parts = append(parts, currentPart)
				currentPart = ""
			}
			for len(line) > maxLen {
				parts = append(parts, line[:maxLen])
				line = line[maxLen:]
			}
			if line != "" {
				currentPart = line
			}
			continue
		}

		testPart := currentPart
		if testPart != "" {
			testPart += "\n"
		}
		testPart += line

		if len(testPart) > maxLen {
			if currentPart != "" {
				parts = append(parts, currentPart)
			}
			currentPart = line
		} else {
			currentPart = testPart
		}
	}

	if currentPart != "" {
		parts = append(parts, currentPart)
	}

	return parts
}

// InitTelegram 初始化Telegram客户端
func InitTelegram() error {
	if config.GlobalConfig.TelegramBotToken == "" {
		logrus.Warn("未配置Telegram Bot Token，跳过Telegram初始化")
		return nil
	}

	bot, err := tgbotapi.NewBotAPI(config.GlobalConfig.TelegramBotToken)
	if err != nil {
		return fmt.Errorf("创建Telegram Bot失败: %v", err)
	}

	bot.Debug = false

	chatID, err := strconv.ParseInt(config.GlobalConfig.TelegramChatID, 10, 64)
	if err != nil {
		return fmt.Errorf("telegram chat ID格式错误: %v", err)
	}

	GlobalTelegramClient = &TelegramClient{
		bot:    bot,
		chatID: chatID,
	}

	GlobalTelegramClient.setupCustomKeyboard()

	go GlobalTelegramClient.startCommandListener()

	logrus.Info("Telegram客户端初始化成功")
	return nil
}

// SendMessage 发送普通消息
func (t *TelegramClient) SendMessage(text string) error {
	if t == nil || t.bot == nil {
		return fmt.Errorf("telegram客户端未初始化")
	}

	if len(text) > MaxMessageLength {
		return t.sendMessageSafely(text)
	}

	msg := tgbotapi.NewMessage(t.chatID, text)
	msg.ParseMode = "Markdown"

	_, err := t.bot.Send(msg)
	if err != nil {
		return fmt.Errorf("发送消息失败: %v", err)
	}

	return nil
}

// SendError 发送错误通知
func (t *TelegramClient) SendError(operation string, err error) error {
	message := fmt.Sprintf("%s\n\n错误详情: %v", operation, err)

	return t.SendMessage(message)
}

// SendServiceStatus 发送服务状态通知
func (t *TelegramClient) SendServiceStatus(status, message string) error {
	statusMap := map[string]string{
		"starting": "启动中",
		"started":  "已启动",
		"stopping": "停止中",
		"stopped":  "已停止",
		"error":    "错误",
	}

	statusText, exists := statusMap[status]
	if !exists {
		statusText = "信息"
	}

	text := fmt.Sprintf(`%s

%s

时间: %s`, statusText, message, formatCreationTime(time.Now()))

	return t.SendMessage(text)
}

// startCommandListener 启动命令监听
func (t *TelegramClient) startCommandListener() {
	if t == nil || t.bot == nil {
		logrus.Error("Telegram客户端未初始化，无法启动命令监听")
		return
	}

	logrus.Info("启动Telegram命令监听...")

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := t.bot.GetUpdatesChan(u)

	for update := range updates {
		// 处理消息命令
		if update.Message != nil {
			// 检查消息是否来自指定的聊天ID
			if update.Message.Chat.ID != t.chatID {
				continue
			}

			if update.Message.IsCommand() {
				t.handleCommand(update.Message)
			}
		}

	}
}

// handleCommand 处理命令
func (t *TelegramClient) handleCommand(message *tgbotapi.Message) {
	command := message.Command()
	args := strings.Fields(message.CommandArguments())

	logrus.WithFields(logrus.Fields{
		"command": command,
		"args":    args,
		"user":    message.From.UserName,
	}).Info("收到Telegram命令")

	switch command {
	case "os": // 做空开仓
		t.handleTradingCommand(command, args, models.ActionTypeOpen, types.PositionSideShort)
	case "ol": // 做多开仓
		t.handleTradingCommand(command, args, models.ActionTypeOpen, types.PositionSideLong)
	case "as": // 做空加仓
		t.handleTradingCommand(command, args, models.ActionTypeAddition, types.PositionSideShort)
	case "al": // 做多加仓
		t.handleTradingCommand(command, args, models.ActionTypeAddition, types.PositionSideLong)
	case "ts": // 做空止盈
		t.handleTradingCommand(command, args, models.ActionTypeTakeProfit, types.PositionSideShort)
	case "tl": // 做多止盈
		t.handleTradingCommand(command, args, models.ActionTypeTakeProfit, types.PositionSideLong)
	case "ps": // 做空平仓（等同于做空止盈）
		t.handleTradingCommand(command, args, models.ActionTypeTakeProfit, types.PositionSideShort)
	case "pl": // 做多平仓（等同于做多止盈）
		t.handleTradingCommand(command, args, models.ActionTypeTakeProfit, types.PositionSideLong)
	case "estimates": // 价格监听查询
		t.handleEstimatesCommand()
	case "show": // 显示交易对信息
		t.handleShowCommand(args)
	case "start": // 启动命令，显示帮助信息
		t.handleStartCommand()
	default:
		t.handleUnknownCommand(command)
	}
}

// handleTradingCommand 处理交易命令
func (t *TelegramClient) handleTradingCommand(command string, args []string, actionType, side string) {
	logrus.WithFields(logrus.Fields{
		"command":     command,
		"args":        args,
		"action_type": actionType,
		"side":        side,
	}).Info("开始处理交易命令")

	if len(args) < 2 {
		t.SendMessage(fmt.Sprintf("参数错误\n用法: /%s <symbol> <usdt数量> [price]\n例如: /%s BTCUSDT 100 50000", command, command))
		return
	}

	symbol := strings.ToUpper(args[0])
	if !strings.HasSuffix(symbol, "USDT") {
		symbol += "USDT"
	}

	// 解析USDT数量
	usdtAmount, err := strconv.ParseFloat(args[1], 64)
	if err != nil || usdtAmount <= 0 {
		t.SendMessage("错误: USDT数量格式错误，请输入有效数字")
		return
	}

	// 解析价格（可选）
	var price float64
	if len(args) >= 3 {
		price, err = strconv.ParseFloat(args[2], 64)
		if err != nil || price <= 0 {
			t.SendMessage("错误: 价格格式错误，请输入有效数字")
			return
		}
	} else {
		// 获取当前价格
		if !t.checkRedisClient() {
			return
		}

		// 直接使用symbol作为MarketID获取标记价格
		markPriceData, err := redis.GlobalRedisClient.GetMarkPrice(symbol)
		if err != nil {
			t.SendMessage(fmt.Sprintf("错误: 获取 %s 当前价格失败: %v", symbol, err))
			return
		}
		price = markPriceData.MarkPrice
	}

	// 创建价格预估并执行
	t.executeTradingOrder(symbol, actionType, side, usdtAmount, price)
}

// checkListeningEstimateExists 检查指定交易对、方向和操作类型的监听中估价是否存在
func (t *TelegramClient) checkListeningEstimateExists(symbol, side, actionType string) (*models.PriceEstimate, bool) {
	if !t.checkRedisClient() {
		return nil, false
	}

	estimate, err := redis.GlobalRedisClient.GetListeningEstimateBySymbolSideAction(symbol, side, actionType)
	if err != nil {
		logrus.WithFields(logrus.Fields{
			"symbol":      symbol,
			"side":        side,
			"action_type": actionType,
			"error":       err,
		}).Error("检查监听中估价时发生错误")
		return nil, false
	}

	if estimate == nil {
		return nil, false
	}

	return estimate, true
}

// executeTradingOrder 创建交易价格监听
func (t *TelegramClient) executeTradingOrder(symbol, actionType, side string, usdtAmount, price float64) {
	logrus.WithFields(logrus.Fields{
		"symbol":      symbol,
		"action_type": actionType,
		"side":        side,
		"usdt_amount": usdtAmount,
		"price":       price,
	}).Info("开始创建交易价格监听")

	if !t.checkRedisClient() {
		logrus.Error("Redis客户端未初始化")
		return
	}

	// 检查币种是否被选中，如果没有选中则报错
	if !redis.GlobalRedisClient.IsCoinSelected(symbol) {
		t.SendMessage(fmt.Sprintf("币种 %s 未选中\n", symbol))
		return
	}

	_, hasListeningEstimate := t.checkListeningEstimateExists(symbol, side, actionType)
	if hasListeningEstimate {
		t.SendMessage(fmt.Sprintf("%s %s %s 已存在监听",
			symbol, t.getActionText(actionType), t.getPositionText(side)))
		return
	}

	// 默认杠杆3倍
	leverage := 3

	// 对于Telegram机器人，使用默认100%比例
	percentage := 100.0

	// 创建价格预估
	estimate := &models.PriceEstimate{
		ID:          fmt.Sprintf("tg_%d", time.Now().UnixNano()),
		Symbol:      symbol,
		Side:        side,
		ActionType:  actionType,
		TargetPrice: price,
		Percentage:  percentage, // 使用百分比而不是具体数量
		Leverage:    leverage,
		OrderType:   types.OrderTypeLimit,
		MarginMode:  types.MarginModeCrossed,
		Status:      models.EstimateStatusListening,
		Enabled:     true,
		TriggerType: models.TriggerTypeCondition, // 使用条件触发，等待价格监听
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// 保存价格预估到Redis
	err := redis.GlobalRedisClient.SetPriceEstimate(estimate)
	if err != nil {
		t.SendMessage(fmt.Sprintf("错误: 创建价格监听失败: %v", err))
		return
	}

	// 发送确认消息
	actionText := t.getActionText(actionType)
	positionText := t.getPositionText(side)

	// 获取当前价格用于对比显示
	currentPrice := 0.0
	if markPriceData, err := redis.GlobalRedisClient.GetMarkPrice(symbol); err == nil {
		currentPrice = markPriceData.MarkPrice
	}

	combinedStatusText := t.getCombinedStatusText(estimate.Status, estimate.Enabled)

	var confirmMessage string
	if currentPrice > 0 {
		// 计算价格差距
		priceDiff := price - currentPrice
		priceDiffPercent := (priceDiff / currentPrice) * 100
		diffSymbol := ""
		if priceDiff > 0 {
			diffSymbol = "+"
		}

		confirmMessage = fmt.Sprintf(`价格监听已创建

%s %s %s
比例: %.1f%%
当前价格: %.4f
目标价格: %.4f
价格差距: %s%.4f (%.2f%%)
杠杆: %dx
状态: %s`,
			actionText, symbol, positionText,
			percentage, currentPrice, price, diffSymbol, priceDiff, priceDiffPercent,
			leverage, combinedStatusText)
	} else {
		confirmMessage = fmt.Sprintf(`价格监听已创建

%s %s %s
比例: %.1f%%
目标价格: %.4f
杠杆: %dx
状态: %s`,
			actionText, symbol, positionText,
			percentage, price, leverage,
			combinedStatusText)
	}

	t.SendMessage(confirmMessage)
}

// handleEstimatesCommand 处理价格监听查询命令
func (t *TelegramClient) handleEstimatesCommand() {
	if !t.checkRedisClient() {
		return
	}

	estimates, err := redis.GlobalRedisClient.GetAllEstimates()
	if err != nil {
		t.SendMessage(fmt.Sprintf("错误: 获取价格监听失败: %v", err))
		return
	}

	// 显示所有价格监听
	allEstimates := estimates

	if len(allEstimates) == 0 {
		t.SendMessage("当前无价格监听")
		return
	}

	// 按创建时间排序，最新的在前
	sort.Slice(allEstimates, func(i, j int) bool {
		return allEstimates[i].CreatedAt.After(allEstimates[j].CreatedAt)
	})

	// 限制显示数量，最多显示最近的5个
	displayCount := len(allEstimates)
	if displayCount > 5 {
		displayCount = 5
	}

	message := fmt.Sprintf("*价格监听* (%d/%d)\n", displayCount, len(allEstimates))

	for i := 0; i < displayCount; i++ {
		estimate := allEstimates[i]
		actionText := t.getActionText(estimate.ActionType)
		positionText := t.getPositionText(estimate.Side)

		message += fmt.Sprintf("*%s* %s %s\n", estimate.Symbol, actionText, positionText)
		message += fmt.Sprintf("比例　　%.1f%%\n", estimate.Percentage)

		// 获取当前价格
		currentPrice := 0.0
		if markPriceData, err := redis.GlobalRedisClient.GetMarkPrice(estimate.Symbol); err == nil {
			currentPrice = markPriceData.MarkPrice
		}

		message += fmt.Sprintf("当前价　%.4f\n", currentPrice)
		message += fmt.Sprintf("目标价　%.4f\n", estimate.TargetPrice)

		// 计算价格差距和百分比
		if currentPrice > 0 {
			priceDiff := estimate.TargetPrice - currentPrice
			priceDiffPercent := (priceDiff / currentPrice) * 100
			diffSymbol := ""
			if priceDiff > 0 {
				diffSymbol = "+"
			}
			message += fmt.Sprintf("差距　　%s%.4f (%.2f%%)\n", diffSymbol, priceDiff, priceDiffPercent)
		}

		message += fmt.Sprintf("杠杆　　%dx\n", estimate.Leverage)

		combinedStatusText := t.getCombinedStatusText(estimate.Status, estimate.Enabled)
		message += fmt.Sprintf("状态　　%s\n", combinedStatusText)
		message += fmt.Sprintf("创建　　%s\n", formatCreationTime(estimate.CreatedAt))

		if i < displayCount-1 {
			message += "\n\n"
		}
	}

	// 直接发送消息，不使用按钮和消息编辑
	err = t.SendMessage(message)
	if err != nil {
		t.SendMessage(fmt.Sprintf("发送价格监听信息失败: %v", err))
	}
}

// handleShowCommand 处理显示交易对信息命令
func (t *TelegramClient) handleShowCommand(args []string) {
	if !t.checkRedisClient() {
		return
	}

	if len(args) == 0 {
		t.SendMessage("请输入交易对\n用法: /show <symbol>\n例如: /show BTCUSDT")
		return
	}

	symbol := strings.ToUpper(args[0])
	if !strings.HasSuffix(symbol, "USDT") {
		symbol += "USDT"
	}

	// 获取当前价格
	markPriceData, err := redis.GlobalRedisClient.GetMarkPrice(symbol)
	if err != nil {
		t.SendMessage(fmt.Sprintf("获取 %s 价格失败: %v", symbol, err))
		return
	}

	// 检查币种是否被选中
	isSelected := redis.GlobalRedisClient.IsCoinSelected(symbol)
	selectionStatus := "❌ 未选中"
	if isSelected {
		selectionStatus = "✅ 已选中"
	}

	// 获取该交易对的价格监听
	estimates, err := redis.GlobalRedisClient.GetAllEstimates()
	if err != nil {
		t.SendMessage(fmt.Sprintf("获取价格监听失败: %v", err))
		return
	}

	// 过滤出该交易对的监听
	var symbolEstimates []*models.PriceEstimate
	for i := range estimates {
		if estimates[i].Symbol == symbol {
			symbolEstimates = append(symbolEstimates, estimates[i])
		}
	}

	message := fmt.Sprintf("*%s 交易对信息*\n\n", symbol)
	message += fmt.Sprintf("当前价格: %.4f\n", markPriceData.MarkPrice)
	message += fmt.Sprintf("币种状态: %s\n", selectionStatus)
	message += fmt.Sprintf("价格监听: %d个\n", len(symbolEstimates))

	if len(symbolEstimates) > 0 {
		message += "\n*监听详情*:\n"
		for i, estimate := range symbolEstimates {
			actionText := t.getActionText(estimate.ActionType)
			positionText := t.getPositionText(estimate.Side)
			statusText := t.getCombinedStatusText(estimate.Status, estimate.Enabled)

			message += fmt.Sprintf("%d. %s %s\n", i+1, actionText, positionText)
			message += fmt.Sprintf("   目标价: %.4f | 状态: %s\n", estimate.TargetPrice, statusText)
		}
	}

	t.SendMessage(message)
}

// handleStartCommand 处理启动命令
func (t *TelegramClient) handleStartCommand() {
	message := `交易助手机器人

交易命令:
• /os <symbol> <usdt数量> [price] - 做空开仓
• /ol <symbol> <usdt数量> [price] - 做多开仓
• /as <symbol> <usdt数量> [price] - 做空加仓
• /al <symbol> <usdt数量> [price] - 做多加仓
• /ts <symbol> <usdt数量> [price] - 做空止盈
• /tl <symbol> <usdt数量> [price] - 做多止盈
• /ps <symbol> <usdt数量> [price] - 做空平仓
• /pl <symbol> <usdt数量> [price] - 做多平仓

💡 注意：/ts、/tl（止盈）和 /ps、/pl（平仓）效果相同

查询命令:
• /estimates - 查看价格监听
• /show <symbol> - 显示交易对信息

使用说明:
• symbol: 交易对 (如 BTC、BTCUSDT)
• usdt数量: 使用的USDT金额
• price: 限价 (可选，不填则使用当前价格)
• 默认杠杆: 3倍
• 默认订单类型: 限价单

示例:
• /ol BTC 100 50000 - 做多开仓BTC，使用100 USDT，价格50000
• /os ETH 50 - 做空开仓ETH，使用50 USDT，当前价格
• /show BTC - 显示BTCUSDT的详细信息`

	// 直接发送消息，不使用按钮
	err := t.SendMessage(message)
	if err != nil {
		t.SendMessage(fmt.Sprintf("发送帮助信息失败: %v", err))
	}
}

// handleUnknownCommand 处理未知命令
func (t *TelegramClient) handleUnknownCommand(command string) {
	t.SendMessage(fmt.Sprintf("未知命令: /%s\n\n发送 /start 查看可用命令", command))
}

// getActionText 获取操作类型的中文描述
func (t *TelegramClient) getActionText(actionType string) string {
	switch actionType {
	case models.ActionTypeOpen:
		return "🔵  开仓"
	case models.ActionTypeAddition:
		return "🔷  加仓"
	case models.ActionTypeTakeProfit:
		return "✅  止盈"
	default:
		return "⚫  交易"
	}
}

// getPositionText 获取仓位方向的中文描述
func (t *TelegramClient) getPositionText(side string) string {
	switch side {
	case types.PositionSideLong:
		return "🟢  做多"
	case types.PositionSideShort:
		return "🔴  做空"
	default:
		return "🟡  未知"
	}
}

// getCombinedStatusText 获取合并状态和启用的中文描述
func (t *TelegramClient) getCombinedStatusText(status string, enabled bool) string {
	if !enabled {
		// 如果未启用，显示禁用状态
		return "🔴  已禁用"
	}

	// 如果启用，根据状态显示
	switch status {
	case models.EstimateStatusListening:
		return "👁️  监听中"
	case models.EstimateStatusTriggered:
		return "✅  已触发"
	case models.EstimateStatusFailed:
		return "❌  触发失败"
	default:
		return "❓  未知状态"
	}
}

// getMarginModeText 获取保证金模式的中文描述
func (t *TelegramClient) getMarginModeText(marginMode string) string {
	switch marginMode {
	case types.MarginModeCross, types.MarginModeCrossed:
		return "全仓"
	case types.MarginModeIsolated:
		return "逐仓"
	default:
		return marginMode // 如果未知，返回原值
	}
}

// getPositionSideText 获取仓位方向的中文描述
func (t *TelegramClient) getPositionSideText(side string) string {
	switch strings.ToUpper(side) {
	case "LONG":
		return "🟢  多头"
	case "SHORT":
		return "🔴  空头"
	case "BOTH":
		return "🟡  双向"
	default:
		return "🟡  " + side // 如果未知，返回原值
	}
}

// setupCustomKeyboard 设置自定义键盘
func (t *TelegramClient) setupCustomKeyboard() {
	if t == nil || t.bot == nil {
		return
	}

	// 创建空的自定义键盘
	keyboard := tgbotapi.NewReplyKeyboard()
	keyboard.ResizeKeyboard = true               // 自动调整键盘大小
	keyboard.OneTimeKeyboard = false             // 键盘持久显示
	keyboard.InputFieldPlaceholder = "输入交易命令..." // 输入框提示

	// 发送带键盘的消息
	msg := tgbotapi.NewMessage(t.chatID, "交易助手已就绪，请输入交易命令")
	msg.ReplyMarkup = keyboard

	_, err := t.bot.Send(msg)
	if err != nil {
		logrus.Errorf("设置自定义键盘失败: %v", err)
	} else {
		logrus.Info("自定义键盘设置成功")
	}
}
