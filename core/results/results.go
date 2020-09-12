// Package results contains the information about the results and handles the
// processing and display / logging of the information. The results are passed
// from all secondaries and collated by the primary.
// The goal of this package is to provide a central point to display the results
// which over time will develop into more complex processing and utilisation of
// available information.
package results

import "sort"

// Results is the generic result structure that will be encoded and sent back to the primary and combined
type Results struct {
	TxLatencies    []float64 `json:"TxLatencies"`    // Latency of each transaction, can be used in CDF
	AverageLatency float64   `json:"AverageLatency"` // Averaged latency of the transactions
	Throughput     float64   `json:"Throughput"`     // Number of transactions per second "committed"
}

// AggregatedResults returns all the information from all secondaries, and
// stores the calculated information (e.g. max, min, ...)
type AggregatedResults struct {
	SecondaryResults  []Results // All results from secondaries
	TotalThroughput   float64   // Total cumulative throughput
	MaxThroughput     float64   // maximum throughput observed
	MinThroughput     float64   // minimum throughput observed
	AverageThroughput float64   // average throughput
	MaxLatency        float64   // highest latency observed
	MinLatency        float64   // smallest latency observed
	AverageLatency    float64   // average latency
	MedianLatency     float64   // median latency
}

// CalculateAggregatedResults calculates the aggregated results given the set of results from the secondaries
func CalculateAggregatedResults(secondaryResults []Results) AggregatedResults {

	if len(secondaryResults) == 0 {
		return AggregatedResults{}
	}

	// First, we want to get all the information
	var averageThroughput float64
	var maxThroughput float64
	minThroughput := secondaryResults[0].Throughput
	var totalThroughput float64

	var allLatencies []float64
	var averageLatency float64

	for _, res := range secondaryResults {
		allLatencies = append(allLatencies, res.TxLatencies...)

		// Averages
		averageLatency += res.AverageLatency
		averageThroughput += res.Throughput
		totalThroughput += res.Throughput

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
	averageThroughput = averageThroughput / float64(len(secondaryResults))
	averageLatency = averageLatency / float64(len(secondaryResults))
	var medianLatency float64

	// If it's even
	midNumber := len(allLatencies) / 2
	if len(allLatencies)%2 == 0 {
		medianLatency = (allLatencies[midNumber-1] + allLatencies[midNumber]) / 2
	} else {
		medianLatency = allLatencies[midNumber]
	}

	return AggregatedResults{
		SecondaryResults:  secondaryResults,
		TotalThroughput:   totalThroughput,
		MaxThroughput:     maxThroughput,
		MinThroughput:     minThroughput,
		AverageThroughput: averageThroughput,
		MaxLatency:        allLatencies[len(allLatencies)-1],
		MinLatency:        allLatencies[0],
		AverageLatency:    averageLatency,
		MedianLatency:     medianLatency,
	}
}
