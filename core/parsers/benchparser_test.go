package parsers

import (
	"diablo-benchmark/core/configs"
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

func TestParseSampleBenchConfig(t *testing.T) {

	check := func(fn string, expected, got interface{}) {
		if got != expected {
			t.Errorf(
				"%s mismatch: expected %s, got: %s",
				fn,
				expected,
				got,
			)
		}
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
		check("fullTxLength", 30, len(bConfig.TxInfo.Intervals))
	})
}
