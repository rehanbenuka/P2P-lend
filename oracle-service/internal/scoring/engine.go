package scoring

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"github.com/yourusername/p2p-lend/oracle-service/internal/models"
)

// Scoring weights based on architecture doc
const (
	OnChainWeight  = 0.40  // 40%
	OffChainWeight = 0.40  // 40%
	HybridWeight   = 0.20  // 20%

	MinScore = 300
	MaxScore = 850
)

// Engine handles credit score calculations
type Engine struct{}

// NewEngine creates a new scoring engine
func NewEngine() *Engine {
	return &Engine{}
}

// CalculateScore computes the final credit score
func (e *Engine) CalculateScore(
	onChain *models.OnChainMetrics,
	offChain *models.OffChainMetrics,
) (*models.CreditScore, error) {

	// Calculate component scores
	onChainScore := e.calculateOnChainScore(onChain)
	offChainScore := e.calculateOffChainScore(offChain)
	hybridScore := e.calculateHybridScore(onChain, offChain)

	// Calculate weighted final score
	finalScore := uint16(
		float64(onChainScore)*OnChainWeight +
		float64(offChainScore)*OffChainWeight +
		float64(hybridScore)*HybridWeight,
	)

	// Ensure score is within valid range
	if finalScore < MinScore {
		finalScore = MinScore
	}
	if finalScore > MaxScore {
		finalScore = MaxScore
	}

	// Calculate confidence level
	confidence := e.calculateConfidence(onChain, offChain)

	// Generate data hash for integrity
	dataHash := e.generateDataHash(onChain, offChain, finalScore)

	score := &models.CreditScore{
		Score:          finalScore,
		OnChainScore:   onChainScore,
		OffChainScore:  offChainScore,
		HybridScore:    hybridScore,
		Confidence:     confidence,
		DataHash:       dataHash,
		LastUpdated:    time.Now(),
		NextUpdateDue:  time.Now().Add(30 * 24 * time.Hour), // 30 days
		IsActive:       true,
	}

	return score, nil
}

// calculateOnChainScore computes score from on-chain metrics (40% weight)
func (e *Engine) calculateOnChainScore(metrics *models.OnChainMetrics) uint16 {
	if metrics == nil {
		return MinScore
	}

	var score float64 = 0

	// Wallet age (25% of on-chain score)
	walletAgeScore := e.scoreWalletAge(metrics.WalletAge)
	score += walletAgeScore * 0.25

	// Transaction activity (20%)
	activityScore := e.scoreTransactionActivity(
		metrics.TotalTransactions,
		metrics.AvgTransactionValue,
	)
	score += activityScore * 0.20

	// DeFi interactions (15%)
	defiScore := e.scoreDeFiActivity(metrics.DeFiInteractions)
	score += defiScore * 0.15

	// Borrowing/Repayment history (30%)
	borrowingScore := e.scoreBorrowingHistory(
		metrics.BorrowingHistory,
		metrics.RepaymentHistory,
		metrics.LiquidationEvents,
	)
	score += borrowingScore * 0.30

	// Collateral holdings (10%)
	collateralScore := e.scoreCollateral(metrics.CollateralValue)
	score += collateralScore * 0.10

	// Convert to 300-850 range
	finalScore := MinScore + uint16(score*float64(MaxScore-MinScore))

	return finalScore
}

// calculateOffChainScore computes score from off-chain data (40% weight)
func (e *Engine) calculateOffChainScore(metrics *models.OffChainMetrics) uint16 {
	if metrics == nil {
		return MinScore
	}

	var score float64 = 0

	// Traditional credit score (50% of off-chain score)
	if metrics.TraditionalCreditScore > 0 {
		traditionalScore := float64(metrics.TraditionalCreditScore-MinScore) / float64(MaxScore-MinScore)
		score += traditionalScore * 0.50
	}

	// Bank account history (20%)
	bankScore := float64(metrics.BankAccountHistory) / 100.0
	score += bankScore * 0.20

	// Income verification (15%)
	incomeScore := e.scoreIncome(metrics.IncomeVerified, metrics.IncomeLevel)
	score += incomeScore * 0.15

	// Debt-to-income ratio (15%)
	dtiScore := e.scoreDTI(metrics.DebtToIncomeRatio)
	score += dtiScore * 0.15

	// Convert to 300-850 range
	finalScore := MinScore + uint16(score*float64(MaxScore-MinScore))

	return finalScore
}

// calculateHybridScore combines cross-chain and social metrics (20% weight)
func (e *Engine) calculateHybridScore(
	onChain *models.OnChainMetrics,
	offChain *models.OffChainMetrics,
) uint16 {
	var score float64 = 0

	// Cross-verification bonus
	if onChain != nil && offChain != nil {
		// Bonus if both on-chain and off-chain data are strong
		if onChain.RepaymentHistory > 5 && offChain.IncomeVerified {
			score += 0.30
		}

		// Activity recency bonus
		if time.Since(onChain.LastActivity) < 30*24*time.Hour {
			score += 0.20
		}

		// Collateral + income verification bonus
		if onChain.CollateralValue > 1000 && offChain.IncomeVerified {
			score += 0.25
		}

		// Employment stability bonus
		if offChain.EmploymentStatus == "full-time" || offChain.EmploymentStatus == "self-employed" {
			score += 0.25
		}
	}

	// Normalize to 0-1
	if score > 1.0 {
		score = 1.0
	}

	// Convert to 300-850 range
	finalScore := MinScore + uint16(score*float64(MaxScore-MinScore))

	return finalScore
}

// calculateConfidence determines confidence level (0-100)
func (e *Engine) calculateConfidence(
	onChain *models.OnChainMetrics,
	offChain *models.OffChainMetrics,
) uint8 {
	confidence := 0

	if onChain != nil {
		// Data recency
		if time.Since(onChain.LastActivity) < 7*24*time.Hour {
			confidence += 20
		} else if time.Since(onChain.LastActivity) < 30*24*time.Hour {
			confidence += 10
		}

		// Data completeness
		if onChain.TotalTransactions > 10 {
			confidence += 15
		}
		if onChain.BorrowingHistory > 0 {
			confidence += 15
		}
	}

	if offChain != nil {
		// Traditional credit score available
		if offChain.TraditionalCreditScore > 0 {
			confidence += 25
		}

		// Verification status
		if offChain.IncomeVerified {
			confidence += 15
		}

		// Data freshness
		if time.Since(offChain.LastVerified) < 30*24*time.Hour {
			confidence += 10
		}
	}

	if confidence > 100 {
		confidence = 100
	}

	return uint8(confidence)
}

// Helper scoring functions

func (e *Engine) scoreWalletAge(ageInDays uint32) float64 {
	// Score increases with wallet age, maxing at 2 years
	if ageInDays >= 730 {
		return 1.0
	}
	return float64(ageInDays) / 730.0
}

func (e *Engine) scoreTransactionActivity(txCount uint32, avgValue float64) float64 {
	// Higher transaction count and value indicates more activity
	txScore := math.Min(float64(txCount)/100.0, 1.0) * 0.6
	valueScore := math.Min(avgValue/1000.0, 1.0) * 0.4
	return txScore + valueScore
}

func (e *Engine) scoreDeFiActivity(interactions uint32) float64 {
	// More DeFi interactions = better score
	return math.Min(float64(interactions)/50.0, 1.0)
}

func (e *Engine) scoreBorrowingHistory(borrowed, repaid, liquidations uint32) float64 {
	if borrowed == 0 {
		return 0.5 // Neutral score for no history
	}

	// Repayment ratio
	repaymentRatio := float64(repaid) / float64(borrowed)

	// Penalize liquidations heavily
	liquidationPenalty := float64(liquidations) * 0.2

	score := repaymentRatio - liquidationPenalty

	if score < 0 {
		score = 0
	}
	if score > 1.0 {
		score = 1.0
	}

	return score
}

func (e *Engine) scoreCollateral(value float64) float64 {
	// Higher collateral value = better score
	return math.Min(value/10000.0, 1.0)
}

func (e *Engine) scoreIncome(verified bool, level string) float64 {
	if !verified {
		return 0.3
	}

	switch level {
	case "high":
		return 1.0
	case "medium":
		return 0.7
	case "low":
		return 0.5
	default:
		return 0.3
	}
}

func (e *Engine) scoreDTI(ratio float64) float64 {
	// Lower debt-to-income is better
	// Ideal DTI is below 0.36 (36%)
	if ratio <= 0.36 {
		return 1.0
	} else if ratio <= 0.43 {
		return 0.7
	} else if ratio <= 0.50 {
		return 0.4
	}
	return 0.2
}

// generateDataHash creates a hash of the input data for integrity verification
func (e *Engine) generateDataHash(
	onChain *models.OnChainMetrics,
	offChain *models.OffChainMetrics,
	score uint16,
) string {
	data := struct {
		OnChain   *models.OnChainMetrics
		OffChain  *models.OffChainMetrics
		Score     uint16
		Timestamp time.Time
	}{
		OnChain:   onChain,
		OffChain:  offChain,
		Score:     score,
		Timestamp: time.Now(),
	}

	jsonData, _ := json.Marshal(data)
	hash := sha256.Sum256(jsonData)
	return hex.EncodeToString(hash[:])
}

// ValidateScore checks if a score is within valid range
func (e *Engine) ValidateScore(score uint16) error {
	if score < MinScore || score > MaxScore {
		return fmt.Errorf("score %d is outside valid range [%d-%d]", score, MinScore, MaxScore)
	}
	return nil
}
