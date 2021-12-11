package diem


import (
	"fmt"
	"math/rand"

	"go.uber.org/zap"

	"diablo-benchmark/core/configs"
	"diablo-benchmark/core/workload"
)


type Controller struct {
	benchConfig  *configs.BenchConfig
	intervals    []int
	blockchain   *blockchain
}


func NewController() *Controller {
	return &Controller{}
}


func (this *Controller) Init(c *configs.ChainConfig, b *configs.BenchConfig, txs []int) error {
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

	this.benchConfig = b
	this.intervals = txs

	return nil
}

func (this *Controller) Setup() error {
	return nil
}


func (this *Controller) generateSimpleTransaction(dest *workload.Workload, random *rand.Rand, secondaryId, threadId, interval int, froms []int, sequences []int) error {
	var population, endpoint, from, to int
	var tx *transaction
	var err error

	population = this.blockchain.population()

	endpoint = random.Int() % this.blockchain.size()
	from = froms[random.Int() % len(froms)]
	to = random.Int() % population

	zap.L().Debug("new transaction", zap.Int("secondary", secondaryId),
		zap.Int("thread", threadId), zap.Int("interval", interval),
		zap.Int("from", from), zap.Int("to", to),
		zap.Int("sequence", sequences[from]),
		zap.Int("endpoint", endpoint))

	tx = newTransaction(from, to, 100, sequences[from], endpoint)
	if err != nil {
		return err
	}

	sequences[from] += 1
	dest.Add(secondaryId, threadId, interval, tx.encode())

	return nil
}

func (this *Controller) generateTransaction(dest *workload.Workload, random *rand.Rand, secondaryId, threadId, interval int, froms []int, sequences []int) error {
	switch this.benchConfig.TxInfo.TxType {
	case configs.TxTypeSimple:
		return this.generateSimpleTransaction(dest, random,
			secondaryId, threadId, interval, froms, sequences)
	default:
		return fmt.Errorf("unknown workload type: %s",
			this.benchConfig.TxInfo.TxType)
	}
}

func (this *Controller) Generate() (*workload.Workload, error) {
	var secondaryId, threadId, interval, txnum, home, i int
	var ret *workload.Workload = workload.New()
	var secondaries, threads, population int
	var sequences []int
	var rgen *rand.Rand
	var homes [][]int
	var err error

	secondaries = this.benchConfig.Secondaries
	threads = this.benchConfig.Threads
	population = this.blockchain.population()

	sequences = make([]int, population)
	homes = make([][]int, secondaries * threads)

	if population < len(homes) {
		err = fmt.Errorf("less accounts (%d) than clients (%d)",
			population, len(homes))
		return nil, err
	}

	for i = 0; i < population; i++ {
		home = i % len(homes)
		homes[home] = append(homes[home], i)
		sequences[i] = 0
	}

	rgen = rand.New(rand.NewSource(int64(secondaries * threads)))

	for secondaryId = 0; secondaryId < secondaries; secondaryId++ {
		for threadId = 0; threadId < threads; threadId++ {
			home = secondaryId * threads + threadId
			for interval, txnum = range this.intervals {
				for i = 0; i < txnum; i++ {
					err = this.generateTransaction(ret,
						rgen, secondaryId, threadId,
						interval, homes[home],
						sequences)

					if err != nil {
						return nil, err
					}
				}
			}
		}
	}

	return ret, nil
}
