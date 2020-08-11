package results

import "sort"

// Generic result structure that will be encoded and sent back to the master and combined
type Results struct {
	TxLatencies    []float64 `json:TxLatencies`    // Latency of each transaction, can be used in CDF
	AverageLatency float64   `json:AverageLatency` // Averaged latency of the transactions
	Throughput     float64   `json:Throughput`     // Number of transactions per second "committed"
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

func CalculateAggregatedResults(clientResults []Results) AggregatedResults {

	if len(clientResults) == 0 {
		return AggregatedResults{}
	}

	// First, we want to get all the information
	var averageThroughput float64 = 0
	var maxThroughput float64 = 0
	minThroughput := clientResults[0].Throughput

	var allLatencies []float64
	var averageLatency float64 = 0

	for _, res := range clientResults {
		allLatencies = append(allLatencies, res.TxLatencies...)

		// Averages
		averageLatency += res.AverageLatency
		averageThroughput += res.Throughput

		// Maximum and minimums
		if res.Throughput > maxThroughput {
			maxThroughput = res.Throughput
		}
		if res.Throughput < minThroughput {
			minThroughput = res.Throughput
		}
	}

	// If empty
	if allLatencies == nil {
		allLatencies = []float64{0}
	}

	sort.Float64s(allLatencies)
	averageThroughput = averageThroughput / float64(len(clientResults))
	averageLatency = averageLatency / float64(len(clientResults))
	var medianLatency float64 = 0

	// If it's even
	midNumber := len(allLatencies) / 2
	if len(allLatencies)%2 == 0 {
		medianLatency = (allLatencies[midNumber-1] + allLatencies[midNumber]) / 2
	} else {
		medianLatency = allLatencies[midNumber]
	}

	return AggregatedResults{
		ClientResults:     clientResults,
		MaxThroughput:     maxThroughput,
		MinThroughput:     minThroughput,
		AverageThroughput: averageThroughput,
		MaxLatency:        allLatencies[len(allLatencies)-1],
		MinLatency:        allLatencies[0],
		AverageLatency:    averageLatency,
		MedianLatency:     medianLatency,
	}
}
