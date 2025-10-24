package service

import (
	"context"
	"fmt"

	"github.com/yourusername/p2p-lend/oracle-service/internal/aggregator"
	"github.com/yourusername/p2p-lend/oracle-service/internal/models"
	"github.com/yourusername/p2p-lend/oracle-service/internal/providers"
	"github.com/yourusername/p2p-lend/oracle-service/pkg/logger"
	"go.uber.org/zap"
)

// EnhancedOracleService provides credit scoring with 3rd party integrations
type EnhancedOracleService struct {
	baseService          *OracleService
	enhancedOnChainAgg   *aggregator.EnhancedOnChainAggregator
	enhancedOffChainAgg  *aggregator.EnhancedOffChainAggregator
	creditBureauProvider *providers.CreditBureauProvider
	plaidProvider        *providers.PlaidProvider
	blockchainProvider   *providers.BlockchainDataProvider
}

// ProviderData contains data fetched from all providers
type ProviderData struct {
	Sources          []string
	CreditBureauData *providers.CreditBureauResponse
	PlaidData        *providers.PlaidAccountSummary
	BlockchainData   *providers.BlockchainSummary
}

// NewEnhancedOracleService creates an enhanced oracle service
func NewEnhancedOracleService(
	baseService *OracleService,
	enhancedOnChainAgg *aggregator.EnhancedOnChainAggregator,
	enhancedOffChainAgg *aggregator.EnhancedOffChainAggregator,
	creditBureauProvider *providers.CreditBureauProvider,
	plaidProvider *providers.PlaidProvider,
	blockchainProvider *providers.BlockchainDataProvider,
) *EnhancedOracleService {
	return &EnhancedOracleService{
		baseService:          baseService,
		enhancedOnChainAgg:   enhancedOnChainAgg,
		enhancedOffChainAgg:  enhancedOffChainAgg,
		creditBureauProvider: creditBureauProvider,
		plaidProvider:        plaidProvider,
		blockchainProvider:   blockchainProvider,
	}
}

// CalculateWithProviders calculates credit score using selected 3rd party providers
func (s *EnhancedOracleService) CalculateWithProviders(
	ctx context.Context,
	address, bureauUserID, plaidUserID, plaidAccessToken string,
	fetchCreditBureau, fetchPlaid, fetchBlockchain bool,
) (*models.CreditScore, *ProviderData, error) {

	logger.Info("Calculating credit score with providers",
		zap.String("address", address),
		zap.String("bureauUserID", bureauUserID),
		zap.String("plaidUserID", plaidUserID),
		zap.Bool("creditBureau", fetchCreditBureau),
		zap.Bool("plaid", fetchPlaid),
		zap.Bool("blockchain", fetchBlockchain),
	)

	providerData := &ProviderData{
		Sources: []string{},
	}

	var onChainMetrics *models.OnChainMetrics
	var offChainMetrics *models.OffChainMetrics
	var err error

	// Fetch on-chain data
	if fetchBlockchain {
		logger.Info("Fetching blockchain data via providers")
		onChainMetrics, err = s.enhancedOnChainAgg.FetchMetrics(ctx, address)
		if err != nil {
			logger.Error("Failed to fetch enhanced on-chain metrics", zap.Error(err))
			return nil, nil, fmt.Errorf("failed to fetch blockchain data: %w", err)
		}
		providerData.Sources = append(providerData.Sources, "blockchain_provider")

		// Also get the raw blockchain data for response
		providerData.BlockchainData = s.blockchainProvider.MockBlockchainData(address)
	} else {
		// Use basic on-chain aggregation
		logger.Info("Fetching on-chain data via direct RPC")
		onChainMetrics, err = s.baseService.onChainAgg.FetchMetrics(ctx, address)
		if err != nil {
			logger.Error("Failed to fetch on-chain metrics", zap.Error(err))
		}
		providerData.Sources = append(providerData.Sources, "ethereum_rpc")
	}

	// Fetch off-chain data
	if fetchCreditBureau || fetchPlaid {
		logger.Info("Fetching off-chain data via providers")
		// Use bureauUserID for off-chain aggregation (as it's the primary identifier)
		userIDForOffChain := bureauUserID
		if userIDForOffChain == "" {
			userIDForOffChain = plaidUserID
		}

		offChainMetrics, err = s.enhancedOffChainAgg.FetchMetrics(ctx, userIDForOffChain, address)
		if err != nil {
			logger.Error("Failed to fetch enhanced off-chain metrics", zap.Error(err))
		}

		// Get detailed provider data
		if fetchCreditBureau && bureauUserID != "" {
			providerData.CreditBureauData = s.creditBureauProvider.MockCreditBureauData(bureauUserID)
			providerData.Sources = append(providerData.Sources, "credit_bureau")
		}

		if fetchPlaid && plaidUserID != "" {
			if plaidAccessToken != "" {
				// In production, use the access token
				// plaidData, err := s.plaidProvider.GetAccountSummary(ctx, plaidAccessToken)
				providerData.PlaidData = s.plaidProvider.MockPlaidData(plaidUserID)
			} else {
				providerData.PlaidData = s.plaidProvider.MockPlaidData(plaidUserID)
			}
			providerData.Sources = append(providerData.Sources, "plaid")
		}
	} else {
		// Use basic off-chain aggregation
		logger.Info("Fetching off-chain data via basic aggregation")
		// Use bureauUserID or plaidUserID as fallback
		userIDForOffChain := bureauUserID
		if userIDForOffChain == "" {
			userIDForOffChain = plaidUserID
		}

		offChainMetrics, err = s.baseService.offChainAgg.FetchMetrics(ctx, userIDForOffChain, address)
		if err != nil {
			logger.Error("Failed to fetch off-chain metrics", zap.Error(err))
		}
		providerData.Sources = append(providerData.Sources, "basic_aggregation")
	}

	// Save metrics
	if onChainMetrics != nil {
		onChainMetrics.UserAddress = address
		if err := s.baseService.repo.UpsertOnChainMetrics(ctx, onChainMetrics); err != nil {
			logger.Error("Failed to save on-chain metrics", zap.Error(err))
		}
	}

	if offChainMetrics != nil {
		offChainMetrics.UserAddress = address
		if err := s.baseService.repo.UpsertOffChainMetrics(ctx, offChainMetrics); err != nil {
			logger.Error("Failed to save off-chain metrics", zap.Error(err))
		}
	}

	// Calculate credit score
	score, err := s.baseService.scoringEngine.CalculateScore(onChainMetrics, offChainMetrics)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to calculate score: %w", err)
	}

	score.UserAddress = address

	// Save score
	existingScore, err := s.baseService.repo.GetByAddress(ctx, address)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to check existing score: %w", err)
	}

	if existingScore != nil {
		score.ID = existingScore.ID
		score.CreatedAt = existingScore.CreatedAt
		score.UpdateCount = existingScore.UpdateCount + 1
		if err := s.baseService.repo.Update(ctx, score); err != nil {
			return nil, nil, fmt.Errorf("failed to update score: %w", err)
		}
	} else {
		score.UpdateCount = 1
		if err := s.baseService.repo.Create(ctx, score); err != nil {
			return nil, nil, fmt.Errorf("failed to create score: %w", err)
		}
	}

	// Save history
	history := &models.ScoreHistory{
		UserAddress: address,
		Score:       score.Score,
		Confidence:  score.Confidence,
		DataHash:    score.DataHash,
		Timestamp:   score.LastUpdated,
	}
	if err := s.baseService.repo.CreateHistory(ctx, history); err != nil {
		logger.Error("Failed to save score history", zap.Error(err))
	}

	logger.Info("Credit score calculated with providers",
		zap.String("address", address),
		zap.Uint16("score", score.Score),
		zap.Strings("sources", providerData.Sources),
	)

	return score, providerData, nil
}

// PublishScoreToBlockchain publishes score to blockchain
func (s *EnhancedOracleService) PublishScoreToBlockchain(ctx context.Context, address string) error {
	return s.baseService.PublishScoreToBlockchain(ctx, address)
}

// GetProviderStatus checks health of all providers
func (s *EnhancedOracleService) GetProviderStatus(ctx context.Context) map[string]interface{} {
	status := make(map[string]interface{})

	// Check credit bureau
	if err := s.creditBureauProvider.HealthCheck(ctx); err != nil {
		status["credit_bureau"] = map[string]interface{}{
			"healthy": false,
			"error":   err.Error(),
		}
	} else {
		status["credit_bureau"] = map[string]interface{}{
			"healthy": true,
		}
	}

	// Check Plaid
	if err := s.plaidProvider.HealthCheck(ctx); err != nil {
		status["plaid"] = map[string]interface{}{
			"healthy": false,
			"error":   err.Error(),
		}
	} else {
		status["plaid"] = map[string]interface{}{
			"healthy": true,
		}
	}

	// Check blockchain provider
	if err := s.blockchainProvider.HealthCheck(ctx); err != nil {
		status["blockchain_provider"] = map[string]interface{}{
			"healthy": false,
			"error":   err.Error(),
		}
	} else {
		status["blockchain_provider"] = map[string]interface{}{
			"healthy": true,
		}
	}

	// Check on-chain aggregators
	if err := s.enhancedOnChainAgg.HealthCheck(ctx); err != nil {
		status["onchain_aggregator"] = map[string]interface{}{
			"healthy": false,
			"error":   err.Error(),
		}
	} else {
		status["onchain_aggregator"] = map[string]interface{}{
			"healthy": true,
		}
	}

	// Check off-chain aggregators
	if err := s.enhancedOffChainAgg.HealthCheck(ctx); err != nil {
		status["offchain_aggregator"] = map[string]interface{}{
			"healthy": false,
			"error":   err.Error(),
		}
	} else {
		status["offchain_aggregator"] = map[string]interface{}{
			"healthy": true,
		}
	}

	return status
}

// GetScore retrieves a credit score
func (s *EnhancedOracleService) GetScore(ctx context.Context, address string) (*models.CreditScore, error) {
	return s.baseService.GetScore(ctx, address)
}

// GetScoreHistory retrieves score history
func (s *EnhancedOracleService) GetScoreHistory(ctx context.Context, address string, limit int) ([]*models.ScoreHistory, error) {
	return s.baseService.GetScoreHistory(ctx, address, limit)
}

// GetStats retrieves service statistics
func (s *EnhancedOracleService) GetStats(ctx context.Context) (map[string]interface{}, error) {
	return s.baseService.GetStats(ctx)
}
