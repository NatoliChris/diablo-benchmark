package clientinterfaces

import (
	"diablo-benchmark/core/configs"
	"errors"
)

func GetBlockchainInterface(config *configs.ChainConfig) (BlockchainInterface, error) {
	switch config.Name {
	case "ethereum":
		bci := EthereumInterface{}
		return &bci, nil
	default:
		return nil, errors.New("unsupported blockchain in chain config")
	}
}
