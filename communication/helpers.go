package communication

import (
	"encoding/json"
)

// Helper function to standardise the way that workloads and information is encoded/decoded through
// communication
func EncodeWorkload(workload [][]byte) ([]byte, error) {
	return json.Marshal(workload)
}

// Helper function to standardise the way workloads and information is decoded through
// communication.
func DecodeWorkload(data []byte) ([][]byte, error) {
	var decodedWorkload [][]byte

	err := json.Unmarshal(data, &decodedWorkload)

	return decodedWorkload, err
}
