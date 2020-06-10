package blockchains

import (
	"errors"
	"strings"
)

type BlockchainTypeMessage byte

const (
	BCEthereum BlockchainTypeMessage = '\xb0'
)

// Matches the name in the configuration to the blockchain
func MatchStringToMessage(configBCType string) (BlockchainTypeMessage, error) {
	switch strings.ToLower(configBCType) {
	case "ethereum":
		return BCEthereum, nil
	default:
		return '\x00', errors.New("Blockchain not supported")
	}
}

// Matches the byte received from the master to the interface that is required
// to interact with the blockchain system we are benchmarking.
func MatchMessageToInterface(msg byte) {
	// TODO implement
}
