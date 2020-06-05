package configs

// Transaction types (simple, contract, ...)
type BenchTransactionType string

const TxTypeSimple BenchTransactionType = "simple"
const TxTypeContract BenchTransactionType = "contract"

// Transactions Per Second intervals
type TPSIntervals map[int]int
