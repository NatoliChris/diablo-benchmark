package handlers

import (
	"diablo-benchmark/blockchains/clientinterfaces"
	"diablo-benchmark/core/results"
	"errors"
	"fmt"
	"go.uber.org/zap"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Handler loop that dispatches the workload into channels and creates routines that will read and send
type WorkloadHandler struct {
	numThread            uint32                                 // Number of workers
	workerThreadChannels []chan interface{}                     // Channels between the threads to have the workload from
	activeClients        []clientinterfaces.BlockchainInterface // Number of client threads that are run
	FullWorkload         []interface{}                          // Workload
	readyChannels        []chan bool                            // channels that signal ready to start
	wg                   *sync.WaitGroup                        // All threads done
	numTx                uint64                                 // number of transactions sent
	numErrors            uint64                                 // Number of errors during workload
}

func NewWorkloadHandler(numThread uint32, clients []clientinterfaces.BlockchainInterface) *WorkloadHandler {
	// Generate the channels to speak to the workers.
	return &WorkloadHandler{
		numThread:     numThread,
		activeClients: clients,
	}
}

// Initialise the clients and connect
func (wh *WorkloadHandler) Connect(nodes []string, ID int) error {
	var combinedErr []string
	for _, v := range wh.activeClients {
		v.Init(nodes)
		e := v.ConnectAll(ID)
		if e != nil {
			combinedErr = append(combinedErr, e.Error())
		}
	}

	if len(combinedErr) > 0 {
		return errors.New(strings.Join(combinedErr[:], ", "))
	}

	return nil
}

// Parse the workloads on each client, populate the channels
func (wh *WorkloadHandler) ParseWorkloads(rawWorkload [][]byte) error {
	// Should be able to parse the workloads from transactions into
	// bytes
	fullWorkload, err := wh.activeClients[0].ParseWorkload(rawWorkload)

	if err != nil {
		return err
	}

	wh.FullWorkload = fullWorkload
	perThreadWL := len(fullWorkload) / int(wh.numThread)

	// Set up the workload channels
	var readyChannels []chan bool
	var wg sync.WaitGroup

	// Distribute the workload into the channels
	for i := 0; i < int(wh.numThread); i++ {
		// Set up
		rch := make(chan bool, 0)
		readyChannels = append(readyChannels, rch)
		workerChannel := make(chan interface{}, len(rawWorkload))
		for _, v := range fullWorkload[i*perThreadWL : (i+1)*perThreadWL] {
			workerChannel <- v
		}
		close(workerChannel)
		// Distribute and start the goroutine
		wg.Add(1)
		go wh.runner(
			wh.activeClients[i],
			workerChannel,
			&wg,
			rch,
		)
	}
	wh.readyChannels = readyChannels
	wh.wg = &wg
	return nil
}

// Runner for the thread
func (wh *WorkloadHandler) runner(blockchainInterface clientinterfaces.BlockchainInterface, workload chan interface{}, wg *sync.WaitGroup, ready chan bool) {
	var errs []error
	defer wg.Done()

	// Wait for the signal to go
	<-ready
	zap.L().Info("ready")
	for tx := range workload {
		e := blockchainInterface.SendRawTransaction(tx)
		if e != nil {
			errs = append(errs, e)
		}
		atomic.AddUint64(&wh.numTx, 1)
	}

	// TODO handle errors
}

func (wh *WorkloadHandler) statusPrinter(stopCh chan bool) {
	var timer *time.Timer = time.NewTimer(5 * time.Second)
	for {
		select {
		case <-stopCh:
			return
		case <-timer.C:
			// print
			zap.L().Info(fmt.Sprintf("%d tx | %d errors", wh.numTx, wh.numErrors))
			timer = time.NewTimer(5 * time.Second)
		}
	}
}

// Run the benchmark()
func (wh *WorkloadHandler) RunBench() error {
	stopPrinting := make(chan bool, 0)

	go wh.statusPrinter(stopPrinting)

	for _, ch := range wh.readyChannels {
		ch <- true
	}

	wh.wg.Wait()
	stopPrinting <- true

	// TODO get errors
	// add error channel to runner so that it can append the errors
	return nil
}

// Get the results
func (wh *WorkloadHandler) HandleCleanup() results.Results {
	// Aggregate the results
	allLatencies := make([]float64, 0)
	var avgThroughput float64 = 0
	var avgLatency float64 = 0
	for i, c := range wh.activeClients {
		zap.L().Debug("processing cleanup",
			zap.Int("client", i))
		res := c.Cleanup()
		avgThroughput += res.Throughput
		allLatencies = append(allLatencies, res.TxLatencies...)
		avgLatency += res.AverageLatency
	}

	zap.L().Debug("Cleanup results",
		zap.Float64("avg throughput", avgThroughput/float64(wh.numThread)),
		zap.Float64("avg latency", avgLatency/float64(wh.numThread)),
		zap.Float64s("latencies", allLatencies))

	// Return the aggregated results
	return results.Results{
		AverageLatency: avgLatency / float64(wh.numThread),
		Throughput:     avgThroughput / float64(wh.numThread),
		TxLatencies:    allLatencies,
	}
}

// Close the clients and the channels
func (wh *WorkloadHandler) CloseAll() {
	for _, c := range wh.activeClients {
		c.Close()
	}
}
