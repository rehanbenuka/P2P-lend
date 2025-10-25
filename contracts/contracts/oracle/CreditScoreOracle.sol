// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import "@openzeppelin/contracts/access/AccessControl.sol";
import "@openzeppelin/contracts/utils/Pausable.sol";
import "@openzeppelin/contracts/utils/ReentrancyGuard.sol";
import "../interfaces/ICreditScoreOracle.sol";

/**
 * @title CreditScoreOracle
 * @dev Oracle contract specifically designed for credit scoring in P2P lending
 * @dev Implements credit score storage, risk assessment, and data validation
 */
contract CreditScoreOracle is ICreditScoreOracle, AccessControl, Pausable, ReentrancyGuard {
    // Role for oracle operators who can update credit scores
    bytes32 public constant ORACLE_OPERATOR_ROLE = keccak256("ORACLE_OPERATOR_ROLE");
    
    // Role for admin who can manage the contract
    bytes32 public constant ADMIN_ROLE = keccak256("ADMIN_ROLE");

    // Constants for credit scoring
    uint256 public constant MAX_CREDIT_SCORE = 850;
    uint256 public constant MIN_CREDIT_SCORE = 300;
    uint8 public constant MAX_RISK_LEVEL = 5;
    uint8 public constant MIN_RISK_LEVEL = 1;

    // Maximum age for credit data before it's considered stale (in seconds)
    uint256 public maxDataAge;

    // Credit score data structure
    struct CreditData {
        uint256 creditScore;
        uint8 riskLevel;
        bytes additionalData;
        uint256 lastUpdated;
        bool isValid;
    }

    // Storage mappings
    mapping(address => CreditData) private creditScores;
    mapping(address => bytes32) private dataHashes;

    // Additional events
    event MaxDataAgeUpdated(uint256 oldAge, uint256 newAge);

    /**
     * @dev Constructor
     * @param admin The address that will have admin role
     * @param maxAge Maximum age for credit data before it's considered stale
     */
    constructor(address admin, uint256 maxAge) {
        _grantRole(ADMIN_ROLE, admin);
        _grantRole(ORACLE_OPERATOR_ROLE, admin);
        _setRoleAdmin(ORACLE_OPERATOR_ROLE, ADMIN_ROLE);
        maxDataAge = maxAge;
    }

    /**
     * @dev Updates credit score for a user
     * @param userAddress The address of the user
     * @param creditScore The credit score (300-850)
     * @param riskLevel The risk level (1-5)
     * @param additionalData Additional credit data as bytes
     */
    function updateCreditScore(
        address userAddress,
        uint256 creditScore,
        uint8 riskLevel,
        bytes calldata additionalData
    ) external override onlyRole(ORACLE_OPERATOR_ROLE) whenNotPaused nonReentrant {
        require(userAddress != address(0), "CreditScoreOracle: Invalid user address");
        require(
            creditScore >= MIN_CREDIT_SCORE && creditScore <= MAX_CREDIT_SCORE,
            "CreditScoreOracle: Invalid credit score range"
        );
        require(
            riskLevel >= MIN_RISK_LEVEL && riskLevel <= MAX_RISK_LEVEL,
            "CreditScoreOracle: Invalid risk level range"
        );

        // Calculate data hash for additional data
        bytes32 dataHash = keccak256(abi.encodePacked(additionalData, block.timestamp));

        // Update credit data
        creditScores[userAddress] = CreditData({
            creditScore: creditScore,
            riskLevel: riskLevel,
            additionalData: additionalData,
            lastUpdated: block.timestamp,
            isValid: true
        });

        dataHashes[userAddress] = dataHash;

        emit CreditScoreUpdated(userAddress, creditScore, riskLevel, block.timestamp);
        emit CreditDataUpdated(userAddress, dataHash, block.timestamp);
    }

    /**
     * @dev Gets the credit score for a user
     * @param userAddress The address of the user
     * @return creditScore The credit score
     * @return riskLevel The risk level
     * @return lastUpdated The timestamp of last update
     */
    function getCreditScore(address userAddress)
        external
        view
        override
        returns (uint256 creditScore, uint8 riskLevel, uint256 lastUpdated)
    {
        require(hasValidCreditScore(userAddress), "CreditScoreOracle: No valid credit score found");
        
        CreditData memory data = creditScores[userAddress];
        return (data.creditScore, data.riskLevel, data.lastUpdated);
    }

    /**
     * @dev Gets additional credit data for a user
     * @param userAddress The address of the user
     * @return data The additional credit data
     * @return lastUpdated The timestamp of last update
     */
    function getCreditData(address userAddress)
        external
        view
        override
        returns (bytes memory data, uint256 lastUpdated)
    {
        require(hasValidCreditScore(userAddress), "CreditScoreOracle: No valid credit score found");
        
        CreditData memory creditData = creditScores[userAddress];
        return (creditData.additionalData, creditData.lastUpdated);
    }

    /**
     * @dev Checks if a user has a valid credit score
     * @param userAddress The address of the user
     * @return hasScore True if user has a valid credit score
     */
    function hasValidCreditScore(address userAddress) public view override returns (bool hasScore) {
        CreditData memory data = creditScores[userAddress];
        return data.isValid && !isCreditDataStale(userAddress);
    }

    /**
     * @dev Gets the maximum credit score possible
     * @return maxScore The maximum credit score
     */
    function getMaxCreditScore() external pure override returns (uint256 maxScore) {
        return MAX_CREDIT_SCORE;
    }

    /**
     * @dev Gets the minimum credit score possible
     * @return minScore The minimum credit score
     */
    function getMinCreditScore() external pure override returns (uint256 minScore) {
        return MIN_CREDIT_SCORE;
    }

    /**
     * @dev Checks if credit data is stale based on maxDataAge
     * @param userAddress The address of the user
     * @return stale True if credit data is stale
     */
    function isCreditDataStale(address userAddress) public view returns (bool stale) {
        CreditData memory data = creditScores[userAddress];
        if (!data.isValid) return true;
        return block.timestamp - data.lastUpdated > maxDataAge;
    }

    /**
     * @dev Gets the risk level for a user
     * @param userAddress The address of the user
     * @return riskLevel The risk level (1-5)
     */
    function getRiskLevel(address userAddress) external view returns (uint8 riskLevel) {
        require(hasValidCreditScore(userAddress), "CreditScoreOracle: No valid credit score found");
        return creditScores[userAddress].riskLevel;
    }

    /**
     * @dev Gets comprehensive credit information for a user
     * @param userAddress The address of the user
     * @return creditScore The credit score
     * @return riskLevel The risk level
     * @return lastUpdated The timestamp of last update
     * @return isStale True if data is stale
     * @return dataHash The hash of additional data
     */
    function getCreditInfo(address userAddress)
        external
        view
        returns (
            uint256 creditScore,
            uint8 riskLevel,
            uint256 lastUpdated,
            bool isStale,
            bytes32 dataHash
        )
    {
        require(hasValidCreditScore(userAddress), "CreditScoreOracle: No valid credit score found");
        
        CreditData memory data = creditScores[userAddress];
        return (
            data.creditScore,
            data.riskLevel,
            data.lastUpdated,
            isCreditDataStale(userAddress),
            dataHashes[userAddress]
        );
    }

    // IOracle interface implementations
    function updateData(bytes32 dataId, bytes calldata data) external override {
        // This function is not used in CreditScoreOracle
        // Use updateCreditScore instead
        revert("CreditScoreOracle: Use updateCreditScore instead of updateData");
    }

    function getData(bytes32 dataId) external view override returns (bytes memory, uint256) {
        // This function is not used in CreditScoreOracle
        revert("CreditScoreOracle: Use getCreditScore instead of getData");
    }

    function hasData(bytes32 dataId) external view override returns (bool) {
        // This function is not used in CreditScoreOracle
        revert("CreditScoreOracle: Use hasValidCreditScore instead of hasData");
    }

    function getLastUpdateTime(bytes32 dataId) external view override returns (uint256) {
        // This function is not used in CreditScoreOracle
        revert("CreditScoreOracle: Use getCreditScore instead of getLastUpdateTime");
    }

    /**
     * @dev Updates the maximum data age
     * @param newMaxAge The new maximum age in seconds
     */
    function setMaxDataAge(uint256 newMaxAge) external onlyRole(ADMIN_ROLE) {
        require(newMaxAge > 0, "CreditScoreOracle: Max data age must be greater than 0");
        uint256 oldAge = maxDataAge;
        maxDataAge = newMaxAge;
        emit MaxDataAgeUpdated(oldAge, newMaxAge);
    }

    /**
     * @dev Pauses the contract
     */
    function pause() external onlyRole(ADMIN_ROLE) {
        _pause();
    }

    /**
     * @dev Unpauses the contract
     */
    function unpause() external onlyRole(ADMIN_ROLE) {
        _unpause();
    }

    /**
     * @dev Grants oracle operator role to an address
     * @param operator The address to grant the role to
     */
    function grantOracleOperatorRole(address operator) external onlyRole(ADMIN_ROLE) {
        grantRole(ORACLE_OPERATOR_ROLE, operator);
    }

    /**
     * @dev Revokes oracle operator role from an address
     * @param operator The address to revoke the role from
     */
    function revokeOracleOperatorRole(address operator) external onlyRole(ADMIN_ROLE) {
        revokeRole(ORACLE_OPERATOR_ROLE, operator);
    }
}
