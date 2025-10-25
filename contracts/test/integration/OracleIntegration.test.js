const { expect } = require("chai");
const { ethers } = require("hardhat");

describe("Oracle Integration Tests", function () {
  let oracle;
  let creditScoreOracle;
  let admin;
  let operator;
  let user1;
  let user2;
  let maxDataAge;

  beforeEach(async function () {
    [admin, operator, user1, user2] = await ethers.getSigners();
    maxDataAge = 7 * 24 * 60 * 60; // 7 days

    // Deploy Oracle
    const Oracle = await ethers.getContractFactory("Oracle");
    oracle = await Oracle.deploy(admin.address, maxDataAge);
    await oracle.waitForDeployment();

    // Deploy CreditScoreOracle
    const CreditScoreOracle = await ethers.getContractFactory("CreditScoreOracle");
    creditScoreOracle = await CreditScoreOracle.deploy(admin.address, maxDataAge);
    await creditScoreOracle.waitForDeployment();

    // Grant operator roles
    await oracle.connect(admin).grantOracleOperatorRole(operator.address);
    await creditScoreOracle.connect(admin).grantOracleOperatorRole(operator.address);
  });

  describe("Multi-User Credit Scoring", function () {
    it("Should handle multiple users with different credit scores", async function () {
      // User 1: High credit score
      await creditScoreOracle.connect(operator).updateCreditScore(
        user1.address,
        800,
        1, // Low risk
        ethers.toUtf8Bytes("Excellent payment history, low debt")
      );

      // User 2: Medium credit score
      await creditScoreOracle.connect(operator).updateCreditScore(
        user2.address,
        650,
        3, // Medium risk
        ethers.toUtf8Bytes("Good payment history, moderate debt")
      );

      // Verify user1 data
      const [score1, risk1] = await creditScoreOracle.getCreditScore(user1.address);
      expect(score1).to.equal(800);
      expect(risk1).to.equal(1);

      // Verify user2 data
      const [score2, risk2] = await creditScoreOracle.getCreditScore(user2.address);
      expect(score2).to.equal(650);
      expect(risk2).to.equal(3);

      // Both should have valid scores
      expect(await creditScoreOracle.hasValidCreditScore(user1.address)).to.be.true;
      expect(await creditScoreOracle.hasValidCreditScore(user2.address)).to.be.true;
    });

    it("Should update credit scores over time", async function () {
      // Initial score
      await creditScoreOracle.connect(operator).updateCreditScore(
        user1.address,
        600,
        4,
        ethers.toUtf8Bytes("Initial assessment")
      );

      // Update score after improvement
      await creditScoreOracle.connect(operator).updateCreditScore(
        user1.address,
        750,
        2,
        ethers.toUtf8Bytes("Improved payment history")
      );

      const [finalScore, finalRisk] = await creditScoreOracle.getCreditScore(user1.address);
      expect(finalScore).to.equal(750);
      expect(finalRisk).to.equal(2);
    });
  });

  describe("Data Staleness Scenarios", function () {
    it("Should handle stale data correctly", async function () {
      // Set initial data
      await creditScoreOracle.connect(operator).updateCreditScore(
        user1.address,
        700,
        2,
        ethers.toUtf8Bytes("Fresh data")
      );

      expect(await creditScoreOracle.hasValidCreditScore(user1.address)).to.be.true;

      // Fast forward time beyond maxDataAge
      await ethers.provider.send("evm_increaseTime", [maxDataAge + 1]);
      await ethers.provider.send("evm_mine", []);

      // Data should now be stale
      expect(await creditScoreOracle.isCreditDataStale(user1.address)).to.be.true;
      expect(await creditScoreOracle.hasValidCreditScore(user1.address)).to.be.false;
    });

    it("Should allow fresh data to override stale data", async function () {
      // Set initial data
      await creditScoreOracle.connect(operator).updateCreditScore(
        user1.address,
        700,
        2,
        ethers.toUtf8Bytes("Initial data")
      );

      // Make data stale
      await ethers.provider.send("evm_increaseTime", [maxDataAge + 1]);
      await ethers.provider.send("evm_mine", []);

      expect(await creditScoreOracle.hasValidCreditScore(user1.address)).to.be.false;

      // Update with fresh data
      await creditScoreOracle.connect(operator).updateCreditScore(
        user1.address,
        750,
        1,
        ethers.toUtf8Bytes("Updated fresh data")
      );

      expect(await creditScoreOracle.hasValidCreditScore(user1.address)).to.be.true;
    });
  });

  describe("Oracle Data Types", function () {
    it("Should register and use different data types", async function () {
      // Register data types
      await oracle.connect(admin).registerDataType("CREDIT_SCORE", "Credit score data");
      await oracle.connect(admin).registerDataType("MARKET_RATES", "Market interest rates");
      await oracle.connect(admin).registerDataType("RISK_FACTORS", "Risk assessment factors");

      // Verify registration
      expect(await oracle.isDataTypeRegistered("CREDIT_SCORE")).to.be.true;
      expect(await oracle.isDataTypeRegistered("MARKET_RATES")).to.be.true;
      expect(await oracle.isDataTypeRegistered("RISK_FACTORS")).to.be.true;

      // Store different types of data
      const creditDataId = ethers.keccak256(ethers.toUtf8Bytes("user1-credit"));
      const marketDataId = ethers.keccak256(ethers.toUtf8Bytes("current-rates"));
      const riskDataId = ethers.keccak256(ethers.toUtf8Bytes("market-risk"));

      await oracle.connect(operator).updateData(
        creditDataId,
        ethers.toUtf8Bytes(JSON.stringify({ score: 750, risk: 2 }))
      );

      await oracle.connect(operator).updateData(
        marketDataId,
        ethers.toUtf8Bytes(JSON.stringify({ lendingRate: 5.5, borrowingRate: 7.2 }))
      );

      await oracle.connect(operator).updateData(
        riskDataId,
        ethers.toUtf8Bytes(JSON.stringify({ volatility: 0.15, correlation: 0.3 }))
      );

      // Verify data retrieval
      const [creditData] = await oracle.getData(creditDataId);
      const [marketData] = await oracle.getData(marketDataId);
      const [riskData] = await oracle.getData(riskDataId);

      expect(JSON.parse(ethers.toUtf8String(creditData))).to.deep.include({ score: 750, risk: 2 });
      expect(JSON.parse(ethers.toUtf8String(marketData))).to.deep.include({ lendingRate: 5.5, borrowingRate: 7.2 });
      expect(JSON.parse(ethers.toUtf8String(riskData))).to.deep.include({ volatility: 0.15, correlation: 0.3 });
    });
  });

  describe("Access Control Integration", function () {
    it("Should properly manage roles across contracts", async function () {
      // Admin should have all roles
      expect(await oracle.hasRole(await oracle.ADMIN_ROLE(), admin.address)).to.be.true;
      expect(await creditScoreOracle.hasRole(await creditScoreOracle.ADMIN_ROLE(), admin.address)).to.be.true;

      // Operator should have operator roles
      expect(await oracle.hasRole(await oracle.ORACLE_OPERATOR_ROLE(), operator.address)).to.be.true;
      expect(await creditScoreOracle.hasRole(await creditScoreOracle.ORACLE_OPERATOR_ROLE(), operator.address)).to.be.true;

      // User should not have any roles
      expect(await oracle.hasRole(await oracle.ORACLE_OPERATOR_ROLE(), user1.address)).to.be.false;
      expect(await creditScoreOracle.hasRole(await creditScoreOracle.ORACLE_OPERATOR_ROLE(), user1.address)).to.be.false;
    });

    it("Should allow role management", async function () {
      // Grant user1 operator role
      await oracle.connect(admin).grantOracleOperatorRole(user1.address);
      await creditScoreOracle.connect(admin).grantOracleOperatorRole(user1.address);

      // User1 should now be able to update data
      await expect(creditScoreOracle.connect(user1).updateCreditScore(
        user2.address,
        600,
        3,
        ethers.toUtf8Bytes("Updated by user1")
      )).to.not.be.reverted;

      // Revoke role
      await oracle.connect(admin).revokeOracleOperatorRole(user1.address);
      await creditScoreOracle.connect(admin).revokeOracleOperatorRole(user1.address);

      // User1 should no longer be able to update data
      await expect(creditScoreOracle.connect(user1).updateCreditScore(
        user2.address,
        700,
        2,
        ethers.toUtf8Bytes("Should fail")
      )).to.be.revertedWith(/AccessControl: account .* is missing role .*/);
    });
  });

  describe("Pause/Unpause Integration", function () {
    it("Should pause both contracts and prevent operations", async function () {
      // Pause both contracts
      await oracle.connect(admin).pause();
      await creditScoreOracle.connect(admin).pause();

      // Operations should fail
      const dataId = ethers.keccak256(ethers.toUtf8Bytes("test"));
      const data = ethers.toUtf8Bytes("test data");

      await expect(oracle.connect(operator).updateData(dataId, data))
        .to.be.revertedWith("Pausable: paused");

      await expect(creditScoreOracle.connect(operator).updateCreditScore(
        user1.address,
        750,
        2,
        ethers.toUtf8Bytes("test")
      )).to.be.revertedWith("Pausable: paused");

      // Unpause and operations should work
      await oracle.connect(admin).unpause();
      await creditScoreOracle.connect(admin).unpause();

      await expect(oracle.connect(operator).updateData(dataId, data))
        .to.not.be.reverted;

      await expect(creditScoreOracle.connect(operator).updateCreditScore(
        user1.address,
        750,
        2,
        ethers.toUtf8Bytes("test")
      )).to.not.be.reverted;
    });
  });

  describe("Gas Optimization", function () {
    it("Should handle batch operations efficiently", async function () {
      const users = [user1, user2];
      const creditScores = [750, 650];
      const riskLevels = [2, 3];

      // Batch update credit scores
      for (let i = 0; i < users.length; i++) {
        await creditScoreOracle.connect(operator).updateCreditScore(
          users[i].address,
          creditScores[i],
          riskLevels[i],
          ethers.toUtf8Bytes(`Batch data for user ${i + 1}`)
        );
      }

      // Verify all updates
      for (let i = 0; i < users.length; i++) {
        const [score, risk] = await creditScoreOracle.getCreditScore(users[i].address);
        expect(score).to.equal(creditScores[i]);
        expect(risk).to.equal(riskLevels[i]);
      }
    });
  });
});
