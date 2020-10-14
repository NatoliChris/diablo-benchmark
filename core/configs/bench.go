// Package configs provides the parsers and validators that deal specifically
// with the configuration files. The configuration files (bench, chain) provide
// critical information for the benchmark and all processing must be done prior
// to the generation of the workload.
package configs

// BenchConfig provides the main benchmark configuration structure, all information about the specified workload
type BenchConfig struct {
	Name         string       `yaml:"name"` // Name of the benchmark.
	Path         string       // The path of the config
	Description  string       `yaml:"description,omitempty"` // Description of what it is.
	Threads      int          `yaml:"threads"`               // Number of threads per secondary expected.
	Secondaries  int          `yaml:"secondaries"`           // Number of secondary machines.
	Timeout      int          `yaml:"timeout"`               // Timeout for the benchmark after sending
	TxInfo       BenchInfo    `yaml:"bench,flow"`            // Benchmark transaction information.
	ContractInfo ContractInfo `yaml:"contract,omitempty"`    // Contract Information
}

// BenchInfo provides specific information about transaction type and intervals
type BenchInfo struct {
	TxType    BenchTransactionType `yaml:"type"` // Type of the transactions (simple, contract).
	Intervals TPSIntervals         `yaml:"txs"`  // Transactions.
}

// ContractParam defines the contract function parameters
type ContractParam struct {
	Type  string `yaml:"type"`  // The argument type, (e.g. uint64).
	Value string `yaml:"value"` // The value of the argument (as string for easy conversion).
}

// ContractFunction implements the details for a contract function
type ContractFunction struct {
	Name   string          `yaml:"name"`  // The name identifier for the function e.g. storeVal(uint32)
	Type   string          `yaml:"ftype"` // Function type: "read", "write", "deploy" (note: deploy is for the constructor).
	Ratio  int             `yaml:"ratio"` // Percentage of the workload that this function calls take.
	Params []ContractParam `yaml:"params,flow"`
}

// ContractInfo defining the path and functions that would be called.
type ContractInfo struct {
	Path      string             `yaml:"path"`           // Path of the contract file to be deployed (e.g. Solidity File).
	Functions []ContractFunction `yaml:"functions,flow"` // Functions that should be called.
}
