package blockchains

type FabricTX struct {
	ID uint64 `json:"id"`
	ContractName string `json:"contract_name"`
	FunctionName string  `json:"function_name"`
	Args []string `json:"args"`
}

type FabricWorkload []FabricTX