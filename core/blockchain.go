package core


import (
	"fmt"
	"encoding/binary"
)


type Interaction interface {
	Payload() interface{}

	ReportSubmit()

	ReportCommit()

	ReportAbort()
}


type BlockchainInterface interface {
	// Create a client for the given `view` of this blockchain.
	// A `view` is a list of addresses indicating how to contact the
	// blockchain endpoints (i.e. the nodes).
	// These addresses are among the ones specified in the setup
	// configuration file and the address format is used specified.
	//
	Client(view []string) (BlockchainClient, error)

	CreateBalance(int) (interface{}, error)

	//
	// Interactions implemented by the blockchain.
	// If the blockchain does not implement a specific interaction, the
	// associated encoding method returns an error.
	//

	// Encode a transfer interaction.
	// A transfer moves a fungible amount of currencies `stake` from an
	// account balance `from` to an account balance `to`.
	//
	EncodeTransfer(int, interface{}, interface{}) ([]byte, error)
}

type BlockchainClient interface {
	DecodePayload(bytes []byte) (interface{}, error)

	TriggerInteraction(iact Interaction) error
}


type TestBlockchainInterface struct {
	nextAccountId  int
}

func (this *TestBlockchainInterface) Client(view []string) (BlockchainClient, error) {
	var client testBlockchainClient

	client.init(view)

	return &client, nil
}

func (this *TestBlockchainInterface) CreateBalance(stake int) (interface{}, error) {
	var id int = this.nextAccountId

	fmt.Printf("mint new account %d (with stake %d)\n", id, stake)

	this.nextAccountId += 1

	return id, nil
}

func (this *TestBlockchainInterface) EncodeTransfer(stake int, from, to interface{}) ([]byte, error) {
	var bytes []byte = make([]byte, 12)

	binary.LittleEndian.PutUint32(bytes[0:], uint32(stake))
	binary.LittleEndian.PutUint32(bytes[4:], uint32(from.(int)))
	binary.LittleEndian.PutUint32(bytes[8:], uint32(to.(int)))

	return bytes, nil
}


type testBlockchainClient struct {
	view  []string
}

func (this *testBlockchainClient) init(view []string) {
	this.view = view
}

func (this *testBlockchainClient) DecodePayload(bytes []byte) (interface{}, error) {
	var tx testDecodedTransaction

	if len(bytes) != 12 {
		return nil, fmt.Errorf("not a valid payload")
	}

	tx.stake = int(binary.LittleEndian.Uint32(bytes[0:]))
	tx.from = int(binary.LittleEndian.Uint32(bytes[4:]))
	tx.to = int(binary.LittleEndian.Uint32(bytes[8:]))

	fmt.Printf("transfer: %d : %d => %d\n", tx.stake, tx.from, tx.to)

	return &tx, nil
}

func (this *testBlockchainClient) TriggerInteraction(iact Interaction) error {
	iact.ReportSubmit()
	iact.ReportCommit()
	return nil
}


type testDecodedTransaction struct {
	stake  int
	from   int
	to     int
}
