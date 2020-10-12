// Package clientinterfaces provides the implementations and interactions of the
// specific blockchain interface. The "blockchain_interface" is the main interface
// defining the required functionality of all implementations for the blockchain.
// This package is defined to integrate the interactions of the different
// blockchain RPC interactions.
package clientinterfaces

import (
	"diablo-benchmark/blockchains/workloadgenerators"
	"diablo-benchmark/core/results"
)


// GenericInterface provides the required fields of the blockchain interface so that
// All information can be accessed
type GenericInterface struct {
	NumTxDone uint64 // The number of completed transactions
	NumTxSent uint64 // Number of transactions sent
}

// GetTxDone returns the number of transactions completed
func (gi *GenericInterface) GetTxDone() uint64 {
	return gi.NumTxDone
}

// BlockchainInterface provides the basic funcitonality that will be tested
// with the blockchains.
// It _should_ cover most interaction, but will be extendible in the event that
// more complex functionality is created with blockchains.
type BlockchainInterface interface {
	// Provides the client with the list of all hosts, this is the pair of (host, port) in an array.
	// This will be used for the secure reads.
	Init(otherHosts []string)

	// Finishes up and performs any post-benchmark operations.
	// Can be used to format the results to parse back
	Cleanup() results.Results

	// Start handles the starting aspects of the benchmark
	// Is primarily used for setting the start time and allocating resources for
	// metrics
	Start()

	// Handles the workload, converts the bytes to usable transactions.
	// This takes the worker's workload - and transitions to transactions
	ParseWorkload(workload workloadgenerators.WorkerThreadWorkload) ([][]interface{}, error)

	// ConnectOne will connect  to the blockchain node in the array slot of the
	// given array
	ConnectOne(id int) error

	// ConnectAll connects to all nodes given in the hosts
	ConnectAll(primaryID int) error

	// DeploySmartContract deploys the smart contract, provided the path to the contract to deploy
	// Returns the address of the contract
	DeploySmartContract(tx interface{}) (interface{}, error)

	// SendRawTransactions sends the raw transaction bytes to the blockchain
	// It is safe to assume that these bytes will be formatted correctly according to the chosen blockchain.
	// The transactions are generated through the workload to relieve the signing and encoding during timed
	// benchmarks
	SendRawTransaction(tx interface{}) error

	// SecureRead reads the value from the chain, this requires the client to connect to _multiple_ nodes and asks
	// for the value. This ensures that the value read is "secure" - the same value must be returned
	// from t+1 to be considered "correct".
	SecureRead(callFunc string, callParams []byte) (interface{}, error)

	// GetBlockByNumber retrieves the block information at the given index
	// TODO: maybe implement getBlockByHash?
	GetBlockByNumber(index uint64) (GenericBlock, error)

	// GetBlockHeight returns the current height of the chain
	GetBlockHeight() (uint64, error)

	// Get Tx Done returns the number of transactions completed
	// This is already implemented with the GenericInterface
	GetTxDone() uint64

	// ParseBlocksForTransactions retrieves block information from start to end index and
	// is used as a post-benchmark check to learn about the block and transactions.
	ParseBlocksForTransactions(startNumber uint64, endNumber uint64) error

	// Close the connection to the blockchain node
	Close()
}
