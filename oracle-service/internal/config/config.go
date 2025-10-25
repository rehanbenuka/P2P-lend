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

	// Multi-Chain Support
	EnableMultiChain bool     // Enable fetching from multiple chains
	TargetChains     []string // List of chains to fetch from (empty = all supported)
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

		// Multi-Chain
		EnableMultiChain: getBoolEnv("ENABLE_MULTI_CHAIN", true),
		TargetChains:     getSliceEnv("TARGET_CHAINS", []string{"ethereum", "polygon", "arbitrum", "optimism", "base"}),
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

func getSliceEnv(key string, fallback []string) []string {
	if value := os.Getenv(key); value != "" {
		// Support comma-separated values: "ethereum,polygon,arbitrum"
		var result []string
		for _, v := range splitAndTrim(value, ",") {
			if v != "" {
				result = append(result, v)
			}
		}
		if len(result) > 0 {
			return result
		}
	}
	return fallback
}

func splitAndTrim(s, sep string) []string {
	var result []string
	for _, v := range splitString(s, sep) {
		trimmed := trimString(v)
		if trimmed != "" {
			result = append(result, trimmed)
		}
	}
	return result
}

func splitString(s, sep string) []string {
	// Simple string split implementation
	if s == "" {
		return []string{}
	}
	var result []string
	current := ""
	for _, c := range s {
		if string(c) == sep {
			result = append(result, current)
			current = ""
		} else {
			current += string(c)
		}
	}
	result = append(result, current)
	return result
}

func trimString(s string) string {
	// Trim leading and trailing whitespace
	start := 0
	end := len(s)
	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\n' || s[start] == '\r') {
		start++
	}
	for end > start && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\n' || s[end-1] == '\r') {
		end--
	}
	return s[start:end]
}
