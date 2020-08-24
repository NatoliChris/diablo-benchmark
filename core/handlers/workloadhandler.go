package handlers

import (
	"diablo-benchmark/blockchains/clientinterfaces"
	"diablo-benchmark/blockchains/workloadgenerators"
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
	FullWorkload         [][][]interface{}                      // Workload
	readyChannels        []chan bool                            // channels that signal ready to start
	wg                   *sync.WaitGroup                        // All threads done
	numTx                uint64                                 // number of transactions sent
	numErrors            uint64                                 // Number of errors during workload
	StartEnd             []time.Time                            // Start and end of the benchmark
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
func (wh *WorkloadHandler) ParseWorkloads(rawWorkload workloadgenerators.ClientWorkload) error {

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

// Worker producer, handles the transaction per second producing into the queue.
func (wh *WorkloadHandler) workloadProducer(workload [][]interface{}, workerChan chan interface{}, ready chan bool, id int) {
	zap.L().Debug(fmt.Sprintf("producer %d ready", id))
	<-ready
	currentIterator := 1
	for _, v := range workload[0] {
		workerChan <- v
	}

	// Set up the timer
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			for _, v := range workload[currentIterator] {
				workerChan <- v
			}
			currentIterator++
			if currentIterator >= len(workload) {
				close(workerChan)
				return
			}
		}
	}

}

// Runner for the thread
func (wh *WorkloadHandler) runnerConsumer(blockchainInterface clientinterfaces.BlockchainInterface, workload chan interface{}, wg *sync.WaitGroup) {
	var errs []error
	defer wg.Done()

	// Wait for the signal to go
	for tx := range workload {
		e := blockchainInterface.SendRawTransaction(tx)
		if e != nil {
			errs = append(errs, e)
		}
		atomic.AddUint64(&wh.numTx, 1)
	}

	// TODO handle errors
}

// Periodically adds the workloads to the channels, allowing for a "curve" of
// transactions to be sent (rate-limiting the sending of transactions)
func (wh *WorkloadHandler) txAdder(workloads chan interface{}, ready chan bool) {
	// TODO - channel for every second, adds more to the workload channel one
	// at a time - then checks if the channel has ended.
	// Changes to implement
	// - workload is now [][][]byte - allows for per-second intervals
	// - blockchain interface must take ^ for parseworkload
	// - sending / receiving the workload through primary must also be handled
	// - workload generation must return [][][]byte
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
	wh.StartEnd = append(wh.StartEnd, time.Now())
	stopPrinting := make(chan bool, 0)

	go wh.statusPrinter(stopPrinting)

	for _, ch := range wh.readyChannels {
		ch <- true
	}

	wh.wg.Wait()
	stopPrinting <- true

	wh.StartEnd = append(wh.StartEnd, time.Now())
	zap.L().Info("Benchmark complete:",
		zap.Time("start", wh.StartEnd[0]),
		zap.Time("end", wh.StartEnd[1]),
		zap.Duration("duration", wh.StartEnd[1].Sub(wh.StartEnd[0])))
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
		zap.Float64("avg throughput", avgThroughput),
		zap.Float64("avg latency", avgLatency/float64(wh.numThread)),
		zap.Float64s("latencies", allLatencies))

	// Return the aggregated results
	return results.Results{
		AverageLatency: avgLatency / float64(wh.numThread),
		Throughput:     avgThroughput,
		TxLatencies:    allLatencies,
	}
}

// Close the clients and the channels
func (wh *WorkloadHandler) CloseAll() {
	for _, c := range wh.activeClients {
		c.Close()
	}
}
