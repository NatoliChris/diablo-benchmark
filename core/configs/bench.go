package configs

// Benchmark configuration structure, all the information from the event.
type BenchConfig struct {
	Name        string    `yaml:"name"`                  // Name of the benchmark
	Description string    `yaml:"description,omitempty"` // Description of what it is
	Threads     int       `yaml:"threads"`               // Number of threads per secondary expected
	Secondaries int       `yaml:"secondaries"`           // Number of secondary machines
	TxInfo      BenchInfo `yaml:"bench,flow"`            // Benchmark transaction information
}

// Benchmark information about transaction intervals and types.
type BenchInfo struct {
	TxType    BenchTransactionType `yaml:"type"` // Type of the transactions (simple, contract)
	Intervals TPSIntervals         `yaml:"txs"`  // Transactions
}
