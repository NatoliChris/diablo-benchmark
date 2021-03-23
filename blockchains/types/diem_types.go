package types

import (
	"time"
)

type DiemTX struct {
	ID           uint64   `json:"id"`            	// id used in the client interface to keep track of the transaction and register departure and arrival time
	Name		 string   `json:name`				// name of the function to execute
	FunctionType string   `json:"function_type"` 	// "write" or "read", indicates whether we query or submit, it is given in the benchmark config of the workload (ftype in bench.go)
	SenderRefId  uint64	  `json:"sender_ref_id"`	// reference id of the sender's account address
	ScriptPath	 string	  `json:"script_path"`
	Args         []string `json:"args"`          	// arguments to invoke the chaincode

}
type DiemAccount struct {
	Address 		string `json:"address"` 			// address used to identify account
	SequenceNumber 	uint64 `json:"sequenceNumber"`		// mark the latest sequence number of transaction
}
type DiemCommitEvent struct{
	Valid bool
	ID     uint64 // the ID used in client to keep track of the transaction and register throughput
	CommitTime time.Time // the time the transaction was committed
}