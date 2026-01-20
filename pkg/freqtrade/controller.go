package freqtrade

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
	"trading_assistant/models"
	"trading_assistant/pkg/redis"

	"github.com/sirupsen/logrus"
)

type Controller struct {
	BaseUrl        string
	Username       string
	Password       string
	AccessToken    string
	RefreshToken   string
	stopChan       chan struct{}
	httpClient     *http.Client
	PositionStatus models.PositionStatus
	TradeStatus    []models.TradePosition
	redisClient    *redis.Client
	messageChan    chan string
}

func NewController(baseUrl, username, password string, redisClient *redis.Client) *Controller {
	return &Controller{
		BaseUrl:     baseUrl,
		Username:    username,
		Password:    password,
		redisClient: redisClient,
		httpClient:  &http.Client{Timeout: 10 * time.Second},
	}
}

// Stop 优雅停止所有定时器
func (fc *Controller) Stop() {
	logrus.Info("正在停止Freqtrade控制器...")

	if fc.stopChan != nil {
		close(fc.stopChan)
		fc.stopChan = nil
	}

	logrus.Info("Freqtrade控制器已停止")
}

func (fc *Controller) startTokenRefresher() {
	if fc.stopChan != nil {
		close(fc.stopChan) // 防止重复启动
	}
	fc.stopChan = make(chan struct{})

	go func() {
		logrus.Info("Token 刷新器已启动")
		ticker := time.NewTicker(10 * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				go fc.refreshToken()
			case <-fc.stopChan:
				logrus.Info("Token 刷新器已停止")
				return
			}
		}
	}()
}

func (fc *Controller) doRequest(method, url string, body io.Reader, useAccessToken bool) ([]byte, error) {
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, err
	}

	if useAccessToken {
		req.Header.Set("Authorization", "Bearer "+fc.AccessToken)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := fc.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s %s 请求失败: %s", method, url, string(respBody))
	}
	return respBody, nil
}

func (fc *Controller) Init(messageChan chan string) error {
	fc.messageChan = messageChan
	url := fmt.Sprintf("%v/api/v1/token/login", fc.BaseUrl)
	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		return fmt.Errorf("创建登录请求失败: %v", err)
	}
	req.SetBasicAuth(fc.Username, fc.Password)

	resp, err := fc.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("执行登录请求失败: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("登录失败: %d %s", resp.StatusCode, resp.Status)
	}

	body, _ := io.ReadAll(resp.Body)
	var loginResp models.LoginResponse
	if err := json.Unmarshal(body, &loginResp); err != nil {
		return fmt.Errorf("解析登录响应失败: %v", err)
	}

	fc.AccessToken = loginResp.AccessToken
	fc.RefreshToken = loginResp.RefreshToken

	logrus.Info("freq 首次登录成功")

	// 只启动token刷新器
	go fc.startTokenRefresher()

	return nil
}

func (fc *Controller) refreshToken() {
	url := fmt.Sprintf("%v/api/v1/token/refresh", fc.BaseUrl)

	req, err := http.NewRequest("POST", url, nil)
	if err != nil {
		logrus.Errorf("创建刷新请求失败: %v", err)
		return
	}
	req.Header.Set("Authorization", "Bearer "+fc.RefreshToken)

	resp, err := fc.httpClient.Do(req)
	if err != nil {
		logrus.Errorf("刷新 token 请求失败: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		logrus.Errorf("刷新 token 失败: %v", resp.Status)
		return
	}

	body, _ := io.ReadAll(resp.Body)
	var loginResp models.LoginResponse
	if err := json.Unmarshal(body, &loginResp); err != nil {
		logrus.Errorf("解析刷新响应失败: %v", err)
		return
	}

	fc.AccessToken = loginResp.AccessToken
	logrus.Info("刷新 token 成功")
}

func (fc *Controller) ForceBuy(payload models.ForceBuyPayload) error {
	url := fmt.Sprintf("%s/api/v1/forcebuy", fc.BaseUrl)

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	respBody, err := fc.doRequest("POST", url, bytes.NewReader(body), true)
	if err != nil {
		return err
	}

	logrus.Infof("forcebuy 成功: %s", string(respBody))
	return nil
}

func (fc *Controller) ForceAdjustBuy(pair string, price float64, side string, stakeAmount float64, entryTag string) error {
	url := fmt.Sprintf("%s/api/v1/forcebuy", fc.BaseUrl)
	payload := models.ForceAdjustBuyPayload{
		Pair:        pair,
		Price:       price,
		OrderType:   "limit",
		Side:        side,
		EntryTag:    entryTag,
		StakeAmount: stakeAmount,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	respBody, err := fc.doRequest("POST", url, bytes.NewReader(body), true)
	if err != nil {
		return err
	}

	logrus.Infof("forceadjustbuy 成功: %s", string(respBody))
	return nil
}

func (fc *Controller) ForceSell(tradeId string, orderType string, amount string) error {
	url := fmt.Sprintf("%s/api/v1/forcesell", fc.BaseUrl)
	payload := models.ForceSellPayload{
		TradeId:   tradeId,
		OrderType: orderType,
		Amount:    amount,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	respBody, err := fc.doRequest("POST", url, bytes.NewReader(body), true)
	if err != nil {
		return err
	}

	logrus.Infof("forcesell 成功: %s", string(respBody))
	return nil
}

func (fc *Controller) getCount() error {
	url := fmt.Sprintf("%v/api/v1/count", fc.BaseUrl)
	body, err := fc.doRequest("GET", url, nil, true)
	if err != nil {
		return err
	}

	var positions models.PositionStatus
	if err = json.Unmarshal(body, &positions); err != nil {
		return err
	}
	fc.PositionStatus = positions
	return nil
}

func (fc *Controller) getStatus() error {
	url := fmt.Sprintf("%s/api/v1/status", fc.BaseUrl)
	body, err := fc.doRequest("GET", url, nil, true)
	if err != nil {
		return err
	}

	var trades []models.TradePosition
	if err := json.Unmarshal(body, &trades); err != nil {
		return err
	}
	fc.TradeStatus = trades
	return nil
}

func (fc *Controller) fetchTradeData() error {
	err := fc.getStatus()
	if err != nil {
		return err
	}
	// 获取当前持仓数量
	err = fc.getCount()
	if err != nil {
		return err
	}
	return nil
}

// GetTradeStatus 获取当前交易状态
func (fc *Controller) GetTradeStatus() ([]models.TradePosition, error) {
	err := fc.getStatus()
	if err != nil {
		return nil, err
	}
	return fc.TradeStatus, nil
}

// 检查是否可以强制买入
func (fc *Controller) CheckForceBuy(pair string) bool {
	err := fc.fetchTradeData()
	if err != nil {
		logrus.Errorf("获取交易数据失败: %v", err)
		return false
	}

	tradeStatus := fc.TradeStatus
	for i := range tradeStatus {
		trade := tradeStatus[i]
		if trade.Pair == pair {
			return false
		}
	}

	return len(tradeStatus) < fc.PositionStatus.Max
}

// calculateGrindSummary 根据订单列表计算 grind 状态
func calculateGrindSummary(orders []models.FreqtradeOrder, isShort bool, totalAmount float64, leverage float64) *models.TradeGrindSummary {
	summary := &models.TradeGrindSummary{}

	// 确定入场方向：多头是 buy，空头是 sell
	entrySide := "buy"
	exitSide := "sell"
	if isShort {
		entrySide = "sell"
		exitSide = "buy"
	}

	// 追踪是否已找到对应的 exit
	grind1IsExitFound := false
	grind2IsExitFound := false
	grind3IsExitFound := false
	grindXIsExitFound := false

	// 倒序遍历订单
	for i := len(orders) - 1; i >= 0; i-- {
		order := orders[i]
		if order.Status != "closed" {
			continue
		}

		tag := ""
		if order.FtOrderTag != nil {
			tag = *order.FtOrderTag
		}

		// 处理入场订单（只计算 exit 之后的 entries）
		if order.FtOrderSide == entrySide {
			switch tag {
			case "grind_1_entry":
				if !grind1IsExitFound {
					summary.Grind1.HasEntry = true
					summary.Grind1.EntryCount++
					summary.Grind1.TotalAmount += order.Filled
					summary.Grind1.TotalCost += order.Filled * order.SafePrice
				}
			case "grind_2_entry":
				if !grind2IsExitFound {
					summary.Grind2.HasEntry = true
					summary.Grind2.EntryCount++
					summary.Grind2.TotalAmount += order.Filled
					summary.Grind2.TotalCost += order.Filled * order.SafePrice
				}
			case "grind_3_entry":
				if !grind3IsExitFound {
					summary.Grind3.HasEntry = true
					summary.Grind3.EntryCount++
					summary.Grind3.TotalAmount += order.Filled
					summary.Grind3.TotalCost += order.Filled * order.SafePrice
				}
			default:
				// 所有其他非 grind_1/2/3_entry 的订单都算 grind_x
				if !grindXIsExitFound {
					summary.GrindX.HasEntry = true
					summary.GrindX.EntryCount++
					summary.GrindX.TotalAmount += order.Filled
					summary.GrindX.TotalCost += order.Filled * order.SafePrice
				}
			}
		}

		// 处理退出订单
		if order.FtOrderSide == exitSide {
			orderTagParts := strings.Split(tag, " ")
			orderMode := orderTagParts[0]

			switch orderMode {
			case "grind_1_exit", "grind_1_derisk":
				if !grind1IsExitFound {
					grind1IsExitFound = true
					summary.Grind1.HasExit = true
				}
			case "grind_2_exit", "grind_2_derisk":
				if !grind2IsExitFound {
					grind2IsExitFound = true
					summary.Grind2.HasExit = true
				}
			case "grind_3_exit", "grind_3_derisk":
				if !grind3IsExitFound {
					grind3IsExitFound = true
					summary.Grind3.HasExit = true
				}
			case "grind_x_exit", "grind_x_derisk":
				if !grindXIsExitFound {
					grindXIsExitFound = true
					summary.GrindX.HasExit = true
				}
			}
		}
	}

	// 计算平均开仓价格
	if summary.Grind1.TotalAmount > 0 {
		summary.Grind1.OpenRate = summary.Grind1.TotalCost / summary.Grind1.TotalAmount
	}
	if summary.Grind2.TotalAmount > 0 {
		summary.Grind2.OpenRate = summary.Grind2.TotalCost / summary.Grind2.TotalAmount
	}
	if summary.Grind3.TotalAmount > 0 {
		summary.Grind3.OpenRate = summary.Grind3.TotalCost / summary.Grind3.TotalAmount
	}
	if summary.GrindX.TotalAmount > 0 {
		summary.GrindX.OpenRate = summary.GrindX.TotalCost / summary.GrindX.TotalAmount
	}

	// 计算占总仓位的比例
	// 计算占总仓位的比例和保证金金额
	if totalAmount > 0 {
		summary.Grind1.Percentage = summary.Grind1.TotalAmount / totalAmount * 100
		summary.Grind2.Percentage = summary.Grind2.TotalAmount / totalAmount * 100
		summary.Grind3.Percentage = summary.Grind3.TotalAmount / totalAmount * 100
		summary.GrindX.Percentage = summary.GrindX.TotalAmount / totalAmount * 100
	}

	// 计算保证金金额（TotalCost / Leverage）
	if leverage > 0 {
		summary.Grind1.StakeAmount = summary.Grind1.TotalCost / leverage
		summary.Grind2.StakeAmount = summary.Grind2.TotalCost / leverage
		summary.Grind3.StakeAmount = summary.Grind3.TotalCost / leverage
		summary.GrindX.StakeAmount = summary.GrindX.TotalCost / leverage
	} else {
		// 避免除以零，如果是现货或无杠杆，StakeAmount = TotalCost
		summary.Grind1.StakeAmount = summary.Grind1.TotalCost
		summary.Grind2.StakeAmount = summary.Grind2.TotalCost
		summary.Grind3.StakeAmount = summary.Grind3.TotalCost
		summary.GrindX.StakeAmount = summary.GrindX.TotalCost
	}

	return summary
}

// GetPositions 获取当前持仓数据，直接返回freqtrade格式
func (fc *Controller) GetPositions() ([]models.TradePosition, error) {
	// 获取freqtrade交易状态
	tradePositions, err := fc.GetTradeStatus()
	if err != nil {
		return nil, fmt.Errorf("获取freqtrade交易状态失败: %v", err)
	}

	// 只返回开仓的交易，并计算 grind 状态
	var openPositions []models.TradePosition
	for i := range tradePositions {
		trade := tradePositions[i]
		if trade.IsOpen {
			// 检查该币种是否已选中，如果未选中则自动选中
			// 确保有仓位的币种能够订阅到价格数据
			if fc.redisClient != nil && trade.Pair != "" {
				if !fc.redisClient.IsCoinSelected(trade.Pair) {
					if err := fc.redisClient.SetCoinSelection(trade.Pair, models.CoinSelectionActive); err != nil {
						logrus.Warnf("自动选中币种 %s 失败: %v", trade.Pair, err)
					} else {
						logrus.Infof("自动选中有仓位的币种: %s", trade.Pair)
					}
				}
			}

			// 获取杠杆倍数
			var leverage float64 = 1.0
			if trade.Leverage != nil {
				leverage = *trade.Leverage
			}

			// 计算 grind 状态汇总
			trade.GrindSummary = calculateGrindSummary(trade.Orders, trade.IsShort, trade.Amount, leverage)
			openPositions = append(openPositions, trade)
		}
	}

	logrus.Infof("从freqtrade获取到 %d 个持仓", len(openPositions))
	return openPositions, nil
}
