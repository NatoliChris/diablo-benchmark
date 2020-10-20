package parsers

import (
	"diablo-benchmark/core/configs"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
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

	return parseChainYaml(configFileBytes, filePath)
}

// parseKeyFile is the dedicated function to read "{privateKey, address}" pairs from a
// JSON file.
// NOTE: in future we can have a switch on the type of file and have it read from different inputs.
func parseKeyFile(path string) ([]configs.ChainKey, error) {
	// Check that the file exists
	_, fErr := os.Stat(path)
	if os.IsNotExist(fErr) {
		return nil, fmt.Errorf("Key file does not exist: %s", path)
	}

	fileType := strings.Split(path, ".")

	var keys []configs.ChainKey
	var err error

	switch fileType[len(fileType)-1] {
	case "json", "JSON":
		// filetype is json
		b, bErr := ioutil.ReadFile(path)

		if bErr != nil {
			return nil, bErr
		}

		err = json.Unmarshal(b, &keys)
	case "yaml", "yml", "YAML", "YML":
		// filetype is yaml
		b, bErr := ioutil.ReadFile(path)

		if bErr != nil {
			return nil, bErr
		}

		err = yaml.Unmarshal(b, &keys)
	default:
		// unsupported
		return nil, fmt.Errorf("Unsupported filetype for key file: %s", fileType[len(fileType)-1])
	}

	return keys, err
}

// parseChainYaml parses the chain configuration in the YAML files.
// This will get the bytes of the file.
func parseChainYaml(fileContents []byte, path string) (*configs.ChainConfig, error) {
	var chainConfig configs.ChainConfig
	err := yaml.Unmarshal(fileContents, &chainConfig)

	if err != nil {
		return nil, err
	}

	chainConfig.Path = path

	// Check if there is the "keys_file" and take that as preference.
	if chainConfig.KeyFile != "" {
		// parse key file
		kf, err := parseKeyFile(chainConfig.KeyFile)
		if err != nil {
			return nil, err
		}

		chainConfig.Keys = kf
	}

	return &chainConfig, nil
}
