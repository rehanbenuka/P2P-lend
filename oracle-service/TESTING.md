# Oracle Service - Testing Documentation

This document provides comprehensive information about testing the Credit Score Oracle Service.

## Table of Contents

1. [Overview](#overview)
2. [Test Structure](#test-structure)
3. [Running Tests](#running-tests)
4. [Test Coverage](#test-coverage)
5. [Test Cases](#test-cases)
6. [Mock Data](#mock-data)
7. [Integration Testing](#integration-testing)
8. [Performance Testing](#performance-testing)

## Overview

The oracle service has three levels of testing:

- **Unit Tests**: Test individual components in isolation
- **Integration Tests**: Test API endpoints and component interactions
- **End-to-End Tests**: Test complete workflows from API to database

## Test Structure

```
oracle-service/
├── internal/
│   ├── scoring/
│   │   ├── engine.go
│   │   └── engine_test.go           # Scoring algorithm tests
│   ├── repository/
│   │   ├── score_repository.go
│   │   └── score_repository_test.go # Database operation tests
│   └── service/
│       ├── oracle_service.go
│       └── oracle_service_test.go   # Service layer tests
└── tests/
    └── integration_test.go           # API integration tests
```

## Running Tests

### Run All Tests

```bash
cd oracle-service
go test ./... -v
```

### Run Specific Test Package

```bash
# Scoring engine tests
go test ./internal/scoring -v

# Repository tests
go test ./internal/repository -v

# Service tests
go test ./internal/service -v

# Integration tests
go test ./tests -v
```

### Run with Coverage

```bash
go test ./... -cover -coverprofile=coverage.out
go tool cover -html=coverage.out -o coverage.html
```

### Run Benchmarks

```bash
go test ./internal/scoring -bench=. -benchmem
```

## Test Coverage

### Scoring Engine Tests (`internal/scoring/engine_test.go`)

| Test Function | Description | Coverage |
|--------------|-------------|----------|
| `TestCalculateScore` | Tests complete score calculation with various data combinations | High/Low quality data, on-chain only, off-chain only, no data |
| `TestCalculateOnChainScore` | Tests on-chain score calculation | Perfect metrics, nil metrics, minimal activity |
| `TestCalculateOffChainScore` | Tests off-chain score calculation | Excellent/poor credit, nil metrics |
| `TestScoreWalletAge` | Tests wallet age scoring function | Various ages from 0 to 1000+ days |
| `TestScoreBorrowingHistory` | Tests borrowing history scoring | Perfect repayment, defaults, no history, liquidations |
| `TestScoreDTI` | Tests debt-to-income ratio scoring | Various DTI ratios from excellent to poor |
| `TestCalculateConfidence` | Tests confidence level calculation | All data, some data, minimal data |
| `TestValidateScore` | Tests score validation | Valid/invalid scores |
| `TestGenerateDataHash` | Tests data hash generation | Consistency and uniqueness |
| `BenchmarkCalculateScore` | Performance benchmark | N/A |

**Expected Results:**
- All scores should be between 300-850
- Confidence should be 0-100
- Data hashes should be unique for different inputs
- Benchmark should complete in < 1ms per operation

### Repository Tests (`internal/repository/score_repository_test.go`)

| Test Function | Description | Test Cases |
|--------------|-------------|-----------|
| `TestCreateCreditScore` | Tests credit score creation | Successful creation, ID assignment |
| `TestGetByAddress` | Tests score retrieval | Found, not found cases |
| `TestUpdateCreditScore` | Tests score updates | Field updates, version increments |
| `TestGetDueForUpdate` | Tests scheduled update queries | Overdue, not due filtering |
| `TestCreateAndGetHistory` | Tests score history | Creation, retrieval, ordering |
| `TestUpsertOnChainMetrics` | Tests on-chain metrics upsert | Create, update operations |
| `TestUpsertOffChainMetrics` | Tests off-chain metrics upsert | Create, update operations |
| `TestCreateAndGetOracleUpdate` | Tests oracle update records | Creation, retrieval by tx hash |
| `TestGetPendingOracleUpdates` | Tests pending update queries | Status filtering |
| `TestGetStats` | Tests statistics aggregation | Count, average calculations |

**Expected Results:**
- All database operations should succeed with in-memory SQLite
- Retrieved data should match inserted data
- Queries should return correct filtered results

### Service Tests (`internal/service/oracle_service_test.go`)

| Test Function | Description | Test Scenarios |
|--------------|-------------|---------------|
| `TestCalculateAndUpdateScore` | Tests complete score calculation flow | First calculation, updates |
| `TestGetScore` | Tests score retrieval | Existing, non-existent addresses |
| `TestGetScoreHistory` | Tests history retrieval | Multiple updates |
| `TestPublishScoreToBlockchain` | Tests blockchain publishing | Success, not found cases |
| `TestProcessScheduledUpdates` | Tests batch update processing | Multiple overdue scores |
| `TestGetStats` | Tests service statistics | Aggregated metrics |
| `TestHealthCheck` | Tests health check | Component status |
| `TestCalculateScoreWithOnChainOnly` | Tests fallback to on-chain only | Off-chain failure handling |
| `TestConcurrentScoreUpdates` | Tests concurrent operations | Thread safety |

**Expected Results:**
- Service should handle errors gracefully
- Concurrent operations should be safe
- Health checks should validate all components

### Integration Tests (`tests/integration_test.go`)

| Test Function | Description | HTTP Status | Response Validation |
|--------------|-------------|-------------|-------------------|
| `TestHealthEndpoint` | Tests health check endpoint | 200 or 503 | Status field present |
| `TestUpdateCreditScoreEndToEnd` | Tests score update API | 200 | Score in valid range |
| `TestGetCreditScoreEndToEnd` | Tests score retrieval API | 200 | All fields present |
| `TestGetCreditScoreNotFound` | Tests 404 handling | 404 | Error message |
| `TestGetScoreHistoryEndToEnd` | Tests history endpoint | 200 | Array with entries |
| `TestGetStatsEndToEnd` | Tests stats endpoint | 200 | Correct counts |
| `TestInvalidRequestHandling` | Tests error handling | 400 | Error responses |
| `TestFullWorkflow` | Tests complete user workflow | Various | End-to-end validation |
| `TestConcurrentAPIRequests` | Tests concurrent API calls | 200 | Consistent results |

**Expected Results:**
- All endpoints should return correct HTTP status codes
- Response bodies should match expected schemas
- Concurrent requests should be handled correctly

## Test Cases

### 1. Score Calculation Test Cases

#### Case 1: High Quality User
```go
Input:
- On-chain: 2-year-old wallet, 100+ transactions, DeFi activity, perfect repayment
- Off-chain: 750+ credit score, verified income, low DTI

Expected Output:
- Score: 700-850
- Confidence: 80-100
```

#### Case 2: Poor Quality User
```go
Input:
- On-chain: New wallet, few transactions, liquidations
- Off-chain: Low credit score, high DTI, unverified income

Expected Output:
- Score: 300-550
- Confidence: 20-50
```

#### Case 3: On-Chain Only
```go
Input:
- On-chain: Moderate activity
- Off-chain: nil (API failure)

Expected Output:
- Score: 450-650
- Confidence: 30-60
- Should not fail, graceful degradation
```

### 2. API Endpoint Test Cases

#### POST /api/v1/credit-score/update

**Test Case 1: Valid Request**
```json
Request:
{
  "address": "0x1234567890123456789012345678901234567890",
  "user_id": "user123",
  "publish": false
}

Expected Response: 200 OK
{
  "address": "0x1234567890123456789012345678901234567890",
  "score": 720,
  "confidence": 85,
  "on_chain_score": 700,
  "off_chain_score": 740,
  "hybrid_score": 720,
  "data_hash": "abc123...",
  "last_updated": "2025-10-22T10:30:00Z",
  "next_update_due": "2025-11-21T10:30:00Z",
  "update_count": 1
}
```

**Test Case 2: Invalid Address**
```json
Request:
{
  "address": "",
  "user_id": "user123"
}

Expected Response: 400 Bad Request
{
  "error": "Invalid request",
  "message": "address is required"
}
```

#### GET /api/v1/credit-score/:address

**Test Case 1: Existing Score**
```
Request: GET /api/v1/credit-score/0x1234567890123456789012345678901234567890

Expected Response: 200 OK
{
  "address": "0x1234567890123456789012345678901234567890",
  "score": 720,
  ...
}
```

**Test Case 2: Non-Existent Score**
```
Request: GET /api/v1/credit-score/0xNonExistent

Expected Response: 404 Not Found
{
  "error": "Credit score not found",
  "message": "No credit score exists for this address"
}
```

#### GET /api/v1/credit-score/:address/history

**Test Case: History Retrieval**
```
Request: GET /api/v1/credit-score/0x1234.../history?limit=10

Expected Response: 200 OK
[
  {
    "score": 750,
    "confidence": 90,
    "data_hash": "hash3",
    "timestamp": "2025-10-22T10:00:00Z"
  },
  {
    "score": 720,
    "confidence": 85,
    "data_hash": "hash2",
    "timestamp": "2025-09-22T10:00:00Z"
  }
]
```

## Mock Data

### Mock On-Chain Metrics
```go
&models.OnChainMetrics{
    UserAddress:         "0x1234567890123456789012345678901234567890",
    WalletAge:           365,      // 1 year
    TotalTransactions:   100,
    AvgTransactionValue: 500,
    DeFiInteractions:    25,
    BorrowingHistory:    10,
    RepaymentHistory:    9,
    LiquidationEvents:   0,
    CollateralValue:     5000,
    LastActivity:        time.Now(),
}
```

### Mock Off-Chain Metrics
```go
&models.OffChainMetrics{
    UserAddress:            "0x1234567890123456789012345678901234567890",
    TraditionalCreditScore: 720,
    BankAccountHistory:     85,
    IncomeVerified:         true,
    IncomeLevel:            "medium",
    EmploymentStatus:       "full-time",
    DebtToIncomeRatio:      0.30,
    DataSource:             "mock",
    LastVerified:           time.Now(),
}
```

## Integration Testing

### Setup

Integration tests use an in-memory SQLite database for fast, isolated testing:

```go
db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
```

### Test Workflow

1. **Setup Phase**: Create test database and initialize components
2. **Execution Phase**: Make HTTP requests to test endpoints
3. **Verification Phase**: Assert responses and database state
4. **Cleanup Phase**: Automatic (in-memory DB discarded)

### Running Integration Tests

```bash
# Run all integration tests
go test ./tests -v

# Run specific integration test
go test ./tests -v -run TestFullWorkflow

# Run with race detection
go test ./tests -v -race
```

## Performance Testing

### Benchmarks

Run performance benchmarks:

```bash
go test ./internal/scoring -bench=BenchmarkCalculateScore -benchmem
```

**Expected Performance:**
- Score calculation: < 1ms per operation
- Database operations: < 10ms per operation
- API requests: < 50ms per request

### Load Testing

For load testing the API, use tools like `wrk` or `ab`:

```bash
# Install wrk
apt-get install wrk

# Run load test
wrk -t4 -c100 -d30s --latency http://localhost:8080/health
```

**Expected Results:**
- 1000+ requests/second
- 95th percentile latency < 100ms
- No errors under normal load

## Test Data Cleanup

Tests use in-memory databases that are automatically cleaned up. For persistent database testing:

```bash
# Clean test database
rm -f test.db

# Run tests with fresh database
go test ./... -v
```

## Continuous Integration

### GitHub Actions Example

```yaml
name: Test
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v2
      - uses: actions/setup-go@v2
        with:
          go-version: 1.21
      - run: go test ./... -v -cover
      - run: go test ./... -race
```

## Troubleshooting

### Common Issues

**Issue**: Tests fail with "database locked"
**Solution**: Ensure you're using in-memory SQLite (`:memory:`) for tests

**Issue**: Race condition detected
**Solution**: Review concurrent access patterns, add proper locking

**Issue**: Mock data not being used
**Solution**: Verify mock aggregators are properly injected into service

## Test Maintenance

### Adding New Tests

1. Create test file with `_test.go` suffix
2. Import testing package and dependencies
3. Write test functions starting with `Test`
4. Use table-driven tests for multiple scenarios
5. Add assertions for all expected outcomes

### Best Practices

- Use descriptive test names
- Test both success and failure cases
- Use table-driven tests for multiple inputs
- Mock external dependencies
- Keep tests independent and isolated
- Clean up resources in defer statements
- Use subtests for better organization

## Coverage Goals

Target coverage by package:
- `internal/scoring`: > 90%
- `internal/repository`: > 85%
- `internal/service`: > 80%
- `internal/api/handlers`: > 75%

Check current coverage:

```bash
go test ./... -coverprofile=coverage.out
go tool cover -func=coverage.out
```

## Further Reading

- [Go Testing Package](https://golang.org/pkg/testing/)
- [Table Driven Tests](https://github.com/golang/go/wiki/TableDrivenTests)
- [Testify Framework](https://github.com/stretchr/testify)
- [GORM Testing](https://gorm.io/docs/testing.html)
