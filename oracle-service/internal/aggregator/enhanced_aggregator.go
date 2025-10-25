package aggregator

import (
	"context"
	"fmt"
	"time"

	"github.com/yourusername/p2p-lend/oracle-service/internal/models"
	"github.com/yourusername/p2p-lend/oracle-service/internal/providers"
	"github.com/yourusername/p2p-lend/oracle-service/pkg/logger"
	"go.uber.org/zap"
)

// EnhancedOffChainAggregator uses real 3rd party APIs to fetch credit data
type EnhancedOffChainAggregator struct {
	creditBureauProvider *providers.CreditBureauProvider
	plaidProvider        *providers.PlaidProvider
	useMockData          bool
}

// NewEnhancedOffChainAggregator creates an enhanced off-chain aggregator
func NewEnhancedOffChainAggregator(
	creditBureauProvider *providers.CreditBureauProvider,
	plaidProvider *providers.PlaidProvider,
	useMockData bool,
) *EnhancedOffChainAggregator {
	return &EnhancedOffChainAggregator{
		creditBureauProvider: creditBureauProvider,
		plaidProvider:        plaidProvider,
		useMockData:          useMockData,
	}
}

// FetchMetrics gathers comprehensive off-chain metrics
func (a *EnhancedOffChainAggregator) FetchMetrics(ctx context.Context, userID, address string) (*models.OffChainMetrics, error) {
	logger.Info("Fetching enhanced off-chain metrics",
		zap.String("userID", userID),
		zap.String("address", address),
		zap.Bool("mockData", a.useMockData),
	)

	metrics := &models.OffChainMetrics{
		UserAddress: address,
	}

	// Fetch credit bureau data
	if a.useMockData {
		logger.Info("Using mock credit bureau data")
		creditData := a.creditBureauProvider.MockCreditBureauData(userID)
		metrics.TraditionalCreditScore = uint16(creditData.CreditScore)
		metrics.DebtToIncomeRatio = creditData.DebtToIncomeRatio
		metrics.EmploymentStatus = creditData.EmploymentStatus
		metrics.DataSource = creditData.DataSource
	} else {
		creditData, err := a.creditBureauProvider.GetCreditReport(ctx, userID)
		if err != nil {
			logger.Error("Failed to fetch credit bureau data", zap.Error(err))
			// Continue with partial data
		} else {
			metrics.TraditionalCreditScore = uint16(creditData.CreditScore)
			metrics.DebtToIncomeRatio = creditData.DebtToIncomeRatio
			metrics.EmploymentStatus = creditData.EmploymentStatus
			metrics.DataSource = creditData.DataSource
		}
	}

	// Fetch Plaid banking data
	if a.useMockData {
		logger.Info("Using mock Plaid data")
		plaidData := a.plaidProvider.MockPlaidData(userID)
		if plaidData.IncomeData != nil {
			metrics.IncomeVerified = plaidData.IncomeData.IncomeVerified
			metrics.IncomeLevel = a.categorizeIncome(plaidData.IncomeData.AnnualIncome)

			// Calculate bank account history score
			metrics.BankAccountHistory = a.calculateBankScore(plaidData)
		}
	} else {
		// Note: In production, you'd get the Plaid access token from your database
		// For now, we'll use mock data
		logger.Warn("Plaid requires access token - using mock data")
		plaidData := a.plaidProvider.MockPlaidData(userID)
		if plaidData.IncomeData != nil {
			metrics.IncomeVerified = plaidData.IncomeData.IncomeVerified
			metrics.IncomeLevel = a.categorizeIncome(plaidData.IncomeData.AnnualIncome)
			metrics.BankAccountHistory = a.calculateBankScore(plaidData)
		}
	}

	metrics.LastVerified = time.Now()
	metrics.UpdatedAt = time.Now()

	logger.Info("Enhanced off-chain metrics fetched successfully",
		zap.Uint16("creditScore", metrics.TraditionalCreditScore),
		zap.Bool("incomeVerified", metrics.IncomeVerified),
		zap.String("incomeLevel", metrics.IncomeLevel),
	)

	return metrics, nil
}

// categorizeIncome categorizes annual income into levels
func (a *EnhancedOffChainAggregator) categorizeIncome(annualIncome float64) string {
	if annualIncome >= 100000 {
		return "high"
	} else if annualIncome >= 50000 {
		return "medium"
	}
	return "low"
}

// calculateBankScore creates a bank account history score (0-100)
func (a *EnhancedOffChainAggregator) calculateBankScore(plaidData *providers.PlaidAccountSummary) uint8 {
	score := 0.0

	// Account age (30 points)
	if plaidData.AccountAgeMonths >= 36 {
		score += 30
	} else {
		score += float64(plaidData.AccountAgeMonths) / 36.0 * 30
	}

	// Average balance (25 points)
	if plaidData.AverageBalance >= 5000 {
		score += 25
	} else {
		score += (plaidData.AverageBalance / 5000.0) * 25
	}

	// Transaction activity (20 points)
	if plaidData.TransactionCount >= 100 {
		score += 20
	} else {
		score += float64(plaidData.TransactionCount) / 100.0 * 20
	}

	// Savings rate (25 points)
	if plaidData.IncomeData != nil && plaidData.IncomeData.MonthlyIncome > 0 {
		savingsRate := (plaidData.AverageBalance - plaidData.AverageMonthlySpend) / plaidData.IncomeData.MonthlyIncome
		if savingsRate >= 0.20 { // 20% savings rate
			score += 25
		} else if savingsRate > 0 {
			score += savingsRate / 0.20 * 25
		}
	}

	if score > 100 {
		score = 100
	}

	return uint8(score)
}

// HealthCheck verifies all providers are healthy
func (a *EnhancedOffChainAggregator) HealthCheck(ctx context.Context) error {
	if a.useMockData {
		return nil // Mock data always healthy
	}

	// Check credit bureau
	if err := a.creditBureauProvider.HealthCheck(ctx); err != nil {
		return fmt.Errorf("credit bureau unhealthy: %w", err)
	}

	// Check Plaid
	if err := a.plaidProvider.HealthCheck(ctx); err != nil {
		return fmt.Errorf("plaid unhealthy: %w", err)
	}

	return nil
}

// EnhancedOnChainAggregator uses blockchain data providers
type EnhancedOnChainAggregator struct {
	blockchainProvider *providers.BlockchainDataProvider
	blockscoutProvider *providers.BlockscoutProvider
	ethClient          *OnChainAggregator // Fallback to direct RPC
	useMockData        bool
	preferBlockscout   bool     // Prefer Blockscout over other providers
	enableMultiChain   bool     // Enable multi-chain data fetching
	targetChains       []string // Target chains to fetch from
}

// NewEnhancedOnChainAggregator creates an enhanced on-chain aggregator
func NewEnhancedOnChainAggregator(
	blockchainProvider *providers.BlockchainDataProvider,
	blockscoutProvider *providers.BlockscoutProvider,
	ethClient *OnChainAggregator,
	useMockData bool,
	preferBlockscout bool,
	enableMultiChain bool,
	targetChains []string,
) *EnhancedOnChainAggregator {
	return &EnhancedOnChainAggregator{
		blockchainProvider: blockchainProvider,
		blockscoutProvider: blockscoutProvider,
		ethClient:          ethClient,
		useMockData:        useMockData,
		preferBlockscout:   preferBlockscout,
		enableMultiChain:   enableMultiChain,
		targetChains:       targetChains,
	}
}

// FetchMetrics gathers enhanced on-chain metrics
func (a *EnhancedOnChainAggregator) FetchMetrics(ctx context.Context, address string) (*models.OnChainMetrics, error) {
	logger.Info("Fetching enhanced on-chain metrics",
		zap.String("address", address),
		zap.Bool("mockData", a.useMockData),
		zap.Bool("preferBlockscout", a.preferBlockscout),
		zap.Bool("multiChain", a.enableMultiChain),
		zap.Strings("targetChains", a.targetChains),
	)

	var blockchainData *providers.BlockchainSummary
	var err error

	// MULTI-CHAIN FETCHING: Aggregate data from multiple EVM chains
	if a.enableMultiChain && a.blockscoutProvider != nil {
		logger.Info("Fetching from multiple chains", zap.Strings("chains", a.targetChains))
		multiChainData, err := providers.GetMultiChainAnalytics(ctx, address, a.targetChains)
		if err != nil {
			logger.Error("Failed to fetch multi-chain data", zap.Error(err))
		} else if multiChainData.TotalTransactions > 0 {
			blockchainData = providers.ConvertMultiChainToBlockchainSummary(multiChainData)
			logger.Info("Multi-chain data fetched successfully",
				zap.Int("activeChains", multiChainData.TotalChains),
				zap.Strings("chains", multiChainData.ActiveChains),
				zap.Int("totalTxs", multiChainData.TotalTransactions),
			)
		}
	}

	// SINGLE CHAIN FALLBACK: Try Blockscout for single chain if multi-chain failed
	if blockchainData == nil && a.preferBlockscout && a.blockscoutProvider != nil {
		logger.Info("Fetching from Blockscout (single chain)")
		blockscoutData, err := a.blockscoutProvider.GetAnalytics(ctx, address)
		if err != nil {
			logger.Error("Failed to fetch from Blockscout, trying alternative provider", zap.Error(err))
		} else {
			blockchainData = a.blockscoutProvider.ConvertToBlockchainSummary(blockscoutData)
		}
	}

	// Fallback to Covalent/Moralis if Blockscout failed or not preferred
	if blockchainData == nil {
		logger.Info("Fetching from blockchain data provider (Covalent/Moralis)")
		blockchainData, err = a.blockchainProvider.GetBlockchainSummary(ctx, address, "1") // Ethereum mainnet
		if err != nil {
			logger.Error("Failed to fetch from blockchain provider, trying direct RPC", zap.Error(err))
		}
	}

	// Final fallback to direct RPC if all providers failed
	if blockchainData == nil {
		logger.Warn("All blockchain providers failed, falling back to direct RPC")
		return a.ethClient.FetchMetrics(ctx, address)
	}

	// NOTE: On-chain data should ALWAYS be real, never use mock data
	// useMockData flag only applies to off-chain APIs (Plaid, Credit Bureau)
	// If all blockchain data sources fail, the direct RPC fallback above will handle it

	// Convert blockchain summary to OnChainMetrics
	metrics := &models.OnChainMetrics{
		UserAddress:         address,
		WalletAge:           uint32(blockchainData.WalletAge),
		TotalTransactions:   uint32(blockchainData.TotalTransactions),
		AvgTransactionValue: blockchainData.AverageTransactionSize,
		DeFiInteractions:    uint32(len(blockchainData.DeFiActivities)),
		CollateralValue:     blockchainData.TotalPortfolioValue,
		LastActivity:        blockchainData.LastTransaction,
		UpdatedAt:           time.Now(),
	}

	// Calculate borrowing metrics from lending positions
	borrowCount := 0
	repayCount := 0
	for _, pos := range blockchainData.LendingPositions {
		if pos.BorrowedAmount > 0 {
			borrowCount++
			if pos.HealthFactor > 1.5 { // Healthy position = good repayment
				repayCount++
			}
		}
	}

	metrics.BorrowingHistory = uint32(borrowCount)
	metrics.RepaymentHistory = uint32(repayCount)
	metrics.LiquidationEvents = uint32(len(blockchainData.LiquidationEvents))

	logger.Info("Enhanced on-chain metrics fetched successfully",
		zap.Uint32("walletAge", metrics.WalletAge),
		zap.Uint32("transactions", metrics.TotalTransactions),
		zap.Uint32("defiInteractions", metrics.DeFiInteractions),
	)

	return metrics, nil
}

// HealthCheck verifies blockchain provider is healthy
func (a *EnhancedOnChainAggregator) HealthCheck(ctx context.Context) error {
	if a.useMockData {
		return nil
	}

	// Check Blockscout if available
	if a.blockscoutProvider != nil {
		if err := a.blockscoutProvider.HealthCheck(ctx); err != nil {
			logger.Warn("Blockscout health check failed", zap.Error(err))
		}
	}

	// Check blockchain provider
	if err := a.blockchainProvider.HealthCheck(ctx); err != nil {
		// Try fallback
		return a.ethClient.HealthCheck(ctx)
	}

	return nil
}

// Close closes connections
func (a *EnhancedOnChainAggregator) Close() {
	if a.ethClient != nil {
		a.ethClient.Close()
	}
}
