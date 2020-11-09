package clientinterfaces

import (
	"diablo-benchmark/blockchains/workloadgenerators"
	"diablo-benchmark/core/results"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/hyperledger/fabric-sdk-go/pkg/core/config"
	"github.com/hyperledger/fabric-sdk-go/pkg/gateway"
)

//FabricInterface is the Hyperledger Fabric implementation of the clientinterface
// Provides functionality to communicate with the Fabric blockchain
type FabricInterface struct {
	Gateway   *gateway.Gateway
	Wallet    *gateway.Wallet
	Network   *gateway.Network
	Contracts map[string][]*gateway.Contract
	GenericInterface
}

// Init initializes the wallet, gateway, network and map of contracts available in the network
func (f *FabricInterface) Init(otherHosts []string) {
	f.Nodes = otherHosts
	f.NumTxDone = 0
	f.Contracts = map[string][]*gateway.Contract{}

	// create the gateaway, network and contract ?

	err := os.Setenv("DISCOVERY_AS_LOCALHOST", "true")
	if err != nil {
		log.Fatalf("Error setting DISCOVERY_AS_LOCALHOST environemnt variable: %v", err)
	}

	wallet, err := gateway.NewFileSystemWallet("wallet")
	fmt.Println("FOUND WALLET")
	if err != nil {
		log.Fatalf("Failed to create wallet: %v", err)
	}

	if !wallet.Exists("appUser") {
		err = populateWallet(wallet)
		if err != nil {
			log.Fatalf("Failed to populate wallet contents: %v", err)
		}
	}

	ccpPath := filepath.Join(
		"..",
		"..",
		"localImplementation",
		"artifacts",
		"channel",
		"crypto-config",
		"peerOrganizations",
		"org1.example.com",
		"connection-org1.yaml",
	)

	f.Gateway, err = gateway.Connect(
		gateway.WithConfig(config.FromFile(filepath.Clean(ccpPath))),
		gateway.WithIdentity(wallet, "appUser"))

	if err != nil {
		log.Fatalf("Failed to connect to gateway: %v", err)
	}

	f.Network, err = f.Gateway.GetNetwork("mychannel")

	if err != nil {
		log.Fatalf("Failed to get network: %v", err)
	}

	contract := f.Network.GetContract("basic")

	f.Contracts[contract.Name()] = []*gateway.Contract{contract}
}

// Called when the wallet hasn't been instantiated yet
// Creates the wallet/identity of the gateway peer we connect to
func populateWallet(wallet *gateway.Wallet) error {
	log.Println("============ Populating wallet ============")
	credPath := filepath.Join(
		"..",
		"..",
		"localImplementation",
		"artifacts",
		"channel",
		"crypto-config",
		"peerOrganizations",
		"org1.example.com",
		"users",
		"User1@org1.example.com",
		"msp",
	)

	certPath := filepath.Join(credPath, "signcerts", "User1@org1.example.com-cert.pem")
	// read the certificate pem
	cert, err := ioutil.ReadFile(filepath.Clean(certPath))
	if err != nil {
		return err
	}

	keyDir := filepath.Join(credPath, "keystore")
	// there's a single file in this dir containing the private key
	files, err := ioutil.ReadDir(keyDir)
	if err != nil {
		return err
	}
	if len(files) != 1 {
		return fmt.Errorf("keystore folder should have contain one file")
	}
	keyPath := filepath.Join(keyDir, files[0].Name())
	key, err := ioutil.ReadFile(filepath.Clean(keyPath))
	if err != nil {
		return err
	}

	identity := gateway.NewX509Identity("Org1MSP", string(cert), string(key))

	return wallet.Put("appUser", identity)
}

// Cleanup Finishes up and performs any post-benchmark operations.
// Can be used to format the results to parse back
func (f *FabricInterface) Cleanup() results.Results {
	return results.Results{
		TxLatencies:       nil,
		AverageLatency:    0,
		MedianLatency:     0,
		Throughput:        0,
		ThroughputSeconds: nil,
		Success:           0,
		Fail:              0,
	}
}

// Start handles the starting aspects of the benchmark
// Is primarily used for setting the start time and allocating resources for
// metrics
func (f *FabricInterface) Start() {}

//ParseWorkload Handles the workload, converts the bytes to usable transactions.
// This takes the worker's workload - and transitions to transactions
func (f *FabricInterface) ParseWorkload(workload workloadgenerators.WorkerThreadWorkload) ([][]interface{}, error) {
	return nil, nil
}

// ConnectOne will connect  to the blockchain node in the array slot of the
// given array
func (f *FabricInterface) ConnectOne(id int) error {
	return nil
}

// ConnectAll connects to all nodes given in the hosts
func (f *FabricInterface) ConnectAll(primaryID int) error {
	return nil
}

// DeploySmartContract deploys the smart contract, provided the path to the contract to deploy
// Returns the address of the contract
func (f *FabricInterface) DeploySmartContract(tx interface{}) (interface{}, error) {
	return nil, nil
}

// SendRawTransaction sends the raw transaction bytes to the blockchain
// It is safe to assume that these bytes will be formatted correctly according to the chosen blockchain.
// The transactions are generated through the workload to relieve the signing and encoding during timed
// benchmarks
func (f *FabricInterface) SendRawTransaction(tx interface{}) error {
	return nil
}

// SecureRead reads the value from the chain, this requires the client to connect to _multiple_ nodes and asks
// for the value. This ensures that the value read is "secure" - the same value must be returned
// from t+1 to be considered "correct".
func (f *FabricInterface) SecureRead(callFunc string, callParams []byte) (interface{}, error) {
	return nil, nil
}

// GetBlockByNumber retrieves the block information at the given index
// TODO: maybe implement getBlockByHash?
func (f *FabricInterface) GetBlockByNumber(index uint64) (GenericBlock, error) {
	return GenericBlock{
		Hash:              "",
		Index:             0,
		Timestamp:         0,
		TransactionNumber: 0,
		TransactionHashes: nil,
	}, nil
}

// GetBlockHeight returns the current height of the chain
func (f *FabricInterface) GetBlockHeight() (uint64, error) {
	return 0, nil
}

// GetTxDone returns the number of transactions completed
// This is already implemented with the GenericInterface
func (f *FabricInterface) GetTxDone() uint64 {
	return 0
}

// ParseBlocksForTransactions retrieves block information from start to end index and
// is used as a post-benchmark check to learn about the block and transactions.
func (f *FabricInterface) ParseBlocksForTransactions(startNumber uint64, endNumber uint64) error {
	return nil
}

// Close the connection to the blockchain node
func (f *FabricInterface) Close() {
	f.Gateway.Close()
}
