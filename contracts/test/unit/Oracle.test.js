const { expect } = require("chai");
const { ethers } = require("hardhat");

describe("Oracle", function () {
  let oracle;
  let admin;
  let operator;
  let user;
  let maxDataAge;

  beforeEach(async function () {
    [admin, operator, user] = await ethers.getSigners();
    maxDataAge = 7 * 24 * 60 * 60; // 7 days

    const Oracle = await ethers.getContractFactory("Oracle");
    oracle = await Oracle.deploy(admin.address, maxDataAge);
    await oracle.waitForDeployment();

    // Grant operator role to operator address
    await oracle.connect(admin).grantOracleOperatorRole(operator.address);
  });

  describe("Deployment", function () {
    it("Should set the correct admin role", async function () {
      expect(await oracle.hasRole(await oracle.ADMIN_ROLE(), admin.address)).to.be.true;
    });

    it("Should set the correct max data age", async function () {
      expect(await oracle.maxDataAge()).to.equal(maxDataAge);
    });

    it("Should be unpaused by default", async function () {
      expect(await oracle.paused()).to.be.false;
    });
  });

  describe("Data Management", function () {
    it("Should allow operator to update data", async function () {
      const dataId = ethers.keccak256(ethers.toUtf8Bytes("test-data"));
      const data = ethers.toUtf8Bytes("test data content");

      await expect(oracle.connect(operator).updateData(dataId, data))
        .to.emit(oracle, "DataUpdated")
        .withArgs(dataId, data, await oracle.provider.getBlock("latest").then(b => b.timestamp));
    });

    it("Should retrieve data correctly", async function () {
      const dataId = ethers.keccak256(ethers.toUtf8Bytes("test-data"));
      const data = ethers.toUtf8Bytes("test data content");

      await oracle.connect(operator).updateData(dataId, data);
      const [retrievedData, timestamp] = await oracle.getData(dataId);

      expect(ethers.toUtf8String(retrievedData)).to.equal("test data content");
      expect(timestamp).to.be.greaterThan(0);
    });

    it("Should check if data exists", async function () {
      const dataId = ethers.keccak256(ethers.toUtf8Bytes("test-data"));
      const data = ethers.toUtf8Bytes("test data content");

      expect(await oracle.hasData(dataId)).to.be.false;

      await oracle.connect(operator).updateData(dataId, data);

      expect(await oracle.hasData(dataId)).to.be.true;
    });

    it("Should reject empty data", async function () {
      const dataId = ethers.keccak256(ethers.toUtf8Bytes("test-data"));
      const emptyData = ethers.toUtf8Bytes("");

      await expect(oracle.connect(operator).updateData(dataId, emptyData))
        .to.be.revertedWith("Oracle: Data cannot be empty");
    });

    it("Should reject zero data ID", async function () {
      const data = ethers.toUtf8Bytes("test data content");

      await expect(oracle.connect(operator).updateData(ethers.ZeroHash, data))
        .to.be.revertedWith("Oracle: Invalid data ID");
    });
  });

  describe("Access Control", function () {
    it("Should reject non-operator from updating data", async function () {
      const dataId = ethers.keccak256(ethers.toUtf8Bytes("test-data"));
      const data = ethers.toUtf8Bytes("test data content");

      await expect(oracle.connect(user).updateData(dataId, data))
        .to.be.revertedWith(/AccessControl: account .* is missing role .*/);
    });

    it("Should allow admin to grant operator role", async function () {
      await oracle.connect(admin).grantOracleOperatorRole(user.address);
      expect(await oracle.hasRole(await oracle.ORACLE_OPERATOR_ROLE(), user.address)).to.be.true;
    });

    it("Should allow admin to revoke operator role", async function () {
      await oracle.connect(admin).revokeOracleOperatorRole(operator.address);
      expect(await oracle.hasRole(await oracle.ORACLE_OPERATOR_ROLE(), operator.address)).to.be.false;
    });
  });

  describe("Data Type Registration", function () {
    it("Should allow admin to register data types", async function () {
      await expect(oracle.connect(admin).registerDataType("CREDIT_SCORE", "Credit score data"))
        .to.emit(oracle, "DataTypeRegistered")
        .withArgs("CREDIT_SCORE", "Credit score data");
    });

    it("Should check if data type is registered", async function () {
      await oracle.connect(admin).registerDataType("CREDIT_SCORE", "Credit score data");
      expect(await oracle.isDataTypeRegistered("CREDIT_SCORE")).to.be.true;
      expect(await oracle.isDataTypeRegistered("UNKNOWN_TYPE")).to.be.false;
    });

    it("Should reject non-admin from registering data types", async function () {
      await expect(oracle.connect(operator).registerDataType("CREDIT_SCORE", "Credit score data"))
        .to.be.revertedWith(/AccessControl: account .* is missing role .*/);
    });
  });

  describe("Data Staleness", function () {
    it("Should detect stale data", async function () {
      const dataId = ethers.keccak256(ethers.toUtf8Bytes("test-data"));
      const data = ethers.toUtf8Bytes("test data content");

      await oracle.connect(operator).updateData(dataId, data);
      
      // Fast forward time beyond maxDataAge
      await ethers.provider.send("evm_increaseTime", [maxDataAge + 1]);
      await ethers.provider.send("evm_mine", []);

      expect(await oracle.isDataStale(dataId)).to.be.true;
    });

    it("Should not detect fresh data as stale", async function () {
      const dataId = ethers.keccak256(ethers.toUtf8Bytes("test-data"));
      const data = ethers.toUtf8Bytes("test data content");

      await oracle.connect(operator).updateData(dataId, data);
      expect(await oracle.isDataStale(dataId)).to.be.false;
    });
  });

  describe("Pausable", function () {
    it("Should allow admin to pause", async function () {
      await oracle.connect(admin).pause();
      expect(await oracle.paused()).to.be.true;
    });

    it("Should prevent data updates when paused", async function () {
      await oracle.connect(admin).pause();
      
      const dataId = ethers.keccak256(ethers.toUtf8Bytes("test-data"));
      const data = ethers.toUtf8Bytes("test data content");

      await expect(oracle.connect(operator).updateData(dataId, data))
        .to.be.revertedWith("Pausable: paused");
    });

    it("Should allow admin to unpause", async function () {
      await oracle.connect(admin).pause();
      await oracle.connect(admin).unpause();
      expect(await oracle.paused()).to.be.false;
    });
  });

  describe("Configuration", function () {
    it("Should allow admin to update max data age", async function () {
      const newMaxAge = 14 * 24 * 60 * 60; // 14 days
      
      await expect(oracle.connect(admin).setMaxDataAge(newMaxAge))
        .to.emit(oracle, "MaxDataAgeUpdated")
        .withArgs(maxDataAge, newMaxAge);
      
      expect(await oracle.maxDataAge()).to.equal(newMaxAge);
    });

    it("Should reject zero max data age", async function () {
      await expect(oracle.connect(admin).setMaxDataAge(0))
        .to.be.revertedWith("Oracle: Max data age must be greater than 0");
    });
  });
});
