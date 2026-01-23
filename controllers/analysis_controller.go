package controllers

import (
	"math"
	"net/http"
	"strconv"

	"trading_assistant/pkg/database"
	"trading_assistant/pkg/models"

	"github.com/gin-gonic/gin"
)

type AnalysisController struct{}

func NewAnalysisController() *AnalysisController {
	return &AnalysisController{}
}

// GetAnalysisResults retrieves analysis results with optional filtering and pagination
func (ac *AnalysisController) GetAnalysisResults(c *gin.Context) {
	var results []models.AnalysisResult
	
	// Query parameters
	symbol := c.Query("symbol")
	exchange := c.Query("exchange")
	marketType := c.Query("market_type")
	pageStr := c.DefaultQuery("page", "1")
	pageSizeStr := c.DefaultQuery("pageSize", "10")

	// Parse pagination
	page, _ := strconv.Atoi(pageStr)
	pageSize, _ := strconv.Atoi(pageSizeStr)
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}

	offset := (page - 1) * pageSize

	// Build query
	query := database.GetDB().Model(&models.AnalysisResult{})
	
	if symbol != "" {
		// 支持模糊查询：匹配包含symbol的记录
		query = query.Where("symbol LIKE ?", "%"+symbol+"%")
	}
	if exchange != "" {
		query = query.Where("exchange = ?", exchange)
	}
	if marketType != "" {
		query = query.Where("market_type = ?", marketType)
	}

	// Get total count
	var total int64
	query.Count(&total)

	// Get data
	result := query.Order("updated_at desc").Offset(offset).Limit(pageSize).Find(&results)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to fetch analysis results"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":       results,
		"total":      total,
		"page":       page,
		"pageSize":   pageSize,
		"totalPages": int(math.Ceil(float64(total) / float64(pageSize))),
	})
}

// GetAnalysisByID retrieves a single analysis result by ID
func (ac *AnalysisController) GetAnalysisByID(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "ID is required"})
		return
	}

	var result models.AnalysisResult
	if err := database.GetDB().First(&result, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Analysis not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": result,
	})
}
