package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/yourusername/p2p-lend/oracle-service/internal/models"
	"gorm.io/gorm"
)

// ScoreRepository handles database operations for credit scores
type ScoreRepository struct {
	db *gorm.DB
}

// NewScoreRepository creates a new score repository
func NewScoreRepository(db *gorm.DB) *ScoreRepository {
	return &ScoreRepository{db: db}
}

// Create creates a new credit score record
func (r *ScoreRepository) Create(ctx context.Context, score *models.CreditScore) error {
	return r.db.WithContext(ctx).Create(score).Error
}

// Update updates an existing credit score
func (r *ScoreRepository) Update(ctx context.Context, score *models.CreditScore) error {
	return r.db.WithContext(ctx).Save(score).Error
}

// GetByAddress retrieves a credit score by user address
func (r *ScoreRepository) GetByAddress(ctx context.Context, address string) (*models.CreditScore, error) {
	var score models.CreditScore
	err := r.db.WithContext(ctx).
		Where("user_address = ? AND is_active = ?", address, true).
		First(&score).Error

	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get credit score: %w", err)
	}

	return &score, nil
}

// GetAll retrieves all active credit scores with pagination
func (r *ScoreRepository) GetAll(ctx context.Context, limit, offset int) ([]*models.CreditScore, error) {
	var scores []*models.CreditScore
	err := r.db.WithContext(ctx).
		Where("is_active = ?", true).
		Order("last_updated DESC").
		Limit(limit).
		Offset(offset).
		Find(&scores).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get credit scores: %w", err)
	}

	return scores, nil
}

// GetDueForUpdate retrieves scores that need updating
func (r *ScoreRepository) GetDueForUpdate(ctx context.Context, limit int) ([]*models.CreditScore, error) {
	var scores []*models.CreditScore
	err := r.db.WithContext(ctx).
		Where("is_active = ? AND next_update_due <= ?", true, time.Now()).
		Order("next_update_due ASC").
		Limit(limit).
		Find(&scores).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get scores due for update: %w", err)
	}

	return scores, nil
}

// CreateHistory creates a historical score record
func (r *ScoreRepository) CreateHistory(ctx context.Context, history *models.ScoreHistory) error {
	return r.db.WithContext(ctx).Create(history).Error
}

// GetHistory retrieves score history for a user
func (r *ScoreRepository) GetHistory(ctx context.Context, address string, limit int) ([]*models.ScoreHistory, error) {
	var history []*models.ScoreHistory
	err := r.db.WithContext(ctx).
		Where("user_address = ?", address).
		Order("timestamp DESC").
		Limit(limit).
		Find(&history).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get score history: %w", err)
	}

	return history, nil
}

// UpsertOnChainMetrics creates or updates on-chain metrics
func (r *ScoreRepository) UpsertOnChainMetrics(ctx context.Context, metrics *models.OnChainMetrics) error {
	var existing models.OnChainMetrics
	err := r.db.WithContext(ctx).
		Where("user_address = ?", metrics.UserAddress).
		First(&existing).Error

	if err == gorm.ErrRecordNotFound {
		return r.db.WithContext(ctx).Create(metrics).Error
	}
	if err != nil {
		return fmt.Errorf("failed to check existing metrics: %w", err)
	}

	metrics.ID = existing.ID
	metrics.CreatedAt = existing.CreatedAt
	return r.db.WithContext(ctx).Save(metrics).Error
}

// UpsertOffChainMetrics creates or updates off-chain metrics
func (r *ScoreRepository) UpsertOffChainMetrics(ctx context.Context, metrics *models.OffChainMetrics) error {
	var existing models.OffChainMetrics
	err := r.db.WithContext(ctx).
		Where("user_address = ?", metrics.UserAddress).
		First(&existing).Error

	if err == gorm.ErrRecordNotFound {
		return r.db.WithContext(ctx).Create(metrics).Error
	}
	if err != nil {
		return fmt.Errorf("failed to check existing metrics: %w", err)
	}

	metrics.ID = existing.ID
	metrics.CreatedAt = existing.CreatedAt
	return r.db.WithContext(ctx).Save(metrics).Error
}

// GetOnChainMetrics retrieves on-chain metrics for a user
func (r *ScoreRepository) GetOnChainMetrics(ctx context.Context, address string) (*models.OnChainMetrics, error) {
	var metrics models.OnChainMetrics
	err := r.db.WithContext(ctx).
		Where("user_address = ?", address).
		First(&metrics).Error

	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get on-chain metrics: %w", err)
	}

	return &metrics, nil
}

// GetOffChainMetrics retrieves off-chain metrics for a user
func (r *ScoreRepository) GetOffChainMetrics(ctx context.Context, address string) (*models.OffChainMetrics, error) {
	var metrics models.OffChainMetrics
	err := r.db.WithContext(ctx).
		Where("user_address = ?", address).
		First(&metrics).Error

	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get off-chain metrics: %w", err)
	}

	return &metrics, nil
}

// CreateOracleUpdate records an oracle update transaction
func (r *ScoreRepository) CreateOracleUpdate(ctx context.Context, update *models.OracleUpdate) error {
	return r.db.WithContext(ctx).Create(update).Error
}

// UpdateOracleUpdate updates an oracle update record
func (r *ScoreRepository) UpdateOracleUpdate(ctx context.Context, update *models.OracleUpdate) error {
	return r.db.WithContext(ctx).Save(update).Error
}

// GetOracleUpdateByTxHash retrieves an oracle update by transaction hash
func (r *ScoreRepository) GetOracleUpdateByTxHash(ctx context.Context, txHash string) (*models.OracleUpdate, error) {
	var update models.OracleUpdate
	err := r.db.WithContext(ctx).
		Where("tx_hash = ?", txHash).
		First(&update).Error

	if err == gorm.ErrRecordNotFound {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get oracle update: %w", err)
	}

	return &update, nil
}

// GetPendingOracleUpdates retrieves pending oracle updates
func (r *ScoreRepository) GetPendingOracleUpdates(ctx context.Context) ([]*models.OracleUpdate, error) {
	var updates []*models.OracleUpdate
	err := r.db.WithContext(ctx).
		Where("status = ?", "pending").
		Order("created_at ASC").
		Find(&updates).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get pending updates: %w", err)
	}

	return updates, nil
}

// GetStats retrieves database statistics
func (r *ScoreRepository) GetStats(ctx context.Context) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Total active scores
	var totalScores int64
	if err := r.db.WithContext(ctx).Model(&models.CreditScore{}).Where("is_active = ?", true).Count(&totalScores).Error; err != nil {
		return nil, err
	}
	stats["total_active_scores"] = totalScores

	// Average score (use COALESCE to handle NULL when no records exist)
	var avgScore sql.NullFloat64
	if err := r.db.WithContext(ctx).Model(&models.CreditScore{}).Where("is_active = ?", true).Select("COALESCE(AVG(score), 0)").Scan(&avgScore).Error; err != nil {
		return nil, err
	}
	if avgScore.Valid {
		stats["average_score"] = avgScore.Float64
	} else {
		stats["average_score"] = 0.0
	}

	// Scores due for update
	var dueForUpdate int64
	if err := r.db.WithContext(ctx).Model(&models.CreditScore{}).Where("is_active = ? AND next_update_due <= ?", true, time.Now()).Count(&dueForUpdate).Error; err != nil {
		return nil, err
	}
	stats["due_for_update"] = dueForUpdate

	// Pending oracle updates
	var pendingUpdates int64
	if err := r.db.WithContext(ctx).Model(&models.OracleUpdate{}).Where("status = ?", "pending").Count(&pendingUpdates).Error; err != nil {
		return nil, err
	}
	stats["pending_oracle_updates"] = pendingUpdates

	return stats, nil
}
