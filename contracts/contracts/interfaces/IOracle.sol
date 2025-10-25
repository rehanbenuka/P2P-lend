// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

/**
 * @title IOracle
 * @dev Interface for oracle contracts that provide external data
 */
interface IOracle {
    /**
     * @dev Emitted when data is updated
     * @param dataId The unique identifier for the data
     * @param data The updated data
     * @param timestamp The block timestamp when data was updated
     */
    event DataUpdated(bytes32 indexed dataId, bytes data, uint256 timestamp);

    /**
     * @dev Emitted when a new data type is registered
     * @param dataType The data type identifier
     * @param description Description of the data type
     */
    event DataTypeRegistered(string indexed dataType, string description);

    /**
     * @dev Updates data for a specific identifier
     * @param dataId The unique identifier for the data
     * @param data The data to store
     */
    function updateData(bytes32 dataId, bytes calldata data) external;

    /**
     * @dev Retrieves the latest data for a specific identifier
     * @param dataId The unique identifier for the data
     * @return data The stored data
     * @return timestamp The timestamp when data was last updated
     */
    function getData(bytes32 dataId) external view returns (bytes memory data, uint256 timestamp);

    /**
     * @dev Checks if data exists for a specific identifier
     * @param dataId The unique identifier for the data
     * @return exists True if data exists, false otherwise
     */
    function hasData(bytes32 dataId) external view returns (bool exists);

    /**
     * @dev Gets the timestamp of the last update for a specific identifier
     * @param dataId The unique identifier for the data
     * @return timestamp The timestamp of the last update
     */
    function getLastUpdateTime(bytes32 dataId) external view returns (uint256 timestamp);
}
