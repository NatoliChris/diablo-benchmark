// Package validators provides the validation for configurations
// With more complex chain and benchmark configurations, the validators should
// verify the existence of core fields and ensure that the correct definitions
// are available. This does NOT verify that the YAML is well formed, that is
// performed through the decoding of the yaml in the parsers
package validators

import (
	"diablo-benchmark/core/configs"
	"errors"
	"fmt"
	"go.uber.org/zap"
	"os"
	"reflect"
)

// ValidateBenchConfig validates all fields of the benchmark configuration
// Determines the validity and returns a boolean whether it is
// valid or invalid.
// TODO major improvements for validations!
func ValidateBenchConfig(c *configs.BenchConfig) (bool, error) {
	// Empty name is an error
	if len(c.Name) == 0 {
		return false, errors.New("missing benchmark name")
	}

	// Description can be omitted, but we will warn.
	if len(c.Description) == 0 {
		zap.L().Warn("Missing description in configuration file.")
	}

	// Contract Checks!
	if c.TxInfo.TxType == configs.TxTypeContract {
		// Check if it's empty
		if reflect.DeepEqual(configs.ContractInfo{}, c.ContractInfo) {
			return false, fmt.Errorf("[%s] empty contract info for contract workload in", c.Name)
		}

		// Check that the contract exists.
		if c.ContractInfo.Path == "" {
			return false, fmt.Errorf("[%s] empty path for contract in config", c.Name)
		}

		info, err := os.Stat(c.ContractInfo.Path)
		if err != nil {
			return false, err
		}

		// If it is a directory - then error
		if info.IsDir() {
			return false, fmt.Errorf("[%s] contract path (%s) is a directory", c.Name, c.ContractInfo.Path)
		}

		// Check that the functions aren't empty.
		if len(c.ContractInfo.Functions) == 0 {
			return false, fmt.Errorf("[%s] no functions provided for contract", c.Name)
		}
	}

	// Intervals cannot be empty.
	if len(c.TxInfo.Intervals) == 0 {
		return false, errors.New("no tps intervals provided")
	}

	// Check that there are no negative values.
	for k := range c.TxInfo.Intervals {
		if k < 0 {
			return false, fmt.Errorf("tps key %d cannot be negative", k)
		}

		if c.TxInfo.Intervals[k] < 0 {
			return false, fmt.Errorf(
				"tps value %d at key %d cannot be negative",
				c.TxInfo.Intervals[k],
				k)
		}
	}

	if c.Secondaries <= 0 {
		return false, fmt.Errorf("number of secondaries must be minimum 1")
	}

	if c.Threads <= 0 {
		return false, fmt.Errorf("number of threads must be minimum 1")
	}

	return true, nil
}
