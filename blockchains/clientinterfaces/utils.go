package clientinterfaces

import (
	"diablo-benchmark/core/configs"
	"errors"
)

// GetBlockchainInterface maps the name of the blockchain in the config with the interface to implement.
// Is used by the clients to select the correct chain configuration
func GetBlockchainInterface(config *configs.ChainConfig) (BlockchainInterface, error) {
	switch config.Name {
	case "ethereum":
		bci := EthereumInterface{}
		return &bci, nil
	case "fabric":
		bci := FabricInterface{}
		return &bci,nil

	case "diem":
		bci := DiemInterface{}
		return &bci,nil

	default:
		return nil, errors.New("unsupported blockchain in chain config")
	}
}
