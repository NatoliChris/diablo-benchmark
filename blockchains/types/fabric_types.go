package types

type FabricTX struct {
	ID uint64 `json:"id"`
	FunctionName string  `json:"function_name"`
	FunctionType string  `json:function_type`
	Args []string `json:"args"`
}

type FabricWorkload []FabricTX