package workloadgenerators

import (
	"diablo-benchmark/core/configs"
	"math/big"
)

// Workload definitions for ease of use
type Workload [][][][][]byte     // Workload: [client][worker][time][txlist][txbytes]
type ClientWorkload [][][][]byte // Client workload: [worker][time][txlist][txbytes]
type WorkerWorkload [][][]byte   // Worker workload: [time][txlist][txbytes]

type WorkloadGenerator interface {
	// Creates a new instance of the workload generator for the specific type of blockchain
	NewGenerator(chainConfig *configs.ChainConfig, benchConfig *configs.BenchConfig) WorkloadGenerator

	// Sets up the blockchain, creates necessary genesis, starts the blockchain through SSH commands, etc.
	BlockchainSetup() error

	// Initialises useful params for generation of the workloads
	// For example, set up a connection to a node to get gas price / chainID, ... etc.
	InitParams() error

	// Creates an account and returns the <bytes(privateKey), address>
	// TODO: should this be chainKey, or interface{} for a blockchain account of their own?
	// CreateAccount() (configs.ChainKey, error)
	CreateAccount() (interface{}, error)

	// Deploys the contract and returns the contract address used in the chain.
	DeployContract(fromPrivKey []byte, contractPath string) (string, error)

	// Creates the raw signed transaction that will deploy a contract
	CreateContractDeployTX(fromPrivKey []byte, contractPath string) ([]byte, error)

	// Create a signed transaction that performs actions on a smart contract at the given address
	CreateInteractionTX(fromPrivKey []byte, contractAddress string, functionName string, contractParams map[string]interface{}) ([]byte, error)

	// Creates a transaction that is signed and ready to send from the given private key.
	CreateSignedTransaction(fromPrivKey []byte, toAddress string, value *big.Int) ([]byte, error)

	// Generates the workload specified in the chain configurations.
	GenerateWorkload() (Workload, error)
}
