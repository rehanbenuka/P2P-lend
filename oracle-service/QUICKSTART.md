# Oracle Service - Quick Start Guide

Get the Credit Score Oracle Service up and running in 5 minutes.

## Prerequisites

- Go 1.21+ installed
- Git installed
- (Optional) Docker installed

## Quick Setup

### 1. Navigate to the project

```bash
cd oracle-service
```

### 2. Install dependencies

```bash
go mod download
```

### 3. Run the service (development mode)

```bash
# Service will run without database (in-memory)
go run cmd/oracle/main.go
```

The service will start on `http://localhost:8080`

## Test the Service

### 1. Health Check

```bash
curl http://localhost:8080/health
```

Expected response:
```json
{
  "status": "healthy",
  "components": {
    "onchain_aggregator": true,
    "offchain_aggregator": true,
    "blockchain_client": true
  }
}
```

### 2. Calculate a Credit Score

```bash
curl -X POST http://localhost:8080/api/v1/credit-score/update \
  -H "Content-Type: application/json" \
  -d '{
    "address": "0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb",
    "user_id": "test_user_001",
    "publish": false
  }'
```

Expected response:
```json
{
  "address": "0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb",
  "score": 685,
  "confidence": 75,
  "on_chain_score": 650,
  "off_chain_score": 720,
  "hybrid_score": 685,
  "data_hash": "a7f3e9...",
  "last_updated": "2025-10-22T14:30:00Z",
  "next_update_due": "2025-11-21T14:30:00Z",
  "update_count": 1
}
```

### 3. Retrieve the Score

```bash
curl http://localhost:8080/api/v1/credit-score/0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb
```

### 4. View Score History

```bash
curl http://localhost:8080/api/v1/credit-score/0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb/history?limit=10
```

### 5. Get Service Statistics

```bash
curl http://localhost:8080/api/v1/admin/stats
```

## Run Tests

### Run all tests

```bash
go test ./... -v
```

### Run specific test package

```bash
# Scoring engine tests
go test ./internal/scoring -v

# Integration tests
go test ./tests -v
```

### Run with coverage

```bash
go test ./... -cover
```

## Configuration (Optional)

For production setup, create a `.env` file:

```bash
cat > .env << EOF
# Server
PORT=8080

# Database (optional, leave empty for in-memory)
DATABASE_URL=

# Blockchain (optional for testing)
ETHEREUM_RPC_URL=https://mainnet.infura.io/v3/YOUR_PROJECT_ID
CONTRACT_ADDRESS=0x0000000000000000000000000000000000000000
PRIVATE_KEY=your_private_key_here
EOF
```

## Docker Quick Start

### Build and run with Docker

```bash
# Build image
docker build -t oracle-service .

# Run container
docker run -p 8080:8080 oracle-service
```

### Using Docker Compose

```bash
docker-compose up
```

## API Endpoints Summary

| Method | Endpoint | Description |
|--------|----------|-------------|
| GET | `/health` | Health check |
| GET | `/api/v1/credit-score/:address` | Get credit score |
| POST | `/api/v1/credit-score/update` | Calculate/update score |
| GET | `/api/v1/credit-score/:address/history` | Get score history |
| GET | `/api/v1/admin/stats` | Get service statistics |

## Example Workflow

```bash
# 1. Create a credit score
ADDRESS="0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb"

curl -X POST http://localhost:8080/api/v1/credit-score/update \
  -H "Content-Type: application/json" \
  -d "{\"address\": \"$ADDRESS\", \"user_id\": \"user_001\"}"

# 2. Retrieve it
curl http://localhost:8080/api/v1/credit-score/$ADDRESS | jq

# 3. Update it
curl -X POST http://localhost:8080/api/v1/credit-score/update \
  -H "Content-Type: application/json" \
  -d "{\"address\": \"$ADDRESS\", \"user_id\": \"user_001\"}"

# 4. Check history
curl http://localhost:8080/api/v1/credit-score/$ADDRESS/history | jq

# 5. View stats
curl http://localhost:8080/api/v1/admin/stats | jq
```

## Understanding the Score

### Score Range: 300-850

| Range | Rating | Description |
|-------|--------|-------------|
| 800-850 | Excellent | Exceptional credit profile |
| 740-799 | Very Good | Strong creditworthiness |
| 670-739 | Good | Favorable credit terms |
| 580-669 | Fair | Below average |
| 300-579 | Poor | Significant credit risk |

### Scoring Components

- **On-Chain (40%)**: Wallet age, transaction history, DeFi activity, borrowing/repayment
- **Off-Chain (40%)**: Traditional credit score, bank history, income
- **Hybrid (20%)**: Cross-verification, activity patterns

### Confidence Level

- **90-100**: Excellent - Recent data from multiple sources
- **70-89**: Good - Sufficient data quality
- **50-69**: Fair - Some data gaps
- **0-49**: Poor - Limited or outdated data

## Troubleshooting

### Port already in use
```bash
# Change port in .env or use:
PORT=8081 go run cmd/oracle/main.go
```

### Tests failing
```bash
# Clean and reinstall dependencies
go clean -cache
go mod download
go test ./... -v
```

### Cannot connect to database
```bash
# For development, leave DATABASE_URL empty to use in-memory mode
# No database required for basic testing
```

## Next Steps

1. **Read the full documentation**: See [README.md](README.md)
2. **Explore test cases**: See [TEST_CASES.md](TEST_CASES.md)
3. **Configure for production**: Add proper environment variables
4. **Deploy to production**: Follow deployment guide in README.md

## Sample Test Scenarios

### Scenario 1: New User
```bash
# User with minimal on-chain activity
curl -X POST http://localhost:8080/api/v1/credit-score/update \
  -H "Content-Type: application/json" \
  -d '{
    "address": "0x1111111111111111111111111111111111111111",
    "user_id": "new_user"
  }'

# Expected: Score around 400-500, low confidence
```

### Scenario 2: Established User
```bash
# User with good on-chain history
curl -X POST http://localhost:8080/api/v1/credit-score/update \
  -H "Content-Type: application/json" \
  -d '{
    "address": "0x2222222222222222222222222222222222222222",
    "user_id": "established_user"
  }'

# Expected: Score around 650-750, medium-high confidence
```

### Scenario 3: Multiple Updates
```bash
ADDRESS="0x3333333333333333333333333333333333333333"

# First calculation
curl -X POST http://localhost:8080/api/v1/credit-score/update \
  -H "Content-Type: application/json" \
  -d "{\"address\": \"$ADDRESS\", \"user_id\": \"multi_update\"}"

# Second calculation (after some time)
sleep 1
curl -X POST http://localhost:8080/api/v1/credit-score/update \
  -H "Content-Type: application/json" \
  -d "{\"address\": \"$ADDRESS\", \"user_id\": \"multi_update\"}"

# Check update count
curl http://localhost:8080/api/v1/credit-score/$ADDRESS | jq '.update_count'
# Should show: 2
```

## Development Tips

### Live Reload
```bash
# Install air for live reload
go install github.com/cosmtrek/air@latest

# Run with air
air
```

### Debug Mode
```bash
# Run with verbose logging
GIN_MODE=debug go run cmd/oracle/main.go
```

### Test Individual Components
```bash
# Test scoring engine only
go test ./internal/scoring -v -run TestCalculateScore

# Test with coverage
go test ./internal/scoring -cover -coverprofile=coverage.out
go tool cover -html=coverage.out
```

## Support

- **Documentation**: See [README.md](README.md) and [TESTING.md](TESTING.md)
- **Issues**: Open an issue on GitHub
- **Testing**: See [TEST_CASES.md](TEST_CASES.md) for all test scenarios

## Quick Reference

### Common Commands
```bash
# Start service
go run cmd/oracle/main.go

# Run tests
go test ./... -v

# Run with coverage
go test ./... -cover

# Build for production
go build -o oracle cmd/oracle/main.go

# Run production binary
./oracle
```

### Environment Variables
```bash
PORT=8080                    # Server port
DATABASE_URL=                # Database connection (empty = in-memory)
ETHEREUM_RPC_URL=            # Blockchain RPC endpoint
CONTRACT_ADDRESS=            # Smart contract address
PRIVATE_KEY=                 # Oracle private key
```

---

**You're now ready to use the Oracle Service! 🚀**

For more detailed information, refer to:
- [README.md](README.md) - Complete documentation
- [TESTING.md](TESTING.md) - Testing guide
- [TEST_CASES.md](TEST_CASES.md) - All test cases
