package clientinterfaces

import (
	"diablo-benchmark/blockchains/workloadgenerators"
	"diablo-benchmark/core/results"
)

// This struct provides the basic funcitonality that will be tested with the blockchains.
// It _should_ cover most interaction, but will be extendible in the event that more
// complex functionality is created with blockchains.
type BlockchainInterface interface {
	// Provides the client with the list of all hosts, this is the pair of (host, port) in an array.
	// This will be used for the secure reads.
	Init(otherHosts []string)

	// Finishes up and performs any post-benchmark operations.
	// Can be used to format the results to parse back
	Cleanup() results.Results

	// Handles the workload, converts the bytes to usable transactions.
	// This takes the worker's workload - and transitions to transactions
	ParseWorkload(workload workloadgenerators.WorkerWorkload) ([][]interface{}, error)

	// Connect to the blockchain node in the array slot of the given array
	ConnectOne(id int) error

	// Connect to all nodes
	ConnectAll(primaryId int) error

	// Deploy the smart contract, we will provide the path to the contract to deploy
	// Returns the address of the contract deploy
	DeploySmartContract(tx interface{}) (interface{}, error)

	// Send the raw transaction bytes to the blockchain
	// It is safe to assume that these bytes will be formatted correctly according to the chosen blockchain.
	// The transactions are generated through the workload to relieve the signing and encoding during timed
	// benchmarks
	SendRawTransaction(tx interface{}) error

	// Securely read the value from the chain, this requires the client to connect to _multiple_ nodes and asks
	// for the value.
	SecureRead(call_func string, call_params []byte) (interface{}, error)

	// Asks for the block information
	// TODO: maybe implement getBlockByHash?
	GetBlockByNumber(index uint64) (GenericBlock, error)

	// Asks for the height of the current block
	GetBlockHeight() (uint64, error)

	// Parse blocks for transactions
	ParseBlocksForTransactions(startNumber uint64, endNumber uint64) error

	// Close the connection to the blockchain node
	Close()
}
