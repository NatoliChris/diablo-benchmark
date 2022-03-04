package nalgorand


import (
	"context"
	"diablo-benchmark/util"
	"encoding/binary"
	"fmt"
	"io"

	"github.com/algorand/go-algorand-sdk/client/v2/algod"
	"github.com/algorand/go-algorand-sdk/crypto"
	"github.com/algorand/go-algorand-sdk/future"
	algotx "github.com/algorand/go-algorand-sdk/transaction"
	"github.com/algorand/go-algorand-sdk/types"

	"golang.org/x/crypto/ed25519"
)


const (
	transaction_type_transfer uint8 = 0
	transaction_type_invoke   uint8 = 1
)


type transaction interface {
	getRaw() ([]byte, error)
	getUid() uint64
}


func uidToNote(uid uint64) []byte {
	var ret []byte = make([]byte, 8)

	binary.LittleEndian.PutUint64(ret, uid)

	return ret
}

func noteToUid(note []byte) (uint64, bool) {
	if len(note) != 8 {
		return 0, false
	}

	return binary.LittleEndian.Uint64(note), true
}


type outerTransaction struct {
	inner  virtualTransaction
}

func (this *outerTransaction) getRaw() ([]byte, error) {
	var ni virtualTransaction
	var raw []byte
	var err error

	ni, raw, err = this.inner.getRaw()
	this.inner = ni

	if err != nil {
		return nil, err
	}

	return raw, nil
}

func (this *outerTransaction) getUid() uint64 {
	return this.inner.getUid()
}

func decodeTransaction(src io.Reader, provider parameterProvider) (transaction, error) {
	var inner virtualTransaction
	var txtype uint8
	var err error

	err = util.NewMonadInputReader(src).ReadUint8(&txtype).Error()
	if err != nil {
		return nil, err
	}

	switch (txtype) {
	case transaction_type_transfer:
		inner, err = decodeTransferTransaction(src, provider)
	case transaction_type_invoke:
		inner, err = decodeInvokeTransaction(src, provider)
	default:
		return nil, fmt.Errorf("unknown transaction type %v", txtype)
	}

	if err != nil {
		return nil, err
	}

	return &outerTransaction{ inner }, nil
}


type virtualTransaction interface {
	getRaw() (virtualTransaction, []byte, error)

	getUid() uint64
}


type baseTransaction struct {
	uid  uint64
}

func (this *baseTransaction) init(uid uint64) {
	this.uid = uid
}

func (this *baseTransaction) getUid() uint64 {
	return this.uid
}


type signedTransaction struct {
	baseTransaction
	raw  []byte
}

func newSignedTransaction(uid uint64, raw []byte) *signedTransaction {
	var this signedTransaction

	this.baseTransaction.init(uid)
	this.raw = raw

	return &this
}

func (this *signedTransaction) getRaw() (virtualTransaction, []byte, error) {
	return this, this.raw, nil
}


type unsignedTransaction struct {
	baseTransaction
	tx   types.Transaction
	key  ed25519.PrivateKey
}

func newUnsignedTransaction(uid uint64, tx types.Transaction, key ed25519.PrivateKey) *unsignedTransaction {
	var this unsignedTransaction

	this.baseTransaction.init(uid)
	this.tx = tx
	this.key = key

	return &this
}

func (this *unsignedTransaction) getRaw() (virtualTransaction, []byte, error) {
	var raw []byte
	var err error

	_, raw, err = crypto.SignTransaction(this.key, this.tx)
	if err != nil {
		return this, nil, err
	}

	return newSignedTransaction(this.getUid(), raw), raw, nil
}


type transferTransaction struct {
	baseTransaction
	amount    uint64
	from      string
	to        string
	key       ed25519.PrivateKey
	provider  parameterProvider
}

func newTransferTransaction(uid, amount uint64, from, to string, key ed25519.PrivateKey, provider parameterProvider) *transferTransaction {
	var this transferTransaction

	this.baseTransaction.init(uid)
	this.amount = amount
	this.from = from
	this.to = to
	this.key = key
	this.provider = provider

	return &this
}

func decodeTransferTransaction(src io.Reader, provider parameterProvider) (*transferTransaction, error) {
	var key ed25519.PrivateKey
	var lenfrom, lento int
	var uid, amount uint64
	var from, to string
	var buf []byte
	var err error

	err = util.NewMonadInputReader(src).
		SetOrder(binary.LittleEndian).
		ReadUint8(&lenfrom).
		ReadUint8(&lento).
		ReadUint64(&uid).
		ReadUint64(&amount).
		ReadString(&from, lenfrom).
		ReadString(&to, lento).
		ReadBytes(&buf, ed25519.SeedSize).
		Error()
	if err != nil {
		return nil, err
	}

	key = ed25519.NewKeyFromSeed(buf)

	return newTransferTransaction(uid, amount, from, to, key, provider),nil
}

func (this *transferTransaction) encode(dest io.Writer) error {
	if len(this.from) > 255 {
		return fmt.Errorf("from address too long (%d bytes)",
			len(this.from))
	}

	if len(this.to) > 255 {
		return fmt.Errorf("to address too long (%d bytes)",
			len(this.to))
	}

	return util.NewMonadOutputWriter(dest).
		SetOrder(binary.LittleEndian).
		WriteUint8(transaction_type_transfer).
		WriteUint8(uint8(len(this.from))).
		WriteUint8(uint8(len(this.to))).
		WriteUint64(this.getUid()).
		WriteUint64(this.amount).
		WriteString(this.from).
		WriteString(this.to).
		WriteBytes(this.key.Seed()).
		Error()
}

func (this *transferTransaction) getRaw() (virtualTransaction, []byte, error) {
	var params *types.SuggestedParams
	var tx types.Transaction
	var err error

	params, err = this.provider.getParams()
	if err != nil {
		return this, nil, err
	}

	tx, err = algotx.MakePaymentTxnWithFlatFee(this.from, this.to, 0,
		this.amount, uint64(params.FirstRoundValid),
		uint64(params.LastRoundValid), uidToNote(this.getUid()), "",
		params.GenesisID, params.GenesisHash)
	if err != nil {
		return this, nil, err
	}

	return newUnsignedTransaction(this.getUid(), tx, this.key).getRaw()
}


type deployContractTransaction struct {
	baseTransaction
	appli     *application
	from      string
	key       ed25519.PrivateKey
	provider  parameterProvider
}

func newDeployContractTransaction(uid uint64, appli *application, from string, key ed25519.PrivateKey, provider parameterProvider) *deployContractTransaction {
	var this deployContractTransaction

	this.baseTransaction.init(uid)
	this.appli = appli
	this.from = from
	this.key = key
	this.provider = provider

	return &this
}

func (this *deployContractTransaction) getRaw() (virtualTransaction, []byte, error) {
	var params *types.SuggestedParams
	var tx types.Transaction
	var addr types.Address
	var err error

	params, err = this.provider.getParams()
	if err != nil {
		return this, nil, err
	}

	addr, err = types.DecodeAddress(this.from)
	if err != nil {
		return this, nil, err
	}

	tx, err = future.MakeApplicationCreateTx(false,
		this.appli.approvalCode, this.appli.clearCode,
		this.appli.globalSchema, this.appli.localSchema, nil, nil,
		nil, nil, *params, addr, uidToNote(this.getUid()),
		types.Digest{}, [32]byte{}, types.Address{})
	if err != nil {
		return this, nil, err
	}

	return newUnsignedTransaction(this.getUid(), tx, this.key).getRaw()
}


type invokeTransaction struct {
	baseTransaction
	appid     uint64
	args      [][]byte
	from      string
	key       ed25519.PrivateKey
	provider  parameterProvider
}

func newInvokeTransaction(uid, appid uint64, args [][]byte, from string, key ed25519.PrivateKey, provider parameterProvider) *invokeTransaction {
	var this invokeTransaction

	this.baseTransaction.init(uid)
	this.appid = appid
	this.args = args
	this.from = from
	this.key = key
	this.provider = provider

	return &this
}

func decodeInvokeTransaction(src io.Reader, provider parameterProvider) (*invokeTransaction, error) {
	var i, lenfrom, lenargs, lenarg int
	var key ed25519.PrivateKey
	var input util.MonadInput
	var uid, appid uint64
	var args [][]byte
	var from string
	var buf []byte
	var err error

	input = util.NewMonadInputReader(src).
		SetOrder(binary.LittleEndian).
		ReadUint8(&lenfrom).
		ReadUint8(&lenargs).
		ReadUint64(&uid).
		ReadUint64(&appid).
		ReadString(&from, lenfrom).
		ReadBytes(&buf, ed25519.SeedSize)

	args = make([][]byte, lenargs)

	for i = range args {
		input.ReadUint8(&lenarg).ReadBytes(&args[i], lenarg)
	}

	err = input.Error()
	if err != nil {
		return nil, err
	}

	key = ed25519.NewKeyFromSeed(buf)

	return newInvokeTransaction(uid, appid, args, from, key, provider), nil
}

func (this *invokeTransaction) encode(dest io.Writer) error {
	var output util.MonadOutput
	var i int

	if len(this.args) > 255 {
		return fmt.Errorf("too many invoke arguments (%d)",
			len(this.args))
	}

	for i = range this.args {
		if len(this.args[i]) <= 255 {
			continue
		}

		return fmt.Errorf("invoke argument %d too large (%d bytes)",
			i, len(this.args[i]))
	}

	output = util.NewMonadOutputWriter(dest).
		SetOrder(binary.LittleEndian).
		WriteUint8(transaction_type_invoke).
		WriteUint8(uint8(len(this.from))).
		WriteUint8(uint8(len(this.args))).
		WriteUint64(this.getUid()).
		WriteUint64(this.appid).
		WriteString(this.from).
		WriteBytes(this.key.Seed())
		
	for i = range this.args {
		output.WriteUint8(uint8(len(this.args[i]))).
			WriteBytes(this.args[i])
	}

	return output.Error()
}

func (this *invokeTransaction) getRaw() (virtualTransaction, []byte, error) {
	var params *types.SuggestedParams
	var tx types.Transaction
	var addr types.Address
	var err error

	params, err = this.provider.getParams()
	if err != nil {
		return this, nil, err
	}

	addr, err = types.DecodeAddress(this.from)
	if err != nil {
		return this, nil, err
	}

	tx, err = future.MakeApplicationNoOpTx(this.appid, this.args, nil, nil,
		nil, *params, addr, uidToNote(this.getUid()), types.Digest{},
		[32]byte{}, types.Address{})
	if err != nil {
		return this, nil, err
	}

	return newUnsignedTransaction(this.getUid(), tx, this.key).getRaw()
}


type parameterProvider interface {
	getParams() (*types.SuggestedParams, error)
}


type staticParameterProvider struct {
	params  types.SuggestedParams
}

func newStaticParameterProvider(params *types.SuggestedParams) *staticParameterProvider {
	return &staticParameterProvider{
		params: *params,
	}
}

func makeStaticParameterProvider(client *algod.Client) (*staticParameterProvider, error) {
	var this staticParameterProvider
	var err error

	this.params, err = client.SuggestedParams().Do(context.Background())
	if err != nil {
		return nil, err
	}

	this.params.LastRoundValid = this.params.FirstRoundValid + 1000

	return &this, nil
}

func (this *staticParameterProvider) getParams() (*types.SuggestedParams, error) {
	return &this.params, nil
}


type lazyParameterProvider struct {
	client  *algod.Client
	inner   *staticParameterProvider
}

func newLazyParameterProvider(client *algod.Client) *lazyParameterProvider {
	return &lazyParameterProvider{
		client: client,
		inner: nil,
	}
}

func (this *lazyParameterProvider) getParams() (*types.SuggestedParams, error) {
	var err error

	if this.inner == nil {
		this.inner, err = makeStaticParameterProvider(this.client)
		if err != nil {
			return nil, err
		}
	}

	return this.inner.getParams()
}


type directParameterProvider struct {
	client  *algod.Client
}

func newDirectParameterProvider(client *algod.Client) *directParameterProvider {
	return &directParameterProvider{
		client: client,
	}
}

func (this *directParameterProvider) getParams() (*types.SuggestedParams, error) {
	var ret types.SuggestedParams
	var err error

	ret, err = this.client.SuggestedParams().Do(context.Background())

	return &ret, err
}
