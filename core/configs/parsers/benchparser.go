// Package parsers presents the parsing of configuration files, which will
// parse and generate the related information necessary for the use in the
// benchmark file
package parsers

import (
	"diablo-benchmark/core/configs"
	"diablo-benchmark/core/configs/validators"
	"errors"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"math"
	"sort"
)

// ParseBenchConfig parses the benchmark configuration file from YAML.
// Reads the filepath to see if we can extract the YAML.
// TODO: proper error handling
func ParseBenchConfig(filepath string) (*configs.BenchConfig, error) {
	// Get the configuration information from the filepath
	configFileBytes, err := ioutil.ReadFile(filepath)

	if err != nil {
		return nil, err
	}

	return parseBenchYaml(configFileBytes, filepath)
}

// parseBenchYaml provides the full unmarshal of the YAML and performs relevant calculations
// TODO: proper error handling
func parseBenchYaml(content []byte, path string) (*configs.BenchConfig, error) {
	// Try to read the YAML.
	var benchConfig configs.BenchConfig

	err := yaml.Unmarshal(content, &benchConfig)

	if err != nil {
		return nil, err
	}

	// Check validity
	if ok, err := validators.ValidateBenchConfig(&benchConfig); !ok {
		return nil, err
	}

	// Generate the intervals from the benchmark config
	fullIntervals, err := generateFullIntervals(benchConfig.TxInfo.Intervals)

	if err != nil {
		return nil, err
	}

	// Add the full intervals into the benchmark configurations
	benchConfig.TxInfo.Intervals = fullIntervals

	benchConfig.Path = path

	return &benchConfig, nil
}

// generateFullIntervals fills the intervals into all seconds defined.
// NOTE: this is only a naive implementation, will definitely require more work ;)
// TODO: more complex generation of transaction intervals.
func generateFullIntervals(intervals configs.TPSIntervals) (configs.TPSIntervals, error) {

	if len(intervals) == 0 {
		return nil, errors.New("empty intervals found in benchmark")
	}

	intervalKeys := make([]int, 0)
	for k := range intervals {
		intervalKeys = append(intervalKeys, k)
	}

	// Sort
	sort.Ints(intervalKeys)

	// Check that it starts at 0
	// TODO: If the values don't start at 0, do we start at the next rate or 0?
	// NOTE: I'm going to go with 0, since it seems logical for ramp-up. People can define
	// Their own start with a 0 index if they want.
	if _, ok := intervals[0]; !ok {
		// if 0 doesn't exist, we need it to.
		intervalKeys = append([]int{0}, intervalKeys...)
		intervals[0] = 0
	}

	// make the list of transaction intervals
	finalIntervals := make(configs.TPSIntervals, intervalKeys[len(intervalKeys)-1])

	// Go through each interval
	// Fill in the gaps by calculating a linear ramp-up.
	// TODO: this needs improvement, big time! Currently doing linear, but maybe we can have a smoothing curve?
	currentKey := intervalKeys[0]
	for _, nextKey := range intervalKeys[1:] {
		// Note: extremely naive linear ramp-up.
		// Next value - current value / number of intervals between keys.
		// e.g 10sec=30tps, 40sec=100tps; increment_val = (100-30) / (40-10) => 2.33333 increment per second.

		numberOfIntervals := nextKey - currentKey
		startTPS := intervals[currentKey]
		endTPS := intervals[nextKey]

		incrementValue := float64(endTPS-startTPS) / float64(numberOfIntervals)

		currentTPS := float64(startTPS)
		for i := currentKey; i < nextKey; i++ {
			finalIntervals[i] = int(math.Floor(currentTPS))
			currentTPS += incrementValue
		}

		currentKey = nextKey
	}

	finalIntervals[intervalKeys[len(intervalKeys)-1]] = intervals[intervalKeys[len(intervalKeys)-1]]

	return finalIntervals, nil
}

// GetTotalNumberOfTransactions calculates the total number of transactions in the entire benchmark
func GetTotalNumberOfTransactions(config *configs.BenchConfig) (int, error) {
	totalNumberOfTransactions := 0

	for _, v := range config.TxInfo.Intervals {
		totalNumberOfTransactions += v
	}

	return totalNumberOfTransactions, nil
}
