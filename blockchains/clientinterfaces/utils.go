package clientinterfaces

import (
	"diablo-benchmark/blockchains/algorand"
	"diablo-benchmark/blockchains/diem"
	"diablo-benchmark/core/configs"
	"fmt"
)

// GetBlockchainInterface maps the name of the blockchain in the config with the interface to implement.
// Is used by the clients to select the correct chain configuration
func GetBlockchainInterface(config *configs.ChainConfig) (BlockchainInterface, error) {
	switch config.Name {
	case "algorand":
		bci := NewWorkerBridge(algorand.NewWorker())
		return bci, nil
	case "diem":
		bci := NewWorkerBridge(diem.NewWorker())
		return bci, nil
	case "ethereum":
		bci := EthereumInterface{}
		return &bci, nil
	case "fabric":
		bci := FabricInterface{}
		return &bci, nil
	case "solana":
		bci := NewSolanaInterface()
		return bci, nil
	default:
		return nil, fmt.Errorf("unsupported blockchain '%s' in chain config", config.Name)
	}
}
