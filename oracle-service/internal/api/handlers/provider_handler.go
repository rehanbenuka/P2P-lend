package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/yourusername/p2p-lend/oracle-service/internal/service"
	"github.com/yourusername/p2p-lend/oracle-service/pkg/logger"
	"go.uber.org/zap"
)

// ProviderHandler handles requests related to 3rd party data providers
type ProviderHandler struct {
	service *service.EnhancedOracleService
}

// NewProviderHandler creates a new provider handler
func NewProviderHandler(service *service.EnhancedOracleService) *ProviderHandler {
	return &ProviderHandler{
		service: service,
	}
}

// UpdateWithProvidersRequest represents request to update score using 3rd party providers
type UpdateWithProvidersRequest struct {
	Address           string `json:"address" binding:"required"`
	BureauUserID      string `json:"bureau_user_id"`     // Credit Bureau user ID (SSN or similar)
	PlaidUserID       string `json:"plaid_user_id"`      // Plaid user identifier
	PlaidAccessToken  string `json:"plaid_access_token"` // Plaid access token
	Publish           bool   `json:"publish"`
	FetchCreditBureau bool   `json:"fetch_credit_bureau"` // Fetch from credit bureau
	FetchPlaid        bool   `json:"fetch_plaid"`         // Fetch from Plaid
	FetchBlockchain   bool   `json:"fetch_blockchain"`    // Fetch from blockchain providers
}

// ProviderDataResponse shows what data was fetched from each provider
type ProviderDataResponse struct {
	Address      string            `json:"address"`
	Score        uint16            `json:"score"`
	Confidence   uint8             `json:"confidence"`
	DataSources  []string          `json:"data_sources"`
	CreditBureau *CreditBureauData `json:"credit_bureau,omitempty"`
	Plaid        *PlaidData        `json:"plaid,omitempty"`
	Blockchain   *BlockchainData   `json:"blockchain,omitempty"`
	LastUpdated  string            `json:"last_updated"`
}

type CreditBureauData struct {
	CreditScore       int     `json:"credit_score"`
	DebtToIncomeRatio float64 `json:"debt_to_income_ratio"`
	PaymentHistory    string  `json:"payment_history"`
	Delinquencies     int     `json:"delinquencies"`
	Provider          string  `json:"provider"`
}

type PlaidData struct {
	TotalBalance   float64 `json:"total_balance"`
	AverageBalance float64 `json:"average_balance"`
	AccountAge     int     `json:"account_age_months"`
	IncomeVerified bool    `json:"income_verified"`
	AnnualIncome   float64 `json:"annual_income"`
	AccountsCount  int     `json:"accounts_count"`
}

type BlockchainData struct {
	WalletAge         int     `json:"wallet_age_days"`
	TotalTransactions int     `json:"total_transactions"`
	DeFiActivities    int     `json:"defi_activities"`
	PortfolioValue    float64 `json:"portfolio_value"`
	Liquidations      int     `json:"liquidations"`
}

// UpdateWithProviders calculates credit score using 3rd party data providers
// @Summary Update credit score with 3rd party providers
// @Description Fetch data from credit bureaus, Plaid, and blockchain providers to calculate credit score
// @Tags credit-score
// @Accept json
// @Produce json
// @Param request body UpdateWithProvidersRequest true "Update request with provider options"
// @Success 200 {object} ProviderDataResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /api/v1/credit-score/update-with-providers [post]
func (h *ProviderHandler) UpdateWithProviders(c *gin.Context) {
	var req UpdateWithProvidersRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		logger.Error("Invalid request", zap.Error(err))
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "Invalid request",
			Message: err.Error(),
		})
		return
	}

	logger.Info("Updating credit score with providers",
		zap.String("address", req.Address),
		zap.String("bureauUserID", req.BureauUserID),
		zap.String("plaidUserID", req.PlaidUserID),
		zap.Bool("creditBureau", req.FetchCreditBureau),
		zap.Bool("plaid", req.FetchPlaid),
		zap.Bool("blockchain", req.FetchBlockchain),
	)

	// Calculate score using selected providers
	score, providerData, err := h.service.CalculateWithProviders(
		c.Request.Context(),
		req.Address,
		req.BureauUserID,
		req.PlaidUserID,
		req.PlaidAccessToken,
		req.FetchCreditBureau,
		req.FetchPlaid,
		req.FetchBlockchain,
	)

	if err != nil {
		logger.Error("Failed to calculate score with providers", zap.Error(err))
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "Failed to calculate credit score",
			Message: err.Error(),
		})
		return
	}

	// Publish to blockchain if requested
	if req.Publish {
		if err := h.service.PublishScoreToBlockchain(c.Request.Context(), req.Address); err != nil {
			logger.Error("Failed to publish to blockchain", zap.Error(err))
			// Don't fail the request, just log
		}
	}

	// Build response
	response := ProviderDataResponse{
		Address:     score.UserAddress,
		Score:       score.Score,
		Confidence:  score.Confidence,
		DataSources: providerData.Sources,
		LastUpdated: score.LastUpdated.Format("2006-01-02T15:04:05Z"),
	}

	// Add provider-specific data
	if providerData.CreditBureauData != nil {
		response.CreditBureau = &CreditBureauData{
			CreditScore:       providerData.CreditBureauData.CreditScore,
			DebtToIncomeRatio: providerData.CreditBureauData.DebtToIncomeRatio,
			PaymentHistory:    providerData.CreditBureauData.PaymentHistory,
			Delinquencies:     providerData.CreditBureauData.Delinquencies,
			Provider:          providerData.CreditBureauData.DataSource,
		}
	}

	if providerData.PlaidData != nil {
		response.Plaid = &PlaidData{
			TotalBalance:   providerData.PlaidData.TotalBalance,
			AverageBalance: providerData.PlaidData.AverageBalance,
			AccountAge:     providerData.PlaidData.AccountAgeMonths,
			IncomeVerified: providerData.PlaidData.IncomeData != nil && providerData.PlaidData.IncomeData.IncomeVerified,
			AccountsCount:  len(providerData.PlaidData.Accounts),
		}
		if providerData.PlaidData.IncomeData != nil {
			response.Plaid.AnnualIncome = providerData.PlaidData.IncomeData.AnnualIncome
		}
	}

	if providerData.BlockchainData != nil {
		response.Blockchain = &BlockchainData{
			WalletAge:         providerData.BlockchainData.WalletAge,
			TotalTransactions: providerData.BlockchainData.TotalTransactions,
			DeFiActivities:    len(providerData.BlockchainData.DeFiActivities),
			PortfolioValue:    providerData.BlockchainData.TotalPortfolioValue,
			Liquidations:      len(providerData.BlockchainData.LiquidationEvents),
		}
	}

	c.JSON(http.StatusOK, response)
}

// GetProviderStatus returns the status of all 3rd party providers
// @Summary Get provider status
// @Description Check health status of all integrated 3rd party providers
// @Tags providers
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/providers/status [get]
func (h *ProviderHandler) GetProviderStatus(c *gin.Context) {
	status := h.service.GetProviderStatus(c.Request.Context())
	c.JSON(http.StatusOK, status)
}

// ListAvailableProviders returns list of available providers and their capabilities
// @Summary List available providers
// @Description Get list of all available 3rd party data providers
// @Tags providers
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/providers/list [get]
func (h *ProviderHandler) ListAvailableProviders(c *gin.Context) {
	providers := map[string]interface{}{
		"credit_bureaus": []map[string]interface{}{
			{
				"name":          "experian",
				"description":   "Experian Credit Bureau - Traditional credit scores",
				"data_provided": []string{"credit_score", "payment_history", "debt_to_income", "delinquencies"},
				"available":     true,
			},
			{
				"name":          "equifax",
				"description":   "Equifax Credit Bureau - Credit reports and scores",
				"data_provided": []string{"credit_score", "credit_history", "inquiries"},
				"available":     false,
			},
		},
		"banking": []map[string]interface{}{
			{
				"name":          "plaid",
				"description":   "Plaid - Banking and income verification",
				"data_provided": []string{"account_balances", "transactions", "income_verification"},
				"available":     true,
				"requires":      "access_token",
			},
		},
		"blockchain": []map[string]interface{}{
			{
				"name":          "covalent",
				"description":   "Covalent - Multi-chain blockchain data",
				"data_provided": []string{"token_balances", "transactions", "nft_holdings"},
				"available":     true,
			},
			{
				"name":          "moralis",
				"description":   "Moralis - Web3 data and analytics",
				"data_provided": []string{"defi_positions", "nft_data", "wallet_history"},
				"available":     true,
			},
			{
				"name":          "thegraph",
				"description":   "The Graph - DeFi protocol data",
				"data_provided": []string{"lending_positions", "swap_history", "liquidity_provision"},
				"available":     false,
			},
		},
	}

	c.JSON(http.StatusOK, providers)
}
