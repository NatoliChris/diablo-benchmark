package results


// import (
// 	"go.uber.org/zap"
// )


type transactionStat struct {
	submit  int64                                   // -1 if never happened
	commit  int64                                   // -1 if never happened
}


type EventLog struct {
	stats  map[int]*transactionStat             // index -> transactionStat
}


func NewEventLog() *EventLog {
	return &EventLog{
		stats:  make(map[int]*transactionStat, 0),
	}
}


func (this *EventLog) getStat(index int) *transactionStat {
	var stat *transactionStat
	var present bool

	stat, present = this.stats[index]

	if present == false {
		stat = &transactionStat{
			submit:  -1,
			commit:  -1,
		}

		this.stats[index] = stat
	}

	return stat
}


// Log that a transaction has been submitted to the blockchain.
//
// index: index of the transaction within thread workload
// when:  submission time relative to benchmark start in milliseconds
//
func (this *EventLog) AddSubmit(index int, when int64) {
	this.getStat(index).submit = when
}

// Log that a transaction has been committed in the blockchain.
// It is now safe to assume that the transaction is in the ledger and will not
// disappear.
//
// index: index of the transaction within thread workload
// when: commit time relative to benchmark start in milliseconds
//
func (this *EventLog) AddCommit(index int, when int64) {
	this.getStat(index).commit = when
}


// Bridge with old Diablo result format.
//
func (this *EventLog) Format(window int) Results {
	var i, txsum, total, maxCommit int
	var commitPerSecond []int
	var s *transactionStat
	var lat, thr float64
	var ret Results

	ret.TxLatencies = make([]float64, 0)
	ret.Success = 0
	ret.Fail = 0

	maxCommit = 0

	for _, s = range this.stats {
		if (s.submit >= 0) && (s.commit >= 0) {
			lat = float64(s.commit - s.submit)
			ret.TxLatencies = append(ret.TxLatencies, lat)
			ret.Success += 1
			if int(s.commit / 1000) > maxCommit {
				maxCommit = int(s.commit / 1000)
			}
		} else {
			ret.Fail += 1
		}
	}

	commitPerSecond = make([]int, maxCommit + 1)

	for _, s = range this.stats {
		if (s.submit >= 0) && (s.commit >= 0) {
			commitPerSecond[int(s.commit / 1000)] += 1
		}
	}

	ret.ThroughputSeconds = make([]float64, len(commitPerSecond))
	total = 0
	txsum = 0

	for i = 0; i < len(commitPerSecond); i++ {
		total += commitPerSecond[i]
		txsum += commitPerSecond[i]

		if i >= window {
			txsum -= commitPerSecond[i - window]
			thr = float64(txsum) / float64(window)
		} else {
			thr = float64(txsum) / float64(i+1)
		}

		ret.ThroughputSeconds[i] = thr
	}

	if len(commitPerSecond) > 0 {
		ret.Throughput = float64(total) / float64(len(commitPerSecond))
	}

	return ret
}
