package repository

import (
	"context"
	"testing"
	"time"

	"github.com/yourusername/p2p-lend/oracle-service/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
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

	return db
}

func TestCreateCreditScore(t *testing.T) {
	db := setupTestDB(t)
	repo := NewScoreRepository(db)
	ctx := context.Background()

	score := &models.CreditScore{
		UserAddress:     "0x1234567890123456789012345678901234567890",
		Score:           720,
		Confidence:      85,
		OnChainScore:    700,
		OffChainScore:   740,
		HybridScore:     720,
		DataHash:        "hash123",
		LastUpdated:     time.Now(),
		NextUpdateDue:   time.Now().Add(30 * 24 * time.Hour),
		UpdateCount:     1,
		IsActive:        true,
	}

	err := repo.Create(ctx, score)
	if err != nil {
		t.Fatalf("Failed to create credit score: %v", err)
	}

	if score.ID == 0 {
		t.Error("Expected ID to be set after creation")
	}
}

func TestGetByAddress(t *testing.T) {
	db := setupTestDB(t)
	repo := NewScoreRepository(db)
	ctx := context.Background()

	address := "0x1234567890123456789012345678901234567890"

	// Create test score
	testScore := &models.CreditScore{
		UserAddress:   address,
		Score:         720,
		Confidence:    85,
		DataHash:      "hash123",
		LastUpdated:   time.Now(),
		NextUpdateDue: time.Now().Add(30 * 24 * time.Hour),
		IsActive:      true,
	}

	err := repo.Create(ctx, testScore)
	if err != nil {
		t.Fatalf("Failed to create test score: %v", err)
	}

	// Retrieve score
	retrieved, err := repo.GetByAddress(ctx, address)
	if err != nil {
		t.Fatalf("Failed to get credit score: %v", err)
	}

	if retrieved == nil {
		t.Fatal("Expected to retrieve score but got nil")
	}

	if retrieved.UserAddress != address {
		t.Errorf("Expected address %s, got %s", address, retrieved.UserAddress)
	}

	if retrieved.Score != 720 {
		t.Errorf("Expected score 720, got %d", retrieved.Score)
	}
}

func TestGetByAddressNotFound(t *testing.T) {
	db := setupTestDB(t)
	repo := NewScoreRepository(db)
	ctx := context.Background()

	score, err := repo.GetByAddress(ctx, "0xNonExistent")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if score != nil {
		t.Error("Expected nil score for non-existent address")
	}
}

func TestUpdateCreditScore(t *testing.T) {
	db := setupTestDB(t)
	repo := NewScoreRepository(db)
	ctx := context.Background()

	address := "0x1234567890123456789012345678901234567890"

	// Create initial score
	score := &models.CreditScore{
		UserAddress:   address,
		Score:         700,
		Confidence:    80,
		DataHash:      "hash1",
		LastUpdated:   time.Now(),
		NextUpdateDue: time.Now().Add(30 * 24 * time.Hour),
		UpdateCount:   1,
		IsActive:      true,
	}

	err := repo.Create(ctx, score)
	if err != nil {
		t.Fatalf("Failed to create score: %v", err)
	}

	// Update score
	score.Score = 750
	score.Confidence = 90
	score.UpdateCount = 2

	err = repo.Update(ctx, score)
	if err != nil {
		t.Fatalf("Failed to update score: %v", err)
	}

	// Verify update
	updated, err := repo.GetByAddress(ctx, address)
	if err != nil {
		t.Fatalf("Failed to retrieve updated score: %v", err)
	}

	if updated.Score != 750 {
		t.Errorf("Expected score 750, got %d", updated.Score)
	}

	if updated.UpdateCount != 2 {
		t.Errorf("Expected update count 2, got %d", updated.UpdateCount)
	}
}

func TestGetDueForUpdate(t *testing.T) {
	db := setupTestDB(t)
	repo := NewScoreRepository(db)
	ctx := context.Background()

	// Create scores with different update due dates
	scores := []*models.CreditScore{
		{
			UserAddress:   "0x1111",
			Score:         700,
			Confidence:    80,
			DataHash:      "hash1",
			LastUpdated:   time.Now().Add(-31 * 24 * time.Hour),
			NextUpdateDue: time.Now().Add(-1 * 24 * time.Hour), // Overdue
			IsActive:      true,
		},
		{
			UserAddress:   "0x2222",
			Score:         720,
			Confidence:    85,
			DataHash:      "hash2",
			LastUpdated:   time.Now().Add(-25 * 24 * time.Hour),
			NextUpdateDue: time.Now().Add(5 * 24 * time.Hour), // Not due yet
			IsActive:      true,
		},
		{
			UserAddress:   "0x3333",
			Score:         750,
			Confidence:    90,
			DataHash:      "hash3",
			LastUpdated:   time.Now().Add(-30 * 24 * time.Hour),
			NextUpdateDue: time.Now().Add(-2 * time.Hour), // Due
			IsActive:      true,
		},
	}

	for _, s := range scores {
		if err := repo.Create(ctx, s); err != nil {
			t.Fatalf("Failed to create test score: %v", err)
		}
	}

	// Get scores due for update
	due, err := repo.GetDueForUpdate(ctx, 10)
	if err != nil {
		t.Fatalf("Failed to get scores due for update: %v", err)
	}

	if len(due) != 2 {
		t.Errorf("Expected 2 scores due for update, got %d", len(due))
	}
}

func TestCreateAndGetHistory(t *testing.T) {
	db := setupTestDB(t)
	repo := NewScoreRepository(db)
	ctx := context.Background()

	address := "0x1234567890123456789012345678901234567890"

	// Create history entries
	entries := []*models.ScoreHistory{
		{
			UserAddress: address,
			Score:       700,
			Confidence:  80,
			DataHash:    "hash1",
			Timestamp:   time.Now().Add(-60 * 24 * time.Hour),
		},
		{
			UserAddress: address,
			Score:       720,
			Confidence:  85,
			DataHash:    "hash2",
			Timestamp:   time.Now().Add(-30 * 24 * time.Hour),
		},
		{
			UserAddress: address,
			Score:       750,
			Confidence:  90,
			DataHash:    "hash3",
			Timestamp:   time.Now(),
		},
	}

	for _, entry := range entries {
		if err := repo.CreateHistory(ctx, entry); err != nil {
			t.Fatalf("Failed to create history entry: %v", err)
		}
	}

	// Retrieve history
	history, err := repo.GetHistory(ctx, address, 10)
	if err != nil {
		t.Fatalf("Failed to get history: %v", err)
	}

	if len(history) != 3 {
		t.Errorf("Expected 3 history entries, got %d", len(history))
	}

	// Verify order (should be descending by timestamp)
	if history[0].Score != 750 {
		t.Errorf("Expected most recent score 750, got %d", history[0].Score)
	}
}

func TestUpsertOnChainMetrics(t *testing.T) {
	db := setupTestDB(t)
	repo := NewScoreRepository(db)
	ctx := context.Background()

	address := "0x1234567890123456789012345678901234567890"

	// Create metrics
	metrics := &models.OnChainMetrics{
		UserAddress:      address,
		WalletAge:        365,
		TotalTransactions: 50,
		CollateralValue:  1000,
	}

	err := repo.UpsertOnChainMetrics(ctx, metrics)
	if err != nil {
		t.Fatalf("Failed to create metrics: %v", err)
	}

	// Update metrics
	metrics.WalletAge = 400
	metrics.TotalTransactions = 75

	err = repo.UpsertOnChainMetrics(ctx, metrics)
	if err != nil {
		t.Fatalf("Failed to update metrics: %v", err)
	}

	// Verify update
	retrieved, err := repo.GetOnChainMetrics(ctx, address)
	if err != nil {
		t.Fatalf("Failed to retrieve metrics: %v", err)
	}

	if retrieved.WalletAge != 400 {
		t.Errorf("Expected wallet age 400, got %d", retrieved.WalletAge)
	}

	if retrieved.TotalTransactions != 75 {
		t.Errorf("Expected 75 transactions, got %d", retrieved.TotalTransactions)
	}
}

func TestUpsertOffChainMetrics(t *testing.T) {
	db := setupTestDB(t)
	repo := NewScoreRepository(db)
	ctx := context.Background()

	address := "0x1234567890123456789012345678901234567890"

	// Create metrics
	metrics := &models.OffChainMetrics{
		UserAddress:            address,
		TraditionalCreditScore: 700,
		IncomeVerified:         true,
	}

	err := repo.UpsertOffChainMetrics(ctx, metrics)
	if err != nil {
		t.Fatalf("Failed to create metrics: %v", err)
	}

	// Update metrics
	metrics.TraditionalCreditScore = 750
	metrics.IncomeLevel = "high"

	err = repo.UpsertOffChainMetrics(ctx, metrics)
	if err != nil {
		t.Fatalf("Failed to update metrics: %v", err)
	}

	// Verify update
	retrieved, err := repo.GetOffChainMetrics(ctx, address)
	if err != nil {
		t.Fatalf("Failed to retrieve metrics: %v", err)
	}

	if retrieved.TraditionalCreditScore != 750 {
		t.Errorf("Expected credit score 750, got %d", retrieved.TraditionalCreditScore)
	}

	if retrieved.IncomeLevel != "high" {
		t.Errorf("Expected income level 'high', got '%s'", retrieved.IncomeLevel)
	}
}

func TestCreateAndGetOracleUpdate(t *testing.T) {
	db := setupTestDB(t)
	repo := NewScoreRepository(db)
	ctx := context.Background()

	update := &models.OracleUpdate{
		UserAddress: "0x1234",
		Score:       720,
		Confidence:  85,
		DataHash:    "hash123",
		TxHash:      "0xabcdef",
		BlockNumber: 12345,
		Status:      "confirmed",
	}

	err := repo.CreateOracleUpdate(ctx, update)
	if err != nil {
		t.Fatalf("Failed to create oracle update: %v", err)
	}

	// Retrieve by tx hash
	retrieved, err := repo.GetOracleUpdateByTxHash(ctx, "0xabcdef")
	if err != nil {
		t.Fatalf("Failed to retrieve oracle update: %v", err)
	}

	if retrieved == nil {
		t.Fatal("Expected to retrieve update but got nil")
	}

	if retrieved.Score != 720 {
		t.Errorf("Expected score 720, got %d", retrieved.Score)
	}

	if retrieved.Status != "confirmed" {
		t.Errorf("Expected status 'confirmed', got '%s'", retrieved.Status)
	}
}

func TestGetPendingOracleUpdates(t *testing.T) {
	db := setupTestDB(t)
	repo := NewScoreRepository(db)
	ctx := context.Background()

	// Create updates with different statuses
	updates := []*models.OracleUpdate{
		{
			UserAddress: "0x1111",
			Score:       700,
			Confidence:  80,
			DataHash:    "hash1",
			TxHash:      "0xaaa",
			Status:      "pending",
		},
		{
			UserAddress: "0x2222",
			Score:       720,
			Confidence:  85,
			DataHash:    "hash2",
			TxHash:      "0xbbb",
			Status:      "confirmed",
		},
		{
			UserAddress: "0x3333",
			Score:       750,
			Confidence:  90,
			DataHash:    "hash3",
			TxHash:      "0xccc",
			Status:      "pending",
		},
	}

	for _, u := range updates {
		if err := repo.CreateOracleUpdate(ctx, u); err != nil {
			t.Fatalf("Failed to create oracle update: %v", err)
		}
	}

	// Get pending updates
	pending, err := repo.GetPendingOracleUpdates(ctx)
	if err != nil {
		t.Fatalf("Failed to get pending updates: %v", err)
	}

	if len(pending) != 2 {
		t.Errorf("Expected 2 pending updates, got %d", len(pending))
	}
}

func TestGetStats(t *testing.T) {
	db := setupTestDB(t)
	repo := NewScoreRepository(db)
	ctx := context.Background()

	// Create test data
	scores := []*models.CreditScore{
		{
			UserAddress:   "0x1111",
			Score:         700,
			Confidence:    80,
			DataHash:      "hash1",
			LastUpdated:   time.Now(),
			NextUpdateDue: time.Now().Add(-1 * 24 * time.Hour), // Overdue
			IsActive:      true,
		},
		{
			UserAddress:   "0x2222",
			Score:         800,
			Confidence:    90,
			DataHash:      "hash2",
			LastUpdated:   time.Now(),
			NextUpdateDue: time.Now().Add(10 * 24 * time.Hour),
			IsActive:      true,
		},
	}

	for _, s := range scores {
		if err := repo.Create(ctx, s); err != nil {
			t.Fatalf("Failed to create test score: %v", err)
		}
	}

	// Create pending oracle update
	update := &models.OracleUpdate{
		UserAddress: "0x1111",
		Score:       700,
		Confidence:  80,
		DataHash:    "hash1",
		TxHash:      "0xaaa",
		Status:      "pending",
	}
	if err := repo.CreateOracleUpdate(ctx, update); err != nil {
		t.Fatalf("Failed to create oracle update: %v", err)
	}

	// Get stats
	stats, err := repo.GetStats(ctx)
	if err != nil {
		t.Fatalf("Failed to get stats: %v", err)
	}

	if stats["total_active_scores"] != int64(2) {
		t.Errorf("Expected 2 active scores, got %v", stats["total_active_scores"])
	}

	if stats["average_score"] != float64(750) {
		t.Errorf("Expected average score 750, got %v", stats["average_score"])
	}

	if stats["due_for_update"] != int64(1) {
		t.Errorf("Expected 1 score due for update, got %v", stats["due_for_update"])
	}

	if stats["pending_oracle_updates"] != int64(1) {
		t.Errorf("Expected 1 pending oracle update, got %v", stats["pending_oracle_updates"])
	}
}
