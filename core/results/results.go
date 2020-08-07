package results

// Generic result structure that will be encoded and sent back to the master and combined
type Results struct {
	TxLatencies    []float64 // Latency of each transaction, can be used in CDF
	AverageLatency float64   // Averaged latency of the transactions
	Throughput     float64   // Number of transactions per second "committed"
}

type AggregatedResults struct {
	ClientResults     []Results // All results from clients
	MaxThroughput     float64   // maximum throughput observed
	MinThroughput     float64   // minimum throughput observed
	AverageThroughput float64   // average throughput
	MaxLatency        float64   // highest latency observed
	MinLatency        float64   // smallest latency observed
	AverageLatency    float64   // average latency
	MedianLatency     float64   // median latency
}
