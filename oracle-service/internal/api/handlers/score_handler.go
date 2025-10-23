package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/yourusername/p2p-lend/oracle-service/internal/service"
	"github.com/yourusername/p2p-lend/oracle-service/pkg/logger"
	"go.uber.org/zap"
)

// ScoreHandler handles credit score API requests
type ScoreHandler struct {
	service *service.OracleService
}

// NewScoreHandler creates a new score handler
func NewScoreHandler(service *service.OracleService) *ScoreHandler {
	return &ScoreHandler{
		service: service,
	}
}

// GetCreditScoreRequest represents the request to get a credit score
type GetCreditScoreRequest struct {
	Address string `uri:"address" binding:"required"`
}

// UpdateCreditScoreRequest represents the request to update a credit score
type UpdateCreditScoreRequest struct {
	Address string `json:"address" binding:"required"`
	UserID  string `json:"user_id"`
	Publish bool   `json:"publish"`
}

// GetCreditScoreResponse represents the credit score response
type GetCreditScoreResponse struct {
	Address       string `json:"address"`
	Score         uint16 `json:"score"`
	Confidence    uint8  `json:"confidence"`
	OnChainScore  uint16 `json:"on_chain_score"`
	OffChainScore uint16 `json:"off_chain_score"`
	HybridScore   uint16 `json:"hybrid_score"`
	DataHash      string `json:"data_hash"`
	LastUpdated   string `json:"last_updated"`
	NextUpdateDue string `json:"next_update_due"`
	UpdateCount   uint32 `json:"update_count"`
}

// GetCreditScore retrieves a credit score for an address
// @Summary Get credit score
// @Description Get the current credit score for a blockchain address
// @Tags credit-score
// @Accept json
// @Produce json
// @Param address path string true "Blockchain address"
// @Success 200 {object} GetCreditScoreResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/credit-score/{address} [get]
func (h *ScoreHandler) GetCreditScore(c *gin.Context) {
	var req GetCreditScoreRequest
	if err := c.ShouldBindUri(&req); err != nil {
		logger.Error("Invalid request", zap.Error(err))
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request",
			Message: err.Error(),
		})
		return
	}

	score, err := h.service.GetScore(c.Request.Context(), req.Address)
	if err != nil {
		logger.Error("Failed to get credit score", zap.Error(err))
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to retrieve credit score",
			Message: err.Error(),
		})
		return
	}

	if score == nil {
		c.JSON(http.StatusNotFound, ErrorResponse{
			Error:   "Credit score not found",
			Message: "No credit score exists for this address",
		})
		return
	}

	response := GetCreditScoreResponse{
		Address:       score.UserAddress,
		Score:         score.Score,
		Confidence:    score.Confidence,
		OnChainScore:  score.OnChainScore,
		OffChainScore: score.OffChainScore,
		HybridScore:   score.HybridScore,
		DataHash:      score.DataHash,
		LastUpdated:   score.LastUpdated.Format("2006-01-02T15:04:05Z"),
		NextUpdateDue: score.NextUpdateDue.Format("2006-01-02T15:04:05Z"),
		UpdateCount:   score.UpdateCount,
	}

	c.JSON(http.StatusOK, response)
}

// UpdateCreditScore calculates and updates a credit score
// @Summary Update credit score
// @Description Calculate and update credit score for an address
// @Tags credit-score
// @Accept json
// @Produce json
// @Param request body UpdateCreditScoreRequest true "Update request"
// @Success 200 {object} GetCreditScoreResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/credit-score/update [post]
func (h *ScoreHandler) UpdateCreditScore(c *gin.Context) {
	var req UpdateCreditScoreRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error("Invalid request", zap.Error(err))
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request",
			Message: err.Error(),
		})
		return
	}

	// Calculate and update score
	score, err := h.service.CalculateAndUpdateScore(c.Request.Context(), req.Address, req.UserID)
	if err != nil {
		logger.Error("Failed to update credit score", zap.Error(err))
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to update credit score",
			Message: err.Error(),
		})
		return
	}

	// Publish to blockchain if requested
	if req.Publish {
		if err := h.service.PublishScoreToBlockchain(c.Request.Context(), req.Address); err != nil {
			logger.Error("Failed to publish to blockchain", zap.Error(err))
			// Don't fail the request, just log the error
		}
	}

	response := GetCreditScoreResponse{
		Address:       score.UserAddress,
		Score:         score.Score,
		Confidence:    score.Confidence,
		OnChainScore:  score.OnChainScore,
		OffChainScore: score.OffChainScore,
		HybridScore:   score.HybridScore,
		DataHash:      score.DataHash,
		LastUpdated:   score.LastUpdated.Format("2006-01-02T15:04:05Z"),
		NextUpdateDue: score.NextUpdateDue.Format("2006-01-02T15:04:05Z"),
		UpdateCount:   score.UpdateCount,
	}

	c.JSON(http.StatusOK, response)
}

// GetScoreHistory retrieves credit score history
// @Summary Get credit score history
// @Description Get historical credit scores for an address
// @Tags credit-score
// @Accept json
// @Produce json
// @Param address path string true "Blockchain address"
// @Param limit query int false "Number of records to return" default(10)
// @Success 200 {array} ScoreHistoryResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/credit-score/{address}/history [get]
func (h *ScoreHandler) GetScoreHistory(c *gin.Context) {
	address := c.Param("address")
	limitStr := c.DefaultQuery("limit", "10")

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 100 {
		limit = 10
	}

	history, err := h.service.GetScoreHistory(c.Request.Context(), address, limit)
	if err != nil {
		logger.Error("Failed to get score history", zap.Error(err))
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to retrieve score history",
			Message: err.Error(),
		})
		return
	}

	response := make([]ScoreHistoryResponse, len(history))
	for i, h := range history {
		response[i] = ScoreHistoryResponse{
			Score:      h.Score,
			Confidence: h.Confidence,
			DataHash:   h.DataHash,
			Timestamp:  h.Timestamp.Format("2006-01-02T15:04:05Z"),
		}
	}

	c.JSON(http.StatusOK, response)
}

// GetStats retrieves oracle service statistics
// @Summary Get service statistics
// @Description Get statistics about the oracle service
// @Tags admin
// @Accept json
// @Produce json
// @Success 200 {object} StatsResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/admin/stats [get]
func (h *ScoreHandler) GetStats(c *gin.Context) {
	stats, err := h.service.GetStats(c.Request.Context())
	if err != nil {
		logger.Error("Failed to get stats", zap.Error(err))
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to retrieve statistics",
			Message: err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, stats)
}

// HealthCheck performs health checks
// @Summary Health check
// @Description Check health of all oracle components
// @Tags health
// @Accept json
// @Produce json
// @Success 200 {object} HealthResponse
// @Router /health [get]
func (h *ScoreHandler) HealthCheck(c *gin.Context) {
	health := h.service.HealthCheck(c.Request.Context())

	allHealthy := true
	for _, v := range health {
		if !v {
			allHealthy = false
			break
		}
	}

	status := http.StatusOK
	if !allHealthy {
		status = http.StatusServiceUnavailable
	}

	c.JSON(status, HealthResponse{
		Status:     map[bool]string{true: "healthy", false: "unhealthy"}[allHealthy],
		Components: health,
	})
}

// Response types

type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

type ScoreHistoryResponse struct {
	Score      uint16 `json:"score"`
	Confidence uint8  `json:"confidence"`
	DataHash   string `json:"data_hash"`
	Timestamp  string `json:"timestamp"`
}

type StatsResponse struct {
	TotalActiveScores     int64   `json:"total_active_scores"`
	AverageScore          float64 `json:"average_score"`
	DueForUpdate          int64   `json:"due_for_update"`
	PendingOracleUpdates  int64   `json:"pending_oracle_updates"`
}

type HealthResponse struct {
	Status     string          `json:"status"`
	Components map[string]bool `json:"components"`
}
