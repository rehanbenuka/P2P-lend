package models

import (
	"time"
)

// CreditScore represents a user's credit score data
type CreditScore struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	UserAddress     string    `gorm:"uniqueIndex;not null" json:"user_address"`
	Score           uint16    `gorm:"not null" json:"score"`           // 300-850 range
	Confidence      uint8     `gorm:"not null" json:"confidence"`      // 0-100
	OnChainScore    uint16    `json:"on_chain_score"`                  // Component scores
	OffChainScore   uint16    `json:"off_chain_score"`
	HybridScore     uint16    `json:"hybrid_score"`
	DataHash        string    `gorm:"not null" json:"data_hash"`       // Hash of source data
	LastUpdated     time.Time `gorm:"not null" json:"last_updated"`
	NextUpdateDue   time.Time `json:"next_update_due"`
	UpdateCount     uint32    `json:"update_count"`
	IsActive        bool      `gorm:"default:true" json:"is_active"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// ScoreHistory tracks historical credit scores
type ScoreHistory struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	UserAddress string    `gorm:"index;not null" json:"user_address"`
	Score       uint16    `gorm:"not null" json:"score"`
	Confidence  uint8     `gorm:"not null" json:"confidence"`
	DataHash    string    `gorm:"not null" json:"data_hash"`
	Timestamp   time.Time `gorm:"not null;index" json:"timestamp"`
	CreatedAt   time.Time `json:"created_at"`
}

// OnChainMetrics stores on-chain activity data
type OnChainMetrics struct {
	ID                  uint      `gorm:"primaryKey" json:"id"`
	UserAddress         string    `gorm:"uniqueIndex;not null" json:"user_address"`
	WalletAge           uint32    `json:"wallet_age"`              // Days since first transaction
	TotalTransactions   uint32    `json:"total_transactions"`
	AvgTransactionValue float64   `json:"avg_transaction_value"`
	DeFiInteractions    uint32    `json:"defi_interactions"`
	BorrowingHistory    uint32    `json:"borrowing_history"`
	RepaymentHistory    uint32    `json:"repayment_history"`
	LiquidationEvents   uint32    `json:"liquidation_events"`
	CollateralValue     float64   `json:"collateral_value"`
	LastActivity        time.Time `json:"last_activity"`
	CreatedAt           time.Time `json:"created_at"`
	UpdatedAt           time.Time `json:"updated_at"`
}

// OffChainMetrics stores off-chain/external data
type OffChainMetrics struct {
	ID                    uint      `gorm:"primaryKey" json:"id"`
	UserAddress           string    `gorm:"uniqueIndex;not null" json:"user_address"`
	TraditionalCreditScore uint16   `json:"traditional_credit_score"` // 300-850
	BankAccountHistory    uint8     `json:"bank_account_history"`     // Score 0-100
	IncomeVerified        bool      `json:"income_verified"`
	IncomeLevel           string    `json:"income_level"`             // low/medium/high
	EmploymentStatus      string    `json:"employment_status"`
	DebtToIncomeRatio     float64   `json:"debt_to_income_ratio"`
	DataSource            string    `json:"data_source"`
	LastVerified          time.Time `json:"last_verified"`
	CreatedAt             time.Time `json:"created_at"`
	UpdatedAt             time.Time `json:"updated_at"`
}

// OracleUpdate tracks oracle updates sent to blockchain
type OracleUpdate struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	UserAddress     string    `gorm:"index;not null" json:"user_address"`
	Score           uint16    `gorm:"not null" json:"score"`
	Confidence      uint8     `gorm:"not null" json:"confidence"`
	DataHash        string    `gorm:"not null" json:"data_hash"`
	TxHash          string    `gorm:"uniqueIndex" json:"tx_hash"`
	BlockNumber     uint64    `json:"block_number"`
	Status          string    `gorm:"default:'pending'" json:"status"` // pending/confirmed/failed
	GasUsed         uint64    `json:"gas_used"`
	ErrorMessage    string    `json:"error_message"`
	RetryCount      uint8     `json:"retry_count"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}
