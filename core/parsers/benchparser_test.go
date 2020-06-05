package parsers

import (
	"diablo-benchmark/core/configs"
	"fmt"
	"testing"
)

const sampleConfig = `
name: "sample"
description: "descriptions"
bench:
  type: "simple"
  txs:
    0: 70
    10: 70
    30: 40
`

const exampleIncorrectName = `name: 12345
description: "descriptions"
bench:
  type: "simple"
  txs:
    0: 70
    10: 70
    30: 40`

const exampleIncorrectNameTwo = `
name:
  type: "Hello"
description: "descriptions"
bench:
  type: "simple"
  txs:
    0: 70
    10: 70
    30: 40`

const exampleMissingName = `
description: "descriptions"
bench:
  type: "simple"
  txs:
    0: 70
    10: 70
    30: 40
`
const exampleIncorrectDescription = `name: "sample"
description: 12371598
bench:
  type: "simple"
  txs:
    0: 70
    10: 70
    30: 40`

const exampleMissingDescription = `name: "sample"
bench:
  type: "simple"
  txs:
    0: 70
    10: 70
    30: 40`

const exampleIncorrectTxType = `name: "sample"
description: "descriptions"
bench:
  type: "transaction"
  txs:
    0: 70
    10: 70
    30: 40`

const exampleIncorrectTxTypeTwo = `name: "sample"
description: "descriptions"
bench:
  type: 123123132
  txs:
    0: 70
    10: 70
    30: 40`

const exampleEmptyTx = `name: "sample"
description: "descriptions"
bench:
  type: "simple"
  txs:`

const exampleInvalidKeys = `name: "sample"
description: "descriptions"
bench:
  type: "simple"
  txs:
    hello: 70
	10: 70
	30: 40`

const exampleInvalidKeysTwo = `name: "sample"
description: "descriptions"
bench:
  type: "simple"
  txs:
    - 0: 70
    - 10: 70
    - 30: 40`

const exampleNegativeTxKey = `name: "sample"
description: "descriptions"
bench:
	type: "simple"
	txs:
		0: 70
		-10: 70
		30: 40`

func TestParseSampleBenchConfig(t *testing.T) {

	check := func(fn string, expected, got interface{}) bool {
		if got != expected {
			t.Errorf(
				"%s mismatch: expected %v, got: %v",
				fn,
				expected,
				got,
			)
			return false
		}

		return true
	}

	t.Run("test no errors", func(t *testing.T) {
		sampleBytes := []byte(sampleConfig)

		_, err := parseBenchYaml(sampleBytes)

		if err != nil {
			t.Errorf("Failed to parse yaml, reason: %s", err.Error())
		}
	})

	t.Run("test all values present", func(t *testing.T) {
		sampleBytes := []byte(sampleConfig)

		bConfig, err := parseBenchYaml(sampleBytes)

		if err != nil {
			t.Errorf("failed to parse yaml, err: %s", err)
		}

		check("name", "sample", bConfig.Name)
		check("description", "descriptions", bConfig.Description)
		check("txtype", configs.TxTypeSimple, bConfig.TxInfo.TxType)
		// Should be finalValue + 1 - this accounts for the 0th second starting interval.
		check("fullTxLength", 31, len(bConfig.TxInfo.Intervals))
	})

	t.Run("test filling values onerate", func(t *testing.T) {
		exampleOneRateConfig := `
name: "sample"
description: "descriptions"
bench:
  type: "simple"
  txs:
    0: 70
    10: 70
    40: 70
`
		sampleBytes := []byte(exampleOneRateConfig)

		bConfig, err := parseBenchYaml(sampleBytes)

		if err != nil {
			t.Errorf("failed to parse yaml, err: %s", err)
		}

		for i := 0; i < 40; i++ {
			check(fmt.Sprintf("oneRate array [%d]", i),
				70,
				bConfig.TxInfo.Intervals[i],
			)
		}
	})

	t.Run("test contract rate", func(t *testing.T) {
		exampleContractConfig := `
name: "sample"
description: "descriptions"
bench:
  type: "contract"
  txs:
    0: 70
    10: 70
`
		sampleBytes := []byte(exampleContractConfig)

		_, err := parseBenchYaml(sampleBytes)

		if err != nil {
			t.Errorf("failed to parse yaml, err: %s", err)
			t.FailNow()
		}

	})
	t.Run("test non_zero provided should start at 0", func(t *testing.T) {
		exampleNonZeroStart := `
name: "sample"
description: "descriptions"
bench:
  type: "simple"
  txs:
    10: 10
    40: 70
`
		sampleBytes := []byte(exampleNonZeroStart)

		bConfig, err := parseBenchYaml(sampleBytes)

		if err != nil {
			t.Errorf("failed to parse yaml, err: %s", err)
			t.FailNow()
		}

		for i := 0; i < 10; i++ {
			if !check(fmt.Sprintf("non-Zero starting [%d]", i),
				i,
				bConfig.TxInfo.Intervals[i],
			) {
				t.FailNow()
			}
		}

		intervalValue := 2
		currentValue := 10
		for i := 10; i < 40; i++ {
			if !check(
				fmt.Sprintf("non-Zero start linear rate [%d]", i),
				currentValue,
				bConfig.TxInfo.Intervals[i],
			) {
				t.FailNow()
			}
			currentValue += intervalValue
		}
	})

	t.Run("test ramp-up no clear divisions", func(t *testing.T) {
		exampleBytes := []byte(sampleConfig)

		bConfig, err := parseBenchYaml(exampleBytes)

		if err != nil {
			t.Errorf("Failed to parse YAML")
			t.FailNow()
		}

		for i := 0; i <= 10; i++ {
			if !check(
				fmt.Sprintf("single-rate send at start"),
				70,
				bConfig.TxInfo.Intervals[i],
			) {
				t.FailNow()
			}
		}

		check(
			fmt.Sprintf("ramp-up-values"),
			40,
			bConfig.TxInfo.Intervals[30],
		)
	})
}

func TestFailures(t *testing.T) {
	t.Run("non yaml", func(t *testing.T) {
		exampleNonYaml := "128471798fsd7f9"

		_, err := parseBenchYaml([]byte(exampleNonYaml))

		if err == nil {
			t.Errorf("Expected to fail on non-valid yaml")
			t.FailNow()
		}
	})

	t.Run("wrong config", func(t *testing.T) {

		exampleCorrectYaml := `name: "ethereum"
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

		_, err := parseBenchYaml([]byte(exampleCorrectYaml))

		if err == nil {
			t.Errorf("expected to fail incorrect config")
			t.FailNow()
		}
	})

}

func TestInvalidTypes(t *testing.T) {

	checkShouldntParse := func(msg string, s string) bool {
		_, err := parseBenchYaml([]byte(s))

		if err == nil {
			t.Errorf("[%s] Expected to fail on non-valid yaml", msg)
			return false
		}
		return true
	}

	checkShouldParse := func(msg string, s string) bool {
		_, err := parseBenchYaml([]byte(s))

		if err != nil {
			t.Errorf("[%s] Expected not to fail on valid yaml", msg)
			return false
		}

		return true
	}

	t.Run("invalid name", func(t *testing.T) {
		if !checkShouldParse("incorrect name", exampleIncorrectName) {
			t.FailNow()
		}
		if !checkShouldntParse("incorrect name 2", exampleIncorrectNameTwo) {
			t.FailNow()
		}
		if !checkShouldntParse("missing name", exampleMissingName) {
			t.FailNow()
		}
	})

	t.Run("invalid description", func(t *testing.T) {
		if !checkShouldParse("incorrect description", exampleIncorrectDescription) {
			t.FailNow()
		}
		if !checkShouldParse("missing description", exampleMissingDescription) {
			t.FailNow()
		}
	})

	t.Run("invalid txType", func(t *testing.T) {
		if !checkShouldntParse("incorrect tx type", exampleIncorrectTxType) {
			t.FailNow()
		}
		if !checkShouldntParse("incorrect tx type 2", exampleIncorrectTxTypeTwo) {
			t.FailNow()
		}
	})

	t.Run("empty tx list", func(t *testing.T) {
		if !checkShouldntParse("empty tx type", exampleEmptyTx) {
			t.FailNow()
		}
	})

	t.Run("invalid key for tx", func(t *testing.T) {
		if !checkShouldntParse("invalid keys", exampleInvalidKeys) {
			t.FailNow()
		}
		if !checkShouldntParse("invalid keys 2", exampleInvalidKeysTwo) {
			t.FailNow()
		}
	})

	t.Run("negative key for tx", func(t *testing.T) {
		if !checkShouldntParse("negative keys", exampleNegativeTxKey) {
			t.FailNow()
		}
	})

	// t.Run("invalid value for tps", func(t *testing.T) {

	// 	exampleInvalidTPS := `name: "sample"
	// description: "descriptions"
	// bench:
	// 	type: "simple"
	// txs:
	// 	0: 70
	// 	10: 70
	// 	30: 40`
	// })

	// t.Run("negative value for tps", func(t *testing.T) {

	// 	exampleNegativeTPS := `name: "sample"
	// description: "descriptions"
	// bench:
	// 	type: "simple"
	// txs:
	// 	0: 70
	// 	10: 70
	// 	30: 40`
	// })
}
