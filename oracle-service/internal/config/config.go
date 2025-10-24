package config

import (
	"os"
	"strconv"
)

type Config struct {
	// Server Configuration
	Port string

	// Database Configuration
	DatabaseURL string
	RedisURL    string

	// Blockchain Configuration
	EthereumRPC     string
	PrivateKey      string
	ContractAddress string

	// Provider Configuration
	UseMockData bool

	// Credit Bureau Configuration
	CreditBureauProvider string
	CreditBureauURL      string
	CreditBureauAPIKey   string

	// Plaid Configuration
	PlaidClientID string
	PlaidSecret   string
	PlaidEnv      string

	// Covalent Configuration
	CovalentAPIKey  string
	CovalentBaseURL string

	// Moralis Configuration
	MoralisAPIKey  string
	MoralisBaseURL string

	// Blockscout Configuration
	BlockscoutBaseURL string
	BlockscoutChain   string
	PreferBlockscout  bool
}

func Load() *Config {
	return &Config{
		// Server
		Port: getEnv("PORT", "8080"),

		// Database
		DatabaseURL: os.Getenv("DATABASE_URL"),
		RedisURL:    os.Getenv("REDIS_URL"),

		// Blockchain
		EthereumRPC:     os.Getenv("ETHEREUM_RPC_URL"),
		PrivateKey:      os.Getenv("PRIVATE_KEY"),
		ContractAddress: os.Getenv("CONTRACT_ADDRESS"),

		// Provider
		UseMockData: getBoolEnv("USE_MOCK_DATA", false),

		// Credit Bureau
		CreditBureauProvider: getEnv("CREDIT_BUREAU_PROVIDER", "experian"),
		CreditBureauURL:      os.Getenv("CREDIT_BUREAU_URL"),
		CreditBureauAPIKey:   os.Getenv("CREDIT_BUREAU_API_KEY"),

		// Plaid
		PlaidClientID: os.Getenv("PLAID_CLIENT_ID"),
		PlaidSecret:   os.Getenv("PLAID_SECRET"),
		PlaidEnv:      getEnv("PLAID_ENV", "sandbox"),

		// Covalent
		CovalentAPIKey:  os.Getenv("COVALENT_API_KEY"),
		CovalentBaseURL: getEnv("COVALENT_BASE_URL", "https://api.covalenthq.com/v1"),

		// Moralis
		MoralisAPIKey:  os.Getenv("MORALIS_API_KEY"),
		MoralisBaseURL: getEnv("MORALIS_BASE_URL", "https://deep-index.moralis.io/api/v2"),

		// Blockscout
		BlockscoutBaseURL: getEnv("BLOCKSCOUT_BASE_URL", "https://eth.blockscout.com"),
		BlockscoutChain:   getEnv("BLOCKSCOUT_CHAIN", "ethereum"),
		PreferBlockscout:  getBoolEnv("PREFER_BLOCKSCOUT", true),
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func getBoolEnv(key string, fallback bool) bool {
	if value := os.Getenv(key); value != "" {
		boolVal, err := strconv.ParseBool(value)
		if err != nil {
			return fallback
		}
		return boolVal
	}
	return fallback
}
