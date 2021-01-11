// Package configs provides the parsers and validators that deal specifically
// with the configuration files. The configuration files (bench, chain) provide
// critical information for the benchmark and all processing must be done prior
// to the generation of the workload.
package configs

import "diablo-benchmark/core/workload"

// BenchConfig provides the main benchmark configuration structure, all information about the specified workload
type BenchConfig struct {
	Name         string       `yaml:"name"` // Name of the benchmark.
	Path         string       // The location of this benchmark file (to be used in result printing)
	Description  string       `yaml:"description,omitempty"` // Description of what it is.
	Threads      int          `yaml:"threads"`               // Number of threads per secondary expected.
	Secondaries  int          `yaml:"secondaries"`           // Number of secondary machines.
	Timeout      int          `yaml:"timeout"`               // Timeout for the benchmark after sending
	TxInfo       BenchInfo    `yaml:"bench,flow"`            // Benchmark transaction information.
	ContractInfo ContractInfo `yaml:"contract,omitempty"`    // Contract Information
}

// BenchInfo provides specific information about transaction type and intervals
type BenchInfo struct {
	TxType      BenchTransactionType              `yaml:"type"`               // Type of the transactions (simple, contract).
	DataPath    string                            `yaml:"datapath,omitempty"` // Data path of the transactions
	Intervals   TPSIntervals                      `yaml:"txs"`                // Transactions.
	PremadeInfo workload.PremadeBenchmarkWorkload // Premade workload (if exists)
}

// ContractParam defines the contract function parameters
type ContractParam struct {
	Type  string `yaml:"type"`  // The argument type, (e.g. uint64).
	Value string `yaml:"value"` // The value of the argument (as string for easy conversion).
}

// ContractFunction implements the details for a contract function
type ContractFunction struct {
	Name     string          `yaml:"name"`        // The name identifier for the function e.g. storeVal(uint32)
	Type     string          `yaml:"ftype"`       // Function type: "read", "write", "deploy" (note: deploy is for the constructor).
	Ratio    int             `yaml:"ratio"`       // Percentage of the workload that this function calls take.
	PayValue string          `yaml:"value"`       // Monetary value to send to the contract with this transaction, default is 0.
	Params   []ContractParam `yaml:"params,flow"` // Parameters of the function
}

// ContractInfo defining the path and functions that would be called.
type ContractInfo struct {
	Path      string             `yaml:"path"`           // Path of the contract file to be deployed (e.g. Solidity File).
	Name      string             `yaml:"name"`           // The contract name (required for multiple deployed contracts)
	Functions []ContractFunction `yaml:"functions,flow"` // Functions that should be called.
}
