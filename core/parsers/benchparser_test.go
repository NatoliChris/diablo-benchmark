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
				"%s mismatch: expected %v, got: %v",
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

		intervalValue := 2
		currentValue := 10
		for i := 10; i < 40; i++ {
			check(
				fmt.Sprintf("non-Zero start linear rate [%d]", i),
				currentValue,
				bConfig.TxInfo.Intervals[i],
			)
			currentValue += intervalValue
		}
	})

	t.Run("test ramp-up no clear divisions", func(t *testing.T) {
		exampleBytes := []byte(sampleConfig)

		bConfig, err := parseBenchYaml(exampleBytes)

		if err != nil {
			t.Errorf("Failed to parse YAML")
		}

		for i := 0; i <= 10; i++ {
			check(
				fmt.Sprintf("single-rate send at start"),
				70,
				bConfig.TxInfo.Intervals[i],
			)
		}

		check(
			fmt.Sprintf("ramp-up-values"),
			40,
			bConfig.TxInfo.Intervals[30],
		)
	})
}
