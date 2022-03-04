package ndiem


import (
	"bytes"
	"context"
	"diablo-benchmark/core"
	"fmt"
	"golang.org/x/crypto/ed25519"

	"github.com/diem/client-sdk-go/diemclient"
	"github.com/diem/client-sdk-go/diemjsonrpctypes"
	"github.com/diem/client-sdk-go/diemkeys"
	"github.com/diem/client-sdk-go/diemtypes"
)


type BlockchainBuilder struct {
	logger           core.Logger
	client           diemclient.Client
	ctx              context.Context
	premadeAccounts  []account
	usedAccounts     int
}

type account struct {
	key       ed25519.PrivateKey
	addr      diemtypes.AccountAddress
	sequence  uint64
}


func newBuilder(logger core.Logger, client diemclient.Client, ctx context.Context) *BlockchainBuilder {
	return &BlockchainBuilder{
		logger: logger,
		client: client,
		ctx: ctx,
		premadeAccounts: make([]account, 0),
		usedAccounts: 0,
	}
}

func (this *BlockchainBuilder) addAccount(key ed25519.PrivateKey) {
	var addr diemtypes.AccountAddress
	var keys *diemkeys.Keys

	keys = diemkeys.NewKeysFromPublicAndPrivateKeys(
		diemkeys.NewEd25519PublicKey(key.Public().(ed25519.PublicKey)),
		diemkeys.NewEd25519PrivateKey(key))

	addr = keys.AccountAddress()

	this.premadeAccounts = append(this.premadeAccounts, account{
		key: key,
		addr: addr,
		sequence: 0,
	})
}

func (this *BlockchainBuilder) CreateAccount(stake int) (interface{}, error) {
	var acc *diemjsonrpctypes.Account
	var ret *account
	var err error

	if this.usedAccounts < len(this.premadeAccounts) {
		ret = &this.premadeAccounts[this.usedAccounts]
		this.usedAccounts += 1

		acc, err = this.client.GetAccount(ret.addr)
		if err != nil {
			return nil, err
		}

		if acc == nil {
			return nil, fmt.Errorf("account does not exist")
		}

		ret.sequence = acc.SequenceNumber
		this.logger.Tracef("sequence = %d", ret.sequence)
	} else {
		return nil, fmt.Errorf("can only use %d premade accounts",
			this.usedAccounts)
	}

	return ret, nil
}

func (this *BlockchainBuilder) CreateContract(name string) (interface{}, error) {
	return 0, nil
}

func (this *BlockchainBuilder) CreateResource(domain string) (core.SampleFactory, bool) {
	return nil, false
}

func (this *BlockchainBuilder) EncodeTransfer(amount int, from, to interface{}, info core.InteractionInfo) ([]byte, error) {
	var tx *transferTransaction
	var buffer bytes.Buffer
	var err error

	tx = newTransferTransaction(from.(*account).key, to.(*account).addr,
		uint64(amount), from.(*account).sequence)

	this.logger.Tracef("encode = %v %d", from.(*account).key.Seed()[:3], from.(*account).sequence)

	err = tx.encode(&buffer)
	if err != nil {
		return nil, err
	}

	from.(*account).sequence += 1

	return buffer.Bytes(), nil
}

func (this *BlockchainBuilder) EncodeInvoke(from interface{}, contract interface{}, function string, info core.InteractionInfo) ([]byte, error) {
	return []byte{}, nil
}

func (this *BlockchainBuilder) EncodeInteraction(itype string, expr core.BenchmarkExpression, info core.InteractionInfo) ([]byte, error) {
	return []byte{}, nil
}
