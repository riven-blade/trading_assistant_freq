package redis

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
	"trading_assistant/models"

	"github.com/sirupsen/logrus"
)

// SetCoinSelection 设置币种选择状态 (使用MarketID作为key)
func (c *Client) SetCoinSelection(marketID string, status string) error {
	selection := &models.CoinSelection{
		Symbol:    marketID, // 直接使用MarketID
		Status:    status,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	key := fmt.Sprintf("%s:%s", KeyCoinSelection, marketID)
	data, err := json.Marshal(selection)
	if err != nil {
		return fmt.Errorf("序列化币种选择状态失败: %v", err)
	}

	err = c.rdb.Set(c.ctx, key, data, 0).Err()
	if err != nil {
		return fmt.Errorf("保存币种选择状态失败: %v", err)
	}

	logrus.WithFields(logrus.Fields{
		"marketID": marketID,
		"status":   status,
	}).Info("币种选择状态已更新")

	return nil
}

// GetCoinSelection 获取币种选择状态 (通过MarketID)
func (c *Client) GetCoinSelection(marketID string) (*models.CoinSelection, error) {
	key := fmt.Sprintf("%s:%s", KeyCoinSelection, marketID)
	data, err := c.rdb.Get(c.ctx, key).Result()
	if err != nil {
		return nil, err
	}

	var selection models.CoinSelection
	err = json.Unmarshal([]byte(data), &selection)
	return &selection, err
}

// IsCoinSelected 检查币种是否选中 (通过MarketID)
func (c *Client) IsCoinSelected(marketID string) bool {
	selection, err := c.GetCoinSelection(marketID)
	if err != nil {
		return false
	}
	return selection.Status == models.CoinSelectionActive
}

// GetSelectedCoinMarketIDs 获取所有选中的币种MarketID
func (c *Client) GetSelectedCoinMarketIDs() ([]string, error) {
	keys, err := c.rdb.Keys(c.ctx, fmt.Sprintf("%s:*", KeyCoinSelection)).Result()
	if err != nil {
		return nil, err
	}

	var selectedMarketIDs []string
	for i := range keys {
		key := keys[i]
		data, err := c.rdb.Get(c.ctx, key).Result()
		if err != nil {
			continue
		}

		var selection models.CoinSelection
		if err := json.Unmarshal([]byte(data), &selection); err != nil {
			continue
		}

		if selection.Status == models.CoinSelectionActive {
			// 从key中提取MarketID (key格式为: "coin_selection:BTCUSDT")
			parts := strings.Split(key, ":")
			if len(parts) == 2 {
				selectedMarketIDs = append(selectedMarketIDs, parts[1])
			}
		}
	}

	return selectedMarketIDs, nil
}

// GetSelectedCoinsWithDetails 获取选中的币种及其详细信息
func (c *Client) GetSelectedCoinsWithDetails() ([]*models.Coin, error) {
	selectedMarketIDs, err := c.GetSelectedCoinMarketIDs()
	if err != nil {
		return nil, fmt.Errorf("获取选中币种MarketID失败: %v", err)
	}

	var selectedCoins []*models.Coin
	for i := range selectedMarketIDs {
		marketID := selectedMarketIDs[i]
		coin, err := c.GetCoin(marketID)
		if err != nil {
			logrus.Warnf("获取币种详情失败 %s: %v", marketID, err)
			continue
		}
		selectedCoins = append(selectedCoins, coin)
	}

	return selectedCoins, nil
}

// RemoveCoinSelection 移除币种选择状态
func (c *Client) RemoveCoinSelection(marketID string) error {
	key := fmt.Sprintf("%s:%s", KeyCoinSelection, marketID)
	err := c.rdb.Del(c.ctx, key).Err()
	if err != nil {
		return fmt.Errorf("删除币种选择状态失败: %v", err)
	}

	logrus.WithFields(logrus.Fields{
		"marketID": marketID,
	}).Info("币种选择状态已移除")

	return nil
}

// GetAllCoinSelections 获取所有币种选择状态
func (c *Client) GetAllCoinSelections() ([]*models.CoinSelection, error) {
	keys, err := c.rdb.Keys(c.ctx, fmt.Sprintf("%s:*", KeyCoinSelection)).Result()
	if err != nil {
		return nil, err
	}

	var selections []*models.CoinSelection
	for _, key := range keys {
		data, err := c.rdb.Get(c.ctx, key).Result()
		if err != nil {
			logrus.Errorf("获取币种选择状态失败 %s: %v", key, err)
			continue
		}

		var selection models.CoinSelection
		if err := json.Unmarshal([]byte(data), &selection); err != nil {
			logrus.Errorf("解析币种选择状态失败 %s: %v", key, err)
			continue
		}
		selections = append(selections, &selection)
	}

	return selections, nil
}
