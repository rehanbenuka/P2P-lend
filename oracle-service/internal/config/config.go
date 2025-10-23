package config

import "os"

type Config struct {
	DatabaseURL     string
	RedisURL        string
	EthereumRPC     string
	PrivateKey      string
	ContractAddress string
	Port            string
}

func Load() *Config {
	return &Config{
		DatabaseURL:     os.Getenv("DATABASE_URL"),
		RedisURL:        os.Getenv("REDIS_URL"),
		EthereumRPC:     os.Getenv("ETHEREUM_RPC_URL"),
		PrivateKey:      os.Getenv("PRIVATE_KEY"),
		ContractAddress: os.Getenv("CONTRACT_ADDRESS"),
		Port:            getEnv("PORT", "8080"),
	}
}

func getEnv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
