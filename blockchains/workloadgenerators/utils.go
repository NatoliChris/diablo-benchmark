package workloadgenerators

import (
	"errors"
	"math"
	"math/rand"
	"time"

	"go.uber.org/zap"

	"diablo-benchmark/blockchains/algorand"
	"diablo-benchmark/core/configs"
)

// GetWorkloadGenerator matches the workload generator with the name provided of the chain in the configuration
// If there is no match, there is an error returned. This is defined in the
// chain configuration.
func GetWorkloadGenerator(config *configs.ChainConfig) (WorkloadGenerator, error) {
	var wg WorkloadGenerator

	switch config.Name {
	case "algorand":
		wg = NewControllerBridge(algorand.NewController())
	case "ethereum":
		// Return the ethereum workload generator
		// TODO get the type of the ethereum workload generator
		wg = &EthereumWorkloadGenerator{}
	case "fabric":
		wg = &FabricWorkloadGenerator{}
	case "solana":
		wg = NewSolanaWorkloadGenerator()
	default:
		zap.L().Warn("unknown chain defined in config",
			zap.String("chain_name", config.Name))
		return nil, errors.New("unknown chain when parsing config")
	}

	return wg, nil
}

// ShuffleFunctionCalls shuffles the function calls to interleave execution
func ShuffleFunctionCalls(functionsToGet []int) {
	// start with a source of randomness
	randomness := rand.New(rand.NewSource(time.Now().Unix()))

	// In-place shuffle
	for len(functionsToGet) > 0 {
		currentN := len(functionsToGet)
		randIndex := randomness.Intn(currentN)
		functionsToGet[currentN-1], functionsToGet[randIndex] = functionsToGet[randIndex], functionsToGet[currentN-1]
		functionsToGet = functionsToGet[:currentN-1]
	}
}

// GetIntervalPerThread generates the number of transactions per thread per interval to create for the workload, this is a utility function to help generate the numbers
func GetIntervalPerThread(tpsIntervals configs.TPSIntervals, secondaries int, threads int) []int {
	// get the total number of threads to divide the intervals by
	totalThreads := secondaries * threads

	// loop through the intervals, then create each one
	txPerThread := make([]int, len(tpsIntervals))
	for i, v := range tpsIntervals {
		txPerThread[i] = int(math.Ceil(float64(v) / float64(totalThreads)))
		zap.L().Debug("tps interval",
			zap.Int("total", v),
			zap.Int("thread", txPerThread[i]),
		)
	}
	return txPerThread
}
