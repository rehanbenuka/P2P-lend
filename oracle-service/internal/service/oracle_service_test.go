package service

import (
	"context"
	"testing"
	"time"

	"github.com/yourusername/p2p-lend/oracle-service/internal/aggregator"
	"github.com/yourusername/p2p-lend/oracle-service/internal/models"
	"github.com/yourusername/p2p-lend/oracle-service/internal/repository"
	"github.com/yourusername/p2p-lend/oracle-service/internal/scoring"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Mock on-chain aggregator for testing
type mockOnChainAggregator struct{}

func (m *mockOnChainAggregator) FetchMetrics(ctx context.Context, address string) (*models.OnChainMetrics, error) {
	return &models.OnChainMetrics{
		UserAddress:         address,
		WalletAge:           365,
		TotalTransactions:   100,
		AvgTransactionValue: 500,
		DeFiInteractions:    25,
		BorrowingHistory:    10,
		RepaymentHistory:    10,
		LiquidationEvents:   0,
		CollateralValue:     5000,
		LastActivity:        time.Now(),
	}, nil
}

func (m *mockOnChainAggregator) HealthCheck(ctx context.Context) error {
	return nil
}

func (m *mockOnChainAggregator) Close() {}

// Mock off-chain aggregator for testing
type mockOffChainAggregator struct{}

func (m *mockOffChainAggregator) FetchMetrics(ctx context.Context, userID, address string) (*models.OffChainMetrics, error) {
	return &models.OffChainMetrics{
		UserAddress:            address,
		TraditionalCreditScore: 720,
		BankAccountHistory:     85,
		IncomeVerified:         true,
		IncomeLevel:            "medium",
		EmploymentStatus:       "full-time",
		DebtToIncomeRatio:      0.30,
		DataSource:             "mock",
		LastVerified:           time.Now(),
	}, nil
}

func (m *mockOffChainAggregator) HealthCheck(ctx context.Context) error {
	return nil
}

// Mock blockchain client for testing
type mockBlockchainClient struct{}

func (m *mockBlockchainClient) UpdateCreditScore(ctx context.Context, address string, score uint16, confidence uint8, dataHash string) (interface{}, error) {
	// Return nil to simulate no actual blockchain interaction
	return nil, nil
}

func (m *mockBlockchainClient) HealthCheck(ctx context.Context) error {
	return nil
}

func setupTestService(t *testing.T) (*OracleService, *gorm.DB) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	// Auto-migrate models
	err = db.AutoMigrate(
		&models.CreditScore{},
		&models.ScoreHistory{},
		&models.OnChainMetrics{},
		&models.OffChainMetrics{},
		&models.OracleUpdate{},
	)
	if err != nil {
		t.Fatalf("Failed to migrate test database: %v", err)
	}

	repo := repository.NewScoreRepository(db)
	engine := scoring.NewEngine()
	onChainAgg := &mockOnChainAggregator{}
	offChainAgg := &mockOffChainAggregator{}
	blockchainClient := &mockBlockchainClient{}

	service := NewOracleService(repo, engine, onChainAgg, offChainAgg, blockchainClient)

	return service, db
}

func TestCalculateAndUpdateScore(t *testing.T) {
	service, _ := setupTestService(t)
	ctx := context.Background()

	address := "0x1234567890123456789012345678901234567890"
	userID := "user123"

	// Calculate score for the first time
	score, err := service.CalculateAndUpdateScore(ctx, address, userID)
	if err != nil {
		t.Fatalf("Failed to calculate score: %v", err)
	}

	if score == nil {
		t.Fatal("Expected score but got nil")
	}

	if score.Score < 300 || score.Score > 850 {
		t.Errorf("Score %d is outside valid range [300-850]", score.Score)
	}

	if score.UpdateCount != 1 {
		t.Errorf("Expected update count 1, got %d", score.UpdateCount)
	}

	if !score.IsActive {
		t.Error("Score should be active")
	}

	// Update score again
	score2, err := service.CalculateAndUpdateScore(ctx, address, userID)
	if err != nil {
		t.Fatalf("Failed to update score: %v", err)
	}

	if score2.UpdateCount != 2 {
		t.Errorf("Expected update count 2, got %d", score2.UpdateCount)
	}

	if score2.ID != score.ID {
		t.Error("Score ID should remain the same on update")
	}
}

func TestGetScore(t *testing.T) {
	service, _ := setupTestService(t)
	ctx := context.Background()

	address := "0x1234567890123456789012345678901234567890"
	userID := "user123"

	// Calculate and save score
	_, err := service.CalculateAndUpdateScore(ctx, address, userID)
	if err != nil {
		t.Fatalf("Failed to calculate score: %v", err)
	}

	// Retrieve score
	retrieved, err := service.GetScore(ctx, address)
	if err != nil {
		t.Fatalf("Failed to get score: %v", err)
	}

	if retrieved == nil {
		t.Fatal("Expected to retrieve score but got nil")
	}

	if retrieved.UserAddress != address {
		t.Errorf("Expected address %s, got %s", address, retrieved.UserAddress)
	}
}

func TestGetScoreNotFound(t *testing.T) {
	service, _ := setupTestService(t)
	ctx := context.Background()

	score, err := service.GetScore(ctx, "0xNonExistent")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if score != nil {
		t.Error("Expected nil score for non-existent address")
	}
}

func TestGetScoreHistory(t *testing.T) {
	service, _ := setupTestService(t)
	ctx := context.Background()

	address := "0x1234567890123456789012345678901234567890"
	userID := "user123"

	// Create multiple score updates
	for i := 0; i < 3; i++ {
		_, err := service.CalculateAndUpdateScore(ctx, address, userID)
		if err != nil {
			t.Fatalf("Failed to calculate score: %v", err)
		}
		time.Sleep(10 * time.Millisecond) // Ensure different timestamps
	}

	// Get history
	history, err := service.GetScoreHistory(ctx, address, 10)
	if err != nil {
		t.Fatalf("Failed to get history: %v", err)
	}

	if len(history) != 3 {
		t.Errorf("Expected 3 history entries, got %d", len(history))
	}
}

func TestPublishScoreToBlockchain(t *testing.T) {
	service, _ := setupTestService(t)
	ctx := context.Background()

	address := "0x1234567890123456789012345678901234567890"
	userID := "user123"

	// Calculate score first
	_, err := service.CalculateAndUpdateScore(ctx, address, userID)
	if err != nil {
		t.Fatalf("Failed to calculate score: %v", err)
	}

	// Publish to blockchain
	err = service.PublishScoreToBlockchain(ctx, address)
	if err != nil {
		t.Fatalf("Failed to publish score: %v", err)
	}

	// Note: In a real test, you would verify the blockchain interaction
	// For now, we just verify it doesn't error
}

func TestPublishScoreNotFound(t *testing.T) {
	service, _ := setupTestService(t)
	ctx := context.Background()

	err := service.PublishScoreToBlockchain(ctx, "0xNonExistent")
	if err == nil {
		t.Error("Expected error when publishing non-existent score")
	}
}

func TestProcessScheduledUpdates(t *testing.T) {
	service, db := setupTestService(t)
	ctx := context.Background()

	// Create scores that are overdue for update
	overdueAddresses := []string{"0x1111", "0x2222", "0x3333"}

	for _, addr := range overdueAddresses {
		score := &models.CreditScore{
			UserAddress:   addr,
			Score:         700,
			Confidence:    80,
			DataHash:      "hash",
			LastUpdated:   time.Now().Add(-31 * 24 * time.Hour),
			NextUpdateDue: time.Now().Add(-1 * 24 * time.Hour), // Overdue
			IsActive:      true,
		}

		if err := db.Create(score).Error; err != nil {
			t.Fatalf("Failed to create test score: %v", err)
		}
	}

	// Process updates
	err := service.ProcessScheduledUpdates(ctx, 10)
	if err != nil {
		t.Fatalf("Failed to process scheduled updates: %v", err)
	}

	// Verify scores were updated
	for _, addr := range overdueAddresses {
		score, err := service.GetScore(ctx, addr)
		if err != nil {
			t.Fatalf("Failed to get updated score: %v", err)
		}

		if score.UpdateCount < 2 {
			t.Errorf("Expected score to be updated, update count: %d", score.UpdateCount)
		}
	}
}

func TestGetStats(t *testing.T) {
	service, _ := setupTestService(t)
	ctx := context.Background()

	// Create test data
	addresses := []string{"0x1111", "0x2222"}
	for _, addr := range addresses {
		_, err := service.CalculateAndUpdateScore(ctx, addr, "user"+addr)
		if err != nil {
			t.Fatalf("Failed to create test score: %v", err)
		}
	}

	// Get stats
	stats, err := service.GetStats(ctx)
	if err != nil {
		t.Fatalf("Failed to get stats: %v", err)
	}

	if stats["total_active_scores"] == nil {
		t.Error("Expected total_active_scores in stats")
	}

	totalScores := stats["total_active_scores"].(int64)
	if totalScores != 2 {
		t.Errorf("Expected 2 active scores, got %d", totalScores)
	}
}

func TestHealthCheck(t *testing.T) {
	service, _ := setupTestService(t)
	ctx := context.Background()

	health := service.HealthCheck(ctx)

	expectedComponents := []string{"onchain_aggregator", "offchain_aggregator", "blockchain_client"}

	for _, component := range expectedComponents {
		if status, exists := health[component]; !exists {
			t.Errorf("Expected %s in health check results", component)
		} else if !status {
			t.Errorf("Expected %s to be healthy", component)
		}
	}
}

func TestCalculateScoreWithOnChainOnly(t *testing.T) {
	// Create service with nil off-chain aggregator
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	db.AutoMigrate(
		&models.CreditScore{},
		&models.ScoreHistory{},
		&models.OnChainMetrics{},
		&models.OffChainMetrics{},
	)

	repo := repository.NewScoreRepository(db)
	engine := scoring.NewEngine()
	onChainAgg := &mockOnChainAggregator{}

	// Off-chain aggregator that returns error
	type failingOffChainAgg struct{}
	offChainAgg := &failingOffChainAgg{}

	service := &OracleService{
		repo:          repo,
		scoringEngine: engine,
		onChainAgg:    onChainAgg,
		offChainAgg:   aggregator.NewOffChainAggregator("", "", ""),
	}

	ctx := context.Background()
	address := "0x1234567890123456789012345678901234567890"

	// Should still work with on-chain data only
	score, err := service.CalculateAndUpdateScore(ctx, address, "user123")
	if err != nil {
		t.Fatalf("Should calculate score with on-chain data only: %v", err)
	}

	if score == nil {
		t.Fatal("Expected score but got nil")
	}

	if score.Score < 300 || score.Score > 850 {
		t.Errorf("Score %d is outside valid range", score.Score)
	}
}

func TestConcurrentScoreUpdates(t *testing.T) {
	service, _ := setupTestService(t)
	ctx := context.Background()

	address := "0x1234567890123456789012345678901234567890"
	userID := "user123"

	// Concurrent updates
	done := make(chan bool, 5)

	for i := 0; i < 5; i++ {
		go func() {
			_, err := service.CalculateAndUpdateScore(ctx, address, userID)
			if err != nil {
				t.Errorf("Concurrent update failed: %v", err)
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 5; i++ {
		<-done
	}

	// Verify final score
	score, err := service.GetScore(ctx, address)
	if err != nil {
		t.Fatalf("Failed to get final score: %v", err)
	}

	if score.UpdateCount < 1 {
		t.Error("Score should have been updated at least once")
	}
}
