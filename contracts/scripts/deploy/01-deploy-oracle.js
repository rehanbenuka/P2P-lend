const { ethers } = require("hardhat");

async function main() {
  console.log("🚀 Starting Oracle deployment...");

  // Get the deployer account
  const [deployer] = await ethers.getSigners();
  console.log("Deploying contracts with the account:", deployer.address);
  console.log("Account balance:", (await deployer.provider.getBalance(deployer.address)).toString());

  // Deployment parameters
  const maxDataAge = 7 * 24 * 60 * 60; // 7 days in seconds

  // Deploy Oracle contract
  console.log("\n📦 Deploying Oracle contract...");
  const Oracle = await ethers.getContractFactory("Oracle");
  const oracle = await Oracle.deploy(deployer.address, maxDataAge);
  await oracle.waitForDeployment();
  const oracleAddress = await oracle.getAddress();
  console.log("✅ Oracle deployed to:", oracleAddress);

  // Deploy CreditScoreOracle contract
  console.log("\n📦 Deploying CreditScoreOracle contract...");
  const CreditScoreOracle = await ethers.getContractFactory("CreditScoreOracle");
  const creditScoreOracle = await CreditScoreOracle.deploy(deployer.address, maxDataAge);
  await creditScoreOracle.waitForDeployment();
  const creditScoreOracleAddress = await creditScoreOracle.getAddress();
  console.log("✅ CreditScoreOracle deployed to:", creditScoreOracleAddress);

  // Register data types
  console.log("\n📝 Registering data types...");
  await oracle.registerDataType("CREDIT_SCORE", "Credit score data for P2P lending");
  await oracle.registerDataType("RISK_ASSESSMENT", "Risk assessment data");
  await oracle.registerDataType("MARKET_DATA", "Market data for lending rates");
  console.log("✅ Data types registered");

  // Verify contracts (if on testnet)
  if (network.name !== "hardhat" && network.name !== "localhost") {
    console.log("\n🔍 Verifying contracts...");
    try {
      await hre.run("verify:verify", {
        address: oracleAddress,
        constructorArguments: [deployer.address, maxDataAge],
      });
      console.log("✅ Oracle verified");

      await hre.run("verify:verify", {
        address: creditScoreOracleAddress,
        constructorArguments: [deployer.address, maxDataAge],
      });
      console.log("✅ CreditScoreOracle verified");
    } catch (error) {
      console.log("❌ Verification failed:", error.message);
    }
  }

  // Save deployment info
  const deploymentInfo = {
    network: network.name,
    chainId: network.config.chainId,
    deployer: deployer.address,
    timestamp: new Date().toISOString(),
    contracts: {
      Oracle: {
        address: oracleAddress,
        maxDataAge: maxDataAge,
      },
      CreditScoreOracle: {
        address: creditScoreOracleAddress,
        maxDataAge: maxDataAge,
      },
    },
  };

  console.log("\n📋 Deployment Summary:");
  console.log("Network:", deploymentInfo.network);
  console.log("Chain ID:", deploymentInfo.chainId);
  console.log("Deployer:", deploymentInfo.deployer);
  console.log("Oracle Address:", oracleAddress);
  console.log("CreditScoreOracle Address:", creditScoreOracleAddress);

  // Save to file
  const fs = require("fs");
  const path = require("path");
  const deploymentsDir = path.join(__dirname, "../../deployments");
  if (!fs.existsSync(deploymentsDir)) {
    fs.mkdirSync(deploymentsDir, { recursive: true });
  }
  
  const filename = `deployment-${network.name}-${Date.now()}.json`;
  const filepath = path.join(deploymentsDir, filename);
  fs.writeFileSync(filepath, JSON.stringify(deploymentInfo, null, 2));
  console.log(`\n💾 Deployment info saved to: ${filepath}`);

  console.log("\n🎉 Deployment completed successfully!");
}

main()
  .then(() => process.exit(0))
  .catch((error) => {
    console.error("❌ Deployment failed:", error);
    process.exit(1);
  });
