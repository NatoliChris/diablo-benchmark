package validators

import (
	"diablo-benchmark/core/configs"
	"errors"
	"fmt"
	"go.uber.org/zap"
)

// Validates all fields of the benchmark configuration
// Determines the validity and returns a boolean whether it is
// valid or invalid.
func ValidateBenchConfig(c *configs.BenchConfig) (bool, error) {
	// Empty name is an error
	if len(c.Name) == 0 {
		return false, errors.New("missing benchmark name")
	}

	// Description can be omitted, but we will warn.
	if len(c.Description) == 0 {
		zap.L().Warn("Missing description in configuration file.")
	}

	// Intervals cannot be empty.
	if len(c.TxInfo.Intervals) == 0 {
		return false, errors.New("no tps intervals provided")
	}

	// Check that there are no negative values.
	for k := range c.TxInfo.Intervals {
		if k < 0 {
			return false, errors.New(fmt.Sprintf("tps key %d cannot be negative", k))
		}

		if c.TxInfo.Intervals[k] < 0 {
			return false, errors.New(
				fmt.Sprintf(
					"tps value %d at key %d cannot be negative",
					c.TxInfo.Intervals[k],
					k))
		}
	}

	return true, nil
}
