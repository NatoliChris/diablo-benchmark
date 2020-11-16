package configs

import (
	"encoding/hex"
	"encoding/json"
	"errors"
)

// BenchTransactionType provides the type of benchmark
// Currently:
type BenchTransactionType string

// TxTypeSimple denoting just value transfer transactions in the workload.
const TxTypeSimple BenchTransactionType = "simple"

// TxTypeContract indicates that it will be a contract workload that has
// contract interaction and deployment
const TxTypeContract BenchTransactionType = "contract"

const TxTypeTest BenchTransactionType = "test"

// DefaultTimeout is the default timeout for the benchmark if not provided
// or overwritten by the args
const DefaultTimeout int = 20

// TPSIntervals defines the "Transactions Per Second" send rate intervals
type TPSIntervals map[int]int

// ChainKey is a basic "key" structure for generic blockchains, provides a private key in
// bytes, and the "Address" as a string - the address may change depending on
// the blockchain.
type ChainKey struct {
	PrivateKey []byte `yaml:"private"` // Private key information
	Address    string `yaml:address`   // Address that it is from
}

// checkPrefix Naive check if the prefixed PrivateKey has "0x" leading.
func checkPrefix(keyHex string) bool {
	return len(keyHex) >= 2 && // Length must be 0x or more
		keyHex[0] == '0' && // Starts with 0
		(keyHex[1] == 'x' || keyHex[1] == 'X') // followed by an x or X
}

// UnmarshalYAML provides a custom YAML decoding for the chain key.
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

// UnmarshalJSON provides a custom JSON unmarshal for 'Chain Key' type.
// It expects the object: {"private": "", "address": ""}
func (ck *ChainKey) UnmarshalJSON(b []byte) error {

	var c struct {
		PrivateKey string `json:"private"`
		Address    string `json:"address"`
	}

	err := json.Unmarshal(b, &c)

	if err != nil {
		return err
	}

	if len(c.PrivateKey) == 0 {
		return errors.New("empty PrivateKey, or invalid, passed to unmarshal")
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

	ck.PrivateKey = privateKeyBytes
	ck.Address = c.Address

	return nil
}

// UnmarshalYAML provides a custom YAML Decode for a "bench transaction" type. (Simple / Contract)
// Note, this will need to be updated with future improvements.
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
