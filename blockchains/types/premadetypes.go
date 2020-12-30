package types

// PremadeTransaction defines the premade transaction information that will be defined
// in the JSON for the benchmark
type PremadeTransaction struct {
	From       string       `json:"from"`        // From which account
	To         string       `json:"to"`          // To which account
	Value      string       `json:"value"`       // Value of the transcation
	DataParams []DataParams `json:"params,flow"` // Parameters to invoke a function call
}

// DataParams are the parameters passed into a function
type DataParams struct {
	Name  string `json:"name"`  // Parameter Name
	Type  string `json:"type"`  // Type of the parameter
	Value string `json:"value"` // Value of the parameter
}

// PremadeBenchmarkWorkload is the entire workload produced for premade transaction
// information
type PremadeBenchmarkWorkload []PremadeTransaction
