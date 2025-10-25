// SPDX-License-Identifier: MIT
pragma solidity ^0.8.20;

import "@openzeppelin/contracts/access/AccessControl.sol";
import "@openzeppelin/contracts/utils/Pausable.sol";
import "@openzeppelin/contracts/utils/ReentrancyGuard.sol";
import "../interfaces/IOracle.sol";

/**
 * @title Oracle
 * @dev A general-purpose oracle contract for storing and retrieving external data
 * @dev Implements access control, pausable functionality, and reentrancy protection
 */
contract Oracle is IOracle, AccessControl, Pausable, ReentrancyGuard {
    // Role for oracle operators who can update data
    bytes32 public constant ORACLE_OPERATOR_ROLE = keccak256("ORACLE_OPERATOR_ROLE");
    
    // Role for admin who can manage the contract
    bytes32 public constant ADMIN_ROLE = keccak256("ADMIN_ROLE");

    // Maximum age for data before it's considered stale (in seconds)
    uint256 public maxDataAge;

    // Data storage mapping
    mapping(bytes32 => bytes) private dataStorage;
    mapping(bytes32 => uint256) private dataTimestamps;
    mapping(string => bool) private registeredDataTypes;

    // Additional events
    event MaxDataAgeUpdated(uint256 oldAge, uint256 newAge);

    /**
     * @dev Constructor
     * @param admin The address that will have admin role
     * @param maxAge Maximum age for data before it's considered stale
     */
    constructor(address admin, uint256 maxAge) {
        _grantRole(ADMIN_ROLE, admin);
        _grantRole(ORACLE_OPERATOR_ROLE, admin);
        _setRoleAdmin(ORACLE_OPERATOR_ROLE, ADMIN_ROLE);
        maxDataAge = maxAge;
    }

    /**
     * @dev Updates data for a specific identifier
     * @param dataId The unique identifier for the data
     * @param data The data to store
     */
    function updateData(bytes32 dataId, bytes calldata data) 
        external 
        override 
        onlyRole(ORACLE_OPERATOR_ROLE) 
        whenNotPaused 
        nonReentrant 
    {
        require(dataId != bytes32(0), "Oracle: Invalid data ID");
        require(data.length > 0, "Oracle: Data cannot be empty");

        dataStorage[dataId] = data;
        dataTimestamps[dataId] = block.timestamp;

        emit DataUpdated(dataId, data, block.timestamp);
    }

    /**
     * @dev Retrieves the latest data for a specific identifier
     * @param dataId The unique identifier for the data
     * @return data The stored data
     * @return timestamp The timestamp when data was last updated
     */
    function getData(bytes32 dataId) 
        external 
        view 
        override 
        returns (bytes memory data, uint256 timestamp) 
    {
        require(hasData(dataId), "Oracle: Data not found");
        return (dataStorage[dataId], dataTimestamps[dataId]);
    }

    /**
     * @dev Checks if data exists for a specific identifier
     * @param dataId The unique identifier for the data
     * @return exists True if data exists, false otherwise
     */
    function hasData(bytes32 dataId) public view override returns (bool exists) {
        return dataTimestamps[dataId] > 0;
    }

    /**
     * @dev Gets the timestamp of the last update for a specific identifier
     * @param dataId The unique identifier for the data
     * @return timestamp The timestamp of the last update
     */
    function getLastUpdateTime(bytes32 dataId) 
        external 
        view 
        override 
        returns (uint256 timestamp) 
    {
        require(hasData(dataId), "Oracle: Data not found");
        return dataTimestamps[dataId];
    }

    /**
     * @dev Registers a new data type
     * @param dataType The data type identifier
     * @param description Description of the data type
     */
    function registerDataType(string calldata dataType, string calldata description) 
        external 
        onlyRole(ADMIN_ROLE) 
    {
        require(bytes(dataType).length > 0, "Oracle: Data type cannot be empty");
        require(!registeredDataTypes[dataType], "Oracle: Data type already registered");

        registeredDataTypes[dataType] = true;
        emit DataTypeRegistered(dataType, description);
    }

    /**
     * @dev Checks if a data type is registered
     * @param dataType The data type identifier
     * @return registered True if data type is registered
     */
    function isDataTypeRegistered(string calldata dataType) external view returns (bool registered) {
        return registeredDataTypes[dataType];
    }

    /**
     * @dev Checks if data is stale based on maxDataAge
     * @param dataId The unique identifier for the data
     * @return stale True if data is stale
     */
    function isDataStale(bytes32 dataId) external view returns (bool stale) {
        if (!hasData(dataId)) return true;
        return block.timestamp - dataTimestamps[dataId] > maxDataAge;
    }

    /**
     * @dev Updates the maximum data age
     * @param newMaxAge The new maximum age in seconds
     */
    function setMaxDataAge(uint256 newMaxAge) external onlyRole(ADMIN_ROLE) {
        require(newMaxAge > 0, "Oracle: Max data age must be greater than 0");
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
