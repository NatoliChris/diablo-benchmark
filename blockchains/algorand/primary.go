package algorand


import (
	"fmt"
	"io/ioutil"
	"math/rand"

	"go.uber.org/zap"

	"diablo-benchmark/core/configs"
	"diablo-benchmark/core/workload"
)


type Primary struct {
	benchConfig  *configs.BenchConfig                  // parsed bench.yaml
	intervals    []int             // precomputed tx/thread for each second
	blockchain   *Blockchain              // interface to actual blockchain
	appid        uint64            // appId of the deployed contract if any
}


func NewController() *Primary {
	return &Primary{}
}

func (this *Primary) Init(c *configs.ChainConfig, b *configs.BenchConfig, txs []int) error {
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

	this.benchConfig = b
	this.intervals = txs
	this.blockchain = bc

	return nil
}


func (this *Primary) deployContract() error {
	var path string = this.benchConfig.ContractInfo.Path
	var source []byte
	var err error

	source, err = ioutil.ReadFile(path)
	if err != nil {
		return err
	}

	this.appid, err = this.blockchain.DeployContract(0,0, source, 1,1,1,1)
	if err != nil {
		return err
	}

	zap.L().Info("Contract Deployed", zap.String("path", path))

	return nil
}

func (this *Primary) Setup() error {
	var err error

	if this.benchConfig.TxInfo.TxType == configs.TxTypeContract {
		err = this.deployContract()
		if err != nil {
			return err
		}
	}

	return nil
}


const simpleTransactionAmount = 10000

func (this *Primary) newSimpleTransaction(endpoint, from, to, uid int) (transaction, error) {
	var note, raw []byte
	var tx transaction
	var err error

	note = makeTransactionNote(uid)
	raw, err = this.blockchain.PrepareSimpleTransaction(from, to,
		simpleTransactionAmount, note)
	if err != nil {
		return tx, err
	}

	tx = newTransaction(endpoint, uid, raw)

	return tx, nil
}

func (this *Primary) generateSimpleTransaction(dest *workload.Workload, random *rand.Rand, secondaryId, threadId, interval, uid int) error {
	var population, endpoint, from, to int
	var tx transaction
	var err error

	population = this.blockchain.Population()

	endpoint = random.Int() % this.blockchain.Size()
	from = random.Int() % population
	to = random.Int() % population

	tx, err = this.newSimpleTransaction(endpoint, from, to, uid)
	if err != nil {
		return err
	}

	dest.Add(secondaryId, threadId, interval, tx.encode())

	return nil
}

func (this *Primary) newOptInTransaction(endpoint, from, uid int) (transaction, error) {
	var note, raw []byte
	var tx transaction
	var appid uint64
	var err error

	appid = this.appid
	note = makeTransactionNote(uid)
	raw, err = this.blockchain.PrepareOptInTransaction(from, appid, note)
	if err != nil {
		return tx, err
	}

	tx = newTransaction(endpoint, uid, raw)

	return tx, nil
}

func (this *Primary) newNoOpTransaction(endpoint, from, uid int, args [][]byte) (transaction, error) {
	var note, raw []byte
	var tx transaction
	var appid uint64
	var err error

	appid = this.appid
	note = makeTransactionNote(uid)
	raw, err = this.blockchain.PrepareNoOpTransaction(from,appid,args,note)
	if err != nil {
		return tx, err
	}

	tx = newTransaction(endpoint, uid, raw)

	return tx, nil
}

func (this *Primary) pickFunction(random *rand.Rand) int {
	var function configs.ContractFunction
	var functions, choice, index int

	functions = 0
	for _, function = range this.benchConfig.ContractInfo.Functions {
		functions += function.Ratio
	}

	choice = random.Int() % functions

	functions = 0
	for index, function = range this.benchConfig.ContractInfo.Functions {
		functions += function.Ratio
		if functions > choice {
			return index
		}
	}

	panic(fmt.Errorf(
		"failed to pick function: choice = %d / functions = %d",
		choice, functions))

	return -1
}

func (this *Primary) generateContractTransaction(dest *workload.Workload, random *rand.Rand, secondaryId, threadId, interval, uid int) error {
	var function configs.ContractFunction
	var endpoint, from, funcid int
	var tx transaction
	var args [][]byte
	var err error

	endpoint = random.Int() % this.blockchain.Size()
	from = random.Int() % this.blockchain.Population()

	funcid = this.pickFunction(random)
	function = this.benchConfig.ContractInfo.Functions[funcid]

	args = make([][]byte, 1)
	args[0] = []byte(function.Name)

	tx, err = this.newNoOpTransaction(endpoint, from, uid, args)
	if err != nil {
		return err
	}

	dest.Add(secondaryId, threadId, interval, tx.encode())

	return nil
}

func (this *Primary) generateTransaction(dest *workload.Workload, random *rand.Rand, secondaryId, threadId, interval, uid int) error {
	switch this.benchConfig.TxInfo.TxType {
	case configs.TxTypeSimple:
		return this.generateSimpleTransaction(dest, random,
			secondaryId, threadId, interval, uid)
	case configs.TxTypeContract:
		return this.generateContractTransaction(dest, random,
			secondaryId, threadId, interval, uid)
	default:
		return fmt.Errorf("unknown workload type: %s",
			this.benchConfig.TxInfo.TxType)
	}
}

func (this *Primary) Generate() (*workload.Workload, error) {
	var uid, secondaryId, threadId, interval, txnum, i int
	var ret *workload.Workload = workload.New()
	var secondaries, threads int
	var rgen *rand.Rand
	var err error

	secondaries = this.benchConfig.Secondaries
	threads = this.benchConfig.Threads
	rgen = rand.New(rand.NewSource(int64(secondaries * threads)))
	uid = 0

	for secondaryId = 0; secondaryId < secondaries; secondaryId++ {
		for threadId = 0; threadId < threads; threadId++ {
			for interval, txnum = range this.intervals {
				for i = 0; i < txnum; i++ {
					err = this.generateTransaction(ret,
						rgen, secondaryId, threadId,
						interval, uid)

					if err != nil {
						return nil, err
					}

					uid += 1
				}
			}
		}
	}

	return ret, nil
}
