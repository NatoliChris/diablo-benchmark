package clientinterfaces

// This client is based off the examples:
// https://github.com/ethereum/go-ethereum/blob/master/rpc/client_example_test.go

import (
	"diablo-benchmark/blockchains"
	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rpc"
	"strconv"
	"strings"
)

type EthereumInterface struct {
	Nodes          [][]string    // List of the nodes host:port combinations
	PrimaryNode    *rpc.Client   // The primary node connected for this client.
	SecondaryNodes []*rpc.Client // The other node information (for secure reads etc.)
}

// Initialise the list of nodes
func (e *EthereumInterface) Init(otherHosts [][]string) {
	e.Nodes = otherHosts
}

// Connect to one node with credentials in the ID.
func (e *EthereumInterface) ConnectOne(id int) (bool, error) {
	// If our ID is greater than the nodes we know, there's a problem!
	if id >= len(e.Nodes) {
		return false, errors.New("invalid client ID")
	}

	// Connect to the node
	c, err := rpc.Dial(fmt.Sprintf("ws://%s:%s", e.Nodes[id][0], e.Nodes[id][1]))

	// If there's an error, raise it.
	if err != nil {
		return false, err
	}

	e.PrimaryNode = c

	return true, nil
}

func (e *EthereumInterface) DeploySmartContract(contractPath string) (interface{}, error) {

	return nil, nil
}

func (e *EthereumInterface) SendRawTransaction(b []byte) (bool, error) {
	return false, nil
}

func (e *EthereumInterface) SecureRead(call_func string, call_params []byte) (interface{}, error) {

	return nil, nil
}

// Get the block information
func (e *EthereumInterface) GetBlockByNumber(index uint64) (block blockchains.GenericBlock, error error) {

	var ethBlock types.Block

	err := e.PrimaryNode.Call(&ethBlock, "eth_getBlockByNumber", index, true)

	if err != nil {
		return blockchains.GenericBlock{}, err
	}

	if &ethBlock == nil {
		return blockchains.GenericBlock{}, errors.New("nil block returned")
	}

	// If the block fails to decode (Genesis usually causes this error)
	defer func() {
		if p := recover(); p != nil {
			// Return a generic error
			block = blockchains.GenericBlock{}
			error = errors.New("failed to decode block")
		}
	}()

	return blockchains.GenericBlock{
		Hash:              ethBlock.Hash().String(),
		Index:             ethBlock.NumberU64(),
		Timestamp:         ethBlock.Time(),
		TransactionNumber: ethBlock.Transactions().Len(),
	}, nil
}

// Get the block height through the RPC interaction.
func (e *EthereumInterface) GetBlockHeight() (uint64, error) {

	// Get the hex string
	var num string
	err := e.PrimaryNode.Call(&num, "eth_blockNumber")

	if err != nil {
		return 0, err
	}

	// Convert to uint64
	height, err := strconv.ParseUint(strings.Replace(num, "0x", "", -1), 16, 64)

	if err != nil {
		return 0, err
	}

	return height, nil
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
