// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import "./IOracle.sol";

/**
 * @title ICreditScoreOracle
 * @dev Interface for credit score oracle that provides credit scoring data
 */
interface ICreditScoreOracle is IOracle {
    /**
     * @dev Emitted when a credit score is updated
     * @param userAddress The address of the user
     * @param creditScore The updated credit score
     * @param riskLevel The risk level (1-5, where 1 is lowest risk)
     * @param timestamp The block timestamp when score was updated
     */
    event CreditScoreUpdated(
        address indexed userAddress,
        uint256 creditScore,
        uint8 riskLevel,
        uint256 timestamp
    );

    /**
     * @dev Emitted when a user's credit data is updated
     * @param userAddress The address of the user
     * @param dataHash The hash of the credit data
     * @param timestamp The block timestamp when data was updated
     */
    event CreditDataUpdated(
        address indexed userAddress,
        bytes32 indexed dataHash,
        uint256 timestamp
    );

    /**
     * @dev Updates credit score for a user
     * @param userAddress The address of the user
     * @param creditScore The credit score (0-850)
     * @param riskLevel The risk level (1-5)
     * @param additionalData Additional credit data as bytes
     */
    function updateCreditScore(
        address userAddress,
        uint256 creditScore,
        uint8 riskLevel,
        bytes calldata additionalData
    ) external;

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
        returns (uint256 creditScore, uint8 riskLevel, uint256 lastUpdated);

    /**
     * @dev Gets additional credit data for a user
     * @param userAddress The address of the user
     * @return data The additional credit data
     * @return lastUpdated The timestamp of last update
     */
    function getCreditData(address userAddress)
        external
        view
        returns (bytes memory data, uint256 lastUpdated);

    /**
     * @dev Checks if a user has a valid credit score
     * @param userAddress The address of the user
     * @return hasScore True if user has a valid credit score
     */
    function hasValidCreditScore(address userAddress) external view returns (bool hasScore);

    /**
     * @dev Gets the maximum credit score possible
     * @return maxScore The maximum credit score
     */
    function getMaxCreditScore() external pure returns (uint256 maxScore);

    /**
     * @dev Gets the minimum credit score possible
     * @return minScore The minimum credit score
     */
    function getMinCreditScore() external pure returns (uint256 minScore);
}
