package blockchain

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/yourusername/p2p-lend/oracle-service/pkg/logger"
	"go.uber.org/zap"
)

// OracleClient handles blockchain interactions
type OracleClient struct {
	client          *ethclient.Client
	contractAddress common.Address
	privateKey      *ecdsa.PrivateKey
	chainID         *big.Int
}

// NewOracleClient creates a new blockchain oracle client
func NewOracleClient(rpcURL, contractAddr, privateKeyHex string) (*OracleClient, error) {
	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to ethereum node: %w", err)
	}

	privateKey, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		return nil, fmt.Errorf("invalid private key: %w", err)
	}

	chainID, err := client.ChainID(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to get chain ID: %w", err)
	}

	return &OracleClient{
		client:          client,
		contractAddress: common.HexToAddress(contractAddr),
		privateKey:      privateKey,
		chainID:         chainID,
	}, nil
}

// UpdateCreditScore submits a credit score update to the blockchain
func (oc *OracleClient) UpdateCreditScore(
	ctx context.Context,
	userAddress string,
	score uint16,
	confidence uint8,
	dataHash string,
) (*types.Transaction, error) {

	// Get the public address from private key
	publicKey := oc.privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("error casting public key to ECDSA")
	}

	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)

	// Get nonce
	nonce, err := oc.client.PendingNonceAt(ctx, fromAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to get nonce: %w", err)
	}

	// Get gas price
	gasPrice, err := oc.client.SuggestGasPrice(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get gas price: %w", err)
	}

	// Create auth transactor
	auth, err := bind.NewKeyedTransactorWithChainID(oc.privateKey, oc.chainID)
	if err != nil {
		return nil, fmt.Errorf("failed to create transactor: %w", err)
	}

	auth.Nonce = big.NewInt(int64(nonce))
	auth.Value = big.NewInt(0)
	auth.GasLimit = uint64(300000)
	auth.GasPrice = gasPrice

	// In production, you would use the generated contract binding
	// For now, we'll create a raw transaction
	// This is a placeholder - actual implementation would use contract ABI

	logger.Info("Submitting credit score update",
		zap.String("user", userAddress),
		zap.Uint16("score", score),
		zap.Uint8("confidence", confidence),
		zap.String("dataHash", dataHash),
	)

	// TODO: Replace with actual contract call using generated bindings
	// Example:
	// contract, err := NewCreditScoreOracle(oc.contractAddress, oc.client)
	// tx, err := contract.UpdateScore(auth, common.HexToAddress(userAddress), score, dataHash)

	// For now, return a mock transaction hash
	logger.Info("Credit score update submitted (mock)")

	return nil, nil // Placeholder
}

// GetCreditScore retrieves a credit score from the blockchain
func (oc *OracleClient) GetCreditScore(ctx context.Context, userAddress string) (uint16, uint8, string, error) {
	// In production, this would call the contract's view function
	// Using the generated contract binding

	logger.Info("Fetching credit score from blockchain",
		zap.String("user", userAddress),
	)

	// TODO: Replace with actual contract call
	// Example:
	// contract, err := NewCreditScoreOracle(oc.contractAddress, oc.client)
	// scoreData, err := contract.GetScore(&bind.CallOpts{Context: ctx}, common.HexToAddress(userAddress))

	// Placeholder return
	return 0, 0, "", fmt.Errorf("not implemented - requires contract binding")
}

// SignData creates a cryptographic signature of the score data
func (oc *OracleClient) SignData(userAddress string, score uint16, confidence uint8, dataHash string) ([]byte, error) {
	// Create message to sign
	message := fmt.Sprintf("%s:%d:%d:%s", userAddress, score, confidence, dataHash)
	messageHash := crypto.Keccak256Hash([]byte(message))

	// Sign the message
	signature, err := crypto.Sign(messageHash.Bytes(), oc.privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to sign data: %w", err)
	}

	return signature, nil
}

// VerifySignature verifies a signature against oracle data
func (oc *OracleClient) VerifySignature(
	userAddress string,
	score uint16,
	confidence uint8,
	dataHash string,
	signature []byte,
) (bool, error) {

	// Recreate message
	message := fmt.Sprintf("%s:%d:%d:%s", userAddress, score, confidence, dataHash)
	messageHash := crypto.Keccak256Hash([]byte(message))

	// Recover public key from signature
	pubKey, err := crypto.SigToPub(messageHash.Bytes(), signature)
	if err != nil {
		return false, fmt.Errorf("failed to recover public key: %w", err)
	}

	// Get expected address
	recoveredAddr := crypto.PubkeyToAddress(*pubKey)
	expectedAddr := crypto.PubkeyToAddress(*oc.privateKey.Public().(*ecdsa.PublicKey))

	return recoveredAddr == expectedAddr, nil
}

// GetTransactionReceipt gets the receipt for a transaction
func (oc *OracleClient) GetTransactionReceipt(ctx context.Context, txHash common.Hash) (*types.Receipt, error) {
	receipt, err := oc.client.TransactionReceipt(ctx, txHash)
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction receipt: %w", err)
	}
	return receipt, nil
}

// WaitForConfirmation waits for a transaction to be confirmed
func (oc *OracleClient) WaitForConfirmation(ctx context.Context, txHash common.Hash, confirmations uint64) error {
	receipt, err := bind.WaitMined(ctx, oc.client, &types.Transaction{})
	if err != nil {
		return fmt.Errorf("transaction failed: %w", err)
	}

	if receipt.Status != types.ReceiptStatusSuccessful {
		return fmt.Errorf("transaction failed with status: %d", receipt.Status)
	}

	return nil
}

// EstimateGas estimates gas for a score update transaction
func (oc *OracleClient) EstimateGas(ctx context.Context) (uint64, error) {
	// In production, this would call estimateGas on the contract
	// For now, return a reasonable estimate
	return 200000, nil
}

// GetBlockNumber gets the current block number
func (oc *OracleClient) GetBlockNumber(ctx context.Context) (uint64, error) {
	return oc.client.BlockNumber(ctx)
}

// Close closes the client connection
func (oc *OracleClient) Close() {
	if oc.client != nil {
		oc.client.Close()
	}
}

// HealthCheck verifies blockchain connection
func (oc *OracleClient) HealthCheck(ctx context.Context) error {
	_, err := oc.client.BlockNumber(ctx)
	if err != nil {
		return fmt.Errorf("blockchain health check failed: %w", err)
	}
	return nil
}
