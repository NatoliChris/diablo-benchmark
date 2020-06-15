package workloadgenerators

import "diablo-benchmark/core/configs"

type WorkloadGenerator interface {
	// Initialises the information to start the workload generator
	Init()

	// Creates a transaction to deploy the contract
	CreateContractDeployTransaction(contractPath string, key configs.ChainKey) ([]byte, error)

	// Creates an interaction with the contract
	CreateContractInteraction(contractAddress string, contractFunction string, params map[string]interface{}) ([]byte, error)

	// Create a signed transaction that returns the bytes
	CreateSignedTransaction(to string, value string, data []byte, key configs.ChainKey) ([]byte, error)

	// Generate the workload, returning the slice of transactions. [clientID = [ list of transactions ] ]
	GenerateWorkload(numClients int, numTransactions int, transactionInformation map[string]interface{}) ([][][]byte, error)
}
