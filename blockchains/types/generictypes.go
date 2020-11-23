// Package types provides some useful types used in the packages clientinterfaces and workloadgenerators
package types

// TransactionBenchmarkInformation contains generic information about the
// transaction, stores hash, sent time and time that it was mined into a block
type TransactionBenchmarkInformation struct {
	Hash            string // Unique transaction hash
	SentTime        uint64 // Time that the transaction request was sent
	RequestResponse uint64 // Response time that was returned
	BlockTime       uint64 // Time that it was mined into a block.
}
