// Package blockchains provides the blockchain integration with the benchmark.
// It defines two packages: clientinterfaces and workloadgenerators, which are
// both used to implement and integrate new blockchains into the benchmark.
//
// Client Interfaces
//
// The clientinterfaces package is where the interaction between the blockchain
// is implemented. The files contained in there are implementation of client
// rpc interactions to send the transactions to the network and record the
// relevant statistics. The client interface runs on each secondary node thread.
//
// Workload Generators
//
// The workload generators are where the transaction creation is done on the
// primary node. This handles the creation of the entire workload for each
// given secondary thread, and handles the account information relevant to
// the workload. Additionally, it will also create and deploy the smart contract.
package blockchains

// TransactionBenchmarkInformation contains generic information about the
// transaction, stores hash, sent time and time that it was mined into a block
type TransactionBenchmarkInformation struct {
	Hash            string // Unique transaction hash
	SentTime        uint64 // Time that the transaction request was sent
	RequestResponse uint64 // Response time that was returned
	BlockTime       uint64 // Time that it was mined into a block.
}
