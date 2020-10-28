## Adding new blockchains

The diablo benchmark aims to have an open, easily extensible benchmark
framework that allows other blockchains to be integrated.  This can be in the
form of a new smart contract or transaction encoding, and/or a new interface.

### blockchains/clientinterfaces

The client interfaces package is the home of all the relevant blockchain
interaction. It defines a generic interface that should be implemented by each
new blockchain to interact throughout the benchmark.

The interface lives in `diablo-benchmark/blockchains/clientinterfaces/blockchain_interface.go`
and it defines all the functions that MUST be implemented to integrate a new blockchain.

```go
type BlockchainInterface interface {
	// Provides the client with the list of all hosts, this is the pair of (host, port) in an array.
	// This will be used for the secure reads.
	Init(otherHosts []string)

	// Finishes up and performs any post-benchmark operations.
	// Can be used to format the results to parse back
	Cleanup() results.Results

	// Handles the workload, converts the bytes to usable transactions.
	// This takes the worker's workload - and transitions to transactions
	ParseWorkload(workload workloadgenerators.WorkerThreadWorkload) ([][]interface{}, error)

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
```

**Init**

The `Init` function's core responsibility is to set up any requirements on the
client and store the "otherHosts". The `otherHosts` is a list of addresses of
all the blockchain nodes available to connect to.

**Cleanup**

The `Cleanup` function performs any cleanup and processing and returns the
results of the benchmark. This is called after the completion of the benchmark
and should be sued to perform any post-benchmark operations and formatting and
calculation of results.

**ParseWorkload**

Initially, the benchmark is defined in the WorkloadGenerator, however, upon
communicating between the Primary and the Secondary, it is converted into bytes.
The "ParseWorkload" function exists to convert the transactions from bytes into
the usable type that can be used later in the benchmark.


**ConnectOne**

This is used to connect the current secondary to **ONE** blockchain node as the
primary connection. This blockchain node will serve as the single communication
point between the secondary and the blockchain.

**ConnectAll**

ConnectAll connects the secondary to all the available blockchain nodes, using
one node as the primary connection. The primary node will be the single point
of contact for sending transactions, but the other connected nodes will be used
for read operations and to ensure that information has propagated and committed
correctly.

**DeploySmartContract**

This function is used to Deploy a smart contract and returns the address of the
created contract. This is primarily used in the setup of the benchmark, but can
also be used throughout the benchmark if deployments are required. This most
often will just be a wrapper around sending the transaction and waiting for the
address, but may also have different functionality.

**SendRawTransaction**

The SendRawTransaction is the primary function that is called throughout the
benchmark. This function accepts an already-signed transaction and is used to
forward on to the blockchain node using an RPC call. This send transaction
should have little to no processing, but sends the transaction recording the
metrics of sending time and any response.

**SecureRead**

A Secure Read is defined as a read of state from multiple machines to ensure
that the state returned is correct. The secure read sends a "call" to all
connected nodes, and compares the result.

**GetBlockByNumber**

This gets the entire block from the index/number. This may be useful for chains
where there is a sequential single-blockchain, but may require modifications for
DAGs or sharded blockchains.


**GetBlockHeight**

This function returns the height of the chain.


**ParseBlocksForTransactions**

This function is used as a post-benchmark analysis function, which iterates
through all the blocks from the start of the benchmark to gather statistics
about the blocks and the transactions included.

**Close**

This function closes the connection to the connected blockchain node(s).

### workloadgenerators/workloadgeneratorinterface.go

The workload generator interface provides the necessary functions to generate
and create a workload of transactions. As part of an integration of a new
blockchain, any changes in the virtual machine or transaction encoding should
be implemented in a new workload generator.

```go
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
	CreateInteractionTX(fromPrivKey []byte, contractAddress string, functionName string, contractParams []ContractParam) ([]byte, error)

	// Creates a transaction that is signed and ready to send from the given private key.
	CreateSignedTransaction(fromPrivKey []byte, toAddress string, value *big.Int, data []byte) ([]byte, error)

	// Generates the workload specified in the chain configurations.
	GenerateWorkload() (Workload, error)
}
```

**NewGenerator**

The new generator sets up all relevant fields in the generator and returns a
new instance of the implemented generator.

**BlockchainSetup**

This function is provided to perform any necessary setup for blockchain nodes
such as the creation of a genesis block, the creation of accounts, the
distribution and starting of the blockchain. This step exists as a preliminary
set up so that any blockchain-related starting functions can be called.

**InitParams**

The InitParams function is called to initialise any parameters that may be used
during the workload generation. This could be contacting the blockchain for the
account Nonce, the gas prices, etc.

**CreateAccount**

A generic account creation function that is used to generate a public:private
key pair that will be used for signing and sending, or receiving transactions.

**DeployContract**

Provided a smart contract, the "DeployContract" deploys the smart contract
and returns the address. This is an initial step for the contract workloads, as
they must send a transaction to a known address.

**CreateContractDeployTX**

Generates a transaction that is used to deploy a contract. This is most likely
used in the `DeployContract` function, but may be used in workloads where there
is a measurement of the performance of contract deployment.

**CreateInteractionTX**

Generates a transaction that interacts with a smart contract given a function,
arguments and the types. This is a low level function that converts high-level
properties to interact with the contract into a transaction and relevant bytes.
For example: `string: "hello"` will be converted into the bytecode for the
contract interaction.

**CreateSignedTransaction**

Creates a generic signed transaction with the value, data signed from a private
key and sent to a specified address. This is a basic form of transaction
generation and should return a transaction ready to send in raw format on the
secondary nodes with the `clientinterface`.

**GenerateWorkload**

Generates the workload specified in the benchmark configuration. This function
should perform the checks and calculations to generate the entire workload for
all secondaries in the benchmark.


### Other files

#### clientinterfaces/utils.go

This file provides the `GetBlockchainInterface` function which reads the string
of the blockchain name and maps it to an implemented interface. For example,
the configuration "ethereum" will result in returning the "EthereumInterface".
This mapping should be updated with any new blockchain implementations.

#### workloadgenerators/utils.go

This file provides the utility function `GetWorkloadGenerator` function, which
reads the name of the blockchain and returns the implemented workload generator.
The "ethereum" string in the chain configuration passed to this function should
return the "EthereumWorkloadGenerator", which will be used to generate the
workloads for any Ethereum benchmark.
