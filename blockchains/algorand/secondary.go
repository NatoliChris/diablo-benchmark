package algorand


import (
	"time"

	"go.uber.org/zap"

	"diablo-benchmark/core/configs"
	"diablo-benchmark/core/results"
)


type submitEvent struct {
	index  int
	when   time.Time
}

type commitEvent struct {
	index  int
	when   time.Time
}

type Secondary struct {
	chainConfig   *configs.ChainConfig          // (half-)parsed chain.yaml
	blockchain    *Blockchain             // interface to actual blockchain
	transactions  []transaction                     // transactions to send
	uidmap        map[int]int               // uid -> index of transactions
	pollBlocks    bool            // get committed transactions from blocks
	notifySubmit  chan submitEvent
	notifyCommit  chan commitEvent
	notifyStop    chan bool
	resultsChan   chan *results.EventLog
}


func NewWorker() *Secondary {
	return &Secondary{}
}

func (this *Secondary) Init(c *configs.ChainConfig) error {
	var bc *Blockchain
	var conf *Config
	var err error

	conf, err = parseConfig(c)
	if err != nil {
		return err
	}

	bc, err = NewBlockchain(conf)
	if err != nil {
		return err
	}

	this.chainConfig = c
	this.blockchain = bc
	this.pollBlocks = true
	this.notifySubmit = make(chan submitEvent, 128)
	this.notifyCommit = make(chan commitEvent, 128)
	this.notifyStop = make(chan bool)
	this.resultsChan = make(chan *results.EventLog)

	return nil
}


func (this *Secondary) ParseWorkload(workload [][]byte) error {
	var err error
	var i int

	this.transactions = make([]transaction, len(workload))
	this.uidmap = make(map[int]int, len(workload))

	for i = range workload {
		this.transactions[i], err = decodeTransaction(workload[i])
		if err != nil {
			return err
		}

		// Quick fix for deadline of 2021-12-14
		// Purpose: force per-region access
		// Remove me
		this.transactions[i].endpoint = this.transactions[i].endpoint % this.blockchain.Population()

		this.uidmap[this.transactions[i].uid] = i
	}

	return nil
}


func (this *Secondary) StartBenchmark() error {
	var now time.Time = time.Now()

	go this.collectEvents(now)

	if this.pollBlocks {
		go this.pollTransactions()
	}

	return nil
}

func (this *Secondary) StopBenchmark() error {
	close(this.notifyStop)

	return nil
}


func (this *Secondary) collectEvents(start time.Time) {
	var submits []int64 = make([]int64, len(this.transactions))
	var commits []int64 = make([]int64, len(this.transactions))
	var se submitEvent
	var ce commitEvent
	var i int

	for i = range this.transactions {
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

func (this *Secondary) pollTransactions() {
	var round uint64 = 0
	var notes [][]byte
	var now time.Time
	var note []byte
	var err error
	var uid int

	loop: for {
		select {
		case <-this.notifyStop:
			break loop
		default:
			round, notes, err = this.blockchain.PollBlock(0, round)
			now = time.Now()

			if err != nil {
				zap.L().Error("Block polling failed",
					zap.Uint64("round", round),
					zap.Error(err))
				continue
			}

			for _, note = range notes {
				uid = getUidFromTransactionNote(note)
				this.notifyCommit <- commitEvent{
					index:  this.uidmap[uid],
					when:   now,
				}
			}
		}
	}
}

func (this *Secondary) waitTransaction(index int, txid string) {
	var tx *transaction = &this.transactions[index]
	var now time.Time
	var err error

	err = this.blockchain.WaitTransaction(tx.endpoint, txid)
	now = time.Now()

	if err != nil {
		return
	}

	this.notifyCommit <- commitEvent{
		index:  index,
		when:   now,
	}
}

func (this *Secondary) sendTransaction(index int, tx *transaction) error {
	var txid string
	var err error

	txid, err = this.blockchain.SendTransaction(tx.endpoint, tx.raw)
	if err != nil {
		return err
	}

	if this.pollBlocks == false {
		this.waitTransaction(index, txid)
	}

	return nil
}

func (this *Secondary) SendTransaction(index int) error {
	var tx *transaction = &this.transactions[index]
	var now time.Time

	now = time.Now()
	this.notifySubmit <- submitEvent{
		index:  index,
		when:   now,
	}

	go this.sendTransaction(index, tx)

	return nil
}


func (this *Secondary) computeResults(submits, commits []int64) *results.EventLog {
	var ret *results.EventLog = results.NewEventLog()
	var i int

	for i = range this.transactions {
		if submits[i] >= 0 {
			ret.AddSubmit(i, submits[i])
		}
		if commits[i] >= 0 {
			ret.AddCommit(i, commits[i])
		}
	}

	return ret
}

func (this *Secondary) Generate() *results.EventLog {
	var ret *results.EventLog = <-this.resultsChan

	close(this.notifySubmit)
	close(this.notifyCommit)
	close(this.resultsChan)

	return ret
}
