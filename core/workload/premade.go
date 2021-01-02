package workload

import (
	"encoding/json"
	"io/ioutil"
	"os"
)

// PremadeTransaction defines the premade transaction information that will be defined
// in the JSON for the benchmark
type PremadeTransaction struct {
	ID         string       `json:"ID"`                 // Unique transaction number
	From       string       `json:"from"`               // From which account
	To         string       `json:"to"`                 // To which account
	Value      string       `json:"value"`              // Value of the transcation
	Function   string       `json:"function,omitempty"` // Function Name
	DataParams []DataParams `json:"params,flow"`        // Parameters to invoke a function call
	Rawdata    string       `json:"rawdata, omitempty"` // Raw data for an already-existing function
}

// DataParams are the parameters passed into a function
type DataParams struct {
	Name  string `json:"name"`  // Parameter Name
	Type  string `json:"type"`  // Type of the parameter
	Value string `json:"value"` // Value of the parameter
	Raw   string `json:"raw"`   // Raw value of the data params, will only be [{raw: ___}]
}

// PremadeBenchmarkWorkload is the entire workload produced for premade transaction
// information already in the form of [secondary][thread]
type PremadeBenchmarkWorkload [][][][]PremadeTransaction

// ParsePremade parses the json file associated with the premade workload.
// This file must contain all the information for all transactions in the workload
func ParsePremade(filepath string) (*PremadeBenchmarkWorkload, error) {
	// Attempt to open the file
	fp, err := os.Open(filepath)

	if err != nil {
		return nil, err
	}

	// Defer closing the file
	defer fp.Close()

	var premade PremadeBenchmarkWorkload

	fileBytes, err := ioutil.ReadAll(fp)
	err = json.Unmarshal(fileBytes, &premade)
	if err != nil {
		return nil, err
	}

	return &premade, nil
}
