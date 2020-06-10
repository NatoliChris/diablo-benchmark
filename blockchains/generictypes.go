package blockchains

// Generic block for the blockchains, this may or may not be fully filled.
// This should be extended to accompany for other blockchains but MUST retain
// base functionality for other chains.
type GenericBlock struct {
	Hash              string // Unique identifier for the block
	Index             uint64 // Height of the block as an index
	Timestamp         uint64 // Unix timestamp of the block
	TransactionNumber int    // Number of transactions included in the block
}
