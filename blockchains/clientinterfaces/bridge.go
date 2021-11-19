package clientinterfaces


import (
	"diablo-benchmark/blockchains"
	"diablo-benchmark/blockchains/workloadgenerators"
	"diablo-benchmark/core/configs"
	"diablo-benchmark/core/results"
)


type WorkerBridge struct {
	GenericInterface
	chainConfig  *configs.ChainConfig
	inner        blockchain.Worker
}

// All this crap is to make a bridge between the old Diablo model and the new
// one.
//
// Old model: One of the workers parses for everyone and send the transactions
//            to Diablo. Diablo later dispatches the transactions between
//            workers.
//
// New model: Each worker parses its own transactions and store them locally.
//            Diablo just tell each worker which one to send during the
//            benchmark.
//
// This avoids transfering transactions back and forth between Diablo and the
// workers, with unecessary conversions making the code more complicated.
//
// Also, storing all the transactions which can be sent allows to store a state
// along the transactions, which simplifies stats collection.
//
var allBridges []*WorkerBridge = make([]*WorkerBridge, 0)
var parsedWorkloads int = 0


func NewWorkerBridge(inner blockchain.Worker) *WorkerBridge {
	return &WorkerBridge{
		inner:  inner,
	}
}

func (this *WorkerBridge) Init(chainConfig *configs.ChainConfig) {
	this.chainConfig = chainConfig
	allBridges = append(allBridges, this)
}

func (this *WorkerBridge) Cleanup() results.Results {
	var log *results.EventLog

	this.inner.StopBenchmark()
	log = this.inner.Generate()

	return log.Format(this.Window)
}

func (this *WorkerBridge) Start() {
	this.inner.StartBenchmark()
}

func (this *WorkerBridge) parseWorkload(w workloadgenerators.WorkerThreadWorkload) ([][]interface{}, error) {
	var ret [][]interface{}
	var flat [][]byte
	var i, j, n int
	var err error

	flat = make([][]byte, 0)
	for i = range w {
		for j = range w[i] {
			flat = append(flat, w[i][j])
		}
	}

	err = this.inner.ParseWorkload(flat)
	if err != nil {
		return nil, err
	}

	n = 0
	ret = make([][]interface{}, len(w))
	for i = range w {
		ret[i] = make([]interface{}, len(w[i]))
		for j = range w[i] {
			ret[i][j] = n
			n += 1
		}
	}

	return ret, nil
}

func (this *WorkerBridge) ParseWorkload(w workloadgenerators.WorkerThreadWorkload) (ret [][]interface{}, err error) {
	ret, err = allBridges[parsedWorkloads].parseWorkload(w)
	parsedWorkloads += 1
	return
}

func (this *WorkerBridge) ConnectOne(id int) error {
	return nil
}

func (this *WorkerBridge) ConnectAll(secondaryId int) error {
	var err error

	err = this.inner.Init(this.chainConfig)

	return err
}

func (this *WorkerBridge) DeploySmartContract(tx interface{}) (interface{}, error) {
	return nil, nil
}

func (this *WorkerBridge) SendRawTransaction(tx interface{}) error {
	return this.inner.SendTransaction(tx.(int))
}

func (this *WorkerBridge) SecureRead(callFunc string, callParams []byte) (interface{}, error) {
	return nil, nil
}

func (this *WorkerBridge) GetBlockByNumber(index uint64) (GenericBlock, error) {
	return GenericBlock{}, nil
}

func (this *WorkerBridge) GetBlockHeight() (uint64, error) {
	return 0, nil
}

func (this *WorkerBridge) GetTxDone() uint64 {
	return 0
}

func (this *WorkerBridge) ParseBlocksForTransactions(startNumber uint64, endNumber uint64) error {
	return nil
}

func (this *WorkerBridge) Close() {
}
