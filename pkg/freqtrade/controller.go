package freqtrade

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
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
	stopChanPair   chan struct{}
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

	if fc.stopChanPair != nil {
		close(fc.stopChanPair)
		fc.stopChanPair = nil
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

func (fc *Controller) pairRefresher() {
	if fc.stopChanPair != nil {
		close(fc.stopChanPair) // 防止重复启动
	}
	fc.stopChanPair = make(chan struct{})

	go func() {
		logrus.Info("交易对刷新器已启动")
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				go fc.setPairWhiteList()
			case <-fc.stopChanPair:
				logrus.Info("交易对刷新器已停止")
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

	// 启动交易对刷新器和token刷新器
	go fc.setPairWhiteList()
	go fc.pairRefresher()
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

// GetWhitelist 获取交易对白名单
func (fc *Controller) getWhitelist() ([]string, error) {
	url := fmt.Sprintf("%s/api/v1/whitelist", fc.BaseUrl)
	body, err := fc.doRequest("GET", url, nil, true)
	if err != nil {
		logrus.Errorf("获取whitelist失败: %v", err)
		return nil, err
	}

	var whitelistResp models.WhitelistResponse
	if err := json.Unmarshal(body, &whitelistResp); err != nil {
		logrus.Errorf("解析whitelist响应失败: %v", err)
		return nil, err
	}

	logrus.Infof("获取whitelist成功，共 %d 个交易对", whitelistResp.Length)
	return whitelistResp.Whitelist, nil
}

func (fc *Controller) setPairWhiteList() {
	whitelist, err := fc.getWhitelist()
	if err != nil {
		logrus.Errorf("获取交易对白名单失败: %v", err)
		return
	}
	err = fc.SetWatchedPairs(whitelist)
	if err != nil {
		logrus.Errorf("设置交易对白名单失败: %v", err)
		return
	}
	logrus.Info("交易对白名单已刷新")
}

// GetPositions 获取当前持仓数据，直接返回freqtrade格式
func (fc *Controller) GetPositions() ([]models.TradePosition, error) {
	// 获取freqtrade交易状态
	tradePositions, err := fc.GetTradeStatus()
	if err != nil {
		return nil, fmt.Errorf("获取freqtrade交易状态失败: %v", err)
	}

	// 只返回开仓的交易
	var openPositions []models.TradePosition
	for i := range tradePositions {
		trade := tradePositions[i]
		if trade.IsOpen {
			openPositions = append(openPositions, trade)
		}
	}

	logrus.Infof("从freqtrade获取到 %d 个持仓", len(openPositions))
	return openPositions, nil
}
