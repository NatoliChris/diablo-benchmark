package parsers

import (
	"diablo-benchmark/blockchains/workloadgenerators"
	"diablo-benchmark/core/configs"
	"errors"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
	"io/ioutil"
)

// Parse the chain configuration file.
// This function both (a) reads the file from disk, and (b) calls the YAML
// to be parsed.
func ParseChainConfig(filePath string) (*configs.ChainConfig, error) {

	// Get the bytes of the file
	configFileBytes, err := ioutil.ReadFile(filePath)

	if err != nil {
		return nil, err
	}

	return parseChainYaml(configFileBytes)
}

// Parse the chain configuration in the YAML files.
// This will get the bytes of the file.
func parseChainYaml(fileContents []byte) (*configs.ChainConfig, error) {
	var chainConfig configs.ChainConfig
	err := yaml.Unmarshal(fileContents, &chainConfig)

	if err != nil {
		return nil, err
	}

	return &chainConfig, nil
}

func GetWorkloadGenerator(config *configs.ChainConfig) (workloadgenerators.WorkloadGenerator, error) {
	switch config.Name {
	case "ethereum":
		// Return the ethereum workload generator
		// TODO get the type of the ethereum workload generator
		var wg workloadgenerators.WorkloadGenerator
		wg = &workloadgenerators.EthereumWorkloadGenerator{}
		return wg, nil

	default:
		zap.L().Warn("unknown chain defined in config",
			zap.String("chain_name", config.Name))
		return nil, errors.New("unknown chain when parsing config")
	}
}
