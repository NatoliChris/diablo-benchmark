package types

//FabricTX represents all the necessary information for an
// Hyperledger Fabric transaction
type FabricTX struct {
	ID           uint64   `json:"id"`            // id used in the client interface to keep track of the transaction and register departure and arrival time
	FunctionName string   `json:"function_name"` // name of the function to be invoked in the chaincode/smart contract
	FunctionType string   `json:"function_type"` // "write" or "read", indicates whether we query or submit, it is given in the benchmark config of the workload (ftype in bench.go)
	Args         []string `json:"args"`          // arguments to invoke the chaincode
}

//FabricUser represents a user/account which is the identity by
// which the secondary will contact the Fabric network
type FabricUser struct {
	Label string `yaml:label` // user label in the identity wallet used by the gateway
	MspID string `yaml:"mspID"`
	Cert  string `yaml:"cert"`
	Key   string `yaml:"key"`
}

type FabricCommitEvent struct {
	Valid bool
	ID     uint64 // the ID used in client to keep track of the transaction and register throughput
}
