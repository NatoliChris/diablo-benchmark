package blockchains

// Generic block for the blockchains, this may or may not be fully filled.
// This should be extended to accompany for other blockchains but MUST retain
// base functionality for other chains.
type GenericBlock struct {
	Hash              string // Unique identifier for the block
	Index             uint64 // Height of the block as an index
	Timestamp         string // String of the timestamp in ISO8601 format (YYYY-MM-DDTHH:MM:SSZ[+/-]HH:MM)
	TransactionNumber int    // Number of transactions included in the block
}
