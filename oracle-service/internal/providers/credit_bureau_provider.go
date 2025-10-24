package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/yourusername/p2p-lend/oracle-service/pkg/logger"
	"go.uber.org/zap"
)

// CreditBureauProvider integrates with credit bureau APIs (Experian, Equifax, TransUnion)
type CreditBureauProvider struct {
	httpClient *http.Client
	apiKey     string
	baseURL    string
	provider   string // "experian", "equifax", "transunion"
}

// CreditBureauResponse represents the standardized response from credit bureaus
type CreditBureauResponse struct {
	UserID            string    `json:"user_id"`
	CreditScore       int       `json:"credit_score"`
	ScoreRange        string    `json:"score_range"` // "300-850"
	DebtToIncomeRatio float64   `json:"debt_to_income_ratio"`
	TotalDebt         float64   `json:"total_debt"`
	TotalIncome       float64   `json:"total_income"`
	PaymentHistory    string    `json:"payment_history"`    // "excellent", "good", "fair", "poor"
	CreditUtilization float64   `json:"credit_utilization"` // Percentage
	NumberOfAccounts  int       `json:"number_of_accounts"`
	OldestAccountAge  int       `json:"oldest_account_age"` // Months
	RecentInquiries   int       `json:"recent_inquiries"`   // Last 6 months
	Delinquencies     int       `json:"delinquencies"`
	PublicRecords     int       `json:"public_records"` // Bankruptcies, liens, etc.
	EmploymentStatus  string    `json:"employment_status"`
	EmploymentLength  int       `json:"employment_length"` // Months
	LastUpdated       time.Time `json:"last_updated"`
	DataSource        string    `json:"data_source"`
}

// NewCreditBureauProvider creates a new credit bureau provider
func NewCreditBureauProvider(provider, baseURL, apiKey string) *CreditBureauProvider {
	return &CreditBureauProvider{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		apiKey:   apiKey,
		baseURL:  baseURL,
		provider: provider,
	}
}

// GetCreditReport fetches credit report for a user
func (p *CreditBureauProvider) GetCreditReport(ctx context.Context, userID string) (*CreditBureauResponse, error) {
	logger.Info("Fetching credit report",
		zap.String("provider", p.provider),
		zap.String("userID", userID),
	)

	// Build request URL
	url := fmt.Sprintf("%s/v1/credit-reports/%s", p.baseURL, userID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	// Execute request
	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("credit bureau API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var creditData CreditBureauResponse
	if err := json.NewDecoder(resp.Body).Decode(&creditData); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	creditData.DataSource = p.provider
	creditData.LastUpdated = time.Now()

	logger.Info("Credit report fetched successfully",
		zap.String("provider", p.provider),
		zap.String("userID", userID),
		zap.Int("score", creditData.CreditScore),
	)

	return &creditData, nil
}

// GetCreditScore fetches only the credit score (lightweight endpoint)
func (p *CreditBureauProvider) GetCreditScore(ctx context.Context, userID string) (int, error) {
	url := fmt.Sprintf("%s/v1/credit-score/%s", p.baseURL, userID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return 0, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	req.Header.Set("Accept", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return 0, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return 0, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Score int `json:"score"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return 0, fmt.Errorf("failed to decode response: %w", err)
	}

	return result.Score, nil
}

// HealthCheck verifies the credit bureau API is accessible
func (p *CreditBureauProvider) HealthCheck(ctx context.Context) error {
	url := fmt.Sprintf("%s/health", p.baseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check returned status %d", resp.StatusCode)
	}

	return nil
}

// MockCreditBureauData generates mock data for testing
func (p *CreditBureauProvider) MockCreditBureauData(userID string) *CreditBureauResponse {
	// Generate deterministic mock data based on userID
	score := 650 + (len(userID) % 200) // Score between 650-850

	return &CreditBureauResponse{
		UserID:            userID,
		CreditScore:       score,
		ScoreRange:        "300-850",
		DebtToIncomeRatio: 0.35,
		TotalDebt:         45000,
		TotalIncome:       85000,
		PaymentHistory:    "good",
		CreditUtilization: 0.42,
		NumberOfAccounts:  8,
		OldestAccountAge:  72, // 6 years
		RecentInquiries:   2,
		Delinquencies:     0,
		PublicRecords:     0,
		EmploymentStatus:  "full-time",
		EmploymentLength:  48, // 4 years
		LastUpdated:       time.Now(),
		DataSource:        p.provider + "_mock",
	}
}
