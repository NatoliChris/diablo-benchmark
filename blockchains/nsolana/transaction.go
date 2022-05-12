package nsolana

import (
	"context"
	"diablo-benchmark/util"
	"encoding/binary"
	"fmt"
	"io"
	"sync"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/gagliardetto/solana-go"
	bpfloader "github.com/gagliardetto/solana-go/programs/bpf-loader"
	"github.com/gagliardetto/solana-go/programs/system"
	"github.com/gagliardetto/solana-go/rpc"
)

const (
	transaction_type_transfer uint8 = 0
	transaction_type_invoke   uint8 = 1

	default_ms_per_slot = 400
)

type transaction interface {
	getTx() (*solana.Transaction, error)
}

type outerTransaction struct {
	inner virtualTransaction
}

func (this *outerTransaction) getTx() (*solana.Transaction, error) {
	ni, tx, err := this.inner.getTx()
	this.inner = ni

	if err != nil {
		return nil, err
	}

	return tx, nil
}

func decodeTransaction(src io.Reader, provider parameterProvider) (*outerTransaction, error) {
	var txtype uint8
	err := util.NewMonadInputReader(src).
		SetOrder(binary.LittleEndian).
		ReadUint8(&txtype).
		Error()
	if err != nil {
		return nil, err
	}

	var inner virtualTransaction
	switch txtype {
	case transaction_type_transfer:
		inner, err = decodeTransferTransaction(src, provider)
	case transaction_type_invoke:
		inner, err = decodeInvokeTransaction(src, provider)
	default:
		return nil, fmt.Errorf("unknown transaction type %d", txtype)
	}

	if err != nil {
		return nil, err
	}

	return &outerTransaction{inner}, nil
}

type virtualTransaction interface {
	getTx() (virtualTransaction, *solana.Transaction, error)
}

type signedTransaction struct {
	tx *solana.Transaction
}

func newSignedTransaction(tx *solana.Transaction) *signedTransaction {
	return &signedTransaction{
		tx: tx,
	}
}

func (this *signedTransaction) getTx() (virtualTransaction, *solana.Transaction, error) {
	return this, this.tx, nil
}

type unsignedTransaction struct {
	tx      *solana.Transaction
	signers map[solana.PublicKey]*solana.PrivateKey
}

func newUnsignedTransaction(tx *solana.Transaction, signers []solana.PrivateKey) *unsignedTransaction {
	signersMap := make(map[solana.PublicKey]*solana.PrivateKey)
	for _, signer := range signers {
		signersMap[signer.PublicKey()] = &signer
	}
	return &unsignedTransaction{
		tx:      tx,
		signers: signersMap,
	}
}

func (this *unsignedTransaction) getTx() (virtualTransaction, *solana.Transaction, error) {
	stx := *this.tx
	_, err := stx.Sign(func(key solana.PublicKey) *solana.PrivateKey {
		return this.signers[key]
	})
	if err != nil {
		return this, nil, err
	}

	return newSignedTransaction(&stx).getTx()
}

type parameterlessTransaction struct {
	txBuilder *solana.TransactionBuilder
	signers   []solana.PrivateKey
	provider  parameterProvider
}

func newParameterlessTransaction(txBuilder *solana.TransactionBuilder, signers []solana.PrivateKey, provider parameterProvider) *parameterlessTransaction {
	return &parameterlessTransaction{
		txBuilder: txBuilder,
		signers:   signers,
		provider:  provider,
	}
}

func (this *parameterlessTransaction) getTx() (virtualTransaction, *solana.Transaction, error) {
	builder := *this.txBuilder

	params, err := this.provider.getParams()
	if err != nil {
		return this, nil, err
	}

	builder.SetRecentBlockHash(*params.blockhash)
	utx, err := builder.Build()
	if err != nil {
		return this, nil, err
	}

	return newUnsignedTransaction(utx, this.signers).getTx()
}

type transferTransaction struct {
	amount   uint64
	from     solana.PrivateKey
	to       *solana.PublicKey
	provider parameterProvider
}

func newTransferTransaction(amount uint64, from solana.PrivateKey, to *solana.PublicKey, provider parameterProvider) *transferTransaction {
	return &transferTransaction{
		amount:   amount,
		from:     from,
		to:       to,
		provider: provider,
	}
}

func decodeTransferTransaction(src io.Reader, provider parameterProvider) (*transferTransaction, error) {
	var frombuf, tobuf []byte
	var amount uint64
	var lenkey int

	err := util.NewMonadInputReader(src).
		SetOrder(binary.LittleEndian).
		ReadUint8(&lenkey).
		ReadUint64(&amount).
		ReadBytes(&frombuf, lenkey).
		ReadBytes(&tobuf, solana.PublicKeyLength).
		Error()

	if err != nil {
		return nil, err
	}

	to := solana.PublicKeyFromBytes(tobuf)

	return newTransferTransaction(amount, frombuf, &to, provider), nil
}

func (this *transferTransaction) encode(dest io.Writer) error {
	if len(this.from) > 255 {
		return fmt.Errorf("private key too long (%d bytes)", len(this.from))
	}

	return util.NewMonadOutputWriter(dest).
		SetOrder(binary.LittleEndian).
		WriteUint8(transaction_type_transfer).
		WriteUint8(uint8(len(this.from))).
		WriteUint64(this.amount).
		WriteBytes(this.from).
		WriteBytes(this.to.Bytes()).
		Error()
}

func (this *transferTransaction) getTx() (virtualTransaction, *solana.Transaction, error) {
	from := this.from.PublicKey()

	params, err := this.provider.getParams()
	if err != nil {
		return this, nil, err
	}

	instruction := system.NewTransferInstruction(
		this.amount,
		from,
		*this.to).Build()

	tx, err := solana.NewTransaction(
		[]solana.Instruction{instruction},
		*params.blockhash,
		solana.TransactionPayer(from))
	if err != nil {
		return this, nil, err
	}

	return newUnsignedTransaction(tx, []solana.PrivateKey{this.from}).getTx()
}

func newDeployContractTransactionBatches(appli *application, from, program, storage *account, programLamports, storageLamports uint64, provider parameterProvider) ([][]virtualTransaction, error) {
	// 1 - create program account
	// 2 - call loader writes
	// 3 - call loader finalize
	// 4 - create storage account and call contract constructor
	transactionBatches := make([][]virtualTransaction, 4)

	initialBuilder, writeBuilders, finalBuilder, _, err := bpfloader.Deploy(
		from.public, nil, appli.text, programLamports, solana.BPFLoaderProgramID, program.public, false)
	if err != nil {
		return nil, err
	}

	transactionBatches[0] = append(transactionBatches[0],
		newParameterlessTransaction(initialBuilder, []solana.PrivateKey{from.private}, provider))
	for _, builder := range writeBuilders {
		transactionBatches[1] = append(transactionBatches[1],
			newParameterlessTransaction(builder, []solana.PrivateKey{from.private}, provider))
	}
	transactionBatches[2] = append(transactionBatches[2],
		newParameterlessTransaction(finalBuilder, []solana.PrivateKey{from.private}, provider))

	// assuming that constructor does not have arguments
	{
		input, err := appli.abi.Constructor.Inputs.Pack()
		if err != nil {
			return nil, err
		}

		hash := crypto.Keccak256([]byte(appli.name))

		value := make([]byte, 8)
		binary.LittleEndian.PutUint64(value[0:], 0)

		data := []byte{}
		data = append(data, storage.public.Bytes()...)
		data = append(data, from.public.Bytes()...)
		data = append(data, value...)
		data = append(data, hash[:4]...)
		data = append(data, encodeSeeds()...)
		data = append(data, input...)

		builder := solana.NewTransactionBuilder().AddInstruction(
			system.NewCreateAccountInstruction(
				storageLamports,
				8192*8,
				program.public,
				from.public,
				storage.public).Build(),
		).AddInstruction(
			solana.NewInstruction(
				program.public,
				[]*solana.AccountMeta{
					solana.NewAccountMeta(
						storage.public,
						true,
						false),
				}, data),
		)
		transactionBatches[3] = append(transactionBatches[3],
			newParameterlessTransaction(builder, []solana.PrivateKey{from.private}, provider))
	}

	return transactionBatches, nil
}

type invokeTransaction struct {
	amount           uint64
	from             solana.PrivateKey
	program, storage *solana.PublicKey
	payload          []byte
	provider         parameterProvider
}

func newInvokeTransaction(amount uint64, from solana.PrivateKey, program, storage *solana.PublicKey, payload []byte, provider parameterProvider) *invokeTransaction {
	return &invokeTransaction{
		amount:   amount,
		from:     from,
		program:  program,
		storage:  storage,
		payload:  payload,
		provider: provider,
	}
}

func decodeInvokeTransaction(src io.Reader, provider parameterProvider) (*invokeTransaction, error) {
	var frombuf, programbuf, storagebuf, payload []byte
	var amount uint64
	var lenfrom, lenpayload int

	err := util.NewMonadInputReader(src).
		SetOrder(binary.LittleEndian).
		ReadUint8(&lenfrom).
		ReadUint16(&lenpayload).
		ReadUint64(&amount).
		ReadBytes(&frombuf, lenfrom).
		ReadBytes(&programbuf, solana.PublicKeyLength).
		ReadBytes(&storagebuf, solana.PublicKeyLength).
		ReadBytes(&payload, lenpayload).
		Error()

	if err != nil {
		return nil, err
	}

	program := solana.PublicKeyFromBytes(programbuf)
	storage := solana.PublicKeyFromBytes(storagebuf)

	return newInvokeTransaction(amount, frombuf, &program, &storage, payload, provider), nil
}

func (this *invokeTransaction) encode(dest io.Writer) error {
	if len(this.from) > 255 {
		return fmt.Errorf("private key too long (%d bytes)", len(this.from))
	}

	if len(this.payload) > 65535 {
		return fmt.Errorf("arguments too large (%d bytes)",
			len(this.payload))
	}

	return util.NewMonadOutputWriter(dest).
		SetOrder(binary.LittleEndian).
		WriteUint8(transaction_type_invoke).
		WriteUint8(uint8(len(this.from))).
		WriteUint16(uint16(len(this.payload))).
		WriteUint64(this.amount).
		WriteBytes(this.from).
		WriteBytes(this.program.Bytes()).
		WriteBytes(this.storage.Bytes()).
		WriteBytes(this.payload).
		Error()
}

type solangSeed struct {
	seed []byte
	// address solana.PublicKey
}

func encodeSeeds(seeds ...solangSeed) []byte {
	var length uint64 = 1
	for _, seed := range seeds {
		length += uint64(len(seed.seed)) + 1
	}
	seedEncoded := make([]byte, 0, length)

	seedEncoded = append(seedEncoded, uint8(len(seeds)))
	for _, seed := range seeds {
		seedEncoded = append(seedEncoded, uint8(len(seed.seed)))
		seedEncoded = append(seedEncoded, seed.seed...)
	}

	return seedEncoded
}

func (this *invokeTransaction) getTx() (virtualTransaction, *solana.Transaction, error) {
	from := this.from.PublicKey()

	params, err := this.provider.getParams()
	if err != nil {
		return this, nil, err
	}

	valueBytes := make([]byte, 8)
	binary.LittleEndian.PutUint64(valueBytes[0:], this.amount)

	instructionData := []byte{}
	instructionData = append(instructionData, this.storage.Bytes()...)
	instructionData = append(instructionData, from.Bytes()...)
	instructionData = append(instructionData, valueBytes...)
	instructionData = append(instructionData, make([]byte, 4)...)
	instructionData = append(instructionData, encodeSeeds()...)
	instructionData = append(instructionData, this.payload...)

	instruction := solana.NewInstruction(
		*this.program,
		[]*solana.AccountMeta{
			solana.NewAccountMeta(*this.storage, true, false),
			solana.NewAccountMeta(solana.SysVarClockPubkey, false, false),
			solana.NewAccountMeta(solana.PublicKey{}, false, false),
		}, instructionData)

	tx, err := solana.NewTransaction(
		[]solana.Instruction{instruction},
		*params.blockhash,
		solana.TransactionPayer(from))
	if err != nil {
		return this, nil, err
	}

	return newUnsignedTransaction(tx, []solana.PrivateKey{this.from}).getTx()
}

type parameters struct {
	blockhash *solana.Hash
}

type parameterProvider interface {
	getParams() (*parameters, error)
}

type parameterObsrever interface {
	updateParameters(parameters *parameters)
}

type observerParameterProvider struct {
	lock       sync.RWMutex
	parameters *parameters
}

func newObserverParameterProvider() *observerParameterProvider {
	return &observerParameterProvider{}
}

func (this *observerParameterProvider) updateParameters(parameters *parameters) {
	this.lock.Lock()
	defer this.lock.Unlock()
	this.parameters = parameters
}

func (this *observerParameterProvider) getParams() (*parameters, error) {
	this.lock.RLock()
	defer this.lock.RUnlock()
	return this.parameters, nil
}

type directParameterProvider struct {
	client *rpc.Client
	ctx    context.Context
}

func newDirectParameterProvider(client *rpc.Client, ctx context.Context) *directParameterProvider {
	return &directParameterProvider{
		client: client,
		ctx:    ctx,
	}
}

func (this *directParameterProvider) getParams() (*parameters, error) {
	var params parameters

	blockhash, err := this.client.GetRecentBlockhash(this.ctx, rpc.CommitmentFinalized)
	if err != nil {
		return nil, err
	}

	params.blockhash = &blockhash.Value.Blockhash

	return &params, nil
}
