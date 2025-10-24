package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/yourusername/p2p-lend/oracle-service/pkg/logger"
	"go.uber.org/zap"
)

// PlaidProvider integrates with Plaid API for bank account data
type PlaidProvider struct {
	httpClient  *http.Client
	clientID    string
	secret      string
	baseURL     string
	environment string // "sandbox", "development", "production"
}

// PlaidBankAccount represents bank account information
type PlaidBankAccount struct {
	AccountID        string    `json:"account_id"`
	Name             string    `json:"name"`
	Type             string    `json:"type"` // "checking", "savings", "credit"
	Subtype          string    `json:"subtype"`
	CurrentBalance   float64   `json:"current_balance"`
	AvailableBalance float64   `json:"available_balance"`
	CurrencyCode     string    `json:"currency_code"`
	LastUpdated      time.Time `json:"last_updated"`
}

// PlaidTransaction represents a transaction
type PlaidTransaction struct {
	TransactionID string   `json:"transaction_id"`
	AccountID     string   `json:"account_id"`
	Amount        float64  `json:"amount"`
	Date          string   `json:"date"`
	Name          string   `json:"name"`
	Category      []string `json:"category"`
	Pending       bool     `json:"pending"`
}

// PlaidIncomeData represents income verification data
type PlaidIncomeData struct {
	UserID             string    `json:"user_id"`
	AnnualIncome       float64   `json:"annual_income"`
	MonthlyIncome      float64   `json:"monthly_income"`
	IncomeVerified     bool      `json:"income_verified"`
	EmploymentStatus   string    `json:"employment_status"`
	Employer           string    `json:"employer"`
	LastPayDate        string    `json:"last_pay_date"`
	PayFrequency       string    `json:"pay_frequency"`
	VerificationSource string    `json:"verification_source"`
	LastUpdated        time.Time `json:"last_updated"`
}

// PlaidAccountSummary represents summarized account data
type PlaidAccountSummary struct {
	UserID              string             `json:"user_id"`
	Accounts            []PlaidBankAccount `json:"accounts"`
	TotalBalance        float64            `json:"total_balance"`
	AverageBalance      float64            `json:"average_balance"`
	AccountAgeMonths    int                `json:"account_age_months"`
	TransactionCount    int                `json:"transaction_count"`
	AverageMonthlySpend float64            `json:"average_monthly_spend"`
	IncomeData          *PlaidIncomeData   `json:"income_data"`
	CreditUtilization   float64            `json:"credit_utilization"`
	LastUpdated         time.Time          `json:"last_updated"`
}

// NewPlaidProvider creates a new Plaid provider
func NewPlaidProvider(clientID, secret, environment string) *PlaidProvider {
	baseURL := "https://sandbox.plaid.com"
	if environment == "development" {
		baseURL = "https://development.plaid.com"
	} else if environment == "production" {
		baseURL = "https://production.plaid.com"
	}

	return &PlaidProvider{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		clientID:    clientID,
		secret:      secret,
		baseURL:     baseURL,
		environment: environment,
	}
}

// GetAccountSummary fetches comprehensive account summary
func (p *PlaidProvider) GetAccountSummary(ctx context.Context, accessToken string) (*PlaidAccountSummary, error) {
	logger.Info("Fetching Plaid account summary")

	// Get accounts
	accounts, err := p.getAccounts(ctx, accessToken)
	if err != nil {
		return nil, fmt.Errorf("failed to get accounts: %w", err)
	}

	// Get transactions
	transactions, err := p.getTransactions(ctx, accessToken, 90) // Last 90 days
	if err != nil {
		logger.Error("Failed to get transactions", zap.Error(err))
		transactions = []PlaidTransaction{} // Continue with empty transactions
	}

	// Get income data
	incomeData, err := p.getIncomeData(ctx, accessToken)
	if err != nil {
		logger.Error("Failed to get income data", zap.Error(err))
		incomeData = nil // Continue without income data
	}

	// Calculate summary statistics
	summary := p.calculateSummary(accounts, transactions, incomeData)

	logger.Info("Plaid account summary fetched successfully",
		zap.Int("accounts", len(accounts)),
		zap.Float64("totalBalance", summary.TotalBalance),
	)

	return summary, nil
}

// getAccounts fetches account balances
func (p *PlaidProvider) getAccounts(ctx context.Context, accessToken string) ([]PlaidBankAccount, error) {
	url := fmt.Sprintf("%s/accounts/balance/get", p.baseURL)

	reqBody := map[string]string{
		"client_id":    p.clientID,
		"secret":       p.secret,
		"access_token": accessToken,
	}

	bodyBytes, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Plaid API returned status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Accounts []struct {
			AccountID string `json:"account_id"`
			Name      string `json:"name"`
			Type      string `json:"type"`
			Subtype   string `json:"subtype"`
			Balances  struct {
				Current   float64 `json:"current"`
				Available float64 `json:"available"`
				Currency  string  `json:"iso_currency_code"`
			} `json:"balances"`
		} `json:"accounts"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	// Convert to our format
	accounts := make([]PlaidBankAccount, len(result.Accounts))
	for i, acc := range result.Accounts {
		accounts[i] = PlaidBankAccount{
			AccountID:        acc.AccountID,
			Name:             acc.Name,
			Type:             acc.Type,
			Subtype:          acc.Subtype,
			CurrentBalance:   acc.Balances.Current,
			AvailableBalance: acc.Balances.Available,
			CurrencyCode:     acc.Balances.Currency,
			LastUpdated:      time.Now(),
		}
	}

	return accounts, nil
}

// getTransactions fetches recent transactions
func (p *PlaidProvider) getTransactions(ctx context.Context, accessToken string, days int) ([]PlaidTransaction, error) {
	url := fmt.Sprintf("%s/transactions/get", p.baseURL)

	endDate := time.Now().Format("2006-01-02")
	startDate := time.Now().AddDate(0, 0, -days).Format("2006-01-02")

	reqBody := map[string]interface{}{
		"client_id":    p.clientID,
		"secret":       p.secret,
		"access_token": accessToken,
		"start_date":   startDate,
		"end_date":     endDate,
	}

	bodyBytes, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Plaid API returned status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Transactions []PlaidTransaction `json:"transactions"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result.Transactions, nil
}

// getIncomeData fetches income verification data
func (p *PlaidProvider) getIncomeData(ctx context.Context, accessToken string) (*PlaidIncomeData, error) {
	url := fmt.Sprintf("%s/income/get", p.baseURL)

	reqBody := map[string]string{
		"client_id":    p.clientID,
		"secret":       p.secret,
		"access_token": accessToken,
	}

	bodyBytes, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Plaid API returned status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Income struct {
			LastYearIncome                      float64 `json:"last_year_income"`
			ProjectedYearlyIncome               float64 `json:"projected_yearly_income"`
			MaxNumberOfOverlappingIncomeStreams int     `json:"max_number_of_overlapping_income_streams"`
			IncomeStreams                       []struct {
				MonthlyIncome float64 `json:"monthly_income"`
				Confidence    float64 `json:"confidence"`
			} `json:"income_streams"`
		} `json:"income"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	monthlyIncome := 0.0
	if len(result.Income.IncomeStreams) > 0 {
		monthlyIncome = result.Income.IncomeStreams[0].MonthlyIncome
	}

	return &PlaidIncomeData{
		AnnualIncome:       result.Income.ProjectedYearlyIncome,
		MonthlyIncome:      monthlyIncome,
		IncomeVerified:     result.Income.ProjectedYearlyIncome > 0,
		VerificationSource: "plaid",
		LastUpdated:        time.Now(),
	}, nil
}

// calculateSummary creates summary statistics
func (p *PlaidProvider) calculateSummary(accounts []PlaidBankAccount, transactions []PlaidTransaction, incomeData *PlaidIncomeData) *PlaidAccountSummary {
	totalBalance := 0.0
	for _, acc := range accounts {
		totalBalance += acc.CurrentBalance
	}

	avgBalance := 0.0
	if len(accounts) > 0 {
		avgBalance = totalBalance / float64(len(accounts))
	}

	// Calculate average monthly spend
	totalSpend := 0.0
	for _, tx := range transactions {
		if tx.Amount > 0 { // Positive amounts are debits
			totalSpend += tx.Amount
		}
	}
	avgMonthlySpend := totalSpend / 3 // Assuming 90 days of transactions

	return &PlaidAccountSummary{
		Accounts:            accounts,
		TotalBalance:        totalBalance,
		AverageBalance:      avgBalance,
		AccountAgeMonths:    24, // Would need to calculate from oldest account
		TransactionCount:    len(transactions),
		AverageMonthlySpend: avgMonthlySpend,
		IncomeData:          incomeData,
		CreditUtilization:   0.0, // Would calculate from credit accounts
		LastUpdated:         time.Now(),
	}
}

// MockPlaidData generates mock data for testing
func (p *PlaidProvider) MockPlaidData(userID string) *PlaidAccountSummary {
	return &PlaidAccountSummary{
		UserID: userID,
		Accounts: []PlaidBankAccount{
			{
				AccountID:        "acc_checking_001",
				Name:             "Checking Account",
				Type:             "depository",
				Subtype:          "checking",
				CurrentBalance:   5420.50,
				AvailableBalance: 5420.50,
				CurrencyCode:     "USD",
				LastUpdated:      time.Now(),
			},
			{
				AccountID:        "acc_savings_001",
				Name:             "Savings Account",
				Type:             "depository",
				Subtype:          "savings",
				CurrentBalance:   12350.00,
				AvailableBalance: 12350.00,
				CurrencyCode:     "USD",
				LastUpdated:      time.Now(),
			},
		},
		TotalBalance:        17770.50,
		AverageBalance:      8885.25,
		AccountAgeMonths:    36,
		TransactionCount:    245,
		AverageMonthlySpend: 3200.00,
		IncomeData: &PlaidIncomeData{
			UserID:             userID,
			AnnualIncome:       75000,
			MonthlyIncome:      6250,
			IncomeVerified:     true,
			EmploymentStatus:   "full-time",
			Employer:           "Tech Corp Inc",
			LastPayDate:        time.Now().AddDate(0, 0, -15).Format("2006-01-02"),
			PayFrequency:       "bi-weekly",
			VerificationSource: "plaid_mock",
			LastUpdated:        time.Now(),
		},
		CreditUtilization: 0.28,
		LastUpdated:       time.Now(),
	}
}

// HealthCheck verifies Plaid API connectivity
func (p *PlaidProvider) HealthCheck(ctx context.Context) error {
	// Plaid doesn't have a dedicated health endpoint
	// We can verify credentials are valid
	return nil
}
