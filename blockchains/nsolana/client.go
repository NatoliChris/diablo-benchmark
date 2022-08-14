package nsolana

import (
	"bytes"
	"context"
	"diablo-benchmark/core"
	"sync"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/gagliardetto/solana-go/rpc/ws"
)

type BlockchainClient struct {
	logger     core.Logger
	client     *rpc.Client
	ctx        context.Context
	commitment rpc.CommitmentType
	provider   parameterProvider
	preparer   transactionPreparer
	confirmer  transactionConfirmer
}

func newClient(logger core.Logger, client *rpc.Client, provider parameterProvider, preparer transactionPreparer, confirmer transactionConfirmer) *BlockchainClient {
	return &BlockchainClient{
		logger:     logger,
		client:     client,
		ctx:        context.Background(),
		commitment: rpc.CommitmentFinalized,
		provider:   provider,
		preparer:   preparer,
		confirmer:  confirmer,
	}
}

func (this *BlockchainClient) DecodePayload(encoded []byte) (interface{}, error) {
	buffer := bytes.NewBuffer(encoded)

	tx, err := decodeTransaction(buffer, this.provider)
	if err != nil {
		return nil, err
	}

	this.logger.Tracef("decode transaction %p", tx)

	err = this.preparer.prepare(tx)
	if err != nil {
		return nil, err
	}

	return tx, nil
}

func (this *BlockchainClient) TriggerInteraction(iact core.Interaction) error {
	tx := iact.Payload().(transaction)

	this.logger.Tracef("schedule transaction %p", tx)

	stx, err := tx.getTx()
	if err != nil {
		return err
	}

	this.logger.Tracef("submit transaction %p", tx)

	iact.ReportSubmit()

	_, err = this.client.SendTransactionWithOpts(
		this.ctx,
		stx,
		rpc.TransactionOpts{
			SkipPreflight:       false,
			PreflightCommitment: this.commitment,
		})
	if err != nil {
		iact.ReportAbort()
		return err
	}

	return this.confirmer.confirm(iact)
}

type transactionPreparer interface {
	prepare(transaction) error
}

type nothingTransactionPreparer struct {
}

func newNothingTransactionPreparer() transactionPreparer {
	return &nothingTransactionPreparer{}
}

func (this *nothingTransactionPreparer) prepare(transaction) error {
	return nil
}

type transactionConfirmer interface {
	confirm(core.Interaction) error
}

type pollblkTransactionConfirmer struct {
	logger   core.Logger
	client   *rpc.Client
	wsClient *ws.Client
	ctx      context.Context
	err      error
	lock     sync.Mutex
	pendings map[solana.Signature]*pollblkTransactionConfirmerPending
	observer parameterObsrever
}

type pollblkTransactionConfirmerPending struct {
	channel chan<- error
	iact    core.Interaction
}

func newPollblkTransactionConfirmer(logger core.Logger, client *rpc.Client, wsClient *ws.Client, ctx context.Context, observer parameterObsrever) *pollblkTransactionConfirmer {
	var this pollblkTransactionConfirmer

	this.logger = logger
	this.client = client
	this.wsClient = wsClient
	this.ctx = ctx
	this.err = nil
	this.pendings = make(map[solana.Signature]*pollblkTransactionConfirmerPending)
	this.observer = observer

	go this.run()

	return &this
}

func (this *pollblkTransactionConfirmer) confirm(iact core.Interaction) error {
	tx := iact.Payload().(transaction)

	stx, err := tx.getTx()
	if err != nil {
		return err
	}

	hash := &stx.Signatures[0]

	channel := make(chan error)

	pending := &pollblkTransactionConfirmerPending{
		channel: channel,
		iact:    iact,
	}

	this.lock.Lock()

	var done bool
	if this.pendings == nil {
		done = true
	} else {
		this.pendings[*hash] = pending
		done = false
	}

	this.lock.Unlock()

	if done {
		close(channel)
		return this.err
	} else {
		return <-channel
	}
}

func (this *pollblkTransactionConfirmer) reportHashes(hashes []solana.Signature) {
	pendings := make([]*pollblkTransactionConfirmerPending, 0, len(hashes))

	this.lock.Lock()

	for _, hash := range hashes {
		pending, ok := this.pendings[hash]
		if !ok {
			continue
		}

		delete(this.pendings, hash)

		pendings = append(pendings, pending)
	}

	this.lock.Unlock()

	for _, pending := range pendings {
		this.logger.Tracef("commit transaction %p",
			pending.iact.Payload())
		pending.iact.ReportCommit()
		pending.channel <- nil
		close(pending.channel)
	}
}

func (this *pollblkTransactionConfirmer) flushPendings(err error) {
	pendings := make([]*pollblkTransactionConfirmerPending, 0)

	this.lock.Lock()

	for _, pending := range this.pendings {
		pendings = append(pendings, pending)
	}

	this.pendings = nil
	this.err = err

	this.lock.Unlock()

	for _, pending := range pendings {
		this.logger.Tracef("abort transaction %p",
			pending.iact.Payload())
		pending.iact.ReportAbort()
		pending.channel <- err
		close(pending.channel)
	}
}

func (this *pollblkTransactionConfirmer) processBlock(number uint64) error {
	var block *rpc.GetBlockResult
	var err error

	this.logger.Tracef("poll new block (number = %d)", number)

	includeRewards := false
	for attempt := 0; attempt < 100; attempt++ {
		block, err = this.client.GetBlockWithOpts(
			this.ctx,
			number,
			&rpc.GetBlockOpts{
				TransactionDetails: rpc.TransactionDetailsSignatures,
				Rewards:            &includeRewards,
				Commitment:         rpc.CommitmentFinalized,
			})

		if err != nil || block == nil {
			time.Sleep(50 * time.Millisecond)
			continue
		}
		break
	}
	if err != nil {
		return err
	}

	this.observer.updateParameters(&parameters{blockhash: &block.Blockhash})

	if len(block.Signatures) == 0 {
		return nil
	}

	this.reportHashes(block.Signatures)

	return nil
}

func (this *pollblkTransactionConfirmer) run() {
	params, err := newDirectParameterProvider(this.client, this.ctx).getParams()
	if err != nil {
		this.flushPendings(err)
		return
	}
	this.observer.updateParameters(params)

	subcription, err := this.wsClient.RootSubscribe()
	if err != nil {
		this.flushPendings(err)
		return
	}

	this.logger.Tracef("subscribe to new head events")

	var currentNumber ws.RootResult = 0
loop:
	for {
		event, err := subcription.Recv()
		if err != nil {
			break loop
		}
		if event == nil {
			continue
		}
		if *event <= currentNumber {
			continue
		} else if *event > currentNumber+1 {
			for currentNumber+1 < *event {
				currentNumber++
				err = this.processBlock(uint64(currentNumber))
			}
		}
		currentNumber = *event
		err = this.processBlock(uint64(currentNumber))
		if err != nil {
			break loop
		}
	}

	subcription.Unsubscribe()

	this.flushPendings(err)
}
