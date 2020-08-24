package clientinterfaces

// This client is based off the examples:
// https://github.com/ethereum/go-ethereum/blob/master/rpc/client_example_test.go

import (
	"context"
	"diablo-benchmark/blockchains/workloadgenerators"
	"diablo-benchmark/core/results"
	"errors"
	"fmt"
	ethtypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"go.uber.org/zap"
	"math/big"
	"sync/atomic"
	"time"
)

type EthereumInterface struct {
	Nodes           []string               // List of the nodes host:port combinations
	PrimaryNode     *ethclient.Client      // The primary node connected for this client.
	SecondaryNodes  []*ethclient.Client    // The other node information (for secure reads etc.)
	SubscribeDone   chan bool              // Event channel that will unsub from events
	TransactionInfo map[string][]time.Time // Transaction information
	HandlersStarted bool                   // Have the handlers been initiated?
	NumTxDone       uint64                 // Number of transactions done
	NumTxSent       uint64                 // Number of transactions currently sent
	TotalTx         int                    // TotalTx
}

// Initialise the list of nodes
func (e *EthereumInterface) Init(otherHosts []string) {
	e.Nodes = otherHosts
	e.TransactionInfo = make(map[string][]time.Time, 0)
	e.SubscribeDone = make(chan bool)
	e.HandlersStarted = false
	e.NumTxDone = 0
}

// Cleans up and formats the results
func (e *EthereumInterface) Cleanup() results.Results {
	// clean up connections and format results
	if e.HandlersStarted {
		e.SubscribeDone <- true
	}

	txLatencies := make([]float64, 0)
	var avgLatency float64 = 0

	startTime := time.Now()
	var endTime time.Time

	for _, v := range e.TransactionInfo {
		if len(v) > 1 {
			txLatency := v[1].Sub(v[0]).Milliseconds()
			txLatencies = append(txLatencies, float64(txLatency))
			avgLatency += float64(txLatency)
			if v[1].After(endTime) {
				endTime = v[1]
			}
		}
		if startTime.After(v[0]) {
			startTime = v[0]
		}
	}

	throughput := float64(e.NumTxDone) / (endTime.Sub(startTime).Seconds())

	return results.Results{
		TxLatencies:    txLatencies,
		AverageLatency: avgLatency / float64(len(txLatencies)),
		Throughput:     throughput,
	}
}

// Parses the workload and convert into the type for the benchmark.
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

func (e *EthereumInterface) parseBlocksForTransactions(blockNumber *big.Int) {
	block, err := e.PrimaryNode.BlockByNumber(context.Background(), blockNumber)

	if err != nil {
		zap.L().Warn(err.Error())
		return
	}

	tNow := time.Now()
	for _, v := range block.Transactions() {
		tHash := v.Hash().String()
		if _, ok := e.TransactionInfo[tHash]; ok {
			e.TransactionInfo[tHash] = append(e.TransactionInfo[tHash], tNow)
		}
	}

	atomic.AddUint64(&e.NumTxDone, uint64(len(block.Transactions())))
}

// Handles the incoming information about the Transactions
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

func (e *EthereumInterface) ParseBlocksForTransactions(startNumber uint64, endNumber uint64) error {
	for i := startNumber; i <= endNumber; i++ {
		b, err := e.GetBlockByNumber(i)

		if err != nil {
			return err
		}

		for _, v := range b.TransactionHashes {
			if _, ok := e.TransactionInfo[v]; ok {
				e.TransactionInfo[v] = append(e.TransactionInfo[v], time.Unix(int64(b.Timestamp), 0))
			}
		}
	}

	return nil
}

// Connect to one node with credentials in the ID.
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

// Connect to all the nodes with one primary
func (e *EthereumInterface) ConnectAll(primaryId int) error {
	// If our ID is greater than the nodes we know, there's a problem!
	if primaryId >= len(e.Nodes) {
		return errors.New("invalid client primary ID")
	}

	// primary connect
	err := e.ConnectOne(primaryId)

	if err != nil {
		return err
	}

	// Connect all the others
	for idx, node := range e.Nodes {
		if idx != primaryId {
			c, err := ethclient.Dial(fmt.Sprintf("ws://%s", node))
			if err != nil {
				return err
			}

			e.SecondaryNodes = append(e.SecondaryNodes, c)
		}
	}

	return nil
}

// Deploy the smart contract, respond with the contract address.
func (e *EthereumInterface) DeploySmartContract(tx interface{}) (interface{}, error) {
	txSigned := tx.(*ethtypes.Transaction)
	timeoutCTX, _ := context.WithTimeout(context.Background(), 5*time.Second)

	err := e.PrimaryNode.SendTransaction(timeoutCTX, txSigned)

	if err != nil {
		return nil, err
	}

	// TODO: fix to wait for deploy
	// Wait for transaction receipt
	r, err := e.PrimaryNode.TransactionReceipt(context.Background(), txSigned.Hash())

	if err != nil {
		return nil, err
	}

	return r.ContractAddress, nil
}

func (e *EthereumInterface) SendRawTransaction(tx interface{}) error {
	// NOTE: type conversion might be slow, there might be a better way to send this.
	txSigned := tx.(*ethtypes.Transaction)
	timoutCTX, _ := context.WithTimeout(context.Background(), 5*time.Second)
	e.TransactionInfo[txSigned.Hash().String()] = []time.Time{time.Now()}
	err := e.PrimaryNode.SendTransaction(timoutCTX, txSigned)

	if err != nil {
		return err
	}

	atomic.AddUint64(&e.NumTxSent, 1)
	return nil
}

func (e *EthereumInterface) SecureRead(call_func string, call_params []byte) (interface{}, error) {
	// TODO implement
	return nil, nil
}

// Get the block information
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

// Get the block height through the RPC interaction.
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
