package clientinterfaces

// This client is based off the examples:
// https://github.com/ethereum/go-ethereum/blob/master/rpc/client_example_test.go

import (
	"context"
	"diablo-benchmark/blockchains"
	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"math/big"
	"time"
)

type EthereumInterface struct {
	Nodes          [][]string          // List of the nodes host:port combinations
	PrimaryNode    *ethclient.Client   // The primary node connected for this client.
	SecondaryNodes []*ethclient.Client // The other node information (for secure reads etc.)
	WorkloadTxs    []*types.Transaction
}

// Initialise the list of nodes
func (e *EthereumInterface) Init(otherHosts [][]string) {
	e.Nodes = otherHosts
}

func (e *EthereumInterface) ParseWorkload(workload [][]byte) error {
	parsedWorkload := make([]*types.Transaction, 0)

	for _, v := range workload {
		t := types.Transaction{}
		err := t.UnmarshalJSON(v)

		if err != nil {
			return err
		}

		parsedWorkload = append(parsedWorkload, &t)
	}

	e.WorkloadTxs = parsedWorkload
	return nil
}

// Connect to one node with credentials in the ID.
func (e *EthereumInterface) ConnectOne(id int) (bool, error) {
	// If our ID is greater than the nodes we know, there's a problem!

	if id >= len(e.Nodes) {
		return false, errors.New("invalid client ID")
	}

	// Connect to the node
	c, err := ethclient.Dial(fmt.Sprintf("ws://%s:%s", e.Nodes[id][0], e.Nodes[id][1]))

	// If there's an error, raise it.
	if err != nil {
		return false, err
	}

	e.PrimaryNode = c

	return true, nil
}

// Connect to all the nodes with one primary
func (e *EthereumInterface) ConnectAll(primaryId int) (bool, error) {
	// If our ID is greater than the nodes we know, there's a problem!
	if primaryId >= len(e.Nodes) {
		return false, errors.New("invalid client primary ID")
	}

	// primary connect
	_, err := e.ConnectOne(primaryId)

	if err != nil {
		return false, err
	}

	// Connect all the others
	for idx, node := range e.Nodes {
		if idx != primaryId {
			c, err := ethclient.Dial(fmt.Sprintf("ws://%s:%s", node[0], node[1]))
			if err != nil {
				return false, err
			}

			e.SecondaryNodes = append(e.SecondaryNodes, c)
		}
	}

	return true, nil
}

func (e *EthereumInterface) DeploySmartContract(tx interface{}) (interface{}, error) {
	txSigned := tx.(*types.Transaction)
	timeoutCTX, _ := context.WithTimeout(context.Background(), 5*time.Second)

	err := e.PrimaryNode.SendTransaction(timeoutCTX, txSigned)

	if err != nil {
		return nil, err
	}

	// Wait for transaction receipt
	r, err := e.PrimaryNode.TransactionReceipt(context.Background(), txSigned.Hash())

	if err != nil {
		return nil, err
	}

	return r.ContractAddress, nil
}

func (e *EthereumInterface) SendRawTransaction(tx interface{}) error {
	// NOTE: type conversion might be slow, there might be a better way to send this.
	txSigned := tx.(*types.Transaction)
	timoutCTX, _ := context.WithTimeout(context.Background(), 5*time.Second)
	err := e.PrimaryNode.SendTransaction(timoutCTX, txSigned)

	if err != nil {
		return err
	}

	// Wait for receipt
	r, err := e.PrimaryNode.TransactionReceipt(context.Background(), txSigned.Hash())

	if err != nil {
		return err
	}

	fmt.Println(r.BlockHash)

	return nil
}

func (e *EthereumInterface) SecureRead(call_func string, call_params []byte) (interface{}, error) {

	return nil, nil
}

// Get the block information
func (e *EthereumInterface) GetBlockByNumber(index uint64) (block blockchains.GenericBlock, error error) {

	var ethBlock map[string]interface{}

	bigIndex := big.NewInt(0).SetUint64(index)

	b, err := e.PrimaryNode.BlockByNumber(context.Background(), bigIndex)

	if err != nil {
		return blockchains.GenericBlock{}, err
	}

	if &ethBlock == nil {
		return blockchains.GenericBlock{}, errors.New("nil block returned")
	}

	return blockchains.GenericBlock{
		Hash:              b.Hash().String(),
		Index:             b.NumberU64(),
		Timestamp:         b.Time(),
		TransactionNumber: b.Transactions().Len(),
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
