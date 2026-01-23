package models

import (
	"encoding/json"
	"time"
)

// AnalysisResult 对应 MySQL 中的 analysis_results 表
type AnalysisResult struct {
	ID               uint            `json:"id" gorm:"primarykey"`
	Exchange         string          `json:"exchange"`
	Symbol           string          `json:"symbol"`
	MarketType       string          `json:"market_type"`
	Timeframe        string          `json:"timeframe"`
	InputLimit       int             `json:"input_limit"`
	SupportLevels    json.RawMessage `json:"support_levels" gorm:"type:json"`
	ResistanceLevels json.RawMessage `json:"resistance_levels" gorm:"type:json"`
	LastPrice        float64         `json:"last_price"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
}
