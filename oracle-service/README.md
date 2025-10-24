# Credit Score Oracle Service

A decentralized oracle service that calculates and publishes credit scores for blockchain addresses to smart contracts. This service aggregates on-chain DeFi activity with off-chain traditional credit data to generate comprehensive credit scores.

## Features

- **Multi-Source Data Aggregation**
  - On-chain metrics (wallet age, DeFi interactions, borrowing history)
  - Off-chain metrics (traditional credit scores, bank data, income verification)
  - Hybrid scoring combining multiple data sources

- **Sophisticated Scoring Engine**
  - Weighted scoring algorithm (40% on-chain, 40% off-chain, 20% hybrid)
  - Score range: 300-850 (aligned with traditional credit scores)
  - Confidence levels (0-100) based on data quality and recency

- **Blockchain Integration**
  - Publishes scores to smart contracts
  - Cryptographic signing and verification
  - Transaction tracking and confirmation monitoring

- **RESTful API**
  - Get credit scores by address
  - Calculate and update scores
  - Historical score tracking
  - Service statistics and health checks

- **Data Persistence**
  - PostgreSQL for production
  - Score history and audit trail
  - Metrics storage and retrieval

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│              OFF-CHAIN DATA SOURCES                      │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌─────────┐ │
│  │ Credit   │  │  Bank    │  │   DeFi   │  │ Wallet  │ │
│  │ Bureaus  │  │   APIs   │  │ Activity │  │ History │ │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘  └────┬────┘ │
└───────┼─────────────┼─────────────┼──────────────┼──────┘
        │             │             │              │
        └─────────────┼─────────────┼──────────────┘
                      │             │
        ┌─────────────▼─────────────▼──────────────┐
        │       ORACLE SERVICE                      │
        │  ┌──────────────────────────────────┐    │
        │  │  Data Aggregators                │    │
        │  │  - On-Chain  - Off-Chain         │    │
        │  └────────────┬─────────────────────┘    │
        │               │                           │
        │  ┌────────────▼─────────────────────┐    │
        │  │  Scoring Engine                  │    │
        │  │  - Calculate weighted score      │    │
        │  │  - Generate confidence level     │    │
        │  └────────────┬─────────────────────┘    │
        │               │                           │
        │  ┌────────────▼─────────────────────┐    │
        │  │  Repository Layer                │    │
        │  │  - Store scores & history        │    │
        │  └────────────┬─────────────────────┘    │
        │               │                           │
        │  ┌────────────▼─────────────────────┐    │
        │  │  Blockchain Publisher            │    │
        │  │  - Sign & submit to contract     │    │
        │  └──────────────────────────────────┘    │
        └───────────────┬───────────────────────────┘
                        │
        ┌───────────────▼───────────────────────────┐
        │         BLOCKCHAIN LAYER                   │
        │  ┌──────────────────────────────────┐     │
        │  │   CreditScoreOracle Contract     │     │
        │  │  - Store scores on-chain         │     │
        │  │  - Emit update events            │     │
        │  └──────────────────────────────────┘     │
        └────────────────────────────────────────────┘
```

## Project Structure

```
oracle-service/
├── cmd/
│   └── oracle/
│       └── main.go                 # Application entry point
├── internal/
│   ├── aggregator/                 # Data collection
│   │   ├── onchain.go             # Blockchain data fetcher
│   │   └── offchain.go            # External API integrations
│   ├── api/
│   │   ├── handlers/              # HTTP handlers
│   │   │   └── score_handler.go
│   │   └── routes/                # Route definitions
│   │       └── routes.go
│   ├── blockchain/                 # Blockchain interactions
│   │   └── oracle.go              # Smart contract interface
│   ├── config/                     # Configuration
│   │   └── config.go
│   ├── models/                     # Data models
│   │   └── credit_score.go
│   ├── repository/                 # Database layer
│   │   ├── score_repository.go
│   │   └── score_repository_test.go
│   ├── scoring/                    # Scoring algorithm
│   │   ├── engine.go
│   │   └── engine_test.go
│   └── service/                    # Business logic
│       ├── oracle_service.go
│       └── oracle_service_test.go
├── pkg/
│   └── logger/                     # Logging utilities
│       └── logger.go
├── tests/
│   └── integration_test.go         # Integration tests
├── go.mod
├── go.sum
├── README.md
└── TESTING.md                      # Testing documentation
```

## Installation

### Prerequisites

- Go 1.21 or higher
- PostgreSQL 13+ (optional, SQLite for development)
- Ethereum node access (Infura, Alchemy, or local node)

### Setup

1. Clone the repository:
```bash
cd oracle-service
```

2. Install dependencies:
```bash
go mod download
```

3. Configure environment variables:
```bash
cp .env.example .env
# Edit .env with your configuration
```

4. Run database migrations (if using PostgreSQL):
```bash
# Migrations will auto-run on startup
```

## Configuration

Create a `.env` file with the following variables:

```env
# Server Configuration
PORT=8080

# Database
DATABASE_URL=postgresql://user:password@localhost:5432/oracle_db
# Or use SQLite for development:
# DATABASE_URL=

# Blockchain
ETHEREUM_RPC_URL=https://mainnet.infura.io/v3/YOUR_PROJECT_ID
CONTRACT_ADDRESS=0x...
PRIVATE_KEY=your_private_key_without_0x

# External APIs (optional)
CREDIT_BUREAU_URL=https://api.creditbureau.com
BANK_API_URL=https://api.plaid.com
API_KEY=your_api_key

# Redis (optional, for caching)
REDIS_URL=redis://localhost:6379
```

### Real Data Sources Configuration

For production use with real-world data sources, configure these additional environment variables:

```env
# Provider Configuration
USE_MOCK_DATA=false

# Credit Bureau Configuration
CREDIT_BUREAU_PROVIDER=experian
CREDIT_BUREAU_URL=https://api.experian.com
CREDIT_BUREAU_API_KEY=your_credit_bureau_api_key

# Plaid Configuration (Bank Data)
PLAID_CLIENT_ID=your_plaid_client_id
PLAID_SECRET=your_plaid_secret
PLAID_ENV=sandbox

# Blockchain Data Providers
COVALENT_API_KEY=your_covalent_api_key
COVALENT_BASE_URL=https://api.covalenthq.com/v1

MORALIS_API_KEY=your_moralis_api_key
MORALIS_BASE_URL=https://deep-index.moralis.io/api/v2

# Blockscout Configuration (Preferred - Free)
BLOCKSCOUT_BASE_URL=https://eth.blockscout.com
BLOCKSCOUT_CHAIN=ethereum
PREFER_BLOCKSCOUT=true
```

**Free Data Sources:**
- **Blockscout**: Free blockchain data (no API key required)
- **Public RPC endpoints**: Some free tiers available
- **Mock data**: Set `USE_MOCK_DATA=true` for testing

## Usage

### Running the Service

```bash
# Development mode
go run cmd/oracle/main.go

# Production build
go build -o oracle cmd/oracle/main.go
./oracle
```

### API Endpoints

#### Get Credit Score
```bash
GET /api/v1/credit-score/:address

curl http://localhost:8080/api/v1/credit-score/0x1234567890123456789012345678901234567890
```

Response:
```json
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
  "update_count": 3
}
```

#### Update Credit Score
```bash
POST /api/v1/credit-score/update

curl -X POST http://localhost:8080/api/v1/credit-score/update \
  -H "Content-Type: application/json" \
  -d '{
    "address": "0x1234567890123456789012345678901234567890",
    "user_id": "user123",
    "publish": true
  }'
```

#### Get Score History
```bash
GET /api/v1/credit-score/:address/history?limit=10

curl http://localhost:8080/api/v1/credit-score/0x1234.../history?limit=10
```

#### Get Service Statistics
```bash
GET /api/v1/admin/stats

curl http://localhost:8080/api/v1/admin/stats
```

Response:
```json
{
  "total_active_scores": 1523,
  "average_score": 685.4,
  "due_for_update": 43,
  "pending_oracle_updates": 5
}
```

#### Health Check
```bash
GET /health

curl http://localhost:8080/health
```

## Testing

### Run All Tests
```bash
go test ./... -v
```

### Run with Coverage
```bash
go test ./... -cover -coverprofile=coverage.out
go tool cover -html=coverage.out
```

### Run Integration Tests
```bash
go test ./tests -v
```

### Run Benchmarks
```bash
go test ./internal/scoring -bench=. -benchmem
```

For detailed testing information, see [TESTING.md](TESTING.md).

## Scoring Algorithm

### Weights
- **On-Chain Metrics**: 40%
  - Wallet age: 25%
  - Transaction activity: 20%
  - DeFi interactions: 15%
  - Borrowing/Repayment history: 30%
  - Collateral holdings: 10%

- **Off-Chain Metrics**: 40%
  - Traditional credit score: 50%
  - Bank account history: 20%
  - Income verification: 15%
  - Debt-to-income ratio: 15%

- **Hybrid Metrics**: 20%
  - Cross-verification bonuses
  - Activity recency
  - Employment stability

### Score Range
- Minimum: 300
- Maximum: 850
- Aligned with FICO scoring for familiarity

### Confidence Calculation
Confidence level (0-100) based on:
- Data recency
- Data completeness
- Multiple source verification
- Historical data availability

## Deployment

### Docker

```bash
# Build image
docker build -t oracle-service .

# Run container
docker run -p 8080:8080 --env-file .env oracle-service
```

### Docker Compose

```bash
docker-compose up -d
```

### Production Considerations

1. **Security**
   - Store private keys in secure key management system (AWS KMS, HashiCorp Vault)
   - Use HTTPS/TLS for all API endpoints
   - Implement rate limiting
   - Add authentication/authorization

2. **Scalability**
   - Use connection pooling for database
   - Implement Redis caching for frequent queries
   - Deploy multiple instances behind load balancer
   - Use message queue for blockchain publishing

3. **Monitoring**
   - Set up logging aggregation (ELK, Datadog)
   - Configure alerting for failures
   - Track metrics (Prometheus, Grafana)
   - Monitor blockchain transaction status

4. **Reliability**
   - Implement retry logic for external API calls
   - Use circuit breakers for failing services
   - Set up database replication
   - Regular backups

## Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests
5. Submit a pull request

## License

MIT License - see LICENSE file for details

## Support

For questions or issues:
- Open an issue on GitHub
- Contact: support@example.com

## Roadmap

- [ ] Multi-chain support (Polygon, Arbitrum, Optimism)
- [ ] Zero-knowledge proofs for privacy
- [ ] Decentralized oracle node network
- [ ] Additional data sources integration
- [ ] Machine learning-based scoring
- [ ] Real-time score updates via WebSocket
- [ ] Advanced fraud detection
- [ ] Dispute resolution mechanism
