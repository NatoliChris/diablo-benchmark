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
