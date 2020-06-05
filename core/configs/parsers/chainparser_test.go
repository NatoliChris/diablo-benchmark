package parsers

import (
	"bytes"
	"fmt"
	"testing"
)

const exampleCorrectYaml = `name: "ethereum"
nodes:
  - 127.0.0.1:30303
  - 127.0.0.1:30304
  - 127.0.0.1:30305
  - 127.0.0.1:30306
keys:
  - private: "0xf5981d1c9cbdc1e0e570d19d833e0db96af31d3b65f6b67f8e5b2ab7afc5ffc8"
    address: "0x27c40e0fc653679a205754ca76f3371ec127baba"
  - private: "0xb33cb58af3686ce54cc081b0ae095242702618d8f9b2b1f421fa523d337fca9c"
    address: "0x3438d5c33bc1f8c4ef69affb891a58b1d67f8ad7"`

func TestCanParseCorrectYaml(t *testing.T) {
	t.Run("test no error", func(t *testing.T) {
		exampleBytes := []byte(exampleCorrectYaml)

		_, err := parseChainYaml(exampleBytes)

		if err != nil {
			t.Errorf("Failed to parse yaml, reason: %s", err.Error())
		}
	})

	t.Run("test all struct fields", func(t *testing.T) {
		exampleBytes := []byte(exampleCorrectYaml)

		correctNodes := []string{
			"127.0.0.1:30303",
			"127.0.0.1:30304",
			"127.0.0.1:30305",
			"127.0.0.1:30306",
		}

		c, err := parseChainYaml(exampleBytes)

		if err != nil {
			fmt.Println(err.Error())
			t.FailNow()
		}

		// Should have the correct name
		if c.Name != "ethereum" {
			fmt.Println("Failed name")
			t.FailNow()
		}

		// Should have correct number of nodes
		if len(c.Nodes) != len(correctNodes) {
			fmt.Println("Failed nodes length")
			t.FailNow()
		}

		// Compare the nodes
		for i, node := range c.Nodes {
			if node != correctNodes[i] {
				fmt.Println("Failed node comparison")
				t.FailNow()
			}
		}

		// We should get the correct number of keys
		if len(c.Keys) != 2 {
			t.Errorf("Failed to get keys in parse, length was %d", len(c.Keys))
		}

		// Make sure that we have the correct bytes.
		// Known byte sequence of the key above.
		keyBytes := []byte{245, 152, 29, 28, 156, 189, 193, 224, 229, 112, 209, 157, 131, 62, 13, 185, 106, 243, 29, 59, 101, 246, 182, 127, 142, 91, 42, 183, 175, 197, 255, 200}

		if bytes.Compare(keyBytes, c.Keys[0].PrivateKey) != 0 {
			t.Errorf("private key did not unmarshal to correct bytes")
		}
	})
}
