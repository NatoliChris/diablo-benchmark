package clientinterfaces

import (
	"context"
	"diablo-benchmark/blockchains/workloadgenerators"
	"diablo-benchmark/core/configs"
	"diablo-benchmark/core/results"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gagliardetto/solana-go"
	"github.com/gagliardetto/solana-go/rpc"
	"github.com/gagliardetto/solana-go/rpc/ws"
	"go.uber.org/zap"
)

type solanaClient struct {
	rpcClient *rpc.Client
	wsClient  *ws.Client
}

type SolanaInterface struct {
	Connections      []*solanaClient // Active connections to a blockchain node for information
	NextConnection   uint64
	SubscribeDone    chan bool                        // Event channel that will unsub from events
	TransactionInfo  map[solana.Signature][]time.Time // Transaction information
	bigLock          sync.Mutex
	HandlersStarted  bool         // Have the handlers been initiated?
	StartTime        time.Time    // Start time of the benchmark
	ThroughputTicker *time.Ticker // Ticker for throughput (1s)
	Throughputs      []float64    // Throughput over time with 1 second intervals
	logger           *zap.Logger
	GenericInterface
}

func (s *SolanaInterface) ActiveConn() *solanaClient {
	i := atomic.AddUint64(&s.NextConnection, 1)
	client := s.Connections[i%uint64(len(s.Connections))]
	return client
}

func NewSolanaInterface() *SolanaInterface {
	return &SolanaInterface{logger: zap.L().Named("SolanaInterface")}
}

func (s *SolanaInterface) Init(chainConfig *configs.ChainConfig) {
	s.logger.Debug("Init")
	s.Nodes = chainConfig.Nodes
	s.TransactionInfo = make(map[solana.Signature][]time.Time, 0)
	s.SubscribeDone = make(chan bool)
	s.HandlersStarted = false
	s.NumTxDone = 0
}

func (s *SolanaInterface) Cleanup() results.Results {
	s.logger.Debug("Cleanup")
	// Stop the ticker
	s.ThroughputTicker.Stop()

	// clean up connections and format results
	if s.HandlersStarted {
		s.SubscribeDone <- true
	}

	txLatencies := make([]float64, 0)
	var avgLatency float64

	var endTime time.Time

	success := uint(0)
	fails := uint(s.Fail)

	for sig, v := range s.TransactionInfo {
		if len(v) > 1 {
			txLatency := v[1].Sub(v[0]).Milliseconds()
			txLatencies = append(txLatencies, float64(txLatency))
			avgLatency += float64(txLatency)
			if v[1].After(endTime) {
				endTime = v[1]
			}

			success++
		} else {
			s.logger.Debug("Missing", zap.String("sig", sig.String()))
			status, err := s.ActiveConn().rpcClient.GetSignatureStatuses(context.Background(), true, sig)
			if err != nil {
				s.logger.Debug("Status", zap.Error(err))
			} else {
				s.logger.Debug("Status", zap.Any("status", status.Value))
			}
			fails++
		}
	}

	s.logger.Debug("Statistics being returned",
		zap.Uint("success", success),
		zap.Uint("fail", fails))

	// Calculate the throughput and latencies
	var throughput float64
	if len(txLatencies) > 0 {
		throughput = (float64(s.NumTxDone) - float64(s.Fail)) / (endTime.Sub(s.StartTime).Seconds())
		avgLatency = avgLatency / float64(len(txLatencies))
	} else {
		avgLatency = 0
		throughput = 0
	}

	averageThroughput := float64(0)
	var calculatedThroughputSeconds = []float64{s.Throughputs[0]}
	for i := 1; i < len(s.Throughputs); i++ {
		calculatedThroughputSeconds = append(calculatedThroughputSeconds, float64(s.Throughputs[i]-s.Throughputs[i-1]))
		averageThroughput += float64(s.Throughputs[i] - s.Throughputs[i-1])
	}

	averageThroughput = averageThroughput / float64(len(s.Throughputs))

	s.logger.Debug("Results being returned",
		zap.Float64("avg throughput", averageThroughput),
		zap.Float64("throughput (as is)", throughput),
		zap.Float64("latency", avgLatency),
		zap.String("ThroughputWindow", fmt.Sprintf("%v", calculatedThroughputSeconds)),
	)

	return results.Results{
		TxLatencies:       txLatencies,
		AverageLatency:    avgLatency,
		Throughput:        averageThroughput,
		ThroughputSeconds: calculatedThroughputSeconds,
		Success:           success,
		Fail:              fails,
	}
}

func (s *SolanaInterface) throughputSeconds() {
	s.ThroughputTicker = time.NewTicker((time.Duration(s.Window) * time.Second))
	seconds := float64(0)

	for range s.ThroughputTicker.C {
		seconds += float64(s.Window)
		s.Throughputs = append(s.Throughputs, float64(s.NumTxDone-s.Fail))
	}
}

func (s *SolanaInterface) Start() {
	s.logger.Debug("Start")
	s.StartTime = time.Now()
	go s.throughputSeconds()
}

func (s *SolanaInterface) ParseWorkload(workload workloadgenerators.WorkerThreadWorkload) ([][]interface{}, error) {
	s.logger.Debug("ParseWorkload")
	parsedWorkload := make([][]interface{}, 0)

	for _, v := range workload {
		intervalTxs := make([]interface{}, 0)
		for _, txBytes := range v {
			t := solana.Transaction{}
			err := json.Unmarshal(txBytes, &t)
			if err != nil {
				return nil, err
			}

			intervalTxs = append(intervalTxs, &t)
		}
		parsedWorkload = append(parsedWorkload, intervalTxs)
	}

	s.TotalTx = len(parsedWorkload)

	return parsedWorkload, nil
}

// parseBlocksForTransactions parses the the given block number for the transactions
func (s *SolanaInterface) parseBlocksForTransactions(slot uint64) {
	s.logger.Debug("parseBlocksForTransactions", zap.Uint64("slot", slot))

	var block *rpc.GetBlockResult
	var err error
	for attempt := 0; attempt < 100; attempt++ {
		includeRewards := false
		block, err = s.ActiveConn().rpcClient.GetBlockWithOpts(
			context.Background(),
			slot,
			&rpc.GetBlockOpts{
				TransactionDetails: rpc.TransactionDetailsSignatures,
				Rewards:            &includeRewards,
				Commitment:         rpc.CommitmentFinalized,
			})

		if err != nil {
			time.Sleep(50 * time.Millisecond)
			continue
		}
		if block == nil {
			time.Sleep(50 * time.Millisecond)
			continue
		}
		break
	}

	if block == nil {
		s.logger.Warn("Empty block", zap.Uint64("slot", slot))
		return
	}

	tNow := time.Now()
	var tAdd uint64

	s.bigLock.Lock()

	for _, sig := range block.Signatures {
		if info, ok := s.TransactionInfo[sig]; ok && len(info) == 1 {
			s.TransactionInfo[sig] = append(info, tNow)
			tAdd++
		}
	}

	s.bigLock.Unlock()

	atomic.AddUint64(&s.NumTxDone, tAdd)
	s.logger.Debug("Stats", zap.Uint64("sent", atomic.LoadUint64(&s.NumTxSent)), zap.Uint64("done", atomic.LoadUint64(&s.NumTxDone)))
}

// EventHandler subscribes to the blocks and handles the incoming information about the transactions
func (s *SolanaInterface) EventHandler() {
	s.logger.Debug("EventHandler")
	sub, err := s.ActiveConn().wsClient.RootSubscribe()
	if err != nil {
		s.logger.Warn("RootSubscribe", zap.Error(err))
		return
	}
	defer sub.Unsubscribe()
	go func() {
		for range s.SubscribeDone {
			sub.Unsubscribe()
			return
		}
	}()

	var currentSlot uint64 = 0
	for {
		got, err := sub.Recv()
		if err != nil {
			s.logger.Warn("RootResult", zap.Error(err))
			return
		}
		if got == nil {
			s.logger.Warn("Empty root")
			return
		}
		newSlot := uint64(*got)
		if currentSlot == 0 {
			s.logger.Debug("First slot", zap.Uint64("got", newSlot))
		} else if newSlot <= currentSlot {
			s.logger.Debug("Slot skipped", zap.Uint64("got", newSlot), zap.Uint64("current", currentSlot))
			continue
		} else if newSlot > currentSlot+1 {
			s.logger.Debug("Missing slot update, requesting missing slots", zap.Uint64("got", newSlot), zap.Uint64("current", currentSlot))
			for currentSlot+1 < newSlot {
				currentSlot++
				go s.parseBlocksForTransactions(currentSlot)
			}
		}
		currentSlot = newSlot
		// Got a head
		go s.parseBlocksForTransactions(currentSlot)
	}
}

func (s *SolanaInterface) ConnectOne(id int) error {
	s.logger.Debug("ConnectOne")
	return errors.New("do not use")
}

func (s *SolanaInterface) ConnectAll(primaryID int) error {
	s.logger.Debug("ConnectAll")
	// If our ID is greater than the nodes we know, there's a problem!
	if primaryID >= len(s.Nodes) {
		return errors.New("invalid client primary ID")
	}

	// Connect all the others
	for _, node := range s.Nodes {
		conn := rpc.New(fmt.Sprintf("http://%s", node))

		ip, portStr, err := net.SplitHostPort(node)
		if err != nil {
			return err
		}
		port, err := strconv.Atoi(portStr)
		if err != nil {
			return err
		}

		sock, err := ws.Connect(context.Background(), fmt.Sprintf("ws://%s", net.JoinHostPort(ip, strconv.Itoa(port+1))))
		if err != nil {
			return err
		}

		s.Connections = append(s.Connections, &solanaClient{conn, sock})
	}

	if !s.HandlersStarted {
		go s.EventHandler()
		s.HandlersStarted = true
	}

	return nil
}

func (s *SolanaInterface) DeploySmartContract(tx interface{}) (interface{}, error) {
	s.logger.Debug("DeploySmartContract")
	return nil, errors.New("not implemented")
}

func (s *SolanaInterface) SendRawTransaction(tx interface{}) error {
	go func() {
		transaction := tx.(*solana.Transaction)

		sig, err := s.ActiveConn().rpcClient.SendTransactionWithOpts(context.Background(), transaction, false, rpc.CommitmentFinalized)
		if err != nil {
			s.logger.Debug("Err",
				zap.Error(err),
			)
			atomic.AddUint64(&s.Fail, 1)
			atomic.AddUint64(&s.NumTxDone, 1)
		}

		s.bigLock.Lock()
		s.TransactionInfo[sig] = []time.Time{time.Now()}
		s.bigLock.Unlock()

		atomic.AddUint64(&s.NumTxSent, 1)
	}()

	return nil
}

func (s *SolanaInterface) SecureRead(callFunc string, callParams []byte) (interface{}, error) {
	s.logger.Debug("SecureRead")
	return nil, errors.New("not implemented")
}

func (s *SolanaInterface) GetBlockByNumber(index uint64) (GenericBlock, error) {
	s.logger.Debug("GetBlockByNumber")
	return GenericBlock{}, errors.New("not implemented")
}

func (s *SolanaInterface) GetBlockHeight() (uint64, error) {
	s.logger.Debug("GetBlockHeight")
	return 0, errors.New("not implemented")
}

func (s *SolanaInterface) ParseBlocksForTransactions(startNumber uint64, endNumber uint64) error {
	s.logger.Debug("ParseBlocksForTransactions")
	return errors.New("not implemented")
}

func (s *SolanaInterface) Close() {
	s.logger.Debug("Close")
	// Close all connections
	for _, client := range s.Connections {
		client.wsClient.Close()
	}
}
