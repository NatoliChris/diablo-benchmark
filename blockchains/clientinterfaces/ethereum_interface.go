package clientinterfaces

// This client is based off the examples:
// https://github.com/ethereum/go-ethereum/blob/master/rpc/client_example_test.go

import (
	"errors"
	"fmt"
	"github.com/ethereum/go-ethereum/rpc"
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
	c, err := rpc.Dial(fmt.Sprintf("%s:%s", e.Nodes[id][0], e.Nodes[id][1]))

	// If there's an error, raise it.
	if err != nil {
		return false, err
	}

	e.PrimaryNode = c

	return true, nil
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
