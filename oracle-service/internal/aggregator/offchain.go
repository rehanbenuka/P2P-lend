package aggregator

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/yourusername/p2p-lend/oracle-service/internal/models"
	"github.com/yourusername/p2p-lend/oracle-service/pkg/logger"
	"go.uber.org/zap"
)

// OffChainAggregator fetches off-chain/external data
type OffChainAggregator struct {
	httpClient    *http.Client
	creditBureauURL string
	bankAPIURL      string
	apiKey          string
}

// NewOffChainAggregator creates a new off-chain data aggregator
func NewOffChainAggregator(creditBureauURL, bankAPIURL, apiKey string) *OffChainAggregator {
	return &OffChainAggregator{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		creditBureauURL: creditBureauURL,
		bankAPIURL:      bankAPIURL,
		apiKey:          apiKey,
	}
}

// CreditBureauResponse represents credit bureau API response
type CreditBureauResponse struct {
	CreditScore      uint16  `json:"credit_score"`
	DebtToIncome     float64 `json:"debt_to_income_ratio"`
	EmploymentStatus string  `json:"employment_status"`
	IncomeLevel      string  `json:"income_level"`
}

// BankDataResponse represents bank API response
type BankDataResponse struct {
	AccountHistory   uint8  `json:"account_history_score"`
	IncomeVerified   bool   `json:"income_verified"`
	AverageBalance   float64 `json:"average_balance"`
}

// FetchMetrics gathers off-chain metrics for a user
func (a *OffChainAggregator) FetchMetrics(ctx context.Context, userID, address string) (*models.OffChainMetrics, error) {
	metrics := &models.OffChainMetrics{
		UserAddress: address,
	}

	// Fetch credit bureau data
	creditData, err := a.fetchCreditBureauData(ctx, userID)
	if err != nil {
		logger.Error("Failed to fetch credit bureau data", zap.Error(err))
		// Don't fail completely, just log and continue with partial data
	} else {
		metrics.TraditionalCreditScore = creditData.CreditScore
		metrics.DebtToIncomeRatio = creditData.DebtToIncome
		metrics.EmploymentStatus = creditData.EmploymentStatus
		metrics.IncomeLevel = creditData.IncomeLevel
	}

	// Fetch bank data
	bankData, err := a.fetchBankData(ctx, userID)
	if err != nil {
		logger.Error("Failed to fetch bank data", zap.Error(err))
	} else {
		metrics.BankAccountHistory = bankData.AccountHistory
		metrics.IncomeVerified = bankData.IncomeVerified
	}

	metrics.DataSource = "credit_bureau,bank_api"
	metrics.LastVerified = time.Now()
	metrics.UpdatedAt = time.Now()

	return metrics, nil
}

// fetchCreditBureauData queries credit bureau API
func (a *OffChainAggregator) fetchCreditBureauData(ctx context.Context, userID string) (*CreditBureauResponse, error) {
	// In production, this would call actual credit bureau APIs (Experian, Equifax, etc.)
	// For now, we'll simulate the response

	url := fmt.Sprintf("%s/credit-score?user_id=%s", a.creditBureauURL, userID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+a.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("credit bureau API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("credit bureau API returned status %d: %s", resp.StatusCode, string(body))
	}

	var creditData CreditBureauResponse
	if err := json.NewDecoder(resp.Body).Decode(&creditData); err != nil {
		return nil, fmt.Errorf("failed to decode credit bureau response: %w", err)
	}

	return &creditData, nil
}

// fetchBankData queries bank API
func (a *OffChainAggregator) fetchBankData(ctx context.Context, userID string) (*BankDataResponse, error) {
	// In production, this would use Plaid, Yodlee, or similar banking APIs
	// For now, we'll simulate the response

	url := fmt.Sprintf("%s/account-info?user_id=%s", a.bankAPIURL, userID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+a.apiKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("bank API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("bank API returned status %d: %s", resp.StatusCode, string(body))
	}

	var bankData BankDataResponse
	if err := json.NewDecoder(resp.Body).Decode(&bankData); err != nil {
		return nil, fmt.Errorf("failed to decode bank response: %w", err)
	}

	return &bankData, nil
}

// FetchMockMetrics generates mock off-chain metrics for testing
func (a *OffChainAggregator) FetchMockMetrics(address string) *models.OffChainMetrics {
	return &models.OffChainMetrics{
		UserAddress:            address,
		TraditionalCreditScore: 720,
		BankAccountHistory:     85,
		IncomeVerified:         true,
		IncomeLevel:            "medium",
		EmploymentStatus:       "full-time",
		DebtToIncomeRatio:      0.28,
		DataSource:             "mock",
		LastVerified:           time.Now(),
		CreatedAt:              time.Now(),
		UpdatedAt:              time.Now(),
	}
}

// HealthCheck verifies connectivity to external APIs
func (a *OffChainAggregator) HealthCheck(ctx context.Context) error {
	// Check credit bureau API
	if a.creditBureauURL != "" {
		req, err := http.NewRequestWithContext(ctx, "GET", a.creditBureauURL+"/health", nil)
		if err != nil {
			return fmt.Errorf("failed to create health check request: %w", err)
		}

		resp, err := a.httpClient.Do(req)
		if err != nil {
			return fmt.Errorf("credit bureau health check failed: %w", err)
		}
		resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("credit bureau unhealthy: status %d", resp.StatusCode)
		}
	}

	return nil
}
