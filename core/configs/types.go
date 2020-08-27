package configs

import (
	"encoding/hex"
	"errors"
)

// Transaction types (simple, contract, ...)
type BenchTransactionType string

// Simple transaction type, denoting just an asset transfer transaction.
const TxTypeSimple BenchTransactionType = "simple"

// Contract type, indicates that it will be a complex worklaod that has
// contract interaction and deployment
const TxTypeContract BenchTransactionType = "contract"

// Transactions Per Second intervals
type TPSIntervals map[int]int

// Basic "key" structure for generic blockchains, provides a private key in
// bytes, and the "Address" as a string - the address may change depending on
// the blockchain.
type ChainKey struct {
	PrivateKey []byte `yaml:"private"` // Private key information
	Address    string `yaml:address`   // Address that it is from
}

// Naive check if the prefixed PrivateKey has "0x" leading.
func checkPrefix(keyHex string) bool {
	return len(keyHex) >= 2 && // Length must be 0x or more
		keyHex[0] == '0' && // Starts with 0
		(keyHex[1] == 'x' || keyHex[1] == 'X') // followed by an x or X
}

// Custom YAML decoding for the chain key.
func (ck *ChainKey) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var c struct {
		PrivateKey string `yaml:"private"`
		Address    string `yaml:"address"`
	}
	err := unmarshal(&c)

	if err != nil {
		return err
	}

	if len(c.PrivateKey) == 0 {
		return errors.New("empty PrivateKey passed to unmarshal")
	}

	var privateKeyBytes []byte

	if checkPrefix(c.PrivateKey) {
		// If the prefix exists, decode from [2:]
		privateKeyBytes, err = hex.DecodeString(c.PrivateKey[2:])
		// If we couldn't decode
		if err != nil {
			return err
		}
	} else {
		privateKeyBytes, err = hex.DecodeString(c.PrivateKey)
		// If we couldn't decode
		if err != nil {
			return err
		}
	}

	(*ck).PrivateKey = privateKeyBytes
	(*ck).Address = c.Address

	return nil
}

// Custom YAML Decode for a "bench transaction" type. (Simple / Contract)
func (bt *BenchTransactionType) UnmarshalYAML(unmarshal func(interface{}) error) error {
	var unmarshaled string

	err := unmarshal(&unmarshaled)

	if err != nil {
		return err
	}

	if len(unmarshaled) == 0 {
		return errors.New("empty transaction type provided")
	}

	switch unmarshaled {
	case "simple":
		*bt = TxTypeSimple
	case "contract":
		*bt = TxTypeContract
	default:
		return errors.New("TX Type is incorrectly defined")
	}

	return nil
}
