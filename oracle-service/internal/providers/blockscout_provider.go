package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/yourusername/p2p-lend/oracle-service/pkg/logger"
	"go.uber.org/zap"
)

// BlockscoutProvider integrates with Blockscout API for blockchain data
type BlockscoutProvider struct {
	httpClient *http.Client
	baseURL    string
	chainName  string // "ethereum", "polygon", "optimism", etc.
}

// BlockscoutAddressInfo represents address information from Blockscout
type BlockscoutAddressInfo struct {
	Hash                string `json:"hash"`
	Balance             string `json:"balance"`
	TransactionsCount   int    `json:"transactions_count"`
	TokenTransfersCount int    `json:"token_transfers_count"`
	GasUsed             string `json:"gas_used"`
	IsContract          bool   `json:"is_contract"`
	IsVerified          bool   `json:"is_verified"`
}

// BlockscoutTransaction represents a transaction from Blockscout
type BlockscoutTransaction struct {
	Hash             string    `json:"hash"`
	BlockNumber      string    `json:"block_number"`
	TimeStamp        string    `json:"timestamp"`
	From             string    `json:"from"`
	To               string    `json:"to"`
	Value            string    `json:"value"`
	Gas              string    `json:"gas"`
	GasPrice         string    `json:"gas_price"`
	GasUsed          string    `json:"gas_used"`
	Status           string    `json:"status"`
	MethodID         string    `json:"method_id"`
	FunctionName     string    `json:"function_name"`
	ConfirmationTime time.Time `json:"confirmation_time"`
}

// BlockscoutTokenBalance represents ERC20 token balance
type BlockscoutTokenBalance struct {
	TokenAddress  string `json:"token_address"`
	TokenName     string `json:"token_name"`
	TokenSymbol   string `json:"token_symbol"`
	TokenDecimals int    `json:"token_decimals"`
	Balance       string `json:"balance"`
	TokenType     string `json:"token_type"` // "ERC-20", "ERC-721", "ERC-1155"
	TokenID       string `json:"token_id"`
	TokenValue    string `json:"token_value"`
}

// BlockscoutInternalTx represents internal transactions (contract calls)
type BlockscoutInternalTx struct {
	TransactionHash string `json:"transaction_hash"`
	BlockNumber     string `json:"block_number"`
	TimeStamp       string `json:"timestamp"`
	From            string `json:"from"`
	To              string `json:"to"`
	Value           string `json:"value"`
	Type            string `json:"type"`
	GasUsed         string `json:"gas_used"`
}

// BlockscoutAnalytics represents aggregated analytics
type BlockscoutAnalytics struct {
	Address                string                   `json:"address"`
	Balance                float64                  `json:"balance_eth"`
	BalanceUSD             float64                  `json:"balance_usd"`
	FirstTransactionDate   time.Time                `json:"first_transaction_date"`
	LastTransactionDate    time.Time                `json:"last_transaction_date"`
	WalletAgeDays          int                      `json:"wallet_age_days"`
	TotalTransactions      int                      `json:"total_transactions"`
	TotalTokenTransfers    int                      `json:"total_token_transfers"`
	TotalInternalTxs       int                      `json:"total_internal_txs"`
	TotalGasUsed           float64                  `json:"total_gas_used"`
	AverageTransactionSize float64                  `json:"average_transaction_size"`
	Tokens                 []BlockscoutTokenBalance `json:"tokens"`
	NFTCount               int                      `json:"nft_count"`
	IsContract             bool                     `json:"is_contract"`
	DeFiInteractionCount   int                      `json:"defi_interaction_count"`
	UniqueContractsCount   int                      `json:"unique_contracts_count"`
	LastUpdated            time.Time                `json:"last_updated"`
}

// NewBlockscoutProvider creates a new Blockscout provider
func NewBlockscoutProvider(baseURL, chainName string) *BlockscoutProvider {
	return &BlockscoutProvider{
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL:   baseURL,
		chainName: chainName,
	}
}

// GetAddressInfo fetches basic address information
func (p *BlockscoutProvider) GetAddressInfo(ctx context.Context, address string) (*BlockscoutAddressInfo, error) {
	url := fmt.Sprintf("%s/api?module=account&action=balance&address=%s", p.baseURL, address)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch address info: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Blockscout API returned status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Status  string `json:"status"`
		Message string `json:"message"`
		Result  string `json:"result"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if result.Status != "1" {
		return nil, fmt.Errorf("Blockscout API error: %s", result.Message)
	}

	return &BlockscoutAddressInfo{
		Hash:    address,
		Balance: result.Result,
	}, nil
}

// GetTransactions fetches transactions for an address
func (p *BlockscoutProvider) GetTransactions(ctx context.Context, address string, page, offset int) ([]BlockscoutTransaction, error) {
	url := fmt.Sprintf("%s/api?module=account&action=txlist&address=%s&page=%d&offset=%d&sort=desc",
		p.baseURL, address, page, offset)

	logger.Info("Fetching transactions from Blockscout",
		zap.String("address", address),
		zap.String("url", url),
	)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch transactions: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Blockscout API returned status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Status  string                  `json:"status"`
		Message string                  `json:"message"`
		Result  []BlockscoutTransaction `json:"result"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if result.Status != "1" {
		// Empty result is ok
		if result.Message == "No transactions found" {
			return []BlockscoutTransaction{}, nil
		}
		return nil, fmt.Errorf("Blockscout API error: %s", result.Message)
	}

	return result.Result, nil
}

// GetTokenBalances fetches ERC20 token balances
func (p *BlockscoutProvider) GetTokenBalances(ctx context.Context, address string) ([]BlockscoutTokenBalance, error) {
	url := fmt.Sprintf("%s/api?module=account&action=tokenlist&address=%s", p.baseURL, address)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch token balances: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("Blockscout API returned status %d: %s", resp.StatusCode, string(body))
	}

	var result struct {
		Status  string                   `json:"status"`
		Message string                   `json:"message"`
		Result  []BlockscoutTokenBalance `json:"result"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	if result.Status != "1" {
		if result.Message == "No tokens found" {
			return []BlockscoutTokenBalance{}, nil
		}
		return nil, fmt.Errorf("Blockscout API error: %s", result.Message)
	}

	return result.Result, nil
}

// GetInternalTransactions fetches internal transactions (contract interactions)
func (p *BlockscoutProvider) GetInternalTransactions(ctx context.Context, address string, page, offset int) ([]BlockscoutInternalTx, error) {
	url := fmt.Sprintf("%s/api?module=account&action=txlistinternal&address=%s&page=%d&offset=%d&sort=desc",
		p.baseURL, address, page, offset)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch internal transactions: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return []BlockscoutInternalTx{}, nil // Return empty on error
	}

	var result struct {
		Status  string                 `json:"status"`
		Message string                 `json:"message"`
		Result  []BlockscoutInternalTx `json:"result"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return []BlockscoutInternalTx{}, nil
	}

	if result.Status != "1" {
		return []BlockscoutInternalTx{}, nil
	}

	return result.Result, nil
}

// GetAnalytics fetches comprehensive analytics for an address
func (p *BlockscoutProvider) GetAnalytics(ctx context.Context, address string) (*BlockscoutAnalytics, error) {
	logger.Info("Fetching comprehensive analytics from Blockscout",
		zap.String("address", address),
		zap.String("chain", p.chainName),
	)

	analytics := &BlockscoutAnalytics{
		Address:     address,
		LastUpdated: time.Now(),
	}

	// Get basic address info
	addressInfo, err := p.GetAddressInfo(ctx, address)
	if err != nil {
		logger.Error("Failed to get address info", zap.Error(err))
	} else {
		// Convert balance from wei to ETH
		balanceWei, _ := strconv.ParseFloat(addressInfo.Balance, 64)
		analytics.Balance = balanceWei / 1e18
		analytics.IsContract = addressInfo.IsContract
	}

	// Get transactions (first 100)
	transactions, err := p.GetTransactions(ctx, address, 1, 100)
	if err != nil {
		logger.Error("Failed to get transactions", zap.Error(err))
	} else {
		analytics.TotalTransactions = len(transactions)

		// Calculate metrics from transactions
		if len(transactions) > 0 {
			// Get first and last transaction dates
			firstTx := transactions[len(transactions)-1]
			lastTx := transactions[0]

			firstTime, _ := strconv.ParseInt(firstTx.TimeStamp, 10, 64)
			lastTime, _ := strconv.ParseInt(lastTx.TimeStamp, 10, 64)

			analytics.FirstTransactionDate = time.Unix(firstTime, 0)
			analytics.LastTransactionDate = time.Unix(lastTime, 0)
			analytics.WalletAgeDays = int(time.Since(analytics.FirstTransactionDate).Hours() / 24)

			// Calculate average transaction size and total gas used
			totalValue := 0.0
			totalGas := 0.0
			contractInteractions := make(map[string]bool)

			for _, tx := range transactions {
				// Convert value from wei to ETH
				value, _ := strconv.ParseFloat(tx.Value, 64)
				totalValue += value / 1e18

				// Track gas used
				gasUsed, _ := strconv.ParseFloat(tx.GasUsed, 64)
				totalGas += gasUsed

				// Count DeFi interactions (contract calls with function names)
				if tx.To != "" && tx.FunctionName != "" {
					contractInteractions[tx.To] = true
					analytics.DeFiInteractionCount++
				}
			}

			if analytics.TotalTransactions > 0 {
				analytics.AverageTransactionSize = totalValue / float64(analytics.TotalTransactions)
			}
			analytics.TotalGasUsed = totalGas
			analytics.UniqueContractsCount = len(contractInteractions)
		}
	}

	// Get token balances
	tokens, err := p.GetTokenBalances(ctx, address)
	if err != nil {
		logger.Error("Failed to get token balances", zap.Error(err))
	} else {
		analytics.Tokens = tokens
		analytics.TotalTokenTransfers = len(tokens)

		// Count NFTs (ERC-721 and ERC-1155)
		for _, token := range tokens {
			if token.TokenType == "ERC-721" || token.TokenType == "ERC-1155" {
				analytics.NFTCount++
			}
		}
	}

	// Get internal transactions
	internalTxs, err := p.GetInternalTransactions(ctx, address, 1, 100)
	if err != nil {
		logger.Error("Failed to get internal transactions", zap.Error(err))
	} else {
		analytics.TotalInternalTxs = len(internalTxs)
	}

	logger.Info("Blockscout analytics fetched successfully",
		zap.String("address", address),
		zap.Int("transactions", analytics.TotalTransactions),
		zap.Int("walletAge", analytics.WalletAgeDays),
		zap.Int("defiInteractions", analytics.DeFiInteractionCount),
	)

	return analytics, nil
}

// ConvertToBlockchainSummary converts Blockscout analytics to standard BlockchainSummary
func (p *BlockscoutProvider) ConvertToBlockchainSummary(analytics *BlockscoutAnalytics) *BlockchainSummary {
	tokenBalances := make(map[string]float64)

	for _, token := range analytics.Tokens {
		if token.TokenType == "ERC-20" {
			// Convert token balance based on decimals
			balance, _ := strconv.ParseFloat(token.Balance, 64)
			decimals := float64(token.TokenDecimals)
			if decimals == 0 {
				decimals = 18 // Default to 18 decimals
			}
			tokenBalances[token.TokenSymbol] = balance / (1e18 / decimals)
		}
	}

	// Add ETH balance
	tokenBalances["ETH"] = analytics.Balance

	return &BlockchainSummary{
		Address:                analytics.Address,
		WalletAge:              analytics.WalletAgeDays,
		FirstTransaction:       analytics.FirstTransactionDate,
		LastTransaction:        analytics.LastTransactionDate,
		TotalTransactions:      analytics.TotalTransactions,
		TotalVolume:            analytics.AverageTransactionSize * float64(analytics.TotalTransactions),
		AverageTransactionSize: analytics.AverageTransactionSize,
		DeFiActivities:         []DeFiActivity{}, // Would need to parse transactions for this
		LendingPositions:       []LendingPosition{},
		LiquidationEvents:      []LiquidationEvent{},
		NFTHoldings:            analytics.NFTCount,
		TokenBalances:          tokenBalances,
		TotalPortfolioValue:    analytics.BalanceUSD,
		LastUpdated:            analytics.LastUpdated,
	}
}

// HealthCheck verifies Blockscout API is accessible
func (p *BlockscoutProvider) HealthCheck(ctx context.Context) error {
	// Try to get info for a known address (null address)
	url := fmt.Sprintf("%s/api?module=account&action=balance&address=0x0000000000000000000000000000000000000000", p.baseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("Blockscout health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Blockscout returned status %d", resp.StatusCode)
	}

	return nil
}

// GetSupportedChains returns list of Blockscout instances for different chains
func GetSupportedBlockscoutChains() map[string]string {
	return map[string]string{
		"ethereum": "https://eth.blockscout.com",
		"polygon":  "https://polygon.blockscout.com",
		"gnosis":   "https://gnosis.blockscout.com",
		"optimism": "https://optimism.blockscout.com",
		"base":     "https://base.blockscout.com",
		"arbitrum": "https://arbitrum.blockscout.com",
		"zksync":   "https://zksync.blockscout.com",
		"scroll":   "https://scroll.blockscout.com",
	}
}

// MockBlockscoutData generates mock Blockscout data for testing
func (p *BlockscoutProvider) MockBlockscoutData(address string) *BlockscoutAnalytics {
	now := time.Now()
	firstTx := now.AddDate(0, -18, 0) // 18 months ago

	return &BlockscoutAnalytics{
		Address:                address,
		Balance:                2.5,
		BalanceUSD:             5000.00,
		FirstTransactionDate:   firstTx,
		LastTransactionDate:    now.AddDate(0, 0, -2),
		WalletAgeDays:          540,
		TotalTransactions:      342,
		TotalTokenTransfers:    156,
		TotalInternalTxs:       89,
		TotalGasUsed:           0.45,
		AverageTransactionSize: 0.25,
		Tokens: []BlockscoutTokenBalance{
			{
				TokenAddress:  "0xA0b86991c6218b36c1d19D4a2e9Eb0cE3606eB48",
				TokenName:     "USD Coin",
				TokenSymbol:   "USDC",
				TokenDecimals: 6,
				Balance:       "5000000000", // 5000 USDC
				TokenType:     "ERC-20",
			},
			{
				TokenAddress:  "0x6B175474E89094C44Da98b954EedeAC495271d0F",
				TokenName:     "Dai Stablecoin",
				TokenSymbol:   "DAI",
				TokenDecimals: 18,
				Balance:       "1200000000000000000000", // 1200 DAI
				TokenType:     "ERC-20",
			},
		},
		NFTCount:             3,
		IsContract:           false,
		DeFiInteractionCount: 45,
		UniqueContractsCount: 12,
		LastUpdated:          now,
	}
}
