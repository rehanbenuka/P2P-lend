# P2P Lend Oracle Contracts

A comprehensive oracle system for P2P lending built with Hardhat 3, featuring credit score management and external data publishing capabilities.

## 🚀 Features

- **Credit Score Oracle**: Specialized contract for managing user credit scores and risk assessments
- **General Oracle**: Flexible contract for storing and retrieving any type of external data
- **Access Control**: Role-based permissions for secure data management
- **Data Staleness Detection**: Automatic detection of outdated data
- **Pausable Operations**: Emergency pause functionality
- **Gas Optimized**: Efficient storage and retrieval mechanisms
- **Hardhat 3 Compatible**: Built with the latest Hardhat features

## 📋 Contract Overview

### Oracle.sol
General-purpose oracle contract for storing and retrieving external data:
- Store any type of data with unique identifiers
- Data type registration system
- Staleness detection based on configurable time limits
- Role-based access control

### CreditScoreOracle.sol
Specialized oracle for credit scoring in P2P lending:
- Credit score storage (300-850 range)
- Risk level assessment (1-5 scale)
- Additional credit data storage
- Credit score validation and staleness checks

## 🛠️ Installation

1. Install dependencies:
```bash
npm install
```

2. Copy environment variables:
```bash
cp .env.example .env
```

3. Update `.env` with your configuration:
   - Add your private key
   - Add RPC URLs for your target networks
   - Add API keys for verification and gas reporting

## 🧪 Testing

Run the test suite:
```bash
# Run all tests
npm test

# Run unit tests only
npx hardhat test test/unit/

# Run integration tests only
npx hardhat test test/integration/

# Run with gas reporting
REPORT_GAS=true npm test
```

## 🚀 Deployment

### Local Development
```bash
# Start local node
npm run node

# Deploy to local network
npm run deploy:local
```

### Sepolia Testnet
```bash
# Deploy to Sepolia
npm run deploy:sepolia
```

## 📊 Contract Architecture

```
Oracle System
├── Oracle.sol (General Data Storage)
│   ├── Data storage and retrieval
│   ├── Data type registration
│   └── Staleness detection
│
└── CreditScoreOracle.sol (Credit Scoring)
    ├── Credit score management
    ├── Risk level assessment
    └── Credit data validation
```

## 🔐 Access Control

### Roles
- **ADMIN_ROLE**: Full contract management permissions
- **ORACLE_OPERATOR_ROLE**: Data update permissions

### Functions
- `grantOracleOperatorRole(address)`: Grant operator permissions
- `revokeOracleOperatorRole(address)`: Revoke operator permissions
- `pause()`: Pause all operations
- `unpause()`: Resume operations

## 📈 Usage Examples

### Updating Credit Scores
```javascript
// Update user credit score
await creditScoreOracle.updateCreditScore(
  userAddress,
  750,        // Credit score (300-850)
  2,          // Risk level (1-5)
  additionalData // Additional credit information
);
```

### Retrieving Credit Data
```javascript
// Get credit score
const [score, risk, timestamp] = await creditScoreOracle.getCreditScore(userAddress);

// Check if data is valid
const isValid = await creditScoreOracle.hasValidCreditScore(userAddress);

// Get comprehensive info
const [score, risk, timestamp, isStale, dataHash] = await creditScoreOracle.getCreditInfo(userAddress);
```

### General Data Storage
```javascript
// Store data
const dataId = ethers.keccak256(ethers.toUtf8Bytes("unique-id"));
await oracle.updateData(dataId, data);

// Retrieve data
const [data, timestamp] = await oracle.getData(dataId);
```

## 🔧 Configuration

### Data Staleness
- Default max data age: 7 days
- Configurable via `setMaxDataAge(uint256)`
- Automatic staleness detection

### Credit Score Ranges
- Minimum: 300
- Maximum: 850
- Risk levels: 1-5 (1 = lowest risk)

## 🛡️ Security Features

- **Reentrancy Protection**: All external calls are protected
- **Access Control**: Role-based permissions
- **Input Validation**: Comprehensive parameter validation
- **Pausable Operations**: Emergency stop functionality
- **Data Integrity**: Hash-based data verification

## 📝 Events

### Oracle Events
- `DataUpdated(bytes32 indexed dataId, bytes data, uint256 timestamp)`
- `DataTypeRegistered(string indexed dataType, string description)`
- `MaxDataAgeUpdated(uint256 oldAge, uint256 newAge)`

### CreditScoreOracle Events
- `CreditScoreUpdated(address indexed userAddress, uint256 creditScore, uint8 riskLevel, uint256 timestamp)`
- `CreditDataUpdated(address indexed userAddress, bytes32 indexed dataHash, uint256 timestamp)`

## 🚀 Hardhat 3 Features

This project leverages Hardhat 3's new capabilities:
- **Enhanced Testing**: Improved test runner and debugging
- **Better Compilation**: Via IR optimization for gas efficiency
- **Multichain Support**: Easy deployment across different networks
- **Modern CLI**: Streamlined command interface
- **Performance**: Rust-based components for critical operations

## 📄 License

MIT License - see LICENSE file for details

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch
3. Make your changes
4. Add tests for new functionality
5. Submit a pull request

## 📞 Support

For questions or support, please open an issue in the repository.
