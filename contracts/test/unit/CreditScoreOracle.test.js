const { expect } = require("chai");
const { ethers } = require("hardhat");

describe("CreditScoreOracle", function () {
  let creditScoreOracle;
  let admin;
  let operator;
  let user;
  let maxDataAge;

  beforeEach(async function () {
    [admin, operator, user] = await ethers.getSigners();
    maxDataAge = 7 * 24 * 60 * 60; // 7 days

    const CreditScoreOracle = await ethers.getContractFactory("CreditScoreOracle");
    creditScoreOracle = await CreditScoreOracle.deploy(admin.address, maxDataAge);
    await creditScoreOracle.waitForDeployment();

    // Grant operator role to operator address
    await creditScoreOracle.connect(admin).grantOracleOperatorRole(operator.address);
  });

  describe("Deployment", function () {
    it("Should set the correct admin role", async function () {
      expect(await creditScoreOracle.hasRole(await creditScoreOracle.ADMIN_ROLE(), admin.address)).to.be.true;
    });

    it("Should set the correct max data age", async function () {
      expect(await creditScoreOracle.maxDataAge()).to.equal(maxDataAge);
    });

    it("Should return correct min and max credit scores", async function () {
      expect(await creditScoreOracle.getMinCreditScore()).to.equal(300);
      expect(await creditScoreOracle.getMaxCreditScore()).to.equal(850);
    });
  });

  describe("Credit Score Management", function () {
    it("Should allow operator to update credit score", async function () {
      const creditScore = 750;
      const riskLevel = 2;
      const additionalData = ethers.toUtf8Bytes("Good payment history");

      await expect(creditScoreOracle.connect(operator).updateCreditScore(
        user.address,
        creditScore,
        riskLevel,
        additionalData
      ))
        .to.emit(creditScoreOracle, "CreditScoreUpdated")
        .withArgs(user.address, creditScore, riskLevel, await creditScoreOracle.provider.getBlock("latest").then(b => b.timestamp));
    });

    it("Should retrieve credit score correctly", async function () {
      const creditScore = 750;
      const riskLevel = 2;
      const additionalData = ethers.toUtf8Bytes("Good payment history");

      await creditScoreOracle.connect(operator).updateCreditScore(
        user.address,
        creditScore,
        riskLevel,
        additionalData
      );

      const [retrievedScore, retrievedRisk, lastUpdated] = await creditScoreOracle.getCreditScore(user.address);
      
      expect(retrievedScore).to.equal(creditScore);
      expect(retrievedRisk).to.equal(riskLevel);
      expect(lastUpdated).to.be.greaterThan(0);
    });

    it("Should retrieve credit data correctly", async function () {
      const creditScore = 750;
      const riskLevel = 2;
      const additionalData = ethers.toUtf8Bytes("Good payment history");

      await creditScoreOracle.connect(operator).updateCreditScore(
        user.address,
        creditScore,
        riskLevel,
        additionalData
      );

      const [retrievedData, lastUpdated] = await creditScoreOracle.getCreditData(user.address);
      
      expect(ethers.toUtf8String(retrievedData)).to.equal("Good payment history");
      expect(lastUpdated).to.be.greaterThan(0);
    });

    it("Should check if user has valid credit score", async function () {
      expect(await creditScoreOracle.hasValidCreditScore(user.address)).to.be.false;

      await creditScoreOracle.connect(operator).updateCreditScore(
        user.address,
        750,
        2,
        ethers.toUtf8Bytes("Good payment history")
      );

      expect(await creditScoreOracle.hasValidCreditScore(user.address)).to.be.true;
    });

    it("Should get comprehensive credit info", async function () {
      const creditScore = 750;
      const riskLevel = 2;
      const additionalData = ethers.toUtf8Bytes("Good payment history");

      await creditScoreOracle.connect(operator).updateCreditScore(
        user.address,
        creditScore,
        riskLevel,
        additionalData
      );

      const [retrievedScore, retrievedRisk, lastUpdated, isStale, dataHash] = 
        await creditScoreOracle.getCreditInfo(user.address);
      
      expect(retrievedScore).to.equal(creditScore);
      expect(retrievedRisk).to.equal(riskLevel);
      expect(lastUpdated).to.be.greaterThan(0);
      expect(isStale).to.be.false;
      expect(dataHash).to.not.equal(ethers.ZeroHash);
    });
  });

  describe("Validation", function () {
    it("Should reject invalid credit score range", async function () {
      await expect(creditScoreOracle.connect(operator).updateCreditScore(
        user.address,
        200, // Below minimum
        2,
        ethers.toUtf8Bytes("data")
      )).to.be.revertedWith("CreditScoreOracle: Invalid credit score range");

      await expect(creditScoreOracle.connect(operator).updateCreditScore(
        user.address,
        900, // Above maximum
        2,
        ethers.toUtf8Bytes("data")
      )).to.be.revertedWith("CreditScoreOracle: Invalid credit score range");
    });

    it("Should reject invalid risk level range", async function () {
      await expect(creditScoreOracle.connect(operator).updateCreditScore(
        user.address,
        750,
        0, // Below minimum
        ethers.toUtf8Bytes("data")
      )).to.be.revertedWith("CreditScoreOracle: Invalid risk level range");

      await expect(creditScoreOracle.connect(operator).updateCreditScore(
        user.address,
        750,
        6, // Above maximum
        ethers.toUtf8Bytes("data")
      )).to.be.revertedWith("CreditScoreOracle: Invalid risk level range");
    });

    it("Should reject zero address", async function () {
      await expect(creditScoreOracle.connect(operator).updateCreditScore(
        ethers.ZeroAddress,
        750,
        2,
        ethers.toUtf8Bytes("data")
      )).to.be.revertedWith("CreditScoreOracle: Invalid user address");
    });
  });

  describe("Access Control", function () {
    it("Should reject non-operator from updating credit score", async function () {
      await expect(creditScoreOracle.connect(user).updateCreditScore(
        user.address,
        750,
        2,
        ethers.toUtf8Bytes("data")
      )).to.be.revertedWith(/AccessControl: account .* is missing role .*/);
    });

    it("Should allow admin to grant operator role", async function () {
      await creditScoreOracle.connect(admin).grantOracleOperatorRole(user.address);
      expect(await creditScoreOracle.hasRole(await creditScoreOracle.ORACLE_OPERATOR_ROLE(), user.address)).to.be.true;
    });
  });

  describe("Data Staleness", function () {
    it("Should detect stale credit data", async function () {
      await creditScoreOracle.connect(operator).updateCreditScore(
        user.address,
        750,
        2,
        ethers.toUtf8Bytes("data")
      );
      
      // Fast forward time beyond maxDataAge
      await ethers.provider.send("evm_increaseTime", [maxDataAge + 1]);
      await ethers.provider.send("evm_mine", []);

      expect(await creditScoreOracle.isCreditDataStale(user.address)).to.be.true;
      expect(await creditScoreOracle.hasValidCreditScore(user.address)).to.be.false;
    });

    it("Should not detect fresh data as stale", async function () {
      await creditScoreOracle.connect(operator).updateCreditScore(
        user.address,
        750,
        2,
        ethers.toUtf8Bytes("data")
      );

      expect(await creditScoreOracle.isCreditDataStale(user.address)).to.be.false;
      expect(await creditScoreOracle.hasValidCreditScore(user.address)).to.be.true;
    });
  });

  describe("Risk Level", function () {
    it("Should get risk level correctly", async function () {
      const riskLevel = 3;
      
      await creditScoreOracle.connect(operator).updateCreditScore(
        user.address,
        650,
        riskLevel,
        ethers.toUtf8Bytes("data")
      );

      expect(await creditScoreOracle.getRiskLevel(user.address)).to.equal(riskLevel);
    });

    it("Should reject getting risk level for non-existent data", async function () {
      await expect(creditScoreOracle.getRiskLevel(user.address))
        .to.be.revertedWith("CreditScoreOracle: No valid credit score found");
    });
  });

  describe("Interface Compliance", function () {
    it("Should reject updateData calls", async function () {
      const dataId = ethers.keccak256(ethers.toUtf8Bytes("test"));
      const data = ethers.toUtf8Bytes("test");

      await expect(creditScoreOracle.connect(operator).updateData(dataId, data))
        .to.be.revertedWith("CreditScoreOracle: Use updateCreditScore instead of updateData");
    });

    it("Should reject getData calls", async function () {
      const dataId = ethers.keccak256(ethers.toUtf8Bytes("test"));

      await expect(creditScoreOracle.getData(dataId))
        .to.be.revertedWith("CreditScoreOracle: Use getCreditScore instead of getData");
    });
  });

  describe("Pausable", function () {
    it("Should prevent credit score updates when paused", async function () {
      await creditScoreOracle.connect(admin).pause();
      
      await expect(creditScoreOracle.connect(operator).updateCreditScore(
        user.address,
        750,
        2,
        ethers.toUtf8Bytes("data")
      )).to.be.revertedWith("Pausable: paused");
    });
  });
});
