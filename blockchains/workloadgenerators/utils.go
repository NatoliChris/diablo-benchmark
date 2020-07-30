package workloadgenerators

import (
	"diablo-benchmark/core/configs"
	"errors"
	"go.uber.org/zap"
)

func GetWorkloadGenerator(config *configs.ChainConfig) (WorkloadGenerator, error) {
	switch config.Name {
	case "ethereum":
		// Return the ethereum workload generator
		// TODO get the type of the ethereum workload generator
		var wg WorkloadGenerator
		wg = &EthereumWorkloadGenerator{}
		return wg, nil

	default:
		zap.L().Warn("unknown chain defined in config",
			zap.String("chain_name", config.Name))
		return nil, errors.New("unknown chain when parsing config")
	}
}
