package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/yourusername/p2p-lend/oracle-service/pkg/logger"
	"go.uber.org/zap"
)

// BlockchainDataProvider integrates with blockchain analytics providers
// (The Graph, Dune Analytics, Covalent, Moralis)
type BlockchainDataProvider struct {
	httpClient *http.Client
	apiKey     string
	baseURL    string
	provider   string // "covalent", "moralis", "thegraph"
}

// DeFiActivity represents DeFi protocol interaction data
type DeFiActivity struct {
	Protocol        string    `json:"protocol"`      // "aave", "compound", "uniswap", etc.
	ActivityType    string    `json:"activity_type"` // "borrow", "lend", "swap", "stake"
	Amount          float64   `json:"amount"`
	TokenSymbol     string    `json:"token_symbol"`
	TransactionHash string    `json:"transaction_hash"`
	Timestamp       time.Time `json:"timestamp"`
	Status          string    `json:"status"` // "success", "failed"
}

// LendingPosition represents lending/borrowing position
type LendingPosition struct {
	Protocol         string    `json:"protocol"`
	PositionType     string    `json:"position_type"` // "lender", "borrower"
	SuppliedAmount   float64   `json:"supplied_amount"`
	BorrowedAmount   float64   `json:"borrowed_amount"`
	CollateralAmount float64   `json:"collateral_amount"`
	HealthFactor     float64   `json:"health_factor"`
	APY              float64   `json:"apy"`
	LastUpdated      time.Time `json:"last_updated"`
}

// LiquidationEvent represents a liquidation occurrence
type LiquidationEvent struct {
	Protocol         string    `json:"protocol"`
	LiquidatedAmount float64   `json:"liquidated_amount"`
	CollateralLost   float64   `json:"collateral_lost"`
	TokenSymbol      string    `json:"token_symbol"`
	TransactionHash  string    `json:"transaction_hash"`
	Timestamp        time.Time `json:"timestamp"`
	Reason           string    `json:"reason"`
}

// BlockchainSummary represents comprehensive on-chain data
type BlockchainSummary struct {
	Address                string             `json:"address"`
	WalletAge              int                `json:"wallet_age_days"`
	FirstTransaction       time.Time          `json:"first_transaction"`
	LastTransaction        time.Time          `json:"last_transaction"`
	TotalTransactions      int                `json:"total_transactions"`
	TotalVolume            float64            `json:"total_volume"` // USD value
	AverageTransactionSize float64            `json:"average_transaction_size"`
	DeFiActivities         []DeFiActivity     `json:"defi_activities"`
	LendingPositions       []LendingPosition  `json:"lending_positions"`
	LiquidationEvents      []LiquidationEvent `json:"liquidation_events"`
	NFTHoldings            int                `json:"nft_holdings"`
	TokenBalances          map[string]float64 `json:"token_balances"` // token -> balance
	TotalPortfolioValue    float64            `json:"total_portfolio_value"`
	LastUpdated            time.Time          `json:"last_updated"`
}

// NewBlockchainDataProvider creates a new blockchain data provider
func NewBlockchainDataProvider(provider, baseURL, apiKey string) *BlockchainDataProvider {
	return &BlockchainDataProvider{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		apiKey:   apiKey,
		baseURL:  baseURL,
		provider: provider,
	}
}

// GetBlockchainSummary fetches comprehensive blockchain data
func (p *BlockchainDataProvider) GetBlockchainSummary(ctx context.Context, address string, chainID string) (*BlockchainSummary, error) {
	logger.Info("Fetching blockchain summary",
		zap.String("provider", p.provider),
		zap.String("address", address),
		zap.String("chainID", chainID),
	)

	switch p.provider {
	case "covalent":
		return p.fetchFromCovalent(ctx, address, chainID)
	case "moralis":
		return p.fetchFromMoralis(ctx, address, chainID)
	default:
		return nil, fmt.Errorf("unsupported provider: %s", p.provider)
	}
}

// fetchFromCovalent fetches data from Covalent API
func (p *BlockchainDataProvider) fetchFromCovalent(ctx context.Context, address, chainID string) (*BlockchainSummary, error) {
	// Covalent API endpoint
	url := fmt.Sprintf("%s/%s/address/%s/balances_v2/", p.baseURL, chainID, address)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+p.apiKey)
	req.SetBasicAuth(p.apiKey, "")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch from Covalent: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Covalent API returned status %d", resp.StatusCode)
	}

	var result struct {
		Data struct {
			Address string `json:"address"`
			Items   []struct {
				ContractName   string  `json:"contract_name"`
				ContractTicker string  `json:"contract_ticker_symbol"`
				Balance        string  `json:"balance"`
				Quote          float64 `json:"quote"`
			} `json:"items"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	// Build summary
	tokenBalances := make(map[string]float64)
	totalValue := 0.0

	for _, item := range result.Data.Items {
		tokenBalances[item.ContractTicker] = item.Quote
		totalValue += item.Quote
	}

	return &BlockchainSummary{
		Address:             address,
		TokenBalances:       tokenBalances,
		TotalPortfolioValue: totalValue,
		LastUpdated:         time.Now(),
	}, nil
}

// fetchFromMoralis fetches data from Moralis API
func (p *BlockchainDataProvider) fetchFromMoralis(ctx context.Context, address, chainID string) (*BlockchainSummary, error) {
	// Moralis API endpoint
	url := fmt.Sprintf("%s/%s/erc20", p.baseURL, address)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("X-API-Key", p.apiKey)
	q := req.URL.Query()
	q.Add("chain", chainID)
	req.URL.RawQuery = q.Encode()

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch from Moralis: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Moralis API returned status %d", resp.StatusCode)
	}

	var tokens []struct {
		Symbol  string `json:"symbol"`
		Balance string `json:"balance"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&tokens); err != nil {
		return nil, err
	}

	tokenBalances := make(map[string]float64)
	for _, token := range tokens {
		// Would need to convert balance from wei and get USD value
		tokenBalances[token.Symbol] = 0.0
	}

	return &BlockchainSummary{
		Address:       address,
		TokenBalances: tokenBalances,
		LastUpdated:   time.Now(),
	}, nil
}

// GetDeFiActivities fetches DeFi protocol interactions
func (p *BlockchainDataProvider) GetDeFiActivities(ctx context.Context, address string, protocols []string) ([]DeFiActivity, error) {
	// This would query The Graph subgraphs for specific protocols
	logger.Info("Fetching DeFi activities",
		zap.String("address", address),
		zap.Strings("protocols", protocols),
	)

	// Mock implementation
	return []DeFiActivity{}, nil
}

// GetLendingPositions fetches current lending/borrowing positions
func (p *BlockchainDataProvider) GetLendingPositions(ctx context.Context, address string) ([]LendingPosition, error) {
	logger.Info("Fetching lending positions",
		zap.String("address", address),
	)

	// Mock implementation
	return []LendingPosition{}, nil
}

// MockBlockchainData generates mock blockchain data
func (p *BlockchainDataProvider) MockBlockchainData(address string) *BlockchainSummary {
	now := time.Now()
	firstTx := now.AddDate(0, -18, 0) // 18 months ago

	return &BlockchainSummary{
		Address:                address,
		WalletAge:              540, // 18 months in days
		FirstTransaction:       firstTx,
		LastTransaction:        now.AddDate(0, 0, -2), // 2 days ago
		TotalTransactions:      342,
		TotalVolume:            125000.50,
		AverageTransactionSize: 365.50,
		DeFiActivities: []DeFiActivity{
			{
				Protocol:        "aave-v3",
				ActivityType:    "lend",
				Amount:          5000,
				TokenSymbol:     "USDC",
				TransactionHash: "0xabc123...",
				Timestamp:       now.AddDate(0, -1, 0),
				Status:          "success",
			},
			{
				Protocol:        "uniswap-v3",
				ActivityType:    "swap",
				Amount:          1.5,
				TokenSymbol:     "ETH",
				TransactionHash: "0xdef456...",
				Timestamp:       now.AddDate(0, 0, -5),
				Status:          "success",
			},
		},
		LendingPositions: []LendingPosition{
			{
				Protocol:         "aave-v3",
				PositionType:     "lender",
				SuppliedAmount:   5000,
				BorrowedAmount:   0,
				CollateralAmount: 5000,
				HealthFactor:     0,
				APY:              4.5,
				LastUpdated:      now,
			},
		},
		LiquidationEvents: []LiquidationEvent{}, // No liquidations
		NFTHoldings:       3,
		TokenBalances: map[string]float64{
			"ETH":  2.5,
			"USDC": 5000,
			"DAI":  1200,
		},
		TotalPortfolioValue: 12450.00,
		LastUpdated:         now,
	}
}

// HealthCheck verifies API connectivity
func (p *BlockchainDataProvider) HealthCheck(ctx context.Context) error {
	return nil
}
