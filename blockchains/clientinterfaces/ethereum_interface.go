package clientinterfaces

// This client is based off the examples:
// https://github.com/ethereum/go-ethereum/blob/master/rpc/client_example_test.go

import (
	"context"
	"diablo-benchmark/blockchains/workloadgenerators"
	"diablo-benchmark/core/configs"
	"diablo-benchmark/core/results"
	"errors"
	"fmt"
	"math/big"
	"sync"
	"sync/atomic"
	"time"

	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"go.uber.org/zap"
)

// EthereumInterface is the the Ethereum implementation of the clientinterface
// Provides functionality to interaact with the Ethereum blockchain
type EthereumInterface struct {
	PrimaryNode      *ethclient.Client      // The primary node connected for this client.
	SecondaryNodes   []*ethclient.Client    // The other node information (for secure reads etc.)
	SubscribeDone    chan bool              // Event channel that will unsub from events
	TransactionInfo  map[string][]time.Time // Transaction information
	bigLock          sync.Mutex
	HandlersStarted  bool         // Have the handlers been initiated?
	StartTime        time.Time    // Start time of the benchmark
	ThroughputTicker *time.Ticker // Ticker for throughput (1s)
	Throughputs      []float64    // Throughput over time with 1 second intervals

	// Quick fix for OSDI22
	rrEndpoint int

	GenericInterface
}

// Init initialises the list of nodes
func (e *EthereumInterface) Init(chainConfig *configs.ChainConfig) {
	e.Nodes = chainConfig.Nodes
	e.TransactionInfo = make(map[string][]time.Time, 0)
	e.SubscribeDone = make(chan bool)
	e.HandlersStarted = false
	e.NumTxDone = 0
	e.rrEndpoint = 0
}

// Cleanup formats results and unsubscribes from the blockchain
func (e *EthereumInterface) Cleanup() results.Results {
	// Stop the ticker
	e.ThroughputTicker.Stop()

	// clean up connections and format results
	if e.HandlersStarted {
		e.SubscribeDone <- true
	}

	txLatencies := make([]float64, 0)
	var avgLatency float64

	var endTime time.Time

	success := uint(0)
	fails := uint(e.Fail)

	zap.L().Debug("Fail", zap.Uint64("count", e.Fail))

	for _, v := range e.TransactionInfo {
		if len(v) > 1 {
			if v[0] == v[1] {
				continue
			}
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

	zap.L().Debug("TransactionInfo", zap.Int("len", len(e.TransactionInfo)))

	zap.L().Debug("Statistics being returned",
		zap.Uint("success", success),
		zap.Uint("fail", fails))

	// Calculate the throughput and latencies
	var throughput float64
	if len(txLatencies) > 0 {
		throughput = (float64(e.NumTxDone) - float64(e.Fail)) / (endTime.Sub(e.StartTime).Seconds())
		avgLatency = avgLatency / float64(len(txLatencies))
	} else {
		avgLatency = 0
		throughput = 0
	}

	averageThroughput := float64(0)
	var calculatedThroughputSeconds = []float64{e.Throughputs[0]}
	for i := 1; i < len(e.Throughputs); i++ {
		calculatedThroughputSeconds = append(calculatedThroughputSeconds, float64(e.Throughputs[i]-e.Throughputs[i-1]))
		averageThroughput += float64(e.Throughputs[i] - e.Throughputs[i-1])
	}

	averageThroughput = averageThroughput / float64(len(e.Throughputs))

	zap.L().Debug("Results being returned",
		zap.Float64("avg throughput", averageThroughput),
		zap.Float64("throughput (as is)", throughput),
		zap.Float64("latency", avgLatency),
		zap.String("ThroughputWindow", fmt.Sprintf("%v", calculatedThroughputSeconds)),
	)

	return results.Results{
		TxLatencies:       txLatencies,
		AverageLatency:    avgLatency,
		Throughput:        averageThroughput,
		ThroughputSeconds: calculatedThroughputSeconds,
		Success:           success,
		Fail:              fails,
	}
}

// throughputSeconds calculates the throughput over time, to show dynamic
func (e *EthereumInterface) throughputSeconds() {
	e.ThroughputTicker = time.NewTicker((time.Duration(e.Window) * time.Second))
	seconds := float64(0)

	for {
		select {
		case <-e.ThroughputTicker.C:
			seconds += float64(e.Window)
			e.Throughputs = append(e.Throughputs, float64(e.NumTxDone-e.Fail))
		}
	}
}

// Start sets up the start time and starts the periodic checking of the
// throughput.
func (e *EthereumInterface) Start() {
	e.StartTime = time.Now()
	go e.throughputSeconds()
}

// ParseWorkload parses the workload and converts into the type for the benchmark.
func (e *EthereumInterface) ParseWorkload(workload workloadgenerators.WorkerThreadWorkload) ([][]interface{}, error) {
	parsedWorkload := make([][]interface{}, 0)

	for _, v := range workload {
		intervalTxs := make([]interface{}, 0)
		for _, txBytes := range v {
			t := ethtypes.Transaction{}
			err := t.UnmarshalJSON(txBytes)
			if err != nil {
				return nil, err
			}

			intervalTxs = append(intervalTxs, &t)
		}
		parsedWorkload = append(parsedWorkload, intervalTxs)
	}

	e.TotalTx = len(parsedWorkload)

	return parsedWorkload, nil
}

// parseBlocksForTransactions parses the the given block number for the transactions
func (e *EthereumInterface) parseBlocksForTransactions(blockNumber *big.Int) {
	block, err := e.PrimaryNode.BlockByNumber(context.Background(), blockNumber)

	if err != nil {
		zap.L().Warn(err.Error())
		return
	}

	tNow := time.Now()
	var tAdd uint64

	e.bigLock.Lock()

	for _, v := range block.Transactions() {
		tHash := v.Hash().String()
		if _, ok := e.TransactionInfo[tHash]; ok {
			e.TransactionInfo[tHash] = append(e.TransactionInfo[tHash], tNow)
			tAdd++
		}
	}

	e.bigLock.Unlock()

	atomic.AddUint64(&e.NumTxDone, tAdd)
}

// EventHandler subscribes to the blocks and handles the incoming information about the transactions
func (e *EthereumInterface) EventHandler() {
	// Channel for the events
	eventCh := make(chan *ethtypes.Header)

	sub, err := e.PrimaryNode.SubscribeNewHead(context.Background(), eventCh)
	if err != nil {
		zap.Error(err)
		return
	}

	for {
		select {
		case <-e.SubscribeDone:
			sub.Unsubscribe()
			return
		case header := <-eventCh:
			// Got a head
			go e.parseBlocksForTransactions(header.Number)
		case err := <-sub.Err():
			zap.L().Warn(err.Error())
		}
	}
}

// ParseBlocksForTransactions Goes through all the blocks between start and end index, and check for the
// transactions contained in the blocks. This can help with (A) latency, and
// (B) correctness to ensure that committed transactions are actually in the blocks.
func (e *EthereumInterface) ParseBlocksForTransactions(startNumber uint64, endNumber uint64) error {
	for i := startNumber; i <= endNumber; i++ {
		b, err := e.GetBlockByNumber(i)

		if err != nil {
			return err
		}

		e.bigLock.Lock()

		for _, v := range b.TransactionHashes {
			if _, ok := e.TransactionInfo[v]; ok {
				e.TransactionInfo[v] = append(e.TransactionInfo[v], time.Unix(int64(b.Timestamp), 0))
			}
		}

		e.bigLock.Unlock()
	}

	return nil
}

// ConnectOne connects to one node with the node index matching the "ID".
func (e *EthereumInterface) ConnectOne(id int) error {
	// If our ID is greater than the nodes we know, there's a problem!

	if id >= len(e.Nodes) {
		return errors.New("invalid client ID")
	}

	// Connect to the node
	c, err := ethclient.Dial(fmt.Sprintf("ws://%s", e.Nodes[id]))

	// If there's an error, raise it.
	if err != nil {
		return err
	}

	e.PrimaryNode = c

	if !e.HandlersStarted {
		go e.EventHandler()
		e.HandlersStarted = true
	}

	return nil
}

// ConnectAll connects to all nodes given in the hosts
func (e *EthereumInterface) ConnectAll(primaryID int) error {
	// If our ID is greater than the nodes we know, there's a problem!
	// OSDI22 fix: there is no problem, keep going...
	if primaryID >= len(e.Nodes) {
		// return errors.New("invalid client primary ID")
		primaryID = primaryID % len(e.Nodes)
	}

	// primary connect
	err := e.ConnectOne(primaryID)

	if err != nil {
		return err
	}

	// Connect all the others
	for idx, node := range e.Nodes {
		if idx != primaryID {
			c, err := ethclient.Dial(fmt.Sprintf("ws://%s", node))
			if err != nil {
				return err
			}

			e.SecondaryNodes = append(e.SecondaryNodes, c)
		}
	}

	return nil
}

// DeploySmartContract will deploy the transaction and wait for the contract address to be returned.
func (e *EthereumInterface) DeploySmartContract(tx interface{}) (interface{}, error) {
	txSigned := tx.(*ethtypes.Transaction)
	timeoutCTX, _ := context.WithTimeout(context.Background(), 5*time.Second)

	err := e.PrimaryNode.SendTransaction(timeoutCTX, txSigned)

	if err != nil {
		return nil, err
	}

	// TODO: fix to wait for deploy - look at workloadGenerator!
	// Wait for transaction receipt
	r, err := e.PrimaryNode.TransactionReceipt(context.Background(), txSigned.Hash())

	if err != nil {
		return nil, err
	}

	return r.ContractAddress, nil
}

func (e *EthereumInterface) _sendTx(endpoint int, txSigned ethtypes.Transaction) {
	// timoutCTX, _ := context.WithTimeout(context.Background(), 5*time.Second)

	var client *ethclient.Client

	if endpoint == 0 {
		client = e.PrimaryNode
	} else {
		client = e.SecondaryNodes[endpoint-1]
	}

	sendTime := time.Now()
	transactionInfo := []time.Time{sendTime}
	err := client.SendTransaction(context.Background(), &txSigned)

	// The transaction failed - this could be if it was reproposed, or, just failed.
	// We need to make sure that if it was re-proposed it doesn't count as a "success" on this node.
	if err != nil {
		zap.L().Debug("Err",
			zap.Error(err),
			zap.String("sendTime", sendTime.String()),
		)
		atomic.AddUint64(&e.Fail, 1)
		atomic.AddUint64(&e.NumTxDone, 1)
		transactionInfo = append(transactionInfo, sendTime)
	}

	e.bigLock.Lock()
	e.TransactionInfo[txSigned.Hash().String()] = transactionInfo
	e.bigLock.Unlock()

	atomic.AddUint64(&e.NumTxSent, 1)
}

// SendRawTransaction sends a raw transaction to the blockchain node.
// It assumes that the transaction is the correct type
// and has already been signed and is ready to send into the network.
func (e *EthereumInterface) SendRawTransaction(tx interface{}) error {
	// NOTE: type conversion might be slow, there might be a better way to send this.
	txSigned := tx.(*ethtypes.Transaction)
	var endpoint = e.rrEndpoint

	e.rrEndpoint = (e.rrEndpoint + 1) % (len(e.SecondaryNodes) + 1)

	go e._sendTx(endpoint, *txSigned)

	return nil
}

// SecureRead will implement a "secure read" - will read a value from all connected nodes to ensure that the
// value is the same.
func (e *EthereumInterface) SecureRead(callFunc string, callPrams []byte) (interface{}, error) {
	// TODO implement
	return nil, nil
}

// GetBlockByNumber will request the block information by passing it the height number.
func (e *EthereumInterface) GetBlockByNumber(index uint64) (block GenericBlock, error error) {

	var ethBlock map[string]interface{}
	var txList []string

	bigIndex := big.NewInt(0).SetUint64(index)

	b, err := e.PrimaryNode.BlockByNumber(context.Background(), bigIndex)

	if err != nil {
		return GenericBlock{}, err
	}

	if &ethBlock == nil {
		return GenericBlock{}, errors.New("nil block returned")
	}

	for _, v := range b.Transactions() {
		txList = append(txList, v.Hash().String())
	}

	return GenericBlock{
		Hash:              b.Hash().String(),
		Index:             b.NumberU64(),
		Timestamp:         b.Time(),
		TransactionNumber: b.Transactions().Len(),
		TransactionHashes: txList,
	}, nil
}

// GetBlockHeight will get the block height through the RPC interaction. Should return the index
// of the block.
func (e *EthereumInterface) GetBlockHeight() (uint64, error) {

	h, err := e.PrimaryNode.HeaderByNumber(context.Background(), nil)

	if err != nil {
		return 0, err
	}

	return h.Number.Uint64(), nil
}

// Close all the client connections
func (e *EthereumInterface) Close() {
	// Close the main client connection
	e.PrimaryNode.Close()

	// Close all other connections
	for _, client := range e.SecondaryNodes {
		client.Close()
	}
}
