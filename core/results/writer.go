package results

import (
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"time"

	"go.uber.org/zap"
)

// checkFileExists is a simple stat check to ensure that the file
// exists at the given path.
func checkFileExists(path string) bool {
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		return false
	}
	return true
}

// checkIsRegular checks if the file is a regular file, else it's a special
// file (that can't be copied)
func checkIsRegular(path string) bool {
	stat, err := os.Stat(path)
	if err != nil {
		return false
	}

	if !stat.Mode().IsRegular() {
		return false
	}

	return true
}

// copyFile copies a file from the source to the destination.
// Note: It can only copy regular files.
func copyFile(fromPath string, toPath string) error {
	// Make sure we can copy the configurations
	if !checkIsRegular(fromPath) {
		return fmt.Errorf("%s is not a regular file that can be copied", fromPath)
	}

	// Open and check the files
	source, err := os.Open(fromPath)
	if err != nil {
		return err
	}
	defer source.Close()

	dest, err := os.Create(toPath)
	if err != nil {
		return err
	}
	defer dest.Close()

	_, err = io.Copy(dest, source)

	return err
}

// writeResults marshals the data into JSON and writes the result as a JSON file
func writeResults(path string, data AggregatedResults) error {
	f, err := json.MarshalIndent(data, "", " ")

	if err != nil {
		return err
	}

	err = ioutil.WriteFile(path, f, 0644)
	return err
}

// WriteResultsToFile is dedicated to bundle all result information into a given directory, writing the results to a JSON as
// well as the containing benchmark and chain configuration files
func WriteResultsToFile(benchConfig string, chainConfig string, results AggregatedResults, resultDir string) error {
	// First, check that the directory exists
	if !checkFileExists(resultDir) {
		zap.L().Warn(fmt.Sprintf("Directory %s does not exist, creating it", resultDir))
		err := os.Mkdir(resultDir, 0755)
		if err != nil {
			return err
		}
	}

	ts := fmt.Sprintf("%v", time.Now().Format(time.RFC3339))

	// Write the results to file
	err := writeResults(fmt.Sprintf("%s/%s_results.json", resultDir, ts), results)
	if err != nil {
		return err
	}

	zap.L().Info(fmt.Sprintf("Results saved in: %s/%s_results.json", resultDir, ts))

	err = copyFile(benchConfig, fmt.Sprintf("%s/%s_workload.yaml", resultDir, ts))
	if err != nil {
		return err
	}

	err = copyFile(chainConfig, fmt.Sprintf("%s/%s_chain.yaml", resultDir, ts))
	return err
}

// Display presents the formatting to display the results to stdout.
// TODO: future - this can be made to show graphs, and present the results in a much nicer way!
func Display(results AggregatedResults) {

	fmt.Println()
	fmt.Println("--------------------------")
	fmt.Println("Benchmark Complete")
	fmt.Println("--------------------------")
	fmt.Println("[*] Aggregated Stats")
	fmt.Println(fmt.Sprintf("\t [-] Throughput [tx/sec]: %.3f [Min: %.3f | Max: %.3f]", results.AverageThroughput, results.MinThroughput, results.MaxThroughput))
	fmt.Println(fmt.Sprintf("\t [-] Latency        [ms]: %.3f [Min: %+v | Max: %+v]", results.AverageLatency, results.MinLatency, results.MaxLatency))

	for i, v := range results.SecondaryResults {
		fmt.Println(fmt.Sprintf("[*] Secondary %d Stats", i))
		fmt.Println(fmt.Sprintf("\t [-] Throughput [tx/sec]: %.3f", v.Throughput))
		fmt.Println(fmt.Sprintf("\t [-] Latency        [ms]: %.3f", v.AverageLatency))
	}

	fmt.Println()

}
