package configs

import (
	"encoding/hex"
	"errors"
)

type ChainConfig struct {
	Name  string     `yaml:name`      // Name of the chain (will be used in config print)
	Nodes []string   `yaml:nodes`     // Address of the nodes.
	Keys  []ChainKey `yaml:keys,flow` // Key information
}

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
