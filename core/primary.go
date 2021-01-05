// Package core provides the critical handlers and structures that run the
// benchmark. This includes the code for the primary and secondary nodes as well
// as any major handlers (workload, results, etc.). This code should not be
// required to be augmented when integrating a new blockchain or adding
// new workloads.
package core

import (
	"diablo-benchmark/blockchains/workloadgenerators"
	"diablo-benchmark/communication"
	"diablo-benchmark/core/configs"
	"diablo-benchmark/core/results"
	"fmt"
	"time"

	"go.uber.org/zap"
)

// Primary benchmark server, acts as the orchestrator for the benchmark
type Primary struct {
	Server            *communication.PrimaryServer         // TCP server identified with the primary for all secondaries to connect to
	workloadGenerator workloadgenerators.WorkloadGenerator // Workload generator implementation that will generate the transactions
	benchmarkConfig   *configs.BenchConfig                 // Benchmark configuration about the workload
	chainConfig       *configs.ChainConfig                 // Chain configuration containing information about the nodes
}

// InitPrimary initialises the primary server and returns an instance of the primary
// This will be passed back to the main
func InitPrimary(listenAddr string, expectedSecondaries int, wg workloadgenerators.WorkloadGenerator, bConfig *configs.BenchConfig, cConfig *configs.ChainConfig) *Primary {
	s, err := communication.SetupPrimaryTCP(listenAddr, expectedSecondaries)
	if err != nil {
		// TODO remove panic
		panic(err)
	}

	// Return a new primary instance with the active communication set up
	return &Primary{
		Server:            s,
		workloadGenerator: wg,
		benchmarkConfig:   bConfig,
		chainConfig:       cConfig,
	}
}

// closeAllConns closes all connections and exits
func (p *Primary) closeAllConns() {
	p.Server.CloseSecondaries()
	p.Server.Close()
}

// Run provides the main functionality to run
// Holds the majority of the work
// TODO: under construction!
func (p *Primary) Run() {
	// First, set up the blockchain
	err := p.workloadGenerator.BlockchainSetup()

	if err != nil {
		zap.L().Error("encountered error with blockchain setup",
			zap.String("error", err.Error()))
		return
	}

	// Next, init the workload generator
	err = p.workloadGenerator.InitParams()
	if err != nil {
		zap.L().Error("encountered error with workloadgenerator InitParams",
			zap.String("error", err.Error()))
		return
	}

	// Get the secondary connections ready
	secondaryReadyChannel := make(chan bool, 1)
	go p.Server.HandleSecondaries(secondaryReadyChannel)
	<-secondaryReadyChannel
	close(secondaryReadyChannel)

	// Parse the config files
	// Run all preparation

	// Run through the benchmark suite
	// Step 1: send "PREPARE" to secondaries, make sure we can communicate.
	errs := p.Server.PrepareBenchmarkSecondaries(uint32(p.benchmarkConfig.Threads))

	if errs != nil {
		// We have errors
		p.closeAllConns()
		zap.L().Error("Encountered errors in secondaries",
			zap.Strings("errors", errs))
		return
	}

	// Number of secondaries connected
	zap.L().Info("Benchmark secondaries all connected.",
		zap.Int("secondaries", len(p.Server.Secondaries)))

	// Step 3: Prepare the workload for the benchmark
	// TODO: generate workloads
	workload, err := p.workloadGenerator.GenerateWorkload()

	if err != nil {
		zap.L().Error("failed to generate workload",
			zap.String("error", err.Error()))
		p.closeAllConns()
		return
	}

	// Step 4: Distribute benchmark
	errs = p.Server.SendWorkload(workload)
	if errs != nil {
		zap.L().Error("Encountered Error sending workload",
			zap.String("errs", fmt.Sprintf("%v", errs)),
		)
		p.closeAllConns()
		return
	}

	// Step 5: run the bench
	errs = p.Server.RunBenchmark()
	if errs != nil {
		zap.L().Error("Encountered Error sending workload",
			zap.String("errs", fmt.Sprintf("%v", errs)),
		)
		p.closeAllConns()
		return
	}

	// Wait until everyone is done and give some room for final messages
	time.Sleep(2 * time.Second)

	// Step 6 (once all have completed) - get the results
	// TODO: Need to store the results
	rawResults, errs := p.Server.GetResults()
	if errs != nil {
		zap.L().Error("GetResults returned client errors",
			zap.Strings("errors", errs))
		p.closeAllConns()
	}

	// TODO: @CHRIS
	aggregatedResults := results.CalculateAggregatedResults(rawResults)

	// Step 7 - store results
	p.Server.SendFin()

	time.Sleep(2 * time.Second)

	// Display the results
	results.Display(aggregatedResults)
	// Write the results to a file
	err = results.WriteResultsToFile(p.benchmarkConfig.Path, p.chainConfig.Path, aggregatedResults, "results")
	if err != nil {
		zap.L().Error("Encountered error when saving results",
			zap.Error(err))
	}

	// Step 8: Close all connections
	p.Server.CloseSecondaries()
	p.Server.Close()
}
