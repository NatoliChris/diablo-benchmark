// Package workloadgenerators provides the workload generators for the
// DIABLO benchmark.
// This handles the creation of the entire workload for each given secondary
// thread, and handles the account information relevant to the workload.
// It handles the deployment and compilation of smart
// contracts and the generation/creation of transactions that will be used
// to send the information through to the blockchain network.
package workloadgenerators

import (
	"diablo-benchmark/core/configs"
	"math/big"
)

// Workload definitions for ease of use: [secondary][worker][time][txlist][txbytes]
type Workload [][][][][]byte

// SecondaryWorkload is the workload per secondary: [worker][time][txlist][txbytes]
type SecondaryWorkload [][][][]byte

// WorkerThreadWorkload is the workload executed per thread on the secondary: [time][txlist][txbytes]
type WorkerThreadWorkload [][][]byte

// GenericWorkloadGenerator provides generic aspects for the workload generator that will be
// standard across all generators
type GenericWorkloadGenerator struct {
	TPSIntervals []int
}

// SetThreadIntervals sets the TPS intervals to be made for each thread
// This is the number of transactions per second for each thread to be made
func (g *GenericWorkloadGenerator) SetThreadIntervals(intervals []int) {
	g.TPSIntervals = intervals
}

// WorkloadGenerator provides the interface and basic functionality to generate a workload given the configurations.
// The workload generator handles the creation of the transactions and additionally sets
// up the blockchain and starts the blockchain nodes.
type WorkloadGenerator interface {

	// NewGenerator returns a new instance of the workload generator for the specific type of blockchain.
	// This instance should initialise all required variables
	NewGenerator(chainConfig *configs.ChainConfig, benchConfig *configs.BenchConfig) WorkloadGenerator

	// BlockchainSetup sets up the blockchain, creates necessary genesis, starts the blockchain through SSH commands, etc.
	BlockchainSetup() error

	// InitParms initialises useful params for generation of the workloads
	// For example, set up a connection to a node to get gas price / chainID, ... etc.
	InitParams() error

	// CreateAccount Creates an account and returns the <bytes(privateKey), address>
	// TODO: should this be chainKey, or interface{} for a blockchain account of their own?
	// CreateAccount() (configs.ChainKey, error)
	CreateAccount() (interface{}, error)

	// DeployContract deploys the contract and returns the contract address used in the chain.
	DeployContract(fromPrivKey []byte, contractPath string) (string, error)

	// CreateContractDeployTX creates the raw signed transaction that will deploy a contract
	CreateContractDeployTX(fromPrivKey []byte, contractPath string) ([]byte, error)

	// CreateInteractionTX create a signed transaction that performs actions on a smart contract at the given address
	CreateInteractionTX(fromPrivKey []byte, contractAddress string, functionName string, contractParams []configs.ContractParam, value string) ([]byte, error)

	// CreateSignedTransaction creates a transaction that is signed and ready to send from the given private key.
	CreateSignedTransaction(fromPrivKey []byte, toAddress string, value *big.Int, data []byte) ([]byte, error)

	// GenerateWorkload generates the workload specified in the chain configurations.
	GenerateWorkload() (Workload, error)

	// SetThreadIntervals sets the number of transactions per thread to create for each interval
	SetThreadIntervals(interval []int)
}
