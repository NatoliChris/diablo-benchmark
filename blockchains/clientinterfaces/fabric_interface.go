package clientinterfaces

import (
	"diablo-benchmark/blockchains/types"
	"diablo-benchmark/blockchains/workloadgenerators"
	"diablo-benchmark/core/configs"
	"diablo-benchmark/core/results"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	"github.com/hyperledger/fabric-sdk-go/pkg/core/config"
	"github.com/hyperledger/fabric-sdk-go/pkg/gateway"
	"go.uber.org/zap"
)

//FabricInterface is the Hyperledger Fabric implementation of the clientinterface
// Provides functionality to communicate with the Fabric blockchain
type FabricInterface struct {
	Gateway  *gateway.Gateway  // Gateway manages the network interaction on behalf of the application
	Wallet   *gateway.Wallet   // Wallet containing user identity configured for the gateway
	Network  *gateway.Network  // Network object originating from gateway
	Contract *gateway.Contract // The smart contract we will be interacting with (only supporting one contract workload for now)
	ccpPath  string            // connection-profile path to configure the gateway

	TransactionInfo  map[uint64][]time.Time // Transaction information (used for throughput calculation)
	StartTime        time.Time              // Start time of the benchmark
	ThroughputTicker *time.Ticker           // Ticker for throughput (1s)
	Throughputs      []float64              // Throughput over time with 1 second intervals
	GenericInterface
}

// Init initializes the wallet, gateway, network and map of contracts available in the network
func (f *FabricInterface) Init(chainConfig *configs.ChainConfig) {
	f.Nodes = chainConfig.Nodes
	mapConfig := chainConfig.Extra[0].(map[string]interface{})
	user := types.FabricUser{
		Label: mapConfig["label"].(string),
		MspID: mapConfig["mspID"].(string),
		Cert:  mapConfig["cert"].(string),
		Key:   mapConfig["key"].(string),
	}
	f.NumTxDone = 0
	f.TransactionInfo = make(map[uint64][]time.Time, 0)

	err := os.Setenv("DISCOVERY_AS_LOCALHOST", mapConfig["localHost"].(string))
	if err != nil {
		zap.L().Warn("Error setting DISCOVERY_AS_LOCALHOST environemnt variable: " + err.Error())
	}

	wallet, err := gateway.NewFileSystemWallet("wallet")
	if err != nil {
		zap.L().Warn("Failed to create wallet" + err.Error())
	}

	if !wallet.Exists(user.Label) {
		err = f.populateWallet(wallet, user)
		if err != nil {
			zap.L().Warn("Failed to populate wallet" + err.Error())
		}
	}

	ccpPath := mapConfig["ccpPath"].(string)

	f.Gateway, err = gateway.Connect(
		gateway.WithConfig(config.FromFile(filepath.Clean(ccpPath))),
		gateway.WithIdentity(wallet, user.Label))

	if err != nil {
		zap.L().Warn("Failed to connect to gateway" + err.Error())
	}

	f.Network, err = f.Gateway.GetNetwork(mapConfig["channelName"].(string))

	if err != nil {
		zap.L().Warn("Failed to get network" + err.Error())
	}

	contract := f.Network.GetContract(mapConfig["contractName"].(string))

	f.Contract = contract
}

// Called when the wallet hasn't been instantiated yet
// Creates the wallet/identity of the gateway peer we connect to
func (f *FabricInterface) populateWallet(wallet *gateway.Wallet, user types.FabricUser) error {
	identity := gateway.NewX509Identity(user.MspID, user.Cert, user.Key)

	return wallet.Put(user.Label, identity)
}

// Cleanup Finishes up and performs any post-benchmark operations.
// Can be used to format the results to parse back
func (f *FabricInterface) Cleanup() results.Results {

	// Stop the ticker
	f.ThroughputTicker.Stop()

	txLatencies := make([]float64, 0)
	var avgLatency float64

	var endTime time.Time

	success := uint(0)
	fails := uint(f.Fail)

	for _, v := range f.TransactionInfo {
		if len(v) > 1 {
			txLatency := v[1].Sub(v[0]).Milliseconds()
			txLatencies = append(txLatencies, float64(txLatency))
			avgLatency += float64(txLatency)
			if v[1].After(endTime) {
				endTime = v[1]
			}

			success++
		} else {
			fails++
		}
	}

	zap.L().Debug("Statistics being returned",
		zap.Uint("success", success),
		zap.Uint("fail", fails))

	var throughput float64

	if len(txLatencies) > 0 {
		throughput = float64(f.NumTxDone) / (endTime.Sub(f.StartTime).Seconds())
		avgLatency = avgLatency / float64(len(txLatencies))
	} else {
		avgLatency = 0
		throughput = 0
	}

	var calculatedThroughputSeconds = []float64{f.Throughputs[0]}
	for i := 1; i < len(f.Throughputs); i++ {
		calculatedThroughputSeconds = append(calculatedThroughputSeconds, float64(f.Throughputs[i]-f.Throughputs[i-1]))
	}

	zap.L().Debug("Results being returned",
		zap.Float64("throughput", throughput),
		zap.Float64("latency", avgLatency),
		zap.String("ThroughputWindow", fmt.Sprintf("%v", f.Throughputs)),
	)

	return results.Results{
		TxLatencies:       txLatencies,
		AverageLatency:    avgLatency,
		Throughput:        throughput,
		ThroughputSeconds: f.Throughputs,
		Success:           success,
		Fail:              fails,
	}
}

// throughputSeconds calculates the throughput over time, to show dynamic
func (f *FabricInterface) throughputSeconds() {
	f.ThroughputTicker = time.NewTicker(time.Duration(f.Window) * time.Second)
	seconds := float64(0)

	for {
		select {
		case <-f.ThroughputTicker.C:
			seconds += float64(f.Window)
			f.Throughputs = append(f.Throughputs, float64(f.NumTxDone-f.Fail))
		}
	}
}

// Start handles the starting aspects of the benchmark
// Is primarily used for setting the start time and allocating resources for
// metrics
func (f *FabricInterface) Start() {
	f.StartTime = time.Now()
	go f.throughputSeconds()
}

//ParseWorkload Handles the workload, converts the bytes to usable transactions.
// This takes the worker's workload and transforms into transactions
func (f *FabricInterface) ParseWorkload(workload workloadgenerators.WorkerThreadWorkload) ([][]interface{}, error) {

	// Thread workload = list of transactions in intervals
	// [interval][tx] = [][][]byte
	parsedWorkload := make([][]interface{}, 0)

	for _, v := range workload {
		intervalTxs := make([]interface{}, 0)
		for _, txBytes := range v {
			var t types.FabricTX
			err := json.Unmarshal(txBytes, &t)
			if err != nil {
				return nil, err
			}

			intervalTxs = append(intervalTxs, &t)
		}
		parsedWorkload = append(parsedWorkload, intervalTxs)
	}

	f.TotalTx = len(parsedWorkload)

	return parsedWorkload, nil
}

// ConnectOne will connect  to the blockchain node in the array slot of the given array
// (NOT NEEDED IN FABRIC) Init() already does it
func (f *FabricInterface) ConnectOne(id int) error {
	return nil
}

// ConnectAll connects to all nodes given in the hosts
//
func (f *FabricInterface) ConnectAll(primaryID int) error {
	return nil
}

// DeploySmartContract deploys the smart contract
// (NOT NEEDED IN FABRIC) Contract deployment is not something useful to
// be benchmarked in a Hyperledger Fabric implementation as it is a permissioned
// blockchain and contract deployment is something agreed upon by organisations and
//not done regularly enough to hinder throughput (usually done during while low traffic)
func (f *FabricInterface) DeploySmartContract(tx interface{}) (interface{}, error) {
	return nil, nil
}

// SendRawTransaction sends the transaction by the gateway
func (f *FabricInterface) SendRawTransaction(tx interface{}) error {

	f.submitTransaction(tx)

	return nil
}

// submitTransaction utility function to submit a transaction, to be used in a different thread
// as the main thread as it may hang
func (f *FabricInterface) submitTransaction(tx interface{}) {
	transaction := tx.(*types.FabricTX)

	zap.L().Debug("Submitting TX",
		zap.Uint64("ID", transaction.ID))

	// making note of the time we send the transaction
	f.TransactionInfo[transaction.ID] = []time.Time{time.Now()}
	atomic.AddUint64(&f.NumTxSent, 1)

	var err error

	if transaction.FunctionType == "write" {
		//submitTransaction does everything under the hood for us.
		// Rather than interacting with a single peer, the SDK will send the submitTransaction proposal
		//to every required organization’s peer in the blockchain network based on the chaincode’s endorsement policy.
		//Each of these peers will execute the requested smart contract using this proposal, to generate a transaction response
		//which it endorses (signs) and returns to the SDK. The SDK collects all the endorsed transaction responses into
		//a single transaction, which it then submits to the orderer. The orderer collects and sequences transactions from various application clients into a block of transactions.
		//These blocks are distributed to every peer in the network, where every transaction is validated and committed.
		//Finally, the SDK is notified via an event, allowing it to return control to the application.
		_, err = f.Contract.SubmitTransaction(transaction.FunctionName, transaction.Args...)

	} else {

		//EvaluteTransaction is much less expensive and only queries one peer for its world state
		_, err = f.Contract.EvaluateTransaction(transaction.FunctionName, transaction.Args...)
	}

	// transaction failed, incrementing number of done and failed transactions
	if err != nil {
		zap.L().Debug("Failed transaction",
			zap.Error(err))
		atomic.AddUint64(&f.Fail, 1)
		atomic.AddUint64(&f.NumTxDone, 1)
	}

	//transaction validated, making the note of the time of return
	f.TransactionInfo[transaction.ID] = append(f.TransactionInfo[transaction.ID], time.Now())
	atomic.AddUint64(&f.Success, 1)
	atomic.AddUint64(&f.NumTxDone, 1)

}

// SecureRead reads the value from the chain
// (NOT NEEDED IN FABRIC) SecureRead is useful in permissionless blockchains where transaction
// validation is not always clear but transactions are always clearly rejected or commited in Hyperledger Fabric
func (f *FabricInterface) SecureRead(callFunc string, callParams []byte) (interface{}, error) {
	return nil, nil
}

// GetBlockByNumber retrieves the block information at the given index
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

// ParseBlocksForTransactions retrieves block information from start to end index and
// is used as a post-benchmark check to learn about the block and transactions.
// (NOT NEEDED IN FABRIC) As transactions are confirmed to be validated whenever we submit a transaction
func (f *FabricInterface) ParseBlocksForTransactions(startNumber uint64, endNumber uint64) error {
	return nil
}

// Close the connection to the blockchain node
func (f *FabricInterface) Close() {
	f.Gateway.Close()
}
