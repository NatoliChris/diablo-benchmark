package parsers

import (
	"diablo-benchmark/core/configs"
	"gopkg.in/yaml.v3"
	"io/ioutil"
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
	intervalKeys := make([]int, 0)
	for k := range intervals {
		intervalKeys = append(intervalKeys, k)
	}

	// make the list of transaction intervals
	finalIntervals := make(map[int]int, intervalKeys[len(intervalKeys)-1])

	prev := intervalKeys[0]
	for _, v := range intervalKeys[1:] {

	}

	return finalIntervals, nil
}
