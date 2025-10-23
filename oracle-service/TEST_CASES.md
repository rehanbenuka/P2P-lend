# Oracle Service - Complete Test Cases

This document provides a comprehensive list of all test cases for the Credit Score Oracle Service.

## Test Summary

| Category | Test Files | Test Count | Coverage Target |
|----------|-----------|------------|-----------------|
| Unit Tests - Scoring | `internal/scoring/engine_test.go` | 10 | >90% |
| Unit Tests - Repository | `internal/repository/score_repository_test.go` | 12 | >85% |
| Unit Tests - Service | `internal/service/oracle_service_test.go` | 10 | >80% |
| Integration Tests | `tests/integration_test.go` | 10 | >75% |
| **Total** | **4 files** | **42 tests** | **>80%** |

---

## 1. Scoring Engine Tests (`internal/scoring/engine_test.go`)

### Test 1.1: TestCalculateScore

**Purpose**: Test complete credit score calculation with various data combinations

**Test Cases**:

| # | Scenario | On-Chain Data | Off-Chain Data | Expected Score Range | Expected Confidence |
|---|----------|---------------|----------------|---------------------|-------------------|
| 1.1.1 | High quality data | Excellent (730 days, 100 tx, no liquidations) | Excellent (750 score, verified income) | 700-850 | 80-100 |
| 1.1.2 | Poor quality data | Poor (30 days, 10 tx, 3 liquidations) | Poor (550 score, high DTI) | 300-550 | 20-50 |
| 1.1.3 | On-chain only | Moderate (365 days, 50 tx) | nil | 450-650 | 30-60 |
| 1.1.4 | Off-chain only | nil | Good (680 score, verified) | 450-700 | 30-60 |
| 1.1.5 | No data | nil | nil | 300-400 | 0-20 |

**Assertions**:
- Score is within 300-850 range
- Confidence is within 0-100 range
- DataHash is not empty
- LastUpdated is set
- NextUpdateDue is set

### Test 1.2: TestCalculateOnChainScore

**Purpose**: Test on-chain score calculation in isolation

**Test Cases**:

| # | Scenario | Wallet Age | Transactions | DeFi Activity | Borrowing | Expected Score |
|---|----------|-----------|--------------|---------------|-----------|----------------|
| 1.2.1 | Perfect metrics | 1000 days | 200 | 100 | 20/20/0 (borrowed/repaid/liquidated) | ~800 |
| 1.2.2 | Nil metrics | - | - | - | - | 300 |
| 1.2.3 | Minimal activity | 7 days | 5 | 0 | 0/0/0 | ~350 |

**Assertions**:
- Score within ±50 points of expected
- Valid score range (300-850)

### Test 1.3: TestCalculateOffChainScore

**Purpose**: Test off-chain score calculation in isolation

**Test Cases**:

| # | Scenario | Credit Score | Bank History | Income | DTI | Expected Score |
|---|----------|-------------|--------------|--------|-----|----------------|
| 1.3.1 | Excellent profile | 800 | 95 | Verified/High | 0.20 | ~750 |
| 1.3.2 | Nil metrics | - | - | - | - | 300 |
| 1.3.3 | Poor profile | 500 | 30 | Unverified/Low | 0.60 | ~450 |

**Assertions**:
- Score within ±100 points of expected
- Valid score range (300-850)

### Test 1.4: TestScoreWalletAge

**Purpose**: Test wallet age scoring function

**Test Cases**:

| Input (Days) | Expected Output |
|-------------|----------------|
| 0 | 0.0 |
| 365 | 0.5 |
| 730 | 1.0 |
| 1000+ | 1.0 |

**Assertions**:
- Exact match of expected values

### Test 1.5: TestScoreBorrowingHistory

**Purpose**: Test borrowing history scoring

**Test Cases**:

| # | Borrowed | Repaid | Liquidations | Expected Range | Scenario |
|---|----------|--------|--------------|----------------|----------|
| 1.5.1 | 10 | 10 | 0 | 0.9-1.0 | Perfect repayment |
| 1.5.2 | 10 | 7 | 1 | 0.3-0.6 | Some defaults |
| 1.5.3 | 0 | 0 | 0 | 0.5-0.5 | No history |
| 1.5.4 | 10 | 5 | 5 | 0.0-0.3 | Many liquidations |

**Assertions**:
- Score within expected range

### Test 1.6: TestScoreDTI

**Purpose**: Test debt-to-income ratio scoring

**Test Cases**:

| DTI Ratio | Expected Score | Classification |
|-----------|---------------|----------------|
| 0.30 | 1.0 | Excellent |
| 0.36 | 1.0 | Good |
| 0.40 | 0.7 | Moderate |
| 0.45 | 0.4 | High |
| 0.60 | 0.2 | Very high |

**Assertions**:
- Exact match of expected values

### Test 1.7: TestCalculateConfidence

**Purpose**: Test confidence level calculation

**Test Cases**:

| # | On-Chain Data | Off-Chain Data | Expected Confidence |
|---|--------------|----------------|-------------------|
| 1.7.1 | Recent, complete | Full, verified | 80-100 |
| 1.7.2 | Partial | Partial | 20-45 |
| 1.7.3 | nil | nil | 0-10 |

**Assertions**:
- Confidence within expected range
- Never exceeds 100

### Test 1.8: TestValidateScore

**Purpose**: Test score validation

**Test Cases**:

| Input Score | Should Pass | Reason |
|------------|-------------|--------|
| 300 | ✓ | Minimum valid |
| 500 | ✓ | Mid-range |
| 850 | ✓ | Maximum valid |
| 299 | ✗ | Below minimum |
| 851 | ✗ | Above maximum |
| 0 | ✗ | Invalid |
| 1000 | ✗ | Too high |

**Assertions**:
- Error returned for invalid scores
- No error for valid scores

### Test 1.9: TestGenerateDataHash

**Purpose**: Test data hash generation

**Test Cases**:

| # | Test | Assertion |
|---|------|-----------|
| 1.9.1 | Hash not empty | Hash length = 64 (SHA256 hex) |
| 1.9.2 | Different scores | Different hashes |
| 1.9.3 | Same inputs | Consistent hash (within same second) |

### Test 1.10: BenchmarkCalculateScore

**Purpose**: Performance benchmark

**Expected Performance**:
- < 1ms per operation
- < 1000 B/op memory allocation

---

## 2. Repository Tests (`internal/repository/score_repository_test.go`)

### Test 2.1: TestCreateCreditScore

**Purpose**: Test credit score creation in database

**Test Steps**:
1. Create new credit score
2. Verify ID is assigned
3. Verify all fields persisted

**Expected**:
- No error
- ID > 0
- CreatedAt set

### Test 2.2: TestGetByAddress

**Purpose**: Test score retrieval by address

**Test Cases**:

| # | Scenario | Setup | Expected Result |
|---|----------|-------|----------------|
| 2.2.1 | Score exists | Create score | Score returned |
| 2.2.2 | Score not found | Empty DB | nil, no error |

### Test 2.3: TestUpdateCreditScore

**Purpose**: Test score updates

**Test Steps**:
1. Create initial score (700, count=1)
2. Update to 750, count=2
3. Retrieve and verify
4. Verify ID unchanged

**Expected**:
- Score updated to 750
- UpdateCount = 2
- Same ID

### Test 2.4: TestGetDueForUpdate

**Purpose**: Test query for overdue scores

**Test Data**:

| Address | NextUpdateDue | Should Return |
|---------|--------------|---------------|
| 0x1111 | -1 day (overdue) | ✓ |
| 0x2222 | +5 days | ✗ |
| 0x3333 | -2 hours (due) | ✓ |

**Expected**:
- Returns 2 scores
- Ordered by due date

### Test 2.5: TestCreateAndGetHistory

**Purpose**: Test score history tracking

**Test Steps**:
1. Create 3 history entries at different times
2. Retrieve history
3. Verify order (newest first)

**Expected**:
- 3 entries returned
- Descending timestamp order
- Most recent score = 750

### Test 2.6: TestUpsertOnChainMetrics

**Purpose**: Test on-chain metrics upsert

**Test Steps**:
1. Insert metrics (age=365, tx=50)
2. Update metrics (age=400, tx=75)
3. Verify no duplicate created
4. Verify values updated

**Expected**:
- Single record
- Updated values

### Test 2.7: TestUpsertOffChainMetrics

**Purpose**: Test off-chain metrics upsert

**Test Steps**:
1. Insert metrics (score=700)
2. Update metrics (score=750, level=high)
3. Verify update

**Expected**:
- Single record
- Updated fields

### Test 2.8: TestCreateAndGetOracleUpdate

**Purpose**: Test oracle update tracking

**Test Steps**:
1. Create oracle update
2. Retrieve by tx hash
3. Verify all fields

**Expected**:
- Record found
- Correct status and score

### Test 2.9: TestGetPendingOracleUpdates

**Purpose**: Test pending updates query

**Test Data**:

| TxHash | Status | Should Return |
|--------|--------|---------------|
| 0xaaa | pending | ✓ |
| 0xbbb | confirmed | ✗ |
| 0xccc | pending | ✓ |

**Expected**:
- Returns 2 pending updates

### Test 2.10: TestGetStats

**Purpose**: Test statistics aggregation

**Test Setup**:
- 2 active scores (700, 800)
- 1 overdue score
- 1 pending oracle update

**Expected Stats**:
- total_active_scores: 2
- average_score: 750
- due_for_update: 1
- pending_oracle_updates: 1

### Test 2.11: TestGetOnChainMetrics

**Purpose**: Test on-chain metrics retrieval

**Expected**:
- Metrics returned for existing address
- nil for non-existent address

### Test 2.12: TestGetOffChainMetrics

**Purpose**: Test off-chain metrics retrieval

**Expected**:
- Metrics returned for existing address
- nil for non-existent address

---

## 3. Service Tests (`internal/service/oracle_service_test.go`)

### Test 3.1: TestCalculateAndUpdateScore

**Purpose**: Test complete score calculation flow

**Test Steps**:
1. Calculate score (first time)
2. Verify update_count = 1
3. Calculate again
4. Verify update_count = 2
5. Verify ID unchanged

**Expected**:
- Score in valid range
- Update count increments
- Same record updated

### Test 3.2: TestGetScore

**Purpose**: Test score retrieval

**Test Cases**:

| # | Scenario | Expected |
|---|----------|----------|
| 3.2.1 | Existing score | Score returned |
| 3.2.2 | Non-existent | nil, no error |

### Test 3.3: TestGetScoreHistory

**Purpose**: Test history retrieval

**Test Steps**:
1. Create 3 score updates
2. Retrieve history (limit=10)
3. Verify count = 3

**Expected**:
- 3 entries
- Chronological order

### Test 3.4: TestPublishScoreToBlockchain

**Purpose**: Test blockchain publishing

**Test Steps**:
1. Calculate score
2. Publish to blockchain
3. Verify no error (mock)

**Expected**:
- No error
- Oracle update record created

### Test 3.5: TestPublishScoreNotFound

**Purpose**: Test error handling

**Test Steps**:
1. Attempt publish for non-existent address
2. Verify error returned

**Expected**:
- Error returned

### Test 3.6: TestProcessScheduledUpdates

**Purpose**: Test batch update processing

**Test Setup**:
- 3 overdue scores

**Test Steps**:
1. Run ProcessScheduledUpdates
2. Verify all updated

**Expected**:
- All update_count >= 2
- Blockchain publish attempted

### Test 3.7: TestGetStats

**Purpose**: Test service statistics

**Test Setup**:
- 2 scores created

**Expected**:
- total_active_scores = 2
- Stats object complete

### Test 3.8: TestHealthCheck

**Purpose**: Test health check

**Expected Components**:
- onchain_aggregator: true
- offchain_aggregator: true
- blockchain_client: true

### Test 3.9: TestCalculateScoreWithOnChainOnly

**Purpose**: Test graceful degradation

**Test Steps**:
1. Configure failing off-chain aggregator
2. Calculate score
3. Verify succeeds with on-chain only

**Expected**:
- No error
- Valid score generated

### Test 3.10: TestConcurrentScoreUpdates

**Purpose**: Test thread safety

**Test Steps**:
1. Launch 5 concurrent updates
2. Wait for completion
3. Verify final state consistent

**Expected**:
- No race conditions
- update_count >= 1

---

## 4. Integration Tests (`tests/integration_test.go`)

### Test 4.1: TestHealthEndpoint

**HTTP Request**:
```
GET /health
```

**Expected Response**:
- Status: 200 or 503
- Body: `{"status": "healthy", "components": {...}}`

### Test 4.2: TestUpdateCreditScoreEndToEnd

**HTTP Request**:
```
POST /api/v1/credit-score/update
{
  "address": "0x1234...",
  "user_id": "user123",
  "publish": false
}
```

**Expected Response**:
- Status: 200
- Body includes: score, confidence, all component scores
- Score in range 300-850

### Test 4.3: TestGetCreditScoreEndToEnd

**HTTP Request**:
```
GET /api/v1/credit-score/0x1234...
```

**Expected Response**:
- Status: 200
- Body: Complete score object
- Address matches request

### Test 4.4: TestGetCreditScoreNotFound

**HTTP Request**:
```
GET /api/v1/credit-score/0xNonExistent
```

**Expected Response**:
- Status: 404
- Body: Error message

### Test 4.5: TestGetScoreHistoryEndToEnd

**HTTP Request**:
```
GET /api/v1/credit-score/0x1234.../history?limit=10
```

**Expected Response**:
- Status: 200
- Body: Array of history entries
- Each entry has: score, confidence, timestamp

### Test 4.6: TestGetStatsEndToEnd

**HTTP Request**:
```
GET /api/v1/admin/stats
```

**Expected Response**:
- Status: 200
- Body: Statistics object
- Correct counts

### Test 4.7: TestInvalidRequestHandling

**Test Cases**:

| # | Request | Expected Status | Reason |
|---|---------|----------------|--------|
| 4.7.1 | Invalid JSON | 400 | Parse error |
| 4.7.2 | Missing fields | 400 | Validation error |
| 4.7.3 | Invalid address | 404 | Not found |

### Test 4.8: TestFullWorkflow

**Purpose**: Complete end-to-end user journey

**Steps**:
1. Verify score doesn't exist (404)
2. Create score (200)
3. Retrieve score (200)
4. Update score (200, count=2)
5. Check history (200, 2 entries)
6. Verify in stats (200)

**Expected**:
- All steps succeed
- Data consistent across requests

### Test 4.9: TestConcurrentAPIRequests

**Purpose**: Load testing

**Test Steps**:
1. Send 10 concurrent POST requests
2. Wait for all completions
3. Verify final state

**Expected**:
- All requests succeed (200)
- No race conditions
- update_count >= 1

### Test 4.10: TestMockOnChainAggregator

**Purpose**: Verify mock data generation

**Expected**:
- Consistent mock data
- Realistic values

---

## Test Execution Commands

### Run All Tests
```bash
go test ./... -v
```

### Run Specific Package
```bash
go test ./internal/scoring -v
go test ./internal/repository -v
go test ./internal/service -v
go test ./tests -v
```

### Run Specific Test
```bash
go test ./internal/scoring -v -run TestCalculateScore
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

### Run Race Detection
```bash
go test ./... -race
```

---

## Expected Test Results

### Coverage Targets

| Package | Target | Current |
|---------|--------|---------|
| internal/scoring | >90% | TBD |
| internal/repository | >85% | TBD |
| internal/service | >80% | TBD |
| internal/api/handlers | >75% | TBD |
| **Overall** | **>80%** | **TBD** |

### Performance Targets

| Operation | Target | Benchmark |
|-----------|--------|-----------|
| Score calculation | <1ms | TBD |
| DB query | <10ms | TBD |
| API request | <50ms | TBD |

### Success Criteria

All tests must:
- ✓ Pass without errors
- ✓ Complete within timeout (default 10 minutes)
- ✓ No race conditions detected
- ✓ Meet coverage targets
- ✓ Meet performance targets

---

## Test Data

### Sample Addresses
```
0x1234567890123456789012345678901234567890 - Standard test address
0x1111111111111111111111111111111111111111 - Test address 1
0x2222222222222222222222222222222222222222 - Test address 2
0xNonExistent - Non-existent address for 404 tests
```

### Sample Scores
```
700 - Average score
300 - Minimum valid score
850 - Maximum valid score
720 - Typical good score
500 - Typical poor score
```

---

## Troubleshooting Test Failures

### Common Issues

**Issue**: "database is locked"
- **Cause**: SQLite concurrency
- **Solution**: Use `:memory:` database

**Issue**: Race condition detected
- **Cause**: Concurrent access without locking
- **Solution**: Review synchronization

**Issue**: Test timeout
- **Cause**: Infinite loop or blocking
- **Solution**: Check for deadlocks

**Issue**: Flaky tests
- **Cause**: Time-dependent logic
- **Solution**: Use fixed timestamps in tests

---

## Continuous Integration

Tests run automatically on:
- Every push to main
- Every pull request
- Nightly builds

CI Requirements:
- All tests must pass
- Coverage >= 80%
- No race conditions
- Build succeeds
