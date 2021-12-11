package diem


import (
	"crypto/ed25519"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/diem/client-sdk-go/diemclient"
	"github.com/diem/client-sdk-go/diemjsonrpctypes"
	"github.com/diem/client-sdk-go/diemkeys"
	"github.com/diem/client-sdk-go/diemsigner"
	"github.com/diem/client-sdk-go/diemtypes"
	"github.com/diem/client-sdk-go/stdlib"
)


const chainId = 4
const currency = "XUS"
const expirationDelay = 86400 * time.Second


type blockchain struct {
	clients      []diemclient.Client
	accounts     []*diemkeys.Keys
	waitingTime  time.Duration
}


func newBlockchain(conf *config) (*blockchain, error) {
	var dsk *diemkeys.Ed25519PrivateKey
	var dpk *diemkeys.Ed25519PublicKey
	var sk ed25519.PrivateKey
	var pk ed25519.PublicKey
	var ret blockchain
	var addr string
	var seed []byte
	var err error
	var i int


	ret.clients = make([]diemclient.Client, conf.size())

	for i = 0; i < conf.size(); i++ {
		addr = fmt.Sprintf("http://%s", conf.getNodeUrl(i))
		ret.clients[i] = diemclient.New(chainId, addr)
	}


	ret.accounts = make([]*diemkeys.Keys, conf.population())

	for i = 0; i < conf.population(); i++ {
		seed, err = hex.DecodeString(conf.getAccountKey(i))
		if err != nil {
			return nil, err
		}

		sk = ed25519.NewKeyFromSeed(seed)
		pk = sk.Public().(ed25519.PublicKey)
		dsk = diemkeys.NewEd25519PrivateKey(sk)
		dpk = diemkeys.NewEd25519PublicKey(pk)

		ret.accounts[i] =
			diemkeys.NewKeysFromPublicAndPrivateKeys(dpk, dsk)
	}

	ret.waitingTime = 30 * time.Second

	return &ret, nil
}

func (this *blockchain) size() int {
	return len(this.clients)
}

func (this *blockchain) population() int {
	return len(this.accounts)
}

func (this *blockchain) prepareSimpleTransaction(from, to, amount int, sequence uint64) ([]byte, error) {
	var stx *diemtypes.SignedTransaction
	var address diemtypes.AccountAddress
	var script diemtypes.Script
	var expiration uint64

	address = this.accounts[from].AccountAddress()
	expiration = uint64(time.Now().Add(expirationDelay).Unix())

	script = stdlib.EncodePeerToPeerWithMetadataScript(
		diemtypes.Currency(currency),
		this.accounts[to].AccountAddress(),
		uint64(amount),
		nil,
		nil)

	stx = diemsigner.Sign(this.accounts[from], address, sequence, script,
		1000000, 0, "XUS", expiration, chainId)

	return stx.BcsSerialize()
}

func (this *blockchain) sendTransaction(endpoint int, raw []byte) error {
	var stx diemtypes.SignedTransaction
	var err error

	stx, err = diemtypes.BcsDeserializeSignedTransaction(raw)
	if err != nil {
		return err
	}

	return this.clients[endpoint].SubmitTransaction(&stx)
}

func (this *blockchain) waitTransaction(endpoint int, raw []byte) error {
	var transaction *diemjsonrpctypes.Transaction
	var stx diemtypes.SignedTransaction
	var err error

	stx, err = diemtypes.BcsDeserializeSignedTransaction(raw)
	if err != nil {
		return err
	}

	transaction, err = this.clients[endpoint].
		WaitForTransaction2(&stx, this.waitingTime)
	if err != nil {
		return err
	}

	if transaction.VmStatus.GetType() != "executed" {
		return fmt.Errorf("transaction failed (%s)",
			transaction.VmStatus.GetType())
	}

	return nil
}
