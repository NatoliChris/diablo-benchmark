// Package results contains the information about the results and handles the
// processing and display / logging of the information. The results are passed
// from all secondaries and collated by the primary.
// The goal of this package is to provide a central point to display the results
// which over time will develop into more complex processing and utilisation of
// available information.
package results

import (
	"fmt"
	"sort"
	"time"

	"go.uber.org/zap"
)

// Results is the generic result structure that will be encoded and sent back to the primary and combined
type Results struct {
	TxLatencies       []float64              `json:"TxLatencies"`       // Latency of each transaction, can be used in CDF
	AverageLatency    float64                `json:"AverageLatency"`    // Averaged latency of the transactions
	LatencySeconds    map[string][]time.Time `json:"TxTimes"`           // All the transaction sent times.
	MedianLatency     float64                `json:"MedianLatency"`     // Median Latency of the transaction
	Throughput        float64                `json:"Throughput"`        // Number of transactions per second "committed"
	ThroughputSeconds []float64              `json:"ThroughputSeconds"` // Number of transactions "committed" over second periods to measure dynamic throughput
	Success           uint                   `json:"success"`           // Number of successful transactions
	Fail              uint                   `json:"fail"`              // Number of failed transactions
	Timeout           uint                   `json:"timeout"`           // Number of timeouts that occurred (subset of fails)
}

// AggregatedResults returns all the information from all secondaries, and
// stores the calculated information (e.g. max, min, ...)
type AggregatedResults struct {
	// Results
	RawResults       [][]Results `json:"RawResults"`       // Results of [secondary][thread]
	SecondaryResults []Results   `json:"SecondaryResults"` // Aggregation of results per secondary

	// Latency
	MinLatency     float64                `json:"MinLatency"`     // Minimum latency across all workers and secondaries
	AverageLatency float64                `json:"AverageLatency"` // Average latency across all workers and secondaries
	MaxLatency     float64                `json:"MaxLatency"`     // Maximum Latency across all workers and secondaries
	MedianLatency  float64                `json:"MedianLatency"`  // Median Latency across all workers and secondaries
	AllTxLatencies []float64              `json:"AllTxLatencies"` // All Transaction Latencies
	AllTxTimes     map[string][]time.Time `json:"AllTxTimes"`     // All transaction times

	// Throughput
	TotalThroughputTimes         []float64   `json:"TotalThroughputOverTime"`              // Total throughput over time per window
	AverageThroughputSecondary   []float64   `json:"AverageThroughputSecondaries"`         // Average throughput per secondary
	TotalThroughputSecondaryTime [][]float64 `json:"TotalThrouhgputPerSecondaryPerWindow"` // Total throughput per secondary
	MaxThroughput                float64     `json:"MaximumThroughput"`                    // Maximum Throughput reached over time
	MinThroughput                float64     `json:"MinimumThroughput"`                    // Miniumum Throughput reached overall
	AverageThroughput            float64     `json:"AverageThroughput"`                    // Average throughput reached overall

	// Success and Fail
	TotalSuccess uint `json:"TotalSuccess"` // Total number of successes
	TotalFails   uint `json:"TotalFails"`   // Total number of fails
	TotalTimeout uint `json:"TotalTimeout"` // Total number of timeouts (subset of fails)
}

// Return the median of a list
func getMedian(arr []float64) float64 {
	arrSorted := arr
	sort.Float64s(arrSorted)

	midNumber := len(arrSorted) / 2
	if len(arrSorted)%2 == 0 {
		return (arrSorted[midNumber-1] + arrSorted[midNumber]) / 2
	}
	return arrSorted[midNumber]
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
	minTotalLatency := secondaryResults[0][0].AverageLatency

	var latencyPerSecondary []float64
	var allTxLatencies []float64

	// Throughput total
	maxTotalThroughput := float64(0)
	averageTotalThroughput := float64(0)

	totalSuccess := uint(0)
	totalFails := uint(0)
	totalTimeouts := uint(0)

	// Iterate through the results
	for secondaryID, secondaryResult := range secondaryResults {
		txLatencies := make([]float64, 0)
		averageLatencyPerSecondary := float64(0)
		secondaryThroughputs := make([]float64, 0)
		latencyEntries := float64(0)
		avgThroughputPerSecondary := float64(0)
		// For each worker
		numSuccess := uint(0)
		numFails := uint(0)
		numTimeout := uint(0)
		for workerID, workerResult := range secondaryResult {
			// 1. get the latency average per secondary
			latencyEntries += float64(len(workerResult.TxLatencies))
			numSuccess += workerResult.Success
			numFails += workerResult.Fail
			numTimeout += workerResult.Timeout
			for _, v := range workerResult.TxLatencies {
				averageLatencyPerSecondary += v
				if minTotalLatency > v && v > 0 {
					minTotalLatency = v
				}

				if v > maxTotalLatency {
					maxTotalLatency = v
				}

				txLatencies = append(txLatencies, v)
				allTxLatencies = append(allTxLatencies, v)
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

			zap.L().Debug(fmt.Sprintf("Partial Calculations [S: %d, W: %d]", secondaryID, workerID),
				zap.Float64("Sec Throughput Total", avgThroughputPerSecondary),
				zap.Float64("Worker Throughput", workerResult.Throughput),
			)
		}

		// fix the average latencies
		avgLatency := float64(0)
		for idx := 0; idx < len(txLatencies); idx++ {
			avgLatency += txLatencies[idx]
		}
		if len(txLatencies) > 0 {
			avgLatency = avgLatency / float64(len(txLatencies))
		}
		latencyPerSecondary = append(latencyPerSecondary, avgLatency)

		// Get the median latency by sorting -> getting middle number
		sortedLatencies := txLatencies
		sort.Float64s(sortedLatencies)
		medianLatency := float64(0)
		if len(txLatencies) > 0 {
			midNumber := len(txLatencies) / 2
			if len(txLatencies)%2 == 0 {
				medianLatency = (txLatencies[midNumber-1] + txLatencies[midNumber]) / 2
			} else {
				medianLatency = txLatencies[midNumber]
			}
		}

		// Total and averages
		averageTotalLatency += avgLatency
		throughputOverTimeSecondary = append(throughputOverTimeSecondary, secondaryThroughputs)
		throughputPerSecondary = append(throughputPerSecondary, avgThroughputPerSecondary/float64(len(secondaryResult)))

		ResultsPerSecondary = append(ResultsPerSecondary, Results{
			TxLatencies:       txLatencies,
			ThroughputSeconds: secondaryThroughputs,
			Throughput:        avgThroughputPerSecondary / float64(len(secondaryResult)),
			AverageLatency:    avgLatency,
			MedianLatency:     medianLatency,
			Success:           numSuccess,
			Fail:              numFails,
			Timeout:           numTimeout,
		})

		// Update the number of total success and failures
		totalSuccess += numSuccess
		totalFails += numFails
		totalTimeouts += numTimeout
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

		// NOTE - need to check out the minimum throughput, because of the 0 throughput if waiting for timeouts
		if v < minTotalThroughput {
			minTotalThroughput = v
		}
		averageTotalThroughput += v
	}

	averageTotalThroughput = averageTotalThroughput / float64(len(totalThroughputOverTime))

	// DEBUG PURPOSES ONLY
	var avgThroughputAvg float64
	for _, v := range ResultsPerSecondary {
		avgThroughputAvg += v.Throughput
	}
	// END

	zap.L().Debug("Total Throughput Calculations",
		zap.Float64("averageTotal over time", averageTotalThroughput),
		zap.Float64("averageTotal average", avgThroughputAvg),
	)

	// Return the absolute mass of results chunked together!
	return AggregatedResults{
		RawResults:                   secondaryResults,
		SecondaryResults:             ResultsPerSecondary,
		MinLatency:                   minTotalLatency,
		AverageLatency:               averageTotalLatency,
		MedianLatency:                medianLatencyTotal,
		MaxLatency:                   maxTotalLatency,
		TotalThroughputTimes:         totalThroughputOverTime,
		AverageThroughputSecondary:   throughputPerSecondary,
		TotalThroughputSecondaryTime: throughputOverTimeSecondary,
		MaxThroughput:                maxTotalThroughput,
		MinThroughput:                minTotalThroughput,
		AverageThroughput:            avgThroughputAvg,
		TotalSuccess:                 totalSuccess,
		TotalFails:                   totalFails,
		TotalTimeout:                 totalTimeouts,
		AllTxLatencies:               allTxLatencies,
	}
}
