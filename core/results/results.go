// Package results contains the information about the results and handles the
// processing and display / logging of the information. The results are passed
// from all secondaries and collated by the primary.
// The goal of this package is to provide a central point to display the results
// which over time will develop into more complex processing and utilisation of
// available information.
package results

import "sort"

const MAXLAT float64 = 3000

// Results is the generic result structure that will be encoded and sent back to the primary and combined
type Results struct {
	TxLatencies       []float64 `json:"TxLatencies"`       // Latency of each transaction, can be used in CDF
	AverageLatency    float64   `json:"AverageLatency"`    // Averaged latency of the transactions
	MedianLatency     float64   `json:"MedianLatency"`     // Median Latency of the transaction
	Throughput        float64   `json:"Throughput"`        // Number of transactions per second "committed"
	ThroughputSeconds []float64 `json:"ThroughputSeconds"` // Number of transactions "committed" over second periods to measure dynamic throughput
}

// AggregatedResults returns all the information from all secondaries, and
// stores the calculated information (e.g. max, min, ...)
type AggregatedResults struct {
	RawResults                    [][]Results `json:"RawResults"`                           // Result of [secondary][thread]
	ResultsPerSecondary           []Results   `json:"ResultsPerSecondary"`                  // Aggregation of results per secondary
	MinLatency                    float64     `json:"MinLatency"`                           // Minimum latency across all workers and secondaries
	AverageLatency                float64     `json:"AverageLatency"`                       // Average latency across all workers and secondaries
	AverageLatencyPerSecondary    []float64   `json:"AverageLatencyPerSecond"`              // Average latency per secondary
	MedianLatency                 float64     `json:"MedianAverageLatency"`                 // Median latency value
	MaxLatency                    float64     `json:"MaxAverageLatency"`                    // Highest latency across all secondaries and workers
	TotalThroughputTimes          []float64   `json:"TotalThroughputPerSecond"`             // Throughput over time
	AverageThroughputPerSecondary []float64   `json:"OverallThroughputPerSecondary"`        // Throughput per secondary
	TotalThroughputSecondaryTime  [][]float64 `json:"TotalThroughputPerSecondaryPerSecond"` // Throughput over time for each secondary
	MaxThroughput                 float64     `json:"MaximumOverallThroughput"`             // Highest throughput
	MinThroughput                 float64     `json:"MinimumOverallThroughput"`             // Highest throughput
	OverallThroughput             float64     `json:"OverallAverageThroughput"`             // Overall throughput measured as success / time
}

// Return the median of a list
func getMedian(arr []float64) float64 {
	arrSorted := arr
	sort.Float64s(arrSorted)

	midNumber := len(arrSorted) / 2
	if len(arrSorted)%2 == 0 {
		return (arrSorted[midNumber-1] + arrSorted[midNumber]) / 2
	} else {
		return arrSorted[midNumber]
	}
}

// CalculateAggregatedResults calculates the aggregated results given the set of results from the secondaries
func CalculateAggregatedResults(secondaryResults [][]Results) AggregatedResults {

	// Check that it's not empty
	if len(secondaryResults) == 0 {
		return AggregatedResults{}
	}

	// Now let's go through and calculate all the things
	// Results aggregated per secondary
	var ResultsPerSecondary []Results
	// Total throughputs cumulated over time (should range from 0 to end of benchmark)
	totalThroughputOverTime := make([]float64, 0)
	// Average throughput per secondary
	var throughputPerSecondary []float64
	// Total throughput per secondary per second (throughput over time)
	var throughputOverTimeSecondary [][]float64

	// Min/Max/Average Latency
	maxTotalLatency := float64(0)
	averageTotalLatency := float64(0)
	minTotalLatency := MAXLAT

	var latencyPerSecondary []float64

	// Throughput total
	maxTotalThroughput := float64(0)
	averageTotalThroughput := float64(0)

	// Iterate through the results
	for _, secondaryResult := range secondaryResults {
		txLatencies := make([]float64, 0)
		averageLatencyPerSecondary := float64(0)
		secondaryThroughputs := make([]float64, 0)
		latencyEntries := float64(0)
		avgThroughputPerSecondary := float64(0)
		// For each worker
		for _, workerResult := range secondaryResult {
			// 1. get the latency average per secondary
			latencyEntries += float64(len(workerResult.TxLatencies))
			for latencyIdx, v := range workerResult.TxLatencies {
				averageLatencyPerSecondary += v
				if v < minTotalLatency {
					minTotalLatency = v
				}

				if v > maxTotalLatency {
					maxTotalLatency = v
				}

				if latencyIdx >= len(txLatencies) {
					txLatencies = append(txLatencies, 0)
				}
				txLatencies[latencyIdx] += v
			}

			// 2. Obtain throughputs
			for timeIndex, v := range workerResult.ThroughputSeconds {
				if timeIndex >= len(totalThroughputOverTime) {
					totalThroughputOverTime = append(totalThroughputOverTime, 0)
				}
				totalThroughputOverTime[timeIndex] += v

				if timeIndex >= len(secondaryThroughputs) {
					secondaryThroughputs = append(secondaryThroughputs, 0)
				}
				secondaryThroughputs[timeIndex] += v
			}
			avgThroughputPerSecondary += workerResult.Throughput
		}

		// fix the average latencies
		avgLatency := float64(0)
		for idx := 0; idx < len(txLatencies); idx++ {
			txLatencies[idx] = txLatencies[idx] / float64(len(secondaryResult))
			avgLatency += txLatencies[idx]
		}
		avgLatency = avgLatency / float64(len(txLatencies))
		latencyPerSecondary = append(latencyPerSecondary, avgLatency)

		sortedLatencies := txLatencies
		sort.Float64s(sortedLatencies)
		medianLatency := float64(0)
		midNumber := len(txLatencies) / 2
		if len(txLatencies)%2 == 0 {
			medianLatency = (txLatencies[midNumber-1] + txLatencies[midNumber]) / 2
		} else {
			medianLatency = txLatencies[midNumber]
		}

		averageTotalLatency += avgLatency
		throughputOverTimeSecondary = append(throughputOverTimeSecondary, secondaryThroughputs)

		throughputPerSecondary = append(throughputPerSecondary, avgThroughputPerSecondary/float64(len(secondaryResult)))
		ResultsPerSecondary = append(ResultsPerSecondary, Results{
			TxLatencies:       txLatencies,
			ThroughputSeconds: secondaryThroughputs,
			Throughput:        avgThroughputPerSecondary / float64(len(secondaryResult)),
			AverageLatency:    avgLatency,
			MedianLatency:     medianLatency,
		})
	}

	// Fix up the average and median latency
	averageTotalLatency = averageTotalLatency / float64(len(secondaryResults))
	medianLatencyTotal := getMedian(latencyPerSecondary)

	// Fix up the overall throughput and average throughput
	minTotalThroughput := totalThroughputOverTime[0]
	for _, v := range totalThroughputOverTime {
		if v > maxTotalThroughput {
			maxTotalThroughput = v
		}

		if v < minTotalThroughput {
			minTotalThroughput = v
		}
		averageTotalThroughput += v
	}

	averageTotalThroughput = averageTotalThroughput / float64(len(totalThroughputOverTime))

	// Return the absolute mass of results chunked together!
	return AggregatedResults{
		RawResults:                    secondaryResults,
		ResultsPerSecondary:           ResultsPerSecondary,
		MinLatency:                    minTotalLatency,
		AverageLatency:                averageTotalLatency,
		AverageLatencyPerSecondary:    latencyPerSecondary,
		MedianLatency:                 medianLatencyTotal,
		MaxLatency:                    maxTotalLatency,
		TotalThroughputTimes:          totalThroughputOverTime,
		AverageThroughputPerSecondary: throughputPerSecondary,
		TotalThroughputSecondaryTime:  throughputOverTimeSecondary,
		MaxThroughput:                 maxTotalThroughput,
		MinThroughput:                 minTotalThroughput,
		OverallThroughput:             averageTotalThroughput,
	}
}
