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
	transaction_type_invoke   uint8 = 1

	currency = "XUS"

	maximumGasAmount = 4_000_000

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
	case transaction_type_invoke:
		inner, err = decodeInvokeTransaction(src)
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
	payload         diemtypes.TransactionPayload
	maxGasAmount    uint64
	gasUnitPrice    uint64
	expirationTime  uint64
	chainId         byte
	name            string
}

func newUnsignedTransaction(from *diemkeys.Keys, to diemtypes.AccountAddress, sequence uint64, payload diemtypes.TransactionPayload, name string) *unsignedTransaction {
	return &unsignedTransaction{
		from: from,
		to: to,
		sequence: sequence,
		payload: payload,
		name: name,
	}
}

func (this *unsignedTransaction) getSigned() (virtualTransaction, *diemtypes.SignedTransaction, error) {
	var stx *diemtypes.SignedTransaction
	var expiration uint64

	expiration = uint64(time.Now().Add(expirationDelay).Unix())

	stx = diemsigner.SignTxn(this.from, this.to, this.sequence,
		this.payload, maximumGasAmount, 0, currency, expiration,
		chainId)

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
		this.sequence, &diemtypes.TransactionPayload__Script{script},
		this.getName()).getSigned()
}

func (this *transferTransaction) getName() string {
	var seed []byte = this.from.Seed()

	return fmt.Sprintf("%02x%02x%02x%02x%02x%02x:%d", seed[0], seed[1],
		seed[2], seed[3], seed[4], seed[5], this.sequence)
}


type deployContractTransaction struct {
	from      ed25519.PrivateKey
	code      []byte
	sequence  uint64
}

func newDeployContractTransaction(from ed25519.PrivateKey, code []byte, sequence uint64) *deployContractTransaction {
	return &deployContractTransaction{
		from: from,
		code: code,
		sequence: sequence,
	}
}

func (this *deployContractTransaction) getSigned() (virtualTransaction, *diemtypes.SignedTransaction, error) {
	var from *diemkeys.Keys
	var module diemtypes.Module

	from = diemkeys.NewKeysFromPublicAndPrivateKeys(
		diemkeys.NewEd25519PublicKey(this.from.Public().
			(ed25519.PublicKey)),
		diemkeys.NewEd25519PrivateKey(this.from))

	module = diemtypes.Module{ this.code }

	return newUnsignedTransaction(from, from.AccountAddress(),
		this.sequence, &diemtypes.TransactionPayload__Module{module},
		this.getName()).getSigned()
}

func (this *deployContractTransaction) getName() string {
	var seed []byte = this.from.Seed()

	return fmt.Sprintf("%02x%02x%02x%02x%02x%02x:%d", seed[0], seed[1],
		seed[2], seed[3], seed[4], seed[5], this.sequence)
}


type invokeTransaction struct {
	from      ed25519.PrivateKey
	code      []byte
	args      []diemtypes.TransactionArgument
	sequence  uint64
}

func newInvokeTransaction(from ed25519.PrivateKey, code []byte, args []diemtypes.TransactionArgument, sequence uint64) *invokeTransaction {
	return &invokeTransaction{
		from: from,
		code: code,
		args: args,
		sequence: sequence,
	}
}

func decodeInvokeTransaction(src io.Reader) (*invokeTransaction, error) {
	var args []diemtypes.TransactionArgument
	var input util.MonadInput
	var fromSeed, code []byte
	var sequence uint64
	var bargs [][]byte
	var err error
	var i, n int

	input = util.NewMonadInputReader(src).
		SetOrder(binary.LittleEndian).
		ReadBytes(&fromSeed, ed25519.SeedSize).
		ReadUint64(&sequence).
		ReadUint16(&n).
		ReadBytes(&code, n).
		ReadUint8(&n)

	bargs = make([][]byte, n)

	for i = range bargs {
		input.ReadUint16(&n).ReadBytes(&bargs[i], n)
	}

	err = input.Error()
	if err != nil {
		return nil, err
	}

	args = make([]diemtypes.TransactionArgument, len(bargs))

	for i = range args {
		args[i], err =
			diemtypes.BcsDeserializeTransactionArgument(bargs[i])
		if err != nil {
			return nil, err
		}
	}

	return newInvokeTransaction(ed25519.NewKeyFromSeed(fromSeed), code,
		args, sequence), nil
}

func (this *invokeTransaction) encode(dest io.Writer) error {
	var arg diemtypes.TransactionArgument
	var output util.MonadOutput
	var bargs [][]byte
	var err error
	var i int

	if len(this.code) > 65535 {
		return fmt.Errorf("code too large (%d bytes)", len(this.code))
	}

	if len(this.args) > 255 {
		return fmt.Errorf("too many arguments (%d)", len(this.args))
	}

	bargs = make([][]byte, len(this.args))

	for i, arg = range this.args {
		bargs[i], err = arg.BcsSerialize()
		if err != nil {
			return err
		} else if len(bargs[i]) > 65535 {
			return fmt.Errorf("invoke arguments %d is too long " +
				"(%d bytes)", len(bargs[i]))
		}
	}

	output = util.NewMonadOutputWriter(dest).
		SetOrder(binary.LittleEndian).
		WriteUint8(transaction_type_invoke).
		WriteBytes(this.from.Seed()).
		WriteUint64(this.sequence).
		WriteUint16(uint16(len(this.code))).
		WriteBytes(this.code).
		WriteUint8(uint8(len(bargs)))

	for i = range bargs {
		output.WriteUint16(uint16(len(bargs[i]))).WriteBytes(bargs[i])
	}

	return output.Error()
}

func (this *invokeTransaction) getSigned() (virtualTransaction, *diemtypes.SignedTransaction, error) {
	var script diemtypes.Script
	var from *diemkeys.Keys

	from = diemkeys.NewKeysFromPublicAndPrivateKeys(
		diemkeys.NewEd25519PublicKey(this.from.Public().
			(ed25519.PublicKey)),
		diemkeys.NewEd25519PrivateKey(this.from))

	script = diemtypes.Script{
		Code: this.code,
		TyArgs: []diemtypes.TypeTag{},
		Args: this.args,
	}

	return newUnsignedTransaction(from, from.AccountAddress(),
		this.sequence, &diemtypes.TransactionPayload__Script{script},
		this.getName()).getSigned()
}

func (this *invokeTransaction) getName() string {
	var seed []byte = this.from.Seed()

	return fmt.Sprintf("%02x%02x%02x%02x%02x%02x:%d", seed[0], seed[1],
		seed[2], seed[3], seed[4], seed[5], this.sequence)
}
