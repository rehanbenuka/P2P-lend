package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/yourusername/p2p-lend/oracle-service/internal/api/routes"
	"github.com/yourusername/p2p-lend/oracle-service/internal/config"
	"github.com/yourusername/p2p-lend/oracle-service/pkg/logger"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	// Initialize logger
	logger.Init()

	// Load configuration
	cfg := config.Load()

	// Initialize Gin router
	router := gin.Default()

	// Setup routes
	routes.Setup(router, cfg)

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	logger.Info("Starting oracle service on port " + port)
	if err := router.Run(":" + port); err != nil {
		logger.Fatal("Failed to start server: " + err.Error())
	}
}
