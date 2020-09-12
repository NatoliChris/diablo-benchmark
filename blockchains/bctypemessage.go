package blockchains

// TODO implement this module
// This functionality was designed to provide a generic "diablo secondary" that runs on a machine, but is not tied
// to a blockchain, but can dynamically change the blockchain. This allows for a secondary to run multiple tests
// for different blockchains without having to start the secondary again.
// Just a potential ease-of-use feature, nothing more than convenience, but most likely a bad idea.

import (
	"diablo-benchmark/blockchains/clientinterfaces"
	"errors"
	"go.uber.org/zap"
	"strings"
)

// BlockchainTypeMessage represents the type of blockchain during communication.
type BlockchainTypeMessage byte

const (
	// BCEthereum defines the message to use the Ethereum blockchain.
	BCEthereum BlockchainTypeMessage = '\xb0'
)

// MatchStringToMessage Matches the name in the configuration to the blockchain
func MatchStringToMessage(configBCType string) (BlockchainTypeMessage, error) {
	switch strings.ToLower(configBCType) {
	case "ethereum":
		return BCEthereum, nil
	default:
		return '\x00', errors.New("Blockchain not supported")
	}
}

// MatchMessageToInterface Matches the byte received from the primary to the
// interface that is required to interact with the blockchain system we are
// benchmarking.
func MatchMessageToInterface(msg byte) (clientinterfaces.BlockchainInterface, error) {
	switch BlockchainTypeMessage(msg) {
	case BCEthereum:
		zap.L().Info("Match message to Ethereum blockchain")
		return &clientinterfaces.EthereumInterface{}, nil
	default:
		return nil, errors.New("unsupported blockchain")
	}
}
