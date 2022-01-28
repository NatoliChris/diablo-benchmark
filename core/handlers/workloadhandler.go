// Package handlers provides the core handlers within the benchmark.
package handlers

import (
	"diablo-benchmark/blockchains/clientinterfaces"
	"diablo-benchmark/blockchains/workloadgenerators"
	"diablo-benchmark/core/configs"
	"diablo-benchmark/core/results"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
)

// WorkloadHandler is the main handler loop that dispatches the workload into channels and creates routines that will read and send
type WorkloadHandler struct {
	numThread            uint32                                 // Number of workers
	workerThreadChannels []chan interface{}                     // Channels between the threads to have the workload from
	activeClients        []clientinterfaces.BlockchainInterface // Number of client threads that are run
	FullWorkload         [][][]interface{}                      // Workload
	readyChannels        []chan bool                            // channels that signal ready to start
	wg                   *sync.WaitGroup                        // All threads done
	numTx                uint64                                 // number of transactions sent
	numErrors            uint64                                 // Number of errors during workload
	StartEnd             []time.Time                            // Start and end of the benchmark
	timeout              int                                    // Timeout to wait for the benchmark
}

// NewWorkloadHandler provides a new workload handler with number of threads and clients
func NewWorkloadHandler(numThread uint32, clients []clientinterfaces.BlockchainInterface, timeout int) *WorkloadHandler {
	// Generate the channels to speak to the workers.
	return &WorkloadHandler{
		numThread:     numThread,
		activeClients: clients,
		timeout:       timeout,
	}
}

// Connect initialises the clients and connects to the nodes
func (wh *WorkloadHandler) Connect(chainConfig *configs.ChainConfig, ID int) error {
	var combinedErr []string
	for _, v := range wh.activeClients {
		v.Init(chainConfig)
		v.SetWindow(chainConfig.ThroughputWindow)
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

// ParseWorkloads parse the workloads on each client, populate the channels
func (wh *WorkloadHandler) ParseWorkloads(rawWorkload workloadgenerators.SecondaryWorkload) error {

	// Set up the workload channels
	var readyChannels []chan bool
	var wg sync.WaitGroup

	var fullWorkload [][][]interface{}

	for i, workerWorkload := range rawWorkload {
		// Should be able to parse the workloads from transactions into bytes
		parsedWorkerWorkload, err := wh.activeClients[0].ParseWorkload(workerWorkload)
		if err != nil {
			return err
		}

		channelSize := 0
		for _, v := range parsedWorkerWorkload {
			channelSize += len(v)
		}

		readyChannel := make(chan bool, 0)
		readyChannels = append(readyChannels, readyChannel)

		workerChannel := make(chan interface{}, channelSize)
		wg.Add(1)
		// Make my consumer
		go wh.runnerConsumer(
			wh.activeClients[i],
			workerChannel,
			&wg,
		)

		// Start the worker producer
		go wh.workloadProducer(
			parsedWorkerWorkload,
			workerChannel,
			readyChannel,
			i,
		)

		fullWorkload = append(fullWorkload, parsedWorkerWorkload)
	}

	wh.FullWorkload = fullWorkload
	wh.readyChannels = readyChannels
	wh.wg = &wg
	return nil
}

// workloadProducer producer that places transactions into the queue and handles naive rate limiting.
func (wh *WorkloadHandler) workloadProducer(workload [][]interface{}, workerChan chan interface{}, ready chan bool, id int) {
	zap.L().Debug(fmt.Sprintf("producer %d ready", id))
	<-ready
	// Set up the timer
	ticker := time.NewTicker(1 * time.Second)
	batchSize := 10
	for i := 0; i < len(workload); _, i = <-ticker.C, i+1 {
		innerTicker := time.NewTicker(1 * time.Second / (time.Duration(len(workload[i])) / time.Duration(batchSize)))
		for j := 0; j < len(workload[i]); <-innerTicker.C {
			for k := 0; j < len(workload[i]) && k < batchSize; j, k = j+1, k+1 {
				workerChan <- workload[i][j]
			}
		}
		innerTicker.Stop()
	}
	ticker.Stop()
	close(workerChan)
}

// runnerConsumer consumer that runs the workload pulling from the channel
func (wh *WorkloadHandler) runnerConsumer(blockchainInterface clientinterfaces.BlockchainInterface, workload chan interface{}, wg *sync.WaitGroup) {
	var errs []error
	defer wg.Done()

	// Wait for the signal to go
	for tx := range workload {
		e := blockchainInterface.SendRawTransaction(tx)
		if e != nil {
			zap.L().Debug("Error sending tx",
				zap.Error(e))
			errs = append(errs, e)
			atomic.AddUint64(&wh.numErrors, 1)
		}
		atomic.AddUint64(&wh.numTx, 1)
	}

	// TODO handle errors
}

// statusPrinter periodically prints the status of the workload progress
func (wh *WorkloadHandler) statusPrinter(stopCh chan bool) {
	timer := time.NewTicker(5 * time.Second)
	for {
		select {
		case <-stopCh:
			timer.Stop()
			return
		case <-timer.C:
			// print
			zap.L().Info(fmt.Sprintf("SENT: %d tx (%d errors)", wh.numTx, wh.numErrors))
		}
	}
}

func (wh *WorkloadHandler) getTxCheck() uint64 {
	fullTx := uint64(0)
	for _, v := range wh.activeClients {
		fullTx += v.GetTxDone()
	}

	return fullTx
}

// RunBench executes the benchmark
func (wh *WorkloadHandler) RunBench() error {
	wh.StartEnd = append(wh.StartEnd, time.Now())
	stopPrinting := make(chan bool, 0)

	go wh.statusPrinter(stopPrinting)

	for i, ch := range wh.readyChannels {
		wh.activeClients[i].Start()
		ch <- true
	}

	// All of the threads have stopped sending, we should wait some time for
	// confirmations
	wh.wg.Wait()

	// Sending finished waiting for timeout
	// TODO: add configurable timeout that will exit if benchmark not complete
	zap.L().Info("Sending complete, waiting for finish")
	stopPrinting <- true

	waitingTicker := time.NewTicker(1 * time.Second)
	waitCount := 0
	td := uint64(0)
	for {
		select {
		case <-waitingTicker.C:
			waitCount++
			td = wh.getTxCheck()
			zap.L().Debug("TX Done:",
				zap.Uint64("tx", td),
				zap.Uint64("total", wh.numTx),
			)
			break
		}
		if waitCount >= wh.timeout || (td/wh.numTx) == 1 {
			break
		}
	}

	wh.StartEnd = append(wh.StartEnd, time.Now())

	zap.L().Info("Benchmark complete:",
		zap.Time("start", wh.StartEnd[0]),
		zap.Time("end", wh.StartEnd[1]),
		zap.Duration("duration", wh.StartEnd[1].Sub(wh.StartEnd[0])))
	// TODO get errors
	// add error channel to runner so that it can append the errors
	return nil
}

// HandleCleanup performs all post-benchmark calculation and returns the result set
func (wh *WorkloadHandler) HandleCleanup() []results.Results {

	var resList []results.Results
	for _, c := range wh.activeClients {
		resList = append(resList, c.Cleanup())
	}

	zap.L().Debug("Results being returned",
		zap.Int("len", len(resList)))

	fmt.Println(resList)
	return resList
}

// CloseAll closes the clients and the channels
func (wh *WorkloadHandler) CloseAll() {
	for _, c := range wh.activeClients {
		c.Close()
	}
}
