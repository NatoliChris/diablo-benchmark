package diem


import (
	"time"
	"strings"
	"sync/atomic"

	"go.uber.org/zap"

	"diablo-benchmark/core/configs"
	"diablo-benchmark/core/results"
)


type signedTransaction struct {
	tx    *transaction
	raw   []byte
}

type signedTransactionQueue struct {
	nextToSubmit  int64
	nextToCommit  int64
	queue         []signedTransaction
}

type submitEvent struct {
	index  int
	when   time.Time
}

type commitEvent struct {
	index  int
	when   time.Time
}

type Worker struct {
	chainConfig         *configs.ChainConfig
	blockchain          *blockchain
	txqueues            []signedTransactionQueue
	transactionOrigins  []int
	notifySubmit        chan submitEvent
	notifyCommit        chan commitEvent
	notifyStop          chan bool
	resultsChan         chan *results.EventLog
}


func NewWorker() *Worker {
	return &Worker{}
}


func (this *Worker) Init(c *configs.ChainConfig) error {
	var conf *config
	var err error

	conf, err = parseConfig(c)
	if err != nil {
		return err
	}

	this.blockchain, err = newBlockchain(conf)
	if err != nil {
		return err
	}

	this.chainConfig = c
	this.notifySubmit = make(chan submitEvent, 128)
	this.notifyCommit = make(chan commitEvent, 128)
	this.notifyStop = make(chan bool)
	this.resultsChan = make(chan *results.EventLog)

	return nil
}

func (this *Worker) ParseWorkload(workload [][]byte) error {
	var tx *transaction
	var raw []byte
	var err error
	var i int

	this.transactionOrigins = make([]int, len(workload))

	this.txqueues = make([]signedTransactionQueue,
		this.blockchain.population())
	for i = 0; i < len(this.txqueues); i++ {
		this.txqueues[i].nextToSubmit = 0
		this.txqueues[i].queue = make([]signedTransaction, 0)
	}

	for i = range workload {
		tx, err = decodeTransaction(workload[i])
		if err != nil {
			return err
		}

		raw, err = this.blockchain.prepareSimpleTransaction(
			tx.from, tx.to, tx.amount, uint64(tx.sequence))
		if err != nil {
			return err
		}

		this.transactionOrigins[i] = tx.from
		this.txqueues[tx.from].queue =
			append(this.txqueues[tx.from].queue, signedTransaction{
			tx:   tx,
			raw:  raw,
		})
	}

	return nil
}

func (this *Worker) StartBenchmark() error {
	var now time.Time = time.Now()

	go this.collectEvents(now)

	return nil
}

func (this *Worker) StopBenchmark() error {
	close(this.notifyStop)

	return nil
}

func (this *Worker) collectEvents(start time.Time) {
	var submits []int64 = make([]int64, len(this.transactionOrigins))
	var commits []int64 = make([]int64, len(this.transactionOrigins))
	var se submitEvent
	var ce commitEvent
	var i int

	for i = range this.transactionOrigins {
		submits[i] = -1
		commits[i] = -1
	}

	loop: for {
		select {
		case se = <-this.notifySubmit:
			submits[se.index] = se.when.Sub(start).Milliseconds()
		case ce = <-this.notifyCommit:
			commits[ce.index] = ce.when.Sub(start).Milliseconds()
		case <-this.notifyStop:
			break loop
		}
	}

	this.resultsChan <- this.computeResults(submits, commits)
}

func (this *Worker) getNextTxSlot(from int) int {
	var slot int64 = atomic.AddInt64(&this.txqueues[from].nextToSubmit, 1)
	return int(slot - 1)
}

func (this *Worker) commitTxSlot(from, slot int) {
	var addr *int64 = &this.txqueues[from].nextToCommit
	var old, new int64

	new = int64(slot)

	for {
		old = atomic.LoadInt64(addr)

		if old >= new {
			return
		}

		if atomic.CompareAndSwapInt64(addr, old, new) {
			return
		}
	}
}

func (this *Worker) resetTxSlot(from, slot int) {
	var maddr *int64 = &this.txqueues[from].nextToCommit
	var addr *int64 = &this.txqueues[from].nextToSubmit
	var min, old, new int64

	new = int64(slot)

	for {
		min = atomic.LoadInt64(maddr)
		old = atomic.LoadInt64(addr)

		if new < min {
			return
		}

		if old <= new {
			return
		}

		if atomic.CompareAndSwapInt64(addr, old, new) {
			return
		}
	}
}

func (this *Worker) sendTransaction(index int) {
	var from int = this.transactionOrigins[index]
	var slot int = this.getNextTxSlot(from)
	var stx *signedTransaction = &this.txqueues[from].queue[slot]
	var submit, commit time.Time
	var err error

	defer func() { recover() } ()  // don't panic

	zap.L().Debug("Send transaction",
		zap.Int("from", stx.tx.from), zap.Int("to", stx.tx.to),
		zap.Int("sequence", stx.tx.sequence),
		zap.Int("endpoint", stx.tx.endpoint))


	submit = time.Now()
	err = this.blockchain.sendTransaction(stx.tx.endpoint, stx.raw)

	if err != nil {
		if !strings.Contains(err.Error(), "SEQUENCE_NUMBER_TOO_OLD") {
			this.resetTxSlot(from, slot)
		}

		this.notifySubmit <- submitEvent{
			index:  index,
			when:   submit,
		}

		zap.L().Warn("Cannot send transaction",
			zap.Int("from", stx.tx.from), zap.Int("to", stx.tx.to),
			zap.Int("sequence", stx.tx.sequence),
			zap.Int("endpoint", stx.tx.endpoint),
			zap.Error(err))

		return
	}

	err = this.blockchain.waitTransaction(stx.tx.endpoint, stx.raw)
	commit = time.Now()

	if err != nil {
		this.resetTxSlot(from, slot)

		this.notifySubmit <- submitEvent{
			index:  index,
			when:   submit,
		}

		zap.L().Warn("Transaction failed",
			zap.Int("from", stx.tx.from), zap.Int("to", stx.tx.to),
			zap.Int("sequence", stx.tx.sequence),
			zap.Int("endpoint", stx.tx.endpoint),
			zap.Error(err))

		return
	}

	this.commitTxSlot(from, slot)

	this.notifySubmit <- submitEvent{
		index:  index,
		when:   submit,
	}
	this.notifyCommit <- commitEvent{
		index:  index,
		when:   commit,
	}
}

func (this *Worker) SendTransaction(index int) error {
	go this.sendTransaction(index)

	return nil
}

func (this *Worker) computeResults(submits, commits []int64) *results.EventLog {
	var ret *results.EventLog = results.NewEventLog()
	var i int

	for i = range this.transactionOrigins {
		if submits[i] >= 0 {
			ret.AddSubmit(i, submits[i])
		}
		if commits[i] >= 0 {
			ret.AddCommit(i, commits[i])
		}
	}

	return ret
}

func (this *Worker) Generate() *results.EventLog {
	var ret *results.EventLog = <-this.resultsChan

	close(this.notifySubmit)
	close(this.notifyCommit)
	close(this.resultsChan)

	return ret
}
