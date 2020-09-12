package parsers

import (
	"diablo-benchmark/core/configs"
	"gopkg.in/yaml.v3"
	"io/ioutil"
)

// ParseChainConfig parses the chain configuration file.
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

// parseChainYaml parses the chain configuration in the YAML files.
// This will get the bytes of the file.
func parseChainYaml(fileContents []byte) (*configs.ChainConfig, error) {
	var chainConfig configs.ChainConfig
	err := yaml.Unmarshal(fileContents, &chainConfig)

	if err != nil {
		return nil, err
	}

	return &chainConfig, nil
}
