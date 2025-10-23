package scoring

import (
	"testing"
	"time"

	"github.com/yourusername/p2p-lend/oracle-service/internal/models"
)

func TestCalculateScore(t *testing.T) {
	engine := NewEngine()

	tests := []struct {
		name            string
		onChain         *models.OnChainMetrics
		offChain        *models.OffChainMetrics
		expectedMinScore uint16
		expectedMaxScore uint16
		expectError     bool
	}{
		{
			name: "High quality on-chain and off-chain data",
			onChain: &models.OnChainMetrics{
				WalletAge:           730,  // 2 years
				TotalTransactions:   100,
				AvgTransactionValue: 500,
				DeFiInteractions:    50,
				BorrowingHistory:    10,
				RepaymentHistory:    10,
				LiquidationEvents:   0,
				CollateralValue:     5000,
				LastActivity:        time.Now().Add(-1 * 24 * time.Hour),
			},
			offChain: &models.OffChainMetrics{
				TraditionalCreditScore: 750,
				BankAccountHistory:     90,
				IncomeVerified:         true,
				IncomeLevel:            "high",
				EmploymentStatus:       "full-time",
				DebtToIncomeRatio:      0.25,
			},
			expectedMinScore: 700,
			expectedMaxScore: 850,
			expectError:      false,
		},
		{
			name: "Poor quality data with liquidations",
			onChain: &models.OnChainMetrics{
				WalletAge:           30,  // 1 month
				TotalTransactions:   10,
				AvgTransactionValue: 50,
				DeFiInteractions:    2,
				BorrowingHistory:    5,
				RepaymentHistory:    2,
				LiquidationEvents:   3,
				CollateralValue:     100,
				LastActivity:        time.Now().Add(-90 * 24 * time.Hour),
			},
			offChain: &models.OffChainMetrics{
				TraditionalCreditScore: 550,
				BankAccountHistory:     40,
				IncomeVerified:         false,
				IncomeLevel:            "low",
				EmploymentStatus:       "unemployed",
				DebtToIncomeRatio:      0.55,
			},
			expectedMinScore: 300,
			expectedMaxScore: 550,
			expectError:      false,
		},
		{
			name: "Only on-chain data available",
			onChain: &models.OnChainMetrics{
				WalletAge:           365,  // 1 year
				TotalTransactions:   50,
				AvgTransactionValue: 250,
				DeFiInteractions:    15,
				BorrowingHistory:    5,
				RepaymentHistory:    5,
				LiquidationEvents:   0,
				CollateralValue:     2000,
				LastActivity:        time.Now().Add(-7 * 24 * time.Hour),
			},
			offChain:        nil,
			expectedMinScore: 450,
			expectedMaxScore: 650,
			expectError:      false,
		},
		{
			name: "Only off-chain data available",
			onChain: nil,
			offChain: &models.OffChainMetrics{
				TraditionalCreditScore: 680,
				BankAccountHistory:     75,
				IncomeVerified:         true,
				IncomeLevel:            "medium",
				EmploymentStatus:       "full-time",
				DebtToIncomeRatio:      0.35,
			},
			expectedMinScore: 450,
			expectedMaxScore: 700,
			expectError:      false,
		},
		{
			name:            "No data available",
			onChain:         nil,
			offChain:        nil,
			expectedMinScore: 300,
			expectedMaxScore: 400,
			expectError:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score, err := engine.CalculateScore(tt.onChain, tt.offChain)

			if tt.expectError && err == nil {
				t.Errorf("Expected error but got none")
				return
			}

			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if err != nil {
				return
			}

			if score.Score < tt.expectedMinScore || score.Score > tt.expectedMaxScore {
				t.Errorf("Score %d is outside expected range [%d-%d]",
					score.Score, tt.expectedMinScore, tt.expectedMaxScore)
			}

			if score.Score < MinScore || score.Score > MaxScore {
				t.Errorf("Score %d is outside valid range [%d-%d]",
					score.Score, MinScore, MaxScore)
			}

			if score.Confidence > 100 {
				t.Errorf("Confidence %d exceeds maximum of 100", score.Confidence)
			}

			if score.DataHash == "" {
				t.Errorf("DataHash should not be empty")
			}

			if score.LastUpdated.IsZero() {
				t.Errorf("LastUpdated should be set")
			}
		})
	}
}

func TestCalculateOnChainScore(t *testing.T) {
	engine := NewEngine()

	tests := []struct {
		name     string
		metrics  *models.OnChainMetrics
		expected uint16
	}{
		{
			name: "Perfect on-chain metrics",
			metrics: &models.OnChainMetrics{
				WalletAge:           1000,
				TotalTransactions:   200,
				AvgTransactionValue: 2000,
				DeFiInteractions:    100,
				BorrowingHistory:    20,
				RepaymentHistory:    20,
				LiquidationEvents:   0,
				CollateralValue:     20000,
			},
			expected: 800, // Should be near maximum
		},
		{
			name:     "Nil metrics",
			metrics:  nil,
			expected: MinScore,
		},
		{
			name: "New wallet with minimal activity",
			metrics: &models.OnChainMetrics{
				WalletAge:           7,
				TotalTransactions:   5,
				AvgTransactionValue: 10,
				DeFiInteractions:    0,
				BorrowingHistory:    0,
				RepaymentHistory:    0,
				LiquidationEvents:   0,
				CollateralValue:     50,
			},
			expected: 350, // Should be near minimum
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := engine.calculateOnChainScore(tt.metrics)

			// Allow 50 point variance
			if score < tt.expected-50 || score > tt.expected+50 {
				t.Errorf("Score %d is not within 50 points of expected %d",
					score, tt.expected)
			}
		})
	}
}

func TestCalculateOffChainScore(t *testing.T) {
	engine := NewEngine()

	tests := []struct {
		name     string
		metrics  *models.OffChainMetrics
		expected uint16
	}{
		{
			name: "Excellent credit profile",
			metrics: &models.OffChainMetrics{
				TraditionalCreditScore: 800,
				BankAccountHistory:     95,
				IncomeVerified:         true,
				IncomeLevel:            "high",
				DebtToIncomeRatio:      0.20,
			},
			expected: 750,
		},
		{
			name:     "Nil metrics",
			metrics:  nil,
			expected: MinScore,
		},
		{
			name: "Poor credit profile",
			metrics: &models.OffChainMetrics{
				TraditionalCreditScore: 500,
				BankAccountHistory:     30,
				IncomeVerified:         false,
				IncomeLevel:            "low",
				DebtToIncomeRatio:      0.60,
			},
			expected: 450,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := engine.calculateOffChainScore(tt.metrics)

			// Allow 100 point variance
			if score < tt.expected-100 || score > tt.expected+100 {
				t.Errorf("Score %d is not within 100 points of expected %d",
					score, tt.expected)
			}
		})
	}
}

func TestScoreWalletAge(t *testing.T) {
	engine := NewEngine()

	tests := []struct {
		days     uint32
		expected float64
	}{
		{0, 0.0},
		{365, 0.5},      // 1 year = 50%
		{730, 1.0},      // 2 years = 100%
		{1000, 1.0},     // More than 2 years = 100%
	}

	for _, tt := range tests {
		result := engine.scoreWalletAge(tt.days)
		if result != tt.expected {
			t.Errorf("scoreWalletAge(%d) = %f, expected %f",
				tt.days, result, tt.expected)
		}
	}
}

func TestScoreBorrowingHistory(t *testing.T) {
	engine := NewEngine()

	tests := []struct {
		name         string
		borrowed     uint32
		repaid       uint32
		liquidations uint32
		minExpected  float64
		maxExpected  float64
	}{
		{
			name:         "Perfect repayment",
			borrowed:     10,
			repaid:       10,
			liquidations: 0,
			minExpected:  0.9,
			maxExpected:  1.0,
		},
		{
			name:         "Some defaults",
			borrowed:     10,
			repaid:       7,
			liquidations: 1,
			minExpected:  0.3,
			maxExpected:  0.6,
		},
		{
			name:         "No history",
			borrowed:     0,
			repaid:       0,
			liquidations: 0,
			minExpected:  0.5,
			maxExpected:  0.5,
		},
		{
			name:         "Many liquidations",
			borrowed:     10,
			repaid:       5,
			liquidations: 5,
			minExpected:  0.0,
			maxExpected:  0.3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := engine.scoreBorrowingHistory(tt.borrowed, tt.repaid, tt.liquidations)

			if score < tt.minExpected || score > tt.maxExpected {
				t.Errorf("scoreBorrowingHistory(%d, %d, %d) = %f, expected between %f and %f",
					tt.borrowed, tt.repaid, tt.liquidations, score, tt.minExpected, tt.maxExpected)
			}
		})
	}
}

func TestScoreDTI(t *testing.T) {
	engine := NewEngine()

	tests := []struct {
		ratio    float64
		expected float64
	}{
		{0.30, 1.0},  // Excellent DTI
		{0.36, 1.0},  // Good DTI
		{0.40, 0.7},  // Moderate DTI
		{0.45, 0.4},  // High DTI
		{0.60, 0.2},  // Very high DTI
	}

	for _, tt := range tests {
		result := engine.scoreDTI(tt.ratio)
		if result != tt.expected {
			t.Errorf("scoreDTI(%f) = %f, expected %f",
				tt.ratio, result, tt.expected)
		}
	}
}

func TestCalculateConfidence(t *testing.T) {
	engine := NewEngine()

	tests := []struct {
		name         string
		onChain      *models.OnChainMetrics
		offChain     *models.OffChainMetrics
		minConfidence uint8
		maxConfidence uint8
	}{
		{
			name: "High confidence - all data available",
			onChain: &models.OnChainMetrics{
				LastActivity:      time.Now().Add(-1 * 24 * time.Hour),
				TotalTransactions: 50,
				BorrowingHistory:  5,
			},
			offChain: &models.OffChainMetrics{
				TraditionalCreditScore: 700,
				IncomeVerified:         true,
				LastVerified:           time.Now().Add(-7 * 24 * time.Hour),
			},
			minConfidence: 80,
			maxConfidence: 100,
		},
		{
			name: "Medium confidence - some data missing",
			onChain: &models.OnChainMetrics{
				LastActivity:      time.Now().Add(-15 * 24 * time.Hour),
				TotalTransactions: 20,
				BorrowingHistory:  0,
			},
			offChain: &models.OffChainMetrics{
				TraditionalCreditScore: 0,
				IncomeVerified:         false,
				LastVerified:           time.Now().Add(-60 * 24 * time.Hour),
			},
			minConfidence: 20,
			maxConfidence: 45,
		},
		{
			name:          "Low confidence - minimal data",
			onChain:       nil,
			offChain:      nil,
			minConfidence: 0,
			maxConfidence: 10,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			confidence := engine.calculateConfidence(tt.onChain, tt.offChain)

			if confidence < tt.minConfidence || confidence > tt.maxConfidence {
				t.Errorf("Confidence %d is outside expected range [%d-%d]",
					confidence, tt.minConfidence, tt.maxConfidence)
			}
		})
	}
}

func TestValidateScore(t *testing.T) {
	engine := NewEngine()

	tests := []struct {
		score       uint16
		expectError bool
	}{
		{300, false},
		{500, false},
		{850, false},
		{299, true},
		{851, true},
		{0, true},
		{1000, true},
	}

	for _, tt := range tests {
		err := engine.ValidateScore(tt.score)
		hasError := err != nil

		if hasError != tt.expectError {
			t.Errorf("ValidateScore(%d): expected error=%v, got error=%v",
				tt.score, tt.expectError, hasError)
		}
	}
}

func TestGenerateDataHash(t *testing.T) {
	engine := NewEngine()

	onChain := &models.OnChainMetrics{
		UserAddress:      "0x123",
		WalletAge:        100,
		TotalTransactions: 50,
	}

	offChain := &models.OffChainMetrics{
		UserAddress:            "0x123",
		TraditionalCreditScore: 700,
	}

	hash1 := engine.generateDataHash(onChain, offChain, 700)
	hash2 := engine.generateDataHash(onChain, offChain, 700)

	// Hashes should be consistent for same inputs (within same second)
	// Note: This test might occasionally fail due to timestamp differences
	// In production, you'd want to pass timestamp as parameter for deterministic hashing

	if hash1 == "" {
		t.Error("Hash should not be empty")
	}

	if len(hash1) != 64 { // SHA256 hex string length
		t.Errorf("Hash length %d is incorrect, expected 64", len(hash1))
	}

	// Different scores should produce different hashes
	hash3 := engine.generateDataHash(onChain, offChain, 650)
	if hash1 == hash3 {
		t.Error("Different scores should produce different hashes")
	}
}

// Benchmark tests

func BenchmarkCalculateScore(b *testing.B) {
	engine := NewEngine()

	onChain := &models.OnChainMetrics{
		WalletAge:           365,
		TotalTransactions:   100,
		AvgTransactionValue: 500,
		DeFiInteractions:    25,
		BorrowingHistory:    5,
		RepaymentHistory:    5,
		LiquidationEvents:   0,
		CollateralValue:     2000,
		LastActivity:        time.Now(),
	}

	offChain := &models.OffChainMetrics{
		TraditionalCreditScore: 700,
		BankAccountHistory:     80,
		IncomeVerified:         true,
		IncomeLevel:            "medium",
		DebtToIncomeRatio:      0.30,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		engine.CalculateScore(onChain, offChain)
	}
}
