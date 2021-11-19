package algorand


import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"

	"golang.org/x/crypto/ed25519"

	"github.com/algorand/go-algorand-sdk/client/v2/algod"
	"github.com/algorand/go-algorand-sdk/client/v2/common/models"
	"github.com/algorand/go-algorand-sdk/crypto"
	"github.com/algorand/go-algorand-sdk/future"
	"github.com/algorand/go-algorand-sdk/mnemonic"
	algorandtx "github.com/algorand/go-algorand-sdk/transaction"
	"github.com/algorand/go-algorand-sdk/types"
)


type account struct {
	address   string                           // account blockchan address
	key       ed25519.PrivateKey                     // account private key
}

type Blockchain struct {
	clients   []*algod.Client                  // clients to endpoint nodes
	accounts  []account                              // blockchain accounts
	params  types.SuggestedParams                 // transaction parameters
}

type ContractValue struct {
	Bytes  string
	Type   uint64
	Uint   uint64
}

type ContractState struct {
	Global  map[string]ContractValue
}


func NewBlockchain(config *Config) (*Blockchain, error) {
	var pk ed25519.PrivateKey
	var cli *algod.Client
	var ret Blockchain
	var err error
	var i int

	ret.clients = make([]*algod.Client, config.Size())
	for i = 0; i < config.Size(); i++ {
		ret.clients[i], err = algod.MakeClient(
			"http://" + config.GetNodeAddress(i),
			config.GetNodeToken(i))
		if err != nil {
			return nil, err
		}
	}

	ret.accounts = make([]account, config.Population())
	for i = 0; i < config.Population(); i++ {
		pk, err = mnemonic.ToPrivateKey(config.GetAccountMnemonic(i))
		if err != nil {
			return nil, err
		}

		ret.accounts[i] = account{
			address:  config.GetAccountAddress(i),
			key:      pk,
		}
	}

	cli = ret.clients[0]
	ret.params, err = cli.SuggestedParams().Do(context.Background())
	ret.params.FirstRoundValid = 0
	ret.params.LastRoundValid = 1000
	// ret.params.Fee = 0

	return &ret, nil
}


func (this *Blockchain) Size() int {
	return len(this.clients)
}

func (this *Blockchain) Population() int {
	return len(this.accounts)
}


const clearContractSource = "#pragma version 5\nint 1\nreturn\n"

func (this *Blockchain) compileTeal(source []byte) ([]byte, error) {
	var client *algod.Client = this.clients[0]
	var compilation models.CompileResponse
	var err error

	compilation, err = client.TealCompile(source).Do(context.Background())
	if err != nil {
		return nil, err
	}

	return base64.StdEncoding.DecodeString(compilation.Result)
}

func (this *Blockchain) PrepareDeployTransaction(from int, source []byte, localInts, localBytes, globalInts, globalBytes int, note []byte) ([]byte, error) {
	var localSchema, globalSchema types.StateSchema
	var ret, approvalCode, clearCode []byte
	var utx types.Transaction
	var addr types.Address
	var err error

	localSchema = types.StateSchema{
		NumUint:       uint64(localInts),
		NumByteSlice:  uint64(localBytes),
	}

	globalSchema = types.StateSchema{
		NumUint:       uint64(globalInts),
		NumByteSlice:  uint64(globalBytes),
	}

	approvalCode, err = this.compileTeal(source)
	if err != nil {
		return nil, err
	}

	clearCode, err = this.compileTeal([]byte(clearContractSource))
	if err != nil {
		return nil, err
	}

	addr, err = types.DecodeAddress(this.accounts[from].address)
	if err != nil {
		return nil, err
	}

	utx, err = future.MakeApplicationCreateTx(false, approvalCode,
		clearCode, globalSchema, localSchema, nil, nil, nil, nil,
		this.params, addr, note, types.Digest{}, [32]byte{},
		types.Address{})
	if err != nil {
		return nil, err
	}

	_, ret, err = crypto.SignTransaction(this.accounts[from].key, utx)

	return ret, err
}

func (this *Blockchain) DeployContract(endpoint, from int, source []byte, localInts, localBytes, globalInts, globalBytes int) (uint64, error) {
	var info models.PendingTransactionInfoResponse
	var txid string
	var tx []byte
	var err error

	tx, err = this.PrepareDeployTransaction(from, source, localInts,
		localBytes, globalInts, globalBytes, nil)
	if err != nil {
		return 0, err
	}

	txid, err = this.SendTransaction(endpoint, tx)
	if err != nil {
		return 0, err
	}

	info, err = this.waitTransactionInfo(endpoint, txid)
	if err != nil {
		return 0, err
	}

	return info.ApplicationIndex, nil
}

func (this *Blockchain) ReadContractGlobalState(endpoint, from int, appid uint64) (*ContractState, error) {
	var client *algod.Client = this.clients[endpoint]
	var app models.Application
	var kv models.TealKeyValue
	var info models.Account
	var ret ContractState
	var addr string
	var key []byte
	var err error

	addr = this.accounts[from].address
	info, err = client.AccountInformation(addr).Do(context.Background())
	if err != nil {
		return nil, err
	}

	for _, app = range info.CreatedApps {
		if app.Id != appid {
			continue
		}

		ret.Global = make(map[string]ContractValue)
		for _, kv = range app.Params.GlobalState {
			key, err = base64.StdEncoding.DecodeString(kv.Key)

			if err != nil {
				return nil, err
			}

			ret.Global[string(key)] = ContractValue{
				Bytes: kv.Value.Bytes,
				Type:  kv.Value.Type,
				Uint:  kv.Value.Uint,
			}
		}

		return &ret, nil
	}

	return nil, fmt.Errorf("cannot find application %u", appid)
}


func (this *Blockchain) PrepareOptInTransaction(from int, appid uint64, note []byte) ([]byte, error) {
	var utx types.Transaction
	var addr types.Address
	var ret []byte
	var err error

	addr, err = types.DecodeAddress(this.accounts[from].address)
	if err != nil {
		return nil, err
	}

	utx, err = future.MakeApplicationOptInTx(appid, nil, nil, nil, nil,
		this.params, addr, note, types.Digest{}, [32]byte{},
		types.Address{})
	if err != nil {
		return nil, err
	}

	_, ret, err = crypto.SignTransaction(this.accounts[from].key, utx)

	return ret, err
}


func (this *Blockchain) PrepareNoOpTransaction(from int, appid uint64, args [][]byte, note []byte) ([]byte, error) {
	var utx types.Transaction
	var addr types.Address
	var ret []byte
	var err error

	addr, err = types.DecodeAddress(this.accounts[from].address)
	if err != nil {
		return nil, err
	}

	utx, err = future.MakeApplicationNoOpTx(appid, args, nil, nil, nil,
		this.params, addr, note, types.Digest{}, [32]byte{},
		types.Address{})
	if err != nil {
		return nil, err
	}

	_, ret, err = crypto.SignTransaction(this.accounts[from].key, utx)

	return ret, err
}


func (this *Blockchain) PrepareSimpleTransaction(from, to, amount int, note []byte) ([]byte, error) {
	var utx types.Transaction
	var ret []byte
	var err error

	utx, err = algorandtx.MakePaymentTxnWithFlatFee(
		this.accounts[from].address, this.accounts[to].address, 0,
		uint64(amount), uint64(this.params.FirstRoundValid),
		uint64(this.params.LastRoundValid), note, "",
		this.params.GenesisID, this.params.GenesisHash)
	if err != nil {
		return nil, err
	}

	_, ret, err = crypto.SignTransaction(this.accounts[from].key, utx)

	return ret, err
}


func (this *Blockchain) SendTransaction(endpoint int, raw []byte) (string, error) {
	var client *algod.Client = this.clients[endpoint]

	return client.SendRawTransaction(raw).Do(context.Background())
}


func (this *Blockchain) waitNextRound(endpoint int, round *uint64) error {
	var client *algod.Client = this.clients[endpoint]
	var s models.NodeStatus
	var err error

	if *round == 0 {
		s, err = client.Status().Do(context.Background())
		if err != nil {
			return err
		}

		*round = s.LastRound + 1
	} else {
		*round += 1
	}

	// Waiting happens here
	s, err = client.StatusAfterBlock(*round).Do(context.Background())

	return nil
}

func (this *Blockchain) waitTransactionInfo(endpoint int, txid string) (models.PendingTransactionInfoResponse, error) {
	var client *algod.Client = this.clients[endpoint]
	var info models.PendingTransactionInfoResponse
	var round uint64 = 0
	var err error

	for {
		info, _, err = client.PendingTransactionInformation(txid).Do(context.Background())
		if err != nil {
			return info, err
		}

		if info.PoolError != "" {
			return info, errors.New(info.PoolError)
		}

		if info.ConfirmedRound > 0 {
			return info, nil
		}

		err = this.waitNextRound(endpoint, &round)
		if err != nil {
			return info, err
		}
	}
}

func (this *Blockchain) WaitTransaction(endpoint int, txid string) error {
	var err error

	_, err = this.waitTransactionInfo(endpoint, txid)

	return err
}


func (this *Blockchain) PollBlock(endpoint int, round uint64) (uint64, [][]byte, error) {
	var client *algod.Client = this.clients[endpoint]
	var ret [][]byte = make([][]byte, 0)
	var tx types.SignedTxnInBlock
	var status models.NodeStatus
	var block types.Block
	var err error

	if round == 0 {
		status, err = client.Status().Do(context.Background())
		if err != nil {
			return 0, nil, err
		}

		round = status.LastRound + 1
	}

	// Waiting happens here
	status, err = client.StatusAfterBlock(round).Do(context.Background())
	round += 1

	if err != nil {
		return round, nil, err
	}

	block, err = client.Block(round).Do(context.Background())
	if err != nil {
		return round, nil, err
	}
	
	for _, tx = range block.Payset {
		ret = append(ret, tx.Txn.Note)
	}

	return round, ret, nil
}
