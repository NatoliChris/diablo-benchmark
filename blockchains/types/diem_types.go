package types

type DiemTX struct {
	ID           uint64   `json:"id"`            	// id used in the client interface to keep track of the transaction and register departure and arrival time
	Name		 string   `json:name`				// name of the function to execute
	Path		 string   `json:"path"` 			// path to the compiled move byte code
	FunctionType string   `json:"function_type"` 	// "write" or "read", indicates whether we query or submit, it is given in the benchmark config of the workload (ftype in bench.go)
	Args         []string `json:"args"`          	// arguments to invoke the chaincode

}
