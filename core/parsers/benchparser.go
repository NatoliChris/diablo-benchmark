package parsers

import (
	"diablo-benchmark/core/configs"
	"errors"
	"gopkg.in/yaml.v3"
	"io/ioutil"
	"sort"
)

// Parse the benchmark configuration file, read the filepath to see
// if we can extract the YAML.
// TODO: proper error handling
func ParseBenchConfig(filepath string) (*configs.BenchConfig, error) {
	// Get the configuration information from the filepath
	configFileBytes, err := ioutil.ReadFile(filepath)

	if err != nil {
		return nil, err
	}

	return parseBenchYaml(configFileBytes)
}

// Unmarshal the YAML into the required context.
// TODO: proper error handling
func parseBenchYaml(content []byte) (*configs.BenchConfig, error) {
	// Try to read the YAML.
	var benchConfig configs.BenchConfig

	err := yaml.Unmarshal(content, &benchConfig)

	if err != nil {
		return nil, err
	}

	// Generate the intervals from the benchmark config
	fullIntervals, err := generateFullIntervals(benchConfig.TxInfo.Intervals)

	if err != nil {
		return nil, err
	}

	// Add the full intervals into the benchmark configurations
	benchConfig.TxInfo.Intervals = fullIntervals

	return &benchConfig, nil
}

// Fills the intervals into all seconds defined.
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
	// TODO: this needs improvement, big time!
	currentKey := intervalKeys[0]
	for _, nextKey := range intervalKeys[1:] {
		// Note: extremely naive linear ramp-up.
		// Next value - current value / number of intervals between keys.
		// e.g 10sec=30tps, 40sec=100tps; increment_val = (100-30) / (40-10) => 2.33333 increment per second.

		numberOfIntervals := nextKey - currentKey
		startTPS := intervals[currentKey]
		endTPS := intervals[nextKey]

		incrementValue := (endTPS - startTPS) / numberOfIntervals

		currentTPS := startTPS
		for i := currentKey; i < nextKey; i++ {
			finalIntervals[i] = currentTPS
			currentTPS += incrementValue
		}

		currentKey = nextKey
	}

	return finalIntervals, nil
}
