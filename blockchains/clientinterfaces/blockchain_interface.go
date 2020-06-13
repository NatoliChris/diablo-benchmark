package clientinterfaces

// This struct provides the basic funcitonality that will be tested with the blockchains.
// It _should_ cover most interaction, but will be extendible in the event that more
// complex functionality is created with blockchains.
type BlockchainInterface interface {
	// Provides the client with the list of all hosts, this is the pair of (host, port) in an array.
	// This will be used for the secure reads.
	Init(otherHosts [][]string)

	// Connect to the blockchain node in the array slot of the given array
	ConnectOne(id int) (bool, error)

	// Connect to all nodes
	ConnectAll(primaryId int) (bool, error)

	// Deploy the smart contract, we will provide the path to the contract to deploy
	// Returns the address of the contract deploy
	DeploySmartContract(contractPath string) (interface{}, error)

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
	GetBlockByNumber(index uint64) (map[string]interface{}, error)

	// Asks for the height of the current block
	GetBlockHeight() (uint64, error)

	// Close the connection to the blockchain node
	Close()
}
