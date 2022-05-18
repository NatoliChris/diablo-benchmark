package nethereum

import (
	"context"
	"crypto/ecdsa"
	"diablo-benchmark/core"
	"diablo-benchmark/util"
	"encoding/binary"
	"fmt"
	"io"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

const (
	transaction_type_transfer uint8 = 0
	transaction_type_invoke   uint8 = 1

	transaction_gas_limit uint64 = 2000000
)

type transaction interface {
	getTx() (*types.Transaction, error)
}

type outerTransaction struct {
	inner virtualTransaction
}

func (this *outerTransaction) getTx() (*types.Transaction, error) {
	var ni virtualTransaction
	var tx *types.Transaction
	var err error

	ni, tx, err = this.inner.getTx()
	this.inner = ni

	if err != nil {
		return nil, err
	}

	return tx, nil
}

func decodeTransaction(src io.Reader, manager nonceManager, provider parameterProvider) (*outerTransaction, error) {
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

	switch txtype {
	case transaction_type_transfer:
		inner, err = decodeTransferTransaction(src, manager, provider)
	case transaction_type_invoke:
		inner, err = decodeInvokeTransaction(src, manager, provider)
	default:
		return nil, fmt.Errorf("unknown transaction type %d", txtype)
	}

	if err != nil {
		return nil, err
	}

	return &outerTransaction{inner}, nil
}

type virtualTransaction interface {
	getTx() (virtualTransaction, *types.Transaction, error)
}

type signedTransaction struct {
	tx *types.Transaction
}

func newSignedTransaction(tx *types.Transaction) *signedTransaction {
	return &signedTransaction{
		tx: tx,
	}
}

func (this *signedTransaction) getTx() (virtualTransaction, *types.Transaction, error) {
	return this, this.tx, nil
}

type unsignedTransaction struct {
	chain *big.Int
	tx    *types.Transaction
	from  *ecdsa.PrivateKey
}

func newUnsignedTransaction(chain *big.Int, tx *types.Transaction, from *ecdsa.PrivateKey) *unsignedTransaction {
	return &unsignedTransaction{
		chain: chain,
		tx:    tx,
		from:  from,
	}
}

func (this *unsignedTransaction) getTx() (virtualTransaction, *types.Transaction, error) {
	var stx *types.Transaction
	var err error

	stx, err = types.SignTx(this.tx, types.NewEIP155Signer(this.chain),
		this.from)
	if err != nil {
		return this, nil, err
	}

	return newSignedTransaction(stx).getTx()
}

type transferTransaction struct {
	nonce    uint64
	amount   uint64
	from     *ecdsa.PrivateKey
	to       common.Address
	manager  nonceManager
	provider parameterProvider
}

func newTransferTransaction(nonce, amount uint64, from *ecdsa.PrivateKey, to common.Address, manager nonceManager, provider parameterProvider) *transferTransaction {
	return &transferTransaction{
		nonce:    nonce,
		amount:   amount,
		from:     from,
		to:       to,
		manager:  manager,
		provider: provider,
	}
}

func decodeTransferTransaction(src io.Reader, manager nonceManager, provider parameterProvider) (*transferTransaction, error) {
	var from *ecdsa.PrivateKey
	var frombuf, tobuf []byte
	var nonce, amount uint64
	var lenkey int
	var err error

	err = util.NewMonadInputReader(src).
		SetOrder(binary.LittleEndian).
		ReadUint8(&lenkey).
		ReadUint64(&nonce).
		ReadUint64(&amount).
		ReadBytes(&frombuf, lenkey).
		ReadBytes(&tobuf, common.AddressLength).
		Error()

	if err != nil {
		return nil, err
	}

	from, err = crypto.ToECDSA(frombuf)
	if err != nil {
		return nil, err
	}

	return newTransferTransaction(nonce, amount, from,
		common.BytesToAddress(tobuf), manager, provider), nil
}

func (this *transferTransaction) encode(dest io.Writer) error {
	var from []byte

	from = crypto.FromECDSA(this.from)

	if len(from) > 255 {
		return fmt.Errorf("private key too long (%d bytes)", len(from))
	}

	return util.NewMonadOutputWriter(dest).
		SetOrder(binary.LittleEndian).
		WriteUint8(transaction_type_transfer).
		WriteUint8(uint8(len(from))).
		WriteUint64(this.nonce).
		WriteUint64(this.amount).
		WriteBytes(from).
		WriteBytes(this.to.Bytes()).
		Error()
}

func (this *transferTransaction) getTx() (virtualTransaction, *types.Transaction, error) {
	var tx *types.Transaction
	var from common.Address
	var params *parameters
	var nonce uint64
	var err error

	from = crypto.PubkeyToAddress(this.from.PublicKey)

	nonce, err = this.manager.nextNonce(from, this.nonce)
	if err != nil {
		return this, nil, err
	}

	params, err = this.provider.getParams()
	if err != nil {
		return this, nil, err
	}

	tx = types.NewTransaction(nonce, this.to,
		big.NewInt(int64(this.amount)), params.gasLimit,
		params.gasPrice, []byte{})

	return newUnsignedTransaction(params.chainId, tx,
		this.from).getTx()
}

type deployContractTransaction struct {
	nonce    uint64
	appli    *application
	from     *ecdsa.PrivateKey
	manager  nonceManager
	provider parameterProvider
}

func newDeployContractTransaction(nonce uint64, appli *application, from *ecdsa.PrivateKey, manager nonceManager, provider parameterProvider) *deployContractTransaction {
	return &deployContractTransaction{
		nonce:    nonce,
		appli:    appli,
		from:     from,
		manager:  manager,
		provider: provider,
	}
}

func (this *deployContractTransaction) getTx() (virtualTransaction, *types.Transaction, error) {
	var tx *types.Transaction
	var from common.Address
	var params *parameters
	var nonce uint64
	var err error

	from = crypto.PubkeyToAddress(this.from.PublicKey)

	nonce, err = this.manager.nextNonce(from, this.nonce)
	if err != nil {
		return this, nil, err
	}

	params, err = this.provider.getParams()
	if err != nil {
		return this, nil, err
	}

	tx = types.NewContractCreation(nonce, big.NewInt(0),
		params.gasLimit, params.gasPrice, this.appli.text)

	return newUnsignedTransaction(params.chainId, tx,
		this.from).getTx()
}

type invokeTransaction struct {
	nonce    uint64
	from     *ecdsa.PrivateKey
	appid    common.Address
	payload  []byte
	manager  nonceManager
	provider parameterProvider
}

func newInvokeTransaction(nonce uint64, from *ecdsa.PrivateKey, appid common.Address, payload []byte, manager nonceManager, provider parameterProvider) *invokeTransaction {
	return &invokeTransaction{
		nonce:    nonce,
		from:     from,
		appid:    appid,
		payload:  payload,
		manager:  manager,
		provider: provider,
	}
}

func decodeInvokeTransaction(src io.Reader, manager nonceManager, provider parameterProvider) (*invokeTransaction, error) {
	var frombuf, appidbuf, payload []byte
	var lenfrom, lenpayload int
	var from *ecdsa.PrivateKey
	var nonce uint64
	var err error

	err = util.NewMonadInputReader(src).
		SetOrder(binary.LittleEndian).
		ReadUint8(&lenfrom).
		ReadUint16(&lenpayload).
		ReadUint64(&nonce).
		ReadBytes(&frombuf, lenfrom).
		ReadBytes(&appidbuf, common.AddressLength).
		ReadBytes(&payload, lenpayload).
		Error()

	if err != nil {
		return nil, err
	}

	from, err = crypto.ToECDSA(frombuf)
	if err != nil {
		return nil, err
	}

	return newInvokeTransaction(nonce, from,
		common.BytesToAddress(appidbuf), payload, manager,
		provider), nil
}

func (this *invokeTransaction) encode(dest io.Writer) error {
	var from []byte = crypto.FromECDSA(this.from)

	if len(from) > 255 {
		return fmt.Errorf("private key too long (%d bytes)", len(from))
	}

	if len(this.payload) > 65535 {
		return fmt.Errorf("arguments too large (%d bytes)",
			len(this.payload))
	}

	return util.NewMonadOutputWriter(dest).
		SetOrder(binary.LittleEndian).
		WriteUint8(transaction_type_invoke).
		WriteUint8(uint8(len(from))).
		WriteUint16(uint16(len(this.payload))).
		WriteUint64(this.nonce).
		WriteBytes(from).
		WriteBytes(this.appid.Bytes()).
		WriteBytes(this.payload).
		Error()
}

func (this *invokeTransaction) getTx() (virtualTransaction, *types.Transaction, error) {
	var tx *types.Transaction
	var from common.Address
	var params *parameters
	var nonce uint64
	var err error

	from = crypto.PubkeyToAddress(this.from.PublicKey)

	nonce, err = this.manager.nextNonce(from, this.nonce)
	if err != nil {
		return this, nil, err
	}

	params, err = this.provider.getParams()
	if err != nil {
		return this, nil, err
	}

	tx = types.NewTransaction(nonce, this.appid, big.NewInt(int64(0)),
		params.gasLimit, params.gasPrice, this.payload)

	return newUnsignedTransaction(params.chainId, tx,
		this.from).getTx()
}

type nonceManager interface {
	nextNonce(common.Address, uint64) (uint64, error)
}

type staticNonceManager struct {
	logger core.Logger
	client *ethclient.Client
	ctx    context.Context
	lock   sync.Mutex
	bases  map[string]*staticNonce
}

type staticNonce struct {
	lock   sync.Mutex
	synced bool
	base   uint64
}

func newStaticNonceManager(logger core.Logger, client *ethclient.Client) *staticNonceManager {
	return &staticNonceManager{
		logger: logger,
		client: client,
		ctx:    context.Background(),
		bases:  make(map[string]*staticNonce),
	}
}

// Return the optimisticNonce locked.
//
func (this *staticNonceManager) getNonce(from common.Address) (*staticNonce, error) {
	var key string = from.String()
	var ret *staticNonce
	var err error
	var ok bool

	this.lock.Lock()

	ret, ok = this.bases[key]

	if !ok {
		ret = &staticNonce{
			synced: false,
		}
		this.bases[key] = ret
	}

	this.lock.Unlock()

	ret.lock.Lock()

	if ret.synced {
		err = nil
	} else {
		ret.base, err = this.client.PendingNonceAt(this.ctx, from)

		if err == nil {
			this.logger.Tracef("pending nonce for '%s' = %d", key,
				ret.base)
			ret.synced = true
		} else {
			this.logger.Errorf("fail to fetch pending nonce "+
				"for '%s': %s", key, err.Error())
		}
	}

	return ret, err
}

func (this *staticNonceManager) nextNonce(from common.Address, offset uint64) (uint64, error) {
	var slot *staticNonce
	var base uint64
	var err error

	slot, err = this.getNonce(from)
	defer slot.lock.Unlock()

	if err != nil {
		return 0, err
	}

	base = slot.base

	return base + offset, nil
}

type parameters struct {
	chainId  *big.Int
	gasLimit uint64
	gasPrice *big.Int
}

type parameterProvider interface {
	getParams() (*parameters, error)
}

type staticParameterProvider struct {
	params parameters
}

func newStaticParameterProvider(params *parameters) *staticParameterProvider {
	return &staticParameterProvider{
		params: *params,
	}
}

func makeStaticParameterProvider(client *ethclient.Client) (*staticParameterProvider, error) {
	params, err := newDirectParameterProvider(client).getParams()
	if err != nil {
		return nil, err
	}

	return newStaticParameterProvider(params), nil
}

func (this *staticParameterProvider) getParams() (*parameters, error) {
	return &this.params, nil
}

type lazyParameterProvider struct {
	client *ethclient.Client
	inner  *staticParameterProvider
}

func newLazyParameterProvider(client *ethclient.Client) *lazyParameterProvider {
	return &lazyParameterProvider{
		client: client,
		inner:  nil,
	}
}

func (this *lazyParameterProvider) getParams() (*parameters, error) {
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
	client *ethclient.Client
}

func newDirectParameterProvider(client *ethclient.Client) *directParameterProvider {
	return &directParameterProvider{
		client: client,
	}
}

func (this *directParameterProvider) getParams() (*parameters, error) {
	var ctx context.Context = context.Background()
	var params parameters
	var err error

	params.chainId, err = this.client.NetworkID(ctx)
	if err != nil {
		return nil, err
	}

	params.gasPrice, err = this.client.SuggestGasPrice(ctx)
	if err != nil {
		return nil, err
	}

	params.gasLimit = transaction_gas_limit

	return &params, nil
}
