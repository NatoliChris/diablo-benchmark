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
type BlockchainInterface interface 
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
