# ✅ Oracle Service Setup Complete

## What's Been Done

### 1. In-Memory Database Configuration ✓
- **Configured** in-memory SQLite database (no PostgreSQL required)
- **Updated** `.env` file to use in-memory mode
- **Added** automatic database migration on startup
- **Works** without any external database server

### 2. Code Updates
- Updated `internal/api/routes/routes.go`:
  - Added SQLite driver import
  - Added models import for auto-migration
  - Enhanced `initDatabase()` function to:
    - Use in-memory SQLite when `DATABASE_URL` is empty
    - Auto-migrate all models on startup
    - Support both SQLite and PostgreSQL

### 3. Database Models Auto-Migration
The following models are automatically created on startup:
- ✅ CreditScore
- ✅ ScoreHistory
- ✅ OnChainMetrics
- ✅ OffChainMetrics
- ✅ OracleUpdate

## How to Run

### Quick Start (Development Mode)

```bash
cd oracle-service

# Install dependencies (if not already done)
go mod tidy

# Run the service
go run cmd/oracle/main.go
```

**Expected Output:**
```
{"level":"info","timestamp":"...","msg":"No database URL configured, using in-memory SQLite"}
{"level":"info","timestamp":"...","msg":"Database initialized successfully"}
{"level":"info","timestamp":"...","msg":"Starting oracle service on port 8080"}
```

### Test the Service

```bash
# 1. Health check
curl http://localhost:8080/health

# 2. Calculate a credit score
curl -X POST http://localhost:8080/api/v1/credit-score/update \
  -H "Content-Type: application/json" \
  -d '{
    "address": "0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb",
    "user_id": "test_user"
  }'

# 3. Retrieve the score
curl http://localhost:8080/api/v1/credit-score/0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb

# 4. Check service stats
curl http://localhost:8080/api/v1/admin/stats
```

### Run Tests

```bash
cd oracle-service

# Run all tests
go test ./... -v

# Run with coverage
go test ./... -cover

# Run specific test package
go test ./internal/scoring -v
go test ./tests -v
```

## Configuration

### Current Setup (.env)

```env
# Database - Empty = In-Memory SQLite
DATABASE_URL=

# Server
PORT=8080

# Blockchain (optional)
ETHEREUM_RPC_URL=http://localhost:8545
CONTRACT_ADDRESS=0x0000000000000000000000000000000000000000
PRIVATE_KEY=
```

### Switching to PostgreSQL (Production)

If you want to use PostgreSQL later:

1. Start PostgreSQL server
2. Create database:
   ```sql
   CREATE DATABASE p2p_lend;
   ```
3. Update `.env`:
   ```env
   DATABASE_URL=postgres://user:password@localhost:5432/p2p_lend?sslmode=disable
   ```
4. Restart service - migrations run automatically!

## Database Features

### In-Memory Mode (Current)
- ✅ No installation required
- ✅ Fast performance
- ✅ Perfect for development/testing
- ✅ Auto-reset on restart
- ✅ No external dependencies

### PostgreSQL Mode (Production)
- ✅ Persistent storage
- ✅ Production-ready
- ✅ Concurrent access
- ✅ Backup/restore support
- ✅ Same code, just change DATABASE_URL

## Architecture

```
Oracle Service
├── In-Memory SQLite Database
│   ├── credit_scores (user scores)
│   ├── score_histories (historical data)
│   ├── on_chain_metrics (blockchain data)
│   ├── off_chain_metrics (external data)
│   └── oracle_updates (blockchain transactions)
│
├── API Endpoints
│   ├── GET  /health
│   ├── GET  /api/v1/credit-score/:address
│   ├── POST /api/v1/credit-score/update
│   ├── GET  /api/v1/credit-score/:address/history
│   └── GET  /api/v1/admin/stats
│
└── Services
    ├── Scoring Engine (calculates credit scores)
    ├── On-Chain Aggregator (fetches blockchain data)
    ├── Off-Chain Aggregator (fetches external data)
    └── Blockchain Publisher (publishes to smart contracts)
```

## What Happens on Startup

1. **Load Environment Variables** from `.env`
2. **Initialize Logger** (structured logging with Zap)
3. **Check DATABASE_URL**:
   - Empty → Use in-memory SQLite
   - Set → Connect to PostgreSQL
4. **Auto-Migrate Models** (create all tables)
5. **Initialize Services**:
   - On-chain aggregator
   - Off-chain aggregator
   - Scoring engine
   - Blockchain client (if configured)
6. **Setup API Routes**
7. **Start HTTP Server** on port 8080

## Troubleshooting

### Issue: Service won't start
**Solution**: Make sure you're in the `oracle-service` directory:
```bash
cd /mnt/d/projects/P2P-lend/oracle-service
go run cmd/oracle/main.go
```

### Issue: Tests fail
**Solution**: Run from the correct directory:
```bash
cd oracle-service
go test ./... -v
```

### Issue: Port 8080 in use
**Solution**: Change port in `.env`:
```env
PORT=8081
```

### Issue: Database connection error
**Solution**: Verify `.env` has empty DATABASE_URL:
```env
DATABASE_URL=
```

## Data Persistence

### In-Memory Mode (Current)
- Data is lost when service restarts
- Perfect for development/testing
- Each restart gives you a fresh database

### Want Persistent Data?
1. Use SQLite file instead of memory:
   ```env
   DATABASE_URL=sqlite:./oracle.db
   ```
2. Or use PostgreSQL (see configuration above)

## Next Steps

✅ **Service is ready to use!**

1. **Start the service**: `go run cmd/oracle/main.go`
2. **Run tests**: `go test ./... -v`
3. **Test API**: Use the curl commands above
4. **Read docs**: Check README.md, TESTING.md, QUICKSTART.md

## Files Modified

- ✅ `internal/api/routes/routes.go` - Added in-memory DB support
- ✅ `.env` - Configured for in-memory mode
- ✅ `go.mod` - Already has SQLite driver (v1.6.0)

## Summary

🎉 **Everything is configured and ready!**

- No PostgreSQL required
- No Redis required
- No external blockchain required
- Just run: `go run cmd/oracle/main.go`

The service will:
- Start on http://localhost:8080
- Use in-memory SQLite database
- Auto-migrate all tables
- Be ready to accept API requests

**Happy coding!** 🚀
