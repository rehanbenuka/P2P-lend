package routes

import (
	"fmt"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"github.com/yourusername/p2p-lend/oracle-service/internal/aggregator"
	"github.com/yourusername/p2p-lend/oracle-service/internal/api/handlers"
	"github.com/yourusername/p2p-lend/oracle-service/internal/blockchain"
	"github.com/yourusername/p2p-lend/oracle-service/internal/config"
	"github.com/yourusername/p2p-lend/oracle-service/internal/models"
	"github.com/yourusername/p2p-lend/oracle-service/internal/providers"
	"github.com/yourusername/p2p-lend/oracle-service/internal/repository"
	"github.com/yourusername/p2p-lend/oracle-service/internal/scoring"
	"github.com/yourusername/p2p-lend/oracle-service/internal/service"
	"github.com/yourusername/p2p-lend/oracle-service/pkg/logger"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func Setup(router *gin.Engine, cfg *config.Config) {
	// Initialize database
	db, err := initDatabase(cfg)
	if err != nil {
		logger.Fatal("Failed to initialize database", zap.Error(err))
	}

	// Initialize components
	repo := repository.NewScoreRepository(db)
	scoringEngine := scoring.NewEngine()

	// Initialize basic aggregators (for fallback)
	basicOnChainAgg, err := aggregator.NewOnChainAggregator(cfg.EthereumRPC)
	if err != nil {
		logger.Fatal("Failed to initialize on-chain aggregator", zap.Error(err))
	}

	basicOffChainAgg := aggregator.NewOffChainAggregator(
		cfg.CreditBureauURL,
		"", // bankAPIURL - not used in basic mode
		cfg.CreditBureauAPIKey,
	)

	// Initialize 3rd party providers
	creditBureauProvider := providers.NewCreditBureauProvider(
		cfg.CreditBureauProvider,
		cfg.CreditBureauURL,
		cfg.CreditBureauAPIKey,
	)

	plaidProvider := providers.NewPlaidProvider(
		cfg.PlaidClientID,
		cfg.PlaidSecret,
		cfg.PlaidEnv,
	)

	// Initialize blockchain data provider (Covalent)
	blockchainProvider := providers.NewBlockchainDataProvider(
		"covalent",
		cfg.CovalentBaseURL,
		cfg.CovalentAPIKey,
	)

	blockscoutProvider := providers.NewBlockscoutProvider(
		cfg.BlockscoutBaseURL,
		cfg.BlockscoutChain,
	)

	// Initialize enhanced aggregators
	enhancedOffChainAgg := aggregator.NewEnhancedOffChainAggregator(
		creditBureauProvider,
		plaidProvider,
		cfg.UseMockData,
	)

	enhancedOnChainAgg := aggregator.NewEnhancedOnChainAggregator(
		blockchainProvider,
		blockscoutProvider,
		basicOnChainAgg,
		cfg.UseMockData,
		cfg.PreferBlockscout,
	)

	var blockchainClient *blockchain.OracleClient
	if cfg.EthereumRPC != "" && cfg.ContractAddress != "" && cfg.PrivateKey != "" {
		blockchainClient, err = blockchain.NewOracleClient(
			cfg.EthereumRPC,
			cfg.ContractAddress,
			cfg.PrivateKey,
		)
		if err != nil {
			logger.Error("Failed to initialize blockchain client", zap.Error(err))
		}
	}

	// Initialize base oracle service
	baseService := service.NewOracleService(
		repo,
		scoringEngine,
		basicOnChainAgg,
		basicOffChainAgg,
		blockchainClient,
	)

	// Initialize enhanced oracle service
	enhancedService := service.NewEnhancedOracleService(
		baseService,
		enhancedOnChainAgg,
		enhancedOffChainAgg,
		creditBureauProvider,
		plaidProvider,
		blockchainProvider,
	)

	// Initialize handlers
	scoreHandler := handlers.NewScoreHandler(baseService)
	providerHandler := handlers.NewProviderHandler(enhancedService)

	// Health check
	router.GET("/health", scoreHandler.HealthCheck)

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// Credit score routes
		v1.GET("/credit-score/:address", scoreHandler.GetCreditScore)
		v1.POST("/credit-score/update", scoreHandler.UpdateCreditScore)
		v1.GET("/credit-score/:address/history", scoreHandler.GetScoreHistory)

		// Enhanced credit score routes with 3rd party providers
		v1.POST("/credit-score/update-with-providers", providerHandler.UpdateWithProviders)

		// Provider routes
		providers := v1.Group("/providers")
		{
			providers.GET("/status", providerHandler.GetProviderStatus)
			providers.GET("/list", providerHandler.ListAvailableProviders)
		}

		// Admin routes
		admin := v1.Group("/admin")
		{
			admin.GET("/stats", scoreHandler.GetStats)
		}
	}
}

func initDatabase(cfg *config.Config) (*gorm.DB, error) {
	var db *gorm.DB
	var err error

	if cfg.DatabaseURL == "" {
		logger.Info("No database URL configured, using in-memory SQLite")
		// Use pure Go SQLite (no CGO required)
		db, err = gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
		if err != nil {
			return nil, fmt.Errorf("failed to initialize in-memory database: %w", err)
		}
	} else {
		logger.Info("Connecting to PostgreSQL database")
		db, err = gorm.Open(postgres.Open(cfg.DatabaseURL), &gorm.Config{})
		if err != nil {
			return nil, err
		}
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
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	logger.Info("Database initialized successfully")
	return db, nil
}
