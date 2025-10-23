package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/yourusername/p2p-lend/oracle-service/internal/aggregator"
	"github.com/yourusername/p2p-lend/oracle-service/internal/api/handlers"
	"github.com/yourusername/p2p-lend/oracle-service/internal/models"
	"github.com/yourusername/p2p-lend/oracle-service/internal/repository"
	"github.com/yourusername/p2p-lend/oracle-service/internal/scoring"
	"github.com/yourusername/p2p-lend/oracle-service/internal/service"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// Integration test setup
func setupTestRouter(t *testing.T) (*gin.Engine, *service.OracleService, *gorm.DB) {
	gin.SetMode(gin.TestMode)

	// Setup test database
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	// Auto-migrate
	db.AutoMigrate(
		&models.CreditScore{},
		&models.ScoreHistory{},
		&models.OnChainMetrics{},
		&models.OffChainMetrics{},
		&models.OracleUpdate{},
	)

	// Setup service
	repo := repository.NewScoreRepository(db)
	engine := scoring.NewEngine()

	// Use mock aggregators for testing
	type mockOnChainAgg struct{}
	onChainAgg := &mockOnChainAgg{}

	type mockOffChainAgg struct{}
	offChainAgg := aggregator.NewOffChainAggregator("", "", "")

	oracleService := service.NewOracleService(repo, engine, onChainAgg, offChainAgg, nil)

	// Setup router
	router := gin.New()
	scoreHandler := handlers.NewScoreHandler(oracleService)

	router.GET("/health", scoreHandler.HealthCheck)
	v1 := router.Group("/api/v1")
	{
		v1.GET("/credit-score/:address", scoreHandler.GetCreditScore)
		v1.POST("/credit-score/update", scoreHandler.UpdateCreditScore)
		v1.GET("/credit-score/:address/history", scoreHandler.GetScoreHistory)
		v1.GET("/admin/stats", scoreHandler.GetStats)
	}

	return router, oracleService, db
}

// Mock on-chain aggregator for integration tests
type mockOnChainAgg struct{}

func (m *mockOnChainAgg) FetchMetrics(ctx context.Context, address string) (*models.OnChainMetrics, error) {
	return &models.OnChainMetrics{
		UserAddress:         address,
		WalletAge:           365,
		TotalTransactions:   100,
		AvgTransactionValue: 500,
		DeFiInteractions:    25,
		BorrowingHistory:    10,
		RepaymentHistory:    9,
		LiquidationEvents:   0,
		CollateralValue:     5000,
		LastActivity:        time.Now(),
	}, nil
}

func (m *mockOnChainAgg) HealthCheck(ctx context.Context) error {
	return nil
}

func (m *mockOnChainAgg) Close() {}

func TestHealthEndpoint(t *testing.T) {
	router, _, _ := setupTestRouter(t)

	req, _ := http.NewRequest("GET", "/health", nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK && resp.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status 200 or 503, got %d", resp.Code)
	}

	var result map[string]interface{}
	json.Unmarshal(resp.Body.Bytes(), &result)

	if result["status"] == nil {
		t.Error("Health check should return status")
	}
}

func TestUpdateCreditScoreEndToEnd(t *testing.T) {
	router, _, _ := setupTestRouter(t)

	address := "0x1234567890123456789012345678901234567890"

	// Update credit score
	updateReq := map[string]interface{}{
		"address": address,
		"user_id": "user123",
		"publish": false,
	}

	body, _ := json.Marshal(updateReq)
	req, _ := http.NewRequest("POST", "/api/v1/credit-score/update", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d. Body: %s", resp.Code, resp.Body.String())
	}

	var result map[string]interface{}
	json.Unmarshal(resp.Body.Bytes(), &result)

	if result["score"] == nil {
		t.Error("Response should contain score")
	}

	score := result["score"].(float64)
	if score < 300 || score > 850 {
		t.Errorf("Score %f is outside valid range [300-850]", score)
	}

	if result["confidence"] == nil {
		t.Error("Response should contain confidence")
	}

	if result["data_hash"] == nil {
		t.Error("Response should contain data_hash")
	}
}

func TestGetCreditScoreEndToEnd(t *testing.T) {
	router, service, _ := setupTestRouter(t)

	address := "0x1234567890123456789012345678901234567890"

	// First create a score
	_, err := service.CalculateAndUpdateScore(context.Background(), address, "user123")
	if err != nil {
		t.Fatalf("Failed to create test score: %v", err)
	}

	// Then retrieve it via API
	req, _ := http.NewRequest("GET", "/api/v1/credit-score/"+address, nil)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.Code)
	}

	var result map[string]interface{}
	json.Unmarshal(resp.Body.Bytes(), &result)

	if result["address"] != address {
		t.Errorf("Expected address %s, got %v", address, result["address"])
	}

	if result["score"] == nil {
		t.Error("Response should contain score")
	}
}

func TestGetCreditScoreNotFound(t *testing.T) {
	router, _, _ := setupTestRouter(t)

	req, _ := http.NewRequest("GET", "/api/v1/credit-score/0xNonExistent", nil)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusNotFound {
		t.Errorf("Expected status 404, got %d", resp.Code)
	}
}

func TestGetScoreHistoryEndToEnd(t *testing.T) {
	router, service, _ := setupTestRouter(t)

	address := "0x1234567890123456789012345678901234567890"

	// Create multiple score updates
	for i := 0; i < 3; i++ {
		_, err := service.CalculateAndUpdateScore(context.Background(), address, "user123")
		if err != nil {
			t.Fatalf("Failed to create test score: %v", err)
		}
		time.Sleep(10 * time.Millisecond)
	}

	// Get history via API
	req, _ := http.NewRequest("GET", "/api/v1/credit-score/"+address+"/history?limit=10", nil)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.Code)
	}

	var result []map[string]interface{}
	json.Unmarshal(resp.Body.Bytes(), &result)

	if len(result) != 3 {
		t.Errorf("Expected 3 history entries, got %d", len(result))
	}

	// Verify each entry has required fields
	for _, entry := range result {
		if entry["score"] == nil {
			t.Error("History entry should contain score")
		}
		if entry["confidence"] == nil {
			t.Error("History entry should contain confidence")
		}
		if entry["timestamp"] == nil {
			t.Error("History entry should contain timestamp")
		}
	}
}

func TestGetStatsEndToEnd(t *testing.T) {
	router, service, _ := setupTestRouter(t)

	// Create test scores
	addresses := []string{"0x1111", "0x2222", "0x3333"}
	for _, addr := range addresses {
		_, err := service.CalculateAndUpdateScore(context.Background(), addr, "user"+addr)
		if err != nil {
			t.Fatalf("Failed to create test score: %v", err)
		}
	}

	// Get stats via API
	req, _ := http.NewRequest("GET", "/api/v1/admin/stats", nil)
	resp := httptest.NewRecorder()

	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.Code)
	}

	var result map[string]interface{}
	json.Unmarshal(resp.Body.Bytes(), &result)

	if result["total_active_scores"] == nil {
		t.Error("Stats should contain total_active_scores")
	}

	totalScores := result["total_active_scores"].(float64)
	if totalScores != 3 {
		t.Errorf("Expected 3 active scores, got %f", totalScores)
	}

	if result["average_score"] == nil {
		t.Error("Stats should contain average_score")
	}
}

func TestInvalidRequestHandling(t *testing.T) {
	router, _, _ := setupTestRouter(t)

	tests := []struct {
		name           string
		method         string
		path           string
		body           string
		expectedStatus int
	}{
		{
			name:           "Invalid JSON in update request",
			method:         "POST",
			path:           "/api/v1/credit-score/update",
			body:           "invalid json",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Missing required fields",
			method:         "POST",
			path:           "/api/v1/credit-score/update",
			body:           "{}",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "Invalid address format (too short)",
			method:         "GET",
			path:           "/api/v1/credit-score/0x123",
			body:           "",
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req *http.Request
			if tt.body != "" {
				req, _ = http.NewRequest(tt.method, tt.path, bytes.NewBufferString(tt.body))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req, _ = http.NewRequest(tt.method, tt.path, nil)
			}

			resp := httptest.NewRecorder()
			router.ServeHTTP(resp, req)

			if resp.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, resp.Code)
			}
		})
	}
}

func TestFullWorkflow(t *testing.T) {
	router, _, _ := setupTestRouter(t)

	address := "0xFullWorkflowTest1234567890123456789012"

	// Step 1: Verify score doesn't exist
	req, _ := http.NewRequest("GET", "/api/v1/credit-score/"+address, nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusNotFound {
		t.Errorf("Step 1: Expected 404, got %d", resp.Code)
	}

	// Step 2: Create initial score
	updateReq := map[string]interface{}{
		"address": address,
		"user_id": "testuser",
		"publish": false,
	}

	body, _ := json.Marshal(updateReq)
	req, _ = http.NewRequest("POST", "/api/v1/credit-score/update", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Errorf("Step 2: Expected 200, got %d", resp.Code)
	}

	var createResult map[string]interface{}
	json.Unmarshal(resp.Body.Bytes(), &createResult)
	initialScore := createResult["score"].(float64)

	// Step 3: Retrieve the score
	req, _ = http.NewRequest("GET", "/api/v1/credit-score/"+address, nil)
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Errorf("Step 3: Expected 200, got %d", resp.Code)
	}

	var getResult map[string]interface{}
	json.Unmarshal(resp.Body.Bytes(), &getResult)

	if getResult["score"].(float64) != initialScore {
		t.Error("Step 3: Retrieved score doesn't match created score")
	}

	// Step 4: Update score again
	time.Sleep(100 * time.Millisecond) // Ensure different timestamp

	req, _ = http.NewRequest("POST", "/api/v1/credit-score/update", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Errorf("Step 4: Expected 200, got %d", resp.Code)
	}

	var updateResult map[string]interface{}
	json.Unmarshal(resp.Body.Bytes(), &updateResult)

	if updateResult["update_count"].(float64) != 2 {
		t.Errorf("Step 4: Expected update count 2, got %f", updateResult["update_count"].(float64))
	}

	// Step 5: Check history
	req, _ = http.NewRequest("GET", "/api/v1/credit-score/"+address+"/history", nil)
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Errorf("Step 5: Expected 200, got %d", resp.Code)
	}

	var historyResult []map[string]interface{}
	json.Unmarshal(resp.Body.Bytes(), &historyResult)

	if len(historyResult) != 2 {
		t.Errorf("Step 5: Expected 2 history entries, got %d", len(historyResult))
	}

	// Step 6: Verify stats include our score
	req, _ = http.NewRequest("GET", "/api/v1/admin/stats", nil)
	resp = httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Errorf("Step 6: Expected 200, got %d", resp.Code)
	}

	var statsResult map[string]interface{}
	json.Unmarshal(resp.Body.Bytes(), &statsResult)

	if statsResult["total_active_scores"].(float64) < 1 {
		t.Error("Step 6: Expected at least 1 active score in stats")
	}
}

func TestConcurrentAPIRequests(t *testing.T) {
	router, _, _ := setupTestRouter(t)

	address := "0xConcurrentTest123456789012345678901234"

	done := make(chan bool, 10)

	// Create 10 concurrent update requests
	for i := 0; i < 10; i++ {
		go func(id int) {
			updateReq := map[string]interface{}{
				"address": address,
				"user_id": "concurrent_user",
				"publish": false,
			}

			body, _ := json.Marshal(updateReq)
			req, _ := http.NewRequest("POST", "/api/v1/credit-score/update", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			resp := httptest.NewRecorder()

			router.ServeHTTP(resp, req)

			if resp.Code != http.StatusOK {
				t.Errorf("Concurrent request %d failed with status %d", id, resp.Code)
			}

			done <- true
		}(i)
	}

	// Wait for all requests to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify final state
	req, _ := http.NewRequest("GET", "/api/v1/credit-score/"+address, nil)
	resp := httptest.NewRecorder()
	router.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Errorf("Failed to retrieve score after concurrent updates: %d", resp.Code)
	}

	var result map[string]interface{}
	json.Unmarshal(resp.Body.Bytes(), &result)

	updateCount := result["update_count"].(float64)
	if updateCount < 1 {
		t.Error("Score should have been updated at least once")
	}
}
