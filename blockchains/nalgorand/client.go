package nalgorand


import (
	"bytes"
	"context"
	"diablo-benchmark/core"
	"sync"

	"github.com/algorand/go-algorand-sdk/client/v2/algod"
	"github.com/algorand/go-algorand-sdk/client/v2/common/models"
	"github.com/algorand/go-algorand-sdk/types"
)


type BlockchainClient struct {
	logger     core.Logger
	client     *algod.Client
	provider   parameterProvider
	preparer   transactionPreparer
	confirmer  transactionConfirmer
}

func newClient(logger core.Logger, client *algod.Client, preparer transactionPreparer, provider parameterProvider, confirmer transactionConfirmer) *BlockchainClient {
	return &BlockchainClient{
		logger: logger,
		client: client,
		preparer: preparer,
		provider: provider,
		confirmer: confirmer,
	}
}

func (this *BlockchainClient) DecodePayload(encoded []byte) (interface{}, error) {
	var buffer *bytes.Buffer = bytes.NewBuffer(encoded)
	var tx transaction
	var err error

	tx, err = decodeTransaction(buffer, this.provider)
	if err != nil {
		return nil, err
	}

	this.logger.Tracef("decode transaction %d", tx.getUid())

	err = this.preparer.prepare(tx)
	if err != nil {
		return nil, err
	}

	return tx, nil
}

func (this *BlockchainClient) TriggerInteraction(iact core.Interaction) error {
	var tx transaction
	var txid string
	var raw []byte
	var err error

	tx = iact.Payload().(transaction)

	this.logger.Tracef("schedule transaction %d", tx.getUid())

	raw, err = tx.getRaw()
	if err != nil {
		return err
	}

	this.logger.Tracef("submit transaction %d", tx.getUid())

	txid,err = this.client.SendRawTransaction(raw).Do(context.Background())
	if err != nil {
		return err
	}

	iact.ReportSubmit()

	return this.confirmer.confirm(iact, txid)
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

type signatureTransactionPreparer struct {
	logger  core.Logger
}

func newSignatureTransactionPreparer(logger core.Logger) transactionPreparer {
	return &signatureTransactionPreparer{
		logger: logger,
	}
}

func (this *signatureTransactionPreparer) prepare(tx transaction) error {
	var err error

	this.logger.Tracef("sign transaction %d", tx.getUid())

	_, err = tx.getRaw()
	if err != nil {
		return err
	}

	return nil
}


type transactionConfirmer interface {
	confirm(core.Interaction, string) error
}


type polltxTransactionConfirmer struct {
	logger  core.Logger
	client  *algod.Client
	ctx     context.Context
}

func newPolltxTransactionConfirmer(logger core.Logger, client *algod.Client, ctx context.Context) *polltxTransactionConfirmer {
	return &polltxTransactionConfirmer{
		logger: logger,
		client: client,
		ctx: ctx,
	}
}

func (this *polltxTransactionConfirmer) confirm(iact core.Interaction, txid string) error {
	var tx transaction = iact.Payload().(transaction)
	var info models.PendingTransactionInfoResponse
	var cli *algod.Client = this.client
	var c context.Context = this.ctx
	var round uint64
	var err error

	for {
		info, _, err = cli.PendingTransactionInformation(txid).Do(c)
		if err != nil {
			return err
		}

		if info.PoolError != "" {
			this.logger.Tracef("transaction %d aborted",
				tx.getUid())
			iact.ReportAbort()
			return nil
		}

		if info.ConfirmedRound > 0 {
			this.logger.Tracef("transaction %d commit at round %d",
				tx.getUid(), info.ConfirmedRound)
			iact.ReportCommit()
			return nil
		}

		err = this.waitNextRound(&round)
		if err != nil {
			return err
		}
	}

	return nil
}

func (this *polltxTransactionConfirmer) waitNextRound(round *uint64) error {
	var cli *algod.Client = this.client
	var c context.Context = this.ctx
	var s models.NodeStatus
	var err error

	if *round == 0 {
		s, err = cli.Status().Do(c)
		if err != nil {
			return err
		}

		*round = s.LastRound + 1
	} else {
		*round += 1
	}

	s, err = cli.StatusAfterBlock(*round).Do(c)
	if err != nil {
		return err
	}

	return nil
}


type pollblkTransactionConfirmer struct {
	logger    core.Logger
	client    *algod.Client
	ctx       context.Context
	err       error
	lock      sync.Mutex
	pendings  map[uint64]*pollblkTransactionConfirmerPending
}

type pollblkTransactionConfirmerPending struct {
	channel  chan<- error
	iact     core.Interaction
}

func newPollblkTransactionConfirmer(logger core.Logger, client *algod.Client, ctx context.Context) *pollblkTransactionConfirmer {
	var this pollblkTransactionConfirmer

	this.logger = logger
	this.client = client
	this.ctx = ctx
	this.err = nil
	this.pendings = make(map[uint64]*pollblkTransactionConfirmerPending)

	go this.run()

	return &this
}

func (this *pollblkTransactionConfirmer) confirm(iact core.Interaction, txid string) error {
	var tx transaction = iact.Payload().(transaction)
	var pending *pollblkTransactionConfirmerPending
	var uid uint64 = tx.getUid()
	var channel chan error
	var done bool

	channel = make(chan error)

	pending = &pollblkTransactionConfirmerPending{
		channel: channel,
		iact: iact,
	}

	this.lock.Lock()

	if this.pendings == nil {
		done = true
	} else {
		this.pendings[uid] = pending
		done = false
	}

	this.lock.Unlock()

	if done {
		close(channel)
		return this.err
	} else {
		return <- channel
	}
}

func (this *pollblkTransactionConfirmer) parseBlock(dest []uint64, block types.Block) []uint64 {
	var tx types.SignedTxnInBlock
	var uid uint64
	var ok bool

	for _, tx = range block.Payset {
		uid, ok = noteToUid(tx.Txn.Note)
		if !ok {
			continue
		}

		dest = append(dest, uid)
	}

	return dest
}

func (this *pollblkTransactionConfirmer) reportTransactions(uids []uint64) {
	var pendings []*pollblkTransactionConfirmerPending
	var pending *pollblkTransactionConfirmerPending
	var uid uint64
	var ok bool

	pendings = make([]*pollblkTransactionConfirmerPending, 0, len(uids))

	this.lock.Lock()

	for _, uid = range uids {
		pending, ok = this.pendings[uid]
		if !ok {
			continue
		}

		delete(this.pendings, uid)

		pendings = append(pendings, pending)
	}

	this.lock.Unlock()

	for _, pending = range pendings {
		pending.iact.ReportCommit()
	}

	for _, pending = range pendings {
		this.logger.Tracef("transaction %d committed",
			pending.iact.Payload().(transaction).getUid())

		pending.channel <- nil

		close(pending.channel)
	}
}

func (this *pollblkTransactionConfirmer) flushPendings(err error) {
	var pendings []*pollblkTransactionConfirmerPending
	var pending *pollblkTransactionConfirmerPending

	pendings = make([]*pollblkTransactionConfirmerPending, 0)

	this.lock.Lock()

	for _, pending = range this.pendings {
		pendings = append(pendings, pending)
	}

	this.pendings = nil
	this.err = err

	this.lock.Unlock()

	for _, pending = range pendings {
		pending.iact.ReportAbort()

		this.logger.Tracef("transaction %d aborted",
			pending.iact.Payload().(transaction).getUid())

		pending.channel <- err

		close(pending.channel)
	}
}

func (this *pollblkTransactionConfirmer) run() {
	var client *algod.Client = this.client
	var uids []uint64 = make([]uint64, 0)
	var status models.NodeStatus
	var block types.Block
	var round uint64
	var err error

	status, err = client.Status().Do(this.ctx)
	if err != nil {
		this.flushPendings(err)
		return
	}

	round = status.LastRound + 1
	this.logger.Tracef("start polling block at round %d", round)

	loop: for {
		status, err = client.StatusAfterBlock(round).Do(this.ctx)
		if err != nil {
			break loop
		}

		uids = uids[:0]

		for round < status.LastRound {
			this.logger.Tracef("poll block for round %d", round)

			block, err = client.Block(round).Do(this.ctx)
			if err != nil {
				if this.ctx.Err() != nil {
					break loop
				}

				this.logger.Warnf("block polling failed: %s",
					err.Error())

				continue
			}

			uids = this.parseBlock(uids, block)

			round += 1
		}

		this.reportTransactions(uids)
	}

	this.flushPendings(err)
}
