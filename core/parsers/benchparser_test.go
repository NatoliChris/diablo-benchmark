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
		}

		for i := 0; i < 10; i++ {
			check(fmt.Sprintf("non-Zero starting [%d]", i),
				i,
				bConfig.TxInfo.Intervals[i],
			)
		}
	})
}
