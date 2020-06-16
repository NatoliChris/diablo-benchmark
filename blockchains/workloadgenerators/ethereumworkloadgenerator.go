package workloadgenerators

import (
	"context"
	"diablo-benchmark/core/configs"
	"encoding/hex"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
	"math/big"
)

// Generates the workload for the Ethereum blockchain
type EthereumWorkloadGenerator struct {
	ActiveConn        *ethclient.Client
	SuggestedGasPrice *big.Int
	BenchConfig       configs.BenchConfig
	Nonces            map[string]uint64
}

// Sets the suggested gas price and sets up a small connection to get information from the blockchain.
func (e *EthereumWorkloadGenerator) Init(chainConfig configs.ChainConfig, benchConfig configs.BenchConfig) error {

	// Connect to the blockchain
	c, err := ethclient.Dial(fmt.Sprintf("ws://%s:%s", chainConfig.Nodes[0][0], chainConfig.Nodes[0][1]))

	if err != nil {
		return err
	}

	e.ActiveConn = c

	e.SuggestedGasPrice, err = e.ActiveConn.SuggestGasPrice(context.Background())

	if err != nil {
		return err
	}

	// nonces
	e.Nonces = make(map[string]uint64, 0)

	for _, key := range chainConfig.Keys {
		v, err := e.ActiveConn.NonceAt(context.Background(), common.HexToAddress(key.Address), -1)
		if err != nil {
			return err
		}

		e.Nonces[key.Address] = v
	}

	return nil
}

// Creates a transaction to deploy the contract
func (e *EthereumWorkloadGenerator) CreateContractDeployTransaction(contractPath string, key configs.ChainKey) ([]byte, error) {
	return []byte{}, nil
}

// Creates an interaction with the contract
func (e *EthereumWorkloadGenerator) CreateContractInteraction(contractAddress string, contractFunction string, params map[string]interface{}) ([]byte, error) {
	return []byte{}, nil
}

// Create a signed transaction that returns the bytes
func (e *EthereumWorkloadGenerator) CreateSignedTransaction(to string, value string, data []byte, key configs.ChainKey) ([]byte, error) {
	priv, err := crypto.HexToECDSA(hex.EncodeToString(key.PrivateKey))

	if err != nil {
		return []byte{}, err
	}

	fromAddress := common.HexToAddress(key.Address)
	toAddress := common.HexToAddress(to)

}

// Generate the workload, returning the slice of transactions. [clientID = [ list of transactions ] ]
func (e *EthereumWorkloadGenerator) GenerateWorkload(numClients int, numTransactions int, transactionInformation map[string]interface{}) ([][][]byte, error) {
	clientWorkloads := make([][][]byte, 0)

	return clientWorkloads, nil
}
