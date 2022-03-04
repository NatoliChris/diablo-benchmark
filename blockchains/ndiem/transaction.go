package ndiem


import (
	"diablo-benchmark/util"
	"encoding/binary"
	"fmt"
	"golang.org/x/crypto/ed25519"
	"io"
	"time"

	"github.com/diem/client-sdk-go/diemkeys"
	"github.com/diem/client-sdk-go/diemsigner"
	"github.com/diem/client-sdk-go/diemtypes"
	"github.com/diem/client-sdk-go/stdlib"
)


const (
	transaction_type_transfer uint8 = 0

	currency = "XUS"

	maximumGasAmount = 1_000_000

	expirationDelay = 86400 * time.Second
)


type transaction interface {
	getSigned() (*diemtypes.SignedTransaction, error)
	getName() string
}


type outerTransaction struct {
	inner  virtualTransaction
}

func (this *outerTransaction) getSigned() (*diemtypes.SignedTransaction, error) {
	var inner virtualTransaction
	var stx *diemtypes.SignedTransaction
	var err error

	inner, stx, err = this.inner.getSigned()

	this.inner = inner

	return stx, err
}

func (this *outerTransaction) getName() string {
	return this.inner.getName()
}

func decodeTransaction(src io.Reader) (*outerTransaction, error) {
	var inner virtualTransaction
	var txtype uint8
	var err error

	err = util.NewMonadInputReader(src).
		SetOrder(binary.LittleEndian).
		ReadUint8(&txtype).
		Error()
	if err != nil {
		return nil, err
	}

	switch (txtype) {
	case transaction_type_transfer:
		inner, err = decodeTransferTransaction(src)
	default:
		return nil, fmt.Errorf("unknown transaction type %d", txtype)
	}

	if err != nil {
		return nil, err
	}

	return &outerTransaction{ inner }, nil
}


type virtualTransaction interface {
	getSigned() (virtualTransaction, *diemtypes.SignedTransaction, error)
	getName() string
}


type signedTransaction struct {
	inner  *diemtypes.SignedTransaction
	name   string
}

func newSignedTransaction(inner *diemtypes.SignedTransaction, name string) *signedTransaction {
	return &signedTransaction{
		inner: inner,
		name: name,
	}
}

func (this *signedTransaction) getSigned() (virtualTransaction, *diemtypes.SignedTransaction, error) {
	return this, this.inner, nil
}

func (this *signedTransaction) getName() string {
	return this.name
}


type unsignedTransaction struct {
	from            *diemkeys.Keys
	to              diemtypes.AccountAddress
	sequence        uint64
	script          diemtypes.Script
	maxGasAmount    uint64
	gasUnitPrice    uint64
	expirationTime  uint64
	chainId         byte
	name            string
}

func newUnsignedTransaction(from *diemkeys.Keys, to diemtypes.AccountAddress, sequence uint64, script diemtypes.Script, name string) *unsignedTransaction {
	return &unsignedTransaction{
		from: from,
		to: to,
		sequence: sequence,
		script: script,
		name: name,
	}
}

func (this *unsignedTransaction) getSigned() (virtualTransaction, *diemtypes.SignedTransaction, error) {
	var stx *diemtypes.SignedTransaction
	var expiration uint64

	expiration = uint64(time.Now().Add(expirationDelay).Unix())

	stx = diemsigner.Sign(this.from, this.to, this.sequence, this.script,
		maximumGasAmount, 0, currency, expiration, chainId)

	return newSignedTransaction(stx, this.name).getSigned()
}

func (this *unsignedTransaction) getName() string {
	return this.name
}


type transferTransaction struct {
	from      ed25519.PrivateKey
	to        diemtypes.AccountAddress
	amount    uint64
	sequence  uint64
}

func newTransferTransaction(from ed25519.PrivateKey, to diemtypes.AccountAddress, amount, sequence uint64) *transferTransaction {
	return &transferTransaction{
		from: from,
		to: to,
		amount: amount,
		sequence: sequence,
	}
}

func decodeTransferTransaction(src io.Reader) (*transferTransaction, error) {
	var addr diemtypes.AccountAddress
	var amount, sequence uint64
	var fromSeed, toAddr []byte
	var toLen int
	var err error

	err = util.NewMonadInputReader(src).
		SetOrder(binary.LittleEndian).
		ReadUint8(&toLen).
		ReadBytes(&fromSeed, ed25519.SeedSize).
		ReadBytes(&toAddr, toLen).
		ReadUint64(&amount).
		ReadUint64(&sequence).
		Error()

	if err != nil {
		return nil, err
	}

	addr, err = diemtypes.BcsDeserializeAccountAddress(toAddr)
	if err != nil {
		return nil, err
	}

	return newTransferTransaction(ed25519.NewKeyFromSeed(fromSeed), addr,
		amount, sequence), nil
}

func (this *transferTransaction) encode(dest io.Writer) error {
	var to []byte
	var err error

	to, err = this.to.BcsSerialize()
	if err != nil {
		return err
	} else if len(to) > 255 {
		return fmt.Errorf("to address too long (%d bytes)",
			len(this.to))
	}

	return util.NewMonadOutputWriter(dest).
		SetOrder(binary.LittleEndian).
		WriteUint8(transaction_type_transfer).
		WriteUint8(uint8(len(to))).
		WriteBytes(this.from.Seed()).
		WriteBytes(to).
		WriteUint64(this.amount).
		WriteUint64(this.sequence).
		Error()
}

func (this *transferTransaction) getSigned() (virtualTransaction, *diemtypes.SignedTransaction, error) {
	var from *diemkeys.Keys
	var script diemtypes.Script

	from = diemkeys.NewKeysFromPublicAndPrivateKeys(
		diemkeys.NewEd25519PublicKey(this.from.Public().
			(ed25519.PublicKey)),
		diemkeys.NewEd25519PrivateKey(this.from))

	script = stdlib.EncodePeerToPeerWithMetadataScript(
		diemtypes.Currency(currency), this.to, this.amount, nil, nil)

	return newUnsignedTransaction(from, from.AccountAddress(),
		this.sequence, script, this.getName()).getSigned()
}

func (this *transferTransaction) getName() string {
	var seed []byte = this.from.Seed()

	return fmt.Sprintf("%02x%02x%02x%02x%02x%02x:%d", seed[0], seed[1],
		seed[2], seed[3], seed[4], seed[5], this.sequence)
}
