package workloadgenerators

import (
	"diablo-benchmark/core/configs"
	"math/big"
)

type WorkloadGenerator interface {
	// Initialises the information to start the workload generator
	Init(chainConfig *configs.ChainConfig, benchConfig *configs.BenchConfig) error

	// Creates a transaction to deploy the contract
	CreateContractDeployTransaction(contractPath string, key configs.ChainKey) ([]byte, error)

	// Creates a genesis block with the specified params
	CreateGenesisBlock(addresses []string, balance *big.Int, otherParams map[string]interface{})

	// Create the accounts
	// TODO: should this be configs.types.chainKey?
	CreateNewAccount() (interface{}, error)

	// Creates an interaction with the contract
	CreateContractInteraction(contractAddress string, contractFunction string, params map[string]interface{}) ([]byte, error)

	// Create a signed transaction that returns the bytes
	CreateSignedTransaction(to string, value string, data []byte, key configs.ChainKey) ([]byte, error)

	// TODO add contaracts
	GenerateWorkloadAndAccounts(numClients int, numTransactionsPerClient int, transactionInformation map[string]interface{}) ([][][]byte, error)

	// Generate the workload, returning the slice of transactions. [clientID = [ list of transactions ] ]
	// TODO: add contracts
	GenerateWorkload(numClients int, numTransactionsPerClient int, transactionInformation map[string]interface{}, accounts []configs.ChainKey) ([][][]byte, error)
}
