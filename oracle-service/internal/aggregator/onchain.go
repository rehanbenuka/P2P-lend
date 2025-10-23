package aggregator

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/yourusername/p2p-lend/oracle-service/internal/models"
	"github.com/yourusername/p2p-lend/oracle-service/pkg/logger"
	"go.uber.org/zap"
)

// OnChainAggregator fetches and aggregates on-chain data
type OnChainAggregator struct {
	client *ethclient.Client
	rpcURL string
}

// NewOnChainAggregator creates a new on-chain data aggregator
func NewOnChainAggregator(rpcURL string) (*OnChainAggregator, error) {
	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to ethereum node: %w", err)
	}

	return &OnChainAggregator{
		client: client,
		rpcURL: rpcURL,
	}, nil
}

// FetchMetrics gathers on-chain metrics for a user address
func (a *OnChainAggregator) FetchMetrics(ctx context.Context, address string) (*models.OnChainMetrics, error) {
	addr := common.HexToAddress(address)

	metrics := &models.OnChainMetrics{
		UserAddress: address,
	}

	// Fetch wallet age
	walletAge, err := a.getWalletAge(ctx, addr)
	if err != nil {
		logger.Error("Failed to get wallet age", zap.Error(err))
	} else {
		metrics.WalletAge = walletAge
	}

	// Fetch transaction stats
	txCount, avgValue, err := a.getTransactionStats(ctx, addr)
	if err != nil {
		logger.Error("Failed to get transaction stats", zap.Error(err))
	} else {
		metrics.TotalTransactions = txCount
		metrics.AvgTransactionValue = avgValue
	}

	// Fetch balance as collateral indicator
	balance, err := a.client.BalanceAt(ctx, addr, nil)
	if err != nil {
		logger.Error("Failed to get balance", zap.Error(err))
	} else {
		// Convert wei to ETH
		ethBalance := new(big.Float).Quo(
			new(big.Float).SetInt(balance),
			big.NewFloat(1e18),
		)
		ethValue, _ := ethBalance.Float64()
		metrics.CollateralValue = ethValue
	}

	// Fetch DeFi interactions (would need specific contract calls)
	defiInteractions := a.getDeFiInteractions(ctx, addr)
	metrics.DeFiInteractions = defiInteractions

	// Fetch borrowing history (would query lending protocol contracts)
	borrowed, repaid, liquidations := a.getBorrowingHistory(ctx, addr)
	metrics.BorrowingHistory = borrowed
	metrics.RepaymentHistory = repaid
	metrics.LiquidationEvents = liquidations

	metrics.LastActivity = time.Now()
	metrics.UpdatedAt = time.Now()

	return metrics, nil
}

// getWalletAge calculates wallet age in days
func (a *OnChainAggregator) getWalletAge(ctx context.Context, address common.Address) (uint32, error) {
	// In a real implementation, you would:
	// 1. Query an indexer (like The Graph) for first transaction
	// 2. Or scan blocks backwards to find first transaction
	// For now, we'll use a simplified approach

	// Get current block number
	blockNum, err := a.client.BlockNumber(ctx)
	if err != nil {
		return 0, err
	}

	// Check transaction count
	nonce, err := a.client.NonceAt(ctx, address, big.NewInt(int64(blockNum)))
	if err != nil {
		return 0, err
	}

	// Simple estimation: if account has transactions, estimate age
	// In production, use proper block explorer API or indexer
	if nonce > 0 {
		// Rough estimate: 1 block per 12 seconds
		// Assume old accounts have been around proportional to nonce
		estimatedDays := uint32(nonce / 7200) // ~1 day worth of blocks
		if estimatedDays > 1825 { // Cap at 5 years
			estimatedDays = 1825
		}
		return estimatedDays, nil
	}

	return 0, nil
}

// getTransactionStats calculates transaction statistics
func (a *OnChainAggregator) getTransactionStats(ctx context.Context, address common.Address) (uint32, float64, error) {
	// Get transaction count
	nonce, err := a.client.NonceAt(ctx, address, nil)
	if err != nil {
		return 0, 0, err
	}

	txCount := uint32(nonce)

	// In production, you would:
	// 1. Use an indexer to get all transactions
	// 2. Calculate average value
	// For now, use a simplified estimation

	balance, err := a.client.BalanceAt(ctx, address, nil)
	if err != nil {
		return txCount, 0, err
	}

	// Simple average estimation
	avgValue := 0.0
	if txCount > 0 {
		ethBalance := new(big.Float).Quo(
			new(big.Float).SetInt(balance),
			big.NewFloat(1e18),
		)
		avgValue, _ = ethBalance.Float64()
		avgValue = avgValue / float64(txCount) * 2 // Rough estimation
	}

	return txCount, avgValue, nil
}

// getDeFiInteractions counts DeFi protocol interactions
func (a *OnChainAggregator) getDeFiInteractions(ctx context.Context, address common.Address) uint32 {
	// In production, you would:
	// 1. Query known DeFi protocol contracts (Aave, Compound, Uniswap, etc.)
	// 2. Count interactions via events or transaction history
	// 3. Use The Graph or similar indexer

	// For now, return a mock value based on transaction count
	nonce, err := a.client.NonceAt(ctx, address, nil)
	if err != nil {
		return 0
	}

	// Estimate ~20% of transactions are DeFi related
	return uint32(nonce) / 5
}

// getBorrowingHistory fetches lending protocol history
func (a *OnChainAggregator) getBorrowingHistory(ctx context.Context, address common.Address) (uint32, uint32, uint32) {
	// In production, you would:
	// 1. Query lending protocols (Aave, Compound, MakerDAO)
	// 2. Track borrow/repay events
	// 3. Monitor liquidation events

	// Mock implementation for demonstration
	nonce, err := a.client.NonceAt(ctx, address, nil)
	if err != nil {
		return 0, 0, 0
	}

	// Simple estimation
	borrowed := uint32(nonce) / 10
	repaid := borrowed - (borrowed / 10) // 90% repayment rate
	liquidations := borrowed / 20        // 5% liquidation rate

	return borrowed, repaid, liquidations
}

// Close closes the ethereum client connection
func (a *OnChainAggregator) Close() {
	if a.client != nil {
		a.client.Close()
	}
}

// HealthCheck verifies the connection to the ethereum node
func (a *OnChainAggregator) HealthCheck(ctx context.Context) error {
	_, err := a.client.BlockNumber(ctx)
	if err != nil {
		return fmt.Errorf("ethereum node health check failed: %w", err)
	}
	return nil
}
