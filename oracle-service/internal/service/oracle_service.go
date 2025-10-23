package service

import (
	"context"
	"fmt"
	"time"

	"github.com/yourusername/p2p-lend/oracle-service/internal/aggregator"
	"github.com/yourusername/p2p-lend/oracle-service/internal/blockchain"
	"github.com/yourusername/p2p-lend/oracle-service/internal/models"
	"github.com/yourusername/p2p-lend/oracle-service/internal/repository"
	"github.com/yourusername/p2p-lend/oracle-service/internal/scoring"
	"github.com/yourusername/p2p-lend/oracle-service/pkg/logger"
	"go.uber.org/zap"
)

// OracleService orchestrates credit score calculation and updates
type OracleService struct {
	repo            *repository.ScoreRepository
	scoringEngine   *scoring.Engine
	onChainAgg      *aggregator.OnChainAggregator
	offChainAgg     *aggregator.OffChainAggregator
	blockchainClient *blockchain.OracleClient
}

// NewOracleService creates a new oracle service
func NewOracleService(
	repo *repository.ScoreRepository,
	scoringEngine *scoring.Engine,
	onChainAgg *aggregator.OnChainAggregator,
	offChainAgg *aggregator.OffChainAggregator,
	blockchainClient *blockchain.OracleClient,
) *OracleService {
	return &OracleService{
		repo:             repo,
		scoringEngine:    scoringEngine,
		onChainAgg:       onChainAgg,
		offChainAgg:      offChainAgg,
		blockchainClient: blockchainClient,
	}
}

// CalculateAndUpdateScore calculates a new credit score for a user
func (s *OracleService) CalculateAndUpdateScore(ctx context.Context, address, userID string) (*models.CreditScore, error) {
	logger.Info("Starting credit score calculation",
		zap.String("address", address),
		zap.String("userID", userID),
	)

	// Fetch on-chain metrics
	onChainMetrics, err := s.onChainAgg.FetchMetrics(ctx, address)
	if err != nil {
		logger.Error("Failed to fetch on-chain metrics", zap.Error(err))
		return nil, fmt.Errorf("failed to fetch on-chain metrics: %w", err)
	}

	// Save on-chain metrics
	if err := s.repo.UpsertOnChainMetrics(ctx, onChainMetrics); err != nil {
		logger.Error("Failed to save on-chain metrics", zap.Error(err))
	}

	// Fetch off-chain metrics
	offChainMetrics, err := s.offChainAgg.FetchMetrics(ctx, userID, address)
	if err != nil {
		logger.Error("Failed to fetch off-chain metrics", zap.Error(err))
		// Continue with on-chain data only
		offChainMetrics = nil
	}

	// Save off-chain metrics if available
	if offChainMetrics != nil {
		if err := s.repo.UpsertOffChainMetrics(ctx, offChainMetrics); err != nil {
			logger.Error("Failed to save off-chain metrics", zap.Error(err))
		}
	}

	// Calculate credit score
	score, err := s.scoringEngine.CalculateScore(onChainMetrics, offChainMetrics)
	if err != nil {
		logger.Error("Failed to calculate score", zap.Error(err))
		return nil, fmt.Errorf("failed to calculate score: %w", err)
	}

	score.UserAddress = address

	// Save or update credit score
	existingScore, err := s.repo.GetByAddress(ctx, address)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing score: %w", err)
	}

	if existingScore != nil {
		// Update existing score
		score.ID = existingScore.ID
		score.CreatedAt = existingScore.CreatedAt
		score.UpdateCount = existingScore.UpdateCount + 1

		if err := s.repo.Update(ctx, score); err != nil {
			return nil, fmt.Errorf("failed to update score: %w", err)
		}
	} else {
		// Create new score
		score.UpdateCount = 1
		if err := s.repo.Create(ctx, score); err != nil {
			return nil, fmt.Errorf("failed to create score: %w", err)
		}
	}

	// Save to history
	history := &models.ScoreHistory{
		UserAddress: address,
		Score:       score.Score,
		Confidence:  score.Confidence,
		DataHash:    score.DataHash,
		Timestamp:   time.Now(),
	}
	if err := s.repo.CreateHistory(ctx, history); err != nil {
		logger.Error("Failed to save score history", zap.Error(err))
	}

	logger.Info("Credit score calculated successfully",
		zap.String("address", address),
		zap.Uint16("score", score.Score),
		zap.Uint8("confidence", score.Confidence),
	)

	return score, nil
}

// PublishScoreToBlockchain publishes a credit score to the blockchain
func (s *OracleService) PublishScoreToBlockchain(ctx context.Context, address string) error {
	// Get current score
	score, err := s.repo.GetByAddress(ctx, address)
	if err != nil {
		return fmt.Errorf("failed to get score: %w", err)
	}
	if score == nil {
		return fmt.Errorf("no score found for address %s", address)
	}

	logger.Info("Publishing score to blockchain",
		zap.String("address", address),
		zap.Uint16("score", score.Score),
	)

	// Submit to blockchain
	tx, err := s.blockchainClient.UpdateCreditScore(
		ctx,
		address,
		score.Score,
		score.Confidence,
		score.DataHash,
	)

	// Create oracle update record
	update := &models.OracleUpdate{
		UserAddress: address,
		Score:       score.Score,
		Confidence:  score.Confidence,
		DataHash:    score.DataHash,
		Status:      "pending",
	}

	if err != nil {
		update.Status = "failed"
		update.ErrorMessage = err.Error()
		logger.Error("Failed to publish to blockchain", zap.Error(err))
	} else if tx != nil {
		update.TxHash = tx.Hash().Hex()
	}

	if err := s.repo.CreateOracleUpdate(ctx, update); err != nil {
		logger.Error("Failed to save oracle update", zap.Error(err))
	}

	if err != nil {
		return fmt.Errorf("failed to publish to blockchain: %w", err)
	}

	logger.Info("Score published to blockchain successfully",
		zap.String("txHash", update.TxHash),
	)

	return nil
}

// GetScore retrieves a credit score for a user
func (s *OracleService) GetScore(ctx context.Context, address string) (*models.CreditScore, error) {
	return s.repo.GetByAddress(ctx, address)
}

// GetScoreHistory retrieves score history for a user
func (s *OracleService) GetScoreHistory(ctx context.Context, address string, limit int) ([]*models.ScoreHistory, error) {
	return s.repo.GetHistory(ctx, address, limit)
}

// ProcessScheduledUpdates processes scores that are due for update
func (s *OracleService) ProcessScheduledUpdates(ctx context.Context, batchSize int) error {
	scores, err := s.repo.GetDueForUpdate(ctx, batchSize)
	if err != nil {
		return fmt.Errorf("failed to get scores due for update: %w", err)
	}

	logger.Info("Processing scheduled updates",
		zap.Int("count", len(scores)),
	)

	for _, score := range scores {
		// Calculate new score
		_, err := s.CalculateAndUpdateScore(ctx, score.UserAddress, "")
		if err != nil {
			logger.Error("Failed to update score",
				zap.String("address", score.UserAddress),
				zap.Error(err),
			)
			continue
		}

		// Publish to blockchain
		if err := s.PublishScoreToBlockchain(ctx, score.UserAddress); err != nil {
			logger.Error("Failed to publish score",
				zap.String("address", score.UserAddress),
				zap.Error(err),
			)
		}
	}

	return nil
}

// GetStats retrieves service statistics
func (s *OracleService) GetStats(ctx context.Context) (map[string]interface{}, error) {
	return s.repo.GetStats(ctx)
}

// HealthCheck performs health checks on all components
func (s *OracleService) HealthCheck(ctx context.Context) map[string]bool {
	health := make(map[string]bool)

	// Check on-chain aggregator
	if err := s.onChainAgg.HealthCheck(ctx); err != nil {
		logger.Error("On-chain aggregator health check failed", zap.Error(err))
		health["onchain_aggregator"] = false
	} else {
		health["onchain_aggregator"] = true
	}

	// Check off-chain aggregator
	if err := s.offChainAgg.HealthCheck(ctx); err != nil {
		logger.Error("Off-chain aggregator health check failed", zap.Error(err))
		health["offchain_aggregator"] = false
	} else {
		health["offchain_aggregator"] = true
	}

	// Check blockchain client
	if err := s.blockchainClient.HealthCheck(ctx); err != nil {
		logger.Error("Blockchain client health check failed", zap.Error(err))
		health["blockchain_client"] = false
	} else {
		health["blockchain_client"] = true
	}

	return health
}
