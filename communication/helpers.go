package communication

import (
	"diablo-benchmark/blockchains/workloadgenerators"
	"encoding/json"
)

// Helper function to standardise the way that workloads and information is encoded/decoded through
// communication
func EncodeWorkload(workload workloadgenerators.SecondaryWorkload) ([]byte, error) {
	return json.Marshal(workload)
}

// Helper function to standardise the way workloads and information is decoded through
// communication.
func DecodeWorkload(data []byte) (workloadgenerators.SecondaryWorkload, error) {
	var decodedWorkload workloadgenerators.SecondaryWorkload

	err := json.Unmarshal(data, &decodedWorkload)

	return decodedWorkload, err
}
